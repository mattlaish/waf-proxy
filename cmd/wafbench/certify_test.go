package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func benchmarkFile(system SystemInfo, results ...Result) ResultFile {
	return ResultFile{ToolVersion: toolVersion, GeneratedAt: "2026-09-16T00:00:00Z", System: system, Results: results}
}

func writeBenchmarkFile(t *testing.T, dir, name string, rf ResultFile) string {
	t.Helper()
	b, err := json.Marshal(rf)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, b, 0600); err != nil {
		t.Fatal(err)
	}
	return p
}

func resultFor(name string, rps, p99 float64) Result {
	return Result{Mode: "http", Scenario: name, DurationSeconds: 10, Requests: 1000, Successes: 1000, RPS: rps, P99MS: p99, ErrorRatePercent: 0, GOMAXPROCS: 8}
}

func TestPerformanceCertificationWithoutTargetStaysNotRun(t *testing.T) {
	dir := t.TempDir()
	sys := SystemInfo{OS: "linux", Arch: "amd64", GoVersion: "go1.25.0", NumCPU: 8, CPUModel: "test"}
	proxy := writeBenchmarkFile(t, dir, "proxy.json", benchmarkFile(sys, resultFor("clean-get", 1000, 10)))
	coraza := writeBenchmarkFile(t, dir, "coraza.json", benchmarkFile(sys, resultFor("clean-get", 800, 15)))

	report, err := buildPerformanceCertification(proxy, coraza, "", "", "", "", "", []string{"clean-get"})
	if err != nil {
		t.Fatal(err)
	}
	if report.EvidenceStatus != "PASS" {
		t.Fatalf("evidence status=%s", report.EvidenceStatus)
	}
	if report.Status != "NOT_RUN" {
		t.Fatalf("certification status=%s; missing approved target must remain NOT_RUN", report.Status)
	}
}

func TestPerformanceCertificationTargetPass(t *testing.T) {
	dir := t.TempDir()
	sys := SystemInfo{OS: "linux", Arch: "amd64", GoVersion: "go1.25.0", NumCPU: 8, CPUModel: "test"}
	proxy := writeBenchmarkFile(t, dir, "proxy.json", benchmarkFile(sys, resultFor("clean-get", 1000, 10)))
	coraza := writeBenchmarkFile(t, dir, "coraza.json", benchmarkFile(sys, resultFor("clean-get", 850, 12)))
	minRPS, maxP99, maxErr := 800.0, 20.0, 0.1
	target := PerformanceTarget{Format: "waf-proxy-performance-target-v1", Profile: "approved-test", Requirements: []PerformanceScenarioRequirement{{Role: "coraza_crs", Scenario: "clean-get", MinRPS: &minRPS, MaxP99MS: &maxP99, MaxErrorRatePercent: &maxErr}}}
	b, _ := json.Marshal(target)
	targetPath := filepath.Join(dir, "target.json")
	if err := os.WriteFile(targetPath, b, 0600); err != nil {
		t.Fatal(err)
	}

	report, err := buildPerformanceCertification(proxy, coraza, "", "", targetPath, "", "", []string{"clean-get"})
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "PASS" || report.TargetStatus != "PASS" {
		t.Fatalf("status=%s target=%s", report.Status, report.TargetStatus)
	}
}

func TestVectorScanEvidenceRequiresZeroFalseNegativeGate(t *testing.T) {
	dir := t.TempDir()
	sys := SystemInfo{OS: "linux", Arch: "amd64", GoVersion: "go1.25.0", NumCPU: 8, CPUModel: "test"}
	proxy := writeBenchmarkFile(t, dir, "proxy.json", benchmarkFile(sys, resultFor("clean-get", 1000, 10)))
	coraza := writeBenchmarkFile(t, dir, "coraza.json", benchmarkFile(sys, resultFor("clean-get", 800, 15)))
	vector := writeBenchmarkFile(t, dir, "vector.json", benchmarkFile(sys, resultFor("clean-get", 900, 13)))
	minRPS := 850.0
	target := PerformanceTarget{Format: "waf-proxy-performance-target-v1", Profile: "accelerated", Requirements: []PerformanceScenarioRequirement{{Role: "vectorscan_assisted", Scenario: "clean-get", MinRPS: &minRPS}}}
	targetBytes, _ := json.Marshal(target)
	targetPath := filepath.Join(dir, "target.json")
	_ = os.WriteFile(targetPath, targetBytes, 0600)

	report, err := buildPerformanceCertification(proxy, coraza, vector, "", targetPath, "", "", []string{"clean-get"})
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "BLOCKED" {
		t.Fatalf("status=%s; vectorscan evidence without real zero-FN gate must block certification", report.Status)
	}

	q := vectorQualificationReport{Format: "waf-phase1-vectorscan-differential-v1", Result: "PASS", FalseNegativeCount: 0, ZeroFalseNegatives: true}
	qb, _ := json.Marshal(q)
	qPath := filepath.Join(dir, "phase1.json")
	_ = os.WriteFile(qPath, qb, 0600)
	report, err = buildPerformanceCertification(proxy, coraza, vector, qPath, targetPath, "", "", []string{"clean-get"})
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "PASS" || report.VectorScanSecurity.Status != "PASS" {
		t.Fatalf("status=%s security=%s", report.Status, report.VectorScanSecurity.Status)
	}
}

func TestMismatchedSystemsBlockComparison(t *testing.T) {
	dir := t.TempDir()
	proxySystem := SystemInfo{OS: "linux", Arch: "amd64", GoVersion: "go1.25.0", NumCPU: 8, CPUModel: "cpu-a"}
	corazaSystem := proxySystem
	corazaSystem.CPUModel = "cpu-b"
	proxy := writeBenchmarkFile(t, dir, "proxy.json", benchmarkFile(proxySystem, resultFor("clean-get", 1000, 10)))
	coraza := writeBenchmarkFile(t, dir, "coraza.json", benchmarkFile(corazaSystem, resultFor("clean-get", 800, 15)))

	report, err := buildPerformanceCertification(proxy, coraza, "", "", "", "", "", []string{"clean-get"})
	if err != nil {
		t.Fatal(err)
	}
	if report.EvidenceStatus != "BLOCKED" {
		t.Fatalf("evidence=%s", report.EvidenceStatus)
	}
}

func TestRegressionComparison(t *testing.T) {
	old := PerformanceCertificationReport{Format: performanceCertificationFormat, Scenarios: []PerformanceScenarioEvidence{{Scenario: "clean-get", CorazaCRS: &PerformanceMetricSnapshot{RPS: 100, P99MS: 10}}}}
	cur := PerformanceCertificationReport{Format: performanceCertificationFormat, Scenarios: []PerformanceScenarioEvidence{{Scenario: "clean-get", CorazaCRS: &PerformanceMetricSnapshot{RPS: 90, P99MS: 12}}}}
	d := compareCertificationReports(old, cur)
	if len(d) != 1 {
		t.Fatalf("len=%d", len(d))
	}
	if d[0].RPSChangePercent != -10 {
		t.Fatalf("rps delta=%f", d[0].RPSChangePercent)
	}
	if d[0].P99ChangePercent != 20 {
		t.Fatalf("p99 delta=%f", d[0].P99ChangePercent)
	}
}

func TestRunShapeMismatchBlocksEvidence(t *testing.T) {
	dir := t.TempDir()
	sys := SystemInfo{Hostname: "host-a", OS: "linux", Arch: "amd64", GoVersion: "go1.25.0", NumCPU: 8, CPUModel: "test"}
	base := resultFor("clean-get", 1000, 10)
	base.Workers = 64
	base.AvgRequestBytes = 128
	base.AvgResponseBytes = 1024
	corazaResult := resultFor("clean-get", 800, 15)
	corazaResult.Workers = 32
	corazaResult.AvgRequestBytes = 128
	corazaResult.AvgResponseBytes = 1024
	proxy := writeBenchmarkFile(t, dir, "proxy.json", benchmarkFile(sys, base))
	coraza := writeBenchmarkFile(t, dir, "coraza.json", benchmarkFile(sys, corazaResult))

	report, err := buildPerformanceCertification(proxy, coraza, "", "", "", "", "", []string{"clean-get"})
	if err != nil {
		t.Fatal(err)
	}
	if report.EvidenceStatus != "BLOCKED" {
		t.Fatalf("evidence=%s", report.EvidenceStatus)
	}
}
