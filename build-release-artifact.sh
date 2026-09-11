#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: ./build-release-artifact.sh <output.zip> --version <label> [--flavor portable|native] [--test-summary <file>] [--crs-version <version>] [--require-govulncheck]

Builds a clean complete-source WAF ZIP, writes RELEASE_MANIFEST.txt inside the
artifact, then verifies the extracted ZIP against the current source tree.
USAGE
}

[[ $# -ge 3 ]] || { usage >&2; exit 2; }
output=$1; shift
version=""
test_summary=""
flavor="portable"
crs_version=""
require_govulncheck=0
while [[ $# -gt 0 ]]; do
  case "$1" in
    --version) [[ $# -ge 2 ]] || { usage >&2; exit 2; }; version=$2; shift 2 ;;
    --test-summary) [[ $# -ge 2 ]] || { usage >&2; exit 2; }; test_summary=$2; shift 2 ;;
    --flavor) [[ $# -ge 2 ]] || { usage >&2; exit 2; }; flavor=$2; shift 2 ;;
    --crs-version) [[ $# -ge 2 ]] || { usage >&2; exit 2; }; crs_version=$2; shift 2 ;;
    --require-govulncheck) require_govulncheck=1; shift ;;
    *) echo "ERROR: unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
done
[[ -n "$version" ]] || { echo "ERROR: --version is required" >&2; exit 2; }
[[ "$flavor" == portable || "$flavor" == native ]] || { echo "ERROR: --flavor must be portable or native" >&2; exit 2; }
command -v python3 >/dev/null 2>&1 || { echo "ERROR: python3 is required" >&2; exit 1; }

root=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
output=$(python3 -c 'import os,sys; print(os.path.abspath(sys.argv[1]))' "$output")
if [[ -n "$test_summary" ]]; then
  test_summary=$(python3 -c 'import os,sys; print(os.path.abspath(sys.argv[1]))' "$test_summary")
  [[ -f "$test_summary" ]] || { echo "ERROR: test summary not found: $test_summary" >&2; exit 1; }
fi
[[ "$output" != "$root"/* ]] || { echo "ERROR: output ZIP must be outside the source tree" >&2; exit 1; }
mkdir -p "$(dirname "$output")"
rm -f "$output"
stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT

python3 - "$root" "$stage" <<'PY'
import pathlib, shutil, sys
src=pathlib.Path(sys.argv[1]); dst=pathlib.Path(sys.argv[2])
excluded_roots={'.git','bin','.cache','release-evidence'}
excluded_names={'RELEASE_MANIFEST.txt','SHA256SUMS.txt'}
for p in src.rglob('*'):
    rel=p.relative_to(src)
    if any(part in excluded_roots for part in rel.parts):
        continue
    if p.is_symlink():
        raise SystemExit(f"ERROR: symlink not allowed in complete source package: {rel}")
    if p.is_dir():
        continue
    if rel.name in excluded_names or rel.suffix in {'.zip','.patch'} or rel.name.endswith(('.log','.tmp')):
        continue
    out=dst/rel
    out.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(p,out)
PY

if [[ -n "${SOURCE_DATE_EPOCH:-}" ]]; then
  build_ts=$(TZ=Asia/Taipei date -d "@${SOURCE_DATE_EPOCH}" '+%Y-%m-%dT%H:%M:%S%z')
else
  build_ts=$(TZ=Asia/Taipei date '+%Y-%m-%dT%H:%M:%S%z')
fi
evidence_args=("$stage/release-evidence" --version "$version" --flavor "$flavor")
[[ -n "$crs_version" ]] && evidence_args+=(--crs-version "$crs_version")
[[ $require_govulncheck -eq 1 ]] && evidence_args+=(--require-govulncheck)
SOURCE_DATE_EPOCH="${SOURCE_DATE_EPOCH:-}" "$root/generate-release-evidence.sh" "${evidence_args[@]}"
python3 - "$stage" "$version" "$build_ts" "$test_summary" "$flavor" <<'PY'
import hashlib, pathlib, sys
root=pathlib.Path(sys.argv[1]); version=sys.argv[2]; ts=sys.argv[3]; summary=sys.argv[4]; flavor=sys.argv[5]
def sha(p):
    h=hashlib.sha256()
    with p.open('rb') as f:
        for c in iter(lambda:f.read(1024*1024), b''): h.update(c)
    return h.hexdigest()
files=sorted(p for p in root.rglob('*') if p.is_file())
source_files=[p for p in files if not p.relative_to(root).parts or p.relative_to(root).parts[0] != "release-evidence"]
generated_evidence=[p for p in files if p not in source_files]
lines=[f"Version: {version}", f"Flavor: {flavor}", f"Build Timestamp: {ts}", f"Source File Count: {len(source_files)}", f"Generated Evidence File Count: {len(generated_evidence)}", "", "Tests:"]
if summary:
    text=pathlib.Path(summary).read_text(encoding='utf-8', errors='replace')
    # Keep the artifact manifest bounded while retaining the current evidence header/tail.
    excerpt='\n'.join(text.splitlines()[:80])
    lines.extend('  '+line for line in excerpt.splitlines())
else:
    lines.append('  See TESTING_RESULTS.md inside this artifact.')
lines.extend(['', 'Files:'])
for p in files:
    lines.append(f"{sha(p)}  {p.relative_to(root).as_posix()}")
(root/'RELEASE_MANIFEST.txt').write_text('\n'.join(lines)+'\n', encoding='utf-8')
PY

python3 - "$stage" "$output" "${SOURCE_DATE_EPOCH:-}" <<'PY'
import pathlib, stat, sys, zipfile
root=pathlib.Path(sys.argv[1]); out=sys.argv[2]; epoch=sys.argv[3]
with zipfile.ZipFile(out,'w',compression=zipfile.ZIP_DEFLATED,compresslevel=9) as zf:
    for p in sorted(root.rglob('*')):
        if not p.is_file(): continue
        rel=p.relative_to(root).as_posix()
        info=zipfile.ZipInfo(rel)
        info.compress_type=zipfile.ZIP_DEFLATED

        if epoch:
            import datetime
            dt=datetime.datetime.fromtimestamp(int(epoch), datetime.timezone.utc)
            y=max(1980, dt.year)
            info.date_time=(y,dt.month,dt.day,dt.hour,dt.minute,dt.second//2*2)
        else:
            info.date_time=(2026,1,1,0,0,0)  # deterministic archive metadata; real build time is in manifest
        mode=stat.S_IMODE(p.stat().st_mode)
        info.external_attr=(stat.S_IFREG|mode)<<16
        zf.writestr(info,p.read_bytes())
PY

"$root/verify-release-artifact.sh" "$output" "$root"
sha256sum "$output"
