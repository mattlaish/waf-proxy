# API-7.3 — BOLA Detection

Status: `IMPLEMENTED_TESTING_DEFERRED`

Date: 2026-09-24

## Goal

API-7.3 converts the privacy-preserving API-7.2 identity/object relationship history into bounded **BOLA candidate evidence**. It is a detection layer only. It does not infer authoritative ownership, does not decide tenant authorization, and cannot block requests.

## Evidence authority

API-7.3 consumes only evidence that already passed earlier authority boundaries:

- API-5 cryptographically verified identity context, represented to API-7.3 only through API-7.2 HMAC pseudonyms;
- API-7.1 active, non-suppressed object locators and keyed object fingerprints;
- API-7.2 bounded relationship history.

It does not consume raw JWTs, Authorization headers, cookies, caller-supplied tenant/owner headers, raw object IDs, raw query values, or raw request-body values. OpenAI is absent from the detector authority path.

## Candidate classes

### `IDENTITY_OBJECT_DIVERGENCE`

Generated only when a new verified identity pseudonym touches an object that already has a repeated historical relationship with another verified identity. A single historical observation is insufficient. Shared-resource applications can legitimately trigger this evidence, so it is never treated as an ownership mismatch verdict.

### `TENANT_OBJECT_DIVERGENCE`

Generated only when the current request carries a cryptographically verified tenant context and the keyed object already has repeated historical evidence under a different verified tenant pseudonym. Caller-supplied tenant headers never reach this detector as authority.

### `OBJECT_ENUMERATION`

Generated only when one verified identity pseudonym reaches a bounded high fan-out of distinct keyed objects under the same locator in the recent detection window. A novel object by itself is not a candidate.

Default threshold: 20 distinct objects under the same locator within 10 minutes.

## Candidate evidence

`BOLACandidate` retains only:

- candidate ID and candidate type;
- normalized API operation ID;
- API-7.1 locator ID/location/field;
- API-7.2 identity fingerprint;
- optional verified-tenant fingerprint;
- API-7.1 keyed object fingerprint;
- evidence confidence (`MEDIUM` or `HIGH`);
- baseline identity/tenant/observation counts;
- recent identity-object fan-out count;
- bounded evidence count;
- source labels and timestamps/expiry.

No raw identity or object value is persisted.

## Runtime placement

API-7.3 executes inside the existing API-7.2 asynchronous processing plane. The request path only performs the already-established bounded API-7.2 enqueue. The detector evaluates relationship history **before** the current observation is merged into API-7.2 state, preventing the current event from creating its own baseline.

Suppressed API-7.1 locators do not reach the detector.

## Resource bounds

- candidate TTL: 7 days;
- maximum candidates: 4,096;
- maximum candidates per identity pseudonym: 256;
- maximum candidates per keyed object/locator: 128;
- minimum repeated foreign baseline: 2 observations;
- enumeration window: 10 minutes;
- enumeration threshold: 20 distinct keyed objects.

State is lock-protected, versioned, revalidated on restart, and included in API security autosave/final flush.

## Admin surface

Reviewer-or-higher, read-only and audited:

- `GET /api/security/bola/candidates`
- `GET /api/security/bola/status`

API-7.3 intentionally adds no BOLA mutation/policy route. Policy, evidence workflow, exceptions and full console work remain API-7.4 scope.

## Security boundary

API-7.3 has no `BLOCK`, `DENY`, request-path `403`, `ENFORCE`, owner-match, owner-mismatch or authoritative tenant-boundary decision. Candidate evidence is an input to later analysis/operator policy only.

## Verification status

Executed exact-source gates:

- API-1/2 source gate: 69 PASS
- API-3 source gate: 47 PASS
- API-4 source gate: 46 PASS
- API-5 source gate: 72 PASS
- API-6.1 source gate: 33 PASS
- API-6.2 source gate: 56 PASS
- API-6.3 source gate: 45 PASS
- API-6.4 source gate: 58 PASS
- API-7.1 source gate: 83 PASS
- API-7.2 source gate: 100 PASS
- API-7.3 source gate: 110 PASS
- API-7.3 targeted Go test functions present: 9
- OpenAI source contract: 16/16 PASS
- OpenAI isolated package tests: PASS
- package-source gate: PASS
- package-builder tests: 9/9 PASS
- root Go source-shape gate: 111 files PASS
- changed Go `gofmt`: PASS
- Admin embedded JavaScript syntax: PASS
- primary shell/Python syntax: PASS

Canonical Go 1.25 `go mod tidy -diff`, root build, API-7.3 targeted tests and API-7.3 race tests are `BLOCKED_ENVIRONMENT / NOT_RUN` on this host because the installed Go toolchain is 1.23.2 while `go.mod` requires Go 1.25.0, and automatic toolchain retrieval is unavailable.

Therefore API-7.3 remains `IMPLEMENTED_TESTING_DEFERRED` and is not `TESTED` or `RELEASED`.

## Next slice

API-7.4 — BOLA Policy/Evidence/Console.
