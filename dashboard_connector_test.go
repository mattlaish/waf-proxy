package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testOWIConfig(t *testing.T, rate float64, burst int) (owiConnectorConfig, string, *server) {
	t.Helper()
	dir := t.TempDir()
	secretPath := filepath.Join(dir, "cursor.secret")
	if err := os.WriteFile(secretPath, []byte(strings.Repeat("c", 48)), 0o600); err != nil {
		t.Fatal(err)
	}
	token := strings.Repeat("a", 64)
	now := time.Now().UTC()
	reg := owiTokenRegistry{Version: 1, Tokens: []owiReaderToken{{ID: "reader-1", Principal: "dashboard-reader-1", TenantID: "tenant-1", DigestSHA256: owiTokenDigest(token), Scopes: []string{owiScopeHealth, owiScopeCaps, owiScopeAssets, owiScopeDetections, owiScopePolicies, owiScopeHealthObs}, IssuedAt: now.Add(-time.Hour).Format(time.RFC3339), ExpiresAt: now.Add(24 * time.Hour).Format(time.RFC3339)}}}
	b, _ := json.Marshal(reg)
	regPath := filepath.Join(dir, "tokens.json")
	if err := os.WriteFile(regPath, b, 0o600); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfgPath, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := &server{configPath: cfgPath, metrics: newMetrics(), observations: newObservationPlane(nil, nil, nil, nil, nil, 8), matchLogs: newMatchLogPlane(nil, 8, 8, time.Second)}
	rt := &runtimeState{cfg: Config{EngineMode: "On", Nodes: []NodeConfig{{Name: "app-1", Host: "192.0.2.10"}, {Name: "app-2", Host: "192.0.2.11"}}, Pools: []PoolConfig{{Name: "origin-a", Scheme: "http", LBMethod: "round_robin", Members: []MemberConfig{{Node: "app-1", Port: 8080, Weight: 1}}}}, Policies: []PolicyConfig{{Name: "default", RulesPath: "coraza.conf", ParanoiaLevel: 1}}, Sites: []SiteConfig{{Name: "site-a", Listen: "127.0.0.2:8443", Hostnames: []string{"app.example.test", "api.example.test"}, Pool: "origin-a", Policy: "default"}}}}
	s.rt.Store(rt)
	cfg := owiConnectorConfig{Enabled: true, Addr: "127.0.0.1:19405", AllowInsecureTest: true, SourceInstanceID: "waf-test-01", TokenRegistryPath: regPath, CursorSecretFile: secretPath, StatePath: filepath.Join(dir, "state.json"), RatePerSecond: rate, Burst: burst, PerPrincipalConcurrent: 2, GlobalConcurrent: 4, RequestDeadline: 2 * time.Second, MaxPageSize: 500, MaxResponseBytes: 4 << 20}
	return cfg, token, s
}

func newTestOWI(t *testing.T, rate float64, burst int) (*owiConnector, string, *server) {
	t.Helper()
	cfg, token, s := testOWIConfig(t, rate, burst)
	c, err := newOWIConnector(s, cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	s.dashboard = c
	t.Cleanup(func() { _ = c.close() })
	return c, token, s
}

func owiRequest(t *testing.T, h http.Handler, token, method, path string) (int, http.Header, []byte) {
	t.Helper()
	r := httptest.NewRequest(method, path, nil)
	r.Header.Set("Accept", "application/json")
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w.Code, w.Header(), w.Body.Bytes()
}

func TestOWIAuthCapabilitiesDisabledAndMutation(t *testing.T) {
	c, token, _ := newTestOWI(t, 100, 100)
	h := c.handler()
	if code, _, _ := owiRequest(t, h, "", http.MethodGet, owiBasePath+"/health"); code != 401 {
		t.Fatalf("unauth health=%d", code)
	}
	cookieReq := httptest.NewRequest(http.MethodGet, owiBasePath+"/health", nil)
	cookieReq.Header.Set("Accept", "application/json")
	cookieReq.AddCookie(&http.Cookie{Name: "session", Value: "browser-only"})
	cookieRec := httptest.NewRecorder()
	h.ServeHTTP(cookieRec, cookieReq)
	if cookieRec.Code != http.StatusUnauthorized {
		t.Fatalf("cookie-only auth=%d", cookieRec.Code)
	}
	code, _, body := owiRequest(t, h, token, http.MethodGet, owiBasePath+"/capabilities")
	if code != 200 {
		t.Fatalf("caps=%d %s", code, body)
	}
	var caps owiCapabilitiesResponse
	if err := json.Unmarshal(body, &caps); err != nil {
		t.Fatal(err)
	}
	if len(caps.Resources) != 6 || len(caps.Actions) != 0 {
		t.Fatalf("unexpected caps %#v", caps)
	}
	if code, _, _ := owiRequest(t, h, token, http.MethodGet, owiBasePath+"/events"); code != 501 {
		t.Fatalf("disabled events=%d", code)
	}
	if code, _, _ := owiRequest(t, h, token, http.MethodPost, owiBasePath+"/assets"); code != 405 {
		t.Fatalf("mutation=%d", code)
	}
}

func TestOWISnapshotThreePagesDeltaAndTamper(t *testing.T) {
	c, token, s := newTestOWI(t, 100, 100)
	h := c.handler()
	path := owiBasePath + "/assets?limit=2"
	pages := 0
	var final string
	total := 0
	for {
		code, _, body := owiRequest(t, h, token, http.MethodGet, path)
		if code != 200 {
			t.Fatalf("page=%d %s", code, body)
		}
		var p owiPage
		if err := json.Unmarshal(body, &p); err != nil {
			t.Fatal(err)
		}
		pages++
		total += len(p.Records)
		final = p.NextCursor
		if !p.HasMore {
			break
		}
		path = owiBasePath + "/assets?limit=2&cursor=" + p.NextCursor
	}
	if pages < 3 || total < 5 {
		t.Fatalf("wanted >=3 pages and >=5 assets, pages=%d records=%d", pages, total)
	}
	rt := s.rt.Load()
	next := rt.cfg
	next.Nodes = append(next.Nodes, NodeConfig{Name: "app-3", Host: "192.0.2.12"})
	s.rt.Store(&runtimeState{cfg: next})
	code, _, body := owiRequest(t, h, token, http.MethodGet, owiBasePath+"/assets?limit=2&cursor="+final)
	if code != 200 {
		t.Fatalf("delta=%d %s", code, body)
	}
	var delta owiPage
	_ = json.Unmarshal(body, &delta)
	if delta.SyncMode != "INCREMENTAL" || len(delta.Records) == 0 {
		t.Fatalf("expected non-empty delta %#v", delta)
	}
	tampered := final[:len(final)-1] + "A"
	if code, _, _ := owiRequest(t, h, token, http.MethodGet, owiBasePath+"/assets?limit=2&cursor="+tampered); code != 400 {
		t.Fatalf("tampered cursor=%d", code)
	}
}

func TestOWISnapshotSurvivesConnectorRestart(t *testing.T) {
	cfg, token, s := testOWIConfig(t, 100, 100)
	c, err := newOWIConnector(s, cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	s.dashboard = c
	h := c.handler()
	code, _, body := owiRequest(t, h, token, http.MethodGet, owiBasePath+"/assets?limit=2")
	if code != 200 {
		t.Fatalf("first=%d %s", code, body)
	}
	var p owiPage
	_ = json.Unmarshal(body, &p)
	if !p.HasMore {
		t.Fatal("expected paged snapshot")
	}
	cursor := p.NextCursor
	if err := c.close(); err != nil {
		t.Fatal(err)
	}
	c2, err := newOWIConnector(s, cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c2.close()
	s.dashboard = c2
	code, _, body = owiRequest(t, c2.handler(), token, http.MethodGet, owiBasePath+"/assets?limit=2&cursor="+cursor)
	if code != 200 {
		t.Fatalf("resume=%d %s", code, body)
	}
	var p2 owiPage
	_ = json.Unmarshal(body, &p2)
	if p2.SourceSnapshotID == nil || *p2.SourceSnapshotID != *p.SourceSnapshotID {
		t.Fatalf("snapshot changed across restart")
	}
}

func TestOWIDetectionSanitizationAndHistory(t *testing.T) {
	c, token, _ := newTestOWI(t, 100, 100)
	c.noteWAFMatch(matchRec{Site: "site-a", RuleID: 942100, Severity: "CRITICAL", URI: "/users/123456?token=secret"}, time.Now().UTC())
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		code, _, body := owiRequest(t, c.handler(), token, http.MethodGet, owiBasePath+"/detections?limit=10")
		if code == 200 {
			var p owiPage
			_ = json.Unmarshal(body, &p)
			if len(p.Records) > 0 {
				raw := string(body)
				if strings.Contains(raw, "secret") || strings.Contains(raw, "123456") {
					t.Fatalf("sensitive/raw path leaked: %s", raw)
				}
				if !strings.Contains(raw, "{id}") {
					t.Fatalf("expected templated path: %s", raw)
				}
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("detection was not durably exported")
}

func TestOWIRateLimit429RetryAfter(t *testing.T) {
	c, token, _ := newTestOWI(t, 0.01, 1)
	h := c.handler()
	if code, _, _ := owiRequest(t, h, token, http.MethodGet, owiBasePath+"/health"); code != 200 {
		t.Fatalf("first=%d", code)
	}
	code, hdr, body := owiRequest(t, h, token, http.MethodGet, owiBasePath+"/health")
	if code != 429 {
		t.Fatalf("second=%d %s", code, body)
	}
	if hdr.Get("Retry-After") == "" {
		t.Fatal("missing Retry-After")
	}
}

func TestOWIUncleanRestartMarksDetectionCoverageGap(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	s, err := newOWIStore(path, "waf-test-01")
	if err != nil {
		t.Fatal(err)
	}
	// Deliberately do not close clean; a second process sees dirty state.
	s2, err := newOWIStore(path, "waf-test-01")
	if err != nil {
		t.Fatal(err)
	}
	defer s2.closeClean()
	if got := s2.coverage("DETECTION").State; got != "GAP" {
		t.Fatalf("coverage=%s", got)
	}
	_ = s
}

func TestOWITokenRevocationReload(t *testing.T) {
	c, token, _ := newTestOWI(t, 100, 100)
	if code, _, _ := owiRequest(t, c.handler(), token, http.MethodGet, owiBasePath+"/health"); code != 200 {
		t.Fatalf("before revoke=%d", code)
	}
	reg, err := c.tokens.load()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	reg.Tokens[0].RevokedAt = &now
	b, _ := json.MarshalIndent(reg, "", "  ")
	if err := os.WriteFile(c.cfg.TokenRegistryPath, append(b, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, _, _ := owiRequest(t, c.handler(), token, http.MethodGet, owiBasePath+"/health"); code != 401 {
		t.Fatalf("after revoke=%d", code)
	}
}

func TestOWIDeadlineReleasesLimiter(t *testing.T) {
	c, token, _ := newTestOWI(t, 100, 100)
	c.cfg.RequestDeadline = 20 * time.Millisecond
	c.syncHook = func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }
	code, _, body := owiRequest(t, c.handler(), token, http.MethodGet, owiBasePath+"/assets?limit=1")
	if code != 503 {
		t.Fatalf("deadline=%d %s", code, body)
	}
	c.syncHook = c.syncNativeState
	code, _, body = owiRequest(t, c.handler(), token, http.MethodGet, owiBasePath+"/assets?limit=1")
	if code != 200 {
		t.Fatalf("limiter leaked after timeout: %d %s", code, body)
	}
}

func TestOWIPlaneSeparation(t *testing.T) {
	if err := owiPlaneCheck("192.0.2.20:19405", "192.0.2.20:19405", nil); err == nil {
		t.Fatal("reader/admin overlap accepted")
	}
	if err := owiPlaneCheck("192.0.2.20:443", "127.0.0.1:9090", []SiteConfig{{Listen: "192.0.2.20:443"}}); err == nil {
		t.Fatal("reader/data overlap accepted")
	}
	if err := owiPlaneCheck("192.0.2.21:443", "127.0.0.1:9090", []SiteConfig{{Listen: "192.0.2.20:443"}}); err != nil {
		t.Fatalf("dedicated mgmt IP rejected: %v", err)
	}
}

func TestOWIWrongScopeAcceptAndNoRedirect(t *testing.T) {
	c, _, _ := newTestOWI(t, 100, 100)
	token2 := strings.Repeat("b", 64)
	reg, err := c.tokens.load()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	reg.Tokens = append(reg.Tokens, owiReaderToken{ID: "reader-health-only", Principal: "dashboard-health-reader", TenantID: "tenant-1", DigestSHA256: owiTokenDigest(token2), Scopes: []string{owiScopeHealth}, IssuedAt: now.Add(-time.Hour).Format(time.RFC3339), ExpiresAt: now.Add(time.Hour).Format(time.RFC3339)})
	b, _ := json.MarshalIndent(reg, "", "  ")
	if err := os.WriteFile(c.cfg.TokenRegistryPath, append(b, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, _, _ := owiRequest(t, c.handler(), token2, http.MethodGet, owiBasePath+"/assets"); code != 403 {
		t.Fatalf("wrong scope=%d", code)
	}
	r := httptest.NewRequest(http.MethodGet, owiBasePath+"/health", nil)
	r.Header.Set("Authorization", "Bearer "+token2)
	r.Header.Set("Accept", "text/html")
	w := httptest.NewRecorder()
	c.handler().ServeHTTP(w, r)
	if w.Code != 400 {
		t.Fatalf("invalid accept=%d", w.Code)
	}
	if code, hdr, _ := owiRequest(t, c.handler(), token2, http.MethodGet, owiBasePath+"/health/"); code != 404 || hdr.Get("Location") != "" {
		t.Fatalf("trailing slash redirect/response code=%d location=%q", code, hdr.Get("Location"))
	}
}

func TestOWIAttentionMappingsHaveCounterexamples(t *testing.T) {
	c, _, s := newTestOWI(t, 100, 100)
	now := time.Now().UTC().Truncate(time.Minute)
	hi := owiDetectionRecord(c.cfg.SourceInstanceID, owiNativeDetection{At: now, Site: "site-a", RuleID: 942100, Severity: "CRITICAL", Path: "/login"})
	lo := owiDetectionRecord(c.cfg.SourceInstanceID, owiNativeDetection{At: now, Site: "site-a", RuleID: 920100, Severity: "INFO", Path: "/status"})
	if hi.Payload["attention"] == nil || lo.Payload["attention"] != nil {
		t.Fatal("high-signal attention mapping/counterexample invalid")
	}
	s.draining.Store(true)
	degraded := c.healthRecord(s.rt.Load().cfg, now)
	if degraded.Payload["attention"] == nil {
		t.Fatal("missing protection degraded attention")
	}
	s.draining.Store(false)
	for i := 0; i < 7; i++ {
		s.observations.queue <- observationEvent{kind: observationHost}
	}
	capRec := c.healthRecord(s.rt.Load().cfg, now)
	if capRec.Payload["attention"] == nil {
		t.Fatal("missing capacity pressure attention")
	}
	for len(s.observations.queue) > 0 {
		<-s.observations.queue
	}
	normal := c.healthRecord(s.rt.Load().cfg, now)
	if normal.Payload["attention"] != nil {
		t.Fatal("normal health must be attention counterexample")
	}
}

func TestOWIExpiredTokenAndCursorIdentityReset(t *testing.T) {
	c, token, _ := newTestOWI(t, 100, 100)
	h := c.handler()
	code, _, body := owiRequest(t, h, token, http.MethodGet, owiBasePath+"/assets?limit=2")
	if code != 200 {
		t.Fatalf("first snapshot=%d %s", code, body)
	}
	var page owiPage
	if err := json.Unmarshal(body, &page); err != nil {
		t.Fatal(err)
	}

	reg, err := c.tokens.load()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	other := strings.Repeat("d", 64)
	otherTenant := strings.Repeat("c", 64)
	expired := strings.Repeat("e", 64)
	reg.Tokens = append(reg.Tokens,
		owiReaderToken{ID: "reader-2", Principal: "dashboard-reader-2", TenantID: "tenant-1", DigestSHA256: owiTokenDigest(other), Scopes: []string{owiScopeAssets}, IssuedAt: now.Add(-time.Hour).Format(time.RFC3339), ExpiresAt: now.Add(time.Hour).Format(time.RFC3339)},
		owiReaderToken{ID: "reader-other-tenant", Principal: "dashboard-reader-1", TenantID: "tenant-2", DigestSHA256: owiTokenDigest(otherTenant), Scopes: []string{owiScopeAssets}, IssuedAt: now.Add(-time.Hour).Format(time.RFC3339), ExpiresAt: now.Add(time.Hour).Format(time.RFC3339)},
		owiReaderToken{ID: "reader-expired", Principal: "dashboard-reader-expired", TenantID: "tenant-1", DigestSHA256: owiTokenDigest(expired), Scopes: []string{owiScopeHealth}, IssuedAt: now.Add(-2 * time.Hour).Format(time.RFC3339), ExpiresAt: now.Add(-time.Hour).Format(time.RFC3339)},
	)
	b, _ := json.MarshalIndent(reg, "", "  ")
	if err := os.WriteFile(c.cfg.TokenRegistryPath, append(b, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, _, _ := owiRequest(t, h, expired, http.MethodGet, owiBasePath+"/health"); code != 401 {
		t.Fatalf("expired token=%d", code)
	}
	if code, _, _ := owiRequest(t, h, other, http.MethodGet, owiBasePath+"/assets?limit=2&cursor="+page.NextCursor); code != 409 {
		t.Fatalf("cross-principal cursor=%d", code)
	}
	if code, _, _ := owiRequest(t, h, otherTenant, http.MethodGet, owiBasePath+"/assets?limit=2&cursor="+page.NextCursor); code != 409 {
		t.Fatalf("cross-tenant cursor=%d", code)
	}
}

func TestOWILimiterIdentityIsolationAndGlobalBound(t *testing.T) {
	cfg := owiConnectorConfig{RatePerSecond: 0.01, Burst: 1, PerPrincipalConcurrent: 1, GlobalConcurrent: 1}
	l := newOWILimiter(cfg)
	now := time.Now()
	releaseA, _, ok := l.acquire("tenant-a\x00reader-a", now)
	if !ok {
		t.Fatal("reader A initial acquire rejected")
	}
	if _, _, ok := l.acquire("tenant-b\x00reader-b", now); ok {
		t.Fatal("global concurrency bound not enforced")
	}
	releaseA()
	if _, _, ok := l.acquire("tenant-a\x00reader-a", now); ok {
		t.Fatal("reader A rate budget unexpectedly replenished")
	}
	releaseB, _, ok := l.acquire("tenant-b\x00reader-b", now)
	if !ok {
		t.Fatal("reader B must have an independent rate budget")
	}
	releaseB()
}

func TestOWISnapshotCapacityRejectsWithoutInvalidatingExistingCursor(t *testing.T) {
	c, token, _ := newTestOWI(t, 100, 100)
	c.store.limits.SnapshotMaxCount = 1
	h := c.handler()
	code, _, body := owiRequest(t, h, token, http.MethodGet, owiBasePath+"/assets?limit=2")
	if code != 200 {
		t.Fatalf("first=%d %s", code, body)
	}
	var page owiPage
	_ = json.Unmarshal(body, &page)
	if code, _, _ := owiRequest(t, h, token, http.MethodGet, owiBasePath+"/policies?limit=1"); code != 503 {
		t.Fatalf("snapshot capacity=%d", code)
	}
	if code, _, body := owiRequest(t, h, token, http.MethodGet, owiBasePath+"/assets?limit=2&cursor="+page.NextCursor); code != 200 {
		t.Fatalf("existing cursor after reject=%d %s", code, body)
	}
}

func TestOWIDetectionExportCapacityMarksGap(t *testing.T) {
	c, _, _ := newTestOWI(t, 100, 100)
	c.store.limits.DetectionMaxRows = 1
	c.store.limits.DetectionMaxBytes = 1 << 20
	now := time.Now().UTC()
	c.noteWAFMatch(matchRec{Site: "site-a", RuleID: 1, Severity: "HIGH", URI: "/a"}, now)
	c.noteWAFMatch(matchRec{Site: "site-a", RuleID: 2, Severity: "HIGH", URI: "/b"}, now.Add(time.Millisecond))
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		_, rows, _ := c.store.stats()
		if rows >= 1 && c.store.coverage("DETECTION").State == "GAP" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("capacity did not produce explicit gap: coverage=%#v", c.store.coverage("DETECTION"))
}

func TestOWIStrictGETOnlyRejectsHEAD(t *testing.T) {
	c, token, _ := newTestOWI(t, 100, 100)
	if code, _, _ := owiRequest(t, c.handler(), token, http.MethodHead, owiBasePath+"/health"); code != http.StatusMethodNotAllowed {
		t.Fatalf("HEAD health=%d", code)
	}
	if code, _, _ := owiRequest(t, c.handler(), token, http.MethodHead, owiBasePath+"/events"); code != http.StatusMethodNotAllowed {
		t.Fatalf("HEAD disabled route=%d", code)
	}
}

func TestOWIDisabledOptionalScopesRejected(t *testing.T) {
	now := time.Now().UTC()
	reg := owiTokenRegistry{Version: 1, Tokens: []owiReaderToken{{
		ID: "reader-bad-scope", Principal: "dashboard-reader", TenantID: "tenant-1",
		DigestSHA256: owiTokenDigest(strings.Repeat("f", 64)),
		Scopes:       []string{owiScopeHealth, owiScopeEvents},
		IssuedAt:     now.Add(-time.Minute).Format(time.RFC3339), ExpiresAt: now.Add(time.Hour).Format(time.RFC3339),
	}}}
	if err := validateOWITokenRegistry(reg); err == nil {
		t.Fatal("disabled optional scope accepted")
	}
}

func TestOWISecretAndTokenFilesRejectSymlinkAndBroadMode(t *testing.T) {
	dir := t.TempDir()
	secret := filepath.Join(dir, "cursor.secret")
	if err := os.WriteFile(secret, []byte(strings.Repeat("s", 48)), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadOWISecretFile(secret, "test secret"); err == nil {
		t.Fatal("broad-mode cursor secret accepted")
	}
	if err := os.Chmod(secret, 0o600); err != nil {
		t.Fatal(err)
	}
	secretLink := filepath.Join(dir, "cursor.link")
	if err := os.Symlink(secret, secretLink); err != nil {
		t.Fatal(err)
	}
	if _, err := loadOWISecretFile(secretLink, "test secret"); err == nil {
		t.Fatal("symlink cursor secret accepted")
	}

	now := time.Now().UTC()
	reg := owiTokenRegistry{Version: 1, Tokens: []owiReaderToken{{ID: "reader-1", Principal: "reader-1", TenantID: "tenant-1", DigestSHA256: owiTokenDigest(strings.Repeat("a", 64)), Scopes: []string{owiScopeHealth}, IssuedAt: now.Add(-time.Minute).Format(time.RFC3339), ExpiresAt: now.Add(time.Hour).Format(time.RFC3339)}}}
	b, _ := json.Marshal(reg)
	regPath := filepath.Join(dir, "tokens.json")
	if err := os.WriteFile(regPath, b, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := (&owiTokenSource{path: regPath}).load(); err == nil {
		t.Fatal("broad-mode token registry accepted")
	}
	if err := os.Chmod(regPath, 0o600); err != nil {
		t.Fatal(err)
	}
	regLink := filepath.Join(dir, "tokens.link")
	if err := os.Symlink(regPath, regLink); err != nil {
		t.Fatal(err)
	}
	if _, err := (&owiTokenSource{path: regLink}).load(); err == nil {
		t.Fatal("symlink token registry accepted")
	}
}

func TestOWIOversizeResponseFailsClosed(t *testing.T) {
	c, token, _ := newTestOWI(t, 100, 100)
	c.cfg.MaxResponseBytes = 128
	code, _, body := owiRequest(t, c.handler(), token, http.MethodGet, owiBasePath+"/capabilities")
	if code != http.StatusServiceUnavailable {
		t.Fatalf("oversize response=%d %s", code, body)
	}
	if !strings.Contains(string(body), "TEMPORARY_UNAVAILABLE") {
		t.Fatalf("unexpected oversize error: %s", body)
	}
}

func TestOWIClientCancelReleasesLimiter(t *testing.T) {
	c, token, _ := newTestOWI(t, 100, 100)
	c.cfg.RequestDeadline = 5 * time.Second
	c.syncHook = func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }
	r := httptest.NewRequest(http.MethodGet, owiBasePath+"/assets?limit=1", nil)
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer "+token)
	ctx, cancel := context.WithCancel(r.Context())
	r = r.WithContext(ctx)
	cancel()
	w := httptest.NewRecorder()
	c.handler().ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("cancel response=%d", w.Code)
	}
	c.syncHook = c.syncNativeState
	if code, _, body := owiRequest(t, c.handler(), token, http.MethodGet, owiBasePath+"/assets?limit=1"); code != http.StatusOK {
		t.Fatalf("limiter leaked after client cancel: %d %s", code, body)
	}
}

func TestOWIScopeReductionInvalidatesExistingCursor(t *testing.T) {
	c, token, _ := newTestOWI(t, 100, 100)
	code, _, body := owiRequest(t, c.handler(), token, http.MethodGet, owiBasePath+"/assets?limit=2")
	if code != http.StatusOK {
		t.Fatalf("first snapshot=%d %s", code, body)
	}
	var page owiPage
	if err := json.Unmarshal(body, &page); err != nil {
		t.Fatal(err)
	}
	reg, err := c.tokens.load()
	if err != nil {
		t.Fatal(err)
	}
	// Keep the same tenant/principal/token and required assets scope, but shrink
	// visibility elsewhere. The scope hash must invalidate the old cursor.
	reg.Tokens[0].Scopes = []string{owiScopeAssets}
	b, _ := json.MarshalIndent(reg, "", "  ")
	if err := os.WriteFile(c.cfg.TokenRegistryPath, append(b, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, _, _ := owiRequest(t, c.handler(), token, http.MethodGet, owiBasePath+"/assets?limit=2&cursor="+page.NextCursor); code != http.StatusConflict {
		t.Fatalf("scope-reduced cursor=%d", code)
	}
}

func TestOWICursorAndDetectionRetentionExpiry(t *testing.T) {
	c, token, _ := newTestOWI(t, 100, 100)
	code, _, body := owiRequest(t, c.handler(), token, http.MethodGet, owiBasePath+"/assets?limit=2")
	if code != http.StatusOK {
		t.Fatalf("snapshot=%d %s", code, body)
	}
	var page owiPage
	if err := json.Unmarshal(body, &page); err != nil {
		t.Fatal(err)
	}
	cur, err := decodeOWICursor(c.cursorSecret, page.NextCursor)
	if err != nil {
		t.Fatal(err)
	}
	cur.ExpiresAtUnix = time.Now().Add(-time.Second).Unix()
	expired, err := encodeOWICursor(c.cursorSecret, cur)
	if err != nil {
		t.Fatal(err)
	}
	if code, _, _ := owiRequest(t, c.handler(), token, http.MethodGet, owiBasePath+"/assets?limit=2&cursor="+expired); code != http.StatusConflict {
		t.Fatalf("expired cursor=%d", code)
	}

	old := time.Now().UTC().Add(-owiHistoryRetain - time.Hour)
	rec := owiDetectionRecord(c.cfg.SourceInstanceID, owiNativeDetection{At: old, Site: "site-a", RuleID: 1, Severity: "HIGH", Path: "/old"})
	if err := c.store.appendDetection(rec, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	rows, _, _, _ := c.store.journalAfter("DETECTION", 0)
	for _, r := range rows {
		if r.ExternalID == rec.ExternalID {
			t.Fatal("expired detection retained beyond declared history window")
		}
	}
}

func TestOWIStoreCrashHelper(t *testing.T) {
	if os.Getenv("WAF_OWI_CRASH_HELPER") != "1" {
		t.Skip("subprocess helper")
	}
	statePath := os.Getenv("WAF_OWI_CRASH_STATE")
	readyPath := os.Getenv("WAF_OWI_CRASH_READY")
	if statePath == "" || readyPath == "" {
		os.Exit(92)
	}
	if _, err := newOWIStore(statePath, "waf-test-01"); err != nil {
		os.Exit(93)
	}
	if err := os.WriteFile(readyPath, []byte("ready\n"), 0o600); err != nil {
		os.Exit(94)
	}
	for {
		time.Sleep(time.Second)
	}
}

func TestOWIRealProcessKillMarksCoverageGap(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	readyPath := filepath.Join(dir, "ready")
	cmd := exec.Command(os.Args[0], "-test.run=^TestOWIStoreCrashHelper$", "-test.count=1")
	cmd.Env = append(os.Environ(), "WAF_OWI_CRASH_HELPER=1", "WAF_OWI_CRASH_STATE="+statePath, "WAF_OWI_CRASH_READY="+readyPath)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		if _, err := os.Stat(readyPath); err == nil {
			break
		}
		if time.Now().After(deadline) {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
			t.Fatal("crash helper did not become ready")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_, _ = cmd.Process.Wait()
	s, err := newOWIStore(statePath, "waf-test-01")
	if err != nil {
		t.Fatal(err)
	}
	defer s.closeClean()
	if got := s.coverage("DETECTION").State; got != "GAP" {
		t.Fatalf("coverage after SIGKILL=%s", got)
	}
}

func TestOWIStoreContentionHonorsDeadlineAndRetry(t *testing.T) {
	c, token, _ := newTestOWI(t, 100, 100)
	c.cfg.RequestDeadline = 25 * time.Millisecond
	h := c.handler()

	c.store.mu.Lock()
	started := time.Now()
	code, _, body := owiRequest(t, h, token, http.MethodGet, owiBasePath+"/assets?limit=1")
	elapsed := time.Since(started)
	c.store.mu.Unlock()
	if code != http.StatusServiceUnavailable {
		t.Fatalf("store contention=%d %s", code, body)
	}
	if elapsed > 150*time.Millisecond {
		t.Fatalf("store contention exceeded bounded deadline: %s", elapsed)
	}
	if code, _, body := owiRequest(t, h, token, http.MethodGet, owiBasePath+"/assets?limit=1"); code != http.StatusOK {
		t.Fatalf("retry after contention=%d %s", code, body)
	}
}

func TestOWIStoreWriteFailureFailsClosedAndRecovers(t *testing.T) {
	c, token, _ := newTestOWI(t, 100, 100)
	h := c.handler()
	goodPath := c.store.path
	badParent := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(badParent, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	c.store.path = filepath.Join(badParent, "state.json")
	code, _, body := owiRequest(t, h, token, http.MethodGet, owiBasePath+"/assets?limit=1")
	if code != http.StatusServiceUnavailable {
		t.Fatalf("write failure=%d %s", code, body)
	}
	if !c.store.writeFailed {
		t.Fatal("store write failure was not latched fail-closed")
	}
	if cov, err := c.store.coverageContext(context.Background(), "DETECTION"); err == nil || cov.State != "" {
		t.Fatalf("durability failure remained readable: coverage=%#v err=%v", cov, err)
	}
	c.store.path = goodPath
	if code, _, body := owiRequest(t, h, token, http.MethodGet, owiBasePath+"/assets?limit=1"); code != http.StatusOK {
		t.Fatalf("retry after storage recovery=%d %s", code, body)
	}
	if c.store.writeFailed {
		t.Fatal("store durability latch did not clear after successful persist")
	}
}
