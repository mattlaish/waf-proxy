package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func openAIResponsesBody(text string) string {
	b, _ := json.Marshal(map[string]any{
		"status": "completed",
		"output": []any{map[string]any{
			"type":    "message",
			"content": []any{map[string]any{"type": "output_text", "text": text}},
		}},
	})
	return string(b)
}

func testAIEngineWithoutWorkers(cfg AIConfig) *aiEngine {
	e := &aiEngine{}
	e.configure(cfg)
	return e
}

func TestOpenAIResponsesStructuredVerdictContract(t *testing.T) {
	t.Setenv("OPENAI_TEST_KEY", "test-openai-key")
	var seen bool
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = true
		if r.Method != http.MethodPost || r.URL.Path != "/v1/responses" {
			t.Errorf("request = %s %s, want POST /v1/responses", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-openai-key" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body["model"] != "gpt-test" || body["store"] != false || body["max_output_tokens"] != float64(300) {
			t.Errorf("unexpected Responses request envelope: %#v", body)
		}
		text, ok := body["text"].(map[string]any)
		if !ok {
			t.Fatalf("text missing: %#v", body["text"])
		}
		format, ok := text["format"].(map[string]any)
		if !ok || format["type"] != "json_schema" || format["name"] != "waf_verdict" || format["strict"] != true {
			t.Fatalf("structured output format = %#v", text["format"])
		}
		schema, ok := format["schema"].(map[string]any)
		if !ok || schema["type"] != "object" || schema["additionalProperties"] != false {
			t.Fatalf("schema = %#v", format["schema"])
		}
		required, ok := schema["required"].([]any)
		if !ok || len(required) != 4 {
			t.Fatalf("required = %#v", schema["required"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(openAIResponsesBody(`{"verdict":"malicious","score":96,"category":"sqli","reason":"SQL injection payload"}`)))
	}))
	defer srv.Close()

	cfg := defaultAIConfig()
	cfg.Enabled = true
	cfg.APIStyle = "responses"
	cfg.BaseURL = srv.URL + "/v1"
	cfg.APIKeyRef = "env:OPENAI_TEST_KEY"
	cfg.Model = "gpt-test"
	e := testAIEngineWithoutWorkers(cfg)
	e.client.Store(srv.Client())
	got, err := e.callLLM(context.Background(), cfg, "<request>test</request>")
	if err != nil {
		t.Fatalf("callLLM: %v", err)
	}
	if !seen || got.Verdict != "malicious" || got.Score != 96 || got.Category != "sqli" {
		t.Fatalf("verdict = %#v, seen=%v", got, seen)
	}
}

func TestOpenAIResponsesProfileReviewUsesStructuredOutput(t *testing.T) {
	t.Setenv("OPENAI_TEST_KEY", "test-openai-key")
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		text := body["text"].(map[string]any)
		format := text["format"].(map[string]any)
		if format["name"] != "waf_profile_review" || format["strict"] != true {
			t.Errorf("profile review format = %#v", format)
		}
		_, _ = w.Write([]byte(openAIResponsesBody(`{"agree":true,"confidence":88,"reason":"Signals fit the proposed profile"}`)))
	}))
	defer srv.Close()

	cfg := defaultAIConfig()
	cfg.Enabled = true
	cfg.APIStyle = "responses"
	cfg.BaseURL = srv.URL + "/v1"
	cfg.APIKeyRef = "env:OPENAI_TEST_KEY"
	e := testAIEngineWithoutWorkers(cfg)
	e.client.Store(srv.Client())

	agree, confidence, reason, err := e.reviewProfile(context.Background(), "/login", "password form", "login")
	if err != nil || !agree || confidence != 88 || reason == "" {
		t.Fatalf("profile review = agree=%v confidence=%d reason=%q err=%v", agree, confidence, reason, err)
	}
}

func TestOpenAIResponsesRefusalIncompleteAndMalformedFailOpen(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{"refusal", `{"status":"completed","output":[{"type":"message","content":[{"type":"refusal","refusal":"cannot comply"}]}]}`, "refusal"},
		{"incomplete", `{"status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},"output":[]}`, "incomplete"},
		{"failed-status", `{"status":"failed","output":[]}`, "status"},
		{"missing-output-text", `{"status":"completed","output":[{"type":"message","content":[]}]}`, "no output_text"},
		{"malformed-json", `{"status":`, "unexpected"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("OPENAI_TEST_KEY", "test-openai-key")
			srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(tc.body)) }))
			defer srv.Close()
			cfg := defaultAIConfig()
			cfg.Enabled = true
			cfg.APIStyle = "responses"
			cfg.BaseURL = srv.URL
			cfg.APIKeyRef = "env:OPENAI_TEST_KEY"
			e := testAIEngineWithoutWorkers(cfg)
			e.client.Store(srv.Client())
			if _, err := e.callLLM(context.Background(), cfg, "test"); err == nil || !strings.Contains(strings.ToLower(err.Error()), tc.want) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestOpenAIResponsesHTTPFailuresDoNotReflectProviderBody(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusTooManyRequests, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			t.Setenv("OPENAI_TEST_KEY", "test-openai-key")
			srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"error":{"message":"echo-sensitive-data"}}`))
			}))
			defer srv.Close()
			cfg := defaultAIConfig()
			cfg.Enabled = true
			cfg.APIStyle = "responses"
			cfg.BaseURL = srv.URL
			cfg.APIKeyRef = "env:OPENAI_TEST_KEY"
			e := testAIEngineWithoutWorkers(cfg)
			e.client.Store(srv.Client())
			_, err := e.callLLM(context.Background(), cfg, "test")
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), fmt.Sprintf("http %d", status)) || strings.Contains(err.Error(), "sensitive") {
				t.Fatalf("status %d error = %v", status, err)
			}
		})
	}
}

func TestOpenAIResponsesTimeoutIsAnError(t *testing.T) {
	t.Setenv("OPENAI_TEST_KEY", "test-openai-key")
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(80 * time.Millisecond)
		_, _ = w.Write([]byte(openAIResponsesBody(`{"verdict":"benign","score":1,"category":"normal","reason":"ok"}`)))
	}))
	defer srv.Close()
	cfg := defaultAIConfig()
	cfg.Enabled = true
	cfg.APIStyle = "responses"
	cfg.BaseURL = srv.URL
	cfg.APIKeyRef = "env:OPENAI_TEST_KEY"
	e := testAIEngineWithoutWorkers(cfg)
	e.client.Store(&http.Client{Transport: srv.Client().Transport, Timeout: 10 * time.Millisecond})
	if _, err := e.callLLM(context.Background(), cfg, "test"); err == nil {
		t.Fatal("timeout unexpectedly produced a verdict")
	}
}

func TestOpenAIChatCompletionsCompatibilityPath(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"verdict\":\"benign\",\"score\":2,\"category\":\"normal\",\"reason\":\"ordinary\"}"}}]}`))
	}))
	defer srv.Close()
	cfg := defaultAIConfig()
	cfg.Enabled = true
	cfg.APIStyle = "chat_completions"
	cfg.BaseURL = srv.URL + "/v1"
	if err := cfg.validate(); err != nil {
		t.Fatalf("compatibility config rejected: %v", err)
	}
	e := testAIEngineWithoutWorkers(cfg)
	e.client.Store(srv.Client())
	v, err := e.callLLM(context.Background(), cfg, "test")
	if err != nil || v.Verdict != "benign" || path != "/v1/chat/completions" {
		t.Fatalf("compatibility result=%#v err=%v path=%q", v, err, path)
	}
}

func TestAISecretReferenceFileAndValidation(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "openai.key")
	if err := os.WriteFile(p, []byte("file-key\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := defaultAIConfig()
	cfg.Enabled = true
	cfg.APIStyle = "responses"
	cfg.BaseURL = "https://api.openai.com/v1"
	cfg.APIKeyRef = "file:" + p
	if err := cfg.validate(); err != nil {
		t.Fatalf("valid file secret ref rejected: %v", err)
	}
	key, err := cfg.resolveAPIKey()
	if err != nil || string(key) != "file-key" {
		t.Fatalf("resolve API key = %q, %v", key, err)
	}
	cfg.APIKey = "legacy-inline"
	if err := cfg.validate(); err == nil {
		t.Fatal("inline api_key plus api_key_ref accepted")
	}
}

func TestOpenAIResponsesValidationRequiresHTTPSAndSecretReference(t *testing.T) {
	cfg := defaultAIConfig()
	cfg.Enabled = true
	cfg.APIStyle = "responses"
	cfg.APIKeyRef = "env:OPENAI_API_KEY"
	cfg.BaseURL = "http://api.openai.com/v1"
	if err := cfg.validate(); err == nil || !strings.Contains(err.Error(), "https") {
		t.Fatalf("http Responses endpoint accepted: %v", err)
	}
	cfg.BaseURL = "https://api.openai.com/v1"
	cfg.APIKeyRef = ""
	if err := cfg.validate(); err == nil || !strings.Contains(err.Error(), "api_key_ref") {
		t.Fatalf("missing Responses secret reference accepted: %v", err)
	}
}

func TestAISecretReferenceRedactionAndPreservation(t *testing.T) {
	cur := Config{AI: defaultAIConfig()}
	cur.AI.APIKeyRef = "file:/etc/waf/secrets/openai.key"
	next := Config{AI: defaultAIConfig()}
	next.AI.APIKeyRef = ""
	preserveAISecrets(cur, &next)
	if next.AI.APIKeyRef != cur.AI.APIKeyRef {
		t.Fatalf("secret reference not preserved: %q", next.AI.APIKeyRef)
	}
	redacted := redactAISecrets(next)
	if redacted.AI.APIKeyRef != "" || redacted.AI.APIKey != "" {
		t.Fatalf("redacted config leaked AI secret material: %#v", redacted.AI)
	}
	if next.AI.APIKeyRef == "" {
		t.Fatal("redaction mutated source config")
	}

	legacy := Config{AI: defaultAIConfig()}
	legacy.AI.APIKey = "legacy-inline"
	replacement := Config{AI: defaultAIConfig()}
	replacement.AI.APIKeyRef = "env:OPENAI_API_KEY"
	preserveAISecrets(legacy, &replacement)
	if replacement.AI.APIKey != "" || replacement.AI.APIKeyRef != "env:OPENAI_API_KEY" {
		t.Fatalf("new secret reference did not replace legacy key: %#v", replacement.AI)
	}

	legacyView := redactAISecrets(legacy)
	if legacyView.AI.APIStyle != "chat_completions" {
		t.Fatalf("legacy redacted view changed wire contract: api_style=%q", legacyView.AI.APIStyle)
	}
}

func TestLegacyInlineOpenAIConfigRetainsChatCompletionsWireContract(t *testing.T) {
	cfg := defaultAIConfig()
	cfg.Enabled = true
	cfg.APIStyle = ""
	cfg.BaseURL = "https://api.openai.com/v1"
	cfg.APIKey = "legacy-inline-key"
	if got := cfg.effectiveOpenAIAPIStyle(); got != "chat_completions" {
		t.Fatalf("legacy config style = %q, want chat_completions", got)
	}
	if err := cfg.validate(); err != nil {
		t.Fatalf("legacy config rejected during migration: %v", err)
	}

	modern := defaultAIConfig()
	modern.APIStyle = ""
	modern.BaseURL = "https://api.openai.com/v1"
	modern.APIKeyRef = "env:OPENAI_API_KEY"
	if got := modern.effectiveOpenAIAPIStyle(); got != "responses" {
		t.Fatalf("modern official OpenAI config style = %q, want responses", got)
	}
}
