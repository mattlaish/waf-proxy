# TLS Session Resumption + Handshake Observability — Source Gate Result

**Date:** 2026-09-30  
**Result:** `PASS` — 33 dependency-free source assertions.  
**Qualification boundary:** source/static evidence only; not Go 1.25 build/test/race evidence.

The gate verifies purpose-separated shared ticket-key derivation, secret/file
handling, `SetSessionTicketKeys` use and live rotation, listener attach/detach,
no direct HA token reuse, authoritative `DidResume` observation,
`VerifyConnection` coverage, TLS version/certificate/latency metrics, Admin API
and shipping Console wiring, regression-test presence, and one-time software
certificate leaf parsing.

Run:

```bash
python3 tools/tests/test-tls-session-resumption-observability-source.py
```

Expected output:

```text
TLS_SESSION_RESUMPTION_OBSERVABILITY_SOURCE_GATE_PASS checks=33
```

<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
