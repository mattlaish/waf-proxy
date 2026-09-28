package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	objectLocatorStateVersion     = 1
	objectLocatorDefaultTTL       = 30 * 24 * time.Hour
	objectLocatorMaxLocators      = 4096
	objectLocatorMaxPerOperation  = 32
	objectLocatorMaxFingerprints  = 8
	objectLocatorMaxOverrides     = 1024
	objectLocatorSourceAPI1       = "API1_NORMALIZED_PATH"
	objectLocatorSourceAPI2       = "API2_TYPED_SCHEMA"
	objectLocatorSourceAPI3       = "API3_OPENAPI_CONTRACT"
	objectLocatorSourceOperator   = "OPERATOR"
	objectLocatorSourceAPI8       = "API8_GRAPHQL_VARIABLE"
	objectLocatorOverrideInclude  = "INCLUDE"
	objectLocatorOverrideSuppress = "SUPPRESS"
	objectLocatorStatusActive     = "ACTIVE"
	objectLocatorStatusSuppressed = "SUPPRESSED"
)

// ObjectValueFingerprint is bounded keyed evidence that a locator has observed
// values. The raw object value is never copied into this durable model.
type ObjectValueFingerprint struct {
	Digest    string    `json:"digest"`
	Count     uint64    `json:"count"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
}

// ObjectLocator is the API-7.1 discovery primitive. It identifies where an
// object selector exists in a normalized API-1 operation without making any
// statement about ownership or authorization.
type ObjectLocator struct {
	ID                string                   `json:"id"`
	OperationID       string                   `json:"operation_id"`
	Location          string                   `json:"location"` // path | query | body
	Field             string                   `json:"field"`
	SchemaType        string                   `json:"schema_type,omitempty"`
	SemanticName      string                   `json:"semantic_name"`
	Confidence        float64                  `json:"confidence"`
	Sources           []string                 `json:"sources"`
	Status            string                   `json:"status"`
	ObservationCount  uint64                   `json:"observation_count"`
	ValueFingerprints []ObjectValueFingerprint `json:"value_fingerprints,omitempty"`
	FirstSeen         time.Time                `json:"first_seen"`
	LastSeen          time.Time                `json:"last_seen"`
	ExpiresAt         time.Time                `json:"expires_at"`
}

// ObjectLocatorOverride is operator configuration, not an ownership policy.
// INCLUDE explicitly declares a locator. SUPPRESS hides a false-positive
// discovery. Neither action can authorize or block a request.
type ObjectLocatorOverride struct {
	ID           string    `json:"id"`
	OperationID  string    `json:"operation_id"`
	Location     string    `json:"location"`
	Field        string    `json:"field"`
	SchemaType   string    `json:"schema_type,omitempty"`
	SemanticName string    `json:"semantic_name,omitempty"`
	Action       string    `json:"action"` // INCLUDE | SUPPRESS
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type objectLocatorStateFile struct {
	Version   int                     `json:"version"`
	Saved     time.Time               `json:"saved"`
	Locators  []ObjectLocator         `json:"locators"`
	Overrides []ObjectLocatorOverride `json:"overrides"`
}

type objectLocatorStore struct {
	mu        sync.RWMutex
	keyMu     sync.RWMutex
	key       []byte
	locators  map[string]*ObjectLocator
	overrides map[string]ObjectLocatorOverride
	now       func() time.Time
}

func newObjectLocatorStore() *objectLocatorStore {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic("object locator fingerprint key generation failed: " + err.Error())
	}
	return newObjectLocatorStoreWithKey(key)
}

func newObjectLocatorStoreWithKey(key []byte) *objectLocatorStore {
	if len(key) != 32 {
		panic("object locator fingerprint key must be exactly 32 bytes")
	}
	return &objectLocatorStore{
		key:       append([]byte(nil), key...),
		locators:  map[string]*ObjectLocator{},
		overrides: map[string]ObjectLocatorOverride{},
		now:       func() time.Time { return time.Now().UTC() },
	}
}

func objectLocatorID(operationID, location, field string) string {
	h := sha256.Sum256([]byte("object-locator|" + operationID + "|" + strings.ToLower(location) + "|" + field))
	return hex.EncodeToString(h[:16])
}

func objectLocatorOverrideID(operationID, location, field string) string {
	h := sha256.Sum256([]byte("object-locator-override|" + operationID + "|" + strings.ToLower(location) + "|" + field))
	return hex.EncodeToString(h[:16])
}

func validObjectLocatorLocation(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "path", "query", "body", "graphql_variable":
		return true
	default:
		return false
	}
}

func validObjectLocatorField(v string) bool {
	v = strings.TrimSpace(v)
	if v == "" || len(v) > 256 {
		return false
	}
	for _, r := range v {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

func validObjectLocator(v ObjectLocator, now time.Time) bool {
	if len(v.ID) != 32 || !validSequenceOperationID(v.OperationID) || !validObjectLocatorLocation(v.Location) ||
		!validObjectLocatorField(v.Field) || v.Confidence < 0 || v.Confidence > 1 || v.FirstSeen.IsZero() || v.LastSeen.IsZero() || !v.ExpiresAt.After(now) ||
		len(v.Sources) == 0 || len(v.ValueFingerprints) > objectLocatorMaxFingerprints {
		return false
	}
	if _, err := hex.DecodeString(v.ID); err != nil {
		return false
	}
	for _, fp := range v.ValueFingerprints {
		if len(fp.Digest) != 40 {
			return false
		}
		if _, err := hex.DecodeString(fp.Digest); err != nil {
			return false
		}
	}
	return true
}

func validObjectLocatorOverride(v ObjectLocatorOverride) bool {
	if len(v.ID) != 32 || !validSequenceOperationID(v.OperationID) || !validObjectLocatorLocation(v.Location) ||
		!validObjectLocatorField(v.Field) || (v.Action != objectLocatorOverrideInclude && v.Action != objectLocatorOverrideSuppress) {
		return false
	}
	if _, err := hex.DecodeString(v.ID); err != nil {
		return false
	}
	return v.Action != objectLocatorOverrideInclude || (strings.TrimSpace(v.SemanticName) != "" && len(v.SemanticName) <= 128)
}

func objectSemanticName(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || path == "$" || strings.HasPrefix(strings.ToLower(path), "variables.") || isSensitiveFieldName(path) {
		return ""
	}
	parts := strings.FieldsFunc(path, func(r rune) bool { return r == '.' || r == '[' || r == ']' || r == '/' })
	if len(parts) == 0 {
		return ""
	}
	leaf := strings.TrimSpace(parts[len(parts)-1])
	canonical := strings.ToLower(strings.NewReplacer("_", "", "-", "").Replace(leaf))
	if canonical == "id" || canonical == "uuid" || (len(canonical) > 2 && strings.HasSuffix(canonical, "id")) {
		return leaf
	}
	return ""
}

func objectLocatorTypeEligible(typ, format string) bool {
	typ = strings.ToLower(strings.TrimSpace(typ))
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "uuid" || format == "ulid" {
		return true
	}
	switch typ {
	case "integer", "number", "string":
		return true
	default:
		return false
	}
}

func (s *objectLocatorStore) digestValue(operationID, location, field, raw string) string {
	if s == nil || raw == "" {
		return ""
	}
	s.keyMu.RLock()
	key := append([]byte(nil), s.key...)
	s.keyMu.RUnlock()
	h := hmac.New(sha256.New, key)
	_, _ = h.Write([]byte(operationID + "\x00" + location + "\x00" + field + "\x00" + raw))
	return hex.EncodeToString(h.Sum(nil)[:20])
}

func appendLocatorSource(sources []string, source string) []string {
	for _, v := range sources {
		if v == source {
			return sources
		}
	}
	return append(sources, source)
}

func locatorConfidence(sources []string) float64 {
	best := 0.0
	for _, source := range sources {
		var v float64
		switch source {
		case objectLocatorSourceOperator:
			v = 1.0
		case objectLocatorSourceAPI3:
			v = 0.95
		case objectLocatorSourceAPI1:
			v = 0.85
		case objectLocatorSourceAPI2:
			v = 0.75
		case objectLocatorSourceAPI8:
			v = 0.95
		}
		if v > best {
			best = v
		}
	}
	return best
}

func (s *objectLocatorStore) countOperationLocked(operationID string) int {
	n := 0
	for _, v := range s.locators {
		if v.OperationID == operationID {
			n++
		}
	}
	return n
}

func (s *objectLocatorStore) mergeLocked(operationID, location, field, schemaType, semanticName, source, rawValue string, at time.Time) *ObjectLocator {
	location = strings.ToLower(strings.TrimSpace(location))
	field = strings.TrimSpace(field)
	semanticName = strings.TrimSpace(semanticName)
	if !validSequenceOperationID(operationID) || !validObjectLocatorLocation(location) || !validObjectLocatorField(field) || semanticName == "" {
		return nil
	}
	id := objectLocatorID(operationID, location, field)
	v := s.locators[id]
	if v == nil {
		if len(s.locators) >= objectLocatorMaxLocators || s.countOperationLocked(operationID) >= objectLocatorMaxPerOperation {
			return nil
		}
		v = &ObjectLocator{ID: id, OperationID: operationID, Location: location, Field: field, SemanticName: semanticName, FirstSeen: at.UTC()}
		s.locators[id] = v
	}
	if schemaType != "" {
		v.SchemaType = strings.ToLower(strings.TrimSpace(schemaType))
	}
	if v.SemanticName == "" || source == objectLocatorSourceAPI3 || source == objectLocatorSourceOperator {
		v.SemanticName = semanticName
	}
	v.Sources = appendLocatorSource(v.Sources, source)
	sort.Strings(v.Sources)
	v.Confidence = locatorConfidence(v.Sources)
	v.Status = objectLocatorStatusActive
	v.ObservationCount++
	v.LastSeen = at.UTC()
	v.ExpiresAt = at.UTC().Add(objectLocatorDefaultTTL)
	if digest := s.digestValue(operationID, location, field, rawValue); digest != "" {
		found := false
		for i := range v.ValueFingerprints {
			if v.ValueFingerprints[i].Digest == digest {
				v.ValueFingerprints[i].Count++
				v.ValueFingerprints[i].LastSeen = at.UTC()
				found = true
				break
			}
		}
		if !found && len(v.ValueFingerprints) < objectLocatorMaxFingerprints {
			v.ValueFingerprints = append(v.ValueFingerprints, ObjectValueFingerprint{Digest: digest, Count: 1, FirstSeen: at.UTC(), LastSeen: at.UTC()})
		}
	}
	return v
}

func (s *objectLocatorStore) noteObservation(op apiOperation, samples []schemaFieldSample) {
	if s == nil || op.ID == "" || op.Ignored {
		return
	}
	at := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(at)

	// API-1 path parameters are normalized before they become locator nodes.
	for _, p := range op.PathParameters {
		if p.Kind != "identifier" && p.Kind != "opaque_token" {
			continue
		}
		field := p.Name
		semantic := objectSemanticName(field)
		if semantic == "" {
			semantic = field
		}
		var raw string
		for _, sample := range samples {
			if sample.Location == "path" && sample.Path == field {
				raw = sample.objectValue
				break
			}
		}
		s.mergeLocked(op.ID, "path", field, p.Kind, semantic, objectLocatorSourceAPI1, raw, at)
	}

	// API-2 contributes typed query/body candidates. Headers/cookies are never
	// object locators in API-7.1, and GraphQL variables are deferred to API-8.
	for _, sample := range samples {
		if sample.Location != "path" && sample.Location != "query" && sample.Location != "body" {
			continue
		}
		if sample.Location == "body" && strings.HasPrefix(strings.ToLower(sample.Path), "variables.") {
			continue
		}
		semantic := objectSemanticName(sample.Path)
		if sample.Location == "path" && semantic == "" {
			semantic = sample.Path
		}
		if semantic == "" || !objectLocatorTypeEligible(sample.Type, sample.Format) {
			continue
		}
		s.mergeLocked(op.ID, sample.Location, sample.Path, sample.Type, semantic, objectLocatorSourceAPI2, sample.objectValue, at)
	}
}

func (s *objectLocatorStore) pruneLocked(now time.Time) {
	for id, v := range s.locators {
		if v == nil || !v.ExpiresAt.After(now) {
			if v != nil && (containsLocatorSource(v.Sources, objectLocatorSourceOperator) || containsLocatorSource(v.Sources, objectLocatorSourceAPI3)) {
				continue
			}
			delete(s.locators, id)
		}
	}
}

func containsLocatorSource(in []string, source string) bool {
	for _, v := range in {
		if v == source {
			return true
		}
	}
	return false
}

func cloneObjectLocator(v ObjectLocator) ObjectLocator {
	v.Sources = append([]string(nil), v.Sources...)
	v.ValueFingerprints = append([]ObjectValueFingerprint(nil), v.ValueFingerprints...)
	return v
}

func (s *objectLocatorStore) snapshot() []ObjectLocator {
	if s == nil {
		return []ObjectLocator{}
	}
	now := s.now()
	s.mu.Lock()
	s.pruneLocked(now)
	out := make([]ObjectLocator, 0, len(s.locators))
	for _, p := range s.locators {
		v := cloneObjectLocator(*p)
		if ov, ok := s.overrides[objectLocatorOverrideID(v.OperationID, v.Location, v.Field)]; ok && ov.Action == objectLocatorOverrideSuppress {
			v.Status = objectLocatorStatusSuppressed
		} else {
			v.Status = objectLocatorStatusActive
		}
		out = append(out, v)
	}
	s.mu.Unlock()
	sort.Slice(out, func(i, j int) bool {
		if out[i].OperationID != out[j].OperationID {
			return out[i].OperationID < out[j].OperationID
		}
		if out[i].Location != out[j].Location {
			return out[i].Location < out[j].Location
		}
		return out[i].Field < out[j].Field
	})
	return out
}

func (s *objectLocatorStore) listOverrides() []ObjectLocatorOverride {
	if s == nil {
		return []ObjectLocatorOverride{}
	}
	s.mu.RLock()
	out := make([]ObjectLocatorOverride, 0, len(s.overrides))
	for _, v := range s.overrides {
		out = append(out, v)
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *objectLocatorStore) removeOperatorSourceLocked(operationID, location, field string) {
	locatorID := objectLocatorID(operationID, location, field)
	loc := s.locators[locatorID]
	if loc == nil {
		return
	}
	filtered := loc.Sources[:0]
	for _, source := range loc.Sources {
		if source != objectLocatorSourceOperator {
			filtered = append(filtered, source)
		}
	}
	loc.Sources = filtered
	loc.Confidence = locatorConfidence(loc.Sources)
	if len(loc.Sources) == 0 {
		delete(s.locators, locatorID)
	}
}

func (s *objectLocatorStore) upsertOverride(v ObjectLocatorOverride) (ObjectLocatorOverride, error) {
	if s == nil {
		return ObjectLocatorOverride{}, errors.New("object locator store unavailable")
	}
	v.OperationID = strings.TrimSpace(v.OperationID)
	v.Location = strings.ToLower(strings.TrimSpace(v.Location))
	v.Field = strings.TrimSpace(v.Field)
	v.SchemaType = strings.ToLower(strings.TrimSpace(v.SchemaType))
	v.SemanticName = strings.TrimSpace(v.SemanticName)
	v.Action = strings.ToUpper(strings.TrimSpace(v.Action))
	if v.Action != objectLocatorOverrideInclude && v.Action != objectLocatorOverrideSuppress {
		return ObjectLocatorOverride{}, errors.New("action must be INCLUDE or SUPPRESS")
	}
	if !validSequenceOperationID(v.OperationID) || !validObjectLocatorLocation(v.Location) || !validObjectLocatorField(v.Field) {
		return ObjectLocatorOverride{}, errors.New("valid normalized operation_id, location, and field are required")
	}
	if v.Action == objectLocatorOverrideInclude {
		if v.SemanticName == "" {
			v.SemanticName = objectSemanticName(v.Field)
		}
		if v.SemanticName == "" || len(v.SemanticName) > 128 {
			return ObjectLocatorOverride{}, errors.New("INCLUDE requires a bounded semantic_name")
		}
	}
	id := objectLocatorOverrideID(v.OperationID, v.Location, v.Field)
	now := s.now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	if old, ok := s.overrides[id]; ok {
		v.CreatedAt = old.CreatedAt
		if old.Action == objectLocatorOverrideInclude && v.Action != objectLocatorOverrideInclude {
			s.removeOperatorSourceLocked(old.OperationID, old.Location, old.Field)
		}
	} else {
		if len(s.overrides) >= objectLocatorMaxOverrides {
			return ObjectLocatorOverride{}, errors.New("maximum object locator overrides reached")
		}
		v.CreatedAt = now
	}
	v.ID = id
	v.UpdatedAt = now
	s.overrides[id] = v
	if v.Action == objectLocatorOverrideInclude {
		s.mergeLocked(v.OperationID, v.Location, v.Field, v.SchemaType, v.SemanticName, objectLocatorSourceOperator, "", now)
	}
	return v, nil
}

func (s *objectLocatorStore) deleteOverride(id string) (ObjectLocatorOverride, error) {
	if s == nil {
		return ObjectLocatorOverride{}, errors.New("object locator store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.overrides[id]
	if !ok {
		return ObjectLocatorOverride{}, errors.New("object locator override not found")
	}
	delete(s.overrides, id)
	if v.Action == objectLocatorOverrideInclude {
		s.removeOperatorSourceLocked(v.OperationID, v.Location, v.Field)
	}
	return v, nil
}

type contractLocatorCandidate struct {
	OperationID  string
	Location     string
	Field        string
	SchemaType   string
	SemanticName string
}

func (s *contractStore) objectLocatorCandidates() []contractLocatorCandidate {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []contractLocatorCandidate
	for versionID, bindings := range s.bindings {
		ops := s.operations[versionID]
		byID := make(map[string]ContractOperation, len(ops))
		for _, op := range ops {
			byID[op.ID] = op
		}
		for _, binding := range bindings {
			if binding.Status != "MATCHED" || !validSequenceOperationID(binding.APIOperationID) {
				continue
			}
			op, ok := byID[binding.ContractOperationID]
			if !ok {
				continue
			}
			for _, p := range op.Parameters {
				if p.In != "path" && p.In != "query" {
					continue
				}
				semantic := objectSemanticName(p.Name)
				if p.In == "path" && semantic == "" {
					semantic = p.Name
				}
				if semantic == "" || (p.In != "path" && !objectLocatorTypeEligible(p.Type, p.Format)) {
					continue
				}
				out = append(out, contractLocatorCandidate{OperationID: binding.APIOperationID, Location: p.In, Field: p.Name, SchemaType: p.Type, SemanticName: semantic})
			}
			for _, f := range op.RequestFields {
				if strings.HasPrefix(strings.ToLower(f.Path), "variables.") {
					continue
				}
				semantic := objectSemanticName(f.Path)
				if semantic == "" || !objectLocatorTypeEligible(f.Type, f.Format) {
					continue
				}
				out = append(out, contractLocatorCandidate{OperationID: binding.APIOperationID, Location: "body", Field: f.Path, SchemaType: f.Type, SemanticName: semantic})
			}
		}
	}
	return out
}

func (s *objectLocatorStore) refreshContractSources(contracts *contractStore) {
	if s == nil || contracts == nil {
		return
	}
	candidates := contracts.objectLocatorCandidates()
	if len(candidates) == 0 {
		return
	}
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range candidates {
		s.mergeLocked(c.OperationID, c.Location, c.Field, c.SchemaType, c.SemanticName, objectLocatorSourceAPI3, "", now)
	}
}

// noteGraphQLVariable is API-8's bounded bridge into the API-7.1 locator
// inventory. The raw variable value is used only to derive the existing keyed
// fingerprint and is never copied into durable state.
func (s *objectLocatorStore) noteGraphQLVariable(operationID, field, schemaType, rawValue string, at time.Time) (ObjectLocator, bool) {
	if s == nil || !validSequenceOperationID(operationID) || !validGraphQLName(field) || rawValue == "" {
		return ObjectLocator{}, false
	}
	semantic := graphqlVariableSemantic(field, schemaType)
	if semantic == "" {
		return ObjectLocator{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(at)
	v := s.mergeLocked(operationID, "graphql_variable", field, schemaType, semantic, objectLocatorSourceAPI8, rawValue, at)
	if v == nil {
		return ObjectLocator{}, false
	}
	cp := cloneObjectLocator(*v)
	if ov, ok := s.overrides[objectLocatorOverrideID(operationID, "graphql_variable", field)]; ok && ov.Action == objectLocatorOverrideSuppress {
		return ObjectLocator{}, false
	}
	return cp, true
}

func (s *objectLocatorStore) activeLocator(operationID, location, field string) (ObjectLocator, bool) {
	if s == nil {
		return ObjectLocator{}, false
	}
	location = strings.ToLower(strings.TrimSpace(location))
	field = strings.TrimSpace(field)
	id := objectLocatorID(operationID, location, field)
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(now)
	v := s.locators[id]
	if v == nil {
		return ObjectLocator{}, false
	}
	if ov, ok := s.overrides[objectLocatorOverrideID(operationID, location, field)]; ok && ov.Action == objectLocatorOverrideSuppress {
		return ObjectLocator{}, false
	}
	cp := cloneObjectLocator(*v)
	cp.Status = objectLocatorStatusActive
	return cp, true
}

func (s *objectLocatorStore) ensureFingerprintKey(configPath string) error {
	path := apiSecurityStatePath(configPath, "api-object-locator.key")
	info, err := os.Lstat(path)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return errors.New("api-object-locator.key must be a regular file")
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if len(b) != 32 {
			return errors.New("api-object-locator.key must contain exactly 32 bytes")
		}
		s.keyMu.Lock()
		s.key = append(s.key[:0], b...)
		s.keyMu.Unlock()
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	s.keyMu.RLock()
	key := append([]byte(nil), s.key...)
	s.keyMu.RUnlock()
	if len(key) != 32 {
		return errors.New("object locator fingerprint key unavailable")
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".api-object-locator.key.tmp-*")
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
	if err := f.Chmod(0o600); err != nil {
		return err
	}
	if _, err := f.Write(key); err != nil {
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
	return nil
}

func (s *objectLocatorStore) save(configPath string) error {
	if s == nil {
		return nil
	}
	if err := s.ensureFingerprintKey(configPath); err != nil {
		return err
	}
	state := objectLocatorStateFile{Version: objectLocatorStateVersion, Saved: time.Now().UTC(), Locators: s.snapshot(), Overrides: s.listOverrides()}
	return atomicWriteJSON(apiSecurityStatePath(configPath, "api-object-locators.json"), state)
}

func (s *objectLocatorStore) load(configPath string) error {
	if s == nil {
		return nil
	}
	if err := s.ensureFingerprintKey(configPath); err != nil {
		return err
	}
	var state objectLocatorStateFile
	if err := readJSONIfExists(apiSecurityStatePath(configPath, "api-object-locators.json"), &state); err != nil {
		return err
	}
	if state.Version != 0 && state.Version != objectLocatorStateVersion {
		return fmt.Errorf("unsupported object locator state version %d", state.Version)
	}
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, v := range state.Locators {
		if len(s.locators) >= objectLocatorMaxLocators || !validObjectLocator(v, now) || s.countOperationLocked(v.OperationID) >= objectLocatorMaxPerOperation {
			continue
		}
		cp := cloneObjectLocator(v)
		s.locators[v.ID] = &cp
	}
	for _, v := range state.Overrides {
		if len(s.overrides) >= objectLocatorMaxOverrides || !validObjectLocatorOverride(v) {
			continue
		}
		s.overrides[v.ID] = v
		if v.Action == objectLocatorOverrideInclude {
			s.mergeLocked(v.OperationID, v.Location, v.Field, v.SchemaType, v.SemanticName, objectLocatorSourceOperator, "", now)
		}
	}
	return nil
}

func (a *adminServer) persistObjectLocatorMutation(w http.ResponseWriter) bool {
	if a == nil || a.srv == nil || a.srv.objectLocators == nil {
		http.Error(w, "object locator store unavailable", http.StatusServiceUnavailable)
		return false
	}
	if err := a.srv.objectLocators.save(a.srv.configPath); err != nil {
		http.Error(w, "object locator persistence failed: "+err.Error(), http.StatusInternalServerError)
		return false
	}
	return true
}

func (a *adminServer) handleObjectLocators(w http.ResponseWriter, r *http.Request) {
	a.srv.objectLocators.refreshContractSources(a.srv.contracts)
	rows := a.srv.objectLocators.snapshot()
	a.audit.add(who(r).user, "api_object_locator.view", fmt.Sprintf("count=%d", len(rows)))
	writeJSON(w, rows)
}

func (a *adminServer) handleObjectLocatorOverrides(w http.ResponseWriter, r *http.Request) {
	rows := a.srv.objectLocators.listOverrides()
	a.audit.add(who(r).user, "api_object_locator.override_view", fmt.Sprintf("count=%d", len(rows)))
	writeJSON(w, rows)
}

func (a *adminServer) handleObjectLocatorOverrideUpsert(w http.ResponseWriter, r *http.Request) {
	var req ObjectLocatorOverride
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<14)).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	v, err := a.srv.objectLocators.upsertOverride(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	if !a.persistObjectLocatorMutation(w) {
		return
	}
	a.audit.add(who(r).user, "api_object_locator.override_upsert", v.OperationID+" "+v.Location+" "+v.Field+" "+v.Action)
	writeJSON(w, v)
}

func (a *adminServer) handleObjectLocatorOverrideDelete(w http.ResponseWriter, r *http.Request) {
	v, err := a.srv.objectLocators.deleteOverride(r.PathValue("override_id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if !a.persistObjectLocatorMutation(w) {
		return
	}
	a.audit.add(who(r).user, "api_object_locator.override_delete", v.OperationID+" "+v.Location+" "+v.Field)
	writeJSON(w, map[string]any{"ok": true, "id": v.ID})
}
