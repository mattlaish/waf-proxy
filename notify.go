package main

// Notifications.
//
// A lightweight in-memory queue of noteworthy events (learner suggestions,
// AI blocks, pool member down, config sync results, peer state changes). Each
// is surfaced in the console bell and optionally POSTed to a webhook
// (Slack/Teams/generic JSON). Some carry an "apply" action payload so the
// operator can act on them with one click — nothing is ever auto-applied.
//
// State is in-memory and resets on restart (a clean seam exists to back it
// with a JSON snapshot later).

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type NotifyConfig struct {
	WebhookURL          string `json:"webhook_url,omitempty"`
	WebhookKind         string `json:"webhook_kind,omitempty"`  // slack | generic
	ClearWebhook        bool   `json:"clear_webhook,omitempty"` // draft intent, consumed on Apply; never HA-shared
	WebhookAllowHTTP    bool   `json:"webhook_allow_http,omitempty"`
	WebhookAllowPrivate bool   `json:"webhook_allow_private,omitempty"`
	// WebhookProxyURL is an optional egress proxy (http:// or https://, no
	// credentials) for networks that reach chat or ticketing services only
	// through a proxy. Environment proxy variables are never used.
	WebhookProxyURL     string `json:"webhook_proxy_url,omitempty"`
	OnAttack            bool   `json:"on_attack"`
	AttackThreshold     int    `json:"attack_threshold,omitempty"`
	AttackWindowSeconds int    `json:"attack_window_seconds,omitempty"`
	AttackMinSeverity   string `json:"attack_min_severity,omitempty"`
	// event toggles
	OnSuggestion bool `json:"on_suggestion"`
	OnAIBlock    bool `json:"on_ai_block"`
	OnMemberDown bool `json:"on_member_down"`
	OnSync       bool `json:"on_sync"`
	OnPeer       bool `json:"on_peer"`
}

func redactNotifySecrets(c Config) Config {
	if c.Notify.WebhookURL != "" {
		c.Notify.WebhookURL = ""
	}
	return c
}

func preserveNotifySecrets(cur Config, next *Config) {
	if next == nil {
		return
	}
	if next.Notify.ClearWebhook {
		next.Notify.WebhookURL = ""
		return
	}
	if strings.TrimSpace(next.Notify.WebhookURL) == "" {
		next.Notify.WebhookURL = cur.Notify.WebhookURL
	}
}

func defaultNotifyConfig() NotifyConfig {
	return NotifyConfig{
		WebhookKind: "slack",
		OnAttack:    true, AttackThreshold: 10, AttackWindowSeconds: 60, AttackMinSeverity: "CRITICAL",
		OnSuggestion: true, OnAIBlock: true, OnMemberDown: true, OnSync: true, OnPeer: true,
	}
}

// notification kinds
const (
	notifySuggestion = "suggestion"
	notifyAIBlock    = "ai_block"
	notifyMemberDown = "member_down"
	notifySync       = "sync"
	notifyPeer       = "peer"
	notifyInfo       = "info"
	notifyAttack     = "attack"
)

type notification struct {
	ID      int64                `json:"id"`
	Time    string               `json:"time"`
	Kind    string               `json:"kind"`
	Level   string               `json:"level"` // info | warn | alert
	Title   string               `json:"title"`
	Body    string               `json:"body"`
	Read    bool                 `json:"read"`
	Handled bool                 `json:"handled"`
	Context *notificationContext `json:"context,omitempty"`
	Action  string               `json:"action,omitempty"` // e.g. "apply_exclusion"
	Payload map[string]any       `json:"payload,omitempty"`
}

type webhookJob struct {
	url    string
	kind   string
	item   notification
	policy NotifyConfig
}

const notificationDedupeMax = 4096

type notifier struct {
	mu             sync.Mutex
	cfg            NotifyConfig
	items          []notification
	nextID         int64
	cap            int
	log            *slog.Logger
	webhookQ       chan webhookJob
	webhookDropped uint64
	delivery       deliveryHealth
	attacks        map[string]*attackWindow
	attackOn       atomic.Bool // fast path: skip the notifier lock when attack alerts are off

	// dedupe map + optional external sink (e.g. syslog)
	dedupe map[string]time.Time
	sink   func(level, kind, title, body string)
}

func newNotifier(log *slog.Logger) *notifier {
	n := &notifier{
		cfg:      defaultNotifyConfig(),
		cap:      200,
		log:      log,
		dedupe:   map[string]time.Time{},
		webhookQ: make(chan webhookJob, 256),
		attacks:  make(map[string]*attackWindow),
	}
	for i := 0; i < 2; i++ {
		go n.webhookWorker()
	}
	return n
}

func (n *notifier) configure(c NotifyConfig) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if c != n.cfg {
		n.delivery.invalidate()
		n.attacks = make(map[string]*attackWindow)
	}
	n.attackOn.Store(c.OnAttack)
	n.cfg = c
}

func (n *notifier) enabledFor(kind string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	switch kind {
	case notifyAttack:
		return n.cfg.OnAttack
	case notifySuggestion:
		return n.cfg.OnSuggestion
	case notifyAIBlock:
		return n.cfg.OnAIBlock
	case notifyMemberDown:
		return n.cfg.OnMemberDown
	case notifySync:
		return n.cfg.OnSync
	case notifyPeer:
		return n.cfg.OnPeer
	}
	return true
}

// push adds a notification (respecting per-kind toggles and a dedupe window)
// and fires the webhook asynchronously.
func (n *notifier) push(kind, level, title, body, dedupeKey string, action string, payload map[string]any) {
	n.pushContext(kind, level, title, body, dedupeKey, action, payload, nil)
}

type notificationContext struct {
	Site      string    `json:"site"`
	Kind      string    `json:"kind"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
}

func (n *notifier) pushContext(kind, level, title, body, dedupeKey string, action string, payload map[string]any, ctx *notificationContext) {
	if !n.enabledFor(kind) {
		return
	}
	n.mu.Lock()
	if dedupeKey != "" {
		now := time.Now()
		if exp, ok := n.dedupe[dedupeKey]; ok && now.Before(exp) {
			n.mu.Unlock()
			return
		}
		// Dedupe is convenience state, not notification authority. Prune expired
		// keys and cap retained identities so attacker-controlled IP/path churn
		// cannot grow the process indefinitely. At saturation we still emit the
		// notification; we simply stop remembering additional dedupe keys.
		for key, exp := range n.dedupe {
			if !now.Before(exp) {
				delete(n.dedupe, key)
			}
		}
		if len(n.dedupe) < notificationDedupeMax {
			n.dedupe[dedupeKey] = now.Add(60 * time.Second)
		}
	}
	n.nextID++
	item := notification{
		ID: n.nextID, Time: time.Now().Format("15:04:05"), Kind: kind, Level: level,
		Title: title, Body: body, Action: action, Payload: payload,
		Context: ctx,
	}
	n.items = append(n.items, item)
	if len(n.items) > n.cap {
		n.items = n.items[len(n.items)-n.cap:]
	}
	hook := n.cfg.WebhookURL
	kindHook := n.cfg.WebhookKind
	policy := n.cfg
	sink := n.sink
	n.mu.Unlock()

	if sink != nil {
		sink(level, kind, title, body)
	}
	if hook != "" {
		select {
		case n.webhookQ <- webhookJob{url: hook, kind: kindHook, item: item, policy: policy}:
		default:
			atomic.AddUint64(&n.webhookDropped, 1)
			n.delivery.fail(n.delivery.current(), errors.New("delivery queue full"))
			n.delivery.noteDrop()
			n.log.Warn("notification webhook queue full; dropping delivery")
		}
	}
}

func (n *notifier) webhookWorker() {
	for job := range n.webhookQ {
		n.mu.Lock()
		current := n.cfg == job.policy
		generation := n.delivery.current()
		n.mu.Unlock()
		if !current {
			// Queued under settings that have since changed: discarded on purpose,
			// counted, but not flagged against the new settings.
			atomic.AddUint64(&n.webhookDropped, 1)
			continue
		}
		for attempt := 0; attempt < 3; attempt++ {
			n.mu.Lock()
			current = n.cfg == job.policy
			n.mu.Unlock()
			if !current {
				atomic.AddUint64(&n.webhookDropped, 1)
				break
			}
			retry, err := n.sendWebhook(job.url, job.kind, job.item, job.policy, generation)
			if err == nil || !retry {
				break
			}
			time.Sleep(time.Duration(1<<attempt) * 250 * time.Millisecond)
		}
	}
}

func (n *notifier) sendWebhook(url, kind string, item notification, policy NotifyConfig, versions ...uint64) (retry bool, err error) {
	generation := n.delivery.current()
	if len(versions) > 0 {
		generation = versions[0]
	}
	defer func() {
		if err == nil {
			n.delivery.recordAt(generation, true)
		} else {
			n.delivery.fail(generation, err)
		}
	}()
	if err := validateWebhookURL(url, policy); err != nil {
		return false, err
	}
	var payload any
	text := "[" + item.Level + "] " + item.Title + " — " + item.Body
	if kind == "slack" {
		payload = map[string]any{"text": text}
	} else {
		payload = map[string]any{
			"level": item.Level, "kind": item.Kind, "title": item.Title,
			"body": item.Body, "time": item.Time,
		}
	}
	b, _ := json.Marshal(payload)
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	client, transport := webhookClient(policy)
	defer transport.CloseIdleConnections()
	resp, err := client.Do(req)
	if err != nil {
		n.log.Warn("notify webhook failed") // URL/query may contain credentials.
		return true, err
	}
	_ = resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return false, nil
	}
	err = fmt.Errorf("webhook HTTP %d", resp.StatusCode)
	n.log.Warn("notify webhook rejected", "status", resp.StatusCode)
	return resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500, err
}

func (n *notifier) list(limit int) []notification {
	n.mu.Lock()
	defer n.mu.Unlock()
	total := len(n.items)
	if limit <= 0 || limit > total {
		limit = total
	}
	out := make([]notification, limit)
	for i := 0; i < limit; i++ {
		out[i] = n.items[total-1-i] // newest first
	}
	return out
}

func (n *notifier) unreadCount() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	c := 0
	for _, it := range n.items {
		if !it.Read {
			c++
		}
	}
	return c
}

func (n *notifier) webhookDeliveryStats() map[string]any {
	if n == nil || n.webhookQ == nil {
		return map[string]any{"queue_depth": 0, "queue_capacity": 0, "dropped": uint64(0)}
	}
	n.mu.Lock()
	enabled := n.cfg.WebhookURL != ""
	defer n.mu.Unlock()
	stats := n.delivery.snapshot(enabled, len(n.webhookQ), cap(n.webhookQ), atomic.LoadUint64(&n.webhookDropped))
	stats["config_generation"] = n.delivery.current()
	stats["config_token"] = n.delivery.configToken()
	return stats
}

func (n *notifier) markHandled(id int64) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	for i := range n.items {
		if n.items[i].ID == id {
			n.items[i].Handled = true
			n.items[i].Read = true
			return true
		}
	}
	return false
}

func (n *notifier) markRead(id int64, all bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for i := range n.items {
		if all || n.items[i].ID == id {
			n.items[i].Read = true
		}
	}
}

func (n *notifier) dismiss(id int64, all bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if all {
		n.items = nil
		return
	}
	out := n.items[:0]
	for _, it := range n.items {
		if it.ID != id {
			out = append(out, it)
		}
	}
	n.items = out
}
