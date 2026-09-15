#!/usr/bin/env bash
# Negative packaging/truth-boundary tests for the mandatory artifact gate.
set -euo pipefail
[[ $# -ge 1 && $# -le 2 ]] || { echo "Usage: $0 <good.zip> [source-tree]" >&2; exit 2; }
archive=$(python3 -c 'import os,sys; print(os.path.abspath(sys.argv[1]))' "$1")
root=$(python3 -c 'import os,sys; print(os.path.abspath(sys.argv[1]))' "${2:-.}")
verifier="$root/verify-release-artifact.sh"
[[ -x "$verifier" ]] || { echo "ERROR: verifier not executable: $verifier" >&2; exit 1; }
[[ -f "$archive" ]] || { echo "ERROR: archive not found" >&2; exit 1; }
tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT

mutate() {
  local kind=$1 out=$2
  python3 - "$archive" "$out" "$kind" <<'PY'
import hashlib,json,pathlib,stat,sys,zipfile
src,dst,kind=sys.argv[1:]
with zipfile.ZipFile(src) as zin:
    entries=[(i,zin.read(i.filename)) for i in zin.infolist()]

data={i.filename:b for i,b in entries}
info_by={i.filename:i for i,_ in entries}

def sha_bytes(b): return hashlib.sha256(b).hexdigest()
def rewrite_release_hash(name):
    text=data['RELEASE_MANIFEST.txt'].decode()
    lines=[]
    for line in text.splitlines():
        if line.endswith('  '+name): line=f"{sha_bytes(data[name])}  {name}"
        lines.append(line)
    data['RELEASE_MANIFEST.txt']=('\n'.join(lines)+'\n').encode()

def sync_provenance_evidence_hash():
    prov=json.loads(data['release-evidence/PROVENANCE.json'])
    prov['release_evidence_sha256']=sha_bytes(data['release-evidence/RELEASE_EVIDENCE.json'])
    data['release-evidence/PROVENANCE.json']=(json.dumps(prov,indent=2,sort_keys=True)+'\n').encode()
    rewrite_release_hash('release-evidence/PROVENANCE.json')

extra=[]
if kind=='mode':
    zi=info_by['install.sh']; zi.external_attr=(stat.S_IFREG|0o644)<<16
elif kind=='replace':
    data['install.sh']=b'# README replacement\nnot an installer\n'
elif kind=='traversal': extra.append(('../escape.txt',b'x',None))
elif kind=='secret': extra.append(('.env',b'WAF_ADMIN_TOKEN=should-not-ship\n',None))
elif kind=='symlink': extra.append(('evil-link',b'/etc/passwd','symlink'))
elif kind=='duplicate': extra.append(('README.md',data['README.md'],'duplicate'))
elif kind=='manifest-omit':
    text=data['RELEASE_MANIFEST.txt'].decode().splitlines()
    removed=False; out=[]
    for line in text:
        if not removed and line.endswith('  README.md'):
            removed=True; continue
        out.append(line)
    data['RELEASE_MANIFEST.txt']=('\n'.join(out)+'\n').encode()
elif kind=='evidence-forge':
    ev=json.loads(data['release-evidence/RELEASE_EVIDENCE.json'])
    ev['govulncheck']['status']='PASS'
    ev['govulncheck']['reason']='forged negative test'
    ev['govulncheck']['tool_version']='NOT_AVAILABLE'
    data['release-evidence/RELEASE_EVIDENCE.json']=(json.dumps(ev,indent=2,sort_keys=True)+'\n').encode()
    rewrite_release_hash('release-evidence/RELEASE_EVIDENCE.json'); sync_provenance_evidence_hash()
elif kind=='artifact-type':
    ev=json.loads(data['release-evidence/RELEASE_EVIDENCE.json']); ev['artifact_type']='NATIVE_BINARY'; ev['build_variant']='native-vectorscan'
    data['release-evidence/RELEASE_EVIDENCE.json']=(json.dumps(ev,indent=2,sort_keys=True)+'\n').encode()
    rewrite_release_hash('release-evidence/RELEASE_EVIDENCE.json'); sync_provenance_evidence_hash()
elif kind=='provenance-mismatch':
    prov=json.loads(data['release-evidence/PROVENANCE.json']); prov['source_manifest_sha256']='0'*64
    data['release-evidence/PROVENANCE.json']=(json.dumps(prov,indent=2,sort_keys=True)+'\n').encode(); rewrite_release_hash('release-evidence/PROVENANCE.json')
elif kind=='source-manifest-omit':
    lines=data['release-evidence/SOURCE_MANIFEST.sha256'].decode().splitlines()
    removed=False; kept=[]
    for line in lines:
        if not removed and line.endswith('  README.md'):
            removed=True; continue
        kept.append(line)
    data['release-evidence/SOURCE_MANIFEST.sha256']=('\n'.join(kept)+'\n').encode()
    smsha=sha_bytes(data['release-evidence/SOURCE_MANIFEST.sha256'])
    ev=json.loads(data['release-evidence/RELEASE_EVIDENCE.json']); ev['source_manifest_sha256']=smsha; ev['govulncheck']['source_manifest_sha256']=smsha
    data['release-evidence/RELEASE_EVIDENCE.json']=(json.dumps(ev,indent=2,sort_keys=True)+'\n').encode()
    prov=json.loads(data['release-evidence/PROVENANCE.json']); prov['source_manifest_sha256']=smsha; prov['release_evidence_sha256']=sha_bytes(data['release-evidence/RELEASE_EVIDENCE.json'])
    data['release-evidence/PROVENANCE.json']=(json.dumps(prov,indent=2,sort_keys=True)+'\n').encode()
    for name in ['release-evidence/SOURCE_MANIFEST.sha256','release-evidence/RELEASE_EVIDENCE.json','release-evidence/PROVENANCE.json']:
        rewrite_release_hash(name)
elif kind=='sbom-drift':
    cdx=json.loads(data['release-evidence/sbom.cyclonedx.json'])
    if cdx.get('components'):
        cdx['components'][0]['version']='v0.0.0-forged'
    data['release-evidence/sbom.cyclonedx.json']=(json.dumps(cdx,indent=2,sort_keys=True)+'\n').encode()
    prov=json.loads(data['release-evidence/PROVENANCE.json']); prov['cyclonedx_sha256']=sha_bytes(data['release-evidence/sbom.cyclonedx.json'])
    data['release-evidence/PROVENANCE.json']=(json.dumps(prov,indent=2,sort_keys=True)+'\n').encode()
    rewrite_release_hash('release-evidence/sbom.cyclonedx.json'); rewrite_release_hash('release-evidence/PROVENANCE.json')

with zipfile.ZipFile(dst,'w',compression=zipfile.ZIP_DEFLATED) as zout:
    for info,_ in entries:
        zout.writestr(info,data[info.filename])
    for name,b,typ in extra:
        if typ=='symlink':
            zi=zipfile.ZipInfo(name); zi.create_system=3; zi.external_attr=(stat.S_IFLNK|0o777)<<16; zout.writestr(zi,b)
        elif typ=='duplicate':
            zout.writestr(info_by[name],b)
        else: zout.writestr(name,b)
PY
}

if [[ -n "${WAF_NEGATIVE_KINDS:-}" ]]; then
  read -r -a kinds <<<"$WAF_NEGATIVE_KINDS"
else
  kinds=(traversal secret symlink duplicate mode replace manifest-omit source-manifest-omit evidence-forge artifact-type provenance-mismatch sbom-drift)
fi
for kind in "${kinds[@]}"; do
  bad="$tmp/$kind.zip"; mutate "$kind" "$bad"
  if "$verifier" "$bad" "$root" >/dev/null 2>&1; then
    echo "FAIL negative test unexpectedly accepted: $kind" >&2; exit 1
  fi
  echo "PASS negative rejection: $kind"
done
