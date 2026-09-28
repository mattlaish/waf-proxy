package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func api74Candidate(t *testing.T, locators *objectLocatorStore, relationships *objectRelationshipStore, detector *bolaDetectionStore, operationID, rawObject, identity string, at time.Time) BOLACandidate {
	t.Helper()
	locator := api72IncludeLocator(t, locators, operationID, "path", "id")
	objectFP := locators.digestValue(operationID, "path", "id", rawObject)
	identityFP := relationships.digestIdentity("subject", "issuer", identity)
	id := bolaCandidateID(bolaCandidateIdentityDivergence, identityFP, locator.ID, objectFP, "")
	row := BOLACandidate{
		ID: id, Type: bolaCandidateIdentityDivergence, OperationID: operationID, LocatorID: locator.ID, Location: "path", Field: "id",
		IdentityFingerprint: identityFP, ObjectFingerprint: objectFP, Confidence: bolaConfidenceHigh,
		BaselineIdentities: 1, BaselineObservations: 3, EvidenceCount: 1,
		Sources:   []string{objectRelationshipSourceAPI5, objectRelationshipSourceAPI71, "API72_RELATIONSHIP_BASELINE"},
		FirstSeen: at, LastSeen: at, ExpiresAt: at.Add(bolaCandidateDefaultTTL),
	}
	detector.mu.Lock()
	detector.candidates[id] = &row
	detector.mu.Unlock()
	return row
}

func newAPI74Stores() (*objectLocatorStore, *objectRelationshipStore, *bolaDetectionStore, *bolaPolicyStore) {
	locators, relationships, detector := newAPI73Stores()
	policy := newBOLAPolicyStore()
	return locators, relationships, detector, policy
}

func TestAPI74PolicyScopePrecedenceAndSuppressionNeverDeletesCandidate(t *testing.T) {
	locators, relationships, detector, policy := newAPI74Stores()
	now := time.Unix(1_700_100_000, 0).UTC()
	policy.now = func() time.Time { return now }
	detector.now = func() time.Time { return now }
	opID := apiOperationID("shop", "GET", "/orders/{id}")
	candidate := api74Candidate(t, locators, relationships, detector, opID, "42", "bob", now)

	broad, err := policy.upsertPolicy(BOLAPolicy{OperationID: opID, MinConfidence: bolaConfidenceMedium, Action: bolaPolicyActionSuppress, Enabled: true}, time.Hour, locators)
	if err != nil {
		t.Fatal(err)
	}
	rows := policy.evidenceSnapshot(detector)
	if len(rows) != 1 || !rows[0].Suppressed || rows[0].Policy == nil || rows[0].Policy.ID != broad.ID {
		t.Fatalf("broad suppression mismatch: %#v", rows)
	}
	if got := len(detector.snapshot()); got != 1 {
		t.Fatalf("evidence policy deleted detector candidate: %d", got)
	}

	specific, err := policy.upsertPolicy(BOLAPolicy{OperationID: opID, LocatorID: candidate.LocatorID, CandidateType: candidate.Type, MinConfidence: bolaConfidenceMedium, Action: bolaPolicyActionReview, Enabled: true}, time.Hour, locators)
	if err != nil {
		t.Fatal(err)
	}
	rows = policy.evidenceSnapshot(detector)
	if len(rows) != 1 || rows[0].Suppressed || rows[0].Policy == nil || rows[0].Policy.ID != specific.ID || rows[0].EffectiveAction != bolaPolicyActionReview {
		t.Fatalf("specific REVIEW did not override broad SUPPRESS: %#v", rows)
	}
}

func TestAPI74PolicyRejectsUnnormalizedOrUnknownLocatorAndInvalidTTL(t *testing.T) {
	locators, _, _, policy := newAPI74Stores()
	policy.now = func() time.Time { return time.Unix(1_700_100_100, 0).UTC() }
	if _, err := policy.upsertPolicy(BOLAPolicy{OperationID: "/orders/42", Action: bolaPolicyActionSuppress, Enabled: true}, time.Hour, locators); err == nil {
		t.Fatal("raw path accepted as normalized operation selector")
	}
	opID := apiOperationID("shop", "GET", "/orders/{id}")
	if _, err := policy.upsertPolicy(BOLAPolicy{OperationID: opID, LocatorID: strings.Repeat("a", 32), Action: bolaPolicyActionSuppress, Enabled: true}, time.Hour, locators); err == nil {
		t.Fatal("unknown locator accepted")
	}
	api72IncludeLocator(t, locators, opID, "path", "id")
	if _, err := policy.upsertPolicy(BOLAPolicy{OperationID: opID, Action: "ENFORCE", Enabled: true}, time.Hour, locators); err == nil {
		t.Fatal("enforcement action accepted")
	}
	if _, err := policy.upsertPolicy(BOLAPolicy{OperationID: opID, Action: bolaPolicyActionReview, Enabled: true}, bolaPolicyMaxTTL+time.Second, locators); err == nil {
		t.Fatal("unbounded TTL accepted")
	}
}

func TestAPI74EvidenceWorkflowUsesEnumeratedStateAndReasonOnly(t *testing.T) {
	locators, relationships, detector, policy := newAPI74Stores()
	now := time.Unix(1_700_100_200, 0).UTC()
	policy.now = func() time.Time { return now }
	detector.now = func() time.Time { return now }
	candidate := api74Candidate(t, locators, relationships, detector, apiOperationID("shop", "GET", "/orders/{id}"), "42", "bob", now)
	if _, err := policy.reviewCandidate(candidate, bolaWorkflowDismissed, ""); err == nil {
		t.Fatal("dismiss without structured reason accepted")
	}
	if _, err := policy.reviewCandidate(candidate, bolaWorkflowDismissed, "raw object 42 belongs to Alice"); err == nil {
		t.Fatal("free-text reason accepted")
	}
	r, err := policy.reviewCandidate(candidate, bolaWorkflowDismissed, "FALSE_POSITIVE")
	if err != nil {
		t.Fatal(err)
	}
	if r.State != bolaWorkflowDismissed || r.ReasonCode != "FALSE_POSITIVE" || r.EvidenceCountAtReview != 1 {
		t.Fatalf("unexpected review: %#v", r)
	}
	rows := policy.evidenceSnapshot(detector)
	if len(rows) != 1 || rows[0].WorkflowState != bolaWorkflowDismissed || rows[0].ReasonCode != "FALSE_POSITIVE" {
		t.Fatalf("workflow not joined into evidence: %#v", rows)
	}
}

func TestAPI74DismissedOrResolvedEvidenceReopensOnNewDetection(t *testing.T) {
	locators, relationships, detector, policy := newAPI74Stores()
	now := time.Unix(1_700_100_300, 0).UTC()
	policy.now = func() time.Time { return now }
	detector.now = func() time.Time { return now }
	candidate := api74Candidate(t, locators, relationships, detector, apiOperationID("shop", "GET", "/orders/{id}"), "42", "bob", now)
	if _, err := policy.reviewCandidate(candidate, bolaWorkflowResolved, "FIX_DEPLOYED"); err != nil {
		t.Fatal(err)
	}
	detector.mu.Lock()
	detector.candidates[candidate.ID].EvidenceCount++
	detector.candidates[candidate.ID].LastSeen = now.Add(time.Minute)
	detector.mu.Unlock()
	rows := policy.evidenceSnapshot(detector)
	if len(rows) != 1 || !rows[0].Reopened || rows[0].WorkflowState != bolaWorkflowOpen {
		t.Fatalf("new evidence did not reopen resolved review: %#v", rows)
	}
}

func TestAPI74PersistenceIsBoundedRestartSafeAndPrivacyPreserving(t *testing.T) {
	locators, relationships, detector, policy := newAPI74Stores()
	base := time.Unix(1_700_100_400, 0).UTC()
	policy.now = func() time.Time { return base }
	detector.now = func() time.Time { return base }
	opID := apiOperationID("shop", "GET", "/orders/{id}")
	candidate := api74Candidate(t, locators, relationships, detector, opID, "raw-object-secret", "raw-user-secret", base)
	if _, err := policy.upsertPolicy(BOLAPolicy{OperationID: opID, LocatorID: candidate.LocatorID, CandidateType: candidate.Type, MinConfidence: bolaConfidenceMedium, Action: bolaPolicyActionSuppress, Enabled: true}, time.Hour, locators); err != nil {
		t.Fatal(err)
	}
	if _, err := policy.reviewCandidate(candidate, bolaWorkflowAcknowledged, "INVESTIGATING"); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.json")
	if err := policy.save(cfg); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "api-bola-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"raw-object-secret", "raw-user-secret", "Authorization", "Cookie", "X-Owner", "X-Tenant", "Bearer "} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("raw sensitive material persisted: %q", forbidden)
		}
	}
	restored := newBOLAPolicyStore()
	restored.now = func() time.Time { return base }
	if err := restored.load(cfg); err != nil {
		t.Fatal(err)
	}
	if len(restored.policiesSnapshot()) != 1 || len(restored.evidenceSnapshot(detector)) != 1 {
		t.Fatal("policy/review state failed to restore")
	}
	restored.now = func() time.Time { return base.Add(bolaPolicyMaxTTL + bolaReviewDefaultTTL + time.Hour) }
	if len(restored.policiesSnapshot()) != 0 {
		t.Fatal("expired policy survived TTL pruning")
	}
}

func TestAPI74PolicyAndReviewCardinalityAreBounded(t *testing.T) {
	locators, relationships, detector, policy := newAPI74Stores()
	now := time.Unix(1_700_100_500, 0).UTC()
	policy.now = func() time.Time { return now }
	detector.now = func() time.Time { return now }
	opID := apiOperationID("shop", "GET", "/orders/{id}")
	candidate := api74Candidate(t, locators, relationships, detector, opID, "42", "bob", now)
	policy.mu.Lock()
	for i := 0; i < bolaPolicyMaxRules; i++ {
		id := fmt.Sprintf("%040x", i+1)
		policy.policies[id] = &BOLAPolicy{ID: id, OperationID: opID, MinConfidence: bolaConfidenceMedium, Action: bolaPolicyActionReview, Enabled: true, CreatedAt: now, UpdatedAt: now, ExpiresAt: now.Add(time.Hour)}
	}
	policy.mu.Unlock()
	if _, err := policy.upsertPolicy(BOLAPolicy{OperationID: opID, LocatorID: candidate.LocatorID, CandidateType: candidate.Type, MinConfidence: bolaConfidenceHigh, Action: bolaPolicyActionReview, Enabled: true}, time.Hour, locators); err == nil {
		t.Fatal("policy cap not enforced")
	}
	policy.mu.Lock()
	policy.policies = map[string]*BOLAPolicy{}
	for i := 0; i < bolaReviewMaxRecords; i++ {
		id := fmt.Sprintf("%040x", i+1)
		policy.reviews[id] = &BOLAEvidenceReview{CandidateID: id, State: bolaWorkflowAcknowledged, ReasonCode: "INVESTIGATING", EvidenceCountAtReview: 1, CandidateLastSeen: now, UpdatedAt: now, ExpiresAt: now.Add(time.Hour)}
	}
	policy.mu.Unlock()
	if _, err := policy.reviewCandidate(candidate, bolaWorkflowAcknowledged, "INVESTIGATING"); err == nil {
		t.Fatal("review cap not enforced")
	}
}

func TestAPI74StrictMutationJSONRejectsUnknownRawSelectors(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/security/bola/policies", bytes.NewBufferString(`{"operation_id":"abcd","raw_object_id":"42"}`))
	rr := httptest.NewRecorder()
	var dst struct {
		OperationID string `json:"operation_id"`
	}
	if err := decodeAPI74JSON(rr, req, &dst); err == nil {
		t.Fatal("unknown raw selector field accepted")
	}
}

func TestAPI74StatusDeclaresInferredBOLANonBlocking(t *testing.T) {
	locators, relationships, detector, policy := newAPI74Stores()
	now := time.Unix(1_700_100_600, 0).UTC()
	policy.now = func() time.Time { return now }
	detector.now = func() time.Time { return now }
	api74Candidate(t, locators, relationships, detector, apiOperationID("shop", "GET", "/orders/{id}"), "42", "bob", now)
	status := policy.status(detector)
	if status.InferenceBlocking {
		t.Fatal("inferred BOLA unexpectedly reported request-blocking authority")
	}
	if status.EvidenceCount != 1 || status.MaxPolicies != bolaPolicyMaxRules || status.MaxReviews != bolaReviewMaxRecords {
		t.Fatalf("unexpected status: %#v", status)
	}
}

func TestAPI74ConcurrentPolicyEvidenceSnapshotsRemainBounded(t *testing.T) {
	locators, relationships, detector, policy := newAPI74Stores()
	now := time.Unix(1_700_100_700, 0).UTC()
	policy.now = func() time.Time { return now }
	detector.now = func() time.Time { return now }
	opID := apiOperationID("shop", "GET", "/orders/{id}")
	candidate := api74Candidate(t, locators, relationships, detector, opID, "42", "bob", now)
	var wg sync.WaitGroup
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%3 == 0 {
				_, _ = policy.upsertPolicy(BOLAPolicy{OperationID: opID, LocatorID: candidate.LocatorID, CandidateType: candidate.Type, MinConfidence: bolaConfidenceMedium, Action: bolaPolicyActionReview, Enabled: true}, time.Hour, locators)
			}
			if i%3 == 1 {
				_, _ = policy.reviewCandidate(candidate, bolaWorkflowAcknowledged, "INVESTIGATING")
			}
			_ = policy.evidenceSnapshot(detector)
		}(i)
	}
	wg.Wait()
	if len(policy.policiesSnapshot()) > bolaPolicyMaxRules || len(policy.evidenceSnapshot(detector)) > bolaCandidateMaxCandidates {
		t.Fatal("concurrent API-7.4 state exceeded bounds")
	}
}

func TestAPI74StateFileRejectsUnknownVersion(t *testing.T) {
	policy := newBOLAPolicyStore()
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.json")
	data, _ := json.Marshal(bolaPolicyStateFile{Version: bolaPolicyStateVersion + 1, Saved: time.Now().UTC()})
	if err := os.WriteFile(filepath.Join(dir, "api-bola-policy.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := policy.load(cfg); err == nil {
		t.Fatal("unknown state version accepted")
	}
}
