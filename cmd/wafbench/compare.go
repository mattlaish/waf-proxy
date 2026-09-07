package main

import (
	"flag"
	"fmt"
	"sort"
	"strings"
)

type Comparison struct {
	Scenario    string
	HTTPCPUUS   float64
	CorazaCPUUS float64
	CorazaShare float64
	SoftIRQ     float64
	PPS         float64
}

func runCompare(args []string) error {
	fs := flag.NewFlagSet("compare", flag.ContinueOnError)
	httpPath := fs.String("http", "", "JSON file from wafbench http")
	corazaPath := fs.String("coraza", "", "JSON file from wafbench coraza")
	l4Path := fs.String("l4", "", "optional JSON file from wafbench l4")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *httpPath == "" || *corazaPath == "" {
		return fmt.Errorf("--http and --coraza are required")
	}
	hf, err := loadResultFile(*httpPath)
	if err != nil {
		return err
	}
	cf, err := loadResultFile(*corazaPath)
	if err != nil {
		return err
	}
	comps := compareResults(hf.Results, cf.Results)
	if len(comps) == 0 {
		return fmt.Errorf("no matching scenarios with CPU-us/request data")
	}
	var l4Results []Result
	if *l4Path != "" {
		lf, err := loadResultFile(*l4Path)
		if err != nil {
			return err
		}
		l4Results = lf.Results
	}

	fmt.Printf("%-18s %12s %12s %11s %10s %10s\n", "SCENARIO", "HTTP us/r", "CORAZA us/r", "CORAZA %", "SOFTIRQ%", "PPS")
	shares := make([]float64, 0, len(comps))
	var maxSoft, maxPPS float64
	for _, c := range comps {
		fmt.Printf("%-18s %12.3f %12.3f %10.1f%% %10.2f %10.0f\n", c.Scenario, c.HTTPCPUUS, c.CorazaCPUUS, c.CorazaShare*100, c.SoftIRQ, c.PPS)
		if c.CorazaShare > 0 && isBaselineScenario(c.Scenario) {
			shares = append(shares, c.CorazaShare)
		}
		if c.SoftIRQ > maxSoft {
			maxSoft = c.SoftIRQ
		}
		if c.PPS > maxPPS {
			maxPPS = c.PPS
		}
	}
	for _, r := range l4Results {
		if r.HostSoftIRQPercent > maxSoft {
			maxSoft = r.HostSoftIRQPercent
		}
		pps := r.NetRXPPS + r.NetTXPPS
		if pps > maxPPS {
			maxPPS = pps
		}
	}
	med := median(shares)
	fmt.Printf("\nHeuristic baseline Coraza share median: %.1f%%\n", med*100)
	fmt.Printf("Max sampled softirq: %.2f%%; max sampled RX+TX packet rate: %.0f PPS\n", maxSoft, maxPPS)
	for _, r := range hf.Results {
		if r.GOMAXPROCS > 0 && r.GeneratorCPUCores >= float64(r.GOMAXPROCS)*0.80 {
			fmt.Printf("Warning: load generator reached %.2f cores of GOMAXPROCS=%d in %s; that scenario may be generator-limited.\n", r.GeneratorCPUCores, r.GOMAXPROCS, r.Scenario)
		}
	}
	regexSignal := med >= 0.35
	xdpSignal := maxSoft >= 5 && maxPPS >= 100000
	switch {
	case regexSignal && xdpSignal:
		fmt.Println("Direction: both signals qualify. For Internet-facing attack resilience, test XDP first; for normal application throughput, test VectorScan/Hyperscan first.")
	case regexSignal:
		fmt.Println("Direction: VectorScan/Hyperscan feasibility is the stronger next experiment (Coraza inspection is >=35% of measured waf-proxy CPU/request).")
	case xdpSignal:
		fmt.Println("Direction: XDP prefilter is the stronger next experiment (host softirq and packet rate are high while Coraza share is below the threshold).")
	default:
		fmt.Println("Direction: no strong XDP-vs-regex signal yet. Inspect TLS/proxy/backend and CPU profiles before a dataplane rewrite.")
	}
	fmt.Println("Note: this is a sizing heuristic, not proof. For CPU-share/XDP decisions run l4/http sampling on the WAF host with --iface and isolate the generator on a disjoint CPU set. A remote generator needs equivalent WAF-host CPU metrics before compare is meaningful.")
	return nil
}

func compareResults(httpResults, corazaResults []Result) []Comparison {
	cm := map[string]Result{}
	for _, r := range corazaResults {
		if r.Mode == "coraza" {
			cm[r.Scenario] = r
		}
	}
	var out []Comparison
	for _, h := range httpResults {
		c, ok := cm[h.Scenario]
		if !ok || h.CPUUSPerRequest <= 0 || c.CPUUSPerRequest <= 0 {
			continue
		}
		share := c.CPUUSPerRequest / h.CPUUSPerRequest
		pps := h.NetRXPPS + h.NetTXPPS
		out = append(out, Comparison{Scenario: h.Scenario, HTTPCPUUS: h.CPUUSPerRequest, CorazaCPUUS: c.CPUUSPerRequest, CorazaShare: share, SoftIRQ: h.HostSoftIRQPercent, PPS: pps})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Scenario < out[j].Scenario })
	return out
}

func median(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	cp := append([]float64(nil), v...)
	sort.Float64s(cp)
	n := len(cp)
	if n%2 == 1 {
		return cp[n/2]
	}
	return (cp[n/2-1] + cp[n/2]) / 2
}

func isBaselineScenario(name string) bool {
	return name == "clean-get" || strings.HasPrefix(name, "json-")
}
