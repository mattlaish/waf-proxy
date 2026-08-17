package main

// Pool engine — the load-balancing data plane behind each site.
//
// Model (borrowed from F5 LTM, since that's the mental model):
//
//	node    = a backend server, addressed by host/IP, reusable across pools
//	member  = a node + port + weight inside a specific pool
//	pool    = a set of members + a load-balancing method + a health monitor
//	site    = a listener (address + hostnames) that forwards to one pool
//
// A pool runtime picks a member per request among the healthy ones, tracks
// in-flight connections for least-connections balancing, and runs an active
// monitor goroutine (HTTP or TCP) that marks members up/down with rise/fall
// hysteresis. Monitors are bound to the runtime's context so a config swap
// cleanly stops the old ones.

import (
	"context"
	"hash/fnv"
	"io"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"
)

type ctxKey int

const memberCtxKey ctxKey = 0

// ── runtime types ───────────────────────────────────────────────────────

type memberRuntime struct {
	node   string
	target *url.URL // scheme://host:port
	weight int

	active  int64 // in-flight requests (atomic)
	healthy int32 // 1 up / 0 down (atomic)

	// streak counters are only touched by the single monitor goroutine.
	okStreak   int
	failStreak int
}

type poolRuntime struct {
	name    string
	method  string
	members []*memberRuntime
	monitor MonitorConfig
	rr      uint64 // round-robin cursor (atomic)
}

func (p *poolRuntime) healthyMembers() []*memberRuntime {
	out := make([]*memberRuntime, 0, len(p.members))
	for _, m := range p.members {
		if atomic.LoadInt32(&m.healthy) == 1 {
			out = append(out, m)
		}
	}
	if len(out) == 0 {
		// Fail open: a misconfigured or flapping monitor should not take the
		// whole site offline. Better to try a member than to blind-503.
		return p.members
	}
	return out
}

func fnv32(s string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return h.Sum32()
}

// pick selects a member according to the pool's LB method.
func (p *poolRuntime) pick(clientIP string) *memberRuntime {
	ms := p.healthyMembers()
	if len(ms) == 0 {
		return nil
	}
	switch p.method {
	case "least_conn":
		best, bestv := ms[0], atomic.LoadInt64(&ms[0].active)
		for _, m := range ms[1:] {
			if v := atomic.LoadInt64(&m.active); v < bestv {
				best, bestv = m, v
			}
		}
		return best
	case "ip_hash":
		return ms[fnv32(clientIP)%uint32(len(ms))]
	case "random":
		return ms[rand.Intn(len(ms))]
	default: // round_robin, weighted by member weight
		total := 0
		for _, m := range ms {
			w := m.weight
			if w < 1 {
				w = 1
			}
			total += w
		}
		idx := int(atomic.AddUint64(&p.rr, 1) % uint64(total))
		for _, m := range ms {
			w := m.weight
			if w < 1 {
				w = 1
			}
			if idx < w {
				return m
			}
			idx -= w
		}
		return ms[0]
	}
}

// ── connection accounting transport ─────────────────────────────────────

// lbTransport brackets each proxied request with active-connection accounting
// for the chosen member, so least-connections balancing sees real in-flight
// counts. The count is held until the response body is fully closed.
type lbTransport struct{ base http.RoundTripper }

type countingBody struct {
	io.ReadCloser
	dec func()
	one int32
}

func (c *countingBody) Close() error {
	if atomic.CompareAndSwapInt32(&c.one, 0, 1) {
		c.dec()
	}
	return c.ReadCloser.Close()
}

func (t lbTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	m, _ := r.Context().Value(memberCtxKey).(*memberRuntime)
	if m != nil {
		atomic.AddInt64(&m.active, 1)
	}
	resp, err := t.base.RoundTrip(r)
	if m == nil {
		return resp, err
	}
	if err != nil {
		atomic.AddInt64(&m.active, -1)
		return resp, err
	}
	dec := func() { atomic.AddInt64(&m.active, -1) }
	resp.Body = &countingBody{ReadCloser: resp.Body, dec: dec}
	return resp, nil
}

// ── health monitor ──────────────────────────────────────────────────────

func secOr(v, def int) time.Duration {
	if v <= 0 {
		return time.Duration(def) * time.Second
	}
	return time.Duration(v) * time.Second
}

func intOr(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}

// startMonitor launches the pool's health checks (or marks all members up if
// the monitor is disabled). Bound to ctx so a config swap stops it.
func (p *poolRuntime) startMonitor(ctx context.Context, log *slog.Logger, n *notifier) {
	mon := p.monitor
	// Start optimistic so traffic flows immediately; the monitor demotes bad
	// members within fall*interval.
	for _, m := range p.members {
		atomic.StoreInt32(&m.healthy, 1)
	}
	if mon.Type == "" || mon.Type == "none" {
		return
	}
	interval := secOr(mon.IntervalSec, 5)
	timeout := secOr(mon.TimeoutSec, 2)
	rise := intOr(mon.Rise, 2)
	fall := intOr(mon.Fall, 3)
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		check := func() {
			for _, m := range p.members {
				ok := probe(ctx, client, mon, m, timeout)
				if ok {
					m.okStreak++
					m.failStreak = 0
					if m.okStreak >= rise {
						if atomic.SwapInt32(&m.healthy, 1) == 0 {
							log.Info("pool member up", "pool", p.name, "member", m.target.String())
						}
					}
				} else {
					m.failStreak++
					m.okStreak = 0
					if m.failStreak >= fall {
						if atomic.SwapInt32(&m.healthy, 0) == 1 {
							log.Warn("pool member down", "pool", p.name, "member", m.target.String())
							if n != nil {
								n.push(notifyMemberDown, "alert", "Pool member down",
									p.name+" → "+m.target.String(), "down:"+p.name+"|"+m.target.String(), "", nil)
							}
						}
					}
				}
			}
		}
		check()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				check()
			}
		}
	}()
}

func probe(ctx context.Context, client *http.Client, mon MonitorConfig, m *memberRuntime, timeout time.Duration) bool {
	switch mon.Type {
	case "tcp":
		d := net.Dialer{Timeout: timeout}
		conn, err := d.DialContext(ctx, "tcp", m.target.Host)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	case "http", "https":
		u := *m.target
		if mon.Path != "" {
			u.Path = mon.Path
		} else {
			u.Path = "/"
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return false
		}
		req.Header.Set("User-Agent", "waf-proxy-monitor/1.0")
		resp, err := client.Do(req)
		if err != nil {
			return false
		}
		defer func() { _, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096)); resp.Body.Close() }()
		if mon.ExpectStatus > 0 {
			return resp.StatusCode == mon.ExpectStatus
		}
		return resp.StatusCode >= 200 && resp.StatusCode < 400
	default:
		return true
	}
}

// ── snapshot for the admin API ──────────────────────────────────────────

type memberStatus struct {
	Node    string `json:"node"`
	Target  string `json:"target"`
	Weight  int    `json:"weight"`
	Healthy bool   `json:"healthy"`
	Active  int64  `json:"active"`
}

type poolStatus struct {
	Name    string         `json:"name"`
	Method  string         `json:"method"`
	Monitor string         `json:"monitor"`
	Members []memberStatus `json:"members"`
}

func (p *poolRuntime) status() poolStatus {
	mon := p.monitor.Type
	if mon == "" {
		mon = "none"
	}
	if (mon == "http" || mon == "https") && p.monitor.Path != "" {
		mon += " " + p.monitor.Path
	}
	ms := make([]memberStatus, 0, len(p.members))
	for _, m := range p.members {
		ms = append(ms, memberStatus{
			Node:    m.node,
			Target:  m.target.String(),
			Weight:  m.weight,
			Healthy: atomic.LoadInt32(&m.healthy) == 1,
			Active:  atomic.LoadInt64(&m.active),
		})
	}
	return poolStatus{Name: p.name, Method: p.method, Monitor: mon, Members: ms}
}
