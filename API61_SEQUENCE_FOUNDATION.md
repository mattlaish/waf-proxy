# API-6.1 — Sequence Foundation

Status: `IMPLEMENTED_TESTING_DEFERRED` (2026-09-23).

## Implemented scope

- `SequenceSession`, `SequenceTransition`, and immutable `SequenceModel` runtime snapshots.
- API-1 normalized operation IDs are the only sequence nodes; query values and raw object identifiers are not nodes.
- Identity correlation reads only API-5 `VerifiedAPIIdentity` after cryptographic verification. A bearer token or unverified claim cannot select identity correlation.
- Anonymous correlation stores only an HMAC digest derived from a preferred session cookie or the trusted client identity/User-Agent fallback. Raw Cookie, IP, User-Agent, Authorization, JWT, subject, tenant and client values are not retained.
- The request path performs only HMAC/normalization plus a non-blocking bounded enqueue. A full queue drops telemetry and increments a counter; it does not delay or block the WAF path.
- Sessions, transitions and per-session recent operation history have fixed caps. Sessions and transitions have independent TTLs with deterministic oldest-entry eviction at cardinality limits.
- A single background consumer mutates bounded state and publishes deep-cloned immutable snapshots through `atomic.Pointer[SequenceModel]`.
- State restores and autosaves through `api-sequence.json`; the 32-byte HMAC key is kept separately in `api-sequence.key` with mode `0600`. Shutdown stops admission, drains accepted events and participates in the final API security save.
- Read-only admin model/session/transition APIs are available. Session and transition details require Reviewer RBAC and create audit entries. The embedded console labels the feature visibility-only.

## Explicitly out of scope

- API-6.2 workflow learning semantics, confidence or minimum-observation thresholds.
- API-6.3 anomaly detection, evidence, exception handling or any block action.
- API-6.4 reset/relearn/mode mutation and the full operations console.
- API-7 BOLA/object authorization analytics.

API-6.1 cannot influence API-1 through API-5 enforcement results.

## Evidence boundary

- `tools/tests/test-api61-source.py`: `PASS`, 33 checks on the working source and again on the clean-extracted artifact.
- `sequence_api61_test.go` contains five targeted tests for normalized/verified correlation, deterministic transitions, TTL/cardinality bounds, restart/privacy persistence and concurrent non-blocking snapshots.
- The current host has no Go or gofmt executable. Targeted Go tests and `go test -race` are therefore `BLOCKED_ENVIRONMENT/NOT_RUN`, not PASS.
- Repository-wide historical deferred gates were not rerun for this slice, per scope. Canonical Go 1.25 real-dependency qualification remains separate.
