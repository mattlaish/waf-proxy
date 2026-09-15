package crs

import "waf-proxy/internal/coverage"

// Analyze converts CRS metadata into the conservative VectorScan capability
// model. This mirrors current runtime eligibility and does not promote a rule.
func Analyze(rule Rule) coverage.RuleCapability {
	return coverage.EvaluateRule(coverage.RuleCapability{
		RuleID:     rule.ID,
		Phase:      rule.Phase,
		Operator:   rule.Operator,
		Pattern:    rule.Pattern,
		Variables:  append([]string(nil), rule.Variables...),
		Transforms: append([]string(nil), rule.Transforms...),
		HasChain:   rule.HasChain,
		Negated:    rule.Negated,
	})
}
