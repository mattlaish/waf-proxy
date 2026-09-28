# API-6.2 — Workflow Learning

Status: `IMPLEMENTED_TESTING_DEFERRED` as of 2026-09-24.

## Scope implemented

API-6.2 extends the API-6.1 sequence foundation with LEARN-only workflow construction. It does **not** add anomaly verdicts, BLOCK/DENY/403 behavior, sequence enforcement, BOLA verdicts, reset/relearn controls, or an ENFORCE mode.

Implemented learning evidence:

- transition `from_operation_id` / `to_operation_id` from API-1 normalized operation IDs only;
- `observation_count`, bounded unique `session_count`, `first_seen`, `last_seen`, frequency-derived `confidence`, and `LEARNING` / `MATURE` / `STALE` maturity;
- workflow cohort `observation_count`, `session_count`, first/last seen, maximum observed depth, bounded entry operations, and bounded terminal operations;
- cold-start gates: minimum observations, minimum sessions, minimum learning duration, and confidence threshold;
- stale-state semantics based on observation age;
- idle timeout plus absolute session lifetime;
- bounded sessions, transitions, workflows, per-workflow operation summaries, recent operations, and per-session seen-transition evidence;
- non-blocking request-path enqueue with background state mutation and atomic immutable snapshots;
- durable workflow learning state in `api-sequence.json`, separate mode-0600 HMAC key, and explicit API-6.1 state v1 -> v2 migration.

## Identity and privacy boundary

Anonymous traffic uses keyed pseudonymous session correlation. Authenticated learning uses only `VerifiedAPIIdentity` attached after API-5 cryptographic verification. The per-session key may distinguish verified subjects, while workflow cohorts intentionally exclude subject and use a keyed fingerprint of verified issuer/tenant/client/role/scope context. Roles/scopes are normalized before fingerprinting. Raw JWTs, Authorization values, cookies, client IPs, user-agent strings, raw verified claims, query values, and object IDs are not stored in workflow nodes or durable learning state.

Unverified JWT claims never participate in sequence correlation or workflow cohorts.

## Cardinality and request-path safety

Operation nodes continue to use `apiOperationID(site, method, normalizeAPIOperationPath(path))`; routes such as `/users/1` and `/users/999999` therefore share the same normalized node. Random-session floods cannot grow active session state past `MaxSessions`. Transition, workflow, workflow-operation and per-session transition-history state also have fixed caps.

The request path performs bounded observation construction and a non-blocking channel enqueue. It does not acquire the background learning-state mutex. OpenAI is absent from API-6 request-path learning and has no sequence decision authority.

## Read-only visibility

Reviewer-gated, audited read-only endpoints now include:

- `GET /api/security/sequence/model`
- `GET /api/security/sequence/sessions`
- `GET /api/security/sequence/transitions`
- `GET /api/security/sequence/workflows`

The embedded console shows workflow maturity, observations, sessions, depth, transition confidence and maturity. API-6.4 mutation controls are intentionally not present.

## Evidence boundary

- API-6.1 source gate: `PASS`, 33 checks.
- API-6.2 source gate: `PASS`, 56 checks.
- Admin inline JavaScript syntax: `PASS` with Node `--check`.
- Seven targeted API-6.2 Go tests are present for learning/confidence/maturity, cold-start/stale state, absolute session lifetime/terminal learning, verified identity cohorts, normalized-cardinality/random-session bounds, restart/privacy persistence, and concurrent bounded learning/snapshots.
- Canonical Go 1.25 targeted tests and race execution: `BLOCKED_ENVIRONMENT / NOT_RUN`. The available local toolchain is Go 1.23.2 while `go.mod` requires Go 1.25.0; automatic toolchain/dependency retrieval is unavailable in this environment.
- A Go 1.23 compatibility compile attempt with module lookup disabled was also blocked by unavailable real dependencies and is not qualification evidence.

No stub or downgraded-toolchain result is used to promote the slice. Status therefore remains `IMPLEMENTED_TESTING_DEFERRED`.
