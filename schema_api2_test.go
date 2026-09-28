package main

import (
	"context"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSchemaCollectorNestedQueryPathHeaderAndPrivacy(t *testing.T) {
	body := `{"amount":10.5,"currency":"TWD","customer":{"id":"550e8400-e29b-41d4-a716-446655440000","name":"Alice","alias":"PublicNickname"},"items":[{"sku":"ABC-1","qty":2}]}`
	r, err := http.NewRequest(http.MethodPost, "https://api.example.test/api/users/12345?limit=10&mode=compact", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer should-never-be-sampled")
	r.Header.Set("X-API-Key", "super-secret-api-key")
	r.Header.Set("X-Region", "tw")
	capture := &requestCapture{bodyPrefix: []byte(body)}
	r = r.WithContext(context.WithValue(r.Context(), requestCaptureContextKey{}, capture))

	meta := buildAPIObservationMeta(r, r.URL.RawQuery, r.Header.Get("Content-Type"))
	if meta.AuthScheme != "apikey+bearer" || meta.Host != "api.example.test" {
		t.Fatalf("unexpected meta: %+v", meta)
	}
	samples := collectSchemaSamplesFromObservation(r.URL.Path, r.URL.RawQuery, r.Header.Get("Content-Type"), meta)
	byKey := map[string]schemaFieldSample{}
	for _, sample := range samples {
		byKey[sample.Location+"|"+sample.Path] = sample
	}
	for _, key := range []string{"path|param_1", "query|limit", "query|mode", "header|x-region", "body|amount", "body|currency", "body|customer.id", "body|items", "body|items[].sku", "body|items[].qty"} {
		if _, ok := byKey[key]; !ok {
			t.Fatalf("missing sample %s in %+v", key, samples)
		}
	}
	if _, ok := byKey["header|authorization"]; ok {
		t.Fatal("authorization header must never be sampled")
	}
	if _, ok := meta.Headers["x-api-key"]; ok {
		t.Fatal("X-API-Key must be excluded before the observation queue")
	}
	if _, ok := byKey["header|x-api-key"]; ok {
		t.Fatal("X-API-Key must never be sampled")
	}
	name := byKey["body|customer.name"]
	if !name.Sensitive || name.EnumValue != "" || name.ValueHash != "" {
		t.Fatalf("PII-like name must be redacted: %+v", name)
	}
	id := byKey["body|customer.id"]
	if id.Format != "uuid" || id.ValueHash != "" {
		t.Fatalf("UUID values must not be retained/fingerprinted: %+v", id)
	}
	alias := byKey["body|customer.alias"]
	if alias.EnumValue != "" || alias.ValueHash != "" {
		t.Fatalf("free-form string values must not be enumed/fingerprinted by default: %+v", alias)
	}
	header := byKey["header|x-region"]
	if header.ValueHash != "" {
		t.Fatalf("header values must never be fingerprinted: %+v", header)
	}
}

func TestSchemaLearningCandidateLifecycleStatisticsAndEnum(t *testing.T) {
	store := newSchemaStore()
	op := apiOperation{ID: "op-1", Method: "POST", Path: "/api/payment"}
	for i := 0; i < 20; i++ {
		samples := []schemaFieldSample{
			{Path: "amount", Location: "body", Type: "number", ValueHash: "amount-hash"},
			{Path: "currency", Location: "body", Type: "string", EnumValue: "TWD", ValueHash: "currency-hash"},
		}
		if i < 10 {
			samples = append(samples, schemaFieldSample{Path: "memo", Location: "body", Type: "string", Sensitive: true})
		}
		store.note(op, samples)
	}
	candidate, ok := store.findByOperationID(op.ID)
	if !ok {
		t.Fatal("candidate missing")
	}
	if candidate.Status != "CANDIDATE" || candidate.SampleCount != 20 || candidate.Confidence <= 0 {
		t.Fatalf("unexpected candidate: %+v", candidate)
	}
	fields := schemaFieldsByKey(candidate)
	if !fields["body|amount"].RequiredCandidate || fields["body|amount"].PresenceRate != 1 {
		t.Fatalf("amount required evidence incorrect: %+v", fields["body|amount"])
	}
	if fields["body|memo"].RequiredCandidate || fields["body|memo"].PresenceRate != 0.5 {
		t.Fatalf("memo presence incorrect: %+v", fields["body|memo"])
	}
	currency := fields["body|currency"]
	if len(currency.EnumCandidate) != 1 || currency.EnumCandidate[0] != "TWD" {
		t.Fatalf("enum candidate missing: %+v", currency)
	}
	if currency.DistinctCardinality != 1 {
		t.Fatalf("cardinality=%d want 1", currency.DistinctCardinality)
	}
}

func TestSchemaReviewedCandidateBecomesCandidateWhenEvidenceChanges(t *testing.T) {
	store := newSchemaStore()
	op := apiOperation{ID: "op-review", Method: "POST", Path: "/api/review"}
	for i := 0; i < 20; i++ {
		store.note(op, []schemaFieldSample{{Path: "kind", Location: "body", Type: "string", EnumValue: "A", ValueHash: "a"}})
	}
	candidate, _ := store.findByOperationID(op.ID)
	review := SchemaReviewRecommendation{Recommendation: "approved", Confidence: 100, Reason: "operator accepted", ReviewedAt: time.Now().UTC(), Source: "operator"}
	approved, ok := store.setReview(candidate.ID, "REVIEWED", review)
	if !ok || approved.Status != "REVIEWED" || approved.Review == nil || approved.Review.CandidateVersion != approved.Version {
		t.Fatalf("review not recorded: %+v", approved)
	}
	store.note(op, []schemaFieldSample{{Path: "kind", Location: "body", Type: "string", EnumValue: "A", ValueHash: "a"}})
	stable, _ := store.findByOperationID(op.ID)
	if stable.Status != "REVIEWED" || stable.Version != approved.Version {
		t.Fatalf("equivalent evidence must not invalidate review: %+v", stable)
	}
	store.note(op, []schemaFieldSample{{Path: "kind", Location: "body", Type: "string", EnumValue: "B", ValueHash: "b"}})
	changed, _ := store.findByOperationID(op.ID)
	if changed.Status != "CANDIDATE" {
		t.Fatalf("new evidence must require re-review: %+v", changed)
	}
	if changed.Review == nil || changed.Review.CandidateVersion >= changed.Version {
		t.Fatalf("previous review should be retained as stale evidence: %+v", changed.Review)
	}
}

func TestSchemaPersistenceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	store := newSchemaStore()
	op := apiOperation{ID: "op-persist", Method: "GET", Path: "/api/items/{id}"}
	for i := 0; i < 20; i++ {
		store.note(op, []schemaFieldSample{{Path: "param_1", Location: "path", Type: "integer", ValueHash: "42"}})
	}
	if err := store.save(configPath); err != nil {
		t.Fatal(err)
	}
	loaded := newSchemaStore()
	if err := loaded.load(configPath); err != nil {
		t.Fatal(err)
	}
	candidate, ok := loaded.findByOperationID(op.ID)
	if !ok || candidate.SampleCount != 20 || candidate.Status != "CANDIDATE" {
		t.Fatalf("loaded candidate mismatch: ok=%v candidate=%+v", ok, candidate)
	}
	loaded.note(op, []schemaFieldSample{{Path: "param_1", Location: "path", Type: "integer", ValueHash: "43"}})
	candidate, _ = loaded.findByOperationID(op.ID)
	if candidate.SampleCount != 21 {
		t.Fatalf("aggregate state did not resume: %+v", candidate)
	}
}

func schemaFieldsByKey(candidate *SchemaCandidate) map[string]SchemaFieldObservation {
	out := map[string]SchemaFieldObservation{}
	if candidate == nil {
		return out
	}
	for _, f := range candidate.Fields {
		out[f.Location+"|"+f.Path] = f
	}
	return out
}

func TestSchemaAggregateFieldGrowthIsBounded(t *testing.T) {
	store := newSchemaStore()
	op := apiOperation{ID: "op-bounded", Method: "GET", Path: "/api/search"}
	for batch := 0; batch < 3; batch++ {
		samples := make([]schemaFieldSample, 0, schemaMaxFields)
		for i := 0; i < schemaMaxFields; i++ {
			samples = append(samples, schemaFieldSample{Path: fmt.Sprintf("q_%d_%d", batch, i), Location: "query", Type: "string"})
		}
		store.note(op, samples)
	}
	candidate, ok := store.findByOperationID(op.ID)
	if !ok {
		t.Fatal("candidate missing")
	}
	if len(candidate.Fields) != schemaMaxFields {
		t.Fatalf("aggregate fields=%d want bounded %d", len(candidate.Fields), schemaMaxFields)
	}
}

func TestSchemaDominantTypeChangeInvalidatesReview(t *testing.T) {
	store := newSchemaStore()
	op := apiOperation{ID: "op-dominant", Method: "POST", Path: "/api/value"}
	for i := 0; i < 20; i++ {
		store.note(op, []schemaFieldSample{{Path: "value", Location: "body", Type: "string"}})
	}
	candidate, _ := store.findByOperationID(op.ID)
	reviewed, ok := store.setReview(candidate.ID, "REVIEWED", SchemaReviewRecommendation{Recommendation: "approved", Confidence: 100, Reason: "accepted", ReviewedAt: time.Now().UTC(), Source: "operator"})
	if !ok || reviewed.Status != "REVIEWED" {
		t.Fatalf("review failed: %+v", reviewed)
	}
	for i := 0; i < 25; i++ {
		store.note(op, []schemaFieldSample{{Path: "value", Location: "body", Type: "integer"}})
	}
	changed, _ := store.findByOperationID(op.ID)
	if changed.Status != "CANDIDATE" || changed.Version <= reviewed.Version {
		t.Fatalf("dominant type change did not invalidate review: before=%+v after=%+v", reviewed, changed)
	}
}

func TestInferAuthSchemePreservesMultiMechanismEvidence(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "https://api.example.test/", nil)
	r.Header.Set("Authorization", "Bearer token")
	r.Header.Set("X-API-Key", "key")
	r.TLS.PeerCertificates = []*x509.Certificate{{}}
	if got := inferAuthScheme(r); got != "apikey+bearer+mtls" {
		t.Fatalf("auth scheme=%q", got)
	}
}
