package main

import (
	"fmt"
	"net"
	"net/http"
	"sort"
	"sync"
	"time"
)

type CIDRPolicyEntry struct {
	Name      string `json:"name"`
	CIDR      string `json:"cidr"`
	Action    string `json:"action"` // allow | deny
	Priority  int    `json:"priority,omitempty"`
	Reason    string `json:"reason,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

type CIDRPolicyConfig struct {
	Enabled bool              `json:"enabled,omitempty"`
	Rules   []CIDRPolicyEntry `json:"rules,omitempty"`
}

type cidrPolicyRule struct {
	CIDR     *net.IPNet
	Name     string
	Action   string
	Priority int
	Reason   string
	Expires  time.Time
}

type cidrPolicyDecision struct {
	Allowed bool
	Matched bool
	Rule    string
	Reason  string
}

type cidrPolicyEngine struct {
	rules []cidrPolicyRule
	mu    sync.RWMutex
}

func newCIDRPolicyEngine(cfg CIDRPolicyConfig) (*cidrPolicyEngine, error) {
	e := &cidrPolicyEngine{}
	for _, r := range cfg.Rules {
		_, n, err := net.ParseCIDR(r.CIDR)
		if err != nil {
			return nil, fmt.Errorf("cidr policy %q: %w", r.Name, err)
		}
		if r.Action != "allow" && r.Action != "deny" {
			return nil, fmt.Errorf("cidr policy %q: invalid action", r.Name)
		}
		e.rules = append(e.rules, cidrPolicyRule{CIDR: n, Name: r.Name, Action: r.Action, Priority: r.Priority, Reason: r.Reason})
	}
	sort.SliceStable(e.rules, func(i, j int) bool { return e.rules[i].Priority > e.rules[j].Priority })
	return e, nil
}

func (e *cidrPolicyEngine) evaluate(ip string) cidrPolicyDecision {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return cidrPolicyDecision{}
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	for _, r := range e.rules {
		if !r.Expires.IsZero() && time.Now().After(r.Expires) {
			continue
		}
		if r.CIDR.Contains(parsed) {
			return cidrPolicyDecision{Matched: true, Allowed: r.Action == "allow", Rule: r.Name, Reason: r.Reason}
		}
	}
	return cidrPolicyDecision{}
}

func (e *cidrPolicyEngine) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if e == nil {
			next.ServeHTTP(w, r)
			return
		}
		ip := r.RemoteAddr
		if d, ok := clientIdentityFromContext(r.Context()); ok && d.ResolvedClientIP != "" {
			ip = d.ResolvedClientIP
		}
		if h, _, err := net.SplitHostPort(ip); err == nil {
			ip = h
		}
		d := e.evaluate(ip)
		if d.Matched && !d.Allowed {
			writeBlockResponse(w, r, http.StatusForbidden, "cidr_policy_denied:"+d.Rule, false)
			return
		}
		next.ServeHTTP(w, r)
	})
}
