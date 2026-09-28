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

	"waf-proxy/internal/tlsfront"

	_ "embed"
)

//go:embed static/admin.html
var adminHTML []byte

//go:embed static/theme.css
var adminThemeCSS []byte

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
	next int
	size int
}

func newMatchRing(capacity int) *matchRing {
	if capacity < 0 {
		capacity = 0
	}
	return &matchRing{recs: make([]matchRec, capacity), cap: capacity}
}

func (r *matchRing) add(m matchRec) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cap == 0 {
		return
	}
	r.recs[r.next] = m
	r.next = (r.next + 1) % r.cap
	if r.size < r.cap {
		r.size++
	}
}

// snapshot returns up to limit records, newest first.
func (r *matchRing) snapshot(limit int) []matchRec {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit <= 0 || limit > r.size {
		limit = r.size
	}
	out := make([]matchRec, limit)
	for i := 0; i < limit; i++ {
		idx := (r.next - 1 - i + r.cap) % r.cap
		out[i] = r.recs[idx]
	}
	return out
}

func (r *matchRing) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.size
}

// accessRec is one line of the request/access log.
type accessRec struct {
	At     time.Time `json:"-"`
	Site   string    `json:"site"`
	Client string    `json:"client"`
	Method string    `json:"method"`
	Path   string    `json:"path"`
	Status int       `json:"status"`
}

func (a accessRec) MarshalJSON() ([]byte, error) {
	type accessWire struct {
		Time   string `json:"time"`
		Site   string `json:"site"`
		Client string `json:"client"`
		Method string `json:"method"`
		Path   string `json:"path"`
		Status int    `json:"status"`
	}
	return json.Marshal(accessWire{
		Time: a.At.Format("15:04:05"), Site: a.Site, Client: a.Client,
		Method: a.Method, Path: a.Path, Status: a.Status,
	})
}

type accessRing struct {
	mu   sync.Mutex
	recs []accessRec
	cap  int
	next int
	size int
}

func newAccessRing(capacity int) *accessRing {
	if capacity < 0 {
		capacity = 0
	}
	return &accessRing{recs: make([]accessRec, capacity), cap: capacity}
}

func (r *accessRing) add(m accessRec) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cap == 0 {
		return
	}
	r.recs[r.next] = m
	r.next = (r.next + 1) % r.cap
	if r.size < r.cap {
		r.size++
	}
}

func (r *accessRing) snapshot(limit int) []accessRec {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit <= 0 || limit > r.size {
		limit = r.size
	}
	out := make([]accessRec, limit)
	for i := 0; i < limit; i++ {
		idx := (r.next - 1 - i + r.cap) % r.cap
		out[i] = r.recs[idx]
	}
	return out
}

// ── admin server ────────────────────────────────────────────────────────

type adminServer struct {
	srv               *server
	token             string
	haPeerToken       string
	haPeerTokenPinned bool
	sessions          *sessionStore
	loginLimiter      *loginAttemptLimiter
	audit             *auditLog
	staged            stagedUpdate
	log               *slog.Logger
	started           time.Time
}

func newAdminServer(s *server, token, haPeerToken string, log *slog.Logger) *adminServer {
	haPeerToken = strings.TrimSpace(haPeerToken)
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
	as := &adminServer{srv: s, token: token, haPeerToken: haPeerToken, haPeerTokenPinned: haPeerToken != "", sessions: newSessionStore(), loginLimiter: newLoginAttemptLimiter(), audit: newAuditLog(), log: log, started: time.Now()}
	as.audit.sink = s.syslog.forwardAudit // fan audit entries out to syslog
	auditPath := strings.TrimSpace(os.Getenv("WAF_ADMIN_AUDIT_FILE"))
	if auditPath == "" && s.configPath != "" {
		auditPath = filepath.Join(filepath.Dir(s.configPath), "admin-audit.jsonl")
	}
	if auditPath != "" {
		if err := as.audit.configurePersistence(auditPath, log); err != nil {
			log.Error("admin audit persistence unavailable", "path", auditPath, "err", err)
		}
	}
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

// haPeerAuth is intentionally narrower than normal admin authentication. HA
// config replication must use the receiver's break-glass peer token and the
// dedicated sync protocol marker; browser/admin sessions cannot masquerade as
// replication traffic merely by setting a header.
func (a *adminServer) haPeerAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.haPeerTokenPinned {
			http.Error(w, "HA config sync requires WAF_HA_PEER_TOKEN or -ha-peer-token", http.StatusServiceUnavailable)
			return
		}
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if got == "" || subtle.ConstantTimeCompare([]byte(got), []byte(a.haPeerToken)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Header.Get("X-WAF-HA-Sync") != "v1" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		id := identity{user: "(ha-peer)", role: roleAdmin}
		next(w, r.WithContext(context.WithValue(r.Context(), identityKey, id)))
	}
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
		r = r.WithContext(context.WithValue(r.Context(), identityKey, id))
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next(w, r)
			return
		}
		aw := &mutationAuditWriter{ResponseWriter: w}
		next(aw, r)
		if aw.status == 0 {
			aw.status = http.StatusOK
		}
		if aw.status < 400 {
			a.audit.add(id.user, "http.mutation", r.Method+" "+r.URL.Path)
		}
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
			"default-src 'none'; script-src 'unsafe-inline'; style-src 'self' 'unsafe-inline'; connect-src 'self'")
		_, _ = w.Write(adminHTML)
	})

	mux.HandleFunc("GET /theme.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		_, _ = w.Write(adminThemeCSS)
	})

	mux.HandleFunc("GET /api/status", a.auth(a.handleStatus))
	mux.HandleFunc("GET /api/config", a.auth(a.handleGetConfig))
	mux.HandleFunc("GET /api/interfaces", a.auth(a.handleInterfaces))
	mux.HandleFunc("PUT /api/config", a.authRole(roleOperator, a.handlePutConfig))
	mux.HandleFunc("POST /api/reload", a.authRole(roleOperator, a.handleReload))
	mux.HandleFunc("GET /api/matches", a.auth(a.handleMatches))
	mux.HandleFunc("GET /api/access", a.auth(a.handleAccess))
	mux.HandleFunc("GET /api/pools", a.auth(a.handlePools))
	mux.HandleFunc("GET /api/ai/verdicts", a.auth(a.handleAIVerdicts))
	mux.HandleFunc("GET /api/ai/blocklist", a.auth(a.handleAIBlocklist))
	mux.HandleFunc("POST /api/ai/unblock", a.authRole(roleOperator, a.handleAIUnblock))
	mux.HandleFunc("POST /api/ai/test", a.authRole(roleOperator, a.handleAITest))
	mux.HandleFunc("POST /api/syslog/test", a.authRole(roleOperator, a.handleSyslogTest))
	mux.HandleFunc("GET /api/learn", a.auth(a.handleLearn))
	mux.HandleFunc("POST /api/learn/apply", a.authRole(roleReviewer, a.handleLearnApply))
	mux.HandleFunc("POST /api/learn/clear", a.authRole(roleReviewer, a.handleLearnClear))
	mux.HandleFunc("GET /api/notifications", a.auth(a.handleNotifications))
	mux.HandleFunc("POST /api/notifications/read", a.authRole(roleReviewer, a.handleNotifyRead))
	mux.HandleFunc("POST /api/notifications/dismiss", a.authRole(roleReviewer, a.handleNotifyDismiss))
	mux.HandleFunc("DELETE /api/notifications/webhook", a.authRole(roleOperator, a.handleNotifyWebhookClear))
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
	mux.HandleFunc("PUT /api/ha/peer-config", a.haPeerAuth(a.handleHAPeerConfig))
	mux.HandleFunc("GET /api/pagepolicy", a.auth(a.handlePagePolicies))
	mux.HandleFunc("GET /api/forms", a.auth(a.handleDiscoveredForms))
	mux.HandleFunc("GET /api/security/contracts", a.auth(a.handleContracts))
	mux.HandleFunc("POST /api/security/contracts/import", a.authRole(roleOperator, a.handleContractImport))
	mux.HandleFunc("GET /api/security/contracts/{id}", a.auth(a.handleContractDetail))
	mux.HandleFunc("GET /api/security/contracts/{id}/versions", a.auth(a.handleContractVersions))
	mux.HandleFunc("POST /api/security/contracts/{id}/match", a.authRole(roleReviewer, a.handleContractMatch))
	mux.HandleFunc("GET /api/security/contracts/{id}/bindings", a.auth(a.handleContractBindings))
	mux.HandleFunc("POST /api/security/contracts/{id}/compare", a.authRole(roleReviewer, a.handleContractCompare))
	mux.HandleFunc("GET /api/security/contracts/{id}/diffs", a.auth(a.handleContractDiffs))
	mux.HandleFunc("GET /api/security/contracts/{id}/drift", a.auth(a.handleContractDrift))
	mux.HandleFunc("GET /api/security/contracts/{id}/export", a.auth(a.handleContractExport))
	mux.HandleFunc("GET /api/security/schema/candidates", a.auth(a.handleSchemaCandidates))
	mux.HandleFunc("GET /api/security/schema/enforcement", a.auth(a.handlePositiveSchemaList))
	mux.HandleFunc("GET /api/security/schema/enforcement/violations", a.authRole(roleReviewer, a.handlePositiveSchemaViolations))
	mux.HandleFunc("GET /api/security/schema/enforcement/{operation_id}", a.auth(a.handlePositiveSchemaDetail))
	mux.HandleFunc("POST /api/security/schema/enforcement/{operation_id}/mode", a.authRole(roleReviewer, a.handlePositiveSchemaMode))
	mux.HandleFunc("POST /api/security/schema/enforcement/{operation_id}/activate", a.authRole(roleReviewer, a.handlePositiveSchemaActivate))
	mux.HandleFunc("POST /api/security/schema/enforcement/{operation_id}/rollback", a.authRole(roleReviewer, a.handlePositiveSchemaRollback))
	mux.HandleFunc("POST /api/security/schema/enforcement/{operation_id}/exceptions", a.authRole(roleReviewer, a.handlePositiveSchemaExceptionCreate))
	mux.HandleFunc("POST /api/security/schema/enforcement/exceptions/{exception_id}", a.authRole(roleReviewer, a.handlePositiveSchemaExceptionToggle))
	mux.HandleFunc("GET /api/security/schema/{id}", a.auth(a.handleSchemaDetail))
	mux.HandleFunc("POST /api/security/schema/{id}/review", a.authRole(roleReviewer, a.handleSchemaReview))
	mux.HandleFunc("POST /api/security/schema/{id}/ai-review", a.authRole(roleReviewer, a.handleSchemaAIReview))
	mux.HandleFunc("POST /api/security/schema/{id}/enforcement", a.authRole(roleReviewer, a.handlePositiveSchemaPromote))
	mux.HandleFunc("GET /api/security/identity/issuers", a.auth(a.handleIdentityIssuers))
	mux.HandleFunc("POST /api/security/identity/issuers", a.authRole(roleReviewer, a.handleIdentityIssuerUpsert))
	mux.HandleFunc("POST /api/security/identity/issuers/{issuer_id}/refresh", a.authRole(roleReviewer, a.handleIdentityIssuerRefresh))
	mux.HandleFunc("GET /api/security/identity/policies", a.auth(a.handleIdentityPolicies))
	mux.HandleFunc("POST /api/security/identity/policies/{operation_id}", a.authRole(roleReviewer, a.handleIdentityPolicyUpsert))
	mux.HandleFunc("POST /api/security/identity/policies/{operation_id}/mode", a.authRole(roleReviewer, a.handleIdentityPolicyMode))
	mux.HandleFunc("DELETE /api/security/identity/policies/{operation_id}", a.authRole(roleReviewer, a.handleIdentityPolicyDelete))
	mux.HandleFunc("GET /api/security/identity/violations", a.authRole(roleReviewer, a.handleIdentityViolations))
	mux.HandleFunc("GET /api/security/sequence/model", a.auth(a.handleSequenceModel))
	mux.HandleFunc("GET /api/security/sequence/sessions", a.authRole(roleReviewer, a.handleSequenceSessions))
	mux.HandleFunc("GET /api/security/sequence/transitions", a.authRole(roleReviewer, a.handleSequenceTransitions))
	mux.HandleFunc("GET /api/security/sequence/workflows", a.authRole(roleReviewer, a.handleSequenceWorkflows))
	mux.HandleFunc("GET /api/security/sequence/violations", a.authRole(roleReviewer, a.handleSequenceViolations))
	mux.HandleFunc("GET /api/security/sequence/recent-sessions", a.authRole(roleReviewer, a.handleSequenceRecentSessions))
	mux.HandleFunc("GET /api/security/sequence/exceptions", a.authRole(roleReviewer, a.handleSequenceExceptions))
	mux.HandleFunc("GET /api/security/sequence/learning-state", a.authRole(roleReviewer, a.handleSequenceLearningState))
	mux.HandleFunc("POST /api/security/sequence/mode", a.authRole(roleReviewer, a.handleSequenceMode))
	mux.HandleFunc("POST /api/security/sequence/relearn", a.authRole(roleReviewer, a.handleSequenceRelearn))
	mux.HandleFunc("POST /api/security/sequence/exceptions", a.authRole(roleReviewer, a.handleSequenceExceptionCreate))
	mux.HandleFunc("DELETE /api/security/sequence/exceptions/{exception_id}", a.authRole(roleReviewer, a.handleSequenceExceptionDelete))
	mux.HandleFunc("GET /api/security/object-locators", a.authRole(roleReviewer, a.handleObjectLocators))
	mux.HandleFunc("GET /api/security/object-locators/overrides", a.authRole(roleReviewer, a.handleObjectLocatorOverrides))
	mux.HandleFunc("POST /api/security/object-locators/overrides", a.authRole(roleReviewer, a.handleObjectLocatorOverrideUpsert))
	mux.HandleFunc("DELETE /api/security/object-locators/overrides/{override_id}", a.authRole(roleReviewer, a.handleObjectLocatorOverrideDelete))
	mux.HandleFunc("GET /api/security/object-relationships", a.authRole(roleReviewer, a.handleObjectRelationships))
	mux.HandleFunc("GET /api/security/object-relationships/status", a.authRole(roleReviewer, a.handleObjectRelationshipStatus))
	mux.HandleFunc("GET /api/security/bola/candidates", a.authRole(roleReviewer, a.handleBOLACandidates))
	mux.HandleFunc("GET /api/security/bola/status", a.authRole(roleReviewer, a.handleBOLAStatus))
	mux.HandleFunc("GET /api/security/bola/evidence", a.authRole(roleReviewer, a.handleBOLAEvidence))
	mux.HandleFunc("POST /api/security/bola/evidence/{candidate_id}/workflow", a.authRole(roleReviewer, a.handleBOLAEvidenceWorkflow))
	mux.HandleFunc("GET /api/security/bola/policies", a.authRole(roleReviewer, a.handleBOLAPolicies))
	mux.HandleFunc("POST /api/security/bola/policies", a.authRole(roleReviewer, a.handleBOLAPolicyUpsert))
	mux.HandleFunc("DELETE /api/security/bola/policies/{policy_id}", a.authRole(roleReviewer, a.handleBOLAPolicyDelete))
	mux.HandleFunc("GET /api/security/bola/policy-status", a.authRole(roleReviewer, a.handleBOLAPolicyStatus))
	mux.HandleFunc("GET /api/security/graphql/operations", a.authRole(roleReviewer, a.handleGraphQLOperations))
	mux.HandleFunc("GET /api/security/graphql/persisted-queries", a.authRole(roleReviewer, a.handleGraphQLPersisted))
	mux.HandleFunc("GET /api/security/graphql/schema-contracts", a.authRole(roleReviewer, a.handleGraphQLSchemaContracts))
	mux.HandleFunc("POST /api/security/graphql/schema-contracts", a.authRole(roleReviewer, a.handleGraphQLSchemaContractImport))
	mux.HandleFunc("DELETE /api/security/graphql/schema-contracts/{contract_id}", a.authRole(roleReviewer, a.handleGraphQLSchemaContractDelete))
	mux.HandleFunc("GET /api/security/graphql/policies", a.authRole(roleReviewer, a.handleGraphQLPolicies))
	mux.HandleFunc("POST /api/security/graphql/policies", a.authRole(roleReviewer, a.handleGraphQLPolicyUpsert))
	mux.HandleFunc("POST /api/security/graphql/policies/{policy_id}/mode", a.authRole(roleReviewer, a.handleGraphQLPolicyMode))
	mux.HandleFunc("DELETE /api/security/graphql/policies/{policy_id}", a.authRole(roleReviewer, a.handleGraphQLPolicyDelete))
	mux.HandleFunc("GET /api/security/graphql/violations", a.authRole(roleReviewer, a.handleGraphQLViolations))
	mux.HandleFunc("GET /api/security/graphql/status", a.authRole(roleReviewer, a.handleGraphQLStatus))
	mux.HandleFunc("POST /api/pagepolicy/upsert", a.authRole(roleReviewer, a.handlePagePolicyUpsert))
	mux.HandleFunc("POST /api/pagepolicy/delete", a.authRole(roleReviewer, a.handlePagePolicyDelete))
	mux.HandleFunc("GET /api/profiles", a.auth(a.handleProfiles))
	mux.HandleFunc("GET /api/profiles/suggest", a.auth(a.handleProfileSuggest))
	mux.HandleFunc("POST /api/profiles/apply", a.authRole(roleReviewer, a.handleProfileApply))
	mux.HandleFunc("POST /api/profiles/auto", a.authRole(roleReviewer, a.handleProfileAuto))
	mux.HandleFunc("GET /api/fs", a.auth(a.handleFS))
	mux.HandleFunc("GET /api/sitemap", a.auth(a.handleSitemap))
	mux.HandleFunc("POST /api/crawl", a.authRole(roleOperator, a.handleCrawl))
	mux.HandleFunc("POST /api/sitemap/clear", a.authRole(roleReviewer, a.handleSitemapClear))
	mux.HandleFunc("GET /api/discovered", a.auth(a.handleDiscovered))
	mux.HandleFunc("GET /api/security/operations", a.auth(a.handleAPIOperations))
	mux.HandleFunc("GET /api/security/operations/{id}", a.auth(a.handleAPIOperationDetail))
	mux.HandleFunc("POST /api/security/operations/{id}/ignore", a.authRole(roleReviewer, a.handleAPIOperationIgnore))
	mux.HandleFunc("POST /api/security/operations/reclassify", a.authRole(roleReviewer, a.handleAPIOperationReclassify))
	mux.HandleFunc("GET /api/metrics", a.auth(a.handleMetrics))
	mux.HandleFunc("GET /api/vector-acceleration", a.auth(a.handleVectorAcceleration))
	mux.HandleFunc("POST /api/vector-acceleration/reset", a.authRole(roleReviewer, a.handleVectorAccelerationReset))
	mux.HandleFunc("GET /api/tls-acceleration", a.auth(a.handleTLSAcceleration))
	mux.HandleFunc("GET /api/hsm/status", a.auth(a.handleHSMStatus))
	mux.HandleFunc("GET /api/hsm/audit", a.authRole(roleReviewer, a.handleHSMAudit))
	mux.HandleFunc("GET /api/doctor", a.auth(a.handleDoctor))
	mux.HandleFunc("GET /api/debug/status", a.auth(a.handleDebugStatus))
	mux.HandleFunc("POST /api/debug/capture", a.authRole(roleOperator, a.handleDebugCapture))
	mux.HandleFunc("GET /api/debug/evidence", a.authRole(roleReviewer, a.handleDebugEvidence))
	mux.HandleFunc("GET /api/debug/export", a.authRole(roleReviewer, a.handleDebugExport))
	a.registerUpdateRoutes(mux) // signed self-update (localhost + admin, 404 when no key)
	return mux
}

func setJSONSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

func writeJSON(w http.ResponseWriter, v any) {
	setJSONSecurityHeaders(w)
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
			"name":             sc.Name,
			"listen":           sc.Listen,
			"hostnames":        sc.Hostnames,
			"pool":             sc.Pool,
			"engine":           mode,
			"tls":              siteTLSEnabled(sc),
			"tls_key_provider": hsmStatusLabel(sc),
		})
	}
	anyTLS := false
	for _, t := range publicListenerSet(rt.cfg) {
		anyTLS = anyTLS || t
	}
	writeJSON(w, map[string]any{
		"uptime":           time.Since(a.started).Truncate(time.Second).String(),
		"engine_mode":      rt.cfg.EngineMode,
		"listeners":        sortedListenAddrs(rt.cfg),
		"sites":            sites,
		"pools":            len(rt.cfg.Pools),
		"policies":         len(rt.cfg.Policies),
		"nodes":            len(rt.cfg.Nodes),
		"tls":              anyTLS,
		"tls_acceleration": rt.cfg.TLSAcceleration.Effective(),
		"tls_frontend":     a.srv.tlsFrontend.status(),
		"match_count":      a.srv.matches.count(),
		"restart_pending":  a.srv.restartPending(rt.cfg),
		"rules_built_at":   rt.builtAt.Format(time.RFC3339),
		"ai_enabled":       rt.cfg.AI.Enabled,
		"ai_key_set":       rt.cfg.AI.secretConfigured(),
		"ai_api_style":     rt.cfg.AI.effectiveOpenAIAPIStyle(),
		"version":          buildVersion,
		"commit":           buildCommit,
		"ha_enabled":       rt.cfg.HA.Enabled,
		"ha":               a.srv.ha.status(),
		"ha_token_set":     rt.cfg.HA.PeerToken != "",
		"notify_unread":    a.srv.notify.unreadCount(),
	})
}

func (a *adminServer) handleHSMStatus(w http.ResponseWriter, r *http.Request) {
	rt := a.srv.rt.Load()
	if rt == nil {
		http.Error(w, "runtime unavailable", http.StatusServiceUnavailable)
		return
	}
	statuses := make([]any, 0, len(rt.hsmSigners))
	for _, signer := range rt.hsmSigners {
		if signer == nil {
			continue
		}
		statuses = append(statuses, signer.HealthCheck(r.Context()))
	}
	writeJSON(w, map[string]any{"providers": statuses})
}

func (a *adminServer) handleHSMAudit(w http.ResponseWriter, _ *http.Request) {
	if a.srv.hsmAudit == nil {
		writeJSON(w, []any{})
		return
	}
	writeJSON(w, a.srv.hsmAudit.List())
}

func (a *adminServer) handleTLSAcceleration(w http.ResponseWriter, _ *http.Request) {
	rt := a.srv.rt.Load()
	if rt == nil {
		http.Error(w, "runtime unavailable", http.StatusServiceUnavailable)
		return
	}
	effective := rt.cfg.TLSAcceleration.Effective()
	mappings := []map[string]any{}
	for addr, isTLS := range publicListenerSet(rt.cfg) {
		if !isTLS {
			continue
		}
		mappings = append(mappings, map[string]any{
			"public":   addr,
			"internal": tlsfront.InternalListenerKey(addr),
		})
	}
	sort.Slice(mappings, func(i, j int) bool { return mappings[i]["public"].(string) < mappings[j]["public"].(string) })
	writeJSON(w, map[string]any{
		"configured": rt.cfg.TLSAcceleration,
		"effective":  effective,
		"runtime":    a.srv.tlsFrontend.status(),
		"mappings":   mappings,
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
	now := time.Now()
	keys := loginAttemptKeys(r, req.Username)
	if ok, retry := a.loginLimiter.allow(keys, now); !ok {
		secs := int(retry.Seconds())
		if secs < 1 {
			secs = 1
		}
		w.Header().Set("Retry-After", strconv.Itoa(secs))
		a.audit.add("(login)", "login.throttled", "remote="+adminRemoteHost(r))
		http.Error(w, "too many login attempts", http.StatusTooManyRequests)
		return
	}
	cfg := a.srv.rt.Load().cfg
	for _, u := range cfg.Users {
		if strings.EqualFold(u.Username, req.Username) {
			if u.Disabled || !verifyPassword(req.Password, u.PasswordHash) {
				break
			}
			tok, err := a.sessions.createWithError(u.Username, u.Role)
			if err != nil {
				a.log.Error("session token generation failed", "err", err)
				http.Error(w, "login unavailable", http.StatusServiceUnavailable)
				return
			}
			a.loginLimiter.success(keys)
			a.audit.add(u.Username, "login", "")
			writeJSON(w, map[string]any{"token": tok, "user": u.Username, "role": u.Role})
			return
		}
	}
	// constant-ish: run a verify against a dummy to blunt user enumeration timing
	verifyPassword(req.Password, "pbkdf2$sha256$210000$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
	a.loginLimiter.failure(keys, now)
	a.audit.add("(login)", "login.failed", "remote="+adminRemoteHost(r))
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

// mutateUsers derives the change from the newest runtime under the server's
// config transaction lock so simultaneous page-policy/config operations cannot
// be overwritten by a stale user snapshot.
func (a *adminServer) mutateUsers(fn func(users []UserConfig) ([]UserConfig, error)) error {
	_, err := a.srv.mutatePersisted(false, func(cfg *Config) error {
		next, err := fn(append([]UserConfig(nil), cfg.Users...))
		if err != nil {
			return err
		}
		cfg.Users = next
		return nil
	})
	if err != nil {
		return fmt.Errorf("apply/persist failed: %w", err)
	}
	return nil
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
		"unread":           a.srv.notify.unreadCount(),
		"items":            a.srv.notify.list(limit),
		"webhook_delivery": a.srv.notify.webhookDeliveryStats(),
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
	a.audit.add(who(r).user, "notification.read", fmt.Sprintf("id=%d all=%t", id, all))
	writeJSON(w, map[string]any{"ok": true})
}

func (a *adminServer) handleNotifyDismiss(w http.ResponseWriter, r *http.Request) {
	id, all, ok := decodeIDReq(w, r)
	if !ok {
		return
	}
	a.srv.notify.dismiss(id, all)
	a.audit.add(who(r).user, "notification.dismiss", fmt.Sprintf("id=%d all=%t", id, all))
	writeJSON(w, map[string]any{"ok": true})
}

// handleNotifyApply executes a notification's attached action (currently only
// apply_exclusion, which writes a learned exclusion into the site's policy).
func (a *adminServer) handleNotifyWebhookClear(w http.ResponseWriter, r *http.Request) {
	if _, err := a.srv.mutatePersisted(false, func(cfg *Config) error {
		cfg.Notify.WebhookURL = ""
		return nil
	}); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	a.audit.add(who(r).user, "notification.webhook_clear", "")
	writeJSON(w, map[string]any{"ok": true})
}

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

func (a *adminServer) handleHASync(w http.ResponseWriter, r *http.Request) {
	if !a.srv.ha.snapshotCfg().Enabled {
		http.Error(w, "HA is disabled", http.StatusBadRequest)
		return
	}
	a.srv.ha.pushConfig(a.srv.rt.Load().cfg)
	a.audit.add(who(r).user, "ha.sync_requested", "")
	writeJSON(w, map[string]any{"ok": true})
}

func (a *adminServer) handleHAPeerConfig(w http.ResponseWriter, r *http.Request) {
	var env haSyncEnvelope
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&env); err != nil {
		http.Error(w, "bad peer config: "+err.Error(), http.StatusBadRequest)
		return
	}
	if env.Version != haSyncEnvelopeVersion {
		http.Error(w, "unsupported HA sync version", http.StatusConflict)
		return
	}
	next, err := a.srv.mutatePersisted(true, func(local *Config) error {
		merged := mergePeerConfig(*local, env.Config)
		*local = merged
		return nil
	})
	if err != nil {
		http.Error(w, "peer apply/persist failed: "+err.Error(), http.StatusUnprocessableEntity)
		return
	}
	a.audit.add("(ha-peer)", "ha.peer_config_applied", fmt.Sprintf("%d sites, %d pools, %d policies", len(next.Sites), len(next.Pools), len(next.Policies)))
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
	a.audit.add(who(r).user, "learner.clear", "site="+site)
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
	_, err := a.srv.mutatePersisted(false, func(cfg *Config) error {
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
		return nil
	})
	if err != nil {
		return fmt.Errorf("apply/persist failed: %w", err)
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
	if _, err := a.srv.mutatePersisted(false, func(cfg *Config) error {
		si := -1
		for i := range cfg.Sites {
			if cfg.Sites[i].Name == req.Site {
				si = i
				break
			}
		}
		if si < 0 {
			return fmt.Errorf("unknown site")
		}
		out := make([]PagePolicy, 0, len(cfg.Sites[si].PagePolicies))
		for _, e := range cfg.Sites[si].PagePolicies {
			if e.Path == req.Path && (req.Match == "" || e.Match == req.Match) {
				continue
			}
			out = append(out, e)
		}
		cfg.Sites[si].PagePolicies = out
		return nil
	}); err != nil {
		if strings.Contains(err.Error(), "unknown site") {
			http.Error(w, "unknown site", http.StatusNotFound)
		} else {
			http.Error(w, "apply/persist failed: "+err.Error(), http.StatusUnprocessableEntity)
		}
		return
	}
	a.audit.add(who(r).user, "pagepolicy.delete", "site="+req.Site+" path="+req.Path)
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
		IP   string `json:"ip"`
		Site string `json:"site"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil || req.IP == "" {
		http.Error(w, "ip required", http.StatusBadRequest)
		return
	}
	a.srv.ai.unblock(req.Site, req.IP)
	a.audit.add(who(r).user, "ai.unblock", "site="+req.Site)
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
	a.audit.add(who(r).user, "ai.connector_test", "")
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
	root = filepath.Clean(root)
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		http.Error(w, "cannot resolve browse root: "+err.Error(), http.StatusInternalServerError)
		return
	}
	p := r.URL.Query().Get("path")
	if p == "" {
		p = resolvedRoot
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(resolvedRoot, p)
	}
	p = filepath.Clean(p)
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil {
		http.Error(w, "cannot resolve path: "+err.Error(), http.StatusBadRequest)
		return
	}
	if rel, err := filepath.Rel(resolvedRoot, resolved); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		http.Error(w, "path escapes browse root", http.StatusForbidden)
		return
	}
	p = resolved

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

func (a *adminServer) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	rt := a.srv.rt.Load()
	if rt == nil {
		http.Error(w, "runtime unavailable", http.StatusServiceUnavailable)
		return
	}
	c := rt.cfg
	if r.URL.Query().Get("draft") == "1" {
		if draft, ok, err := loadDraftConfig(a.srv.configPath); err != nil {
			http.Error(w, "load draft: "+err.Error(), http.StatusInternalServerError)
			return
		} else if ok {
			c = draft
			w.Header().Set("X-WAF-Config-Source", "draft")
		} else {
			w.Header().Set("X-WAF-Config-Source", "live")
		}
	}
	c = redactAISecrets(c)
	c.HA.PeerToken = ""
	c = redactHSMSecretRefs(c)
	c = redactNotifySecrets(c)
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
	// Preserve the historical passive-discovery behavior for older API clients
	// that submit a full config without the newly added toggle.
	c := Config{PassiveDiscoveryEnabled: true}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	migrateTrustedProxyConfig(&c)
	draft := r.URL.Query().Get("draft") == "1"

	// Full-config submissions still contain redacted secret fields and no user
	// password hashes. Preserve those node-local values from the newest runtime
	// while holding applyMu, then validate and persist in the same transaction.
	// This prevents a concurrent user/secret mutation from being lost between a
	// stale GET /api/config snapshot and this PUT.
	errKind := ""
	txnErr := func() error {
		a.srv.applyMu.Lock()
		defer a.srv.applyMu.Unlock()
		rt := a.srv.rt.Load()
		if rt == nil {
			errKind = "runtime"
			return errors.New("runtime unavailable")
		}
		cur := rt.cfg
		preserveAISecrets(cur, &c)
		preserveNotifySecrets(cur, &c)
		if c.HA.PeerToken == "" {
			c.HA.PeerToken = cur.HA.PeerToken
		}
		preserveHSMSecretRefs(cur, &c)
		// Users are managed only through the dedicated user endpoints; a general
		// config save can never replace password hashes with redacted UI data.
		c.Users = append([]UserConfig(nil), cur.Users...)
		if c.HA.Enabled && c.HA.SyncConfig && !a.haPeerTokenPinned {
			errKind = "validation"
			return errors.New("HA config sync requires a dedicated WAF_HA_PEER_TOKEN or -ha-peer-token before enabling sync")
		}
		if draft {
			if err := c.validateDraft(); err != nil {
				errKind = "validation"
				return err
			}
			// A draft is durable operator work-in-progress, not startup authority.
			// Persist it separately so a restart cannot silently promote an
			// unapplied Console draft to live traffic.
			if err := saveConfig(draftConfigPath(a.srv.configPath), c); err != nil {
				errKind = "persist"
				return err
			}
			return nil
		}
		if err := c.validate(); err != nil {
			errKind = "validation"
			return err
		}
		if err := a.srv.applyPersistedLocked(c, false); err != nil {
			errKind = "apply"
			return err
		}
		return nil
	}()
	if txnErr != nil {
		switch errKind {
		case "runtime":
			http.Error(w, txnErr.Error(), http.StatusServiceUnavailable)
		case "persist":
			http.Error(w, "not persisted: "+txnErr.Error(), http.StatusInternalServerError)
		default:
			http.Error(w, txnErr.Error(), http.StatusUnprocessableEntity)
		}
		return
	}

	if draft {
		a.audit.add(who(r).user, "config.draft_saved", fmt.Sprintf("%d sites, %d pools, %d nodes", len(c.Sites), len(c.Pools), len(c.Nodes)))
		applyErr := ""
		if err := c.validate(); err != nil {
			applyErr = err.Error()
		}
		resp := redactAISecrets(c)
		resp.HA.PeerToken = ""
		resp = redactHSMSecretRefs(resp)
		resp = redactNotifySecrets(resp)
		writeJSON(w, map[string]any{"config": resp, "draft": true, "apply_ready": applyErr == "", "apply_error": applyErr})
		return
	}

	draftCleanupWarning := ""
	if err := removeConfigFileDurable(draftConfigPath(a.srv.configPath)); err != nil {
		draftCleanupWarning = err.Error()
		a.log.Warn("applied config but stale draft cleanup failed", "err", err)
	}
	a.log.Info("config applied via admin API", "engine_mode", c.EngineMode, "sites", len(c.Sites))
	a.audit.add(who(r).user, "config.apply", fmt.Sprintf("%d sites, %d pools, %d policies", len(c.Sites), len(c.Pools), len(c.Policies)))
	resp := redactAISecrets(c)
	resp.HA.PeerToken = ""
	resp = redactHSMSecretRefs(resp)
	resp = redactNotifySecrets(resp)
	writeJSON(w, map[string]any{
		"config":                resp,
		"applied":               true,
		"restart_required":      a.srv.restartPending(c),
		"draft_cleanup_warning": draftCleanupWarning,
	})
}

func (a *adminServer) handleReload(w http.ResponseWriter, r *http.Request) {
	cfg := a.srv.rt.Load().cfg
	if err := a.srv.apply(cfg); err != nil {
		http.Error(w, "reload failed: "+err.Error(), http.StatusUnprocessableEntity)
		return
	}
	a.log.Info("rules reloaded via admin API", "rules", cfg.Rules)
	a.audit.add(who(r).user, "config.reload", "rules="+cfg.Rules)
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
	var vector any = map[string]any{"mode": "off", "native_available": false, "groups": []any{}}
	if rt := a.srv.rt.Load(); rt != nil && rt.vector != nil {
		vector = rt.vector.Status()
	}
	writeJSON(w, map[string]any{
		"current":             cur,
		"history":             hist,
		"cpus":                runtime.NumCPU(),
		"observations":        a.srv.observations.snapshot(),
		"match_logging":       a.srv.matchLogs.snapshot(),
		"ai_queue":            a.srv.ai.queueStats(),
		"vector_acceleration": vector,
	})
}

func (a *adminServer) handleVectorAcceleration(w http.ResponseWriter, _ *http.Request) {
	rt := a.srv.rt.Load()
	if rt == nil || rt.vector == nil {
		writeJSON(w, map[string]any{"mode": "off", "native_available": false, "groups": []any{}})
		return
	}
	writeJSON(w, rt.vector.Status())
}

func (a *adminServer) handleVectorAccelerationReset(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Site string `json:"site"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	in.Site = strings.TrimSpace(in.Site)
	if in.Site == "" {
		http.Error(w, "site is required", http.StatusBadRequest)
		return
	}
	rt := a.srv.rt.Load()
	if rt == nil || rt.vector == nil {
		http.Error(w, "VectorScan accelerator unavailable", http.StatusServiceUnavailable)
		return
	}
	n := rt.vector.ResetFailsafe(in.Site)
	writeJSON(w, map[string]any{"ok": true, "site": in.Site, "reset_groups": n})
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
	a.audit.add(who(r).user, "sitemap.crawl", fmt.Sprintf("site=%s max_pages=%d max_depth=%d", site.Name, opts.maxPages, opts.maxDepth))
	writeJSON(w, map[string]any{"started": true, "site": site.Name})
}

func (a *adminServer) handleSitemapClear(w http.ResponseWriter, r *http.Request) {
	site := r.URL.Query().Get("site")
	if site == "" {
		http.Error(w, "site query param required", http.StatusBadRequest)
		return
	}
	before := a.srv.maps.snapshot(site)
	if before.Crawl.Running {
		http.Error(w, "cannot clear site map while crawl is running", http.StatusConflict)
		return
	}
	a.srv.maps.clear(site)
	if err := a.srv.maps.save(a.srv.configPath); err != nil {
		a.srv.maps.restoreSnapshot(before)
		http.Error(w, "clear not persisted; in-memory map restored: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// Only clear dependent in-memory learning after the authoritative map
	// snapshot has durably accepted the destructive change.
	a.srv.signals.clear(site)
	a.srv.learn.clear(site)
	a.audit.add(who(r).user, "sitemap.clear", "site="+site)
	writeJSON(w, map[string]any{"ok": true})
}

func (a *adminServer) handleAPIOperations(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.srv == nil || a.srv.apiOps == nil {
		writeJSON(w, []any{})
		return
	}
	writeJSON(w, a.srv.apiOps.snapshot())
}
