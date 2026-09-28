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
    'api3_contract.go',
    'api3_export.go',
    'api3_contract_test.go',
    'api_security_persist.go',
    'API3_OPENAPI_CONTRACT_IMPLEMENTATION.md',
    'API3_TEST_MATRIX.md',
    'API3_ACCEPTANCE_CRITERIA.md',
]
for rel in required:
    require((root / rel).is_file(), f'missing required API-3 file: {rel}')

api3 = (root / 'api3_contract.go').read_text()
admin = (root / 'admin.go').read_text()
main = (root / 'main.go').read_text()
obs = (root / 'observations.go').read_text()
gomod = (root / 'go.mod').read_text()

for symbol in [
    'parseOpenAPIDocument',
    'flattenContractSchema',
    'matchContractOperations',
    'compareContractVersions',
    'compareContractFields',
    'driftAgainstLearnedSchema',
    'handleContractImport',
    'handleContractMatch',
    'handleContractCompare',
    'handleContractDrift',
    'handleContractExport',
    'parseContractParameters',
    'parseContractSecurity',
    'securitySchemesFingerprint',
]:
    source = api3 + (root / 'api3_export.go').read_text()
    require(symbol in source, f'missing API-3 symbol: {symbol}')

routes = [
    'GET /api/security/contracts',
    'POST /api/security/contracts/import',
    'GET /api/security/contracts/{id}',
    'GET /api/security/contracts/{id}/versions',
    'POST /api/security/contracts/{id}/match',
    'GET /api/security/contracts/{id}/bindings',
    'POST /api/security/contracts/{id}/compare',
    'GET /api/security/contracts/{id}/diffs',
    'GET /api/security/contracts/{id}/drift',
    'GET /api/security/contracts/{id}/export',
]
for route in routes:
    exact = len(re.findall(r'mux\.HandleFunc\("' + re.escape(route) + r'"', admin))
    require(exact == 1, f'route {route!r} registered {exact} times, want 1')

require('apiOps:' in main and 'newAPIOperationStore()' in main, 'apiOps store not initialized in server constructor')
require('schema:' in main and 'newSchemaStore()' in main, 'schema store not initialized in server constructor')
require('s.observations.apiOps = s.apiOps' in main, 'API-1 operation store not wired into production observation plane')
require('p.apiOps.note(ev.site, ev.meta.Host, ev.method, ev.path, ev.contentType, ev.meta.AuthScheme, ev.code)' in obs, 'observation plane does not feed API-1 operation store')
require('p.schema.note(op, samples)' in obs, 'observation plane does not feed API-2 schema store')
require('s.apiOps.load(*configPath)' in main, 'API-1 persisted state not restored')
require('s.schema.load(*configPath)' in main, 'API-2 persisted state not restored')
require('s.contracts.load(*configPath)' in main, 'API-3 persisted state not restored')
require('startAPISecurityAutosave' in main, 'API security autosave not started')

require('validateContractRefsLocalOnly' in api3 and 'external OpenAPI $ref is not supported' in api3, 'external OpenAPI refs are not rejected')
require('matchedSites' in api3 and 'UNDECLARED_ENDPOINT' in api3, 'undeclared-endpoint drift is not scoped to matched application traffic')
require('authObservationFamilies' in api3 and 'anonymous' in api3, 'OpenAPI auth OR/AND/anonymous semantics hardening missing')
require('RequestMediaTypes' in api3 and 'observedContentTypeMatches' in api3, 'multi-content-type contract drift support missing')

require('github.com/goccy/go-yaml v1.18.0' in gomod, 'OpenAPI YAML parser dependency missing from go.mod')
# Direct dependency: it must appear before the second require block and must not be marked indirect.
yaml_line = next((line for line in gomod.splitlines() if 'github.com/goccy/go-yaml v1.18.0' in line), '')
require('// indirect' not in yaml_line, 'goccy/go-yaml is used directly but still marked indirect')

# Security boundary: API-3 source must not invoke known blocking/enforcement entry points.
for forbidden in ['blocklist.add', 'setBlock', 'applyPagePolicy', 'EngineMode = "On"']:
    require(forbidden not in api3, f'API-3 contract layer unexpectedly contains enforcement token: {forbidden}')

require((root / 'api_security_integration_test.go').is_file(), 'missing API1->API2->API3 integration test')

if errors:
    for err in errors:
        print('FAIL:', err)
    sys.exit(1)
print('API3_SOURCE_GATE_PASS checks=', len(required) + len(routes) + 30)
