package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestProductionCorrelationIDValidation(t *testing.T) {
	valid := []string{"request-1234", "abc_DEF.123:xyz", "01234567"}
	for _, id := range valid {
		if !validRequestCorrelationID(id) {
			t.Fatalf("expected valid request id %q", id)
		}
	}
	invalid := []string{"short", "<script>alert(1)</script>", "contains space", strings.Repeat("a", 129)}
	for _, id := range invalid {
		if validRequestCorrelationID(id) {
			t.Fatalf("expected invalid request id %q", id)
		}
	}
}

func TestProductionBlockResponseEscapesReasonAndRequestID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://example.test/", nil)
	req = req.WithContext(context.WithValue(req.Context(), requestCorrelationKey{}, `<img src=x onerror=alert(1)>`))
	rr := httptest.NewRecorder()
	writeBlockResponse(rr, req, http.StatusForbidden, `<script>alert("reason")</script>`, false)
	body := rr.Body.String()
	if strings.Contains(body, "<script>") || strings.Contains(body, "<img src=x") {
		t.Fatalf("block page reflected raw HTML: %s", body)
	}
	if !strings.Contains(body, "&lt;script&gt;") || !strings.Contains(body, "&lt;img") {
		t.Fatalf("block page did not HTML-escape untrusted values: %s", body)
	}
	if got := rr.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control=%q", got)
	}
}

func TestProductionHASharedConfigStripsNodeLocalAndSecrets(t *testing.T) {
	cfg := defaultConfig()
	cfg.HA = HAConfig{Enabled: true, Role: "primary", PeerURL: "https://peer.example", PeerToken: "peer-secret", SyncConfig: true}
	cfg.Users = []UserConfig{{Username: "admin", PasswordHash: "secret-hash", Role: roleAdmin}}
	cfg.AI.APIKey = "inline-ai-secret"
	cfg.AI.APIKeyRef = "env:AI_SECRET"
	cfg.Notify.WebhookURL = "https://hooks.example/secret"
	out := sharedConfigForPeer(cfg)
	if out.HA != (HAConfig{}) {
		t.Fatalf("HA node identity leaked into shared payload: %+v", out.HA)
	}
	if len(out.Users) != 0 {
		t.Fatalf("users leaked into shared payload")
	}
	if out.AI.APIKey != "" || out.AI.APIKeyRef != "" || out.Notify.WebhookURL != "" {
		t.Fatalf("secret material leaked into shared payload")
	}
}

func TestProductionHAMergeRestoresReceiverLocalState(t *testing.T) {
	local := defaultConfig()
	local.HA = HAConfig{Enabled: true, Role: "secondary", PeerURL: "https://primary.example", PeerToken: "receiver-secret", SyncConfig: true}
	local.Users = []UserConfig{{Username: "receiver", PasswordHash: "receiver-hash", Role: roleAdmin}}
	local.AI.APIKeyRef = "env:RECEIVER_AI_KEY"
	local.Notify.WebhookURL = "https://hooks.example/receiver"
	incoming := defaultConfig()
	incoming.EngineMode = "On"
	incoming.HA = HAConfig{Enabled: true, Role: "primary", PeerURL: "https://wrong.example", PeerToken: "wrong", SyncConfig: true}
	merged := mergePeerConfig(local, incoming)
	if merged.HA != local.HA {
		t.Fatalf("receiver HA identity was not preserved: %+v", merged.HA)
	}
	if len(merged.Users) != 1 || merged.Users[0].Username != "receiver" {
		t.Fatalf("receiver users were not preserved")
	}
	if merged.AI.APIKeyRef != local.AI.APIKeyRef || merged.Notify.WebhookURL != local.Notify.WebhookURL {
		t.Fatalf("receiver secrets were not preserved")
	}
	if merged.EngineMode != "On" {
		t.Fatalf("non-local shared config did not merge")
	}
}

func TestProductionHARequiresHTTPSOrigin(t *testing.T) {
	bad := []string{"http://peer.example", "https://user@peer.example", "https://peer.example/path", "https://peer.example?q=1"}
	for _, peer := range bad {
		cfg := HAConfig{Enabled: true, Role: "primary", PeerURL: peer, PeerToken: "x", SyncConfig: true}
		if err := cfg.validate(); err == nil {
			t.Fatalf("expected HA peer URL rejection for %q", peer)
		}
	}
	good := HAConfig{Enabled: true, Role: "primary", PeerURL: "https://peer.example:9443", PeerToken: "x", SyncConfig: true}
	if err := good.validate(); err != nil {
		t.Fatalf("valid HA peer rejected: %v", err)
	}
}

func TestProductionHAPeerAuthUsesDedicatedToken(t *testing.T) {
	a := &adminServer{token: "admin-secret", haPeerToken: "replica-secret", haPeerTokenPinned: true}
	h := a.haPeerAuth(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodPut, "/api/ha/peer-config", nil)
	req.Header.Set("Authorization", "Bearer admin-secret")
	req.Header.Set("X-WAF-HA-Sync", "v1")
	rr := httptest.NewRecorder()
	h(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("break-glass admin token unexpectedly authenticated HA replication: %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodPut, "/api/ha/peer-config", nil)
	req.Header.Set("Authorization", "Bearer replica-secret")
	rr = httptest.NewRecorder()
	h(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("HA protocol marker must be required: %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodPut, "/api/ha/peer-config", nil)
	req.Header.Set("Authorization", "Bearer replica-secret")
	req.Header.Set("X-WAF-HA-Sync", "v1")
	rr = httptest.NewRecorder()
	h(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("dedicated HA token rejected: %d", rr.Code)
	}
}

func TestProductionHASyncCoalescesLatestPendingConfig(t *testing.T) {
	var mu sync.Mutex
	modes := []string{}
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var env haSyncEnvelope
		if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
			t.Errorf("decode sync envelope: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		modes = append(modes, env.Config.EngineMode)
		idx := len(modes)
		mu.Unlock()
		if idx == 1 {
			close(firstStarted)
			<-releaseFirst
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	notify := newNotifier(log)
	h := newHAEngine(log, notify)
	h.client = server.Client()
	h.configure(HAConfig{Enabled: true, Role: "primary", PeerURL: server.URL, PeerToken: "peer-secret", SyncConfig: true})

	first := defaultConfig()
	first.EngineMode = "DetectionOnly"
	latest := first
	latest.EngineMode = "On"
	h.pushConfig(first)
	select {
	case <-firstStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("first HA sync did not start")
	}
	h.pushConfig(latest)
	close(releaseFirst)

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		count := len(modes)
		mu.Unlock()
		h.syncMu.Lock()
		running := h.syncRunning
		h.syncMu.Unlock()
		if count >= 2 && !running {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(modes) != 2 || modes[0] != "DetectionOnly" || modes[1] != "On" {
		t.Fatalf("HA coalescing did not converge to latest config: %#v", modes)
	}
}

func TestProductionCloneConfigDoesNotShareNestedSlices(t *testing.T) {
	cfg := defaultConfig()
	cfg.Sites[0].Hostnames = []string{"original.example"}
	cloned, err := cloneConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	cloned.Sites[0].Hostnames[0] = "changed.example"
	if cfg.Sites[0].Hostnames[0] != "original.example" {
		t.Fatal("cloneConfig shared nested slice storage with live config")
	}
}

func TestProductionLoginLimiterBlocksAndResets(t *testing.T) {
	l := newLoginAttemptLimiter()
	now := time.Unix(1_700_000_000, 0)
	keys := []string{"ip:192.0.2.10", "user:admin"}
	for i := 0; i < loginAttemptLimit; i++ {
		l.failure(keys, now.Add(time.Duration(i)*time.Second))
	}
	if ok, retry := l.allow(keys, now.Add(time.Duration(loginAttemptLimit)*time.Second)); ok || retry <= 0 {
		t.Fatalf("expected blocked login, ok=%v retry=%v", ok, retry)
	}
	l.success(keys)
	if ok, _ := l.allow(keys, now.Add(time.Minute)); !ok {
		t.Fatal("successful login did not reset limiter")
	}
}

func TestProductionLoginLimiterSaturationDoesNotEvictBlockedKeys(t *testing.T) {
	l := newLoginAttemptLimiter()
	now := time.Unix(1_700_000_000, 0)
	blocked := []string{"ip:192.0.2.44"}
	for i := 0; i < loginAttemptLimit; i++ {
		l.failure(blocked, now.Add(time.Duration(i)*time.Millisecond))
	}
	for i := 0; i < loginAttemptMaxKey+128; i++ {
		l.failure([]string{"user:flood-" + strconv.Itoa(i)}, now)
	}
	if ok, _ := l.allow(blocked, now.Add(time.Second)); ok {
		t.Fatal("bounded login limiter evicted a blocked identity under key-space saturation")
	}
	// Once the key map is full, unseen identities share the bounded overflow
	// bucket rather than creating unbounded state or flushing established bans.
	for i := 0; i < loginAttemptLimit; i++ {
		l.failure([]string{"user:overflow-" + strconv.Itoa(i)}, now)
	}
	if ok, _ := l.allow([]string{"user:another-unseen"}, now.Add(time.Second)); ok {
		t.Fatal("saturated login limiter failed open for a new identity")
	}
}

func TestProductionHealthJSONEncodesRole(t *testing.T) {
	rr := httptest.NewRecorder()
	writeHealth(rr, true, false, `active\"<script>`)
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("health response is not valid JSON: %v body=%q", err, rr.Body.String())
	}
	if body["role"] != `active\"<script>` {
		t.Fatalf("role round-trip mismatch: %#v", body["role"])
	}
}

func TestProductionPBKDF2RejectsAbsurdIterationCount(t *testing.T) {
	encoded := "pbkdf2$sha256$1000001$c2FsdHNhbHQ=$AAAAAAAAAAAAAAAAAAAAAA=="
	if verifyPassword("password", encoded) {
		t.Fatal("absurd PBKDF2 iteration count must be rejected")
	}
}

func TestProductionDirectUpdateDisabledWithoutExplicitDirectory(t *testing.T) {
	t.Setenv("WAF_UPDATE_INSTALL_DIR", "")
	if directUpdateInstallEnabled() {
		t.Fatal("direct update install must be disabled by default")
	}
	t.Setenv("WAF_UPDATE_INSTALL_DIR", "relative/path")
	if directUpdateInstallEnabled() {
		t.Fatal("relative update install directory must not enable direct install")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("WAF_UPDATE_INSTALL_DIR", filepath.Dir(exe))
	if !directUpdateInstallEnabled() {
		t.Fatal("current executable directory should enable explicit standalone install")
	}
	t.Setenv("WAF_UPDATE_INSTALL_DIR", t.TempDir())
	if directUpdateInstallEnabled() {
		t.Fatal("a different directory would install bytes that ReExec cannot activate")
	}
}

func TestProductionUpdateLocalhostRejectsForwardedRequests(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "http://localhost/api/update/install", nil)
	r.RemoteAddr = "127.0.0.1:12345"
	if !updateLocalhost(r) {
		t.Fatal("direct loopback request should be accepted")
	}
	r.Header.Set("X-Forwarded-For", "203.0.113.7")
	if updateLocalhost(r) {
		t.Fatal("forwarded loopback request must not satisfy localhost-only guard")
	}
}

func TestProductionL7SaturationUsesBoundedOverflowBucket(t *testing.T) {
	c := newL7AbuseController(L7AbuseConfig{Enabled: true, RequestsPerWindow: 1, WindowSeconds: 60})
	now := time.Unix(1_700_000_000, 0)
	const targetShard = 0
	filled := 0
	for i := 0; filled < l7AbuseMaxEntriesPerShard; i++ {
		identity := "198.51.100." + strconv.Itoa(i)
		key := abuseClientKey{site: "site", identity: identity}
		if l7AbuseShardIndex(key) != targetShard {
			continue
		}
		state, ok := c.allowAt("site", identity, now)
		if !ok || state == nil {
			t.Fatalf("failed filling bounded shard at %d", filled)
		}
		c.releaseState(state)
		filled++
	}
	var firstOverflow, secondOverflow string
	for i := 100000; secondOverflow == ""; i++ {
		identity := "203.0.113." + strconv.Itoa(i)
		key := abuseClientKey{site: "site", identity: identity}
		if l7AbuseShardIndex(key) != targetShard {
			continue
		}
		if firstOverflow == "" {
			firstOverflow = identity
		} else {
			secondOverflow = identity
		}
	}
	state, ok := c.allowAt("site", firstOverflow, now)
	if !ok || state == nil {
		t.Fatal("first overflow admission should consume the shared bounded token")
	}
	c.releaseState(state)
	if state, ok = c.allowAt("site", secondOverflow, now); ok || state != nil {
		t.Fatal("saturated unknown identity unexpectedly failed open")
	}
}

func TestProductionStageConfigUsesPrivateAtomicFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := defaultConfig()
	staged, err := stageConfig(path, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer staged.abort()
	if staged.tmp == path || !strings.HasPrefix(filepath.Base(staged.tmp), ".config.json.tmp-") {
		t.Fatalf("unexpected staging file %q", staged.tmp)
	}
	if info, err := os.Stat(staged.tmp); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("staged config mode/stat: %v %v", info, err)
	}
	committed, err := staged.commit()
	if err != nil || !committed {
		t.Fatalf("commit: committed=%v err=%v", committed, err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("committed config missing: %v", err)
	}
}

func TestProductionAPISecurityStateVersionFailsClosed(t *testing.T) {
	if err := validateAPISecurityStateVersion("test", 0); err != nil {
		t.Fatalf("legacy v0 must remain readable: %v", err)
	}
	if err := validateAPISecurityStateVersion("test", apiSecurityStateVersion); err != nil {
		t.Fatalf("current state version rejected: %v", err)
	}
	if err := validateAPISecurityStateVersion("test", apiSecurityStateVersion+1); err == nil {
		t.Fatal("future state version must fail closed")
	}
}

func TestProductionSitemapPersistenceUsesUniqueDurableTemp(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	maps := newSiteMaps(128)
	maps.record("site-a", http.MethodGet, "/products/42", http.StatusOK, srcObserved)

	var wg sync.WaitGroup
	errCh := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errCh <- maps.save(configPath)
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent sitemap save failed: %v", err)
		}
	}

	b, err := os.ReadFile(filepath.Join(dir, "sitemap.json"))
	if err != nil {
		t.Fatal(err)
	}
	var state siteMapsFile
	if err := json.Unmarshal(b, &state); err != nil {
		t.Fatalf("persisted sitemap is not valid JSON: %v", err)
	}
	if len(state.Sites) != 1 || state.Sites[0].Site != "site-a" {
		t.Fatalf("unexpected sitemap state: %+v", state.Sites)
	}
	leftovers, err := filepath.Glob(filepath.Join(dir, ".sitemap.json.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(leftovers) != 0 {
		t.Fatalf("temporary sitemap files leaked: %v", leftovers)
	}
}

func TestProductionSitemapRestoreSnapshotAfterFailedClearPath(t *testing.T) {
	maps := newSiteMaps(128)
	maps.record("site-a", http.MethodGet, "/orders/123", http.StatusOK, srcObserved)
	before := maps.snapshot("site-a")
	maps.clear("site-a")
	maps.restoreSnapshot(before)
	after := maps.snapshot("site-a")
	if after.Site != before.Site || after.Nodes != before.Nodes {
		t.Fatalf("restored sitemap metadata differs: before=%+v after=%+v", before, after)
	}
	beforeJSON, err := json.Marshal(before.Tree)
	if err != nil {
		t.Fatal(err)
	}
	afterJSON, err := json.Marshal(after.Tree)
	if err != nil {
		t.Fatal(err)
	}
	if string(beforeJSON) != string(afterJSON) {
		t.Fatalf("restored sitemap tree differs: before=%s after=%s", beforeJSON, afterJSON)
	}
}

func TestProductionDraftConfigIsSeparateFromStartupAuthority(t *testing.T) {
	dir := t.TempDir()
	livePath := filepath.Join(dir, "config.json")
	live := defaultConfig()
	live.EngineMode = "DetectionOnly"
	if err := saveConfig(livePath, live); err != nil {
		t.Fatal(err)
	}

	draft := live
	draft.EngineMode = "On"
	if err := saveConfig(draftConfigPath(livePath), draft); err != nil {
		t.Fatal(err)
	}
	// Ensure the draft is observably newer even on coarse timestamp filesystems.
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(draftConfigPath(livePath), future, future); err != nil {
		t.Fatal(err)
	}

	startup, err := loadConfig(livePath)
	if err != nil {
		t.Fatal(err)
	}
	if startup.EngineMode != "DetectionOnly" {
		t.Fatalf("startup authority unexpectedly read draft: %q", startup.EngineMode)
	}
	resumed, ok, err := loadDraftConfig(livePath)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || resumed.EngineMode != "On" {
		t.Fatalf("newer draft was not resumable: ok=%t mode=%q", ok, resumed.EngineMode)
	}
}

func TestProductionStaleDraftIgnoredAfterNewerLiveCommit(t *testing.T) {
	dir := t.TempDir()
	livePath := filepath.Join(dir, "config.json")
	base := defaultConfig()
	if err := saveConfig(livePath, base); err != nil {
		t.Fatal(err)
	}
	draft := base
	draft.EngineMode = "DetectionOnly"
	if err := saveConfig(draftConfigPath(livePath), draft); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-2 * time.Second)
	if err := os.Chtimes(draftConfigPath(livePath), past, past); err != nil {
		t.Fatal(err)
	}
	live := base
	live.EngineMode = "On"
	if err := saveConfig(livePath, live); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := loadDraftConfig(livePath); err != nil || ok {
		t.Fatalf("stale draft should be ignored after newer live commit: ok=%t err=%v", ok, err)
	}
}

func TestProductionNotificationDedupeStateIsBounded(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	n := newNotifier(log)
	for i := 0; i < notificationDedupeMax+1024; i++ {
		n.push(notifySuggestion, "info", "test", "body", "key-"+strconv.Itoa(i), "", nil)
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if len(n.dedupe) > notificationDedupeMax {
		t.Fatalf("notification dedupe state exceeded cap: %d", len(n.dedupe))
	}
	if len(n.items) != n.cap {
		t.Fatalf("notification ring cap not preserved: got %d want %d", len(n.items), n.cap)
	}
}
