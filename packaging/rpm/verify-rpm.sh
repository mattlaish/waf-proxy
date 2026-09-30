#!/usr/bin/env bash
set -euo pipefail
PKG="${1:-}"
[ -n "$PKG" ] && [ -f "$PKG" ] || { echo "usage: $0 PACKAGE.rpm" >&2; exit 2; }
for cmd in rpm rpm2cpio cpio sha256sum python3; do
  command -v "$cmd" >/dev/null 2>&1 || { echo "$cmd is required" >&2; exit 1; }
done

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
mkdir -p "$TMP/root"
(
  cd "$TMP/root"
  rpm2cpio "$PKG" | cpio -idm --quiet --no-absolute-filenames
)

required=(
  usr/bin/waf-proxy usr/bin/wafctl usr/bin/waf-tlsfront
  usr/sbin/waf-doctor usr/sbin/waf-setup-interfaces
  usr/lib/systemd/system/waf-proxy.service usr/lib/systemd/system/waf-tls-frontend.service
  etc/waf/config.json etc/waf/coraza.conf etc/waf/waf-tls-frontend.env
  usr/share/doc/waf-proxy/README.md usr/share/doc/waf-proxy/INSTALL.md
  usr/share/doc/waf-proxy/CRS-PROVISIONING.md usr/share/doc/waf-proxy/SELINUX.md
  usr/share/doc/waf-proxy/PACKAGE_BUILD.json
)
for path in "${required[@]}"; do
  [ -e "$TMP/root/$path" ] || { echo "RPM_VERIFY_FAIL missing $path" >&2; exit 1; }
done

[ ! -e "$TMP/root/etc/waf/waf-proxy.env" ] || { echo "RPM_VERIFY_FAIL admin secret must not be packaged" >&2; exit 1; }
[ ! -e "$TMP/root/usr/local/bin/waf-proxy" ] || { echo "RPM_VERIFY_FAIL package must not install into /usr/local" >&2; exit 1; }
grep -q 'ExecStart=/usr/bin/waf-proxy' "$TMP/root/usr/lib/systemd/system/waf-proxy.service" || { echo "RPM_VERIFY_FAIL main unit path" >&2; exit 1; }
grep -q 'ExecStart=/usr/bin/waf-tlsfront' "$TMP/root/usr/lib/systemd/system/waf-tls-frontend.service" || { echo "RPM_VERIFY_FAIL TLS unit path" >&2; exit 1; }
grep -q '^StateDirectory=waf-proxy$' "$TMP/root/usr/lib/systemd/system/waf-proxy.service" || { echo "RPM_VERIFY_FAIL persistent StateDirectory missing" >&2; exit 1; }
if find "$TMP/root/etc/waf/crs" -type f -print -quit 2>/dev/null | grep -q .; then
  echo "RPM_VERIFY_FAIL CRS runtime content must not be bundled in the main package" >&2
  exit 1
fi

NAME="$(rpm -qp --qf '%{NAME}' "$PKG")"
ARCH="$(rpm -qp --qf '%{ARCH}' "$PKG")"
[ "$NAME" = "waf-proxy" ] || { echo "RPM_VERIFY_FAIL unexpected package name: $NAME" >&2; exit 1; }
case "$ARCH" in x86_64|aarch64) ;; *) echo "RPM_VERIFY_FAIL unsupported architecture: $ARCH" >&2; exit 1;; esac

SCRIPTS="$(rpm -qp --scripts "$PKG")"
if printf '%s\n' "$SCRIPTS" | grep -Eiq '(^|[;&|[:space:]])(curl|wget|dnf|yum|git[[:space:]]+clone)([[:space:]]|$)'; then
  echo "RPM_VERIFY_FAIL network/package fetch command found in RPM scriptlets" >&2
  exit 1
fi
printf '%s\n' "$SCRIPTS" | grep -q 'created initial admin token' || { echo "RPM_VERIFY_FAIL secret lifecycle missing" >&2; exit 1; }
printf '%s\n' "$SCRIPTS" | grep -q 'try-restart waf-proxy.service' || { echo "RPM_VERIFY_FAIL controlled upgrade restart missing" >&2; exit 1; }

# Verify RPM config+noreplace flags numerically. RPMFILE_CONFIG=1 and
# RPMFILE_NOREPLACE=16, so both bits must be set for each operator config file.
rpm -qp --qf '[%{FILENAMES}\t%{FILEFLAGS}\n]' "$PKG" > "$TMP/fileflags"
python3 - "$TMP/fileflags" <<'PY'
import sys
required={'/etc/waf/config.json','/etc/waf/coraza.conf','/etc/waf/waf-tls-frontend.env'}
seen={}
for raw in open(sys.argv[1], encoding='utf-8', errors='replace'):
    raw=raw.rstrip('\n')
    if '\t' not in raw: continue
    path,flags=raw.rsplit('\t',1)
    try: f=int(flags)
    except ValueError: continue
    if path in required: seen[path]=f
missing=required-set(seen)
if missing: raise SystemExit('RPM_VERIFY_FAIL missing config metadata: '+','.join(sorted(missing)))
for path,f in seen.items():
    if (f & 1) == 0 or (f & 16) == 0:
        raise SystemExit(f'RPM_VERIFY_FAIL {path} is not config(noreplace): flags={f}')
PY

printf 'RPM_VERIFY_PASS package=%s sha256=%s\n' "$PKG" "$(sha256sum "$PKG" | awk '{print $1}')"
