package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type attackSignalKey struct{}

// attackSignals is per-request evidence gathered while the request is served:
// whether a security control blocked it, and the highest severity of the WAF
// rules it matched. It lets the alert engine count requests, not rule hits.
type attackSignals struct {
	mu       sync.Mutex
	blocked  bool
	severity string // highest matched rule severity; "" when nothing matched
}

// markAttackBlock records a block by a security control (WAF interruption or
// API schema/identity/GraphQL enforcement). Rate limiting and IP allowlists are
// access controls and deliberately do not call this.
func markAttackBlock(r *http.Request) {
	if r == nil {
		return
	}
	if signals, ok := r.Context().Value(attackSignalKey{}).(*attackSignals); ok {
		signals.mu.Lock()
		signals.blocked = true
		signals.mu.Unlock()
	}
}

func (a *attackSignals) noteSeverity(severity string) {
	rank := severityRank(severity)
	if a == nil || rank < 0 {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if current := severityRank(a.severity); current < 0 || rank < current {
		a.severity = strings.ToUpper(severity)
	}
}

func (a *attackSignals) snapshot() (blocked bool, severity string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.blocked, a.severity
}

func requestAttackSignals(r *http.Request) (*http.Request, *attackSignals) {
	signals := &attackSignals{}
	return r.WithContext(context.WithValue(r.Context(), attackSignalKey{}, signals)), signals
}

// attackWindow tracks one site. Requests are counted inside a window; once an
// alert fires, the site is in an incident until activity stops for one full
// window, with at most one follow-up per attackReminderInterval while it lasts.
type attackWindow struct {
	start         time.Time
	requests      int
	blocks        int
	lastEvent     time.Time
	incidentStart time.Time
	alertedAt     time.Time
	incidentTotal int
}

const attackReminderInterval = time.Hour

func severityRank(value string) int {
	switch strings.ToUpper(value) {
	case "EMERGENCY":
		return 0
	case "ALERT":
		return 1
	case "CRITICAL":
		return 2
	case "ERROR":
		return 3
	case "WARNING":
		return 4
	case "NOTICE":
		return 5
	case "INFO":
		return 6
	case "DEBUG":
		return 7
	}
	return -1
}

func (c NotifyConfig) validate() error {
	if err := validateWebhookURL(c.WebhookURL, c); err != nil {
		return err
	}
	if err := validateWebhookProxyURL(c.WebhookProxyURL); err != nil {
		return err
	}
	if c.AttackThreshold < 0 || c.AttackThreshold > 1000000 {
		return errors.New("notify.attack_threshold must be 0..1000000 (0 defaults to 10)")
	}
	if c.AttackWindowSeconds < 0 || c.AttackWindowSeconds > 86400 {
		return errors.New("notify.attack_window_seconds must be 0..86400 (0 defaults to 60)")
	}
	if c.AttackMinSeverity != "" && severityRank(c.AttackMinSeverity) < 0 {
		return errors.New("notify.attack_min_severity is invalid")
	}
	return nil
}

// observeAttack records one suspicious request for a site: kind "block" when
// a security control blocked it, or "match" with the request's highest matched
// rule severity. It is called at most once per request, so the threshold
// counts requests, not rule hits. No client, URL, rule ID or request content
// is kept, and nothing here makes network calls.
func (n *notifier) observeAttack(site, kind, severity string, now time.Time) {
	if n == nil || !n.attackOn.Load() {
		return
	}
	if kind != "match" && kind != "block" {
		return
	}
	n.mu.Lock()
	cfg := n.cfg
	minimum := cfg.AttackMinSeverity
	if minimum == "" {
		minimum = "CRITICAL"
	}
	if kind == "match" && (severityRank(severity) < 0 || severityRank(severity) > severityRank(minimum)) {
		n.mu.Unlock()
		return
	}
	window := time.Duration(cfg.AttackWindowSeconds) * time.Second
	if window == 0 {
		window = time.Minute
	}
	threshold := cfg.AttackThreshold
	if threshold == 0 {
		threshold = 10
	}
	if n.attacks == nil {
		n.attacks = make(map[string]*attackWindow)
	}
	state := n.attacks[site]
	if state == nil {
		for k, old := range n.attacks {
			if !now.Before(old.lastEvent.Add(window)) {
				delete(n.attacks, k)
			}
		}
		if len(n.attacks) >= 4096 {
			n.mu.Unlock()
			return
		}
		state = &attackWindow{start: now, lastEvent: now}
		n.attacks[site] = state
	}

	// An open incident ends after one full window without suspicious requests.
	if !state.alertedAt.IsZero() && !now.Before(state.lastEvent.Add(window)) {
		*state = attackWindow{start: now, lastEvent: now}
	}
	if !state.alertedAt.IsZero() {
		state.lastEvent = now
		state.incidentTotal++
		if kind == "block" {
			state.blocks++
		}
		if now.Sub(state.alertedAt) < attackReminderInterval {
			n.mu.Unlock()
			return
		}
		state.alertedAt = now
		total, blocks, startedAt := state.incidentTotal, state.blocks, state.incidentStart
		n.mu.Unlock()
		n.pushContext(notifyAttack, "alert", "WAF activity still ongoing",
			fmt.Sprintf("Site %s: %d suspicious requests (%d blocked) since this incident started; matches at or above %s. Matches are not necessarily blocked requests.", site, total, blocks, minimum), "", "", nil,
			&notificationContext{Site: site, Kind: attackContextKind(blocks), StartedAt: startedAt.UTC(), EndedAt: now.UTC()})
		return
	}

	if !now.Before(state.start.Add(window)) {
		*state = attackWindow{start: now}
	}
	state.lastEvent = now
	state.requests++
	if kind == "block" {
		state.blocks++
	}
	if state.requests < threshold {
		n.mu.Unlock()
		return
	}
	state.alertedAt = now
	state.incidentStart = state.start
	state.incidentTotal = state.requests
	requests, blocks, startedAt := state.requests, state.blocks, state.start
	n.mu.Unlock()
	// Incident-level suppression above is authoritative; do not use the generic
	// dedupe map, which could shorten a deliberately configured window.
	n.pushContext(notifyAttack, "alert", "WAF activity threshold reached",
		fmt.Sprintf("Site %s: %d suspicious requests in %ds (%d blocked); matches at or above %s. One alert per incident; a follow-up is sent hourly while it continues. Matches are not necessarily blocked requests.", site, requests, int(window.Seconds()), blocks, minimum), "", "", nil,
		&notificationContext{Site: site, Kind: attackContextKind(blocks), StartedAt: startedAt.UTC(), EndedAt: now.UTC()})
}

func attackContextKind(blocks int) string {
	if blocks > 0 {
		return "block"
	}
	return "match"
}
