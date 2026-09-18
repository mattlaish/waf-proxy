package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

const performanceCertificationFormat = "waf-proxy-performance-certification-v1"

var defaultCertificationScenarios = []string{
	"clean-get",
	"json-1k",
	"json-16k",
	"json-64k",
	"json-256k",
	"sqli-64k",
	"xss-64k",
	"traversal",
}

type PerformanceMetricSnapshot struct {
	Requests           uint64  `json:"requests"`
	Workers            int     `json:"workers"`
	RPS                float64 `json:"rps"`
	ErrorRatePercent   float64 `json:"error_rate_percent"`
	P50MS              float64 `json:"p50_ms"`
	P95MS              float64 `json:"p95_ms"`
	P99MS              float64 `json:"p99_ms"`
	AvgRequestBytes    float64 `json:"avg_request_bytes"`
	AvgResponseBytes   float64 `json:"avg_response_bytes"`
	AppMbps            float64 `json:"app_mbps"`
	AppGbps            float64 `json:"app_gbps"`
	NetRXGbps          float64 `json:"net_rx_gbps,omitempty"`
	NetTXGbps          float64 `json:"net_tx_gbps,omitempty"`
	ProcessCPUPercent  float64 `json:"process_cpu_percent,omitempty"`
	CPUUSPerRequest    float64 `json:"cpu_us_per_request,omitempty"`
	RSSStartMiB        float64 `json:"rss_start_mib,omitempty"`
	RSSEndMiB          float64 `json:"rss_end_mib,omitempty"`
	HostSoftIRQPercent float64 `json:"host_softirq_percent,omitempty"`
	GeneratorCPUCores  float64 `json:"generator_cpu_cores,omitempty"`
	GOMAXPROCS         int     `json:"gomaxprocs,omitempty"`
	DurationSeconds    float64 `json:"duration_seconds"`
}

type PerformanceEvidenceInput struct {
	Role        string     `json:"role"`
	Path        string     `json:"path"`
	SHA256      string     `json:"sha256"`
	ToolVersion string     `json:"tool_version"`
	GeneratedAt string     `json:"generated_at"`
	System      SystemInfo `json:"system"`
}

type PerformanceScenarioEvidence struct {
	Scenario                   string                     `json:"scenario"`
	ReverseProxyBaseline       *PerformanceMetricSnapshot `json:"reverse_proxy_baseline,omitempty"`
	CorazaCRS                  *PerformanceMetricSnapshot `json:"coraza_crs,omitempty"`
	VectorScanAssisted         *PerformanceMetricSnapshot `json:"vectorscan_assisted,omitempty"`
	CorazaRPSChangePercent     *float64                   `json:"coraza_rps_change_percent,omitempty"`
	CorazaP99ChangePercent     *float64                   `json:"coraza_p99_change_percent,omitempty"`
	VectorScanRPSChangePercent *float64                   `json:"vectorscan_rps_change_percent,omitempty"`
	VectorScanP99ChangePercent *float64                   `json:"vectorscan_p99_change_percent,omitempty"`
	Comparable                 bool                       `json:"comparable"`
	Issues                     []string                   `json:"issues,omitempty"`
}

type PerformanceScenarioRequirement struct {
	Role                 string   `json:"role"`
	Scenario             string   `json:"scenario"`
	MinRPS               *float64 `json:"min_rps,omitempty"`
	MinAppGbps           *float64 `json:"min_app_gbps,omitempty"`
	MaxP99MS             *float64 `json:"max_p99_ms,omitempty"`
	MaxErrorRatePercent  *float64 `json:"max_error_rate_percent,omitempty"`
	MaxProcessCPUPercent *float64 `json:"max_process_cpu_percent,omitempty"`
	MaxRSSMiB            *float64 `json:"max_rss_mib,omitempty"`
}

type PerformanceTarget struct {
	Format       string                           `json:"format"`
	Profile      string                           `json:"profile"`
	Requirements []PerformanceScenarioRequirement `json:"requirements"`
}

type PerformanceTargetEvaluation struct {
	Role                    string   `json:"role"`
	Scenario                string   `json:"scenario"`
	Status                  string   `json:"status"`
	MinRPS                  *float64 `json:"min_rps,omitempty"`
	ActualRPS               *float64 `json:"actual_rps,omitempty"`
	MinAppGbps              *float64 `json:"min_app_gbps,omitempty"`
	ActualAppGbps           *float64 `json:"actual_app_gbps,omitempty"`
	MaxP99MS                *float64 `json:"max_p99_ms,omitempty"`
	ActualP99MS             *float64 `json:"actual_p99_ms,omitempty"`
	MaxErrorRatePercent     *float64 `json:"max_error_rate_percent,omitempty"`
	ActualErrorRatePercent  *float64 `json:"actual_error_rate_percent,omitempty"`
	MaxProcessCPUPercent    *float64 `json:"max_process_cpu_percent,omitempty"`
	ActualProcessCPUPercent *float64 `json:"actual_process_cpu_percent,omitempty"`
	MaxRSSMiB               *float64 `json:"max_rss_mib,omitempty"`
	ActualRSSMiB            *float64 `json:"actual_rss_mib,omitempty"`
	Reasons                 []string `json:"reasons,omitempty"`
}

type VectorScanSecurityGate struct {
	Status             string `json:"status"`
	Path               string `json:"path,omitempty"`
	SHA256             string `json:"sha256,omitempty"`
	Format             string `json:"format,omitempty"`
	Result             string `json:"result,omitempty"`
	FalseNegativeCount int    `json:"false_negative_count,omitempty"`
	ZeroFalseNegatives bool   `json:"zero_false_negatives,omitempty"`
	Reason             string `json:"reason,omitempty"`
}

type PerformanceRegressionDelta struct {
	Role             string  `json:"role"`
	Scenario         string  `json:"scenario"`
	RPSChangePercent float64 `json:"rps_change_percent"`
	P99ChangePercent float64 `json:"p99_change_percent"`
}

type PerformanceCertificationReport struct {
	Format               string                        `json:"format"`
	GeneratedAt          string                        `json:"generated_at"`
	Status               string                        `json:"status"`
	EvidenceStatus       string                        `json:"evidence_status"`
	TargetStatus         string                        `json:"target_status"`
	Profile              string                        `json:"profile,omitempty"`
	SourceArtifactSHA256 string                        `json:"source_artifact_sha256,omitempty"`
	RequiredScenarios    []string                      `json:"required_scenarios"`
	Inputs               []PerformanceEvidenceInput    `json:"inputs,omitempty"`
	Scenarios            []PerformanceScenarioEvidence `json:"scenarios,omitempty"`
	TargetEvaluations    []PerformanceTargetEvaluation `json:"target_evaluations,omitempty"`
	VectorScanSecurity   VectorScanSecurityGate        `json:"vectorscan_security"`
	Regression           []PerformanceRegressionDelta  `json:"regression,omitempty"`
	Warnings             []string                      `json:"warnings,omitempty"`
	TruthBoundary        string                        `json:"truth_boundary"`
}

type vectorQualificationReport struct {
	Format             string `json:"format"`
	FalseNegativeCount int    `json:"false_negative_count"`
	ZeroFalseNegatives bool   `json:"zero_false_negatives"`
	Result             string `json:"result"`
}

func runCertify(args []string) error {
	fs := flag.NewFlagSet("certify", flag.ContinueOnError)
	proxyPath := fs.String("proxy-baseline", "", "JSON from wafbench http against TLS reverse proxy with WAF rules disabled")
	corazaPath := fs.String("coraza-crs", "", "JSON from wafbench http against TLS reverse proxy with Coraza + production CRS enabled")
	vectorPath := fs.String("vectorscan", "", "optional JSON from wafbench http against the same Coraza/CRS configuration with validated VectorScan acceleration enabled")
	securityPath := fs.String("vectorscan-qualification", "", "Phase 1 wafqualify JSON proving zero false negatives for the VectorScan run")
	targetPath := fs.String("target", "", "optional approved performance target JSON; without it certification status remains NOT_RUN")
	previousPath := fs.String("previous", "", "optional prior performance-certification-v1 JSON for regression deltas")
	scenariosFlag := fs.String("scenarios", strings.Join(defaultCertificationScenarios, ","), "comma-separated scenarios required in each supplied benchmark file")
	sourceSHA := fs.String("source-sha256", "", "optional SHA-256 of the source/release artifact under test")
	out := fs.String("out", "performance-certification.json", "output JSON report")
	if err := fs.Parse(args); err != nil {
		return err
	}

	required := splitNonEmpty(*scenariosFlag)
	if len(required) == 0 {
		return fmt.Errorf("--scenarios must contain at least one scenario")
	}
	report, err := buildPerformanceCertification(*proxyPath, *corazaPath, *vectorPath, *securityPath, *targetPath, *previousPath, *sourceSHA, required)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if err := os.WriteFile(*out, b, 0640); err != nil {
		return err
	}
	fmt.Printf("RESULT %s evidence=%s target=%s output=%s\n", report.Status, report.EvidenceStatus, report.TargetStatus, *out)
	return nil
}

func buildPerformanceCertification(proxyPath, corazaPath, vectorPath, securityPath, targetPath, previousPath, sourceSHA string, required []string) (PerformanceCertificationReport, error) {
	report := PerformanceCertificationReport{
		Format:               performanceCertificationFormat,
		GeneratedAt:          time.Now().UTC().Format(time.RFC3339Nano),
		Status:               "NOT_RUN",
		EvidenceStatus:       "NOT_RUN",
		TargetStatus:         "NOT_RUN",
		SourceArtifactSHA256: strings.ToLower(strings.TrimSpace(sourceSHA)),
		RequiredScenarios:    append([]string(nil), required...),
		VectorScanSecurity:   VectorScanSecurityGate{Status: "NOT_RUN", Reason: "VectorScan benchmark evidence not supplied"},
		TruthBoundary:        "PASS certifies only the supplied, hash-bound benchmark evidence against an explicit approved target. Missing target or missing real benchmark evidence remains NOT_RUN; synthetic estimates are never promoted to measured sizing evidence.",
	}
	if report.SourceArtifactSHA256 != "" && !validSHA256(report.SourceArtifactSHA256) {
		return report, fmt.Errorf("--source-sha256 must be a 64-character hexadecimal SHA-256")
	}

	paths := map[string]string{
		"reverse_proxy_baseline": strings.TrimSpace(proxyPath),
		"coraza_crs":             strings.TrimSpace(corazaPath),
		"vectorscan_assisted":    strings.TrimSpace(vectorPath),
	}
	files := map[string]ResultFile{}
	for _, role := range []string{"reverse_proxy_baseline", "coraza_crs", "vectorscan_assisted"} {
		path := paths[role]
		if path == "" {
			continue
		}
		rf, err := loadResultFile(path)
		if err != nil {
			return report, fmt.Errorf("%s evidence: %w", role, err)
		}
		hash, err := sha256File(path)
		if err != nil {
			return report, err
		}
		if err := validateHTTPBenchmarkFile(role, rf, required); err != nil {
			report.Warnings = append(report.Warnings, err.Error())
		}
		files[role] = rf
		report.Inputs = append(report.Inputs, PerformanceEvidenceInput{Role: role, Path: path, SHA256: hash, ToolVersion: rf.ToolVersion, GeneratedAt: rf.GeneratedAt, System: rf.System})
	}

	// Core certification requires both proxy baseline and Coraza+CRS evidence.
	if _, ok := files["reverse_proxy_baseline"]; !ok {
		report.Warnings = append(report.Warnings, "reverse_proxy_baseline evidence is missing")
	}
	if _, ok := files["coraza_crs"]; !ok {
		report.Warnings = append(report.Warnings, "coraza_crs evidence is missing")
	}

	report.Scenarios = buildScenarioEvidence(files, required)
	report.Warnings = append(report.Warnings, benchmarkEvidenceWarnings(files)...)
	report.EvidenceStatus = evaluateEvidenceStatus(files, report.Scenarios, required)
	if report.EvidenceStatus == "PASS" && systemsComparable(files) == false {
		report.EvidenceStatus = "BLOCKED"
		report.Warnings = append(report.Warnings, "benchmark files were collected on different system identities; cross-configuration comparison is blocked")
	}

	if _, ok := files["vectorscan_assisted"]; ok {
		gate, err := loadVectorScanSecurityGate(securityPath)
		if err != nil {
			return report, err
		}
		report.VectorScanSecurity = gate
	} else if strings.TrimSpace(securityPath) != "" {
		report.Warnings = append(report.Warnings, "VectorScan qualification report supplied without a vectorscan benchmark input")
	}

	var target *PerformanceTarget
	if strings.TrimSpace(targetPath) != "" {
		t, err := loadPerformanceTarget(targetPath)
		if err != nil {
			return report, err
		}
		target = &t
		report.Profile = t.Profile
		report.TargetEvaluations = evaluatePerformanceTarget(t, files)
		report.TargetStatus = aggregateTargetStatus(report.TargetEvaluations)
	} else {
		report.Warnings = append(report.Warnings, "no approved target supplied; measured evidence may be reviewed, but production performance certification remains NOT_RUN")
	}

	if strings.TrimSpace(previousPath) != "" {
		previous, err := loadPerformanceCertification(previousPath)
		if err != nil {
			return report, err
		}
		report.Regression = compareCertificationReports(previous, report)
	}

	// Certification status is intentionally stricter than evidence completeness.
	_, hasVector := files["vectorscan_assisted"]
	switch {
	case report.EvidenceStatus == "FAIL" || report.TargetStatus == "FAIL" || report.VectorScanSecurity.Status == "FAIL":
		report.Status = "FAIL"
	case report.EvidenceStatus == "BLOCKED" || report.TargetStatus == "BLOCKED":
		report.Status = "BLOCKED"
	case report.EvidenceStatus != "PASS" || target == nil:
		report.Status = "NOT_RUN"
	case hasVector && report.VectorScanSecurity.Status != "PASS":
		report.Status = "BLOCKED"
	case report.TargetStatus == "PASS":
		report.Status = "PASS"
	default:
		report.Status = "NOT_RUN"
	}

	return report, nil
}

func validateHTTPBenchmarkFile(role string, rf ResultFile, required []string) error {
	if strings.TrimSpace(rf.ToolVersion) == "" {
		return fmt.Errorf("%s evidence is missing tool_version", role)
	}
	byScenario := indexResults(rf.Results)
	var issues []string
	for _, name := range required {
		r, ok := byScenario[name]
		if !ok {
			issues = append(issues, "missing scenario "+name)
			continue
		}
		if r.Mode != "http" {
			issues = append(issues, fmt.Sprintf("scenario %s mode=%q, expected http full-proxy evidence", name, r.Mode))
		}
		if r.DurationSeconds <= 0 || r.Requests == 0 || r.RPS <= 0 {
			issues = append(issues, "scenario "+name+" has no executed throughput evidence")
		}
	}
	if len(issues) != 0 {
		return fmt.Errorf("%s evidence invalid: %s", role, strings.Join(issues, "; "))
	}
	return nil
}

func buildScenarioEvidence(files map[string]ResultFile, required []string) []PerformanceScenarioEvidence {
	indexes := map[string]map[string]Result{}
	for role, f := range files {
		indexes[role] = indexResults(f.Results)
	}
	out := make([]PerformanceScenarioEvidence, 0, len(required))
	for _, name := range required {
		e := PerformanceScenarioEvidence{Scenario: name, Comparable: true}
		if r, ok := indexes["reverse_proxy_baseline"][name]; ok {
			s := metricSnapshot(r)
			e.ReverseProxyBaseline = &s
		} else {
			e.Comparable = false
			e.Issues = append(e.Issues, "missing reverse_proxy_baseline")
		}
		if r, ok := indexes["coraza_crs"][name]; ok {
			s := metricSnapshot(r)
			e.CorazaCRS = &s
		} else {
			e.Comparable = false
			e.Issues = append(e.Issues, "missing coraza_crs")
		}
		if _, supplied := files["vectorscan_assisted"]; supplied {
			if r, ok := indexes["vectorscan_assisted"][name]; ok {
				s := metricSnapshot(r)
				e.VectorScanAssisted = &s
			} else {
				e.Comparable = false
				e.Issues = append(e.Issues, "missing vectorscan_assisted")
			}
		}
		if e.ReverseProxyBaseline != nil && e.CorazaCRS != nil {
			e.CorazaRPSChangePercent = pctChangePtr(e.ReverseProxyBaseline.RPS, e.CorazaCRS.RPS)
			e.CorazaP99ChangePercent = pctChangePtr(e.ReverseProxyBaseline.P99MS, e.CorazaCRS.P99MS)
		}
		if e.CorazaCRS != nil && e.VectorScanAssisted != nil {
			e.VectorScanRPSChangePercent = pctChangePtr(e.CorazaCRS.RPS, e.VectorScanAssisted.RPS)
			e.VectorScanP99ChangePercent = pctChangePtr(e.CorazaCRS.P99MS, e.VectorScanAssisted.P99MS)
		}
		if e.ReverseProxyBaseline != nil && e.CorazaCRS != nil {
			e.Issues = append(e.Issues, compareRunShape("reverse_proxy_baseline", *e.ReverseProxyBaseline, "coraza_crs", *e.CorazaCRS)...)
		}
		if e.CorazaCRS != nil && e.VectorScanAssisted != nil {
			e.Issues = append(e.Issues, compareRunShape("coraza_crs", *e.CorazaCRS, "vectorscan_assisted", *e.VectorScanAssisted)...)
		}
		if len(e.Issues) != 0 {
			e.Comparable = false
		}
		out = append(out, e)
	}
	return out
}

func evaluateEvidenceStatus(files map[string]ResultFile, scenarios []PerformanceScenarioEvidence, required []string) string {
	if _, ok := files["reverse_proxy_baseline"]; !ok {
		return "NOT_RUN"
	}
	if _, ok := files["coraza_crs"]; !ok {
		return "NOT_RUN"
	}
	for _, s := range scenarios {
		if !s.Comparable || s.ReverseProxyBaseline == nil || s.CorazaCRS == nil {
			return "BLOCKED"
		}
		if _, vectorSupplied := files["vectorscan_assisted"]; vectorSupplied && s.VectorScanAssisted == nil {
			return "BLOCKED"
		}
	}
	for role, rf := range files {
		if err := validateHTTPBenchmarkFile(role, rf, required); err != nil {
			return "BLOCKED"
		}
	}
	return "PASS"
}

func systemsComparable(files map[string]ResultFile) bool {
	var base *SystemInfo
	for _, role := range []string{"reverse_proxy_baseline", "coraza_crs", "vectorscan_assisted"} {
		rf, ok := files[role]
		if !ok {
			continue
		}
		if base == nil {
			b := rf.System
			base = &b
			continue
		}
		if base.OS != rf.System.OS || base.Arch != rf.System.Arch || base.NumCPU != rf.System.NumCPU || base.CPUModel != rf.System.CPUModel || base.GoVersion != rf.System.GoVersion {
			return false
		}
		if base.Hostname != "" && rf.System.Hostname != "" && base.Hostname != rf.System.Hostname {
			return false
		}
	}
	return true
}

func loadVectorScanSecurityGate(path string) (VectorScanSecurityGate, error) {
	gate := VectorScanSecurityGate{Status: "NOT_RUN", Reason: "VectorScan benchmark evidence requires a real Phase 1 zero-false-negative qualification report"}
	path = strings.TrimSpace(path)
	if path == "" {
		return gate, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return gate, err
	}
	var r vectorQualificationReport
	if err := json.Unmarshal(b, &r); err != nil {
		return gate, fmt.Errorf("VectorScan qualification report: %w", err)
	}
	hash, err := sha256Bytes(b)
	if err != nil {
		return gate, err
	}
	gate.Path, gate.SHA256, gate.Format, gate.Result = path, hash, r.Format, r.Result
	gate.FalseNegativeCount, gate.ZeroFalseNegatives = r.FalseNegativeCount, r.ZeroFalseNegatives
	if r.Format != "waf-phase1-vectorscan-differential-v1" {
		gate.Status = "FAIL"
		gate.Reason = "unexpected VectorScan qualification report format"
		return gate, nil
	}
	if r.Result == "PASS" && r.FalseNegativeCount == 0 && r.ZeroFalseNegatives {
		gate.Status = "PASS"
		gate.Reason = "real differential report records PASS with zero observed false negatives"
		return gate, nil
	}
	if r.Result == "BLOCKED" || r.Result == "NOT_RUN" || r.Result == "" {
		gate.Status = "BLOCKED"
		gate.Reason = "VectorScan differential gate is not qualified"
		return gate, nil
	}
	gate.Status = "FAIL"
	gate.Reason = "VectorScan differential gate did not pass zero-false-negative requirement"
	return gate, nil
}

func loadPerformanceTarget(path string) (PerformanceTarget, error) {
	var t PerformanceTarget
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	if err := json.Unmarshal(b, &t); err != nil {
		return t, err
	}
	if t.Format != "waf-proxy-performance-target-v1" {
		return t, fmt.Errorf("performance target format must be waf-proxy-performance-target-v1")
	}
	if strings.TrimSpace(t.Profile) == "" {
		return t, fmt.Errorf("performance target profile is required")
	}
	if len(t.Requirements) == 0 {
		return t, fmt.Errorf("performance target requires at least one scenario requirement")
	}
	for i, r := range t.Requirements {
		if r.Role != "coraza_crs" && r.Role != "vectorscan_assisted" && r.Role != "reverse_proxy_baseline" {
			return t, fmt.Errorf("requirement %d has unsupported role %q", i, r.Role)
		}
		if strings.TrimSpace(r.Scenario) == "" {
			return t, fmt.Errorf("requirement %d scenario is required", i)
		}
		if r.MinRPS == nil && r.MinAppGbps == nil && r.MaxP99MS == nil && r.MaxErrorRatePercent == nil && r.MaxProcessCPUPercent == nil && r.MaxRSSMiB == nil {
			return t, fmt.Errorf("requirement %d must define at least one threshold", i)
		}
	}
	return t, nil
}

func evaluatePerformanceTarget(target PerformanceTarget, files map[string]ResultFile) []PerformanceTargetEvaluation {
	indexes := map[string]map[string]Result{}
	for role, f := range files {
		indexes[role] = indexResults(f.Results)
	}
	out := make([]PerformanceTargetEvaluation, 0, len(target.Requirements))
	for _, req := range target.Requirements {
		e := PerformanceTargetEvaluation{Role: req.Role, Scenario: req.Scenario, Status: "PASS", MinRPS: req.MinRPS, MinAppGbps: req.MinAppGbps, MaxP99MS: req.MaxP99MS, MaxErrorRatePercent: req.MaxErrorRatePercent, MaxProcessCPUPercent: req.MaxProcessCPUPercent, MaxRSSMiB: req.MaxRSSMiB}
		r, ok := indexes[req.Role][req.Scenario]
		if !ok {
			e.Status = "BLOCKED"
			e.Reasons = append(e.Reasons, "required benchmark evidence is missing")
			out = append(out, e)
			continue
		}
		e.ActualRPS = floatPtr(r.RPS)
		e.ActualAppGbps = floatPtr(r.AppMbps / 1000)
		e.ActualP99MS = floatPtr(r.P99MS)
		e.ActualErrorRatePercent = floatPtr(r.ErrorRatePercent)
		e.ActualProcessCPUPercent = floatPtr(r.ProcessCPUPercent)
		e.ActualRSSMiB = floatPtr(r.RSSEndMiB)
		if req.MinRPS != nil && r.RPS < *req.MinRPS {
			e.Status = "FAIL"
			e.Reasons = append(e.Reasons, fmt.Sprintf("rps %.3f < minimum %.3f", r.RPS, *req.MinRPS))
		}
		if req.MinAppGbps != nil && r.AppMbps/1000 < *req.MinAppGbps {
			e.Status = "FAIL"
			e.Reasons = append(e.Reasons, fmt.Sprintf("application throughput %.6f Gbps < minimum %.6f Gbps", r.AppMbps/1000, *req.MinAppGbps))
		}
		if req.MaxP99MS != nil && r.P99MS > *req.MaxP99MS {
			e.Status = "FAIL"
			e.Reasons = append(e.Reasons, fmt.Sprintf("p99 %.3fms > maximum %.3fms", r.P99MS, *req.MaxP99MS))
		}
		if req.MaxErrorRatePercent != nil && r.ErrorRatePercent > *req.MaxErrorRatePercent {
			e.Status = "FAIL"
			e.Reasons = append(e.Reasons, fmt.Sprintf("error rate %.6f%% > maximum %.6f%%", r.ErrorRatePercent, *req.MaxErrorRatePercent))
		}
		if req.MaxProcessCPUPercent != nil && r.ProcessCPUPercent > *req.MaxProcessCPUPercent {
			e.Status = "FAIL"
			e.Reasons = append(e.Reasons, fmt.Sprintf("process CPU %.3f%% > maximum %.3f%%", r.ProcessCPUPercent, *req.MaxProcessCPUPercent))
		}
		if req.MaxRSSMiB != nil && r.RSSEndMiB > *req.MaxRSSMiB {
			e.Status = "FAIL"
			e.Reasons = append(e.Reasons, fmt.Sprintf("RSS %.3f MiB > maximum %.3f MiB", r.RSSEndMiB, *req.MaxRSSMiB))
		}
		out = append(out, e)
	}
	return out
}

func aggregateTargetStatus(evals []PerformanceTargetEvaluation) string {
	if len(evals) == 0 {
		return "NOT_RUN"
	}
	status := "PASS"
	for _, e := range evals {
		switch e.Status {
		case "FAIL":
			return "FAIL"
		case "BLOCKED":
			status = "BLOCKED"
		case "NOT_RUN":
			if status == "PASS" {
				status = "NOT_RUN"
			}
		}
	}
	return status
}

func loadPerformanceCertification(path string) (PerformanceCertificationReport, error) {
	var r PerformanceCertificationReport
	b, err := os.ReadFile(path)
	if err != nil {
		return r, err
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return r, err
	}
	if r.Format != performanceCertificationFormat {
		return r, fmt.Errorf("previous report has incompatible format %q", r.Format)
	}
	return r, nil
}

func compareCertificationReports(previous, current PerformanceCertificationReport) []PerformanceRegressionDelta {
	type key struct{ role, scenario string }
	prev := map[key]PerformanceMetricSnapshot{}
	curr := map[key]PerformanceMetricSnapshot{}
	index := func(dst map[key]PerformanceMetricSnapshot, report PerformanceCertificationReport) {
		for _, s := range report.Scenarios {
			if s.ReverseProxyBaseline != nil {
				dst[key{"reverse_proxy_baseline", s.Scenario}] = *s.ReverseProxyBaseline
			}
			if s.CorazaCRS != nil {
				dst[key{"coraza_crs", s.Scenario}] = *s.CorazaCRS
			}
			if s.VectorScanAssisted != nil {
				dst[key{"vectorscan_assisted", s.Scenario}] = *s.VectorScanAssisted
			}
		}
	}
	index(prev, previous)
	index(curr, current)
	keys := make([]key, 0)
	for k := range curr {
		if _, ok := prev[k]; ok {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].role == keys[j].role {
			return keys[i].scenario < keys[j].scenario
		}
		return keys[i].role < keys[j].role
	})
	out := make([]PerformanceRegressionDelta, 0, len(keys))
	for _, k := range keys {
		p, c := prev[k], curr[k]
		out = append(out, PerformanceRegressionDelta{Role: k.role, Scenario: k.scenario, RPSChangePercent: pctChange(p.RPS, c.RPS), P99ChangePercent: pctChange(p.P99MS, c.P99MS)})
	}
	return out
}

func indexResults(results []Result) map[string]Result {
	out := make(map[string]Result, len(results))
	for _, r := range results {
		out[r.Scenario] = r
	}
	return out
}

func metricSnapshot(r Result) PerformanceMetricSnapshot {
	return PerformanceMetricSnapshot{
		Requests: r.Requests, Workers: r.Workers, RPS: r.RPS, ErrorRatePercent: r.ErrorRatePercent,
		P50MS: r.P50MS, P95MS: r.P95MS, P99MS: r.P99MS,
		AvgRequestBytes: r.AvgRequestBytes, AvgResponseBytes: r.AvgResponseBytes,
		AppMbps: r.AppMbps, AppGbps: r.AppMbps / 1000, NetRXGbps: r.NetRXMbps / 1000, NetTXGbps: r.NetTXMbps / 1000,
		ProcessCPUPercent: r.ProcessCPUPercent, CPUUSPerRequest: r.CPUUSPerRequest,
		RSSStartMiB: r.RSSStartMiB, RSSEndMiB: r.RSSEndMiB, HostSoftIRQPercent: r.HostSoftIRQPercent,
		GeneratorCPUCores: r.GeneratorCPUCores, GOMAXPROCS: r.GOMAXPROCS, DurationSeconds: r.DurationSeconds,
	}
}

func compareRunShape(aName string, a PerformanceMetricSnapshot, bName string, b PerformanceMetricSnapshot) []string {
	var issues []string
	if a.Workers != b.Workers {
		issues = append(issues, fmt.Sprintf("%s workers=%d differs from %s workers=%d", aName, a.Workers, bName, b.Workers))
	}
	if a.GOMAXPROCS != b.GOMAXPROCS {
		issues = append(issues, fmt.Sprintf("%s GOMAXPROCS=%d differs from %s GOMAXPROCS=%d", aName, a.GOMAXPROCS, bName, b.GOMAXPROCS))
	}
	if !nearlyEqual(a.AvgRequestBytes, b.AvgRequestBytes) {
		issues = append(issues, fmt.Sprintf("%s average request bytes %.3f differs from %s %.3f", aName, a.AvgRequestBytes, bName, b.AvgRequestBytes))
	}
	if !nearlyEqual(a.AvgResponseBytes, b.AvgResponseBytes) {
		issues = append(issues, fmt.Sprintf("%s average response bytes %.3f differs from %s %.3f", aName, a.AvgResponseBytes, bName, b.AvgResponseBytes))
	}
	return issues
}

func benchmarkEvidenceWarnings(files map[string]ResultFile) []string {
	var warnings []string
	for role, rf := range files {
		for _, r := range rf.Results {
			if r.GOMAXPROCS > 0 && r.GeneratorCPUCores >= float64(r.GOMAXPROCS)*0.80 {
				warnings = append(warnings, fmt.Sprintf("%s/%s load generator used %.2f cores of GOMAXPROCS=%d; result may be generator-limited", role, r.Scenario, r.GeneratorCPUCores, r.GOMAXPROCS))
			}
		}
	}
	sort.Strings(warnings)
	return warnings
}

func nearlyEqual(a, b float64) bool {
	if a == b {
		return true
	}
	d := a - b
	if d < 0 {
		d = -d
	}
	max := a
	if max < 0 {
		max = -max
	}
	absb := b
	if absb < 0 {
		absb = -absb
	}
	if absb > max {
		max = absb
	}
	if max < 1 {
		max = 1
	}
	return d/max < 0.001
}

func splitNonEmpty(v string) []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range strings.Split(v, ",") {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

func pctChangePtr(old, current float64) *float64 {
	if old == 0 {
		return nil
	}
	v := pctChange(old, current)
	return &v
}

func pctChange(old, current float64) float64 {
	if old == 0 {
		return 0
	}
	return (current - old) / old * 100
}

func floatPtr(v float64) *float64 { return &v }

func sha256File(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return sha256Bytes(b)
}

func sha256Bytes(b []byte) (string, error) {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func validSHA256(v string) bool {
	if len(v) != 64 {
		return false
	}
	_, err := hex.DecodeString(v)
	return err == nil
}
