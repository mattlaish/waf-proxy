package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"os"
	"runtime"
	"sync"
	"time"
)

func runL4(args []string) error {
	fs := flag.NewFlagSet("l4", flag.ContinueOnError)
	target := fs.String("target", "", "TCP host:port of the WAF listener")
	useTLS := fs.Bool("tls", false, "perform a TLS handshake before close")
	serverName := fs.String("server-name", "", "TLS server name/SNI")
	insecure := fs.Bool("insecure", false, "skip TLS certificate verification (lab only)")
	concurrency := fs.Int("concurrency", 128, "parallel connection workers")
	gomax := fs.Int("gomaxprocs", 0, "load-generator GOMAXPROCS; 0 leaves runtime default")
	duration := fs.Duration("duration", 10*time.Second, "measurement duration")
	warmup := fs.Duration("warmup", 2*time.Second, "warm-up duration")
	timeout := fs.Duration("timeout", 3*time.Second, "connect/handshake timeout")
	pid := fs.Int("pid", 0, "waf-proxy PID for process CPU/RSS sampling (Linux)")
	iface := fs.String("iface", "", "network interface for Mbps/PPS sampling (Linux)")
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
	if *warmup > 0 {
		_, _ = runL4One(*target, *useTLS, *serverName, *insecure, *concurrency, *warmup, *timeout, 0, "")
	}
	r, err := runL4One(*target, *useTLS, *serverName, *insecure, *concurrency, *duration, *timeout, *pid, *iface)
	if err != nil {
		return err
	}
	if *out != "" && *format == "table" {
		*format = "json"
	}
	return writeResults([]Result{r}, *format, *out)
}

func runL4One(target string, useTLS bool, serverName string, insecure bool, concurrency int, duration, timeout time.Duration, pid int, iface string) (Result, error) {
	before, err := sampleProc(pid, iface)
	if err != nil && pid > 0 {
		return Result{}, err
	}
	selfBefore, _ := sampleProc(os.Getpid(), "")
	start := time.Now()
	deadline := start.Add(duration)
	results := make(chan workerResult, concurrency)
	var wg sync.WaitGroup
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			var wr workerResult
			d := net.Dialer{Timeout: timeout, KeepAlive: -1}
			for time.Now().Before(deadline) {
				t0 := time.Now()
				c, e := d.Dial("tcp", target)
				if e == nil && useTLS {
					tc := tls.Client(c, &tls.Config{InsecureSkipVerify: insecure, ServerName: serverName}) // #nosec G402 -- explicit lab flag
					_ = tc.SetDeadline(time.Now().Add(timeout))
					e = tc.Handshake()
					c = tc
				}
				wr.hist.addNs(time.Since(t0).Nanoseconds())
				wr.requests++
				if e != nil {
					wr.errors++
					if c != nil {
						_ = c.Close()
					}
					continue
				}
				wr.successes++
				_ = c.Close()
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
		agg.hist.merge(&wr.hist)
	}
	sec := elapsed.Seconds()
	name := "tcp-connect"
	if useTLS {
		name = "tls-handshake"
	}
	r := Result{Mode: "l4", Scenario: name, StartedAt: start.Format(time.RFC3339), DurationSeconds: sec, Workers: concurrency, GOMAXPROCS: runtime.GOMAXPROCS(0), Requests: agg.requests, Successes: agg.successes, Errors: agg.errors, NetInterface: iface}
	if agg.requests > 0 {
		r.RPS = float64(agg.requests) / sec
		r.ErrorRatePercent = float64(agg.errors) / float64(agg.requests) * 100
	}
	r.P50MS = agg.hist.quantile(.5)
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
		r.Notes = append(r.Notes, "--pid not supplied: waf-proxy CPU/RSS and CPU-us/connection are unavailable")
	}
	if iface == "" {
		r.Notes = append(r.Notes, "--iface not supplied: NIC Mbps/PPS counters are unavailable")
	}
	return r, nil
}
