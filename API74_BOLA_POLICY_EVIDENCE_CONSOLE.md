# API-7.4 — BOLA Policy / Evidence / Console

Status: `IMPLEMENTED_TESTING_DEFERRED`

Date: 2026-09-24

## Goal

API-7.4 completes the API-7 BOLA implementation track by adding explicit, bounded operator policy and evidence workflows on top of API-7.3 detection candidates. API-7.4 does **not** turn inferred BOLA evidence into request-path authorization. Learned/inferred BOLA evidence remains non-enforcing.

## Evidence-handling policy

`BOLAPolicy` is an evidence-routing policy, not an ownership or request-authorization rule. A policy is scoped only by:

- API-1 normalized `operation_id`;
- optional API-7.1 `locator_id` that must belong to the selected operation;
- optional API-7.3 candidate type;
- minimum candidate confidence (`MEDIUM` or `HIGH`).

The only actions are:

- `REVIEW` — keep matching evidence in the actionable review queue;
- `SUPPRESS` — mark matching evidence suppressed from the actionable queue while retaining the original API-7.3 candidate evidence.

There is no `ENFORCE`, `BLOCK`, `DENY`, ownership assertion, raw object selector, raw identity selector, tenant header selector, cookie selector, or arbitrary-header selector.

More-specific locator/type policy wins over a broader operation policy. Equal-specificity rules are resolved deterministically using update time and stable policy ID.

## Evidence workflow

`BOLAEvidenceReview` stores bounded operator workflow state against an API-7.3 candidate ID:

- `OPEN`
- `ACKNOWLEDGED`
- `DISMISSED`
- `RESOLVED`

Reviewed states use only enumerated reason codes:

- `INVESTIGATING`
- `EXPECTED_SHARED_RESOURCE`
- `AUTHORIZED_CROSS_TENANT`
- `TEST_TRAFFIC`
- `FALSE_POSITIVE`
- `FIX_DEPLOYED`
- `OTHER_REVIEWED`

No arbitrary free-text note is accepted. This avoids turning the evidence state into an unbounded secret/PII storage channel.

Dismissed/resolved evidence is automatically presented as `OPEN` with `reopened=true` if the underlying API-7.3 candidate receives newer detector evidence after the review action. A stale dismissal cannot permanently hide recurring evidence.

## Privacy and authority boundary

API-7.4 references existing pseudonymous/keyed evidence only. Durable API-7.4 state contains no raw JWT, subject, client, tenant claim, Authorization header, Cookie, raw path object value, query/body object value, owner header, or tenant header.

API-7.4 is deliberately absent from the API-7.2 request/background relationship processor and the API-7.3 detector path. Policy therefore cannot suppress detector generation, mutate request authorization, or create a request-path 403.

OpenAI is absent from API-7.4 policy/evidence authority.

## Resource bounds

- maximum evidence policies: 1,024;
- policy default TTL: 30 days;
- policy maximum TTL: 180 days;
- maximum persisted review records: 4,096;
- review TTL: 30 days;
- mutation request body: 16 KiB maximum;
- unknown JSON fields: rejected.

Policy/review state is mutex-protected, TTL-pruned, versioned, revalidated on restart, and persisted in `api-bola-policy.json` through the API security autosave/final-flush path.

## Admin API

Reviewer-or-higher, audited:

- `GET /api/security/bola/evidence`
- `POST /api/security/bola/evidence/{candidate_id}/workflow`
- `GET /api/security/bola/policies`
- `POST /api/security/bola/policies`
- `DELETE /api/security/bola/policies/{policy_id}`
- `GET /api/security/bola/policy-status`

The existing API-7.3 read-only candidate/status endpoints remain available.

## Admin Console

The API Security console now includes:

- API-7.3 candidate/evidence inventory;
- effective `REVIEW`/`SUPPRESS` policy state;
- `ACKNOWLEDGED`, `DISMISSED`, `RESOLVED`, and manual `OPEN`/reopen workflow actions;
- reopened-evidence visibility;
- bounded policy creation/deletion;
- operation/locator/type/confidence selectors;
- policy/evidence capacity and suppression/reopen status.

The UI explicitly states that inferred BOLA evidence cannot directly block traffic or declare ownership.

## Verification status

Executed exact-source evidence:

- API-1/2 source gate: 69 PASS
- API-3: 47 PASS
- API-4: 46 PASS
- API-5: 72 PASS
- API-6.1: 33 PASS
- API-6.2: 56 PASS
- API-6.3: 45 PASS
- API-6.4: 58 PASS
- API-7.1: 83 PASS
- API-7.2: 100 PASS
- API-7.3: 110 PASS
- API-7.4: 156 PASS
- API-7.4 targeted Go test functions present: 10
- OpenAI source contract: 16/16 PASS plus isolated package tests PASS
- WAF package-source gate: PASS
- package-builder Python tests: 9/9 PASS
- root Go source-shape gate: 113 files PASS
- changed Go `gofmt`: PASS
- embedded Admin JavaScript syntax: PASS
- primary shell/Python syntax: PASS

Canonical Go 1.25 `go mod tidy -diff`, root build/test, API-7.4 targeted tests and API-7.4 race tests remain `BLOCKED_ENVIRONMENT / NOT_RUN`: this host has Go 1.23.2, `go.mod` requires Go 1.25.0, and external toolchain retrieval is unavailable.

Therefore API-7.4 remains `IMPLEMENTED_TESTING_DEFERRED`, not `TESTED` or `RELEASED`.

## Next slice

API-8 — GraphQL Security remains `PLANNED`.
