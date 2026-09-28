#!/usr/bin/env python3
from pathlib import Path
import re
ROOT=Path(__file__).resolve().parents[2]
src=(ROOT/'graphql_api8.go').read_text(encoding='utf-8')
tests=(ROOT/'graphql_api8_test.go').read_text(encoding='utf-8')
main=(ROOT/'main.go').read_text(encoding='utf-8')
admin=(ROOT/'admin.go').read_text(encoding='utf-8')
persist=(ROOT/'api_security_persist.go').read_text(encoding='utf-8')
loc=(ROOT/'object_locator_api71.go').read_text(encoding='utf-8')
rel=(ROOT/'object_relationship_api72.go').read_text(encoding='utf-8')
bola=(ROOT/'bola_detection_api73.go').read_text(encoding='utf-8')
ui=(ROOT/'static/admin.html').read_text(encoding='utf-8')
build=(ROOT/'build.sh').read_text(encoding='utf-8')
ci=(ROOT/'.github/workflows/ci.yml').read_text(encoding='utf-8')
checks=[]
def req(cond,msg):
    if not cond: raise SystemExit('API8_SOURCE_GATE_FAIL: '+msg)
    checks.append(msg)

# Operation discovery and privacy.
for token in ('type GraphQLOperation struct','EndpointOperationID','OperationType','OperationName','DocumentFingerprint','Depth','Complexity','Fields','Variables','Introspection','PersistedQueryHash','ObservationCount','FirstSeen','LastSeen','ExpiresAt'):
    req(token in src,f'GraphQL operation model includes {token}')
op_block=src[src.index('type GraphQLOperation struct'):src.index('// GraphQLPersistedProfile')]
for forbidden in ('RawQuery','QueryText','LiteralValue','RawVariable','JWT','Authorization','Cookie'):
    req(forbidden not in op_block,f'operation durable model excludes {forbidden}')
req('Literal values are discarded by the parser' in src,'source documents literal-value privacy boundary')
req('graphqlMaxOperations         = 4096' in src,'operation cardinality bounded')
req('graphqlOperationTTL          = 30 * 24 * time.Hour' in src,'operation TTL bounded')
req('graphqlMaxQueryBytes         = 64 << 10' in src,'GraphQL document bytes bounded')
req('graphqlMaxTokens             = 8192' in src,'GraphQL token count bounded')
req('graphqlMaxFields             = 2048' in src,'GraphQL field count bounded')
req('graphqlMaxVariables          = 256' in src,'GraphQL variable count bounded')

# Parser / normalization.
for token in ('lexGraphQL','parseGraphQLDocument','selectionSet','variableDefinitions','typeRef','skipDirectives','skipArguments','skipValue','analyzeGraphQLDocument'):
    req(token in src,f'bounded parser includes {token}')
req('cyclic GraphQL fragment' in src,'fragment cycles fail closed')
req('operationName is required for a multi-operation GraphQL document' in src,'multi-operation selection is explicit')
req('operationName does not match a GraphQL operation' in src,'operationName mismatch rejected')
req('GraphQL depth exceeds absolute safety limit' in src,'absolute depth parser guard exists')
req('GraphQL complexity exceeds absolute safety limit' in src,'absolute complexity parser guard exists')
req('sel.field == "__schema" || sel.field == "__type"' in src,'schema introspection fields detected without treating __typename as introspection')
req('strings.Join(parts, "\\x00")' in src,'structural fingerprint derives from normalized metadata')

# Persisted query/APQ support.
for token in ('type GraphQLPersistedProfile struct','SHA256Hash','GraphQLOperationID','DocumentFingerprint','Depth','Complexity','Fields','Variables','Introspection'):
    req(token in src,f'persisted profile includes {token}')
pq_block=src[src.index('type GraphQLPersistedProfile struct'):src.index('// GraphQLPolicy')]
for forbidden in ('Query string','RawQuery','QueryText','Literal','RawVariable'):
    req(forbidden not in pq_block,f'persisted profile excludes raw document material {forbidden}')
req('graphqlMaxPersistedQueries   = 4096' in src,'persisted query registry bounded')
req('graphqlPersistedTTL          = 30 * 24 * time.Hour' in src,'persisted query TTL bounded')
req('persistedQuery hash does not match query' in src,'APQ query/hash mismatch detected')
req('unknown persistedQuery hash' in src,'unknown hash-only APQ rejected from analysis')
req('x.PersistedQuery.Version != 1' in src,'APQ version validated')
req('validGraphQLHash' in src,'APQ hash syntax validated')
req('s.persistedProfile(env.PersistedHash)' in src,'hash-only request resolves from bounded structural registry')

# GraphQL SDL schema contract.
for token in ('type GraphQLSchemaContract struct','SourceSHA256','QueryRoot','MutationRoot','SubscriptionRoot','Types','FieldTypes','parseGraphQLSDL','upsertSchemaContract','contractsSnapshot'):
    req(token in src,f'GraphQL schema contract includes {token}')
contract_block=src[src.index('type GraphQLSchemaContract struct'):src.index('// GraphQLPolicy')]
for forbidden in ('SDL string','RawSDL','Document string','RawQuery'):
    req(forbidden not in contract_block,f'schema contract durable model excludes raw SDL field {forbidden}')
req('graphqlMaxSchemaContracts    = 128' in src,'schema contract cardinality bounded')
req('graphqlMaxSchemaBytes        = 256 << 10' in src,'SDL import bytes bounded')
req('graphqlMaxSchemaTypes        = 1024' in src,'SDL type count bounded')
req('graphqlMaxSchemaFields       = 8192' in src,'SDL field count bounded')
req('GraphQL SDL query root' in src,'SDL import requires defined query root')
req('SourceSHA256: hex.EncodeToString(sum[:])' in src,'raw SDL replaced by source digest')
req('validGraphQLSchemaContract(v)' in src,'restored schema contracts revalidated')
req('graphqlSchemaAllowsField' in src and '"SCHEMA_FIELD_MISMATCH"' in src,'imported SDL can participate in deterministic field validation')
req('schema_contract_id must reference a GraphQL contract for endpoint_operation_id' in src,'policy schema binding is endpoint-scoped')
for route in ('GET /api/security/graphql/schema-contracts','POST /api/security/graphql/schema-contracts','DELETE /api/security/graphql/schema-contracts/{contract_id}'):
    req(route in admin,f'GraphQL schema contract route exists: {route}')
for handler in ('handleGraphQLSchemaContracts','handleGraphQLSchemaContractImport','handleGraphQLSchemaContractDelete'):
    req(f'authRole(roleReviewer, a.{handler})' in admin,f'{handler} requires Reviewer-or-higher')
for event in ('api_graphql.schema_contracts_view','api_graphql.schema_contract_import','api_graphql.schema_contract_delete'):
    req(event in src,f'schema contract audit event exists: {event}')

# Deterministic policy and authority.
for token in ('type GraphQLPolicy struct','EndpointOperationID','OperationName','SchemaContractID','Mode','MaxDepth','MaxComplexity','AllowIntrospection','AllowMutation','AllowSubscription','RequirePersisted','AllowedFields','DeniedFields','BlockStatus'):
    req(token in src,f'GraphQL policy includes {token}')
policy_block=src[src.index('type GraphQLPolicy struct'):src.index('type GraphQLViolation struct')]
for forbidden in ('IdentityFingerprint','ObjectFingerprint','TenantFingerprint','RawQuery','VariableValue','JWT','Authorization','Cookie','BOLACandidate'):
    req(forbidden not in policy_block,f'GraphQL policy excludes non-structural authority {forbidden}')
for mode in ('graphqlModeLearn   = "LEARN"','graphqlModeDetect  = "DETECT"','graphqlModeEnforce = "ENFORCE"'):
    req(mode in src,f'explicit GraphQL mode exists: {mode}')
req('case graphqlModeLearn:' in src and 'return to == graphqlModeDetect' in src,'LEARN can only promote to DETECT')
req('case graphqlModeDetect:' in src and 'to == graphqlModeEnforce' in src,'DETECT can promote to ENFORCE')
req('ENFORCE promotion requires a reason' in src,'ENFORCE promotion requires operator reason')
req('hasPolicy && policy.Mode == graphqlModeEnforce && len(violations) > 0' in src,'only explicit ENFORCE policy can block')
req('action = "detect"' in src and 'action = "block"' in src,'DETECT and ENFORCE actions are distinct')
req('request rejected by GraphQL security policy' in src,'deterministic GraphQL policy owns its rejection response')
req('policy.BlockStatus' in src,'GraphQL policy controls bounded 4xx status')
req('block_status must be a 4xx status' in src,'block status validation is fail closed')

# Policy coverage.
for typ in ('DEPTH_LIMIT','COMPLEXITY_LIMIT','INTROSPECTION_DISABLED','MUTATION_DISABLED','SUBSCRIPTION_DISABLED','PERSISTED_QUERY_REQUIRED','FIELD_NOT_ALLOWED','FIELD_DENIED','SCHEMA_FIELD_MISMATCH'):
    req(f'"{typ}"' in src,f'policy violation implemented: {typ}')
req('graphqlAbsoluteMaxDepth      = 64' in src,'absolute depth ceiling bounded')
req('graphqlAbsoluteMaxComplexity = 10000' in src,'absolute complexity ceiling bounded')
req('graphqlMaxPolicyFields       = 512' in src,'field policy selectors bounded')
req('normalizeGraphQLFieldList' in src,'field selectors normalized and validated')
req('sort.Strings(out)' in src,'field policy ordering deterministic')

# Async observation and restart safety.
req('graphqlQueueCapacity         = 1024' in src,'GraphQL observation queue bounded')
req('select {' in src and 'case s.queue <- ev:' in src and 'default:' in src,'GraphQL observation enqueue is non-blocking')
req('atomic.Bool' in src and 'atomic.Uint64' in src,'GraphQL queue telemetry is atomic')
req('func (s *graphqlStore) stopAndDrain()' in src,'GraphQL worker drains on shutdown')
req('func (s *graphqlStore) processObservation' in src,'GraphQL durable learning runs in background worker')
req('graphqlStateVersion          = 1' in src,'GraphQL durable state is versioned')
req('api-graphql.json' in src,'GraphQL state has dedicated file')
req('state.Version != 0 && state.Version != graphqlStateVersion' in src,'unknown GraphQL state version rejected')
req('validGraphQLOperation(v, now)' in src,'restored operations revalidated')
req('validGraphQLPersisted(v, now)' in src,'restored persisted profiles revalidated')
req('validGraphQLPolicy(v)' in src,'restored policies revalidated')
req('graphql.save(configPath)' in persist,'GraphQL state included in API autosave/final flush')
req('s.graphql.load(*configPath)' in main,'GraphQL state restored before traffic')
req('s.graphql.start()' in main,'GraphQL background worker starts')
req('s.graphql.stopAndDrain()' in main,'GraphQL worker drains before final persistence')

# GraphQL variable -> API-7 integration.
req('objectLocatorSourceAPI8       = "API8_GRAPHQL_VARIABLE"' in loc,'API-8 GraphQL variable locator source is explicit')
req('case "path", "query", "body", "graphql_variable"' in loc,'locator model supports distinct GraphQL variable location')
req('noteGraphQLVariable' in loc,'GraphQL variable locator bridge exists')
req('mergeLocked(operationID, "graphql_variable"' in loc,'GraphQL variable locator uses normalized GraphQL operation ID')
req('digestValue(ev.Operation.ID, "graphql_variable"' in src,'GraphQL raw variable converted to keyed fingerprint')
req('RawValue' in src,'raw variable exists only in transient observation sample')
req('type graphqlVariableSample struct' in src,'GraphQL variable raw sample is transient-only type')
req('RawValue' not in op_block and 'RawValue' not in pq_block and 'RawValue' not in policy_block,'raw variable absent from durable GraphQL models')
req('objectRelationshipSourceAPI8       = "API8_GRAPHQL_VARIABLE"' in rel,'API-7.2 relationship source distinguishes API-8')
req('enqueueKeyed' in rel,'GraphQL relationship bridge is non-blocking')
req('DirectObjectFingerprint' in rel and 'DirectSource' in rel,'relationship queue carries keyed GraphQL evidence only')
req('s.detector.observe(ev, locator, ev.DirectObjectFingerprint, s)' in rel,'GraphQL keyed object can feed existing non-enforcing BOLA detector')
req('mergeWithSource(ev, locator, ev.DirectObjectFingerprint, ev.DirectSource)' in rel,'GraphQL relationship merged after detector baseline evaluation')
req('containsString(v.Sources, objectRelationshipSourceAPI8)' in rel,'restart validation accepts API-8 relationship source')
req('bolaPolicy' not in src,'API-7.4 evidence routing policy is not GraphQL authorization authority')
req('OpenAI' not in src,'OpenAI absent from GraphQL parser/policy authority source')
req('openai' not in src.lower(),'no OpenAI request-path GraphQL authority')

# Request-path placement and capture.
req('graphql             *graphqlStore' in main,'server owns GraphQL store')
req('graphql:             newGraphQLStore()' in main,'server initializes GraphQL store')
req('s.graphql.objectLocators = s.objectLocators' in main,'GraphQL locator integration wired')
req('s.graphql.objectRelationships = s.objectRelationships' in main,'GraphQL relationship integration wired')
req('graphqlHandler = s.graphql.wrap(siteName, graphqlHandler)' in main,'GraphQL middleware wired')
req('requestBodyPrefixWrap(captureForAI, cfg.PassiveDiscoveryEnabled, true, graphqlHandler)' in main,'GraphQL middleware runs inside bounded shared body capture')
req(main.index('handler = s.identity.wrap(siteName, handler)') > main.index('requestBodyPrefixWrap(captureForAI, cfg.PassiveDiscoveryEnabled, true, graphqlHandler)'), 'API-5 identity wraps outside GraphQL middleware')
req('requestBodyPrefixFromRequest(r)' in src,'GraphQL POST body uses shared bounded capture')
req('requestBodyTruncatedFromRequest(r)' in src,'GraphQL detects truncated capture')
req('application/graphql' in src and 'application/json' in src,'GraphQL supports raw and JSON envelopes')
req('r.Method == http.MethodGet' in src,'GraphQL GET query form supported')

# Reviewer-only APIs and audit.
routes=(
'GET /api/security/graphql/operations','GET /api/security/graphql/persisted-queries','GET /api/security/graphql/policies','POST /api/security/graphql/policies','POST /api/security/graphql/policies/{policy_id}/mode','DELETE /api/security/graphql/policies/{policy_id}','GET /api/security/graphql/violations','GET /api/security/graphql/status')
for route in routes: req(route in admin,f'admin route exists: {route}')
for handler in ('handleGraphQLOperations','handleGraphQLPersisted','handleGraphQLPolicies','handleGraphQLPolicyUpsert','handleGraphQLPolicyMode','handleGraphQLPolicyDelete','handleGraphQLViolations','handleGraphQLStatus'):
    req(f'authRole(roleReviewer, a.{handler})' in admin,f'{handler} requires Reviewer-or-higher')
for event in ('api_graphql.operations_view','api_graphql.persisted_view','api_graphql.policy_upsert','api_graphql.policy_mode','api_graphql.policy_delete','api_graphql.violations_view','api_graphql.status_view'):
    req(event in src,f'audit event exists: {event}')
req('persistGraphQLMutation' in src and 'graphql.save(a.srv.configPath)' in src,'GraphQL mutations persist before success')
req('DisallowUnknownFields()' in src,'GraphQL mutation JSON rejects unknown fields')
req('http.MaxBytesReader(w, r.Body, 1<<16)' in src,'GraphQL policy mutation body bounded')

# Console coverage.
for token in ('API-8 GraphQL Security','api_graphql_operations','api_graphql_persisted','api_graphql_policies','api_graphql_violations','api_graphql_status','api_graphql_policy_save'):
    req(token in ui,f'Admin Console includes {token}')
req('LEARN → DETECT → ENFORCE' in ui,'console explains explicit enforcement promotion')
req('API-6/API-7 learned sequence/BOLA evidence remains non-enforcing' in ui,'console preserves learned-signal authority boundary')
req('/api/security/graphql/operations' in ui and '/api/security/graphql/policies' in ui,'console calls GraphQL APIs')
req('require_persisted_queries' in ui and 'allow_introspection' in ui and 'allow_mutation' in ui,'console exposes GraphQL policy controls')

# API-8 governance/restart hardening.
req('p.Mode = graphqlModeLearn' in src,'GraphQL policy edits return to LEARN before renewed enforcement')
req('schema contract changed; revalidation required' in src,'schema contract content changes demote dependent policies')
req('GraphQL schema contract is referenced by policy' in src,'referenced schema contracts cannot be deleted out from under enforcement policy')
req('func graphqlFallbackPolicyBetter' in src,'malformed-endpoint fallback policy selection is deterministic')
req('strings.HasPrefix(field, "__")' in src,'schema validation defers introspection topology to explicit introspection policy')
req('dec.UseNumber()' in src and 'large GraphQL ID values must never lose precision before HMAC' in src,'GraphQL numeric variable fingerprints avoid float64 precision loss')
req('func validGraphQLViolation' in src and 'validGraphQLViolation(v)' in src,'restored GraphQL violation evidence is revalidated')
req('graphqlMaxPolicyReason       = 512' in src,'GraphQL policy reasons are bounded')

# Test matrix.
expected=(
'TestAPI8GraphQLOperationDiscoveryNormalizesLiteralValues','TestAPI8GraphQLFragmentsDepthComplexityAndIntrospection','TestAPI8GraphQLRejectsCyclicFragmentsAndOversizedInput','TestAPI8GraphQLPolicyRequiresLearnDetectEnforcePromotion','TestAPI8GraphQLDeterministicPolicyCoversDepthComplexityFieldsMutationIntrospectionPersisted','TestAPI8PersistedQueryProfileContainsNoRawQueryAndSupportsHashOnlyLookup','TestAPI8GraphQLVariableLocatorUsesKeyedEvidenceAndVerifiedIdentityOnly','TestAPI8GraphQLWrapperDetectDoesNotBlockButEnforceDoes','TestAPI8GraphQLPersistedHashMismatchFailsClosedUnderEnforce','TestAPI8GraphQLPersistenceRevalidatesAndRestoresBoundedState','TestAPI8GraphQLStateAndPolicyCardinalityAreBounded','TestAPI8GraphQLConcurrentSnapshotsPolicyUpdatesAndObservations','TestAPI8OpenAIHasNoGraphQLAuthorityAndBOLAInferenceRemainsSeparate','TestAPI8ConfiguredGraphQLEndpointRejectsMalformedJSONOnlyInEnforce','TestAPI8GraphQLSchemaContractParsesSDLWithoutPersistingRawDocument','TestAPI8GraphQLSchemaContractRejectsMissingQueryRootAndBoundsState','TestAPI8GraphQLSchemaContractParticipatesInDeterministicFieldValidation','TestAPI8GraphQLPolicyEditAndSchemaChangeReturnToLearn','TestAPI8GraphQLSchemaDeleteRejectsReferencedContract','TestAPI8GraphQLSchemaIntrospectionDoesNotConflictWithExplicitIntrospectionPolicy','TestAPI8GraphQLNumericVariableFingerprintInputPreservesPrecision','TestAPI8GraphQLViolationRestoreDropsUntrustedDurablePayload')
for name in expected: req(f'func {name}(' in tests,f'targeted test present: {name}')
req(len(re.findall(r'^func TestAPI8',tests,re.M))>=22,'at least 22 targeted API-8 tests present')

# Build/CI integration.
req('tools/tests/test-api8-source.py' in build,'build.sh runs API-8 source gate')
req('tools/tests/test-api8-source.py' in ci,'CI runs API-8 source gate')

print(f'API8_SOURCE_GATE_PASS checks={len(checks)}')
