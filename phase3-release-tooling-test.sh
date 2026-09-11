#!/usr/bin/env bash
set -euo pipefail
root=$(cd "$(dirname "$0")" && pwd); tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT
export SOURCE_DATE_EPOCH=1789092000
"$root/generate-release-evidence.sh" "$tmp/e1" --version test-phase3 --flavor portable
"$root/generate-release-evidence.sh" "$tmp/e2" --version test-phase3 --flavor portable
for f in versions.json provenance.json sbom.spdx.json sbom.cdx.json govulncheck-status.json SHA256SUMS.txt; do cmp "$tmp/e1/$f" "$tmp/e2/$f"; done
python3 - "$tmp/e1" <<'PY'
import json,pathlib,sys
p=pathlib.Path(sys.argv[1]); v=json.loads((p/'versions.json').read_text()); s=json.loads((p/'sbom.spdx.json').read_text()); c=json.loads((p/'sbom.cdx.json').read_text())
assert v['flavor']=='portable' and v['vectorscan']['version']=='not-included'
assert s['spdxVersion']=='SPDX-2.3'; assert c['bomFormat']=='CycloneDX' and c['specVersion']=='1.5'
PY
set +e
WAF_RELEASE_SIGNING_KEY= "$root/sign-release-artifact.sh" "$root/go.mod" >/dev/null 2>&1
rc=$?
set -e
[[ $rc -eq 3 ]] || { echo "expected unsigned signing gate exit 3, got $rc" >&2; exit 1; }
# Exercise the signing implementation with an ephemeral test-only key. This is not
# organizational release signing evidence.
if command -v openssl >/dev/null 2>&1; then
  openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out "$tmp/test-key.pem" >/dev/null 2>&1
  openssl pkey -in "$tmp/test-key.pem" -pubout -out "$tmp/test-pub.pem" >/dev/null 2>&1
  cp "$root/go.mod" "$tmp/artifact.bin"
  WAF_RELEASE_SIGNING_KEY="$tmp/test-key.pem" "$root/sign-release-artifact.sh" "$tmp/artifact.bin" "$tmp/artifact.sig" >/dev/null
  "$root/verify-release-signature.sh" "$tmp/artifact.bin" "$tmp/artifact.sig" "$tmp/test-pub.pem" >/dev/null
fi
echo PHASE3_RELEASE_TOOLING_PASS
