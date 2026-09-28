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
	"net/url"
	"strings"
	"sync"
	"time"
)

type HAConfig struct {
	Enabled    bool   `json:"enabled"`
	Role       string `json:"role"`       // primary | secondary — tie-breaker for split-brain
	PeerURL    string `json:"peer_url"`   // e.g. https://10.0.0.6:9090
	PeerToken  string `json:"peer_token"` // admin bearer token of the peer (masked on read)
	SyncConfig bool   `json:"sync_config"`
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
	u, err := url.Parse(c.PeerURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return fmt.Errorf("ha: peer_url must be an https origin without userinfo, query, fragment, or path")
	}
	if c.SyncConfig && strings.TrimSpace(c.PeerToken) == "" {
		return fmt.Errorf("ha: peer_token is required when sync_config is enabled")
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

	syncMu      sync.Mutex
	syncRunning bool
	syncPending *haSyncJob
}

type haSyncJob struct {
	peerURL   string
	peerToken string
	body      []byte
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

type haSyncEnvelope struct {
	Version int    `json:"version"`
	Config  Config `json:"config"`
}

const haSyncEnvelopeVersion = 1

// sharedConfigForPeer removes node-local control-plane identity and secret
// material before HA synchronization. The receiver always merges its own HA
// identity, users and secret references back in before validation/apply.
func sharedConfigForPeer(cfg Config) Config {
	out := cfg
	out.HA = HAConfig{}
	out.Users = nil
	out = redactAISecrets(out)
	out = redactHSMSecretRefs(out)
	out = redactNotifySecrets(out)
	return out
}

func mergePeerConfig(local, incoming Config) Config {
	incoming.HA = local.HA
	incoming.Users = local.Users
	preserveAISecrets(local, &incoming)
	preserveNotifySecrets(local, &incoming)
	preserveHSMSecretRefs(local, &incoming)
	return incoming
}

// pushConfig sends the given config to the peer's dedicated replication API.
// There is at most one network request in flight. If more local Applies commit
// while it is running, they replace a single pending slot so the peer always
// converges to the newest committed shared config without an unbounded queue.
func (h *haEngine) pushConfig(cfg Config) {
	hc := h.snapshotCfg()
	if !hc.Enabled || !hc.SyncConfig || hc.PeerURL == "" {
		return
	}
	payload := haSyncEnvelope{Version: haSyncEnvelopeVersion, Config: sharedConfigForPeer(cfg)}
	body, err := json.Marshal(payload)
	if err != nil {
		h.recordSync("error: encode peer config: "+err.Error(), true)
		return
	}
	job := haSyncJob{peerURL: hc.PeerURL, peerToken: hc.PeerToken, body: body}

	h.syncMu.Lock()
	if h.syncRunning {
		// Latest wins: older unsent pending state is obsolete once a newer local
		// config has durably committed.
		pending := job
		h.syncPending = &pending
		h.syncMu.Unlock()
		return
	}
	h.syncRunning = true
	h.syncMu.Unlock()
	go h.runSyncLoop(job)
}

func (h *haEngine) runSyncLoop(job haSyncJob) {
	for {
		h.sendSyncJob(job)
		h.syncMu.Lock()
		if h.syncPending != nil {
			job = *h.syncPending
			h.syncPending = nil
			h.syncMu.Unlock()
			continue
		}
		h.syncRunning = false
		h.syncMu.Unlock()
		return
	}
}

func (h *haEngine) sendSyncJob(job haSyncJob) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPut,
		trimSlash(job.peerURL)+"/api/ha/peer-config", bytes.NewReader(job.body))
	if err != nil {
		h.recordSync("error: "+err.Error(), true)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+job.peerToken)
	req.Header.Set("X-WAF-HA-Sync", "v1")
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
