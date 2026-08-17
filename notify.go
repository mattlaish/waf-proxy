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
	"log/slog"
	"net/http"
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

type notifier struct {
	mu     sync.Mutex
	cfg    NotifyConfig
	items  []notification
	nextID int64
	cap    int
	client *http.Client
	log    *slog.Logger

	// dedupe map + optional external sink (e.g. syslog)
	dedupe map[string]time.Time
	sink   func(level, kind, title, body string)
}

func newNotifier(log *slog.Logger) *notifier {
	return &notifier{
		cfg:    defaultNotifyConfig(),
		cap:    200,
		client: &http.Client{Timeout: 6 * time.Second},
		log:    log,
		dedupe: map[string]time.Time{},
	}
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
		if exp, ok := n.dedupe[dedupeKey]; ok && time.Now().Before(exp) {
			n.mu.Unlock()
			return
		}
		n.dedupe[dedupeKey] = time.Now().Add(60 * time.Second)
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
		go n.sendWebhook(hook, kindHook, item)
	}
}

func (n *notifier) sendWebhook(url, kind string, item notification) {
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
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := n.client.Do(req)
	if err != nil {
		n.log.Warn("notify webhook failed", "err", err)
		return
	}
	_ = resp.Body.Close()
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

// pending atomic guard so periodic scanners don't stack.
type gate struct{ busy int32 }

func (g *gate) enter() bool { return atomic.CompareAndSwapInt32(&g.busy, 0, 1) }
func (g *gate) leave()      { atomic.StoreInt32(&g.busy, 0) }
