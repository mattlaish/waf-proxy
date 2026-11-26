package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestWorkflowWebhookClearIntentAndHAIsolation(t *testing.T) {
	active := defaultConfig()
	active.Notify.WebhookURL = "https://node.example/secret"
	draft := active
	draft.Notify.WebhookURL = ""
	draft.Notify.ClearWebhook = true
	preserveNotifySecrets(active, &draft)
	if draft.Notify.WebhookURL != "" || active.Notify.WebhookURL == "" {
		t.Fatal("clear intent lost or live config mutated")
	}
	data, err := json.Marshal(redactNotifySecrets(draft))
	if err != nil || !strings.Contains(string(data), `"clear_webhook":true`) || strings.Contains(string(data), "node.example") {
		t.Fatal("draft intent must round-trip without stored secret")
	}
	peer := mergePeerConfig(active, sharedConfigForPeer(draft))
	if peer.Notify.ClearWebhook || peer.Notify.WebhookURL != active.Notify.WebhookURL {
		t.Fatal("node-local clear was applied to HA peer")
	}
	blank := active
	blank.Notify.WebhookURL = ""
	preserveNotifySecrets(active, &blank)
	if blank.Notify.WebhookURL != active.Notify.WebhookURL {
		t.Fatal("blank no longer preserves stored secret")
	}
}

func TestWorkflowNotificationReadAndHandledAreIndependent(t *testing.T) {
	n := newNotifier(slog.New(slog.NewTextHandler(io.Discard, nil)))
	n.push(notifyInfo, "info", "test", "body", "", "", nil)
	id := n.list(1)[0].ID
	if n.list(1)[0].Read || n.list(1)[0].Handled {
		t.Fatal("new alert already marked handled")
	}
	n.markRead(id, false)
	if !n.list(1)[0].Read || n.list(1)[0].Handled {
		t.Fatal("reading silently handles alert")
	}
	if !n.markHandled(id) || !n.list(1)[0].Handled || n.markHandled(id+1) {
		t.Fatal("handled state or missing-id result incorrect")
	}
}

func TestWorkflowDeliveryTestGenerationAndHealth(t *testing.T) {
	engine := newSyslogEngine(slog.New(slog.NewTextHandler(io.Discard, nil)))
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan string, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			done <- ""
			return
		}
		defer conn.Close()
		_ = conn.SetReadDeadline(time.Now().Add(time.Second))
		buf := make([]byte, 2048)
		n, _ := conn.Read(buf)
		done <- string(buf[:n])
	}()
	cfg := defaultSyslogConfig()
	cfg.Enabled = true
	cfg.Protocol = "tcp"
	cfg.Host = "127.0.0.1"
	cfg.Port = listener.Addr().(*net.TCPAddr).Port
	cfg.Facility = 0
	engine.storeConfig(cfg)
	wrong := engine.delivery.current() + 1
	if engine.testForGeneration(&wrong) != errDeliveryConfigChanged {
		t.Fatal("stale test destination not rejected")
	}
	gen := engine.delivery.current()
	if err := engine.testForGeneration(&gen); err != nil {
		t.Fatal(err)
	}
	if line := <-done; !strings.HasPrefix(line, "<6>") {
		t.Fatalf("facility zero not used: %q", line)
	}
	stats := engine.deliveryStats()
	if stats["state"] != "last_send_succeeded" || stats["sent"] != uint64(1) || stats["receipt_verified"] != false {
		t.Fatalf("test health incorrect: %v", stats)
	}
}

func TestWorkflowDeliveryTestsRequireCorrectRole(t *testing.T) {
	n := newNotifier(slog.New(slog.NewTextHandler(io.Discard, nil)))
	a := &adminServer{srv: &server{notify: n, syslog: newSyslogEngine(slog.New(slog.NewTextHandler(io.Discard, nil)))}, token: "admin-test", sessions: newSessionStore(), audit: newAuditLog()}
	h := a.handler()
	for _, path := range []string{"/api/notifications/test", "/api/syslog/test", "/api/notifications/handle"} {
		for _, role := range []string{"", roleViewer, roleReviewer} {
			if path == "/api/notifications/handle" && role == roleReviewer {
				continue
			}
			r := httptest.NewRequest("POST", path, strings.NewReader(`{"id":1}`))
			want := 401
			if role != "" {
				want = 403
				r.Header.Set("Authorization", "Bearer "+a.sessions.create("user", role))
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != want {
				t.Fatalf("path=%s role=%s status=%d want=%d", path, role, w.Code, want)
			}
		}
	}
}

func TestWorkflowWebhookActiveTestAndConflict(t *testing.T) {
	var hits atomic.Int64
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1); w.WriteHeader(204) }))
	defer destination.Close()
	n := newNotifier(slog.New(slog.NewTextHandler(io.Discard, nil)))
	n.configure(NotifyConfig{WebhookURL: destination.URL, WebhookAllowHTTP: true, WebhookAllowPrivate: true})
	a := &adminServer{srv: &server{notify: n}, token: "admin-test", sessions: newSessionStore(), audit: newAuditLog()}
	for _, body := range []string{`{"expected_generation":999}`, `{"expected_generation":1}`, `{"url":"http://attacker.example"}`} {
		r := httptest.NewRequest("POST", "/api/notifications/test", strings.NewReader(body))
		r.Header.Set("Authorization", "Bearer admin-test")
		w := httptest.NewRecorder()
		a.handler().ServeHTTP(w, r)
		want := 409
		if strings.Contains(body, `":1}`) {
			want = 200
		}
		if strings.Contains(body, `"url"`) {
			want = 400
		}
		if w.Code != want {
			t.Fatalf("body=%s code=%d want=%d result=%s", body, w.Code, want, w.Body)
		}
	}
	if hits.Load() != 1 || n.delivery.snapshot(true, 0, 1, 0)["sent"] != uint64(1) {
		t.Fatal("test sent to stale/arbitrary target or failed to record health")
	}
}

func TestWorkflowAttackContextAndTimestampAreStructured(t *testing.T) {
	n := newNotifier(slog.New(slog.NewTextHandler(io.Discard, nil)))
	n.configure(NotifyConfig{OnAttack: true, AttackThreshold: 1})
	now := time.Now().UTC()
	n.observeAttack("site", "block", "", now)
	item := n.list(1)[0]
	if item.Context == nil || item.Context.Site != "site" || item.Context.Kind != "block" || item.Context.EndedAt != now {
		t.Fatal("missing structured investigation context")
	}
	data, err := json.Marshal(accessRec{At: now, Site: "site", Status: 403})
	if err != nil || !strings.Contains(string(data), `"occurred_at"`) {
		t.Fatal("access record has no precise timestamp")
	}
}

func TestWorkflowDeliveryTokenBindsProcessAndGeneration(t *testing.T) {
	var a, b deliveryHealth
	first := a.configToken()
	if first == "" || first != a.configToken() || first == b.configToken() {
		t.Fatal("delivery nonce does not bind the process")
	}
	a.invalidate()
	if first == a.configToken() {
		t.Fatal("token survived configuration change")
	}
	s := newSyslogEngine(slog.New(slog.NewTextHandler(io.Discard, nil)))
	gen := s.delivery.current()
	if s.testForTarget(&gen, "another-node") != errDeliveryConfigChanged {
		t.Fatal("same generation on another node accepted")
	}
}
