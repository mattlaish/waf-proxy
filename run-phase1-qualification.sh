#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: ./run-phase1-qualification.sh --rules /etc/waf/coraza.conf [options]

Options:
  --corpus PATH              JSONL replay corpus (default qualification/corpus/default.jsonl)
  --output PATH              JSON evidence output (default phase1-qualification.json)
  --min-eligible-matches N   minimum eligible Coraza match events for PASS (default 1)

Exit codes: 0 PASS, 1 FAIL, 3 BLOCKED/NOT_RUN prerequisites or insufficient evidence.
USAGE
}

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
RULES=""
CORPUS="$ROOT/qualification/corpus/default.jsonl"
OUTPUT="$ROOT/phase1-qualification.json"
MIN_MATCHES=1
while [[ $# -gt 0 ]]; do
  case "$1" in
    --rules) RULES=${2:-}; shift 2 ;;
    --corpus) CORPUS=${2:-}; shift 2 ;;
    --output) OUTPUT=${2:-}; shift 2 ;;
    --min-eligible-matches) MIN_MATCHES=${2:-}; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "ERROR: unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
done
[[ -n "$RULES" ]] || { echo "BLOCKED: --rules is required" >&2; exit 3; }
[[ -f "$RULES" ]] || { echo "BLOCKED: rules file not found: $RULES" >&2; exit 3; }
[[ -f "$CORPUS" ]] || { echo "BLOCKED: corpus file not found: $CORPUS" >&2; exit 3; }
command -v go >/dev/null 2>&1 || { echo "BLOCKED: Go is unavailable" >&2; exit 3; }
command -v pkg-config >/dev/null 2>&1 || { echo "BLOCKED: pkg-config is unavailable" >&2; exit 3; }
pkg-config --exists libhs || { echo "BLOCKED: real libhs/VectorScan is unavailable" >&2; exit 3; }

ver=$(GOTOOLCHAIN=local go version 2>/dev/null | awk '{print $3}' | sed 's/^go//' || true)
version_ge() { [ "$(printf '%s\n%s\n' "$2" "$1" | sort -V | head -n1)" = "$2" ]; }
if [[ -z "$ver" ]] || ! version_ge "$ver" 1.25.0; then
  echo "BLOCKED: Go >=1.25.0 required (found ${ver:-unavailable})" >&2
  exit 3
fi

before_mod=$(sha256sum "$ROOT/go.mod" "$ROOT/go.sum")
set +e
(
  cd "$ROOT"
  CGO_ENABLED=1 GOTOOLCHAIN=local go run -tags vectorscan ./cmd/wafqualify \
    --rules "$RULES" --corpus "$CORPUS" --output "$OUTPUT" \
    --min-eligible-matches "$MIN_MATCHES"
)
rc=$?
set -e
after_mod=$(sha256sum "$ROOT/go.mod" "$ROOT/go.sum")
if [[ "$before_mod" != "$after_mod" ]]; then
  echo "FAIL: qualification changed go.mod/go.sum" >&2
  exit 1
fi
exit "$rc"
