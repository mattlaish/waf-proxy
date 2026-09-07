package main

import (
	"hash/fnv"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func healthyPoolMembers(n int) []*memberRuntime {
	ms := make([]*memberRuntime, 0, n)
	for i := 0; i < n; i++ {
		m := &memberRuntime{
			node:   string(rune('a' + i)),
			target: &url.URL{Scheme: "http", Host: "127.0.0.1:8080"},
			weight: 1,
		}
		atomic.StoreInt32(&m.healthy, 1)
		ms = append(ms, m)
	}
	return ms
}

func TestFNV32MatchesStdlibFNV1a(t *testing.T) {
	for _, s := range []string{"", "192.0.2.44", "2001:db8::1", "client.example"} {
		h := fnv.New32a()
		_, _ = h.Write([]byte(s))
		if got, want := fnv32(s), h.Sum32(); got != want {
			t.Fatalf("fnv32(%q) = %d, want %d", s, got, want)
		}
	}
}

func TestPoolPickP0BHasNoAllocations(t *testing.T) {
	methods := []string{"round_robin", "least_conn", "ip_hash", "random"}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			p := &poolRuntime{method: method, members: healthyPoolMembers(8)}
			allocs := testing.AllocsPerRun(1000, func() {
				if p.pick("192.0.2.44") == nil {
					t.Fatal("pick returned nil")
				}
			})
			if allocs != 0 {
				t.Fatalf("pick allocations = %v, want 0", allocs)
			}
		})
	}
}

func TestLBTransportSelectsTargetAndAccountsWithoutContextValue(t *testing.T) {
	m := &memberRuntime{
		node:   "app1",
		target: &url.URL{Scheme: "https", Host: "backend.example:8443"},
		weight: 1,
	}
	atomic.StoreInt32(&m.healthy, 1)
	p := &poolRuntime{method: "round_robin", members: []*memberRuntime{m}}

	base := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got, want := r.URL.Scheme, "https"; got != want {
			t.Fatalf("scheme = %q, want %q", got, want)
		}
		if got, want := r.URL.Host, "backend.example:8443"; got != want {
			t.Fatalf("URL host = %q, want %q", got, want)
		}
		if got := r.Host; got != "" {
			t.Fatalf("Host = %q, want empty so Transport uses backend host", got)
		}
		if got, want := r.URL.RequestURI(), "/api/v1?q=1"; got != want {
			t.Fatalf("request URI = %q, want %q", got, want)
		}
		if got := atomic.LoadInt64(&m.active); got != 1 {
			t.Fatalf("active during RoundTrip = %d, want 1", got)
		}
		return &http.Response{
			StatusCode: 200,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("ok")),
			Request:    r,
		}, nil
	})

	req, err := http.NewRequest(http.MethodGet, "http://waf.local/api/v1?q=1", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.RemoteAddr = "192.0.2.44:12345"
	resp, err := (lbTransport{base: base, pool: p}).RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	if got := atomic.LoadInt64(&m.active); got != 1 {
		t.Fatalf("active before body close = %d, want 1", got)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if got := atomic.LoadInt64(&m.active); got != 0 {
		t.Fatalf("active after body close = %d, want 0", got)
	}
}

func TestLBTransportPreservesHostWhenConfigured(t *testing.T) {
	m := &memberRuntime{target: &url.URL{Scheme: "http", Host: "127.0.0.1:8080"}, weight: 1}
	atomic.StoreInt32(&m.healthy, 1)
	p := &poolRuntime{members: []*memberRuntime{m}}
	base := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got, want := r.Host, "public.example"; got != want {
			t.Fatalf("Host = %q, want %q", got, want)
		}
		return &http.Response{StatusCode: 204, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
	})
	req, err := http.NewRequest(http.MethodGet, "http://public.example/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "public.example"
	req.RemoteAddr = "192.0.2.44:12345"
	resp, err := (lbTransport{base: base, pool: p, preserveHost: true}).RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
}

func BenchmarkPoolPickP0B(b *testing.B) {
	for _, method := range []string{"round_robin", "least_conn", "ip_hash", "random"} {
		b.Run(method, func(b *testing.B) {
			p := &poolRuntime{method: method, members: healthyPoolMembers(8)}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = p.pick("192.0.2.44")
			}
		})
	}
}
