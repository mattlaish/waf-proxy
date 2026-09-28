package vectoraccel

import "testing"

func TestEligibleMatchedRuleIDsFiltersAndDeduplicates(t *testing.T) {
	p := &SitePlan{byRule: map[int]*groupRuntime{100: {}, 200: {}}}
	got := p.EligibleMatchedRuleIDs([]int{300, 200, 100, 200})
	if len(got) != 2 || got[0] != 100 || got[1] != 200 {
		t.Fatalf("unexpected eligible IDs: %#v", got)
	}
}
