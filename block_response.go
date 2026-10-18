package main

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strings"
	"time"
)

type blockResponse struct {
	Error     string `json:"error"`
	Reason    string `json:"reason,omitempty"`
	RequestID string `json:"request_id,omitempty"`
	Timestamp string `json:"timestamp"`
}

var blockPageTemplate = template.Must(template.New("block-page").Parse(`<!doctype html><html><head><meta charset="utf-8"><title>Request blocked</title></head><body><h1>Request blocked</h1><p>{{.Reason}}</p><p>Request ID: {{.RequestID}}</p></body></html>`))

func blockPageReason(reason string) string {
	if strings.HasPrefix(reason, "cidr_policy_denied:") {
		return "Access from your IP network is not allowed by this site's access rules. Contact the site administrator if you believe this is a mistake."
	}
	if reason == "l7_abuse_limit" {
		return "Too many requests or simultaneous connections. Please wait and try again."
	}
	return reason
}

// acceptsHTML reports whether the client explicitly accepts an HTML page,
// which browsers do on navigation. API clients that send no Accept header or
// only application/json are not treated as browsers.
func acceptsHTML(r *http.Request) bool {
	return r != nil && strings.Contains(r.Header.Get("Accept"), "text/html")
}

func writeBlockResponse(w http.ResponseWriter, r *http.Request, status int, reason string, jsonMode bool) {
	id := requestCorrelationID(r)
	if id != "" {
		w.Header().Set("X-Request-ID", id)
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if jsonMode || r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(blockResponse{Error: "request_blocked", Reason: reason, RequestID: id, Timestamp: time.Now().UTC().Format(time.RFC3339)})
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; base-uri 'none'; frame-ancestors 'none'")
	w.Header().Set("X-Frame-Options", "DENY")
	w.WriteHeader(status)
	_ = blockPageTemplate.Execute(w, blockResponse{Reason: blockPageReason(reason), RequestID: id})
}
