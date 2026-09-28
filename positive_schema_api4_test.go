package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func api4ReviewedCandidate(site, method, path string) (*SchemaCandidate, apiOperation) {
	now := time.Now().UTC()
	normalized := normalizeAPIOperationPath(path)
	opID := apiOperationID(site, method, normalized)
	candidate := &SchemaCandidate{
		ID: schemaCandidateID(opID), OperationID: opID, Status: "REVIEWED", Confidence: 1, SampleCount: 100, Version: 3,
		Fields: []SchemaFieldObservation{
			{Path: "amount", Location: "body", TypeCounts: map[string]int64{"integer": 100}, Types: []SchemaTypeEvidence{{Type: "integer", Count: 100, Confidence: 1}}, Samples: 100, PresenceRate: 1, RequiredCandidate: true},
			{Path: "currency", Location: "body", TypeCounts: map[string]int64{"string": 100}, Types: []SchemaTypeEvidence{{Type: "string", Count: 100, Confidence: 1}}, Samples: 100, PresenceRate: 1, RequiredCandidate: true, EnumCandidate: []string{"TWD", "USD"}},
		},
		Review:    &SchemaReviewRecommendation{Recommendation: "approved", Confidence: 100, ReviewedAt: now, Source: "operator", CandidateVersion: 3},
		CreatedAt: now, UpdatedAt: now,
	}
	op := apiOperation{
		ID: opID, Site: site, Host: "api.example.test", Method: method, Path: normalized,
		ContentTypes: map[string]int64{"application/json": 100}, AuthObserved: map[string]int64{"bearer": 100},
	}
	return candidate, op
}

func api4PromoteForTest(t *testing.T, store *positiveSchemaStore, candidate *SchemaCandidate, op apiOperation, unknown string) (PositiveSchemaProfileVersion, PositiveSchemaDeployment) {
	t.Helper()
	profile, deployment, err := store.promote(candidate, op, positiveSchemaPromoteOptions{
		UnknownFields: unknown, EnforceEnums: true, EnforceContentType: true, CreatedBy: "tester", BlockStatus: http.StatusUnprocessableEntity,
	})
	if err != nil {
		t.Fatalf("promote: %v", err)
	}
	return profile, deployment
}

func api4Serve(store *positiveSchemaStore, site, body string, headers map[string]string) *httptest.ResponseRecorder {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	handler := requestBodyPrefixWrap(false, false, true, store.wrap(site, next))
	req := httptest.NewRequest(http.MethodPost, "http://api.example.test/api/payment/123", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestAPI4PromotionRequiresCurrentOperatorReview(t *testing.T) {
	store := newPositiveSchemaStore()
	candidate, op := api4ReviewedCandidate("checkout", http.MethodPost, "/api/payment/123")
	candidate.Review.Source = "openai"
	if _, _, err := store.promote(candidate, op, positiveSchemaPromoteOptions{}); err == nil {
		t.Fatal("AI-only review must not activate positive schema enforcement")
	}
	candidate.Review.Source = "operator"
	candidate.Review.CandidateVersion = candidate.Version - 1
	if _, _, err := store.promote(candidate, op, positiveSchemaPromoteOptions{}); err == nil {
		t.Fatal("stale operator review must not activate positive schema enforcement")
	}
}

func TestAPI4ModeLifecycleRequiresDetectBeforeEnforce(t *testing.T) {
	store := newPositiveSchemaStore()
	candidate, op := api4ReviewedCandidate("checkout", http.MethodPost, "/api/payment/123")
	_, deployment := api4PromoteForTest(t, store, candidate, op, positiveSchemaUnknownAllow)
	if deployment.Mode != positiveSchemaModeLearn {
		t.Fatalf("initial mode = %s, want LEARN", deployment.Mode)
	}
	if _, err := store.setMode(op.ID, positiveSchemaModeEnforce, "tester", "skip", 422); err == nil {
		t.Fatal("LEARN -> ENFORCE should be rejected")
	}
	deployment, err := store.setMode(op.ID, positiveSchemaModeDetect, "tester", "shadow", 422)
	if err != nil || deployment.Mode != positiveSchemaModeDetect {
		t.Fatalf("DETECT transition: deployment=%+v err=%v", deployment, err)
	}
	deployment, err = store.setMode(op.ID, positiveSchemaModeEnforce, "tester", "approved shadow evidence", 422)
	if err != nil || deployment.Mode != positiveSchemaModeEnforce {
		t.Fatalf("ENFORCE transition: deployment=%+v err=%v", deployment, err)
	}
}

func TestAPI4DetectShadowsAndEnforceBlocks(t *testing.T) {
	store := newPositiveSchemaStore()
	candidate, op := api4ReviewedCandidate("checkout", http.MethodPost, "/api/payment/123")
	_, _ = api4PromoteForTest(t, store, candidate, op, positiveSchemaUnknownDeny)
	if _, err := store.setMode(op.ID, positiveSchemaModeDetect, "tester", "shadow", 422); err != nil {
		t.Fatal(err)
	}
	bad := `{"amount":"not-a-number","currency":"EUR","unexpected":true}`
	rec := api4Serve(store, "checkout", bad, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DETECT blocked request: status=%d body=%s", rec.Code, rec.Body.String())
	}
	violations := store.listViolations(op.ID, 20)
	if len(violations) == 0 || violations[0].Action != "detect" {
		t.Fatalf("expected shadow violation evidence, got %+v", violations)
	}
	if _, err := store.setMode(op.ID, positiveSchemaModeEnforce, "tester", "promote", 422); err != nil {
		t.Fatal(err)
	}
	rec = api4Serve(store, "checkout", bad, nil)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ENFORCE status=%d want 422; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "API schema policy") {
		t.Fatalf("unexpected block body: %s", rec.Body.String())
	}
}

func TestAPI4ExceptionBypassesSpecificViolation(t *testing.T) {
	store := newPositiveSchemaStore()
	candidate, op := api4ReviewedCandidate("checkout", http.MethodPost, "/api/payment/123")
	profile, _ := api4PromoteForTest(t, store, candidate, op, positiveSchemaUnknownAllow)
	if _, err := store.setMode(op.ID, positiveSchemaModeDetect, "tester", "shadow", 422); err != nil {
		t.Fatal(err)
	}
	if _, err := store.setMode(op.ID, positiveSchemaModeEnforce, "tester", "promote", 422); err != nil {
		t.Fatal(err)
	}
	ex, err := store.addException(PositiveSchemaException{
		OperationID: op.ID, ProfileVersionID: profile.ID, ViolationType: "ENUM_MISMATCH", Field: "body|currency",
		Reason: "legacy partner currency during migration", CreatedBy: "tester",
	})
	if err != nil {
		t.Fatal(err)
	}
	rec := api4Serve(store, "checkout", `{"amount":12,"currency":"EUR"}`, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("exception did not bypass only violation: status=%d body=%s", rec.Code, rec.Body.String())
	}
	violations := store.listViolations(op.ID, 5)
	if len(violations) == 0 || !violations[0].Exempted || violations[0].ExceptionID != ex.ID {
		t.Fatalf("exception evidence missing: %+v", violations)
	}
}

func TestAPI4RollbackRestoresPreviousProfileAndMode(t *testing.T) {
	store := newPositiveSchemaStore()
	candidate, op := api4ReviewedCandidate("checkout", http.MethodPost, "/api/payment/123")
	profile1, _ := api4PromoteForTest(t, store, candidate, op, positiveSchemaUnknownAllow)
	if _, err := store.setMode(op.ID, positiveSchemaModeDetect, "tester", "shadow", 422); err != nil {
		t.Fatal(err)
	}
	if _, err := store.setMode(op.ID, positiveSchemaModeEnforce, "tester", "promote", 422); err != nil {
		t.Fatal(err)
	}
	candidate.Version++
	candidate.Review.CandidateVersion = candidate.Version
	candidate.UpdatedAt = time.Now().UTC()
	profile2, _, err := store.promote(candidate, op, positiveSchemaPromoteOptions{UnknownFields: positiveSchemaUnknownDeny, EnforceEnums: true, Activate: true, CreatedBy: "tester", BlockStatus: 422})
	if err != nil {
		t.Fatal(err)
	}
	if profile1.ID == profile2.ID {
		t.Fatal("new candidate version should create a new immutable profile version")
	}
	rolled, err := store.rollback(op.ID, "tester", "bad profile")
	if err != nil {
		t.Fatal(err)
	}
	if rolled.ProfileVersionID != profile1.ID || rolled.Mode != positiveSchemaModeEnforce {
		t.Fatalf("rollback=%+v want profile1 ENFORCE", rolled)
	}
}

func TestAPI4PersistenceRestoresRuntimePolicyAndViolations(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	store := newPositiveSchemaStore()
	candidate, op := api4ReviewedCandidate("checkout", http.MethodPost, "/api/payment/123")
	_, _ = api4PromoteForTest(t, store, candidate, op, positiveSchemaUnknownDeny)
	if _, err := store.setMode(op.ID, positiveSchemaModeDetect, "tester", "shadow", 400); err != nil {
		t.Fatal(err)
	}
	_ = api4Serve(store, "checkout", `{"amount":"bad","currency":"TWD"}`, nil)
	if err := store.save(configPath); err != nil {
		t.Fatal(err)
	}
	restored := newPositiveSchemaStore()
	if err := restored.load(configPath); err != nil {
		t.Fatal(err)
	}
	policy, ok := restored.runtimePolicy(op.ID)
	if !ok || policy.Deployment.Mode != positiveSchemaModeDetect {
		t.Fatalf("runtime policy not restored: ok=%v policy=%+v", ok, policy.Deployment)
	}
	if got := restored.listViolations(op.ID, 10); len(got) == 0 {
		t.Fatal("violation evidence was not restored")
	}
}

func TestAPI4OversizedBodyCannotPassOnPrefixOnly(t *testing.T) {
	store := newPositiveSchemaStore()
	candidate, op := api4ReviewedCandidate("checkout", http.MethodPost, "/api/payment/123")
	_, _ = api4PromoteForTest(t, store, candidate, op, positiveSchemaUnknownAllow)
	if _, err := store.setMode(op.ID, positiveSchemaModeDetect, "tester", "shadow", 400); err != nil {
		t.Fatal(err)
	}
	if _, err := store.setMode(op.ID, positiveSchemaModeEnforce, "tester", "promote", 400); err != nil {
		t.Fatal(err)
	}
	body := append([]byte(`{"amount":12,"currency":"TWD","padding":"`), bytes.Repeat([]byte("x"), passiveBodyLimit+1024)...)
	body = append(body, []byte(`"}`)...)
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	handler := requestBodyPrefixWrap(false, false, true, store.wrap("checkout", next))
	req := httptest.NewRequest(http.MethodPost, "http://api.example.test/api/payment/123", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("oversized body status=%d want 400", rec.Code)
	}
	violations := store.listViolations(op.ID, 20)
	found := false
	for _, v := range violations {
		if v.Type == "BODY_TOO_LARGE_FOR_SCHEMA_VALIDATION" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing oversized-body violation: %+v", violations)
	}
}
