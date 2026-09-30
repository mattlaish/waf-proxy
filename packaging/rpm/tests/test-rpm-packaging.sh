#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
missing=()
for cmd in rpmbuild rpm rpm2cpio cpio; do command -v "$cmd" >/dev/null 2>&1 || missing+=("$cmd"); done
if [ "${#missing[@]}" -ne 0 ]; then
  printf 'RPM_PACKAGING_TEST_BLOCKED missing=%s\n' "$(IFS=,; echo "${missing[*]}")"
  exit 77
fi
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
BIN="$TMP/bin"; OUT1="$TMP/out1"; OUT2="$TMP/out2"
mkdir -p "$BIN" "$OUT1" "$OUT2"
for name in waf-proxy wafctl waf-tlsfront; do cp /bin/true "$BIN/$name"; chmod 0755 "$BIN/$name"; done
(
  cd "$BIN"
  sha256sum waf-proxy wafctl waf-tlsfront > BUILD_SHA256SUMS.txt
)
cat > "$BIN/BUILD_PROVENANCE.json" <<'JSON'
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
JSON
HOST_ARCH="$(rpm --eval '%{_arch}')"
case "$HOST_ARCH" in x86_64|aarch64) ;; *) echo "RPM_PACKAGING_TEST_BLOCKED unsupported_host_arch=$HOST_ARCH"; exit 77;; esac
SOURCE_DATE_EPOCH=1700000000 "$ROOT/packaging/rpm/build-rpm.sh" --binaries-dir "$BIN" --output-dir "$OUT1" --arch "$HOST_ARCH" >/dev/null
SOURCE_DATE_EPOCH=1700000000 "$ROOT/packaging/rpm/build-rpm.sh" --binaries-dir "$BIN" --output-dir "$OUT2" --arch "$HOST_ARCH" >/dev/null
P1="$(find "$OUT1" -name '*.rpm' -type f -print -quit)"; P2="$(find "$OUT2" -name '*.rpm' -type f -print -quit)"
[ -n "$P1" ] && [ -n "$P2" ]
H1="$(sha256sum "$P1" | awk '{print $1}')"; H2="$(sha256sum "$P2" | awk '{print $1}')"
[ "$H1" = "$H2" ] || { echo "FAIL non-reproducible fixture RPM" >&2; exit 1; }
"$ROOT/packaging/rpm/verify-rpm.sh" "$P1" >/dev/null
python3 - "$BIN/BUILD_PROVENANCE.json" <<'PY'
import json,sys
p=json.load(open(sys.argv[1])); p['artifact_type']='NATIVE_BINARY'; p['build_variant']='native-vectorscan'
open(sys.argv[1],'w').write(json.dumps(p,sort_keys=True)+'\n')
PY
if SOURCE_DATE_EPOCH=1700000000 "$ROOT/packaging/rpm/build-rpm.sh" --binaries-dir "$BIN" --output-dir "$TMP/native" --arch "$HOST_ARCH" >/dev/null 2>&1; then
  echo "FAIL native VectorScan RPM accepted without explicit runtime dependency" >&2
  exit 1
fi
printf 'RPM_PACKAGING_TEST_PASS reproducible_sha256=%s\n' "$H1"
