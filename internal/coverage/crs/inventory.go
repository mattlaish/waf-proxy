package crs

import (
	"sort"
	"strings"
)

const InventorySchemaVersion = "phase2-coverage-inventory-v2"

// InventoryEntry is the operator-visible rule capability record.
type InventoryEntry struct {
	RuleID         string   `json:"rule_id"`
	File           string   `json:"file"`
	Line           int      `json:"line"`
	Phase          int      `json:"phase,omitempty"`
	Operator       string   `json:"operator"`
	Variables      []string `json:"variables"`
	Transforms     []string `json:"transforms"`
	Tags           []string `json:"tags,omitempty"`
	Severity       string   `json:"severity,omitempty"`
	Classification string   `json:"classification"`
	Eligible       bool     `json:"eligible"`
	Reason         string   `json:"reason"`
}

// Inventory contains every parsed SecRule and is safe to feed into reporting;
// it is not an acceleration plan and cannot promote rules by itself.
type Inventory struct {
	SchemaVersion    string                    `json:"schema_version"`
	Ruleset          string                    `json:"ruleset"`
	Files            []string                  `json:"files"`
	Rules            []InventoryEntry          `json:"rules"`
	DuplicateRuleIDs map[string][]RuleLocation `json:"duplicate_rule_ids,omitempty"`
	Warnings         []Warning                 `json:"warnings,omitempty"`
}

func BuildInventory(rs Ruleset) Inventory {
	inv := Inventory{
		SchemaVersion:    InventorySchemaVersion,
		Ruleset:          rs.Root,
		Files:            append([]string(nil), rs.Files...),
		DuplicateRuleIDs: rs.DuplicateRuleIDs,
		Warnings:         append([]Warning(nil), rs.Warnings...),
	}
	for _, r := range rs.Rules {
		entry := InventoryEntry{
			RuleID:     r.ID,
			File:       r.File,
			Line:       r.Line,
			Phase:      r.Phase,
			Operator:   r.Operator,
			Variables:  append([]string(nil), r.Variables...),
			Transforms: append([]string(nil), r.Transforms...),
			Tags:       append([]string(nil), r.Tags...),
			Severity:   r.Severity,
		}
		if r.ID == "" || r.Operator == "" || len(r.Variables) == 0 {
			entry.Classification = "UNSUPPORTED"
			entry.Reason = "incomplete rule metadata"
		} else if _, dup := rs.DuplicateRuleIDs[r.ID]; dup {
			entry.Classification = "CORAZA_ONLY"
			entry.Reason = "duplicate rule id"
		} else {
			capability := Analyze(r)
			entry.Eligible = capability.Eligible
			entry.Reason = capability.Reason
			if capability.Eligible {
				entry.Classification = "ELIGIBLE"
			} else {
				entry.Classification = "CORAZA_ONLY"
			}
		}
		inv.Rules = append(inv.Rules, entry)
	}
	sort.Slice(inv.Rules, func(i, j int) bool {
		if inv.Rules[i].RuleID != inv.Rules[j].RuleID {
			return numericLess(inv.Rules[i].RuleID, inv.Rules[j].RuleID)
		}
		if inv.Rules[i].File != inv.Rules[j].File {
			return inv.Rules[i].File < inv.Rules[j].File
		}
		return inv.Rules[i].Line < inv.Rules[j].Line
	})
	return inv
}

func numericLess(a, b string) bool {
	if len(a) != len(b) {
		return len(a) < len(b)
	}
	return strings.Compare(a, b) < 0
}
