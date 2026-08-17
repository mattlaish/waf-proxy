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
UNIT=/etc/systemd/system/waf-proxy.service
CORAZA=/etc/waf/coraza.conf

[[ $EUID -eq 0 ]] || { echo "run as root: sudo $0" >&2; exit 1; }

# 1. build if the binary isn't already built in this dir
if [[ ! -x "$SRC/waf-proxy" ]]; then
  echo "==> building (no ./waf-proxy present)"
  ( cd "$SRC" && ./build.sh )
fi
[[ -x "$SRC/waf-proxy" ]] || { echo "!! build did not produce ./waf-proxy" >&2; exit 1; }

echo "==> stopping service"
systemctl stop waf-proxy || true

echo "==> installing binary"
install -o root -g root -m 0755 "$SRC/waf-proxy" "$BIN_DST"

# 2. unit: reinstall only if changed
if [[ -f "$SRC/waf-proxy.service" ]] && ! cmp -s "$SRC/waf-proxy.service" "$UNIT"; then
  echo "==> unit changed — updating"
  install -o root -g root -m 0644 "$SRC/waf-proxy.service" "$UNIT"
  systemctl daemon-reload
else
  echo "    unit unchanged — skipped"
fi

# 3. coraza.conf: reinstall only if changed (never clobbers if you customised it
#    beyond ours — cmp just tells us whether the shipped file differs)
if [[ -f "$SRC/coraza.conf" ]] && [[ -f "$CORAZA" ]] && ! cmp -s "$SRC/coraza.conf" "$CORAZA"; then
  echo "==> coraza.conf differs from installed — NOT overwriting automatically"
  echo "    (yours may be customised). To take ours:  sudo install -o root -g waf -m 0640 $SRC/coraza.conf $CORAZA"
else
  echo "    coraza.conf unchanged — skipped"
fi

# 4. ensure enabled (idempotent; only matters the first time) and start
systemctl enable waf-proxy >/dev/null 2>&1 || true
echo "==> starting service"
systemctl start waf-proxy

sleep 1
echo
systemctl is-active waf-proxy >/dev/null 2>&1 \
  && echo "OK — waf-proxy is active" \
  || { echo "!! service not active — check: journalctl -u waf-proxy -n 20 --no-pager" >&2; exit 1; }
ss -tlnp 2>/dev/null | grep waf-proxy || true
