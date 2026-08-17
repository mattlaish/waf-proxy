package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "embed"
)

//go:embed static/admin.html
var adminHTML []byte

// ── rule-match ring buffer ──────────────────────────────────────────────

type matchRec struct {
	Time     string `json:"time"`
	Site     string `json:"site"`
	RuleID   int    `json:"rule_id"`
	Severity string `json:"severity"`
	Phase    int    `json:"phase"`
	Client   string `json:"client"`
	URI      string `json:"uri"`
	Msg      string `json:"msg"`
	Data     string `json:"data"`
}

type matchRing struct {
	mu   sync.Mutex
	recs []matchRec
	cap  int
}

func newMatchRing(capacity int) *matchRing { return &matchRing{cap: capacity} }

func (r *matchRing) add(m matchRec) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.recs = append(r.recs, m)
	if len(r.recs) > r.cap {
		r.recs = r.recs[len(r.recs)-r.cap:]
	}
}

// snapshot returns up to limit records, newest first.
func (r *matchRing) snapshot(limit int) []matchRec {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := len(r.recs)
	if limit <= 0 || limit > n {
		limit = n
	}
	out := make([]matchRec, limit)
	for i := 0; i < limit; i++ {
		out[i] = r.recs[n-1-i]
	}
	return out
}

func (r *matchRing) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.recs)
}

// accessRec is one line of the request/access log.
type accessRec struct {
	Time   string `json:"time"`
	Site   string `json:"site"`
	Client string `json:"client"`
	Method string `json:"method"`
	Path   string `json:"path"`
	Status int    `json:"status"`
}

type accessRing struct {
	mu   sync.Mutex
	recs []accessRec
	cap  int
}

func newAccessRing(capacity int) *accessRing { return &accessRing{cap: capacity} }

func (r *accessRing) add(m accessRec) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.recs = append(r.recs, m)
	if len(r.recs) > r.cap {
		r.recs = r.recs[len(r.recs)-r.cap:]
	}
}

func (r *accessRing) snapshot(limit int) []accessRec {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := len(r.recs)
	if limit <= 0 || limit > n {
		limit = n
	}
	out := make([]accessRec, limit)
	for i := 0; i < limit; i++ {
		out[i] = r.recs[n-1-i]
	}
	return out
}

// ── admin server ────────────────────────────────────────────────────────

type adminServer struct {
	srv      *server
	token    string
	sessions *sessionStore
	audit    *auditLog
	staged   stagedUpdate
	log      *slog.Logger
	started  time.Time
}

func newAdminServer(s *server, token string, log *slog.Logger) *adminServer {
	if token == "" {
		b := make([]byte, 24)
		if _, err := rand.Read(b); err != nil {
			log.Error("could not generate admin token", "err", err)
			os.Exit(1)
		}
		token = hex.EncodeToString(b)
		// Printed once at startup; not logged again.
		fmt.Fprintf(os.Stderr, "\n  admin token: %s\n  (set WAF_ADMIN_TOKEN or -admin-token to pin one)\n\n", token)
	}
	as := &adminServer{srv: s, token: token, sessions: newSessionStore(), audit: newAuditLog(), log: log, started: time.Now()}
	as.audit.sink = s.syslog.forwardAudit // fan audit entries out to syslog
	return as
}

type identity struct {
	user string
	role string
}

type identityCtxKey int

const identityKey identityCtxKey = 1

// auth resolves the caller's identity: the break-glass master token grants
// admin; otherwise a valid session token is required. Identity is stashed in
// the request context for handlers and the audit log.
func (a *adminServer) auth(next http.HandlerFunc) http.HandlerFunc {
	return a.authRole("", next)
}

// authRole is auth plus a minimum-role requirement ("" = any authenticated).
func (a *adminServer) authRole(minRole string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		var id identity
		switch {
		case got != "" && subtle.ConstantTimeCompare([]byte(got), []byte(a.token)) == 1:
			id = identity{user: "(token)", role: roleAdmin}
		default:
			if sess, ok := a.sessions.lookup(got); ok {
				id = identity{user: sess.user, role: sess.role}
			} else {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		if minRole != "" && roleRank(id.role) < roleRank(minRole) {
			http.Error(w, "forbidden: requires "+minRole, http.StatusForbidden)
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), identityKey, id)))
	}
}

func who(r *http.Request) identity {
	if id, ok := r.Context().Value(identityKey).(identity); ok {
		return id
	}
	return identity{user: "(unknown)", role: ""}
}

func (a *adminServer) handler() http.Handler {
	mux := http.NewServeMux()

	// The page shell contains no secrets; every API behind it requires the token.
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy",
			"default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; connect-src 'self'")
		_, _ = w.Write(adminHTML)
	})

	mux.HandleFunc("GET /api/status", a.auth(a.handleStatus))
	mux.HandleFunc("GET /api/config", a.auth(a.handleGetConfig))
	mux.HandleFunc("GET /api/interfaces", a.auth(a.handleInterfaces))
	mux.HandleFunc("PUT /api/config", a.authRole(roleOperator, a.handlePutConfig))
	mux.HandleFunc("POST /api/reload", a.auth(a.handleReload))
	mux.HandleFunc("GET /api/matches", a.auth(a.handleMatches))
	mux.HandleFunc("GET /api/access", a.auth(a.handleAccess))
	mux.HandleFunc("GET /api/pools", a.auth(a.handlePools))
	mux.HandleFunc("GET /api/ai/verdicts", a.auth(a.handleAIVerdicts))
	mux.HandleFunc("GET /api/ai/blocklist", a.auth(a.handleAIBlocklist))
	mux.HandleFunc("POST /api/ai/unblock", a.auth(a.handleAIUnblock))
	mux.HandleFunc("POST /api/ai/test", a.auth(a.handleAITest))
	mux.HandleFunc("POST /api/syslog/test", a.authRole(roleOperator, a.handleSyslogTest))
	mux.HandleFunc("GET /api/learn", a.auth(a.handleLearn))
	mux.HandleFunc("POST /api/learn/apply", a.authRole(roleReviewer, a.handleLearnApply))
	mux.HandleFunc("POST /api/learn/clear", a.auth(a.handleLearnClear))
	mux.HandleFunc("GET /api/notifications", a.auth(a.handleNotifications))
	mux.HandleFunc("POST /api/notifications/read", a.auth(a.handleNotifyRead))
	mux.HandleFunc("POST /api/notifications/dismiss", a.auth(a.handleNotifyDismiss))
	mux.HandleFunc("POST /api/notifications/apply", a.authRole(roleReviewer, a.handleNotifyApply))
	mux.HandleFunc("POST /api/login", a.handleLogin) // unauthenticated
	mux.HandleFunc("POST /api/logout", a.auth(a.handleLogout))
	mux.HandleFunc("GET /api/whoami", a.auth(a.handleWhoami))
	mux.HandleFunc("GET /api/users", a.authRole(roleAdmin, a.handleUsers))
	mux.HandleFunc("POST /api/users/create", a.authRole(roleAdmin, a.handleUserCreate))
	mux.HandleFunc("POST /api/users/update", a.authRole(roleAdmin, a.handleUserUpdate))
	mux.HandleFunc("POST /api/users/password", a.auth(a.handleUserPassword)) // self or admin (checked inside)
	mux.HandleFunc("POST /api/users/delete", a.authRole(roleAdmin, a.handleUserDelete))
	mux.HandleFunc("GET /api/audit", a.auth(a.handleAudit))
	mux.HandleFunc("GET /api/ha", a.auth(a.handleHA))
	mux.HandleFunc("POST /api/ha/sync", a.authRole(roleOperator, a.handleHASync))
	mux.HandleFunc("GET /api/pagepolicy", a.auth(a.handlePagePolicies))
	mux.HandleFunc("GET /api/forms", a.auth(a.handleDiscoveredForms))
	mux.HandleFunc("POST /api/pagepolicy/upsert", a.authRole(roleReviewer, a.handlePagePolicyUpsert))
	mux.HandleFunc("POST /api/pagepolicy/delete", a.authRole(roleReviewer, a.handlePagePolicyDelete))
	mux.HandleFunc("GET /api/profiles", a.auth(a.handleProfiles))
	mux.HandleFunc("GET /api/profiles/suggest", a.auth(a.handleProfileSuggest))
	mux.HandleFunc("POST /api/profiles/apply", a.authRole(roleReviewer, a.handleProfileApply))
	mux.HandleFunc("POST /api/profiles/auto", a.authRole(roleReviewer, a.handleProfileAuto))
	mux.HandleFunc("GET /api/fs", a.auth(a.handleFS))
	mux.HandleFunc("GET /api/sitemap", a.auth(a.handleSitemap))
	mux.HandleFunc("POST /api/crawl", a.auth(a.handleCrawl))
	mux.HandleFunc("POST /api/sitemap/clear", a.auth(a.handleSitemapClear))
	mux.HandleFunc("GET /api/discovered", a.auth(a.handleDiscovered))
	mux.HandleFunc("GET /api/metrics", a.auth(a.handleMetrics))
	a.registerUpdateRoutes(mux) // signed self-update (localhost + admin, 404 when no key)
	return mux
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (a *adminServer) handleStatus(w http.ResponseWriter, _ *http.Request) {
	rt := a.srv.rt.Load()
	sites := make([]map[string]any, 0, len(rt.cfg.Sites))
	for _, sc := range rt.cfg.Sites {
		mode := sc.EngineMode
		if mode == "" {
			mode = rt.cfg.EngineMode
		}
		sites = append(sites, map[string]any{
			"name":      sc.Name,
			"listen":    sc.Listen,
			"hostnames": sc.Hostnames,
			"pool":      sc.Pool,
			"engine":    mode,
			"tls":       sc.TLSCert != "",
		})
	}
	anyTLS := false
	for _, t := range listenerSet(rt.cfg) {
		anyTLS = anyTLS || t
	}
	writeJSON(w, map[string]any{
		"uptime":          time.Since(a.started).Truncate(time.Second).String(),
		"engine_mode":     rt.cfg.EngineMode,
		"listeners":       sortedListenAddrs(rt.cfg),
		"sites":           sites,
		"pools":           len(rt.cfg.Pools),
		"policies":        len(rt.cfg.Policies),
		"nodes":           len(rt.cfg.Nodes),
		"tls":             anyTLS,
		"match_count":     a.srv.matches.count(),
		"restart_pending": a.srv.restartPending(rt.cfg),
		"rules_built_at":  rt.builtAt.Format(time.RFC3339),
		"ai_enabled":      rt.cfg.AI.Enabled,
		"ai_key_set":      rt.cfg.AI.APIKey != "",
		"version":         buildVersion,
		"commit":          buildCommit,
		"ha_enabled":      rt.cfg.HA.Enabled,
		"ha":              a.srv.ha.status(),
		"ha_token_set":    rt.cfg.HA.PeerToken != "",
		"notify_unread":   a.srv.notify.unreadCount(),
	})
}

// ── auth + user management ──────────────────────────────────────────────

// handleLogin authenticates a named user and returns a session token. The
// break-glass startup token is not a "user" and doesn't log in here — it's
// used directly as the bearer.
func (a *adminServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	cfg := a.srv.rt.Load().cfg
	for _, u := range cfg.Users {
		if strings.EqualFold(u.Username, req.Username) {
			if u.Disabled || !verifyPassword(req.Password, u.PasswordHash) {
				break
			}
			tok := a.sessions.create(u.Username, u.Role)
			a.audit.add(u.Username, "login", "")
			writeJSON(w, map[string]any{"token": tok, "user": u.Username, "role": u.Role})
			return
		}
	}
	// constant-ish: run a verify against a dummy to blunt user enumeration timing
	verifyPassword(req.Password, "pbkdf2$sha256$210000$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
	http.Error(w, "invalid credentials", http.StatusUnauthorized)
}

func (a *adminServer) handleLogout(w http.ResponseWriter, r *http.Request) {
	got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	a.sessions.destroy(got)
	writeJSON(w, map[string]any{"ok": true})
}

func (a *adminServer) handleWhoami(w http.ResponseWriter, r *http.Request) {
	id := who(r)
	writeJSON(w, map[string]any{
		"user": id.user, "role": id.role,
		"can_manage_users": canManageUsers(id.role),
		"can_edit_config":  canEditConfig(id.role),
		"can_review":       canReview(id.role),
	})
}

// handleUsers lists accounts (no hashes).
func (a *adminServer) handleUsers(w http.ResponseWriter, _ *http.Request) {
	cfg := a.srv.rt.Load().cfg
	out := make([]map[string]any, 0, len(cfg.Users))
	for _, u := range cfg.Users {
		out = append(out, map[string]any{"username": u.Username, "role": u.Role, "disabled": u.Disabled})
	}
	writeJSON(w, out)
}

// mutateUsers applies a mutation to a copy of the user list, then apply+save.
func (a *adminServer) mutateUsers(fn func(users []UserConfig) ([]UserConfig, error)) error {
	cfg := a.srv.rt.Load().cfg
	next, err := fn(append([]UserConfig(nil), cfg.Users...))
	if err != nil {
		return err
	}
	cfg.Users = next
	if err := a.srv.apply(cfg); err != nil {
		return fmt.Errorf("apply failed: %w", err)
	}
	return saveConfig(a.srv.configPath, cfg)
}

func (a *adminServer) handleUserCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username, Password, Role string
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || !validRole(req.Role) {
		http.Error(w, "username and valid role required", http.StatusBadRequest)
		return
	}
	hash, err := hashPassword(req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = a.mutateUsers(func(users []UserConfig) ([]UserConfig, error) {
		for _, u := range users {
			if strings.EqualFold(u.Username, req.Username) {
				return nil, fmt.Errorf("user already exists")
			}
		}
		return append(users, UserConfig{Username: req.Username, PasswordHash: hash, Role: req.Role}), nil
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	a.audit.add(who(r).user, "user.create", req.Username+" ("+req.Role+")")
	writeJSON(w, map[string]any{"ok": true})
}

func (a *adminServer) handleUserUpdate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Role     string `json:"role"`
		Disabled *bool  `json:"disabled"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	err := a.mutateUsers(func(users []UserConfig) ([]UserConfig, error) {
		for i := range users {
			if strings.EqualFold(users[i].Username, req.Username) {
				if req.Role != "" {
					if !validRole(req.Role) {
						return nil, fmt.Errorf("invalid role")
					}
					users[i].Role = req.Role
				}
				if req.Disabled != nil {
					users[i].Disabled = *req.Disabled
				}
				return users, nil
			}
		}
		return nil, fmt.Errorf("no such user")
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	a.sessions.revoke(req.Username) // force re-login under new role/disabled state
	a.audit.add(who(r).user, "user.update", req.Username)
	writeJSON(w, map[string]any{"ok": true})
}

// handleUserPassword resets a password. Admins can reset anyone; a user may
// change their own.
func (a *adminServer) handleUserPassword(w http.ResponseWriter, r *http.Request) {
	var req struct{ Username, Password string }
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	id := who(r)
	if !canManageUsers(id.role) && !strings.EqualFold(id.user, req.Username) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	hash, err := hashPassword(req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = a.mutateUsers(func(users []UserConfig) ([]UserConfig, error) {
		for i := range users {
			if strings.EqualFold(users[i].Username, req.Username) {
				users[i].PasswordHash = hash
				return users, nil
			}
		}
		return nil, fmt.Errorf("no such user")
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	a.audit.add(id.user, "user.password", req.Username)
	writeJSON(w, map[string]any{"ok": true})
}

func (a *adminServer) handleUserDelete(w http.ResponseWriter, r *http.Request) {
	var req struct{ Username string }
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	err := a.mutateUsers(func(users []UserConfig) ([]UserConfig, error) {
		out := users[:0]
		found := false
		for _, u := range users {
			if strings.EqualFold(u.Username, req.Username) {
				found = true
				continue
			}
			out = append(out, u)
		}
		if !found {
			return nil, fmt.Errorf("no such user")
		}
		return out, nil
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	a.sessions.revoke(req.Username)
	a.audit.add(who(r).user, "user.delete", req.Username)
	writeJSON(w, map[string]any{"ok": true})
}

func (a *adminServer) handleAudit(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 200
	}
	writeJSON(w, a.audit.list(limit))
}

// handleLogin ends the auth block.

// handleNotifications lists recent notifications, newest first.
func (a *adminServer) handleNotifications(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 100
	}
	writeJSON(w, map[string]any{
		"unread": a.srv.notify.unreadCount(),
		"items":  a.srv.notify.list(limit),
	})
}

func decodeIDReq(w http.ResponseWriter, r *http.Request) (int64, bool, bool) {
	var req struct {
		ID  int64 `json:"id"`
		All bool  `json:"all"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return 0, false, false
	}
	return req.ID, req.All, true
}

func (a *adminServer) handleNotifyRead(w http.ResponseWriter, r *http.Request) {
	id, all, ok := decodeIDReq(w, r)
	if !ok {
		return
	}
	a.srv.notify.markRead(id, all)
	writeJSON(w, map[string]any{"ok": true})
}

func (a *adminServer) handleNotifyDismiss(w http.ResponseWriter, r *http.Request) {
	id, all, ok := decodeIDReq(w, r)
	if !ok {
		return
	}
	a.srv.notify.dismiss(id, all)
	writeJSON(w, map[string]any{"ok": true})
}

// handleNotifyApply executes a notification's attached action (currently only
// apply_exclusion, which writes a learned exclusion into the site's policy).
func (a *adminServer) handleNotifyApply(w http.ResponseWriter, r *http.Request) {
	id, _, ok := decodeIDReq(w, r)
	if !ok {
		return
	}
	var target *notification
	for _, it := range a.srv.notify.list(a.srv.notify.cap) {
		if it.ID == id {
			t := it
			target = &t
			break
		}
	}
	if target == nil {
		http.Error(w, "notification not found", http.StatusNotFound)
		return
	}
	if target.Action != "apply_exclusion" {
		http.Error(w, "notification has no applicable action", http.StatusBadRequest)
		return
	}
	site, _ := target.Payload["site"].(string)
	path, _ := target.Payload["path"].(string)
	ids := toIntSlice(target.Payload["rule_ids"])
	if site == "" || len(ids) == 0 {
		http.Error(w, "malformed action payload", http.StatusBadRequest)
		return
	}
	if err := a.applyExclusion(site, path, ids, "notification apply"); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	a.srv.notify.dismiss(id, false)
	writeJSON(w, map[string]any{"ok": true})
}

func toIntSlice(v any) []int {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]int, 0, len(arr))
	for _, e := range arr {
		if f, ok := e.(float64); ok {
			out = append(out, int(f))
		}
	}
	return out
}

func (a *adminServer) handleHA(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, a.srv.ha.status())
}

func (a *adminServer) handleHASync(w http.ResponseWriter, _ *http.Request) {
	if !a.srv.ha.snapshotCfg().Enabled {
		http.Error(w, "HA is disabled", http.StatusBadRequest)
		return
	}
	a.srv.ha.pushConfig(a.srv.rt.Load().cfg)
	writeJSON(w, map[string]any{"ok": true})
}

// handleLearn returns per-page policy recommendations for a site.
func (a *adminServer) handleLearn(w http.ResponseWriter, r *http.Request) {
	site := r.URL.Query().Get("site")
	if site == "" {
		http.Error(w, "site query param required", http.StatusBadRequest)
		return
	}
	if _, ok := a.findSite(site); !ok {
		http.Error(w, "unknown site: "+site, http.StatusNotFound)
		return
	}
	writeJSON(w, a.srv.learn.recommend(site))
}

// handleLearnClear resets the learner's accumulated stats for a site.
func (a *adminServer) handleLearnClear(w http.ResponseWriter, r *http.Request) {
	site := r.URL.Query().Get("site")
	if site == "" {
		http.Error(w, "site query param required", http.StatusBadRequest)
		return
	}
	a.srv.learn.clear(site)
	writeJSON(w, map[string]any{"ok": true})
}

// handleLearnApply writes a suggested page-scoped exclusion into the policy the
// given site uses, then applies + persists. This is how a learned suggestion
// becomes an enforced rule change.
func (a *adminServer) handleLearnApply(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Site    string `json:"site"`
		Path    string `json:"path"`
		RuleIDs []int  `json:"rule_ids"`
		Note    string `json:"note"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if len(req.RuleIDs) == 0 {
		http.Error(w, "rule_ids required", http.StatusBadRequest)
		return
	}
	if err := a.applyExclusion(req.Site, req.Path, req.RuleIDs, req.Note); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	site, _ := a.findSite(req.Site)
	writeJSON(w, map[string]any{"ok": true, "policy": site.Policy})
}

// applyExclusion writes learned rule exclusions as a URL-scoped PAGE POLICY on
// the site (not the shared base policy — so one site's tuning never leaks into
// another site using the same policy). Shared by learner + notification paths.
func (a *adminServer) applyExclusion(siteName, path string, ruleIDs []int, note string) error {
	if len(ruleIDs) == 0 {
		return fmt.Errorf("rule_ids required")
	}
	if note == "" {
		note = "learned exclusion for " + path
	}
	return a.savePagePolicy(siteName, PagePolicy{
		Path: path, Match: "prefix", ExcludeRuleIDs: ruleIDs, Note: note, Source: "learned",
	}, true)
}

// savePagePolicy upserts a page policy on a site by path. When mergeExclusions
// is true (the learner path), it unions exclude rule ids into an existing entry
// rather than overwriting the whole policy.
func (a *adminServer) savePagePolicy(siteName string, pp PagePolicy, mergeExclusions bool) error {
	if strings.TrimSpace(pp.Path) == "" {
		return fmt.Errorf("path required")
	}
	if pp.Match == "" {
		pp.Match = "prefix"
	}
	cfg := a.srv.rt.Load().cfg // copy
	si := -1
	for i := range cfg.Sites {
		if cfg.Sites[i].Name == siteName {
			si = i
			break
		}
	}
	if si < 0 {
		return fmt.Errorf("unknown site: %s", siteName)
	}
	found := -1
	for i, e := range cfg.Sites[si].PagePolicies {
		if e.Path == pp.Path && (e.Match == pp.Match || (e.Match == "" && pp.Match == "prefix")) {
			found = i
			break
		}
	}
	if found >= 0 && mergeExclusions {
		cfg.Sites[si].PagePolicies[found].ExcludeRuleIDs = unionInts(
			cfg.Sites[si].PagePolicies[found].ExcludeRuleIDs, pp.ExcludeRuleIDs)
		if pp.Note != "" {
			cfg.Sites[si].PagePolicies[found].Note = pp.Note
		}
	} else if found >= 0 {
		pp.Source = firstNonEmpty(pp.Source, cfg.Sites[si].PagePolicies[found].Source)
		cfg.Sites[si].PagePolicies[found] = pp
	} else {
		cfg.Sites[si].PagePolicies = append(cfg.Sites[si].PagePolicies, pp)
	}
	if err := a.srv.apply(cfg); err != nil {
		return fmt.Errorf("apply failed: %w", err)
	}
	if err := saveConfig(a.srv.configPath, cfg); err != nil {
		return fmt.Errorf("applied but not persisted: %w", err)
	}
	a.log.Info("page policy saved", "site", siteName, "path", pp.Path)
	return nil
}

func unionInts(a, b []int) []int {
	seen := map[int]bool{}
	out := []int{}
	for _, x := range append(append([]int{}, a...), b...) {
		if x > 0 && !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	sort.Ints(out)
	return out
}

// allProfiles returns built-ins plus any custom profiles from config (custom
// overrides a built-in of the same name).
func (a *adminServer) allProfiles() []Profile {
	out := builtinProfiles()
	custom := a.srv.rt.Load().cfg.Profiles
	for _, c := range custom {
		replaced := false
		for i := range out {
			if out[i].Name == c.Name {
				out[i] = c
				replaced = true
				break
			}
		}
		if !replaced {
			out = append(out, c)
		}
	}
	return out
}

// boundProfiles maps path -> profile name for page policies that came from a
// profile (Source "profile:<name>").
func boundProfiles(site SiteConfig) map[string]string {
	m := map[string]string{}
	for _, pp := range site.PagePolicies {
		if strings.HasPrefix(pp.Source, "profile:") {
			m[pp.Path] = strings.TrimPrefix(pp.Source, "profile:")
		}
	}
	return m
}

func (a *adminServer) handleProfiles(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, a.allProfiles())
}

// handleProfileSuggest returns content/structure-driven profile suggestions
// for a site's pages, marking any already bound.
func (a *adminServer) handleProfileSuggest(w http.ResponseWriter, r *http.Request) {
	site, ok := a.findSite(r.URL.Query().Get("site"))
	if !ok {
		http.Error(w, "unknown site", http.StatusNotFound)
		return
	}
	writeJSON(w, a.srv.signals.suggestProfiles(site.Name, boundProfiles(site)))
}

// handleProfileApply binds one profile to a page (human accept).
func (a *adminServer) handleProfileApply(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Site    string `json:"site"`
		Path    string `json:"path"`
		Profile string `json:"profile"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	prof, ok := profileByName(a.allProfiles(), req.Profile)
	if !ok {
		http.Error(w, "unknown profile: "+req.Profile, http.StatusBadRequest)
		return
	}
	pp := prof.toPagePolicy(req.Path)
	if err := a.savePagePolicy(req.Site, pp, false); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

// handleProfileAuto applies suggestions at/above a confidence threshold. If
// use_llm is set and the AI connector is enabled, each candidate is first
// reviewed by the LLM and only applied if it agrees (review by LLM); otherwise
// deterministic confidence alone gates it.
func (a *adminServer) handleProfileAuto(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Site      string `json:"site"`
		Threshold int    `json:"threshold"`
		UseLLM    bool   `json:"use_llm"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	site, ok := a.findSite(req.Site)
	if !ok {
		http.Error(w, "unknown site", http.StatusNotFound)
		return
	}
	if req.Threshold <= 0 {
		req.Threshold = 90
	}
	sugg := a.srv.signals.suggestProfiles(site.Name, boundProfiles(site))
	profs := a.allProfiles()
	applied := []map[string]any{}
	skipped := []map[string]any{}
	llmOn := req.UseLLM && a.srv.ai.snapshotCfg().Enabled

	for _, s := range sugg {
		if s.Bound == s.Profile {
			continue // already bound to this profile
		}
		if s.Confidence < req.Threshold {
			skipped = append(skipped, map[string]any{"path": s.Path, "reason": "below threshold", "confidence": s.Confidence})
			continue
		}
		if llmOn {
			ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
			agree, conf, reason, err := a.srv.ai.reviewProfile(ctx, s.Path, a.srv.signals.summary(site.Name, s.Path), s.Profile)
			cancel()
			if err != nil || !agree || conf < req.Threshold {
				skipped = append(skipped, map[string]any{"path": s.Path, "reason": "llm did not confirm", "detail": reason})
				continue
			}
		}
		prof, ok := profileByName(profs, s.Profile)
		if !ok {
			continue
		}
		if err := a.savePagePolicy(site.Name, prof.toPagePolicy(s.Path), false); err != nil {
			skipped = append(skipped, map[string]any{"path": s.Path, "reason": err.Error()})
			continue
		}
		applied = append(applied, map[string]any{"path": s.Path, "profile": s.Profile, "confidence": s.Confidence})
		a.srv.notify.push(notifySuggestion, "info", "Auto-applied page profile",
			s.Profile+" → "+s.Path+" on "+site.Name, "autoprof:"+site.Name+s.Path, "", nil)
	}
	writeJSON(w, map[string]any{"applied": applied, "skipped": skipped, "llm": llmOn})
}

// handlePagePolicies returns a site's URL-scoped policies.
func (a *adminServer) handlePagePolicies(w http.ResponseWriter, r *http.Request) {
	site, ok := a.findSite(r.URL.Query().Get("site"))
	if !ok {
		http.Error(w, "unknown site", http.StatusNotFound)
		return
	}
	pp := site.PagePolicies
	if pp == nil {
		pp = []PagePolicy{}
	}
	writeJSON(w, pp)
}

func (a *adminServer) handleDiscoveredForms(w http.ResponseWriter, r *http.Request) {
	site, path := r.URL.Query().Get("site"), r.URL.Query().Get("path")
	if site == "" || path == "" {
		http.Error(w, "site and path query params required", http.StatusBadRequest)
		return
	}
	if _, ok := a.findSite(site); !ok {
		http.Error(w, "unknown site", http.StatusNotFound)
		return
	}
	writeJSON(w, a.srv.signals.discoveredFields(site, path))
}

// handlePagePolicyUpsert creates/replaces a page policy (manual editing).
func (a *adminServer) handlePagePolicyUpsert(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Site   string     `json:"site"`
		Policy PagePolicy `json:"policy"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<18)).Decode(&req); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Policy.Source == "" {
		req.Policy.Source = "manual"
	}
	if err := a.savePagePolicy(req.Site, req.Policy, false); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

// handlePagePolicyDelete removes a page policy by path.
func (a *adminServer) handlePagePolicyDelete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Site  string `json:"site"`
		Path  string `json:"path"`
		Match string `json:"match"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	cfg := a.srv.rt.Load().cfg
	si := -1
	for i := range cfg.Sites {
		if cfg.Sites[i].Name == req.Site {
			si = i
			break
		}
	}
	if si < 0 {
		http.Error(w, "unknown site", http.StatusNotFound)
		return
	}
	out := cfg.Sites[si].PagePolicies[:0]
	for _, e := range cfg.Sites[si].PagePolicies {
		if e.Path == req.Path {
			continue
		}
		out = append(out, e)
	}
	cfg.Sites[si].PagePolicies = out
	if err := a.srv.apply(cfg); err != nil {
		http.Error(w, "apply failed: "+err.Error(), http.StatusUnprocessableEntity)
		return
	}
	if err := saveConfig(a.srv.configPath, cfg); err != nil {
		http.Error(w, "applied but not persisted: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

// handleAIVerdicts returns recent AI analysis verdicts, newest first.
func (a *adminServer) handleAIVerdicts(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 60
	}
	writeJSON(w, a.srv.ai.verdicts.snapshot(limit))
}

// handleAIBlocklist returns active AI-imposed blocks.
func (a *adminServer) handleAIBlocklist(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, a.srv.ai.blocklist())
}

// handleAIUnblock removes an IP from the AI blocklist.
func (a *adminServer) handleAIUnblock(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IP string `json:"ip"`
		Site string `json:"site"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil || req.IP == "" {
		http.Error(w, "ip required", http.StatusBadRequest)
		return
	}
	a.srv.ai.unblock(req.Site, req.IP)
	writeJSON(w, map[string]any{"ok": true})
}

// handleAITest runs a canned malicious sample through the configured LLM to
// validate the connector, returning the raw verdict (or the error).
func (a *adminServer) handleAITest(w http.ResponseWriter, r *http.Request) {
	if !a.srv.ai.snapshotCfg().Enabled {
		http.Error(w, "AI is disabled — enable and save first", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	v, err := a.srv.ai.test(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, v)
}

// handlePools returns live pool/member health and connection counts.
func (a *adminServer) handlePools(w http.ResponseWriter, _ *http.Request) {
	rt := a.srv.rt.Load()
	out := make([]poolStatus, 0, len(rt.cfg.Pools))
	for _, pc := range rt.cfg.Pools { // stable config order
		if pr := rt.pools[pc.Name]; pr != nil {
			out = append(out, pr.status())
		}
	}
	writeJSON(w, out)
}

// fsEntry is one directory entry. Metadata only — file contents are never
// returned, so private keys can't be exfiltrated through the browser.
type fsEntry struct {
	Name  string `json:"name"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size"`
}

// handleFS lists a directory for the cert/key file picker. Read-only, and
// clamped to -tls-browse-root so it can't wander above the configured root.
// It only ever returns names/sizes/is-dir; it does not read file bodies.
func (a *adminServer) handleFS(w http.ResponseWriter, r *http.Request) {
	root := a.srv.tlsBrowseRoot
	if root == "" {
		root = "/etc"
	}
	p := r.URL.Query().Get("path")
	if p == "" {
		p = root
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(root, p)
	}
	p = filepath.Clean(p)

	// Clamp inside root: reject anything that resolves above it.
	if rel, err := filepath.Rel(root, p); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		p = root
	}

	info, err := os.Stat(p)
	if err != nil {
		http.Error(w, "cannot access path: "+err.Error(), http.StatusBadRequest)
		return
	}
	if !info.IsDir() {
		p = filepath.Dir(p) // a file was passed — list its directory
	}

	des, err := os.ReadDir(p)
	if err != nil {
		http.Error(w, "cannot list directory: "+err.Error(), http.StatusBadRequest)
		return
	}

	entries := make([]fsEntry, 0, len(des))
	for i, de := range des {
		if i >= 3000 { // guard against pathological directories
			break
		}
		name := de.Name()
		if strings.HasPrefix(name, ".") { // skip dotfiles for a cleaner picker
			continue
		}
		e := fsEntry{Name: name, IsDir: de.IsDir()}
		if fi, err := de.Info(); err == nil && !de.IsDir() {
			e.Size = fi.Size()
		}
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir // directories first
		}
		return entries[i].Name < entries[j].Name
	})

	// parent is empty when already at the clamp root, so the UI hides "up".
	parent := ""
	if p != root {
		parent = filepath.Dir(p)
	}

	writeJSON(w, map[string]any{
		"root":    root,
		"path":    p,
		"parent":  parent,
		"entries": entries,
	})
}

func (a *adminServer) handleGetConfig(w http.ResponseWriter, _ *http.Request) {
	c := a.srv.rt.Load().cfg
	c.AI.APIKey = ""     // never expose secrets; UI shows "set" indicators
	c.HA.PeerToken = ""
	users := make([]UserConfig, len(c.Users)) // copy with hashes blanked
	for i, u := range c.Users {
		u.PasswordHash = ""
		users[i] = u
	}
	c.Users = users
	writeJSON(w, c)
}

type interfaceInfo struct {
	Name      string   `json:"name"`
	Up        bool     `json:"up"`
	Loopback  bool     `json:"loopback"`
	Addresses []string `json:"addresses"`
	Plane     string   `json:"plane,omitempty"`
}

// handleInterfaces exposes only local interface metadata needed by the site
// editor. It does not mutate networking.
func (a *adminServer) handleInterfaces(w http.ResponseWriter, _ *http.Request) {
	ifaces, err := net.Interfaces()
	if err != nil {
		http.Error(w, "list interfaces: "+err.Error(), http.StatusInternalServerError)
		return
	}
	rows := make([]interfaceInfo, 0, len(ifaces))
	for _, ifc := range ifaces {
		row := interfaceInfo{Name: ifc.Name, Up: ifc.Flags&net.FlagUp != 0, Loopback: ifc.Flags&net.FlagLoopback != 0}
		addrs, _ := ifc.Addrs()
		for _, addr := range addrs {
			row.Addresses = append(row.Addresses, addr.String())
		}
		if a.srv.ipmgr != nil && interfaceHasIP(ifc.Name, a.srv.ipmgr.managementIP) {
			row.Plane = "management"
		} else if a.srv.ipmgr != nil && ifc.Name == a.srv.ipmgr.dataInterface {
			row.Plane = "data"
		} else if row.Up && !row.Loopback {
			row.Plane = "data-candidate"
		}
		rows = append(rows, row)
	}
	writeJSON(w, rows)
}

func (a *adminServer) handlePutConfig(w http.ResponseWriter, r *http.Request) {
	var c Config
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&c); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	// Blank secrets on submit mean "keep the current ones" — the UI never
	// receives stored secrets, so it can't echo them back.
	cur := a.srv.rt.Load().cfg
	if c.AI.APIKey == "" {
		c.AI.APIKey = cur.AI.APIKey
	}
	if c.HA.PeerToken == "" {
		c.HA.PeerToken = cur.HA.PeerToken
	}
	// Users are managed only via the dedicated user endpoints; a general config
	// save never touches them (the console can't see the hashes anyway).
	c.Users = cur.Users

	// Draft save: persist work-in-progress to disk WITHOUT applying it to the
	// live engine. Lets you build a config incrementally (add a node, save; add
	// a pool, save) without every intermediate state being fully consistent.
	// The running WAF keeps serving the last APPLIED config untouched.
	if r.URL.Query().Get("draft") == "1" {
		if err := c.validateDraft(); err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		if err := saveConfig(a.srv.configPath, c); err != nil {
			http.Error(w, "not persisted: "+err.Error(), http.StatusInternalServerError)
			return
		}
		a.audit.add(who(r).user, "config.draft_saved", fmt.Sprintf("%d sites, %d pools, %d nodes", len(c.Sites), len(c.Pools), len(c.Nodes)))
		applyErr := ""
		if err := c.validate(); err != nil {
			applyErr = err.Error()
		}
		resp := c
		resp.AI.APIKey = ""
		resp.HA.PeerToken = ""
		writeJSON(w, map[string]any{"config": resp, "draft": true, "apply_ready": applyErr == "", "apply_error": applyErr})
		return
	}

	if err := c.validate(); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	// A PUT carrying X-WAF-Sync came from the HA peer: apply without pushing
	// it back (loop guard).
	fromSync := r.Header.Get("X-WAF-Sync") == "1"
	if err := a.srv.applyEx(c, fromSync); err != nil {
		http.Error(w, "apply failed: "+err.Error(), http.StatusUnprocessableEntity)
		return
	}
	if err := saveConfig(a.srv.configPath, c); err != nil {
		http.Error(w, "applied but not persisted: "+err.Error(), http.StatusInternalServerError)
		return
	}
	a.log.Info("config applied via admin API", "engine_mode", c.EngineMode, "sites", len(c.Sites), "from_sync", fromSync)
	if !fromSync {
		a.audit.add(who(r).user, "config.apply", fmt.Sprintf("%d sites, %d pools, %d policies", len(c.Sites), len(c.Pools), len(c.Policies)))
	}
	resp := c
	resp.AI.APIKey = ""
	resp.HA.PeerToken = ""
	writeJSON(w, map[string]any{
		"config":           resp,
		"applied":          true,
		"restart_required": a.srv.restartPending(c),
	})
}

func (a *adminServer) handleReload(w http.ResponseWriter, _ *http.Request) {
	cfg := a.srv.rt.Load().cfg
	if err := a.srv.apply(cfg); err != nil {
		http.Error(w, "reload failed: "+err.Error(), http.StatusUnprocessableEntity)
		return
	}
	a.log.Info("rules reloaded via admin API", "rules", cfg.Rules)
	writeJSON(w, map[string]any{"ok": true})
}

func (a *adminServer) handleSyslogTest(w http.ResponseWriter, _ *http.Request) {
	if err := a.srv.syslog.test(); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (a *adminServer) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	hist := a.srv.metrics.history()
	var cur metricSample
	if len(hist) > 0 {
		cur = hist[len(hist)-1]
	}
	writeJSON(w, map[string]any{
		"current": cur,
		"history": hist,
		"cpus":    runtime.NumCPU(),
	})
}

func (a *adminServer) handleDiscovered(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, a.srv.hosts.snapshot())
}

func (a *adminServer) handleMatches(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}
	writeJSON(w, a.srv.matches.snapshot(limit))
}

func (a *adminServer) handleAccess(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 300
	}
	writeJSON(w, a.srv.access.snapshot(limit))
}

// handleSitemap returns the path tree for one site (?site=) or all sites.
func (a *adminServer) handleSitemap(w http.ResponseWriter, r *http.Request) {
	cfg := a.srv.rt.Load().cfg
	want := r.URL.Query().Get("site")
	out := make([]siteMapJSON, 0, len(cfg.Sites))
	for _, sc := range cfg.Sites {
		if want != "" && sc.Name != want {
			continue
		}
		out = append(out, a.srv.maps.snapshot(sc.Name))
	}
	writeJSON(w, out)
}

func (a *adminServer) findSite(name string) (SiteConfig, bool) {
	for _, sc := range a.srv.rt.Load().cfg.Sites {
		if sc.Name == name {
			return sc, true
		}
	}
	return SiteConfig{}, false
}

// crawlBackend picks a target for a site's crawl: a healthy member of its
// pool if any, otherwise the first member (so a fully-down monitor still lets
// you probe). Returns nil if the pool has no members.
func (a *adminServer) crawlBackend(site SiteConfig) *url.URL {
	rt := a.srv.rt.Load()
	pr := rt.pools[site.Pool]
	if pr == nil || len(pr.members) == 0 {
		return nil
	}
	m := pr.pick("crawler")
	if m == nil {
		m = pr.members[0]
	}
	u := *m.target
	return &u
}

// handleCrawl starts a bounded, polite crawl of a site's backend (via its pool).
func (a *adminServer) handleCrawl(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Site     string `json:"site"`
		MaxPages int    `json:"max_pages"`
		MaxDepth int    `json:"max_depth"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	site, ok := a.findSite(req.Site)
	if !ok {
		http.Error(w, "unknown site: "+req.Site, http.StatusNotFound)
		return
	}
	backend := a.crawlBackend(site)
	if backend == nil {
		http.Error(w, "site's pool has no members to crawl", http.StatusConflict)
		return
	}
	opts := crawlOpts{maxPages: req.MaxPages, maxDepth: req.MaxDepth}
	if opts.maxPages <= 0 {
		opts.maxPages = 200
	}
	if opts.maxPages > 2000 {
		opts.maxPages = 2000
	}
	if opts.maxDepth <= 0 {
		opts.maxDepth = 4
	}
	if opts.maxDepth > 12 {
		opts.maxDepth = 12
	}
	if !a.srv.startCrawl(site, backend, opts) {
		http.Error(w, "a crawl is already running for this site", http.StatusConflict)
		return
	}
	a.log.Info("crawl started", "site", site.Name, "backend", backend.String(),
		"max_pages", opts.maxPages, "max_depth", opts.maxDepth)
	writeJSON(w, map[string]any{"started": true, "site": site.Name})
}

func (a *adminServer) handleSitemapClear(w http.ResponseWriter, r *http.Request) {
	site := r.URL.Query().Get("site")
	if site == "" {
		http.Error(w, "site query param required", http.StatusBadRequest)
		return
	}
	a.srv.maps.clear(site)
	_ = a.srv.maps.save(a.srv.configPath) // persist the cleared state
	a.srv.signals.clear(site)
	a.srv.learn.clear(site)
	writeJSON(w, map[string]any{"ok": true})
}
