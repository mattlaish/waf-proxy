package main

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type SchemaFieldObservation struct {
	Path       string           `json:"path"`
	TypeCounts map[string]int64 `json:"type_counts"`
	Samples    int64            `json:"samples"`
	Sensitive  bool             `json:"sensitive"`
}

type SchemaCandidate struct {
	ID          string                   `json:"id"`
	OperationID string                   `json:"operation_id"`
	Status      string                   `json:"status"`
	Confidence  float64                  `json:"confidence"`
	Fields      []SchemaFieldObservation `json:"fields"`
	CreatedAt   time.Time                `json:"created_at"`
}

type schemaStore struct {
	mu         sync.RWMutex
	candidates map[string]*SchemaCandidate
}

func newSchemaStore() *schemaStore {
	return &schemaStore{candidates: map[string]*SchemaCandidate{}}
}

func (s *schemaStore) list(filter string) []*SchemaCandidate {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*SchemaCandidate, 0, len(s.candidates))
	for _, c := range s.candidates {
		if filter == "" || strings.Contains(c.OperationID, filter) {
			cp := *c
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Confidence > out[j].Confidence })
	return out
}

func (a *adminServer) handleSchemaCandidates(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("q")
	writeJSON(w, http.StatusOK, a.srv.schema.list(filter))
}

func (a *adminServer) handleSchemaDetail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/security/schema/")
	a.srv.schema.mu.RLock()
	defer a.srv.schema.mu.RUnlock()
	c, ok := a.srv.schema.candidates[id]
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func writeSchemaJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
