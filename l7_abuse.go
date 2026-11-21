package main

import (
	"errors"
	"net"
	"net/http"
	"net/netip"
	"sync"
	"time"

	"waf-proxy/internal/tlsfront"
)

// L7AbuseConfig defines bounded application-layer controls. Disabled by
// default so existing deployments keep the Slice A behavior until configured.
//
// RequestsPerWindow and WindowSeconds configure a token bucket for backward
// configuration compatibility: RequestsPerWindow is the bucket capacity and
// tokens refill continuously at RequestsPerWindow / WindowSeconds.
type L7AbuseConfig struct {
	Enabled                  bool `json:"enabled,omitempty"`
	RequestsPerWindow        int  `json:"requests_per_window,omitempty"`
	WindowSeconds            int  `json:"window_seconds,omitempty"`
	MaxConcurrentConnections int  `json:"max_concurrent_connections,omitempty"`
	TLSHandshakePerWindow    int  `json:"tls_handshake_per_window,omitempty"`
	IPv6PrefixLength         int  `json:"ipv6_prefix_length,omitempty"` // 0 = /64; /128 is an explicit per-address trade-off
}

const (
	l7AbuseShardCount         = 64
	l7AbuseMaxEntriesPerShard = 1024
	l7AbusePruneEvery         = 256
	l7AbuseMinIdleTTL         = 5 * time.Minute
)

type abuseClientKey struct {
	site     string
	identity string
}

type abuseClientState struct {
	mu         sync.Mutex
	tokens     float64
	lastRefill time.Time
	lastSeen   time.Time
	active     int
}

type l7AbuseShard struct {
	mu      sync.Mutex
	clients map[abuseClientKey]*abuseClientState
	ops     uint64
}

type l7AbuseController struct {
	cfg        L7AbuseConfig
	shards     [l7AbuseShardCount]l7AbuseShard
	tlsShards  [l7AbuseShardCount]l7AbuseShard
	overflowMu sync.Mutex
	overflows  map[abuseClientKey]*abuseClientState
}

func newL7AbuseController(cfg L7AbuseConfig) *l7AbuseController {
	c := &l7AbuseController{cfg: cfg, overflows: make(map[abuseClientKey]*abuseClientState)}
	for i := range c.shards {
		c.shards[i].clients = make(map[abuseClientKey]*abuseClientState)
		c.tlsShards[i].clients = make(map[abuseClientKey]*abuseClientState)
	}
	return c
}

func validateL7AbuseConfig(cfg L7AbuseConfig, accel tlsfront.AccelerationConfig) error {
	if cfg.IPv6PrefixLength != 0 && (cfg.IPv6PrefixLength < 64 || cfg.IPv6PrefixLength > 128) {
		return errors.New("l7_abuse.ipv6_prefix_length must be 0 (default /64) or 64..128")
	}
	if cfg.RequestsPerWindow < 0 || cfg.RequestsPerWindow > 10_000_000 {
		return errors.New("l7_abuse.requests_per_window must be 0..10000000")
	}
	if cfg.WindowSeconds < 0 || cfg.WindowSeconds > 86_400 {
		return errors.New("l7_abuse.window_seconds must be 0..86400")
	}
	if cfg.MaxConcurrentConnections < 0 || cfg.MaxConcurrentConnections > 1_000_000 {
		return errors.New("l7_abuse.max_concurrent_connections must be 0..1000000")
	}
	if cfg.TLSHandshakePerWindow < 0 || cfg.TLSHandshakePerWindow > 10_000_000 {
		return errors.New("l7_abuse.tls_handshake_per_window must be 0..10000000")
	}
	if cfg.Enabled && cfg.TLSHandshakePerWindow > 0 && tlsfront.FrontendEnabled(accel) {
		return errors.New("l7_abuse.tls_handshake_per_window requires tls_acceleration.mode=go; external TLS frontend handshakes are outside the Go enforcement path")
	}
	return nil
}

// allowTLSHandshakeAt enforces the configured TLS ClientHello rate before the
// expensive certificate/signature path. The identity is the direct TCP peer
// because no HTTP forwarding metadata exists before the TLS handshake. The
// scope is the public listener rather than SNI so rotating hostnames cannot
// evade the bucket.
func (c *l7AbuseController) allowTLSHandshakeAt(scope, identity string, now time.Time) bool {
	if c == nil || !c.cfg.Enabled || c.cfg.TLSHandshakePerWindow <= 0 {
		return true
	}
	key := abuseClientKey{site: scope, identity: c.limiterIdentity(identity)}
	shard := &c.tlsShards[l7AbuseShardIndex(key)]
	shard.mu.Lock()
	defer shard.mu.Unlock()

	shard.ops++
	if shard.ops%l7AbusePruneEvery == 0 {
		c.pruneShardLocked(shard, now)
	}
	st := shard.clients[key]
	if st == nil {
		if len(shard.clients) >= l7AbuseMaxEntriesPerShard {
			c.pruneShardLocked(shard, now)
		}
		if len(shard.clients) >= l7AbuseMaxEntriesPerShard {
			c.evictOldestIdleLocked(shard, now)
		}
		if len(shard.clients) >= l7AbuseMaxEntriesPerShard {
			// Saturation must not turn the limiter off. Unknown identities share a
			// bounded overflow bucket until an idle per-identity slot is available.
			st = c.overflowState(scope, "tls")
			if st == nil {
				return false
			}
		} else {
			st = &abuseClientState{tokens: float64(c.cfg.TLSHandshakePerWindow), lastRefill: now, lastSeen: now}
			shard.clients[key] = st
		}
	}

	st.mu.Lock()
	defer st.mu.Unlock()
	st.lastSeen = now
	window := c.windowDuration()
	capacity := float64(c.cfg.TLSHandshakePerWindow)
	if st.lastRefill.IsZero() {
		st.tokens = capacity
		st.lastRefill = now
	} else if now.After(st.lastRefill) {
		st.tokens += now.Sub(st.lastRefill).Seconds() * (capacity / window.Seconds())
		if st.tokens > capacity {
			st.tokens = capacity
		}
		st.lastRefill = now
	}
	if st.tokens < 1 {
		return false
	}
	st.tokens--
	return true
}

func (c *l7AbuseController) wrap(site string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c == nil || !c.cfg.Enabled {
			next.ServeHTTP(w, r)
			return
		}
		identity := r.RemoteAddr
		if d, ok := clientIdentityFromContext(r.Context()); ok && d.ResolvedClientIP != "" {
			identity = d.ResolvedClientIP
		}
		state, allowed := c.allowAt(site, identity, time.Now())
		if !allowed {
			// API clients keep the machine-readable JSON body; browsers get the
			// visitor page explaining the limit.
			writeBlockResponse(w, r, http.StatusTooManyRequests, "l7_abuse_limit", !acceptsHTML(r))
			return
		}
		if state != nil {
			// Release must run even when a downstream handler panics. For a
			// long-lived/hijacked handler that does not return, the active slot
			// intentionally remains occupied for the lifetime of that handler.
			defer c.releaseState(state)
		}
		next.ServeHTTP(w, r)
	})
}

func (c *l7AbuseController) allowAt(site, identity string, now time.Time) (*abuseClientState, bool) {
	key := abuseClientKey{site: site, identity: c.limiterIdentity(identity)}
	shard := &c.shards[l7AbuseShardIndex(key)]
	shard.mu.Lock()
	defer shard.mu.Unlock()

	shard.ops++
	if shard.ops%l7AbusePruneEvery == 0 {
		c.pruneShardLocked(shard, now)
	}

	st := shard.clients[key]
	if st == nil {
		if len(shard.clients) >= l7AbuseMaxEntriesPerShard {
			c.pruneShardLocked(shard, now)
		}
		if len(shard.clients) >= l7AbuseMaxEntriesPerShard {
			c.evictOldestIdleLocked(shard, now)
		}
		if len(shard.clients) >= l7AbuseMaxEntriesPerShard {
			// Bound memory without creating an attacker-controlled fail-open bypass.
			// One bucket per configured site, shared across shards, not sites.
			st = c.overflowState(site, "http")
			if st == nil {
				return nil, false
			}
		} else {
			st = &abuseClientState{lastSeen: now}
			if c.cfg.RequestsPerWindow > 0 {
				st.tokens = float64(c.cfg.RequestsPerWindow)
				st.lastRefill = now
			}
			shard.clients[key] = st
		}
	}

	// The shard lock pins the map entry until active has been incremented, so
	// an eviction cannot orphan a state pointer between lookup and admission.
	st.mu.Lock()
	defer st.mu.Unlock()
	st.lastSeen = now

	if c.cfg.MaxConcurrentConnections > 0 && st.active >= c.cfg.MaxConcurrentConnections {
		return nil, false
	}

	if c.cfg.RequestsPerWindow > 0 {
		window := c.windowDuration()
		capacity := float64(c.cfg.RequestsPerWindow)
		if st.lastRefill.IsZero() {
			st.tokens = capacity
			st.lastRefill = now
		} else if now.After(st.lastRefill) {
			refillPerSecond := capacity / window.Seconds()
			st.tokens += now.Sub(st.lastRefill).Seconds() * refillPerSecond
			if st.tokens > capacity {
				st.tokens = capacity
			}
			st.lastRefill = now
		}
		if st.tokens < 1 {
			return nil, false
		}
		st.tokens--
	}

	st.active++
	return st, true
}

func (c *l7AbuseController) releaseState(st *abuseClientState) {
	if st == nil {
		return
	}
	st.mu.Lock()
	if st.active > 0 {
		st.active--
	}
	st.mu.Unlock()
}

func (c *l7AbuseController) pruneShardLocked(shard *l7AbuseShard, now time.Time) {
	cutoff := now.Add(-c.idleTTL())
	for key, st := range shard.clients {
		st.mu.Lock()
		idle := st.active == 0 && st.lastSeen.Before(cutoff)
		st.mu.Unlock()
		if idle {
			delete(shard.clients, key)
		}
	}
}

// evictOldestIdleLocked reclaims at most one entry to make room, but only a
// genuinely stale one: idle (no in-flight request) AND not seen within the idle
// TTL. Evicting a recently-seen idle entry would let an attacker cycling many
// identities reclaim a fresh full-capacity bucket per identity, turning a
// saturated shard into a fail-open bypass. When every entry is recent the shard
// stays full and new identities fall to the bounded shared overflow bucket.
func (c *l7AbuseController) evictOldestIdleLocked(shard *l7AbuseShard, now time.Time) {
	cutoff := now.Add(-c.idleTTL())
	var oldestKey abuseClientKey
	var oldest time.Time
	found := false
	for key, st := range shard.clients {
		st.mu.Lock()
		active := st.active
		lastSeen := st.lastSeen
		st.mu.Unlock()
		if active != 0 || !lastSeen.Before(cutoff) {
			continue
		}
		if !found || lastSeen.Before(oldest) {
			oldestKey = key
			oldest = lastSeen
			found = true
		}
	}
	if found {
		delete(shard.clients, oldestKey)
	}
}

func (c *l7AbuseController) windowDuration() time.Duration {
	window := time.Duration(c.cfg.WindowSeconds) * time.Second
	if window <= 0 {
		return time.Second
	}
	return window
}

// Scope comes from configured site/listener names, never Host or SNI input.
// Never evict an overflow state to mint fresh tokens; cap even configured scopes.
func (c *l7AbuseController) overflowState(scope, kind string) *abuseClientState {
	c.overflowMu.Lock()
	defer c.overflowMu.Unlock()
	key := abuseClientKey{site: scope, identity: kind}
	if st := c.overflows[key]; st != nil {
		return st
	}
	if len(c.overflows) >= 8192 {
		return nil
	}
	st := &abuseClientState{}
	c.overflows[key] = st
	return st
}

// Only limiter identity is aggregated; logs/ACLs retain the original client IP.
func (c *l7AbuseController) limiterIdentity(identity string) string {
	if host, _, err := net.SplitHostPort(identity); err == nil {
		identity = host
	}
	ip, err := netip.ParseAddr(identity)
	if err != nil {
		return identity
	}
	ip = ip.Unmap().WithZone("")
	if ip.Is4() {
		return ip.String()
	}
	bits := c.cfg.IPv6PrefixLength
	if bits == 0 {
		bits = 64
	}
	return netip.PrefixFrom(ip, bits).Masked().String()
}

func (c *l7AbuseController) idleTTL() time.Duration {
	window := c.windowDuration()
	ttl := 2 * window
	if ttl < l7AbuseMinIdleTTL {
		return l7AbuseMinIdleTTL
	}
	return ttl
}

func l7AbuseShardIndex(key abuseClientKey) uint64 {
	// Allocation-free FNV-1a over site and trusted identity. Keeping site in
	// the key separates normal per-client buckets. Table capacity is still
	// globally bounded, while overflow allowances are isolated per site/listener.
	const (
		offset64 = uint64(14695981039346656037)
		prime64  = uint64(1099511628211)
	)
	h := offset64
	for i := 0; i < len(key.site); i++ {
		h ^= uint64(key.site[i])
		h *= prime64
	}
	h ^= 0
	h *= prime64
	for i := 0; i < len(key.identity); i++ {
		h ^= uint64(key.identity[i])
		h *= prime64
	}
	return h % l7AbuseShardCount
}
