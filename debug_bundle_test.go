package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDebugEvidenceStoreTenantIsolationAndExport(t *testing.T) {
	s := NewDebugEvidenceStore(10, time.Hour)
	s.Put(DebugBundle{
		Tenant:        "site-a",
		TransactionID: "tx-a",
		Request: map[string]any{
			"path":          "/login",
			"authorization": "Bearer super-secret",
			"cookie":        "session=secret",
		},
		Coraza: map[string]any{"matched_rules": []int{942100}},
	})
	s.Put(DebugBundle{Tenant: "site-b", TransactionID: "tx-b", Request: map[string]any{"path": "/other"}})

	if _, ok := s.Get("site-b", "tx-a"); ok {
		t.Fatal("cross-tenant lookup returned evidence")
	}
	if got := s.List("site-a", 10); len(got) != 1 || got[0].TransactionID != "tx-a" {
		t.Fatalf("unexpected site-a list: %#v", got)
	}

	var buf bytes.Buffer
	if err := s.ExportIncident("site-a", "tx-a", &buf); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string][]byte{}
	for _, f := range zr.File {
		r, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			t.Fatal(err)
		}
		entries[f.Name] = b
	}
	if _, ok := entries["manifest.json"]; !ok {
		t.Fatal("manifest.json missing")
	}
	if bytes.Contains(entries["request.json"], []byte("super-secret")) || bytes.Contains(entries["request.json"], []byte("session=secret")) {
		t.Fatal("sensitive request material leaked into incident bundle")
	}
	if bytes.Contains(buf.Bytes(), []byte("site-b")) || bytes.Contains(buf.Bytes(), []byte("tx-b")) {
		t.Fatal("cross-tenant evidence leaked into incident bundle")
	}
	var manifest struct {
		Tenant string            `json:"tenant"`
		Files  map[string]string `json:"files"`
	}
	if err := json.Unmarshal(entries["manifest.json"], &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Tenant != "site-a" || len(manifest.Files) == 0 {
		t.Fatalf("unexpected manifest: %#v", manifest)
	}
}

func TestDebugEvidenceStoreTTLAndCaptureWindow(t *testing.T) {
	s := NewDebugEvidenceStore(10, time.Millisecond)
	if err := s.EnableTenant("site-a", 20*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if !s.ShouldCapture("site-a", time.Now().UTC()) {
		t.Fatal("capture should be enabled")
	}
	s.Put(DebugBundle{Tenant: "site-a", TransactionID: "tx-a"})
	time.Sleep(5 * time.Millisecond)
	removed := s.Cleanup(time.Now().UTC().Add(time.Second))
	if removed != 1 {
		t.Fatalf("expected one expired item removed, got %d", removed)
	}
	if s.ShouldCapture("site-a", time.Now().UTC().Add(time.Second)) {
		t.Fatal("expired capture window remained active")
	}
}

func TestDebugEvidenceWrapDisabledAndEnabled(t *testing.T) {
	s := NewDebugEvidenceStore(10, time.Hour)
	h := debugEvidenceWrap(s, "site-a", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	}))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "https://example.test/path", nil))
	if rr.Header().Get("X-WAF-Request-ID") != "" {
		t.Fatal("disabled capture added a request ID")
	}
	if len(s.List("site-a", 10)) != 0 {
		t.Fatal("disabled capture stored evidence")
	}

	if err := s.EnableTenant("site-a", time.Minute); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, "https://example.test/path", nil)
	r.Header.Set("Authorization", "Bearer never-capture-me")
	r.Header.Set("User-Agent", "debug-test")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	id := rr.Header().Get("X-WAF-Request-ID")
	if id == "" {
		t.Fatal("enabled capture did not add request ID")
	}
	items := s.List("site-a", 10)
	if len(items) != 1 || items[0].TransactionID != id {
		t.Fatalf("unexpected captured evidence: %#v", items)
	}
	if got, _ := items[0].Request["headers"].(map[string]any); got != nil {
		for k, v := range got {
			if strings.EqualFold(k, "Authorization") || strings.Contains(strings.ToLower(strings.TrimSpace(toDebugString(v))), "never-capture-me") {
				t.Fatalf("authorization leaked into request evidence: %#v", got)
			}
		}
	}
	if items[0].Response["status"] != http.StatusCreated {
		t.Fatalf("response status not captured: %#v", items[0].Response)
	}
}

func toDebugString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
