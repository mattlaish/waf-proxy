#!/usr/bin/env python3
from pathlib import Path
import sys

ROOT=Path(__file__).resolve().parents[2]
checks=[]
def req(cond,msg):
    if not cond:
        print(f"API72_SOURCE_GATE_FAIL: {msg}", file=sys.stderr); sys.exit(1)
    checks.append(msg)

src=(ROOT/'object_relationship_api72.go').read_text(encoding='utf-8')
tests=(ROOT/'object_relationship_api72_test.go').read_text(encoding='utf-8')
locator=(ROOT/'object_locator_api71.go').read_text(encoding='utf-8')
identity=(ROOT/'identity_api5.go').read_text(encoding='utf-8')
main=(ROOT/'main.go').read_text(encoding='utf-8')
admin=(ROOT/'admin.go').read_text(encoding='utf-8')
persist=(ROOT/'api_security_persist.go').read_text(encoding='utf-8')
ui=(ROOT/'static/admin.html').read_text(encoding='utf-8')
build=(ROOT/'build.sh').read_text(encoding='utf-8')
ci=(ROOT/'.github/workflows/ci.yml').read_text(encoding='utf-8')

# Core relationship model.
for token in ('type IdentityObjectRelationship struct','LocatorID','OperationID','Location','Field','IdentityKind','IdentityFingerprint','TenantFingerprint','ClientFingerprint','ObjectFingerprint','EvidenceLevel','ObservationCount','FirstSeen','LastSeen','ExpiresAt'):
    req(token in src, f'API-7.2 relationship carries required field {token}')
req('objectRelationshipSourceAPI5' in src and 'API5_VERIFIED_IDENTITY' in src, 'relationship source is API-5 verified identity')
req('objectRelationshipSourceAPI71' in src and 'API71_KEYED_OBJECT' in src, 'relationship source is API-7.1 keyed object evidence')
req('objectRelationshipEvidenceObserved' in src and 'objectRelationshipEvidenceRepeated' in src, 'relationship evidence levels are observational, not ownership verdicts')
req('It is not an ownership, tenant' in src, 'model comment states non-authoritative boundary')

# Identity authority and privacy.
req('verifiedAPIIdentity(r)' in src, 'only API-5 verified request identity is consumed')
req('apiIdentityContextKey' not in src, 'API-7.2 does not manufacture/replace API-5 verification context')
req('id.Subject' in src and 'id.ClientID' in src and 'id.TenantID' in src, 'only fields from verified API-5 context feed pseudonyms')
req('hmac.New(sha256.New, key)' in src, 'identity evidence uses keyed HMAC-SHA256')
req('api-object-relationship.key' in src and 'f.Chmod(0o600)' in src, 'relationship pseudonym key is dedicated and mode 0600')
req('h.Sum(nil)[:20]' in src, 'identity pseudonym is a bounded keyed digest')
req('Issuer' not in ''.join(line for line in src.splitlines() if 'json:' in line), 'durable relationship JSON has no raw issuer field')
req('Subject ' not in ''.join(line for line in src.splitlines() if 'json:' in line), 'durable relationship JSON has no raw subject field')
req('TenantID' not in ''.join(line for line in src.splitlines() if 'json:' in line), 'durable relationship JSON has no raw tenant claim field')
req('ClientID' not in ''.join(line for line in src.splitlines() if 'json:' in line), 'durable relationship JSON has no raw client_id field')
req('Authorization' not in src and 'Cookie(' not in src and 'Header.Get("X-Tenant' not in src and 'Header.Get("X-Owner' not in src, 'caller-supplied auth/ownership metadata is absent from relationship authority')
req('TenantFingerprint' in src and 'ClientFingerprint' in src, 'verified tenant/client context is pseudonymous when retained')
req('strings.Join(values, "\\x00")' in src and 'kind' in src, 'identity digest domain separates components and identity kind')

# Object authority and raw-value handling.
req('activeLocator(ev.OperationID, sample.Location, sample.Path)' in src, 'only active API-7.1 locator positions can create relations')
req('s.locators.digestValue' in src, 'object evidence reuses API-7.1 keyed object fingerprint')
req('func (s *objectLocatorStore) activeLocator' in locator, 'API-7.1 exposes bounded active-locator lookup for API-7.2')
req('objectLocatorOverrideSuppress' in locator and 'activeLocator' in locator, 'suppressed locators are excluded from API-7.2')
req('sample.objectValue == ""' in src, 'missing transient object value cannot create a relationship')
req('ObjectValue' not in src, 'API-7.2 durable model has no raw ObjectValue field')
req('RawQuery' in src and 'BodyPrefix' in src, 'raw selector material is transient queue input only')
req('json:"raw_query"' not in src and 'json:"body_prefix"' not in src, 'transient raw selector material is not durable JSON')
req('sample.Location != "path" && sample.Location != "query" && sample.Location != "body"' in src, 'relationship processing is limited to path/query/body selector locations')
req('strings.HasPrefix(strings.ToLower(sample.Path), "variables.")' in src, 'GraphQL variables remain deferred to API-8')
req('normalizeAPIOperationPath(path)' in src and 'apiOperationID(site, r.Method' in src, 'relationship event uses API-1 normalized operation identity')
req('objectRelationshipID(identityFingerprint, locatorID, objectFingerprint string)' in src, 'relationship identity is derived from pseudonymous identity+locator+object evidence')

# Bounded async hot path and lifecycle.
req('objectRelationshipQueueCapacity' in src, 'relationship queue is bounded')
req('objectRelationshipMaxRelations' in src, 'global relationship cardinality is bounded')
req('objectRelationshipMaxPerIdentity' in src, 'per-identity relationship cardinality is bounded')
req('objectRelationshipMaxPerObject' in src, 'per-object relationship identity fanout is bounded')
req('objectRelationshipDefaultTTL' in src, 'relationship evidence has bounded TTL')
req('objectRelationshipMaxPathBytes' in src and 'objectRelationshipMaxQueryBytes' in src, 'queued request locator material is size bounded')
req('select {' in src and 'case s.queue <- ev:' in src and 'default:' in src and 's.dropped.Add(1)' in src, 'request path enqueue is non-blocking and counts drops')
req('append([]byte(nil), prefix...)' in src, 'bounded body prefix queue input owns an immutable copy')
req('s.pruneLocked(ev.At)' in src and '!v.ExpiresAt.After(now)' in src, 'expired relationship state is pruned')
req('s.countIdentityLocked(ev.IdentityFingerprint) >= objectRelationshipMaxPerIdentity' in src, 'per-identity cap is enforced on insertion')
req('s.countObjectLocked(locator.ID, objectFingerprint) >= objectRelationshipMaxPerObject' in src, 'per-object cap is enforced on insertion')
req('len(s.relationships) >= objectRelationshipMaxRelations' in src, 'global cap is enforced on insertion')
req('atomic.Bool' in src and 'atomic.Uint64' in src, 'async lifecycle/drop counters are concurrency-safe')
req('sync.RWMutex' in src, 'relationship durable state is lock protected')

# Persistence and restart validation.
req('api-object-relationships.json' in src, 'relationship state has a dedicated persistence file')
req('objectRelationshipStateVersion' in src, 'relationship durable schema is explicitly versioned')
req('state.Version != 0 && state.Version != objectRelationshipStateVersion' in src, 'unknown relationship state versions fail closed')
req('validObjectRelationship(v, now)' in src, 'restored relationship rows are revalidated')
req('len(s.relationships) >= objectRelationshipMaxRelations' in src and 'countIdentityLocked' in src and 'countObjectLocked' in src, 'restart restoration reapplies cardinality limits')
req('objectRelationships.save(configPath)' in persist, 'API autosave/final flush persists API-7.2 state')
req('s.objectRelationships.load(*configPath)' in main, 'API-7.2 state restores before traffic')
req('s.objectRelationships.start()' in main, 'API-7.2 async worker starts before listeners')
req('s.objectRelationships.stopAndDrain()' in main, 'API-7.2 accepted observations drain before final persistence')
req('objectRelationships *objectRelationshipStore' in main, 'server owns API-7.2 relationship store')
req('s.objectRelationships.locators = s.objectLocators' in main, 'API-7.2 is bound to canonical API-7.1 locator store')

# Middleware ordering and authority boundary.
req('relationshipHandler = s.objectRelationships.wrap(siteName, relationshipHandler)' in main, 'API-7.2 relationship wrapper is installed in the API request chain')
req('graphqlHandler := relationshipHandler' in main and 'requestBodyPrefixWrap(captureForAI, cfg.PassiveDiscoveryEnabled, true, graphqlHandler)' in main, 'bounded body capture still feeds API-7.2 through the later API-8 wrapper without a second body read')
req(main.index('handler = s.identity.wrap(siteName, handler)') > main.index('relationshipHandler = s.objectRelationships.wrap(siteName, relationshipHandler)'), 'API-5 identity is outer middleware and therefore verifies before API-7.2 observes')
req('BLOCK' not in src and 'DENY' not in src and 'BOLA_CANDIDATE' not in src and 'OWNER_MATCH' not in src and 'OWNER_MISMATCH' not in src, 'API-7.2 introduces no BOLA/ownership/enforcement verdict')
req('StatusForbidden' not in src and 'StatusUnauthorized' not in src, 'API-7.2 has no request blocking response primitive')
req('OpenAI' not in src, 'OpenAI is absent from API-7.2 relationship authority')
req('ENFORCE' not in src, 'API-7.2 has no enforcement mode')

# Reviewer visibility API and audit.
for route in ('GET /api/security/object-relationships','GET /api/security/object-relationships/status'):
    req(route in admin, f'admin route exists: {route}')
req('authRole(roleReviewer, a.handleObjectRelationships)' in admin and 'authRole(roleReviewer, a.handleObjectRelationshipStatus)' in admin, 'relationship evidence/status require Reviewer-or-higher')
req('api_object_relationship.view' in src and 'api_object_relationship.status_view' in src, 'relationship evidence/status reads are audited')
req('POST /api/security/object-relationships' not in admin and 'DELETE /api/security/object-relationships' not in admin, 'API-7.2 exposes no operator ownership mutation endpoint')

# Console and build/CI.
for label in ('API-7.2 identity/object relationship','Verified identity ↔ keyed object evidence','Caller-supplied tenant/owner headers and cookies are not authority','API-7.2 makes no ownership, tenant-boundary, BOLA, or blocking decision.'):
    req(label in ui, f'Admin Console states API-7.2 boundary: {label}')
req('/api/security/object-relationships' in ui and '/api/security/object-relationships/status' in ui, 'Admin Console reads API-7.2 evidence/status endpoints')
req('identity_fingerprint' in ui and 'object_fingerprint' in ui and 'tenant_fingerprint' in ui, 'Admin Console displays pseudonymous relationship evidence only')
req('test-api71-source.py' in build and 'test-api71-source.py' in ci, 'API-7.1 source gate remains in build/CI')
req('test-api72-source.py' in build, 'local build runs API-7.2 source gate')
req('test-api72-source.py' in ci, 'CI runs API-7.2 source gate')

for name in (
    'TestAPI72VerifiedIdentityAndKeyedObjectCreateRelationship',
    'TestAPI72RejectsUnverifiedAndCallerSuppliedOwnershipMetadata',
    'TestAPI72SubjectAndClientIdentityPseudonymsAreSeparated',
    'TestAPI72NormalizedOperationAndObjectFingerprintAvoidRawCardinality',
    'TestAPI72SuppressedLocatorCannotCreateRelationship',
    'TestAPI72RelationshipTTLAndCardinalityAreBounded',
    'TestAPI72PersistenceRestoresKeyedEvidenceWithoutRawClaimsOrObjects',
    'TestAPI72RepeatedObservationIsEvidenceNotOwnershipVerdict',
    'TestAPI72ConcurrentRelationshipUpdatesRemainBounded',
):
    req(name in tests, f'targeted API-7.2 test exists: {name}')

# Verify API-5 context remains cryptographically verified by its own layer.
req('JWT signature verification failed' in identity and 'VerifiedAPIIdentity exists only in request memory after cryptographic' in identity, 'API-7.2 depends on existing cryptographically verified API-5 identity context')

print(f"API72_SOURCE_GATE_PASS checks={len(checks)}")
