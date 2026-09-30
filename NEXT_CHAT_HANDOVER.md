# Next Chat Handover


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
| API-6.1 Sequence Foundation | `IMPLEMENTED_TESTING_DEFERRED` | bounded/private session-transition foundation; source gate 33 PASS; Go tests/race blocked by missing toolchain |
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


Continuation baseline: API-1/API-2/API-3 source closure dated 2026-09-23.

Completed in source: durable API-1 inventory/controls/UI; live/persistent API-2 typed learning with privacy/lifecycle controls; advisory OpenAI schema review; durable API-3 OpenAPI import/export, parameters/enums/security and expanded drift; deterministic API-1→API-2→API-3 integration evidence.

Qualification state: root Go 1.25 tidy/build/vet/test/race/real-Coraza remains BLOCKED/NOT_RUN on this host.

Next work: continue with API-7.4 BOLA Policy/Evidence/Console unless explicitly directed otherwise; separately run exact-source Go 1.25 qualification when the environment permits.


Local executable evidence for this checkpoint is captured in `TESTING_RESULTS.md`; root Go 1.25 blocker evidence is captured in `SOURCE_BASELINE_GATE_RESULT.md`.

## API-5 JWT + Identity-aware API Security checkpoint — 2026-09-23

API-5 is implemented in source and remains `IMPLEMENTED_TESTING_DEFERRED`. `identity_api5.go` provides trusted issuer/JWKS verification, verified-claims-only request context, operation-scoped role/scope/tenant/client authorization, DETECT→ENFORCE lifecycle, semantic-change rollback to DETECT, bounded privacy-preserving evidence, and durable `api-identity.json` state. JWKS fetch is HTTPS-only and bounded; unknown-`kid` rotation refresh is serialized and rate-bounded. Raw bearer tokens, unverified claims and JWKS key cache are not persisted. OpenAI has no identity-enforcement authority.

Local evidence: 17 API-5 deterministic test functions PASS; targeted race PASS; `tools/tests/test-api5-source.py` 72 checks PASS; root source-shape PASS for 97 Go files; admin inline JavaScript syntax PASS; and the external-dependency stub harness passes whole-repository `go build ./...`, `go vet ./...`, and all test-binary compilation. These are not substitutes for the canonical Go 1.25 real-dependency gate, which remains `BLOCKED_ENVIRONMENT/NOT_RUN` (`go mod tidy -diff`, build, vet, full tests, full race, real-Coraza). API-6 remains `PLANNED` and has not started.

## API-6.1 Sequence Foundation checkpoint — 2026-09-23

Only API-6.1 is implemented. `sequence_api61.go` adds bounded non-blocking sessions/transitions, API-1 normalized nodes, API-5 verified-identity-only correlation, keyed anonymous digests, TTL/caps, atomic snapshots, restart persistence/final drain, and Reviewer-gated/audited visibility. API-6.2 and API-6.3 are implemented with testing deferred; API-6.4 and API-7 remain planned. Source gate 33 checks PASS; targeted Go and race tests are present but `BLOCKED_ENVIRONMENT/NOT_RUN` here because no Go toolchain is installed.


## API-6.2 Workflow Learning checkpoint — 2026-09-24

API-6.2 is implemented in source and remains `IMPLEMENTED_TESTING_DEFERRED`. The slice adds LEARN-only workflow cohorts, transition observation/session counts, frequency-derived confidence, `LEARNING`/`MATURE`/`STALE` cold-start semantics, workflow depth plus entry/terminal operation summaries, idle and absolute session lifetime, bounded workflow/session/transition state, API-5 verified-identity-only cohort input, durable v2 state with API-6.1 v1 migration, and Reviewer-gated/audited read-only workflow visibility. It adds no anomaly verdict, BLOCK/DENY/403 behavior, BOLA verdict, reset/relearn control, or sequence ENFORCE mode.

Executed source evidence: API-6.1 gate **33/33 PASS**, API-6.2 gate **56/56 PASS**, and Admin inline JavaScript syntax **PASS**. Seven targeted API-6.2 Go tests are present, including concurrency/race-relevant, restart/privacy and bounded-resource/adversarial cases. Canonical Go 1.25 targeted/race execution is `BLOCKED_ENVIRONMENT/NOT_RUN`: the local toolchain is Go 1.23.2 while `go.mod` requires Go 1.25.0, and external toolchain/dependency retrieval is unavailable. No stub or downgraded-toolchain evidence is used as qualification. API-6.3 is `IMPLEMENTED_TESTING_DEFERRED`; API-6.4 is `IMPLEMENTED_TESTING_DEFERRED`; API-7.1 through API-7.4 are `IMPLEMENTED_TESTING_DEFERRED`; API-8 remains `PLANNED`.

## API-6.3 Sequence Anomaly Detection checkpoint — 2026-09-24

Continue from the API-6.3 complete source baseline. API-6.3 is `IMPLEMENTED_TESTING_DEFERRED`, not TESTED/RELEASED. Six DETECT-only anomaly classes, bounded violation/exception evidence, v3 durable state and Reviewer/audit violation reads are present; no BLOCK/DENY/403 or sequence ENFORCE exists. Next coding slice is API-6.4 Sequence Operations + Hardening. Do not start API-7 until API-6.4 is completed unless explicitly directed otherwise.


## API-6.4 Sequence Operations + Hardening checkpoint — 2026-09-24

API-6.4 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. It adds explicit site-scoped `LEARN` / `DETECT` controls with mature-only DETECT promotion, bounded persisted recent-session summaries, bounded exception create/delete operations, site-scoped reset/relearn that returns to LEARN, Reviewer RBAC + audit, and the full Sequence Operations console. Durable sequence state advances to v4 with v1/v2/v3 restore compatibility. Sequence analytics still cannot BLOCK, DENY, return request-path 403, or enter ENFORCE. API-6 is implementation-complete through API-6.4; API-7.1 Object Locator Discovery, API-7.2 Identity/Object Relationship, and API-7.3 BOLA Detection are implemented with testing deferred; API-7.4 BOLA Policy/Evidence/Console is `IMPLEMENTED_TESTING_DEFERRED`; API-8 GraphQL Security is the next `PLANNED` slice.

Executed static/source evidence: API gates **69/47/46/72/33/56/45/58 PASS** from API-1/2 through API-6.4; OpenAI source contract **16/16 PASS** plus isolated OpenAI/secretref tests PASS; package-builder **9/9 PASS**; root Go shape **105 files PASS**; Admin JavaScript syntax, shell syntax and changed-file gofmt PASS. Canonical Go 1.25 targeted/race remains `BLOCKED_ENVIRONMENT/NOT_RUN` because the host has Go 1.23.2 and external toolchain/module retrieval is unavailable.


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

API-7.3 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. It consumes only API-5 cryptographically verified identity pseudonyms, ACTIVE/non-suppressed API-7.1 keyed object locators/fingerprints, and API-7.2 relationship history. It emits bounded detection-only evidence for `IDENTITY_OBJECT_DIVERGENCE`, `TENANT_OBJECT_DIVERGENCE`, and `OBJECT_ENUMERATION`; novel objects alone are not candidates.

Candidate state is capped at 4,096 total / 256 per identity / 128 per object-locator, expires after seven days, is versioned/revalidated on restart, and is saved through API security autosave/final flush. Detection runs in the API-7.2 background processor before current relationship merge. Reviewer-only candidate/status APIs are audited. There is no ownership verdict, tenant-boundary verdict, policy mutation API, `BLOCK`, `DENY`, request-path `403`, or `ENFORCE` authority, and OpenAI is absent from the detector path.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83/100/110 PASS** through API-7.3; nine API-7.3 targeted Go tests are present; OpenAI 16/16 + isolated tests, package-source, package-builder 9/9, root Go source-shape 111, gofmt and syntax gates PASS. Canonical Go 1.25 tidy/build/targeted/race remains `BLOCKED_ENVIRONMENT/NOT_RUN`. Next slice: **API-7.4 BOLA Policy/Evidence/Console**; API-8 remains `PLANNED`.

## API-7.4 BOLA Policy/Evidence/Console checkpoint — 2026-09-24

API-7.4 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. It completes the API-7 BOLA implementation track with a bounded evidence-handling policy plane and operator workflow. Policies are scoped only by normalized API-1 operation ID, optional API-7.1 locator ID, optional API-7.3 candidate type, and minimum confidence. The only actions are `REVIEW` and `SUPPRESS`; suppression never deletes the underlying API-7.3 candidate. Operator evidence workflow is `OPEN` / `ACKNOWLEDGED` / `DISMISSED` / `RESOLVED` with enumerated reason codes only. New detector evidence after dismissal/resolution reopens the evidence automatically.

API-7.4 is deliberately absent from the API-7.2 relationship processor and API-7.3 detector authority path. It accepts no raw identity/object/tenant/client selectors, arbitrary headers, cookies, Authorization data, or free-text review notes. There is no ownership verdict, request-path `BLOCK`, `DENY`, `403`, or `ENFORCE` authority, and OpenAI is absent from the API-7.4 authority path. State is versioned/revalidated on restart, mutex-protected, TTL/cardinality bounded (1,024 policies; 4,096 reviews; 30-day default policy/review TTL; 180-day policy maximum), immediately persisted on mutation, and included in API security autosave/final flush. Reviewer-only policy/evidence APIs are audited and the embedded console exposes effective policy, evidence workflow, suppression, and reopen status.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83/100/110/156 PASS** from API-1/2 through API-7.4; **10 API-7.4 targeted Go test functions are present**; OpenAI source contract **16/16 PASS** plus isolated package tests PASS; package-source PASS; root Go source shape **113 files PASS**; package-builder **9/9 PASS**; `gofmt`, Admin embedded JavaScript and primary shell/Python syntax PASS. Canonical Go 1.25 `go mod tidy -diff`, root build/test, API-7.4 targeted tests and API-7.4 race tests remain `BLOCKED_ENVIRONMENT/NOT_RUN` because this host has Go 1.23.2 and external toolchain retrieval is unavailable. API-8 GraphQL Security is the next `PLANNED` slice.

## API-8 GraphQL Security checkpoint — 2026-09-24

API-8 is implemented across parsing/normalization, SDL contract binding, deterministic complexity/field/mutation/subscription/introspection policy, Apollo persisted-query support, verified GraphQL-variable integration into API-7 evidence, and Reviewer-gated `LEARN` / `DETECT` / `ENFORCE` policy plus Admin Console. Any policy edit or bound contract content change returns the policy to `LEARN`; a referenced schema contract cannot be deleted. Only explicit deterministic GraphQL policy in `ENFORCE` may reject traffic. API-6/API-7 learned or inferred evidence and OpenAI cannot promote, mutate or bypass GraphQL enforcement.

Durable GraphQL state excludes raw query documents, literals, raw variable values, Authorization/Cookie/JWT data and unverified claims. SDL is reduced to a digest plus normalized topology; GraphQL object values use API-7 keyed fingerprints and API-5 verified identity only. State and parser resources are bounded, versioned, restart-revalidated, autosaved/final-flushed, and strict mutation APIs are Reviewer-gated and audited.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83/100/110/156/259 PASS** through API-8; **22 targeted API-8 Go test functions are present**; OpenAI source contract **16/16 PASS** plus isolated tests PASS; package-source PASS; package-builder **9/9 PASS**; root Go source shape **115 files PASS**; changed API-8 Go files `gofmt`, Admin embedded JavaScript, relevant shell and Python syntax PASS. Canonical Go 1.25 `go mod tidy -diff`, root build/vet/test, API-8 targeted tests and race tests remain `BLOCKED_ENVIRONMENT/NOT_RUN` because the host has Go 1.23.2 and external toolchain retrieval is unavailable. API-8 remains `IMPLEMENTED_TESTING_DEFERRED`, not `TESTED` or `RELEASED`.

The documented API-security implementation roadmap is now complete through API-8. No API-9 is defined. Next gate: canonical Go 1.25 qualification of the exact source/artifact, followed by release promotion only if all required gates pass.


## API-8 Post-Audit Hardening checkpoint

Continue from the post-audit hardening complete-source artifact, not the pre-hardening API-8 ZIP, once its final artifact gate is recorded. Do not reintroduce `web/` or the deleted model-only Security Operations/debug lifecycle files as if they were implemented features. Phase 5 Slice D remains PLANNED / NOT IMPLEMENTED until a real store/API/Console workflow is built. Before release promotion, run the exact artifact under pinned Go 1.25 and execute required build/vet/test/race plus production dependency qualification.

## 2026-09-25 — Production Correctness & Control-Plane Hardening

Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.

## 2026-09-28 — Next-chat authority

Use the Code Duplication Review and Consolidation artifact once its clean-extract
gate is recorded. Do not restore deleted legacy debug/security-state/reliability/
VectorScan/API-2/PKI/coverage helper layers without a new runtime requirement and
explicit architecture decision. Treat `internal/capability` as the single
SecLang tokenizer/action helper authority shared by coverage and VectorScan.
No API-9 is defined; next product promotion work remains pinned Go 1.25 and live
production qualification.

## 2026-09-30 — Build and Admin Startup Blocker Fix

The current source fixes two release-blocking defects found during external Go 1.25 review of the 2026-09-28 consolidation baseline. `admin.go` now imports the standard-library `errors` package required by the existing `errors.New(...)` validation paths. Positive-Schema exception state changes now use the unambiguous route `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`; the shipping Console uses the same path. The prior `POST /api/security/schema/enforcement/exceptions/{exception_id}` registration is removed because it conflicts with `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+ `http.ServeMux` specificity rules and can panic during Admin handler construction.

`admin_route_registration_test.go` now makes full Admin handler registration a regression test, and `tools/tests/test-build-startup-blocker-fix-source.py` is wired into `build.sh` and CI. The source gate also registers the complete 139-route Admin/update inventory with the Go standard library in a standalone temporary program; this catches wildcard-pattern ambiguity even when the repository module graph cannot be compiled locally. Current source evidence: blocker-fix gate **13/13 PASS**, route inventory **139/139 accepted by `ServeMux`**, API/source gates through API-8/post-audit/production-hardening/duplication remain PASS, OpenAI source contract and isolated tests PASS, package-source PASS, package-builder **9/9 PASS**, root Go source shape **96 files PASS**, Admin embedded JavaScript PASS, and changed Go files are `gofmt` clean.

This does **not** promote the source to `TESTED` or `RELEASED`. On the current execution host, canonical `GOTOOLCHAIN=local` Go 1.25 `go mod tidy -diff`, root build, vet, targeted Admin/API64 tests, and race remain `BLOCKED_ENVIRONMENT / NOT_RUN` because only Go 1.23.2 is installed and external Go toolchain/module retrieval is unavailable. Packaging/source-manifest integrity is not a substitute for those root compilation gates. No API-security authority changes are introduced: API-6/API-7 remain non-enforcing, API-8 retains explicit deterministic GraphQL ENFORCE authority, and OpenAI remains advisory only.

Continuation point: use this blocker-fixed tree as the current source, not the 2026-09-28 consolidation ZIP. Do not restore the ambiguous exception-toggle route. The next qualification step is to run the exact final bytes under pinned Go 1.25 with `go mod tidy -diff`, `CGO_ENABLED=0 go build ./...`, `go vet ./...`, the targeted Admin route/API64 tests, full tests, and race tests before any buildable/PASS or release claim.
## 2026-09-30 — OWI-1.0 R3 Dashboard Connector

Status remains **IMPLEMENTED_TESTING_DEFERRED**. This is a product integration slice, **not API-9**, and it does not change API-1 through API-8 enforcement authority. The WAF now exposes a separate read-only management reader at `/api/integrations/dashboard/v1` for required ASSET, DETECTION, POLICY and HEALTH lanes using a dedicated digest-only opaque bearer identity. Optional EVENT/ACTION_STATUS remain disabled and `actions=[]`.

R3 controls implemented in source include per-tenant+principal rate/concurrency plus a global ceiling, strict GET-only handling including HEAD rejection, an 8s default request deadline propagated into source sync and context-aware store-lock acquisition, hard snapshot/export/response bounds, durable cursor/snapshot/revision state, non-blocking WAF security export with explicit GAP semantics, fail-closed durability after write errors, opaque-token expiry/rotation/revoke, owner-only non-symlink token/cursor-secret files, TLS >=1.2 on the built-in reader listener, tenant/scope-bound cursors, and allow-listed sanitization. No Dashboard registry/validator/UI code is modified.

Supporting exact connector-source evidence on the available Go 1.23.2 host is **25 PASS / 2 intentional SKIP**, with the same set passing under `-race`; real subprocess SIGKILL→restart/GAP, store contention deadline/retry, injected write failure/recovery, client cancellation, oversize failure, scope reduction, and accelerated retention expiry are covered. The R3 source gate is **38/38 PASS**. The regenerated product return validates **61 files, 18 positive responses, 15 negative cases and 11 transport cases**. Integrated repository Go 1.25 build/vet/test/race remains `BLOCKED_ENVIRONMENT / NOT_RUN`; G2 Dashboard offline acceptance and G3 live TLS/ACL/intended-runtime qualification remain `NOT_RUN`.

<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
