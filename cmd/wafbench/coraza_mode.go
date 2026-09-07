package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/corazawaf/coraza/v3"
)

func runCoraza(args []string) error {
	fs := flag.NewFlagSet("coraza", flag.ContinueOnError)
	rules := fs.String("rules", "", "SecLang/Coraza config file (for this project normally /etc/waf/coraza.conf)")
	scenario := fs.String("scenario", "all", "scenario name, comma list, or all")
	duration := fs.Duration("duration", 10*time.Second, "measurement duration per scenario")
	warmup := fs.Duration("warmup", 2*time.Second, "warm-up duration per scenario")
	workers := fs.Int("workers", runtime.NumCPU(), "parallel transaction workers")
	gomax := fs.Int("gomaxprocs", runtime.NumCPU(), "runtime GOMAXPROCS during benchmark")
	responseInspection := fs.String("response-inspection", "inherit", "inherit|on|off")
	responseLimit := fs.Int("response-limit", 0, "optional SecResponseBodyLimit override")
	responseBytes := fs.Int("response-bytes", 0, "response body size override; 0 uses scenario defaults")
	engine := fs.String("engine", "DetectionOnly", "inherit|On|DetectionOnly|Off")
	audit := fs.Bool("audit", false, "leave audit logging enabled; default false disables audit I/O for CPU isolation")
	cpuProfile := fs.String("cpuprofile", "", "write Go CPU profile; if multiple scenarios, scenario is appended")
	heapProfile := fs.String("heapprofile", "", "write Go heap profile after each scenario; scenario is appended when needed")
	format := fs.String("format", "table", "table|json|csv")
	out := fs.String("out", "", "write results to file (defaults to JSON when set)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *rules == "" {
		return fmt.Errorf("--rules is required")
	}
	if *workers < 1 || *gomax < 1 {
		return fmt.Errorf("--workers and --gomaxprocs must be >= 1")
	}
	if *duration <= 0 {
		return fmt.Errorf("--duration must be > 0")
	}
	if *responseLimit < 0 {
		return fmt.Errorf("--response-limit must be >= 0")
	}
	switch strings.ToLower(*responseInspection) {
	case "inherit", "on", "off":
	default:
		return fmt.Errorf("invalid --response-inspection %q", *responseInspection)
	}
	switch strings.ToLower(*engine) {
	case "inherit", "on", "detectiononly", "off":
	default:
		return fmt.Errorf("invalid --engine %q", *engine)
	}
	list, err := chooseScenarios(*scenario, *responseBytes)
	if err != nil {
		return err
	}

	oldGMP := runtime.GOMAXPROCS(*gomax)
	defer runtime.GOMAXPROCS(oldGMP)

	cfg := coraza.NewWAFConfig().WithDirectivesFromFile(*rules)
	var overrides strings.Builder
	if !*audit {
		overrides.WriteString("SecAuditEngine Off\n")
	}
	if !strings.EqualFold(*engine, "inherit") {
		fmt.Fprintf(&overrides, "SecRuleEngine %s\n", canonicalEngine(*engine))
	}
	switch strings.ToLower(*responseInspection) {
	case "on":
		overrides.WriteString("SecResponseBodyAccess On\n")
	case "off":
		overrides.WriteString("SecResponseBodyAccess Off\n")
	}
	if *responseLimit > 0 {
		fmt.Fprintf(&overrides, "SecResponseBodyLimit %d\n", *responseLimit)
	}
	if overrides.Len() > 0 {
		cfg = cfg.WithDirectives(overrides.String())
	}
	waf, err := coraza.NewWAF(cfg)
	if err != nil {
		return fmt.Errorf("compile WAF: %w", err)
	}

	results := make([]Result, 0, len(list))
	for _, sc := range list {
		if *warmup > 0 {
			_, _ = runCorazaOne(waf, sc, *workers, *warmup, "", "")
		}
		cp := profilePath(*cpuProfile, sc.Name, len(list))
		hp := profilePath(*heapProfile, sc.Name, len(list))
		r, err := runCorazaOne(waf, sc, *workers, *duration, cp, hp)
		if err != nil {
			return err
		}
		r.ResponseInspection = strings.ToLower(*responseInspection)
		r.EngineMode = canonicalEngine(*engine)
		r.AuditEnabled = *audit
		if !*audit {
			r.Notes = append(r.Notes, "SecAuditEngine Off override used to isolate WAF inspection CPU from audit-log I/O")
		}
		results = append(results, r)
	}
	if *out != "" && *format == "table" {
		*format = "json"
	}
	return writeResults(results, *format, *out)
}

func canonicalEngine(v string) string {
	switch strings.ToLower(v) {
	case "on":
		return "On"
	case "off":
		return "Off"
	case "detectiononly":
		return "DetectionOnly"
	default:
		return "inherit"
	}
}

func profilePath(base, scenario string, count int) string {
	if base == "" {
		return ""
	}
	if count <= 1 {
		return base
	}
	ext := ""
	stem := base
	if i := strings.LastIndex(base, "."); i >= 0 {
		stem, ext = base[:i], base[i:]
	}
	return stem + "-" + scenario + ext
}

func runCorazaOne(waf coraza.WAF, sc Scenario, workers int, duration time.Duration, cpuProfile, heapProfile string) (Result, error) {
	var cpuFile *os.File
	var err error
	if cpuProfile != "" {
		cpuFile, err = os.Create(cpuProfile)
		if err != nil {
			return Result{}, err
		}
		if err := pprof.StartCPUProfile(cpuFile); err != nil {
			cpuFile.Close()
			return Result{}, err
		}
		defer func() { pprof.StopCPUProfile(); _ = cpuFile.Close() }()
	}
	before, _ := sampleProc(os.Getpid(), "")
	start := time.Now()
	deadline := start.Add(duration)
	out := make(chan workerResult, workers)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(worker int) {
			defer wg.Done()
			var wr workerResult
			clientIP := "198.51.100." + strconv.Itoa(10+(worker%200))
			for time.Now().Before(deadline) {
				t0 := time.Now()
				interrupted, err := processCorazaTransaction(waf, sc, clientIP)
				wr.hist.addNs(time.Since(t0).Nanoseconds())
				wr.requests++
				wr.reqBytes += uint64(len(sc.Body))
				wr.respBytes += uint64(len(sc.ResponseBody))
				if err != nil {
					wr.errors++
				} else {
					wr.successes++
					if interrupted {
						wr.interruptions++
					}
				}
			}
			out <- wr
		}(i)
	}
	wg.Wait()
	close(out)
	elapsed := time.Since(start)
	after, _ := sampleProc(os.Getpid(), "")
	var agg workerResult
	for wr := range out {
		agg.requests += wr.requests
		agg.successes += wr.successes
		agg.errors += wr.errors
		agg.interruptions += wr.interruptions
		agg.reqBytes += wr.reqBytes
		agg.respBytes += wr.respBytes
		agg.hist.merge(&wr.hist)
	}
	sec := elapsed.Seconds()
	r := Result{Mode: "coraza", Scenario: sc.Name, StartedAt: start.Format(time.RFC3339), DurationSeconds: sec, Workers: workers, GOMAXPROCS: runtime.GOMAXPROCS(0), Requests: agg.requests, Successes: agg.successes, Errors: agg.errors, Interruptions: agg.interruptions}
	if agg.requests > 0 {
		r.RPS = float64(agg.requests) / sec
		r.ErrorRatePercent = float64(agg.errors) / float64(agg.requests) * 100
		r.AvgRequestBytes = float64(agg.reqBytes) / float64(agg.requests)
		r.AvgResponseBytes = float64(agg.respBytes) / float64(agg.requests)
		r.AppMbps = float64(agg.reqBytes+agg.respBytes) * 8 / sec / 1e6
	}
	r.P50MS = agg.hist.quantile(.5)
	r.P95MS = agg.hist.quantile(.95)
	r.P99MS = agg.hist.quantile(.99)
	enrichProc(&r, before, after, sec)
	if heapProfile != "" {
		f, err := os.Create(heapProfile)
		if err != nil {
			return r, err
		}
		runtime.GC()
		err = pprof.WriteHeapProfile(f)
		_ = f.Close()
		if err != nil {
			return r, err
		}
	}
	return r, nil
}

func processCorazaTransaction(waf coraza.WAF, sc Scenario, clientIP string) (bool, error) {
	tx := waf.NewTransaction()
	defer tx.Close()
	defer tx.ProcessLogging()
	tx.ProcessConnection(clientIP, 54321, "127.0.0.1", 8443)
	tx.ProcessURI(sc.URI, sc.Method, "HTTP/1.1")
	tx.AddRequestHeader("Host", "bench.local")
	tx.SetServerName("bench.local")
	tx.AddRequestHeader("User-Agent", "wafbench/"+toolVersion)
	if sc.ContentType != "" {
		tx.AddRequestHeader("Content-Type", sc.ContentType)
	}
	if len(sc.Body) > 0 {
		tx.AddRequestHeader("Content-Length", strconv.Itoa(len(sc.Body)))
	}
	if it := tx.ProcessRequestHeaders(); it != nil {
		return true, nil
	}
	if len(sc.Body) > 0 && tx.IsRequestBodyAccessible() {
		it, _, err := tx.WriteRequestBody(sc.Body)
		if err != nil {
			return false, err
		}
		if it != nil {
			return true, nil
		}
	}
	if it, err := tx.ProcessRequestBody(); err != nil {
		return false, err
	} else if it != nil {
		return true, nil
	}
	tx.AddResponseHeader("Content-Type", sc.ResponseContentType)
	tx.AddResponseHeader("Content-Length", strconv.Itoa(len(sc.ResponseBody)))
	if it := tx.ProcessResponseHeaders(200, "HTTP/1.1"); it != nil {
		return true, nil
	}
	if len(sc.ResponseBody) > 0 && tx.IsResponseBodyAccessible() && tx.IsResponseBodyProcessable() {
		it, _, err := tx.WriteResponseBody(sc.ResponseBody)
		if err != nil {
			return false, err
		}
		if it != nil {
			return true, nil
		}
	}
	if it, err := tx.ProcessResponseBody(); err != nil {
		return false, err
	} else if it != nil {
		return true, nil
	}
	return false, nil
}
