package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const dashboardTokenDomain = "waf-proxy/operator-workspace/reader/v1\x00"

type dashboardReaderPrincipal struct {
	ID        string   `json:"id"`
	TokenHash string   `json:"token_digest"`
	Scopes    []string `json:"scopes"`
	NotBefore string   `json:"not_before"`
	ExpiresAt string   `json:"expires_at"`
	Revoked   bool     `json:"revoked"`
}

type dashboardReaderConfig struct {
	Enabled        bool                       `json:"enabled"`
	NetworkProfile string                     `json:"network_profile"`
	Listen         string                     `json:"listen"`
	ServerName     string                     `json:"server_name"`
	TLSCert        string                     `json:"tls_cert"`
	TLSKey         string                     `json:"tls_key"`
	DataDir        string                     `json:"data_dir"`
	SourceInstance string                     `json:"source_instance_id"`
	TenantID       string                     `json:"tenant_id"`
	Principals     []dashboardReaderPrincipal `json:"principals"`
}

type dashboardReader struct {
	server     *server
	store      *dashboardStore
	configPath string
	config     dashboardReaderConfig
	adminToken string
	haToken    string
}

type dashboardCursor struct {
	Kind       string `json:"k"`
	Tenant     string `json:"t"`
	Principal  string `json:"p"`
	Visibility string `json:"v"`
	Instance   string `json:"i"`
	Mode       string `json:"m"` // snapshot, delta, new
	SnapshotID string `json:"s,omitempty"`
	Offset     int    `json:"o,omitempty"`
	Watermark  uint64 `json:"w"`
	GapEpoch   uint64 `json:"g"`
	Expires    int64  `json:"e"`
}

type dashboardCoverage struct {
	State       string   `json:"state"`
	ReasonCodes []string `json:"reason_codes"`
}

type dashboardPage struct {
	Contract         string            `json:"contract"`
	SchemaVersion    string            `json:"schema_version"`
	SourceProduct    string            `json:"source_product"`
	SourceInstance   string            `json:"source_instance_id"`
	ResourceKind     string            `json:"resource_kind"`
	SyncMode         string            `json:"sync_mode"`
	SourceSnapshotID *string           `json:"source_snapshot_id"`
	Coverage         dashboardCoverage `json:"coverage"`
	GeneratedAt      string            `json:"generated_at"`
	Records          []dashboardRecord `json:"records"`
	NextCursor       string            `json:"next_cursor"`
	HasMore          bool              `json:"has_more"`
}

func dashboardValidID(v string) bool {
	if len(v) < 1 || len(v) > 128 {
		return false
	}
	for i, c := range v {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' {
			continue
		}
		if i > 0 && (c == '.' || c == '_' || c == ':' || c == '-') {
			continue
		}
		return false
	}
	return true
}

func dashboardAllowedScope(scope string) bool {
	switch scope {
	case "dashboard:read:health", "dashboard:read:capabilities", "dashboard:read:assets",
		"dashboard:read:detections", "dashboard:read:policies", "dashboard:read:health-observations":
		return true
	}
	return false
}

func loadDashboardReaderConfig(path string) (dashboardReaderConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return dashboardReaderConfig{}, err
	}
	if info, err := os.Lstat(path); err != nil || !info.Mode().IsRegular() ||
		runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return dashboardReaderConfig{}, errors.New("dashboard reader config must be mode 0600 or stricter")
	}
	var cfg dashboardReaderConfig
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return cfg, err
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return cfg, errors.New("dashboard reader config has trailing data")
	}
	if !cfg.Enabled {
		return cfg, nil
	}
	if !dashboardValidID(cfg.SourceInstance) || !dashboardValidID(cfg.TenantID) {
		return cfg, errors.New("invalid dashboard reader source instance or tenant ID")
	}
	if !filepath.IsAbs(cfg.DataDir) || !filepath.IsAbs(cfg.TLSCert) || !filepath.IsAbs(cfg.TLSKey) {
		return cfg, errors.New("dashboard reader data directory and TLS files must be absolute paths")
	}
	host, port, err := net.SplitHostPort(cfg.Listen)
	if err != nil || net.ParseIP(host) == nil || net.ParseIP(host).IsUnspecified() {
		return cfg, errors.New("dashboard reader must bind a concrete management IP")
	}
	switch cfg.NetworkProfile {
	case "DEDICATED_HOST_443":
		if port != "443" {
			return cfg, errors.New("dedicated-host dashboard reader must use TCP 443")
		}
	case "SAME_HOST_PORT_19405":
		if port != "19405" {
			return cfg, errors.New("same-host dashboard reader must use TCP 19405")
		}
	default:
		return cfg, errors.New("invalid dashboard reader network profile")
	}
	if cfg.ServerName == "" || strings.ContainsAny(cfg.ServerName, "/: \\@") || len(cfg.ServerName) > 253 {
		return cfg, errors.New("invalid dashboard reader TLS server name")
	}
	if len(cfg.Principals) == 0 || len(cfg.Principals) > 2 {
		return cfg, errors.New("dashboard reader requires one or two scoped principals")
	}
	seen := map[string]bool{}
	for _, p := range cfg.Principals {
		if !dashboardValidID(p.ID) || seen[p.ID] {
			return cfg, errors.New("invalid or duplicate dashboard reader principal ID")
		}
		seen[p.ID] = true
		if len(p.TokenHash) != 64 {
			return cfg, errors.New("dashboard reader token digest must be SHA-256 hex")
		}
		if _, err := hex.DecodeString(p.TokenHash); err != nil || strings.ToLower(p.TokenHash) != p.TokenHash {
			return cfg, errors.New("invalid dashboard reader token digest")
		}
		starts, startErr := time.Parse(time.RFC3339, p.NotBefore)
		expires, expiryErr := time.Parse(time.RFC3339, p.ExpiresAt)
		if startErr != nil || expiryErr != nil || !starts.Before(expires) ||
			expires.Sub(starts) > 90*24*time.Hour || starts.After(time.Now().Add(24*time.Hour)) {
			return cfg, errors.New("dashboard reader credential validity must be at most 90 days")
		}
		if len(p.Scopes) == 0 {
			return cfg, errors.New("dashboard reader principal has no scopes")
		}
		scopeSeen := map[string]bool{}
		for _, scope := range p.Scopes {
			if !dashboardAllowedScope(scope) || scopeSeen[scope] {
				return cfg, errors.New("dashboard reader principal has invalid or duplicate scope")
			}
			scopeSeen[scope] = true
		}
	}
	if len(cfg.Principals) == 2 {
		aStart, _ := time.Parse(time.RFC3339, cfg.Principals[0].NotBefore)
		aEnd, _ := time.Parse(time.RFC3339, cfg.Principals[0].ExpiresAt)
		bStart, _ := time.Parse(time.RFC3339, cfg.Principals[1].NotBefore)
		bEnd, _ := time.Parse(time.RFC3339, cfg.Principals[1].ExpiresAt)
		if aStart.Before(bEnd) && bStart.Before(aEnd) {
			start, end := aStart, aEnd
			if bStart.After(start) {
				start = bStart
			}
			if bEnd.Before(end) {
				end = bEnd
			}
			if end.Sub(start) > 24*time.Hour {
				return cfg, errors.New("dashboard reader credential overlap exceeds 24 hours")
			}
		}
	}
	return cfg, nil
}

func dashboardTokenDigest(token string) [32]byte {
	return sha256.Sum256([]byte(dashboardTokenDomain + token))
}

// mintDashboardReaderToken is an explicit offline operator command. The raw
// token is printed only by that command; the product config stores its digest.
func mintDashboardReaderToken() (string, string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	digest := dashboardTokenDigest(token)
	return token, hex.EncodeToString(digest[:]), nil
}

func newDashboardReader(s *server, path, adminToken, haToken string) (*dashboardReader, error) {
	cfg, err := loadDashboardReaderConfig(path)
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		return nil, nil
	}
	store, err := openDashboardStore(cfg.DataDir, cfg.SourceInstance)
	if err != nil {
		return nil, err
	}
	r := &dashboardReader{server: s, store: store, configPath: path, config: cfg,
		adminToken: adminToken, haToken: haToken}
	if rt := s.rt.Load(); rt != nil {
		if err := r.reconcileConfig(rt); err != nil {
			store.close()
			return nil, err
		}
	}
	return r, nil
}

func (d *dashboardReader) authenticate(r *http.Request, scope string) (dashboardReaderPrincipal, string, int) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") || len(r.Header.Values("Authorization")) != 1 {
		return dashboardReaderPrincipal{}, "", http.StatusUnauthorized
	}
	token := strings.TrimPrefix(auth, "Bearer ")
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(token) != 43 || len(raw) != 32 || token == d.adminToken || token == d.haToken {
		return dashboardReaderPrincipal{}, "", http.StatusUnauthorized
	}
	cfg, err := loadDashboardReaderConfig(d.configPath)
	if err != nil || !cfg.Enabled || cfg.SourceInstance != d.config.SourceInstance ||
		cfg.TenantID != d.config.TenantID || cfg.Listen != d.config.Listen || cfg.ServerName != d.config.ServerName ||
		cfg.NetworkProfile != d.config.NetworkProfile || cfg.DataDir != d.config.DataDir ||
		cfg.TLSCert != d.config.TLSCert || cfg.TLSKey != d.config.TLSKey {
		return dashboardReaderPrincipal{}, "", http.StatusServiceUnavailable
	}
	digest := dashboardTokenDigest(token)
	for _, p := range cfg.Principals {
		want, _ := hex.DecodeString(p.TokenHash)
		if subtle.ConstantTimeCompare(digest[:], want) != 1 || p.Revoked {
			continue
		}
		exp, _ := time.Parse(time.RFC3339, p.ExpiresAt)
		start, _ := time.Parse(time.RFC3339, p.NotBefore)
		if time.Now().Before(start) || !time.Now().Before(exp) {
			return dashboardReaderPrincipal{}, "", http.StatusUnauthorized
		}
		scopes := append([]string(nil), p.Scopes...)
		sort.Strings(scopes)
		fp := sha256.Sum256([]byte(p.ID + "\x00" + cfg.TenantID + "\x00" + strings.Join(scopes, "\x00")))
		for _, allowed := range p.Scopes {
			if allowed == scope {
				return p, hex.EncodeToString(fp[:]), http.StatusOK
			}
		}
		return dashboardReaderPrincipal{}, "", http.StatusForbidden
	}
	return dashboardReaderPrincipal{}, "", http.StatusUnauthorized
}

func (d *dashboardReader) error(w http.ResponseWriter, status int, code, message string) {
	id, err := dashboardRandomID()
	if err != nil {
		id = "request-unavailable"
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	var retryAfter any
	if status == http.StatusTooManyRequests {
		w.Header().Set("Retry-After", "60")
		retryAfter = 60
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"contract": dashboardContract,
		"error": map[string]any{"code": code, "message": message,
			"request_id": "req-" + id, "retryable": status == 429 || status == 503,
			"retry_after_seconds": retryAfter},
	})
}

func (d *dashboardReader) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func dashboardScopeForSuffix(suffix string) (string, string, bool) {
	switch suffix {
	case "/health":
		return "dashboard:read:health", "", true
	case "/capabilities":
		return "dashboard:read:capabilities", "", true
	case "/assets":
		return "dashboard:read:assets", "ASSET", true
	case "/detections":
		return "dashboard:read:detections", "DETECTION", true
	case "/policies":
		return "dashboard:read:policies", "POLICY", true
	case "/health-observations":
		return "dashboard:read:health-observations", "HEALTH", true
	case "/events", "/action-status", "/findings", "/alerts", "/incidents", "/flows",
		"/audit-events", "/compliance-results", "/sessions", "/reports", "/scans":
		return "", "", false
	default:
		return "", "", false
	}
}

func (d *dashboardReader) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			d.error(w, http.StatusMethodNotAllowed, "INVALID_REQUEST", "Reader accepts GET only.")
			return
		}
		validHost := strings.EqualFold(r.Host, d.config.ServerName)
		if d.config.NetworkProfile == "SAME_HOST_PORT_19405" {
			validHost = strings.EqualFold(r.Host, net.JoinHostPort(d.config.ServerName, "19405"))
		}
		if !validHost {
			d.error(w, http.StatusForbidden, "FORBIDDEN", "Reader hostname is not allowed.")
			return
		}
		if !strings.HasPrefix(r.URL.Path, dashboardPrefix) {
			d.error(w, http.StatusNotFound, "NOT_FOUND", "Resource not found.")
			return
		}
		suffix := strings.TrimPrefix(r.URL.Path, dashboardPrefix)
		scope, kind, supported := dashboardScopeForSuffix(suffix)
		if !supported {
			// Reserved lanes still require a machine principal before revealing
			// capability status, but never accept an admin/session credential.
			if _, _, status := d.authenticate(r, "dashboard:read:capabilities"); status != http.StatusOK {
				d.authError(w, status)
				return
			}
			d.error(w, http.StatusNotImplemented, "CAPABILITY_UNAVAILABLE", "Resource is not supported by this product.")
			return
		}
		principal, visibility, status := d.authenticate(r, scope)
		if status != http.StatusOK {
			d.authError(w, status)
			return
		}
		if r.Header.Get("Accept") != "" && !strings.Contains(r.Header.Get("Accept"), "application/json") {
			d.error(w, http.StatusBadRequest, "INVALID_REQUEST", "Accept application/json is required.")
			return
		}
		if kind == "" && r.URL.RawQuery != "" {
			d.error(w, 400, "INVALID_REQUEST", "Control route takes no query parameters.")
			return
		}
		switch suffix {
		case "/health":
			d.health(w)
		case "/capabilities":
			d.capabilities(w, principal)
		default:
			d.page(w, r, kind, principal, visibility)
		}
	})
}

func (d *dashboardReader) authError(w http.ResponseWriter, status int) {
	switch status {
	case http.StatusForbidden:
		d.error(w, status, "FORBIDDEN", "Reader scope is not granted.")
	case http.StatusServiceUnavailable:
		d.error(w, status, "TEMPORARY_UNAVAILABLE", "Reader identity configuration is unavailable.")
	default:
		d.error(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Machine reader credential is required.")
	}
}

func (d *dashboardReader) health(w http.ResponseWriter) {
	status := "READY"
	reasons := []string{}
	ready := true
	if d.server.rt.Load() == nil || d.server.draining.Load() {
		status, ready = "UNAVAILABLE", false
		reasons = append(reasons, "RUNTIME_UNAVAILABLE")
	}
	if d.store.gap.Load() {
		status, ready = "DEGRADED", false
		reasons = append(reasons, "EXPORT_COVERAGE_GAP")
	}
	d.writeJSON(w, http.StatusOK, map[string]any{
		"contract": dashboardContract, "schema_version": dashboardSchemaVersion,
		"source_product": "WAF", "source_instance_id": d.config.SourceInstance,
		"status": status, "observed_at": dashboardUTC(time.Now()),
		"export_ready": ready, "reason_codes": reasons,
	})
}

func (d *dashboardReader) capabilities(w http.ResponseWriter, principal dashboardReaderPrincipal) {
	type resource struct {
		Path            string   `json:"path"`
		Kind            string   `json:"resource_kind"`
		Capability      string   `json:"capability"`
		Enabled         bool     `json:"enabled"`
		Qualification   string   `json:"qualification"`
		SyncModes       []string `json:"sync_modes"`
		StreamSemantics string   `json:"stream_semantics"`
		RetentionDays   *int     `json:"bootstrap_retention_days"`
		CursorTTL       int      `json:"cursor_ttl_seconds"`
		MaxPageSize     int      `json:"max_page_size"`
		ReasonCodes     []string `json:"reason_codes"`
	}
	rows := make([]resource, 0, 6)
	for _, lane := range []struct {
		suffix, kind, capability, semantics string
		required                            bool
	}{
		{"/assets", "ASSET", "LIST_ASSETS", "STATE", true},
		{"/detections", "DETECTION", "LIST_DETECTIONS", "HISTORY", true},
		{"/policies", "POLICY", "OBSERVE_POLICY", "STATE", true},
		{"/health-observations", "HEALTH", "HEALTH", "STATE", true},
		{"/events", "EVENT", "LIST_EVENTS", "HISTORY", false},
		{"/action-status", "ACTION_STATUS", "LIST_ACTIONS", "STATE", false},
	} {
		enabled := lane.required
		granted := false
		for _, scope := range principal.Scopes {
			if scope == "dashboard:read:"+strings.TrimPrefix(lane.suffix, "/") {
				granted = true
				break
			}
		}
		reasons := []string{}
		qualification := "IMPLEMENTED_UNQUALIFIED"
		modes := []string{"SNAPSHOT"}
		var retention *int
		if lane.semantics == "HISTORY" {
			modes = []string{"SNAPSHOT", "INCREMENTAL"}
			days := dashboardHistoryDays
			retention = &days
		}
		if !enabled {
			qualification, modes = "NOT_IMPLEMENTED", []string{}
			reasons = append(reasons, "NOT_IMPLEMENTED")
		}
		if enabled && !granted {
			enabled, modes = false, []string{}
			reasons = append(reasons, "INSUFFICIENT_SCOPE")
		}
		if d.store.gap.Load() && enabled {
			reasons = append(reasons, "EXPORT_COVERAGE_GAP")
		}
		rows = append(rows, resource{dashboardPrefix + lane.suffix, lane.kind, lane.capability,
			enabled, qualification, modes, lane.semantics, retention, 86400, 500, reasons})
	}
	d.writeJSON(w, http.StatusOK, map[string]any{
		"contract": dashboardContract, "schema_version": dashboardSchemaVersion,
		"source_product": "WAF", "source_instance_id": d.config.SourceInstance,
		"auth_profile": "opaque_bearer", "resources": rows, "actions": []any{},
	})
}

func (d *dashboardReader) signCursor(c dashboardCursor) (string, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	h := hmac.New(sha256.New, d.store.key)
	h.Write(b)
	return base64.RawURLEncoding.EncodeToString(append(b, h.Sum(nil)...)), nil
}

func (d *dashboardReader) parseCursor(raw, kind, principal, visibility string) (dashboardCursor, error) {
	if len(raw) < 1 || len(raw) > 2048 || strings.Contains(raw, "=") {
		return dashboardCursor{}, errors.New("invalid cursor")
	}
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil || len(b) <= sha256.Size {
		return dashboardCursor{}, errors.New("invalid cursor")
	}
	message, tag := b[:len(b)-sha256.Size], b[len(b)-sha256.Size:]
	h := hmac.New(sha256.New, d.store.key)
	h.Write(message)
	if !hmac.Equal(tag, h.Sum(nil)) {
		return dashboardCursor{}, errors.New("invalid cursor")
	}
	var c dashboardCursor
	if err := json.Unmarshal(message, &c); err != nil || c.Kind != kind ||
		c.Tenant != d.config.TenantID || c.Principal != principal ||
		c.Visibility != visibility || c.Instance != d.config.SourceInstance {
		return dashboardCursor{}, errors.New("cursor binding changed")
	}
	if time.Now().Unix() >= c.Expires || c.GapEpoch != d.store.gapEpoch.Load() {
		return dashboardCursor{}, errors.New("cursor expired or export coverage changed")
	}
	if c.Mode != "snapshot" && c.Mode != "delta" && c.Mode != "new" {
		return dashboardCursor{}, errors.New("invalid cursor mode")
	}
	return c, nil
}

func (d *dashboardReader) page(w http.ResponseWriter, r *http.Request, kind string, p dashboardReaderPrincipal, visibility string) {
	q := r.URL.Query()
	for key, values := range q {
		if (key != "limit" && key != "cursor") || len(values) != 1 {
			d.error(w, 400, "INVALID_REQUEST", "Unknown or repeated query parameter.")
			return
		}
	}
	limit := 100
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 500 {
			d.error(w, 400, "INVALID_REQUEST", "limit must be between 1 and 500.")
			return
		}
		limit = n
	}
	c := dashboardCursor{Kind: kind, Tenant: d.config.TenantID, Principal: p.ID,
		Visibility: visibility, Instance: d.config.SourceInstance,
		GapEpoch: d.store.gapEpoch.Load(), Expires: time.Now().Add(24 * time.Hour).Unix()}
	if raw := q.Get("cursor"); raw != "" {
		var err error
		c, err = d.parseCursor(raw, kind, p.ID, visibility)
		if err != nil {
			d.error(w, 409, "RESET_REQUIRED", "Cursor is no longer valid.")
			return
		}
		c.Expires = time.Now().Add(24 * time.Hour).Unix()
	} else if _, exists := q["cursor"]; exists {
		d.error(w, 400, "INVALID_CURSOR", "Cursor is malformed.")
		return
	}
	page := dashboardPage{dashboardContract, dashboardSchemaVersion, "WAF", d.config.SourceInstance,
		kind, "SNAPSHOT", nil, dashboardCoverage{"COMPLETE", []string{}}, dashboardUTC(time.Now()),
		[]dashboardRecord{}, "", false}
	createdSnapshot := false
	snapshotStart := 0
	if d.store.gap.Load() {
		page.Coverage = dashboardCoverage{"GAP", []string{"EXPORT_COVERAGE_GAP"}}
	}
	if c.Mode == "" || c.Mode == "new" {
		id, snap, err := d.store.snapshot(kind, p.ID)
		if err != nil {
			d.error(w, 429, "RATE_LIMITED", "Reader snapshot capacity reached.")
			return
		}
		c.Mode, c.SnapshotID, c.Offset, c.Watermark, c.GapEpoch = "snapshot", id, 0, snap.Watermark, snap.GapEpoch
		createdSnapshot = true
	}
	if c.Mode == "snapshot" {
		snap, ok := d.store.getSnapshot(c.SnapshotID)
		if !ok || snap.Kind != kind || snap.Principal != p.ID || snap.GapEpoch != c.GapEpoch ||
			c.Offset < 0 || c.Offset > len(snap.Records) {
			d.error(w, 409, "RESET_REQUIRED", "Snapshot is no longer valid.")
			return
		}
		snapshotID := c.SnapshotID
		page.SourceSnapshotID = &snapshotID
		snapshotStart = c.Offset
		end := c.Offset + limit
		if end > len(snap.Records) {
			end = len(snap.Records)
		}
		page.Records = append(page.Records, snap.Records[c.Offset:end]...)
		page.HasMore = end < len(snap.Records)
		if page.HasMore {
			c.Offset = end
		} else if kind == "DETECTION" {
			c.Mode, c.Watermark, c.SnapshotID, c.Offset = "delta", snap.Watermark, "", 0
		} else {
			c.Mode, c.SnapshotID, c.Offset = "new", "", 0
		}
	} else if c.Mode == "delta" && kind == "DETECTION" {
		if time.Since(time.Unix(c.Expires-86400, 0)) > dashboardHistoryDays*24*time.Hour {
			d.error(w, 409, "RESET_REQUIRED", "Detection history checkpoint expired.")
			return
		}
		page.SyncMode = "INCREMENTAL"
		rows, watermark, more := d.store.delta(c.Watermark, limit)
		page.Records = append(page.Records, rows...)
		page.HasMore = more
		if len(rows) > 0 {
			c.Watermark, _ = strconv.ParseUint(rows[len(rows)-1].StreamSequence, 10, 64)
		} else if !more {
			c.Watermark = watermark
		}
	} else {
		d.error(w, 409, "RESET_REQUIRED", "Cursor is no longer valid.")
		return
	}
	// Bound the actual uncompressed JSON body. A record which cannot fit by
	// itself fails without advancing the cursor; later records remain available.
	for {
		cursor, err := d.signCursor(c)
		if err != nil {
			d.error(w, 503, "TEMPORARY_UNAVAILABLE", "Reader unavailable.")
			return
		}
		page.NextCursor = cursor
		body, err := json.Marshal(page)
		if err == nil && len(body) <= 4<<20 {
			if createdSnapshot && !page.HasMore {
				if err := d.store.dropSnapshot(*page.SourceSnapshotID); err != nil {
					d.error(w, 503, "TEMPORARY_UNAVAILABLE", "Reader snapshot cleanup failed.")
					return
				}
			}
			d.writeJSON(w, 200, page)
			return
		}
		if len(page.Records) <= 1 {
			d.error(w, 422, "RECORD_UNEXPORTABLE", "Record exceeds export size limit.")
			return
		}
		page.Records = page.Records[:len(page.Records)-1]
		page.HasMore = true
		if page.SyncMode == "SNAPSHOT" {
			c.Mode = "snapshot"
			c.SnapshotID = *page.SourceSnapshotID
			c.Offset = snapshotStart + len(page.Records)
		} else {
			c.Watermark, _ = strconv.ParseUint(page.Records[len(page.Records)-1].StreamSequence, 10, 64)
		}
	}
}

func (d *dashboardReader) tlsServer() (*http.Server, net.Listener, error) {
	cert, err := tls.LoadX509KeyPair(d.config.TLSCert, d.config.TLSKey)
	if err != nil {
		return nil, nil, err
	}
	listener, err := net.Listen("tcp", d.config.Listen)
	if err != nil {
		return nil, nil, err
	}
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12, GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		if !strings.EqualFold(hello.ServerName, d.config.ServerName) {
			return nil, errors.New("dashboard reader SNI mismatch")
		}
		return &cert, nil
	}}
	return &http.Server{Addr: d.config.Listen, Handler: d.handler(), TLSConfig: tlsConfig,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second,
		WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second}, listener, nil
}

func (d *dashboardReader) reconcileConfig(rt *runtimeState) error {
	if rt == nil {
		return errors.New("runtime not available")
	}
	at := time.Now().UTC()
	assetKeep := map[string]bool{}
	policyKeep := map[string]bool{}
	asset := func(kind, native, assetType string) error {
		id := dashboardStableID(kind, native)
		assetKeep[id] = true
		observed := at
		if old, ok := d.store.current("ASSET", id); ok {
			if parsed, err := time.Parse(time.RFC3339Nano, old.SourceObservedAt); err == nil {
				observed = parsed
			}
		}
		payload := map[string]any{
			"entity_refs": []any{}, "summary": "WAF managed " + assetType + ".", "attention": nil,
			"asset_type": assetType, "display_name": id,
			"identities": []any{map[string]any{"kind": "PRODUCT_OBJECT_ID", "value": id,
				"namespace": d.config.SourceInstance, "observed_at": dashboardUTC(observed),
				"valid_until": nil, "assurance": "VERIFIED"}},
			"lifecycle": "ACTIVE", "criticality": "UNKNOWN", "last_seen_at": nil,
		}
		return d.store.upsert("ASSET", id, payload, at)
	}
	if err := asset("waf-node", d.config.SourceInstance, "SENSOR"); err != nil {
		return err
	}
	for _, n := range rt.cfg.Nodes {
		if err := asset("origin-node", n.Name, "HOST"); err != nil {
			return err
		}
	}
	for _, p := range rt.cfg.Pools {
		if err := asset("origin-pool", p.Name, "SERVICE"); err != nil {
			return err
		}
	}
	for _, site := range rt.cfg.Sites {
		if err := asset("site", site.Name, "SITE"); err != nil {
			return err
		}
		if err := asset("virtual-service", site.Name+"\x00"+site.Listen, "SERVICE"); err != nil {
			return err
		}
		id := dashboardStableID("policy", site.Name)
		policyKeep[id] = true
		mode := site.EngineMode
		if mode == "" {
			mode = rt.cfg.EngineMode
		}
		state := "NOT_ENFORCING"
		if mode == "On" {
			state = "ENFORCING"
		}
		siteRef := d.store.ref("ASSET", dashboardStableID("site", site.Name))
		payload := map[string]any{"entity_refs": []any{dashboardEntityRef(siteRef, "SUBJECT", rt.builtAt)},
			"summary": "Active WAF site policy.", "attention": nil, "policy_type": "WAF_SITE",
			"desired_generation": nil, "active_generation": fmt.Sprintf("runtime-%d", rt.builtAt.UnixNano()),
			"enforcement_state": state, "scope_refs": []any{siteRef}, "content_digest": nil}
		if err := d.store.upsert("POLICY", id, payload, at); err != nil {
			return err
		}
	}
	if err := d.store.deleteMissing("ASSET", assetKeep, at); err != nil {
		return err
	}
	return d.store.deleteMissing("POLICY", policyKeep, at)
}

func (s *dashboardStore) current(kind, id string) (dashboardRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.state[kind][id]
	return rec, ok
}

func (d *dashboardReader) observeHealth(rt *runtimeState) error {
	if rt == nil {
		return errors.New("runtime not available")
	}
	at := time.Now().UTC()
	keep := map[string]bool{}
	for name, pool := range rt.pools {
		id := dashboardStableID("health-pool", name)
		keep[id] = true
		healthy := pool.healthyCount()
		state := "HEALTHY"
		reasons := []string{}
		var attention any
		var healthyValue any = healthy
		quality := "MEASURED"
		if pool.monitor.Type == "" || pool.monitor.Type == "none" {
			state, quality, healthyValue = "UNKNOWN", "UNKNOWN", nil
			reasons = append(reasons, "MONITOR_DISABLED")
		} else if healthy == 0 && len(pool.members) > 0 {
			state = "DEGRADED"
			reasons = append(reasons, "ALL_POOL_MEMBERS_UNHEALTHY")
			attention = map[string]any{"code": "WAF.PROTECTION_DEGRADED", "severity": "HIGH",
				"confidence": nil, "status": "OPEN", "reason_codes": reasons,
				"recommended_action_codes": []string{}}
		}
		ref := d.store.ref("ASSET", dashboardStableID("origin-pool", name))
		payload := map[string]any{"entity_refs": []any{dashboardEntityRef(ref, "SUBJECT", at)},
			"summary": "Observed WAF origin pool health.", "attention": attention,
			"component": "origin-pool", "state": state, "observed_at": dashboardUTC(at),
			"reason_codes": reasons, "metrics": []any{
				map[string]any{"name": "healthy_members", "value": healthyValue, "unit": "count",
					"window_seconds": 60, "observed_at": dashboardUTC(at), "threshold": 1,
					"threshold_operator": "LT", "quality": quality},
				map[string]any{"name": "configured_members", "value": len(pool.members), "unit": "count",
					"window_seconds": 60, "observed_at": dashboardUTC(at), "threshold": nil,
					"threshold_operator": nil, "quality": "MEASURED"},
			}}
		if err := d.store.upsert("HEALTH", id, payload, at); err != nil {
			return err
		}
	}
	return d.store.deleteMissing("HEALTH", keep, at)
}
