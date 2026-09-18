#!/usr/bin/env bash
set -euo pipefail

# Generic real-vendor PKCS#11 qualification runner. This does not provision or
# manage the HSM. The token/key/certificate must already exist under the
# organization's normal key-custody process. Never place the HSM PIN on the
# command line; provide only a secret reference through WAF_HSM_PIN_SECRET_REF.
: "${WAF_HSM_MODULE:?set absolute vendor PKCS#11 module path}"
: "${WAF_HSM_CERT:?set PEM certificate matching the vendor-HSM private key}"
: "${WAF_HSM_PIN_SECRET_REF:?set env:NAME or file:/absolute/path; never the PIN itself}"
: "${WAF_HSM_KEY_LABEL:?set exact private-key label}"

OUT=${WAF_HSM_QUALIFICATION_OUT:-qualification/hsm/vendor-hsm-qualification.json}
ALLOW_DIR=${WAF_HSM_ALLOWED_MODULE_DIR:-$(dirname -- "$WAF_HSM_MODULE")}
args=(
  --module "$WAF_HSM_MODULE"
  --key-label "$WAF_HSM_KEY_LABEL"
  --cert "$WAF_HSM_CERT"
  --allow-module-dir "$ALLOW_DIR"
  --out "$OUT"
  --qualification-class real_vendor_hsm
)
if [[ -n "${WAF_HSM_SLOT_ID:-}" ]]; then args+=(--slot "$WAF_HSM_SLOT_ID"); fi
if [[ -n "${WAF_HSM_TOKEN_LABEL:-}" ]]; then args+=(--token-label "$WAF_HSM_TOKEN_LABEL"); fi
if [[ -n "${WAF_HSM_KEY_ID:-}" ]]; then args+=(--key-id "$WAF_HSM_KEY_ID"); fi

exec go run -tags pkcs11 ./cmd/hsmqualify "${args[@]}"
