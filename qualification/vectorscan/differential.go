package vectorscan

// DifferentialResult records the comparison between authoritative Coraza
// matches and optional VectorScan candidate matches.
type DifferentialResult struct {
	RuleID            int    `json:"rule_id"`
	CorazaMatched     bool   `json:"coraza_matched"`
	VectorScanMatched bool   `json:"vectorscan_matched"`
	Result            string `json:"result"`
}

func Compare(ruleID int, corazaMatched bool, vectorScanMatched bool) DifferentialResult {
	result := "MATCH"
	if corazaMatched && !vectorScanMatched {
		result = "FALSE_NEGATIVE"
	} else if !corazaMatched && vectorScanMatched {
		result = "EXTRA_CANDIDATE"
	}
	return DifferentialResult{RuleID: ruleID, CorazaMatched: corazaMatched, VectorScanMatched: vectorScanMatched, Result: result}
}

func ZeroFalseNegative(results []DifferentialResult) bool {
	for _, r := range results {
		if r.Result == "FALSE_NEGATIVE" {
			return false
		}
	}
	return true
}
