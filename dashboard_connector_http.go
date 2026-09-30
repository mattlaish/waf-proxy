package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type owiConnectorConfig struct {
	Enabled                bool
	Addr                   string
	TLSCert                string
	TLSKey                 string
	AllowInsecureTest      bool
	SourceInstanceID       string
	TokenRegistryPath      string
	CursorSecretFile       string
	StatePath              string
	RatePerSecond          float64
	Burst                  int
	PerPrincipalConcurrent int
	GlobalConcurrent       int
	RequestDeadline        time.Duration
	MaxPageSize            int
	MaxResponseBytes       int
	SnapshotMaxCount       int
	SnapshotMaxRecords     int
	SnapshotMaxBytes       int64
	ExportMaxRows          int
	ExportMaxBytes         int64
}

func loadOWIConnectorConfig(configPath string) (owiConnectorConfig, error) {
	cfg := owiConnectorConfig{
		Enabled:                envBool("WAF_DASHBOARD_READER_ENABLED", false),
		Addr:                   envOr("WAF_DASHBOARD_READER_ADDR", "127.0.0.1:19405"),
		TLSCert:                strings.TrimSpace(os.Getenv("WAF_DASHBOARD_READER_TLS_CERT")),
		TLSKey:                 strings.TrimSpace(os.Getenv("WAF_DASHBOARD_READER_TLS_KEY")),
		AllowInsecureTest:      envBool("WAF_DASHBOARD_ALLOW_INSECURE_TEST", false),
		SourceInstanceID:       strings.TrimSpace(os.Getenv("WAF_DASHBOARD_SOURCE_INSTANCE_ID")),
		TokenRegistryPath:      strings.TrimSpace(os.Getenv("WAF_DASHBOARD_READER_TOKEN_REGISTRY")),
		CursorSecretFile:       strings.TrimSpace(os.Getenv("WAF_DASHBOARD_CURSOR_SECRET_FILE")),
		StatePath:              strings.TrimSpace(os.Getenv("WAF_DASHBOARD_STATE_FILE")),
		RatePerSecond:          envFloat("WAF_DASHBOARD_RATE_PER_SECOND", 10),
		Burst:                  envInt("WAF_DASHBOARD_RATE_BURST", 20),
		PerPrincipalConcurrent: envInt("WAF_DASHBOARD_PER_PRINCIPAL_CONCURRENCY", 2),
		GlobalConcurrent:       envInt("WAF_DASHBOARD_GLOBAL_CONCURRENCY", 16),
		RequestDeadline:        time.Duration(envInt("WAF_DASHBOARD_REQUEST_DEADLINE_SECONDS", 8)) * time.Second,
		MaxPageSize:            envInt("WAF_DASHBOARD_MAX_PAGE_SIZE", 500),
		MaxResponseBytes:       envInt("WAF_DASHBOARD_MAX_RESPONSE_BYTES", 4<<20),
		SnapshotMaxCount:       envInt("WAF_DASHBOARD_SNAPSHOT_MAX_COUNT", 256),
		SnapshotMaxRecords:     envInt("WAF_DASHBOARD_SNAPSHOT_MAX_RECORDS", 10000),
		SnapshotMaxBytes:       int64(envInt("WAF_DASHBOARD_SNAPSHOT_MAX_BYTES", 32<<20)),
		ExportMaxRows:          envInt("WAF_DASHBOARD_EXPORT_MAX_ROWS", 200000),
		ExportMaxBytes:         int64(envInt("WAF_DASHBOARD_EXPORT_MAX_BYTES", 128<<20)),
	}
	if cfg.TokenRegistryPath == "" {
		cfg.TokenRegistryPath = filepath.Join(filepath.Dir(configPath), "dashboard-reader-tokens.json")
	}
	if cfg.StatePath == "" {
		cfg.StatePath = defaultOWIStatePath(configPath)
	}
	if !cfg.Enabled {
		return cfg, nil
	}
	if !validOWIID(cfg.SourceInstanceID) {
		return cfg, errors.New("WAF_DASHBOARD_SOURCE_INSTANCE_ID is required and must match the OWI safe ID syntax")
	}
	_, port, err := net.SplitHostPort(cfg.Addr)
	if err != nil || (port != "443" && port != "19405") {
		return cfg, errors.New("dashboard reader address must be host:443 or host:19405")
	}
	if !cfg.AllowInsecureTest && (cfg.TLSCert == "" || cfg.TLSKey == "") {
		return cfg, errors.New("dashboard reader requires TLS cert/key; insecure HTTP is test-only")
	}
	if (cfg.TLSCert == "") != (cfg.TLSKey == "") {
		return cfg, errors.New("dashboard reader TLS cert and key must be configured together")
	}
	if cfg.CursorSecretFile == "" {
		return cfg, errors.New("WAF_DASHBOARD_CURSOR_SECRET_FILE is required when the dashboard reader is enabled")
	}
	if cfg.RatePerSecond <= 0 || cfg.Burst < 1 || cfg.PerPrincipalConcurrent < 1 || cfg.GlobalConcurrent < 1 {
		return cfg, errors.New("dashboard reader rate/concurrency limits must be positive")
	}
	if cfg.PerPrincipalConcurrent > cfg.GlobalConcurrent {
		return cfg, errors.New("per-principal dashboard concurrency cannot exceed global concurrency")
	}
	if cfg.RequestDeadline < time.Second || cfg.RequestDeadline > 30*time.Second {
		return cfg, errors.New("dashboard reader request deadline must be between 1s and 30s")
	}
	if cfg.MaxPageSize < 1 || cfg.MaxPageSize > 500 || cfg.MaxResponseBytes < 1024 || cfg.MaxResponseBytes > 4<<20 {
		return cfg, errors.New("dashboard reader page/response bounds exceed OWI-1.0 limits")
	}
	if cfg.SnapshotMaxCount < 1 || cfg.SnapshotMaxRecords < 1 || cfg.SnapshotMaxBytes < 1024 || cfg.ExportMaxRows < 1 || cfg.ExportMaxBytes < 1024 {
		return cfg, errors.New("dashboard reader snapshot/export capacity limits must be positive")
	}
	return cfg, nil
}

func envBool(name string, def bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	if v == "" {
		return def
	}
	return v == "1" || v == "true" || v == "yes" || v == "on"
}
func envInt(name string, def int) int {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
func envFloat(name string, def float64) float64 {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			return n
		}
	}
	return def
}

var owiIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

func validOWIID(v string) bool { return owiIDPattern.MatchString(v) }

func owiTokenDigest(token string) string {
	h := sha256.New()
	_, _ = h.Write([]byte("waf-proxy/dashboard-reader-token/v1\x00"))
	_, _ = h.Write([]byte(token))
	return hex.EncodeToString(h.Sum(nil))
}

type owiTokenSource struct {
	path  string
	mu    sync.Mutex
	mtime time.Time
	size  int64
	reg   owiTokenRegistry
}

func (s *owiTokenSource) load() (owiTokenRegistry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := os.Lstat(s.path)
	if err != nil {
		return owiTokenRegistry{}, fmt.Errorf("stat reader token registry: %w", err)
	}
	if st.Mode()&os.ModeSymlink != 0 || !st.Mode().IsRegular() || st.Mode().Perm()&0o077 != 0 {
		return owiTokenRegistry{}, errors.New("reader token registry must be a regular owner-only non-symlink file")
	}
	if st.ModTime().Equal(s.mtime) && st.Size() == s.size && s.reg.Version != 0 {
		return s.reg, nil
	}
	b, err := os.ReadFile(s.path)
	if err != nil {
		return owiTokenRegistry{}, err
	}
	var reg owiTokenRegistry
	if err := json.Unmarshal(b, &reg); err != nil {
		return reg, fmt.Errorf("decode reader token registry: %w", err)
	}
	if err := validateOWITokenRegistry(reg); err != nil {
		return reg, err
	}
	s.reg, s.mtime, s.size = reg, st.ModTime(), st.Size()
	return reg, nil
}

func validateOWITokenRegistry(reg owiTokenRegistry) error {
	if reg.Version != 1 {
		return fmt.Errorf("reader token registry version %d unsupported", reg.Version)
	}
	// Phase A only provisions scopes for enabled resources. Optional EVENT and
	// ACTION_STATUS remain disabled and must not be granted pre-emptively.
	allowed := map[string]bool{
		owiScopeHealth: true, owiScopeCaps: true, owiScopeAssets: true, owiScopeDetections: true,
		owiScopePolicies: true, owiScopeHealthObs: true,
	}
	seen := map[string]bool{}
	for _, t := range reg.Tokens {
		if !validOWIID(t.ID) || !validOWIID(t.Principal) || !validOWIID(t.TenantID) || len(t.DigestSHA256) != 64 {
			return fmt.Errorf("invalid reader token metadata for %q", t.ID)
		}
		if seen[t.ID] {
			return fmt.Errorf("duplicate reader token id %q", t.ID)
		}
		seen[t.ID] = true
		if _, err := hex.DecodeString(t.DigestSHA256); err != nil {
			return fmt.Errorf("invalid reader token digest for %q", t.ID)
		}
		issued, err1 := time.Parse(time.RFC3339, t.IssuedAt)
		expires, err2 := time.Parse(time.RFC3339, t.ExpiresAt)
		if err1 != nil || err2 != nil || !expires.After(issued) || expires.Sub(issued) > 90*24*time.Hour+time.Minute {
			return fmt.Errorf("reader token %q expiry must be RFC3339 and <=90 days", t.ID)
		}
		for _, scope := range t.Scopes {
			if !allowed[scope] {
				return fmt.Errorf("reader token %q has unsupported scope %q", t.ID, scope)
			}
		}
	}
	return nil
}

func (s *owiTokenSource) authenticate(raw string, now time.Time) (owiPrincipal, string) {
	if len(raw) < 32 || len(raw) > 4096 {
		return owiPrincipal{}, "AUTH_REQUIRED"
	}
	reg, err := s.load()
	if err != nil {
		return owiPrincipal{}, "TEMPORARY_UNAVAILABLE"
	}
	gotHex := owiTokenDigest(raw)
	got, _ := hex.DecodeString(gotHex)
	for _, t := range reg.Tokens {
		want, _ := hex.DecodeString(t.DigestSHA256)
		if subtle.ConstantTimeCompare(got, want) != 1 {
			continue
		}
		if t.RevokedAt != nil {
			return owiPrincipal{}, "AUTH_REQUIRED"
		}
		if t.NotBefore != "" {
			nbf, err := time.Parse(time.RFC3339, t.NotBefore)
			if err != nil || now.Before(nbf) {
				return owiPrincipal{}, "AUTH_REQUIRED"
			}
		}
		exp, _ := time.Parse(time.RFC3339, t.ExpiresAt)
		if !exp.After(now) {
			return owiPrincipal{}, "AUTH_REQUIRED"
		}
		scopes := make(map[string]struct{}, len(t.Scopes))
		for _, scope := range t.Scopes {
			scopes[scope] = struct{}{}
		}
		return owiPrincipal{TokenID: t.ID, Principal: t.Principal, TenantID: t.TenantID, Scopes: scopes}, ""
	}
	return owiPrincipal{}, "AUTH_REQUIRED"
}

func (p owiPrincipal) has(scope string) bool { _, ok := p.Scopes[scope]; return ok }

type owiRateState struct {
	tokens float64
	last   time.Time
	active int
}

type owiLimiter struct {
	mu        sync.Mutex
	states    map[string]*owiRateState
	rate      float64
	burst     float64
	perMax    int
	global    int
	globalMax int
}

func newOWILimiter(cfg owiConnectorConfig) *owiLimiter {
	return &owiLimiter{states: map[string]*owiRateState{}, rate: cfg.RatePerSecond, burst: float64(cfg.Burst), perMax: cfg.PerPrincipalConcurrent, globalMax: cfg.GlobalConcurrent}
}

func (l *owiLimiter) acquire(key string, now time.Time) (func(), int, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	st := l.states[key]
	if st == nil {
		st = &owiRateState{tokens: l.burst, last: now}
		l.states[key] = st
	}
	elapsed := now.Sub(st.last).Seconds()
	if elapsed > 0 {
		st.tokens = math.Min(l.burst, st.tokens+elapsed*l.rate)
		st.last = now
	}
	if st.tokens < 1 {
		retry := int(math.Ceil((1 - st.tokens) / l.rate))
		if retry < 1 {
			retry = 1
		}
		return nil, retry, false
	}
	if st.active >= l.perMax || l.global >= l.globalMax {
		return nil, 1, false
	}
	st.tokens--
	st.active++
	l.global++
	released := false
	return func() {
		l.mu.Lock()
		defer l.mu.Unlock()
		if released {
			return
		}
		released = true
		st.active--
		l.global--
	}, 0, true
}

type owiConnector struct {
	srv          *server
	cfg          owiConnectorConfig
	log          *slog.Logger
	store        *owiStore
	tokens       *owiTokenSource
	cursorSecret []byte
	limiter      *owiLimiter
	export       *owiExportPlane
	syncHook     func(context.Context) error
}

func newOWIConnector(s *server, cfg owiConnectorConfig, log *slog.Logger) (*owiConnector, error) {
	d := defaultOWIStoreLimits()
	if cfg.SnapshotMaxCount < 1 {
		cfg.SnapshotMaxCount = d.SnapshotMaxCount
	}
	if cfg.SnapshotMaxRecords < 1 {
		cfg.SnapshotMaxRecords = d.SnapshotMaxRecords
	}
	if cfg.SnapshotMaxBytes < 1 {
		cfg.SnapshotMaxBytes = d.SnapshotMaxBytes
	}
	if cfg.ExportMaxRows < 1 {
		cfg.ExportMaxRows = d.DetectionMaxRows
	}
	if cfg.ExportMaxBytes < 1 {
		cfg.ExportMaxBytes = d.DetectionMaxBytes
	}
	secret, err := loadOWISecretFile(cfg.CursorSecretFile, "dashboard cursor secret")
	if err != nil {
		return nil, err
	}
	store, err := newOWIStoreWithLimits(cfg.StatePath, cfg.SourceInstanceID, owiStoreLimits{
		SnapshotMaxCount: cfg.SnapshotMaxCount, SnapshotMaxRecords: cfg.SnapshotMaxRecords, SnapshotMaxBytes: cfg.SnapshotMaxBytes,
		DetectionMaxRows: cfg.ExportMaxRows, DetectionMaxBytes: cfg.ExportMaxBytes,
	})
	if err != nil {
		return nil, err
	}
	c := &owiConnector{srv: s, cfg: cfg, log: log, store: store, tokens: &owiTokenSource{path: cfg.TokenRegistryPath}, cursorSecret: secret, limiter: newOWILimiter(cfg)}
	if _, err := c.tokens.load(); err != nil {
		_ = store.closeClean()
		return nil, err
	}
	c.export = newOWIExportPlane(store, log, 4096)
	c.syncHook = c.syncNativeState
	return c, nil
}

func (c *owiConnector) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+owiBasePath+"/health", c.wrap(owiScopeHealth, c.handleHealth))
	mux.HandleFunc("GET "+owiBasePath+"/capabilities", c.wrap(owiScopeCaps, c.handleCapabilities))
	mux.HandleFunc("GET "+owiBasePath+"/assets", c.wrap(owiScopeAssets, c.resourceHandler("ASSET")))
	mux.HandleFunc("GET "+owiBasePath+"/detections", c.wrap(owiScopeDetections, c.resourceHandler("DETECTION")))
	mux.HandleFunc("GET "+owiBasePath+"/policies", c.wrap(owiScopePolicies, c.resourceHandler("POLICY")))
	mux.HandleFunc("GET "+owiBasePath+"/health-observations", c.wrap(owiScopeHealthObs, c.resourceHandler("HEALTH")))
	mux.HandleFunc("GET "+owiBasePath+"/events", c.wrapDisabled(c.handleCapabilityUnavailable))
	mux.HandleFunc("GET "+owiBasePath+"/action-status", c.wrapDisabled(c.handleCapabilityUnavailable))
	mux.HandleFunc(owiBasePath, c.wrapFallback)
	mux.HandleFunc(owiBasePath+"/", c.wrapFallback)
	return mux
}

func (c *owiConnector) authRequest(w http.ResponseWriter, r *http.Request) (owiPrincipal, bool) {
	if !acceptsOWIJSON(r.Header.Get("Accept")) {
		c.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Accept must permit application/json.", false, 0)
		return owiPrincipal{}, false
	}
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") || strings.TrimSpace(strings.TrimPrefix(auth, "Bearer ")) == "" {
		c.writeError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Machine bearer authentication is required.", false, 0)
		return owiPrincipal{}, false
	}
	p, code := c.tokens.authenticate(strings.TrimSpace(strings.TrimPrefix(auth, "Bearer ")), time.Now().UTC())
	if code != "" {
		if code == "TEMPORARY_UNAVAILABLE" {
			c.writeError(w, http.StatusServiceUnavailable, code, "Reader authentication state is temporarily unavailable.", true, 5)
		} else {
			c.writeError(w, http.StatusUnauthorized, code, "Machine bearer authentication is required.", false, 0)
		}
		return owiPrincipal{}, false
	}
	return p, true
}

func (c *owiConnector) wrap(scope string, next func(http.ResponseWriter, *http.Request, owiPrincipal)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, ok := c.authRequest(w, r)
		if !ok {
			return
		}
		// Go ServeMux intentionally lets a GET pattern match HEAD. OWI Phase A
		// is an exact GET-only machine-reader contract, so reject HEAD explicitly.
		if r.Method != http.MethodGet {
			c.writeError(w, http.StatusMethodNotAllowed, "INVALID_REQUEST", "The Phase A reader is GET-only.", false, 0)
			return
		}
		if !p.has(scope) {
			c.writeError(w, http.StatusForbidden, "FORBIDDEN", "The reader principal does not have the required scope.", false, 0)
			return
		}
		release, retry, ok := c.limiter.acquire(p.TenantID+"\x00"+p.Principal, time.Now())
		if !ok {
			c.writeError(w, http.StatusTooManyRequests, "RATE_LIMITED", "Reader request quota is temporarily exhausted.", true, retry)
			return
		}
		defer release()
		ctx, cancel := context.WithTimeout(r.Context(), c.cfg.RequestDeadline)
		defer cancel()
		next(w, r.WithContext(ctx), p)
	}
}

func (c *owiConnector) wrapDisabled(next func(http.ResponseWriter, *http.Request, owiPrincipal)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, ok := c.authRequest(w, r)
		if !ok {
			return
		}
		if r.Method != http.MethodGet {
			c.writeError(w, http.StatusMethodNotAllowed, "INVALID_REQUEST", "The Phase A reader is GET-only.", false, 0)
			return
		}
		next(w, r, p)
	}
}

func (c *owiConnector) wrapFallback(w http.ResponseWriter, r *http.Request) {
	_, ok := c.authRequest(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		c.writeError(w, http.StatusMethodNotAllowed, "INVALID_REQUEST", "The Phase A reader is GET-only.", false, 0)
		return
	}
	c.writeError(w, http.StatusNotFound, "NOT_FOUND", "The requested reader route does not exist.", false, 0)
}

func acceptsOWIJSON(v string) bool {
	v = strings.TrimSpace(strings.ToLower(v))
	return v == "" || strings.Contains(v, "application/json") || strings.Contains(v, "*/*")
}

func (c *owiConnector) handleHealth(w http.ResponseWriter, r *http.Request, _ owiPrincipal) {
	if r.URL.RawQuery != "" {
		c.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Control routes do not accept query parameters.", false, 0)
		return
	}
	status := "READY"
	reasons := []string{}
	ready := true
	if c.srv == nil || c.srv.rt.Load() == nil {
		status, ready, reasons = "UNAVAILABLE", false, []string{"RUNTIME_UNAVAILABLE"}
	} else if c.srv.draining.Load() {
		status, ready, reasons = "DEGRADED", false, []string{"DRAINING"}
	} else if cov, err := c.store.coverageContext(r.Context(), "DETECTION"); err != nil {
		c.writeError(w, 503, "TEMPORARY_UNAVAILABLE", "Reader state access exceeded its bounded server deadline.", true, 5)
		return
	} else if cov.State != "COMPLETE" {
		status, reasons = "DEGRADED", append(reasons, cov.ReasonCodes...)
	}
	c.writeJSON(w, 200, owiHealthResponse{owiContract, owiSchemaVersion, owiSourceProduct, c.cfg.SourceInstanceID, status, time.Now().UTC().Format(time.RFC3339), ready, reasons})
}

func owiResources(maxPage int) []owiCapabilityResource {
	thirty := 30
	return []owiCapabilityResource{
		{owiBasePath + "/assets", "ASSET", "LIST_ASSETS", true, "IMPLEMENTED_UNQUALIFIED", []string{"SNAPSHOT", "INCREMENTAL"}, "STATE", nil, 86400, maxPage, []string{}},
		{owiBasePath + "/detections", "DETECTION", "LIST_DETECTIONS", true, "IMPLEMENTED_UNQUALIFIED", []string{"SNAPSHOT", "INCREMENTAL"}, "HISTORY", &thirty, 86400, maxPage, []string{}},
		{owiBasePath + "/policies", "POLICY", "OBSERVE_POLICY", true, "IMPLEMENTED_UNQUALIFIED", []string{"SNAPSHOT", "INCREMENTAL"}, "STATE", nil, 86400, maxPage, []string{}},
		{owiBasePath + "/health-observations", "HEALTH", "HEALTH", true, "IMPLEMENTED_UNQUALIFIED", []string{"SNAPSHOT", "INCREMENTAL"}, "STATE", nil, 86400, maxPage, []string{}},
		{owiBasePath + "/events", "EVENT", "LIST_EVENTS", false, "NOT_IMPLEMENTED", []string{}, "HISTORY", nil, 86400, maxPage, []string{"OPTIONAL_NOT_IMPLEMENTED"}},
		{owiBasePath + "/action-status", "ACTION_STATUS", "LIST_ACTIONS", false, "NOT_IMPLEMENTED", []string{}, "STATE", nil, 86400, maxPage, []string{"OPTIONAL_NOT_IMPLEMENTED"}},
	}
}

func (c *owiConnector) handleCapabilities(w http.ResponseWriter, r *http.Request, _ owiPrincipal) {
	if r.URL.RawQuery != "" {
		c.writeError(w, 400, "INVALID_REQUEST", "Control routes do not accept query parameters.", false, 0)
		return
	}
	c.writeJSON(w, 200, owiCapabilitiesResponse{owiContract, owiSchemaVersion, owiSourceProduct, c.cfg.SourceInstanceID, "opaque_bearer", owiResources(c.cfg.MaxPageSize), []string{}})
}

func (c *owiConnector) handleCapabilityUnavailable(w http.ResponseWriter, _ *http.Request, _ owiPrincipal) {
	c.writeError(w, http.StatusNotImplemented, "CAPABILITY_UNAVAILABLE", "This optional Phase A capability is not implemented.", false, 0)
}

func (c *owiConnector) resourceHandler(kind string) func(http.ResponseWriter, *http.Request, owiPrincipal) {
	return func(w http.ResponseWriter, r *http.Request, p owiPrincipal) {
		limit, cursor, err := parseOWIPageQuery(r, c.cfg.MaxPageSize)
		if err != nil {
			c.writeError(w, 400, "INVALID_REQUEST", "Only limit and cursor query parameters are accepted.", false, 0)
			return
		}
		if err := c.syncHook(r.Context()); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(r.Context().Err(), context.DeadlineExceeded) {
				c.writeError(w, 503, "TEMPORARY_UNAVAILABLE", "Reader source processing exceeded its bounded server deadline.", true, 5)
				return
			}
			c.writeError(w, 503, "TEMPORARY_UNAVAILABLE", "Reader source data is temporarily unavailable.", true, 5)
			return
		}
		page, status, code, msg := c.page(r.Context(), kind, p, limit, cursor)
		if status != 0 {
			c.writeError(w, status, code, msg, false, 0)
			return
		}
		c.writeJSON(w, 200, page)
	}
}

func parseOWIPageQuery(r *http.Request, maxPage int) (int, string, error) {
	q := r.URL.Query()
	for k, vals := range q {
		if k != "limit" && k != "cursor" {
			return 0, "", errors.New("unsupported query")
		}
		if len(vals) != 1 {
			return 0, "", errors.New("duplicate query")
		}
	}
	limit := 100
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > maxPage {
			return 0, "", errors.New("invalid limit")
		}
		limit = n
	}
	cursor := q.Get("cursor")
	if len(cursor) > 2048 {
		return 0, "", errors.New("cursor too long")
	}
	return limit, cursor, nil
}

func (c *owiConnector) page(ctx context.Context, kind string, p owiPrincipal, limit int, rawCursor string) (owiPage, int, string, string) {
	now := time.Now().UTC()
	scopeHash := owiScopeHash(p.Scopes)
	if rawCursor == "" {
		snap, err := c.store.newSnapshotContext(ctx, kind, p, scopeHash, now)
		if err != nil {
			return owiPage{}, 503, "TEMPORARY_UNAVAILABLE", "Reader snapshot capacity is temporarily unavailable."
		}
		cur := owiCursor{Version: 1, TenantID: p.TenantID, Principal: p.Principal, ScopeHash: scopeHash, SourceInstanceID: c.cfg.SourceInstanceID, ResourceKind: kind, Mode: "SNAPSHOT", SnapshotID: snap.ID, Offset: 0, Watermark: snap.Watermark, Limit: limit, IssuedAtUnix: now.Unix(), ExpiresAtUnix: now.Add(owiCursorTTL).Unix()}
		return c.snapshotPage(ctx, snap, cur, limit, 0)
	}
	cur, err := decodeOWICursor(c.cursorSecret, rawCursor)
	if err != nil {
		return owiPage{}, 400, "INVALID_CURSOR", "Cursor is malformed or has an invalid signature."
	}
	if cur.TenantID != p.TenantID || cur.Principal != p.Principal || cur.ScopeHash != scopeHash || cur.SourceInstanceID != c.cfg.SourceInstanceID || cur.ResourceKind != kind || cur.Limit != limit || now.Unix() >= cur.ExpiresAtUnix {
		return owiPage{}, 409, "RESET_REQUIRED", "Cursor no longer matches the reader identity, visibility, source, limit, or retention window."
	}
	if cur.Mode == "SNAPSHOT" {
		snap, ok, snapErr := c.store.getSnapshotContext(ctx, cur.SnapshotID)
		if snapErr != nil {
			return owiPage{}, 503, "TEMPORARY_UNAVAILABLE", "Reader state access exceeded its bounded server deadline."
		}
		if !ok || snap.TenantID != p.TenantID || snap.Principal != p.Principal || snap.ScopeHash != scopeHash || snap.Kind != kind {
			return owiPage{}, 409, "RESET_REQUIRED", "Cursor snapshot is no longer available."
		}
		return c.snapshotPage(ctx, snap, cur, limit, cur.Offset)
	}
	if cur.Mode != "INCREMENTAL" {
		return owiPage{}, 400, "INVALID_CURSOR", "Cursor mode is invalid."
	}
	rows, current, min, coverage, journalErr := c.store.journalAfterContext(ctx, kind, cur.Watermark)
	if journalErr != nil {
		return owiPage{}, 503, "TEMPORARY_UNAVAILABLE", "Reader state access exceeded its bounded server deadline."
	}
	if min > 0 && cur.Watermark+1 < min {
		return owiPage{}, 409, "RESET_REQUIRED", "Cursor is older than the retained export journal."
	}
	end := len(rows)
	if end > limit {
		end = limit
	}
	selected := rows[:end]
	nextWatermark := cur.Watermark
	if len(selected) > 0 {
		nextWatermark, _ = strconv.ParseUint(selected[len(selected)-1].StreamSequence, 10, 64)
	}
	if len(selected) == 0 && current > nextWatermark {
		nextWatermark = current
	}
	next := cur
	next.Watermark = nextWatermark
	next.IssuedAtUnix = now.Unix()
	next.ExpiresAtUnix = now.Add(owiCursorTTL).Unix()
	nextToken, _ := encodeOWICursor(c.cursorSecret, next)
	if len(selected) == 0 {
		nextToken = rawCursor
	}
	return owiPage{owiContract, owiSchemaVersion, owiSourceProduct, c.cfg.SourceInstanceID, kind, "INCREMENTAL", nil, coverage, now.Format(time.RFC3339), selected, nextToken, len(rows) > end}, 0, "", ""
}

func (c *owiConnector) snapshotPage(ctx context.Context, snap owiSnapshot, cur owiCursor, limit, offset int) (owiPage, int, string, string) {
	if offset < 0 || offset > len(snap.Records) {
		return owiPage{}, 400, "INVALID_CURSOR", "Cursor snapshot offset is invalid."
	}
	end := offset + limit
	if end > len(snap.Records) {
		end = len(snap.Records)
	}
	rows := cloneOWIRecords(snap.Records[offset:end])
	more := end < len(snap.Records)
	now := time.Now().UTC()
	next := cur
	if more {
		next.Offset = end
	} else {
		next.Mode = "INCREMENTAL"
		next.SnapshotID = ""
		next.Offset = 0
		next.Watermark = snap.Watermark
	}
	next.IssuedAtUnix = now.Unix()
	next.ExpiresAtUnix = now.Add(owiCursorTTL).Unix()
	token, err := encodeOWICursor(c.cursorSecret, next)
	if err != nil {
		return owiPage{}, 503, "TEMPORARY_UNAVAILABLE", "Reader cursor generation failed."
	}
	coverage, err := c.store.coverageContext(ctx, snap.Kind)
	if err != nil {
		return owiPage{}, 503, "TEMPORARY_UNAVAILABLE", "Reader state access exceeded its bounded server deadline."
	}
	sid := snap.ID
	return owiPage{owiContract, owiSchemaVersion, owiSourceProduct, c.cfg.SourceInstanceID, snap.Kind, "SNAPSHOT", &sid, coverage, now.Format(time.RFC3339), rows, token, more}, 0, "", ""
}

func (c *owiConnector) writeError(w http.ResponseWriter, status int, code, msg string, retry bool, retryAfter int) {
	var after *int
	if retryAfter > 0 {
		after = &retryAfter
		w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
	}
	id, _ := randomOWIID("req")
	c.writeJSON(w, status, owiErrorResponse{owiContract, owiErrorBody{code, msg, id, retry, after}})
}

func (c *owiConnector) writeJSON(w http.ResponseWriter, status int, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		http.Error(w, "internal error", 500)
		return
	}
	if len(b) > c.cfg.MaxResponseBytes {
		id, _ := randomOWIID("req")
		b, _ = json.Marshal(owiErrorResponse{owiContract, owiErrorBody{"TEMPORARY_UNAVAILABLE", "Reader response exceeded the configured response bound.", id, true, nil}})
		status = 503
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_, _ = w.Write(append(b, '\n'))
}

func (c *owiConnector) close() error {
	if c.export != nil {
		c.export.stopAndDrain()
	}
	if c.store != nil {
		return c.store.closeClean()
	}
	return nil
}

func owiSortedScopes(p owiPrincipal) []string {
	out := make([]string, 0, len(p.Scopes))
	for s := range p.Scopes {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func owiGenerateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// owiPlaneCheck keeps the machine-reader listener distinct from both the
// browser/admin plane and WAF data listeners. A dedicated management hostname
// may use 443; same-host deployments use the fixed 19405 port.
func owiPlaneCheck(readerAddr, adminAddr string, sites []SiteConfig) error {
	rh, rp, err := net.SplitHostPort(readerAddr)
	if err != nil {
		return err
	}
	ah, ap, aerr := net.SplitHostPort(adminAddr)
	if aerr == nil && rp == ap && hostsOverlap(rh, ah) {
		return fmt.Errorf("Dashboard reader %s overlaps admin listener %s", readerAddr, adminAddr)
	}
	for _, site := range sites {
		sh, sp, err := net.SplitHostPort(site.Listen)
		if err != nil {
			continue
		}
		if rp == sp && hostsOverlap(rh, sh) {
			return fmt.Errorf("Dashboard reader %s overlaps data-plane listener %s", readerAddr, site.Listen)
		}
	}
	return nil
}
func hostsOverlap(a, b string) bool {
	norm := func(s string) string {
		s = strings.Trim(strings.TrimSpace(s), "[]")
		if s == "" || s == "0.0.0.0" || s == "::" || s == "*" {
			return "*"
		}
		return strings.ToLower(s)
	}
	a, b = norm(a), norm(b)
	return a == "*" || b == "*" || a == b
}
