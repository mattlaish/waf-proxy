package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

func (a *adminServer) handleAPIOperationDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	op, ok := a.srv.apiOps.get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, op)
}

func (a *adminServer) handleAPIOperationIgnore(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Ignored *bool `json:"ignored"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	ignored := true
	if req.Ignored != nil {
		ignored = *req.Ignored
	}
	op, err := a.srv.apiOps.ignore(id, ignored)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := a.srv.apiOps.save(a.srv.configPath); err != nil {
		http.Error(w, "operation persistence failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, op)
}

func (a *adminServer) handleAPIOperationReclassify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID             string `json:"id"`
		NormalizedPath string `json:"normalized_path"`
		Classification string `json:"classification"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	op, err := a.srv.apiOps.reclassify(strings.TrimSpace(req.ID), req.NormalizedPath, req.Classification)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	if err := a.srv.apiOps.save(a.srv.configPath); err != nil {
		http.Error(w, "operation persistence failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, op)
}

func (a *adminServer) handleSchemaReview(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Decision string `json:"decision"` // approve | return_to_candidate
		Reason   string `json:"reason"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	status := "REVIEWED"
	recommendation := "approved"
	if strings.EqualFold(req.Decision, "return_to_candidate") {
		status = "CANDIDATE"
		recommendation = "returned_to_candidate"
	} else if req.Decision != "" && !strings.EqualFold(req.Decision, "approve") {
		http.Error(w, "decision must be approve or return_to_candidate", http.StatusBadRequest)
		return
	}
	current, ok := a.srv.schema.get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if status == "REVIEWED" && current.Status != "CANDIDATE" && current.Status != "REVIEWED" {
		http.Error(w, "schema candidate is still learning", http.StatusConflict)
		return
	}
	if status == "CANDIDATE" && current.Status == "LEARNING" {
		http.Error(w, "learning candidate cannot be promoted manually", http.StatusConflict)
		return
	}
	review := SchemaReviewRecommendation{Recommendation: recommendation, Confidence: 100, Reason: strings.TrimSpace(req.Reason), ReviewedAt: time.Now().UTC(), Source: "operator"}
	candidate, ok := a.srv.schema.setReview(id, status, review)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := a.srv.schema.save(a.srv.configPath); err != nil {
		http.Error(w, "schema review persistence failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, candidate)
}

func (a *adminServer) handleSchemaAIReview(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	candidate, ok := a.srv.schema.get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if a.srv.ai == nil || !a.srv.ai.snapshotCfg().Enabled {
		http.Error(w, "AI connector is not enabled", http.StatusPreconditionFailed)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	review, err := a.srv.ai.reviewSchemaCandidate(ctx, candidate)
	if err != nil {
		http.Error(w, "schema AI review failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	// AI is advisory only: preserve LEARNING/CANDIDATE/REVIEWED lifecycle status.
	updated, ok := a.srv.schema.setReview(id, "", review)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := a.srv.schema.save(a.srv.configPath); err != nil {
		http.Error(w, "schema AI review persistence failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"candidate": updated, "review": review, "enforcement_changed": false})
}
