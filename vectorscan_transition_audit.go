package main

import "time"

type VectorScanTransitionAudit struct {
	Component   string    `json:"component"`
	From        string    `json:"from"`
	To          string    `json:"to"`
	Reason      string    `json:"reason"`
	EvidenceRef string    `json:"evidence_ref,omitempty"`
	At          time.Time `json:"timestamp"`
}

func NewVectorScanTransitionAudit(from, to, reason, evidence string) VectorScanTransitionAudit {
	return VectorScanTransitionAudit{Component: "vectorscan", From: from, To: to, Reason: reason, EvidenceRef: evidence, At: time.Now().UTC()}
}
