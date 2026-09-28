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
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type NotifyConfig struct {
	WebhookURL  string `json:"webhook_url,omitempty"`
	WebhookKind string `json:"webhook_kind,omitempty"` // slack | generic
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
	if strings.TrimSpace(next.Notify.WebhookURL) == "" {
		next.Notify.WebhookURL = cur.Notify.WebhookURL
	}
}

func defaultNotifyConfig() NotifyConfig {
	return NotifyConfig{
		WebhookKind:  "slack",
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
)

type notification struct {
	ID      int64          `json:"id"`
	Time    string         `json:"time"`
	Kind    string         `json:"kind"`
	Level   string         `json:"level"` // info | warn | alert
	Title   string         `json:"title"`
	Body    string         `json:"body"`
	Read    bool           `json:"read"`
	Action  string         `json:"action,omitempty"` // e.g. "apply_exclusion"
	Payload map[string]any `json:"payload,omitempty"`
}

type webhookJob struct {
	url  string
	kind string
	item notification
}

const notificationDedupeMax = 4096

type notifier struct {
	mu             sync.Mutex
	cfg            NotifyConfig
	items          []notification
	nextID         int64
	cap            int
	client         *http.Client
	log            *slog.Logger
	webhookQ       chan webhookJob
	webhookDropped uint64

	// dedupe map + optional external sink (e.g. syslog)
	dedupe map[string]time.Time
	sink   func(level, kind, title, body string)
}

func newNotifier(log *slog.Logger) *notifier {
	n := &notifier{
		cfg:      defaultNotifyConfig(),
		cap:      200,
		client:   &http.Client{Timeout: 6 * time.Second},
		log:      log,
		dedupe:   map[string]time.Time{},
		webhookQ: make(chan webhookJob, 256),
	}
	for i := 0; i < 2; i++ {
		go n.webhookWorker()
	}
	return n
}

func (n *notifier) configure(c NotifyConfig) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.cfg = c
}

func (n *notifier) enabledFor(kind string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	switch kind {
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
	}
	n.items = append(n.items, item)
	if len(n.items) > n.cap {
		n.items = n.items[len(n.items)-n.cap:]
	}
	hook := n.cfg.WebhookURL
	kindHook := n.cfg.WebhookKind
	sink := n.sink
	n.mu.Unlock()

	if sink != nil {
		sink(level, kind, title, body)
	}
	if hook != "" {
		select {
		case n.webhookQ <- webhookJob{url: hook, kind: kindHook, item: item}:
		default:
			atomic.AddUint64(&n.webhookDropped, 1)
			n.log.Warn("notification webhook queue full; dropping delivery")
		}
	}
}

func (n *notifier) webhookWorker() {
	for job := range n.webhookQ {
		for attempt := 0; attempt < 3; attempt++ {
			retry, err := n.sendWebhook(job.url, job.kind, job.item)
			if err == nil || !retry {
				break
			}
			time.Sleep(time.Duration(1<<attempt) * 250 * time.Millisecond)
		}
	}
}

func (n *notifier) sendWebhook(url, kind string, item notification) (bool, error) {
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
	resp, err := n.client.Do(req)
	if err != nil {
		n.log.Warn("notify webhook failed", "err", err)
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
	return map[string]any{
		"queue_depth":    len(n.webhookQ),
		"queue_capacity": cap(n.webhookQ),
		"dropped":        atomic.LoadUint64(&n.webhookDropped),
	}
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
