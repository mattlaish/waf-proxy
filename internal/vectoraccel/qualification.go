package vectoraccel

// QualificationResult is used by Phase 1 learning qualification. It records
// the safety property without changing the authoritative Coraza path.
type QualificationResult struct {
	Samples          int64
	CandidateMatches int64
	CorazaMatches    int64
	FalseNegatives   int64
}

func (r QualificationResult) Passed() bool {
	return r.FalseNegatives == 0 && r.CandidateMatches >= r.CorazaMatches
}

// CompareCandidates records the strict VectorScan safety gate:
// candidates must cover all transaction-final Coraza matches.
func CompareCandidates(candidates, coraza map[int]struct{}) QualificationResult {
	r := QualificationResult{}
	r.CandidateMatches = int64(len(candidates))
	r.CorazaMatches = int64(len(coraza))
	for id := range coraza {
		if _, ok := candidates[id]; !ok {
			r.FalseNegatives++
		}
	}
	r.Samples = 1
	return r
}
