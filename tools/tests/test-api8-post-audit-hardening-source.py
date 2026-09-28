#!/usr/bin/env python3
from pathlib import Path
import re
ROOT=Path(__file__).resolve().parents[2]
read=lambda p:(ROOT/p).read_text(encoding='utf-8')
cidr=read('cidr_policy.go'); cidrt=read('cidr_policy_test.go'); l7=read('l7_abuse.go'); l7t=read('l7_abuse_test.go')
main=read('main.go'); listeners=read('listeners.go'); admin=read('admin.go'); ui=read('static/admin.html')
readme=read('README.md'); roadmap=read('DEVELOPMENT_ROADMAP.md'); docidx=read('DOCUMENTATION_INDEX.md'); manifest=read('MANIFEST.md')
build=read('build.sh'); ci=read('.github/workflows/ci.yml'); api5=read('tools/tests/test-api5-source.py')
checks=[]
def req(cond,msg):
    if not cond: raise SystemExit('API8_POST_AUDIT_HARDENING_SOURCE_GATE_FAIL: '+msg)
    checks.append(msg)

# CIDR correctness.
for t in ('enabled bool','cfg.Enabled','time.Parse(time.RFC3339','Expires: expires','!e.enabled','time.Now().After(r.Expires)'):
    req(t in cidr,'CIDR runtime contains '+t)
for t in ('TestCIDRPolicyExpiryRFC3339IsApplied','TestCIDRPolicyExpiredRuleIsSkipped','TestCIDRPolicyRejectsInvalidExpiry','TestCIDRPolicyDisabledDoesNotMatch'):
    req(t in cidrt,'CIDR regression test exists: '+t)
req('expires_at must be RFC3339' in cidr,'CIDR invalid expiry fails validation')
req('sort.SliceStable' in cidr,'CIDR priority ordering retained')

# L7 TLS-handshake control is real and bounded.
for t in ('TLSHandshakePerWindow','tlsShards','allowTLSHandshakeAt','windowDuration','l7AbuseMaxEntriesPerShard','tlsfront.FrontendEnabled'):
    req(t in l7,'L7 TLS implementation contains '+t)
req('requires tls_acceleration.mode=go' in l7,'external TLS frontend fails closed when handshake limit requested')
req('rt.l7Abuse.allowTLSHandshakeAt' in listeners,'ClientHello path invokes TLS handshake limiter')
req('hello.Conn.RemoteAddr()' in listeners,'TLS limiter keys direct TCP peer')
req('logicalAddr' in listeners,'TLS limiter scopes by listener')
req(re.search(r'\bl7Abuse\s+\*l7AbuseController', main[main.index('type runtimeState struct'):main.index('type server struct')]) is not None,'L7 controller is immutable runtime state')
req(re.search(r'\bl7Abuse\s+\*l7AbuseController', main[main.index('type server struct'):main.index('func defaultConfig')]) is None,'mutable server-level L7 controller removed')
for t in ('TestL7TLSHandshakeBucketEnforcesAndRefills','TestL7TLSHandshakeBucketsAreListenerScoped','TestL7TLSHandshakeValidationRejectsExternalFrontend'):
    req(t in l7t,'L7 TLS regression test exists: '+t)
req(main.count('validateL7AbuseConfig(c.L7Abuse, c.TLSAcceleration)') >= 2,'draft and full config validation both enforce L7/TLS compatibility')

# Console system/diagnostics and HSM/Vector exposure.
for t in ('data-view="system"','id="view-system"','id="doctor_body"','id="debug_evidence"','id="hsm_status"','id="hsm_audit"','id="vector_groups"','id="vector_reset"'):
    req(t in ui,'Console exposes '+t)
for route in ('/api/doctor','/api/debug/status','/api/debug/capture','/api/debug/evidence','/api/debug/export','/api/hsm/status','/api/hsm/audit','/api/vector-acceleration','/api/vector-acceleration/reset'):
    req(route in ui,'Console calls '+route)
req('async function downloadAPI' in ui,'Console has authenticated blob download helper')
req('Reviewer permission required' in ui,'Console degrades restricted evidence surfaces without hiding all diagnostics')

# HSM site editor.
for field in ('provider','module_path','slot_id','token_label','key_label','key_id','pin_secret_ref'):
    req(f'data-hsm-f="{field}"' in ui,'site HSM editor exposes '+field)
req('env:WAF_HSM_PIN or file:/run/secrets/hsm-pin' in ui,'HSM UI documents secret-reference-only PIN')
req('Leave blank to preserve the existing secret reference' in ui,'HSM UI documents redacted secret preservation')
req('s.tls_key_provider=h' in ui and 'if(h.provider) s.tls_key=""' in ui,'HSM editor serializes nested key provider and clears conflicting file key')

# CIDR/L7 traffic controls.
for t in ('id="l7_enabled"','id="l7_requests"','id="l7_window"','id="l7_concurrent"','id="l7_tls_handshakes"','id="cidr_enabled"','id="cidr_rules"','id="cidr_add"','id="traffic_save"','id="traffic_apply"'):
    req(t in ui,'traffic controls expose '+t)
for fn in ('fillTrafficControls','collectL7Abuse','collectCIDRPolicy','renderCIDRRules','syncCIDRRules'):
    req(f'function {fn}' in ui,'traffic control wiring includes '+fn)
req('expires_at' in ui and 'RFC3339' in ui,'CIDR UI exposes explicit expiry')
req('tls_handshake_per_window' in ui,'L7 TLS handshake setting is serialized')

# OpenAPI lifecycle.
for route in ('/api/security/contracts/${encodeURIComponent(contractID)}','/versions','/bindings','/match','/compare','/diffs','/drift','/export?format=json','/export?format=yaml'):
    req(route in ui,'OpenAPI Console lifecycle includes '+route)
for t in ('api-contract-open','loadOpenAPIContract','Match latest','Refresh drift','Compare from','Export JSON','Export YAML'):
    req(t in ui,'OpenAPI lifecycle UI includes '+t)
req('api_contract_detail' in ui,'OpenAPI detail panel exists')

# Positive schema lifecycle.
for t in ('api-enforce-detail','loadPositiveSchemaDetail','api-profile-activate','api-exception-toggle','api-schema-return'):
    req(t in ui,'Positive Schema lifecycle UI includes '+t)
for route in ('/activate','/exceptions/${encodeURIComponent(b.dataset.id)}'):
    req(route in ui,'Positive Schema lifecycle calls '+route)
req('return_to_candidate' in ui,'schema review can return reviewed candidate')
req('expires_at' in ui,'schema exception creation can set expiry')

# Opaque-ID selectors.
req('<select id="api_sequence_ex_workflow">' in ui and '<input id="api_sequence_ex_workflow"' not in ui,'sequence workflow opaque ID uses inventory select')
req('<select id="api_bola_policy_locator">' in ui and '<input id="api_bola_policy_locator"' not in ui,'BOLA locator opaque ID uses inventory select')
req('<select id="api_gql_schema_contract">' in ui and '<input id="api_gql_schema_contract"' not in ui,'GraphQL schema contract opaque ID uses inventory select')
for t in ('setSelectOptions("api_sequence_ex_workflow"','setSelectOptions("api_bola_policy_locator"','setSelectOptions("api_gql_schema_contract"'):
    req(t in ui,'opaque selector populated: '+t)
req('$("api_sequence_ex_site").value=w.site' in ui,'workflow selector carries site scope')
req('$("api_bola_policy_op").value=x.operation_id' in ui,'locator selector carries operation scope')
req('$("api_gql_endpoint").value=x.endpoint_operation_id' in ui,'GraphQL contract selector carries endpoint scope')

# Foundation/dual frontend cleanup.
for f in ('investigation.go','security_timeline.go','change_audit.go','security_export.go','debug_lifecycle_v2.go','debug_retention_worker_v2.go'):
    req(not (ROOT/f).exists(),'misleading unused foundation removed: '+f)
req(not (ROOT/'web').exists(),'obsolete non-shipping dual frontend removed')
req('one authoritative Console source tree' in readme,'README states single-console truth')
req('PLANNED / NOT IMPLEMENTED' in roadmap,'Phase 5 Slice D model-only foundation corrected to not implemented')
req('obsolete experimental `web/` migration tree was removed' in docidx,'documentation index records dual-frontend cleanup')
req('Vite+preact console SCAFFOLD' not in manifest,'manifest no longer advertises removed frontend scaffold')

# Existing authority/RBAC remains backend-authoritative.
for route in ('GET /api/hsm/status','GET /api/hsm/audit','GET /api/doctor','GET /api/debug/status','POST /api/debug/capture','GET /api/debug/evidence','GET /api/debug/export','GET /api/vector-acceleration','POST /api/vector-acceleration/reset'):
    req(route in admin,'backend route retained: '+route)
req('identity policy executes inside abuse gate and before schema/Coraza/backend' in api5 and 'l7Abuse.wrap(siteName, handler)' in api5,'API-5 source gate tracks immutable runtime L7 wrapper')

# Console structural sanity.
ids=re.findall(r'\bid="([^"$]+)"',ui)
refs=re.findall(r'\$\("([^"]+)"\)',ui)
req(not (set(refs)-set(ids)),'all static $(id) references resolve to HTML IDs')
req('function loadSystem' in ui and 'if (v==="system") loadSystem();' in ui,'SYSTEM tab loads diagnostics')

# Gate wiring.
req('test-api8-post-audit-hardening-source.py' in build,'build.sh runs post-audit hardening gate')
req('test-api8-post-audit-hardening-source.py' in ci,'CI runs post-audit hardening gate')
print(f'API8_POST_AUDIT_HARDENING_SOURCE_GATE_PASS checks={len(checks)}')
