package main

import (
	"io"
	"log/slog"
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
