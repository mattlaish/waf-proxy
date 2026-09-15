package coverage

// CoverageReport summarizes conservative acceleration coverage. It is retained
// for callers from Slice A; Slice C's ruleset report lives in coverage/crs.
type CoverageReport struct {
	EligibleRules    int `json:"eligible_rules"`
	AcceleratedRules int `json:"accelerated_rules"`
	CorazaOnlyRules  int `json:"coraza_only_rules"`
	UnsafeRules      int `json:"unsafe_rules"`
}

// BuildReport creates a report from evaluated rule capabilities.
func BuildReport(rules []RuleCapability) CoverageReport {
	var r CoverageReport
	for _, rule := range rules {
		if rule.Eligible {
			r.EligibleRules++
			continue
		}
		r.CorazaOnlyRules++
	}
	return r
}
