#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: ./verify-release-artifact.sh <release.zip> [source-tree]

Validates a WAF complete-source release ZIP after packaging. When source-tree
is supplied (default: current directory), every packaged source file is
compared byte-for-byte and mode-for-mode with that tree. RELEASE_MANIFEST.txt
is generated artifact metadata and is excluded from source-tree equality.
USAGE
}

[[ $# -ge 1 && $# -le 2 ]] || { usage >&2; exit 2; }
archive=$1
source_tree=${2:-.}

[[ -f "$archive" ]] || { echo "ERROR: archive not found: $archive" >&2; exit 1; }
[[ -d "$source_tree" ]] || { echo "ERROR: source tree not found: $source_tree" >&2; exit 1; }
command -v python3 >/dev/null 2>&1 || { echo "ERROR: python3 is required" >&2; exit 1; }
command -v bash >/dev/null 2>&1 || { echo "ERROR: bash is required" >&2; exit 1; }

archive=$(python3 -c 'import os,sys; print(os.path.abspath(sys.argv[1]))' "$archive")
source_tree=$(python3 -c 'import os,sys; print(os.path.abspath(sys.argv[1]))' "$source_tree")
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

python3 - "$archive" "$tmp/extracted" <<'PY'
import os, pathlib, stat, sys, zipfile
archive, out = sys.argv[1], sys.argv[2]
root = pathlib.Path(out)
root.mkdir(parents=True, exist_ok=True)
with zipfile.ZipFile(archive) as zf:
    bad = zf.testzip()
    if bad:
        raise SystemExit(f"ERROR: ZIP CRC failure: {bad}")
    infos = zf.infolist()
    if not infos:
        raise SystemExit("ERROR: empty ZIP")
    for info in infos:
        name = info.filename
        p = pathlib.PurePosixPath(name)
        if name.startswith(("/", "\\")) or p.is_absolute() or ".." in p.parts or "\\" in name:
            raise SystemExit(f"ERROR: unsafe archive path: {name}")
        mode = (info.external_attr >> 16) & 0xFFFF
        if stat.S_ISLNK(mode):
            raise SystemExit(f"ERROR: symbolic link not allowed in source release: {name}")
    zf.extractall(root)
    # Python's ZipFile.extractall does not restore Unix modes. Reapply the
    # stored archive mode so validation reflects what a normal Unix unzip
    # receives from external_attr.
    for info in infos:
        target = root / pathlib.PurePosixPath(info.filename)
        mode = (info.external_attr >> 16) & 0xFFFF
        if target.exists() and mode:
            os.chmod(target, stat.S_IMODE(mode))
PY

extracted="$tmp/extracted"
required=(
  AGENTS.md README.md AI_HANDOFF.md DEVELOPMENT_ROADMAP.md TESTING_RESULTS.md
  MANIFEST.md patch.md INSTALL.md RELEASE_PROCESS.md DEVELOPMENT.md TESTING.md
  build.sh build-release-artifact.sh verify-release-artifact.sh generate-release-evidence.sh
  sign-release-artifact.sh verify-release-signature.sh qualify-release-host.sh install.sh go.mod go.sum
)
for path in "${required[@]}"; do
  [[ -f "$extracted/$path" ]] || { echo "ERROR: required file missing: $path" >&2; exit 1; }
done

# Critical executable files must be materially script-like, not tiny documentation replacements.
for path in build.sh build-release-artifact.sh verify-release-artifact.sh generate-release-evidence.sh sign-release-artifact.sh verify-release-signature.sh qualify-release-host.sh install.sh upgrade.sh uninstall.sh waf-doctor.sh setup-interfaces.sh; do
  [[ -f "$extracted/$path" ]] || continue
  size=$(wc -c < "$extracted/$path")
  (( size >= 200 )) || { echo "ERROR: critical script implausibly small ($size bytes): $path" >&2; exit 1; }
  head -n 1 "$extracted/$path" | grep -Eq '^#!/(usr/bin/env bash|bin/bash)' || {
    echo "ERROR: unexpected shell shebang: $path" >&2; exit 1;
  }
  [[ -x "$extracted/$path" ]] || { echo "ERROR: executable mode lost: $path" >&2; exit 1; }
  bash -n "$extracted/$path"
done
if [[ -f "$extracted/benchmark/build.sh" ]]; then
  [[ -x "$extracted/benchmark/build.sh" ]] || { echo "ERROR: executable mode lost: benchmark/build.sh" >&2; exit 1; }
  bash -n "$extracted/benchmark/build.sh"
fi

[[ -f "$extracted/RELEASE_MANIFEST.txt" ]] || { echo "ERROR: RELEASE_MANIFEST.txt missing" >&2; exit 1; }

python3 - "$source_tree" "$extracted" <<'PY'
import hashlib, os, pathlib, stat, sys
src = pathlib.Path(sys.argv[1])
dst = pathlib.Path(sys.argv[2])

excluded_roots = {'.git', 'release-evidence'}
excluded_suffixes = {'.zip', '.patch'}
excluded_names = {'RELEASE_MANIFEST.txt', 'SHA256SUMS.txt'}

def ignored(rel: pathlib.PurePath):
    if any(part in excluded_roots for part in rel.parts):
        return True
    if rel.name in excluded_names:
        return True
    if rel.suffix in excluded_suffixes:
        return True
    if rel.parts and rel.parts[0] in {'bin', '.cache'}:
        return True
    if rel.name.endswith(('.log', '.tmp')):
        return True
    return False

def files(root):
    out = {}
    for p in root.rglob('*'):
        if not p.is_file() or p.is_symlink():
            continue
        rel = p.relative_to(root)
        if ignored(rel):
            continue
        out[rel.as_posix()] = p
    return out

def sha(p):
    h=hashlib.sha256()
    with p.open('rb') as f:
        for chunk in iter(lambda:f.read(1024*1024), b''):
            h.update(chunk)
    return h.hexdigest()

sfiles=files(src)
dfiles=files(dst)
if set(sfiles) != set(dfiles):
    missing=sorted(set(sfiles)-set(dfiles))
    extra=sorted(set(dfiles)-set(sfiles))
    raise SystemExit(f"ERROR: source/package file-set mismatch; missing={missing[:20]} extra={extra[:20]}")
for rel in sorted(sfiles):
    a,b=sfiles[rel],dfiles[rel]
    if sha(a)!=sha(b):
        raise SystemExit(f"ERROR: source/package SHA-256 mismatch: {rel}")
    am=stat.S_IMODE(a.stat().st_mode); bm=stat.S_IMODE(b.stat().st_mode)
    if am!=bm:
        raise SystemExit(f"ERROR: source/package mode mismatch: {rel}: {am:o}!={bm:o}")

manifest=dst/'RELEASE_MANIFEST.txt'
text=manifest.read_text(encoding='utf-8')
if 'Version:' not in text or 'Build Timestamp:' not in text or '\nFiles:\n' not in text or '\nTests:\n' not in text:
    raise SystemExit('ERROR: release manifest missing required metadata sections')
section=text.split('\nFiles:\n',1)[1]
entries={}
for line in section.splitlines():
    if not line.strip():
        continue
    parts=line.split('  ',1)
    if len(parts)!=2 or len(parts[0])!=64:
        raise SystemExit(f"ERROR: malformed manifest hash line: {line!r}")
    entries[parts[1]]=parts[0]
all_packaged={}
for p in dst.rglob('*'):
    if p.is_file() and p.name != 'RELEASE_MANIFEST.txt': all_packaged[p.relative_to(dst).as_posix()]=p
expected=set(all_packaged)
if set(entries)!=expected:
    raise SystemExit(f"ERROR: manifest file-set mismatch; missing={sorted(expected-set(entries))[:20]} extra={sorted(set(entries)-expected)[:20]}")
for rel, expected_sha in entries.items():
    if sha(all_packaged[rel]) != expected_sha:
        raise SystemExit(f"ERROR: manifest SHA-256 mismatch: {rel}")
# Phase 3 supply-chain evidence is mandatory artifact metadata.
ev=dst/'release-evidence'
required_ev=['versions.json','provenance.json','sbom.spdx.json','sbom.cdx.json','govulncheck-status.json','SHA256SUMS.txt']
for name in required_ev:
    if not (ev/name).is_file(): raise SystemExit(f'ERROR: release evidence missing: {name}')
import json
versions=json.loads((ev/'versions.json').read_text())
if versions.get('flavor') not in {'portable','native'}: raise SystemExit('ERROR: invalid release flavor evidence')
spdx=json.loads((ev/'sbom.spdx.json').read_text()); cdx=json.loads((ev/'sbom.cdx.json').read_text())
if spdx.get('spdxVersion')!='SPDX-2.3': raise SystemExit('ERROR: SPDX version mismatch')
if cdx.get('bomFormat')!='CycloneDX' or cdx.get('specVersion')!='1.5': raise SystemExit('ERROR: CycloneDX version mismatch')
# Independently verify evidence checksums.
for line in (ev/'SHA256SUMS.txt').read_text().splitlines():
    digest,name=line.split('  ',1); p=ev/name
    if not p.is_file() or sha(p)!=digest: raise SystemExit(f'ERROR: release evidence checksum mismatch: {name}')
print(f"ARTIFACT_INTEGRITY_PASS source_files={len(dfiles)} packaged_files={len(all_packaged)} flavor={versions.get('flavor')}")
PY
