package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const apiSecurityStateVersion = 1

func validateAPISecurityStateVersion(kind string, version int) error {
	// Version 0 is the pre-versioning legacy form and remains readable for
	// upgrade compatibility. Any future/non-canonical version fails closed so
	// a downgraded binary cannot silently reinterpret newer durable state.
	if version != 0 && version != apiSecurityStateVersion {
		return fmt.Errorf("%s state version %d is unsupported (expected %d or legacy 0)", kind, version, apiSecurityStateVersion)
	}
	return nil
}

type apiOperationsStateFile struct {
	Version int            `json:"version"`
	Saved   time.Time      `json:"saved"`
	Ops     []apiOperation `json:"operations"`
}

type schemaStateFile struct {
	Version    int                    `json:"version"`
	Saved      time.Time              `json:"saved"`
	Candidates []*SchemaCandidate     `json:"candidates"`
	Aggregates []schemaAggregateState `json:"aggregates"`
}

type contractStateFile struct {
	Version          int                            `json:"version"`
	Saved            time.Time                      `json:"saved"`
	Contracts        map[string]APIContract         `json:"contracts"`
	Versions         map[string]APIContractVersion  `json:"versions"`
	ContractVersions map[string][]string            `json:"contract_versions"`
	Operations       map[string][]ContractOperation `json:"operations"`
	Bindings         map[string][]OperationBinding  `json:"bindings"`
	Diffs            map[string][]ContractDiff      `json:"diffs"`
}

func apiSecurityStatePath(configPath, name string) string {
	return filepath.Join(filepath.Dir(configPath), name)
}

func atomicWriteJSON(path string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	dirPath := filepath.Dir(path)
	base := filepath.Base(path)
	// Use a unique same-directory temporary file. A fixed path+".tmp" races
	// when an operator-triggered save overlaps the periodic autosave and can
	// also follow a pre-created symlink. CreateTemp gives each writer its own
	// O_EXCL-created inode while preserving atomic same-filesystem rename.
	f, err := os.CreateTemp(dirPath, "."+base+".tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	ok := false
	defer func() {
		_ = f.Close()
		if !ok {
			_ = os.Remove(tmp)
		}
	}()
	if err := f.Chmod(0o640); err != nil {
		return err
	}
	if _, err := f.Write(b); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	ok = true
	if dir, err := os.Open(dirPath); err == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}
	return nil
}

func readJSONIfExists(path string, value any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(b, value)
}

func (s *apiOperationStore) save(configPath string) error {
	if s == nil {
		return nil
	}
	state := apiOperationsStateFile{Version: apiSecurityStateVersion, Saved: time.Now().UTC(), Ops: s.snapshot()}
	return atomicWriteJSON(apiSecurityStatePath(configPath, "api-operations.json"), state)
}

func (s *apiOperationStore) load(configPath string) error {
	if s == nil {
		return nil
	}
	var state apiOperationsStateFile
	if err := readJSONIfExists(apiSecurityStatePath(configPath, "api-operations.json"), &state); err != nil {
		return err
	}
	if err := validateAPISecurityStateVersion("api-operations", state.Version); err != nil {
		return err
	}
	if len(state.Ops) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range state.Ops {
		op := cloneAPIOperation(state.Ops[i])
		if op.ID == "" {
			op.ID = apiOperationID(op.Site, op.Method, op.Path)
		}
		if op.Fingerprint == "" {
			op.Fingerprint = operationFingerprint(op.Method, op.Path)
		}
		if op.Status == nil {
			op.Status = map[string]int64{}
		}
		if op.ContentTypes == nil {
			op.ContentTypes = map[string]int64{}
		}
		if op.AuthObserved == nil {
			op.AuthObserved = map[string]int64{}
		}
		s.ops[op.ID] = &op
	}
	return nil
}

func (s *schemaStore) save(configPath string) error {
	if s == nil {
		return nil
	}
	state := s.persistenceSnapshot()
	state.Version = apiSecurityStateVersion
	state.Saved = time.Now().UTC()
	return atomicWriteJSON(apiSecurityStatePath(configPath, "api-schema.json"), state)
}

func (s *schemaStore) load(configPath string) error {
	if s == nil {
		return nil
	}
	var state schemaStateFile
	if err := readJSONIfExists(apiSecurityStatePath(configPath, "api-schema.json"), &state); err != nil {
		return err
	}
	if err := validateAPISecurityStateVersion("api-schema", state.Version); err != nil {
		return err
	}
	return s.restorePersistence(state)
}

func (s *contractStore) save(configPath string) error {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	state := contractStateFile{
		Version:          apiSecurityStateVersion,
		Saved:            time.Now().UTC(),
		Contracts:        cloneMap(s.contracts),
		Versions:         cloneMap(s.versions),
		ContractVersions: cloneStringSliceMap(s.contractVersions),
		Operations:       cloneContractOperationsMap(s.operations),
		Bindings:         cloneBindingsMap(s.bindings),
		Diffs:            cloneDiffsMap(s.diffs),
	}
	s.mu.RUnlock()
	return atomicWriteJSON(apiSecurityStatePath(configPath, "api-contracts.json"), state)
}

func (s *contractStore) load(configPath string) error {
	if s == nil {
		return nil
	}
	var state contractStateFile
	if err := readJSONIfExists(apiSecurityStatePath(configPath, "api-contracts.json"), &state); err != nil {
		return err
	}
	if err := validateAPISecurityStateVersion("api-contracts", state.Version); err != nil {
		return err
	}
	if len(state.Contracts) == 0 && len(state.Versions) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.contracts = state.Contracts
	s.versions = state.Versions
	s.contractVersions = state.ContractVersions
	s.operations = state.Operations
	s.bindings = state.Bindings
	s.diffs = state.Diffs
	if s.contracts == nil {
		s.contracts = map[string]APIContract{}
	}
	if s.versions == nil {
		s.versions = map[string]APIContractVersion{}
	}
	if s.contractVersions == nil {
		s.contractVersions = map[string][]string{}
	}
	if s.operations == nil {
		s.operations = map[string][]ContractOperation{}
	}
	if s.bindings == nil {
		s.bindings = map[string][]OperationBinding{}
	}
	if s.diffs == nil {
		s.diffs = map[string][]ContractDiff{}
	}
	return nil
}

func cloneMap[K comparable, V any](in map[K]V) map[K]V {
	out := make(map[K]V, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneStringSliceMap(in map[string][]string) map[string][]string {
	out := make(map[string][]string, len(in))
	for k, v := range in {
		out[k] = append([]string(nil), v...)
	}
	return out
}

func cloneContractOperationsMap(in map[string][]ContractOperation) map[string][]ContractOperation {
	out := make(map[string][]ContractOperation, len(in))
	for k, v := range in {
		out[k] = append([]ContractOperation(nil), v...)
	}
	return out
}

func cloneBindingsMap(in map[string][]OperationBinding) map[string][]OperationBinding {
	out := make(map[string][]OperationBinding, len(in))
	for k, v := range in {
		out[k] = append([]OperationBinding(nil), v...)
	}
	return out
}

func cloneDiffsMap(in map[string][]ContractDiff) map[string][]ContractDiff {
	out := make(map[string][]ContractDiff, len(in))
	for k, v := range in {
		out[k] = append([]ContractDiff(nil), v...)
	}
	return out
}

func startAPISecurityAutosave(configPath string, every time.Duration, stop <-chan struct{}, wg *sync.WaitGroup, ops *apiOperationStore, schema *schemaStore, contracts *contractStore, positive *positiveSchemaStore, identity *apiIdentityStore, sequence *sequenceStore, objectLocators *objectLocatorStore, objectRelationships *objectRelationshipStore, bolaCandidates *bolaDetectionStore, bolaPolicy *bolaPolicyStore, graphql *graphqlStore) {
	if every <= 0 {
		every = time.Minute
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		t := time.NewTicker(every)
		defer t.Stop()
		save := func() {
			if err := errors.Join(ops.save(configPath), schema.save(configPath), contracts.save(configPath), positive.save(configPath), identity.save(configPath), sequence.save(configPath), objectLocators.save(configPath), objectRelationships.save(configPath), bolaCandidates.save(configPath), bolaPolicy.save(configPath), graphql.save(configPath)); err != nil {
				slog.Error("API security state persistence failed", "err", err)
			}
		}
		for {
			select {
			case <-t.C:
				save()
			case <-stop:
				save()
				return
			}
		}
	}()
}
