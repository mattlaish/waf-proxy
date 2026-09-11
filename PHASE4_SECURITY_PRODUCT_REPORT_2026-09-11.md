# Phase 4 Security / Product Backlog Implementation Report — 2026-09-11

## Status

`IMPLEMENTED_TESTING_DEFERRED / DEPLOYMENT_QUALIFICATION_REQUIRED`

This slice implements the owner-prioritized Phase 4 backlog on top of the Phase 3 release-engineering baseline. It does not change the existing truth boundary for real Go 1.25, Coraza v3.7.0, VectorScan/libhs, CRS replay, or production release-host qualification.

## Implemented

### L7 abuse controls

- Per-normalized-client-IP token-bucket request rate limiting.
- Per-normalized-client-IP in-flight request caps.
- Direct-peer connection caps through `http.Server.ConnState`.
- Direct-peer TLS-handshake token-bucket caps before HTTP identity exists.
- Bounded request/TLS rate-state maps with amortized pruning to prevent unbounded identity-cardinality memory growth.
- Existing trusted-proxy normalization remains authoritative for HTTP-layer client identity; transport/TLS caps necessarily operate on the direct network peer because forwarded HTTP identity is unavailable before the request is parsed.

### CIDR allow / deny

- Manual IPv4/IPv6 CIDR allow and deny lists.
- Allow-over-deny precedence for HTTP-layer decisions.
- Optional absolute RFC3339 expiry. Admin `ttl_sec` input is converted to an absolute timestamp before persistence so restart does not reset TTL.
- Operator API for list/upsert/delete with audit events.
- Expired entries fail open as inactive rather than silently extending their lifetime.

### Correlation ID and block page

- Every public HTTP request receives a WAF-generated 128-bit random `X-WAF-Request-ID`; client-supplied values are deleted and replaced.
- The ID is stored in request context, returned in the response, access logs and syslog, and injected as the Coraza transaction ID via `experimental.WAFWithOptions`.
- Match records therefore use `MatchedRule.TransactionID()` as an exact correlation key rather than IP/URI heuristics.
- Configurable HTML block-page template supports request ID/status/reason and is used for Phase 4 CIDR/rate/in-flight controls and AI blocklist enforcement.

### Persistent security state

- Atomic JSON snapshot at `/var/lib/waf-proxy/security-state.json` by default.
- Persists AI blocklist, learner aggregates, bounded notification queue, session hashes/expiry metadata, audit ring, and cumulative security counters.
- Session store keys are SHA-256 hashes of bearer tokens; reusable raw session tokens are never persisted.
- Save path is directory mode 0700 / file mode 0600 with temp-file + fsync + rename.
- Startup restore discards expired AI blocks and sessions and preserves bounded learner/notification/audit limits.
- systemd unit now declares `StateDirectory=waf-proxy` so persistence remains writable under `ProtectSystem=strict`.

### PKI Slice 3 — CRL URL lifecycle

- `backend_tls.crl_urls` supports HTTPS-only CRL endpoints with no userinfo, fragments, redirects, proxy use, or non-443 port.
- DNS answers are checked before dial; private, loopback, link-local, multicast, unspecified, or otherwise non-global-unicast addresses are rejected. The accepted resolved IP is pinned into the actual dial while TLS SNI/verification retains the original hostname.
- Response sizes/timeouts are bounded; PEM/DER CRLs retain existing date and issuer-signature validation.
- Refresh is deduplicated and memory publication is all-or-nothing; a failed refresh retains the prior valid snapshot.
- Successful URL CRLs are atomically cached under `/var/lib/waf-proxy/crl-cache` and valid cached lists are loaded as last-known-good state across restart.
- Scheduled refresh uses `refresh_sec`; admin endpoints expose per-pool status and operator-triggered refresh with audit logging.
- Hard mode still fails closed when no usable CRL snapshot exists; soft mode exposes refresh errors while retaining its last-known-good/static state.

## Admin API additions

- `GET /api/security`
- `GET /api/security/cidrs`
- `POST /api/security/cidrs/upsert`
- `POST /api/security/cidrs/delete`
- `GET /api/pki/crl`
- `POST /api/pki/crl/refresh`

Mutation routes require operator-or-higher authorization and write audit events.

## Executed validation on the packaging host

- Isolated Phase 4 security suite under the available local Go 1.23.2 toolchain: PASS with `-race`.
  - allow-over-deny
  - attacker request-ID replacement
  - custom deny page / expired CIDR behavior
  - request rate and in-flight caps
  - connection cap and TLS-handshake cap
  - bounded rate-bucket pruning
- Isolated PKI static+URL suite: PASS with `-race`.
  - all previous static CRL tests
  - URL syntax/SSRF negative tests
  - failed refresh preserves last-known-good snapshot
  - refresh deduplication
  - persistent CRL cache round trip
- Isolated persistent security-state round trip: PASS with `-race`.
  - state file mode 0600
  - raw bearer token absent
  - hashed session survives restore
  - AI/learner/notification/audit/security counter state restores
- `python3 -m json.tool config.sample.json`: PASS.
- `bash -n` for root shell scripts: PASS.
- `systemd-analyze verify` against a temporary copy with `ExecStart=/bin/true`: PASS, validating unit syntax including `StateDirectory`. Verification against the source unit itself returns only the expected missing `/usr/local/bin/waf-proxy` error in the packaging workspace.

## Deferred / not run

- Full repository `go test ./...`: BLOCKED because the packaging host has Go 1.23.2 while `go.mod` requires Go 1.25.0.
- Full repository `go vet ./...`: same blocker.
- Exact request-ID → Coraza `MatchedRule.TransactionID()` regression: source test added, but NOT_RUN here because the real Coraza dependency/toolchain gate is unavailable.
- Positive external HTTPS CRL retrieval against a real public/private PKI endpoint: NOT_RUN because the packaging environment has no usable external network/DNS path.
- Deployed Debian/Ubuntu service lifecycle, restart persistence, real backend TLS/CRL rotation, HA behavior, and load/abuse soak: NOT_RUN.
- `govulncheck`: still BLOCKED/NOT_RUN on this host as recorded by Phase 3.
- Real Go 1.25 + Coraza/VectorScan/CRS Phase 0/1/2 qualification remains open and independent.

## Packaging evidence

Pre-final `build-release-artifact.sh` verification passed with 143 source files and 151 manifest-tracked packaged files (`portable`). Final delivery is rebuilt after this report is frozen and must independently pass extraction/source byte+mode comparison and baseline-relative patch reconstruction.
