package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type attackSignalKey struct{}
type attackSignals struct {
	blocked bool
}

func markAttackBlock(r *http.Request) {
	if signals, ok := r.Context().Value(attackSignalKey{}).(*attackSignals); ok {
		signals.blocked = true
	}
}

func requestAttackSignals(r *http.Request) (*http.Request, *attackSignals) {
	signals := &attackSignals{}
	return r.WithContext(context.WithValue(r.Context(), attackSignalKey{}, signals)), signals
}

type attackWindow struct {
	start   time.Time
	matches int
	blocks  int
	emitted bool
}

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

// Counts matches and blocks independently, with one emitted alert per site/window.
// There are no client/URL/rule-ID keys, raw request values or network calls here.
func (n *notifier) observeAttack(site, kind, severity string, now time.Time) {
	if n == nil {
		return
	}
	n.mu.Lock()
	cfg := n.cfg
	if !cfg.OnAttack {
		n.mu.Unlock()
		return
	}
	minimum := cfg.AttackMinSeverity
	if minimum == "" {
		minimum = "CRITICAL"
	}
	if kind == "match" && (severityRank(severity) < 0 || severityRank(severity) > severityRank(minimum)) {
		n.mu.Unlock()
		return
	}
	if kind != "match" && kind != "block" {
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
	key := site
	state := n.attacks[key]
	if state == nil {
		for k, old := range n.attacks {
			if !now.Before(old.start.Add(window)) {
				delete(n.attacks, k)
			}
		}
		if len(n.attacks) >= 4096 {
			n.mu.Unlock()
			return
		}
		state = &attackWindow{start: now}
		n.attacks[key] = state
	}
	if !now.Before(state.start.Add(window)) {
		*state = attackWindow{start: now}
	}
	if state.emitted {
		n.mu.Unlock()
		return
	}
	count := 0
	if kind == "match" {
		state.matches++
		count = state.matches
	} else {
		state.blocks++
		count = state.blocks
	}
	if count < threshold {
		n.mu.Unlock()
		return
	}
	state.emitted = true
	startedAt := state.start
	n.mu.Unlock()
	// Window-level suppression above is authoritative; do not use the generic
	// dedupe map which can shorten a deliberately configured alert window.
	n.pushContext(notifyAttack, "alert", "WAF activity threshold reached",
		fmt.Sprintf("Site %s: %d %s events in %ds; minimum match severity %s. Matches are not necessarily blocked requests.", site, count, kind, int(window.Seconds()), minimum), "", "", nil,
		&notificationContext{Site: site, Kind: kind, StartedAt: startedAt.UTC(), EndedAt: now.UTC()})
}
