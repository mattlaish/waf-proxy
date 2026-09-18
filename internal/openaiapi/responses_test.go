package openaiapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func responseBody(text string) string {
	b, _ := json.Marshal(map[string]any{
		"status": "completed",
		"output": []any{map[string]any{
			"type":    "message",
			"content": []any{map[string]any{"type": "output_text", "text": text}},
		}},
	})
	return string(b)
}

func TestResponsesStructuredOutputWireContract(t *testing.T) {
	var seen bool
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = true
		if r.Method != http.MethodPost || r.URL.Path != "/v1/responses" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "gpt-test" || body["store"] != false || body["max_output_tokens"] != float64(300) {
			t.Fatalf("envelope = %#v", body)
		}
		text := body["text"].(map[string]any)
		format := text["format"].(map[string]any)
		if format["type"] != "json_schema" || format["name"] != "waf_verdict" || format["strict"] != true {
			t.Fatalf("format = %#v", format)
		}
		schema := format["schema"].(map[string]any)
		if schema["additionalProperties"] != false {
			t.Fatalf("schema = %#v", schema)
		}
		_, _ = w.Write([]byte(responseBody(`{"verdict":"malicious","score":96,"category":"sqli","reason":"SQL injection"}`)))
	}))
	defer srv.Close()

	schema := map[string]any{"type": "object", "properties": map[string]any{}, "required": []string{}, "additionalProperties": false}
	req := NewResponsesRequest("gpt-test", 300, "system", "user", "waf_verdict", schema)
	got, err := CallResponses(context.Background(), srv.Client(), srv.URL+"/v1", []byte("test-key"), req)
	if err != nil || !seen || !strings.Contains(got, `"malicious"`) {
		t.Fatalf("got=%q seen=%v err=%v", got, seen, err)
	}
}

func TestResponsesRefusalIncompleteAndBadStatus(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{"refusal", `{"status":"completed","output":[{"type":"message","content":[{"type":"refusal","refusal":"cannot comply"}]}]}`, "refusal"},
		{"incomplete", `{"status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},"output":[]}`, "incomplete"},
		{"failed", `{"status":"failed","output":[]}`, "status"},
		{"malformed", `{"status":`, "unexpected"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(tc.body)) }))
			defer srv.Close()
			_, err := CallResponses(context.Background(), srv.Client(), srv.URL, []byte("key"), NewResponsesRequest("gpt-test", 20, "s", "u", "", nil))
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), tc.want) {
				t.Fatalf("error=%v want=%q", err, tc.want)
			}
		})
	}
}

func TestResponsesHTTPFailuresAreStatusOnly(t *testing.T) {
	for _, status := range []int{401, 429, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"error":{"message":"echo-sensitive-data"}}`))
			}))
			defer srv.Close()
			_, err := CallResponses(context.Background(), srv.Client(), srv.URL, []byte("key"), NewResponsesRequest("gpt-test", 20, "s", "u", "", nil))
			if err == nil || !strings.Contains(err.Error(), fmt.Sprint(status)) || strings.Contains(err.Error(), "sensitive") {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestResponsesTimeoutAndSizeLimit(t *testing.T) {
	t.Run("timeout", func(t *testing.T) {
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(60 * time.Millisecond)
			_, _ = w.Write([]byte(responseBody(`{"ok":true}`)))
		}))
		defer srv.Close()
		client := &http.Client{Transport: srv.Client().Transport, Timeout: 5 * time.Millisecond}
		if _, err := CallResponses(context.Background(), client, srv.URL, []byte("key"), NewResponsesRequest("gpt-test", 20, "s", "u", "", nil)); err == nil {
			t.Fatal("timeout unexpectedly succeeded")
		}
	})

	t.Run("oversize", func(t *testing.T) {
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(strings.Repeat("x", maxResponseBytes+1)))
		}))
		defer srv.Close()
		if _, err := CallResponses(context.Background(), srv.Client(), srv.URL, []byte("key"), NewResponsesRequest("gpt-test", 20, "s", "u", "", nil)); err == nil || !strings.Contains(err.Error(), "size") {
			t.Fatalf("oversize error=%v", err)
		}
	})
}
