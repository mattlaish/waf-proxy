#!/usr/bin/env python3
from pathlib import Path
import re
import sys

root = Path(__file__).resolve().parents[2]
errors = []

def require(cond, msg):
    if not cond:
        errors.append(msg)

required = [
    'api_operations.go',
    'api_operations_test.go',
    'schema_api2.go',
    'schema_api2_test.go',
    'api_security_persist.go',
    'api_security_handlers.go',
    'api_security_integration_test.go',
    'ai_schema_review.go',
    'observations.go',
    'profiles.go',
    'main.go',
    'admin.go',
    'static/admin.html',
]
for rel in required:
    require((root / rel).is_file(), f'missing API-1/API-2 file: {rel}')

ops = (root / 'api_operations.go').read_text()
schema = (root / 'schema_api2.go').read_text()
persist = (root / 'api_security_persist.go').read_text()
handlers = (root / 'api_security_handlers.go').read_text()
ai = (root / 'ai_schema_review.go').read_text()
obs = (root / 'observations.go').read_text()
profiles = (root / 'profiles.go').read_text()
main = (root / 'main.go').read_text()
admin = (root / 'admin.go').read_text()
ui = (root / 'static/admin.html').read_text()
all_go = '\n'.join(p.read_text() for p in root.glob('*.go'))

for token in ['Host', 'ContentTypes', 'AuthObserved', 'RawPathExamples', 'PathParameters', 'apiOperationID', 'classifyPathSegment']:
    require(token in ops, f'API-1 operation model missing {token}')

for route in [
    'GET /api/security/operations',
    'GET /api/security/operations/{id}',
    'POST /api/security/operations/{id}/ignore',
    'POST /api/security/operations/reclassify',
    'GET /api/security/schema/candidates',
    'GET /api/security/schema/{id}',
    'POST /api/security/schema/{id}/review',
    'POST /api/security/schema/{id}/ai-review',
]:
    exact = len(re.findall(r'mux\.HandleFunc\("' + re.escape(route) + r'"', admin))
    require(exact == 1, f'route {route!r} registered {exact} times, want 1')

for token in ['api-operations.json', 'api-schema.json', 'atomicWriteJSON', 'restorePersistence']:
    require(token in persist + schema, f'persistence token missing: {token}')
require('s.apiOps.load(*configPath)' in main, 'API-1 restore missing at startup')
require('s.schema.load(*configPath)' in main, 'API-2 restore missing at startup')
require('startAPISecurityAutosave' in main, 'API security autosave missing')

for symbol in [
    'collectSchemaSamples', 'addPathParameterSamples', 'addQuerySamples', 'selectedSchemaHeaders', 'addHeaderValueSamples',
    'addBodySamples', 'flattenJSONSamples', 'inferScalarType', 'safeEnumValue',
    'buildCandidateFromAggregate', 'RequiredCandidate', 'EnumCandidate', 'PresenceRate',
    'DistinctCardinality',
]:
    require(symbol in schema, f'API-2 learning symbol missing: {symbol}')
for media in ['application/json', 'application/x-www-form-urlencoded', 'multipart/form-data']:
    require(media in schema, f'API-2 collector missing media type: {media}')
require('p.schema.note(op, samples)' in obs, 'live observation plane is not feeding API-2')
require('buildAPIObservationMeta' in main, 'reverse proxy does not build API observation metadata')
require('requestBodyPrefixWrap(captureForAI, cfg.PassiveDiscoveryEnabled, true, ' in main,
        'API-2 body capture still depends on passive discovery or AI')
require('captureForSchema' in profiles and 'captureForPassive || captureForSchema' in profiles,
        'shared request-prefix capture does not model API-2 as an independent consumer')
meta_start = schema.find('func buildAPIObservationMeta')
meta_end = schema.find('func collectSchemaSamplesFromObservation', meta_start)
if meta_start >= 0 and meta_end > meta_start:
    require('collectSchemaSamplesFromObservation(' not in schema[meta_start:meta_end],
            'heavy schema parsing must not run while building request observation metadata')
else:
    require(False, 'could not inspect buildAPIObservationMeta')
require('collectSchemaSamplesFromObservation' in obs,
        'heavy schema parsing is not delegated to the bounded observation worker')

require('Authorization' in schema and 'isSensitiveFieldName' in schema, 'privacy boundary missing')
require('schema AI review failed' in handlers, 'schema AI review handler missing')
require('callOpenAIResponses' in ai, 'schema AI review does not use existing Responses API path')
for forbidden in ['EngineMode = "On"', 'blocklist.add', 'applyPagePolicy', 'setBlock']:
    require(forbidden not in ai + handlers, f'AI schema review unexpectedly contains enforcement token: {forbidden}')
require('enforcement_changed' in handlers and 'false' in handlers, 'AI review response does not explicitly preserve enforcement state')

require('data-view="api"' in ui, 'API Security console tab missing')
for dom_id in ['api_ops_body', 'api_schema_body', 'api_contract_body', 'api_contract_import']:
    require(dom_id in ui, f'API Security UI control missing: {dom_id}')

require('writeJSON(w, http.' not in all_go, 'known writeJSON arity compile defect is still present')

require('os.CreateTemp' in persist and '.tmp-*' in persist, 'API security persistence still uses a fixed temporary filename')
require('len(agg.Fields) >= schemaMaxFields' in schema, 'API-2 aggregate field growth is not bounded across requests')
require('dominantObservedType(af.TypeCounts) != dominantObservedType(bf.TypeCounts)' in schema, 'schema review is not invalidated when dominant type changes')
require('strings.Join(families, "+")' in schema and 'normalizeAuthScheme(strings.Join' in schema, 'multi-mechanism auth evidence is not preserved')

if errors:
    for err in errors:
        print('FAIL:', err)
    sys.exit(1)
print('API12_SOURCE_GATE_PASS checks=', len(required) + 56)
