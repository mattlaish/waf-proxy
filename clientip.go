package main

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

const (
	maxForwardedForBytes = 4096
	maxForwardedForHops  = 32
	maxTrustedProxyCIDRs = 64
)

// clientIPResolver turns a request's socket peer and X-Forwarded-For chain
// into one authoritative address. The chain is ignored unless the immediate
// peer is trusted, then walked from right to left until the first untrusted
// hop. This prevents an external client from choosing its own identity by
// prepending an address to X-Forwarded-For.
type clientIPResolver struct {
	trusted []*net.IPNet
}

func newClientIPResolver(cidrs []string) (*clientIPResolver, error) {
	if err := validateTrustedProxyCIDRs(cidrs); err != nil {
		return nil, err
	}
	r := &clientIPResolver{trusted: make([]*net.IPNet, 0, len(cidrs))}
	for _, raw := range cidrs {
		_, network, _ := net.ParseCIDR(strings.TrimSpace(raw))
		r.trusted = append(r.trusted, network)
	}
	return r, nil
}

func validateTrustedProxyCIDRs(cidrs []string) error {
	if len(cidrs) > maxTrustedProxyCIDRs {
		return fmt.Errorf("trusted_proxy_cidrs: at most %d entries are allowed", maxTrustedProxyCIDRs)
	}
	seen := make(map[string]struct{}, len(cidrs))
	for _, raw := range cidrs {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return fmt.Errorf("trusted_proxy_cidrs: entries must not be empty")
		}
		_, network, err := net.ParseCIDR(raw)
		if err != nil {
			return fmt.Errorf("trusted_proxy_cidrs: invalid CIDR %q", raw)
		}
		key := network.String()
		if _, ok := seen[key]; ok {
			return fmt.Errorf("trusted_proxy_cidrs: duplicate network %q", key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func migrateTrustedProxyConfig(c *Config) {
	if len(c.TrustedProxyCIDRs) == 0 && len(c.AI.TrustedProxyCIDRs) > 0 {
		c.TrustedProxyCIDRs = append([]string(nil), c.AI.TrustedProxyCIDRs...)
	}
	// This field used to belong to AI. Clear it after import so future saves
	// have one source of truth and all request consumers share the same value.
	c.AI.TrustedProxyCIDRs = nil
}

func (r *clientIPResolver) isTrusted(ip net.IP) bool {
	if ip == nil {
		return false
	}
	for _, network := range r.trusted {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func (r *clientIPResolver) resolve(req *http.Request) string {
	peerText := clientIP(req)
	peer := net.ParseIP(peerText)
	if peer == nil || !r.isTrusted(peer) {
		return peerText
	}

	forwarded := strings.Join(req.Header.Values("X-Forwarded-For"), ",")
	if forwarded == "" || len(forwarded) > maxForwardedForBytes {
		return peerText
	}
	parts := strings.Split(forwarded, ",")
	if len(parts) > maxForwardedForHops {
		return peerText
	}
	hops := make([]net.IP, len(parts))
	for i, raw := range parts {
		hops[i] = net.ParseIP(strings.TrimSpace(raw))
		if hops[i] == nil {
			return peerText
		}
	}

	candidate := peer
	for i := len(hops) - 1; i >= 0; i-- {
		if !r.isTrusted(candidate) {
			return candidate.String()
		}
		candidate = hops[i]
	}
	return candidate.String()
}

// wrap resolves the client once, before logging, AI, Coraza, load balancing,
// and proxying. Replacing RemoteAddr makes existing consumers agree without
// letting any of them accidentally re-interpret untrusted forwarding headers.
func (r *clientIPResolver) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		resolved := r.resolve(req)
		_, port, err := net.SplitHostPort(req.RemoteAddr)
		if err == nil {
			req.RemoteAddr = net.JoinHostPort(resolved, port)
		} else {
			req.RemoteAddr = resolved
		}
		next.ServeHTTP(w, req)
	})
}
