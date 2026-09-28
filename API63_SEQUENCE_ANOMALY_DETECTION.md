# API-6.3 Sequence Anomaly Detection

Date: 2026-09-24  
Status: `IMPLEMENTED_TESTING_DEFERRED`

## Scope

API-6.3 adds DETECT-only sequence anomaly evidence on top of the mature API-6.2 workflow model. The sequence engine still has no blocking authority: it does not produce `BLOCK`, `DENY`, `403`, or a sequence `ENFORCE` mode. API-6.4 remains responsible for operator mode controls, exception CRUD, reset/relearn and the full operations console.

The detector consumes only API-1 normalized operation IDs and the privacy-preserving session/workflow fingerprints already established by API-6.1/API-6.2. Verified identity context continues to enter sequence analytics only through API-5 cryptographic verification. Raw JWTs, Authorization headers, cookies, object IDs, query values, IPs and claim values are not retained in sequence violation evidence.

## Implemented anomaly classes

- `UNKNOWN_TRANSITION`: emitted only when a mature workflow has a mature/confident outgoing baseline from the current operation, no learned direct edge exists, and no more-specific skip/reversal condition applies.
- `PREREQUISITE_SKIPPED`: requires mature `A -> B` and `B -> C` evidence before observing `A -> C`; the normalized intermediate operation is retained as related evidence.
- `UNEXPECTED_ENTRY_POINT`: requires a mature workflow, minimum session evidence and a sufficiently dominant learned entry distribution before an unseen entry is reported.
- `SEQUENCE_REVERSAL`: requires a mature reverse edge. It does not require the current source operation to have an outgoing baseline because the reverse transition itself supplies the sequence-order evidence.
- `ABNORMAL_REPETITION`: combines the current session's bounded consecutive-operation history with learned transition probability. It requires a mature source baseline and a minimum run length; it does not trigger from raw request count alone.
- `WORKFLOW_DIVERGENCE`: requires a learned, non-zero direct edge with sufficient observations/sessions/age plus a high-confidence expected branch. A missing edge or `transition probability = 0` alone is not treated as workflow divergence.

Specific `PREREQUISITE_SKIPPED` and `SEQUENCE_REVERSAL` evidence suppresses duplicate generic `UNKNOWN_TRANSITION` evidence for the same observation.

## Cold-start and maturity safeguards

Detection is fail-open to learning unless the relevant workflow is `MATURE`. Transition evidence additionally checks minimum observations, unique-session evidence, minimum learning duration, staleness and confidence where applicable. `LEARNING` and `STALE` workflows do not generate sequence anomaly evidence.

The runtime exposes automatic workflow state as `LEARN` for non-mature models and `DETECT` for mature models. There is no `ENFORCE` state in API-6.3.

Default detector thresholds are bounded configuration defaults in source:

- violation TTL: 7 days;
- max retained violations: 2,048;
- max retained exception primitives: 256;
- unexpected-entry dominant baseline threshold: 50%;
- workflow-divergence expected branch minimum: 75%;
- workflow-divergence observed branch maximum: 1%;
- abnormal-repetition tail probability: 1%;
- abnormal-repetition minimum consecutive run: 3.

## Exceptions

API-6.3 implements the bounded, persisted `SequenceException` matching primitive so detection can honor exceptions. Exception selectors are limited to site, keyed workflow ID, anomaly type and normalized operation IDs. API-6.3 intentionally does not expose exception create/delete/update operations; those operator workflows remain API-6.4 scope.

## Evidence and persistence

`SequenceViolation` evidence contains only bounded normalized/keyed fields: anomaly type, timestamp, site, workflow/session fingerprints, normalized operation IDs, maturity/confidence/sample evidence and expiry. Violations are TTL-pruned and capped with eviction telemetry.

Durable sequence state advances additively from v2 to v3. The loader still accepts API-6.1 v1 and API-6.2 v2 state. v3 persists workflow learning plus bounded violations and exception primitives. Correlation-key handling remains the API-6.1 mode-0600 separate key file.

A Reviewer-gated, audited read endpoint is added at `GET /api/security/sequence/violations`. The existing model summary exposes violation/exception counts and detector thresholds. Full violation/exceptions operations UX remains API-6.4 scope.

## Runtime boundary

The HTTP request path remains the existing non-blocking bounded queue. Detection executes in the sequence state worker before the current observation mutates the transition/session history used by that decision. There is no OpenAI call and no response-writer path in the sequence detector.

## Qualification truth

Executed in this environment:

- API-1/API-2 source gate: 69 PASS;
- API-3 source gate: 47 PASS;
- API-4 source gate: 46 PASS;
- API-5 source gate: 72 PASS;
- API-6.1 source gate: 33 PASS;
- API-6.2 source gate: 56 PASS;
- API-6.3 source gate: 45 PASS;
- OpenAI integration source contract: 16/16 PASS;
- package-builder Python tests: 9/9 PASS;
- package source gate: PASS;
- root Go shape source gate: PASS, 103 root Go files;
- changed API-6.3-related Go files: `gofmt` PASS;
- embedded Admin Console JavaScript: `node --check` PASS.

Canonical Go 1.25 targeted and race tests are `BLOCKED_ENVIRONMENT / NOT_RUN`: the available local toolchain is Go 1.23.2 while `go.mod` requires Go 1.25.0. External module/toolchain retrieval is unavailable. No stub, downgraded-toolchain or source-only evidence promotes API-6.3 to `TESTED` or `RELEASED`.

Seven API-6.3 Go test functions are present for cold-start/mature unknown transitions, prerequisite/reversal semantics, unexpected-entry exceptions, repetition, non-zero-probability divergence, bounded restart/privacy persistence and concurrent bounded snapshots.

## Next slice

`API-6.4 Sequence Operations + Hardening` remains `PLANNED`.
