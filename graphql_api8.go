package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	graphqlStateVersion          = 1
	graphqlQueueCapacity         = 1024
	graphqlMaxOperations         = 4096
	graphqlMaxPersistedQueries   = 4096
	graphqlMaxPolicies           = 1024
	graphqlMaxSchemaContracts    = 128
	graphqlMaxSchemaBytes        = 256 << 10
	graphqlMaxSchemaTypes        = 1024
	graphqlMaxSchemaFields       = 8192
	graphqlMaxSchemaTokens       = 32768
	graphqlMaxViolations         = 4096
	graphqlMaxQueryBytes         = 64 << 10
	graphqlMaxTokens             = 8192
	graphqlAbsoluteMaxDepth      = 64
	graphqlAbsoluteMaxComplexity = 10000
	graphqlMaxFields             = 2048
	graphqlMaxVariables          = 256
	graphqlMaxPolicyFields       = 512
	graphqlMaxPolicyReason       = 512
	graphqlPersistedTTL          = 30 * 24 * time.Hour
	graphqlOperationTTL          = 30 * 24 * time.Hour

	graphqlModeLearn   = "LEARN"
	graphqlModeDetect  = "DETECT"
	graphqlModeEnforce = "ENFORCE"
)

var errGraphQLSchemaContractInUse = errors.New("GraphQL schema contract is referenced by policy")

// GraphQLOperation is bounded, value-free API-8 discovery state. It stores
// normalized structure and counters, never the raw GraphQL document or variable
// values. Literal values are discarded by the parser before this model exists.
type GraphQLOperation struct {
	ID                  string    `json:"id"`
	EndpointOperationID string    `json:"endpoint_operation_id"`
	Site                string    `json:"site"`
	OperationType       string    `json:"operation_type"`
	OperationName       string    `json:"operation_name,omitempty"`
	DocumentFingerprint string    `json:"document_fingerprint"`
	Depth               int       `json:"depth"`
	Complexity          int       `json:"complexity"`
	Fields              []string  `json:"fields,omitempty"`
	Variables           []string  `json:"variables,omitempty"`
	Introspection       bool      `json:"introspection"`
	PersistedQueryHash  string    `json:"persisted_query_hash,omitempty"`
	ObservationCount    uint64    `json:"observation_count"`
	FirstSeen           time.Time `json:"first_seen"`
	LastSeen            time.Time `json:"last_seen"`
	ExpiresAt           time.Time `json:"expires_at"`
}

// GraphQLPersistedProfile stores only pre-computed structural metadata. The raw
// persisted query text and literal values are intentionally not durable.
type GraphQLPersistedProfile struct {
	SHA256Hash          string    `json:"sha256_hash"`
	GraphQLOperationID  string    `json:"graphql_operation_id"`
	EndpointOperationID string    `json:"endpoint_operation_id"`
	OperationType       string    `json:"operation_type"`
	OperationName       string    `json:"operation_name,omitempty"`
	DocumentFingerprint string    `json:"document_fingerprint"`
	Depth               int       `json:"depth"`
	Complexity          int       `json:"complexity"`
	Fields              []string  `json:"fields,omitempty"`
	Variables           []string  `json:"variables,omitempty"`
	Introspection       bool      `json:"introspection"`
	FirstSeen           time.Time `json:"first_seen"`
	LastSeen            time.Time `json:"last_seen"`
	ExpiresAt           time.Time `json:"expires_at"`
}

// GraphQLSchemaContract is an operator-imported bounded SDL contract. Only
// type/field topology and the source digest are durable; the raw SDL document
// is not retained after import.
type GraphQLSchemaContract struct {
	ID                  string                       `json:"id"`
	Name                string                       `json:"name"`
	EndpointOperationID string                       `json:"endpoint_operation_id"`
	SourceSHA256        string                       `json:"source_sha256"`
	QueryRoot           string                       `json:"query_root"`
	MutationRoot        string                       `json:"mutation_root,omitempty"`
	SubscriptionRoot    string                       `json:"subscription_root,omitempty"`
	Types               map[string][]string          `json:"types"`
	FieldTypes          map[string]map[string]string `json:"field_types"`
	CreatedAt           time.Time                    `json:"created_at"`
	UpdatedAt           time.Time                    `json:"updated_at"`
	CreatedBy           string                       `json:"created_by,omitempty"`
}

// GraphQLPolicy is explicit deterministic API-8 policy. Unlike API-6/API-7
// learned evidence, an operator-reviewed GraphQL policy may enter ENFORCE after
// passing through DETECT. The policy is based only on normalized structure.
type GraphQLPolicy struct {
	ID                  string    `json:"id"`
	EndpointOperationID string    `json:"endpoint_operation_id"`
	OperationName       string    `json:"operation_name,omitempty"`
	SchemaContractID    string    `json:"schema_contract_id,omitempty"`
	Mode                string    `json:"mode"`
	MaxDepth            int       `json:"max_depth"`
	MaxComplexity       int       `json:"max_complexity"`
	AllowIntrospection  bool      `json:"allow_introspection"`
	AllowMutation       bool      `json:"allow_mutation"`
	AllowSubscription   bool      `json:"allow_subscription"`
	RequirePersisted    bool      `json:"require_persisted_queries"`
	AllowedFields       []string  `json:"allowed_fields,omitempty"`
	DeniedFields        []string  `json:"denied_fields,omitempty"`
	BlockStatus         int       `json:"block_status"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	CreatedBy           string    `json:"created_by,omitempty"`
	UpdatedBy           string    `json:"updated_by,omitempty"`
	Reason              string    `json:"reason,omitempty"`
}

type GraphQLViolation struct {
	ID                  string    `json:"id"`
	Time                time.Time `json:"time"`
	EndpointOperationID string    `json:"endpoint_operation_id"`
	GraphQLOperationID  string    `json:"graphql_operation_id,omitempty"`
	OperationName       string    `json:"operation_name,omitempty"`
	SchemaContractID    string    `json:"schema_contract_id,omitempty"`
	Mode                string    `json:"mode"`
	Action              string    `json:"action"`
	Type                string    `json:"type"`
	Expected            string    `json:"expected,omitempty"`
	Observed            string    `json:"observed,omitempty"`
	Field               string    `json:"field,omitempty"`
}

type GraphQLStatus struct {
	OperationCount        int    `json:"operation_count"`
	PersistedQueryCount   int    `json:"persisted_query_count"`
	SchemaContractCount   int    `json:"schema_contract_count"`
	PolicyCount           int    `json:"policy_count"`
	ViolationCount        int    `json:"violation_count"`
	QueueDepth            int    `json:"queue_depth"`
	QueueCapacity         int    `json:"queue_capacity"`
	DroppedObservations   uint64 `json:"dropped_observations"`
	ProcessedObservations uint64 `json:"processed_observations"`
	MaxOperations         int    `json:"max_operations"`
	MaxPersistedQueries   int    `json:"max_persisted_queries"`
	AbsoluteMaxDepth      int    `json:"absolute_max_depth"`
	AbsoluteMaxComplexity int    `json:"absolute_max_complexity"`
	Accepting             bool   `json:"accepting"`
}

type graphqlStateFile struct {
	Version    int                       `json:"version"`
	Saved      time.Time                 `json:"saved"`
	Operations []GraphQLOperation        `json:"operations"`
	Persisted  []GraphQLPersistedProfile `json:"persisted_queries"`
	Contracts  []GraphQLSchemaContract   `json:"schema_contracts"`
	Policies   []GraphQLPolicy           `json:"policies"`
	Violations []GraphQLViolation        `json:"violations,omitempty"`
}

type graphqlRuntimeSnapshot struct {
	Policies         map[string]GraphQLPolicy
	Contracts        map[string]GraphQLSchemaContract
	EndpointFallback map[string]GraphQLPolicy
	Persisted        map[string]GraphQLPersistedProfile
}

type graphqlVariableSample struct {
	Name       string
	SchemaType string
	RawValue   string
}

type graphqlObservation struct {
	Operation  GraphQLOperation
	Persisted  *GraphQLPersistedProfile
	Variables  []graphqlVariableSample
	Identity   objectRelationshipObservation
	ObservedAt time.Time
}

type graphqlStore struct {
	mu         sync.RWMutex
	operations map[string]*GraphQLOperation
	persisted  map[string]*GraphQLPersistedProfile
	contracts  map[string]*GraphQLSchemaContract
	policies   map[string]*GraphQLPolicy
	violations []GraphQLViolation
	now        func() time.Time

	runtime atomic.Pointer[graphqlRuntimeSnapshot]
	queue   chan graphqlObservation
	stop    chan struct{}
	done    chan struct{}

	accepting atomic.Bool
	dropped   atomic.Uint64
	processed atomic.Uint64
	startOnce sync.Once
	stopOnce  sync.Once

	objectLocators      *objectLocatorStore
	objectRelationships *objectRelationshipStore
}

func newGraphQLStore() *graphqlStore {
	s := &graphqlStore{
		operations: map[string]*GraphQLOperation{},
		persisted:  map[string]*GraphQLPersistedProfile{},
		contracts:  map[string]*GraphQLSchemaContract{},
		policies:   map[string]*GraphQLPolicy{},
		now:        func() time.Time { return time.Now().UTC() },
		queue:      make(chan graphqlObservation, graphqlQueueCapacity),
		stop:       make(chan struct{}),
		done:       make(chan struct{}),
	}
	s.runtime.Store(&graphqlRuntimeSnapshot{Policies: map[string]GraphQLPolicy{}, Contracts: map[string]GraphQLSchemaContract{}, EndpointFallback: map[string]GraphQLPolicy{}, Persisted: map[string]GraphQLPersistedProfile{}})
	return s
}

func (s *graphqlStore) start() {
	if s == nil {
		return
	}
	s.startOnce.Do(func() { s.accepting.Store(true); go s.run() })
}

func (s *graphqlStore) stopAndDrain() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() { s.accepting.Store(false); close(s.stop) })
	<-s.done
}

func (s *graphqlStore) run() {
	defer close(s.done)
	for {
		select {
		case ev := <-s.queue:
			s.processObservation(ev)
		case <-s.stop:
			for {
				select {
				case ev := <-s.queue:
					s.processObservation(ev)
				default:
					return
				}
			}
		}
	}
}

func (s *graphqlStore) enqueue(ev graphqlObservation) {
	if s == nil || !s.accepting.Load() || ev.Operation.ID == "" {
		return
	}
	select {
	case s.queue <- ev:
	default:
		s.dropped.Add(1)
	}
}

func (s *graphqlStore) processObservation(ev graphqlObservation) {
	if s == nil || ev.Operation.ID == "" {
		return
	}
	now := ev.ObservedAt.UTC()
	if now.IsZero() {
		now = s.now().UTC()
	}
	s.mu.Lock()
	s.pruneLocked(now)
	op := s.operations[ev.Operation.ID]
	if op == nil && len(s.operations) < graphqlMaxOperations {
		cp := cloneGraphQLOperation(ev.Operation)
		cp.ObservationCount = 0
		cp.FirstSeen = now
		op = &cp
		s.operations[cp.ID] = op
	}
	if op != nil {
		if op.ObservationCount < ^uint64(0) {
			op.ObservationCount++
		}
		op.LastSeen = now
		op.ExpiresAt = now.Add(graphqlOperationTTL)
		if op.PersistedQueryHash == "" {
			op.PersistedQueryHash = ev.Operation.PersistedQueryHash
		}
	}
	if ev.Persisted != nil && validGraphQLPersisted(*ev.Persisted, now) {
		p := s.persisted[ev.Persisted.SHA256Hash]
		if p == nil && len(s.persisted) < graphqlMaxPersistedQueries {
			cp := cloneGraphQLPersisted(*ev.Persisted)
			cp.FirstSeen = now
			p = &cp
			s.persisted[cp.SHA256Hash] = p
		}
		if p != nil {
			p.LastSeen = now
			p.ExpiresAt = now.Add(graphqlPersistedTTL)
		}
	}
	s.rebuildRuntimeLocked()
	s.mu.Unlock()

	// API-8.5 integration: GraphQL variable selectors are discovered as a
	// distinct API-8 source and only keyed values are retained. Relationship
	// evidence is enqueued after locator creation and uses API-5 verified
	// pseudonyms already computed on the request path; raw values never enter
	// API-7.2 durable state.
	for _, v := range ev.Variables {
		if s.objectLocators == nil || v.RawValue == "" {
			continue
		}
		locator, ok := s.objectLocators.noteGraphQLVariable(ev.Operation.ID, v.Name, v.SchemaType, v.RawValue, now)
		if !ok {
			continue
		}
		fp := s.objectLocators.digestValue(ev.Operation.ID, "graphql_variable", v.Name, v.RawValue)
		if fp == "" || s.objectRelationships == nil || ev.Identity.IdentityFingerprint == "" {
			continue
		}
		rel := ev.Identity
		rel.OperationID = ev.Operation.ID
		rel.At = now
		s.objectRelationships.enqueueKeyed(rel, locator, fp, objectRelationshipSourceAPI8)
	}
	s.processed.Add(1)
}

func cloneGraphQLOperation(v GraphQLOperation) GraphQLOperation {
	v.Fields = append([]string(nil), v.Fields...)
	v.Variables = append([]string(nil), v.Variables...)
	return v
}
func cloneGraphQLPersisted(v GraphQLPersistedProfile) GraphQLPersistedProfile {
	v.Fields = append([]string(nil), v.Fields...)
	v.Variables = append([]string(nil), v.Variables...)
	return v
}
func cloneGraphQLContract(v GraphQLSchemaContract) GraphQLSchemaContract {
	v.Types = make(map[string][]string, len(v.Types))
	for name, fields := range v.Types {
		v.Types[name] = append([]string(nil), fields...)
	}
	v.FieldTypes = make(map[string]map[string]string, len(v.FieldTypes))
	for typ, fields := range v.FieldTypes {
		cp := make(map[string]string, len(fields))
		for name, target := range fields {
			cp[name] = target
		}
		v.FieldTypes[typ] = cp
	}
	return v
}

func cloneGraphQLPolicy(v GraphQLPolicy) GraphQLPolicy {
	v.AllowedFields = append([]string(nil), v.AllowedFields...)
	v.DeniedFields = append([]string(nil), v.DeniedFields...)
	return v
}

func (s *graphqlStore) pruneLocked(now time.Time) {
	for id, v := range s.operations {
		if v == nil || !v.ExpiresAt.After(now) {
			delete(s.operations, id)
		}
	}
	for id, v := range s.persisted {
		if v == nil || !v.ExpiresAt.After(now) {
			delete(s.persisted, id)
		}
	}
}

func (s *graphqlStore) rebuildRuntimeLocked() {
	snap := &graphqlRuntimeSnapshot{Policies: make(map[string]GraphQLPolicy, len(s.policies)), Contracts: make(map[string]GraphQLSchemaContract, len(s.contracts)), EndpointFallback: map[string]GraphQLPolicy{}, Persisted: make(map[string]GraphQLPersistedProfile, len(s.persisted))}
	for id, p := range s.policies {
		cp := cloneGraphQLPolicy(*p)
		snap.Policies[id] = cp
		cur, ok := snap.EndpointFallback[p.EndpointOperationID]
		if !ok || graphqlFallbackPolicyBetter(cp, cur) {
			snap.EndpointFallback[p.EndpointOperationID] = cp
		}
	}
	for id, c := range s.contracts {
		snap.Contracts[id] = cloneGraphQLContract(*c)
	}
	for h, p := range s.persisted {
		snap.Persisted[h] = cloneGraphQLPersisted(*p)
	}
	s.runtime.Store(snap)
}

func graphqlPolicyID(endpointOperationID, operationName string) string {
	h := sha256.Sum256([]byte("api8-graphql-policy\x00" + endpointOperationID + "\x00" + operationName))
	return hex.EncodeToString(h[:16])
}
func graphqlOperationID(endpointOperationID, operationType, operationName, documentFingerprint string) string {
	h := sha256.Sum256([]byte("api8-graphql-operation\x00" + endpointOperationID + "\x00" + operationType + "\x00" + operationName + "\x00" + documentFingerprint))
	return hex.EncodeToString(h[:16])
}

func validGraphQLMode(v string) bool {
	return v == graphqlModeLearn || v == graphqlModeDetect || v == graphqlModeEnforce
}
func validGraphQLHash(v string) bool {
	if len(v) != 64 {
		return false
	}
	_, err := hex.DecodeString(v)
	return err == nil
}
func validGraphQLName(v string) bool {
	if v == "" || len(v) > 128 {
		return false
	}
	for i, r := range v {
		if !(r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || i > 0 && r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}
func validGraphQLFieldPath(v string) bool {
	if v == "" || len(v) > 512 {
		return false
	}
	parts := strings.Split(v, ".")
	if len(parts) < 2 || len(parts) > graphqlAbsoluteMaxDepth+1 {
		return false
	}
	for _, p := range parts {
		if !validGraphQLName(p) {
			return false
		}
	}
	return true
}
func validGraphQLOperation(v GraphQLOperation, now time.Time) bool {
	if !validSequenceOperationID(v.ID) || !validSequenceOperationID(v.EndpointOperationID) || v.ID != graphqlOperationID(v.EndpointOperationID, v.OperationType, v.OperationName, v.DocumentFingerprint) ||
		(v.OperationType != "query" && v.OperationType != "mutation" && v.OperationType != "subscription") ||
		(v.OperationName != "" && !validGraphQLName(v.OperationName)) || !validGraphQLHash(v.DocumentFingerprint) ||
		v.Depth < 0 || v.Depth > graphqlAbsoluteMaxDepth || v.Complexity < 0 || v.Complexity > graphqlAbsoluteMaxComplexity ||
		len(v.Fields) > graphqlMaxFields || len(v.Variables) > graphqlMaxVariables || v.ObservationCount == 0 ||
		v.FirstSeen.IsZero() || v.LastSeen.IsZero() || !v.ExpiresAt.After(now) {
		return false
	}
	for _, f := range v.Fields {
		if !validGraphQLFieldPath(f) {
			return false
		}
	}
	for _, name := range v.Variables {
		if !validGraphQLName(name) {
			return false
		}
	}
	return v.PersistedQueryHash == "" || validGraphQLHash(v.PersistedQueryHash)
}
func validGraphQLPersisted(v GraphQLPersistedProfile, now time.Time) bool {
	if !validGraphQLHash(v.SHA256Hash) || !validSequenceOperationID(v.GraphQLOperationID) || !validSequenceOperationID(v.EndpointOperationID) ||
		(v.OperationType != "query" && v.OperationType != "mutation" && v.OperationType != "subscription") || (v.OperationName != "" && !validGraphQLName(v.OperationName)) ||
		v.GraphQLOperationID != graphqlOperationID(v.EndpointOperationID, v.OperationType, v.OperationName, v.DocumentFingerprint) || !validGraphQLHash(v.DocumentFingerprint) || v.Depth < 0 || v.Depth > graphqlAbsoluteMaxDepth || v.Complexity < 0 || v.Complexity > graphqlAbsoluteMaxComplexity ||
		len(v.Fields) > graphqlMaxFields || len(v.Variables) > graphqlMaxVariables || v.FirstSeen.IsZero() || v.LastSeen.IsZero() || !v.ExpiresAt.After(now) {
		return false
	}
	return true
}
func validGraphQLPolicy(v GraphQLPolicy) bool {
	if len(v.ID) != 32 || v.ID != graphqlPolicyID(v.EndpointOperationID, v.OperationName) || !validSequenceOperationID(v.EndpointOperationID) || (v.OperationName != "" && !validGraphQLName(v.OperationName)) ||
		(v.SchemaContractID != "" && len(v.SchemaContractID) != 32) || !validGraphQLMode(v.Mode) || v.MaxDepth < 0 || v.MaxDepth > graphqlAbsoluteMaxDepth || v.MaxComplexity < 0 || v.MaxComplexity > graphqlAbsoluteMaxComplexity ||
		len(v.AllowedFields) > graphqlMaxPolicyFields || len(v.DeniedFields) > graphqlMaxPolicyFields || v.BlockStatus < 400 || v.BlockStatus > 499 || v.CreatedAt.IsZero() || v.UpdatedAt.IsZero() {
		return false
	}
	if _, err := hex.DecodeString(v.ID); err != nil {
		return false
	}
	if v.SchemaContractID != "" {
		if _, err := hex.DecodeString(v.SchemaContractID); err != nil {
			return false
		}
	}
	if len(v.Reason) > graphqlMaxPolicyReason {
		return false
	}
	for _, f := range append(append([]string(nil), v.AllowedFields...), v.DeniedFields...) {
		if !validGraphQLFieldPath(f) {
			return false
		}
	}
	return true
}

func validGraphQLViolation(v GraphQLViolation) bool {
	if len(v.ID) != 32 || v.Time.IsZero() || !validSequenceOperationID(v.EndpointOperationID) || !validGraphQLMode(v.Mode) {
		return false
	}
	if _, err := hex.DecodeString(v.ID); err != nil {
		return false
	}
	if v.GraphQLOperationID != "" && !validSequenceOperationID(v.GraphQLOperationID) {
		return false
	}
	if v.OperationName != "" && !validGraphQLName(v.OperationName) {
		return false
	}
	if v.SchemaContractID != "" {
		if len(v.SchemaContractID) != 32 {
			return false
		}
		if _, err := hex.DecodeString(v.SchemaContractID); err != nil {
			return false
		}
	}
	if v.Action != "learn" && v.Action != "detect" && v.Action != "block" {
		return false
	}
	switch v.Type {
	case "MALFORMED_GRAPHQL", "ABSOLUTE_DEPTH_LIMIT", "ABSOLUTE_COMPLEXITY_LIMIT", "DEPTH_LIMIT", "COMPLEXITY_LIMIT", "INTROSPECTION_DISABLED", "MUTATION_DISABLED", "SUBSCRIPTION_DISABLED", "PERSISTED_QUERY_REQUIRED", "SCHEMA_FIELD_MISMATCH", "FIELD_NOT_ALLOWED", "FIELD_DENIED":
	default:
		return false
	}
	if len(v.Expected) > 1024 || len(v.Observed) > 1024 {
		return false
	}
	if v.Field != "" && !validGraphQLFieldPath(v.Field) {
		return false
	}
	return true
}

func graphqlSchemaContractID(endpointOperationID, name string) string {
	h := sha256.Sum256([]byte("api8-graphql-schema\x00" + endpointOperationID + "\x00" + strings.ToLower(strings.TrimSpace(name))))
	return hex.EncodeToString(h[:16])
}

func validGraphQLSchemaContract(v GraphQLSchemaContract) bool {
	if len(v.ID) != 32 || v.ID != graphqlSchemaContractID(v.EndpointOperationID, v.Name) || !validSequenceOperationID(v.EndpointOperationID) ||
		strings.TrimSpace(v.Name) == "" || len(v.Name) > 128 || !validGraphQLHash(v.SourceSHA256) || !validGraphQLName(v.QueryRoot) ||
		(v.MutationRoot != "" && !validGraphQLName(v.MutationRoot)) || (v.SubscriptionRoot != "" && !validGraphQLName(v.SubscriptionRoot)) ||
		len(v.Types) == 0 || len(v.Types) > graphqlMaxSchemaTypes || len(v.FieldTypes) != len(v.Types) || v.CreatedAt.IsZero() || v.UpdatedAt.IsZero() {
		return false
	}
	total := 0
	for typ, fields := range v.Types {
		if !validGraphQLName(typ) || len(fields) > graphqlMaxSchemaFields {
			return false
		}
		total += len(fields)
		if total > graphqlMaxSchemaFields {
			return false
		}
		ft := v.FieldTypes[typ]
		if len(ft) != len(fields) {
			return false
		}
		for _, field := range fields {
			if !validGraphQLName(field) || !validGraphQLName(ft[field]) {
				return false
			}
		}
	}
	return true
}

func (s *graphqlStore) upsertSchemaContract(name, endpointOperationID, sdl, who string) (GraphQLSchemaContract, error) {
	if s == nil {
		return GraphQLSchemaContract{}, errors.New("GraphQL store unavailable")
	}
	name = strings.TrimSpace(name)
	endpointOperationID = strings.TrimSpace(endpointOperationID)
	if name == "" || len(name) > 128 {
		return GraphQLSchemaContract{}, errors.New("bounded schema name is required")
	}
	if !validSequenceOperationID(endpointOperationID) {
		return GraphQLSchemaContract{}, errors.New("endpoint_operation_id must be a normalized API-1 operation ID")
	}
	if len(sdl) == 0 || len(sdl) > graphqlMaxSchemaBytes {
		return GraphQLSchemaContract{}, fmt.Errorf("GraphQL SDL must be between 1 and %d bytes", graphqlMaxSchemaBytes)
	}
	queryRoot, mutationRoot, subscriptionRoot, types, fieldTypes, err := parseGraphQLSDL(sdl)
	if err != nil {
		return GraphQLSchemaContract{}, err
	}
	sum := sha256.Sum256([]byte(sdl))
	now := s.now().UTC()
	id := graphqlSchemaContractID(endpointOperationID, name)
	row := GraphQLSchemaContract{ID: id, Name: name, EndpointOperationID: endpointOperationID, SourceSHA256: hex.EncodeToString(sum[:]), QueryRoot: queryRoot, MutationRoot: mutationRoot, SubscriptionRoot: subscriptionRoot, Types: types, FieldTypes: fieldTypes, UpdatedAt: now, CreatedBy: who}
	s.mu.Lock()
	defer s.mu.Unlock()
	if old := s.contracts[id]; old != nil {
		row.CreatedAt = old.CreatedAt
		if old.SourceSHA256 != row.SourceSHA256 {
			for _, p := range s.policies {
				if p.SchemaContractID == id {
					p.Mode = graphqlModeLearn
					p.UpdatedAt = now
					p.UpdatedBy = who
					p.Reason = "schema contract changed; revalidation required"
				}
			}
		}
	} else {
		if len(s.contracts) >= graphqlMaxSchemaContracts {
			return GraphQLSchemaContract{}, errors.New("GraphQL schema contract capacity reached")
		}
		row.CreatedAt = now
	}
	s.contracts[id] = &row
	s.rebuildRuntimeLocked()
	return cloneGraphQLContract(row), nil
}

func (s *graphqlStore) contractsSnapshot() []GraphQLSchemaContract {
	if s == nil {
		return []GraphQLSchemaContract{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]GraphQLSchemaContract, 0, len(s.contracts))
	for _, v := range s.contracts {
		out = append(out, cloneGraphQLContract(*v))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *graphqlStore) deleteSchemaContract(id string) (GraphQLSchemaContract, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.contracts[id]
	if v == nil {
		return GraphQLSchemaContract{}, errors.New("GraphQL schema contract not found")
	}
	for _, p := range s.policies {
		if p.SchemaContractID == id {
			return GraphQLSchemaContract{}, fmt.Errorf("%w %s; update or delete the policy first", errGraphQLSchemaContractInUse, p.ID)
		}
	}
	out := cloneGraphQLContract(*v)
	delete(s.contracts, id)
	s.rebuildRuntimeLocked()
	return out, nil
}

func normalizeGraphQLFieldList(in []string) ([]string, error) {
	if len(in) > graphqlMaxPolicyFields {
		return nil, errors.New("too many GraphQL field selectors")
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, raw := range in {
		v := strings.TrimSpace(raw)
		if !validGraphQLFieldPath(v) {
			return nil, fmt.Errorf("invalid GraphQL field selector %q", raw)
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out, nil
}

func (s *graphqlStore) upsertPolicy(req GraphQLPolicy, who string) (GraphQLPolicy, error) {
	if s == nil {
		return GraphQLPolicy{}, errors.New("GraphQL store unavailable")
	}
	req.EndpointOperationID = strings.TrimSpace(req.EndpointOperationID)
	req.OperationName = strings.TrimSpace(req.OperationName)
	if !validSequenceOperationID(req.EndpointOperationID) {
		return GraphQLPolicy{}, errors.New("endpoint_operation_id must be a normalized API-1 operation ID")
	}
	if req.OperationName != "" && !validGraphQLName(req.OperationName) {
		return GraphQLPolicy{}, errors.New("invalid operation_name")
	}
	if req.SchemaContractID != "" {
		s.mu.RLock()
		contract := s.contracts[req.SchemaContractID]
		s.mu.RUnlock()
		if contract == nil || contract.EndpointOperationID != req.EndpointOperationID {
			return GraphQLPolicy{}, errors.New("schema_contract_id must reference a GraphQL contract for endpoint_operation_id")
		}
	}
	allowed, err := normalizeGraphQLFieldList(req.AllowedFields)
	if err != nil {
		return GraphQLPolicy{}, err
	}
	denied, err := normalizeGraphQLFieldList(req.DeniedFields)
	if err != nil {
		return GraphQLPolicy{}, err
	}
	if req.MaxDepth < 0 || req.MaxDepth > graphqlAbsoluteMaxDepth {
		return GraphQLPolicy{}, fmt.Errorf("max_depth must be between 0 and %d", graphqlAbsoluteMaxDepth)
	}
	if req.MaxComplexity < 0 || req.MaxComplexity > graphqlAbsoluteMaxComplexity {
		return GraphQLPolicy{}, fmt.Errorf("max_complexity must be between 0 and %d", graphqlAbsoluteMaxComplexity)
	}
	if req.BlockStatus == 0 {
		req.BlockStatus = http.StatusBadRequest
	}
	if req.BlockStatus < 400 || req.BlockStatus > 499 {
		return GraphQLPolicy{}, errors.New("block_status must be a 4xx status")
	}
	req.Reason = strings.TrimSpace(req.Reason)
	if len(req.Reason) > graphqlMaxPolicyReason {
		return GraphQLPolicy{}, fmt.Errorf("reason must be at most %d bytes", graphqlMaxPolicyReason)
	}
	id := graphqlPolicyID(req.EndpointOperationID, req.OperationName)
	now := s.now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	if req.SchemaContractID != "" {
		contract := s.contracts[req.SchemaContractID]
		if contract == nil || contract.EndpointOperationID != req.EndpointOperationID {
			return GraphQLPolicy{}, errors.New("schema_contract_id must reference a GraphQL contract for endpoint_operation_id")
		}
	}
	p := s.policies[id]
	if p == nil {
		if len(s.policies) >= graphqlMaxPolicies {
			return GraphQLPolicy{}, errors.New("GraphQL policy capacity reached")
		}
		p = &GraphQLPolicy{ID: id, EndpointOperationID: req.EndpointOperationID, OperationName: req.OperationName, Mode: graphqlModeLearn, CreatedAt: now, CreatedBy: who}
		s.policies[id] = p
	}
	// Any policy edit returns the policy to LEARN. This prevents a field/depth/
	// schema change from becoming live enforcement without a fresh DETECT review.
	p.Mode = graphqlModeLearn
	p.SchemaContractID = strings.TrimSpace(req.SchemaContractID)
	p.MaxDepth, p.MaxComplexity = req.MaxDepth, req.MaxComplexity
	p.AllowIntrospection, p.AllowMutation, p.AllowSubscription, p.RequirePersisted = req.AllowIntrospection, req.AllowMutation, req.AllowSubscription, req.RequirePersisted
	p.AllowedFields, p.DeniedFields = allowed, denied
	p.BlockStatus, p.UpdatedAt, p.UpdatedBy, p.Reason = req.BlockStatus, now, who, req.Reason
	s.rebuildRuntimeLocked()
	return cloneGraphQLPolicy(*p), nil
}

func graphqlModeTransition(from, to string) bool {
	if from == to {
		return true
	}
	switch from {
	case graphqlModeLearn:
		return to == graphqlModeDetect
	case graphqlModeDetect:
		return to == graphqlModeLearn || to == graphqlModeEnforce
	case graphqlModeEnforce:
		return to == graphqlModeDetect || to == graphqlModeLearn
	default:
		return false
	}
}
func (s *graphqlStore) setMode(id, mode, who, reason string) (GraphQLPolicy, error) {
	mode = strings.ToUpper(strings.TrimSpace(mode))
	reason = strings.TrimSpace(reason)
	if len(reason) > graphqlMaxPolicyReason {
		return GraphQLPolicy{}, fmt.Errorf("reason must be at most %d bytes", graphqlMaxPolicyReason)
	}
	if !validGraphQLMode(mode) {
		return GraphQLPolicy{}, errors.New("mode must be LEARN, DETECT, or ENFORCE")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.policies[id]
	if p == nil {
		return GraphQLPolicy{}, errors.New("GraphQL policy not found")
	}
	if !graphqlModeTransition(p.Mode, mode) {
		return GraphQLPolicy{}, fmt.Errorf("invalid GraphQL mode transition %s -> %s", p.Mode, mode)
	}
	if mode == graphqlModeEnforce && reason == "" {
		return GraphQLPolicy{}, errors.New("ENFORCE promotion requires a reason")
	}
	p.Mode, p.UpdatedAt, p.UpdatedBy, p.Reason = mode, s.now().UTC(), who, reason
	s.rebuildRuntimeLocked()
	return cloneGraphQLPolicy(*p), nil
}
func (s *graphqlStore) deletePolicy(id string) (GraphQLPolicy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.policies[id]
	if p == nil {
		return GraphQLPolicy{}, errors.New("GraphQL policy not found")
	}
	out := cloneGraphQLPolicy(*p)
	delete(s.policies, id)
	s.rebuildRuntimeLocked()
	return out, nil
}

func (s *graphqlStore) runtimePolicy(endpointOperationID, operationName string) (GraphQLPolicy, bool) {
	if s == nil {
		return GraphQLPolicy{}, false
	}
	snap := s.runtime.Load()
	if snap == nil {
		return GraphQLPolicy{}, false
	}
	if operationName != "" {
		if p, ok := snap.Policies[graphqlPolicyID(endpointOperationID, operationName)]; ok {
			return p, true
		}
	}
	p, ok := snap.Policies[graphqlPolicyID(endpointOperationID, "")]
	return p, ok
}
func graphqlModeRank(mode string) int {
	switch mode {
	case graphqlModeEnforce:
		return 3
	case graphqlModeDetect:
		return 2
	case graphqlModeLearn:
		return 1
	default:
		return 0
	}
}

func graphqlFallbackPolicyBetter(candidate, current GraphQLPolicy) bool {
	cr, rr := graphqlModeRank(candidate.Mode), graphqlModeRank(current.Mode)
	if cr != rr {
		return cr > rr
	}
	if candidate.OperationName == "" && current.OperationName != "" {
		return true
	}
	if candidate.OperationName != "" && current.OperationName == "" {
		return false
	}
	return candidate.ID < current.ID
}

func (s *graphqlStore) runtimeContract(id string) (GraphQLSchemaContract, bool) {
	if s == nil || id == "" {
		return GraphQLSchemaContract{}, false
	}
	snap := s.runtime.Load()
	if snap == nil {
		return GraphQLSchemaContract{}, false
	}
	c, ok := snap.Contracts[id]
	return c, ok
}

func (s *graphqlStore) endpointFallbackPolicy(endpointOperationID string) (GraphQLPolicy, bool) {
	if s == nil {
		return GraphQLPolicy{}, false
	}
	snap := s.runtime.Load()
	if snap == nil {
		return GraphQLPolicy{}, false
	}
	p, ok := snap.EndpointFallback[endpointOperationID]
	return p, ok
}

func (s *graphqlStore) persistedProfile(hash string) (GraphQLPersistedProfile, bool) {
	if s == nil || !validGraphQLHash(hash) {
		return GraphQLPersistedProfile{}, false
	}
	snap := s.runtime.Load()
	if snap == nil {
		return GraphQLPersistedProfile{}, false
	}
	p, ok := snap.Persisted[strings.ToLower(hash)]
	return p, ok
}

func (s *graphqlStore) operationsSnapshot() []GraphQLOperation {
	if s == nil {
		return []GraphQLOperation{}
	}
	now := s.now().UTC()
	s.mu.Lock()
	s.pruneLocked(now)
	out := make([]GraphQLOperation, 0, len(s.operations))
	for _, v := range s.operations {
		out = append(out, cloneGraphQLOperation(*v))
	}
	s.mu.Unlock()
	sort.Slice(out, func(i, j int) bool {
		if !out[i].LastSeen.Equal(out[j].LastSeen) {
			return out[i].LastSeen.After(out[j].LastSeen)
		}
		return out[i].ID < out[j].ID
	})
	return out
}
func (s *graphqlStore) persistedSnapshot() []GraphQLPersistedProfile {
	if s == nil {
		return []GraphQLPersistedProfile{}
	}
	now := s.now().UTC()
	s.mu.Lock()
	s.pruneLocked(now)
	out := make([]GraphQLPersistedProfile, 0, len(s.persisted))
	for _, v := range s.persisted {
		out = append(out, cloneGraphQLPersisted(*v))
	}
	s.mu.Unlock()
	sort.Slice(out, func(i, j int) bool { return out[i].LastSeen.After(out[j].LastSeen) })
	return out
}
func (s *graphqlStore) policiesSnapshot() []GraphQLPolicy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]GraphQLPolicy, 0, len(s.policies))
	for _, v := range s.policies {
		out = append(out, cloneGraphQLPolicy(*v))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
func (s *graphqlStore) violationsSnapshot(limit int) []GraphQLViolation {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := len(s.violations)
	start := n - limit
	if start < 0 {
		start = 0
	}
	out := append([]GraphQLViolation(nil), s.violations[start:]...)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}
func (s *graphqlStore) status() GraphQLStatus {
	ops := s.operationsSnapshot()
	persisted := s.persistedSnapshot()
	s.mu.RLock()
	cc, pc, vc := len(s.contracts), len(s.policies), len(s.violations)
	s.mu.RUnlock()
	return GraphQLStatus{OperationCount: len(ops), PersistedQueryCount: len(persisted), SchemaContractCount: cc, PolicyCount: pc, ViolationCount: vc, QueueDepth: len(s.queue), QueueCapacity: cap(s.queue), DroppedObservations: s.dropped.Load(), ProcessedObservations: s.processed.Load(), MaxOperations: graphqlMaxOperations, MaxPersistedQueries: graphqlMaxPersistedQueries, AbsoluteMaxDepth: graphqlAbsoluteMaxDepth, AbsoluteMaxComplexity: graphqlAbsoluteMaxComplexity, Accepting: s.accepting.Load()}
}

func (s *graphqlStore) recordViolations(rows []GraphQLViolation) {
	if s == nil || len(rows) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range rows {
		if rows[i].Time.IsZero() {
			rows[i].Time = s.now().UTC()
		}
		if rows[i].ID == "" {
			h := sha256.Sum256([]byte(fmt.Sprintf("api8-violation\x00%s\x00%s\x00%s\x00%d", rows[i].GraphQLOperationID, rows[i].Type, rows[i].Field, rows[i].Time.UnixNano())))
			rows[i].ID = hex.EncodeToString(h[:16])
		}
		s.violations = append(s.violations, rows[i])
	}
	if len(s.violations) > graphqlMaxViolations {
		s.violations = append([]GraphQLViolation(nil), s.violations[len(s.violations)-graphqlMaxViolations:]...)
	}
}

// parseGraphQLSDL extracts bounded object-type field topology from SDL. Raw SDL
// is used only during this import call and is replaced by a SHA-256 digest plus
// normalized type/field names in durable state.
func parseGraphQLSDL(src string) (string, string, string, map[string][]string, map[string]map[string]string, error) {
	if len(src) == 0 || len(src) > graphqlMaxSchemaBytes {
		return "", "", "", nil, nil, errors.New("GraphQL SDL size is out of bounds")
	}
	toks, err := lexGraphQLBounded(src, graphqlMaxSchemaBytes, graphqlMaxSchemaTokens)
	if err != nil {
		return "", "", "", nil, nil, err
	}
	p := &gqlParser{toks: toks}
	queryRoot, mutationRoot, subscriptionRoot := "Query", "Mutation", "Subscription"
	types := map[string][]string{}
	fieldTypes := map[string]map[string]string{}
	totalFields := 0
	for p.peek().kind != gqlEOF {
		if p.peek().kind == gqlString {
			p.take()
			continue
		}
		kw, err := p.name()
		if err != nil {
			return "", "", "", nil, nil, err
		}
		if kw == "extend" {
			kw, err = p.name()
			if err != nil {
				return "", "", "", nil, nil, err
			}
		}
		switch kw {
		case "schema":
			if err := p.skipDirectives(); err != nil {
				return "", "", "", nil, nil, err
			}
			if err := p.expect("{"); err != nil {
				return "", "", "", nil, nil, err
			}
			for !p.accept("}") {
				kind, err := p.name()
				if err != nil {
					return "", "", "", nil, nil, err
				}
				if err = p.expect(":"); err != nil {
					return "", "", "", nil, nil, err
				}
				root, err := p.name()
				if err != nil {
					return "", "", "", nil, nil, err
				}
				switch kind {
				case "query":
					queryRoot = root
				case "mutation":
					mutationRoot = root
				case "subscription":
					subscriptionRoot = root
				}
			}
		case "type":
			name, err := p.name()
			if err != nil {
				return "", "", "", nil, nil, err
			}
			for p.peek().kind != gqlEOF && p.peek().text != "{" {
				p.take()
			}
			if err := p.expect("{"); err != nil {
				return "", "", "", nil, nil, err
			}
			fields := append([]string(nil), types[name]...)
			ftypes := fieldTypes[name]
			if ftypes == nil {
				ftypes = map[string]string{}
			}
			seen := map[string]struct{}{}
			for _, f := range fields {
				seen[f] = struct{}{}
			}
			for !p.accept("}") {
				if p.peek().kind == gqlString {
					p.take()
					continue
				}
				field, err := p.name()
				if err != nil {
					return "", "", "", nil, nil, err
				}
				if p.accept("(") {
					if err := skipSDLArguments(p); err != nil {
						return "", "", "", nil, nil, err
					}
				}
				if err = p.expect(":"); err != nil {
					return "", "", "", nil, nil, err
				}
				typeRef, typeErr := p.typeRef()
				if typeErr != nil {
					return "", "", "", nil, nil, typeErr
				}
				if err = p.skipDirectives(); err != nil {
					return "", "", "", nil, nil, err
				}
				ftypes[field] = graphqlBaseType(typeRef)
				if _, ok := seen[field]; !ok {
					fields = append(fields, field)
					seen[field] = struct{}{}
					totalFields++
					if totalFields > graphqlMaxSchemaFields {
						return "", "", "", nil, nil, errors.New("GraphQL SDL has too many fields")
					}
				}
			}
			sort.Strings(fields)
			types[name] = fields
			fieldTypes[name] = ftypes
			if len(types) > graphqlMaxSchemaTypes {
				return "", "", "", nil, nil, errors.New("GraphQL SDL has too many types")
			}
		default:
			if err := skipSDLDefinition(p); err != nil {
				return "", "", "", nil, nil, err
			}
		}
	}
	if _, ok := types[queryRoot]; !ok {
		return "", "", "", nil, nil, fmt.Errorf("GraphQL SDL query root %q is not defined", queryRoot)
	}
	if _, ok := types[mutationRoot]; !ok {
		mutationRoot = ""
	}
	if _, ok := types[subscriptionRoot]; !ok {
		subscriptionRoot = ""
	}
	return queryRoot, mutationRoot, subscriptionRoot, types, fieldTypes, nil
}

func graphqlBaseType(typeRef string) string {
	v := strings.TrimSpace(typeRef)
	v = strings.Trim(v, "![]")
	return strings.Trim(v, "![]")
}

func skipSDLArguments(p *gqlParser) error {
	depth := 1
	for depth > 0 {
		t := p.take()
		if t.kind == gqlEOF {
			return errors.New("unterminated GraphQL SDL arguments")
		}
		if t.text == "(" {
			depth++
		} else if t.text == ")" {
			depth--
		}
	}
	return nil
}

func skipSDLDefinition(p *gqlParser) error {
	depth := 0
	for p.peek().kind != gqlEOF {
		if depth == 0 && p.peek().kind == gqlName {
			switch p.peek().text {
			case "schema", "type", "extend", "scalar", "enum", "input", "interface", "union", "directive":
				return nil
			}
		}
		t := p.take()
		switch t.text {
		case "{", "[", "(":
			depth++
		case "}", "]", ")":
			if depth > 0 {
				depth--
			}
		}
	}
	return nil
}

// --- bounded GraphQL lexer/parser ---
type gqlTokenKind uint8

const (
	gqlEOF gqlTokenKind = iota
	gqlName
	gqlNumber
	gqlString
	gqlPunct
	gqlSpread
)

type gqlToken struct {
	kind gqlTokenKind
	text string
}

func lexGraphQL(src string) ([]gqlToken, error) {
	return lexGraphQLBounded(src, graphqlMaxQueryBytes, graphqlMaxTokens)
}

func lexGraphQLBounded(src string, maxBytes, maxTokens int) ([]gqlToken, error) {
	if len(src) == 0 {
		return nil, errors.New("empty GraphQL document")
	}
	if len(src) > maxBytes {
		return nil, fmt.Errorf("GraphQL document exceeds %d bytes", maxBytes)
	}
	out := make([]gqlToken, 0, 128)
	for i := 0; i < len(src); {
		c := src[i]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == ',' {
			i++
			continue
		}
		if c == '#' {
			for i < len(src) && src[i] != '\n' && src[i] != '\r' {
				i++
			}
			continue
		}
		if i+2 < len(src) && src[i:i+3] == "..." {
			out = append(out, gqlToken{kind: gqlSpread, text: "..."})
			i += 3
			if len(out) > maxTokens {
				return nil, errors.New("too many GraphQL tokens")
			}
			continue
		}
		if c == '"' {
			if i+2 < len(src) && src[i:i+3] == "\"\"\"" {
				i += 3
				for {
					if i+2 >= len(src) {
						return nil, errors.New("unterminated GraphQL block string")
					}
					if src[i:i+3] == "\"\"\"" {
						i += 3
						break
					}
					i++
				}
			} else {
				i++
				esc := false
				for i < len(src) {
					ch := src[i]
					i++
					if esc {
						esc = false
						continue
					}
					if ch == '\\' {
						esc = true
						continue
					}
					if ch == '"' {
						break
					}
					if ch == '\n' || ch == '\r' {
						return nil, errors.New("newline in GraphQL string")
					}
					if i == len(src) && ch != '"' {
						return nil, errors.New("unterminated GraphQL string")
					}
				}
			}
			out = append(out, gqlToken{kind: gqlString, text: "<string>"})
		} else if isGQLNameStart(c) {
			j := i + 1
			for j < len(src) && isGQLNameContinue(src[j]) {
				j++
			}
			out = append(out, gqlToken{kind: gqlName, text: src[i:j]})
			i = j
		} else if c == '-' || (c >= '0' && c <= '9') {
			j := i + 1
			for j < len(src) {
				ch := src[j]
				if (ch >= '0' && ch <= '9') || ch == '.' || ch == 'e' || ch == 'E' || ch == '+' || ch == '-' {
					j++
				} else {
					break
				}
			}
			out = append(out, gqlToken{kind: gqlNumber, text: "<number>"})
			i = j
		} else if strings.ContainsRune("!$():=@[]{|}&", rune(c)) {
			out = append(out, gqlToken{kind: gqlPunct, text: string(c)})
			i++
		} else {
			return nil, fmt.Errorf("invalid GraphQL character at byte %d", i)
		}
		if len(out) > maxTokens {
			return nil, errors.New("too many GraphQL tokens")
		}
	}
	out = append(out, gqlToken{kind: gqlEOF})
	return out, nil
}
func isGQLNameStart(c byte) bool    { return c == '_' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') }
func isGQLNameContinue(c byte) bool { return isGQLNameStart(c) || (c >= '0' && c <= '9') }

type gqlSelection struct {
	field    string
	children []gqlSelection
	spread   string
	inline   bool
}
type gqlOperationDef struct {
	typ, name  string
	variables  map[string]string
	selections []gqlSelection
}
type gqlFragmentDef struct {
	name       string
	selections []gqlSelection
}
type gqlDocument struct {
	operations []gqlOperationDef
	fragments  map[string]gqlFragmentDef
}
type gqlParser struct {
	toks []gqlToken
	pos  int
}

func (p *gqlParser) peek() gqlToken {
	if p.pos >= len(p.toks) {
		return gqlToken{kind: gqlEOF}
	}
	return p.toks[p.pos]
}
func (p *gqlParser) take() gqlToken {
	t := p.peek()
	if p.pos < len(p.toks) {
		p.pos++
	}
	return t
}
func (p *gqlParser) accept(text string) bool {
	if p.peek().text == text {
		p.pos++
		return true
	}
	return false
}
func (p *gqlParser) name() (string, error) {
	t := p.take()
	if t.kind != gqlName {
		return "", fmt.Errorf("expected GraphQL name")
	}
	return t.text, nil
}
func (p *gqlParser) expect(text string) error {
	if !p.accept(text) {
		return fmt.Errorf("expected %q", text)
	}
	return nil
}

func parseGraphQLDocument(src string) (gqlDocument, error) {
	toks, err := lexGraphQL(src)
	if err != nil {
		return gqlDocument{}, err
	}
	p := &gqlParser{toks: toks}
	doc := gqlDocument{fragments: map[string]gqlFragmentDef{}}
	for p.peek().kind != gqlEOF {
		if p.peek().text == "{" {
			sels, err := p.selectionSet()
			if err != nil {
				return gqlDocument{}, err
			}
			doc.operations = append(doc.operations, gqlOperationDef{typ: "query", variables: map[string]string{}, selections: sels})
			continue
		}
		kw, err := p.name()
		if err != nil {
			return gqlDocument{}, err
		}
		switch kw {
		case "query", "mutation", "subscription":
			op := gqlOperationDef{typ: kw, variables: map[string]string{}}
			if p.peek().kind == gqlName {
				op.name, _ = p.name()
			}
			if p.accept("(") {
				if err := p.variableDefinitions(op.variables); err != nil {
					return gqlDocument{}, err
				}
			}
			if err := p.skipDirectives(); err != nil {
				return gqlDocument{}, err
			}
			op.selections, err = p.selectionSet()
			if err != nil {
				return gqlDocument{}, err
			}
			doc.operations = append(doc.operations, op)
		case "fragment":
			name, err := p.name()
			if err != nil {
				return gqlDocument{}, err
			}
			if _, exists := doc.fragments[name]; exists {
				return gqlDocument{}, errors.New("duplicate GraphQL fragment")
			}
			if err := p.expect("on"); err != nil {
				return gqlDocument{}, err
			}
			if _, err = p.name(); err != nil {
				return gqlDocument{}, err
			}
			if err := p.skipDirectives(); err != nil {
				return gqlDocument{}, err
			}
			sels, err := p.selectionSet()
			if err != nil {
				return gqlDocument{}, err
			}
			doc.fragments[name] = gqlFragmentDef{name: name, selections: sels}
		default:
			return gqlDocument{}, fmt.Errorf("unsupported GraphQL definition %q", kw)
		}
		if len(doc.operations) > 128 || len(doc.fragments) > 256 {
			return gqlDocument{}, errors.New("too many GraphQL definitions")
		}
	}
	if len(doc.operations) == 0 {
		return gqlDocument{}, errors.New("GraphQL document has no operation")
	}
	return doc, nil
}
func (p *gqlParser) variableDefinitions(dst map[string]string) error {
	for !p.accept(")") {
		if p.peek().kind == gqlEOF {
			return errors.New("unterminated variable definitions")
		}
		if err := p.expect("$"); err != nil {
			return err
		}
		n, err := p.name()
		if err != nil {
			return err
		}
		if len(dst) >= graphqlMaxVariables {
			return errors.New("too many GraphQL variables")
		}
		if err = p.expect(":"); err != nil {
			return err
		}
		typ, err := p.typeRef()
		if err != nil {
			return err
		}
		dst[n] = typ
		if p.accept("=") {
			if err = p.skipValue(); err != nil {
				return err
			}
		}
		if err = p.skipDirectives(); err != nil {
			return err
		}
	}
	return nil
}
func (p *gqlParser) typeRef() (string, error) {
	if p.accept("[") {
		inner, err := p.typeRef()
		if err != nil {
			return "", err
		}
		if err = p.expect("]"); err != nil {
			return "", err
		}
		v := "[" + inner + "]"
		if p.accept("!") {
			v += "!"
		}
		return v, nil
	}
	n, err := p.name()
	if err != nil {
		return "", err
	}
	if p.accept("!") {
		n += "!"
	}
	return n, nil
}
func (p *gqlParser) skipDirectives() error {
	for p.accept("@") {
		if _, err := p.name(); err != nil {
			return err
		}
		if p.accept("(") {
			if err := p.skipArguments(); err != nil {
				return err
			}
		}
	}
	return nil
}
func (p *gqlParser) skipArguments() error {
	for !p.accept(")") {
		if p.peek().kind == gqlEOF {
			return errors.New("unterminated GraphQL arguments")
		}
		if _, err := p.name(); err != nil {
			return err
		}
		if err := p.expect(":"); err != nil {
			return err
		}
		if err := p.skipValue(); err != nil {
			return err
		}
	}
	return nil
}
func (p *gqlParser) skipValue() error {
	t := p.peek()
	if t.kind == gqlString || t.kind == gqlNumber || t.kind == gqlName {
		p.take()
		return nil
	}
	if p.accept("$") {
		_, err := p.name()
		return err
	}
	if p.accept("[") {
		for !p.accept("]") {
			if p.peek().kind == gqlEOF {
				return errors.New("unterminated GraphQL list")
			}
			if err := p.skipValue(); err != nil {
				return err
			}
		}
		return nil
	}
	if p.accept("{") {
		for !p.accept("}") {
			if p.peek().kind == gqlEOF {
				return errors.New("unterminated GraphQL object")
			}
			if _, err := p.name(); err != nil {
				return err
			}
			if err := p.expect(":"); err != nil {
				return err
			}
			if err := p.skipValue(); err != nil {
				return err
			}
		}
		return nil
	}
	return errors.New("invalid GraphQL value")
}
func (p *gqlParser) selectionSet() ([]gqlSelection, error) {
	if err := p.expect("{"); err != nil {
		return nil, err
	}
	out := make([]gqlSelection, 0, 8)
	for !p.accept("}") {
		if p.peek().kind == gqlEOF {
			return nil, errors.New("unterminated GraphQL selection set")
		}
		sel, err := p.selection()
		if err != nil {
			return nil, err
		}
		out = append(out, sel)
		if len(out) > graphqlMaxFields {
			return nil, errors.New("too many GraphQL fields")
		}
	}
	return out, nil
}
func (p *gqlParser) selection() (gqlSelection, error) {
	if p.accept("...") {
		if p.accept("on") {
			if _, err := p.name(); err != nil {
				return gqlSelection{}, err
			}
			if err := p.skipDirectives(); err != nil {
				return gqlSelection{}, err
			}
			kids, err := p.selectionSet()
			return gqlSelection{inline: true, children: kids}, err
		}
		if p.peek().text == "@" {
			if err := p.skipDirectives(); err != nil {
				return gqlSelection{}, err
			}
			kids, err := p.selectionSet()
			return gqlSelection{inline: true, children: kids}, err
		}
		n, err := p.name()
		if err != nil {
			return gqlSelection{}, err
		}
		if err = p.skipDirectives(); err != nil {
			return gqlSelection{}, err
		}
		return gqlSelection{spread: n}, nil
	}
	n, err := p.name()
	if err != nil {
		return gqlSelection{}, err
	}
	field := n
	if p.accept(":") {
		field, err = p.name()
		if err != nil {
			return gqlSelection{}, err
		}
	}
	if p.accept("(") {
		if err = p.skipArguments(); err != nil {
			return gqlSelection{}, err
		}
	}
	if err = p.skipDirectives(); err != nil {
		return gqlSelection{}, err
	}
	var kids []gqlSelection
	if p.peek().text == "{" {
		kids, err = p.selectionSet()
		if err != nil {
			return gqlSelection{}, err
		}
	}
	return gqlSelection{field: field, children: kids}, nil
}

type graphqlAnalysis struct {
	OperationType, OperationName, DocumentFingerprint string
	Depth, Complexity                                 int
	Fields                                            []string
	VariableTypes                                     map[string]string
	Introspection                                     bool
}

func analyzeGraphQLDocument(src, operationName string) (graphqlAnalysis, error) {
	doc, err := parseGraphQLDocument(src)
	if err != nil {
		return graphqlAnalysis{}, err
	}
	var op *gqlOperationDef
	if operationName != "" {
		for i := range doc.operations {
			if doc.operations[i].name == operationName {
				op = &doc.operations[i]
				break
			}
		}
		if op == nil {
			return graphqlAnalysis{}, errors.New("operationName does not match a GraphQL operation")
		}
	} else if len(doc.operations) == 1 {
		op = &doc.operations[0]
	} else {
		return graphqlAnalysis{}, errors.New("operationName is required for a multi-operation GraphQL document")
	}
	fields := make([]string, 0, 64)
	seenFields := map[string]struct{}{}
	complexity := 0
	maxDepth := 0
	introspection := false
	var walk func([]gqlSelection, string, int, map[string]bool) error
	walk = func(sels []gqlSelection, prefix string, depth int, stack map[string]bool) error {
		if depth > graphqlAbsoluteMaxDepth {
			return errors.New("GraphQL depth exceeds absolute safety limit")
		}
		for _, sel := range sels {
			if sel.spread != "" {
				if stack[sel.spread] {
					return errors.New("cyclic GraphQL fragment")
				}
				f, ok := doc.fragments[sel.spread]
				if !ok {
					return errors.New("unknown GraphQL fragment")
				}
				next := make(map[string]bool, len(stack)+1)
				for k, v := range stack {
					next[k] = v
				}
				next[sel.spread] = true
				if err := walk(f.selections, prefix, depth, next); err != nil {
					return err
				}
				continue
			}
			if sel.inline {
				if err := walk(sel.children, prefix, depth, stack); err != nil {
					return err
				}
				continue
			}
			if sel.field == "" {
				continue
			}
			complexity++
			if complexity > graphqlAbsoluteMaxComplexity {
				return errors.New("GraphQL complexity exceeds absolute safety limit")
			}
			path := prefix + "." + sel.field
			if _, ok := seenFields[path]; !ok {
				if len(fields) >= graphqlMaxFields {
					return errors.New("too many GraphQL fields")
				}
				seenFields[path] = struct{}{}
				fields = append(fields, path)
			}
			if sel.field == "__schema" || sel.field == "__type" {
				introspection = true
			}
			if depth > maxDepth {
				maxDepth = depth
			}
			if len(sel.children) > 0 {
				if err := walk(sel.children, path, depth+1, stack); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(op.selections, op.typ, 1, map[string]bool{}); err != nil {
		return graphqlAnalysis{}, err
	}
	sort.Strings(fields)
	vars := make(map[string]string, len(op.variables))
	for k, v := range op.variables {
		vars[k] = v
	}
	// The fingerprint is derived from normalized structural metadata only; raw
	// literals and the original query text are not durable evidence.
	varNames := make([]string, 0, len(vars))
	for k := range vars {
		varNames = append(varNames, k)
	}
	sort.Strings(varNames)
	parts := []string{op.typ, op.name, strconv.Itoa(maxDepth), strconv.Itoa(complexity), strings.Join(fields, "\x1f")}
	for _, n := range varNames {
		parts = append(parts, n+":"+vars[n])
	}
	h := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return graphqlAnalysis{OperationType: op.typ, OperationName: op.name, DocumentFingerprint: hex.EncodeToString(h[:]), Depth: maxDepth, Complexity: complexity, Fields: fields, VariableTypes: vars, Introspection: introspection}, nil
}

func graphqlVariablesList(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for n := range m {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}
func persistedFromAnalysis(hash string, op GraphQLOperation, at time.Time) GraphQLPersistedProfile {
	return GraphQLPersistedProfile{SHA256Hash: hash, GraphQLOperationID: op.ID, EndpointOperationID: op.EndpointOperationID, OperationType: op.OperationType, OperationName: op.OperationName, DocumentFingerprint: op.DocumentFingerprint, Depth: op.Depth, Complexity: op.Complexity, Fields: append([]string(nil), op.Fields...), Variables: append([]string(nil), op.Variables...), Introspection: op.Introspection, FirstSeen: at, LastSeen: at, ExpiresAt: at.Add(graphqlPersistedTTL)}
}
func analysisFromPersisted(p GraphQLPersistedProfile) graphqlAnalysis {
	vars := map[string]string{}
	for _, n := range p.Variables {
		vars[n] = ""
	}
	return graphqlAnalysis{OperationType: p.OperationType, OperationName: p.OperationName, DocumentFingerprint: p.DocumentFingerprint, Depth: p.Depth, Complexity: p.Complexity, Fields: append([]string(nil), p.Fields...), VariableTypes: vars, Introspection: p.Introspection}
}

func scalarGraphQLVariable(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", false
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	var v any
	if dec.Decode(&v) != nil {
		return "", false
	}
	if err := ensureJSONEOF(dec); err != nil {
		return "", false
	}
	switch x := v.(type) {
	case string:
		if len(x) == 0 || len(x) > 512 {
			return "", false
		}
		return x, true
	case json.Number:
		// Preserve the exact JSON number spelling rather than round-tripping via
		// float64; large GraphQL ID values must never lose precision before HMAC.
		v := x.String()
		if len(v) == 0 || len(v) > 128 {
			return "", false
		}
		return v, true
	default:
		return "", false
	}
}
func graphqlVariableSemantic(name, typ string) string {
	if isSensitiveFieldName(name) {
		return ""
	}
	if strings.Contains(strings.ToUpper(typ), "ID") {
		return name
	}
	return objectSemanticName(name)
}

type graphqlEnvelope struct {
	Query         string
	OperationName string
	Variables     map[string]json.RawMessage
	PersistedHash string
	HasPersisted  bool
}

func decodeGraphQLEnvelope(r *http.Request) (graphqlEnvelope, bool, error) {
	if r == nil || r.URL == nil {
		return graphqlEnvelope{}, false, nil
	}
	ct := strings.ToLower(normalizeMediaType(r.Header.Get("Content-Type")))
	if r.Method == http.MethodGet {
		q := r.URL.Query()
		query := q.Get("query")
		ext := q.Get("extensions")
		if query == "" && ext == "" {
			return graphqlEnvelope{}, false, nil
		}
		env := graphqlEnvelope{Query: query, OperationName: q.Get("operationName")}
		if raw := q.Get("variables"); raw != "" {
			if len(raw) > graphqlMaxQueryBytes {
				return env, true, errors.New("GraphQL variables exceed limit")
			}
			if err := json.Unmarshal([]byte(raw), &env.Variables); err != nil {
				return env, true, errors.New("invalid GraphQL variables JSON")
			}
		}
		if err := decodePersistedExtension([]byte(ext), &env); err != nil {
			return env, true, err
		}
		return env, true, nil
	}
	if r.Method != http.MethodPost {
		return graphqlEnvelope{}, false, nil
	}
	prefix := requestBodyPrefixFromRequest(r)
	truncated := requestBodyTruncatedFromRequest(r)
	if ct == "application/graphql" {
		if truncated {
			return graphqlEnvelope{}, true, errors.New("GraphQL document exceeds capture limit")
		}
		if len(prefix) == 0 {
			return graphqlEnvelope{}, true, errors.New("empty GraphQL request")
		}
		return graphqlEnvelope{Query: string(prefix)}, true, nil
	}
	if ct != "application/json" && ct != "application/graphql+json" {
		return graphqlEnvelope{}, false, nil
	}
	if len(prefix) == 0 {
		return graphqlEnvelope{}, false, nil
	}
	if truncated {
		return graphqlEnvelope{}, true, errors.New("GraphQL JSON envelope exceeds capture limit")
	}
	var raw struct {
		Query         string                     `json:"query"`
		OperationName string                     `json:"operationName"`
		Variables     map[string]json.RawMessage `json:"variables"`
		Extensions    json.RawMessage            `json:"extensions"`
	}
	dec := json.NewDecoder(strings.NewReader(string(prefix)))
	if err := dec.Decode(&raw); err != nil {
		return graphqlEnvelope{}, false, nil
	}
	if raw.Query == "" && len(raw.Extensions) == 0 {
		return graphqlEnvelope{}, false, nil
	}
	env := graphqlEnvelope{Query: raw.Query, OperationName: raw.OperationName, Variables: raw.Variables}
	if err := decodePersistedExtension(raw.Extensions, &env); err != nil {
		return env, true, err
	}
	return env, true, nil
}
func decodePersistedExtension(raw []byte, env *graphqlEnvelope) error {
	if len(raw) == 0 {
		return nil
	}
	var x struct {
		PersistedQuery *struct {
			Version    int    `json:"version"`
			SHA256Hash string `json:"sha256Hash"`
		} `json:"persistedQuery"`
	}
	if json.Unmarshal(raw, &x) != nil {
		return errors.New("invalid GraphQL extensions")
	}
	if x.PersistedQuery == nil {
		return nil
	}
	if x.PersistedQuery.Version != 1 {
		return errors.New("unsupported persistedQuery version")
	}
	h := strings.ToLower(strings.TrimSpace(x.PersistedQuery.SHA256Hash))
	if !validGraphQLHash(h) {
		return errors.New("invalid persistedQuery sha256Hash")
	}
	env.PersistedHash = h
	env.HasPersisted = true
	return nil
}

func graphqlViolation(endpointID, gqlID, opName, mode, typ, expected, observed, field string, at time.Time) GraphQLViolation {
	if opName != "" && !validGraphQLName(opName) {
		opName = ""
	}
	if len(expected) > 1024 {
		expected = expected[:1024]
	}
	if len(observed) > 1024 {
		observed = observed[:1024]
	}
	if field != "" && !validGraphQLFieldPath(field) {
		field = ""
	}
	return GraphQLViolation{Time: at, EndpointOperationID: endpointID, GraphQLOperationID: gqlID, OperationName: opName, Mode: mode, Type: typ, Expected: expected, Observed: observed, Field: field}
}
func containsField(xs []string, v string) bool {
	i := sort.SearchStrings(xs, v)
	return i < len(xs) && xs[i] == v
}
func graphqlSchemaAllowsField(contract GraphQLSchemaContract, operationType, path string) bool {
	parts := strings.Split(path, ".")
	if len(parts) < 2 || parts[0] != operationType {
		return false
	}
	current := contract.QueryRoot
	if operationType == "mutation" {
		current = contract.MutationRoot
	} else if operationType == "subscription" {
		current = contract.SubscriptionRoot
	}
	if current == "" {
		return false
	}
	for _, field := range parts[1:] {
		// Introspection topology is defined by the GraphQL specification rather
		// than ordinary application SDL. The explicit AllowIntrospection policy
		// remains the authority for these fields.
		if strings.HasPrefix(field, "__") {
			return true
		}
		fields := contract.FieldTypes[current]
		if fields == nil {
			// Scalars, unions, interfaces or other topology we did not normalize are
			// not guessed. Known object topology is still validated deterministically.
			return true
		}
		next, ok := fields[field]
		if !ok {
			return false
		}
		current = next
	}
	return true
}

func evaluateGraphQLPolicy(policy GraphQLPolicy, contract *GraphQLSchemaContract, analysis graphqlAnalysis, persisted bool, endpointID, gqlID string, at time.Time) []GraphQLViolation {
	var out []GraphQLViolation
	mode := policy.Mode
	if policy.MaxDepth > 0 && analysis.Depth > policy.MaxDepth {
		out = append(out, graphqlViolation(endpointID, gqlID, analysis.OperationName, mode, "DEPTH_LIMIT", strconv.Itoa(policy.MaxDepth), strconv.Itoa(analysis.Depth), "", at))
	}
	if policy.MaxComplexity > 0 && analysis.Complexity > policy.MaxComplexity {
		out = append(out, graphqlViolation(endpointID, gqlID, analysis.OperationName, mode, "COMPLEXITY_LIMIT", strconv.Itoa(policy.MaxComplexity), strconv.Itoa(analysis.Complexity), "", at))
	}
	if analysis.Introspection && !policy.AllowIntrospection {
		out = append(out, graphqlViolation(endpointID, gqlID, analysis.OperationName, mode, "INTROSPECTION_DISABLED", "false", "true", "", at))
	}
	if analysis.OperationType == "mutation" && !policy.AllowMutation {
		out = append(out, graphqlViolation(endpointID, gqlID, analysis.OperationName, mode, "MUTATION_DISABLED", "query-only", "mutation", "", at))
	}
	if analysis.OperationType == "subscription" && !policy.AllowSubscription {
		out = append(out, graphqlViolation(endpointID, gqlID, analysis.OperationName, mode, "SUBSCRIPTION_DISABLED", "query-only", "subscription", "", at))
	}
	if policy.RequirePersisted && !persisted {
		out = append(out, graphqlViolation(endpointID, gqlID, analysis.OperationName, mode, "PERSISTED_QUERY_REQUIRED", "persisted query", "inline query", "", at))
	}
	if contract != nil {
		for _, f := range analysis.Fields {
			if !graphqlSchemaAllowsField(*contract, analysis.OperationType, f) {
				v := graphqlViolation(endpointID, gqlID, analysis.OperationName, mode, "SCHEMA_FIELD_MISMATCH", "field declared by imported GraphQL SDL", f, f, at)
				v.SchemaContractID = contract.ID
				out = append(out, v)
			}
		}
	}
	for _, f := range analysis.Fields {
		if len(policy.AllowedFields) > 0 && !containsField(policy.AllowedFields, f) {
			out = append(out, graphqlViolation(endpointID, gqlID, analysis.OperationName, mode, "FIELD_NOT_ALLOWED", "allowed field", f, f, at))
		}
		if containsField(policy.DeniedFields, f) {
			out = append(out, graphqlViolation(endpointID, gqlID, analysis.OperationName, mode, "FIELD_DENIED", "field not present", f, f, at))
		}
	}
	return out
}

func (s *graphqlStore) wrap(site string, next http.Handler) http.Handler {
	if s == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		endpointID := apiOperationID(site, r.Method, normalizeAPIOperationPath(r.URL.Path))
		env, isGraphQL, decodeErr := decodeGraphQLEnvelope(r)
		if !isGraphQL {
			// A broad policy explicitly marks this normalized endpoint as GraphQL.
			// Once configured, malformed JSON cannot bypass GraphQL validation by
			// failing envelope discovery first.
			if _, configured := s.endpointFallbackPolicy(endpointID); !configured {
				next.ServeHTTP(w, r)
				return
			}
			ct := strings.ToLower(normalizeMediaType(r.Header.Get("Content-Type")))
			if r.Method != http.MethodPost || (ct != "application/json" && ct != "application/graphql+json") {
				next.ServeHTTP(w, r)
				return
			}
			decodeErr = errors.New("invalid GraphQL JSON envelope")
		}
		at := s.now().UTC()
		var analysis graphqlAnalysis
		var err error
		persistedKnown := false
		if decodeErr != nil {
			err = decodeErr
		} else if env.Query != "" {
			if len(env.Query) > graphqlMaxQueryBytes {
				err = errors.New("GraphQL document exceeds 64 KiB")
			} else {
				analysis, err = analyzeGraphQLDocument(env.Query, env.OperationName)
			}
			if err == nil && env.HasPersisted {
				sum := sha256.Sum256([]byte(env.Query))
				actual := hex.EncodeToString(sum[:])
				if actual != env.PersistedHash {
					err = errors.New("persistedQuery hash does not match query")
				} else {
					persistedKnown = true
				}
			}
		} else if env.HasPersisted {
			if p, ok := s.persistedProfile(env.PersistedHash); ok {
				analysis = analysisFromPersisted(p)
				persistedKnown = true
			} else {
				err = errors.New("unknown persistedQuery hash")
			}
		} else {
			err = errors.New("GraphQL query is required")
		}

		gqlID := ""
		if err == nil {
			gqlID = graphqlOperationID(endpointID, analysis.OperationType, analysis.OperationName, analysis.DocumentFingerprint)
		}
		policy, hasPolicy := s.runtimePolicy(endpointID, analysis.OperationName)
		if err != nil {
			policy, hasPolicy = s.endpointFallbackPolicy(endpointID)
		}
		mode := graphqlModeLearn
		if hasPolicy {
			mode = policy.Mode
		}
		var violations []GraphQLViolation
		if err != nil {
			violations = append(violations, graphqlViolation(endpointID, gqlID, env.OperationName, mode, "MALFORMED_GRAPHQL", "valid bounded GraphQL request", err.Error(), "", at))
		} else {
			if analysis.Depth > graphqlAbsoluteMaxDepth {
				violations = append(violations, graphqlViolation(endpointID, gqlID, analysis.OperationName, mode, "ABSOLUTE_DEPTH_LIMIT", strconv.Itoa(graphqlAbsoluteMaxDepth), strconv.Itoa(analysis.Depth), "", at))
			}
			if analysis.Complexity > graphqlAbsoluteMaxComplexity {
				violations = append(violations, graphqlViolation(endpointID, gqlID, analysis.OperationName, mode, "ABSOLUTE_COMPLEXITY_LIMIT", strconv.Itoa(graphqlAbsoluteMaxComplexity), strconv.Itoa(analysis.Complexity), "", at))
			}
			if hasPolicy {
				var contract *GraphQLSchemaContract
				if policy.SchemaContractID != "" {
					if c, ok := s.runtimeContract(policy.SchemaContractID); ok {
						contract = &c
					}
				}
				violations = append(violations, evaluateGraphQLPolicy(policy, contract, analysis, env.HasPersisted && persistedKnown, endpointID, gqlID, at)...)
			}
		}
		action := "learn"
		if hasPolicy && policy.Mode == graphqlModeDetect {
			action = "detect"
		}
		if hasPolicy && policy.Mode == graphqlModeEnforce && len(violations) > 0 {
			action = "block"
		}
		for i := range violations {
			violations[i].Action = action
		}
		s.recordViolations(violations)
		if err == nil {
			op := GraphQLOperation{ID: gqlID, EndpointOperationID: endpointID, Site: site, OperationType: analysis.OperationType, OperationName: analysis.OperationName, DocumentFingerprint: analysis.DocumentFingerprint, Depth: analysis.Depth, Complexity: analysis.Complexity, Fields: append([]string(nil), analysis.Fields...), Variables: graphqlVariablesList(analysis.VariableTypes), Introspection: analysis.Introspection, PersistedQueryHash: env.PersistedHash, FirstSeen: at, LastSeen: at, ExpiresAt: at.Add(graphqlOperationTTL)}
			var pp *GraphQLPersistedProfile
			if env.HasPersisted && persistedKnown && env.Query != "" {
				v := persistedFromAnalysis(env.PersistedHash, op, at)
				pp = &v
			}
			vars := make([]graphqlVariableSample, 0, len(env.Variables))
			for name, raw := range env.Variables {
				typ := analysis.VariableTypes[name]
				semantic := graphqlVariableSemantic(name, typ)
				if semantic == "" {
					continue
				}
				if value, ok := scalarGraphQLVariable(raw); ok {
					vars = append(vars, graphqlVariableSample{Name: name, SchemaType: typ, RawValue: value})
				}
			}
			if len(vars) > graphqlMaxVariables {
				vars = vars[:graphqlMaxVariables]
			}
			identity := objectRelationshipObservation{}
			if s.objectRelationships != nil {
				identity = s.objectRelationships.identityObservation(r)
			}
			s.enqueue(graphqlObservation{Operation: op, Persisted: pp, Variables: vars, Identity: identity, ObservedAt: at})
		}
		if action != "block" {
			next.ServeHTTP(w, r)
			return
		}
		status := policy.BlockStatus
		if status < 400 || status > 499 {
			status = http.StatusBadRequest
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"errors":[{"message":"request rejected by GraphQL security policy"}]}`))
	})
}

func (s *graphqlStore) save(configPath string) error {
	if s == nil {
		return nil
	}
	now := s.now().UTC()
	s.mu.Lock()
	s.pruneLocked(now)
	state := graphqlStateFile{Version: graphqlStateVersion, Saved: now, Operations: make([]GraphQLOperation, 0, len(s.operations)), Persisted: make([]GraphQLPersistedProfile, 0, len(s.persisted)), Contracts: make([]GraphQLSchemaContract, 0, len(s.contracts)), Policies: make([]GraphQLPolicy, 0, len(s.policies)), Violations: append([]GraphQLViolation(nil), s.violations...)}
	for _, v := range s.operations {
		state.Operations = append(state.Operations, cloneGraphQLOperation(*v))
	}
	for _, v := range s.persisted {
		state.Persisted = append(state.Persisted, cloneGraphQLPersisted(*v))
	}
	for _, v := range s.contracts {
		state.Contracts = append(state.Contracts, cloneGraphQLContract(*v))
	}
	for _, v := range s.policies {
		state.Policies = append(state.Policies, cloneGraphQLPolicy(*v))
	}
	s.mu.Unlock()
	return atomicWriteJSON(apiSecurityStatePath(configPath, "api-graphql.json"), state)
}
func (s *graphqlStore) load(configPath string) error {
	if s == nil {
		return nil
	}
	var state graphqlStateFile
	if err := readJSONIfExists(apiSecurityStatePath(configPath, "api-graphql.json"), &state); err != nil {
		return err
	}
	if state.Version != 0 && state.Version != graphqlStateVersion {
		return fmt.Errorf("unsupported GraphQL state version %d", state.Version)
	}
	now := s.now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, v := range state.Operations {
		if len(s.operations) >= graphqlMaxOperations || !validGraphQLOperation(v, now) {
			continue
		}
		cp := cloneGraphQLOperation(v)
		s.operations[v.ID] = &cp
	}
	for _, v := range state.Persisted {
		if len(s.persisted) >= graphqlMaxPersistedQueries || !validGraphQLPersisted(v, now) {
			continue
		}
		cp := cloneGraphQLPersisted(v)
		s.persisted[v.SHA256Hash] = &cp
	}
	for _, v := range state.Contracts {
		if len(s.contracts) >= graphqlMaxSchemaContracts || !validGraphQLSchemaContract(v) {
			continue
		}
		cp := cloneGraphQLContract(v)
		s.contracts[v.ID] = &cp
	}
	for _, v := range state.Policies {
		if len(s.policies) >= graphqlMaxPolicies || !validGraphQLPolicy(v) {
			continue
		}
		if v.SchemaContractID != "" {
			c := s.contracts[v.SchemaContractID]
			if c == nil || c.EndpointOperationID != v.EndpointOperationID {
				continue
			}
		}
		cp := cloneGraphQLPolicy(v)
		s.policies[v.ID] = &cp
	}
	if len(state.Violations) > graphqlMaxViolations {
		state.Violations = state.Violations[len(state.Violations)-graphqlMaxViolations:]
	}
	s.violations = s.violations[:0]
	for _, v := range state.Violations {
		if validGraphQLViolation(v) {
			s.violations = append(s.violations, v)
		}
	}
	s.rebuildRuntimeLocked()
	return nil
}

func decodeGraphQLPolicyRequest(w http.ResponseWriter, r *http.Request) (GraphQLPolicy, error) {
	var req struct {
		EndpointOperationID string   `json:"endpoint_operation_id"`
		OperationName       string   `json:"operation_name"`
		SchemaContractID    string   `json:"schema_contract_id"`
		MaxDepth            int      `json:"max_depth"`
		MaxComplexity       int      `json:"max_complexity"`
		AllowIntrospection  bool     `json:"allow_introspection"`
		AllowMutation       bool     `json:"allow_mutation"`
		AllowSubscription   bool     `json:"allow_subscription"`
		RequirePersisted    bool     `json:"require_persisted_queries"`
		AllowedFields       []string `json:"allowed_fields"`
		DeniedFields        []string `json:"denied_fields"`
		BlockStatus         int      `json:"block_status"`
		Reason              string   `json:"reason"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		return GraphQLPolicy{}, err
	}
	if err := ensureJSONEOF(dec); err != nil {
		return GraphQLPolicy{}, err
	}
	return GraphQLPolicy{EndpointOperationID: req.EndpointOperationID, OperationName: req.OperationName, SchemaContractID: req.SchemaContractID, MaxDepth: req.MaxDepth, MaxComplexity: req.MaxComplexity, AllowIntrospection: req.AllowIntrospection, AllowMutation: req.AllowMutation, AllowSubscription: req.AllowSubscription, RequirePersisted: req.RequirePersisted, AllowedFields: req.AllowedFields, DeniedFields: req.DeniedFields, BlockStatus: req.BlockStatus, Reason: req.Reason}, nil
}
func ensureJSONEOF(dec *json.Decoder) error {
	var extra any
	err := dec.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("multiple JSON values are not allowed")
	}
	return err
}

func (a *adminServer) persistGraphQLMutation(w http.ResponseWriter) bool {
	if a == nil || a.srv == nil || a.srv.graphql == nil {
		http.Error(w, "GraphQL store unavailable", http.StatusServiceUnavailable)
		return false
	}
	if err := a.srv.graphql.save(a.srv.configPath); err != nil {
		http.Error(w, "GraphQL persistence failed: "+err.Error(), http.StatusInternalServerError)
		return false
	}
	return true
}
func (a *adminServer) handleGraphQLOperations(w http.ResponseWriter, r *http.Request) {
	rows := a.srv.graphql.operationsSnapshot()
	a.audit.add(who(r).user, "api_graphql.operations_view", fmt.Sprintf("count=%d", len(rows)))
	writeJSON(w, rows)
}
func (a *adminServer) handleGraphQLPersisted(w http.ResponseWriter, r *http.Request) {
	rows := a.srv.graphql.persistedSnapshot()
	a.audit.add(who(r).user, "api_graphql.persisted_view", fmt.Sprintf("count=%d", len(rows)))
	writeJSON(w, rows)
}
func (a *adminServer) handleGraphQLSchemaContracts(w http.ResponseWriter, r *http.Request) {
	rows := a.srv.graphql.contractsSnapshot()
	a.audit.add(who(r).user, "api_graphql.schema_contracts_view", fmt.Sprintf("count=%d", len(rows)))
	writeJSON(w, rows)
}

func (a *adminServer) handleGraphQLSchemaContractImport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name                string `json:"name"`
		EndpointOperationID string `json:"endpoint_operation_id"`
		SDL                 string `json:"sdl"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 512<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if err := ensureJSONEOF(dec); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	row, err := a.srv.graphql.upsertSchemaContract(req.Name, req.EndpointOperationID, req.SDL, who(r).user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	if !a.persistGraphQLMutation(w) {
		return
	}
	a.audit.add(who(r).user, "api_graphql.schema_contract_import", row.ID+" "+row.Name)
	writeJSON(w, row)
}

func (a *adminServer) handleGraphQLSchemaContractDelete(w http.ResponseWriter, r *http.Request) {
	row, err := a.srv.graphql.deleteSchemaContract(r.PathValue("contract_id"))
	if err != nil {
		status := http.StatusNotFound
		if errors.Is(err, errGraphQLSchemaContractInUse) {
			status = http.StatusConflict
		}
		http.Error(w, err.Error(), status)
		return
	}
	if !a.persistGraphQLMutation(w) {
		return
	}
	a.audit.add(who(r).user, "api_graphql.schema_contract_delete", row.ID)
	writeJSON(w, map[string]any{"ok": true, "id": row.ID})
}

func (a *adminServer) handleGraphQLPolicies(w http.ResponseWriter, r *http.Request) {
	rows := a.srv.graphql.policiesSnapshot()
	a.audit.add(who(r).user, "api_graphql.policies_view", fmt.Sprintf("count=%d", len(rows)))
	writeJSON(w, rows)
}
func (a *adminServer) handleGraphQLPolicyUpsert(w http.ResponseWriter, r *http.Request) {
	req, err := decodeGraphQLPolicyRequest(w, r)
	if err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	p, err := a.srv.graphql.upsertPolicy(req, who(r).user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	if !a.persistGraphQLMutation(w) {
		return
	}
	a.audit.add(who(r).user, "api_graphql.policy_upsert", p.EndpointOperationID+" "+p.OperationName)
	writeJSON(w, p)
}
func (a *adminServer) handleGraphQLPolicyMode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Mode   string `json:"mode"`
		Reason string `json:"reason"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<14))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if err := ensureJSONEOF(dec); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	p, err := a.srv.graphql.setMode(r.PathValue("policy_id"), req.Mode, who(r).user, req.Reason)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if !a.persistGraphQLMutation(w) {
		return
	}
	a.audit.add(who(r).user, "api_graphql.policy_mode", p.ID+" -> "+p.Mode)
	writeJSON(w, p)
}
func (a *adminServer) handleGraphQLPolicyDelete(w http.ResponseWriter, r *http.Request) {
	p, err := a.srv.graphql.deletePolicy(r.PathValue("policy_id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if !a.persistGraphQLMutation(w) {
		return
	}
	a.audit.add(who(r).user, "api_graphql.policy_delete", p.ID)
	writeJSON(w, map[string]any{"ok": true, "id": p.ID})
}
func (a *adminServer) handleGraphQLViolations(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	rows := a.srv.graphql.violationsSnapshot(limit)
	a.audit.add(who(r).user, "api_graphql.violations_view", fmt.Sprintf("count=%d", len(rows)))
	writeJSON(w, rows)
}
func (a *adminServer) handleGraphQLStatus(w http.ResponseWriter, r *http.Request) {
	st := a.srv.graphql.status()
	a.audit.add(who(r).user, "api_graphql.status_view", fmt.Sprintf("operations=%d", st.OperationCount))
	writeJSON(w, st)
}
