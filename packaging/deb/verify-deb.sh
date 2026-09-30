#!/usr/bin/env bash
set -euo pipefail
PKG="${1:-}"
[ -n "$PKG" ] && [ -f "$PKG" ] || { echo "usage: $0 PACKAGE.deb" >&2; exit 2; }
command -v dpkg-deb >/dev/null 2>&1 || { echo "dpkg-deb is required" >&2; exit 1; }

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
dpkg-deb --extract "$PKG" "$TMP/root"
dpkg-deb --control "$PKG" "$TMP/control"

required=(
  usr/bin/waf-proxy
  usr/bin/wafctl
  usr/bin/waf-tlsfront
  usr/sbin/waf-doctor
  usr/sbin/waf-setup-interfaces
  lib/systemd/system/waf-proxy.service
  lib/systemd/system/waf-tls-frontend.service
  etc/waf/config.json
  etc/waf/coraza.conf
  usr/share/doc/waf-proxy/README.md
  usr/share/doc/waf-proxy/INSTALL.md
  usr/share/doc/waf-proxy/CRS-PROVISIONING.md
  usr/share/doc/waf-proxy/PACKAGE_BUILD.json
)
for path in "${required[@]}"; do
  [ -e "$TMP/root/$path" ] || { echo "DEB_VERIFY_FAIL missing $path" >&2; exit 1; }
done

for script in postinst prerm postrm; do
  [ -x "$TMP/control/$script" ] || { echo "DEB_VERIFY_FAIL missing executable maintainer script $script" >&2; exit 1; }
  /bin/sh -n "$TMP/control/$script"
  if grep -Eiq '(^|[;&|[:space:]])(curl|wget|apt|apt-get|dnf|yum|git[[:space:]]+clone)([[:space:]]|$)' "$TMP/control/$script"; then
    echo "DEB_VERIFY_FAIL network/package fetch command found in $script" >&2
    exit 1
  fi
done

for cfg in /etc/waf/config.json /etc/waf/coraza.conf /etc/waf/waf-tls-frontend.env; do
  grep -Fxq "$cfg" "$TMP/control/conffiles" || { echo "DEB_VERIFY_FAIL conffile missing: $cfg" >&2; exit 1; }
done

[ ! -e "$TMP/root/etc/waf/waf-proxy.env" ] || { echo "DEB_VERIFY_FAIL admin secret must not be packaged" >&2; exit 1; }
[ ! -e "$TMP/root/usr/local/bin/waf-proxy" ] || { echo "DEB_VERIFY_FAIL package must not install into /usr/local" >&2; exit 1; }
grep -q 'ExecStart=/usr/bin/waf-proxy' "$TMP/root/lib/systemd/system/waf-proxy.service" || { echo "DEB_VERIFY_FAIL main unit path" >&2; exit 1; }
grep -q 'ExecStart=/usr/bin/waf-tlsfront' "$TMP/root/lib/systemd/system/waf-tls-frontend.service" || { echo "DEB_VERIFY_FAIL TLS unit path" >&2; exit 1; }
grep -q '^StateDirectory=waf-proxy$' "$TMP/root/lib/systemd/system/waf-proxy.service" || { echo "DEB_VERIFY_FAIL persistent StateDirectory missing" >&2; exit 1; }

# CRS runtime content is deliberately absent. Only the empty provisioning target
# directory may exist in /etc/waf/crs.
if find "$TMP/root/etc/waf/crs" -type f -print -quit 2>/dev/null | grep -q .; then
  echo "DEB_VERIFY_FAIL CRS runtime content must not be bundled in the main package" >&2
  exit 1
fi

printf 'DEB_VERIFY_PASS package=%s sha256=%s\n' "$PKG" "$(sha256sum "$PKG" | awk '{print $1}')"
