package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func api71Store() *objectLocatorStore {
	return newObjectLocatorStoreWithKey([]byte("0123456789abcdef0123456789abcdef"))
}

func TestAPI71NormalizedPathLocatorDoesNotExpandWithObjectValue(t *testing.T) {
	s := api71Store()
	at := time.Date(2026, 9, 24, 13, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return at }
	ops := newAPIOperationStore()
	for _, raw := range []string{"/orders/1", "/orders/999999", "/orders/42"} {
		op := ops.note("shop", "api.test", http.MethodGet, raw, "", "none", 200)
		samples := collectSchemaSamplesFromObservation(raw, "", "", apiObservationMeta{})
		s.noteObservation(op, samples)
	}
	rows := s.snapshot()
	if len(rows) != 1 {
		t.Fatalf("locators=%d want 1: %#v", len(rows), rows)
	}
	if rows[0].OperationID != apiOperationID("shop", http.MethodGet, "/orders/{id}") || rows[0].Location != "path" || rows[0].Field != "param_1" {
		t.Fatalf("unexpected normalized locator: %#v", rows[0])
	}
	if len(rows[0].ValueFingerprints) != 3 {
		t.Fatalf("keyed value fingerprints=%d want 3", len(rows[0].ValueFingerprints))
	}
	b, _ := json.Marshal(rows)
	for _, raw := range []string{"999999", "/orders/1", "/orders/42"} {
		if strings.Contains(string(b), raw) {
			t.Fatalf("durable locator leaked raw object value/path %q: %s", raw, b)
		}
	}
}

func TestAPI71TypedQueryBodyDiscoveryAndGraphQLVariablesDeferred(t *testing.T) {
	s := api71Store()
	s.now = func() time.Time { return time.Date(2026, 9, 24, 13, 10, 0, 0, time.UTC) }
	op := apiOperation{ID: apiOperationID("shop", http.MethodPost, "/orders/search")}
	samples := []schemaFieldSample{
		{Path: "order_id", Location: "query", Type: "integer", objectValue: "123"},
		{Path: "customerId", Location: "body", Type: "string", objectValue: "cust-9"},
		{Path: "description", Location: "body", Type: "string", objectValue: "id-like-but-not-locator"},
		{Path: "variables.orderId", Location: "body", Type: "string", objectValue: "graphql-1"},
	}
	s.noteObservation(op, samples)
	rows := s.snapshot()
	if len(rows) != 2 {
		t.Fatalf("locators=%d want 2: %#v", len(rows), rows)
	}
	for _, v := range rows {
		if strings.HasPrefix(v.Field, "variables.") || v.Field == "description" {
			t.Fatalf("deferred/non-locator field discovered: %#v", v)
		}
		if !containsLocatorSource(v.Sources, objectLocatorSourceAPI2) {
			t.Fatalf("missing API-2 source: %#v", v)
		}
	}
}

func TestAPI71SelfReportedTenantHeadersAreNeverLocatorAuthority(t *testing.T) {
	s := api71Store()
	s.now = func() time.Time { return time.Date(2026, 9, 24, 13, 20, 0, 0, time.UTC) }
	op := apiOperation{ID: apiOperationID("shop", http.MethodGet, "/orders")}
	s.noteObservation(op, []schemaFieldSample{
		{Path: "x-tenant-id", Location: "header", Type: "string", objectValue: "tenant-attacker"},
		{Path: "x-owner-id", Location: "header", Type: "string", objectValue: "owner-attacker"},
	})
	if got := len(s.snapshot()); got != 0 {
		t.Fatalf("client-reported ownership/tenant headers created %d locators", got)
	}
}

func TestAPI71MatchedOpenAPIContractSeedsDeclaredLocators(t *testing.T) {
	store := api71Store()
	store.now = func() time.Time { return time.Date(2026, 9, 24, 13, 30, 0, 0, time.UTC) }
	opID := apiOperationID("shop", http.MethodPatch, "/orders/{id}")
	contracts := newContractStore()
	contracts.operations["v1"] = []ContractOperation{{
		ID: "contract-op", VersionID: "v1", Method: http.MethodPatch, Path: "/orders/{orderId}",
		Parameters:    []ContractParameter{{Name: "orderId", In: "path", Required: true, Type: "string"}, {Name: "include", In: "query", Type: "string"}},
		RequestFields: []ContractSchemaField{{Path: "customer_id", Type: "string"}, {Path: "note", Type: "string"}, {Path: "variables.orderId", Type: "string"}},
	}}
	contracts.bindings["v1"] = []OperationBinding{{ContractVersionID: "v1", ContractOperationID: "contract-op", APIOperationID: opID, Status: "MATCHED"}}
	store.refreshContractSources(contracts)
	rows := store.snapshot()
	if len(rows) != 2 {
		t.Fatalf("contract locators=%d want 2: %#v", len(rows), rows)
	}
	for _, v := range rows {
		if !containsLocatorSource(v.Sources, objectLocatorSourceAPI3) || v.Confidence < 0.95 {
			t.Fatalf("contract source/confidence missing: %#v", v)
		}
		if v.Field == "include" || strings.HasPrefix(v.Field, "variables.") {
			t.Fatalf("non-object/deferred field admitted: %#v", v)
		}
	}
}

func TestAPI71OperatorIncludeSuppressDeleteAndPersistence(t *testing.T) {
	s := api71Store()
	at := time.Date(2026, 9, 24, 13, 40, 0, 0, time.UTC)
	s.now = func() time.Time { return at }
	opID := apiOperationID("shop", http.MethodGet, "/legacy")
	include, err := s.upsertOverride(ObjectLocatorOverride{OperationID: opID, Location: "query", Field: "legacy_ref", SchemaType: "string", SemanticName: "legacy_ref", Action: objectLocatorOverrideInclude})
	if err != nil {
		t.Fatal(err)
	}
	rows := s.snapshot()
	if len(rows) != 1 || rows[0].Confidence != 1 || !containsLocatorSource(rows[0].Sources, objectLocatorSourceOperator) {
		t.Fatalf("operator include not reflected: %#v", rows)
	}
	if _, err := s.deleteOverride(include.ID); err != nil {
		t.Fatal(err)
	}
	if len(s.snapshot()) != 0 {
		t.Fatal("operator-only locator survived INCLUDE override deletion")
	}

	// Suppression applies to an independently discovered locator and never
	// turns the operator control into ownership or authorization authority.
	s.noteObservation(apiOperation{ID: opID}, []schemaFieldSample{{Path: "order_id", Location: "query", Type: "string", objectValue: "123"}})
	suppress, err := s.upsertOverride(ObjectLocatorOverride{OperationID: opID, Location: "query", Field: "order_id", Action: objectLocatorOverrideSuppress})
	if err != nil {
		t.Fatal(err)
	}
	if got := s.snapshot()[0].Status; got != objectLocatorStatusSuppressed {
		t.Fatalf("status=%s want SUPPRESSED", got)
	}

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	if err := s.save(configPath); err != nil {
		t.Fatal(err)
	}
	keyInfo, err := os.Stat(filepath.Join(dir, "api-object-locator.key"))
	if err != nil || keyInfo.Mode().Perm() != 0o600 {
		t.Fatalf("fingerprint key mode: info=%v err=%v", keyInfo, err)
	}
	restored := newObjectLocatorStoreWithKey([]byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	restored.now = s.now
	if err := restored.load(configPath); err != nil {
		t.Fatal(err)
	}
	if len(restored.snapshot()) != 1 || restored.snapshot()[0].Status != objectLocatorStatusSuppressed {
		t.Fatalf("restart state mismatch: %#v", restored.snapshot())
	}
	if _, err := restored.deleteOverride(suppress.ID); err != nil {
		t.Fatal(err)
	}
	if got := restored.snapshot()[0].Status; got != objectLocatorStatusActive {
		t.Fatalf("suppression deletion did not reactivate learned locator: %s", got)
	}
}

func TestAPI71CardinalityAndTTLRemainBounded(t *testing.T) {
	s := api71Store()
	at := time.Date(2026, 9, 24, 13, 50, 0, 0, time.UTC)
	s.now = func() time.Time { return at }
	op := apiOperation{ID: apiOperationID("shop", http.MethodPost, "/bulk")}
	var samples []schemaFieldSample
	for i := 0; i < objectLocatorMaxPerOperation+20; i++ {
		samples = append(samples, schemaFieldSample{Path: fmt.Sprintf("field_%d_id", i), Location: "body", Type: "string", objectValue: fmt.Sprintf("value-%d", i)})
	}
	s.noteObservation(op, samples)
	if got := len(s.snapshot()); got != objectLocatorMaxPerOperation {
		t.Fatalf("per-operation locator cap=%d want %d", got, objectLocatorMaxPerOperation)
	}
	s.now = func() time.Time { return at.Add(objectLocatorDefaultTTL + time.Second) }
	if got := len(s.snapshot()); got != 0 {
		t.Fatalf("expired learned locators retained=%d", got)
	}
}

func TestAPI71KeyedFingerprintStableAcrossRestartWithoutRawValue(t *testing.T) {
	s := api71Store()
	at := time.Date(2026, 9, 24, 14, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return at }
	op := apiOperation{ID: apiOperationID("shop", http.MethodGet, "/orders")}
	s.noteObservation(op, []schemaFieldSample{{Path: "order_id", Location: "query", Type: "string", objectValue: "secret-object-991"}})
	before := s.snapshot()[0].ValueFingerprints[0].Digest
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	if err := s.save(configPath); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "api-object-locators.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "secret-object-991") {
		t.Fatalf("raw object value persisted: %s", b)
	}
	restored := newObjectLocatorStoreWithKey([]byte("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"))
	restored.now = s.now
	if err := restored.load(configPath); err != nil {
		t.Fatal(err)
	}
	if got := restored.snapshot()[0].ValueFingerprints[0].Digest; got != before {
		t.Fatalf("fingerprint changed across restart: %s != %s", got, before)
	}
}

func TestAPI71ConcurrentObservationAndSnapshotStayBounded(t *testing.T) {
	s := api71Store()
	s.now = func() time.Time { return time.Date(2026, 9, 24, 14, 10, 0, 0, time.UTC) }
	op := apiOperation{ID: apiOperationID("shop", http.MethodPost, "/orders")}
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				s.noteObservation(op, []schemaFieldSample{{Path: fmt.Sprintf("object_%d_id", i%40), Location: "body", Type: "string", objectValue: fmt.Sprintf("%d-%d", g, i)}})
				_ = s.snapshot()
			}
		}(g)
	}
	wg.Wait()
	if got := len(s.snapshot()); got > objectLocatorMaxPerOperation {
		t.Fatalf("concurrent locator cardinality=%d exceeds cap", got)
	}
}

func TestAPI71NoOwnershipBOLAVerdictOrEnforcementPrimitive(t *testing.T) {
	s := api71Store()
	opID := apiOperationID("shop", http.MethodGet, "/orders")
	if _, err := s.upsertOverride(ObjectLocatorOverride{OperationID: opID, Location: "query", Field: "tenant_id", SemanticName: "tenant_id", Action: objectLocatorOverrideInclude}); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(s.snapshot())
	text := strings.ToUpper(string(b))
	for _, forbidden := range []string{"BOLA_CANDIDATE", "OWNER_MATCH", "OWNER_MISMATCH", "ENFORCE", "BLOCK", "DENY"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("API-7.1 locator model acquired forbidden verdict/authority %s: %s", forbidden, b)
		}
	}
}
