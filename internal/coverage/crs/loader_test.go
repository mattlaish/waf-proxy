package crs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRulesetFollowsIncludesAndDetectsDuplicates(t *testing.T) {
	d := t.TempDir()
	child := filepath.Join(d, "rules.conf")
	if err := os.WriteFile(child, []byte("SecRule REQUEST_URI \"@rx a\" \"id:1001,phase:1,pass,t:none\"\nSecRule ARGS \"@rx b\" \"id:1001,phase:2,pass,t:none\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(d, "coraza.conf")
	if err := os.WriteFile(entry, []byte("Include rules.conf\n"), 0600); err != nil {
		t.Fatal(err)
	}
	rs, err := LoadRuleset(entry)
	if err != nil {
		t.Fatal(err)
	}
	if len(rs.Rules) != 2 || len(rs.DuplicateRuleIDs["1001"]) != 2 {
		t.Fatalf("unexpected ruleset: %#v", rs)
	}
	inv := BuildInventory(rs)
	report := BuildCoverageReportV2(inv)
	if report.TotalRules != 2 || report.EligibleRules != 0 || report.DuplicateRuleIDs != 1 {
		t.Fatalf("unexpected report: %#v", report)
	}
}
