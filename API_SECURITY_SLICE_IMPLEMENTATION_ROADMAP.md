# API Security Slice Implementation Roadmap

## Current canonical baseline — 2026-09-28

The current source is the **Code Duplication Review and Consolidation** working
baseline derived byte-for-byte from the Production Correctness & Control-Plane
Hardening parent artifact (`90acc977db2998f4a3c1c4cacc07e19f0087f8bc033bce6b9934b83415957be8`)
before the changes documented in `CODE_DUPLICATION_REVIEW.md`. Status remains
**IMPLEMENTED_TESTING_DEFERRED**. No API-9 is defined.

The review removed only source layers proven to be unwired, superseded or
functionally duplicative, and consolidated the duplicated SecLang action/token
parser into `internal/capability`. Distinct API-6/API-7/API-8 state machines,
workers and authority boundaries remain separate. The shipping Console remains
`static/admin.html` + `static/theme.css`; the previously removed experimental
`web/` tree is not part of the current source.

Current dependency-free evidence: API source gates
**69/47/46/72/33/56/45/58/83/100/110/156/259/134/190 PASS**, code-duplication
source gate **80/80 PASS with 139 unique Admin/update routes**, OpenAI source
contract **16/16 PASS** plus isolated tests PASS, WAF package-source PASS,
package-builder **9/9 PASS**, and root Go source-shape **95 files PASS**. An
isolated dependency-free `internal/capability` test also passes. Canonical Go
1.25 tidy/build/vet/test/race remains **BLOCKED_ENVIRONMENT / NOT_RUN** on this
host; none of these static/source results promotes the product to TESTED or
RELEASED.

Dated sections below are retained as historical engineering/evidence records.
When an older section conflicts with this section, `CODE_DUPLICATION_REVIEW.md`,
`DOCUMENTATION_INDEX.md`, and the current source tree are authoritative.

## Purpose

This document is the canonical implementation blueprint for API-aware WAF security evolution.

Each slice defines scope, architecture, components, APIs, tests, dependencies, and security boundaries.

## Slice Status

| Slice | Name | Status |
|---|---|---|
| API-1 | API Discovery + Operation Normalization | IMPLEMENTED_TESTING_DEFERRED |
| API-2 | Typed Schema Learning | IMPLEMENTED_TESTING_DEFERRED |
| API-3 | OpenAPI Contract Management | IMPLEMENTED_TESTING_DEFERRED |
| API-4 | Positive Schema Enforcement | IMPLEMENTED_TESTING_DEFERRED |
| API-5 | JWT + Identity-aware API Security | IMPLEMENTED_TESTING_DEFERRED |
| API-6.1 | Sequence Foundation | IMPLEMENTED_TESTING_DEFERRED |
| API-6.2 | Workflow Learning | IMPLEMENTED_TESTING_DEFERRED |
| API-6.3 | Sequence Anomaly Detection | IMPLEMENTED_TESTING_DEFERRED |
| API-6.4 | Sequence Operations + Hardening | IMPLEMENTED_TESTING_DEFERRED |
| API-7.1 | Object Locator Discovery | IMPLEMENTED_TESTING_DEFERRED |
| API-7.2 | Identity/Object Relationship | IMPLEMENTED_TESTING_DEFERRED |
| API-7.3 | BOLA Detection | IMPLEMENTED_TESTING_DEFERRED |
| API-7.4 | BOLA Policy/Evidence/Console | IMPLEMENTED_TESTING_DEFERRED |
| API-8 | GraphQL Security | IMPLEMENTED_TESTING_DEFERRED |

## API-1 — API Discovery + Operation Normalization

Goal:
- discover API operations
- normalize paths
- maintain stable operation identity

Implemented components:
- operation discovery and normalization engine
- site-scoped stable operation identity/fingerprinting
- integer/UUID/ULID/hex/date/opaque-token path classification
- host/status/content-type/auth-family metadata
- bounded raw path examples and path-parameter metadata
- durable `api-operations.json` state
- detail/ignore/reclassify APIs and embedded inventory UI

Current evidence:
- exact-source isolated deterministic API-1 tests: 5/5 PASS, including concurrent atomic persistence
- source integration gate included in `tools/tests/test-api12-source.py`
- root Go 1.25 qualification: BLOCKED/NOT_RUN

## API-2 — Typed Schema Learning

Goal:
Learn application request schema characteristics from observations without storing sensitive values.

Implemented components:
- live bounded schema observation collector independent of Passive Discovery/AI enablement
- recursive JSON/array, query, path, header, form and multipart metadata inference
- type/format stability, presence/confidence, bounded cardinality, conservative enum and required detection
- privacy-first value handling and body-sample backlog shedding
- semantic `LEARNING → CANDIDATE → REVIEWED` lifecycle
- durable `api-schema.json` state and restart continuation
- OpenAI Responses API Structured Outputs review adapter, advisory only
- bounded aggregate field growth and review invalidation on dominant-type changes
- composite auth-family telemetry (`bearer`, API key, mTLS) without secret values

Data:
- SchemaObservation
- SchemaFieldObservation
- SchemaCandidate
- SchemaVersion
- SchemaReview

Boundary:
- learning only
- no automatic enforcement
- no automatic blocking
- no raw secret/value storage

## API-3 — OpenAPI Contract Management

Goal:
- correlate declared OpenAPI contracts with API-1 observed operations and API-2 learned schema metadata.

Components:
- OpenAPI 3.0/3.1 YAML/JSON parser and validation
- bounded local `$ref` resolution
- content-hash immutable contract versions
- operation matcher and binding confidence
- request/response/parameter/security contract comparator
- multi-content-type inventory/drift and OpenAPI OR/AND/anonymous auth-requirement semantics
- undeclared-endpoint drift scoped to matched site/host traffic
- external `$ref` rejection and normalized-operation ambiguity rejection
- declared-vs-live API-1/API-2 drift reporter
- durable `api-contracts.json` state
- OpenAPI JSON/YAML exporter

Data:
- `APIContract`
- `APIContractVersion`
- `ContractOperation`
- `ContractSchemaField`
- `OperationBinding`
- `ContractDiff`
- `DriftEvent`

API surface:
- `GET /api/security/contracts`
- `POST /api/security/contracts/import`
- `GET /api/security/contracts/{id}`
- `GET /api/security/contracts/{id}/versions`
- `POST /api/security/contracts/{id}/match`
- `GET /api/security/contracts/{id}/bindings`
- `POST /api/security/contracts/{id}/compare`
- `GET /api/security/contracts/{id}/diffs`
- `GET /api/security/contracts/{id}/drift`
- `GET /api/security/contracts/{id}/export`

Security boundary:
- intelligence only; no blocking or policy activation
- direct upload only; no remote `$ref` fetch
- import body bounded to 8 MiB
- ambiguous operation matches remain reviewable instead of auto-selected

Tests/evidence:
- `api3_contract_test.go`
- `api_security_integration_test.go`
- `tools/tests/test-api3-source.py`
- `API3_TEST_MATRIX.md`
- `API3_ACCEPTANCE_CRITERIA.md`
- API-3 source gate: 47 checks PASS
- exact-source API-3 compile-only gate: PASS (minimal YAML shim; no YAML runtime claim)
- four deterministic API-3 runtime/integration tests not requiring YAML behavior: PASS
- API-1 → API-2 → API-3 deterministic integration: PASS

Status: `IMPLEMENTED_TESTING_DEFERRED`; Go 1.25 execution is deferred to the final qualification gate.

## API-4 — Positive Schema Enforcement

Implemented with immutable reviewed profiles, LEARN/DETECT/ENFORCE lifecycle, scoped exceptions, rollback, bounded durable violation evidence, and atomic runtime policy snapshots. API-4 deterministic tests 7/7 PASS, targeted race PASS, source gate 46 checks PASS.

Status: `IMPLEMENTED_TESTING_DEFERRED`.

## API-5 — JWT + Identity-aware API Security

Implemented with trusted issuer/JWKS configuration, HTTPS-only bounded JWKS cache/rotation, verified-claims-only request context, issuer/audience/exp/nbf/algorithm/signature validation, role/scope/tenant/client policy, DETECT→ENFORCE lifecycle, semantic-change rollback to DETECT, bounded privacy-preserving violation evidence, and Admin API/UI. Unknown-`kid` refresh is serialized and rate-bounded.

Evidence: 17 deterministic API-5 test functions PASS, targeted race PASS, source gate 72 checks PASS, admin inline JavaScript syntax PASS, and whole-repository compile-shape build/vet/test-binary compilation PASS with external dependencies stubbed.

Status: `IMPLEMENTED_TESTING_DEFERRED`; canonical Go 1.25 real-dependency qualification remains blocked on this host.

## API-6.1 — Sequence Foundation

Implemented with a bounded non-blocking telemetry queue, `SequenceSession` / `SequenceTransition` / immutable atomic `SequenceModel`, API-1 normalized operation nodes, API-5 verified-identity-only correlation, HMAC anonymous correlation, TTL/cardinality/history controls, durable restart state, final drain, and Reviewer-gated/audited read-only admin visibility. It is visibility-only and has no anomaly or block path.

Evidence: API-6.1 source gate 33 checks PASS. Five targeted deterministic/restart/bounded/concurrency tests exist; Go/race execution is `BLOCKED_ENVIRONMENT/NOT_RUN` because the host has no Go toolchain.

Status: `IMPLEMENTED_TESTING_DEFERRED`.

## API-6.2 — Workflow Learning

Implemented as a LEARN-only extension to API-6.1. `SequenceTransition` now records observation count, bounded unique-session count, first/last seen, frequency-derived confidence and `LEARNING` / `MATURE` / `STALE` maturity. `SequenceWorkflowModel` records bounded cohort observations/sessions, workflow depth and entry/terminal operation summaries. Cold-start thresholds require minimum observations, sessions and learning duration; transitions also require confidence threshold. Sessions have idle and absolute lifetime limits. State is bounded, durable and restart-safe with an explicit v1->v2 migration. Identity cohorting consumes only API-5 `VerifiedAPIIdentity`; raw tokens/claims are not persisted, and subject is excluded from workflow cohort material.

Evidence: API-6.1 source gate 33 PASS; API-6.2 source gate 56 PASS; Admin JS syntax PASS. Seven targeted API-6.2 Go tests are present. Canonical Go 1.25 targeted/race execution is `BLOCKED_ENVIRONMENT/NOT_RUN` because only Go 1.23.2 is locally available and dependency/toolchain retrieval is unavailable.

Status: `IMPLEMENTED_TESTING_DEFERRED`. No anomaly verdict, block path, reset/relearn control or ENFORCE mode exists.

## API-6.3 — Sequence Anomaly Detection

Implemented as a DETECT-only extension to the mature API-6.2 workflow model. It emits bounded privacy-preserving evidence for `UNKNOWN_TRANSITION`, `PREREQUISITE_SKIPPED`, `UNEXPECTED_ENTRY_POINT`, `SEQUENCE_REVERSAL`, `ABNORMAL_REPETITION`, and `WORKFLOW_DIVERGENCE`. Detection requires mature workflow and applicable sample/session/age/confidence evidence; probability zero alone is insufficient for workflow divergence. Bounded persisted exception primitives are consumed by the detector, but operator CRUD remains API-6.4 scope. Durable state advances to v3 with v1/v2 restore compatibility. The sequence engine still has no BLOCK/DENY/403 or ENFORCE authority.

Evidence: API-6.3 source gate 45 PASS; retained API-6.1/API-6.2 gates 33/56 PASS; seven targeted Go tests are present; canonical Go 1.25 targeted/race is `BLOCKED_ENVIRONMENT/NOT_RUN` on the local Go 1.23.2 host.

Status: `IMPLEMENTED_TESTING_DEFERRED`.

## API-6.4 — Sequence Operations + Hardening

Implemented with explicit site-scoped `LEARN` / `DETECT` controls, mature-only DETECT promotion, v4 durable control/recent-session state with v1/v2/v3 restore compatibility, bounded recent-session summaries, site-scoped reset/relearn, bounded exception CRUD, Reviewer RBAC + audit, and the complete embedded Sequence Operations console. API-6 learned sequence evidence still has no BLOCK/DENY/403 or ENFORCE authority.

Evidence: API-6.4 source gate 58 PASS; retained API-6.1/API-6.2/API-6.3 gates 33/56/45 PASS; canonical Go 1.25 targeted/race is `BLOCKED_ENVIRONMENT/NOT_RUN` on the local Go 1.23.2 host.

Status: `IMPLEMENTED_TESTING_DEFERRED`.

## Roadmap completion / remaining qualification

API-7.1 through API-7.4 and API-8 GraphQL Security are implemented with testing deferred. The documented API-security implementation roadmap is complete through API-8. No API-9 is defined here. Remaining work is canonical Go 1.25/runtime/race/real-dependency qualification and release promotion only if all required gates pass.

## API-8 — GraphQL Security

Status: `IMPLEMENTED_TESTING_DEFERRED`.

Implemented as six integrated capabilities: bounded parsing/normalization; SDL contract import and deterministic field-topology validation; depth/complexity/field/mutation/subscription controls; introspection and persisted-query policy; verified GraphQL-variable integration into API-7 keyed evidence; and Reviewer-gated deterministic `LEARN -> DETECT -> ENFORCE` policy/evidence/console. Policy edits and bound-contract changes force LEARN; referenced contracts cannot be deleted. Only explicit GraphQL ENFORCE policy can block. API-6/API-7 learned evidence and OpenAI have no GraphQL enforcement authority.

Evidence: API-8 source gate 259 PASS; 22 targeted API-8 Go tests present; all prior API source gates remain PASS. Canonical Go 1.25 targeted/race/build qualification remains `BLOCKED_ENVIRONMENT/NOT_RUN`.

## Design Principles

1. Learn before enforce.
2. AI assists review; AI does not become policy authority.
3. Sensitive values are not stored as evidence.
4. Every slice maintains source, tests, documentation, and handover state.


## API-7.1 Object Locator Discovery checkpoint — 2026-09-24

API-7.1 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. It adds bounded `ObjectLocator` discovery from API-1 normalized path parameters, API-2 typed path/query/body evidence, matched API-3 OpenAPI contracts, and explicit Reviewer-gated operator INCLUDE/SUPPRESS configuration. Locators are keyed by normalized API-1 operation ID plus location/field; raw object values are transient only and durable value evidence is bounded HMAC-SHA256 fingerprint data protected by a separate mode-0600 key. Client-supplied ownership/tenant headers are not authority, and GraphQL `variables.*` discovery is deferred to API-8.

API-7.1 has no identity/object relationship verdict, tenant-boundary verdict, `BOLA_CANDIDATE`, ownership verdict, BLOCK/DENY/403 path, or ENFORCE authority. OpenAI is absent from the locator authority path. State is bounded by global/per-operation/fingerprint/override caps, 30-day learned TTL, versioned restart validation, existing non-blocking observation-plane ingestion, immediate mutation persistence, RBAC and audit.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83 PASS** from API-1/2 through API-7.1; API-7.1 has **9 targeted test functions present**; OpenAI source contract **16/16 PASS** plus isolated tests PASS; package-source PASS; root Go shape **107 files PASS**; package-builder **9/9 PASS**; changed-file gofmt, Admin JavaScript and primary shell syntax PASS. Canonical Go 1.25 targeted/race remains `BLOCKED_ENVIRONMENT/NOT_RUN` because the host has Go 1.23.2 and external toolchain retrieval is unavailable. Next slice: **API-7.2 Identity/Object Relationship**. API-7.3/API-7.4 and API-8 remain `PLANNED`.

## API-7.1 — Object Locator Discovery

Implemented as a bounded discovery-only foundation for later BOLA analytics. Sources are API-1 normalized path parameters, API-2 typed path/query/body evidence, matched API-3 OpenAPI contracts, and explicit operator INCLUDE/SUPPRESS configuration. Durable values are keyed fingerprints only; raw object values, client-reported ownership/tenant headers, and GraphQL variables are excluded from authority. No relationship, BOLA verdict, or blocking primitive exists.

Status: `IMPLEMENTED_TESTING_DEFERRED`. Source gate 83 PASS; nine targeted Go tests are present; canonical Go 1.25 targeted/race remains `BLOCKED_ENVIRONMENT/NOT_RUN` on the Go 1.23.2 host.

## API-7.2 — Identity/Object Relationship

Status: `IMPLEMENTED_TESTING_DEFERRED`. Correlates API-5 cryptographically verified identity context with ACTIVE API-7.1 keyed object evidence using dedicated keyed pseudonyms. Durable state contains no raw subject, tenant, client_id, JWT, cookie, owner header, or object value. The bounded async model has TTL/global/per-identity/per-object caps, restart validation, Reviewer read-only visibility, audit, and no ownership/tenant-boundary/BOLA/blocking authority. Canonical Go 1.25 targeted/race remains `BLOCKED_ENVIRONMENT/NOT_RUN`. Next: API-7.3 BOLA Detection.

## API-7.3 — BOLA Detection

Status: `IMPLEMENTED_TESTING_DEFERRED`. Bounded non-enforcing BOLA candidate evidence is implemented from API-5 verified identity pseudonyms, ACTIVE API-7.1 keyed object evidence and mature API-7.2 relationship history. Candidate classes are identity/object divergence, verified-tenant/object divergence and recent per-locator object enumeration. State is TTL/cardinality bounded and restart-safe; read-only Reviewer APIs are audited. No ownership verdict, tenant-boundary verdict, BOLA blocking, `DENY`, request-path `403` or `ENFORCE` authority exists. Source gate 110 PASS; nine targeted Go tests are present; canonical Go 1.25 targeted/race remains `BLOCKED_ENVIRONMENT/NOT_RUN`. Next: API-7.4 BOLA Policy/Evidence/Console.

## API-7.4 — BOLA Policy/Evidence/Console

Status: `IMPLEMENTED_TESTING_DEFERRED`. Adds bounded `REVIEW`/`SUPPRESS` evidence-routing policy, structured `OPEN`/`ACKNOWLEDGED`/`DISMISSED`/`RESOLVED` operator workflow, recurrence-driven reopen behavior, Reviewer RBAC/audit, restart-safe bounded persistence and the complete BOLA evidence/policy console. Inferred BOLA anomaly remains non-enforcing and cannot directly block, deny, return request-path 403, enter ENFORCE, or declare ownership. Source gate 156 PASS; ten targeted Go tests are present; canonical Go 1.25 targeted/race remains `BLOCKED_ENVIRONMENT/NOT_RUN`. Next: API-8 GraphQL Security.

## API-8 GraphQL Security checkpoint — 2026-09-24

API-8 is implemented across parsing/normalization, SDL contract binding, deterministic complexity/field/mutation/subscription/introspection policy, Apollo persisted-query support, verified GraphQL-variable integration into API-7 evidence, and Reviewer-gated `LEARN` / `DETECT` / `ENFORCE` policy plus Admin Console. Any policy edit or bound contract content change returns the policy to `LEARN`; a referenced schema contract cannot be deleted. Only explicit deterministic GraphQL policy in `ENFORCE` may reject traffic. API-6/API-7 learned or inferred evidence and OpenAI cannot promote, mutate or bypass GraphQL enforcement.

Durable GraphQL state excludes raw query documents, literals, raw variable values, Authorization/Cookie/JWT data and unverified claims. SDL is reduced to a digest plus normalized topology; GraphQL object values use API-7 keyed fingerprints and API-5 verified identity only. State and parser resources are bounded, versioned, restart-revalidated, autosaved/final-flushed, and strict mutation APIs are Reviewer-gated and audited.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83/100/110/156/259 PASS** through API-8; **22 targeted API-8 Go test functions are present**; OpenAI source contract **16/16 PASS** plus isolated tests PASS; package-source PASS; package-builder **9/9 PASS**; root Go source shape **115 files PASS**; changed API-8 Go files `gofmt`, Admin embedded JavaScript, relevant shell and Python syntax PASS. Canonical Go 1.25 `go mod tidy -diff`, root build/vet/test, API-8 targeted tests and race tests remain `BLOCKED_ENVIRONMENT/NOT_RUN` because the host has Go 1.23.2 and external toolchain retrieval is unavailable. API-8 remains `IMPLEMENTED_TESTING_DEFERRED`, not `TESTED` or `RELEASED`.

The documented API-security implementation roadmap is now complete through API-8. No API-9 is defined. Next gate: canonical Go 1.25 qualification of the exact source/artifact, followed by release promotion only if all required gates pass.

## 2026-09-25 — Production Correctness & Control-Plane Hardening

Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.

## 2026-09-28 maintenance note

The repository-wide duplicate-functionality cleanup does not add, remove or
reorder API-security slices. API-1 through API-8 remain implemented with testing
deferred. No API-9 has been defined.

<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
