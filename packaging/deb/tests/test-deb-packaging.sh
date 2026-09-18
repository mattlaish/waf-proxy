#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
BIN="$TMP/bin"
OUT1="$TMP/out1"
OUT2="$TMP/out2"
mkdir -p "$BIN" "$OUT1" "$OUT2"

# Use a real host ELF as a harmless package-layout fixture so architecture
# validation is exercised. These bytes are never delivered as a WAF package.
for name in waf-proxy wafctl waf-tlsfront; do
  cp /bin/true "$BIN/$name"
  chmod 0755 "$BIN/$name"
done
(
  cd "$BIN"
  sha256sum waf-proxy wafctl waf-tlsfront > BUILD_SHA256SUMS.txt
)
cat > "$BIN/BUILD_PROVENANCE.json" <<'EOF2'
{
  "schema_version": 2,
  "version": "5.0.0-test1",
  "artifact_type": "PORTABLE_BINARY",
  "build_variant": "portable-coraza",
  "go_version": "1.25.0",
  "coraza_version": "v3.7.0",
  "vectorscan_version": "not-linked",
  "hsm_pkcs11_enabled": false,
  "binaries": []
}
EOF2

SOURCE_DATE_EPOCH=1700000000 \
  "$ROOT/packaging/deb/build-deb.sh" --binaries-dir "$BIN" --output-dir "$OUT1" >/dev/null
SOURCE_DATE_EPOCH=1700000000 \
  "$ROOT/packaging/deb/build-deb.sh" --binaries-dir "$BIN" --output-dir "$OUT2" >/dev/null

P1="$(find "$OUT1" -name '*.deb' -type f -print -quit)"
P2="$(find "$OUT2" -name '*.deb' -type f -print -quit)"
[ -n "$P1" ] && [ -n "$P2" ]
H1="$(sha256sum "$P1" | awk '{print $1}')"
H2="$(sha256sum "$P2" | awk '{print $1}')"
[ "$H1" = "$H2" ] || { echo "FAIL non-reproducible fixture package" >&2; exit 1; }
"$ROOT/packaging/deb/verify-deb.sh" "$P1" >/dev/null

# Native VectorScan packages must never guess their Debian libhs dependency.
python3 - "$BIN/BUILD_PROVENANCE.json" <<'PY'
import json,sys
p=json.load(open(sys.argv[1])); p['artifact_type']='NATIVE_BINARY'; p['build_variant']='native-vectorscan'
open(sys.argv[1],'w').write(json.dumps(p,sort_keys=True)+'\n')
PY
if SOURCE_DATE_EPOCH=1700000000 \
   "$ROOT/packaging/deb/build-deb.sh" --binaries-dir "$BIN" --output-dir "$TMP/native" >/dev/null 2>&1; then
  echo "FAIL native VectorScan package accepted without explicit runtime dependency" >&2
  exit 1
fi

printf 'DEB_PACKAGING_TEST_PASS reproducible_sha256=%s\n' "$H1"
