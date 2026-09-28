#!/usr/bin/env python3
from pathlib import Path
import re
ROOT=Path(__file__).resolve().parents[2]
read=lambda p:(ROOT/p).read_text(encoding='utf-8')
admin=read('admin.go'); corr=read('correlation.go'); block=read('block_response.go'); ha=read('ha.go')
main=read('main.go'); listeners=read('listeners.go'); users=read('users.go'); login=read('admin_login_hardening.go')
notify=read('notify.go'); l7=read('l7_abuse.go'); debug=read('debug_bundle.go'); update=read('update.go'); tlsctl=read('tls_frontend_control.go')
ui=read('static/admin.html'); svc=read('waf-proxy.service'); tlssvc=read('waf-tls-frontend.service')
build=read('build.sh'); ci=read('.github/workflows/ci.yml'); tests=read('production_control_plane_hardening_test.go'); persist=read('api_security_persist.go'); identity=read('identity_api5.go'); positive=read('positive_schema_api4.go'); sample=read('config.sample.json'); sitemap=read('sitemap.go'); sitemap_persist=read('sitemap_persist.go')
checks=[]
def req(cond,msg):
    if not cond: raise SystemExit('PRODUCTION_CONTROL_PLANE_HARDENING_SOURCE_GATE_FAIL: '+msg)
    checks.append(msg)

# Request-path output encoding/correlation.
for t in ('validRequestCorrelationID','len(id) < 8','len(id) > 128',"c == '-'","c == '_'","c == '.'","c == ':'"):
    req(t in corr,'request correlation validation contains '+t)
req('html/template' in block,'block page uses html/template')
req('{{.Reason}}' in block and '{{.RequestID}}' in block,'block page interpolates only template fields')
req('Content-Security-Policy' in block and 'X-Frame-Options' in block and 'Cache-Control' in block,'block page emits anti-cache/browser hardening headers')

# JSON/admin secret privacy.
req('setJSONSecurityHeaders' in admin and 'Cache-Control", "no-store"' in admin and 'X-Content-Type-Options", "nosniff"' in admin,'Admin JSON responses are no-store/nosniff')
req('redactNotifySecrets' in notify and 'preserveNotifySecrets' in notify,'notification webhook secret has redact/preserve helpers')
req(admin.count('redactNotifySecrets') >= 3,'config/draft responses redact notification webhook secret')
req('maskedDebugClientIP(d.RemoteAddr)' in debug and 'maskedDebugClientIP(d.ResolvedClientIP)' in debug,'nested debug identity evidence masks client IPs')

req('DELETE /api/notifications/webhook' in admin and 'handleNotifyWebhookClear' in admin,'stored webhook secret has explicit operator clear route')
req('notification.webhook_clear' in admin,'webhook clear action is semantically audited')
req('nt_clear_webhook' in ui and '/api/notifications/webhook' in ui,'Console can explicitly clear stored webhook secret')
req('"l7_abuse"' in sample and '"cidr_policy"' in sample,'sample config exposes current L7/CIDR controls')

# Authentication abuse resistance.
for t in ('loginAttemptWindow','loginBlockDuration','loginAttemptLimit','loginAttemptMaxKey','newLoginAttemptLimiter','failure(keys','success(keys'):
    req(t in login,'login limiter contains '+t)
req(re.search(r'loginLimiter\s+\*loginAttemptLimiter', admin) is not None and 'newLoginAttemptLimiter()' in admin,'Admin server owns login limiter')
req('http.StatusTooManyRequests' in admin and 'Retry-After' in admin,'login throttle returns 429 with Retry-After')
req('pbkdf2MaxIter' in users and 'iter > pbkdf2MaxIter' in users,'PBKDF2 iteration count has upper bound')
req('len(want) < 16 || len(want) > 64' in users,'PBKDF2 derived-key length bounded')
req('len(salt) < 8 || len(salt) > 64' in users,'PBKDF2 salt length bounded')
req('createWithError' in users and 'rand.Read(b)' in users,'session token CSPRNG errors can fail closed')
req('a.sessions.createWithError(u.Username, u.Role)' in admin,'production login uses error-returning session creation')
req('overflow loginAttempt' in login and 'stateLocked' in login and 'storeLocked' in login,'login limiter saturation uses bounded shared overflow state')
req('delete(l.entries, k)' in login and 'len(l.entries) >= loginAttemptMaxKey' in login,'login limiter prunes expired keys without arbitrary saturation eviction')

# Viewer read-only backend authority and mutation audit.
role_routes={
'POST /api/reload':'roleOperator','POST /api/ai/unblock':'roleOperator','POST /api/ai/test':'roleOperator',
'POST /api/learn/clear':'roleReviewer','POST /api/notifications/read':'roleReviewer','POST /api/notifications/dismiss':'roleReviewer',
'POST /api/crawl':'roleOperator','POST /api/sitemap/clear':'roleReviewer'}
for route,role in role_routes.items():
    req(route in admin and role in admin[admin.index(route):admin.index(route)+180],f'{route} requires {role}')
req('mutationAuditWriter' in login and 'http.mutation' in admin,'successful authenticated mutations receive generic audit coverage')
for action in ('config.reload','ai.unblock','ai.connector_test','learner.clear','notification.read','notification.dismiss','sitemap.crawl','sitemap.clear'):
    req(action in admin,'semantic audit retained for '+action)

# Durable bounded local audit.
for t in ('configurePersistence','adminAuditRotateBytes','os.O_CREATE|os.O_WRONLY|os.O_APPEND','f.Sync()'):
    req(t in users,'durable audit implementation contains '+t)
req('WAF_ADMIN_AUDIT_FILE' in admin and 'admin-audit.jsonl' in admin,'Admin audit path has env + local fallback')
req('WAF_ADMIN_AUDIT_FILE=/var/lib/waf-proxy/admin-audit.jsonl' in svc,'systemd configures durable audit path')

# HA dedicated replication protocol and node-local separation.
req('X-WAF-Sync' not in ''.join([admin,ha]),'legacy caller-controlled HA sync header removed')
req('PUT /api/ha/peer-config' in admin and 'haPeerAuth' in admin,'HA config uses dedicated peer-only endpoint')
req('X-WAF-HA-Sync' in admin and 'X-WAF-HA-Sync' in ha,'dedicated HA protocol marker required/sent')
req('subtle.ConstantTimeCompare' in admin[admin.index('func (a *adminServer) haPeerAuth'):admin.index('// authRole')],'HA peer token checked constant-time')
for t in ('sharedConfigForPeer','out.HA = HAConfig{}','out.Users = nil','redactAISecrets','redactHSMSecretRefs','redactNotifySecrets'):
    req(t in ha,'HA shared payload strips node-local/secret state: '+t)
for t in ('mergePeerConfig','incoming.HA = local.HA','incoming.Users = local.Users','preserveAISecrets','preserveNotifySecrets','preserveHSMSecretRefs'):
    req(t in ha,'HA receiver restores local state: '+t)
req('u.Scheme != "https"' in ha and 'u.User != nil' in ha and 'u.RawQuery != ""' in ha,'HA peer URL restricted to HTTPS origin')
req('haSyncEnvelopeVersion = 1' in ha and 'DisallowUnknownFields' in admin[admin.index('func (a *adminServer) handleHAPeerConfig'):admin.index('func (a *adminServer) handleHAPeerConfig')+1400],'HA replication envelope versioned and strict')
req('syncRunning bool' in ha and 'syncPending *haSyncJob' in ha,'HA outbound sync has one in-flight request plus one bounded latest pending slot')
req('Latest wins' in ha and 'h.syncPending = &pending' in ha and 'runSyncLoop' in ha,'HA sync coalesces concurrent Apply commits instead of dropping the newest config')
req('haPeerToken       string' in admin and 'haPeerTokenPinned bool' in admin,'HA replication has a credential distinct from the break-glass admin token')
req('subtle.ConstantTimeCompare([]byte(got), []byte(a.haPeerToken))' in admin,'HA peer endpoint authenticates the dedicated replication token')
req('if !a.haPeerTokenPinned' in admin[admin.index('func (a *adminServer) haPeerAuth'):admin.index('func (a *adminServer) haPeerAuth')+900],'HA peer endpoint fails closed when dedicated replication token is absent')
req('c.HA.Enabled && c.HA.SyncConfig && !a.haPeerTokenPinned' in admin,'config draft/apply cannot enable HA sync without local dedicated replication token')
req('WAF_HA_PEER_TOKEN' in main and 'strings.TrimSpace(*haPeerToken) == ""' in main,'startup refuses persisted HA sync without stable dedicated replication token')
req('WAF_HA_PEER_TOKEN' in svc,'systemd documentation names the dedicated HA replication token')

# Transactional config persistence/listener ownership.
for t in ('type stagedConfigWrite','os.CreateTemp','f.Sync()','os.Rename','d.Sync()'):
    req(t in main,'durable staged config contains '+t)
req('func (s *server) applyPersisted' in main,'transactional applyPersisted exists')
req('applyMu             sync.Mutex' in main,'server has a single config-apply serialization mutex')
req('func (s *server) apply(cfg Config) error' in main and 's.applyMu.Lock()' in main[main.index('func (s *server) apply(cfg Config) error'):main.index('func (s *server) apply(cfg Config) error')+180],'non-persisted Apply is serialized')
req('func (s *server) applyPersisted' in main and 's.applyMu.Lock()' in main[main.index('func (s *server) applyPersisted'):main.index('func (s *server) applyPersisted')+220],'persisted Apply transaction is serialized')
req('a.srv.applyMu.Lock()' in admin[admin.index('func (a *adminServer) handlePutConfig'):admin.index('func (a *adminServer) handlePutConfig')+1800],'full config submit preserves secrets/users under the Apply serialization lock')
req('a.srv.applyPersistedLocked(c, false)' in admin,'full config Apply reuses the lock-assumed transaction without recursive locking')
req('func cloneConfig(cfg Config) (Config, error)' in main and 'json.Marshal(cfg)' in main and 'json.Unmarshal(b, &out)' in main,'partial mutations deep-clone Config instead of sharing live slice backing arrays')
req('func (s *server) mutatePersisted' in main and 'applyPersistedLocked(cfg, fromSync)' in main,'partial config mutations derive from the newest runtime inside the serialized transaction')
req(admin.count('mutatePersisted(') >= 5,'user/webhook/HA/page-policy partial mutations use serialized read-modify-write helper')
segment=main[main.index('func (s *server) applyPersisted'):main.index('func (s *server) buildRuntime')]
req(segment.index('stageConfig') < segment.index('s.applyEx') < segment.index('staged.commit'),'config staged before live apply and committed afterward')
req('s.applyEx(oldCfg, true)' in segment,'commit failure rolls live config back')
req(segment.index('staged.commit') < segment.index('s.ha.pushConfig'),'HA sync occurs only after durable commit')
req('func (m *listenerManager) reconcile(cfg Config) error' in listeners,'listener reconcile reports errors')
for t in ('func (m *listenerManager) prepare','net.Listen','restoreOldLocked','closePrepared','bind listener'):
    req(t in listeners,'listener transaction contains '+t)
req('s.listenMgr.reconcile(cfg); err != nil' in main,'runtime apply checks synchronous listener reconciliation')
req('publishAndWait' in tlsctl and 'timed out waiting for TLS frontend' in tlsctl,'TLS frontend ownership changes wait for manager acknowledgement')
req('oldFrontend && !newFrontend' in main and 'disable TLS frontend' in main,'frontend-to-Go ownership releases external public port before Go bind')
req('frontendRollback' in main and 'frontendRestoreAfterListeners' in main,'TLS frontend ownership rollback is ordered around listener restoration')
req('rt.listeners[tlsfront.InternalListenerKey(addr)]' in listeners,'public listener can bridge to prior internal runtime during ownership handoff')
req('rt.listeners[logicalAddr]' in listeners,'internal listener can bridge to prior public runtime during ownership handoff')
req('rt.listeners[tlsfront.InternalListenerKey(addr)]' in main[main.index('func (s *server) getCertificate'):],'TLS certificate lookup bridges external-frontend to Go handoff')
req('releaseForFrontendOwnership' in listeners and 'tls-frontend-ownership' in listeners,'Go TLS public listener is synchronously released before external frontend ownership')
req('Shutdown(ctx)' in listeners[listeners.index('func releaseForFrontendOwnership'):listeners.index('func gracefulClose')],'frontend ownership release closes the listener synchronously before returning')
req('json.NewEncoder(w).Encode' in listeners[listeners.index('func writeHealth'):],'health response uses JSON encoder instead of string concatenation')
req('draftConfigPath(a.srv.configPath)' in admin and 'saveConfig(draftConfigPath(a.srv.configPath), c)' in admin,'Save draft persists to a separate non-startup config file')
req('func loadDraftConfig' in main and 'draftInfo.ModTime().After(liveInfo.ModTime())' in main,'Console resumes only drafts newer than authoritative live config')
req('removeConfigFileDurable(draftConfigPath(a.srv.configPath))' in admin,'successful Apply cleans the auxiliary draft durably')
req('api("/api/config?draft=1")' in ui,'Console initial load resumes a durable draft without changing startup authority')

# Site-map destructive operations and autosave use serialized durable persistence.
req('persistMu sync.Mutex' in sitemap,'site-map persistence has a dedicated serialization mutex')
req('m.persistMu.Lock()' in sitemap_persist and 'defer m.persistMu.Unlock()' in sitemap_persist,'site-map save serializes periodic and operator-triggered writes')
for t in ('os.CreateTemp(dir, ".sitemap.json.tmp-*")','tmp.Sync()','os.Rename(tmpName, path)','d.Sync()'):
    req(t in sitemap_persist,'durable site-map persistence contains '+t)
req('restoreSnapshot(before)' in admin and 'clear not persisted; in-memory map restored' in admin,'destructive sitemap clear restores memory if persistence fails')
req('before.Crawl.Running' in admin and 'cannot clear site map while crawl is running' in admin,'sitemap clear refuses to race an active crawl')

# Durable API-security version compatibility fails closed on future versions.
req('func validateAPISecurityStateVersion' in persist and 'version != 0 && version != apiSecurityStateVersion' in persist,'shared API-security state version guard accepts only current/legacy')
for kind in ('api-operations','api-schema','api-contracts'):
    req(f'validateAPISecurityStateVersion("{kind}"' in persist,'durable state load checks version: '+kind)
req('validateAPISecurityStateVersion("api-identity"' in identity,'identity durable state checks version')
req('validateAPISecurityStateVersion("api-positive-schema"' in positive,'positive-schema durable state checks version')

# Bounded notification delivery.
for t in ('webhookQ       chan webhookJob','make(chan webhookJob, 256)','webhookWorker','webhookDropped','attempt < 3'):
    req(t in notify,'bounded webhook delivery contains '+t)
req('resp.StatusCode >= 200 && resp.StatusCode < 300' in notify,'webhook success requires HTTP 2xx')
req('webhookDeliveryStats' in notify and 'atomic.LoadUint64(&n.webhookDropped)' in notify,'bounded webhook queue exposes drop/depth telemetry')
req('notificationDedupeMax = 4096' in notify and 'len(n.dedupe) < notificationDedupeMax' in notify,'notification dedupe memory is explicitly bounded')
req('webhook_delivery' in admin and 'webhook_delivery||{}' in ui,'notification delivery telemetry is exposed to operators')
req('http.StatusTooManyRequests' in notify and 'resp.StatusCode >= 500' in notify,'webhook retries 429/5xx only')

# L7 saturation cannot be attacker-controlled fail-open.
req('overflow abuseClientState' in l7,'L7 shard has bounded overflow bucket')
req(l7.count('st = &shard.overflow') >= 2,'HTTP and TLS saturation use overflow bucket')
req('attacker-controlled fail-open' in l7,'source documents no fail-open saturation behavior')

# Filesystem/update/deployment controls.
req(admin.count('filepath.EvalSymlinks') >= 2,'admin file browser resolves root and requested symlinks')
req('directUpdateInstallEnabled' in update and 'WAF_UPDATE_INSTALL_DIR' in update,'direct updater requires explicit standalone directory')
req('filepath.Clean(filepath.Dir(exe)) == filepath.Clean(dir)' in update,'direct updater only enables when install directory is the running binary directory')
req(update.count('http.StatusConflict') >= 2,'install/rollback fail closed when direct update unsupported')
for h in ('Forwarded','X-Forwarded-For','X-Real-IP'):
    req(h in update,'localhost update guard rejects '+h)
req('DevicePolicy=closed' in svc and 'DeviceAllow=/dev/watchdog rw' in svc,'main systemd service explicitly allows watchdog device under closed policy')
req('DevicePolicy=closed' in tlssvc and 'DeviceAllow=/dev/qat_adf_ctl rw' in tlssvc and 'DeviceAllow=/dev/qat_dev_processes rw' in tlssvc,'TLS frontend systemd service explicitly allows QAT control devices')

# Console role awareness/operator lifecycle.
for t in ('function requiredCapability','function hasCapability','Clear learning state','/api/learn/clear'):
    req(t in ui,'Console capability/lifecycle contains '+t)
req('can_review' in ui and 'can_edit_config' in ui and 'can_manage_users' in ui,'Console consumes server role capabilities')
req('path==="/api/notifications/webhook"' in ui and 'path==="/api/security/contracts/import"' in ui,'operator-only webhook clear and contract import are role-gated in Console')
req('dedicated peer-only HTTPS' in ui and 'webhook URLs are never returned' in ui,'Console explains HA/webhook secret semantics')

req('install_supported' in ui and 'external package manager' in ui and 'updateInstallSupported' in ui,'Console reflects hardened updater install mode instead of implying in-process install')
req('const roleControlIDs' in ui and 'function applyRoleUI' in ui,'Console proactively disables fixed controls the active role cannot use')
req('data-required-capability="review"' in ui,'dynamic review controls declare required capability')

# Targeted tests committed.
for name in (
'TestProductionCorrelationIDValidation','TestProductionBlockResponseEscapesReasonAndRequestID',
'TestProductionHASharedConfigStripsNodeLocalAndSecrets','TestProductionHAMergeRestoresReceiverLocalState',
'TestProductionHARequiresHTTPSOrigin','TestProductionHAPeerAuthUsesDedicatedToken','TestProductionHASyncCoalescesLatestPendingConfig','TestProductionCloneConfigDoesNotShareNestedSlices','TestProductionLoginLimiterBlocksAndResets',
'TestProductionLoginLimiterSaturationDoesNotEvictBlockedKeys','TestProductionHealthJSONEncodesRole',
'TestProductionPBKDF2RejectsAbsurdIterationCount','TestProductionDirectUpdateDisabledWithoutExplicitDirectory',
'TestProductionUpdateLocalhostRejectsForwardedRequests','TestProductionL7SaturationUsesBoundedOverflowBucket',
'TestProductionStageConfigUsesPrivateAtomicFile','TestProductionAPISecurityStateVersionFailsClosed',
'TestProductionSitemapPersistenceUsesUniqueDurableTemp','TestProductionSitemapRestoreSnapshotAfterFailedClearPath',
'TestProductionDraftConfigIsSeparateFromStartupAuthority','TestProductionStaleDraftIgnoredAfterNewerLiveCommit',
'TestProductionNotificationDedupeStateIsBounded'):
    req(name in tests,'targeted regression test exists: '+name)

# Build/CI gate wiring.
req('test-production-control-plane-hardening-source.py' in build,'build.sh runs production hardening gate')
req('test-production-control-plane-hardening-source.py' in ci,'CI runs production hardening gate')
print(f'PRODUCTION_CONTROL_PLANE_HARDENING_SOURCE_GATE_PASS checks={len(checks)}')
