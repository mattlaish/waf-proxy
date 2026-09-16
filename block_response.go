package main

import (
	"encoding/json"
	"net/http"
	"time"
)

type blockResponse struct {
	Error     string `json:"error"`
	Reason    string `json:"reason,omitempty"`
	RequestID string `json:"request_id,omitempty"`
	Timestamp string `json:"timestamp"`
}

func writeBlockResponse(w http.ResponseWriter, r *http.Request, status int, reason string, jsonMode bool) {
	id := requestCorrelationID(r)
	if id != "" {
		w.Header().Set("X-Request-ID", id)
	}
	if jsonMode || r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(blockResponse{Error: "request_blocked", Reason: reason, RequestID: id, Timestamp: time.Now().UTC().Format(time.RFC3339)})
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte("<html><body><h1>Request blocked</h1><p>" + reason + "</p><p>Request ID: " + id + "</p></body></html>"))
}
