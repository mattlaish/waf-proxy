#!/usr/bin/env bash
# Build disposable Version N / N+1 / intentional-failure package fixtures.
# These fixtures use /bin/true instead of waf-proxy and are ONLY for package
# lifecycle semantics. They are never runtime/security qualification evidence.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
FORMAT=""; OUT=""; EPOCH="${SOURCE_DATE_EPOCH:-1700000000}"
usage(){ echo "Usage: $0 --format deb|rpm --output-dir DIR [--source-date-epoch EPOCH]" >&2; }
while [ "$#" -gt 0 ]; do
  case "$1" in
    --format) FORMAT="$2"; shift 2;;
    --output-dir) OUT="$2"; shift 2;;
    --source-date-epoch) EPOCH="$2"; shift 2;;
    *) usage; exit 2;;
  esac
done
case "$FORMAT" in deb|rpm) ;; *) usage; exit 2;; esac
[ -n "$OUT" ] || { usage; exit 2; }
[[ "$EPOCH" =~ ^[0-9]+$ ]] || { echo "invalid epoch" >&2; exit 2; }
OUT="$(mkdir -p "$OUT" && cd "$OUT" && pwd)"
TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT

make_tree(){
  local name=$1 version=$2 changed=$3
  local tree="$TMP/$name" bin="$TMP/$name-bin"
  cp -a "$ROOT/." "$tree"
  rm -rf "$tree/dist" "$tree/.git" "$tree/release-evidence" "$tree/__pycache__"
  mkdir -p "$bin"
  for x in waf-proxy wafctl waf-tlsfront; do cp /bin/true "$bin/$x"; chmod 0755 "$bin/$x"; done
  (cd "$bin" && sha256sum waf-proxy wafctl waf-tlsfront > BUILD_SHA256SUMS.txt)
  python3 - "$bin/BUILD_PROVENANCE.json" "$version" <<'PY'
import json, pathlib, sys
out,version=sys.argv[1:]
obj={"schema_version":2,"version":version,"artifact_type":"PORTABLE_BINARY","build_variant":"portable-coraza","go_version":"1.25.0","coraza_version":"v3.7.0","vectorscan_version":"not-linked","hsm_pkcs11_enabled":False,"binaries":[]}
pathlib.Path(out).write_text(json.dumps(obj,sort_keys=True)+"\n")
PY
  if [ "$changed" = 1 ]; then
    python3 - "$tree/config.sample.json" <<'PY'
import json, pathlib, sys
p=pathlib.Path(sys.argv[1]); obj=json.loads(p.read_text()); p.write_text(json.dumps(obj,indent=4,sort_keys=True)+"\n   \n")
PY
    printf '\n# lifecycle-candidate-default-change\n' >> "$tree/coraza.conf"
    printf '\n# lifecycle-candidate-default-change\n' >> "$tree/packaging/$FORMAT/config/waf-tls-frontend.env"
  fi
  printf '%s\n' "$tree|$bin"
}

IFS='|' read -r BASE_TREE BASE_BIN < <(make_tree baseline 5.0.0.qualification1 0)
IFS='|' read -r CAND_TREE CAND_BIN < <(make_tree candidate 5.0.0.qualification2 1)
IFS='|' read -r FAIL_TREE FAIL_BIN < <(make_tree failure 5.0.0.qualification3 1)

if [ "$FORMAT" = deb ]; then
  command -v dpkg-deb >/dev/null 2>&1 || { echo "FIXTURE_BUILD_BLOCKED missing=dpkg-deb"; exit 77; }
  SOURCE_DATE_EPOCH="$EPOCH" "$BASE_TREE/packaging/deb/build-deb.sh" --binaries-dir "$BASE_BIN" --output-dir "$OUT/baseline" >/dev/null
  SOURCE_DATE_EPOCH="$EPOCH" "$CAND_TREE/packaging/deb/build-deb.sh" --binaries-dir "$CAND_BIN" --output-dir "$OUT/candidate" >/dev/null
  SOURCE_DATE_EPOCH="$EPOCH" "$FAIL_TREE/packaging/deb/build-deb.sh" --binaries-dir "$FAIL_BIN" --output-dir "$OUT/failure-src" >/dev/null
  FAIL_SRC="$(find "$OUT/failure-src" -name '*.deb' -type f -print -quit)"
  UNPACK="$TMP/failure-unpack"; dpkg-deb -R "$FAIL_SRC" "$UNPACK"
  python3 - "$UNPACK/DEBIAN/postinst" <<'PY'
import pathlib,sys
p=pathlib.Path(sys.argv[1]); s=p.read_text(); marker='set -eu\n'; repl=marker+"echo 'waf-proxy qualification: intentional postinst failure' >&2\nexit 42\n"
if marker not in s: raise SystemExit('postinst missing set -eu')
p.write_text(s.replace(marker,repl,1))
PY
  chmod 0755 "$UNPACK/DEBIAN/postinst"
  find "$UNPACK" -exec touch -h -d "@$EPOCH" {} +
  mkdir -p "$OUT/failure"
  SOURCE_DATE_EPOCH="$EPOCH" dpkg-deb --root-owner-group --build "$UNPACK" "$OUT/failure/waf-proxy_5.0.0.qualification3_$(dpkg-deb -f "$FAIL_SRC" Architecture).deb" >/dev/null
  BASE="$(find "$OUT/baseline" -name '*.deb' -type f -print -quit)"; CAND="$(find "$OUT/candidate" -name '*.deb' -type f -print -quit)"; FAIL="$(find "$OUT/failure" -name '*.deb' -type f -print -quit)"
else
  missing=(); for x in rpmbuild rpm rpm2cpio cpio; do command -v "$x" >/dev/null 2>&1 || missing+=("$x"); done
  if [ "${#missing[@]}" -ne 0 ]; then echo "FIXTURE_BUILD_BLOCKED missing=$(IFS=,; echo "${missing[*]}")"; exit 77; fi
  ARCH="$(rpm --eval '%{_arch}')"; case "$ARCH" in x86_64|aarch64) ;; *) echo "FIXTURE_BUILD_BLOCKED unsupported_arch=$ARCH"; exit 77;; esac
  SOURCE_DATE_EPOCH="$EPOCH" "$BASE_TREE/packaging/rpm/build-rpm.sh" --binaries-dir "$BASE_BIN" --output-dir "$OUT/baseline" --arch "$ARCH" >/dev/null
  SOURCE_DATE_EPOCH="$EPOCH" "$CAND_TREE/packaging/rpm/build-rpm.sh" --binaries-dir "$CAND_BIN" --output-dir "$OUT/candidate" --arch "$ARCH" >/dev/null
  SOURCE_DATE_EPOCH="$EPOCH" "$FAIL_TREE/packaging/rpm/build-rpm.sh" --binaries-dir "$FAIL_BIN" --output-dir "$OUT/failure" --arch "$ARCH" --qualification-fail-post >/dev/null
  BASE="$(find "$OUT/baseline" -name '*.rpm' -type f -print -quit)"; CAND="$(find "$OUT/candidate" -name '*.rpm' -type f -print -quit)"; FAIL="$(find "$OUT/failure" -name '*.rpm' -type f -print -quit)"
fi
cat > "$OUT/FIXTURES.json" <<EOF
{
  "schema_version": 1,
  "format": "$FORMAT",
  "purpose": "package-lifecycle-semantics-only-not-waf-runtime-evidence",
  "baseline": "${BASE#$OUT/}",
  "candidate": "${CAND#$OUT/}",
  "failure": "${FAIL#$OUT/}",
  "source_date_epoch": $EPOCH
}
EOF
printf 'PACKAGE_LIFECYCLE_FIXTURES_PASS format=%s manifest=%s\n' "$FORMAT" "$OUT/FIXTURES.json"
