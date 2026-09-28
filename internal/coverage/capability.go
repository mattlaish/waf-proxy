package coverage

import shared "waf-proxy/internal/capability"

// RuleCapability describes whether a rule can be safely considered for the
// current runtime adapter. The eligibility decision itself lives in
// internal/capability and is shared with internal/vectoraccel.
type RuleCapability struct {
	RuleID     string   `json:"rule_id"`
	Phase      int      `json:"phase,omitempty"`
	Operator   string   `json:"operator"`
	Pattern    string   `json:"pattern,omitempty"`
	Variables  []string `json:"variables"`
	Transforms []string `json:"transforms"`
	HasChain   bool     `json:"has_chain"`
	Negated    bool     `json:"negated"`
	Eligible   bool     `json:"eligible"`
	Reason     string   `json:"reason,omitempty"`
}

func EvaluateRule(rule RuleCapability) RuleCapability {
	result := shared.Classify(shared.Input{
		RuleID:     rule.RuleID,
		Phase:      rule.Phase,
		Operator:   rule.Operator,
		Pattern:    rule.Pattern,
		Variables:  append([]string(nil), rule.Variables...),
		Transforms: append([]string(nil), rule.Transforms...),
		HasChain:   rule.HasChain,
		Negated:    rule.Negated,
	})
	rule.Eligible = result.Eligible
	rule.Reason = result.Reason
	if result.Eligible {
		rule.Phase = result.Phase
	}
	return rule
}
