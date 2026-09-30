# API Security API-1 → API-3 Closure Result


## Current canonical addendum — 2026-09-30 — TLS Session Resumption + Handshake Observability

This source now includes the **TLS Session Resumption + Handshake Observability**
slice on top of `waf-proxy-build-admin-startup-blocker-fix-2026-09-30.zip`
(parent SHA-256 `49450dfffeca2e45f17def136e122fac115c8fba4e8beb70a9307fe5eb6cb0e3`).
Project state remains **IMPLEMENTED_TESTING_DEFERRED**. This is an operational
TLS/performance slice, **not API-9**; API-1 through API-8 enforcement and state
authority boundaries are unchanged.

The built-in Go TLS path can now use a purpose-separated runtime-only shared
session-ticket secret to derive a deterministic rotating current+previous key
ring for restart-stable and HA cross-node resumption. The secret is supplied by
`WAF_TLS_SESSION_TICKET_SECRET` or a private
`WAF_TLS_SESSION_TICKET_SECRET_FILE`; it is not persisted in Config, returned by
Admin APIs, HA-synchronized, or logged, and must not reuse the HA peer-token
value. When unset, Go process-local automatic tickets remain the compatibility
fallback and cross-node/restart resumption is not claimed. External TLS frontend
termination remains outside this key manager.

Handshake observability now records authoritative successful full/resumed state
from `tls.ConnectionState.DidResume`, TLS 1.2/1.3, full-handshake certificate key
algorithm, handshake-processing average/p95, and policy rejects. `/api/metrics`,
`/api/status`, and the shipping Console expose the new data. See
`TLS_SESSION_RESUMPTION_HANDSHAKE_OBSERVABILITY.md`.

Current evidence: dedicated TLS source gate **33/33 PASS**, root Go source-shape
**99 files PASS**, existing API/static source gates remain PASS, duplication and
route gate remains **80/80 with 139 Admin/update patterns**, blocker-fix route
gate remains **13/13 with 139 patterns**, package-source PASS, and package-builder
**9/9 PASS**. A dependency-free isolated exact-core TLS probe on Go 1.23.2 passed all five
repository TLS tests, including TLS 1.2/TLS 1.3 cross-config resumption, and
passed `go test -race`; it is not canonical repository qualification. Exact-source Go 1.25 tidy/build/vet/test/race
and runtime package qualification remain **BLOCKED_ENVIRONMENT / NOT_RUN** here
because the required toolchain/modules cannot be retrieved on this host. No
`TESTED` or `RELEASED` promotion is made.

This section supersedes earlier sections labelled current/canonical when they
conflict; dated historical evidence below remains historical.

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

Date: 2026-09-23 (Asia/Taipei)

Status: `IMPLEMENTED_TESTING_DEFERRED`

This document records the historical API-1 through API-3 closure checkpoint. API-4 Positive Schema Enforcement and API-5 JWT + Identity-aware API Security were implemented later on the same 2026-09-23 development line; use `API_SECURITY_ROADMAP.md` and `NEXT_CHAT_HANDOVER.md` for current slice status. Repository-wide canonical Go 1.25 real-dependency qualification remains `BLOCKED_ENVIRONMENT/NOT_RUN` on this host.

## External CI correction — 2026-09-23

The previous closure artifact passed packaging/source-slice checks but subsequently failed external Go 1.25 CI. Packaging integrity was therefore not buildability evidence. The current working source repairs the confirmed package/test blockers (`ai.go` named-result redeclaration, duplicate AI test helper, and the additional duplicate `normalizeHost` symbol found during this follow-up), adds a dependency-free root source-shape gate, and replaces the Phase 4 Slice B fixed-window/global-mutex limiter with a bounded sharded continuous token bucket.

The current local whole-repository dependency-stub harness passes `go build ./...`, `go vet ./...`, and all test-binary compilation; targeted L7/client-identity/AI/API tests pass. This remains compile-shape evidence only. Exact Go 1.25 real-dependency `go mod tidy -diff`, build, full tests, race and real-Coraza remain mandatory and are not locally claimed. The prior external report of committed module drift remains unresolved until the exact tidy result is run against these bytes.

## API-1 — Discovery + Operation Normalization

Implemented:

- durable site-scoped API operation IDs and filesystem persistence;
- host, content-type, authentication-family and status distributions;
- bounded raw-path examples and typed path-parameter metadata;
- integer, UUID, ULID, long-hex, date and opaque-token normalization;
- operation detail, ignore/unignore and manual reclassification APIs;
- embedded admin-console API operation inventory and controls;
- startup restore, periodic autosave and clean-shutdown flush;
- same-directory unique-temp + fsync + atomic rename persistence, with concurrent-writer coverage.

State file: `api-operations.json`, adjacent to the configured `config.json`.

## API-2 — Typed Schema Learning

Implemented:

- live traffic pipeline from the bounded observation plane into schema learning;
- body learning independent of Passive Discovery and AI body capture settings;
- path/query/header/JSON/form/multipart metadata collection;
- nested objects, arrays, type/format evidence, sample counts, presence rate,
  required candidates, bounded distinct cardinality and conservative enum
  candidates;
- aggregate field growth is bounded across requests, arrays retain container evidence, and dominant-type changes invalidate stale human review;
- authentication telemetry preserves composite mechanism evidence (Bearer/API key/mTLS) without retaining credentials;
- privacy-first value handling: authorization/cookies/secrets are excluded;
  PII-like/format values are not retained or fingerprinted; arbitrary free-form
  strings and header values are not retained/fingerprinted; credential header
  names are punctuation-normalized so variants such as `X-API-Key` are excluded
  before entering the observation queue; enum learning is limited to enum-like
  field names;
- lifecycle `LEARNING → CANDIDATE → REVIEWED`, with operator APIs forbidden
  from promoting `LEARNING` directly to `REVIEWED`, and review invalidation only
  when semantic schema evidence changes rather than on every new request;
- persistent aggregates/candidates with restart continuation;
- body-sample shedding under observation backlog while preserving API-1 request
  metadata and counting `schema_body_dropped`.

State file: `api-schema.json`, adjacent to the configured `config.json`.

## OpenAI SchemaCandidate review

Implemented as control-plane advice only:

- reuses the existing Responses API + Structured Outputs integration;
- strict result shape: `approve|revise|insufficient_evidence`, confidence, reason;
- consumes deterministic aggregate candidate evidence;
- never changes Coraza mode, block lists, PagePolicy, FieldPolicy, or schema
  enforcement state;
- AI review preserves lifecycle status; only an authenticated reviewer can mark
  the candidate `REVIEWED`.

## API-3 — OpenAPI Contract Management

Implemented/extended:

- durable contract/version/binding/diff state;
- OpenAPI 3.0.x / 3.1.x YAML/JSON import with local-only `$ref` resolution and explicit external-`$ref` rejection;
- OpenAPI export as JSON or YAML;
- request/response schema fields including enums and multi-content-type inventory;
- path/query/header/cookie parameters;
- security scheme definitions plus global/operation security requirements, including OpenAPI OR/AND and anonymous alternatives;
- deterministic operation matching against current live API-1 inventory;
- request/response/parameter/content-type/security contract version diffs;
- live declared-vs-observed drift taxonomy:
  `MISSING_ENDPOINT`, `UNDECLARED_ENDPOINT`, `UNKNOWN_FIELD`,
  `MISSING_REQUIRED_FIELD`, `TYPE_DRIFT`, `ENUM_DRIFT`,
  `CONTENT_TYPE_DRIFT`, and `AUTH_DRIFT`, with undeclared-endpoint checks scoped to matched site/host traffic;
- end-to-end deterministic evidence that live API-1 observations feed API-2
  learning and then API-3 contract drift.

State file: `api-contracts.json`, adjacent to the configured `config.json`.

## Admin API additions

- `GET /api/security/operations`
- `GET /api/security/operations/{id}`
- `POST /api/security/operations/{id}/ignore`
- `POST /api/security/operations/reclassify`
- `GET /api/security/schema/candidates`
- `GET /api/security/schema/{id}`
- `POST /api/security/schema/{id}/review`
- `POST /api/security/schema/{id}/ai-review`
- `GET /api/security/contracts/{id}/export`

Existing API-3 inventory/import/match/compare/drift routes remain in place.

## Verification executed on this host

PASS:

- `gofmt` cleanliness for all modified Go sources/tests;
- `tools/tests/test-api12-source.py`: 69 checks;
- `tools/tests/test-api3-source.py`: 47 checks;
- API-1 exact-source isolated deterministic tests: 5/5, including 32-writer atomic persistence;
- API-2 exact-source isolated deterministic tests: 8/8;
- actual API-3 exact-source compile-only gate with a minimal YAML dependency shim: PASS (runtime YAML behavior is not claimed by this evidence);
- four deterministic API-3 runtime/integration tests that do not depend on YAML behavior: PASS, including scoped undeclared-endpoint drift, OpenAPI auth semantics, multi-content-type matching, and API-1 → API-2 → API-3 live learning/drift;
- OpenAI source contract: 16/16 plus isolated `openaiapi` and `secretref` tests;
- OpenAI SchemaCandidate advisory review isolated test: 1/1;
- admin inline JavaScript syntax: PASS;
- WAF package-tool source gate: PASS;
- package-builder Python unit tests: 9/9.

## Required qualification still blocked/not run

The local Go toolchain is Go 1.23.2 while `go.mod` requires Go 1.25.0. Exact command/exit evidence is recorded in `SOURCE_BASELINE_GATE_RESULT.md`. With
`GOTOOLCHAIN=local`, root Go commands fail before compilation. Automatic Go
1.25 retrieval also fails because this environment cannot resolve/reach
`proxy.golang.org`.

Therefore these remain `BLOCKED`/`NOT_RUN`, not PASS:

```text
GOTOOLCHAIN=local go mod tidy -diff
GOTOOLCHAIN=local CGO_ENABLED=0 go build ./...
GOTOOLCHAIN=local CGO_ENABLED=0 go vet ./...
GOTOOLCHAIN=local CGO_ENABLED=0 go test ./...
GOTOOLCHAIN=local CGO_ENABLED=1 go test -race ./...
real Coraza v3.7.0 qualification
real VectorScan qualification where required
```

## Next step

Run the exact delivered source bytes on the pinned Go 1.25 release/CI host. Fix
any root-build/test findings there, rerun artifact packaging integrity, and only
API-4 Positive Schema Enforcement has since been implemented with testing deferred. API-5 through API-8 remain
`PLANNED` at this checkpoint.


Local executable evidence for this checkpoint is captured in `TESTING_RESULTS.md`; root Go 1.25 blocker evidence is captured in `SOURCE_BASELINE_GATE_RESULT.md`.

## Current post-closure status — API-5

API-4 and API-5 are now `IMPLEMENTED_TESTING_DEFERRED`; API-6 through API-8 remain `PLANNED`. API-5 adds trusted JWT/JWKS verification and verified-claims-only identity policy with DETECT/ENFORCE lifecycle. Local API-5 evidence is 17 deterministic test functions PASS, targeted race PASS, and 72-check source gate PASS. Canonical Go 1.25 real-dependency qualification remains blocked and no TESTED/RELEASED claim is made.

## Superseding API-6 status note — 2026-09-24

The API-1→API-5 closure evidence above remains historical. Current source has since advanced through the complete documented API-security roadmap: API-6.1 through API-6.4, API-7.1 through API-7.4, and API-8 GraphQL Security are `IMPLEMENTED_TESTING_DEFERRED`. See `API_SECURITY_ROADMAP.md`, `API8_GRAPHQL_SECURITY.md`, and `API8_SOURCE_GATE_RESULT.md` for current truth.


## API-7.1 Object Locator Discovery checkpoint — 2026-09-24

API-7.1 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. It adds bounded `ObjectLocator` discovery from API-1 normalized path parameters, API-2 typed path/query/body evidence, matched API-3 OpenAPI contracts, and explicit Reviewer-gated operator INCLUDE/SUPPRESS configuration. Locators are keyed by normalized API-1 operation ID plus location/field; raw object values are transient only and durable value evidence is bounded HMAC-SHA256 fingerprint data protected by a separate mode-0600 key. Client-supplied ownership/tenant headers are not authority, and GraphQL `variables.*` discovery is deferred to API-8.

API-7.1 has no identity/object relationship verdict, tenant-boundary verdict, `BOLA_CANDIDATE`, ownership verdict, BLOCK/DENY/403 path, or ENFORCE authority. OpenAI is absent from the locator authority path. State is bounded by global/per-operation/fingerprint/override caps, 30-day learned TTL, versioned restart validation, existing non-blocking observation-plane ingestion, immediate mutation persistence, RBAC and audit.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83 PASS** from API-1/2 through API-7.1; API-7.1 has **9 targeted test functions present**; OpenAI source contract **16/16 PASS** plus isolated tests PASS; package-source PASS; root Go shape **107 files PASS**; package-builder **9/9 PASS**; changed-file gofmt, Admin JavaScript and primary shell syntax PASS. Canonical Go 1.25 targeted/race remains `BLOCKED_ENVIRONMENT/NOT_RUN` because the host has Go 1.23.2 and external toolchain retrieval is unavailable. Next slice: **API-7.2 Identity/Object Relationship**. API-7.3/API-7.4 is `IMPLEMENTED_TESTING_DEFERRED`; API-8 remains `PLANNED`.

## API-7.2 Identity/Object Relationship checkpoint — 2026-09-24

API-7.2 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. It correlates only API-5 cryptographically verified identity context with ACTIVE API-7.1 locator/object evidence. Identity, tenant and client dimensions are persisted only as API-7.2 HMAC-SHA256 pseudonyms protected by a dedicated mode-0600 key; object values remain API-7.1 keyed fingerprints. Raw JWTs, subjects, tenant/client claim values, cookies, caller-supplied owner/tenant headers and raw object values are not durable relationship state.

The relationship plane is bounded and asynchronous: non-blocking queue capacity 2,048; 8,192 total relationships; 512 relationships per verified identity pseudonym; 256 verified identities per keyed object/locator pair; 30-day TTL; bounded path/query/body capture; restart revalidation; autosave/final flush; shutdown drain; Reviewer-gated read-only evidence/status endpoints with audit. SUPPRESSed API-7.1 locators cannot create relationships. GraphQL `variables.*` remains deferred to API-8.

API-7.2 is evidence only. `OBSERVED`/`REPEATED` means recurrence, not ownership. There is no `BOLA_CANDIDATE`, ownership verdict, tenant-boundary verdict, BLOCK/DENY/403 or ENFORCE authority, and OpenAI is absent from this authority path.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83/100 PASS** from API-1/2 through API-7.2; API-7.2 has **9 targeted Go test functions present**; OpenAI source contract **16/16 PASS** plus isolated OpenAI/secretref tests PASS; package-source PASS; root Go source shape **109 files PASS**; package-builder **9/9 PASS**; changed Go files `gofmt` PASS. Canonical Go 1.25 targeted/race remains `BLOCKED_ENVIRONMENT/NOT_RUN` because this host has Go 1.23.2 and external toolchain retrieval is unavailable. Next slice: **API-7.3 BOLA Detection**; API-7.4 is `IMPLEMENTED_TESTING_DEFERRED`; API-8 remains `PLANNED`.



## API-7.3 BOLA Detection checkpoint — 2026-09-24

API-7.3 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. The detector consumes only API-5 cryptographically verified identity pseudonyms, ACTIVE/non-suppressed API-7.1 keyed object locators/fingerprints, and API-7.2 relationship history. It emits bounded evidence-only candidates for `IDENTITY_OBJECT_DIVERGENCE`, `TENANT_OBJECT_DIVERGENCE`, and `OBJECT_ENUMERATION`. A novel object alone is not a candidate; cross-identity and cross-tenant evidence requires repeated historical baseline, while enumeration requires 20 recent distinct keyed objects under the same locator within 10 minutes.

Candidate state persists only pseudonymous/keyed evidence, is capped at 4,096 total / 256 per identity / 128 per object-locator, expires after seven days, is versioned/revalidated on restart, and is saved by the API security autosave/final-flush path. Detection runs inside the API-7.2 background processing plane before current relationship merge. Reviewer-only candidate/status reads are audited. There is no ownership verdict, tenant-boundary verdict, policy mutation API, `BLOCK`, `DENY`, request-path `403`, or `ENFORCE` authority; OpenAI is absent from the detector authority path.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83/100/110 PASS** from API-1/2 through API-7.3; API-7.3 has **9 targeted Go test functions present**; OpenAI source contract **16/16 PASS** plus isolated tests PASS; package-source PASS; root Go source shape **111 files PASS**; package-builder **9/9 PASS**; changed Go `gofmt`, Admin embedded JavaScript and primary shell/Python syntax PASS. Canonical Go 1.25 `go mod tidy -diff`, root build, API-7.3 targeted tests and API-7.3 race tests remain `BLOCKED_ENVIRONMENT/NOT_RUN` because this host has Go 1.23.2 and external toolchain retrieval is unavailable. Next slice: **API-7.4 BOLA Policy/Evidence/Console**; API-8 remains `PLANNED`.

## API-7.4 BOLA Policy/Evidence/Console checkpoint — 2026-09-24

API-7.4 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. It completes the API-7 BOLA implementation track with a bounded evidence-handling policy plane and operator workflow. Policies are scoped only by normalized API-1 operation ID, optional API-7.1 locator ID, optional API-7.3 candidate type, and minimum confidence. The only actions are `REVIEW` and `SUPPRESS`; suppression never deletes the underlying API-7.3 candidate. Operator evidence workflow is `OPEN` / `ACKNOWLEDGED` / `DISMISSED` / `RESOLVED` with enumerated reason codes only. New detector evidence after dismissal/resolution reopens the evidence automatically.

API-7.4 is deliberately absent from the API-7.2 relationship processor and API-7.3 detector authority path. It accepts no raw identity/object/tenant/client selectors, arbitrary headers, cookies, Authorization data, or free-text review notes. There is no ownership verdict, request-path `BLOCK`, `DENY`, `403`, or `ENFORCE` authority, and OpenAI is absent from the API-7.4 authority path. State is versioned/revalidated on restart, mutex-protected, TTL/cardinality bounded (1,024 policies; 4,096 reviews; 30-day default policy/review TTL; 180-day policy maximum), immediately persisted on mutation, and included in API security autosave/final flush. Reviewer-only policy/evidence APIs are audited and the embedded console exposes effective policy, evidence workflow, suppression, and reopen status.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83/100/110/156 PASS** from API-1/2 through API-7.4; **10 API-7.4 targeted Go test functions are present**; OpenAI source contract **16/16 PASS** plus isolated package tests PASS; package-source PASS; root Go source shape **113 files PASS**; package-builder **9/9 PASS**; `gofmt`, Admin embedded JavaScript and primary shell/Python syntax PASS. Canonical Go 1.25 `go mod tidy -diff`, root build/test, API-7.4 targeted tests and API-7.4 race tests remain `BLOCKED_ENVIRONMENT/NOT_RUN` because this host has Go 1.23.2 and external toolchain retrieval is unavailable. API-8 GraphQL Security is the next `PLANNED` slice.

## API-8 GraphQL Security checkpoint — 2026-09-24

API-8 is implemented across parsing/normalization, SDL contract binding, deterministic complexity/field/mutation/subscription/introspection policy, Apollo persisted-query support, verified GraphQL-variable integration into API-7 evidence, and Reviewer-gated `LEARN` / `DETECT` / `ENFORCE` policy plus Admin Console. Any policy edit or bound contract content change returns the policy to `LEARN`; a referenced schema contract cannot be deleted. Only explicit deterministic GraphQL policy in `ENFORCE` may reject traffic. API-6/API-7 learned or inferred evidence and OpenAI cannot promote, mutate or bypass GraphQL enforcement.

Durable GraphQL state excludes raw query documents, literals, raw variable values, Authorization/Cookie/JWT data and unverified claims. SDL is reduced to a digest plus normalized topology; GraphQL object values use API-7 keyed fingerprints and API-5 verified identity only. State and parser resources are bounded, versioned, restart-revalidated, autosaved/final-flushed, and strict mutation APIs are Reviewer-gated and audited.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83/100/110/156/259 PASS** through API-8; **22 targeted API-8 Go test functions are present**; OpenAI source contract **16/16 PASS** plus isolated tests PASS; package-source PASS; package-builder **9/9 PASS**; root Go source shape **115 files PASS**; changed API-8 Go files `gofmt`, Admin embedded JavaScript, relevant shell and Python syntax PASS. Canonical Go 1.25 `go mod tidy -diff`, root build/vet/test, API-8 targeted tests and race tests remain `BLOCKED_ENVIRONMENT/NOT_RUN` because the host has Go 1.23.2 and external toolchain retrieval is unavailable. API-8 remains `IMPLEMENTED_TESTING_DEFERRED`, not `TESTED` or `RELEASED`.

The documented API-security implementation roadmap is now complete through API-8. No API-9 is defined. Next gate: canonical Go 1.25 qualification of the exact source/artifact, followed by release promotion only if all required gates pass.


## Post-closure correction — API-8 Post-Audit Hardening (2026-09-24)

The API-1 through API-8 authority roadmap remains implementation-complete, but a repository-wide audit found and corrected two runtime truth gaps (CIDR Enabled/expiry and an unused TLS-handshake abuse knob) plus operator Console exposure gaps. This hardening does not create API-9 or change API-6/API-7/API-8 authority. It also removes an obsolete non-shipping frontend and model-only Security Operations placeholders that had no runtime/API/Console implementation. Overall release status remains IMPLEMENTED_TESTING_DEFERRED pending canonical Go 1.25 and live qualification.

## 2026-09-25 — Production Correctness & Control-Plane Hardening

Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.

## 2026-09-28 — Duplication review closure addendum

The repository-wide duplication consolidation does not change API-security
policy authority. API-1 through API-8 remain `IMPLEMENTED_TESTING_DEFERRED`;
API-6 and API-7 learned/inferred evidence remains non-blocking; API-8 request
blocking remains limited to explicit deterministic ENFORCE policy; OpenAI has no
request-path authority. The cleanup removes only superseded/unwired source and
centralizes shared SecLang parsing. Source gate: **80/80 PASS**.

## 2026-09-30 — Build and Admin Startup Blocker Fix

The current source fixes two release-blocking defects found during external Go 1.25 review of the 2026-09-28 consolidation baseline. `admin.go` now imports the standard-library `errors` package required by the existing `errors.New(...)` validation paths. Positive-Schema exception state changes now use the unambiguous route `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`; the shipping Console uses the same path. The prior `POST /api/security/schema/enforcement/exceptions/{exception_id}` registration is removed because it conflicts with `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+ `http.ServeMux` specificity rules and can panic during Admin handler construction.

`admin_route_registration_test.go` now makes full Admin handler registration a regression test, and `tools/tests/test-build-startup-blocker-fix-source.py` is wired into `build.sh` and CI. The source gate also registers the complete 139-route Admin/update inventory with the Go standard library in a standalone temporary program; this catches wildcard-pattern ambiguity even when the repository module graph cannot be compiled locally. Current source evidence: blocker-fix gate **13/13 PASS**, route inventory **139/139 accepted by `ServeMux`**, API/source gates through API-8/post-audit/production-hardening/duplication remain PASS, OpenAI source contract and isolated tests PASS, package-source PASS, package-builder **9/9 PASS**, root Go source shape **96 files PASS**, Admin embedded JavaScript PASS, and changed Go files are `gofmt` clean.

This does **not** promote the source to `TESTED` or `RELEASED`. On the current execution host, canonical `GOTOOLCHAIN=local` Go 1.25 `go mod tidy -diff`, root build, vet, targeted Admin/API64 tests, and race remain `BLOCKED_ENVIRONMENT / NOT_RUN` because only Go 1.23.2 is installed and external Go toolchain/module retrieval is unavailable. Packaging/source-manifest integrity is not a substitute for those root compilation gates. No API-security authority changes are introduced: API-6/API-7 remain non-enforcing, API-8 retains explicit deterministic GraphQL ENFORCE authority, and OpenAI remains advisory only.

API contract note: this maintenance fix changes only the versionless Admin Console Positive-Schema exception toggle endpoint from `/api/security/schema/enforcement/exceptions/{exception_id}` to `/api/security/schema/enforcement/exceptions/{exception_id}/toggle`. Positive-Schema authority, RBAC, persistence, audit semantics, API-4 enforcement, and API-8 GraphQL authority are unchanged. No API-9 is defined.
## 2026-09-30 — OWI-1.0 R3 Dashboard Connector

Status remains **IMPLEMENTED_TESTING_DEFERRED**. This is a product integration slice, **not API-9**, and it does not change API-1 through API-8 enforcement authority. The WAF now exposes a separate read-only management reader at `/api/integrations/dashboard/v1` for required ASSET, DETECTION, POLICY and HEALTH lanes using a dedicated digest-only opaque bearer identity. Optional EVENT/ACTION_STATUS remain disabled and `actions=[]`.

R3 controls implemented in source include per-tenant+principal rate/concurrency plus a global ceiling, strict GET-only handling including HEAD rejection, an 8s default request deadline propagated into source sync and context-aware store-lock acquisition, hard snapshot/export/response bounds, durable cursor/snapshot/revision state, non-blocking WAF security export with explicit GAP semantics, fail-closed durability after write errors, opaque-token expiry/rotation/revoke, owner-only non-symlink token/cursor-secret files, TLS >=1.2 on the built-in reader listener, tenant/scope-bound cursors, and allow-listed sanitization. No Dashboard registry/validator/UI code is modified.

Supporting exact connector-source evidence on the available Go 1.23.2 host is **25 PASS / 2 intentional SKIP**, with the same set passing under `-race`; real subprocess SIGKILL→restart/GAP, store contention deadline/retry, injected write failure/recovery, client cancellation, oversize failure, scope reduction, and accelerated retention expiry are covered. The R3 source gate is **38/38 PASS**. The regenerated product return validates **61 files, 18 positive responses, 15 negative cases and 11 transport cases**. Integrated repository Go 1.25 build/vet/test/race remains `BLOCKED_ENVIRONMENT / NOT_RUN`; G2 Dashboard offline acceptance and G3 live TLS/ACL/intended-runtime qualification remain `NOT_RUN`.

<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
