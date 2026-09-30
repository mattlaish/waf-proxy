#!/usr/bin/env python3
from __future__ import annotations
import argparse, hashlib, json, pathlib, sys
try:
    from jsonschema import Draft202012Validator
except Exception as e:
    raise SystemExit(f"jsonschema required: {e}")

def sha(p): return hashlib.sha256(p.read_bytes()).hexdigest()
def load(p): return json.loads(p.read_text(encoding='utf-8'))
def require(cond,msg):
    if not cond: raise AssertionError(msg)

def main():
    ap=argparse.ArgumentParser(); ap.add_argument('--return-dir',required=True); a=ap.parse_args(); r=pathlib.Path(a.return_dir).resolve()
    # SHA256SUMS covers everything except itself.
    lines=(r/'SHA256SUMS').read_text().splitlines(); seen=set()
    for line in lines:
        h,rel=line.split('  ',1); p=r/rel; require(p.is_file(),f'missing {rel}'); require(sha(p)==h,f'hash mismatch {rel}'); seen.add(rel)
    files={p.relative_to(r).as_posix() for p in r.rglob('*') if p.is_file() and p.name!='SHA256SUMS'}
    require(seen==files,f'SHA256SUMS set mismatch missing={sorted(files-seen)} extra={sorted(seen-files)}')
    b=load(r/'binding.fixture.json')
    binding_keys=set('contract schema_version status product_slug source_product source_instance_id tenant_id integration_instance_id enabled qualification network_profile origin alternative_same_host_port ui_origin resolved_address_allowlist base_path tls auth endpoints limits action_plane deployment_role evidence'.split())
    require(set(b)==binding_keys,'binding top-level fields not exact'); require(b['status']=='RETURNED_UNQUALIFIED' and b['qualification']=='IMPLEMENTED_UNQUALIFIED' and b['enabled'] is False,'binding qualification/enabled truth')
    require(b['base_path']=='/api/integrations/dashboard/v1' and b['source_product']=='WAF' and b['auth']['profile']=='opaque_bearer','binding identity/contract')
    require(set(b['auth']['scopes'])==set(['dashboard:read:health','dashboard:read:capabilities','dashboard:read:assets','dashboard:read:detections','dashboard:read:policies','dashboard:read:health-observations']),'scope set mismatch')
    resources=b['endpoints']['resources']; require(len(resources)==6,'resource count'); enabled=[x for x in resources if x['enabled']]; require({x['capability'] for x in enabled}=={'LIST_ASSETS','LIST_DETECTIONS','OBSERVE_POLICY','HEALTH'},'enabled capability set')
    require(all(x['sync_modes']==[] for x in resources if not x['enabled']),'disabled route sync modes')
    m=load(r/'adapter-manifest.fixture.json'); require(set(m)==set('adapter_id product_type vendor protocol_version identity_sha256 capabilities capability_generation qualification_status qualification_evidence_sha256'.split()),'manifest nine fields')
    require(m['identity_sha256']==sha(r/'identity-inputs.json'),'identity hash mismatch'); require(m['qualification_status']=='UNQUALIFIED' and m['qualification_evidence_sha256'] is None,'manifest qualification truth')
    bun=load(r/'fixture-bundle.json'); require(set(bun)=={'manifest','product','responses','requests','negative_pages'},'bundle five fields'); require(bun['manifest']==m,'embedded manifest mismatch'); require(set(bun['product'])=={'tenant_id','product_id','product_type','enabled','secret_ref'},'product five fields')
    pagev=Draft202012Validator(load(r/'schemas/page.schema.json')); ctlv=Draft202012Validator(load(r/'schemas/control.schema.json'))
    positive=0
    for route,bodies in bun['responses'].items():
        for body in bodies:
            errs=list((ctlv if route.endswith('/health') or route.endswith('/capabilities') else pagev).iter_errors(body))
            require(not errs,f'schema errors {route}: {[e.message for e in errs[:3]]}')
            positive += 1
    for n in bun['negative_pages']:
        require(list(pagev.iter_errors(n['page'])),f"negative page unexpectedly valid: {n.get('route_name')}")
    # Ensure all request cursors follow exact preceding response cursor chain for each lane.
    positions={}
    for req in bun['requests']:
        route={'ASSET':'assets','DETECTION':'detections','POLICY':'policies','HEALTH':'health-observations'}[req['resource_kind']]
        path='/api/integrations/dashboard/v1/'+route; i=positions.get(path,0); require(i<len(bun['responses'][path]),f'no response body for request {path} #{i}')
        if i==0: require(req['cursor'] is None,f'first cursor not null {path}')
        else: require(req['cursor']==bun['responses'][path][i-1]['next_cursor'],f'cursor chain mismatch {path} #{i}')
        positions[path]=i+1
    require(all(positions.get(k,0)==len(v) for k,v in bun['responses'].items() if not k.endswith('/health') and not k.endswith('/capabilities')),'unused resource responses')
    neg=load(r/'negative-cases.json'); require(len(neg)>=15 and all(set(x)=={'case_id','stage','input_fixture','request','expected_error','reason'} for x in neg),'negative catalog shape')
    for x in neg: require((r/x['input_fixture']).is_file(),f"negative fixture missing {x['input_fixture']}")
    trans=load(r/'transport-cases.json'); require(len(trans)>=10 and all(set(x)=={'case_id','stage','setup','request_fixture','expected_status','expected_code','expected_headers','actual_result','evidence_ref'} for x in trans),'transport catalog shape')
    require(all(x['actual_result'] in {'PASS','FAIL','NOT_RUN','NOT_IMPLEMENTED'} for x in trans),'transport result enum')
    print('RETURN_PACKAGE=PASS'); print(f'FILES={len(files)+1}'); print(f'POSITIVE_RESPONSES={positive}'); print(f'NEGATIVE_CASES={len(neg)}'); print(f'TRANSPORT_CASES={len(trans)}'); print(f'IDENTITY_SHA256={m["identity_sha256"]}')
    return 0
if __name__=='__main__':
    try: raise SystemExit(main())
    except AssertionError as e: print('RETURN_PACKAGE=FAIL',e,file=sys.stderr); raise SystemExit(1)
