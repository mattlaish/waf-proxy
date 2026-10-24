package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestConsoleDiscardDraftPreservesActiveConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	live := []byte("{\"engine_mode\":\"DetectionOnly\",\"marker\":\"live-secret\"}\n")
	if err := os.WriteFile(path, live, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(draftConfigPath(path), []byte("saved draft"), 0o600); err != nil {
		t.Fatal(err)
	}
	srv := &server{configPath: path}
	rt := &runtimeState{cfg: defaultConfig()}
	srv.rt.Store(rt)
	a := &adminServer{srv: srv, token: "test-admin", sessions: newSessionStore(), audit: newAuditLog()}
	handler := a.handler()
	for _, target := range []string{"/api/config", "/api/config?draft=0"} {
		req := httptest.NewRequest(http.MethodDelete, target, nil)
		req.Header.Set("Authorization", "Bearer test-admin")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("active config DELETE must fail: %s status=%d", target, rec.Code)
		}
		if _, err := os.Stat(draftConfigPath(path)); err != nil {
			t.Fatalf("invalid discard removed draft: %v", err)
		}
	}
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodDelete, "/api/config?draft=1", nil)
		req.Header.Set("Authorization", "Bearer test-admin")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("discard must be durable/idempotent: status=%d body=%s", rec.Code, rec.Body)
		}
		if _, err := os.Stat(draftConfigPath(path)); !os.IsNotExist(err) {
			t.Fatalf("draft still present: %v", err)
		}
		got, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(got, live) || srv.rt.Load() != rt {
			t.Fatalf("discard changed active config/runtime: err=%v", err)
		}
	}
}

func TestConsoleDiscardDraftRequiresOperator(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(draftConfigPath(path), []byte("saved work"), 0o600); err != nil {
		t.Fatal(err)
	}
	a := &adminServer{srv: &server{configPath: path}, token: "test-admin", sessions: newSessionStore(), audit: newAuditLog()}
	handler := a.handler()
	for _, role := range []string{"", roleViewer, roleReviewer} {
		token := ""
		want := http.StatusUnauthorized
		if role != "" {
			token = a.sessions.create("test-"+role, role)
			want = http.StatusForbidden
		}
		req := httptest.NewRequest(http.MethodDelete, "/api/config?draft=1", nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != want {
			t.Fatalf("role %q: status=%d want=%d", role, rec.Code, want)
		}
		if _, err := os.Stat(draftConfigPath(path)); err != nil {
			t.Fatalf("denied operation changed draft: %v", err)
		}
	}
	req := httptest.NewRequest(http.MethodDelete, "/api/config?draft=1", nil)
	req.Header.Set("Authorization", "Bearer "+a.sessions.create("operator-user", roleOperator))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("operator must be able to discard a draft: status=%d body=%s", rec.Code, rec.Body)
	}
}

func TestConsoleDiscardDraftFailureIsNotSuccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	draftPath := draftConfigPath(path)
	if err := os.Mkdir(draftPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(draftPath, "do-not-delete"), []byte("work"), 0o600); err != nil {
		t.Fatal(err)
	}
	a := &adminServer{srv: &server{configPath: path}, token: "test-admin", sessions: newSessionStore(), audit: newAuditLog()}
	req := httptest.NewRequest(http.MethodDelete, "/api/config?draft=1", nil)
	req.Header.Set("Authorization", "Bearer test-admin")
	rec := httptest.NewRecorder()
	a.handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("failed discard must not claim success: status=%d body=%s", rec.Code, rec.Body)
	}
	if _, err := os.Stat(filepath.Join(draftPath, "do-not-delete")); err != nil {
		t.Fatalf("failure changed draft contents: %v", err)
	}
}

func TestConsoleDiscardDraftRejectsMissingStorage(t *testing.T) {
	for _, srv := range []*server{nil, {}} {
		a := &adminServer{srv: srv}
		req := httptest.NewRequest(http.MethodDelete, "/api/config?draft=1", nil)
		rec := httptest.NewRecorder()
		a.handleDiscardConfigDraft(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("missing storage must not resolve to a relative draft: status=%d", rec.Code)
		}
	}
}
