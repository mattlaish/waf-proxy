package main

func VectorScanDifferential(candidates []int, coraza []int) (falseNeg []int) {
	have := map[int]bool{}
	for _, id := range candidates {
		have[id] = true
	}
	for _, id := range coraza {
		if !have[id] {
			falseNeg = append(falseNeg, id)
		}
	}
	return
}
func QualificationPassed(candidates []int, coraza []int) bool {
	return len(VectorScanDifferential(candidates, coraza)) == 0
}
