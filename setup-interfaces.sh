#!/usr/bin/env bash
# setup-interfaces.sh — separate the management and data planes onto two NICs.
#
# Idempotent. What it does:
#   1. lists the machine's interfaces + IPv4 addresses
#   2. you pick which IP is MANAGEMENT (admin console) and which NIC is DATA
#   3. generates a self-signed TLS cert for the admin console (unless you supply one)
#   4. writes a systemd drop-in pinning the admin console to the mgmt IP + TLS
#   5. VALIDATES that no data-plane site listens on the mgmt IP (does NOT touch
#      your site listen addresses — you set those per-site in config/console)
#
# IPs live in config/drop-in, never compiled into the binary: an IP change is an
# edit, not a rebuild. The binary's own startup check enforces separation too.
set -euo pipefail

ETC=/etc/waf
CONFIG="$ETC/config.json"
CERTDIR="$ETC/certs"
DROPIN_DIR=/etc/systemd/system/waf-proxy.service.d
DROPIN="$DROPIN_DIR/interfaces.conf"
ADMIN_PORT_DEFAULT=9090

[[ $EUID -eq 0 ]] || { echo "run as root: sudo $0" >&2; exit 1; }
command -v python3 >/dev/null || { echo "python3 required" >&2; exit 1; }
command -v openssl >/dev/null || { echo "openssl required" >&2; exit 1; }
[[ -f "$CONFIG" ]] || { echo "$CONFIG not found — run install.sh first" >&2; exit 1; }

echo "== Detected interfaces =="
mapfile -t ROWS < <(ip -o -4 addr show scope global | awk '{print $2" "$4}')
if [[ ${#ROWS[@]} -eq 0 ]]; then
  echo "No global IPv4 addresses found. Configure your NICs first." >&2
  exit 1
fi
i=0
for r in "${ROWS[@]}"; do
  iface="${r%% *}"; cidr="${r#* }"; ip="${cidr%%/*}"
  echo "  [$i] $iface  $ip"
  i=$((i+1))
done
echo

read_idx () { # prompt -> echoes "iface ip"
  local prompt="$1" idx
  while :; do
    read -rp "$prompt" idx
    [[ "$idx" =~ ^[0-9]+$ ]] && (( idx < ${#ROWS[@]} )) && break
    echo "  enter a number 0..$(( ${#ROWS[@]} - 1 ))"
  done
  local r="${ROWS[$idx]}"; echo "${r%% *} ${r#* }"
}

read -r MGMT_IFACE MGMT_CIDR < <(read_idx "Which interface/IP is MANAGEMENT (admin console)? ")
MGMT_IP="${MGMT_CIDR%%/*}"
read -r DATA_IFACE DATA_CIDR < <(read_idx "Which interface is the DATA plane (traffic)? ")
DATA_IP="${DATA_CIDR%%/*}"

if [[ "$MGMT_IP" == "$DATA_IP" ]]; then
  echo "!! management and data cannot be the same IP — that defeats plane separation." >&2
  exit 1
fi

read -rp "Admin console port [$ADMIN_PORT_DEFAULT]: " ADMIN_PORT
ADMIN_PORT="${ADMIN_PORT:-$ADMIN_PORT_DEFAULT}"

echo
echo "  MANAGEMENT : $MGMT_IFACE  $MGMT_IP:$ADMIN_PORT   (admin console, TLS)"
echo "  DATA plane : $DATA_IFACE  $DATA_IP              (your sites bind their own IP:port here)"
echo

# ── validate: no configured site may listen on the mgmt IP:port ──
echo "==> validating plane separation against $CONFIG"
python3 - "$CONFIG" "$MGMT_IP" "$ADMIN_PORT" <<'PY'
import json, sys
cfg_path, mgmt_ip, admin_port = sys.argv[1], sys.argv[2], sys.argv[3]
with open(cfg_path) as f: cfg = json.load(f)

def host_port(listen):
    if listen.startswith("["):
        h, p = listen[1:].split("]:", 1); return h, p
    if listen.count(":") == 1:
        h, p = listen.rsplit(":", 1); return h, p
    return "", listen.lstrip(":")

problems = []
sites = cfg.get("sites", [])
for s in sites:
    host, port = host_port(s.get("listen", ""))
    name = s.get("name", "?")
    # A site bound to the mgmt IP, or to all-interfaces on the admin port, would
    # put traffic (or a second service) on the management plane. Refuse.
    if host == mgmt_ip:
        problems.append(f"site {name} listens on the MANAGEMENT IP ({s['listen']})")
    if host in ("", "0.0.0.0", "::") and port == admin_port:
        problems.append(f"site {name} listens on all interfaces at the admin port ({s['listen']})")

if problems:
    print("PLANE SEPARATION VIOLATION:", file=sys.stderr)
    for p in problems: print("  - " + p, file=sys.stderr)
    print("  Fix these site listen addresses (use the data NIC IP) and re-run.", file=sys.stderr)
    sys.exit(2)

print(f"    OK — {len(sites)} site(s), none on the management plane")
PY

# ── TLS cert for the admin console ──
read -rp "Path to an existing admin TLS cert (blank = generate self-signed): " CERT_IN
if [[ -n "$CERT_IN" ]]; then
  read -rp "Path to the matching key: " KEY_IN
  ADMIN_CERT="$CERT_IN"; ADMIN_KEY="$KEY_IN"
  [[ -f "$ADMIN_CERT" && -f "$ADMIN_KEY" ]] || { echo "cert/key not found" >&2; exit 1; }
else
  install -d -o root -g waf -m 0750 "$CERTDIR"
  ADMIN_CERT="$CERTDIR/admin.crt"; ADMIN_KEY="$CERTDIR/admin.key"
  echo "==> generating self-signed cert for $MGMT_IP (10y)"
  openssl req -x509 -newkey rsa:2048 -nodes \
    -keyout "$ADMIN_KEY" -out "$ADMIN_CERT" -days 3650 \
    -subj "/CN=waf-proxy-admin/O=waf-proxy" \
    -addext "subjectAltName=IP:$MGMT_IP" >/dev/null 2>&1
  chown root:waf "$ADMIN_CERT" "$ADMIN_KEY"
  chmod 0640 "$ADMIN_CERT" "$ADMIN_KEY"
  echo "    wrote $ADMIN_CERT / $ADMIN_KEY"
fi

# ── systemd drop-in: pin admin console to mgmt IP + TLS ──
echo "==> writing $DROPIN"
install -d -m 0755 "$DROPIN_DIR"
cat > "$DROPIN" <<EOF
# Managed by setup-interfaces.sh — machine-specific plane pinning.
# Re-run that script to change; do not hand-edit.
[Service]
Environment=WAF_ADMIN_ADDR=$MGMT_IP:$ADMIN_PORT
Environment=WAF_ADMIN_TLS_CERT=$ADMIN_CERT
Environment=WAF_ADMIN_TLS_KEY=$ADMIN_KEY
Environment=WAF_DATA_INTERFACE=$DATA_IFACE
EOF
chmod 0644 "$DROPIN"

systemctl daemon-reload

cat <<EOF

──────────────────────────────────────────────────────────────
Plane separation configured:

  Admin console : https://$MGMT_IP:$ADMIN_PORT   (management NIC, $MGMT_IFACE)
  Data plane    : $DATA_IFACE / $DATA_IP — your sites bind their own IP:port here

Your site listen addresses were NOT changed — set them to data-NIC IPs in the
console or config.json. The startup check will refuse to run if any site lands
on the management IP.

The self-signed cert will warn in the browser (expected on an internal VLAN);
verify the fingerprint on first connect. Signed-update endpoints stay
localhost-only regardless of the admin bind.

Apply:
  sudo systemctl restart waf-proxy
  systemctl status waf-proxy
  sudo ss -tlnp | grep waf-proxy
──────────────────────────────────────────────────────────────
EOF
