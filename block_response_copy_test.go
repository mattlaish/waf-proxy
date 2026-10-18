package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBlockPageProductLanguagePreservesAPIReason(t *testing.T) {
	for _, tc := range []struct {
		reason string
		text   string
	}{
		{"cidr_policy_denied:office", "Access from your IP network is not allowed"},
		{"l7_abuse_limit", "Please wait and try again"},
	} {
		t.Run(tc.reason, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "https://example.test/", nil)
			html := httptest.NewRecorder()
			writeBlockResponse(html, req, http.StatusForbidden, tc.reason, false)
			if !strings.Contains(html.Body.String(), tc.text) || strings.Contains(html.Body.String(), tc.reason) {
				t.Fatalf("unexpected visitor wording: %s", html.Body.String())
			}
			response := httptest.NewRecorder()
			writeBlockResponse(response, req, http.StatusForbidden, tc.reason, true)
			var body blockResponse
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Error != "request_blocked" || body.Reason != tc.reason {
				t.Fatalf("API reason changed: %+v", body)
			}
		})
	}
}

func TestRateLimitPageForBrowsersJSONForAPIClients(t *testing.T) {
	c := newL7AbuseController(L7AbuseConfig{Enabled: true, RequestsPerWindow: 1, WindowSeconds: 60})
	h := c.wrap("site-a", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	for _, tc := range []struct {
		name, accept, contentType, text string
	}{
		{"browser", "text/html,application/xhtml+xml,*/*;q=0.8", "text/html", "Please wait and try again"},
		{"api-no-accept", "", "application/json", `"reason":"l7_abuse_limit"`},
		{"api-json", "application/json", "application/json", `"reason":"l7_abuse_limit"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "https://example.test/", nil)
			req.RemoteAddr = "192.0.2.50:1234"
			if tc.accept != "" {
				req.Header.Set("Accept", tc.accept)
			}
			h.ServeHTTP(httptest.NewRecorder(), req) // consume the single token
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != http.StatusTooManyRequests {
				t.Fatalf("status=%d want 429", rr.Code)
			}
			if !strings.HasPrefix(rr.Header().Get("Content-Type"), tc.contentType) || !strings.Contains(rr.Body.String(), tc.text) {
				t.Fatalf("content-type=%q body=%s", rr.Header().Get("Content-Type"), rr.Body.String())
			}
		})
	}
}
