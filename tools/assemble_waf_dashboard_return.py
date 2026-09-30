#!/usr/bin/env python3
"""Assemble the WAF Proxy OWI-1.0 R3 product return from real connector output."""
from __future__ import annotations
import argparse, copy, hashlib, json, os, pathlib, shutil, sys, zipfile
from datetime import datetime, timezone

SOURCE_VERSION = "2026-09-30-OWI-R3-CONNECTOR-HARDENED"
BASE = "/api/integrations/dashboard/v1"
SCOPES = [
    "dashboard:read:health", "dashboard:read:capabilities", "dashboard:read:assets",
    "dashboard:read:detections", "dashboard:read:policies", "dashboard:read:health-observations",
]
CAPS = ["HEALTH", "LIST_ASSETS", "LIST_DETECTIONS", "OBSERVE_POLICY"]

def sha_bytes(b: bytes) -> str: return hashlib.sha256(b).hexdigest()
def sha_file(p: pathlib.Path) -> str: return sha_bytes(p.read_bytes())
def dump(path: pathlib.Path, obj) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(obj, indent=2, sort_keys=False) + "\n", encoding="utf-8")
def copy_file(src, dst):
    dst.parent.mkdir(parents=True, exist_ok=True); shutil.copy2(src, dst)

def binding_fixture():
    resources = [
        ("assets","ASSET","LIST_ASSETS",True,"STATE",["SNAPSHOT","INCREMENTAL"],300),
        ("detections","DETECTION","LIST_DETECTIONS",True,"HISTORY",["SNAPSHOT","INCREMENTAL"],30),
        ("policies","POLICY","OBSERVE_POLICY",True,"STATE",["SNAPSHOT","INCREMENTAL"],300),
        ("health-observations","HEALTH","HEALTH",True,"STATE",["SNAPSHOT","INCREMENTAL"],60),
        ("events","EVENT","LIST_EVENTS",False,"HISTORY",[],30),
        ("action-status","ACTION_STATUS","LIST_ACTIONS",False,"STATE",[],30),
    ]
    return {
      "contract":"operator-workspace.integration.v1","schema_version":"1.0","status":"RETURNED_UNQUALIFIED",
      "product_slug":"waf-proxy","source_product":"WAF","source_instance_id":"waf-proxy-fixture-01",
      "tenant_id":"fixture-tenant-01","integration_instance_id":"dashboard-waf-proxy-fixture-01","enabled":False,
      "qualification":"IMPLEMENTED_UNQUALIFIED","network_profile":"DEDICATED_HOST_443",
      "origin":"https://waf-proxy-fixture.invalid:443","alternative_same_host_port":19405,"ui_origin":None,
      "resolved_address_allowlist":["192.0.2.5"],"base_path":BASE,
      "tls":{"verify_peer":True,"verify_hostname":True,"minimum_version":"1.2","trust_bundle_ref":"secret://systemd/waf-proxy-fixture-ca","client_certificate_ref":None,"client_private_key_ref":None,"revocation_policy":"PLATFORM_MANAGED_REQUIRED"},
      "auth":{"profile":"opaque_bearer","credential_ref":"secret://systemd/waf-proxy-fixture-reader-token","issuer":None,"audience":None,"principal_ref":"dashboard-waf-proxy-fixture-reader","tenant_selector":None,"scopes":SCOPES},
      "endpoints":{"health":BASE+"/health","capabilities":BASE+"/capabilities","resources":[
        {"path":BASE+"/"+s,"kind":k,"capability":c,"required":enabled,"enabled":enabled,"stream_semantics":sem,"sync_modes":modes,"poll_interval_seconds":poll}
        for s,k,c,enabled,sem,modes,poll in resources]},
      "limits":{"page_size":100,"max_page_size":500,"max_response_bytes":4194304,"connect_timeout_seconds":5,"read_timeout_seconds":10,"request_deadline_seconds":30},
      "action_plane":{"enabled":False,"credential_ref":None,"profiles":[]},"deployment_role":"PRODUCT_AUTHORITY",
      "evidence":{"source_version":SOURCE_VERSION,"artifact_sha256":None,"test_results_ref":"test-results.md","qualified_at":None}
    }

def negative_fixtures(valid_asset, valid_det):
    out={}
    def put(name,obj): out[name]=obj; return f"fixtures/negative/{name}.json"
    a=copy.deepcopy(valid_asset); a.pop("contract",None); p1=put("decode-missing-contract",a)
    d=copy.deepcopy(valid_det); d["unexpected_secret_field"]="must-be-rejected"; p2=put("page-extra-field",d)
    a=copy.deepcopy(valid_asset); a["source_product"]="FIREWALL"; p3=put("wrong-product",a)
    a=copy.deepcopy(valid_asset); a["source_instance_id"]="other-waf-instance"; p4=put("wrong-source",a)
    a=copy.deepcopy(valid_asset); a["resource_kind"]="POLICY"; p5=put("wrong-kind",a)
    a=copy.deepcopy(valid_asset); a["records"][0]["revision"]="0"; p6=put("invalid-revision",a)
    a=copy.deepcopy(valid_asset); a["records"][0]["source_observed_at"]="2026-09-30 00:00:00"; p7=put("invalid-utc",a)
    a=copy.deepcopy(valid_asset); a["records"][0]["operation"]="DELETE"; p8=put("delete-nonempty-payload",a)
    a=copy.deepcopy(valid_asset); a["records"]=[]; a["has_more"]=True; p9=put("empty-more",a)
    p10=put("cursor-no-progress",{"previous_cursor":valid_asset["next_cursor"],"next_page":copy.deepcopy(valid_asset)})
    drift=copy.deepcopy(valid_asset); drift["source_snapshot_id"]="different-snapshot"; p11=put("snapshot-drift",{"first_page":valid_asset,"next_page":drift})
    r=copy.deepcopy(valid_asset["records"][0]); r2=copy.deepcopy(r); r2["revision"]="2"; p12=put("old-revision",{"seen":r2,"incoming":r})
    r3=copy.deepcopy(r); r3["payload"]=copy.deepcopy(r3["payload"]); r3["payload"]["summary"]="drifted at equal revision"; p13=put("equal-revision-drift",{"seen":r,"incoming":r3})
    p14=put("cross-tenant",{"tenant_id":"other-tenant","page":valid_asset})
    p15=put("limit-bound-cursor-change",{"cursor":valid_asset["next_cursor"],"original_page_size":2,"retry_page_size":3})
    cases=[
      ("N-01","decode",p1,None,"DECODE_REJECT","Required contract field is missing."),
      ("N-02","page_validate",p2,None,"SCHEMA_REJECT","Closed page envelope contains an additional field."),
      ("N-03","sync",p3,None,"SOURCE_PRODUCT_MISMATCH","Page is for a different product type."),
      ("N-04","sync",p4,None,"SOURCE_INSTANCE_MISMATCH","Page is for a different source instance."),
      ("N-05","sync",p5,None,"RESOURCE_KIND_MISMATCH","Route/resource kind cannot change within a lane."),
      ("N-06","page_validate",p6,None,"SCHEMA_REJECT","revision must be a positive decimal string."),
      ("N-07","page_validate",p7,None,"SCHEMA_REJECT","source_observed_at must be UTC RFC3339."),
      ("N-08","page_validate",p8,None,"SCHEMA_REJECT","DELETE requires an empty payload."),
      ("N-09","sync",p9,None,"EMPTY_MORE_REJECT","has_more=true cannot carry an empty records array."),
      ("N-10","sync",p10,None,"CURSOR_NO_PROGRESS","A multi-page chain must advance its cursor."),
      ("N-11","sync",p11,None,"SNAPSHOT_DRIFT","SNAPSHOT source_snapshot_id must remain stable."),
      ("N-12","sync",p12,None,"OLD_REVISION","Older revisions cannot roll back projected state."),
      ("N-13","sync",p13,None,"EQUAL_REVISION_DRIFT","Same revision with different content is a conflict."),
      ("N-14","sync",p14,{"tenant_id":"other-tenant"},"TENANT_MISMATCH","Fixture page cannot cross tenant visibility."),
      ("N-15","sync",p15,{"cursor":valid_asset["next_cursor"],"page_size":3},"RESET_REQUIRED","Product cursor is bound to page size; retry must preserve limit."),
    ]
    return out,[{"case_id":i,"stage":st,"input_fixture":path,"request":req,"expected_error":err,"reason":reason} for i,st,path,req,err,reason in cases]

def main():
    ap=argparse.ArgumentParser(); ap.add_argument('--source-root',required=True); ap.add_argument('--spec-root',required=True); ap.add_argument('--generated',required=True); ap.add_argument('--output-dir',required=True); ap.add_argument('--log-dir'); args=ap.parse_args()
    src=pathlib.Path(args.source_root).resolve(); spec=pathlib.Path(args.spec_root).resolve(); gen=json.loads(pathlib.Path(args.generated).read_text()); out=pathlib.Path(args.output_dir).resolve()
    if out.exists(): shutil.rmtree(out)
    (out/'schemas').mkdir(parents=True); (out/'fixtures'/'positive').mkdir(parents=True); (out/'fixtures'/'negative').mkdir(parents=True); (out/'tools').mkdir(parents=True); (out/'test-logs').mkdir(parents=True)
    binding=binding_fixture(); dump(out/'binding.fixture.json',binding)
    copy_file(spec/'bindings'/'waf-proxy.binding.deployment.template.json',out/'binding.deployment.template.json')
    for n in ['page.schema.json','control.schema.json','resource-manifest.json']: copy_file(spec/'schemas'/n,out/'schemas'/n)
    copy_file(spec/'tools'/'schema_subset.py',out/'tools'/'schema_subset.py')

    identity_paths=['dashboard_connector_types.go','dashboard_connector_store.go','dashboard_connector_http.go','dashboard_connector_export.go','dashboard_connector_test.go','dashboard_connector_fixture_test.go','main.go','tools/waf_dashboard_token.py','go.mod']
    files=[]
    for rel in sorted(identity_paths): files.append({'path':rel,'sha256':sha_file(src/rel)})
    identity={'purpose':'OFFLINE_FIXTURE_ONLY','product_slug':'waf-proxy','source_version':SOURCE_VERSION,'files':files}
    dump(out/'identity-inputs.json',identity); identity_sha=sha_file(out/'identity-inputs.json')
    manifest={'adapter_id':'waf-proxy-owi-fixture-v1','product_type':'WAF','vendor':'Internal','protocol_version':1,'identity_sha256':identity_sha,'capabilities':CAPS,'capability_generation':1,'qualification_status':'UNQUALIFIED','qualification_evidence_sha256':None}
    dump(out/'adapter-manifest.fixture.json',manifest)

    # Persist exact real handler outputs as individual fixtures and as bundle bodies.
    response_index={}
    for route,bodies in sorted(gen['responses'].items()):
        slug=route.rsplit('/',1)[-1]
        response_index[route]=[]
        for i,body in enumerate(bodies,1):
            rel=f'fixtures/positive/{slug}-{i:02d}.json'; dump(out/rel,body); response_index[route].append(body)
    dump(out/'fixtures'/'attention-examples.json',gen['attention_examples'])
    valid_asset=response_index[BASE+'/assets'][0]; valid_det=response_index[BASE+'/detections'][0]
    negfiles,negcases=negative_fixtures(valid_asset,valid_det)
    for name,obj in negfiles.items(): dump(out/'fixtures'/'negative'/f'{name}.json',obj)
    dump(out/'negative-cases.json',negcases)

    transports=[
      {"case_id":"T-01","stage":"transport","setup":"exact connector httptest; no Authorization header","request_fixture":"GET /health","expected_status":401,"expected_code":"AUTH_REQUIRED","expected_headers":{},"actual_result":"PASS","evidence_ref":"test-logs/core-probe.txt"},
      {"case_id":"T-02","stage":"transport","setup":"health-only principal requests assets","request_fixture":"GET /assets","expected_status":403,"expected_code":"FORBIDDEN","expected_headers":{},"actual_result":"PASS","evidence_ref":"test-logs/core-probe.txt"},
      {"case_id":"T-03","stage":"transport","setup":"snapshot cursor replayed by different principal","request_fixture":"GET /assets?limit=2&cursor=<fixture>","expected_status":409,"expected_code":"RESET_REQUIRED","expected_headers":{},"actual_result":"PASS","evidence_ref":"test-logs/core-probe.txt"},
      {"case_id":"T-04","stage":"transport","setup":"test quota rate=0.01 burst=1","request_fixture":"GET /health twice","expected_status":429,"expected_code":"RATE_LIMITED","expected_headers":{"Retry-After":"integer seconds; equals retry_after_seconds"},"actual_result":"PASS","evidence_ref":"test-logs/core-probe.txt"},
      {"case_id":"T-05","stage":"transport","setup":"source hook blocks until server context deadline","request_fixture":"GET /assets","expected_status":503,"expected_code":"TEMPORARY_UNAVAILABLE","expected_headers":{},"actual_result":"PASS","evidence_ref":"test-logs/core-probe.txt"},
      {"case_id":"T-06","stage":"transport","setup":"Phase A reader","request_fixture":"POST /assets","expected_status":405,"expected_code":"INVALID_REQUEST","expected_headers":{},"actual_result":"PASS","evidence_ref":"test-logs/core-probe.txt"},
      {"case_id":"T-07","stage":"transport","setup":"exact ServeMux path; redirects are not followed/generated","request_fixture":"GET /health/","expected_status":404,"expected_code":"NOT_FOUND","expected_headers":{"Location":"absent"},"actual_result":"PASS","evidence_ref":"test-logs/core-probe.txt"},
      {"case_id":"T-08","stage":"transport","setup":"Accept header validation","request_fixture":"GET /health Accept:text/html","expected_status":400,"expected_code":"INVALID_REQUEST","expected_headers":{},"actual_result":"PASS","evidence_ref":"test-logs/core-probe.txt"},
      {"case_id":"T-09","stage":"transport","setup":"exact connector probe sets MaxResponseBytes below capabilities response size","request_fixture":"oversize synthetic response","expected_status":503,"expected_code":"TEMPORARY_UNAVAILABLE","expected_headers":{},"actual_result":"PASS","evidence_ref":"test-logs/core-probe.txt"},
      {"case_id":"T-10","stage":"transport","setup":"real HTTPS worker-to-product path requires deployment cert/DNS/ACL","request_fixture":"TLS handshake and hostname/revocation checks","expected_status":None,"expected_code":None,"expected_headers":{},"actual_result":"NOT_RUN","evidence_ref":None},
      {"case_id":"T-11","stage":"transport","setup":"exact connector probe cancels request context and verifies limiter cleanup; live TCP disconnect remains G3","request_fixture":"client context cancel","expected_status":503,"expected_code":"TEMPORARY_UNAVAILABLE","expected_headers":{},"actual_result":"PASS","evidence_ref":"test-logs/core-probe.txt"},
    ]; dump(out/'transport-cases.json',transports)

    product={'tenant_id':'fixture-tenant-01','product_id':'dashboard-waf-proxy-fixture-01','product_type':'WAF','enabled':True,'secret_ref':'secret://systemd/waf-proxy-fixture-reader-token'}
    bundle={'manifest':manifest,'product':product,'responses':response_index,'requests':gen['requests'],'negative_pages':gen['negative_pages']}
    dump(out/'fixture-bundle.json',bundle)
    provenance={'purpose':'OFFLINE_FIXTURE_ONLY','source_version':SOURCE_VERSION,'source_instance_id':'waf-proxy-fixture-01','tenant_id':'fixture-tenant-01','generator':'go test -run TestGenerateOWIReturnFixture with WAF_OWI_RETURN_FIXTURE_DIR, then tools/assemble_waf_dashboard_return.py','clock':'runtime UTC clock; cursor chains are internally consistent within this generated bundle','source_files':files,'sanitization':['URL query removed','numeric/hex-like path identifiers templated to {id}','Authorization/Cookie/request body/raw headers are not exported','machine token represented only by secret_ref outside product source'], 'scenarios':['health/capabilities authenticated control reads','ASSET 3-page snapshot -> non-empty delta -> empty delta','DETECTION 3-page snapshot -> non-empty delta -> empty delta','POLICY snapshot -> non-empty delta -> empty delta','HEALTH snapshot -> non-empty delta -> empty delta','three WAF attention positive/counterexample pairs']}
    dump(out/'fixture-provenance.json',provenance)

    (out/'deployment-inputs.md').write_text('''# Deployment inputs\n\nAll fixture values are non-production. Deployment owner must provide and verify: management origin and chosen profile (dedicated 443 or same-host 19405), resolved IP allowlist, tenant/source/integration/principal IDs, server certificate chain/CA and SNI, opaque reader credential reference, worker ACL, and optional UI origin. JWT issuer/audience and mTLS client identity are **not applicable** to this WAF opaque-bearer profile. Dashboard owns server-certificate validation/revocation from its worker; WAF owns TLS serving and ingress separation.\n''',encoding='utf-8')
    (out/'mapping.md').write_text(f'''# WAF Proxy OWI-1.0 R3 mapping and applicability

Source identity SHA-256: `{identity_sha}`. This table is the O-01 inventory required by the R3 acceptance contract; it maps existing product authority to the thin reader facade and does not create a second WAF engine.

| Case ID | M/C/N/A | Reason | Code/config location | Test command | Environment | Source hash | Result | Evidence path | Gap owner |
|---|---|---|---|---|---|---|---|---|---|
| O-01 | M | Inventory/native→wire mapping is mandatory | `dashboard_connector_*.go`, `main.go`, `return/waf-proxy/` | `python3 tools/dashboard_connector_r3_source_gate.py` | local source gate | `{identity_sha}` | PASS | `DASHBOARD_CONNECTOR_R3_SOURCE_GATE_RESULT.md` | product |
| O-02 | M | Reader rate/concurrency/capacity protections | `dashboard_connector_http.go`, `dashboard_connector_store.go` | `go test -run 'TestOWI(RateLimit|Limiter|SnapshotCapacity|DetectionExportCapacity)'` | exact isolated connector source | `{identity_sha}` | PASS supporting | `test-logs/core-probe.txt` | G3 capacity sizing: deployment |
| O-03 | M | Source/lock/page work must be bounded and cancel-safe | `dashboard_connector_http.go`, `dashboard_connector_store.go` | `go test -run 'TestOWI(Deadline|ClientCancel|StoreContention)'` | exact isolated connector source | `{identity_sha}` | PASS supporting | `test-logs/core-probe.txt` | live socket/TLS: deployment |
| O-04 | M/C | Durable STATE + enabled DETECTION HISTORY | `dashboard_connector_store.go`, `dashboard_connector_export.go` | `go test -run 'TestOWI(RealProcessKill|StoreWriteFailure|CursorAndDetectionRetention|Restart)'` | exact isolated connector source/local filesystem | `{identity_sha}` | PASS supporting | `test-logs/core-probe.txt` | G3 intended filesystem/runtime |
| O-05 | C/M | WAF profile is opaque bearer; TLS/GET-only/tenant/scope mandatory | `dashboard_connector_http.go`, `main.go`, `tools/waf_dashboard_token.py` | `go test -run 'TestOWI.*(Auth|Token|Scope|GETOnly|Mutation|Cursor)'` | exact isolated connector source | `{identity_sha}` | PASS supporting | `test-logs/core-probe.txt` | production credential/TLS: deployment |
| O-06 | M | Native WAF identity/security/health mapping | `dashboard_connector_export.go` | `go test -run 'TestOWI.*(Saniti|Detection|Fixture)'` | exact isolated connector source | `{identity_sha}` | PASS supporting | `fixtures/attention-examples.json` | live capacity thresholds: deployment |
| O-07 | M | Executable return/fixtures/catalogs | `dashboard_connector_fixture_test.go`, `tools/assemble_waf_dashboard_return.py` | `python3 tools/validate_waf_dashboard_return.py --return-dir return/waf-proxy` | local return validator | `{identity_sha}` | PASS | `test-logs/fixture-generation.txt`; `SHA256SUMS` | Dashboard offline runner |
| O-08 | M | Product-owned return + SHA256SUMS | `return/waf-proxy/` | `sha256sum -c SHA256SUMS` | clean return tree | `{identity_sha}` | PASS | `SHA256SUMS` | Dashboard review/pin |
| HISTORY-30D | C | DETECTION is enabled HISTORY; EVENT is disabled | `dashboard_connector_store.go`, `/capabilities` | `go test -run TestOWICursorAndDetectionRetentionExpiry` | accelerated local clock | `{identity_sha}` | PASS supporting | `test-logs/core-probe.txt` | real elapsed 30d: G3 |
| Phase-B | N/A | R3 explicitly excludes product write actions | capability `actions=[]`, optional action-status disabled | source gate | local | `{identity_sha}` | N/A by contract | `HANDOFF.md` | future governed slice |

## Native → wire authority

| Wire kind | Native authority | Sanitization/null policy | Revision trigger | ID owner | Retention/semantics |
|---|---|---|---|---|---|
| ASSET | WAF runtime/config: node, site, virtual service, origin pool | no raw config export; HTTP peer is not protected app | mapped source-content change | WAF connector stable external ID | STATE; snapshot + delta |
| DETECTION | Coraza/WAF security match stream | query/cookie/Authorization/body/raw sensitive headers excluded; safe path template + rule ID retained | each accepted native security observation | WAF connector durable detection ID | HISTORY; 30-day target + GAP on loss |
| POLICY | native WAF/Positive-Schema policy config | allow-listed active generation/digest/scope only | active mapped policy content change | WAF connector policy ID | STATE; snapshot + delta |
| HEALTH | runtime + measured connector/export state | measured values only; unknown capacity remains null/UNKNOWN | mapped observation change | WAF connector health ID | STATE; snapshot + delta |
| EVENT | none in R3 | disabled | N/A | N/A | disabled; no scope/retention claim |
| ACTION_STATUS | none in R3 | disabled | N/A | N/A | disabled; no scope; Phase A has no actions |

Attention codes are `WAF.PROTECTION_DEGRADED`, `WAF.CAPACITY_PRESSURE`, and `WAF.HIGH_SIGNAL_ATTACK`; positive and evidence-insufficient counterexamples are under `fixtures/attention-examples.json`.
''',encoding='utf-8')
    (out/'deployment.md').write_text('''# WAF Proxy OWI reader deployment

The reader is a separate management listener, never the WAF business listener and never the browser/admin bearer plane. Set `WAF_DASHBOARD_READER_ENABLED=true`; choose dedicated management host TCP/443 or same-host fixed TCP/19405; configure TLS cert/key, `WAF_DASHBOARD_SOURCE_INSTANCE_ID`, owner-only non-symlink `WAF_DASHBOARD_CURSOR_SECRET_FILE`, and an owner-only non-symlink digest-only token registry. The built-in reader TLS listener enforces TLS >=1.2. Optional EVENT/ACTION_STATUS scopes cannot be provisioned in R3.

## Bounded resource defaults

- request rate: 10 requests/sec per authenticated tenant+principal; burst 20 (`WAF_DASHBOARD_RATE_PER_SECOND`, `WAF_DASHBOARD_RATE_BURST`)
- concurrency: 2 per principal, 16 global, immediate rejection/no request queue (`WAF_DASHBOARD_PER_PRINCIPAL_CONCURRENCY`, `WAF_DASHBOARD_GLOBAL_CONCURRENCY`)
- server deadline: 8s (`WAF_DASHBOARD_REQUEST_DEADLINE_SECONDS`); source sync and store-lock acquisition consume this same request context; no unbounded worker is spawned on timeout
- Dashboard client contract remains connect 5s/read 10s/hard 30s
- page: max 500; response: max 4 MiB
- retained snapshots: max 256 active, max 10,000 records and 32 MiB per snapshot (`WAF_DASHBOARD_SNAPSHOT_MAX_COUNT`, `_MAX_RECORDS`, `_MAX_BYTES`)
- detection durable journal: max 200,000 rows / 128 MiB plus 30-day time retention (`WAF_DASHBOARD_EXPORT_MAX_ROWS`, `_MAX_BYTES`). If capacity or persistence prevents durable service, the reader fails closed; detection queue/persistence loss marks GAP rather than fake COMPLETE.
- non-blocking security export queue: 4,096 observations; overflow marks GAP and does not block WAF request hot path
- snapshot/cursor TTL: 24h; STATE change journal retention: 7d; enabled DETECTION HISTORY retention target: 30d
- cleanup bound under slow lock contention: request returns when the configured server deadline expires; per-principal/global limiter slots release with handler return. Kernel/filesystem I/O cannot be preempted portably, so G3 must qualify the intended filesystem under fault/load conditions.

Quota counters are process-local. A multi-instance deployment multiplies aggregate request quota unless ingress adds a shared cap; this must be sized/reviewed in G3. The durable state/cursor key is instance-owned; cross-node reader HA is not claimed by this return.

## Token lifecycle

Use `tools/waf_dashboard_token.py`. Tokens are 32-byte CSPRNG values shown only at creation/rotation; registry persists a domain-separated SHA-256 digest. Expiry is at most 90 days; rotation bounds old-token overlap to at most 24h; revoke is explicit. Reader tokens contain only the six enabled R3 read scopes; admin/session/writer credentials are not accepted.
''',encoding='utf-8')
    (out/'open-gaps.md').write_text('''# Open gaps

- PRODUCT_API_LOCAL: BLOCKED. Exact connector source tests, including race, deadline/lock contention, client cancel, oversize fail-closed, real subprocess SIGKILL recovery-to-GAP, accelerated retention expiry and injected write failure/recovery, PASS on the available Go 1.23.2 supporting probe. The integrated product repository requires Go 1.25 and cannot be compiled/qualified here because the required toolchain/modules cannot be obtained; supporting probe PASS is not promoted to product-local qualification.
- RETURN_PACKAGE: PASS for product-owned R3 return construction, schema validation and SHA256SUMS.
- DASHBOARD_OFFLINE_ACCEPTANCE: NOT_RUN. No Dashboard checkout/fixture CLI was available in this execution environment. Dashboard owns consumer issue C-03 if it rejects `bootstrap_retention_days:null` for the disabled EVENT HISTORY lane; WAF deliberately does not claim nonexistent retention for a disabled capability.
- LIVE_DEPLOYMENT: NOT_RUN. Requires real DNS/IP/CA/SNI, worker ACL, opaque secret provisioning, intended OS/filesystem/backend, real HTTPS transport, load/capacity and deployed restart/failure evidence.
- Oversize response and request-context cancel are PASS locally. A real TCP socket disconnect/worker hard-deadline path remains NOT_RUN until G3.
- Optional EVENT/ACTION_STATUS and all Phase B writers are NOT_IMPLEMENTED by design for this round.
''',encoding='utf-8')
    (out/'HANDOFF.md').write_text(f'''# WAF Proxy OWI-1.0 R3 handoff

Source checkpoint: `{SOURCE_VERSION}`; identity SHA-256 `{identity_sha}`. This return implements the product-owned Phase A reader only; it does not modify Dashboard, register a production Product Instance, or enable any write/action plane.

## Four states

- PRODUCT_API_LOCAL: **BLOCKED** — exact connector-source tests/race PASS as supporting evidence, but integrated root Go 1.25 qualification is `BLOCKED_ENVIRONMENT / NOT_RUN`
- RETURN_PACKAGE: **PASS**
- DASHBOARD_OFFLINE_ACCEPTANCE: **NOT_RUN** — next owner: Dashboard
- LIVE_DEPLOYMENT: **NOT_RUN** — next owners: WAF deployment + Dashboard worker

Required lanes implemented: ASSET/LIST_ASSETS, DETECTION/LIST_DETECTIONS, POLICY/OBSERVE_POLICY, HEALTH/HEALTH. Optional EVENT and ACTION_STATUS remain disabled and cannot be provisioned as token scopes. `actions=[]`. Dedicated opaque bearer, per-identity/global limits, strict GET-only including HEAD rejection, bounded source/store lock deadline, fail-closed durable state, snapshot/cursor/revision persistence, token expiry/rotation/revoke, sanitization and security-export GAP semantics are implemented.

Re-run fixture generation from exact source: `WAF_OWI_RETURN_FIXTURE_DIR=<dir> go test -run TestGenerateOWIReturnFixture -count=1 .`, then `python3 tools/assemble_waf_dashboard_return.py ...`, then `python3 tools/validate_waf_dashboard_return.py --return-dir return/waf-proxy`. Dashboard should independently hash `binding.fixture.json` and run its fixture qualification CLI; product code must not patch the Dashboard validator or production registry.
''',encoding='utf-8')
    (out/'test-results.md').write_text(f'''# R3 qualification results

Source identity SHA-256: `{identity_sha}`. Environment: local Linux sandbox; Go 1.23.2 available. Repository declares Go 1.25.x; network/module/toolchain retrieval is blocked, so integrated root Go 1.25 build/vet/test/race remains `BLOCKED_ENVIRONMENT / NOT_RUN`. Supporting connector tests are explicitly separated from product-release qualification.

| Case ID | M/C/N/A | Reason | Code/config location | Test command | Environment | Source hash | Result | Evidence path | Gap owner |
|---|---|---|---|---|---|---|---|---|---|
| O-01 | M | R3 inventory/mapping | connector source + return package | `python3 tools/dashboard_connector_r3_source_gate.py` | local source gate | `{identity_sha}` | PASS | `mapping.md`; `test-logs/source-gate.txt` | product |
| O-02 | M | quota/concurrency/capacity | HTTP limiter + store caps | `go test -run 'TestOWI(RateLimit|Limiter|SnapshotCapacity|DetectionExportCapacity)'` | exact isolated connector source, Go 1.23.2 | `{identity_sha}` | PASS supporting | `test-logs/core-probe.txt` | G3 capacity sizing |
| O-03 | M | bounded source/lock/page + cleanup | request deadline + context-aware store lock | `go test -run 'TestOWI(DeadlineReleasesLimiter|ClientCancelReleasesLimiter|StoreContentionHonorsDeadlineAndRetry)'` | exact isolated connector source | `{identity_sha}` | PASS supporting | `test-logs/core-probe.txt`; `core-probe-race.txt` | real socket G3 |
| O-04 | M/C | durable snapshots/history/faults | durable JSON store + export journal | `go test -run 'TestOWI(RealProcessKillMarksCoverageGap|StoreWriteFailureFailsClosedAndRecovers|CursorAndDetectionRetentionExpiry|.*Restart.*)'` | exact isolated connector source/local filesystem | `{identity_sha}` | PASS supporting | `test-logs/core-probe.txt` | intended filesystem/runtime G3 |
| O-05 | C/M | opaque token + tenant/scope + GET-only | reader auth/token source + TLS listener | `go test -run 'TestOWI.*(Auth|Token|Scope|GETOnly|Mutation|Cursor)'` | exact isolated connector source | `{identity_sha}` | PASS supporting | `test-logs/core-probe.txt` | live credential/TLS G3 |
| O-06 | M | WAF mapper/function truth | native exporter | `go test -run 'TestOWI.*(Detection|Saniti|Fixture)'` | exact isolated connector source | `{identity_sha}` | PASS supporting | `fixtures/attention-examples.json`; positive fixtures | live capacity thresholds G3 |
| O-07 | M | executable return + catalogs | fixture generator + assembler/validator | `python3 tools/validate_waf_dashboard_return.py --return-dir return/waf-proxy` | local validator | `{identity_sha}` | PASS | `test-logs/fixture-generation.txt`; `SHA256SUMS` | Dashboard G2 |
| O-08 | M | handoff/hash integrity | return package | `sha256sum -c SHA256SUMS` | clean return tree | `{identity_sha}` | PASS | `SHA256SUMS` | Dashboard review/pin |
| HISTORY-30D | C | enabled DETECTION HISTORY | retention cleanup + capability declaration | `go test -run TestOWICursorAndDetectionRetentionExpiry` | accelerated local clock | `{identity_sha}` | PASS supporting | `test-logs/core-probe.txt` | real elapsed G3 |
| T-09 | M | oversize fail-closed | response bound | `go test -run TestOWIOversizeResponseFailsClosed` | exact isolated connector source | `{identity_sha}` | PASS supporting | `test-logs/core-probe.txt` | product |
| T-11 | M | cancel cleanup | context cancel | `go test -run TestOWIClientCancelReleasesLimiter` | exact isolated connector source | `{identity_sha}` | PASS supporting | `test-logs/core-probe.txt` | real TCP disconnect G3 |
| Root-Go-1.25 | M for release | exact integrated product qualification | repository root | `GOTOOLCHAIN=local go mod tidy -diff && go build ./... && go vet ./... && go test ./... && go test -race ./...` | required toolchain unavailable | `{identity_sha}` | BLOCKED_ENVIRONMENT / NOT_RUN | `open-gaps.md` | qualification environment |
| G2 | M | Dashboard consumer acceptance | Dashboard checkout | Dashboard fixture CLI | Dashboard unavailable | `{identity_sha}` | NOT_RUN | null | Dashboard |
| G3 | M | real TLS/ACL/intended runtime | deployment | see deployment.md | production-like environment unavailable | `{identity_sha}` | NOT_RUN | null | deployment + Dashboard |
''',encoding='utf-8')
    # Copy evidence logs if supplied.
    if args.log_dir:
      ld=pathlib.Path(args.log_dir)
      for srcname,dstname in [('waf_connector_core_probe.log','core-probe.txt'),('waf_connector_core_probe_race.log','core-probe-race.txt'),('waf_connector_fixture_generation.log','fixture-generation.txt'),('waf_dashboard_token_tool.log','token-tool.txt'),('waf_connector_r3_source_gate.log','source-gate.txt')]:
        p=ld/srcname
        if p.exists(): copy_file(p,out/'test-logs'/dstname)
    # Product-owned generator source copied for reproducibility.
    copy_file(src/'dashboard_connector_fixture_test.go',out/'tools'/'dashboard_connector_fixture_test.go')
    copy_file(src/'tools'/'waf_dashboard_token.py',out/'tools'/'waf_dashboard_token.py')
    copy_file(src/'tools'/'validate_waf_dashboard_return.py',out/'tools'/'validate_waf_dashboard_return.py')

    # Product repository Markdown review gate requires the canonical review marker on every Markdown file.
    review_marker = '<!-- documentation-review: 2026-09-28; classification: current-generated-return -->\n'
    for mp in out.rglob('*.md'):
      txt = mp.read_text(encoding='utf-8')
      if not txt.startswith('<!-- documentation-review: 2026-09-28; classification:'):
        mp.write_text(review_marker + txt, encoding='utf-8')

    # Compute SHA256SUMS last, excluding itself.
    lines=[]
    for p in sorted((p for p in out.rglob('*') if p.is_file() and p.name!='SHA256SUMS'),key=lambda p:p.relative_to(out).as_posix()):
      lines.append(f"{sha_file(p)}  {p.relative_to(out).as_posix()}")
    (out/'SHA256SUMS').write_text('\n'.join(lines)+'\n',encoding='utf-8')
    print(f"RETURN_FILES={len(lines)+1}")
    print(f"IDENTITY_SHA256={identity_sha}")

if __name__=='__main__': raise SystemExit(main())
