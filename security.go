package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const requestIDHeader = "X-WAF-Request-ID"

const maxSecurityRateBuckets = 65536

type requestIDContextKey struct{}

func requestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(requestIDContextKey{}).(string)
	return v
}

func withRequestID(r *http.Request) (*http.Request, string) {
	if r == nil {
		return r, ""
	}
	// Never trust a client-supplied correlation identifier. Generate one at the
	// WAF boundary and carry it through the request context instead.
	r.Header.Del(requestIDHeader)
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		n := time.Now().UnixNano()
		return r.WithContext(context.WithValue(r.Context(), requestIDContextKey{}, fmt.Sprintf("fallback-%x", n))), fmt.Sprintf("fallback-%x", n)
	}
	id := hex.EncodeToString(raw[:])
	return r.WithContext(context.WithValue(r.Context(), requestIDContextKey{}, id)), id
}

type CIDRAccessRule struct {
	CIDR      string `json:"cidr"`
	ExpiresAt string `json:"expires_at,omitempty"` // RFC3339; empty means no expiry.
}

type SecurityConfig struct {
	Enabled                bool             `json:"enabled"`
	RatePerSecond          int              `json:"rate_per_second,omitempty"`
	RateBurst              int              `json:"rate_burst,omitempty"`
	MaxInflightPerIP       int              `json:"max_inflight_per_ip,omitempty"`
	MaxConnectionsPerIP    int              `json:"max_connections_per_ip,omitempty"`
	TLSHandshakesPerMinute int              `json:"tls_handshakes_per_minute,omitempty"`
	TLSHandshakeBurst      int              `json:"tls_handshake_burst,omitempty"`
	AllowCIDRs             []CIDRAccessRule `json:"allow_cidrs,omitempty"`
	DenyCIDRs              []CIDRAccessRule `json:"deny_cidrs,omitempty"`
	BlockStatus            int              `json:"block_status,omitempty"`
	BlockPage              string           `json:"block_page,omitempty"`
	StatePath              string           `json:"state_path,omitempty"`
}

func defaultSecurityConfig() SecurityConfig {
	return SecurityConfig{BlockStatus: http.StatusForbidden, StatePath: "/var/lib/waf-proxy/security-state.json"}
}

func validateSecurityConfig(c SecurityConfig) error {
	for name, v := range map[string]int{
		"rate_per_second":           c.RatePerSecond,
		"rate_burst":                c.RateBurst,
		"max_inflight_per_ip":       c.MaxInflightPerIP,
		"max_connections_per_ip":    c.MaxConnectionsPerIP,
		"tls_handshakes_per_minute": c.TLSHandshakesPerMinute,
		"tls_handshake_burst":       c.TLSHandshakeBurst,
	} {
		if v < 0 {
			return fmt.Errorf("security.%s must be >= 0", name)
		}
	}
	if c.RatePerSecond > 100000 || c.RateBurst > 100000 || c.MaxInflightPerIP > 100000 || c.MaxConnectionsPerIP > 100000 || c.TLSHandshakesPerMinute > 1000000 || c.TLSHandshakeBurst > 100000 {
		return errors.New("security limits exceed supported maximum")
	}
	if c.RatePerSecond > 0 && c.RateBurst == 0 {
		return errors.New("security.rate_burst must be > 0 when rate_per_second is enabled")
	}
	if c.TLSHandshakesPerMinute > 0 && c.TLSHandshakeBurst == 0 {
		return errors.New("security.tls_handshake_burst must be > 0 when tls_handshakes_per_minute is enabled")
	}
	if c.BlockStatus != 0 && c.BlockStatus != http.StatusForbidden && c.BlockStatus != http.StatusTooManyRequests && c.BlockStatus != http.StatusNotFound && c.BlockStatus != http.StatusServiceUnavailable {
		return errors.New("security.block_status must be 403, 404, 429, 503, or 0")
	}
	if c.StatePath != "" && !strings.HasPrefix(c.StatePath, "/") {
		return errors.New("security.state_path must be absolute or empty")
	}
	if len(c.BlockPage) > 64<<10 {
		return errors.New("security.block_page exceeds 64 KiB")
	}
	if _, err := template.New("block").Option("missingkey=zero").Parse(effectiveBlockPage(c)); err != nil {
		return fmt.Errorf("security.block_page: %w", err)
	}
	if err := validateCIDRRules("allow_cidrs", c.AllowCIDRs); err != nil {
		return err
	}
	if err := validateCIDRRules("deny_cidrs", c.DenyCIDRs); err != nil {
		return err
	}
	return nil
}

func validateCIDRRules(label string, rules []CIDRAccessRule) error {
	if len(rules) > 4096 {
		return fmt.Errorf("security.%s supports at most 4096 entries", label)
	}
	seen := map[string]struct{}{}
	for i, rule := range rules {
		p, err := netip.ParsePrefix(strings.TrimSpace(rule.CIDR))
		if err != nil {
			return fmt.Errorf("security.%s[%d]: invalid CIDR %q", label, i, rule.CIDR)
		}
		canon := p.Masked().String()
		if _, ok := seen[canon]; ok {
			return fmt.Errorf("security.%s contains duplicate CIDR %q", label, canon)
		}
		seen[canon] = struct{}{}
		if rule.ExpiresAt != "" {
			if _, err := time.Parse(time.RFC3339, rule.ExpiresAt); err != nil {
				return fmt.Errorf("security.%s[%d].expires_at must be RFC3339", label, i)
			}
		}
	}
	return nil
}

func effectiveBlockPage(c SecurityConfig) string {
	if strings.TrimSpace(c.BlockPage) != "" {
		return c.BlockPage
	}
	return `<!doctype html><html><head><meta charset="utf-8"><title>Request blocked</title></head><body><h1>Request blocked</h1><p>Reference: {{.RequestID}}</p></body></html>`
}

type compiledCIDRRule struct {
	prefix netip.Prefix
	expiry time.Time
}

type rateBucket struct {
	tokens float64
	last   time.Time
}

type securityCounters struct {
	Allowed          uint64 `json:"allowed"`
	CIDRDenied       uint64 `json:"cidr_denied"`
	RateLimited      uint64 `json:"rate_limited"`
	InflightRejected uint64 `json:"inflight_rejected"`
	ConnRejected     uint64 `json:"connection_rejected"`
	TLSRejected      uint64 `json:"tls_handshake_rejected"`
}

type securityManager struct {
	mu sync.Mutex

	cfg   SecurityConfig
	allow []compiledCIDRRule
	deny  []compiledCIDRRule
	block *template.Template

	rate      map[string]rateBucket
	inflight  map[string]int
	connCount map[string]int
	connPeer  map[net.Conn]string
	tlsRate   map[string]rateBucket

	allowed          atomic.Uint64
	cidrDenied       atomic.Uint64
	rateLimited      atomic.Uint64
	inflightRejected atomic.Uint64
	connRejected     atomic.Uint64
	tlsRejected      atomic.Uint64
}

func newSecurityManager() *securityManager {
	s := &securityManager{}
	_ = s.configure(defaultSecurityConfig())
	return s
}

func (s *securityManager) configure(c SecurityConfig) error {
	if err := validateSecurityConfig(c); err != nil {
		return err
	}
	if c.BlockStatus == 0 {
		c.BlockStatus = http.StatusForbidden
	}
	parseRules := func(in []CIDRAccessRule) []compiledCIDRRule {
		out := make([]compiledCIDRRule, 0, len(in))
		for _, rule := range in {
			p, _ := netip.ParsePrefix(strings.TrimSpace(rule.CIDR))
			cr := compiledCIDRRule{prefix: p.Masked()}
			if rule.ExpiresAt != "" {
				cr.expiry, _ = time.Parse(time.RFC3339, rule.ExpiresAt)
			}
			out = append(out, cr)
		}
		return out
	}
	bt, err := template.New("block").Option("missingkey=zero").Parse(effectiveBlockPage(c))
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.cfg = c
	s.allow = parseRules(c.AllowCIDRs)
	s.deny = parseRules(c.DenyCIDRs)
	s.block = bt
	// A config apply is a clean policy boundary. Transport/request counters do
	// not carry across a materially different policy.
	s.rate = map[string]rateBucket{}
	s.inflight = map[string]int{}
	// Preserve live transport connection accounting across config applies; old
	// connections may close after the new policy is active and must still be
	// decremented correctly.
	if s.connCount == nil {
		s.connCount = map[string]int{}
	}
	if s.connPeer == nil {
		s.connPeer = map[net.Conn]string{}
	}
	s.tlsRate = map[string]rateBucket{}
	s.mu.Unlock()
	return nil
}

func activeRule(r compiledCIDRRule, now time.Time) bool {
	return r.expiry.IsZero() || now.Before(r.expiry)
}

func ipInRules(ip string, rules []compiledCIDRRule, now time.Time) bool {
	addr, err := netip.ParseAddr(strings.TrimSpace(ip))
	if err != nil {
		return false
	}
	addr = addr.Unmap()
	for _, r := range rules {
		if activeRule(r, now) && r.prefix.Contains(addr) {
			return true
		}
	}
	return false
}

func boundRateBuckets(m map[string]rateBucket, key string, now time.Time) {
	if _, ok := m[key]; ok || len(m) < maxSecurityRateBuckets {
		return
	}
	// Prune old identities first. If a sustained distributed source set still
	// fills the map, trim to 75%% capacity so the O(n) scan is amortized rather
	// than repeated for every new address. Rate state is defensive telemetry,
	// not an authorization decision, so eviction is safe and bounded.
	staleBefore := now.Add(-10 * time.Minute)
	for k, b := range m {
		if b.last.Before(staleBefore) {
			delete(m, k)
		}
	}
	target := maxSecurityRateBuckets * 3 / 4
	for k := range m {
		if len(m) <= target {
			break
		}
		delete(m, k)
	}
}

func consumeBucket(m map[string]rateBucket, key string, ratePerSec float64, burst int, now time.Time) bool {
	if ratePerSec <= 0 || burst <= 0 {
		return true
	}
	boundRateBuckets(m, key, now)
	b, ok := m[key]
	if !ok {
		b = rateBucket{tokens: float64(burst), last: now}
	}
	elapsed := now.Sub(b.last).Seconds()
	if elapsed > 0 {
		b.tokens += elapsed * ratePerSec
		if b.tokens > float64(burst) {
			b.tokens = float64(burst)
		}
	}
	b.last = now
	if b.tokens < 1 {
		m[key] = b
		return false
	}
	b.tokens--
	m[key] = b
	return true
}

func (s *securityManager) wrap(site string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		id := requestIDFromContext(r.Context())
		if id == "" {
			r, id = withRequestID(r)
		}
		w.Header().Set(requestIDHeader, id)
		ip := clientIP(r)
		now := time.Now()

		s.mu.Lock()
		cfg := s.cfg
		allowlisted := ipInRules(ip, s.allow, now)
		denylisted := !allowlisted && ipInRules(ip, s.deny, now)
		if !cfg.Enabled {
			s.mu.Unlock()
			next.ServeHTTP(w, r)
			return
		}
		if denylisted {
			s.mu.Unlock()
			s.cidrDenied.Add(1)
			s.writeBlock(w, id, cfg.BlockStatus, "cidr_deny")
			return
		}
		if allowlisted {
			s.mu.Unlock()
			s.allowed.Add(1)
			next.ServeHTTP(w, r)
			return
		}
		if cfg.RatePerSecond > 0 && !consumeBucket(s.rate, ip, float64(cfg.RatePerSecond), cfg.RateBurst, now) {
			s.mu.Unlock()
			s.rateLimited.Add(1)
			s.writeBlock(w, id, http.StatusTooManyRequests, "rate_limit")
			return
		}
		if cfg.MaxInflightPerIP > 0 && s.inflight[ip] >= cfg.MaxInflightPerIP {
			s.mu.Unlock()
			s.inflightRejected.Add(1)
			s.writeBlock(w, id, http.StatusTooManyRequests, "inflight_limit")
			return
		}
		s.inflight[ip]++
		s.mu.Unlock()
		s.allowed.Add(1)
		defer func() {
			s.mu.Lock()
			if s.inflight[ip] <= 1 {
				delete(s.inflight, ip)
			} else {
				s.inflight[ip]--
			}
			s.mu.Unlock()
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *securityManager) writeBlock(w http.ResponseWriter, id string, status int, reason string) {
	if status == 0 {
		status = http.StatusForbidden
	}
	s.mu.Lock()
	bt := s.block
	s.mu.Unlock()
	w.Header().Set(requestIDHeader, id)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if bt != nil {
		_ = bt.Execute(w, struct {
			RequestID string
			Status    int
			Reason    string
		}{RequestID: id, Status: status, Reason: reason})
	}
}

func directPeerIP(addr net.Addr) string {
	if addr == nil {
		return ""
	}
	host, _, err := net.SplitHostPort(addr.String())
	if err == nil {
		return strings.Trim(host, "[]")
	}
	return strings.Trim(addr.String(), "[]")
}

func (s *securityManager) allowTLSHandshake(addr net.Addr) bool {
	peer := directPeerIP(addr)
	if peer == "" {
		return true
	}
	now := time.Now()
	s.mu.Lock()
	cfg := s.cfg
	if !cfg.Enabled || cfg.TLSHandshakesPerMinute == 0 || ipInRules(peer, s.allow, now) {
		s.mu.Unlock()
		return true
	}
	ok := consumeBucket(s.tlsRate, peer, float64(cfg.TLSHandshakesPerMinute)/60.0, cfg.TLSHandshakeBurst, now)
	s.mu.Unlock()
	if !ok {
		s.tlsRejected.Add(1)
	}
	return ok
}

func (s *securityManager) connState(c net.Conn, state http.ConnState) {
	if c == nil {
		return
	}
	peer := directPeerIP(c.RemoteAddr())
	s.mu.Lock()
	cfg := s.cfg
	switch state {
	case http.StateNew:
		if !cfg.Enabled || cfg.MaxConnectionsPerIP == 0 || peer == "" || ipInRules(peer, s.allow, time.Now()) {
			s.mu.Unlock()
			return
		}
		if s.connCount[peer] >= cfg.MaxConnectionsPerIP {
			s.mu.Unlock()
			s.connRejected.Add(1)
			_ = c.Close()
			return
		}
		s.connCount[peer]++
		s.connPeer[c] = peer
	case http.StateClosed, http.StateHijacked:
		if p, ok := s.connPeer[c]; ok {
			if s.connCount[p] <= 1 {
				delete(s.connCount, p)
			} else {
				s.connCount[p]--
			}
			delete(s.connPeer, c)
		}
	}
	s.mu.Unlock()
}

func (s *securityManager) counters() securityCounters {
	return securityCounters{
		Allowed: s.allowed.Load(), CIDRDenied: s.cidrDenied.Load(), RateLimited: s.rateLimited.Load(),
		InflightRejected: s.inflightRejected.Load(), ConnRejected: s.connRejected.Load(), TLSRejected: s.tlsRejected.Load(),
	}
}

func (s *securityManager) restoreCounters(c securityCounters) {
	s.allowed.Store(c.Allowed)
	s.cidrDenied.Store(c.CIDRDenied)
	s.rateLimited.Store(c.RateLimited)
	s.inflightRejected.Store(c.InflightRejected)
	s.connRejected.Store(c.ConnRejected)
	s.tlsRejected.Store(c.TLSRejected)
}
