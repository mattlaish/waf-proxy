package main

import (
	"bufio"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestAccessRingCircularOrderingAndJSONTime(t *testing.T) {
	r := newAccessRing(3)
	base := time.Date(2026, 9, 3, 14, 15, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		r.add(accessRec{At: base.Add(time.Duration(i) * time.Second), Path: string(rune('a' + i))})
	}
	got := r.snapshot(0)
	if len(got) != 3 || got[0].Path != "e" || got[1].Path != "d" || got[2].Path != "c" {
		t.Fatalf("snapshot order = %#v", got)
	}
	b, err := json.Marshal(got[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"time":"14:15:04"`) {
		t.Fatalf("access JSON time contract changed: %s", b)
	}
}

func TestMatchRingCircularOrderingAndCount(t *testing.T) {
	r := newMatchRing(2)
	r.add(matchRec{RuleID: 1})
	r.add(matchRec{RuleID: 2})
	r.add(matchRec{RuleID: 3})
	if got := r.count(); got != 2 {
		t.Fatalf("count = %d, want 2", got)
	}
	got := r.snapshot(0)
	if len(got) != 2 || got[0].RuleID != 3 || got[1].RuleID != 2 {
		t.Fatalf("snapshot order = %#v", got)
	}
}

type hijackResponseWriter struct {
	header http.Header
	conn   net.Conn
}

func (w *hijackResponseWriter) Header() http.Header { return w.header }
func (w *hijackResponseWriter) WriteHeader(int)     {}
func (w *hijackResponseWriter) Write(p []byte) (int, error) {
	return len(p), nil
}
func (w *hijackResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.conn, bufio.NewReadWriter(bufio.NewReader(w.conn), bufio.NewWriter(w.conn)), nil
}

func TestStatusRecorderUnwrapPreservesHijacker(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	under := &hijackResponseWriter{header: make(http.Header), conn: server}
	rec := &statusRecorder{ResponseWriter: under, code: http.StatusOK}
	conn, _, err := http.NewResponseController(rec).Hijack()
	if err != nil {
		t.Fatalf("Hijack through statusRecorder: %v", err)
	}
	if conn != server {
		t.Fatal("Hijack did not reach the underlying ResponseWriter")
	}
}

type markerHandler struct{ calls int }

func (h *markerHandler) ServeHTTP(http.ResponseWriter, *http.Request) { h.calls++ }

func TestAIWrapOffReturnsUnderlyingHandler(t *testing.T) {
	e := testAIEngine()
	next := &markerHandler{}
	got := e.wrap(SiteConfig{Name: "site", AIMode: "off"}, next)
	if got != next {
		t.Fatal("AI-off site retained a middleware wrapper")
	}
}

func TestSyslogDisabledAccessGateDoesNotEnqueue(t *testing.T) {
	s := newSyslogEngine(nil)
	s.forwardAccess(accessRec{Site: "site", Status: http.StatusOK})
	if got := len(s.queue); got != 0 {
		t.Fatalf("disabled access syslog queued %d messages", got)
	}
	cfg := defaultSyslogConfig()
	cfg.Enabled = true
	cfg.SendAccess = true
	s.storeConfig(cfg)
	s.forwardAccess(accessRec{Site: "site", Status: http.StatusOK})
	if got := len(s.queue); got != 1 {
		t.Fatalf("enabled access syslog queued %d messages, want 1", got)
	}
}

func TestLoadConfigDefaultsPassiveDiscoveryOnForLegacyFiles(t *testing.T) {
	path := t.TempDir() + "/config.json"
	if err := os.WriteFile(path, []byte(`{"engine_mode":"DetectionOnly"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.PassiveDiscoveryEnabled {
		t.Fatal("legacy config without passive_discovery_enabled unexpectedly disabled discovery")
	}
}

func BenchmarkAccessRingAddFixed(b *testing.B) {
	r := newAccessRing(1000)
	rec := accessRec{At: time.Now(), Site: "site", Client: "192.0.2.1", Method: "GET", Path: "/", Status: 200}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.add(rec)
	}
}

func BenchmarkPassiveJSONLargeValue(b *testing.B) {
	body := []byte(`{"first":"` + strings.Repeat("x", 60000) + `","nested":{"n":1},"last":true}`)
	b.ReportAllocs()
	b.SetBytes(int64(len(body)))
	for i := 0; i < b.N; i++ {
		_ = passiveFields(http.MethodPost, "/api", "application/json", body)
	}
}
