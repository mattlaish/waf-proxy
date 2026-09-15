#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: ./build-release-artifact.sh <output.zip> --version <label>
       [--test-summary <file>]
       [--variant source]
       [--source-date-epoch <unix-seconds>] [--require-reproducible]
       [--crs <path>] [--govulncheck-evidence <json>]
       [--minisign-key <secret-key>] [--minisign-public-key <public-key>] [--require-signature]

Builds a clean complete-source WAF ZIP, generates deterministic Phase 3 release
provenance/SBOM evidence, writes RELEASE_MANIFEST.txt, verifies the extracted
artifact against the source tree, and optionally creates a detached minisign
signature. Use --source-date-epoch (or SOURCE_DATE_EPOCH) for byte-reproducible
metadata; --require-reproducible makes it mandatory.
USAGE
}

[[ $# -ge 3 ]] || { usage >&2; exit 2; }
output=$1; shift
version=""; test_summary=""; variant="source"; source_date_epoch="${SOURCE_DATE_EPOCH:-}"
require_repro=0; crs=""; govuln=""; minisign_key=""; minisign_public_key=""; require_signature=0
while [[ $# -gt 0 ]]; do
  case "$1" in
    --version) [[ $# -ge 2 ]] || { usage >&2; exit 2; }; version=$2; shift 2 ;;
    --test-summary) [[ $# -ge 2 ]] || { usage >&2; exit 2; }; test_summary=$2; shift 2 ;;
    --variant) [[ $# -ge 2 ]] || { usage >&2; exit 2; }; variant=$2; shift 2 ;;
    --source-date-epoch) [[ $# -ge 2 ]] || { usage >&2; exit 2; }; source_date_epoch=$2; shift 2 ;;
    --require-reproducible) require_repro=1; shift ;;
    --crs) [[ $# -ge 2 ]] || { usage >&2; exit 2; }; crs=$2; shift 2 ;;
    --govulncheck-evidence) [[ $# -ge 2 ]] || { usage >&2; exit 2; }; govuln=$2; shift 2 ;;
    --minisign-key) [[ $# -ge 2 ]] || { usage >&2; exit 2; }; minisign_key=$2; shift 2 ;;
    --minisign-public-key) [[ $# -ge 2 ]] || { usage >&2; exit 2; }; minisign_public_key=$2; shift 2 ;;
    --require-signature) require_signature=1; shift ;;
    *) echo "ERROR: unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
done
[[ -n "$version" ]] || { echo "ERROR: --version is required" >&2; exit 2; }
case "$variant" in
  source) ;;
  portable-coraza|native-vectorscan)
    echo "ERROR: build-release-artifact.sh creates SOURCE_ARCHIVE only; binary identities require an actual binary artifact" >&2
    exit 2
    ;;
  *) echo "ERROR: invalid --variant: $variant" >&2; exit 2 ;;
esac
if [[ -n "$source_date_epoch" && ! "$source_date_epoch" =~ ^[0-9]+$ ]]; then
  echo "ERROR: source-date-epoch must be non-negative Unix seconds" >&2; exit 2
fi
if [[ $require_repro -eq 1 && -z "$source_date_epoch" ]]; then
  echo "ERROR: reproducible release required but SOURCE_DATE_EPOCH/--source-date-epoch is unset" >&2; exit 1
fi
command -v python3 >/dev/null 2>&1 || { echo "ERROR: python3 is required" >&2; exit 1; }

root=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
output=$(python3 -c 'import os,sys; print(os.path.abspath(sys.argv[1]))' "$output")
if [[ -n "$test_summary" ]]; then
  test_summary=$(python3 -c 'import os,sys; print(os.path.abspath(sys.argv[1]))' "$test_summary")
  [[ -f "$test_summary" ]] || { echo "ERROR: test summary not found: $test_summary" >&2; exit 1; }
fi
if [[ -n "$crs" ]]; then crs=$(python3 -c 'import os,sys; print(os.path.abspath(sys.argv[1]))' "$crs"); fi
if [[ -n "$govuln" ]]; then
  govuln=$(python3 -c 'import os,sys; print(os.path.abspath(sys.argv[1]))' "$govuln")
  [[ -f "$govuln" ]] || { echo "ERROR: govulncheck evidence not found: $govuln" >&2; exit 1; }
fi
if [[ -n "$minisign_key" ]]; then
  minisign_key=$(python3 -c 'import os,sys; print(os.path.abspath(sys.argv[1]))' "$minisign_key")
  [[ -f "$minisign_key" ]] || { echo "ERROR: minisign key not found" >&2; exit 1; }
  command -v minisign >/dev/null 2>&1 || { echo "ERROR: minisign requested but executable unavailable" >&2; exit 1; }
fi
if [[ -n "$minisign_public_key" ]]; then
  minisign_public_key=$(python3 -c 'import os,sys; print(os.path.abspath(sys.argv[1]))' "$minisign_public_key")
  [[ -f "$minisign_public_key" ]] || { echo "ERROR: minisign public key not found" >&2; exit 1; }
fi
if [[ $require_signature -eq 1 && ( -z "$minisign_key" || -z "$minisign_public_key" ) ]]; then
  echo "ERROR: --require-signature requires both --minisign-key and --minisign-public-key" >&2; exit 1
fi
[[ "$output" != "$root"/* ]] || { echo "ERROR: output ZIP must be outside the source tree" >&2; exit 1; }
mkdir -p "$(dirname "$output")"; rm -f "$output" "$output.minisig"
stage=$(mktemp -d); trap 'rm -rf "$stage"' EXIT

python3 - "$root" "$stage" <<'PY'
import pathlib, shutil, sys
src=pathlib.Path(sys.argv[1]); dst=pathlib.Path(sys.argv[2])
excluded_roots={'.git','bin','.cache','__pycache__','release-evidence'}
excluded_names={'RELEASE_MANIFEST.txt','SHA256SUMS.txt','waf-proxy','waf-tlsfront','wafctl','wafqualify','phase1-qualification.json','BUILD_PROVENANCE.json','BUILD_SHA256SUMS.txt'}
secret_names={'.env','.npmrc','.pypirc','id_rsa','id_ed25519'}
secret_suffixes={'.key','.p12','.pfx','.jks','.keystore'}
for p in src.rglob('*'):
    rel=p.relative_to(src)
    if any(part in excluded_roots for part in rel.parts): continue
    if p.is_symlink(): raise SystemExit(f"ERROR: symlink not allowed in complete source package: {rel}")
    if p.is_dir(): continue
    if rel.name in excluded_names or rel.suffix in {'.zip','.patch'} or rel.name.endswith(('.log','.tmp')): continue
    if rel.name in secret_names or rel.suffix.lower() in secret_suffixes:
        raise SystemExit(f"ERROR: secret/private-key-like file refused from source release: {rel}")
    out=dst/rel; out.parent.mkdir(parents=True,exist_ok=True); shutil.copy2(p,out)
PY

# Generate machine-readable Phase 3 evidence before the release manifest so the
# evidence files themselves are covered by RELEASE_MANIFEST.txt.
evargs=(--root "$stage" --out "$stage/release-evidence" --release "$version" --variant "$variant")
[[ -n "$source_date_epoch" ]] && evargs+=(--source-date-epoch "$source_date_epoch")
[[ -n "$crs" ]] && evargs+=(--crs "$crs")
[[ -n "$govuln" ]] && evargs+=(--govulncheck-json "$govuln")
PYTHONDONTWRITEBYTECODE=1 python3 "$stage/tools/release_evidence.py" "${evargs[@]}"

if [[ -n "$source_date_epoch" ]]; then
  build_ts=$(python3 - "$source_date_epoch" <<'PY'
import datetime,sys
print(datetime.datetime.fromtimestamp(int(sys.argv[1]),datetime.timezone.utc).isoformat().replace('+00:00','Z'))
PY
)
  reproducible=true
else
  build_ts=$(TZ=UTC date '+%Y-%m-%dT%H:%M:%SZ')
  reproducible=false
fi
signing="NOT_CONFIGURED"
[[ -n "$minisign_key" ]] && signing="DETACHED_MINISIGN_REQUESTED"
python3 - "$stage" "$version" "$build_ts" "$test_summary" "$variant" "$reproducible" "$signing" <<'PY'
import hashlib,pathlib,sys
root=pathlib.Path(sys.argv[1]); version,ts,summary,variant,repro,signing=sys.argv[2:]
def sha(p):
 h=hashlib.sha256();
 with p.open('rb') as f:
  for c in iter(lambda:f.read(1024*1024),b''): h.update(c)
 return h.hexdigest()
files=sorted((p for p in root.rglob('*') if p.is_file()),key=lambda p:p.relative_to(root).as_posix())
lines=[f"Version: {version}",f"Build Timestamp: {ts}",f"Build Variant: {variant}",f"Reproducible Metadata: {repro}",f"Signing: {signing}",f"Packaged File Count Before Manifest: {len(files)}","","Tests:"]
if summary:
 text=pathlib.Path(summary).read_text(encoding='utf-8',errors='replace'); excerpt='\n'.join(text.splitlines()[:100]); lines.extend('  '+x for x in excerpt.splitlines())
else: lines.append('  See TESTING_RESULTS.md inside this artifact.')
lines.extend(['','Files:'])
for p in files: lines.append(f"{sha(p)}  {p.relative_to(root).as_posix()}")
(root/'RELEASE_MANIFEST.txt').write_text('\n'.join(lines)+'\n',encoding='utf-8')
PY

python3 - "$stage" "$output" "${source_date_epoch:-}" <<'PY'
import datetime,pathlib,stat,sys,zipfile
root=pathlib.Path(sys.argv[1]); out=sys.argv[2]; epoch=sys.argv[3]
if epoch:
 d=datetime.datetime.fromtimestamp(int(epoch),datetime.timezone.utc)
 # ZIP cannot represent dates before 1980 and stores two-second granularity.
 if d.year < 1980: d=datetime.datetime(1980,1,1,tzinfo=datetime.timezone.utc)
 zdt=(d.year,d.month,d.day,d.hour,d.minute,d.second-(d.second%2))
else:
 zdt=(2026,1,1,0,0,0)
with zipfile.ZipFile(out,'w',compression=zipfile.ZIP_DEFLATED,compresslevel=9) as zf:
 for p in sorted((x for x in root.rglob('*') if x.is_file()),key=lambda p:p.relative_to(root).as_posix()):
  rel=p.relative_to(root).as_posix(); info=zipfile.ZipInfo(rel); info.create_system=3; info.compress_type=zipfile.ZIP_DEFLATED; info.date_time=zdt
  mode=stat.S_IMODE(p.stat().st_mode); info.external_attr=(stat.S_IFREG|mode)<<16; zf.writestr(info,p.read_bytes())
PY

"$root/verify-release-artifact.sh" "$output" "$root"
if [[ -n "$minisign_key" ]]; then
  minisign -S -s "$minisign_key" -m "$output" -x "$output.minisig"
  if [[ -n "$minisign_public_key" ]]; then
    "$root/verify-release-signature.sh" "$output" "$output.minisig" "$minisign_public_key"
    echo "SIGNATURE VERIFIED $output.minisig"
  else
    echo "SIGNATURE CREATED_NOT_VERIFIED $output.minisig"
  fi
else
  echo "SIGNATURE NOT_CONFIGURED"
fi
sha256sum "$output"
