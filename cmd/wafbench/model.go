package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Result struct {
	Mode                string   `json:"mode"`
	Scenario            string   `json:"scenario"`
	StartedAt           string   `json:"started_at"`
	DurationSeconds     float64  `json:"duration_seconds"`
	Workers             int      `json:"workers"`
	GOMAXPROCS          int      `json:"gomaxprocs,omitempty"`
	Requests            uint64   `json:"requests"`
	Successes           uint64   `json:"successes"`
	Errors              uint64   `json:"errors"`
	Interruptions       uint64   `json:"interruptions,omitempty"`
	HTTP2xx             uint64   `json:"http_2xx,omitempty"`
	HTTP3xx             uint64   `json:"http_3xx,omitempty"`
	HTTP4xx             uint64   `json:"http_4xx,omitempty"`
	HTTP5xx             uint64   `json:"http_5xx,omitempty"`
	RPS                 float64  `json:"rps"`
	ErrorRatePercent    float64  `json:"error_rate_percent"`
	P50MS               float64  `json:"p50_ms"`
	P95MS               float64  `json:"p95_ms"`
	P99MS               float64  `json:"p99_ms"`
	AvgRequestBytes     float64  `json:"avg_request_bytes"`
	AvgResponseBytes    float64  `json:"avg_response_bytes"`
	AppMbps             float64  `json:"app_mbps"`
	ProcessCPUPercent   float64  `json:"process_cpu_percent,omitempty"`
	GeneratorCPUPercent float64  `json:"generator_cpu_percent,omitempty"`
	GeneratorCPUCores   float64  `json:"generator_cpu_cores,omitempty"`
	ProcessCPUCores     float64  `json:"process_cpu_cores,omitempty"`
	CPUUSPerRequest     float64  `json:"cpu_us_per_request,omitempty"`
	HostBusyPercent     float64  `json:"host_busy_percent,omitempty"`
	HostSoftIRQPercent  float64  `json:"host_softirq_percent,omitempty"`
	RSSStartMiB         float64  `json:"rss_start_mib,omitempty"`
	RSSEndMiB           float64  `json:"rss_end_mib,omitempty"`
	NetInterface        string   `json:"net_interface,omitempty"`
	NetRXMbps           float64  `json:"net_rx_mbps,omitempty"`
	NetTXMbps           float64  `json:"net_tx_mbps,omitempty"`
	NetRXPPS            float64  `json:"net_rx_pps,omitempty"`
	NetTXPPS            float64  `json:"net_tx_pps,omitempty"`
	ResponseInspection  string   `json:"response_inspection,omitempty"`
	EngineMode          string   `json:"engine_mode,omitempty"`
	AuditEnabled        bool     `json:"audit_enabled,omitempty"`
	Notes               []string `json:"notes,omitempty"`
}

type SystemInfo struct {
	Hostname  string `json:"hostname,omitempty"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	GoVersion string `json:"go_version"`
	NumCPU    int    `json:"num_cpu"`
	CPUModel  string `json:"cpu_model,omitempty"`
	Kernel    string `json:"kernel,omitempty"`
}

type ResultFile struct {
	ToolVersion string     `json:"tool_version"`
	GeneratedAt string     `json:"generated_at"`
	System      SystemInfo `json:"system"`
	Results     []Result   `json:"results"`
}

const toolVersion = "1.0.0"

func collectSystemInfo() SystemInfo {
	host, _ := os.Hostname()
	model, kernel := platformDetails()
	return SystemInfo{Hostname: host, OS: runtime.GOOS, Arch: runtime.GOARCH, GoVersion: runtime.Version(), NumCPU: runtime.NumCPU(), CPUModel: model, Kernel: kernel}
}

func writeResults(results []Result, format, out string) error {
	var w io.Writer = os.Stdout
	var f *os.File
	var err error
	if out != "" {
		f, err = os.Create(out)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}

	switch strings.ToLower(format) {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(ResultFile{ToolVersion: toolVersion, GeneratedAt: time.Now().Format(time.RFC3339), System: collectSystemInfo(), Results: results})
	case "csv":
		cw := csv.NewWriter(w)
		defer cw.Flush()
		header := []string{"mode", "scenario", "workers", "requests", "rps", "error_%", "p50_ms", "p95_ms", "p99_ms", "req_B", "resp_B", "app_Mbps", "proc_CPU_%", "CPU_us_op", "softirq_%", "rx_Mbps", "tx_Mbps", "rx_pps", "tx_pps"}
		if err := cw.Write(header); err != nil {
			return err
		}
		for _, r := range results {
			row := []string{
				r.Mode, r.Scenario, strconv.Itoa(r.Workers), strconv.FormatUint(r.Requests, 10),
				fmt.Sprintf("%.2f", r.RPS), fmt.Sprintf("%.3f", r.ErrorRatePercent),
				fmt.Sprintf("%.3f", r.P50MS), fmt.Sprintf("%.3f", r.P95MS), fmt.Sprintf("%.3f", r.P99MS),
				fmt.Sprintf("%.0f", r.AvgRequestBytes), fmt.Sprintf("%.0f", r.AvgResponseBytes), fmt.Sprintf("%.2f", r.AppMbps),
				fmt.Sprintf("%.2f", r.ProcessCPUPercent), fmt.Sprintf("%.3f", r.CPUUSPerRequest),
				fmt.Sprintf("%.3f", r.HostSoftIRQPercent), fmt.Sprintf("%.2f", r.NetRXMbps), fmt.Sprintf("%.2f", r.NetTXMbps),
				fmt.Sprintf("%.0f", r.NetRXPPS), fmt.Sprintf("%.0f", r.NetTXPPS),
			}
			if err := cw.Write(row); err != nil {
				return err
			}
		}
		return cw.Error()
	case "table", "":
		printTable(w, results)
		if out != "" {
			printTable(os.Stdout, results)
		}
		return nil
	default:
		return fmt.Errorf("unknown output format %q (table|json|csv)", format)
	}
}

func printTable(w io.Writer, results []Result) {
	fmt.Fprintf(w, "%-8s %-18s %7s %10s %9s %8s %8s %8s %10s %9s\n", "MODE", "SCENARIO", "WORKERS", "RPS", "ERR%", "P50ms", "P95ms", "P99ms", "APP Mbps", "CPU us/op")
	for _, r := range results {
		fmt.Fprintf(w, "%-8s %-18s %7d %10.0f %9.3f %8.3f %8.3f %8.3f %10.1f %9.3f\n",
			r.Mode, r.Scenario, r.Workers, r.RPS, r.ErrorRatePercent, r.P50MS, r.P95MS, r.P99MS, r.AppMbps, r.CPUUSPerRequest)
	}
}

func loadResultFile(path string) (ResultFile, error) {
	var rf ResultFile
	b, err := os.ReadFile(path)
	if err != nil {
		return rf, err
	}
	if err := json.Unmarshal(b, &rf); err != nil {
		return rf, err
	}
	return rf, nil
}

func sortedScenarios(m map[string]Scenario) []string {
	names := make([]string, 0, len(m))
	for k := range m {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}
