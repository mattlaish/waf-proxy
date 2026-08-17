package main

import (
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testAIEngine() *aiEngine {
	return newAIEngine(slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestAIBlocklistIsScopedBySite(t *testing.T) {
	e := testAIEngine()
	e.addBlock(blockEntry{IP: "203.0.113.7", Site: "shop", Expires: time.Now().Add(time.Minute)})
	if _, ok := e.isBlocked("shop", "203.0.113.7"); !ok { t.Fatal("shop block missing") }
	if _, ok := e.isBlocked("blog", "203.0.113.7"); ok { t.Fatal("block leaked to another site") }
	e.unblock("shop", "203.0.113.7")
	if _, ok := e.isBlocked("shop", "203.0.113.7"); ok { t.Fatal("site block was not removed") }
}

func TestAIClientIPRequiresTrustedPeer(t *testing.T) {
	r := httptest.NewRequest("GET", "http://example.test/", nil)
	r.RemoteAddr = "192.0.2.10:1234"
	r.Header.Set("X-Forwarded-For", "203.0.113.9")
	if got := aiClientIP(r, defaultAIConfig()); got != "192.0.2.10" { t.Fatalf("untrusted header used: %q", got) }
	cfg := defaultAIConfig()
	cfg.TrustedProxyCIDRs = []string{"192.0.2.0/24"}
	if got := aiClientIP(r, cfg); got != "203.0.113.9" { t.Fatalf("trusted forwarded IP ignored: %q", got) }
}

func TestAIRedaction(t *testing.T) {
	e := testAIEngine()
	r := httptest.NewRequest("GET", "http://example.test/", nil)
	r.Header.Set("X-Session-Token", "do-not-send")
	r.Header.Set("Content-Type", "application/json")
	h := e.redactHeaders(r, defaultAIConfig())
	if _, ok := h["X-Session-Token"]; ok { t.Fatal("sensitive header leaked") }
	if h["Content-Type"] != "application/json" { t.Fatal("safe header removed") }
	q := redactQuery("page=2&access_token=secret&password=pw")
	if strings.Contains(q, "secret") || strings.Contains(q, "pw") { t.Fatalf("query secret leaked: %s", q) }
}

func TestAIConfigValidation(t *testing.T) {
	cfg := defaultAIConfig()
	cfg.Enabled = true
	cfg.TrustedProxyCIDRs = []string{"not-a-cidr"}
	if cfg.validate() == nil { t.Fatal("invalid trusted proxy CIDR accepted") }
	cfg = defaultAIConfig(); cfg.Enabled = true; cfg.Workers = 0
	if cfg.validate() == nil { t.Fatal("invalid worker count accepted") }
}

func TestPromptIncludesMatchedRequestContext(t *testing.T) {
	e := testAIEngine()
	p := e.buildPrompt(defaultAIConfig(), analysisJob{method: "POST", path: "/login", query: "user=bob", rules: []int{942100}, matchedData: []string{"' OR 1=1"}})
	for _, want := range []string{"method: POST", "query: user=bob", "942100", "' OR 1=1"} {
		if !strings.Contains(p, want) { t.Fatalf("prompt missing %q: %s", want, p) }
	}
}

func TestParseVerdictRejectsIncompleteAndHandlesExtraBraces(t *testing.T) {
	if _, err := parseVerdict(`{"verdict":"malicious","category":"sqli","reason":"hit"}`); err == nil {
		t.Fatal("missing score accepted")
	}
	v, err := parseVerdict(`preface {not json} {"verdict":"benign","score":7,"category":"normal","reason":"ordinary request"} suffix`)
	if err != nil || v.Verdict != "benign" { t.Fatalf("valid embedded verdict not parsed: %#v %v", v, err) }
}

func TestDecodeJSONObjectSkipsInvalidBrace(t *testing.T) {
	var got struct { Agree bool `json:"agree"` }
	if err := decodeJSONObject("note {bad} {\"agree\":true}", &got); err != nil || !got.Agree {
		t.Fatalf("decodeJSONObject failed: %#v %v", got, err)
	}
}
