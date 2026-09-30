# TLS Session Resumption + Handshake Observability

**Implementation date:** 2026-09-30  
**Parent complete-source baseline:** `waf-proxy-build-admin-startup-blocker-fix-2026-09-30.zip`  
**Parent SHA-256:** `49450dfffeca2e45f17def136e122fac115c8fba4e8beb70a9307fe5eb6cb0e3`  
**State:** `IMPLEMENTED_TESTING_DEFERRED`  
**API-security boundary:** not API-9; API-1 through API-8 authority is unchanged.

## Scope

This slice addresses the built-in Go TLS termination path only:

1. make TLS session-ticket encryption keys stable across process restarts and
   across an HA pair when both nodes receive the same purpose-separated secret;
2. rotate that key material without dropping immediately previous tickets;
3. measure successful full versus resumed handshakes from authoritative
   `tls.ConnectionState.DidResume` state;
4. expose TLS version, full-handshake certificate key algorithm, successful
   handshake processing latency, and TLS handshake policy rejects;
5. surface the new measurements in the existing Admin metrics API and shipping
   Console without creating a second TLS or WAF enforcement path.

External TLS frontend termination remains a separate termination authority. This
slice does not claim to configure session tickets inside nginx or another
external frontend.

## Session-ticket key design

`crypto/tls` process-local automatic ticket keys remain the compatibility
fallback when no WAF shared secret is configured. In that mode `/api/status`
and `/api/metrics` report `process_local_auto`; HA restart/cross-node resumption
is not claimed.

For HA/restart-stable resumption, operators configure exactly one of:

- `WAF_TLS_SESSION_TICKET_SECRET` — a raw value of at least 32 bytes, or
  `base64:<standard-base64>` which decodes to at least 32 bytes; or
- `WAF_TLS_SESSION_TICKET_SECRET_FILE` — a regular file that is not readable or
  writable by group/other (for example mode `0600`).

The value is runtime-only. It is not added to `Config`, not HA-synchronized in
`config.json`, not returned by Admin APIs, and not logged. The two HA nodes must
receive the same value from the deployment secret system. The value must be
purpose-separated and must not reuse `WAF_HA_PEER_TOKEN`.

The manager derives a `[32]byte` ticket key with HMAC-SHA256 over the fixed
purpose label `waf-proxy/tls-session-ticket/v1/<epoch>`. The epoch is UTC Unix
time divided by 24 hours. The active ring contains seven epochs: the current key
first (sign/encrypt) and six preceding keys (decrypt only through
`SetSessionTicketKeys` ordering). This mirrors Go's prior one-day/seven-day
automatic rotation horizon while making the ring deterministic across nodes.
Calling `SetSessionTicketKeys` disables Go automatic rotation, so the manager
periodically advances every attached built-in Go TLS config when the UTC epoch
changes. Listener teardown unregisters the config.

## Handshake observability

The prior metric `tls_hs_per_sec` is retained for compatibility and now has an
explicit meaning: ClientHello/handshake attempts per second after
`GetConfigForClient` is reached.

For each accepted ClientHello, the server creates a connection-local
`tls.Config` clone and installs `VerifyConnection`. Go documents that
`VerifyConnection` executes for successful full and resumed handshakes. The
callback records:

- successful handshakes;
- full handshakes;
- resumed handshakes and resumption percentage;
- TLS 1.2 versus TLS 1.3;
- RSA, ECDSA, Ed25519, or other/unknown certificate key algorithm for full
  handshakes;
- server-side handshake processing time measured from the ClientHello policy
  callback to successful `VerifyConnection`, with average and bounded p95
  histogram output;
- L7 TLS-handshake policy rejects separately from successful handshakes.

The latency metric intentionally does not claim TCP-connect latency or
client-side certificate validation latency.

Software-loaded certificates now parse and retain `Certificate.Leaf` once at
runtime construction so certificate algorithm inspection does not repeatedly
parse the leaf during every full handshake. HSM-loaded certificates already
provide `Leaf`.

## Metrics surfaces

`GET /api/metrics` now includes cumulative `tls` details and
`tls_session_tickets`, while the rolling `current`/`history` samples add:

- `tls_success_per_sec`
- `tls_full_per_sec`
- `tls_resumed_per_sec`
- `tls_resume_pct`
- `tls_policy_rejected_per_sec`
- `tls_12_per_sec`
- `tls_13_per_sec`
- `tls_ecdsa_per_sec`
- `tls_rsa_per_sec`
- `tls_ed25519_per_sec`
- `tls_handshake_avg_ms`
- `tls_handshake_p95_ms`

The shipping Console Dashboard adds TLS attempt rate, resumption percentage,
and handshake-processing p95 cards. Backend RBAC remains authoritative; the
metrics endpoint keeps its existing authenticated Admin API boundary.

## Tests and qualification truth

Implemented repository regressions in `tls_session_test.go` cover deterministic
HA key derivation, rotation overlap, secret/file validation, cross-config TLS
1.2 resumption with a shared key secret, full/resumed/version/RSA/latency
observation, and policy-reject accounting.

A dependency-free isolated exact-core probe using the host's Go 1.23.2 runtime passed
all five repository TLS tests (including TLS 1.2 and TLS 1.3 cross-config
resumption) and also passed `go test -race`. It validates the new
standard-library-only core; it is **not** a substitute for the repository's
required Go 1.25 qualification. This probe validates the new standard-library-only core; it
is **not** a substitute for the repository's required Go 1.25 qualification.

The exact repository still requires Go >= 1.25.0 and external modules. On this
host, Go 1.25 toolchain auto-fetch and module downloads are blocked by outbound
DNS/network restrictions. Therefore exact-source `go mod tidy -diff`, build,
vet, full tests, race, real-Coraza, runtime package build, and package lifecycle
qualification remain `BLOCKED_ENVIRONMENT / NOT_RUN` unless separately recorded
as actually executed.

Dependency-free source/static gates remain separate evidence and do not promote
the product to `TESTED` or `RELEASED`.

<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
