#!/usr/bin/env python3
from pathlib import Path

root = Path(__file__).resolve().parents[2]
src = (root / 'bola_detection_api73.go').read_text()
rel = (root / 'object_relationship_api72.go').read_text()
main = (root / 'main.go').read_text()
admin = (root / 'admin.go').read_text()
persist = (root / 'api_security_persist.go').read_text()
tests = (root / 'bola_detection_api73_test.go').read_text()
api5 = (root / 'api_identity_api5.go').read_text() if (root / 'api_identity_api5.go').exists() else ''
build = (root / 'build.sh').read_text()
ci = (root / '.github/workflows/ci.yml').read_text()

checks=[]
def req(cond, msg):
    if not cond:
        raise SystemExit('API73_SOURCE_GATE_FAIL: '+msg)
    checks.append(msg)

# Model and explicit non-authoritative semantics.
for token in ('type BOLACandidate struct', 'IdentityFingerprint', 'TenantFingerprint', 'ObjectFingerprint', 'BaselineIdentities', 'BaselineTenants', 'BaselineObservations', 'RecentIdentityObjects', 'Confidence', 'EvidenceCount', 'FirstSeen', 'LastSeen', 'ExpiresAt'):
    req(token in src, f'candidate model includes {token}')
req('DETECT-only evidence' in src, 'candidate comment declares detect-only evidence')
req('not an\n// ownership, tenant-boundary, authorization, or enforcement verdict' in src, 'candidate comment rejects ownership/authorization verdict semantics')
for kind in ('IDENTITY_OBJECT_DIVERGENCE', 'TENANT_OBJECT_DIVERGENCE', 'OBJECT_ENUMERATION'):
    req(kind in src, f'candidate type exists: {kind}')
req('MEDIUM' in src and 'HIGH' in src, 'candidate confidence is evidence confidence only')

# Mature/bounded detection semantics.
req('bolaCandidateMinBaselineObs' in src and '= 2' in src, 'foreign-object divergence requires repeated baseline evidence')
req('!ctx.CurrentRelationship && ctx.ForeignRepeated' in src, 'identity divergence requires a new relation against repeated foreign baseline')
req('ctx.BaselineObservations >= bolaCandidateMinBaselineObs' in src, 'identity/tenant divergence checks baseline observation threshold')
req('!ctx.CurrentRelationship && ctx.ForeignTenantRepeated' in src, 'tenant divergence requires new relation against repeated foreign tenant evidence')
req('ev.TenantFingerprint != ""' in src, 'tenant divergence requires verified pseudonymous tenant context')
req('ctx.RecentIdentityObjects >= bolaCandidateEnumerationMinObjs-1' in src, 'enumeration requires bounded recent fanout threshold')
req('bolaCandidateEnumerationWindow  = 10 * time.Minute' in src, 'enumeration uses bounded recent window')
req('bolaCandidateEnumerationMinObjs = 20' in src, 'enumeration does not fire on low object counts')
req('Novel object' not in src, 'implementation does not define a novel-object-alone detector')
req('Shared-resource applications may legitimately trigger it' in src, 'identity divergence explicitly acknowledges shared-resource ambiguity')

# Privacy and trusted evidence chain.
req('objectRelationshipObservation' in src, 'detector consumes API-7.2 pseudonymous observation type')
req('ObjectLocator' in src, 'detector consumes API-7.1 locator identity')
req('objectRelationshipSourceAPI5' in src and 'objectRelationshipSourceAPI71' in src and 'API72_RELATIONSHIP_BASELINE' in src, 'candidate source chain is API5/API71/API72 only')
req('bolaContext(locator.ID, objectFingerprint, ev.IdentityFingerprint, ev.TenantFingerprint' in src, 'candidate context is keyed by pseudonymous identity/object evidence')
for forbidden in ('Authorization', 'Cookie(', 'Header.Get("X-Tenant', 'Header.Get("X-Owner', 'RawQuery', 'BodyPrefix', 'sample.objectValue', 'id.Subject', 'id.ClientID', 'id.TenantID'):
    req(forbidden not in src, f'API-7.3 durable detector does not consume raw/caller authority: {forbidden}')
req('json:"identity_fingerprint"' in src and 'json:"object_fingerprint"' in src, 'durable candidate stores only fingerprints')
req('json:"tenant_fingerprint,omitempty"' in src, 'tenant evidence remains pseudonymous')
req('RawObject' not in src and 'ObjectValue' not in src, 'candidate model has no raw object value field')

# Relationship context is historical evidence, not owner truth.
req('func (s *objectRelationshipStore) bolaContext' in src, 'relationship store exposes bounded candidate context')
req('rel.ObservationCount >= bolaCandidateMinBaselineObs' in src, 'foreign baseline must be repeated')
req('rel.LastSeen.After(now.Add(-bolaCandidateEnumerationWindow))' in src, 'enumeration fanout ignores stale relation rows')
req('ctx.CurrentRelationship = true' in src, 'known same identity/object relation suppresses divergence candidate')
req('ctx.ForeignRepeated = true' in src and 'ctx.ForeignTenantRepeated = true' in src, 'context separates identity and tenant divergence evidence')
req('map[string]struct{}{}' in src, 'context cardinalities use unique bounded relationship sets')

# Bounded candidate state and restart validation.
req('bolaCandidateMaxCandidates      = 4096' in src, 'global candidate cap is explicit')
req('bolaCandidateMaxPerIdentity     = 256' in src, 'per-identity candidate cap is explicit')
req('bolaCandidateMaxPerObject       = 128' in src, 'per-object candidate cap is explicit')
req('bolaCandidateDefaultTTL         = 7 * 24 * time.Hour' in src, 'candidate evidence TTL is explicit')
req('len(s.candidates) >= bolaCandidateMaxCandidates' in src, 'global candidate cap enforced')
req('s.countIdentityLocked(ev.IdentityFingerprint) >= bolaCandidateMaxPerIdentity' in src, 'per-identity candidate cap enforced')
req('s.countObjectLocked(locator.ID, objectFingerprint) >= bolaCandidateMaxPerObject' in src, 'per-object candidate cap enforced')
req('s.pruneLocked(ev.At)' in src and '!row.ExpiresAt.After(now)' in src, 'expired candidates are pruned')
req('sync.RWMutex' in src, 'candidate state is concurrency protected')
req('bolaCandidateStateVersion' in src, 'candidate persistence is versioned')
req('api-bola-candidates.json' in src, 'candidate state has dedicated persistence file')
req('state.Version != 0 && state.Version != bolaCandidateStateVersion' in src, 'unknown candidate state versions fail closed')
req('validBOLACandidate(row, now)' in src, 'restored candidates are revalidated')
req('len(row.ID) != 40' in src and 'len(row.IdentityFingerprint) != 40' in src and 'len(row.ObjectFingerprint) != 40' in src, 'restored keyed digests are structurally validated')
req('hex.DecodeString(digest)' in src, 'restored keyed digests must be valid hex')

# Runtime placement: API-7.2 background worker before relationship merge.
req('detector      *bolaDetectionStore' in rel, 'API-7.2 store holds API-7.3 detector hook')
req('s.detector.observe(ev, locator, objectFingerprint, s)' in rel, 'API-7.2 background processor invokes detector')
req(rel.index('s.detector.observe(ev, locator, objectFingerprint, s)') < rel.index('s.mergeWithSource(ev, locator, objectFingerprint, objectRelationshipSourceAPI71)'), 'detection evaluates historical state before current relation is merged')
req('bolaCandidates      *bolaDetectionStore' in main, 'server owns API-7.3 store')
req('bolaCandidates:      newBOLADetectionStore()' in main, 'server initializes API-7.3 store')
req('s.objectRelationships.detector = s.bolaCandidates' in main, 'canonical API-7.2 worker is wired to API-7.3 detector')
req('s.bolaCandidates.load(*configPath)' in main, 'API-7.3 state restores before traffic')
req('bolaCandidates.save(configPath)' in persist, 'API autosave persists API-7.3 evidence')
req('s.objectRelationships.start()' in main, 'API-7.2 async worker remains authoritative processing plane')

# No enforcement/ownership authority in API-7.3.
for forbidden in ('StatusForbidden', 'StatusUnauthorized', 'ServeHTTP(', 'http.Error(', 'BLOCK', 'DENY', 'ENFORCE', 'OWNER_MATCH', 'OWNER_MISMATCH', 'OWNED_BY', 'ALLOW'):
    req(forbidden not in src, f'API-7.3 has no enforcement/owner verdict primitive: {forbidden}')
req('OpenAI' not in src, 'OpenAI is absent from BOLA detection authority')
req('BOLA_CANDIDATE' not in src, 'API-7.3 uses evidence objects rather than request-path verdict token')
req('DetectionOnly: true' in src, 'status explicitly reports detection-only behavior')

# Read-only Reviewer API + audit; policy/mutation remains API-7.4.
for route in ('GET /api/security/bola/candidates', 'GET /api/security/bola/status'):
    req(route in admin, f'admin route exists: {route}')
req('authRole(roleReviewer, a.handleBOLACandidates)' in admin and 'authRole(roleReviewer, a.handleBOLAStatus)' in admin, 'BOLA evidence/status require Reviewer-or-higher')
req('api_bola.candidates_view' in src and 'api_bola.status_view' in src, 'BOLA read APIs are audited')
# API-7.4 may add BOLA mutation routes to the shared admin router, but the
# API-7.3 detector source itself must remain read-only/evidence-only.
for handler in ('handleBOLAPolicyUpsert', 'handleBOLAPolicyDelete', 'handleBOLAEvidenceWorkflow'):
    req(handler not in src, f'API-7.3 detector source has no mutation handler: {handler}')

# Tests and build/CI retention.
for name in (
    'TestAPI73IdentityObjectDivergenceRequiresRepeatedBaseline',
    'TestAPI73NovelObjectAloneIsNotBOLA',
    'TestAPI73TenantDivergenceUsesVerifiedPseudonymousTenantEvidence',
    'TestAPI73SameIdentityRelationshipDoesNotCreateDivergence',
    'TestAPI73EnumerationRequiresBoundedRecentFanout',
    'TestAPI73SuppressedLocatorCannotFeedDetector',
    'TestAPI73PersistenceTTLAndPrivacy',
    'TestAPI73CandidateCardinalityIsBounded',
    'TestAPI73ConcurrentDetectionRemainsBounded',
):
    req(name in tests, f'targeted API-7.3 test exists: {name}')
req('test-api71-source.py' in build and 'test-api72-source.py' in build, 'prior API-7 source gates remain in build')
req('test-api71-source.py' in ci and 'test-api72-source.py' in ci, 'prior API-7 source gates remain in CI')
req('test-api73-source.py' in build, 'local build runs API-7.3 source gate')
req('test-api73-source.py' in ci, 'CI runs API-7.3 source gate')

# Existing cryptographic identity authority remains API-5, not API-7.3.
if api5:
    req('JWT signature verification failed' in api5, 'API-5 cryptographic verification remains present')

print(f'API73_SOURCE_GATE_PASS checks={len(checks)}')
