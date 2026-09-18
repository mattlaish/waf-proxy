# Phase 5 Slice G — External HSM / PKCS#11 qualification

> **Documentation baseline — 2026-09-17.** This file documents a component or qualification path. Repository-wide release truth lives in [`DOCUMENTATION_INDEX.md`](../../DOCUMENTATION_INDEX.md), [`SOURCE_BASELINE_GATE_RESULT.md`](../../SOURCE_BASELINE_GATE_RESULT.md), and [`TESTING_RESULTS.md`](../../TESTING_RESULTS.md). Component PASS evidence must not be promoted into a root-build, runtime, package-lifecycle, clean-host, or release PASS outside its stated scope.

This directory contains the qualification path for the Go `crypto/tls` PKCS#11 signer integration.

## Safety boundary

- The WAF never accepts an inline HSM PIN in JSON configuration or qualification CLI arguments.
- `pin_secret_ref` supports only `env:NAME` or `file:/absolute/path` references. Secret files must be regular, non-symlink files with no group/world permissions.
- Runtime and qualification reports omit the PIN, secret reference, module path, and token label.
- HSM-backed sites cannot also configure `tls_key`; there is no automatic filesystem-key fallback.
- HSM-backed sites currently require the built-in Go TLS frontend. `tls_acceleration.mode=frontend` is rejected rather than silently switching key custody.
- PKCS#11 module paths must be absolute, non-symlink regular files, not group/world writable, and located under an operator-approved `hsm.allowed_module_dirs` path.
- Slot/token and key label/ID matching is exact. Ambiguous key matches fail closed.

## SoftHSM path

Provision the SoftHSM token and private key using a secret-safe lab workflow. Create a PEM TLS certificate whose public key matches that token key. Do not put the PIN on a process command line.

Set, for example:

```bash
export WAF_SOFTHSM_MODULE=/usr/lib/softhsm/libsofthsm2.so
export WAF_HSM_SLOT_ID=123456
export WAF_HSM_KEY_LABEL=waf-tls
export WAF_HSM_CERT=/secure/lab/waf-tls-cert.pem
export WAF_HSM_PIN_SECRET_REF=file:/run/secrets/softhsm-user-pin
./qualification/hsm/run-softhsm-qualification.sh
```

The runner invokes `cmd/hsmqualify` with the native `pkcs11` build tag, which opens the real PKCS#11 module/session, logs in using the resolved secret, performs exact private-key lookup, proves certificate/public-key association by signing and verifying a challenge, and completes an in-memory TLS handshake through the HSM-backed `crypto.Signer`.

A SoftHSM PASS is evidence for the software PKCS#11 integration path only. It is **not** vendor-HSM qualification.

## Vendor HSM gate

`vendor-hsm-qualification.json` remains `NOT_RUN` until the exact production vendor library, token/slot, key custody policy, login model, failover behavior, TLS handshake path, and representative signing load are exercised on real hardware/service infrastructure.


### Real vendor runner

On the actual vendor HSM/service host, set the same selector/certificate inputs
using the production PKCS#11 module, then run:

```bash
./qualification/hsm/run-vendor-hsm-qualification.sh
```

The runner writes `vendor-hsm-qualification.json` with
`vendor_class=real_vendor_hsm`. Do not run this against SoftHSM or a mock. The
checked-in file intentionally remains `NOT_RUN` until real production-class
hardware/service infrastructure is exercised.
