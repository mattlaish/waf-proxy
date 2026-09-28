package main

import (
	"encoding/json"
	"html/template"
	"net/http"
	"time"
)

type blockResponse struct {
	Error     string `json:"error"`
	Reason    string `json:"reason,omitempty"`
	RequestID string `json:"request_id,omitempty"`
	Timestamp string `json:"timestamp"`
}

var blockPageTemplate = template.Must(template.New("block-page").Parse(`<!doctype html><html><head><meta charset="utf-8"><title>Request blocked</title></head><body><h1>Request blocked</h1><p>{{.Reason}}</p><p>Request ID: {{.RequestID}}</p></body></html>`))

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
	_ = blockPageTemplate.Execute(w, blockResponse{Reason: reason, RequestID: id})
}
