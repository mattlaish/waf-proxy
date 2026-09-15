package capability

import "testing"

func TestTruthBoundaryCases(t *testing.T) {
	base := Input{RuleID: "1001", Operator: "rx", Pattern: "attack", Phase: 1, Variables: []string{"REQUEST_URI"}, Transforms: []string{"none"}}
	cases := []struct {
		name string
		in   Input
		want bool
	}{
		{"normal", base, true},
		{"implicit-phase", Input{RuleID: "1002", Operator: "rx", Pattern: "a", Variables: []string{"REQUEST_URI"}, Transforms: []string{"none"}}, true},
		{"empty-pattern", Input{RuleID: "1003", Operator: "rx", Pattern: "", Phase: 1, Variables: []string{"REQUEST_URI"}, Transforms: []string{"none"}}, false},
		{"header-bang", Input{RuleID: "1004", Operator: "rx", Pattern: "a", Phase: 1, Variables: []string{"REQUEST_HEADERS:X!Foo"}, Transforms: []string{"none"}}, false},
		{"header-amp", Input{RuleID: "1005", Operator: "rx", Pattern: "a", Phase: 1, Variables: []string{"REQUEST_HEADERS:X&Foo"}, Transforms: []string{"none"}}, false},
		{"chain", Input{RuleID: "1006", Operator: "rx", Pattern: "a", Phase: 1, Variables: []string{"REQUEST_URI"}, Transforms: []string{"none"}, HasChain: true}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Classify(tc.in).Eligible; got != tc.want {
				t.Fatalf("eligible=%v want %v", got, tc.want)
			}
		})
	}
}
