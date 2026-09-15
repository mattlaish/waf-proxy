package crs

import (
	"sort"
	"strings"
)

const ReportSchemaVersion = "phase2-coverage-report-v2"

// CoverageReportV2 summarizes actual analyzer coverage over one ingested
// ruleset. Eligible means analyzer-compatible with the current runtime scope;
// it never means VALIDATED or ACCELERATED.
type CoverageReportV2 struct {
	SchemaVersion      string         `json:"schema_version"`
	Ruleset            string         `json:"ruleset"`
	Files              int            `json:"files"`
	TotalRules         int            `json:"total_rules"`
	EligibleRules      int            `json:"eligible_rules"`
	CorazaOnlyRules    int            `json:"coraza_only_rules"`
	UnsupportedRules   int            `json:"unsupported_rules"`
	DuplicateRuleIDs   int            `json:"duplicate_rule_ids"`
	Warnings           int            `json:"warnings"`
	CoveragePercentage float64        `json:"coverage_percentage"`
	Reasons            map[string]int `json:"reasons"`
}

func BuildCoverageReportV2(inv Inventory) CoverageReportV2 {
	r := CoverageReportV2{
		SchemaVersion:    ReportSchemaVersion,
		Ruleset:          inv.Ruleset,
		Files:            len(inv.Files),
		TotalRules:       len(inv.Rules),
		DuplicateRuleIDs: len(inv.DuplicateRuleIDs),
		Warnings:         len(inv.Warnings),
		Reasons:          map[string]int{},
	}
	for _, entry := range inv.Rules {
		switch entry.Classification {
		case "ELIGIBLE":
			r.EligibleRules++
		case "UNSUPPORTED":
			r.UnsupportedRules++
		default:
			r.CorazaOnlyRules++
		}
		if !entry.Eligible {
			r.Reasons[reasonKey(entry.Reason)]++
		}
	}
	if r.TotalRules > 0 {
		r.CoveragePercentage = float64(r.EligibleRules) * 100 / float64(r.TotalRules)
	}
	return r
}

func reasonKey(reason string) string {
	r := strings.ToLower(strings.TrimSpace(reason))
	replacer := strings.NewReplacer(" ", "_", ":", "", "/", "_", "-", "_")
	r = replacer.Replace(r)
	if r == "" {
		return "unspecified"
	}
	return r
}

// SortedReasonKeys supports deterministic text output without changing the
// JSON map format.
func SortedReasonKeys(reasons map[string]int) []string {
	keys := make([]string, 0, len(reasons))
	for k := range reasons {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
