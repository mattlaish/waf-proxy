package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func quietNotifier(cfg NotifyConfig) *notifier {
	n := newNotifier(slog.New(slog.NewTextHandler(io.Discard, nil)))
	n.configure(cfg)
	return n
}

func attackAlerts(n *notifier) []notification {
	var out []notification
	for _, it := range n.list(1000) {
		if it.Kind == notifyAttack {
			out = append(out, it)
		}
	}
	return out
}

func TestReviewRateLimitAndAllowlistResponsesDoNotRaiseAttackAlerts(t *testing.T) {
	n := quietNotifier(NotifyConfig{OnAttack: true, AttackThreshold: 3, AttackWindowSeconds: 60})
	s := &server{notify: n, metrics: newMetrics(), access: newAccessRing(16), syslog: newSyslogEngine(nil)}
	l7 := newL7AbuseController(L7AbuseConfig{Enabled: true, RequestsPerWindow: 1, WindowSeconds: 60})
	h := s.logWrap("shop", l7.wrap("shop", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})))
	for i := 0; i < 10; i++ { // one ordinary client bursting: 1 allowed, 9 rate-limited
		req := httptest.NewRequest(http.MethodGet, "http://shop/", nil)
		req.RemoteAddr = "198.51.100.7:1000"
		h.ServeHTTP(httptest.NewRecorder(), req)
	}
	if got := attackAlerts(n); len(got) != 0 {
		t.Fatalf("rate limiting raised an attack alert: %+v", got)
	}
}

func TestReviewAlertThresholdCountsRequestsNotRuleHits(t *testing.T) {
	n := quietNotifier(NotifyConfig{OnAttack: true, AttackThreshold: 3, AttackWindowSeconds: 60, AttackMinSeverity: "CRITICAL"})
	s := &server{notify: n, metrics: newMetrics(), access: newAccessRing(16), syslog: newSyslogEngine(nil)}
	// Each request matches five critical rules, as a typical attack request does.
	h := s.logWrap("shop", http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		signals := r.Context().Value(attackSignalKey{}).(*attackSignals)
		for i := 0; i < 5; i++ {
			signals.noteSeverity("CRITICAL")
		}
		signals.noteSeverity("WARNING")
	}))
	serve := func() { h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "http://shop/", nil)) }
	serve()
	serve()
	if got := attackAlerts(n); len(got) != 0 {
		t.Fatalf("2 requests (10 rule hits) crossed a 3-request threshold: %+v", got)
	}
	serve()
	got := attackAlerts(n)
	if len(got) != 1 || !strings.Contains(got[0].Body, "3 suspicious requests") {
		t.Fatalf("third request should raise exactly one alert: %+v", got)
	}
}

func TestReviewOneAlertPerIncidentWithHourlyFollowUp(t *testing.T) {
	n := quietNotifier(NotifyConfig{OnAttack: true, AttackThreshold: 2, AttackWindowSeconds: 60})
	start := time.Now()
	n.observeAttack("shop", "block", "", start)
	n.observeAttack("shop", "block", "", start)
	if len(attackAlerts(n)) != 1 {
		t.Fatal("first alert missing")
	}
	// A sustained attack: one request every 30s for 90 minutes, never quiet.
	at := start
	for i := 0; i < 180; i++ {
		at = at.Add(30 * time.Second)
		n.observeAttack("shop", "block", "", at)
	}
	alerts := attackAlerts(n)
	if len(alerts) != 2 || alerts[0].Title != "WAF activity still ongoing" {
		t.Fatalf("sustained attack should give the alert plus one hourly follow-up, got %d: %+v", len(alerts), alerts)
	}
	// Quiet for a full window ends the incident; the next burst is a new alert.
	at = at.Add(2 * time.Minute)
	n.observeAttack("shop", "block", "", at)
	n.observeAttack("shop", "block", "", at)
	if got := attackAlerts(n); len(got) != 3 || got[0].Title != "WAF activity threshold reached" {
		t.Fatalf("new incident after a quiet window should alert again: %+v", got)
	}
}

func TestReviewQuietIntegrationIsNotStaleAndDropsAgeOut(t *testing.T) {
	var h deliveryHealth
	h.record(true)
	h.lastSuccess = time.Now().Add(-6 * time.Hour) // healthy, simply nothing to send
	if got := h.snapshot(true, 0, 256, 0)["state"]; got != "last_send_succeeded" {
		t.Fatalf("quiet healthy integration flagged: %v", got)
	}
	h.noteDrop()
	if h.snapshot(true, 0, 256, 1)["recent_drops"] != true {
		t.Fatal("a fresh drop should be flagged")
	}
	h.lastDrop = time.Now().Add(-2 * recentDropWindow)
	if h.snapshot(true, 0, 256, 1)["recent_drops"] != false {
		t.Fatal("an old drop should stop flagging after the recent-drop window")
	}
	h.noteDrop()
	h.invalidate()
	if h.snapshot(true, 0, 256, 1)["recent_drops"] != false {
		t.Fatal("a configuration change should clear the drop flag")
	}
}

func TestReviewFailureReasonIsRecordedWithoutURL(t *testing.T) {
	var h deliveryHealth
	secret := &url.Error{Op: "Post", URL: "https://hooks.example.test/services/T000/SECRET?token=abc", Err: errors.New("dial tcp 203.0.113.9:443: connect: connection refused")}
	h.fail(h.current(), secret)
	got := h.snapshot(true, 0, 1, 0)
	reason, _ := got["last_error"].(string)
	if !strings.Contains(reason, "connection refused") || strings.Contains(reason, "SECRET") || strings.Contains(reason, "token=") {
		t.Fatalf("reason=%q", reason)
	}
	h.record(true)
	if h.snapshot(true, 0, 1, 0)["last_error"] != "" {
		t.Fatal("a successful send should clear the last error")
	}
}

func TestReviewSyslogBacksOffWhileSIEMIsDown(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close() // nothing listens: every dial is refused
	s := newSyslogEngine(slog.New(slog.NewTextHandler(io.Discard, nil)))
	cfg := defaultSyslogConfig()
	cfg.Enabled, cfg.Protocol, cfg.Host, cfg.Port = true, "tcp", "127.0.0.1", port
	s.storeConfig(cfg)
	s.deliver("first")
	s.mu.Lock()
	failures, retryAt := s.dialFailures, s.retryAt
	s.mu.Unlock()
	if failures != 1 || !retryAt.After(time.Now()) {
		t.Fatalf("first failure should start a backoff: failures=%d retryAt=%v", failures, retryAt)
	}
	for i := 0; i < 50; i++ {
		s.deliver("during backoff")
	}
	s.mu.Lock()
	failures = s.dialFailures
	s.mu.Unlock()
	stats := s.deliveryStats()
	if failures != 1 {
		t.Fatalf("lines during backoff must not re-dial: dial failures=%d", failures)
	}
	if stats["dropped"] != uint64(51) || stats["recent_drops"] != true || !strings.Contains(fmt.Sprint(stats["last_error"]), "refused") {
		t.Fatalf("backoff drops must be counted and keep the real reason: %v", stats)
	}
	if stats["failures"] != uint64(1) {
		t.Fatalf("backoff drops are not new delivery attempts: %v", stats["failures"])
	}
}

func TestReviewUDPSyslogIsReportedUnconfirmed(t *testing.T) {
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close()
	s := newSyslogEngine(slog.New(slog.NewTextHandler(io.Discard, nil)))
	cfg := defaultSyslogConfig()
	cfg.Enabled, cfg.Protocol, cfg.Host, cfg.Port = true, "udp", "127.0.0.1", pc.LocalAddr().(*net.UDPAddr).Port
	s.storeConfig(cfg)
	s.deliver("hello")
	if got := s.deliveryStats()["state"]; got != "sent_unconfirmed" {
		t.Fatalf("UDP state=%v", got)
	}
}

func TestReviewWebhookProxyValidation(t *testing.T) {
	for _, bad := range []string{"ftp://proxy:21", "http://user:pass@proxy:3128", "http://proxy:3128/path", "http://proxy:3128?x=1", "proxy:3128"} {
		if validateWebhookProxyURL(bad) == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
	for _, good := range []string{"", "http://proxy.internal:3128", "https://10.0.0.5:8443/"} {
		if err := validateWebhookProxyURL(good); err != nil {
			t.Fatalf("rejected %q: %v", good, err)
		}
	}
}

func TestReviewWebhookProxyIsUsedAndDestinationStillChecked(t *testing.T) {
	var proxied atomic.Int32
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A forward proxy receives the absolute destination URL.
		if r.URL.Host == "hooks.example.test" {
			proxied.Add(1)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer proxy.Close()
	public := func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
	}
	internal := func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("10.1.2.3")}, nil
	}
	var dialer net.Dialer
	policy := NotifyConfig{WebhookProxyURL: proxy.URL, WebhookAllowHTTP: true}
	send := func(lookup func(context.Context, string, string) ([]netip.Addr, error)) error {
		client, transport := webhookClientWithNetwork(policy, lookup, dialer.DialContext)
		defer transport.CloseIdleConnections()
		req, _ := http.NewRequest(http.MethodPost, "http://hooks.example.test/notify", strings.NewReader("{}"))
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
		}
		return err
	}
	if err := send(public); err != nil || proxied.Load() != 1 {
		t.Fatalf("delivery through the proxy failed: err=%v proxied=%d", err, proxied.Load())
	}
	if err := send(internal); err == nil || proxied.Load() != 1 {
		t.Fatalf("a destination resolving to a private address must be refused before the proxy: err=%v proxied=%d", err, proxied.Load())
	}
}

func TestReviewRateLimiterSaturationIsVisible(t *testing.T) {
	c := newL7AbuseController(L7AbuseConfig{Enabled: true, RequestsPerWindow: 1, WindowSeconds: 60})
	if c.saturationStats()["active"] != false {
		t.Fatal("fresh limiter reported saturation")
	}
	now := time.Now()
	const target = uint64(0)
	filled := 0
	for n := 0; filled < l7AbuseMaxEntriesPerShard; n++ {
		id := fmt.Sprintf("198.51.%d.%d", n/256, n%256)
		if l7AbuseShardIndex(abuseClientKey{site: "shop", identity: c.limiterIdentity(id)}) != target {
			continue
		}
		if st, ok := c.allowAt("shop", id, now); ok {
			c.releaseState(st)
		}
		filled++
	}
	tried := 0
	for n := 0; tried < 3; n++ {
		id := fmt.Sprintf("203.0.%d.%d", n/256, n%256)
		if l7AbuseShardIndex(abuseClientKey{site: "shop", identity: c.limiterIdentity(id)}) != target {
			continue
		}
		if st, ok := c.allowAt("shop", id, now); ok {
			c.releaseState(st)
		}
		tried++
	}
	stats := c.saturationStats()
	if stats["active"] != true || stats["overflow_admitted"] != uint64(1) || stats["overflow_refused"] != uint64(2) {
		t.Fatalf("saturation not visible: %v", stats)
	}
}
