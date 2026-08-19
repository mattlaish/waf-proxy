package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func smokeSite(name string) *siteRuntime {
	return &siteRuntime{handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Smoke-Site", name)
		w.WriteHeader(http.StatusNoContent)
	})}
}

func TestRuntimeListenerHostRoutingAndDrainHealth(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	notify := newNotifier(log)
	s := &server{log: log, hosts: newHostObserver(log), metrics: newMetrics()}
	s.ha = newHAEngine(log, notify)
	s.rt.Store(&runtimeState{listeners: map[string]*listenerRuntime{
		"127.0.0.1:18080": {
			exact:    map[string]*siteRuntime{"login.example": smokeSite("exact")},
			wildcard: map[string]*siteRuntime{".example.net": smokeSite("wildcard")},
			catchAll: smokeSite("catch-all"),
		},
	}})
	m := newListenerManager(s, log)
	h := m.buildServer("127.0.0.1:18080", false, defaultConfig()).Handler

	for _, tc := range []struct{ host, want string }{
		{"login.example", "exact"},
		{"app.example.net", "wildcard"},
		{"unknown.invalid", "catch-all"},
	} {
		r := httptest.NewRequest(http.MethodGet, "http://"+tc.host+"/", nil)
		r.Host = tc.host
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusNoContent || w.Header().Get("X-Smoke-Site") != tc.want {
			t.Fatalf("host %q routed with status=%d site=%q, want %q", tc.host, w.Code, w.Header().Get("X-Smoke-Site"), tc.want)
		}
	}

	health := func() int {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "http://127.0.0.1:18080/healthz", nil))
		return w.Code
	}
	if got := health(); got != http.StatusOK {
		t.Fatalf("healthy status = %d", got)
	}
	s.draining.Store(true)
	if got := health(); got != http.StatusServiceUnavailable {
		t.Fatalf("draining status = %d", got)
	}
}
