package main

import "testing"

func TestReliabilityReportBoundary(t *testing.T) {
	report := NewReliabilityReport()
	if report.Status != "NOT_RUN" {
		t.Fatalf("unexpected status: %s", report.Status)
	}
	if len(report.Scenarios) == 0 {
		t.Fatal("expected scenarios")
	}
}
