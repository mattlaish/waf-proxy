#!/usr/bin/env bash
# Phase 0 release-host qualification runner.
#
# This script intentionally distinguishes BLOCKED from FAIL. A host that lacks
# Go >=1.25 or verified real libvectorscan provenance has not failed WAF
# correctness; it is simply not a valid host on which to claim the real gates.
#
# Usage:
#   ./qualify-release-host.sh --preflight   # dependency/provenance checks only
#   ./qualify-release-host.sh --core        # preflight + mandatory core gates
#
# Source-installed VectorScan is supported only with explicit operator
# provenance, for example:
#   WAF_VECTORSCAN_PROVENANCE_ACK='VectorScan 5.4.x built from <release/tag>' \
#     ./qualify-release-host.sh --core
#
# Exit codes: 0 PASS, 1 gate FAIL, 3 BLOCKED by release-host prerequisites.
set -euo pipefail

cd "$(dirname "$0")"

MODE="${1:---core}"
case "$MODE" in
  --preflight|--core) ;;
  -h|--help)
    sed -n '1,24p' "$0"
    exit 0
    ;;
  *)
    echo "usage: $0 [--preflight|--core]" >&2
    exit 2
    ;;
esac

blocked=0
pass() { printf 'PASS     %s\n' "$*"; }
info() { printf 'INFO     %s\n' "$*"; }
block() { printf 'BLOCKED  %s\n' "$*" >&2; blocked=1; }
fail() { printf 'FAIL     %s\n' "$*" >&2; exit 1; }

version_ge() {
  [ "$(printf '%s\n%s\n' "$2" "$1" | sort -V | head -n1)" = "$2" ]
}

local_go_version() {
  GOTOOLCHAIN=local go version 2>/dev/null | awk '{print $3}' | sed 's/^go//'
}

echo "== Phase 0 release-host preflight =="
info "host=$(uname -srm)"

if [ "$(uname -s)" != "Linux" ]; then
  block "Linux release host required"
else
  pass "Linux host"
fi

if ! command -v go >/dev/null 2>&1; then
  block "Go executable not found"
else
  gov="$(local_go_version || true)"
  if [ -z "$gov" ]; then
    block "could not determine local Go version without toolchain auto-download"
  elif ! version_ge "$gov" "1.25.0"; then
    block "Go >=1.25.0 required; local toolchain is go${gov}"
  else
    pass "local Go toolchain go${gov}"
  fi
fi

if grep -Eq '^go 1\.25\.0$' go.mod; then
  pass "go.mod pins Go 1.25.0"
else
  fail "go.mod no longer pins Go 1.25.0"
fi
if grep -Eq '^require[[:space:]]+github\.com/corazawaf/coraza/v3[[:space:]]+v3\.7\.0[[:space:]]*$' go.mod; then
  pass "go.mod pins Coraza v3.7.0"
else
  fail "go.mod no longer pins Coraza v3.7.0"
fi

if ! command -v pkg-config >/dev/null 2>&1; then
  block "pkg-config not found"
elif ! pkg-config --exists libhs; then
  block "pkg-config cannot resolve libhs (install real libvectorscan-dev/libhs)"
else
  hs_version="$(pkg-config --modversion libhs 2>/dev/null || echo unknown)"
  pass "pkg-config libhs ${hs_version}"
  info "libhs.pc=$(pkg-config --variable=pcfiledir libhs 2>/dev/null || echo unknown)"
  info "libhs.libs=$(pkg-config --libs libhs 2>/dev/null || echo unknown)"
fi

if command -v cc >/dev/null 2>&1 || command -v gcc >/dev/null 2>&1 || command -v clang >/dev/null 2>&1; then
  pass "C compiler available for CGO"
else
  block "C compiler not found for CGO"
fi

# The previous packaging environment used an ABI-compatible test library.
# Require release provenance so that a semantic scan PASS cannot be mislabeled
# as real VectorScan evidence.
vector_provenance=""
if command -v dpkg-query >/dev/null 2>&1; then
  if vector_provenance="$(dpkg-query -W -f='${Status} ${Version}' libvectorscan-dev 2>/dev/null)" && [[ "$vector_provenance" == "install ok installed "* ]]; then
    pass "real VectorScan package provenance: libvectorscan-dev ${vector_provenance#install ok installed }"
  else
    vector_provenance=""
  fi
fi
if [ -z "$vector_provenance" ]; then
  if [ -n "${WAF_VECTORSCAN_PROVENANCE_ACK:-}" ]; then
    pass "operator-attested VectorScan provenance"
    info "VectorScan provenance: ${WAF_VECTORSCAN_PROVENANCE_ACK}"
  else
    block "real VectorScan provenance not established; install libvectorscan-dev or set WAF_VECTORSCAN_PROVENANCE_ACK for a reviewed source install"
  fi
fi

if command -v nginx >/dev/null 2>&1; then
  info "nginx=$(nginx -v 2>&1 | sed 's#^nginx version: ##')"
else
  info "nginx unavailable; TLS frontend smoke remains a separate NOT_RUN gate"
fi
if command -v openssl >/dev/null 2>&1; then
  info "openssl=$(openssl version | head -1)"
else
  info "OpenSSL CLI unavailable; TLS frontend smoke remains a separate NOT_RUN gate"
fi

if [ "$blocked" -ne 0 ]; then
  echo "RESULT   BLOCKED — release-host prerequisites are incomplete" >&2
  exit 3
fi

echo "RESULT   PRECHECK_PASS"
[ "$MODE" = "--preflight" ] && exit 0

echo
echo "== Phase 0 core correctness gates =="

before_mod="$(sha256sum go.mod go.sum 2>/dev/null || true)"
cleanup() {
  if [ "${WAF_QUALIFY_KEEP_BINARIES:-0}" != "1" ]; then
    rm -f waf-proxy waf-tlsfront
  fi
}
trap cleanup EXIT

# build.sh performs vet, full tests, native race, the realcoraza gate and both
# production builds. WAF_VECTORSCAN=required prevents silent fallback.
if ! GOTOOLCHAIN=local WAF_VECTORSCAN=required ./build.sh; then
  fail "WAF_VECTORSCAN=required ./build.sh"
fi
pass "required native build + vet/test/race + realcoraza gate"

# Keep this explicit in the qualification transcript even though the full
# vectorscan-tag suite above also executes it.
if ! GOTOOLCHAIN=local CGO_ENABLED=1 go test -count=1 -tags vectorscan \
  -run '^TestNativeVectorScanCompileAndScanGate$' ./internal/vectoraccel; then
  fail "real libhs hs_compile_multi/hs_scan semantic gate"
fi
pass "real libhs hs_compile_multi/hs_scan semantic gate"

if ! GOTOOLCHAIN=local CGO_ENABLED=1 go test -count=1 -tags realcoraza \
  -run '^TestRealCorazaDetectionOnlyNologMatchedRulesTruth$' .; then
  fail "real Coraza DetectionOnly+nolog MatchedRules truth gate"
fi
pass "real Coraza DetectionOnly+nolog MatchedRules truth gate"

after_mod="$(sha256sum go.mod go.sum 2>/dev/null || true)"
if [ "$before_mod" != "$after_mod" ]; then
  fail "qualification changed go.mod/go.sum; review dependency drift before release"
fi
pass "go.mod/go.sum unchanged by qualification"

echo "RESULT   CORE_PASS — system install/upgrade/TLS smoke and production Learning qualification are still separate gates"
