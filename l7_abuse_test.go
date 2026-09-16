package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func withClientIdentity(ctx context.Context, d ClientIdentityDecision) context.Context {
	return context.WithValue(ctx, clientIdentityContextKey{}, d)
}

func TestL7AbuseUsesTrustedClientIdentity(t *testing.T) {
	c := newL7AbuseController(L7AbuseConfig{Enabled: true, RequestsPerWindow: 1, WindowSeconds: 60})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	h := c.wrap("site-a", next)
	req := httptest.NewRequest("GET", "http://example/", nil)
	req = req.WithContext(withClientIdentity(req.Context(), ClientIdentityDecision{ResolvedClientIP: "203.0.113.10"}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("first request=%d", rr.Code)
	}
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 429 {
		t.Fatalf("second request=%d", rr.Code)
	}
}
