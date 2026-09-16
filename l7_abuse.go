package main

import (
	"net/http"
	"sync"
	"time"
)

// L7AbuseConfig defines bounded application-layer controls. Disabled by
// default so existing deployments keep the Slice A behavior until configured.
type L7AbuseConfig struct {
	Enabled                  bool `json:"enabled,omitempty"`
	RequestsPerWindow        int  `json:"requests_per_window,omitempty"`
	WindowSeconds            int  `json:"window_seconds,omitempty"`
	MaxConcurrentConnections int  `json:"max_concurrent_connections,omitempty"`
	TLSHandshakePerWindow    int  `json:"tls_handshake_per_window,omitempty"`
}

type abuseClientState struct {
	windowStart time.Time
	count       int
	active      int
}

type l7AbuseController struct {
	cfg     L7AbuseConfig
	mu      sync.Mutex
	clients map[string]*abuseClientState
}

func newL7AbuseController(cfg L7AbuseConfig) *l7AbuseController {
	return &l7AbuseController{cfg: cfg, clients: map[string]*abuseClientState{}}
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
		if !c.allow(identity) {
			writeBlockResponse(w, r, http.StatusTooManyRequests, "l7_abuse_limit", true)
			return
		}
		next.ServeHTTP(w, r)
		c.release(identity)
	})
}

func (c *l7AbuseController) allow(identity string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	st := c.clients[identity]
	if st == nil {
		st = &abuseClientState{}
		c.clients[identity] = st
	}
	if st.windowStart.IsZero() || now.Sub(st.windowStart) >= time.Duration(c.cfg.WindowSeconds)*time.Second {
		st.windowStart = now
		st.count = 0
	}
	if c.cfg.RequestsPerWindow > 0 && st.count >= c.cfg.RequestsPerWindow {
		return false
	}
	if c.cfg.MaxConcurrentConnections > 0 && st.active >= c.cfg.MaxConcurrentConnections {
		return false
	}
	st.count++
	st.active++
	return true
}

func (c *l7AbuseController) release(identity string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if st := c.clients[identity]; st != nil && st.active > 0 {
		st.active--
	}
}
