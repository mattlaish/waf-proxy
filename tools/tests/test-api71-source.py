#!/usr/bin/env python3
from pathlib import Path
import sys

ROOT=Path(__file__).resolve().parents[2]
checks=[]
def req(cond,msg):
    if not cond:
        print(f"API71_SOURCE_GATE_FAIL: {msg}", file=sys.stderr); sys.exit(1)
    checks.append(msg)

src=(ROOT/'object_locator_api71.go').read_text(encoding='utf-8')
tests=(ROOT/'object_locator_api71_test.go').read_text(encoding='utf-8')
schema=(ROOT/'schema_api2.go').read_text(encoding='utf-8')
obs=(ROOT/'observations.go').read_text(encoding='utf-8')
main=(ROOT/'main.go').read_text(encoding='utf-8')
admin=(ROOT/'admin.go').read_text(encoding='utf-8')
contract=(ROOT/'api3_contract.go').read_text(encoding='utf-8')
persist=(ROOT/'api_security_persist.go').read_text(encoding='utf-8')
ui=(ROOT/'static/admin.html').read_text(encoding='utf-8')
build=(ROOT/'build.sh').read_text(encoding='utf-8')
ci=(ROOT/'.github/workflows/ci.yml').read_text(encoding='utf-8')

# Core model and discovery-source boundary.
for token in ('type ObjectLocator struct','OperationID','Location','Field','SchemaType','SemanticName','Confidence','FirstSeen','LastSeen'):
    req(token in src, f'ObjectLocator carries required field {token}')
req('ValueFingerprints []ObjectValueFingerprint' in src, 'locator retains only bounded keyed value evidence')
req('objectLocatorSourceAPI1' in src and 'API1_NORMALIZED_PATH' in src, 'API-1 normalized path discovery source exists')
req('objectLocatorSourceAPI2' in src and 'API2_TYPED_SCHEMA' in src, 'API-2 typed-schema discovery source exists')
req('objectLocatorSourceAPI3' in src and 'API3_OPENAPI_CONTRACT' in src, 'API-3 matched OpenAPI discovery source exists')
req('objectLocatorSourceOperator' in src and 'OPERATOR' in src, 'operator discovery/configuration source exists')
req('op.PathParameters' in src and 'p.Kind != "identifier" && p.Kind != "opaque_token"' in src, 'API-1 discovery uses normalized dynamic path parameters')
req('sample.Location != "path" && sample.Location != "query" && sample.Location != "body"' in src, 'runtime discovery is limited to path/query/body locations')
req('strings.HasPrefix(strings.ToLower(sample.Path), "variables.")' in src, 'GraphQL variables are explicitly deferred from API-7.1 runtime discovery')
req('strings.HasPrefix(strings.ToLower(f.Path), "variables.")' in src, 'GraphQL variables are explicitly deferred from API-7.1 contract discovery')
req('binding.Status != "MATCHED"' in src and 'binding.APIOperationID' in src, 'API-3 source requires matched normalized API-1 operation binding')
req('p.In != "path" && p.In != "query"' in src, 'OpenAPI header/cookie parameters cannot become API-7.1 object locators')
req('objectSemanticName' in src and 'strings.HasSuffix(canonical, "id")' in src, 'query/body discovery requires bounded identifier semantics')
req('objectLocatorTypeEligible' in src, 'query/body discovery uses typed evidence')

# Privacy, keying, and cardinality.
req('objectValue string // transient API-7.1 input; never persisted' in schema, 'raw object sample is unexported transient-only state')
req('hmac.New(sha256.New, key)' in src and 'digestValue' in src, 'object values are keyed with HMAC-SHA256')
req('h.Sum(nil)[:20]' in src, 'persisted object value digest is bounded')
req('api-object-locator.key' in src and 'f.Chmod(0o600)' in src, 'separate fingerprint key is persisted mode 0600')
req('api-object-locators.json' in src, 'object locator state has dedicated durable file')
req('objectLocatorMaxLocators      = 4096' in src, 'global locator cardinality is bounded')
req('objectLocatorMaxPerOperation  = 32' in src, 'per-operation locator cardinality is bounded')
req('objectLocatorMaxFingerprints  = 8' in src, 'per-locator value evidence cardinality is bounded')
req('objectLocatorMaxOverrides     = 1024' in src, 'operator override cardinality is bounded')
req('objectLocatorDefaultTTL       = 30 * 24 * time.Hour' in src, 'learned locator state ages out')
req('s.pruneLocked(at)' in src and '!v.ExpiresAt.After(now)' in src, 'TTL pruning is active')
req('len(v.ValueFingerprints) < objectLocatorMaxFingerprints' in src, 'value fingerprint append is bounded')
req('len(s.locators) >= objectLocatorMaxLocators' in src and 's.countOperationLocked(operationID) >= objectLocatorMaxPerOperation' in src, 'locator insertion enforces both caps')
req('isSensitiveFieldName(path)' in src, 'sensitive field names cannot become heuristic locators')
req('ObjectValue' not in schema, 'no exported raw ObjectValue field exists in schema samples')

# Async integration and persistence/restart.
req('objectLocators *objectLocatorStore' in obs, 'bounded observation plane owns API-7.1 store integration')
req('p.objectLocators.noteObservation(op, samples)' in obs, 'background observation consumer performs locator discovery')
req('s.objectLocators.noteObservation(op, samples)' in main, 'minimal-server fallback retains locator discovery')
req('*objectLocatorStore' in main and 'newObjectLocatorStore()' in main, 'server initializes API-7.1 store')
req('s.objectLocators.load(*configPath)' in main, 'API-7.1 state restores before traffic')
req('s.objectLocators.refreshContractSources(s.contracts)' in main, 'declared contract locators are rebuilt on startup')
req('s.observations.objectLocators = s.objectLocators' in main, 'production observation plane receives API-7.1 store')
req('objectLocators *objectLocatorStore' in persist and 'objectLocators.save(configPath)' in persist, 'API autosave/final flush includes API-7.1 state')
req('a.srv.objectLocators.refreshContractSources(a.srv.contracts)' in contract, 'OpenAPI matching refreshes API-7.1 declared sources')
req('state.Version != 0 && state.Version != objectLocatorStateVersion' in src, 'durable object-locator state is versioned fail-closed')
req('validObjectLocator(v, now)' in src and 'validObjectLocatorOverride(v)' in src, 'restart restore revalidates durable locator/control state')

# Operator semantics, RBAC, audit, and no authorization authority.
req('objectLocatorOverrideInclude' in src and 'objectLocatorOverrideSuppress' in src, 'operator INCLUDE/SUPPRESS semantics exist')
req('maximum object locator overrides reached' in src, 'operator configuration cap is enforced')
req('removeOperatorSourceLocked' in src, 'override replacement/deletion reconciles operator source cleanly')
for route in (
    'GET /api/security/object-locators',
    'GET /api/security/object-locators/overrides',
    'POST /api/security/object-locators/overrides',
    'DELETE /api/security/object-locators/overrides/{override_id}',
):
    req(route in admin, f'admin route exists: {route}')
req('authRole(roleReviewer, a.handleObjectLocators)' in admin, 'locator evidence read requires Reviewer-or-higher')
req('authRole(roleReviewer, a.handleObjectLocatorOverrideUpsert)' in admin and 'authRole(roleReviewer, a.handleObjectLocatorOverrideDelete)' in admin, 'locator mutations require Reviewer-or-higher')
for action in ('api_object_locator.view','api_object_locator.override_view','api_object_locator.override_upsert','api_object_locator.override_delete'):
    req(action in src, f'API-7.1 admin action is audited: {action}')
req('persistObjectLocatorMutation' in src and 'a.srv.objectLocators.save(a.srv.configPath)' in src, 'operator mutations persist immediately')
req('BOLA_CANDIDATE' not in src and 'OWNER_MATCH' not in src and 'OWNER_MISMATCH' not in src, 'API-7.1 introduces no BOLA or ownership verdict')
req('StatusForbidden' not in src and 'StatusUnauthorized' not in src, 'API-7.1 has no request authorization/blocking response primitive')
req('func (s *objectLocatorStore) wrap' not in src, 'API-7.1 does not install a request-path enforcement wrapper')
req('OpenAI' not in src, 'OpenAI is absent from API-7.1 locator authority')

# Console and build/CI retention.
for label in ('API-7.1 object locator discovery','Normalized object selector inventory','Operator include / suppress override','Raw object values are never retained'):
    req(label in ui, f'Admin Console exposes/document boundary: {label}')
req('/api/security/object-locators' in ui and '/api/security/object-locators/overrides' in ui, 'Admin Console uses API-7.1 read/config endpoints')
req('API-7.1 makes no ownership, tenant-boundary, BOLA, or blocking decision.' in ui, 'Admin Console states non-authority boundary')
req('test-api71-source.py' in build, 'local build retains API-7.1 source gate')
req('test-api71-source.py' in ci, 'CI retains API-7.1 source gate')

for name in (
    'TestAPI71NormalizedPathLocatorDoesNotExpandWithObjectValue',
    'TestAPI71TypedQueryBodyDiscoveryAndGraphQLVariablesDeferred',
    'TestAPI71SelfReportedTenantHeadersAreNeverLocatorAuthority',
    'TestAPI71MatchedOpenAPIContractSeedsDeclaredLocators',
    'TestAPI71OperatorIncludeSuppressDeleteAndPersistence',
    'TestAPI71CardinalityAndTTLRemainBounded',
    'TestAPI71KeyedFingerprintStableAcrossRestartWithoutRawValue',
    'TestAPI71ConcurrentObservationAndSnapshotStayBounded',
    'TestAPI71NoOwnershipBOLAVerdictOrEnforcementPrimitive',
):
    req(name in tests, f'targeted API-7.1 test exists: {name}')

print(f"API71_SOURCE_GATE_PASS checks={len(checks)}")
