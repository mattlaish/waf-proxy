//go:build dashboard_reader_isolated

package main

// This test compiles only dashboard_reader.go and dashboard_reader_store.go.
// The full root package is presently blocked by unrelated, pre-existing debug
// symbols; these minimal runtime shapes exercise the reader without a listener.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kaptinlin/jsonschema"
)

type server struct {
	rt       atomic.Pointer[runtimeState]
	draining atomic.Bool
}

type runtimeState struct {
	cfg struct {
		Nodes      []struct{ Name string }
		Pools      []struct{ Name string }
		Sites      []struct{ Name, Listen, EngineMode string }
		EngineMode string
	}
	builtAt time.Time
	pools   map[string]*dashboardTestPool
}

type dashboardTestPool struct {
	monitor struct{ Type string }
	members []string
}

func (p *dashboardTestPool) healthyCount() int { return len(p.members) }

func dashboardTestReader(t *testing.T, scopes ...string) (*dashboardReader, string, string) {
	t.Helper()
	root := t.TempDir()
	token, digest, err := mintDashboardReaderToken()
	if err != nil {
		t.Fatal(err)
	}
	cfg := dashboardReaderConfig{
		Enabled: true, NetworkProfile: "SAME_HOST_PORT_19405",
		Listen: "127.0.0.1:19405", ServerName: "reader.example.test",
		TLSCert: filepath.Join(root, "cert.pem"), TLSKey: filepath.Join(root, "key.pem"),
		DataDir: filepath.Join(root, "data"), SourceInstance: "waf-test-01", TenantID: "tenant-test-01",
		Principals: []dashboardReaderPrincipal{{ID: "dashboard-machine-01", TokenHash: digest,
			Scopes: scopes, NotBefore: time.Now().Add(-time.Hour).UTC().Format(time.RFC3339),
			ExpiresAt: time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)}},
	}
	path := filepath.Join(root, "reader.json")
	dashboardTestWriteConfig(t, path, cfg)
	loaded, err := loadDashboardReaderConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	store, err := openDashboardStore(cfg.DataDir, cfg.SourceInstance)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if !store.closed.Load() {
			_ = store.close()
		}
	})
	return &dashboardReader{server: &server{}, store: store, configPath: path,
		config: loaded, adminToken: "old-admin-token", haToken: "old-ha-token"}, token, path
}

func dashboardTestWriteConfig(t *testing.T, path string, cfg dashboardReaderConfig) {
	t.Helper()
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func dashboardTestGET(d *dashboardReader, token, suffix string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, "https://reader.example.test:19405"+dashboardPrefix+suffix, nil)
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	d.handler().ServeHTTP(w, r)
	return w
}

func dashboardTestPage(t *testing.T, w *httptest.ResponseRecorder) dashboardPage {
	t.Helper()
	if w.Code != 200 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var page dashboardPage
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page.NextCursor == "" {
		t.Fatal("terminal page has no checkpoint cursor")
	}
	return page
}

func TestDashboardReaderAuthScopeRevocation(t *testing.T) {
	d, token, path := dashboardTestReader(t, "dashboard:read:health", "dashboard:read:capabilities", "dashboard:read:assets")
	if got := dashboardTestGET(d, "", "/health").Code; got != 401 {
		t.Fatalf("anonymous=%d", got)
	}
	if got := dashboardTestGET(d, "old-admin-token", "/health").Code; got != 401 {
		t.Fatalf("admin token=%d", got)
	}
	if got := dashboardTestGET(d, token, "/health").Code; got != 200 {
		t.Fatalf("health=%d", got)
	}
	if got := dashboardTestGET(d, token, "/policies").Code; got != 403 {
		t.Fatalf("scope=%d", got)
	}
	if got := dashboardTestGET(d, token, "/events").Code; got != 501 {
		t.Fatalf("optional lane=%d", got)
	}
	if got := dashboardTestGET(d, token, "/assets").Code; got != 200 {
		t.Fatalf("assets=%d", got)
	}
	post := httptest.NewRequest(http.MethodPost, "https://reader.example.test:19405"+dashboardPrefix+"/assets", nil)
	post.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	d.handler().ServeHTTP(w, post)
	if w.Code != 405 {
		t.Fatalf("mutation=%d", w.Code)
	}
	cookieOnly := httptest.NewRequest(http.MethodGet, "https://reader.example.test:19405"+dashboardPrefix+"/health", nil)
	cookieOnly.AddCookie(&http.Cookie{Name: "session", Value: token})
	w = httptest.NewRecorder()
	d.handler().ServeHTTP(w, cookieOnly)
	if w.Code != 401 {
		t.Fatalf("cookie-only=%d", w.Code)
	}
	cfg := d.config
	cfg.Principals[0].Revoked = true
	dashboardTestWriteConfig(t, path, cfg)
	if got := dashboardTestGET(d, token, "/health").Code; got != 401 {
		t.Fatalf("revoked=%d", got)
	}
}

func TestDashboardReaderSnapshotRecoveryAndGap(t *testing.T) {
	d, token, path := dashboardTestReader(t, "dashboard:read:assets")
	for _, id := range []string{"asset-a", "asset-b", "asset-c", "asset-d", "asset-e"} {
		if err := d.store.upsert("ASSET", id, map[string]any{"asset_type": "SERVICE"}, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	page1 := dashboardTestPage(t, dashboardTestGET(d, token, "/assets?limit=2"))
	if !page1.HasMore || len(page1.Records) != 2 {
		t.Fatalf("first page=%+v", page1)
	}
	if err := d.store.upsert("ASSET", "asset-f", map[string]any{"asset_type": "SERVICE"}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := d.store.close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := openDashboardStore(d.config.DataDir, d.config.SourceInstance)
	if err != nil {
		t.Fatal(err)
	}
	d.store = reopened
	t.Cleanup(func() { _ = reopened.close() })
	page2 := dashboardTestPage(t, dashboardTestGET(d, token, "/assets?limit=2&cursor="+page1.NextCursor))
	if !page2.HasMore || len(page2.Records) != 2 {
		t.Fatalf("recovered snapshot=%+v", page2)
	}
	page3 := dashboardTestPage(t, dashboardTestGET(d, token, "/assets?limit=2&cursor="+page2.NextCursor))
	if page3.HasMore || len(page3.Records) != 1 {
		t.Fatalf("third snapshot page=%+v", page3)
	}
	page4 := dashboardTestPage(t, dashboardTestGET(d, token, "/assets?cursor="+page3.NextCursor))
	if len(page4.Records) != 6 {
		t.Fatalf("new snapshot count=%d", len(page4.Records))
	}
	if len(d.store.snapshots) != 1 {
		t.Fatalf("one-page snapshot leaked; retained=%d", len(d.store.snapshots))
	}
	d.store.markGap()
	if got := dashboardTestGET(d, token, "/assets?cursor="+page4.NextCursor).Code; got != 409 {
		t.Fatalf("gap cursor=%d", got)
	}
	cfg := d.config
	cfg.Principals[0].Scopes = append(cfg.Principals[0].Scopes, "dashboard:read:health")
	dashboardTestWriteConfig(t, path, cfg)
	if got := dashboardTestGET(d, token, "/assets?cursor="+page2.NextCursor).Code; got != 409 {
		t.Fatalf("scope-change cursor=%d", got)
	}
}

func TestDashboardReaderDetectionDeltaPrivacy(t *testing.T) {
	d, token, _ := dashboardTestReader(t, "dashboard:read:detections")
	secret := "/login?password=never-export-this"
	if err := d.store.appendDetection(dashboardDetection{Site: "login-site" + secret, RuleID: 942100,
		Severity: "HIGH", At: time.Now()}); err != nil {
		t.Fatal(err)
	}
	initial := dashboardTestPage(t, dashboardTestGET(d, token, "/detections"))
	if initial.HasMore || len(initial.Records) != 1 {
		t.Fatalf("bootstrap=%+v", initial)
	}
	if err := d.store.appendDetection(dashboardDetection{Site: "login-site" + secret, RuleID: 942110,
		Severity: "HIGH", At: time.Now()}); err != nil {
		t.Fatal(err)
	}
	delta := dashboardTestPage(t, dashboardTestGET(d, token, "/detections?cursor="+initial.NextCursor))
	if delta.SyncMode != "INCREMENTAL" || len(delta.Records) != 1 {
		t.Fatalf("delta=%+v", delta)
	}
	if strings.Contains(initial.Records[0].ExternalID, secret) || strings.Contains(delta.Records[0].ExternalID, secret) ||
		strings.Contains(initial.Records[0].SourceObservedAt, secret) || strings.Contains(delta.Records[0].SourceObservedAt, secret) {
		t.Fatal("request secret leaked into exported identity fields")
	}
	journal, err := os.ReadFile(filepath.Join(d.config.DataDir, "journal.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(journal), secret) || strings.Contains(string(journal), "login-site") {
		t.Fatal("raw request or site name leaked into durable export")
	}
}

func TestDashboardReaderSharedPageSchema(t *testing.T) {
	path := os.Getenv("DASHBOARD_PAGE_SCHEMA_PATH")
	if path == "" {
		t.Skip("set DASHBOARD_PAGE_SCHEMA_PATH to the supplied OWI-1.0 page.schema.json")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	schema, err := jsonschema.NewCompiler().Compile(b)
	if err != nil {
		t.Fatal(err)
	}
	d, token, _ := dashboardTestReader(t, "dashboard:read:assets", "dashboard:read:detections",
		"dashboard:read:policies", "dashboard:read:health-observations")
	rt := &runtimeState{builtAt: time.Now().UTC(), pools: map[string]*dashboardTestPool{"origin-pool": {}}}
	rt.cfg.Nodes = append(rt.cfg.Nodes, struct{ Name string }{Name: "origin-node"})
	rt.cfg.Pools = append(rt.cfg.Pools, struct{ Name string }{Name: "origin-pool"})
	rt.cfg.Sites = append(rt.cfg.Sites, struct{ Name, Listen, EngineMode string }{
		Name: "login-site", Listen: "192.0.2.80:443", EngineMode: "On"})
	d.server.rt.Store(rt)
	if err := d.reconcileConfig(rt); err != nil {
		t.Fatal(err)
	}
	if err := d.observeHealth(rt); err != nil {
		t.Fatal(err)
	}
	if err := d.store.appendDetection(dashboardDetection{Site: "login-site", RuleID: 942100, At: time.Now()}); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"/assets", "/detections", "/policies", "/health-observations"} {
		w := dashboardTestGET(d, token, suffix)
		if w.Code != 200 {
			t.Fatalf("%s status=%d body=%s", suffix, w.Code, w.Body.String())
		}
		if result := schema.ValidateJSON(w.Body.Bytes()); !result.IsValid() {
			t.Fatalf("%s fails shared page schema: %+v", suffix, result)
		}
	}
	if err := d.store.deleteMissing("ASSET", map[string]bool{}, time.Now()); err != nil {
		t.Fatal(err)
	}
	w := dashboardTestGET(d, token, "/assets")
	if result := schema.ValidateJSON(w.Body.Bytes()); !result.IsValid() {
		t.Fatalf("asset DELETE tombstone fails schema: %+v", result)
	}
}
