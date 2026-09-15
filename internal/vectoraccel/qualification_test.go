package vectoraccel

import "testing"

func TestCompareCandidatesRequiresSuperset(t *testing.T) {
	got := CompareCandidates(map[int]struct{}{1: {}, 2: {}, 3: {}}, map[int]struct{}{1: {}, 3: {}})
	if !got.Passed() || got.FalseNegatives != 0 {
		t.Fatalf("expected superset to pass: %#v", got)
	}
	got = CompareCandidates(map[int]struct{}{1: {}}, map[int]struct{}{1: {}, 3: {}})
	if got.Passed() || got.FalseNegatives != 1 {
		t.Fatalf("expected missing candidate to fail: %#v", got)
	}
}

func TestEligibleMatchedRuleIDsFiltersAndDeduplicates(t *testing.T) {
	p := &SitePlan{byRule: map[int]*groupRuntime{100: {}, 200: {}}}
	got := p.EligibleMatchedRuleIDs([]int{300, 200, 100, 200})
	if len(got) != 2 || got[0] != 100 || got[1] != 200 {
		t.Fatalf("unexpected eligible IDs: %#v", got)
	}
}
