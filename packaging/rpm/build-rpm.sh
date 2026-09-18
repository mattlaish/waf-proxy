#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
BIN_DIR="$ROOT"
OUT_DIR="$ROOT/dist/rpm"
ARCH="${WAF_RPM_ARCH:-}"
RELEASE="${WAF_RPM_RELEASE:-1}"
EXTRA_REQUIRES="${WAF_RPM_EXTRA_REQUIRES:-}"
PACKAGE_NAME="waf-proxy"
QUAL_FAIL_POST=0

usage() {
  cat <<'USAGE'
Usage: packaging/rpm/build-rpm.sh [options]

Options:
  --binaries-dir DIR   directory containing waf-proxy, wafctl, waf-tlsfront,
                       BUILD_PROVENANCE.json and BUILD_SHA256SUMS.txt
  --output-dir DIR     output directory (default: dist/rpm)
  --arch ARCH          RPM architecture: x86_64 or aarch64
  --release N          RPM Release field (default: 1)
  --qualification-fail-post
                       TEST-ONLY: inject a %post failure into qualification fixtures
  -h, --help           show this help

Required release environment:
  SOURCE_DATE_EPOCH    integer Unix timestamp used for reproducible metadata

Optional:
  WAF_RPM_EXTRA_REQUIRES='pkg1, pkg2'  exact target-distribution runtime
                       requirements. REQUIRED for native-vectorscan builds.
  WAF_RPM_ALLOW_NON_ELF=1              test-only fixture escape hatch.
USAGE
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --binaries-dir) BIN_DIR="$2"; shift 2 ;;
    --output-dir) OUT_DIR="$2"; shift 2 ;;
    --arch) ARCH="$2"; shift 2 ;;
    --release) RELEASE="$2"; shift 2 ;;
    --qualification-fail-post) QUAL_FAIL_POST=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown option: $1" >&2; usage >&2; exit 2 ;;
  esac
done

for cmd in rpmbuild rpm rpm2cpio cpio tar python3 sha256sum; do
  command -v "$cmd" >/dev/null 2>&1 || { echo "$cmd is required" >&2; exit 1; }
done
case "${SOURCE_DATE_EPOCH:-}" in
  ''|*[!0-9]*) echo "SOURCE_DATE_EPOCH must be set to an integer for deterministic RPM builds" >&2; exit 1 ;;
esac
[[ "$RELEASE" =~ ^[0-9]+([.][A-Za-z0-9_]+)*$ ]] || { echo "invalid RPM release: $RELEASE" >&2; exit 1; }

for name in waf-proxy wafctl waf-tlsfront BUILD_PROVENANCE.json BUILD_SHA256SUMS.txt; do
  [ -f "$BIN_DIR/$name" ] || { echo "missing required build artifact: $BIN_DIR/$name" >&2; exit 1; }
done
for name in waf-proxy wafctl waf-tlsfront; do
  [ -x "$BIN_DIR/$name" ] || { echo "binary is not executable: $BIN_DIR/$name" >&2; exit 1; }
  if [ "${WAF_RPM_ALLOW_NON_ELF:-0}" != "1" ]; then
    python3 - "$BIN_DIR/$name" <<'PY'
import pathlib, sys
p=pathlib.Path(sys.argv[1])
if p.read_bytes()[:4] != b"\x7fELF":
    raise SystemExit(f"release package requires ELF binary: {p}")
PY
  fi
done
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
if version.startswith('v') and len(version)>1 and version[1].isdigit():
    version=version[1:]
# RPM Version must not contain '-'. Keep a conservative portable subset.
version=re.sub(r'[^A-Za-z0-9._+~^]', '.', version).strip('.')
if not version:
    raise SystemExit('RPM version normalized to empty')
print(version); print(variant); print(artifact)
PY
)
VERSION="${META[0]}"
BUILD_VARIANT="${META[1]}"
ARTIFACT_TYPE="${META[2]}"

if [ "$QUAL_FAIL_POST" -eq 1 ] && [[ "$VERSION" != *qualification* ]]; then
  echo "--qualification-fail-post is restricted to qualification fixture versions" >&2
  exit 1
fi

if [[ "$BUILD_VARIANT" == *vectorscan* ]] && [ -z "$EXTRA_REQUIRES" ]; then
  cat >&2 <<EOF2
native VectorScan build detected ($BUILD_VARIANT), but WAF_RPM_EXTRA_REQUIRES is empty.
Set the exact approved VectorScan runtime RPM dependency for the target RHEL-family release.
The builder refuses to guess a distribution/repository-specific package name.
EOF2
  exit 1
fi

if [ "${WAF_RPM_ALLOW_NON_ELF:-0}" != "1" ]; then
  ELF_ARCH="$(python3 - "$BIN_DIR/waf-proxy" <<'PY'
import struct, sys
b=open(sys.argv[1],'rb').read(20)
if len(b)<20 or b[:4] != b'\x7fELF': raise SystemExit('invalid ELF')
endian = '<' if b[5] == 1 else '>' if b[5] == 2 else None
if endian is None: raise SystemExit('unsupported ELF endianness')
machine=struct.unpack(endian+'H', b[18:20])[0]
print({62:'x86_64',183:'aarch64'}.get(machine, f'unsupported-{machine}'))
PY
)"
  case "$ELF_ARCH" in x86_64|aarch64) ;; *) echo "unsupported RPM ELF architecture: $ELF_ARCH" >&2; exit 1;; esac
  if [ -z "$ARCH" ]; then ARCH="$ELF_ARCH"; fi
  [ "$ELF_ARCH" = "$ARCH" ] || { echo "package architecture $ARCH does not match waf-proxy ELF architecture $ELF_ARCH" >&2; exit 1; }
else
  [ -n "$ARCH" ] || ARCH="$(rpm --eval '%{_arch}')"
fi
case "$ARCH" in x86_64|aarch64) ;; *) echo "unsupported RPM architecture: $ARCH" >&2; exit 1;; esac

STAGE="$(mktemp -d)"
trap 'rm -rf "$STAGE"' EXIT
ROOTFS="$STAGE/payload/rootfs"
TOP="$STAGE/rpmbuild"
mkdir -p "$ROOTFS"/{usr/bin,usr/sbin,usr/share/doc/waf-proxy,etc/waf/certs,etc/waf/crs,var/lib/waf-proxy,var/log/waf,usr/lib/systemd/system} \
  "$TOP"/{BUILD,BUILDROOT,RPMS,SOURCES,SPECS,SRPMS}

install -m 0755 "$BIN_DIR/waf-proxy" "$ROOTFS/usr/bin/waf-proxy"
install -m 0755 "$BIN_DIR/wafctl" "$ROOTFS/usr/bin/wafctl"
install -m 0755 "$BIN_DIR/waf-tlsfront" "$ROOTFS/usr/bin/waf-tlsfront"
install -m 0755 "$ROOT/waf-doctor.sh" "$ROOTFS/usr/sbin/waf-doctor"
install -m 0755 "$ROOT/setup-interfaces.sh" "$ROOTFS/usr/sbin/waf-setup-interfaces"
install -m 0644 "$ROOT/config.sample.json" "$ROOTFS/etc/waf/config.json"
install -m 0644 "$ROOT/coraza.conf" "$ROOTFS/etc/waf/coraza.conf"
install -m 0644 "$ROOT/packaging/rpm/config/waf-tls-frontend.env" "$ROOTFS/etc/waf/waf-tls-frontend.env"
install -m 0644 "$ROOT/README.md" "$ROOTFS/usr/share/doc/waf-proxy/README.md"
install -m 0644 "$ROOT/INSTALL.md" "$ROOTFS/usr/share/doc/waf-proxy/INSTALL.md"
install -m 0644 "$ROOT/packaging/rpm/CRS-PROVISIONING.md" "$ROOTFS/usr/share/doc/waf-proxy/CRS-PROVISIONING.md"
install -m 0644 "$ROOT/packaging/rpm/SELINUX.md" "$ROOTFS/usr/share/doc/waf-proxy/SELINUX.md"
install -m 0644 "$BIN_DIR/BUILD_PROVENANCE.json" "$ROOTFS/usr/share/doc/waf-proxy/BUILD_PROVENANCE.json"
install -m 0644 "$BIN_DIR/BUILD_SHA256SUMS.txt" "$ROOTFS/usr/share/doc/waf-proxy/BUILD_SHA256SUMS.txt"

sed 's#/usr/local/bin/waf-proxy#/usr/bin/waf-proxy#g; s#/opt/waf-proxy/README.md#/usr/share/doc/waf-proxy/README.md#g' \
  "$ROOT/waf-proxy.service" > "$ROOTFS/usr/lib/systemd/system/waf-proxy.service"
sed 's#/usr/local/bin/waf-tlsfront#/usr/bin/waf-tlsfront#g; s#/opt/waf-proxy/README.md#/usr/share/doc/waf-proxy/README.md#g' \
  "$ROOT/waf-tls-frontend.service" > "$ROOTFS/usr/lib/systemd/system/waf-tls-frontend.service"
chmod 0644 "$ROOTFS/usr/lib/systemd/system/"*.service

python3 - "$ROOTFS/usr/share/doc/waf-proxy/PACKAGE_BUILD.json" "$VERSION" "$RELEASE" "$ARCH" "$BUILD_VARIANT" "$ARTIFACT_TYPE" "$SOURCE_DATE_EPOCH" <<'PY'
import json, pathlib, sys
out, version, release, arch, variant, artifact, epoch = sys.argv[1:]
obj={
  'schema_version': 1,
  'package': 'waf-proxy',
  'format': 'rpm',
  'version': version,
  'release': release,
  'architecture': arch,
  'build_variant': variant,
  'binary_artifact_type': artifact,
  'source_date_epoch': int(epoch),
  'crs_delivery': 'explicit-provisioning-not-rpm-scriptlet-network-fetch',
  'config_upgrade_policy': 'rpm-config-noreplace-preserve-local-changes',
  'persistent_state_path': '/var/lib/waf-proxy',
  'fresh_install_autostart': False,
  'selinux_policy': 'no-auto-generated-or-disable-scriptlet-policy',
}
pathlib.Path(out).write_text(json.dumps(obj, indent=2, sort_keys=True)+'\n', encoding='utf-8')
PY

find "$STAGE/payload" -exec touch -h -d "@$SOURCE_DATE_EPOCH" {} +
tar --sort=name --mtime="@$SOURCE_DATE_EPOCH" --owner=0 --group=0 --numeric-owner --format=gnu \
  -C "$STAGE/payload" -cf "$TOP/SOURCES/waf-proxy-payload.tar" rootfs
cp "$ROOT/packaging/rpm/waf-proxy.spec" "$TOP/SPECS/waf-proxy.spec"

RPM_ARGS=(
  -bb "$TOP/SPECS/waf-proxy.spec"
  --target "$ARCH"
  --define "_topdir $TOP"
  --define "waf_version $VERSION"
  --define "waf_release $RELEASE"
  --define "_buildhost reproducible.invalid"
  --define "use_source_date_epoch_as_buildtime 1"
  --define "clamp_mtime_to_source_date_epoch 1"
  --define "_build_id_links none"
)
[ -z "$EXTRA_REQUIRES" ] || RPM_ARGS+=(--define "waf_extra_requires $EXTRA_REQUIRES")
[ "$QUAL_FAIL_POST" -eq 0 ] || RPM_ARGS+=(--define "waf_qualification_fail_post 1")
SOURCE_DATE_EPOCH="$SOURCE_DATE_EPOCH" rpmbuild "${RPM_ARGS[@]}"

BUILT="$(find "$TOP/RPMS/$ARCH" -maxdepth 1 -type f -name "$PACKAGE_NAME-*.${ARCH}.rpm" -print -quit)"
[ -n "$BUILT" ] || { echo "rpmbuild completed but no binary RPM was found" >&2; exit 1; }
mkdir -p "$OUT_DIR"
OUT="$OUT_DIR/$(basename "$BUILT")"
cp "$BUILT" "$OUT"
sha256sum "$OUT" > "$OUT.sha256"
"$ROOT/packaging/rpm/verify-rpm.sh" "$OUT"
printf 'RPM_PACKAGE=%s\n' "$OUT"
printf 'RPM_SHA256=%s\n' "$(sha256sum "$OUT" | awk '{print $1}')"
printf 'BUILD_VARIANT=%s\n' "$BUILD_VARIANT"
