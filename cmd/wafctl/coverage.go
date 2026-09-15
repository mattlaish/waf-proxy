package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	coveragecrs "waf-proxy/internal/coverage/crs"
)

func runCoverage(args []string) error {
	if len(args) < 1 || args[0] != "analyze" {
		return errors.New("usage: wafctl coverage analyze --rules PATH [--report FILE] [--inventory FILE] [--json]")
	}
	fs := flag.NewFlagSet("coverage analyze", flag.ContinueOnError)
	rules := fs.String("rules", "/etc/waf/coraza.conf", "CRS/Coraza entrypoint or rules directory")
	reportPath := fs.String("report", "", "write coverage report v2 JSON")
	inventoryPath := fs.String("inventory", "", "write full rule capability inventory JSON")
	jsonOut := fs.Bool("json", false, "print coverage report as JSON")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	rs, err := coveragecrs.LoadRuleset(*rules)
	if err != nil {
		return err
	}
	inv := coveragecrs.BuildInventory(rs)
	report := coveragecrs.BuildCoverageReportV2(inv)

	if *reportPath != "" {
		b, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		b = append(b, '\n')
		if err := atomicWrite(*reportPath, b, 0600); err != nil {
			return fmt.Errorf("write report: %w", err)
		}
	}
	if *inventoryPath != "" {
		b, err := json.MarshalIndent(inv, "", "  ")
		if err != nil {
			return err
		}
		b = append(b, '\n')
		if err := atomicWrite(*inventoryPath, b, 0600); err != nil {
			return fmt.Errorf("write inventory: %w", err)
		}
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}
	fmt.Printf("CRS Coverage Report v2\n")
	fmt.Printf("Ruleset:           %s\n", report.Ruleset)
	fmt.Printf("Files:             %d\n", report.Files)
	fmt.Printf("Total rules:       %d\n", report.TotalRules)
	fmt.Printf("Eligible:          %d\n", report.EligibleRules)
	fmt.Printf("Coraza-only:       %d\n", report.CorazaOnlyRules)
	fmt.Printf("Unsupported:       %d\n", report.UnsupportedRules)
	fmt.Printf("Duplicate IDs:     %d\n", report.DuplicateRuleIDs)
	fmt.Printf("Warnings:          %d\n", report.Warnings)
	fmt.Printf("Coverage:          %.2f%%\n", report.CoveragePercentage)
	if len(report.Reasons) > 0 {
		fmt.Println("Blocking reasons:")
		for _, key := range coveragecrs.SortedReasonKeys(report.Reasons) {
			fmt.Printf("  %-42s %d\n", key, report.Reasons[key])
		}
	}
	return nil
}
