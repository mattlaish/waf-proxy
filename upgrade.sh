#!/usr/bin/env bash
# upgrade.sh — routine upgrade for a code/console change.
#
# The common case: you extracted a new package and only the binary (which
# embeds the console) changed. This rebuilds and swaps just the binary, leaving
# your /etc/waf config, certs, CRS, and the systemd unit alone.
#
# It reinstalls the unit / coraza.conf ONLY if they differ from what's installed,
# so you never have to remember which files changed. Run from the extracted
# package dir:  sudo ./upgrade.sh
set -euo pipefail

SRC="$(cd "$(dirname "$0")" && pwd)"
BIN_DST=/usr/local/bin/waf-proxy
TLS_BIN_DST=/usr/local/bin/waf-tlsfront
CTL_BIN_DST=/usr/local/bin/wafctl
UNIT=/etc/systemd/system/waf-proxy.service
TLS_UNIT=/etc/systemd/system/waf-tls-frontend.service
CORAZA=/etc/waf/coraza.conf

[[ $EUID -eq 0 ]] || { echo "run as root: sudo $0" >&2; exit 1; }

# 1. build if the binary isn't already built in this dir
if [[ ! -x "$SRC/waf-proxy" ]]; then
  echo "==> building (no ./waf-proxy present)"
  ( cd "$SRC" && ./build.sh )
fi
[[ -x "$SRC/waf-proxy" && -x "$SRC/waf-tlsfront" && -x "$SRC/wafctl" ]] || { echo "!! build did not produce ./waf-proxy, ./waf-tlsfront and ./wafctl" >&2; exit 1; }

echo "==> stopping services"
systemctl stop waf-tls-frontend waf-proxy || true

echo "==> installing binaries"
install -o root -g root -m 0755 "$SRC/waf-proxy" "$BIN_DST"
install -o root -g root -m 0755 "$SRC/waf-tlsfront" "$TLS_BIN_DST"
install -o root -g root -m 0755 "$SRC/wafctl" "$CTL_BIN_DST"

# 2. unit: reinstall only if changed
UNIT_CHANGED=0
if [[ -f "$SRC/waf-proxy.service" ]] && ! cmp -s "$SRC/waf-proxy.service" "$UNIT"; then
  echo "==> waf-proxy unit changed — updating"
  install -o root -g root -m 0644 "$SRC/waf-proxy.service" "$UNIT"
  UNIT_CHANGED=1
fi
if [[ -f "$SRC/waf-tls-frontend.service" ]] && ! cmp -s "$SRC/waf-tls-frontend.service" "$TLS_UNIT"; then
  echo "==> TLS frontend unit changed — updating"
  install -o root -g root -m 0644 "$SRC/waf-tls-frontend.service" "$TLS_UNIT"
  UNIT_CHANGED=1
fi
[[ $UNIT_CHANGED -eq 0 ]] || systemctl daemon-reload

# 3. coraza.conf: reinstall only if changed (never clobbers if you customised it
#    beyond ours — cmp just tells us whether the shipped file differs)
if [[ -f "$SRC/coraza.conf" ]] && [[ -f "$CORAZA" ]] && ! cmp -s "$SRC/coraza.conf" "$CORAZA"; then
  echo "==> coraza.conf differs from installed — NOT overwriting automatically"
  echo "    (yours may be customised). To take ours:  sudo install -o root -g waf -m 0640 $SRC/coraza.conf $CORAZA"
else
  echo "    coraza.conf unchanged — skipped"
fi

# 4. ensure enabled (idempotent; only matters the first time) and start
systemctl enable waf-proxy waf-tls-frontend >/dev/null 2>&1 || true
echo "==> starting services"
systemctl start waf-proxy
systemctl start waf-tls-frontend

sleep 1
echo
systemctl is-active waf-proxy >/dev/null 2>&1 \
  && systemctl is-active waf-tls-frontend >/dev/null 2>&1 \
  && echo "OK — waf-proxy and waf-tls-frontend are active" \
  || { echo "!! service not active — check: journalctl -u waf-proxy -u waf-tls-frontend -n 40 --no-pager" >&2; exit 1; }
ss -tlnp 2>/dev/null | grep waf-proxy || true
