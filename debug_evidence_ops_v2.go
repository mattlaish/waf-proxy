package main

import (
	"sort"
	"time"
)

// DebugEvidenceOpsV2 provides bounded operator-facing operations.
// Storage backends can implement persistence without changing WAF dataplane.
type DebugEvidenceOpsV2 struct {
	Store *DebugEvidenceStore
}

type DebugEvidenceSummary struct {
	ID        string    `json:"id"`
	Tenant    string    `json:"tenant"`
	CreatedAt time.Time `json:"created_at"`
	Size      int64     `json:"size"`
}

func (o DebugEvidenceOpsV2) List(tenant string) []DebugEvidenceSummary {
	if o.Store == nil {
		return nil
	}
	out := []DebugEvidenceSummary{}
	for _, e := range o.Store.List(tenant, 500) {
		out = append(out, DebugEvidenceSummary{
			ID:        e.TransactionID,
			Tenant:    e.Tenant,
			CreatedAt: e.CapturedAt,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}
