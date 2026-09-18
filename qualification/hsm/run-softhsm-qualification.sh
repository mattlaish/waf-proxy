#!/usr/bin/env bash
set -euo pipefail

# SoftHSM qualification intentionally assumes a pre-provisioned token/key and
# never accepts a raw PIN argument. Provision the lab token using your normal
# secret-safe process, place the user PIN in a 0600 file (or protected env var),
# and set WAF_HSM_PIN_SECRET_REF=file:/... or env:....
: "${WAF_SOFTHSM_MODULE:?set absolute SoftHSM PKCS#11 module path}"
: "${WAF_HSM_CERT:?set PEM certificate matching the SoftHSM private key}"
: "${WAF_HSM_PIN_SECRET_REF:?set env:NAME or file:/absolute/path; never the PIN itself}"
: "${WAF_HSM_KEY_LABEL:?set exact private-key label}"

OUT=${WAF_HSM_QUALIFICATION_OUT:-qualification/hsm/softhsm-qualification-report.json}
ALLOW_DIR=${WAF_HSM_ALLOWED_MODULE_DIR:-$(dirname -- "$WAF_SOFTHSM_MODULE")}
args=(
  --module "$WAF_SOFTHSM_MODULE"
  --key-label "$WAF_HSM_KEY_LABEL"
  --cert "$WAF_HSM_CERT"
  --allow-module-dir "$ALLOW_DIR"
  --out "$OUT"
  --qualification-class softhsm
)
if [[ -n "${WAF_HSM_SLOT_ID:-}" ]]; then args+=(--slot "$WAF_HSM_SLOT_ID"); fi
if [[ -n "${WAF_HSM_TOKEN_LABEL:-}" ]]; then args+=(--token-label "$WAF_HSM_TOKEN_LABEL"); fi
if [[ -n "${WAF_HSM_KEY_ID:-}" ]]; then args+=(--key-id "$WAF_HSM_KEY_ID"); fi

exec go run -tags pkcs11 ./cmd/hsmqualify "${args[@]}"
