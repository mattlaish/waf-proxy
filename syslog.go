package main

// Syslog forwarding to an external SIEM.
//
// Design posture (healthcare environment):
//   * Async + fail-open. A dead or slow SIEM never touches the request path.
//     Events go through a bounded queue with drop-on-full; a single background
//     writer owns the connection and reconnects on failure.
//   * TLS-capable. WAF logs can carry URIs, client IPs, and matched payload
//     fragments (potential PHI), so cleartext off-box is discouraged; TCP+TLS
//     is supported and recommended.
//   * PHI-conservative. The match "data" fragment is omitted by default and can
//     only be included (truncated) via an explicit opt-in.
//
// State is transient (a connection + a queue); nothing persisted.

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type SyslogConfig struct {
	Enabled       bool   `json:"enabled"`
	Host          string `json:"host"`
	Port          int    `json:"port"`
	Protocol      string `json:"protocol"` // udp | tcp | tls
	Format        string `json:"format"`   // rfc5424 (default) | rfc3164
	Facility      int    `json:"facility"` // 0-23 (default 16 = local0)
	AppName       string `json:"app_name"`
	TLSSkipVerify bool   `json:"tls_skip_verify"` // for private CAs / self-signed collectors

	// which streams to forward
	SendWAF    bool `json:"send_waf"`
	SendAccess bool `json:"send_access"`
	SendAudit  bool `json:"send_audit"`
	SendNotify bool `json:"send_notify"`

	// PHI control: include the (truncated) matched payload fragment
	IncludeMatchData bool `json:"include_match_data"`
}

func defaultSyslogConfig() SyslogConfig {
	return SyslogConfig{
		Enabled: false, Port: 6514, Protocol: "tls", Format: "rfc5424",
		Facility: 16, AppName: "waf-proxy",
		SendWAF: true, SendAudit: true, SendNotify: true, SendAccess: false,
		IncludeMatchData: false,
	}
}

func (c SyslogConfig) validate() error {
	if !c.Enabled {
		return nil
	}
	if strings.TrimSpace(c.Host) == "" {
		return fmt.Errorf("syslog: host is required when enabled")
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("syslog: port must be 1-65535")
	}
	switch c.Protocol {
	case "udp", "tcp", "tls":
	default:
		return fmt.Errorf("syslog: protocol must be udp, tcp, or tls")
	}
	if c.Format != "" && c.Format != "rfc5424" && c.Format != "rfc3164" {
		return fmt.Errorf("syslog: format must be rfc5424 or rfc3164")
	}
	if c.Facility < 0 || c.Facility > 23 {
		return fmt.Errorf("syslog: facility must be 0-23")
	}
	return nil
}

// severity levels (syslog numeric)
const (
	sylEmerg   = 0
	sylAlert   = 1
	sylCrit    = 2
	sylErr     = 3
	sylWarning = 4
	sylNotice  = 5
	sylInfo    = 6
	sylDebug   = 7
)

type syslogEngine struct {
	mu   sync.Mutex
	cfg  atomic.Pointer[SyslogConfig]
	log  *slog.Logger
	host string

	enabled       atomic.Bool
	wafEnabled    atomic.Bool
	accessEnabled atomic.Bool
	auditEnabled  atomic.Bool
	notifyEnabled atomic.Bool

	queue   chan string
	conn    net.Conn
	running int32
	cancel  chan struct{}
	dropped uint64
}

func newSyslogEngine(log *slog.Logger) *syslogEngine {
	h, _ := os.Hostname()
	if h == "" {
		h = "waf-proxy"
	}
	e := &syslogEngine{
		log:    log,
		host:   h,
		queue:  make(chan string, 4096),
		cancel: make(chan struct{}),
	}
	e.storeConfig(defaultSyslogConfig())
	return e
}

func (s *syslogEngine) storeConfig(c SyslogConfig) {
	cfg := c
	s.cfg.Store(&cfg)
	s.enabled.Store(c.Enabled)
	s.wafEnabled.Store(c.Enabled && c.SendWAF)
	s.accessEnabled.Store(c.Enabled && c.SendAccess)
	s.auditEnabled.Store(c.Enabled && c.SendAudit)
	s.notifyEnabled.Store(c.Enabled && c.SendNotify)
}

func (s *syslogEngine) configure(c SyslogConfig) {
	s.mu.Lock()
	// Publish the immutable snapshot while the connection is serialized, then
	// force a reconnect so the next background delivery uses the new endpoint.
	s.storeConfig(c)
	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}
	s.mu.Unlock()
	if c.Enabled && atomic.CompareAndSwapInt32(&s.running, 0, 1) {
		go s.writer()
	}
}

func (s *syslogEngine) snapshotCfg() SyslogConfig {
	if cfg := s.cfg.Load(); cfg != nil {
		return *cfg
	}
	return defaultSyslogConfig()
}

// enqueue is non-blocking: drop rather than ever block a caller (request path,
// match callback, etc).
func (s *syslogEngine) enqueue(line string) {
	select {
	case s.queue <- line:
	default:
		atomic.AddUint64(&s.dropped, 1)
	}
}

func (s *syslogEngine) writer() {
	for {
		select {
		case <-s.cancel:
			return
		case line := <-s.queue:
			s.deliver(line)
		}
	}
}

func (s *syslogEngine) dial(cfg SyslogConfig) (net.Conn, error) {
	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	d := &net.Dialer{Timeout: 5 * time.Second}
	switch cfg.Protocol {
	case "udp":
		return d.Dial("udp", addr)
	case "tls":
		return tls.DialWithDialer(d, "tcp", addr, &tls.Config{
			ServerName:         cfg.Host,
			InsecureSkipVerify: cfg.TLSSkipVerify, //nolint:gosec // operator opt-in for private CAs
		})
	default:
		return d.Dial("tcp", addr)
	}
}

func (s *syslogEngine) deliver(line string) {
	cfg := s.snapshotCfg()
	if !cfg.Enabled {
		return
	}
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()

	if conn == nil {
		c, err := s.dial(cfg)
		if err != nil {
			s.log.Warn("syslog dial failed", "err", err) // fail-open: drop this line
			return
		}
		s.mu.Lock()
		s.conn = c
		conn = c
		s.mu.Unlock()
	}

	// TCP/TLS syslog frames are newline-terminated; UDP is a datagram.
	payload := line
	if cfg.Protocol != "udp" {
		payload = line + "\n"
	}
	_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if _, err := conn.Write([]byte(payload)); err != nil {
		s.log.Warn("syslog write failed; will reconnect", "err", err)
		s.mu.Lock()
		if s.conn != nil {
			s.conn.Close()
			s.conn = nil
		}
		s.mu.Unlock()
	}
}

// ── message formatting ──────────────────────────────────────────────────

func (s *syslogEngine) pri(cfg SyslogConfig, severity int) int { return cfg.Facility*8 + severity }

// emit formats and enqueues a message. msgID groups events (WAF, ACCESS, ...).
func (s *syslogEngine) emit(severity int, msgID, msg string) {
	if !s.enabled.Load() {
		return
	}
	s.emitWithConfig(s.snapshotCfg(), severity, msgID, msg)
}

func (s *syslogEngine) emitWithConfig(cfg SyslogConfig, severity int, msgID, msg string) {
	app := cfg.AppName
	if app == "" {
		app = "waf-proxy"
	}
	pri := s.pri(cfg, severity)
	var line string
	if cfg.Format == "rfc3164" {
		// <PRI>Mmm dd hh:mm:ss host tag: msg
		line = fmt.Sprintf("<%d>%s %s %s: %s",
			pri, time.Now().Format("Jan _2 15:04:05"), s.host, app, msg)
	} else {
		// RFC 5424: <PRI>1 TIMESTAMP HOST APP PROCID MSGID SD MSG
		line = fmt.Sprintf("<%d>1 %s %s %s %d %s - %s",
			pri, time.Now().UTC().Format(time.RFC3339), s.host, app, os.Getpid(), msgID, msg)
	}
	s.enqueue(line)
}

func kv(pairs ...string) string {
	var b strings.Builder
	for i := 0; i+1 < len(pairs); i += 2 {
		if i > 0 {
			b.WriteByte(' ')
		}
		v := pairs[i+1]
		if strings.ContainsAny(v, " \"") {
			v = "\"" + strings.ReplaceAll(v, "\"", "'") + "\""
		}
		fmt.Fprintf(&b, "%s=%s", pairs[i], v)
	}
	return b.String()
}

func sevFromString(s string) int {
	switch strings.ToUpper(s) {
	case "CRITICAL", "EMERGENCY":
		return sylCrit
	case "ERROR":
		return sylErr
	case "WARNING":
		return sylWarning
	case "NOTICE":
		return sylNotice
	default:
		return sylInfo
	}
}

// ── stream hooks (called from the event sources) ─────────────────────────

func (s *syslogEngine) forwardMatch(m matchRec) {
	if !s.wafEnabled.Load() {
		return
	}
	cfg := s.snapshotCfg()
	if !cfg.Enabled || !cfg.SendWAF {
		return
	}
	pairs := []string{
		"event", "waf_match", "site", m.Site, "client", m.Client,
		"rule_id", fmt.Sprintf("%d", m.RuleID), "severity", m.Severity,
		"phase", fmt.Sprintf("%d", m.Phase), "uri", m.URI, "msg", m.Msg,
	}
	if cfg.IncludeMatchData && m.Data != "" {
		d := m.Data
		if len(d) > 200 {
			d = d[:200]
		}
		pairs = append(pairs, "data", d)
	}
	s.emitWithConfig(cfg, sevFromString(m.Severity), "WAF", kv(pairs...))
}

func (s *syslogEngine) forwardAccess(a accessRec) {
	if !s.accessEnabled.Load() {
		return
	}
	cfg := s.snapshotCfg()
	if !cfg.Enabled || !cfg.SendAccess {
		return
	}
	sev := sylInfo
	if a.Status >= 500 {
		sev = sylErr
	} else if a.Status >= 400 {
		sev = sylWarning
	}
	s.emitWithConfig(cfg, sev, "ACCESS", kv("event", "access", "site", a.Site, "client", a.Client,
		"method", a.Method, "path", a.Path, "status", fmt.Sprintf("%d", a.Status)))
}

func (s *syslogEngine) forwardAudit(user, action, detail string) {
	if !s.auditEnabled.Load() {
		return
	}
	cfg := s.snapshotCfg()
	if !cfg.Enabled || !cfg.SendAudit {
		return
	}
	s.emitWithConfig(cfg, sylNotice, "AUDIT", kv("event", "audit", "user", user, "action", action, "detail", detail))
}

func (s *syslogEngine) forwardNotify(level, kind, title, body string) {
	if !s.notifyEnabled.Load() {
		return
	}
	cfg := s.snapshotCfg()
	if !cfg.Enabled || !cfg.SendNotify {
		return
	}
	sev := sylNotice
	switch level {
	case "alert":
		sev = sylAlert
	case "warn":
		sev = sylWarning
	}
	s.emitWithConfig(cfg, sev, "NOTIFY", kv("event", "notify", "kind", kind, "title", title, "detail", body))
}

// test sends a single message immediately (synchronously) to validate config.
func (s *syslogEngine) test() error {
	cfg := s.snapshotCfg()
	if !cfg.Enabled {
		return fmt.Errorf("syslog is disabled — enable and save first")
	}
	conn, err := s.dial(cfg)
	if err != nil {
		return err
	}
	defer conn.Close()
	app := cfg.AppName
	if app == "" {
		app = "waf-proxy"
	}
	msg := kv("event", "test", "msg", "waf-proxy syslog test message")
	var line string
	if cfg.Format == "rfc3164" {
		line = fmt.Sprintf("<%d>%s %s %s: %s", s.pri(cfg, sylInfo), time.Now().Format("Jan _2 15:04:05"), s.host, app, msg)
	} else {
		line = fmt.Sprintf("<%d>1 %s %s %s %d TEST - %s", s.pri(cfg, sylInfo), time.Now().UTC().Format(time.RFC3339), s.host, app, os.Getpid(), msg)
	}
	if cfg.Protocol != "udp" {
		line += "\n"
	}
	_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, err = conn.Write([]byte(line))
	return err
}
