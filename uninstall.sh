#!/usr/bin/env bash
# waf-proxy uninstaller. Keeps /etc/waf unless --purge is given.
set -euo pipefail

PURGE=0
[[ "${1:-}" == "--purge" ]] && PURGE=1
[[ $EUID -eq 0 ]] || { echo "run as root: sudo $0 [--purge]" >&2; exit 1; }

echo "==> stopping service"
systemctl disable --now waf-proxy 2>/dev/null || true

echo "==> removing unit + binary"
rm -f /etc/systemd/system/waf-proxy.service
systemctl daemon-reload
rm -f /usr/local/bin/waf-proxy
rm -rf /opt/waf-proxy

if [[ $PURGE -eq 1 ]]; then
  echo "==> purging /etc/waf (config, certs, CRS, admin token)"
  rm -rf /etc/waf
  userdel waf 2>/dev/null || true
  echo "purged."
else
  echo
  echo "Kept /etc/waf (config, certs, CRS, admin token) and the 'waf' user."
  echo "Re-run with --purge to remove them."
fi
