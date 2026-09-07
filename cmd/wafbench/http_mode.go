package main

import (
	"bytes"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"sync"
	"time"
)

type workerResult struct {
	requests      uint64
	successes     uint64
	errors        uint64
	interruptions uint64
	reqBytes      uint64
	respBytes     uint64
	status        [5]uint64
	hist          latencyHist
}

func runHTTP(args []string) error {
	fs := flag.NewFlagSet("http", flag.ContinueOnError)
	target := fs.String("target", "", "base URL of running waf-proxy, e.g. https://127.0.0.1:8443")
	host := fs.String("host", "", "Host header / virtual host")
	serverName := fs.String("server-name", "", "TLS SNI/server name; defaults to --host when set")
	scenario := fs.String("scenario", "all", "scenario name, comma list, or all")
	duration := fs.Duration("duration", 15*time.Second, "measurement duration per scenario")
	warmup := fs.Duration("warmup", 3*time.Second, "warm-up duration per scenario")
	concurrency := fs.Int("concurrency", 64, "concurrent workers")
	gomax := fs.Int("gomaxprocs", 0, "load-generator GOMAXPROCS; 0 leaves runtime default")
	timeout := fs.Duration("timeout", 10*time.Second, "per-request timeout")
	insecure := fs.Bool("insecure", false, "skip TLS certificate verification (benchmark/lab only)")
	disableHTTP2 := fs.Bool("disable-http2", false, "disable HTTP/2 on the load generator")
	pid := fs.Int("pid", 0, "waf-proxy PID for process CPU/RSS sampling (Linux)")
	iface := fs.String("iface", "", "network interface to sample from /proc/net/dev (Linux)")
	format := fs.String("format", "table", "table|json|csv")
	out := fs.String("out", "", "write results to file (defaults to JSON when set)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *target == "" {
		return fmt.Errorf("--target is required")
	}
	if *concurrency < 1 {
		return fmt.Errorf("--concurrency must be >= 1")
	}
	if *duration <= 0 {
		return fmt.Errorf("--duration must be > 0")
	}
	if *gomax < 0 {
		return fmt.Errorf("--gomaxprocs must be >= 0")
	}
	if *gomax > 0 {
		old := runtime.GOMAXPROCS(*gomax)
		defer runtime.GOMAXPROCS(old)
	}
	base, err := url.Parse(*target)
	if err != nil {
		return err
	}
	if base.Scheme != "http" && base.Scheme != "https" {
		return fmt.Errorf("target scheme must be http or https")
	}
	list, err := chooseScenarios(*scenario, 0)
	if err != nil {
		return err
	}

	tlsServerName := *serverName
	if tlsServerName == "" && *host != "" {
		tlsServerName = *host
		if h, _, splitErr := net.SplitHostPort(*host); splitErr == nil {
			tlsServerName = h
		}
	}
	transport := &http.Transport{
		Proxy:                 nil,
		MaxIdleConns:          max(256, *concurrency*4),
		MaxIdleConnsPerHost:   max(256, *concurrency*2),
		MaxConnsPerHost:       0,
		IdleConnTimeout:       90 * time.Second,
		ForceAttemptHTTP2:     !*disableHTTP2,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: *insecure, ServerName: tlsServerName}, // #nosec G402 -- explicit lab flag
		ResponseHeaderTimeout: *timeout,
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: *timeout}

	results := make([]Result, 0, len(list))
	for _, sc := range list {
		if *warmup > 0 {
			_, _ = runHTTPOne(client, base, *host, sc, *concurrency, *warmup, 0, "")
			transport.CloseIdleConnections()
			time.Sleep(100 * time.Millisecond)
		}
		r, err := runHTTPOne(client, base, *host, sc, *concurrency, *duration, *pid, *iface)
		if err != nil {
			return err
		}
		results = append(results, r)
	}
	if *out != "" && *format == "table" {
		*format = "json"
	}
	return writeResults(results, *format, *out)
}

func runHTTPOne(client *http.Client, base *url.URL, host string, sc Scenario, concurrency int, duration time.Duration, pid int, iface string) (Result, error) {
	deadline := time.Now().Add(duration)
	before, err := sampleProc(pid, iface)
	if err != nil && pid > 0 {
		return Result{}, err
	}
	selfBefore, _ := sampleProc(os.Getpid(), "")
	start := time.Now()
	results := make(chan workerResult, concurrency)
	var wg sync.WaitGroup
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			var wr workerResult
			for time.Now().Before(deadline) {
				u := *base
				rel, parseErr := url.Parse(sc.URI)
				if parseErr != nil {
					wr.errors++
					break
				}
				u.Path = rel.Path
				u.RawPath = rel.RawPath
				u.RawQuery = rel.RawQuery
				var body io.Reader
				if len(sc.Body) > 0 {
					body = bytes.NewReader(sc.Body)
				}
				req, reqErr := http.NewRequest(sc.Method, u.String(), body)
				if reqErr != nil {
					wr.errors++
					continue
				}
				if host != "" {
					req.Host = host
				}
				if sc.ContentType != "" {
					req.Header.Set("Content-Type", sc.ContentType)
				}
				req.Header.Set("User-Agent", "wafbench/"+toolVersion)
				req.Header.Set("Accept", "application/json,text/html;q=0.9,*/*;q=0.1")
				t0 := time.Now()
				resp, doErr := client.Do(req)
				elapsed := time.Since(t0)
				wr.requests++
				wr.reqBytes += uint64(len(sc.Body))
				wr.hist.addNs(elapsed.Nanoseconds())
				if doErr != nil {
					wr.errors++
					continue
				}
				switch {
				case resp.StatusCode >= 200 && resp.StatusCode < 300:
					wr.status[2]++
				case resp.StatusCode >= 300 && resp.StatusCode < 400:
					wr.status[3]++
				case resp.StatusCode >= 400 && resp.StatusCode < 500:
					wr.status[4]++
				case resp.StatusCode >= 500:
					wr.status[0]++
				}
				n, copyErr := io.Copy(io.Discard, resp.Body)
				closeErr := resp.Body.Close()
				wr.respBytes += uint64(max64(n, 0))
				if copyErr != nil || closeErr != nil || resp.StatusCode >= 500 {
					wr.errors++
				} else {
					wr.successes++
				}
			}
			results <- wr
		}()
	}
	wg.Wait()
	close(results)
	elapsed := time.Since(start)
	after, err := sampleProc(pid, iface)
	if err != nil && pid > 0 {
		return Result{}, err
	}
	selfAfter, _ := sampleProc(os.Getpid(), "")
	var agg workerResult
	for wr := range results {
		agg.requests += wr.requests
		agg.successes += wr.successes
		agg.errors += wr.errors
		agg.reqBytes += wr.reqBytes
		agg.respBytes += wr.respBytes
		agg.hist.merge(&wr.hist)
		for i := range agg.status {
			agg.status[i] += wr.status[i]
		}
	}
	sec := elapsed.Seconds()
	r := Result{
		Mode: "http", Scenario: sc.Name, StartedAt: start.Format(time.RFC3339), DurationSeconds: sec,
		Workers: concurrency, GOMAXPROCS: runtime.GOMAXPROCS(0), Requests: agg.requests, Successes: agg.successes, Errors: agg.errors,
		HTTP2xx: agg.status[2], HTTP3xx: agg.status[3], HTTP4xx: agg.status[4], HTTP5xx: agg.status[0], NetInterface: iface,
	}
	if agg.requests > 0 {
		r.RPS = float64(agg.requests) / sec
		r.ErrorRatePercent = float64(agg.errors) / float64(agg.requests) * 100
		r.AvgRequestBytes = float64(agg.reqBytes) / float64(agg.requests)
		r.AvgResponseBytes = float64(agg.respBytes) / float64(agg.requests)
		r.AppMbps = float64(agg.reqBytes+agg.respBytes) * 8 / sec / 1e6
	}
	r.P50MS = agg.hist.quantile(.50)
	r.P95MS = agg.hist.quantile(.95)
	r.P99MS = agg.hist.quantile(.99)
	enrichProc(&r, before, after, sec)
	var gen Result
	gen.Requests = 1
	gen.RPS = 1
	enrichProc(&gen, selfBefore, selfAfter, sec)
	r.GeneratorCPUPercent = gen.ProcessCPUPercent
	r.GeneratorCPUCores = gen.ProcessCPUCores
	if pid == 0 {
		r.Notes = append(r.Notes, "--pid not supplied: waf-proxy CPU/RSS and CPU-us/request are unavailable")
	}
	if iface == "" {
		r.Notes = append(r.Notes, "--iface not supplied: NIC Mbps/PPS counters are unavailable")
	}
	if r.GeneratorCPUCores >= float64(r.GOMAXPROCS)*0.80 {
		r.Notes = append(r.Notes, "load generator is near host CPU saturation; use a separate generator host or more CPUs")
	}
	return r, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
