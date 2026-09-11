#!/usr/bin/env bash
set -euo pipefail
usage(){ echo 'Usage: ./generate-release-evidence.sh <output-dir> --version <label> --flavor portable|native [--crs-version <version>] [--require-govulncheck]' >&2; }
[[ $# -ge 5 ]] || { usage; exit 2; }
out=$1; shift
version=''; flavor=''; crs_version=''; require_vuln=0
while [[ $# -gt 0 ]]; do
 case "$1" in
  --version) version=${2:-}; shift 2;;
  --flavor) flavor=${2:-}; shift 2;;
  --crs-version) crs_version=${2:-}; shift 2;;
  --require-govulncheck) require_vuln=1; shift;;
  *) echo "ERROR: unknown argument $1" >&2; usage; exit 2;;
 esac
done
[[ -n "$version" ]] || { echo 'ERROR: --version required' >&2; exit 2; }
[[ "$flavor" == portable || "$flavor" == native ]] || { echo 'ERROR: --flavor must be portable or native' >&2; exit 2; }
root=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
rm -rf "$out"; mkdir -p "$out"
coraza=$(awk '$1=="require" && $2=="github.com/corazawaf/coraza/v3"{print $3; exit} $1=="github.com/corazawaf/coraza/v3"{print $2; exit}' "$root/go.mod")
go_directive=$(awk '$1=="go"{print $2; exit}' "$root/go.mod")
host_go=$(cd /tmp && GOTOOLCHAIN=local go version 2>/dev/null | awk '{print $3}' || true)
[[ -n "$host_go" ]] || host_go=unavailable
nginx_ver=$(nginx -v 2>&1 | head -n1 || true); [[ -n "$nginx_ver" ]] || nginx_ver=unavailable
openssl_ver=$(openssl version 2>/dev/null | head -n1 || true); [[ -n "$openssl_ver" ]] || openssl_ver=unavailable
vectorscan_ver='not-included'; vectorscan_provenance='portable-coraza-only'
if [[ "$flavor" == native ]]; then
  if ! command -v pkg-config >/dev/null 2>&1 || ! pkg-config --exists libhs; then
    echo 'BLOCKED: native flavor requires verified pkg-config libhs' >&2; exit 3
  fi
  vectorscan_ver=$(pkg-config --modversion libhs)
  vectorscan_provenance=${WAF_VECTORSCAN_PROVENANCE_ACK:-pkg-config:libhs}
fi
[[ -n "$crs_version" ]] || crs_version=${WAF_CRS_VERSION:-not-embedded}
if [[ -n "${SOURCE_DATE_EPOCH:-}" ]]; then
  ts=$(date -u -d "@${SOURCE_DATE_EPOCH}" '+%Y-%m-%dT%H:%M:%SZ')
else
  ts=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
fi
export WAF_EVID_VERSION="$version" WAF_EVID_FLAVOR="$flavor" WAF_EVID_TS="$ts" WAF_EVID_GO_DIRECTIVE="$go_directive" WAF_EVID_HOST_GO="$host_go" WAF_EVID_CORAZA="$coraza" WAF_EVID_CRS="$crs_version" WAF_EVID_VECTORSCAN="$vectorscan_ver" WAF_EVID_VECTORSCAN_PROV="$vectorscan_provenance" WAF_EVID_NGINX="$nginx_ver" WAF_EVID_OPENSSL="$openssl_ver"
python3 - "$root" "$out" <<'PY'
import json, os, pathlib, re, hashlib, sys, uuid
root=pathlib.Path(sys.argv[1]); out=pathlib.Path(sys.argv[2])
def env(k): return os.environ[k]
versions={
 "schema_version":1,"version":env('WAF_EVID_VERSION'),"flavor":env('WAF_EVID_FLAVOR'),"generated_at":env('WAF_EVID_TS'),
 "go":{"module_directive":env('WAF_EVID_GO_DIRECTIVE'),"packaging_host":env('WAF_EVID_HOST_GO')},
 "coraza":env('WAF_EVID_CORAZA'),"crs":env('WAF_EVID_CRS'),
 "vectorscan":{"version":env('WAF_EVID_VECTORSCAN'),"provenance":env('WAF_EVID_VECTORSCAN_PROV')},
 "nginx":env('WAF_EVID_NGINX'),"openssl":env('WAF_EVID_OPENSSL')}
(out/'versions.json').write_text(json.dumps(versions,indent=2,sort_keys=True)+'\n')
# Parse module requirements without invoking Go/network.
text=(root/'go.mod').read_text(); deps=[]; in_req=False
for raw in text.splitlines():
 s=raw.strip()
 if s=='require (': in_req=True; continue
 if in_req and s==')': in_req=False; continue
 if s.startswith('require '): s=s[len('require '):]
 elif not in_req: continue
 s=s.split('//',1)[0].strip()
 parts=s.split()
 if len(parts)>=2: deps.append((parts[0],parts[1]))
# De-duplicate while preserving sorted deterministic output.
deps=sorted(set(deps))
name='waf-proxy'; ver=env('WAF_EVID_VERSION'); ts=env('WAF_EVID_TS'); flavor=env('WAF_EVID_FLAVOR')
spdx_pkgs=[{"SPDXID":"SPDXRef-Package-waf-proxy","name":name,"versionInfo":ver,"downloadLocation":"NOASSERTION","filesAnalyzed":False,"licenseConcluded":"NOASSERTION","licenseDeclared":"NOASSERTION"}]
rels=[]
for i,(m,v) in enumerate(deps,1):
 sid=f"SPDXRef-GoModule-{i}"
 spdx_pkgs.append({"SPDXID":sid,"name":m,"versionInfo":v,"downloadLocation":"NOASSERTION","filesAnalyzed":False,"licenseConcluded":"NOASSERTION","licenseDeclared":"NOASSERTION","externalRefs":[{"referenceCategory":"PACKAGE-MANAGER","referenceType":"purl","referenceLocator":f"pkg:golang/{m}@{v}"}]})
 rels.append({"spdxElementId":"SPDXRef-Package-waf-proxy","relationshipType":"DEPENDS_ON","relatedSpdxElement":sid})
if flavor=='native':
 sid='SPDXRef-Native-libhs'; vv=env('WAF_EVID_VECTORSCAN')
 spdx_pkgs.append({"SPDXID":sid,"name":"libhs","versionInfo":vv,"downloadLocation":"NOASSERTION","filesAnalyzed":False,"licenseConcluded":"NOASSERTION","licenseDeclared":"NOASSERTION"})
 rels.append({"spdxElementId":"SPDXRef-Package-waf-proxy","relationshipType":"DEPENDS_ON","relatedSpdxElement":sid})
ns_hash=hashlib.sha256((name+'|'+ver+'|'+flavor).encode()).hexdigest()
spdx={"spdxVersion":"SPDX-2.3","dataLicense":"CC0-1.0","SPDXID":"SPDXRef-DOCUMENT","name":f"{name}-{flavor}-{ver}","documentNamespace":f"https://waf-proxy.invalid/spdx/{ns_hash}","creationInfo":{"created":ts,"creators":["Tool: waf-proxy/generate-release-evidence.sh"]},"packages":spdx_pkgs,"relationships":rels}
(out/'sbom.spdx.json').write_text(json.dumps(spdx,indent=2,sort_keys=True)+'\n')
components=[]
for m,v in deps:
 components.append({"type":"library","name":m,"version":v,"purl":f"pkg:golang/{m}@{v}"})
if flavor=='native': components.append({"type":"library","name":"libhs","version":env('WAF_EVID_VECTORSCAN'),"properties":[{"name":"waf-proxy:provenance","value":env('WAF_EVID_VECTORSCAN_PROV')}]})
serial=uuid.uuid5(uuid.NAMESPACE_URL, name+"|"+ver+"|"+flavor)
cdx={"bomFormat":"CycloneDX","specVersion":"1.5","serialNumber":f"urn:uuid:{serial}","version":1,"metadata":{"timestamp":ts,"component":{"type":"application","name":name,"version":ver,"properties":[{"name":"waf-proxy:release-flavor","value":flavor}]}},"components":components}
(out/'sbom.cdx.json').write_text(json.dumps(cdx,indent=2,sort_keys=True)+'\n')
prov={"schema_version":1,"version":ver,"flavor":flavor,"generated_at":ts,"source":{"module":"waf-proxy","go_mod_sha256":hashlib.sha256((root/'go.mod').read_bytes()).hexdigest(),"go_sum_sha256":hashlib.sha256((root/'go.sum').read_bytes()).hexdigest()},"truth_boundary":{"sbom_source":"go.mod plus native libhs when flavor=native","binary_sbom":False,"real_release_host_qualification":False}}
(out/'provenance.json').write_text(json.dumps(prov,indent=2,sort_keys=True)+'\n')
PY
# govulncheck is deliberately a separate truth-bearing status, never inferred from SBOM generation.
status=NOT_RUN; rc=0; reason='govulncheck executable unavailable'
if command -v govulncheck >/dev/null 2>&1; then
  set +e
  (cd "$root" && GOTOOLCHAIN=local govulncheck -json ./...) >"$out/govulncheck.json" 2>"$out/govulncheck.stderr"
  rc=$?
  set -e
  if [[ $rc -eq 0 ]]; then status=PASS; reason='govulncheck completed'; else status=BLOCKED_OR_FAILED; reason="govulncheck exit $rc"; fi
else
  printf '' >"$out/govulncheck.json"; printf '%s\n' "$reason" >"$out/govulncheck.stderr"
fi
python3 - "$out/govulncheck-status.json" "$status" "$rc" "$reason" <<'PY'
import json,sys
p,status,rc,reason=sys.argv[1:]
open(p,'w').write(json.dumps({'status':status,'exit_code':int(rc),'reason':reason},indent=2,sort_keys=True)+'\n')
PY
if [[ $require_vuln -eq 1 && "$status" != PASS ]]; then echo "BLOCKED: required govulncheck status=$status" >&2; exit 3; fi
( cd "$out" && sha256sum versions.json provenance.json sbom.spdx.json sbom.cdx.json govulncheck.json govulncheck.stderr govulncheck-status.json > SHA256SUMS.txt )
