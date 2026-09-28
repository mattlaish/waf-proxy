package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestAPIOperationNormalizationHardening(t *testing.T) {
	cases := map[string]string{
		"/api/users/12345": "/api/users/{id}",
		"/api/payment/550e8400-e29b-41d4-a716-446655440000": "/api/payment/{id}",
		"/api/report/20260921":                              "/api/report/{date}",
		"/api/item/01ARZ3NDEKTSV4RRFFQ69G5FAV":              "/api/item/{id}",
		"/api/object/abc1234567890XYZuvw987":                "/api/object/{id}",
	}
	for in, want := range cases {
		if got := normalizeAPIOperationPath(in); got != want {
			t.Fatalf("%s: got %s want %s", in, got, want)
		}
	}
	if !classifyPotentialSlug("order-history") {
		t.Fatal("expected slug candidate")
	}
	if got := normalizeAPIOperationPath("/api/articles/order-history"); got != "/api/articles/order-history" {
		t.Fatalf("slugs must not auto-normalize: %s", got)
	}
}

func TestAPIOperationFingerprintStable(t *testing.T) {
	if operationFingerprint("get", "/api/a") != operationFingerprint("GET", "/api/a") {
		t.Fatal("fingerprint must normalize method")
	}
}

func TestAPIOperationMetadataAndPathParameters(t *testing.T) {
	store := newAPIOperationStore()
	op := store.note("payments", "api.example.test:443", "post", "/api/users/12345", "application/json; charset=utf-8", "Bearer", 201)
	if op.ID == "" || op.Host != "api.example.test" || op.Method != "POST" || op.Path != "/api/users/{id}" {
		t.Fatalf("unexpected operation: %+v", op)
	}
	if op.ContentTypes["application/json"] != 1 || op.AuthObserved["bearer"] != 1 || op.Status["201"] != 1 {
		t.Fatalf("metadata not recorded: %+v", op)
	}
	if len(op.PathParameters) != 1 || op.PathParameters[0].Format != "integer" {
		t.Fatalf("path parameter classification missing: %+v", op.PathParameters)
	}
	if len(op.RawPathExamples) != 1 || op.RawPathExamples[0] != "/api/users/12345" {
		t.Fatalf("raw example missing: %+v", op.RawPathExamples)
	}
	if got := normalizeHost("[2001:db8::1]:8443"); got != "2001:db8::1" {
		t.Fatalf("IPv6 host normalization: %q", got)
	}
}

func TestAPIOperationPersistenceAndControls(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	store := newAPIOperationStore()
	op := store.note("site-a", "api.example", "GET", "/v1/users/42", "application/json", "none", 200)
	if _, err := store.ignore(op.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := store.reclassify(op.ID, "/v1/users/{user_id}", "API"); err != nil {
		t.Fatal(err)
	}
	if err := store.save(configPath); err != nil {
		t.Fatal(err)
	}
	loaded := newAPIOperationStore()
	if err := loaded.load(configPath); err != nil {
		t.Fatal(err)
	}
	got, ok := loaded.get(op.ID)
	if !ok || !got.Ignored || got.Path != "/v1/users/{user_id}" || got.Fingerprint == "" {
		t.Fatalf("persistence mismatch: ok=%v op=%+v", ok, got)
	}
	// Manual normalized-path overrides keep the stable discovery ID so future
	// observations of the original raw shape continue the same durable record.
	again := loaded.note("site-a", "api.example", "GET", "/v1/users/99", "application/json", "none", 200)
	if again.ID != op.ID || again.Path != "/v1/users/{user_id}" || again.Samples != 2 {
		t.Fatalf("reclassified operation continuity lost: %+v", again)
	}
	if _, err := loaded.reclassify(op.ID, "/bad path", "API"); err == nil {
		t.Fatal("expected invalid path rejection")
	}
	if _, err := loaded.reclassify(op.ID, "/v1/users/{id}", "arbitrary"); err == nil {
		t.Fatal("expected classification allow-list rejection")
	}
}

func TestAPISecurityAtomicPersistenceConcurrentWriters(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "api-security-state.json")

	const writers = 32
	var wg sync.WaitGroup
	errCh := make(chan error, writers)
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errCh <- atomicWriteJSON(path, map[string]any{"writer": i, "valid": true})
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent atomicWriteJSON failed: %v", err)
		}
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("final state is not valid JSON: %v", err)
	}
	if got["valid"] != true {
		t.Fatalf("unexpected final state: %v", got)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".api-security-state.json.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary persistence files leaked: %v", matches)
	}
}
