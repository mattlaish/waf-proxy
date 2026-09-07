package main

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestFixedBufferPoolShape(t *testing.T) {
	p := newProxyBufferPool()
	buf := p.Get()
	if got, want := len(buf), reverseProxyCopyBufferSize; got != want {
		t.Fatalf("buffer len = %d, want %d", got, want)
	}
	if got, want := cap(buf), reverseProxyCopyBufferSize; got != want {
		t.Fatalf("buffer cap = %d, want %d", got, want)
	}
	p.Put(buf[:1024]) // normal subslice of the same backing buffer is accepted.
	buf = p.Get()
	if len(buf) != reverseProxyCopyBufferSize {
		t.Fatalf("reused buffer len = %d, want %d", len(buf), reverseProxyCopyBufferSize)
	}
	// Wrong-capacity buffers are intentionally not retained.
	p.Put(make([]byte, 1))
	p.Put(make([]byte, reverseProxyCopyBufferSize*2))
}

func TestBuildProxyP0ASettings(t *testing.T) {
	pool := &poolRuntime{name: "test"}
	proxy := buildProxy(pool, SiteConfig{Name: "test"}, Config{BackendTimeoutSec: 2},
		slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
	if proxy.BufferPool == nil {
		t.Fatal("buildProxy did not configure BufferPool")
	}
	if proxy.FlushInterval != 0 {
		t.Fatalf("FlushInterval = %v, want 0", proxy.FlushInterval)
	}
	buf := proxy.BufferPool.Get()
	if len(buf) != reverseProxyCopyBufferSize {
		t.Fatalf("proxy buffer len = %d, want %d", len(buf), reverseProxyCopyBufferSize)
	}
	proxy.BufferPool.Put(buf)
}

func BenchmarkFixedBufferPoolGetPut(b *testing.B) {
	p := newProxyBufferPool()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf := p.Get()
		p.Put(buf)
	}
}

type trackingBufferPool struct {
	base       *proxyBufferPool
	gets, puts int
}

func (p *trackingBufferPool) Get() []byte {
	p.gets++
	return p.base.Get()
}

func (p *trackingBufferPool) Put(buf []byte) {
	p.puts++
	p.base.Put(buf)
}

type flushCountingRecorder struct {
	*httptest.ResponseRecorder
	flushes int
}

func (w *flushCountingRecorder) Flush() {
	w.flushes++
	w.ResponseRecorder.Flush()
}

func TestReverseProxyUsesConfiguredBufferPool(t *testing.T) {
	payload := bytes.Repeat([]byte("x"), 64<<10)
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer backend.Close()
	target, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}
	pool := &poolRuntime{name: "test", method: "round_robin", members: []*memberRuntime{{target: target, weight: 1}}}
	proxy := buildProxy(pool, SiteConfig{Name: "test"}, Config{BackendTimeoutSec: 2},
		slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
	tracking := &trackingBufferPool{base: newProxyBufferPool()}
	proxy.BufferPool = tracking

	req := httptest.NewRequest(http.MethodGet, "http://waf.local/", nil)
	req.RemoteAddr = "192.0.2.10:1234"
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("proxy status = %d, want 200", rec.Code)
	}
	if !bytes.Equal(rec.Body.Bytes(), payload) {
		t.Fatal("proxy response payload changed")
	}
	if tracking.gets == 0 || tracking.puts == 0 || tracking.gets != tracking.puts {
		t.Fatalf("buffer pool calls get=%d put=%d", tracking.gets, tracking.puts)
	}
}

func TestReverseProxyStillFlushesEventStreamWithZeroInterval(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: hello\n\n")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	defer backend.Close()
	target, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}
	pool := &poolRuntime{name: "test", method: "round_robin", members: []*memberRuntime{{target: target, weight: 1}}}
	proxy := buildProxy(pool, SiteConfig{Name: "test"}, Config{BackendTimeoutSec: 2},
		slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
	if proxy.FlushInterval != 0 {
		t.Fatalf("FlushInterval = %v, want 0", proxy.FlushInterval)
	}

	req := httptest.NewRequest(http.MethodGet, "http://waf.local/events", nil)
	req.RemoteAddr = "192.0.2.10:1234"
	w := &flushCountingRecorder{ResponseRecorder: httptest.NewRecorder()}
	proxy.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("proxy status = %d, want 200", w.Code)
	}
	if w.flushes == 0 {
		t.Fatal("event-stream response was not flushed with FlushInterval=0")
	}
}
