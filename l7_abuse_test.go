package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"waf-proxy/internal/tlsfront"
)

func withClientIdentity(ctx context.Context, d ClientIdentityDecision) context.Context {
	return context.WithValue(ctx, clientIdentityContextKey{}, d)
}

func TestL7AbuseUsesTrustedClientIdentity(t *testing.T) {
	c := newL7AbuseController(L7AbuseConfig{Enabled: true, RequestsPerWindow: 1, WindowSeconds: 60})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	h := c.wrap("site-a", next)
	req := httptest.NewRequest(http.MethodGet, "http://example/", nil)
	req = req.WithContext(withClientIdentity(req.Context(), ClientIdentityDecision{ResolvedClientIP: "203.0.113.10"}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("first request=%d", rr.Code)
	}
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("second request=%d", rr.Code)
	}
}

func TestL7AbuseTokenBucketRefillsContinuously(t *testing.T) {
	c := newL7AbuseController(L7AbuseConfig{Enabled: true, RequestsPerWindow: 2, WindowSeconds: 10})
	t0 := time.Unix(1_700_000_000, 0)

	for i := 0; i < 2; i++ {
		st, ok := c.allowAt("site-a", "203.0.113.10", t0)
		if !ok || st == nil {
			t.Fatalf("initial token %d rejected", i+1)
		}
		c.releaseState(st)
	}
	if _, ok := c.allowAt("site-a", "203.0.113.10", t0); ok {
		t.Fatal("third request at t0 should exhaust the bucket")
	}

	// Half a 10-second window refills exactly one of the two-token capacity.
	st, ok := c.allowAt("site-a", "203.0.113.10", t0.Add(5*time.Second))
	if !ok || st == nil {
		t.Fatal("continuous refill did not restore one token after five seconds")
	}
	c.releaseState(st)
	if _, ok := c.allowAt("site-a", "203.0.113.10", t0.Add(5*time.Second)); ok {
		t.Fatal("bucket admitted two requests after only one token refilled")
	}
}

func TestL7AbuseReleaseRunsOnDownstreamPanic(t *testing.T) {
	c := newL7AbuseController(L7AbuseConfig{Enabled: true, MaxConcurrentConnections: 1})
	panicking := c.wrap("site-a", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))
	req := httptest.NewRequest(http.MethodGet, "http://example/", nil)
	req = req.WithContext(withClientIdentity(req.Context(), ClientIdentityDecision{ResolvedClientIP: "203.0.113.20"}))

	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("downstream panic was not propagated")
			}
		}()
		panicking.ServeHTTP(httptest.NewRecorder(), req)
	}()

	okHandler := c.wrap("site-a", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	okHandler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("active slot leaked after panic; second request=%d", rr.Code)
	}
}

func TestL7AbuseStateIsBoundedPerShard(t *testing.T) {
	c := newL7AbuseController(L7AbuseConfig{Enabled: true, RequestsPerWindow: 1, WindowSeconds: 60})
	now := time.Unix(1_700_000_000, 0)
	const targetShard = uint64(0)
	inserted := 0
	for n := 0; inserted < l7AbuseMaxEntriesPerShard+64; n++ {
		identity := fmt.Sprintf("2001:db8::%x", n)
		key := abuseClientKey{site: "site-a", identity: identity}
		if l7AbuseShardIndex(key) != targetShard {
			continue
		}
		st, ok := c.allowAt(key.site, key.identity, now)
		if !ok {
			t.Fatalf("identity %q unexpectedly rejected", identity)
		}
		c.releaseState(st)
		inserted++
	}

	shard := &c.shards[targetShard]
	shard.mu.Lock()
	got := len(shard.clients)
	shard.mu.Unlock()
	if got > l7AbuseMaxEntriesPerShard {
		t.Fatalf("shard grew to %d entries; cap=%d", got, l7AbuseMaxEntriesPerShard)
	}
}

func TestL7AbuseBucketsAreSiteScoped(t *testing.T) {
	c := newL7AbuseController(L7AbuseConfig{Enabled: true, RequestsPerWindow: 1, WindowSeconds: 60})
	now := time.Unix(1_700_000_000, 0)
	for _, site := range []string{"site-a", "site-b"} {
		st, ok := c.allowAt(site, "203.0.113.30", now)
		if !ok || st == nil {
			t.Fatalf("first request for %s rejected", site)
		}
		c.releaseState(st)
	}
	if _, ok := c.allowAt("site-a", "203.0.113.30", now); ok {
		t.Fatal("site-a bucket should be exhausted independently")
	}
}

func BenchmarkL7AbuseParallelSharded(b *testing.B) {
	c := newL7AbuseController(L7AbuseConfig{Enabled: true})
	identities := make([]string, 1024)
	for i := range identities {
		identities[i] = fmt.Sprintf("198.51.100.%d", i)
	}
	var n atomic.Uint64
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			id := identities[n.Add(1)%uint64(len(identities))]
			st, ok := c.allowAt("site-a", id, time.Now())
			if !ok {
				b.Fatal("unexpected rejection")
			}
			c.releaseState(st)
		}
	})
}

func TestL7TLSHandshakeBucketEnforcesAndRefills(t *testing.T) {
	c := newL7AbuseController(L7AbuseConfig{Enabled: true, TLSHandshakePerWindow: 2, WindowSeconds: 10})
	t0 := time.Unix(1_700_000_000, 0)
	if !c.allowTLSHandshakeAt(":443", "203.0.113.40", t0) || !c.allowTLSHandshakeAt(":443", "203.0.113.40", t0) {
		t.Fatal("initial TLS handshake tokens rejected")
	}
	if c.allowTLSHandshakeAt(":443", "203.0.113.40", t0) {
		t.Fatal("third TLS handshake should exhaust the bucket")
	}
	if !c.allowTLSHandshakeAt(":443", "203.0.113.40", t0.Add(5*time.Second)) {
		t.Fatal("TLS handshake bucket did not refill continuously")
	}
}

func TestL7TLSHandshakeBucketsAreListenerScoped(t *testing.T) {
	c := newL7AbuseController(L7AbuseConfig{Enabled: true, TLSHandshakePerWindow: 1, WindowSeconds: 60})
	now := time.Unix(1_700_000_000, 0)
	if !c.allowTLSHandshakeAt(":443", "203.0.113.41", now) {
		t.Fatal("first listener rejected")
	}
	if !c.allowTLSHandshakeAt(":8443", "203.0.113.41", now) {
		t.Fatal("second listener shared the wrong bucket")
	}
	if c.allowTLSHandshakeAt(":443", "203.0.113.41", now) {
		t.Fatal("first listener bucket should be exhausted")
	}
}

func TestL7TLSHandshakeValidationRejectsExternalFrontend(t *testing.T) {
	cfg := L7AbuseConfig{Enabled: true, TLSHandshakePerWindow: 10, WindowSeconds: 60}
	if err := validateL7AbuseConfig(cfg, tlsfront.AccelerationConfig{Mode: tlsfront.ModeFrontend}); err == nil {
		t.Fatal("expected external TLS frontend to reject Go-side TLS handshake limiting")
	}
	if err := validateL7AbuseConfig(cfg, tlsfront.AccelerationConfig{Mode: tlsfront.ModeGo}); err != nil {
		t.Fatalf("built-in Go TLS should accept handshake limiting: %v", err)
	}
}
