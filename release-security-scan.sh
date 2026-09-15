#!/usr/bin/env bash
# Produce source-bound machine-readable govulncheck release evidence.
# Exit 0: scan executed and govulncheck returned success.
# Exit 1: scan executed and failed/found a blocking result, or required prereq missing.
# Exit 3: NOT_RUN/BLOCKED in auto/off mode.
set -euo pipefail

usage() { echo "Usage: $0 --output FILE [--mode auto|required|off]" >&2; }
root=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
output=""; mode="auto"
while [[ $# -gt 0 ]]; do
  case "$1" in
    --output) [[ $# -ge 2 ]] || { usage; exit 2; }; output=$2; shift 2 ;;
    --mode) [[ $# -ge 2 ]] || { usage; exit 2; }; mode=$2; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) usage; exit 2 ;;
  esac
done
[[ -n "$output" ]] || { usage; exit 2; }
case "$mode" in auto|required|off) ;; *) usage; exit 2;; esac
command -v python3 >/dev/null 2>&1 || { echo "ERROR: python3 required" >&2; exit 1; }
output=$(python3 -c 'import os,sys; print(os.path.abspath(sys.argv[1]))' "$output")
# Writing arbitrary evidence into the source tree would change the very source
# manifest it is supposed to bind. release-evidence/ is excluded by definition.
case "$output" in
  "$root"/release-evidence/*) ;;
  "$root"/*) echo "ERROR: evidence output inside source tree must be under release-evidence/" >&2; exit 2 ;;
esac
mkdir -p "$(dirname "$output")"
manifest_sha=$(python3 "$root/tools/source_manifest.py" --root "$root" --digest-only)
executed_at=$(TZ=UTC date '+%Y-%m-%dT%H:%M:%SZ')

write_evidence() {
  local status=$1 reason=$2 tool_version=${3:-NOT_AVAILABLE} exit_code=${4:-null} raw_file=${5:-}
  python3 - "$output" "$status" "$reason" "$tool_version" "$exit_code" "$manifest_sha" "$executed_at" "$raw_file" <<'PY'
import hashlib,json,pathlib,sys
out=pathlib.Path(sys.argv[1]); status,reason,tool_version,exit_code,manifest_sha,executed_at,raw_file=sys.argv[2:]
raw=pathlib.Path(raw_file).read_text(encoding='utf-8',errors='replace') if raw_file else ''
obj={
  'schema_version':1,
  'evidence_type':'govulncheck',
  'status':status,
  'tool':'govulncheck',
  'tool_version':tool_version,
  'source_manifest_sha256':manifest_sha,
  'executed_at':executed_at,
  'result_digest':hashlib.sha256(raw.encode('utf-8')).hexdigest(),
  'output':raw,
  'reason':reason,
}
if exit_code != 'null': obj['exit_code']=int(exit_code)
out.write_text(json.dumps(obj,indent=2,sort_keys=True)+'\n',encoding='utf-8')
PY
}

if [[ "$mode" == "off" ]]; then
  write_evidence NOT_RUN "govulncheck explicitly disabled"
  exit 3
fi
if ! command -v govulncheck >/dev/null 2>&1; then
  write_evidence NOT_RUN "govulncheck executable unavailable"
  [[ "$mode" == "required" ]] && exit 1 || exit 3
fi
govuln_version=$(govulncheck -version 2>&1 | tr '\n' ' ' | sed -E 's/[[:space:]]+/ /g' | sed -E 's/^ | $//g' || true)
[[ -n "$govuln_version" ]] || govuln_version="unknown"
if ! command -v go >/dev/null 2>&1; then
  write_evidence BLOCKED "Go executable unavailable" "$govuln_version"
  [[ "$mode" == "required" ]] && exit 1 || exit 3
fi
gov=$(GOTOOLCHAIN=local go version 2>/dev/null | awk '{print $3}' | sed 's/^go//' || true)
if [[ -z "$gov" ]] || [[ "$(printf '%s\n%s\n' 1.25.0 "$gov" | sort -V | head -n1)" != "1.25.0" ]]; then
  write_evidence BLOCKED "Go >=1.25.0 required by project; local toolchain is ${gov:-unknown}" "$govuln_version"
  [[ "$mode" == "required" ]] && exit 1 || exit 3
fi

tmp=$(mktemp); trap 'rm -f "$tmp"' EXIT
set +e
(cd "$root" && GOTOOLCHAIN=local govulncheck -json ./...) >"$tmp" 2>&1
rc=$?
set -e
if [[ $rc -eq 0 ]]; then status=PASS; reason="govulncheck completed successfully"; else status=FAIL; reason="govulncheck returned non-zero"; fi
write_evidence "$status" "$reason" "$govuln_version" "$rc" "$tmp"
exit "$rc"
