package coverage

import (
	"strings"

	shared "waf-proxy/internal/capability"
)

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

type TransformCapability struct {
	Name      string `json:"name"`
	Supported bool   `json:"supported"`
	Exact     bool   `json:"exact"`
	Reason    string `json:"reason,omitempty"`
}

var transforms = map[string]TransformCapability{
	"none":             {Name: "none", Supported: true, Exact: true},
	"lowercase":        {Name: "lowercase", Supported: true, Exact: true},
	"uppercase":        {Name: "uppercase", Supported: false, Exact: false, Reason: "runtime adapter does not reproduce uppercase yet"},
	"urldecode":        {Name: "urldecode", Supported: false, Exact: false, Reason: "requires exact Coraza decoding semantics"},
	"removewhitespace": {Name: "removewhitespace", Supported: false, Exact: false, Reason: "requires exact Coraza normalization semantics"},
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

func TransformCapabilities() []TransformCapability {
	order := []string{"none", "lowercase", "uppercase", "urldecode", "removewhitespace"}
	out := make([]TransformCapability, 0, len(order))
	for _, name := range order {
		out = append(out, transforms[strings.ToLower(name)])
	}
	return out
}
