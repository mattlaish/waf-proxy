package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/corazawaf/coraza/v3"
	txhttp "github.com/corazawaf/coraza/v3/http"
)

func TestSecurityIPv6RotationAndMappedAddresses(t *testing.T) {
	c := newL7AbuseController(L7AbuseConfig{Enabled: true, RequestsPerWindow: 1, TLSHandshakePerWindow: 1})
	now := time.Now()
	st, ok := c.allowAt("a", "[2001:db8::1]:1000", now)
	if !ok {
		t.Fatal("first IPv6 request denied")
	}
	c.releaseState(st)
	if _, ok = c.allowAt("a", "2001:db8::ffff", now); ok {
		t.Fatal("same /64 got a new bucket")
	}
	if !c.allowTLSHandshakeAt(":443", "2001:db8:1::1", now) || c.allowTLSHandshakeAt(":443", "2001:db8:1::2", now) {
		t.Fatal("TLS IPv6 rotation")
	}
	if c.limiterIdentity("::ffff:192.0.2.1") != c.limiterIdentity("192.0.2.1") {
		t.Fatal("mapped IPv4 identity mismatch")
	}
	c.cfg.IPv6PrefixLength = 128
	if c.limiterIdentity("2001:db8::1") == c.limiterIdentity("2001:db8::2") {
		t.Fatal("explicit /128 trade-off not honored")
	}
}

func TestSecurityOverflowIsSiteAndListenerScoped(t *testing.T) {
	c := newL7AbuseController(L7AbuseConfig{Enabled: true, RequestsPerWindow: 1, MaxConcurrentConnections: 1, TLSHandshakePerWindow: 1})
	now := time.Now()
	// Saturate every shard with recent state; no eviction may reset allowances.
	for i := range c.shards {
		for j := 0; j < l7AbuseMaxEntriesPerShard; j++ {
			key := abuseClientKey{site: "filled", identity: string(rune(j))}
			c.shards[i].clients[key] = &abuseClientState{lastSeen: now}
			c.tlsShards[i].clients[key] = &abuseClientState{lastSeen: now}
		}
	}
	a, ok := c.allowAt("a", "192.0.2.1", now)
	if !ok {
		t.Fatal("first site overflow denied")
	}
	if _, ok := c.allowAt("a", "192.0.2.2", now); ok {
		t.Fatal("overflow reset across identities/shards")
	}
	b, ok := c.allowAt("b", "192.0.2.2", now)
	if !ok || a == b {
		t.Fatal("cross-site overflow/concurrency coupling")
	}
	c.releaseState(a)
	c.releaseState(b)
	if !c.allowTLSHandshakeAt(":443", "192.0.2.1", now) || c.allowTLSHandshakeAt(":443", "192.0.2.2", now) || !c.allowTLSHandshakeAt(":8443", "192.0.2.2", now) {
		t.Fatal("TLS listener isolation")
	}
	for i := 0; i < 9000; i++ {
		c.overflowState(string(rune(i)), "http")
	}
	if len(c.overflows) != 8192 || c.overflowState("new scope", "http") != nil {
		t.Fatal("unbounded overflow memory")
	}
}

func TestSecurityWebhookDestinationPolicy(t *testing.T) {
	for _, raw := range []string{"http://example.com/hook", "https://user:pass@example.com", "https://127.0.0.1/hook", "https://[::ffff:127.0.0.1]/", "https://169.254.169.254/", "file:///tmp/hook", "https://example.com/#fragment"} {
		if validateWebhookURL(raw, NotifyConfig{}) == nil {
			t.Fatalf("accepted unsafe URL %s", raw)
		}
	}
	for _, raw := range []string{"https://example.com/hook?key=secret", "https://8.8.8.8/hook"} {
		if err := validateWebhookURL(raw, NotifyConfig{}); err != nil {
			t.Fatal(err)
		}
	}
	if err := validateWebhookURL("http://127.0.0.1/hook", NotifyConfig{WebhookAllowHTTP: true, WebhookAllowPrivate: true}); err != nil {
		t.Fatal(err)
	}
	for _, ip := range []string{"::", "::1", "127.0.0.1", "10.0.0.1", "169.254.169.254", "100.64.0.1", "64:ff9b::a00:1", "2002:a00:1::1", "::ffff:10.0.0.1"} {
		if webhookIPAllowed(netip.MustParseAddr(ip), false) {
			t.Fatalf("unsafe resolved IP %s", ip)
		}
	}
	if webhookIPAllowed(netip.MustParseAddr("169.254.169.254"), true) {
		t.Fatal("metadata exception is too broad")
	}
	client, transport := webhookClient(NotifyConfig{})
	defer transport.CloseIdleConnections()
	if transport.Proxy != nil {
		t.Fatal("environment proxy can bypass destination policy")
	}
	if _, err := transport.DialContext(context.Background(), "tcp", "localhost:80"); err == nil {
		t.Fatal("DNS loopback not blocked")
	}
	if client.CheckRedirect(nil, nil) != http.ErrUseLastResponse {
		t.Fatal("redirects not refused")
	}
}

func TestSecurityWebhookRedirectAndDeliveryHealth(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	n := newNotifier(log)
	var targetHits atomic.Int64
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { targetHits.Add(1); w.WriteHeader(204) }))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 302) }))
	defer redirect.Close()
	policy := NotifyConfig{WebhookAllowHTTP: true, WebhookAllowPrivate: true}
	if _, err := n.sendWebhook(redirect.URL, "generic", notification{}, policy); err == nil {
		t.Fatal("redirect treated as success")
	}
	if targetHits.Load() != 0 {
		t.Fatal("alert followed redirect")
	}
	if _, err := n.sendWebhook(target.URL, "generic", notification{}, policy); err != nil {
		t.Fatal(err)
	}
	n.cfg.WebhookURL = target.URL
	stats := n.webhookDeliveryStats()
	if stats["state"] != "last_send_succeeded" || stats["sent"] != uint64(1) || stats["failures"] != uint64(1) || stats["last_success"] == "" {
		t.Fatalf("delivery health incorrect: %v", stats)
	}
}

func TestSecurityAttackAlertsThresholdFilterAndPrivacy(t *testing.T) {
	n := newNotifier(slog.New(slog.NewTextHandler(io.Discard, nil)))
	n.configure(NotifyConfig{OnAttack: true, AttackThreshold: 2, AttackWindowSeconds: 60, AttackMinSeverity: "CRITICAL"})
	now := time.Now()
	n.observeAttack("a", "match", "WARNING", now)
	n.observeAttack("a", "match", "CRITICAL", now)
	if len(n.list(100)) != 0 {
		t.Fatal("early/low severity alert")
	}
	n.observeAttack("a", "match", "ALERT", now)
	for i := 0; i < 20; i++ {
		n.observeAttack("a", "match", "CRITICAL", now)
	}
	if len(n.list(100)) != 1 {
		t.Fatal("attack ticket storm")
	}
	n.observeAttack("a", "block", "", now)
	n.observeAttack("a", "block", "", now)
	if len(n.list(100)) != 1 {
		t.Fatal("one attack created both match and block tickets")
	}
	n.observeAttack("b", "block", "", now)
	n.observeAttack("b", "block", "", now)
	if len(n.list(100)) != 2 {
		t.Fatal("site/block threshold not independent")
	}
	n.observeAttack("a", "match", "CRITICAL", now.Add(time.Minute))
	n.observeAttack("a", "match", "CRITICAL", now.Add(time.Minute))
	if len(n.list(100)) != 3 {
		t.Fatal("next period alert missing")
	}
	for _, item := range n.list(100) {
		if item.Kind != notifyAttack || item.Payload != nil || strings.Contains(item.Body, "password=") {
			t.Fatal("unexpected alert payload")
		}
	}
	n.configure(NotifyConfig{})
	n.observeAttack("a", "block", "", now)
	if len(n.list(100)) != 3 {
		t.Fatal("off switch ineffective")
	}
}

func TestSecurityBlockAlertsExcludeApplicationResponses(t *testing.T) {
	r, signals := requestAttackSignals(httptest.NewRequest("GET", "/", nil))
	http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(403) }).ServeHTTP(httptest.NewRecorder(), r)
	if signals.blocked {
		t.Fatal("application 403 misclassified")
	}
	writeBlockResponse(httptest.NewRecorder(), r, 429, "l7_abuse_limit", true)
	if !signals.blocked {
		t.Fatal("explicit WAF block missed")
	}
}

func TestSecuritySyslogFailuresAndQueueDropsVisible(t *testing.T) {
	s := newSyslogEngine(slog.New(slog.NewTextHandler(io.Discard, nil)))
	cfg := defaultSyslogConfig()
	cfg.Enabled = true
	cfg.Protocol = "tcp"
	s.storeConfig(cfg)
	left, right := net.Pipe()
	right.Close()
	s.conn = left
	s.deliver("test")
	if stats := s.deliveryStats(); stats["state"] != "degraded" || stats["dropped"] != uint64(1) || stats["failures"] != uint64(1) {
		t.Fatalf("lost SIEM write invisible: %v", stats)
	}
	for i := 0; i < cap(s.queue)+1; i++ {
		s.enqueue("line")
	}
	if s.deliveryStats()["dropped"] != uint64(2) {
		t.Fatal("queue loss invisible")
	}
}

func TestSecurityDeliveryReconfigureDoesNotReuseOldSuccess(t *testing.T) {
	var health deliveryHealth
	old := health.current()
	health.recordAt(old, true)
	health.invalidate()
	health.recordAt(old, true) // Completion from an in-flight old endpoint.
	if stats := health.snapshot(true, 0, 1, 0); stats["state"] != "unverified" || stats["last_success"] != "" {
		t.Fatalf("old endpoint falsely verifies new endpoint: %v", stats)
	}
	health.record(false)
	if health.snapshot(true, 0, 1, 1)["state"] != "degraded" {
		t.Fatal("queue loss should degrade health")
	}
}

func TestSecurityWebhookDNSAnswersCheckedAndPinned(t *testing.T) {
	for _, mixed := range []bool{false, true} {
		lookups, dials := 0, 0
		lookup := func(context.Context, string, string) ([]netip.Addr, error) {
			lookups++
			if mixed || lookups > 1 {
				return []netip.Addr{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("127.0.0.1")}, nil
			}
			return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
		}
		dial := func(_ context.Context, _ string, address string) (net.Conn, error) {
			dials++
			if address != "8.8.8.8:443" {
				t.Fatalf("not pinned to checked IP: %s", address)
			}
			return nil, errors.New("test does not open a connection")
		}
		_, transport := webhookClientWithNetwork(NotifyConfig{}, lookup, dial)
		_, _ = transport.DialContext(context.Background(), "tcp", "hooks.example:443")
		transport.CloseIdleConnections()
		if lookups != 1 || mixed && dials != 0 || !mixed && dials != 1 {
			t.Fatalf("DNS check/pinning: mixed=%t lookups=%d dials=%d", mixed, lookups, dials)
		}
	}
}

func TestSecurityNotifyValidationAndHASettings(t *testing.T) {
	for _, cfg := range []NotifyConfig{{WebhookURL: "http://example.com"}, {AttackThreshold: -1}, {AttackWindowSeconds: 86401}, {AttackMinSeverity: "unrecognized"}} {
		if cfg.validate() == nil {
			t.Fatalf("unsafe configuration accepted: %+v", cfg)
		}
	}
	local := defaultConfig()
	local.Notify.WebhookURL = "https://local.example/hook"
	peer := local
	peer.Notify.OnAttack = false
	peer.Notify.WebhookAllowPrivate = true
	peer.Notify.WebhookURL = "https://peer.example/hook"
	merged := mergePeerConfig(local, sharedConfigForPeer(peer))
	if merged.Notify.OnAttack || !merged.Notify.WebhookAllowPrivate || merged.Notify.WebhookURL != local.Notify.WebhookURL {
		t.Fatal("HA setting scope differs from console explanation")
	}
}

func TestSecurityAlertStateIsBounded(t *testing.T) {
	n := newNotifier(slog.New(slog.NewTextHandler(io.Discard, nil)))
	n.configure(NotifyConfig{OnAttack: true, AttackThreshold: 100})
	now := time.Now()
	for i := 0; i < 5000; i++ {
		n.observeAttack(string(rune(i)), "block", "", now)
	}
	if len(n.attacks) != 4096 {
		t.Fatal("unbounded alert aggregation")
	}
}

func TestSecurityRealCorazaBlockVersusMonitor(t *testing.T) {
	for _, mode := range []string{"On", "DetectionOnly"} {
		waf, err := coraza.NewWAF(coraza.NewWAFConfig().WithDirectives("SecRuleEngine " + mode + "\nSecRule REQUEST_URI \"@beginsWith /blocked\" \"id:100001,phase:1,deny,status:403,log,msg:'test'\""))
		if err != nil {
			t.Fatal(err)
		}
		upstream := false
		backend := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { upstream = true; w.WriteHeader(403) })
		r, signals := requestAttackSignals(httptest.NewRequest("GET", "/blocked", nil))
		r.RemoteAddr = "192.0.2.1:1234"
		rec := httptest.NewRecorder()
		txhttp.WrapHandler(observeCorazaWAF(waf, nil), backend).ServeHTTP(rec, r)
		if rec.Code != 403 || signals.blocked != (mode == "On") || upstream != (mode == "DetectionOnly") {
			t.Fatalf("mode=%s code=%d signals=%+v", mode, rec.Code, signals)
		}
	}
}

func TestSecurityRealCorazaResponseBlock(t *testing.T) {
	waf, err := coraza.NewWAF(coraza.NewWAFConfig().WithDirectives("SecRuleEngine On\nSecRule RESPONSE_STATUS \"@streq 200\" \"id:100002,phase:3,deny,status:403,log,msg:'response test'\""))
	if err != nil {
		t.Fatal(err)
	}
	r, signals := requestAttackSignals(httptest.NewRequest("GET", "/", nil))
	rec := httptest.NewRecorder()
	txhttp.WrapHandler(observeCorazaWAF(waf, nil), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })).ServeHTTP(rec, r)
	if !signals.blocked || rec.Code != 403 {
		t.Fatalf("response-phase block missed: %+v status=%d", signals, rec.Code)
	}
}
