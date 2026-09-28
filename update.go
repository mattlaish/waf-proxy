package main

// Signed self-update for waf-proxy, built on the shared sigupdate engine
// (internal/sigupdate — copied unchanged, never forked).
//
// Trust model: the publisher signs update packages with an RSA private key held
// OFFLINE; this binary embeds only the PUBLIC key and verifies. A stolen admin
// session cannot forge an update — without the private key, nothing installs.
// If no publisher key is baked in, the whole subsystem is disabled (fail-safe),
// and every endpoint 404s.
//
// Guards (kept strict, per the standard): admin role + localhost-only + a
// feature flag. Applying an update is an admin-only action and always requires
// a restart for a compiled binary; the operator triggers the restart explicitly.

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"waf-proxy/internal/sigupdate"
)

const wafUpdateExtension = ".wafupdate"

// PublisherKeyPEM is the baked-in publisher PUBLIC key (PEM). Empty ⇒ updates
// disabled. Set this before `go build` to enable signed updates. It can also be
// supplied at runtime via WAF_PUBLISHER_KEY_FILE (a path to the public-key PEM)
// so operators can enable updates without a rebuild.
var PublisherKeyPEM = ``

// CatalogURL is the baked-in online-catalog base (e.g.
// "https://updates.example.com/waf/"). Empty ⇒ upload-only. Overridable at
// runtime via WAF_UPDATE_CATALOG_URL.
var CatalogURL = ``

func loadPublisherKey() string {
	if PublisherKeyPEM != "" {
		return PublisherKeyPEM
	}
	if p := os.Getenv("WAF_PUBLISHER_KEY_FILE"); p != "" {
		if b, err := os.ReadFile(p); err == nil {
			return string(b)
		}
	}
	return ""
}

func catalogURL() string {
	if v := os.Getenv("WAF_UPDATE_CATALOG_URL"); v != "" {
		return v
	}
	return CatalogURL
}

// updateInstallDir is used for package inspection. Direct in-process writes
// are disabled by default for systemd/DEB/RPM deployments; operators must use
// the package manager or explicitly opt a standalone/dev install into a
// dedicated writable directory with WAF_UPDATE_INSTALL_DIR.
func updateInstallDir() string {
	if dir := strings.TrimSpace(os.Getenv("WAF_UPDATE_INSTALL_DIR")); dir != "" && filepath.IsAbs(dir) {
		return filepath.Clean(dir)
	}
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

func directUpdateInstallEnabled() bool {
	dir := strings.TrimSpace(os.Getenv("WAF_UPDATE_INSTALL_DIR"))
	if dir == "" || !filepath.IsAbs(dir) {
		return false
	}
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	// ReExec restarts the current executable path. Treat a different writable
	// directory as unsupported rather than reporting a successful install that
	// would restart into the old binary.
	return filepath.Clean(filepath.Dir(exe)) == filepath.Clean(dir)
}

func (a *adminServer) updateConfig() *sigupdate.Config {
	dir := updateInstallDir()
	return &sigupdate.Config{
		PublisherKeyPEM: loadPublisherKey(),
		CatalogURL:      catalogURL(),
		InstallDir:      dir,
		BackupDir:       filepath.Join(dir, ".wafupdate-backup"),
		// waf-proxy ships as a single binary plus optional rules/config assets.
		AllowedExts:  map[string]bool{".json": true, ".conf": true, ".html": true},
		AllowedNames: map[string]bool{"waf-proxy": true}, // the binary itself
	}
}

// staged package (single-process state).
type stagedUpdate struct {
	mu       sync.Mutex
	manifest *sigupdate.Manifest
	payload  map[string][]byte
}

func (a *adminServer) registerUpdateRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/update/status", a.guardedUpdate(a.handleUpdateStatus))
	mux.HandleFunc("GET /api/update/catalog", a.guardedUpdate(a.handleUpdateCatalog))
	mux.HandleFunc("POST /api/update/stage", a.guardedUpdate(a.handleUpdateStage))
	mux.HandleFunc("POST /api/update/download", a.guardedUpdate(a.handleUpdateDownload))
	mux.HandleFunc("POST /api/update/install", a.guardedUpdate(a.handleUpdateInstall))
	mux.HandleFunc("POST /api/update/discard", a.guardedUpdate(a.handleUpdateDiscard))
	mux.HandleFunc("POST /api/update/rollback", a.guardedUpdate(a.handleUpdateRollback))
	mux.HandleFunc("POST /api/update/restart", a.guardedUpdate(a.handleUpdateRestart))
}

// guardedUpdate: feature-enabled (404 when no key) + localhost-only + admin.
// It resolves identity via the same auth path as the rest of the admin API, so
// the caller must present the master token or an admin session.
func (a *adminServer) guardedUpdate(h http.HandlerFunc) http.HandlerFunc {
	inner := a.authRole(roleAdmin, func(w http.ResponseWriter, r *http.Request) {
		if !updateLocalhost(r) {
			writeJSONCode(w, http.StatusForbidden, map[string]string{"error": "updates are localhost-only; tunnel with ssh -L"})
			return
		}
		h(w, r)
	})
	return func(w http.ResponseWriter, r *http.Request) {
		// Feature flag: 404 when no publisher key is present at all.
		if loadPublisherKey() == "" {
			http.NotFound(w, r)
			return
		}
		inner(w, r)
	}
}

func (a *adminServer) handleUpdateStatus(w http.ResponseWriter, _ *http.Request) {
	cfg := a.updateConfig()
	fp, _ := sigupdate.KeyFingerprint(cfg.PublisherKeyPEM)
	a.staged.mu.Lock()
	var staged any
	if a.staged.manifest != nil {
		staged = map[string]any{"version": a.staged.manifest.Version, "notes": a.staged.manifest.Notes}
	}
	a.staged.mu.Unlock()
	writeJSON(w, map[string]any{
		"enabled":           cfg.Enabled(),
		"install_supported": directUpdateInstallEnabled(),
		"install_mode":      map[bool]string{true: "explicit_standalone_directory", false: "external_package_manager"}[directUpdateInstallEnabled()],
		"key_fingerprint":   fp,
		"catalog_url":       cfg.CatalogURL,
		"extension":         wafUpdateExtension,
		"has_backup":        cfg.HasBackup(),
		"staged":            staged,
		"current_version":   buildVersion,
	})
}

func (a *adminServer) handleUpdateCatalog(w http.ResponseWriter, _ *http.Request) {
	cfg := a.updateConfig()
	if cfg.CatalogURL == "" {
		writeJSON(w, map[string]any{"enabled": false, "reason": "no catalog URL in this build"})
		return
	}
	cat, err := cfg.FetchCatalog()
	if err != nil {
		writeJSON(w, map[string]any{"enabled": true, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]any{"enabled": true, "packages": cat.Packages})
}

func (a *adminServer) handleUpdateStage(w http.ResponseWriter, r *http.Request) {
	cfg := a.updateConfig()
	cfg.MaxPackageBytes = 200 << 20
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, cfg.MaxPackageBytes+1))
	if err != nil {
		writeJSONCode(w, http.StatusBadRequest, map[string]string{"error": "read failed"})
		return
	}
	m, payloads, err := cfg.Inspect(data)
	if err != nil {
		a.audit.add(who(r).user, "update.stage_rejected", err.Error())
		writeJSONCode(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	a.stageUpdate(m, payloads)
	a.audit.add(who(r).user, "update.staged", m.Version)
	writeJSON(w, a.stagedSummary(cfg, m))
}

func (a *adminServer) handleUpdateDownload(w http.ResponseWriter, r *http.Request) {
	cfg := a.updateConfig()
	var body struct {
		Filename string `json:"filename"`
	}
	_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&body)
	m, payloads, err := cfg.Download(body.Filename, wafUpdateExtension)
	if err != nil {
		a.audit.add(who(r).user, "update.download_rejected", err.Error())
		writeJSONCode(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	a.stageUpdate(m, payloads)
	a.audit.add(who(r).user, "update.staged", m.Version)
	writeJSON(w, a.stagedSummary(cfg, m))
}

func (a *adminServer) handleUpdateInstall(w http.ResponseWriter, r *http.Request) {
	if !directUpdateInstallEnabled() {
		writeJSONCode(w, http.StatusConflict, map[string]string{"error": "direct in-process install is disabled for packaged/systemd deployments; use the OS package manager or set WAF_UPDATE_INSTALL_DIR for a standalone writable install"})
		return
	}
	cfg := a.updateConfig()
	a.staged.mu.Lock()
	m, payloads := a.staged.manifest, a.staged.payload
	a.staged.mu.Unlock()
	if m == nil {
		writeJSONCode(w, http.StatusConflict, map[string]string{"error": "nothing staged"})
		return
	}
	applied, err := cfg.Apply(payloads)
	if err != nil {
		a.audit.add(who(r).user, "update.install_failed", err.Error())
		writeJSONCode(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	restart := cfg.NeedsRestart(m, applied)
	a.audit.add(who(r).user, "update.installed", m.Version)
	a.clearStaged()
	writeJSON(w, map[string]any{"ok": true, "version": m.Version, "restart_required": restart})
}

func (a *adminServer) handleUpdateDiscard(w http.ResponseWriter, _ *http.Request) {
	a.clearStaged()
	writeJSON(w, map[string]bool{"ok": true})
}

func (a *adminServer) handleUpdateRollback(w http.ResponseWriter, r *http.Request) {
	if !directUpdateInstallEnabled() {
		writeJSONCode(w, http.StatusConflict, map[string]string{"error": "direct rollback is disabled for packaged/systemd deployments; use the OS package manager"})
		return
	}
	cfg := a.updateConfig()
	restored, err := cfg.Rollback()
	if err != nil {
		writeJSONCode(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	a.audit.add(who(r).user, "update.rolledback", "")
	writeJSON(w, map[string]any{"ok": true, "restored": restored, "restart_required": true})
}

func (a *adminServer) handleUpdateRestart(w http.ResponseWriter, r *http.Request) {
	a.audit.add(who(r).user, "update.restart", "")
	writeJSON(w, map[string]string{"message": "restarting; reconnect shortly"})
	// Drain first so upstreams fail over, then re-exec into the new binary.
	go func() {
		a.srv.draining.Store(true)
		_ = sigupdate.ReExec()
	}()
}

// ── staged-package helpers ──

func (a *adminServer) stageUpdate(m *sigupdate.Manifest, p map[string][]byte) {
	a.staged.mu.Lock()
	a.staged.manifest, a.staged.payload = m, p
	a.staged.mu.Unlock()
}

func (a *adminServer) clearStaged() {
	a.staged.mu.Lock()
	a.staged.manifest, a.staged.payload = nil, nil
	a.staged.mu.Unlock()
}

func (a *adminServer) stagedSummary(cfg *sigupdate.Config, m *sigupdate.Manifest) map[string]any {
	var files []map[string]string
	for _, f := range m.Files {
		action := "add"
		if _, err := os.Stat(filepath.Join(cfg.InstallDir, filepath.FromSlash(f.Path))); err == nil {
			action = "replace"
		}
		files = append(files, map[string]string{"path": f.Path, "action": action})
	}
	return map[string]any{
		"ok": true, "version": m.Version, "notes": m.Notes,
		"files": files, "needs_restart": cfg.NeedsRestart(m, nil),
	}
}

func updateLocalhost(r *http.Request) bool {
	// A local reverse proxy makes RemoteAddr loopback even for remote callers.
	// Refuse forwarded requests so localhost-only cannot be silently weakened.
	if r.Header.Get("Forwarded") != "" || r.Header.Get("X-Forwarded-For") != "" || r.Header.Get("X-Real-IP") != "" {
		return false
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// writeJSONCode is writeJSON with an explicit status code.
func writeJSONCode(w http.ResponseWriter, code int, v any) {
	setJSONSecurityHeaders(w)
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
