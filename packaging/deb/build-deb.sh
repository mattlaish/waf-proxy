#!/usr/bin/env bash
# Build the enterprise Debian/Ubuntu binary package from already-built waf-proxy binaries.
#
# The package install is deliberately offline-safe: maintainer scripts never fetch CRS,
# packages, modules, or any other network content. Build the Go binaries first on the
# qualified release host, then package those exact bytes here.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
BIN_DIR="$ROOT"
OUT_DIR="$ROOT/dist/deb"
ARCH="${WAF_DEB_ARCH:-$(dpkg --print-architecture 2>/dev/null || true)}"
MAINTAINER="${WAF_PACKAGE_MAINTAINER:-WAF Reverse Proxy Project <noreply@example.invalid>}"
EXTRA_DEPENDS="${WAF_DEB_EXTRA_DEPENDS:-}"
PACKAGE_NAME="waf-proxy"

usage() {
  cat <<'USAGE'
Usage: packaging/deb/build-deb.sh [options]

Options:
  --binaries-dir DIR   directory containing waf-proxy, wafctl, waf-tlsfront,
                       BUILD_PROVENANCE.json and BUILD_SHA256SUMS.txt
  --output-dir DIR     output directory (default: dist/deb)
  --arch ARCH          Debian architecture (default: dpkg --print-architecture)
  --maintainer TEXT    package Maintainer field
  -h, --help           show this help

Required release environment:
  SOURCE_DATE_EPOCH    integer Unix timestamp used to normalize package mtimes

Optional:
  WAF_DEB_EXTRA_DEPENDS='pkg1, pkg2'  extra runtime dependencies. This is
                       REQUIRED for build variants containing native-vectorscan,
                       because the Debian runtime package name for libhs must be
                       selected explicitly for the target distribution.
  WAF_DEB_ALLOW_NON_ELF=1             test-only fixture escape hatch.
USAGE
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --binaries-dir) BIN_DIR="$2"; shift 2 ;;
    --output-dir) OUT_DIR="$2"; shift 2 ;;
    --arch) ARCH="$2"; shift 2 ;;
    --maintainer) MAINTAINER="$2"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown option: $1" >&2; usage >&2; exit 2 ;;
  esac
done

command -v dpkg-deb >/dev/null 2>&1 || { echo "dpkg-deb is required" >&2; exit 1; }
command -v python3 >/dev/null 2>&1 || { echo "python3 is required" >&2; exit 1; }
command -v sha256sum >/dev/null 2>&1 || { echo "sha256sum is required" >&2; exit 1; }

case "${SOURCE_DATE_EPOCH:-}" in
  ''|*[!0-9]*) echo "SOURCE_DATE_EPOCH must be set to an integer for deterministic package builds" >&2; exit 1 ;;
esac
[ -n "$ARCH" ] || { echo "unable to determine Debian package architecture" >&2; exit 1; }

for name in waf-proxy wafctl waf-tlsfront BUILD_PROVENANCE.json BUILD_SHA256SUMS.txt; do
  [ -f "$BIN_DIR/$name" ] || { echo "missing required build artifact: $BIN_DIR/$name" >&2; exit 1; }
done
for name in waf-proxy wafctl waf-tlsfront; do
  [ -x "$BIN_DIR/$name" ] || { echo "binary is not executable: $BIN_DIR/$name" >&2; exit 1; }
  if [ "${WAF_DEB_ALLOW_NON_ELF:-0}" != "1" ]; then
    python3 - "$BIN_DIR/$name" <<'PY'
import pathlib, sys
p = pathlib.Path(sys.argv[1])
if p.read_bytes()[:4] != b"\x7fELF":
    raise SystemExit(f"release package requires ELF binary: {p}")
PY
  fi
done
if [ "${WAF_DEB_ALLOW_NON_ELF:-0}" != "1" ]; then
  ELF_ARCH="$(python3 - "$BIN_DIR/waf-proxy" <<'PY'
import struct, sys
b=open(sys.argv[1],'rb').read(20)
if len(b)<20 or b[:4] != b'\x7fELF': raise SystemExit('invalid ELF')
endian = '<' if b[5] == 1 else '>' if b[5] == 2 else None
if endian is None: raise SystemExit('unsupported ELF endianness')
machine=struct.unpack(endian+'H', b[18:20])[0]
print({62:'amd64',183:'arm64',3:'i386'}.get(machine, f'unsupported-{machine}'))
PY
)"
  [ "$ELF_ARCH" = "$ARCH" ] || { echo "package architecture $ARCH does not match waf-proxy ELF architecture $ELF_ARCH" >&2; exit 1; }
fi

(
  cd "$BIN_DIR"
  sha256sum -c BUILD_SHA256SUMS.txt >/dev/null
) || { echo "BUILD_SHA256SUMS.txt does not verify the supplied binaries" >&2; exit 1; }

readarray -t META < <(python3 - "$BIN_DIR/BUILD_PROVENANCE.json" <<'PY'
import json, re, sys
p=json.load(open(sys.argv[1], encoding='utf-8'))
version=str(p.get('version') or '').strip()
variant=str(p.get('build_variant') or '').strip()
artifact=str(p.get('artifact_type') or '').strip()
if not version or not variant or artifact not in {'PORTABLE_BINARY','NATIVE_BINARY'}:
    raise SystemExit('invalid BUILD_PROVENANCE.json: version/build_variant/artifact_type required')
# Debian upstream version characters: keep this deliberately conservative.
version=re.sub(r'[^A-Za-z0-9.+:~_-]', '.', version)
if not version:
    raise SystemExit('package version normalized to empty')
print(version)
print(variant)
print(artifact)
PY
)
VERSION="${META[0]}"
BUILD_VARIANT="${META[1]}"
ARTIFACT_TYPE="${META[2]}"

if [[ "$BUILD_VARIANT" == *vectorscan* ]] && [ -z "$EXTRA_DEPENDS" ]; then
  cat >&2 <<EOF2
native VectorScan build detected ($BUILD_VARIANT), but WAF_DEB_EXTRA_DEPENDS is empty.
Set the exact libhs runtime package for the target Debian/Ubuntu release, for example:
  WAF_DEB_EXTRA_DEPENDS='libvectorscan5' ...
The builder refuses to guess a distribution-specific native runtime dependency.
EOF2
  exit 1
fi

BASE_DEPENDS="adduser, openssl, ca-certificates, systemd"
DEPENDS="$BASE_DEPENDS"
[ -z "$EXTRA_DEPENDS" ] || DEPENDS="$DEPENDS, $EXTRA_DEPENDS"

STAGE="$(mktemp -d)"
trap 'rm -rf "$STAGE"' EXIT
PKGROOT="$STAGE/root"
mkdir -p \
  "$PKGROOT/DEBIAN" \
  "$PKGROOT/usr/bin" \
  "$PKGROOT/usr/sbin" \
  "$PKGROOT/usr/share/doc/$PACKAGE_NAME" \
  "$PKGROOT/etc/waf/certs" \
  "$PKGROOT/etc/waf/crs" \
  "$PKGROOT/var/lib/waf-proxy" \
  "$PKGROOT/var/log/waf" \
  "$PKGROOT/lib/systemd/system"

install -m 0755 "$BIN_DIR/waf-proxy" "$PKGROOT/usr/bin/waf-proxy"
install -m 0755 "$BIN_DIR/wafctl" "$PKGROOT/usr/bin/wafctl"
install -m 0755 "$BIN_DIR/waf-tlsfront" "$PKGROOT/usr/bin/waf-tlsfront"
install -m 0755 "$ROOT/waf-doctor.sh" "$PKGROOT/usr/sbin/waf-doctor"
install -m 0755 "$ROOT/setup-interfaces.sh" "$PKGROOT/usr/sbin/waf-setup-interfaces"
install -m 0644 "$ROOT/config.sample.json" "$PKGROOT/etc/waf/config.json"
install -m 0644 "$ROOT/coraza.conf" "$PKGROOT/etc/waf/coraza.conf"
install -m 0644 "$ROOT/README.md" "$PKGROOT/usr/share/doc/$PACKAGE_NAME/README.md"
install -m 0644 "$ROOT/INSTALL.md" "$PKGROOT/usr/share/doc/$PACKAGE_NAME/INSTALL.md"
install -m 0644 "$ROOT/packaging/deb/CRS-PROVISIONING.md" "$PKGROOT/usr/share/doc/$PACKAGE_NAME/CRS-PROVISIONING.md"
install -m 0644 "$BIN_DIR/BUILD_PROVENANCE.json" "$PKGROOT/usr/share/doc/$PACKAGE_NAME/BUILD_PROVENANCE.json"
install -m 0644 "$BIN_DIR/BUILD_SHA256SUMS.txt" "$PKGROOT/usr/share/doc/$PACKAGE_NAME/BUILD_SHA256SUMS.txt"
install -m 0644 "$ROOT/packaging/deb/config/waf-tls-frontend.env" "$PKGROOT/etc/waf/waf-tls-frontend.env"

# Source installer uses /usr/local/bin. Debian packages use /usr/bin by policy.
sed 's#/usr/local/bin/waf-proxy#/usr/bin/waf-proxy#g; s#/opt/waf-proxy/README.md#/usr/share/doc/waf-proxy/README.md#g' \
  "$ROOT/waf-proxy.service" > "$PKGROOT/lib/systemd/system/waf-proxy.service"
sed 's#/usr/local/bin/waf-tlsfront#/usr/bin/waf-tlsfront#g; s#/opt/waf-proxy/README.md#/usr/share/doc/waf-proxy/README.md#g' \
  "$ROOT/waf-tls-frontend.service" > "$PKGROOT/lib/systemd/system/waf-tls-frontend.service"
chmod 0644 "$PKGROOT/lib/systemd/system/"*.service

cat > "$PKGROOT/DEBIAN/control" <<EOF2
Package: $PACKAGE_NAME
Version: $VERSION
Section: net
Priority: optional
Architecture: $ARCH
Maintainer: $MAINTAINER
Depends: $DEPENDS
Description: Coraza reverse-proxy WAF with enterprise operations tooling
 Multi-site reverse-proxy WAF using Coraza/OWASP CRS. The package installs
 the runtime, CLI, systemd units and local configuration skeleton. OWASP CRS
 content is intentionally not downloaded or bundled by post-install scripts;
 provision it explicitly before first service start.
EOF2

cat > "$PKGROOT/DEBIAN/conffiles" <<'EOF2'
/etc/waf/config.json
/etc/waf/coraza.conf
/etc/waf/waf-tls-frontend.env
EOF2

for script in postinst prerm postrm; do
  install -m 0755 "$ROOT/packaging/deb/maintainer/$script" "$PKGROOT/DEBIAN/$script"
done

python3 - "$PKGROOT/usr/share/doc/$PACKAGE_NAME/PACKAGE_BUILD.json" "$VERSION" "$ARCH" "$BUILD_VARIANT" "$ARTIFACT_TYPE" "$SOURCE_DATE_EPOCH" <<'PY'
import json, pathlib, sys
out, version, arch, variant, artifact, epoch = sys.argv[1:]
obj={
  'schema_version': 1,
  'package': 'waf-proxy',
  'format': 'deb',
  'version': version,
  'architecture': arch,
  'build_variant': variant,
  'binary_artifact_type': artifact,
  'source_date_epoch': int(epoch),
  'crs_delivery': 'explicit-provisioning-not-postinst-network-fetch',
  'config_upgrade_policy': 'dpkg-conffiles-preserve-local-changes',
  'persistent_state_path': '/var/lib/waf-proxy',
  'fresh_install_autostart': False,
}
pathlib.Path(out).write_text(json.dumps(obj, indent=2, sort_keys=True)+'\n', encoding='utf-8')
PY

# Normalize every packaged timestamp before dpkg-deb sees the tree.
find "$PKGROOT" -exec touch -h -d "@$SOURCE_DATE_EPOCH" {} +

mkdir -p "$OUT_DIR"
OUT="$OUT_DIR/${PACKAGE_NAME}_${VERSION}_${ARCH}.deb"
rm -f "$OUT"
SOURCE_DATE_EPOCH="$SOURCE_DATE_EPOCH" dpkg-deb --root-owner-group -Zxz -z9 --build "$PKGROOT" "$OUT" >/dev/null
sha256sum "$OUT" > "$OUT.sha256"

"$ROOT/packaging/deb/verify-deb.sh" "$OUT"
printf 'DEB_PACKAGE=%s\n' "$OUT"
printf 'DEB_SHA256=%s\n' "$(sha256sum "$OUT" | awk '{print $1}')"
printf 'BUILD_VARIANT=%s\n' "$BUILD_VARIANT"
