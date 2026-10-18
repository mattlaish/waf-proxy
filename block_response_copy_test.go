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
