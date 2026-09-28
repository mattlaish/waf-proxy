# New-Chat Handover Status

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

## Build-integrity and L7 correction — 2026-09-23

An external Go 1.25 CI audit of the previous API-1→API-3 delivery showed that archive integrity and isolated slice tests had not proved package buildability. The current working source repairs the package-level defects found in that audit and an additional duplicate symbol found during follow-up:

- removed the named-result `err` redeclaration in `ai.go`;
- removed the duplicate `testAIEngineWithoutWorkers` test helper;
- consolidated the duplicate root-package `normalizeHost` helper;
- retained the correct two-argument `writeJSON` / three-argument `writeJSONCode` split and added a dependency-free root source-shape gate to reject wrong-arity `writeJSON`, duplicate top-level declarations, and named-result `var` redeclarations;
- corrected the client-identity proxy test so the normal XFF path and the fail-closed conflicting-XFF/X-Real-IP path are tested separately.

Phase 4 Slice B L7 abuse control is no longer a fixed-window/global-lock implementation. It now uses a continuous token bucket keyed by site + trusted client identity, 64 state shards, per-entry locking, bounded state (1024 entries/shard), amortized idle pruning/idle eviction, and deferred active-request release so downstream panic cannot leak concurrency state. The existing configuration remains global; per-site/per-page limit overrides and TLS-handshake-rate enforcement are still not implemented and must not be claimed.

Current local evidence is deliberately split by scope: the dependency-free root source-shape gate passes; a temporary local external-dependency stub harness makes the whole repository pass `go build ./...` and `go test -run '^$' ./...` type/test-binary compilation; targeted client-identity/L7/AI/API tests pass there; L7 isolated unit + race tests pass; and the local 5-worker limiter microbenchmark is 162.5–179.7 ns/op versus 330.6–337.3 ns/op for the previous global-lock implementation. The benchmark is a control-path microbenchmark, not production proxy throughput evidence.

**Source Buildability remains BLOCKED, not PASS.** The canonical Go 1.25 real-dependency `go mod tidy -diff`, build, vet, full tests, race, and real-Coraza gates have not run on these exact bytes in this environment. The prior external CI also reported committed module metadata drift; no guessed `go.mod` edit is accepted as a substitute for the exact Go 1.25 tidy result.

## API Security checkpoint — 2026-09-23

The uploaded API-3 baseline was audited against source rather than roadmap labels, then repaired/closed through API-3. Current source truth:

| Slice | State | Current evidence |
|---|---|---|
| API-1 Discovery + Operation Normalization | `IMPLEMENTED_TESTING_DEFERRED` | durable operation inventory, metadata, controls/UI; 5/5 exact-source isolated tests PASS |
| API-2 Typed Schema Learning | `IMPLEMENTED_TESTING_DEFERRED` | live typed collector, lifecycle, privacy controls, persistence; 8/8 exact-source isolated tests PASS |
| OpenAI SchemaCandidate review | `IMPLEMENTED_TESTING_DEFERRED` | Responses API Structured Outputs, advisory-only; OpenAI source/isolated gates PASS |
| API-3 OpenAPI Contract Management | `IMPLEMENTED_TESTING_DEFERRED` | persistence, import/export, params/enums/security, scoped drift; 47-check source gate + deterministic isolated/integration tests PASS |
| API-4 Positive Schema Enforcement | `IMPLEMENTED_TESTING_DEFERRED` | immutable reviewed profiles; LEARN/DETECT/ENFORCE, exceptions, rollback, bounded durable violation evidence; API-4 7/7 + race + 46-check source gate PASS |
| API-5 JWT + Identity-aware Policy | `IMPLEMENTED_TESTING_DEFERRED` | verified JWT/JWKS identity context, DETECT/ENFORCE operation policy, bounded durable evidence; API-5 17-test + race + 72-check source gate PASS |
| API-6.1 Sequence Foundation | `IMPLEMENTED_TESTING_DEFERRED` | bounded async sessions/transitions, private correlation, TTL/caps, atomic snapshots and persistence; source gate PASS; Go tests/race blocked by missing toolchain |
| API-6.2 Workflow Learning | `IMPLEMENTED_TESTING_DEFERRED` | LEARN-only workflow model; source gate 58 PASS; Go 1.25 targeted/race BLOCKED_ENVIRONMENT/NOT_RUN |
| API-6.3 Sequence Anomaly Detection | `IMPLEMENTED_TESTING_DEFERRED` | DETECT-only bounded anomaly evidence; source gate 45 PASS; Go 1.25 targeted/race `BLOCKED_ENVIRONMENT/NOT_RUN` |
| API-6.4 Sequence Operations + Hardening | `IMPLEMENTED_TESTING_DEFERRED` | explicit bounded LEARN/DETECT controls, exception CRUD, reset/relearn, recent sessions and full operations console; source gate 58 PASS; Go 1.25 targeted/race `BLOCKED_ENVIRONMENT/NOT_RUN` |
| API-7.1 Object Locator Discovery | `IMPLEMENTED_TESTING_DEFERRED` | normalized locator discovery + bounded keyed evidence; source gate 83 PASS; Go 1.25 targeted/race BLOCKED_ENVIRONMENT/NOT_RUN |
| API-7.2 Identity/Object Relationship | `IMPLEMENTED_TESTING_DEFERRED` | verified identity pseudonym ↔ keyed object relationship evidence; source gate 100 PASS; Go 1.25 targeted/race `BLOCKED_ENVIRONMENT/NOT_RUN` |
| API-7.3 BOLA Detection | `IMPLEMENTED_TESTING_DEFERRED` | bounded DETECT-only identity/object divergence, tenant divergence and enumeration candidate evidence; source gate 110 PASS; Go 1.25 targeted/race `BLOCKED_ENVIRONMENT/NOT_RUN` |
| API-7.4 BOLA Policy/Evidence/Console | `IMPLEMENTED_TESTING_DEFERRED` | bounded REVIEW/SUPPRESS evidence policy + ACK/DISMISS/RESOLVE/REOPEN workflow + console; source gate 156 PASS; Go 1.25 targeted/race `BLOCKED_ENVIRONMENT/NOT_RUN` |
| API-8 GraphQL Security | `IMPLEMENTED_TESTING_DEFERRED` | bounded GraphQL parsing/normalization, SDL contract binding, depth/complexity/field/mutation/subscription/introspection/APQ controls, verified-variable API-7 bridge, staged deterministic LEARN/DETECT/ENFORCE; source gate 259 PASS; Go 1.25 targeted/race `BLOCKED_ENVIRONMENT/NOT_RUN` |

API-1 → API-2 → API-3 deterministic end-to-end learning/drift evidence is PASS. The repository-wide Source Buildability Gate remains **BLOCKED**, not PASS: this host has Go 1.23.2 while `go.mod` requires Go 1.25.0, and network/toolchain acquisition is unavailable. Required Go 1.25 `tidy/build/vet/test/race/real-Coraza` qualification remains `NOT_RUN/BLOCKED`. See `API_SECURITY_CLOSURE_RESULT.md`.

This working baseline came from the user-supplied source archive and contains no `.git` metadata; therefore no new branch/commit/push claim is made for this checkpoint.


> **Canonical checkpoint: 2026-09-23 — API-1 through API-3 source closure.**

## Current source state

- API-1 — API Discovery + Operation Normalization: `IMPLEMENTED_TESTING_DEFERRED`.
- API-2 — Typed Schema Learning: `IMPLEMENTED_TESTING_DEFERRED`.
- API-3 — OpenAPI Contract Management: `IMPLEMENTED_TESTING_DEFERRED`.
- API-4 — Positive Schema Enforcement: `IMPLEMENTED_TESTING_DEFERRED`.
- API-5 — JWT + Identity-aware API Security: `IMPLEMENTED_TESTING_DEFERRED`.
- API-6.1 — Sequence Foundation: `IMPLEMENTED_TESTING_DEFERRED`.
- API-6.2 through API-6.4, API-7.1 through API-7.4, and API-8 GraphQL Security: `IMPLEMENTED_TESTING_DEFERRED`.

API-1 is durable and operator-manageable; API-2 now learns typed candidates from live bounded observations and persists them; OpenAI SchemaCandidate review is advisory-only; API-3 persists declared contracts, imports/exports OpenAPI, models parameters/enums/security requirements, and computes expanded live drift from API-1/API-2 evidence.

## Verification truth

- `tools/tests/test-api12-source.py`: **PASS, 69 checks**.
- `tools/tests/test-api3-source.py`: **PASS, 47 checks**.
- API-1 exact-source isolated tests: **5/5 PASS**, including 32-writer atomic persistence.
- API-2 exact-source isolated tests: **8/8 PASS** (collector, lifecycle, persistence, aggregate bound, semantic invalidation, composite auth).
- API-3 exact-source compile-only gate PASS under a minimal YAML shim; 4 deterministic runtime/integration tests that do not depend on YAML behavior PASS.
- API-1 → API-2 → API-3 live-learning/drift integration: **PASS**.
- OpenAI source contract: **16/16 PASS** plus isolated provider/secret-reference tests.
- modified Go files: `gofmt` clean; admin inline JavaScript syntax PASS.
- Go 1.25 tidy/build/vet/full-test/race/real-Coraza: **BLOCKED/NOT_RUN** on this host.
- Do not claim `TESTED`, `RELEASED`, or repository-root buildability.

## Security boundary

- API-1/API-2/API-3 are intelligence/control-plane features only; they never block traffic or activate policy.
- Schema learning stores bounded aggregate metadata, not Authorization/Cookie/JWT secret values or complete request bodies.
- OpenAPI import is direct authenticated upload only; no remote `$ref` retrieval.
- Coraza remains authoritative enforcement.

## Exact next step

Run the exact API-1 through API-8 source on a pinned Go 1.25 release/CI host: `go mod tidy -diff`, root build/vet/tests, API-8 targeted tests, API-8 race tests, real-Coraza and applicable package qualification. Do not invent a new API slice or claim TESTED/RELEASED before those gates pass.

## API-5 JWT + Identity-aware API Security checkpoint — 2026-09-23

API-5 is implemented in source and remains `IMPLEMENTED_TESTING_DEFERRED`. `identity_api5.go` provides trusted issuer/JWKS verification, verified-claims-only request context, operation-scoped role/scope/tenant/client authorization, DETECT→ENFORCE lifecycle, semantic-change rollback to DETECT, bounded privacy-preserving evidence, and durable `api-identity.json` state. JWKS fetch is HTTPS-only and bounded; unknown-`kid` rotation refresh is serialized and rate-bounded. Raw bearer tokens, unverified claims and JWKS key cache are not persisted. OpenAI has no identity-enforcement authority.

Local evidence: 17 API-5 deterministic test functions PASS; targeted race PASS; `tools/tests/test-api5-source.py` 72 checks PASS; root source-shape PASS for 97 Go files; admin inline JavaScript syntax PASS; and the external-dependency stub harness passes whole-repository `go build ./...`, `go vet ./...`, and all test-binary compilation. These are not substitutes for the canonical Go 1.25 real-dependency gate, which remains `BLOCKED_ENVIRONMENT/NOT_RUN` (`go mod tidy -diff`, build, vet, full tests, full race, real-Coraza). API-6 remains `PLANNED` and has not started.


## API-6.2 Workflow Learning checkpoint — 2026-09-24

API-6.2 is implemented in source and remains `IMPLEMENTED_TESTING_DEFERRED`. The slice adds LEARN-only workflow cohorts, transition observation/session counts, frequency-derived confidence, `LEARNING`/`MATURE`/`STALE` cold-start semantics, workflow depth plus entry/terminal operation summaries, idle and absolute session lifetime, bounded workflow/session/transition state, API-5 verified-identity-only cohort input, durable v2 state with API-6.1 v1 migration, and Reviewer-gated/audited read-only workflow visibility. It adds no anomaly verdict, BLOCK/DENY/403 behavior, BOLA verdict, reset/relearn control, or sequence ENFORCE mode.

Executed source evidence: API-6.1 gate **33/33 PASS**, API-6.2 gate **56/56 PASS**, and Admin inline JavaScript syntax **PASS**. Seven targeted API-6.2 Go tests are present, including concurrency/race-relevant, restart/privacy and bounded-resource/adversarial cases. Canonical Go 1.25 targeted/race execution is `BLOCKED_ENVIRONMENT/NOT_RUN`: the local toolchain is Go 1.23.2 while `go.mod` requires Go 1.25.0, and external toolchain/dependency retrieval is unavailable. No stub or downgraded-toolchain evidence is used as qualification. API-6.3 is `IMPLEMENTED_TESTING_DEFERRED`; API-6.4 is `IMPLEMENTED_TESTING_DEFERRED`; API-7.1 through API-7.4 are `IMPLEMENTED_TESTING_DEFERRED`; API-8 remains `PLANNED`.

## API-6.3 Sequence Anomaly Detection checkpoint — 2026-09-24

API-6.3 is `IMPLEMENTED_TESTING_DEFERRED`. All six planned DETECT-only anomaly classes, bounded evidence/exception primitives, v3 persistence and Reviewer/audit violation visibility are implemented. There is no sequence enforcement authority. API-6.4 is the next planned slice. Source gates through API-6.3 pass; canonical Go 1.25 targeted/race remains `BLOCKED_ENVIRONMENT/NOT_RUN`.


## API-6.4 Sequence Operations + Hardening checkpoint — 2026-09-24

API-6.4 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. It adds explicit site-scoped `LEARN` / `DETECT` controls with mature-only DETECT promotion, bounded persisted recent-session summaries, bounded exception create/delete operations, site-scoped reset/relearn that returns to LEARN, Reviewer RBAC + audit, and the full Sequence Operations console. Durable sequence state advances to v4 with v1/v2/v3 restore compatibility. Sequence analytics still cannot BLOCK, DENY, return request-path 403, or enter ENFORCE. API-6 is implementation-complete through API-6.4; API-7.1 Object Locator Discovery is the next `PLANNED` slice.

Executed static/source evidence: API gates **69/47/46/72/33/56/45/58 PASS** from API-1/2 through API-6.4; OpenAI source contract **16/16 PASS** plus isolated OpenAI/secretref tests PASS; package-builder **9/9 PASS**; root Go shape **105 files PASS**; Admin JavaScript syntax, shell syntax and changed-file gofmt PASS. Canonical Go 1.25 targeted/race remains `BLOCKED_ENVIRONMENT/NOT_RUN` because the host has Go 1.23.2 and external toolchain/module retrieval is unavailable.


## API-7.1 Object Locator Discovery checkpoint — 2026-09-24

API-7.1 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. It adds bounded `ObjectLocator` discovery from API-1 normalized path parameters, API-2 typed path/query/body evidence, matched API-3 OpenAPI contracts, and explicit Reviewer-gated operator INCLUDE/SUPPRESS configuration. Locators are keyed by normalized API-1 operation ID plus location/field; raw object values are transient only and durable value evidence is bounded HMAC-SHA256 fingerprint data protected by a separate mode-0600 key. Client-supplied ownership/tenant headers are not authority, and GraphQL `variables.*` discovery is deferred to API-8.

API-7.1 has no identity/object relationship verdict, tenant-boundary verdict, `BOLA_CANDIDATE`, ownership verdict, BLOCK/DENY/403 path, or ENFORCE authority. OpenAI is absent from the locator authority path. State is bounded by global/per-operation/fingerprint/override caps, 30-day learned TTL, versioned restart validation, existing non-blocking observation-plane ingestion, immediate mutation persistence, RBAC and audit.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83 PASS** from API-1/2 through API-7.1; API-7.1 has **9 targeted test functions present**; OpenAI source contract **16/16 PASS** plus isolated tests PASS; package-source PASS; root Go shape **107 files PASS**; package-builder **9/9 PASS**; changed-file gofmt, Admin JavaScript and primary shell syntax PASS. Canonical Go 1.25 targeted/race remains `BLOCKED_ENVIRONMENT/NOT_RUN` because the host has Go 1.23.2 and external toolchain retrieval is unavailable. Next slice: **API-7.2 Identity/Object Relationship**. API-7.3/API-7.4 is `IMPLEMENTED_TESTING_DEFERRED`; API-8 remains `PLANNED`.

## API-7.2 Identity/Object Relationship checkpoint — 2026-09-24

API-7.2 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. It correlates only API-5 cryptographically verified identity context with ACTIVE API-7.1 locator/object evidence. Identity, tenant and client dimensions are persisted only as API-7.2 HMAC-SHA256 pseudonyms protected by a dedicated mode-0600 key; object values remain API-7.1 keyed fingerprints. Raw JWTs, subjects, tenant/client claim values, cookies, caller-supplied owner/tenant headers and raw object values are not durable relationship state.

The relationship plane is bounded and asynchronous: non-blocking queue capacity 2,048; 8,192 total relationships; 512 relationships per verified identity pseudonym; 256 verified identities per keyed object/locator pair; 30-day TTL; bounded path/query/body capture; restart revalidation; autosave/final flush; shutdown drain; Reviewer-gated read-only evidence/status endpoints with audit. SUPPRESSed API-7.1 locators cannot create relationships. GraphQL `variables.*` remains deferred to API-8.

API-7.2 is evidence only. `OBSERVED`/`REPEATED` means recurrence, not ownership. There is no `BOLA_CANDIDATE`, ownership verdict, tenant-boundary verdict, BLOCK/DENY/403 or ENFORCE authority, and OpenAI is absent from this authority path.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83/100 PASS** from API-1/2 through API-7.2; API-7.2 has **9 targeted Go test functions present**; OpenAI source contract **16/16 PASS** plus isolated OpenAI/secretref tests PASS; package-source PASS; root Go source shape **109 files PASS**; package-builder **9/9 PASS**; changed Go files `gofmt` PASS. Canonical Go 1.25 targeted/race remains `BLOCKED_ENVIRONMENT/NOT_RUN` because this host has Go 1.23.2 and external toolchain retrieval is unavailable. Next slice: **API-7.4 BOLA Policy/Evidence/Console**; API-7.4 is `IMPLEMENTED_TESTING_DEFERRED`; API-8 remains `PLANNED`.



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


## API-8 Post-Audit Hardening — 2026-09-24

**IMPLEMENTED_TESTING_DEFERRED.** CIDR expiry/enable and L7 TLS handshake abuse control are implemented; SYSTEM/HSM/Vector/Debug/Doctor, CIDR/L7 traffic controls, OpenAPI/Positive Schema lifecycle and opaque-ID selectors are exposed in the shipping Console; obsolete `web/` and unwired model-only foundation files are removed. Existing API authority separation remains unchanged. Canonical Go 1.25 runtime/race qualification is still blocked by the local environment.

## 2026-09-25 — Production Correctness & Control-Plane Hardening

Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.

## 2026-09-28 — Current handover status

Code Duplication Review and Consolidation is implemented with testing deferred.
Confirmed superseded/unwired layers were removed; API authority separation and
shipping Console behavior are unchanged. Source gate **80/80 PASS**, 139
Admin/update routes are unique, root Go source shape is **95 files PASS**, and
all prior API/source/package gates listed in the current canonical section remain
PASS. Canonical Go 1.25 qualification remains blocked/not run.

<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
