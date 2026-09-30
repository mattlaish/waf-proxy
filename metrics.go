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
	"crypto/tls"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type metricSample struct {
	T                  int64   `json:"t"` // unix seconds
	ReqPerSec          float64 `json:"req_per_sec"`
	InBps              float64 `json:"in_bps"`         // bytes/sec client->waf
	OutBps             float64 `json:"out_bps"`        // bytes/sec waf->client
	TLSHsPerSec        float64 `json:"tls_hs_per_sec"` // backward-compatible: ClientHello attempts/sec
	TLSSuccessPerSec   float64 `json:"tls_success_per_sec"`
	TLSFullPerSec      float64 `json:"tls_full_per_sec"`
	TLSResumedPerSec   float64 `json:"tls_resumed_per_sec"`
	TLSResumePct       float64 `json:"tls_resume_pct"`
	TLSPolicyRejectSec float64 `json:"tls_policy_rejected_per_sec"`
	TLS12PerSec        float64 `json:"tls_12_per_sec"`
	TLS13PerSec        float64 `json:"tls_13_per_sec"`
	TLSECDSAPerSec     float64 `json:"tls_ecdsa_per_sec"`
	TLSRSAPerSec       float64 `json:"tls_rsa_per_sec"`
	TLSEd25519PerSec   float64 `json:"tls_ed25519_per_sec"`
	TLSHandshakeAvgMS  float64 `json:"tls_handshake_avg_ms"`
	TLSHandshakeP95MS  float64 `json:"tls_handshake_p95_ms"`
	CPUPct             float64 `json:"cpu_pct"` // process CPU %
	MemMB              float64 `json:"mem_mb"`  // process RSS MB
	HeapMB             float64 `json:"heap_mb"` // Go heap in-use MB
	Goroutines         int     `json:"goroutines"`
}

type metrics struct {
	// monotonic counters (touched on the hot path)
	reqs              atomic.Int64
	bytesIn           atomic.Int64
	bytesOut          atomic.Int64
	tlsHS             atomic.Int64
	tlsSuccess        atomic.Int64
	tlsFull           atomic.Int64
	tlsResumed        atomic.Int64
	tlsPolicyRejected atomic.Int64
	tls12             atomic.Int64
	tls13             atomic.Int64
	tlsECDSA          atomic.Int64
	tlsRSA            atomic.Int64
	tlsEd25519        atomic.Int64
	tlsOtherCert      atomic.Int64
	tlsLatencyNS      atomic.Int64
	tlsLatencyBuckets [10]atomic.Int64

	mu      sync.Mutex
	hist    []metricSample
	maxHist int

	// last-sample state for rate/delta computation
	lastReqs, lastIn, lastOut, lastHS                                  int64
	lastTLSSuccess, lastTLSFull, lastTLSResumed, lastTLSPolicyRejected int64
	lastTLS12, lastTLS13, lastTLSECDSA, lastTLSRSA, lastTLSEd25519     int64
	lastTLSLatencyNS                                                   int64
	lastTLSLatencyBuckets                                              [10]int64
	lastProcCPU, lastTotalCPU                                          float64
	lastT                                                              time.Time
}

func newMetrics() *metrics { return &metrics{maxHist: 120} }

func (m *metrics) addReq() { m.reqs.Add(1) }
func (m *metrics) addIn(n int64) {
	if n > 0 {
		m.bytesIn.Add(n)
	}
}
func (m *metrics) addOut(n int64) {
	if n > 0 {
		m.bytesOut.Add(n)
	}
}
func (m *metrics) addTLSHandshake()        { m.addTLSHandshakeAttempt() } // compatibility for older call sites/tests
func (m *metrics) addTLSHandshakeAttempt() { m.tlsHS.Add(1) }
func (m *metrics) addTLSPolicyRejected()   { m.tlsPolicyRejected.Add(1) }

var tlsHandshakeLatencyBounds = [...]time.Duration{
	1 * time.Millisecond, 2 * time.Millisecond, 5 * time.Millisecond, 10 * time.Millisecond, 20 * time.Millisecond,
	50 * time.Millisecond, 100 * time.Millisecond, 250 * time.Millisecond, 500 * time.Millisecond, time.Second,
}

func (m *metrics) observeTLSHandshake(cs tls.ConnectionState, certAlgorithm string, d time.Duration) {
	m.tlsSuccess.Add(1)
	if cs.DidResume {
		m.tlsResumed.Add(1)
	} else {
		m.tlsFull.Add(1)
		switch certAlgorithm {
		case "ecdsa":
			m.tlsECDSA.Add(1)
		case "rsa":
			m.tlsRSA.Add(1)
		case "ed25519":
			m.tlsEd25519.Add(1)
		default:
			m.tlsOtherCert.Add(1)
		}
	}
	switch cs.Version {
	case tls.VersionTLS12:
		m.tls12.Add(1)
	case tls.VersionTLS13:
		m.tls13.Add(1)
	}
	if d < 0 {
		d = 0
	}
	m.tlsLatencyNS.Add(d.Nanoseconds())
	for i, upper := range tlsHandshakeLatencyBounds {
		if d <= upper {
			m.tlsLatencyBuckets[i].Add(1)
			return
		}
	}
}

// sample computes one point from the deltas since the previous call.
func (m *metrics) sample() metricSample {
	now := time.Now()
	reqs, in, out, hs := m.reqs.Load(), m.bytesIn.Load(), m.bytesOut.Load(), m.tlsHS.Load()
	tlsSuccess, tlsFull, tlsResumed := m.tlsSuccess.Load(), m.tlsFull.Load(), m.tlsResumed.Load()
	tlsRejected := m.tlsPolicyRejected.Load()
	tls12, tls13 := m.tls12.Load(), m.tls13.Load()
	tlsECDSA, tlsRSA, tlsEd25519 := m.tlsECDSA.Load(), m.tlsRSA.Load(), m.tlsEd25519.Load()
	tlsLatencyNS := m.tlsLatencyNS.Load()
	var tlsLatencyBuckets [10]int64
	for i := range tlsLatencyBuckets {
		tlsLatencyBuckets[i] = m.tlsLatencyBuckets[i].Load()
	}

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

	tlsSuccessDelta := tlsSuccess - m.lastTLSSuccess
	tlsResumedDelta := tlsResumed - m.lastTLSResumed
	resumePct := 0.0
	if tlsSuccessDelta > 0 {
		resumePct = 100 * float64(tlsResumedDelta) / float64(tlsSuccessDelta)
	}
	latencyAvgMS := 0.0
	if tlsSuccessDelta > 0 {
		latencyAvgMS = float64(tlsLatencyNS-m.lastTLSLatencyNS) / float64(tlsSuccessDelta) / float64(time.Millisecond)
	}
	latencyP95MS := percentileTLSLatencyMS(tlsLatencyBuckets, m.lastTLSLatencyBuckets, tlsSuccessDelta, 0.95)

	s := metricSample{
		T:                  now.Unix(),
		ReqPerSec:          rate(reqs, m.lastReqs),
		InBps:              rate(in, m.lastIn),
		OutBps:             rate(out, m.lastOut),
		TLSHsPerSec:        rate(hs, m.lastHS),
		TLSSuccessPerSec:   rate(tlsSuccess, m.lastTLSSuccess),
		TLSFullPerSec:      rate(tlsFull, m.lastTLSFull),
		TLSResumedPerSec:   rate(tlsResumed, m.lastTLSResumed),
		TLSResumePct:       resumePct,
		TLSPolicyRejectSec: rate(tlsRejected, m.lastTLSPolicyRejected),
		TLS12PerSec:        rate(tls12, m.lastTLS12),
		TLS13PerSec:        rate(tls13, m.lastTLS13),
		TLSECDSAPerSec:     rate(tlsECDSA, m.lastTLSECDSA),
		TLSRSAPerSec:       rate(tlsRSA, m.lastTLSRSA),
		TLSEd25519PerSec:   rate(tlsEd25519, m.lastTLSEd25519),
		TLSHandshakeAvgMS:  latencyAvgMS,
		TLSHandshakeP95MS:  latencyP95MS,
		CPUPct:             clampF(cpuPct, 0, 100*float64(runtime.NumCPU())),
		MemMB:              readRSSMB(),
		HeapMB:             float64(ms.HeapInuse) / (1 << 20),
		Goroutines:         runtime.NumGoroutine(),
	}

	m.lastReqs, m.lastIn, m.lastOut, m.lastHS = reqs, in, out, hs
	m.lastTLSSuccess, m.lastTLSFull, m.lastTLSResumed, m.lastTLSPolicyRejected = tlsSuccess, tlsFull, tlsResumed, tlsRejected
	m.lastTLS12, m.lastTLS13, m.lastTLSECDSA, m.lastTLSRSA, m.lastTLSEd25519 = tls12, tls13, tlsECDSA, tlsRSA, tlsEd25519
	m.lastTLSLatencyNS = tlsLatencyNS
	m.lastTLSLatencyBuckets = tlsLatencyBuckets
	m.lastProcCPU, m.lastTotalCPU, m.lastT = procCPU, totalCPU, now

	m.mu.Lock()
	m.hist = append(m.hist, s)
	if len(m.hist) > m.maxHist {
		m.hist = m.hist[len(m.hist)-m.maxHist:]
	}
	m.mu.Unlock()
	return s
}

func percentileTLSLatencyMS(cur, prev [10]int64, total int64, percentile float64) float64 {
	if total <= 0 {
		return 0
	}
	target := int64(float64(total)*percentile + 0.999999)
	if target < 1 {
		target = 1
	}
	var seen int64
	for i, bound := range tlsHandshakeLatencyBounds {
		seen += cur[i] - prev[i]
		if seen >= target {
			return float64(bound) / float64(time.Millisecond)
		}
	}
	return float64(tlsHandshakeLatencyBounds[len(tlsHandshakeLatencyBounds)-1]) / float64(time.Millisecond)
}

func (m *metrics) tlsStatus() map[string]any {
	success := m.tlsSuccess.Load()
	resumed := m.tlsResumed.Load()
	ratio := 0.0
	if success > 0 {
		ratio = 100 * float64(resumed) / float64(success)
	}
	avgMS := 0.0
	if success > 0 {
		avgMS = float64(m.tlsLatencyNS.Load()) / float64(success) / float64(time.Millisecond)
	}
	var cur [10]int64
	for i := range cur {
		cur[i] = m.tlsLatencyBuckets[i].Load()
	}
	return map[string]any{
		"attempts":        m.tlsHS.Load(),
		"successful":      success,
		"full":            m.tlsFull.Load(),
		"resumed":         resumed,
		"resume_pct":      ratio,
		"policy_rejected": m.tlsPolicyRejected.Load(),
		"versions":        map[string]int64{"tls_1_2": m.tls12.Load(), "tls_1_3": m.tls13.Load()},
		"full_handshake_certificate_key_algorithms": map[string]int64{
			"ecdsa": m.tlsECDSA.Load(), "rsa": m.tlsRSA.Load(), "ed25519": m.tlsEd25519.Load(), "other_or_unknown": m.tlsOtherCert.Load(),
		},
		"handshake_processing_latency_ms": map[string]float64{
			"avg": avgMS, "p95_bucket_upper_bound": percentileTLSLatencyMS(cur, [10]int64{}, success, 0.95),
		},
	}
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
