package qualification

import "time"

// Sample describes one replay qualification corpus item.
// ExpectedRules are authoritative expected Coraza rule IDs.
type Sample struct {
	ID string `json:"id"`
	Category string `json:"category"`
	ExpectedRules []string `json:"expected_rules"`
	PayloadFile string `json:"payload_file"`
}

// DifferentialResult is the stable qualification evidence record.
type DifferentialResult struct {
	SampleID string `json:"sample_id"`
	CorazaRules []string `json:"coraza_rules"`
	VectorScanRules []string `json:"vectorscan_rules"`
	FalseNegatives []string `json:"false_negatives"`
	CheckedAt time.Time `json:"checked_at"`
}

func Compare(sample Sample, coraza, vectorscan []string) DifferentialResult {
	vm := map[string]bool{}
	for _, r := range vectorscan { vm[r]=true }
	var missing []string
	for _, r := range coraza {
		if !vm[r] { missing = append(missing,r) }
	}
	return DifferentialResult{
		SampleID: sample.ID,
		CorazaRules: append([]string(nil), coraza...),
		VectorScanRules: append([]string(nil), vectorscan...),
		FalseNegatives: missing,
		CheckedAt: time.Now().UTC(),
	}
}

func ZeroFalseNegative(results []DifferentialResult) bool {
	for _, r := range results {
		if len(r.FalseNegatives) != 0 { return false }
	}
	return true
}
