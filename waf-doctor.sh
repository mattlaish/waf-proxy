#!/usr/bin/env bash
# waf-doctor — one-shot repair + verify for a waf-proxy install.
#
# Fixes, idempotently and in the right order, every recurring failure class:
#   - /etc/waf group-writable so the console can persist config (atomic save)
#   - config.json owned by waf (the process rewrites it)
#   - waf-proxy.env kept root-only (break-glass token; waf must NOT read it)
#   - audit log + log dir owned by waf (root-owned file = service can't append)
#   - any stray root-owned file under /etc/waf or /var/log/waf (foreground-run footgun)
#   - the systemd unit grants bind/address-management capabilities and permits
#     AF_NETLINK so Go can discover interfaces inside the service sandbox
#     (so sites can bind 80/443 as the unprivileged waf user — no root needed)
#   - certs readable by waf; rules readable by waf
#
# Then it VERIFIES each invariant and prints PASS/FAIL. Safe to run anytime.
# Usage: sudo ./waf-doctor.sh [--fix]   (default is --fix; use --check for read-only)

set -uo pipefail

ETC=/etc/waf
LOGDIR=/var/log/waf
BIN=/usr/local/bin/waf-proxy
UNIT=/etc/systemd/system/waf-proxy.service
ENVF="$ETC/waf-proxy.env"
CFG="$ETC/config.json"
SRC="$(cd "$(dirname "$0")" && pwd)"
USER_=waf
GROUP_=waf

MODE="fix"
[[ "${1:-}" == "--check" ]] && MODE="check"
[[ $EUID -eq 0 ]] || { echo "run as root: sudo $0" >&2; exit 1; }

red(){ printf '\033[31m%s\033[0m\n' "$*"; }
grn(){ printf '\033[32m%s\033[0m\n' "$*"; }
ylw(){ printf '\033[33m%s\033[0m\n' "$*"; }
FAILED=0

# ── ensure user/group exist ──
if ! id -u "$USER_" >/dev/null 2>&1; then
  if [[ "$MODE" == fix ]]; then
    useradd --system --no-create-home --shell /usr/sbin/nologin "$USER_"
    echo "created system user $USER_"
  else
    red "user $USER_ does not exist"; FAILED=1
  fi
fi

if [[ "$MODE" == fix ]]; then
  echo "== repairing =="

  # dirs must exist
  install -d -o root   -g "$GROUP_" -m 0770 "$ETC"            # group-writable: atomic config save
  install -d -o root   -g "$GROUP_" -m 0750 "$ETC/certs"
  install -d -o root   -g "$GROUP_" -m 0750 "$ETC/crs"
  install -d -o "$USER_" -g "$GROUP_" -m 0750 "$LOGDIR"

  # config.json: waf owns it (console writes it), 0600 (secrets)
  if [[ -e "$CFG" ]]; then
    chown "$USER_:$GROUP_" "$CFG"; chmod 0600 "$CFG"
  fi
  # sitemap.json: waf-owned observed state (persisted site content map)
  [[ -e "$ETC/sitemap.json" ]] && { chown "$USER_:$GROUP_" "$ETC/sitemap.json"; chmod 0640 "$ETC/sitemap.json"; }

  # audit log: pre-create waf-owned so a stray root run can't seed it root-owned
  [[ -e "$LOGDIR/audit.log" ]] || install -o "$USER_" -g "$GROUP_" -m 0640 /dev/null "$LOGDIR/audit.log"
  chown "$USER_:$GROUP_" "$LOGDIR/audit.log" 2>/dev/null || true

  # rules + CRS + certs readable by waf (root-owned, group waf)
  [[ -e "$ETC/coraza.conf" ]] && { chown root:"$GROUP_" "$ETC/coraza.conf"; chmod 0640 "$ETC/coraza.conf"; }
  [[ -d "$ETC/crs" ]]   && { chown -R root:"$GROUP_" "$ETC/crs";   chmod -R g+rX "$ETC/crs"; }
  if [[ -d "$ETC/certs" ]]; then
    chown -R root:"$GROUP_" "$ETC/certs"
    find "$ETC/certs" -type f -exec chmod 0640 {} \;
    find "$ETC/certs" -type d -exec chmod 0750 {} \;
  fi

  # env file: root-only, NOT group-writable (waf must never read the token)
  [[ -e "$ENVF" ]] && { chown root:root "$ENVF"; chmod 0600 "$ENVF"; }

  # binary: root-owned
  [[ -e "$BIN" ]] && { chown root:root "$BIN"; chmod 0755 "$BIN"; }

  # sweep any stray root-owned files the atomic save / foreground runs left behind
  # (everything under /etc/waf except the env file should be group waf & writable
  #  where waf needs it; here we just fix ownership of obvious offenders)
  while IFS= read -r -d '' f; do
    [[ "$f" == "$ENVF" ]] && continue
    chgrp "$GROUP_" "$f" 2>/dev/null || true
  done < <(find "$ETC" ! -group "$GROUP_" ! -path "$ENVF" -print0 2>/dev/null)

  # install the packaged unit (has the capability + LogsDirectory) if we have it
  if [[ -f "$SRC/waf-proxy.service" ]]; then
    install -o root -g root -m 0644 "$SRC/waf-proxy.service" "$UNIT"
    systemctl daemon-reload
    echo "installed packaged unit"
  fi
fi

echo "== verifying =="
check(){ # desc, test-cmd
  if eval "$2" >/dev/null 2>&1; then grn "PASS  $1"; else red "FAIL  $1"; FAILED=1; fi
}

check "/etc/waf is group-writable by waf (config persists)"      "sudo -u $USER_ test -w $ETC"
check "waf can read config.json"                                  "[[ ! -e $CFG ]] || sudo -u $USER_ test -r $CFG"
check "config.json owned by waf"                                  "[[ ! -e $CFG ]] || [[ \$(stat -c %U $CFG) == $USER_ ]]"
check "waf CANNOT read waf-proxy.env (token stays root-only)"     "[[ ! -e $ENVF ]] || ! sudo -u $USER_ test -r $ENVF"
check "waf can write the audit log"                               "sudo -u $USER_ test -w $LOGDIR/audit.log"
check "no stray non-waf files under /etc/waf (except env)"        "[[ -z \"\$(find $ETC ! -group $USER_ ! -path $ENVF 2>/dev/null)\" ]]"
check "unit grants CAP_NET_BIND_SERVICE (80/443 bind as waf)"     "grep -q AmbientCapabilities=CAP_NET_BIND_SERVICE $UNIT"
check "unit permits AF_NETLINK (managed-IP interface discovery)" "grep -Eq '^RestrictAddressFamilies=.*AF_NETLINK' $UNIT"
check "unit has LogsDirectory=waf (sandbox log write)"            "grep -q LogsDirectory=waf $UNIT"
check "binary present and root-owned"                             "[[ -x $BIN ]] && [[ \$(stat -c %U $BIN) == root ]]"

echo
if [[ $FAILED -eq 0 ]]; then
  grn "All checks passed. Restart to apply any unit/permission changes:"
  echo "    sudo systemctl restart waf-proxy && sudo ss -tlnp | grep waf-proxy"
else
  red "Some checks failed."
  [[ "$MODE" == check ]] && ylw "Run without --check to repair: sudo ./waf-doctor.sh"
  exit 1
fi
