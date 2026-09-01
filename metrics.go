package main

// Real, dependency-free metrics for the dashboard.
//
// Everything here is measured, not synthesised:
//   - requests, bytes in/out, TLS handshakes: atomic counters on the live path
//   - process + system CPU: /proc/self/stat and /proc/stat (plain file reads)
//   - memory: runtime.MemStats (heap) and /proc/meminfo (system)
//
// A sampler ticks every few seconds, turns the monotonic counters into
// per-second rates, and keeps a small rolling history the UI draws as sparklines.
// History is in-memory (resets on restart), which is fine for a live dashboard.
// /proc reads are Linux-only and degrade to zero rather than erroring elsewhere.

import (
	"bufio"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type metricSample struct {
	T          int64   `json:"t"`           // unix seconds
	ReqPerSec  float64 `json:"req_per_sec"`
	InBps      float64 `json:"in_bps"`      // bytes/sec client->waf
	OutBps     float64 `json:"out_bps"`     // bytes/sec waf->client
	TLSHsPerSec float64 `json:"tls_hs_per_sec"`
	CPUPct     float64 `json:"cpu_pct"`     // process CPU %
	MemMB      float64 `json:"mem_mb"`      // process RSS MB
	HeapMB     float64 `json:"heap_mb"`     // Go heap in-use MB
	Goroutines int     `json:"goroutines"`
}

type metrics struct {
	// monotonic counters (touched on the hot path)
	reqs   atomic.Int64
	bytesIn  atomic.Int64
	bytesOut atomic.Int64
	tlsHS  atomic.Int64

	mu      sync.Mutex
	hist    []metricSample
	maxHist int

	// last-sample state for rate/delta computation
	lastReqs, lastIn, lastOut, lastHS int64
	lastProcCPU, lastTotalCPU         float64
	lastT                             time.Time
}

func newMetrics() *metrics { return &metrics{maxHist: 120} }

func (m *metrics) addReq()            { m.reqs.Add(1) }
func (m *metrics) addIn(n int64)      { if n > 0 { m.bytesIn.Add(n) } }
func (m *metrics) addOut(n int64)     { if n > 0 { m.bytesOut.Add(n) } }
func (m *metrics) addTLSHandshake()   { m.tlsHS.Add(1) }

// sample computes one point from the deltas since the previous call.
func (m *metrics) sample() metricSample {
	now := time.Now()
	reqs, in, out, hs := m.reqs.Load(), m.bytesIn.Load(), m.bytesOut.Load(), m.tlsHS.Load()

	var dt float64 = 1
	if !m.lastT.IsZero() {
		dt = now.Sub(m.lastT).Seconds()
	}
	if dt <= 0 {
		dt = 1
	}
	rate := func(cur, prev int64) float64 { return float64(cur-prev) / dt }

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	procCPU, totalCPU := readCPU()
	cpuPct := 0.0
	if m.lastTotalCPU > 0 && totalCPU > m.lastTotalCPU {
		cpuPct = 100 * (procCPU - m.lastProcCPU) / (totalCPU - m.lastTotalCPU) * float64(runtime.NumCPU())
	}

	s := metricSample{
		T:           now.Unix(),
		ReqPerSec:   rate(reqs, m.lastReqs),
		InBps:       rate(in, m.lastIn),
		OutBps:      rate(out, m.lastOut),
		TLSHsPerSec: rate(hs, m.lastHS),
		CPUPct:      clampF(cpuPct, 0, 100*float64(runtime.NumCPU())),
		MemMB:       readRSSMB(),
		HeapMB:      float64(ms.HeapInuse) / (1 << 20),
		Goroutines:  runtime.NumGoroutine(),
	}

	m.lastReqs, m.lastIn, m.lastOut, m.lastHS = reqs, in, out, hs
	m.lastProcCPU, m.lastTotalCPU, m.lastT = procCPU, totalCPU, now

	m.mu.Lock()
	m.hist = append(m.hist, s)
	if len(m.hist) > m.maxHist {
		m.hist = m.hist[len(m.hist)-m.maxHist:]
	}
	m.mu.Unlock()
	return s
}

func (m *metrics) history() []metricSample {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]metricSample, len(m.hist))
	copy(out, m.hist)
	return out
}

func (m *metrics) startSampler(every time.Duration, stop <-chan struct{}) {
	go func() {
		m.sample() // prime last-sample state
		t := time.NewTicker(every)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				m.sample()
			}
		}
	}()
}

// ── /proc helpers (Linux; return 0 elsewhere) ──

// readCPU returns (process jiffies, total system jiffies).
func readCPU() (proc, total float64) {
	// process: /proc/self/stat fields 14 (utime) + 15 (stime)
	if b, err := os.ReadFile("/proc/self/stat"); err == nil {
		// the comm field can contain spaces/parens; split after the closing ')'
		s := string(b)
		if i := strings.LastIndexByte(s, ')'); i >= 0 && i+2 < len(s) {
			fields := strings.Fields(s[i+2:])
			// after ')' the next field is state; utime is field 14 overall =
			// index 11 in this slice (14 - 3), stime is index 12.
			if len(fields) > 12 {
				ut, _ := strconv.ParseFloat(fields[11], 64)
				st, _ := strconv.ParseFloat(fields[12], 64)
				proc = ut + st
			}
		}
	}
	// total: first line of /proc/stat "cpu  u n s idle ..."
	if f, err := os.Open("/proc/stat"); err == nil {
		defer f.Close()
		sc := bufio.NewScanner(f)
		if sc.Scan() {
			fields := strings.Fields(sc.Text())
			if len(fields) > 1 && fields[0] == "cpu" {
				for _, v := range fields[1:] {
					n, _ := strconv.ParseFloat(v, 64)
					total += n
				}
			}
		}
	}
	return proc, total
}

// readRSSMB reads resident set size from /proc/self/statm (pages).
func readRSSMB() float64 {
	b, err := os.ReadFile("/proc/self/statm")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(b))
	if len(fields) < 2 {
		return 0
	}
	rssPages, _ := strconv.ParseFloat(fields[1], 64)
	pageSize := float64(os.Getpagesize())
	return rssPages * pageSize / (1 << 20)
}

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
