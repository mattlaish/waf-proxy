#!/usr/bin/env bash
# waf-proxy installer (Debian/Ubuntu, systemd).
#
# Idempotent: safe to re-run to upgrade the binary. Never overwrites an
# existing /etc/waf/config.json.
set -euo pipefail

BIN_SRC="$(dirname "$0")/waf-proxy"
BIN_DST=/usr/local/bin/waf-proxy
TLS_BIN_SRC="$(dirname "$0")/waf-tlsfront"
TLS_BIN_DST=/usr/local/bin/waf-tlsfront
ETC=/etc/waf
UNIT=/etc/systemd/system/waf-proxy.service
TLS_UNIT=/etc/systemd/system/waf-tls-frontend.service
SRC="$(cd "$(dirname "$0")" && pwd)"

[[ $EUID -eq 0 ]] || { echo "run as root: sudo $0" >&2; exit 1; }

if [[ ! -x "$BIN_SRC" || ! -x "$TLS_BIN_SRC" ]]; then
  echo "!! ./waf-proxy or ./waf-tlsfront not found — run ./build.sh first" >&2
  exit 1
fi

echo "==> creating service user 'waf'"
id -u waf &>/dev/null || useradd --system --no-create-home --shell /usr/sbin/nologin waf

echo "==> creating $ETC"
# /etc/waf must be group-writable: the console persists config by writing
# config.json.tmp here and atomically renaming it (needs waf-group write on dir).
install -d -o root -g waf -m 0770 "$ETC"
install -d -o root -g waf -m 0750 "$ETC/certs"
install -d -o root -g waf -m 0750 "$ETC/crs"

echo "==> creating audit log dir /var/log/waf"
# systemd's LogsDirectory=waf also creates this, but make it now so running the
# binary directly (outside systemd) can write its audit log too.
install -d -o waf -g waf -m 0750 /var/log/waf
install -d -o waf -g waf -m 0750 /var/lib/waf-proxy
# Pre-create the log file waf-owned so a stray foreground/root run can't seed it
# root-owned (which would then block the waf service with EACCES).
[[ -e /var/log/waf/audit.log ]] || install -o waf -g waf -m 0640 /dev/null /var/log/waf/audit.log

echo "==> installing binaries"
install -o root -g root -m 0755 "$BIN_SRC" "$BIN_DST"
install -o root -g root -m 0755 "$TLS_BIN_SRC" "$TLS_BIN_DST"

echo "==> installing rules"
if [[ ! -f "$ETC/coraza.conf" ]]; then
  install -o root -g waf -m 0640 "$SRC/coraza.conf" "$ETC/coraza.conf"
else
  echo "    $ETC/coraza.conf exists — left untouched"
fi

echo "==> installing config"
if [[ ! -f "$ETC/config.json" ]]; then
  # config.json is written back by the console on save → owned by waf.
  install -o waf -g waf -m 0600 "$SRC/config.sample.json" "$ETC/config.json"
else
  echo "    $ETC/config.json exists — left untouched"
fi

echo "==> admin token"
if [[ ! -f "$ETC/waf-proxy.env" ]]; then
  TOKEN="$(openssl rand -hex 24)"
  cat > "$ETC/waf-proxy.env" <<EOF
# waf-proxy environment. Keep 0600 — this is a break-glass admin credential.
WAF_ADMIN_TOKEN=${TOKEN}
EOF
  chown root:root "$ETC/waf-proxy.env"
  chmod 0600 "$ETC/waf-proxy.env"
  echo
  echo "    ┌────────────────────────────────────────────────────────────┐"
  echo "    │ ADMIN TOKEN (first-time login — save this now):            │"
  echo "    │ ${TOKEN} │"
  echo "    └────────────────────────────────────────────────────────────┘"
  echo "    Stored in $ETC/waf-proxy.env"
else
  echo "    $ETC/waf-proxy.env exists — token unchanged"
fi

echo "==> installing systemd units"
install -o root -g root -m 0644 "$SRC/waf-proxy.service" "$UNIT"
install -o root -g root -m 0644 "$SRC/waf-tls-frontend.service" "$TLS_UNIT"
if [[ ! -f "$ETC/waf-tls-frontend.env" ]]; then
  cat > "$ETC/waf-tls-frontend.env" <<'EOF'
# Optional OpenSSL/QAT frontend environment. Keep values non-secret where possible.
# OPENSSL_MODULES=/usr/lib/x86_64-linux-gnu/ossl-modules
EOF
  chown root:root "$ETC/waf-tls-frontend.env"
  chmod 0644 "$ETC/waf-tls-frontend.env"
fi
systemctl daemon-reload
if grep -q "AmbientCapabilities=CAP_NET_BIND_SERVICE" "$UNIT"; then
  echo "    unit grants CAP_NET_BIND_SERVICE — sites may bind privileged ports (80/443) as the unprivileged waf user"
else
  echo "    WARNING: unit lacks CAP_NET_BIND_SERVICE — sites on ports <1024 will fail to bind" >&2
fi
if grep -Eq '^RestrictAddressFamilies=.*AF_NETLINK' "$UNIT"; then
  echo "    unit permits AF_NETLINK — managed-IP interface discovery is available"
else
  echo "    WARNING: unit lacks AF_NETLINK — manage_interface discovery will fail" >&2
fi

echo "==> installing docs"
install -d -m 0755 /opt/waf-proxy
install -o root -g root -m 0644 "$SRC/README.md" /opt/waf-proxy/README.md
install -o root -g root -m 0644 "$SRC/INSTALL.md" /opt/waf-proxy/INSTALL.md 2>/dev/null || true
install -o root -g root -m 0755 "$SRC/waf-doctor.sh" /opt/waf-proxy/waf-doctor.sh 2>/dev/null || true

echo "==> verifying ownership, permissions, unit, and port capability"
if [[ -x "$SRC/waf-doctor.sh" ]]; then
  bash "$SRC/waf-doctor.sh" || echo "!! waf-doctor found issues above — fix before starting" >&2
fi

cat <<'EOF'

──────────────────────────────────────────────────────────────
Installed. Next steps:

 1. Get the OWASP CRS rules (required — coraza.conf includes them):
      cd /etc/waf/crs
      sudo curl -fsSL -o crs.tar.gz \
        https://github.com/coreruleset/coreruleset/archive/refs/tags/v4.7.0.tar.gz
      sudo tar xzf crs.tar.gz --strip-components=1
      sudo cp crs-setup.conf.example crs-setup.conf
      sudo chown -R root:waf /etc/waf/crs && sudo chmod -R g+rX /etc/waf/crs
    Then confirm the include paths in /etc/waf/coraza.conf point at /etc/waf/crs.

 2. Point the default pool at your real backend:
      sudoedit /etc/waf/config.json     (or do it in the console)

 3. Start it:
      sudo systemctl enable --now waf-proxy waf-tls-frontend
      systemctl status waf-proxy waf-tls-frontend

    Go TLS remains the default. To use the OpenSSL/NGINX frontend, install a
    recent NGINX/OpenSSL and set tls_acceleration.mode=frontend in the console.
    NGINX >= 1.25.1 automatically uses the modern `http2 on;` syntax.

 4. Open the console (admin is localhost-only by default):
      ssh -L 9090:127.0.0.1:9090 user@this-host
      → http://127.0.0.1:9090
    Log in with the ADMIN TOKEN above, then Users tab → create your first
    named admin account. Use that for daily work; keep the token offline.

 5. It starts in DetectionOnly. Watch the Logs tab, tune false positives via
    Site Map → page policies, and only then flip sites to Block.
──────────────────────────────────────────────────────────────
EOF
