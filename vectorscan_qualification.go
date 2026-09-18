package main

import "time"

// VectorScanQualificationRecord is an evidence model for Phase 5 Slice B.
// It records qualification state without making analyzer-only results runtime PASS.
type VectorScanQualificationRecord struct {
	Corpus string    `json:"corpus"`
	RuleID string    `json:"rule_id,omitempty"`
	Status string    `json:"status"`
	FalseNegatives []int `json:"false_negatives,omitempty"`
	Evidence string  `json:"evidence,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func NewVectorScanQualificationRecord(corpus, status string) VectorScanQualificationRecord {
	return VectorScanQualificationRecord{Corpus: corpus, Status: status, CreatedAt: time.Now().UTC()}
}

// VectorScanQualificationGate keeps Coraza authoritative.
// Any false negative blocks acceleration eligibility.
func VectorScanQualificationGate(corazaRules, vectorRules []int) VectorScanQualificationRecord {
	missed := VectorScanDifferential(vectorRules, corazaRules)
	if len(missed) != 0 {
		return VectorScanQualificationRecord{
			Status: "FAILSAFE_CORAZA_ONLY",
			FalseNegatives: missed,
			CreatedAt: time.Now().UTC(),
		}
	}
	return VectorScanQualificationRecord{Status: "VALIDATED", CreatedAt: time.Now().UTC()}
}
