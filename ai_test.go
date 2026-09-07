package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func testAIEngine() *aiEngine {
	return newAIEngine(slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestAIBlocklistIsScopedBySite(t *testing.T) {
	e := testAIEngine()
	e.addBlock(blockEntry{IP: "203.0.113.7", Site: "shop", Expires: time.Now().Add(time.Minute)})
	if _, ok := e.isBlocked("shop", "203.0.113.7"); !ok {
		t.Fatal("shop block missing")
	}
	if _, ok := e.isBlocked("blog", "203.0.113.7"); ok {
		t.Fatal("block leaked to another site")
	}
	e.unblock("shop", "203.0.113.7")
	if _, ok := e.isBlocked("shop", "203.0.113.7"); ok {
		t.Fatal("site block was not removed")
	}
}

func TestAIRedaction(t *testing.T) {
	e := testAIEngine()
	r := newTestRequest("GET", "http://example.test/")
	r.Header.Set("X-Session-Token", "do-not-send")
	r.Header.Set("Content-Type", "application/json")
	h := e.redactHeaders(r, defaultAIConfig())
	if _, ok := h["X-Session-Token"]; ok {
		t.Fatal("sensitive header leaked")
	}
	if h["Content-Type"] != "application/json" {
		t.Fatal("safe header removed")
	}
	q := redactQuery("page=2&access_token=secret&password=pw")
	if strings.Contains(q, "secret") || strings.Contains(q, "pw") {
		t.Fatalf("query secret leaked: %s", q)
	}
}

func TestAIConfigValidation(t *testing.T) {
	cfg := defaultAIConfig()
	cfg.Enabled = true
	cfg.Workers = 0
	if cfg.validate() == nil {
		t.Fatal("invalid worker count accepted")
	}
}

func TestPromptIncludesMatchedRequestContext(t *testing.T) {
	e := testAIEngine()
	p := e.buildPrompt(defaultAIConfig(), analysisJob{method: "POST", path: "/login", query: "user=bob", rules: []int{942100}, matchedData: []string{"' OR 1=1"}})
	for _, want := range []string{"method: POST", "query: user=bob", "942100", "' OR 1=1"} {
		if !strings.Contains(p, want) {
			t.Fatalf("prompt missing %q: %s", want, p)
		}
	}
}

func TestPromptUsesByteBodyPrefix(t *testing.T) {
	e := testAIEngine()
	cfg := defaultAIConfig()
	cfg.IncludeBody = true
	p := e.buildPrompt(cfg, analysisJob{method: "POST", path: "/api", body: []byte(`{"token":"abc"}`)})
	if !strings.Contains(p, `body: {"token":"abc"}`) {
		t.Fatalf("prompt missing byte body: %s", p)
	}
}

func TestParseVerdictRejectsIncompleteAndHandlesExtraBraces(t *testing.T) {
	if _, err := parseVerdict(`{"verdict":"malicious","category":"sqli","reason":"hit"}`); err == nil {
		t.Fatal("missing score accepted")
	}
	v, err := parseVerdict(`preface {not json} {"verdict":"benign","score":7,"category":"normal","reason":"ordinary request"} suffix`)
	if err != nil || v.Verdict != "benign" {
		t.Fatalf("valid embedded verdict not parsed: %#v %v", v, err)
	}
}

func TestDecodeJSONObjectSkipsInvalidBrace(t *testing.T) {
	var got struct {
		Agree bool `json:"agree"`
	}
	if err := decodeJSONObject("note {bad} {\"agree\":true}", &got); err != nil || !got.Agree {
		t.Fatalf("decodeJSONObject failed: %#v %v", got, err)
	}
}

func testAIEngineWithoutWorkers(cfg AIConfig) *aiEngine {
	e := &aiEngine{
		queue:    make(chan analysisJob, 10000),
		log:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		verdicts: newVerdictRing(20),
		dedupe:   map[string]time.Time{},
		pending:  map[pendingKey]*http.Request{},
	}
	emptyBlocks := blockSnapshot{}
	e.block.Store(&emptyBlocks)
	e.cfg.Store(&cfg)
	e.enabled.Store(cfg.Enabled)
	return e
}

func TestAIOnlyOnMatchBuildsJobLazilyFromLiveRequest(t *testing.T) {
	cfg := defaultAIConfig()
	cfg.Enabled = true
	cfg.OnlyOnMatch = true
	cfg.SampleRate = 0
	e := testAIEngineWithoutWorkers(cfg)

	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		e.enqueueMatch("site-a", "advisory", clientIP(r), r.URL.RequestURI(), 942100, "matched")
	})
	h := e.wrap(SiteConfig{Name: "site-a", AIMode: "advisory"}, next)
	r := httptest.NewRequest(http.MethodGet, "http://bench.local/login?token=secret&ok=1", nil)
	r.RemoteAddr = "192.0.2.10:1234"
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Session-Token", "do-not-send")
	h.ServeHTTP(httptest.NewRecorder(), r)

	select {
	case job := <-e.queue:
		if job.method != http.MethodGet || job.path != "/login" {
			t.Fatalf("lazy job request context = %#v", job)
		}
		if strings.Contains(job.query, "secret") || !strings.Contains(job.query, "ok=1") {
			t.Fatalf("lazy job query redaction = %q", job.query)
		}
		if job.headers["Content-Type"] != "application/json" {
			t.Fatalf("safe header missing: %#v", job.headers)
		}
		if _, ok := job.headers["X-Session-Token"]; ok {
			t.Fatalf("sensitive header leaked: %#v", job.headers)
		}
		if len(job.rules) != 1 || job.rules[0] != 942100 {
			t.Fatalf("match rule missing: %#v", job.rules)
		}
	default:
		t.Fatal("match did not enqueue a lazy AI job")
	}

	e.pendingMu.RLock()
	pending := len(e.pending)
	e.pendingMu.RUnlock()
	if pending != 0 {
		t.Fatalf("pending requests leaked after handler: %d", pending)
	}
}

func TestAIUnsampledCleanRequestDoesNotBuildOrEnqueueJob(t *testing.T) {
	cfg := defaultAIConfig()
	cfg.Enabled = true
	cfg.OnlyOnMatch = false
	cfg.SampleRate = 0
	e := testAIEngineWithoutWorkers(cfg)

	h := e.wrap(SiteConfig{Name: "site-a", AIMode: "advisory"}, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	r := httptest.NewRequest(http.MethodGet, "http://bench.local/clean?token=secret", nil)
	r.RemoteAddr = "192.0.2.20:1234"
	h.ServeHTTP(httptest.NewRecorder(), r)
	if got := len(e.queue); got != 0 {
		t.Fatalf("unsampled clean request queued %d jobs", got)
	}
	e.pendingMu.RLock()
	pending := len(e.pending)
	e.pendingMu.RUnlock()
	if pending != 0 {
		t.Fatalf("pending requests leaked after clean handler: %d", pending)
	}
}

func TestAISampleRateHundredEnqueuesCleanRequest(t *testing.T) {
	cfg := defaultAIConfig()
	cfg.Enabled = true
	cfg.OnlyOnMatch = false
	cfg.SampleRate = 100
	e := testAIEngineWithoutWorkers(cfg)

	h := e.wrap(SiteConfig{Name: "site-a", AIMode: "advisory"}, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	r := httptest.NewRequest(http.MethodPost, "http://bench.local/api?ok=1", nil)
	r.RemoteAddr = "192.0.2.30:1234"
	h.ServeHTTP(httptest.NewRecorder(), r)
	if got := len(e.queue); got != 1 {
		t.Fatalf("100%% sampled clean request queued %d jobs, want 1", got)
	}
	job := <-e.queue
	if job.method != http.MethodPost || job.path != "/api" {
		t.Fatalf("sampled job context = %#v", job)
	}
}

func TestAIQueueDropWarningIsRateLimited(t *testing.T) {
	cfg := defaultAIConfig()
	cfg.Enabled = true
	cfg.QueueSize = 1
	var buf strings.Builder
	e := testAIEngineWithoutWorkers(cfg)
	e.queue = make(chan analysisJob, 1)
	e.log = slog.New(slog.NewTextHandler(&buf, nil))

	e.enqueue(analysisJob{site: "site-a", mode: "advisory", client: "1", method: "GET", path: "/1"})
	e.enqueue(analysisJob{site: "site-a", mode: "advisory", client: "2", method: "GET", path: "/2"})
	e.enqueue(analysisJob{site: "site-a", mode: "advisory", client: "3", method: "GET", path: "/3"})
	stats := e.queueStats()
	if stats.Enqueued != 1 || stats.Dropped != 2 || stats.Depth != 1 || stats.LogicalLimit != 1 {
		t.Fatalf("queue stats = %#v", stats)
	}
	if got := strings.Count(buf.String(), "ai analysis queue saturated"); got != 1 {
		t.Fatalf("queue saturation warnings = %d, want 1; log=%q", got, buf.String())
	}
}

type benchmarkResponseWriter struct{ header http.Header }

func (w benchmarkResponseWriter) Header() http.Header       { return w.header }
func (benchmarkResponseWriter) WriteHeader(int)             {}
func (benchmarkResponseWriter) Write(p []byte) (int, error) { return len(p), nil }

func benchmarkAIRequest() *http.Request {
	r := httptest.NewRequest(http.MethodPost, "http://bench.local/api?token=secret&ok=1", nil)
	r.RemoteAddr = "192.0.2.40:1234"
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("User-Agent", "wafbench")
	r.Header.Set("X-Session-Token", "secret")
	return r
}

// legacyAIUnsampledNoMatch mirrors the pre-lazy request bookkeeping so the
// benchmark can quantify the work P2 intentionally removed. It is test-only.
func legacyAIUnsampledNoMatch(e *aiEngine, legacy map[string]*analysisJob, siteName, mode, ip string, r *http.Request, cfg AIConfig) {
	hdrs := e.redactHeaders(r, cfg)
	var body []byte
	if cfg.IncludeBody {
		body = requestBodyPrefixFromRequest(r)
	}
	query := r.URL.RawQuery
	if cfg.RedactHeaders {
		query = redactQuery(query)
	}
	pending := &analysisJob{site: siteName, mode: mode, client: ip, method: r.Method,
		host: r.Host, path: r.URL.Path, query: query, headers: hdrs, body: body, ts: time.Now()}
	legacyRequestKey := func(client, uri string) string { return client + "\x00" + uri }
	keys := []string{legacyRequestKey(ip, r.URL.RequestURI()), legacyRequestKey(ip, r.URL.Path)}
	e.pendingMu.Lock()
	// A separate persistent map models the old pointer-to-job registration
	// without changing the production pending-map type.
	for _, key := range keys {
		legacy[key] = pending
	}
	e.pendingMu.Unlock()
	for _, key := range keys {
		delete(legacy, key)
	}
}

func BenchmarkAIUnsampledNoMatchLazy(b *testing.B) {
	cfg := defaultAIConfig()
	cfg.Enabled = true
	cfg.OnlyOnMatch = false
	cfg.SampleRate = 0
	e := testAIEngineWithoutWorkers(cfg)
	r := benchmarkAIRequest()
	w := benchmarkResponseWriter{header: make(http.Header)}
	h := e.wrap(SiteConfig{Name: "site-a", AIMode: "advisory"}, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.ServeHTTP(w, r)
	}
}

func BenchmarkAIUnsampledNoMatchLegacyEager(b *testing.B) {
	cfg := defaultAIConfig()
	cfg.Enabled = true
	cfg.OnlyOnMatch = false
	cfg.SampleRate = 0
	e := testAIEngineWithoutWorkers(cfg)
	r := benchmarkAIRequest()
	ip := clientIP(r)
	legacy := map[string]*analysisJob{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		legacyAIUnsampledNoMatch(e, legacy, "site-a", "advisory", ip, r, cfg)
	}
}

func TestAIExpiredBlockIsIgnoredAndPrunedOffPath(t *testing.T) {
	e := testAIEngine()
	e.addBlock(blockEntry{IP: "203.0.113.9", Site: "shop", Expires: time.Now().Add(-time.Second)})
	if _, ok := e.isBlocked("shop", "203.0.113.9"); ok {
		t.Fatal("expired block remained active")
	}
	if got := len(e.loadBlocks()); got != 1 {
		t.Fatalf("request-path lookup mutated snapshot; entries=%d want 1 before janitor prune", got)
	}
	e.pruneExpiredBlocks()
	if got := len(e.loadBlocks()); got != 0 {
		t.Fatalf("expired block not pruned off path; entries=%d", got)
	}
}

func BenchmarkAIBlockLookupAtomic(b *testing.B) {
	cfg := defaultAIConfig()
	e := testAIEngineWithoutWorkers(cfg)
	e.addBlock(blockEntry{IP: "203.0.113.7", Site: "shop", Expires: time.Now().Add(time.Hour)})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.isBlocked("shop", "203.0.113.7")
	}
}

func BenchmarkAIBlockLookupAtomicParallel(b *testing.B) {
	cfg := defaultAIConfig()
	e := testAIEngineWithoutWorkers(cfg)
	e.addBlock(blockEntry{IP: "203.0.113.7", Site: "shop", Expires: time.Now().Add(time.Hour)})
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = e.isBlocked("shop", "203.0.113.7")
		}
	})
}

func BenchmarkAIBlockLookupLegacyMutexParallel(b *testing.B) {
	var mu sync.Mutex
	blocks := map[string]blockEntry{
		blockKey("shop", "203.0.113.7"): {IP: "203.0.113.7", Site: "shop", Expires: time.Now().Add(time.Hour)},
	}
	key := blockKey("shop", "203.0.113.7")
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			be, ok := blocks[key]
			if ok && time.Now().After(be.Expires) {
				ok = false
			}
			mu.Unlock()
			_ = ok
		}
	})
}
