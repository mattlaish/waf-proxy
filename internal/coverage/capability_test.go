package coverage

import "testing"

func TestEvaluateRuleRejectsUnsafeScopes(t *testing.T) {
	got := EvaluateRule(RuleCapability{
		RuleID:     "942100",
		Phase:      2,
		Operator:   "rx",
		Pattern:    "attack",
		Variables:  []string{"ARGS"},
		Transforms: []string{"none"},
	})
	if got.Eligible {
		t.Fatal("ARGS must remain Coraza-only")
	}
}

func TestEvaluateRuleAllowsFixedHeaderRegexScope(t *testing.T) {
	got := EvaluateRule(RuleCapability{
		RuleID:     "941100",
		Phase:      1,
		Operator:   "rx",
		Pattern:    "attack",
		Variables:  []string{"REQUEST_HEADERS:User-Agent"},
		Transforms: []string{"none", "lowercase"},
	})
	if !got.Eligible {
		t.Fatalf("expected eligible: %s", got.Reason)
	}
}

func TestEvaluateRuleRejectsBareHeaderCollectionAndUppercase(t *testing.T) {
	for _, tc := range []RuleCapability{
		{RuleID: "1", Phase: 1, Operator: "rx", Pattern: "a", Variables: []string{"REQUEST_HEADERS"}, Transforms: []string{"none"}},
		{RuleID: "2", Phase: 1, Operator: "rx", Pattern: "a", Variables: []string{"REQUEST_URI"}, Transforms: []string{"none", "uppercase"}},
	} {
		if got := EvaluateRule(tc); got.Eligible {
			t.Fatalf("unexpected eligibility: %#v", got)
		}
	}
}
