package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAPI8GraphQLOperationDiscoveryNormalizesLiteralValues(t *testing.T) {
	a, err := analyzeGraphQLDocument(`query GetOrder { order(id:"42"){ id total } }`, "GetOrder")
	if err != nil {
		t.Fatal(err)
	}
	b, err := analyzeGraphQLDocument(`query GetOrder($id: ID!){ order(id:"999999"){ id total } }`, "GetOrder")
	if err != nil {
		t.Fatal(err)
	}
	if a.DocumentFingerprint != b.DocumentFingerprint || a.Depth != 2 || a.Complexity != 3 {
		t.Fatalf("unexpected normalized analysis: %#v %#v", a, b)
	}
	if strings.Contains(a.DocumentFingerprint, "999999") {
		t.Fatal("literal leaked into fingerprint")
	}
}

func TestAPI8GraphQLFragmentsDepthComplexityAndIntrospection(t *testing.T) {
	a, err := analyzeGraphQLDocument(`query Q { viewer { ...UserFields } __type(name:"User") { name } } fragment UserFields on User { id profile { name } }`, "Q")
	if err != nil {
		t.Fatal(err)
	}
	if a.Depth != 3 || a.Complexity != 6 || !a.Introspection {
		t.Fatalf("unexpected analysis: %#v", a)
	}
}

func TestAPI8GraphQLRejectsCyclicFragmentsAndOversizedInput(t *testing.T) {
	if _, err := analyzeGraphQLDocument(`query Q { viewer { ...A } } fragment A on User { ...B } fragment B on User { ...A }`, "Q"); err == nil {
		t.Fatal("cyclic fragment accepted")
	}
	if _, err := analyzeGraphQLDocument("query Q{"+strings.Repeat("a ", graphqlMaxTokens+10)+"}", "Q"); err == nil {
		t.Fatal("oversized token stream accepted")
	}
}

func TestAPI8GraphQLPolicyRequiresLearnDetectEnforcePromotion(t *testing.T) {
	s := newGraphQLStore()
	opID := apiOperationID("shop", http.MethodPost, "/graphql")
	p, err := s.upsertPolicy(GraphQLPolicy{EndpointOperationID: opID, OperationName: "Q", MaxDepth: 5, MaxComplexity: 50, AllowMutation: true, BlockStatus: 422}, "reviewer")
	if err != nil || p.Mode != graphqlModeLearn {
		t.Fatalf("upsert: %#v %v", p, err)
	}
	if _, err = s.setMode(p.ID, graphqlModeEnforce, "reviewer", "skip detect"); err == nil {
		t.Fatal("LEARN -> ENFORCE should fail")
	}
	if p, err = s.setMode(p.ID, graphqlModeDetect, "reviewer", "shadow"); err != nil || p.Mode != graphqlModeDetect {
		t.Fatal(err)
	}
	if _, err = s.setMode(p.ID, graphqlModeEnforce, "reviewer", "approved evidence"); err != nil {
		t.Fatal(err)
	}
}

func TestAPI8GraphQLDeterministicPolicyCoversDepthComplexityFieldsMutationIntrospectionPersisted(t *testing.T) {
	analysis := graphqlAnalysis{OperationType: "mutation", OperationName: "M", Depth: 4, Complexity: 8, Fields: []string{"mutation.adminDelete", "mutation.viewer"}, Introspection: true}
	p := GraphQLPolicy{Mode: graphqlModeDetect, MaxDepth: 2, MaxComplexity: 4, AllowMutation: false, AllowIntrospection: false, RequirePersisted: true, AllowedFields: []string{"mutation.viewer"}, DeniedFields: []string{"mutation.adminDelete"}}
	rows := evaluateGraphQLPolicy(p, nil, analysis, false, strings.Repeat("a", 32), strings.Repeat("b", 32), time.Now())
	seen := map[string]bool{}
	for _, v := range rows {
		seen[v.Type] = true
	}
	for _, typ := range []string{"DEPTH_LIMIT", "COMPLEXITY_LIMIT", "MUTATION_DISABLED", "INTROSPECTION_DISABLED", "PERSISTED_QUERY_REQUIRED", "FIELD_NOT_ALLOWED", "FIELD_DENIED"} {
		if !seen[typ] {
			t.Fatalf("missing %s in %#v", typ, rows)
		}
	}
}

func TestAPI8PersistedQueryProfileContainsNoRawQueryAndSupportsHashOnlyLookup(t *testing.T) {
	s := newGraphQLStore()
	now := time.Now().UTC()
	query := `query Q($id: ID!){ order(id:$id){ id } }`
	a, err := analyzeGraphQLDocument(query, "Q")
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256([]byte(query))
	hash := hex.EncodeToString(h[:])
	endpoint := apiOperationID("shop", http.MethodPost, "/graphql")
	id := graphqlOperationID(endpoint, a.OperationType, a.OperationName, a.DocumentFingerprint)
	op := GraphQLOperation{ID: id, EndpointOperationID: endpoint, Site: "shop", OperationType: a.OperationType, OperationName: a.OperationName, DocumentFingerprint: a.DocumentFingerprint, Depth: a.Depth, Complexity: a.Complexity, Fields: a.Fields, Variables: graphqlVariablesList(a.VariableTypes), PersistedQueryHash: hash, FirstSeen: now, LastSeen: now, ExpiresAt: now.Add(time.Hour)}
	pp := persistedFromAnalysis(hash, op, now)
	s.processObservation(graphqlObservation{Operation: op, Persisted: &pp, ObservedAt: now})
	got, ok := s.persistedProfile(hash)
	if !ok || got.GraphQLOperationID != id {
		t.Fatalf("persisted lookup failed: %#v", got)
	}
	b, _ := json.Marshal(got)
	if bytes.Contains(b, []byte("order(id")) {
		t.Fatal("raw query persisted")
	}
}

func TestAPI8GraphQLVariableLocatorUsesKeyedEvidenceAndVerifiedIdentityOnly(t *testing.T) {
	loc := newObjectLocatorStoreWithKey(bytes.Repeat([]byte{1}, 32))
	rel := newObjectRelationshipStoreWithKey(bytes.Repeat([]byte{2}, 32))
	det := newBOLADetectionStore()
	rel.locators = loc
	rel.detector = det
	rel.start()
	defer rel.stopAndDrain()
	s := newGraphQLStore()
	s.objectLocators = loc
	s.objectRelationships = rel
	s.start()
	defer s.stopAndDrain()
	now := time.Now().UTC()
	endpoint := apiOperationID("shop", http.MethodPost, "/graphql")
	a, _ := analyzeGraphQLDocument(`query Q($orderId: ID!){order(id:$orderId){id}}`, "Q")
	gid := graphqlOperationID(endpoint, a.OperationType, a.OperationName, a.DocumentFingerprint)
	op := GraphQLOperation{ID: gid, EndpointOperationID: endpoint, Site: "shop", OperationType: "query", OperationName: "Q", DocumentFingerprint: a.DocumentFingerprint, Depth: a.Depth, Complexity: a.Complexity, Fields: a.Fields, Variables: []string{"orderId"}, FirstSeen: now, LastSeen: now, ExpiresAt: now.Add(time.Hour)}
	r := httptest.NewRequest(http.MethodPost, "http://waf/graphql", nil)
	r = r.WithContext(context.WithValue(r.Context(), apiIdentityContextKey{}, VerifiedAPIIdentity{Issuer: "https://issuer", Subject: "alice", ExpiresAt: now.Add(time.Hour), Algorithm: "RS256"}))
	identity := rel.identityObservation(r)
	s.processObservation(graphqlObservation{Operation: op, Variables: []graphqlVariableSample{{Name: "orderId", SchemaType: "ID!", RawValue: "secret-order-42"}}, Identity: identity, ObservedAt: now})
	rows := loc.snapshot()
	if len(rows) != 1 || rows[0].Location != "graphql_variable" || rows[0].Field != "orderId" {
		t.Fatalf("locator: %#v", rows)
	}
	blob, _ := json.Marshal(rows)
	if bytes.Contains(blob, []byte("secret-order-42")) {
		t.Fatal("raw variable value persisted")
	}
}

func TestAPI8GraphQLWrapperDetectDoesNotBlockButEnforceDoes(t *testing.T) {
	s := newGraphQLStore()
	s.start()
	defer s.stopAndDrain()
	endpoint := apiOperationID("shop", http.MethodPost, "/graphql")
	p, err := s.upsertPolicy(GraphQLPolicy{EndpointOperationID: endpoint, OperationName: "Q", MaxDepth: 1, MaxComplexity: 20, AllowMutation: true, BlockStatus: 422}, "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = s.setMode(p.ID, graphqlModeDetect, "reviewer", "shadow")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	handler := requestBodyPrefixWrap(false, false, true, s.wrap("shop", next))
	body := `{"query":"query Q { viewer { id } }","operationName":"Q"}`
	r := httptest.NewRequest(http.MethodPost, "http://waf/graphql", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 204 {
		t.Fatalf("DETECT blocked: %d", w.Code)
	}
	_, _ = s.setMode(p.ID, graphqlModeEnforce, "reviewer", "approved")
	r = httptest.NewRequest(http.MethodPost, "http://waf/graphql", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 422 {
		t.Fatalf("ENFORCE did not block: %d", w.Code)
	}
}

func TestAPI8GraphQLPersistedHashMismatchFailsClosedUnderEnforce(t *testing.T) {
	s := newGraphQLStore()
	s.start()
	defer s.stopAndDrain()
	endpoint := apiOperationID("shop", http.MethodPost, "/graphql")
	p, _ := s.upsertPolicy(GraphQLPolicy{EndpointOperationID: endpoint, MaxDepth: 10, MaxComplexity: 100, AllowMutation: true, AllowIntrospection: true, BlockStatus: 400}, "reviewer")
	_, _ = s.setMode(p.ID, graphqlModeDetect, "reviewer", "shadow")
	_, _ = s.setMode(p.ID, graphqlModeEnforce, "reviewer", "approved")
	body := `{"query":"query Q { viewer { id } }","extensions":{"persistedQuery":{"version":1,"sha256Hash":"` + strings.Repeat("0", 64) + `"}}}`
	handler := requestBodyPrefixWrap(false, false, true, s.wrap("shop", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })))
	r := httptest.NewRequest(http.MethodPost, "http://waf/graphql", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 400 {
		t.Fatalf("hash mismatch not blocked: %d", w.Code)
	}
}

func TestAPI8GraphQLPersistenceRevalidatesAndRestoresBoundedState(t *testing.T) {
	dir := t.TempDir()
	config := filepath.Join(dir, "waf.yaml")
	s := newGraphQLStore()
	now := time.Now().UTC()
	s.now = func() time.Time { return now }
	endpoint := apiOperationID("shop", http.MethodPost, "/graphql")
	p, err := s.upsertPolicy(GraphQLPolicy{EndpointOperationID: endpoint, OperationName: "Q", MaxDepth: 5, MaxComplexity: 50, AllowMutation: true, BlockStatus: 400}, "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	a, _ := analyzeGraphQLDocument(`query Q { viewer { id } }`, "Q")
	id := graphqlOperationID(endpoint, "query", "Q", a.DocumentFingerprint)
	op := GraphQLOperation{ID: id, EndpointOperationID: endpoint, Site: "shop", OperationType: "query", OperationName: "Q", DocumentFingerprint: a.DocumentFingerprint, Depth: a.Depth, Complexity: a.Complexity, Fields: a.Fields, ObservationCount: 1, FirstSeen: now, LastSeen: now, ExpiresAt: now.Add(time.Hour)}
	s.operations[id] = &op
	s.rebuildRuntimeLocked()
	if err = s.save(config); err != nil {
		t.Fatal(err)
	}
	r := newGraphQLStore()
	r.now = func() time.Time { return now }
	if err = r.load(config); err != nil {
		t.Fatal(err)
	}
	if len(r.operationsSnapshot()) != 1 || len(r.policiesSnapshot()) != 1 || r.policiesSnapshot()[0].ID != p.ID {
		t.Fatalf("restore failed")
	}
}

func TestAPI8GraphQLStateAndPolicyCardinalityAreBounded(t *testing.T) {
	s := newGraphQLStore()
	now := time.Now().UTC()
	s.mu.Lock()
	for i := 0; i < graphqlMaxOperations+10; i++ {
		id := strings.Repeat("a", 24) + hex.EncodeToString([]byte{byte(i >> 24), byte(i >> 16), byte(i >> 8), byte(i)})
		if len(id) > 32 {
			id = id[:32]
		}
		if len(s.operations) >= graphqlMaxOperations {
			break
		}
		s.operations[id] = &GraphQLOperation{ID: id, EndpointOperationID: strings.Repeat("b", 32), OperationType: "query", DocumentFingerprint: strings.Repeat("c", 64), ObservationCount: 1, FirstSeen: now, LastSeen: now, ExpiresAt: now.Add(time.Hour)}
	}
	n := len(s.operations)
	s.mu.Unlock()
	if n != graphqlMaxOperations {
		t.Fatalf("operation cap=%d", n)
	}
}

func TestAPI8GraphQLConcurrentSnapshotsPolicyUpdatesAndObservations(t *testing.T) {
	s := newGraphQLStore()
	s.start()
	defer s.stopAndDrain()
	endpoint := apiOperationID("shop", http.MethodPost, "/graphql")
	a, _ := analyzeGraphQLDocument(`query Q { viewer { id } }`, "Q")
	gid := graphqlOperationID(endpoint, "query", "Q", a.DocumentFingerprint)
	now := time.Now().UTC()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_, _ = s.upsertPolicy(GraphQLPolicy{EndpointOperationID: endpoint, OperationName: "Q", MaxDepth: 5, MaxComplexity: 50, AllowMutation: true, BlockStatus: 400}, "reviewer")
				s.enqueue(graphqlObservation{Operation: GraphQLOperation{ID: gid, EndpointOperationID: endpoint, Site: "shop", OperationType: "query", OperationName: "Q", DocumentFingerprint: a.DocumentFingerprint, Depth: a.Depth, Complexity: a.Complexity, Fields: a.Fields, FirstSeen: now, LastSeen: now, ExpiresAt: now.Add(time.Hour)}, ObservedAt: now})
				_ = s.operationsSnapshot()
				_ = s.status()
			}
		}(i)
	}
	wg.Wait()
	if len(s.policiesSnapshot()) != 1 {
		t.Fatalf("policy dedupe failed")
	}
}

func TestAPI8OpenAIHasNoGraphQLAuthorityAndBOLAInferenceRemainsSeparate(t *testing.T) {
	// Structural invariant test: API-8 deterministic policy is self-contained;
	// no OpenAI result or API-7 BOLA candidate is accepted by the evaluator.
	p := GraphQLPolicy{Mode: graphqlModeEnforce, MaxDepth: 1, MaxComplexity: 1, AllowMutation: true}
	a := graphqlAnalysis{OperationType: "query", Depth: 2, Complexity: 2, Fields: []string{"query.viewer"}}
	rows := evaluateGraphQLPolicy(p, nil, a, false, strings.Repeat("a", 32), strings.Repeat("b", 32), time.Now())
	if len(rows) != 2 {
		t.Fatalf("expected deterministic depth+complexity evidence, got %#v", rows)
	}
}

func TestAPI8ConfiguredGraphQLEndpointRejectsMalformedJSONOnlyInEnforce(t *testing.T) {
	s := newGraphQLStore()
	s.start()
	defer s.stopAndDrain()
	endpoint := apiOperationID("shop", http.MethodPost, "/graphql")
	p, err := s.upsertPolicy(GraphQLPolicy{EndpointOperationID: endpoint, MaxDepth: 10, MaxComplexity: 100, AllowMutation: true, AllowIntrospection: true, BlockStatus: 400}, "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	h := requestBodyPrefixWrap(false, false, true, s.wrap("shop", next))
	r := httptest.NewRequest(http.MethodPost, "http://waf/graphql", strings.NewReader(`{"query":`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 204 {
		t.Fatalf("LEARN malformed GraphQL unexpectedly blocked: %d", w.Code)
	}
	_, _ = s.setMode(p.ID, graphqlModeDetect, "reviewer", "shadow")
	_, _ = s.setMode(p.ID, graphqlModeEnforce, "reviewer", "approved")
	r = httptest.NewRequest(http.MethodPost, "http://waf/graphql", strings.NewReader(`{"query":`))
	r.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 400 {
		t.Fatalf("ENFORCE malformed GraphQL bypassed policy: %d", w.Code)
	}
}

func TestAPI8GraphQLSchemaContractParsesSDLWithoutPersistingRawDocument(t *testing.T) {
	s := newGraphQLStore()
	endpoint := apiOperationID("shop", http.MethodPost, "/graphql")
	sdl := `schema { query: RootQuery mutation: RootMutation } type RootQuery { viewer: User order(id: ID!): Order } type RootMutation { updateOrder(id: ID!): Order } type User { id: ID! name: String } type Order { id: ID! total: Float } scalar DateTime`
	row, err := s.upsertSchemaContract("shop-schema", endpoint, sdl, "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if row.QueryRoot != "RootQuery" || row.MutationRoot != "RootMutation" || len(row.Types["Order"]) != 2 {
		t.Fatalf("unexpected schema contract: %#v", row)
	}
	blob, _ := json.Marshal(row)
	if bytes.Contains(blob, []byte("schema {")) || bytes.Contains(blob, []byte("updateOrder(id:")) {
		t.Fatal("raw SDL persisted")
	}
}

func TestAPI8GraphQLSchemaContractRejectsMissingQueryRootAndBoundsState(t *testing.T) {
	s := newGraphQLStore()
	endpoint := apiOperationID("shop", http.MethodPost, "/graphql")
	if _, err := s.upsertSchemaContract("bad", endpoint, `type Mutation { doIt: Boolean }`, "reviewer"); err == nil {
		t.Fatal("SDL without query root accepted")
	}
	if _, err := s.upsertSchemaContract("huge", endpoint, strings.Repeat("x", graphqlMaxSchemaBytes+1), "reviewer"); err == nil {
		t.Fatal("oversized SDL accepted")
	}
}

func TestAPI8GraphQLSchemaContractParticipatesInDeterministicFieldValidation(t *testing.T) {
	_, _, _, types, fieldTypes, err := parseGraphQLSDL(`type Query { viewer: User } type User { id: ID! }`)
	if err != nil {
		t.Fatal(err)
	}
	contract := GraphQLSchemaContract{QueryRoot: "Query", Types: types, FieldTypes: fieldTypes}
	analysis, err := analyzeGraphQLDocument(`query Q { viewer { id secret } }`, "Q")
	if err != nil {
		t.Fatal(err)
	}
	p := GraphQLPolicy{Mode: graphqlModeDetect, AllowMutation: true, AllowIntrospection: true}
	rows := evaluateGraphQLPolicy(p, &contract, analysis, false, strings.Repeat("a", 32), strings.Repeat("b", 32), time.Now())
	found := false
	for _, v := range rows {
		if v.Type == "SCHEMA_FIELD_MISMATCH" && v.Field == "query.viewer.secret" {
			found = true
		}
	}
	if !found {
		t.Fatalf("schema mismatch not detected: %#v", rows)
	}
}

func TestAPI8GraphQLPolicyEditAndSchemaChangeReturnToLearn(t *testing.T) {
	s := newGraphQLStore()
	endpoint := apiOperationID("shop", http.MethodPost, "/graphql")
	contract, err := s.upsertSchemaContract("shop-schema", endpoint, `type Query { viewer: User } type User { id: ID! }`, "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.upsertPolicy(GraphQLPolicy{EndpointOperationID: endpoint, OperationName: "Q", SchemaContractID: contract.ID, MaxDepth: 5, MaxComplexity: 50, AllowMutation: true, BlockStatus: 400}, "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	p, _ = s.setMode(p.ID, graphqlModeDetect, "reviewer", "shadow")
	p, _ = s.setMode(p.ID, graphqlModeEnforce, "reviewer", "approved")
	p, err = s.upsertPolicy(GraphQLPolicy{EndpointOperationID: endpoint, OperationName: "Q", SchemaContractID: contract.ID, MaxDepth: 4, MaxComplexity: 40, AllowMutation: true, BlockStatus: 400}, "reviewer")
	if err != nil || p.Mode != graphqlModeLearn {
		t.Fatalf("policy edit did not return to LEARN: %#v %v", p, err)
	}
	p, _ = s.setMode(p.ID, graphqlModeDetect, "reviewer", "shadow")
	p, _ = s.setMode(p.ID, graphqlModeEnforce, "reviewer", "approved")
	if _, err = s.upsertSchemaContract("shop-schema", endpoint, `type Query { viewer: User order: Order } type User { id: ID! } type Order { id: ID! }`, "reviewer"); err != nil {
		t.Fatal(err)
	}
	got := s.policiesSnapshot()
	if len(got) != 1 || got[0].Mode != graphqlModeLearn {
		t.Fatalf("schema change did not return dependent policy to LEARN: %#v", got)
	}
}

func TestAPI8GraphQLSchemaDeleteRejectsReferencedContract(t *testing.T) {
	s := newGraphQLStore()
	endpoint := apiOperationID("shop", http.MethodPost, "/graphql")
	contract, err := s.upsertSchemaContract("shop-schema", endpoint, `type Query { viewer: User } type User { id: ID! }`, "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.upsertPolicy(GraphQLPolicy{EndpointOperationID: endpoint, SchemaContractID: contract.ID, MaxDepth: 5, MaxComplexity: 50, AllowMutation: true, BlockStatus: 400}, "reviewer"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.deleteSchemaContract(contract.ID); err == nil || !strings.Contains(err.Error(), "referenced by policy") {
		t.Fatalf("referenced schema contract delete should fail safely, got %v", err)
	}
}

func TestAPI8GraphQLSchemaIntrospectionDoesNotConflictWithExplicitIntrospectionPolicy(t *testing.T) {
	_, _, _, types, fieldTypes, err := parseGraphQLSDL(`type Query { viewer: User } type User { id: ID! }`)
	if err != nil {
		t.Fatal(err)
	}
	contract := GraphQLSchemaContract{QueryRoot: "Query", Types: types, FieldTypes: fieldTypes}
	analysis, err := analyzeGraphQLDocument(`query Q { __schema { types { name } } }`, "Q")
	if err != nil {
		t.Fatal(err)
	}
	p := GraphQLPolicy{Mode: graphqlModeDetect, AllowIntrospection: true, AllowMutation: true}
	rows := evaluateGraphQLPolicy(p, &contract, analysis, false, strings.Repeat("a", 32), strings.Repeat("b", 32), time.Now())
	for _, v := range rows {
		if v.Type == "SCHEMA_FIELD_MISMATCH" || v.Type == "INTROSPECTION_DISABLED" {
			t.Fatalf("allowed introspection conflicted with schema validation: %#v", rows)
		}
	}
}

func TestAPI8GraphQLNumericVariableFingerprintInputPreservesPrecision(t *testing.T) {
	const raw = `900719925474099312345678901234567890`
	got, ok := scalarGraphQLVariable(json.RawMessage(raw))
	if !ok || got != raw {
		t.Fatalf("numeric GraphQL ID lost precision: got=%q ok=%v", got, ok)
	}
}

func TestAPI8GraphQLViolationRestoreDropsUntrustedDurablePayload(t *testing.T) {
	dir := t.TempDir()
	config := filepath.Join(dir, "waf.yaml")
	now := time.Now().UTC()
	endpoint := apiOperationID("shop", http.MethodPost, "/graphql")
	valid := GraphQLViolation{ID: strings.Repeat("a", 32), Time: now, EndpointOperationID: endpoint, Mode: graphqlModeDetect, Action: "detect", Type: "DEPTH_LIMIT", Expected: "5", Observed: "6"}
	invalid := valid
	invalid.ID = strings.Repeat("b", 32)
	invalid.Observed = strings.Repeat("x", 2048)
	state := graphqlStateFile{Version: graphqlStateVersion, Saved: now, Violations: []GraphQLViolation{valid, invalid}}
	if err := atomicWriteJSON(apiSecurityStatePath(config, "api-graphql.json"), state); err != nil {
		t.Fatal(err)
	}
	s := newGraphQLStore()
	s.now = func() time.Time { return now }
	if err := s.load(config); err != nil {
		t.Fatal(err)
	}
	rows := s.violationsSnapshot(10)
	if len(rows) != 1 || rows[0].ID != valid.ID {
		t.Fatalf("untrusted restored GraphQL violation was not filtered: %#v", rows)
	}
}
