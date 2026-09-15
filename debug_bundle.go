package main

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// DebugBundle is bounded, tenant-scoped diagnostic evidence for one request.
// Tenant is the site name in the current single-control-plane model. Evidence
// is supportability-only and MUST NOT participate in request verdicts.
type DebugBundle struct {
	Tenant        string         `json:"tenant"`
	TransactionID string         `json:"transaction_id"`
	Request       map[string]any `json:"request,omitempty"`
	Response      map[string]any `json:"response,omitempty"`
	TLS           map[string]any `json:"tls,omitempty"`
	Proxy         map[string]any `json:"proxy,omitempty"`
	Coraza        map[string]any `json:"coraza,omitempty"`
	VectorScan    map[string]any `json:"vectorscan,omitempty"`
	CapturedAt    time.Time      `json:"captured_at"`
	ExpiresAt     time.Time      `json:"expires_at,omitempty"`
}

type debugCaptureWindow struct {
	Until time.Time
}

var activeDebugEvidenceStore atomic.Pointer[DebugEvidenceStore]

func SetDebugEvidenceStore(s *DebugEvidenceStore)    { activeDebugEvidenceStore.Store(s) }
func currentDebugEvidenceStore() *DebugEvidenceStore { return activeDebugEvidenceStore.Load() }

type DebugEvidenceStore struct {
	mu      sync.RWMutex
	ttl     time.Duration
	max     int
	items   map[string]DebugBundle
	enabled map[string]debugCaptureWindow
	active  atomic.Bool
}

func NewDebugEvidenceStore(max int, ttl time.Duration) *DebugEvidenceStore {
	if max < 1 {
		max = 1000
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &DebugEvidenceStore{
		max:     max,
		ttl:     ttl,
		items:   map[string]DebugBundle{},
		enabled: map[string]debugCaptureWindow{},
	}
}

func debugStoreKey(tenant, txid string) string { return tenant + "\x00" + txid }

func (s *DebugEvidenceStore) EnableTenant(tenant string, duration time.Duration) error {
	tenant = strings.TrimSpace(tenant)
	if tenant == "" {
		return errors.New("tenant/site is required")
	}
	if duration <= 0 {
		duration = 5 * time.Minute
	}
	if duration > time.Hour {
		return errors.New("debug capture duration must be <= 1h")
	}
	s.mu.Lock()
	s.enabled[tenant] = debugCaptureWindow{Until: time.Now().UTC().Add(duration)}
	s.active.Store(true)
	s.mu.Unlock()
	return nil
}

func (s *DebugEvidenceStore) DisableTenant(tenant string) {
	s.mu.Lock()
	delete(s.enabled, strings.TrimSpace(tenant))
	if len(s.enabled) == 0 {
		s.active.Store(false)
	}
	s.mu.Unlock()
}

func (s *DebugEvidenceStore) ShouldCapture(tenant string, now time.Time) bool {
	if !s.active.Load() {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	w, ok := s.enabled[tenant]
	if !ok {
		return false
	}
	if !w.Until.IsZero() && !now.Before(w.Until) {
		delete(s.enabled, tenant)
		if len(s.enabled) == 0 {
			s.active.Store(false)
		}
		return false
	}
	return true
}

func (s *DebugEvidenceStore) CaptureStatus(now time.Time) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := map[string]string{}
	for tenant, w := range s.enabled {
		if !w.Until.IsZero() && !now.Before(w.Until) {
			delete(s.enabled, tenant)
			continue
		}
		t[tenant] = w.Until.Format(time.RFC3339)
	}
	if len(s.enabled) == 0 {
		s.active.Store(false)
	}
	return map[string]any{"active_tenants": t, "entries": len(s.items), "max_entries": s.max, "ttl_seconds": int64(s.ttl / time.Second)}
}

func (s *DebugEvidenceStore) Put(b DebugBundle) {
	if strings.TrimSpace(b.Tenant) == "" || strings.TrimSpace(b.TransactionID) == "" {
		return
	}
	now := time.Now().UTC()
	b.CapturedAt = now
	b.ExpiresAt = now.Add(s.ttl)
	b.Request = sanitizeDebugMap(b.Request)
	b.Response = sanitizeDebugMap(b.Response)
	b.TLS = sanitizeDebugMap(b.TLS)
	b.Proxy = sanitizeDebugMap(b.Proxy)
	b.Coraza = sanitizeDebugMap(b.Coraza)
	b.VectorScan = sanitizeDebugMap(b.VectorScan)

	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.items) >= s.max {
		// Evict the oldest entry rather than allowing diagnostics to grow without bound.
		var oldestKey string
		var oldest time.Time
		for k, v := range s.items {
			if oldestKey == "" || v.CapturedAt.Before(oldest) {
				oldestKey, oldest = k, v.CapturedAt
			}
		}
		delete(s.items, oldestKey)
	}
	s.items[debugStoreKey(b.Tenant, b.TransactionID)] = b
}

func (s *DebugEvidenceStore) Merge(tenant, txid string, update func(*DebugBundle)) bool {
	if s == nil || update == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := debugStoreKey(tenant, txid)
	b, ok := s.items[key]
	if !ok {
		return false
	}
	update(&b)
	b.Request = sanitizeDebugMap(b.Request)
	b.Response = sanitizeDebugMap(b.Response)
	b.TLS = sanitizeDebugMap(b.TLS)
	b.Proxy = sanitizeDebugMap(b.Proxy)
	b.Coraza = sanitizeDebugMap(b.Coraza)
	b.VectorScan = sanitizeDebugMap(b.VectorScan)
	s.items[key] = b
	return true
}

func (s *DebugEvidenceStore) Get(tenant, txid string) (DebugBundle, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.items[debugStoreKey(tenant, txid)]
	return b, ok
}

func (s *DebugEvidenceStore) List(tenant string, limit int) []DebugBundle {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	s.mu.RLock()
	out := make([]DebugBundle, 0, limit)
	for _, b := range s.items {
		if b.Tenant == tenant {
			out = append(out, b)
		}
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].CapturedAt.After(out[j].CapturedAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func (s *DebugEvidenceStore) Cleanup(now time.Time) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	removed := 0
	for k, v := range s.items {
		if !v.ExpiresAt.IsZero() && !now.Before(v.ExpiresAt) {
			delete(s.items, k)
			removed++
		}
	}
	for tenant, w := range s.enabled {
		if !w.Until.IsZero() && !now.Before(w.Until) {
			delete(s.enabled, tenant)
		}
	}
	if len(s.enabled) == 0 {
		s.active.Store(false)
	}
	return removed
}

func (s *DebugEvidenceStore) RunCleanup(stop <-chan struct{}, every time.Duration) {
	if every <= 0 {
		every = time.Minute
	}
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case now := <-t.C:
			s.Cleanup(now.UTC())
		}
	}
}

func (s *DebugEvidenceStore) ExportIncident(tenant, txid string, out io.Writer) error {
	b, ok := s.Get(tenant, txid)
	if !ok {
		return fmt.Errorf("debug evidence not found for tenant %q transaction %q", tenant, txid)
	}
	files := map[string][]byte{}
	addJSON := func(name string, v any) error {
		data, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
		files[name] = append(data, '\n')
		return nil
	}
	if err := addJSON("metadata.json", map[string]any{
		"tenant": b.Tenant, "transaction_id": b.TransactionID,
		"captured_at": b.CapturedAt, "expires_at": b.ExpiresAt,
	}); err != nil {
		return err
	}
	for name, value := range map[string]any{
		"request.json":                  b.Request,
		"response.json":                 b.Response,
		"coraza/matched_rules.json":     b.Coraza,
		"vectorscan/qualification.json": b.VectorScan,
		"proxy/upstream.json":           b.Proxy,
		"tls/tls.json":                  b.TLS,
	} {
		if err := addJSON(name, value); err != nil {
			return err
		}
	}
	manifest := map[string]any{
		"format":         "waf-debug-incident-v1",
		"tenant":         tenant,
		"transaction_id": txid,
		"generated_at":   time.Now().UTC().Format(time.RFC3339Nano),
		"files":          map[string]string{},
	}
	hashes := manifest["files"].(map[string]string)
	for name, data := range files {
		h := sha256.Sum256(data)
		hashes[name] = hex.EncodeToString(h[:])
	}
	if err := addJSON("manifest.json", manifest); err != nil {
		return err
	}

	zw := zip.NewWriter(out)
	defer zw.Close()
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		h := &zip.FileHeader{Name: name, Method: zip.Deflate}
		h.SetMode(0640)
		h.SetModTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
		w, err := zw.CreateHeader(h)
		if err != nil {
			return err
		}
		if _, err := w.Write(files[name]); err != nil {
			return err
		}
	}
	return nil
}

// TenantExport is kept for compatibility with the earlier foundation API. It
// exports the newest incident for exactly one tenant and never includes another
// tenant's evidence.
func (s *DebugEvidenceStore) TenantExport(tenant string, out io.Writer) error {
	items := s.List(tenant, 1)
	if len(items) == 0 {
		return fmt.Errorf("no evidence for tenant %q", tenant)
	}
	return s.ExportIncident(tenant, items[0].TransactionID, out)
}

type debugRequestContext struct {
	Tenant        string
	TransactionID string
	Started       time.Time
}

type debugRequestContextKey struct{}

func debugContextFrom(ctx context.Context) (debugRequestContext, bool) {
	v, ok := ctx.Value(debugRequestContextKey{}).(debugRequestContext)
	return v, ok
}

func newDebugTransactionID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}
	return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
}

func debugEvidenceWrap(store *DebugEvidenceStore, tenant string, next http.Handler) http.Handler {
	if store == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now().UTC()
		if !store.ShouldCapture(tenant, now) {
			next.ServeHTTP(w, r)
			return
		}
		id := newDebugTransactionID()
		dc := debugRequestContext{Tenant: tenant, TransactionID: id, Started: now}
		r = r.WithContext(context.WithValue(r.Context(), debugRequestContextKey{}, dc))
		req := map[string]any{
			"method":          r.Method,
			"path":            r.URL.Path,
			"host":            r.Host,
			"protocol":        r.Proto,
			"content_length":  r.ContentLength,
			"client":          maskedDebugClientIP(clientIP(r)),
			"client_identity": map[string]any{"remote_addr": clientIP(r), "source": "REMOTE_ADDR"},
			"headers":         debugHeaderSubset(r.Header),
		}
		tlsEvidence := map[string]any{}
		if r.TLS != nil {
			tlsEvidence = map[string]any{
				"version":             tlsVersionName(r.TLS.Version),
				"cipher_suite":        tls.CipherSuiteName(r.TLS.CipherSuite),
				"server_name":         r.TLS.ServerName,
				"negotiated_protocol": r.TLS.NegotiatedProtocol,
				"resumed":             r.TLS.DidResume,
			}
		}
		store.Put(DebugBundle{Tenant: tenant, TransactionID: id, Request: req, TLS: tlsEvidence})
		w.Header().Set("X-WAF-Request-ID", id)
		sw := &statusRecorder{ResponseWriter: w, code: http.StatusOK}
		next.ServeHTTP(sw, r)
		store.Merge(tenant, id, func(b *DebugBundle) {
			b.Response = map[string]any{
				"status":      sw.code,
				"bytes":       sw.nbytes,
				"duration_ms": float64(time.Since(now).Microseconds()) / 1000.0,
			}
		})
	})
}

func debugHeaderSubset(h http.Header) map[string]any {
	out := map[string]any{}
	for _, k := range []string{"Accept", "Content-Type", "User-Agent", "Accept-Encoding"} {
		if v := h.Get(k); v != "" {
			out[k] = truncateDebugString(v, 512)
		}
	}
	return out
}

func maskedDebugClientIP(s string) string {
	ip := net.ParseIP(strings.TrimSpace(strings.Trim(s, "[]")))
	if ip == nil {
		return "[masked]"
	}
	if v4 := ip.To4(); v4 != nil {
		return fmt.Sprintf("%d.%d.%d.0/24", v4[0], v4[1], v4[2])
	}
	v6 := ip.To16()
	if v6 == nil {
		return "[masked]"
	}
	return fmt.Sprintf("%x:%x:%x:%x::/64", uint16(v6[0])<<8|uint16(v6[1]), uint16(v6[2])<<8|uint16(v6[3]), uint16(v6[4])<<8|uint16(v6[5]), uint16(v6[6])<<8|uint16(v6[7]))
}

func truncateDebugString(s string, max int) string {
	if max < 1 || len(s) <= max {
		return s
	}
	return s[:max] + "[truncated]"
}
