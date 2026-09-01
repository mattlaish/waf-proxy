package main

// High availability: config sync + failover role.
//
// Scope, stated honestly:
//   * waf-proxy coordinates STATE and ROLE between two instances. On every
//     config apply it pushes the config to the peer, which validates+applies
//     it. It polls the peer's health and computes whether it should consider
//     itself active or standby.
//   * waf-proxy does NOT move IP addresses. Actual packet failover (who
//     answers on the VIP) belongs to keepalived/VRRP or a load balancer, which
//     can consume this instance's role via GET /api/ha (or /healthz). Building
//     an in-process VIP grab would be a dishonest half-solution.
//
// Blocklist state is intentionally NOT synced (per design): each node makes its
// own AI decisions.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type HAConfig struct {
	Enabled   bool   `json:"enabled"`
	Role      string `json:"role"`       // primary | secondary — tie-breaker for split-brain
	PeerURL   string `json:"peer_url"`   // e.g. https://10.0.0.6:9090
	PeerToken string `json:"peer_token"` // admin bearer token of the peer (masked on read)
	SyncConfig bool  `json:"sync_config"`
}

func defaultHAConfig() HAConfig {
	return HAConfig{Role: "primary", SyncConfig: true}
}

func (c HAConfig) validate() error {
	if !c.Enabled {
		return nil
	}
	if c.Role != "primary" && c.Role != "secondary" {
		return fmt.Errorf("ha: role must be primary or secondary")
	}
	if c.PeerURL == "" {
		return fmt.Errorf("ha: peer_url is required when enabled")
	}
	return nil
}

type haState struct {
	PeerUp    bool      `json:"peer_up"`
	Role      string    `json:"role"`      // active | standby | solo
	LastSync  string    `json:"last_sync"` // result text
	LastSeen  time.Time `json:"-"`
	LastError string    `json:"last_error,omitempty"`
}

type haEngine struct {
	mu     sync.Mutex
	cfg    HAConfig
	state  haState
	client *http.Client
	log    *slog.Logger
	notify *notifier

	syncGate gate
}

func newHAEngine(log *slog.Logger, n *notifier) *haEngine {
	return &haEngine{
		cfg:    defaultHAConfig(),
		state:  haState{Role: "solo"},
		client: &http.Client{Timeout: 5 * time.Second},
		log:    log,
		notify: n,
	}
}

func (h *haEngine) configure(c HAConfig) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cfg = c
	if !c.Enabled {
		h.state = haState{Role: "solo"}
	}
}

func (h *haEngine) snapshotCfg() HAConfig {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.cfg
}

func (h *haEngine) status() haState {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.state
}

// run polls peer health and updates role until ctx is cancelled.
func (h *haEngine) run(ctx context.Context) {
	t := time.NewTicker(3 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			h.tick(ctx)
		}
	}
}

func (h *haEngine) tick(ctx context.Context) {
	cfg := h.snapshotCfg()
	if !cfg.Enabled {
		return
	}
	up := h.pingPeer(ctx, cfg)

	h.mu.Lock()
	prevUp := h.state.PeerUp
	h.state.PeerUp = up
	if up {
		h.state.LastSeen = time.Now()
		// Peer alive: primary is active, secondary is standby.
		if cfg.Role == "primary" {
			h.state.Role = "active"
		} else {
			h.state.Role = "standby"
		}
	} else {
		// Peer down: we take over regardless of role.
		h.state.Role = "active"
	}
	role := h.state.Role
	h.mu.Unlock()

	if prevUp != up {
		if up {
			h.notify.push(notifyPeer, "info", "HA peer is up", "peer reachable; role="+role, "ha_peer", "", nil)
		} else {
			h.notify.push(notifyPeer, "alert", "HA peer is DOWN", "peer unreachable — this node is now active", "ha_peer", "", nil)
		}
	}
}

func (h *haEngine) pingPeer(ctx context.Context, cfg HAConfig) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, trimSlash(cfg.PeerURL)+"/healthz", nil)
	if err != nil {
		return false
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// pushConfig sends the given config to the peer's admin API. Called after a
// successful local apply. Best-effort: a peer error is surfaced, not fatal.
func (h *haEngine) pushConfig(cfg Config) {
	hc := h.snapshotCfg()
	if !hc.Enabled || !hc.SyncConfig || hc.PeerURL == "" {
		return
	}
	if !h.syncGate.enter() {
		return // a sync is already in flight
	}
	go func() {
		defer h.syncGate.leave()
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()

		// Mark the payload so the peer doesn't echo it back to us (loop guard).
		body, _ := json.Marshal(cfg)
		req, err := http.NewRequestWithContext(ctx, http.MethodPut,
			trimSlash(hc.PeerURL)+"/api/config", bytes.NewReader(body))
		if err != nil {
			h.recordSync("error: "+err.Error(), true)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+hc.PeerToken)
		req.Header.Set("X-WAF-Sync", "1") // peer treats this as a sync, won't re-push
		resp, err := h.client.Do(req)
		if err != nil {
			h.recordSync("error: "+err.Error(), true)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			h.recordSync(fmt.Sprintf("peer http %d", resp.StatusCode), true)
			return
		}
		h.recordSync("ok "+time.Now().Format("15:04:05"), false)
	}()
}

func (h *haEngine) recordSync(text string, isErr bool) {
	h.mu.Lock()
	h.state.LastSync = text
	if isErr {
		h.state.LastError = text
	} else {
		h.state.LastError = ""
	}
	h.mu.Unlock()
	if isErr {
		h.notify.push(notifySync, "warn", "Config sync failed", text, "ha_sync_err", "", nil)
		h.log.Warn("ha config sync failed", "detail", text)
	} else {
		h.notify.push(notifySync, "info", "Config synced to peer", text, "", "", nil)
	}
}

func trimSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}
