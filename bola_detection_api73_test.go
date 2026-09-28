package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func newAPI73Stores() (*objectLocatorStore, *objectRelationshipStore, *bolaDetectionStore) {
	locators, relationships := newAPI72Stores()
	detector := newBOLADetectionStore()
	relationships.detector = detector
	return locators, relationships, detector
}

func api73Baseline(t *testing.T, relationships *objectRelationshipStore, locator ObjectLocator, objectFingerprint, identity string, tenant string, at time.Time, observations int) {
	t.Helper()
	for i := 0; i < observations; i++ {
		relationships.merge(objectRelationshipObservation{
			OperationID: locator.OperationID, IdentityKind: "subject", IdentityFingerprint: identity,
			TenantFingerprint: tenant, At: at.Add(time.Duration(i) * time.Second),
		}, locator, objectFingerprint)
	}
}

func TestAPI73IdentityObjectDivergenceRequiresRepeatedBaseline(t *testing.T) {
	locators, relationships, detector := newAPI73Stores()
	now := time.Unix(1_700_000_000, 0).UTC()
	detector.now = func() time.Time { return now }
	relationships.now = func() time.Time { return now }
	opID := apiOperationID("shop", "GET", "/orders/{id}")
	locator := api72IncludeLocator(t, locators, opID, "path", "id")
	objectFP := locators.digestValue(opID, "path", "id", "42")
	alice := relationships.digestIdentity("subject", "issuer", "alice")
	bob := relationships.digestIdentity("subject", "issuer", "bob")

	api73Baseline(t, relationships, locator, objectFP, alice, "", now.Add(-time.Minute), 1)
	detector.observe(objectRelationshipObservation{OperationID: opID, IdentityKind: "subject", IdentityFingerprint: bob, At: now}, locator, objectFP, relationships)
	if got := len(detector.snapshot()); got != 0 {
		t.Fatalf("single baseline observation created BOLA candidate: %d", got)
	}
	api73Baseline(t, relationships, locator, objectFP, alice, "", now.Add(-30*time.Second), 1)
	detector.observe(objectRelationshipObservation{OperationID: opID, IdentityKind: "subject", IdentityFingerprint: bob, At: now}, locator, objectFP, relationships)
	rows := detector.snapshot()
	if len(rows) != 1 || rows[0].Type != bolaCandidateIdentityDivergence || rows[0].Confidence == "" {
		t.Fatalf("unexpected BOLA candidate evidence: %#v", rows)
	}
}

func TestAPI73NovelObjectAloneIsNotBOLA(t *testing.T) {
	locators, relationships, detector := newAPI73Stores()
	now := time.Now().UTC()
	opID := apiOperationID("shop", "GET", "/orders/{id}")
	locator := api72IncludeLocator(t, locators, opID, "path", "id")
	objectFP := locators.digestValue(opID, "path", "id", "new-object")
	detector.observe(objectRelationshipObservation{OperationID: opID, IdentityKind: "subject", IdentityFingerprint: relationships.digestIdentity("subject", "issuer", "alice"), At: now}, locator, objectFP, relationships)
	if got := len(detector.snapshot()); got != 0 {
		t.Fatalf("novel object without baseline created candidate: %d", got)
	}
}

func TestAPI73TenantDivergenceUsesVerifiedPseudonymousTenantEvidence(t *testing.T) {
	locators, relationships, detector := newAPI73Stores()
	now := time.Unix(1_700_000_100, 0).UTC()
	detector.now = func() time.Time { return now }
	relationships.now = func() time.Time { return now }
	opID := apiOperationID("shop", "GET", "/accounts/{id}")
	locator := api72IncludeLocator(t, locators, opID, "path", "id")
	objectFP := locators.digestValue(opID, "path", "id", "acct-42")
	alice := relationships.digestIdentity("subject", "issuer", "alice")
	bob := relationships.digestIdentity("subject", "issuer", "bob")
	tenantA := relationships.digestIdentity("tenant", "issuer", "tenant-a")
	tenantB := relationships.digestIdentity("tenant", "issuer", "tenant-b")
	api73Baseline(t, relationships, locator, objectFP, alice, tenantA, now.Add(-time.Minute), 2)
	detector.observe(objectRelationshipObservation{OperationID: opID, IdentityKind: "subject", IdentityFingerprint: bob, TenantFingerprint: tenantB, At: now}, locator, objectFP, relationships)
	rows := detector.snapshot()
	types := map[string]bool{}
	for _, row := range rows {
		types[row.Type] = true
	}
	if !types[bolaCandidateIdentityDivergence] || !types[bolaCandidateTenantDivergence] {
		t.Fatalf("expected identity + tenant candidate evidence, got %#v", rows)
	}
	for _, row := range rows {
		if strings.Contains(row.TenantFingerprint, "tenant-") || strings.Contains(row.IdentityFingerprint, "alice") || strings.Contains(row.ObjectFingerprint, "acct") {
			t.Fatalf("raw evidence leaked into candidate: %#v", row)
		}
	}
}

func TestAPI73SameIdentityRelationshipDoesNotCreateDivergence(t *testing.T) {
	locators, relationships, detector := newAPI73Stores()
	now := time.Now().UTC()
	opID := apiOperationID("shop", "GET", "/orders/{id}")
	locator := api72IncludeLocator(t, locators, opID, "path", "id")
	objectFP := locators.digestValue(opID, "path", "id", "42")
	alice := relationships.digestIdentity("subject", "issuer", "alice")
	api73Baseline(t, relationships, locator, objectFP, alice, "", now.Add(-time.Minute), 3)
	detector.observe(objectRelationshipObservation{OperationID: opID, IdentityKind: "subject", IdentityFingerprint: alice, At: now}, locator, objectFP, relationships)
	if got := len(detector.snapshot()); got != 0 {
		t.Fatalf("same identity created divergence candidate: %d", got)
	}
}

func TestAPI73EnumerationRequiresBoundedRecentFanout(t *testing.T) {
	locators, relationships, detector := newAPI73Stores()
	now := time.Unix(1_700_000_200, 0).UTC()
	detector.now = func() time.Time { return now }
	relationships.now = func() time.Time { return now }
	opID := apiOperationID("shop", "GET", "/orders/{id}")
	locator := api72IncludeLocator(t, locators, opID, "path", "id")
	alice := relationships.digestIdentity("subject", "issuer", "alice")
	for i := 0; i < bolaCandidateEnumerationMinObjs-1; i++ {
		fp := locators.digestValue(opID, "path", "id", fmt.Sprintf("obj-%d", i))
		relationships.merge(objectRelationshipObservation{OperationID: opID, IdentityKind: "subject", IdentityFingerprint: alice, At: now.Add(-time.Minute)}, locator, fp)
	}
	nextFP := locators.digestValue(opID, "path", "id", "threshold-object")
	detector.observe(objectRelationshipObservation{OperationID: opID, IdentityKind: "subject", IdentityFingerprint: alice, At: now}, locator, nextFP, relationships)
	rows := detector.snapshot()
	if len(rows) != 1 || rows[0].Type != bolaCandidateEnumeration || rows[0].RecentIdentityObjects != bolaCandidateEnumerationMinObjs {
		t.Fatalf("enumeration candidate threshold mismatch: %#v", rows)
	}

	oldRelationships := newObjectRelationshipStoreWithKey([]byte("abcdef0123456789abcdef0123456789"))
	oldRelationships.locators = locators
	for i := 0; i < bolaCandidateEnumerationMinObjs+5; i++ {
		fp := locators.digestValue(opID, "path", "id", fmt.Sprintf("old-%d", i))
		oldRelationships.merge(objectRelationshipObservation{OperationID: opID, IdentityKind: "subject", IdentityFingerprint: alice, At: now.Add(-bolaCandidateEnumerationWindow - time.Minute)}, locator, fp)
	}
	other := newBOLADetectionStore()
	other.now = func() time.Time { return now }
	other.observe(objectRelationshipObservation{OperationID: opID, IdentityKind: "subject", IdentityFingerprint: alice, At: now}, locator, nextFP, oldRelationships)
	if got := len(other.snapshot()); got != 0 {
		t.Fatalf("stale fanout created enumeration candidate: %d", got)
	}
}

func TestAPI73SuppressedLocatorCannotFeedDetector(t *testing.T) {
	locators, relationships, detector := newAPI73Stores()
	now := time.Now().UTC()
	opID := apiOperationID("shop", "GET", "/orders/{id}")
	locator := api72IncludeLocator(t, locators, opID, "path", "id")
	objectFP := locators.digestValue(opID, "path", "id", "42")
	api73Baseline(t, relationships, locator, objectFP, relationships.digestIdentity("subject", "issuer", "alice"), "", now.Add(-time.Minute), 2)
	if _, err := locators.upsertOverride(ObjectLocatorOverride{OperationID: opID, Location: "path", Field: "id", Action: objectLocatorOverrideSuppress}); err != nil {
		t.Fatal(err)
	}
	relationships.process(objectRelationshipObservation{OperationID: opID, Path: "/orders/42", IdentityKind: "subject", IdentityFingerprint: relationships.digestIdentity("subject", "issuer", "bob"), At: now})
	if got := len(detector.snapshot()); got != 0 {
		t.Fatalf("suppressed locator fed BOLA detector: %d", got)
	}
}

func TestAPI73PersistenceTTLAndPrivacy(t *testing.T) {
	locators, relationships, detector := newAPI73Stores()
	base := time.Unix(1_700_000_300, 0).UTC()
	detector.now = func() time.Time { return base }
	relationships.now = func() time.Time { return base }
	opID := apiOperationID("shop", "GET", "/orders/{id}")
	locator := api72IncludeLocator(t, locators, opID, "path", "id")
	objectFP := locators.digestValue(opID, "path", "id", "raw-object-secret")
	alice := relationships.digestIdentity("subject", "issuer", "raw-alice-secret")
	bob := relationships.digestIdentity("subject", "issuer", "raw-bob-secret")
	api73Baseline(t, relationships, locator, objectFP, alice, "", base.Add(-time.Minute), 2)
	detector.observe(objectRelationshipObservation{OperationID: opID, IdentityKind: "subject", IdentityFingerprint: bob, At: base}, locator, objectFP, relationships)
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.json")
	if err := detector.save(cfg); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "api-bola-candidates.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"raw-object-secret", "raw-alice-secret", "raw-bob-secret", "Authorization", "Bearer ", "X-Owner", "X-Tenant"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("raw sensitive material persisted: %q", forbidden)
		}
	}
	restored := newBOLADetectionStore()
	restored.now = func() time.Time { return base }
	if err := restored.load(cfg); err != nil {
		t.Fatal(err)
	}
	if got := len(restored.snapshot()); got != 1 {
		t.Fatalf("candidate did not restore: %d", got)
	}
	restored.now = func() time.Time { return base.Add(bolaCandidateDefaultTTL + time.Second) }
	if got := len(restored.snapshot()); got != 0 {
		t.Fatalf("expired candidate retained: %d", got)
	}
}

func TestAPI73CandidateCardinalityIsBounded(t *testing.T) {
	detector := newBOLADetectionStore()
	now := time.Now().UTC()
	detector.now = func() time.Time { return now }
	opID := apiOperationID("shop", "GET", "/orders/{id}")
	locator := ObjectLocator{ID: strings.Repeat("a", 32), OperationID: opID, Location: "path", Field: "id"}
	identity := strings.Repeat("b", 40)
	for i := 0; i < bolaCandidateMaxPerIdentity+32; i++ {
		fp := fmt.Sprintf("%040x", i+1)
		detector.record(bolaCandidateEnumeration, bolaConfidenceMedium, objectRelationshipObservation{OperationID: opID, IdentityKind: "subject", IdentityFingerprint: identity, At: now.Add(time.Duration(i) * time.Millisecond)}, locator, fp, bolaRelationshipContext{RecentIdentityObjects: bolaCandidateEnumerationMinObjs})
	}
	if got := len(detector.snapshot()); got != bolaCandidateMaxPerIdentity {
		t.Fatalf("per-identity candidate cap not enforced: got=%d want=%d", got, bolaCandidateMaxPerIdentity)
	}
}

func TestAPI73ConcurrentDetectionRemainsBounded(t *testing.T) {
	locators, relationships, detector := newAPI73Stores()
	now := time.Now().UTC()
	opID := apiOperationID("shop", "GET", "/orders/{id}")
	locator := api72IncludeLocator(t, locators, opID, "path", "id")
	objectFP := locators.digestValue(opID, "path", "id", "42")
	api73Baseline(t, relationships, locator, objectFP, relationships.digestIdentity("subject", "issuer", "owner"), "", now.Add(-time.Minute), 3)
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := relationships.digestIdentity("subject", "issuer", fmt.Sprintf("user-%d", i))
			for j := 0; j < 20; j++ {
				detector.observe(objectRelationshipObservation{OperationID: opID, IdentityKind: "subject", IdentityFingerprint: id, At: now.Add(time.Duration(j) * time.Millisecond)}, locator, objectFP, relationships)
			}
		}(i)
	}
	wg.Wait()
	rows := detector.snapshot()
	if len(rows) > bolaCandidateMaxPerObject || len(rows) > bolaCandidateMaxCandidates {
		t.Fatalf("BOLA candidate bounds exceeded: %d", len(rows))
	}
}
