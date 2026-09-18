package vectorscan

// ReplayCase describes a corpus item used by VectorScan production qualification.
// The runner intentionally records evidence only; Coraza remains authoritative.
type ReplayCase struct {
	ID      string `json:"id"`
	Request string `json:"request"`
}

// ReplayResult records a single differential qualification result.
type ReplayResult struct {
	CaseID           string `json:"case_id"`
	CorazaMatches    []int  `json:"coraza_matches,omitempty"`
	VectorCandidates []int  `json:"vector_candidates,omitempty"`
	FalseNegatives   []int  `json:"false_negatives,omitempty"`
	Status           string `json:"status"` // PASS, FAIL, NOT_RUN, BLOCKED
}
