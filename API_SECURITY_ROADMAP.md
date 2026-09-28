# API Security Roadmap — API-aware WAF Evolution

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

This document defines the future API-aware WAF product line built on the existing
WAF foundation.

The goal is not to replace Coraza/CRS. Coraza, CRS, VectorScan and future CPU
acceleration remain the high-speed dataplane. API Intelligence adds positive
security and application-behavior understanding.

## Existing Foundation (Implemented)

The current baseline already contains:

- passive request observation
- site/path discovery foundation
- crawler-based discovery foundation
- request shape signals
- field discovery
- PagePolicy / FieldPolicy positive validation
- profile suggestion
- OpenAI-assisted profile review

These capabilities are foundation only. They do not represent completion of the
API Security slices below.

## Roadmap Status

Current source status is recorded in the table below. API-1 through API-8 are implemented in source and remain `IMPLEMENTED_TESTING_DEFERRED`. The API-security implementation roadmap is complete through API-8; no API-9 is defined. Canonical Go 1.25 runtime/race qualification is still required before release promotion.

---

## Current Slice Status — 2026-09-24

| Slice | Status |
|---|---|
| API-1 Discovery + Operation Normalization | IMPLEMENTED_TESTING_DEFERRED |
| API-2 Typed Schema Learning | IMPLEMENTED_TESTING_DEFERRED |
| API-3 OpenAPI Contract Management | IMPLEMENTED_TESTING_DEFERRED |
| API-4 Positive Schema Enforcement | IMPLEMENTED_TESTING_DEFERRED |
| API-5 JWT + Identity-aware API Security | IMPLEMENTED_TESTING_DEFERRED |
| API-6.1 Sequence Foundation | IMPLEMENTED_TESTING_DEFERRED |
| API-6.2 Workflow Learning | IMPLEMENTED_TESTING_DEFERRED |
| API-6.3 Sequence Anomaly Detection | IMPLEMENTED_TESTING_DEFERRED |
| API-6.4 Sequence Operations + Hardening | IMPLEMENTED_TESTING_DEFERRED |
| API-7.1 Object Locator Discovery | IMPLEMENTED_TESTING_DEFERRED |
| API-7.2 Identity/Object Relationship | IMPLEMENTED_TESTING_DEFERRED |
| API-7.3 BOLA Detection | IMPLEMENTED_TESTING_DEFERRED |
| API-7.4 BOLA Policy/Evidence/Console | IMPLEMENTED_TESTING_DEFERRED |
| API-8 GraphQL Security | IMPLEMENTED_TESTING_DEFERRED |

Go 1.25 build/test execution for the current API Security source line is deferred to the final qualification gate.

Current local source evidence: API-1 5/5 exact-source isolated tests PASS; API-2 8/8 exact-source isolated tests PASS; API-1/API-2 source gate 69 checks PASS; API-3 source gate 47 checks PASS; API-3 compile-only exact-source gate PASS with a minimal YAML shim; four API-3 runtime/integration tests that do not depend on YAML behavior PASS. These scoped results do not promote repository-root buildability.

# API-1 — API Discovery + Operation Normalization

## Goal

Build a canonical API inventory from observed traffic.

## Scope

Implement:

- API operation entity
- HTTP method tracking
- normalized route templates
- path parameter classification
- endpoint lifecycle tracking
- API inventory UI/API

Examples:

```
/api/users/1001
/api/users/1002

becomes

GET /api/users/{id}
```

## Implemented model/runtime

- `apiOperation` with site-scoped stable ID and method/path fingerprint
- host, status, content-type and authentication-family distributions
- bounded raw-path examples and typed `APIPathParameter` metadata
- durable `api-operations.json` restore/autosave/shutdown flush
- detail, ignore/unignore and reclassify APIs plus embedded admin inventory UI

## Security Boundary

Detection and inventory only.

No blocking.

## Acceptance Criteria

- repeated dynamic paths normalize correctly
- sensitive values are not stored
- tenant/site isolation is preserved

---

# API-2 — Typed Schema Learning

## Goal

Convert observed request traffic into candidate API schemas.

## Scope

Learn:

- JSON fields
- query parameters
- headers
- form fields
- arrays
- nested objects

Generate candidates:

```yaml
amount:
  type: number

currency:
  enum:
    - USD
    - TWD

user_id:
  format: uuid
```

## Implemented model/runtime

- live bounded request observations feeding `schemaStore`
- path/query/header/body typed samples with nested JSON/array, form and multipart metadata support
- aggregate type/format/presence/cardinality/enum/required evidence
- `SchemaCandidate` lifecycle `LEARNING → CANDIDATE → REVIEWED`
- semantic-version review invalidation rather than invalidation on every sample
- durable `api-schema.json` state and restart continuation
- advisory OpenAI Structured Outputs review with no enforcement authority

## Security Boundary

Learning only.

No automatic enforcement.

---

# API-3 — OpenAPI Contract Management

## Goal

Connect declared API contracts with observed behavior.

## Scope

Implement:

- OpenAPI 3.x YAML/JSON import and JSON/YAML export
- durable content-hash contract versioning
- path/query/header/cookie parameters, enums, security schemes/requirements
- request/response/parameter/security contract comparison
- live API-1/API-2 contract drift detection

Detect:

- undocumented and missing endpoints
- type drift
- enum drift
- content-type drift
- authentication-family drift
- required-field / undeclared-field drift

---

# API-4 — Positive Schema Enforcement

## Goal

Move approved API profiles into enforcement.

Modes:

```
LEARN
DETECT
ENFORCE
```

Support:

- schema versioning
- exceptions
- rollback
- shadow mode
- violation evidence

Current implementation (2026-09-23):

- only a current operator-approved `SchemaCandidate` may be promoted; AI review alone cannot activate enforcement
- immutable positive-schema profile versions are derived from reviewed candidate evidence
- deployment lifecycle is `LEARN -> DETECT -> ENFORCE`; direct `LEARN -> ENFORCE` is rejected
- `DETECT` is shadow mode; violations are evidenced but the request continues
- `ENFORCE` blocks only non-exempt violations with a generic 400/403/422 response
- required/type/format/enum/content-type/unknown-field checks are supported; format/content-type and unknown-field strictness are explicit promotion choices
- oversized bodies that exceed the bounded validation capture fail closed in ENFORCE rather than being accepted from a prefix
- scoped exceptions can match operation/profile/violation/field and may expire
- activation/mode history supports rollback to the prior profile+mode
- violation evidence is bounded, privacy-conservative, and durable in `api-positive-schema.json`
- data-plane policy reads use an atomic immutable snapshot; control-plane mutations rebuild the snapshot
- Admin UI exposes promotion, lifecycle changes, rollback, exceptions, and recent violation evidence

Status: `IMPLEMENTED_TESTING_DEFERRED`. Local evidence: API-4 deterministic tests 7/7 PASS, targeted race PASS, API-4 source gate 46 checks PASS, and whole-repository test-binary compilation PASS under the dependency-stub compile-shape harness. Canonical Go 1.25 real-dependency tidy/build/vet/full-test/race/real-Coraza qualification remains NOT_RUN/BLOCKED on this host.

---

# API-5 — JWT + Identity-aware API Security

## Goal

Only cryptographically verified identity claims may enter API policy context.

Implemented:

- trusted issuer registry with exact issuer, accepted audiences, HTTPS JWKS URL, algorithm allow-list, bounded clock skew and cache TTL;
- JWT verification for RSA PKCS#1 v1.5, RSA-PSS, ECDSA and Ed25519 families (`RS*`, `PS*`, `ES*`, `EdDSA`) with algorithm/key-type and EC-curve matching;
- fail-closed rejection of `alg=none`, unsupported critical headers, `b64=false`, malformed/trailing JWT JSON, ambiguous JWKS keys, invalid issuer/audience/`exp`/`nbf`, and invalid signatures;
- bounded HTTPS-only JWKS retrieval (1 MiB / 256 usable-key limits), atomic read-mostly cache, per-issuer refresh serialization, rotation refresh on `kid` miss, and a bounded key-miss refresh interval to prevent attacker-controlled unknown-`kid` fetch storms;
- verified request context containing only post-signature subject/tenant/client/role/scope/audience identity;
- operation-scoped identity policy for required roles/scopes, tenant and allowed client IDs;
- lifecycle starts in `DETECT`, requires explicit operator promotion to `ENFORCE`, and automatically returns to `DETECT` when issuer trust or policy semantics change;
- ENFORCE fails closed when a referenced issuer runtime/JWKS is unavailable; DETECT records evidence without blocking;
- bounded durable privacy-preserving violation evidence in `api-identity.json`; raw bearer tokens, unverified claims and JWKS key cache are never persisted;
- Admin API/UI for issuer configuration, JWKS refresh, operation policy, DETECT/ENFORCE transition, deletion and violation review;
- atomic immutable runtime policy snapshot; OpenAI is not in the identity verification or enforcement path.

Status: `IMPLEMENTED_TESTING_DEFERRED`. Local deterministic evidence: 16 API-5 test functions PASS, targeted race PASS, API-5 source gate 72 checks PASS, admin inline JavaScript syntax PASS, and whole-repository build/vet/test-binary compilation PASS under the external-dependency stub compile-shape harness. Canonical Go 1.25 real-dependency `go mod tidy -diff`, build, vet, full tests, race and real-Coraza remain `BLOCKED_ENVIRONMENT/NOT_RUN` on this host.

---

# API-6 — Sequence Analytics

## Goal

Understand API workflows.

Examples:

```
login
 |
account
 |
checkout
 |
payment
```

API-6.1 implemented foundation:

- bounded non-blocking session/transition observation;
- API-1 normalized operation nodes;
- API-5 verified-identity-only correlation and keyed private anonymous correlation;
- fixed session/transition/history caps and TTLs;
- immutable atomic runtime snapshots, restart persistence/autosave/final drain;
- Reviewer-gated/audited read-only admin visibility.

API-6.1 is visibility-only and never blocks. Source gate: 33 checks PASS. Five targeted Go tests are present; execution/race are `BLOCKED_ENVIRONMENT/NOT_RUN` because this host has no Go toolchain.

API-6.2 implemented learning slice:

- bounded workflow cohorts keyed from API-1 normalized operations and privacy-preserving anonymous/API-5 verified identity context;
- transition observation/session counts, first/last seen, frequency-derived confidence and LEARNING/MATURE/STALE semantics;
- cold-start thresholds, workflow depth, entry/terminal operations, idle+absolute session lifecycle and durable v2 state;
- read-only Reviewer/audit workflow visibility; no anomaly verdict or blocking path.

API-6.3 implemented detection slice:

- DETECT-only bounded evidence for unknown transitions, skipped prerequisites, unexpected entry points, sequence reversals, abnormal repetition and workflow divergence;
- mature-model/sample/session/age/confidence safeguards and bounded persisted exception primitives;
- transition probability = 0 alone is never sufficient for a workflow-divergence verdict;
- learned sequence evidence cannot BLOCK/DENY/403 or enter ENFORCE.

Remaining planned slice:

- API-6.4: operational modes, exception CRUD, reset/relearn and full console controls.

Status: API-6.1 through API-6.4 and API-7.1 are `IMPLEMENTED_TESTING_DEFERRED`; API-7.2 is next and remains `PLANNED`.

---

# API-7 — BOLA Analytics

## Goal

Detect suspicious object-level authorization patterns.

Scope:

- identity/object relationship analysis
- tenant boundary signals
- enumeration detection
- cross-subject access candidates

Initial result:

```
BOLA_CANDIDATE
```

not automatic blocking.

---

# API-8 — GraphQL Security

Status: `IMPLEMENTED_TESTING_DEFERRED`.

API-8 is implemented across the full documented GraphQL track:

- API-8.1 bounded parsing/normalization and structural operation discovery;
- API-8.2 bounded SDL import, normalized schema topology and endpoint-scoped schema-contract binding;
- API-8.3 deterministic depth/complexity/field/mutation/subscription controls;
- API-8.4 explicit introspection policy and bounded Apollo persisted-query support;
- API-8.5 verified API-5 identity plus keyed GraphQL-variable integration into API-7 relationship/BOLA evidence without granting API-7 blocking authority;
- API-8.6 Reviewer-gated `LEARN -> DETECT -> ENFORCE` deterministic policy, durable evidence, APIs, audit and Admin Console.

Only an explicit GraphQL policy in `ENFORCE` may reject a request. Any policy edit or bound schema-contract content change returns the policy to `LEARN`; referenced contracts cannot be deleted while in use. API-6/API-7 learned or inferred evidence cannot promote or mutate GraphQL enforcement, and OpenAI has no request-path authority.

Durable GraphQL state stores normalized structure, hashes and bounded evidence rather than raw query documents, literals, raw variables, Authorization/Cookie/JWT data or unverified identity claims.

Exact-source API-8 gate: **259 PASS**; **22 targeted API-8 Go test functions are present**. Canonical Go 1.25 tidy/build/targeted/race remains `BLOCKED_ENVIRONMENT/NOT_RUN` on the local Go 1.23.2 host.

---

# OpenAI Role

OpenAI is an analysis assistant, not an enforcement authority.

Allowed:

- schema explanation
- candidate review
- anomaly explanation
- operator assistance

Not allowed:

- autonomous policy mutation
- direct blocking authority
- bypass of deterministic validation

---

# Relationship With Dataplane

```
API Intelligence
        |
        v
Positive Security Decisions
        |
        v
Coraza / Runtime Enforcement

Coraza / CRS / VectorScan
        |
        v
High-speed detection path
```

## API-1/API-2/API-3 closure evidence — 2026-09-23

API-1, API-2 and API-3 source closure is complete for this checkpoint. Deterministic isolated/source-gate evidence is PASS, including a live API-1 → API-2 → API-3 learning/drift integration test. Repository-root Go 1.25 execution remains BLOCKED/NOT_RUN, so none of these slices are promoted to `TESTED` or `RELEASED`. See `API_SECURITY_CLOSURE_RESULT.md`.

## Slice Implementation Blueprint

See `API_SECURITY_SLICE_IMPLEMENTATION_ROADMAP.md` for detailed slice-by-slice implementation scope.

Current sequence:
API-1 → API-2 → API-3 → API-4/API-5 → API-6/API-7 → API-8


# API-6.4 — Sequence Operations + Hardening

Status: `IMPLEMENTED_TESTING_DEFERRED`. Explicit site-scoped LEARN/DETECT control, mature-only DETECT promotion, bounded recent-session evidence, exception CRUD, site-scoped reset/relearn, Reviewer RBAC/audit and the complete Sequence Operations console are implemented. Sequence analytics remains non-enforcing: no BLOCK/DENY/403 or ENFORCE authority. Durable sequence state is v4 with v1/v2/v3 restore compatibility. API-7.1 is now implemented with testing deferred; API-7.2 Identity/Object Relationship is the next `PLANNED` slice.


## API-7.1 Object Locator Discovery checkpoint — 2026-09-24

API-7.1 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. It adds bounded `ObjectLocator` discovery from API-1 normalized path parameters, API-2 typed path/query/body evidence, matched API-3 OpenAPI contracts, and explicit Reviewer-gated operator INCLUDE/SUPPRESS configuration. Locators are keyed by normalized API-1 operation ID plus location/field; raw object values are transient only and durable value evidence is bounded HMAC-SHA256 fingerprint data protected by a separate mode-0600 key. Client-supplied ownership/tenant headers are not authority, and GraphQL `variables.*` discovery is deferred to API-8.

API-7.1 has no identity/object relationship verdict, tenant-boundary verdict, `BOLA_CANDIDATE`, ownership verdict, BLOCK/DENY/403 path, or ENFORCE authority. OpenAI is absent from the locator authority path. State is bounded by global/per-operation/fingerprint/override caps, 30-day learned TTL, versioned restart validation, existing non-blocking observation-plane ingestion, immediate mutation persistence, RBAC and audit.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83 PASS** from API-1/2 through API-7.1; API-7.1 has **9 targeted test functions present**; OpenAI source contract **16/16 PASS** plus isolated tests PASS; package-source PASS; root Go shape **107 files PASS**; package-builder **9/9 PASS**; changed-file gofmt, Admin JavaScript and primary shell syntax PASS. Canonical Go 1.25 targeted/race remains `BLOCKED_ENVIRONMENT/NOT_RUN` because the host has Go 1.23.2 and external toolchain retrieval is unavailable. Next slice: **API-7.2 Identity/Object Relationship**. API-7.3/API-7.4 and API-8 remain `PLANNED`.

## API-7.1 — Object Locator Discovery

Status: `IMPLEMENTED_TESTING_DEFERRED`.

Implemented scope: normalized object-selector discovery from API-1 path parameters, API-2 typed query/body evidence, matched API-3 OpenAPI contract declarations, and bounded operator INCLUDE/SUPPRESS controls. Durable locator evidence never stores raw object values; only keyed HMAC fingerprints are retained. Client-supplied tenant/ownership headers are excluded. GraphQL variables are deferred to API-8. This slice produces no identity/object relationship, ownership verdict, BOLA verdict, or enforcement action.

Next: API-7.2 Identity/Object Relationship (`PLANNED`), then API-7.3 BOLA Detection and API-7.4 BOLA Policy/Evidence/Console.

## API-7.2 Identity/Object Relationship checkpoint — 2026-09-24

API-7.2 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. It correlates only API-5 cryptographically verified identity context with ACTIVE API-7.1 locator/object evidence. Identity, tenant and client dimensions are persisted only as API-7.2 HMAC-SHA256 pseudonyms protected by a dedicated mode-0600 key; object values remain API-7.1 keyed fingerprints. Raw JWTs, subjects, tenant/client claim values, cookies, caller-supplied owner/tenant headers and raw object values are not durable relationship state.

The relationship plane is bounded and asynchronous: non-blocking queue capacity 2,048; 8,192 total relationships; 512 relationships per verified identity pseudonym; 256 verified identities per keyed object/locator pair; 30-day TTL; bounded path/query/body capture; restart revalidation; autosave/final flush; shutdown drain; Reviewer-gated read-only evidence/status endpoints with audit. SUPPRESSed API-7.1 locators cannot create relationships. GraphQL `variables.*` remains deferred to API-8.

API-7.2 is evidence only. `OBSERVED`/`REPEATED` means recurrence, not ownership. There is no `BOLA_CANDIDATE`, ownership verdict, tenant-boundary verdict, BLOCK/DENY/403 or ENFORCE authority, and OpenAI is absent from this authority path.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83/100 PASS** from API-1/2 through API-7.2; API-7.2 has **9 targeted Go test functions present**; OpenAI source contract **16/16 PASS** plus isolated OpenAI/secretref tests PASS; package-source PASS; root Go source shape **109 files PASS**; package-builder **9/9 PASS**; changed Go files `gofmt` PASS. Canonical Go 1.25 targeted/race remains `BLOCKED_ENVIRONMENT/NOT_RUN` because this host has Go 1.23.2 and external toolchain retrieval is unavailable. Next slice: **API-7.3 BOLA Detection**; API-7.4 and API-8 remain `PLANNED`.


## API-7.3 — BOLA Detection

Status: `IMPLEMENTED_TESTING_DEFERRED`.

API-7.3 consumes only API-5 verified identity pseudonyms, ACTIVE API-7.1 keyed object locators/fingerprints and API-7.2 relationship history. It emits bounded evidence-only candidates for repeated-baseline identity/object divergence, verified-tenant/object divergence and high recent per-locator object fan-out. Novel objects alone do not create BOLA evidence.

Candidate state is capped (4,096 total; 256 per identity; 128 per keyed object/locator), expires after seven days, is versioned/revalidated on restart, and is persisted with the API security autosave/final-flush path. Detection runs inside the API-7.2 background processing plane before the current relationship is merged, so the current event cannot manufacture its own baseline.

Reviewer-or-higher can read `/api/security/bola/candidates` and `/api/security/bola/status`; both reads are audited. API-7.3 has no policy mutation route, ownership verdict, tenant-boundary verdict, `BLOCK`, `DENY`, request-path `403` or `ENFORCE` authority, and OpenAI is absent from the detector path.

Exact-source evidence: API-7.3 source gate 110 PASS; nine targeted Go test functions are present; prior API source gates remain PASS. Canonical Go 1.25 targeted/race execution remains `BLOCKED_ENVIRONMENT/NOT_RUN`. API-7.4 is now implemented with testing deferred; API-8 is next.

## API-7.4 — BOLA Policy/Evidence/Console checkpoint — 2026-09-24

Status: `IMPLEMENTED_TESTING_DEFERRED`. API-7.4 adds explicit evidence-handling policy and operator workflow over API-7.3 candidates without changing request authorization. `BOLAPolicy` is scoped to normalized operation, optional API-7.1 locator, optional candidate type and minimum confidence; actions are `REVIEW` or `SUPPRESS` only. Suppressed candidates remain durable detector evidence. Workflow states are `OPEN`, `ACKNOWLEDGED`, `DISMISSED`, and `RESOLVED` with enumerated reason codes; dismissed/resolved evidence reopens on newer detector evidence.

The policy/evidence store is capped at 1,024 policies and 4,096 reviews, TTL-pruned, versioned, restart-revalidated, strict-JSON/16-KiB mutation bounded, immediately persisted on mutation and included in API security autosave/final flush. Reviewer-only APIs and the embedded console provide evidence, effective policy, workflow and policy-status visibility. API-7.4 is absent from the API-7.2 processor and API-7.3 detector path; inferred BOLA remains non-enforcing and cannot cause `BLOCK`, `DENY`, request-path `403`, `ENFORCE`, ownership or tenant-boundary verdicts. OpenAI remains outside authority.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83/100/110/156 PASS**; 10 targeted API-7.4 Go tests are present; OpenAI source contract 16/16 plus isolated tests PASS; package-source PASS; package-builder 9/9 PASS; root Go shape 113 files PASS; gofmt/Admin JS/shell/Python syntax PASS. Canonical Go 1.25 targeted/race remains `BLOCKED_ENVIRONMENT/NOT_RUN`. Next slice: **API-8 — GraphQL Security**.

## API-8 GraphQL Security checkpoint — 2026-09-24

API-8 is implemented across parsing/normalization, SDL contract binding, deterministic complexity/field/mutation/subscription/introspection policy, Apollo persisted-query support, verified GraphQL-variable integration into API-7 evidence, and Reviewer-gated `LEARN` / `DETECT` / `ENFORCE` policy plus Admin Console. Any policy edit or bound contract content change returns the policy to `LEARN`; a referenced schema contract cannot be deleted. Only explicit deterministic GraphQL policy in `ENFORCE` may reject traffic. API-6/API-7 learned or inferred evidence and OpenAI cannot promote, mutate or bypass GraphQL enforcement.

Durable GraphQL state excludes raw query documents, literals, raw variable values, Authorization/Cookie/JWT data and unverified claims. SDL is reduced to a digest plus normalized topology; GraphQL object values use API-7 keyed fingerprints and API-5 verified identity only. State and parser resources are bounded, versioned, restart-revalidated, autosaved/final-flushed, and strict mutation APIs are Reviewer-gated and audited.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83/100/110/156/259 PASS** through API-8; **22 targeted API-8 Go test functions are present**; OpenAI source contract **16/16 PASS** plus isolated tests PASS; package-source PASS; package-builder **9/9 PASS**; root Go source shape **115 files PASS**; changed API-8 Go files `gofmt`, Admin embedded JavaScript, relevant shell and Python syntax PASS. Canonical Go 1.25 `go mod tidy -diff`, root build/vet/test, API-8 targeted tests and race tests remain `BLOCKED_ENVIRONMENT/NOT_RUN` because the host has Go 1.23.2 and external toolchain retrieval is unavailable. API-8 remains `IMPLEMENTED_TESTING_DEFERRED`, not `TESTED` or `RELEASED`.

The documented API-security implementation roadmap is now complete through API-8. No API-9 is defined. Next gate: canonical Go 1.25 qualification of the exact source/artifact, followed by release promotion only if all required gates pass.


## 2026-09-24 post-API-8 hardening note

No API-9 is defined. API-8 post-audit hardening closes runtime correctness and operator-surface gaps discovered after API-8 implementation and preserves all established authority boundaries. Release promotion remains a qualification task, not a new API-security feature slice.

## 2026-09-25 — Production Correctness & Control-Plane Hardening

Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.

## 2026-09-28 maintenance note

The code-duplication consolidation is a maintenance wave, not API-9 and not a
change to the API-security roadmap. API-1 through API-8 remain
`IMPLEMENTED_TESTING_DEFERRED`; no API-9 is defined.

<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
