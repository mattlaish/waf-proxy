package main

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestPhase4SecurityAllowOverridesDenyAndRequestID(t *testing.T) {
	s := newSecurityManager()
	if err := s.configure(SecurityConfig{
		Enabled:     true,
		AllowCIDRs:  []CIDRAccessRule{{CIDR: "192.0.2.7/32"}},
		DenyCIDRs:   []CIDRAccessRule{{CIDR: "192.0.2.0/24"}},
		BlockStatus: http.StatusForbidden,
	}); err != nil {
		t.Fatal(err)
	}
	hit := false
	h := s.wrap("site", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		if requestIDFromContext(r.Context()) == "" {
			t.Fatal("request ID missing from context")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	r := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	r.RemoteAddr = "192.0.2.7:1234"
	r.Header.Set(requestIDHeader, "attacker-controlled")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if !hit || w.Code != http.StatusNoContent {
		t.Fatalf("allow-over-deny failed: hit=%v code=%d", hit, w.Code)
	}
	if got := w.Header().Get(requestIDHeader); got == "" || got == "attacker-controlled" {
		t.Fatalf("unsafe request id %q", got)
	}
}

func TestPhase4SecurityDenyCustomPageAndExpiredRule(t *testing.T) {
	s := newSecurityManager()
	if err := s.configure(SecurityConfig{
		Enabled: true,
		DenyCIDRs: []CIDRAccessRule{
			{CIDR: "198.51.100.0/24"},
			{CIDR: "203.0.113.0/24", ExpiresAt: time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)},
		},
		BlockPage: "blocked {{.RequestID}} {{.Reason}}",
	}); err != nil {
		t.Fatal(err)
	}
	blocked := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	blocked.RemoteAddr = "198.51.100.9:44"
	bw := httptest.NewRecorder()
	s.wrap("site", http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("deny reached next") })).ServeHTTP(bw, blocked)
	if bw.Code != http.StatusForbidden || !strings.Contains(bw.Body.String(), "cidr_deny") || bw.Header().Get(requestIDHeader) == "" {
		t.Fatalf("unexpected deny: code=%d body=%q", bw.Code, bw.Body.String())
	}

	expiredHit := false
	expired := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	expired.RemoteAddr = "203.0.113.4:55"
	ew := httptest.NewRecorder()
	s.wrap("site", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { expiredHit = true; w.WriteHeader(http.StatusNoContent) })).ServeHTTP(ew, expired)
	if !expiredHit || ew.Code != http.StatusNoContent {
		t.Fatalf("expired deny rule remained active")
	}
}

func TestPhase4SecurityRateAndInflightLimits(t *testing.T) {
	s := newSecurityManager()
	if err := s.configure(SecurityConfig{Enabled: true, RatePerSecond: 1, RateBurst: 1, MaxInflightPerIP: 1}); err != nil {
		t.Fatal(err)
	}
	request := func() *http.Request {
		r := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
		r.RemoteAddr = "203.0.113.8:80"
		return r
	}
	fast := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	w1 := httptest.NewRecorder()
	s.wrap("site", fast).ServeHTTP(w1, request())
	if w1.Code != http.StatusNoContent {
		t.Fatalf("first request rejected: %d", w1.Code)
	}
	w2 := httptest.NewRecorder()
	s.wrap("site", fast).ServeHTTP(w2, request())
	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("rate limit did not reject second request: %d", w2.Code)
	}

	// Isolate in-flight behavior from the token bucket.
	s2 := newSecurityManager()
	if err := s2.configure(SecurityConfig{Enabled: true, MaxInflightPerIP: 1}); err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	slow := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		once.Do(func() { close(entered) })
		<-release
		w.WriteHeader(http.StatusNoContent)
	})
	done := make(chan struct{})
	go func() {
		s2.wrap("site", slow).ServeHTTP(httptest.NewRecorder(), request())
		close(done)
	}()
	<-entered
	w3 := httptest.NewRecorder()
	s2.wrap("site", fast).ServeHTTP(w3, request())
	if w3.Code != http.StatusTooManyRequests {
		t.Fatalf("in-flight limit did not reject: %d", w3.Code)
	}
	close(release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("slow request did not finish")
	}
}

func TestPhase4SecurityTransportCapsAndBoundedBuckets(t *testing.T) {
	s := newSecurityManager()
	cfg := defaultSecurityConfig()
	cfg.Enabled = true
	cfg.MaxConnectionsPerIP = 1
	cfg.TLSHandshakesPerMinute = 60
	cfg.TLSHandshakeBurst = 1
	if err := s.configure(cfg); err != nil {
		t.Fatal(err)
	}

	c1, p1 := net.Pipe()
	defer p1.Close()
	defer c1.Close()
	c2, p2 := net.Pipe()
	defer p2.Close()
	defer c2.Close()
	s.connState(c1, http.StateNew)
	s.connState(c2, http.StateNew)
	if got := s.counters().ConnRejected; got != 1 {
		t.Fatalf("connection rejected=%d, want 1", got)
	}
	s.connState(c1, http.StateClosed)

	peer := &net.TCPAddr{IP: net.ParseIP("203.0.113.9"), Port: 443}
	if !s.allowTLSHandshake(peer) {
		t.Fatal("first TLS handshake unexpectedly rejected")
	}
	if s.allowTLSHandshake(peer) {
		t.Fatal("second immediate TLS handshake unexpectedly allowed")
	}
	if got := s.counters().TLSRejected; got != 1 {
		t.Fatalf("TLS rejected=%d, want 1", got)
	}

	now := time.Now()
	buckets := make(map[string]rateBucket, maxSecurityRateBuckets)
	for i := 0; i < maxSecurityRateBuckets; i++ {
		buckets[fmt.Sprintf("198.51.%d.%d", (i/256)%256, i%256)] = rateBucket{last: now}
	}
	consumeBucket(buckets, "203.0.113.200", 1, 1, now)
	if len(buckets) >= maxSecurityRateBuckets {
		t.Fatalf("bounded bucket map was not pruned: %d", len(buckets))
	}
}
