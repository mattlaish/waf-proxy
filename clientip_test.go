package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func newTestRequest(method, target string) *http.Request {
	return httptest.NewRequest(method, target, nil)
}

func mustClientIPResolver(t *testing.T, cidrs ...string) *clientIPResolver {
	t.Helper()
	r, err := newClientIPResolver(cidrs)
	if err != nil {
		t.Fatalf("newClientIPResolver: %v", err)
	}
	return r
}

func TestClientIPResolverTrustBoundary(t *testing.T) {
	tests := []struct {
		name       string
		cidrs      []string
		remoteAddr string
		forwarded  string
		want       string
	}{
		{
			name:       "untrusted peer cannot spoof",
			cidrs:      []string{"10.0.0.0/8"},
			remoteAddr: "192.0.2.10:1234",
			forwarded:  "203.0.113.9",
			want:       "192.0.2.10",
		},
		{
			name:       "single trusted proxy",
			cidrs:      []string{"192.0.2.0/24"},
			remoteAddr: "192.0.2.10:1234",
			forwarded:  "203.0.113.9",
			want:       "203.0.113.9",
		},
		{
			name:       "trusted chain is walked right to left",
			cidrs:      []string{"10.0.0.0/8"},
			remoteAddr: "10.0.0.3:1234",
			forwarded:  "203.0.113.9, 10.0.0.2",
			want:       "203.0.113.9",
		},
		{
			name:       "first untrusted intermediate ends the chain",
			cidrs:      []string{"10.0.0.0/8"},
			remoteAddr: "10.0.0.3:1234",
			forwarded:  "198.51.100.7, 192.0.2.44",
			want:       "192.0.2.44",
		},
		{
			name:       "IPv6 trusted proxy",
			cidrs:      []string{"2001:db8::/64"},
			remoteAddr: "[2001:db8::10]:1234",
			forwarded:  "2001:db9::5",
			want:       "2001:db9::5",
		},
		{
			name:       "malformed chain fails back to peer",
			cidrs:      []string{"10.0.0.0/8"},
			remoteAddr: "10.0.0.3:1234",
			forwarded:  "203.0.113.9, unknown",
			want:       "10.0.0.3",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := newTestRequest(http.MethodGet, "http://example.test/")
			r.RemoteAddr = tc.remoteAddr
			r.Header.Set("X-Forwarded-For", tc.forwarded)
			if got := mustClientIPResolver(t, tc.cidrs...).resolve(r); got != tc.want {
				t.Fatalf("resolve() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestClientIPResolverRejectsOversizedChain(t *testing.T) {
	r := newTestRequest(http.MethodGet, "http://example.test/")
	r.RemoteAddr = "10.0.0.3:1234"
	r.Header.Set("X-Forwarded-For", strings.Repeat("1", maxForwardedForBytes+1))
	if got := mustClientIPResolver(t, "10.0.0.0/8").resolve(r); got != "10.0.0.3" {
		t.Fatalf("oversized chain resolved to %q", got)
	}
}

func TestTrustedProxyValidationAndLegacyMigration(t *testing.T) {
	for _, cidrs := range [][]string{{"not-a-cidr"}, {"10.0.0.1/8", "10.0.0.0/8"}, {""}} {
		if err := validateTrustedProxyCIDRs(cidrs); err == nil {
			t.Fatalf("invalid trusted proxy list accepted: %#v", cidrs)
		}
	}

	cfg := Config{AI: AIConfig{TrustedProxyCIDRs: []string{"192.0.2.0/24"}}}
	migrateTrustedProxyConfig(&cfg)
	if len(cfg.TrustedProxyCIDRs) != 1 || cfg.TrustedProxyCIDRs[0] != "192.0.2.0/24" {
		t.Fatalf("legacy trusted proxies not migrated: %#v", cfg.TrustedProxyCIDRs)
	}
	if cfg.AI.TrustedProxyCIDRs != nil {
		t.Fatalf("legacy AI trusted proxies retained: %#v", cfg.AI.TrustedProxyCIDRs)
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(b), "trusted_proxy_cidrs") != 1 {
		t.Fatalf("trusted proxies did not serialize once: %s", b)
	}

	cfg = Config{EngineMode: "DetectionOnly", TrustedProxyCIDRs: []string{"bad"}}
	if err := cfg.validateDraft(); err == nil {
		t.Fatal("draft accepted invalid global trusted proxy CIDR")
	}
}

func TestClientIPResolverFeedsBackendCanonicalHeaders(t *testing.T) {
	gotHeaders := make(chan [2]string, 1)
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders <- [2]string{r.Header.Get("X-Forwarded-For"), r.Header.Get("X-Real-IP")}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer backend.Close()
	target, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}
	pool := &poolRuntime{name: "test", method: "round_robin", members: []*memberRuntime{{
		node: "backend", target: target, weight: 1,
	}}}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	proxy := buildProxy(pool, SiteConfig{Name: "test"}, Config{BackendTimeoutSec: 2}, log, nil)
	handler := mustClientIPResolver(t, "10.0.0.0/8").wrap(proxy)

	r := newTestRequest(http.MethodGet, "http://waf.test/")
	r.RemoteAddr = "10.0.0.3:4321"
	r.Header.Set("X-Forwarded-For", "198.51.100.9, 10.0.0.2")
	r.Header.Set("X-Real-IP", "203.0.113.200")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusNoContent {
		t.Fatalf("proxy status = %d, want %d", w.Code, http.StatusNoContent)
	}
	got := <-gotHeaders
	if got[0] != "198.51.100.9" || got[1] != "198.51.100.9" {
		t.Fatalf("backend headers = XFF %q, X-Real-IP %q", got[0], got[1])
	}
}
