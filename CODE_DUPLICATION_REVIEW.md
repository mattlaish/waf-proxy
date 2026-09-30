# Code Duplication and Functional-Overlap Review

**Review date:** 2026-09-28 (Asia/Taipei)  
**Parent source:** `waf-proxy-production-control-plane-hardening-2026-09-25.zip`  
**Parent SHA-256:** `90acc977db2998f4a3c1c4cacc07e19f0087f8bc033bce6b9934b83415957be8`  
**State:** `IMPLEMENTED_TESTING_DEFERRED`

## Scope

This review covers the complete source tree, including the root WAF service,
API-1 through API-8 security slices, control-plane/Admin API, shipping Console,
PKI, HA, debug/support evidence, VectorScan/coverage tooling, qualification
packages, packaging, service units, and release tooling.

The review distinguished three different cases:

1. **Functional overlap** — two implementations claim the same runtime or
   operator responsibility. These must be consolidated or one must be removed.
2. **Code-shape similarity** — small lifecycle/list/snapshot helpers are similar
   but operate on different bounded stores with different authority. These are
   not merged merely for DRY style.
3. **Historical/qualification models** — code under explicit qualification
   scope may intentionally model evidence without participating in the runtime.
   It is retained when it has a real qualification responsibility.

The source archive contains no `.git` metadata. No branch, commit, pull, push,
or fresh-clone claim is made. Source identity is the parent ZIP checksum plus
its manifests.

## Review methods

- enumerated all Go source files and package directories;
- checked Admin `METHOD + path` registrations for collisions;
- compared top-level declarations and substantial exact function bodies;
- scanned production symbols for definitions with no production caller;
- traced runtime stores, persistence, request-path and Admin/API callers;
- compared shipping Console function names and HTML IDs for duplicates;
- reviewed API security enforcement boundaries for competing authority;
- compared offline CRS coverage parsing with live VectorScan parsing;
- re-ran all existing source gates after consolidation.

## Consolidated duplicate or superseded functionality

### Debug evidence

The repository carried four overlapping layers:

- the active tenant-scoped `DebugEvidenceStore` in `debug_bundle.go`;
- a never-initialized compatibility `DebugEvidenceCapture` in
  `debug_evidence.go`;
- a test-only `DebugEvidenceOpsV2` wrapper;
- unused support-bundle provenance structs.

The compatibility capture was never installed by production code, so its
Coraza callback branch was always inactive. The active store is now the only
debug evidence authority. Shared sanitization moved to `debug_sanitize.go`.
The obsolete compatibility wrapper, V2 wrapper and unused support-bundle model
were removed. `DebugEvidenceStore.TenantExport`, which had no caller and merely
wrapped `ExportIncident`, was also removed.

### Generic security-state placeholders

`security_state.go` and `security_event.go` defined a second generic in-memory
indicator/audit/event model with no runtime caller. The product already has
concrete bounded stores and audit/notification/evidence models. The unused
generic layer and its isolated test were removed so there is no second implied
security-state authority.

### Deployment/reliability foundations

`deployment_readiness.go` duplicated the responsibility already served by the
wired `/api/doctor`, `waf-doctor.sh`, package qualification and release-host
qualification flows. `reliability.go` was a standalone `NOT_RUN` report model
used only by its own test. Both were removed; the real diagnostic and
qualification paths remain unchanged.

### VectorScan qualification/audit placeholders

Root-level `vectorscan_qualification.go`, `vectorscan_audit_store.go` and
`vectorscan_transition_audit.go` were not wired to runtime or the real
qualification command. The authoritative paths are:

- `internal/vectoraccel` for runtime planning/learning and native observation;
- `cmd/wafqualify` for real Coraza-versus-VectorScan differential execution;
- `qualification/vectorscan` for qualification evidence schema/tests.

The unused root placeholder models were removed. In
`internal/vectoraccel/qualification.go`, the test-only `QualificationResult` /
`CompareCandidates` duplicate was removed while the live
`QualificationCandidates`, `EligibleRuleIDs`, `CandidateRuleIDs` and
`EligibleMatchedRuleIDs` methods remain.

### API-2 continuation placeholder

`schema_api2_continuation.go` had no caller. The live API-2 schema collector,
review lifecycle, persistence and Admin surface are in `schema_api2.go` and its
existing integrations. The unreferenced continuation model was removed.

### PKI CRL hardening duplicate

`pki_hardening.go` was an isolated helper/test pair and was not used by the
backend TLS path. The wired CRL implementation in `pki.go` + `pki_url.go` is
stricter: HTTPS-only URL syntax, DNS/address SSRF controls, response size
bounds, parsing/signature validation, cache handling and last-known-good
refresh behavior. The unused earlier helper/test pair was removed.

### CRS/VectorScan SecLang parsing

Offline CRS coverage analysis and the live VectorScan rule classifier carried
separate implementations of token parsing, action-list splitting,
`name:value` splitting and quoted-action trimming. These were semantically the
same and could drift independently.

`internal/capability/seclang.go` is now the single parser-helper source used by
both `internal/coverage/crs/parser.go` and `internal/vectoraccel/rules.go`.
Focused tests lock quote/comma/action behavior.

### Legacy coverage report

`internal/coverage/report.go` was the unused Slice-A summary while
`internal/coverage/crs/report.go` is the current ruleset-aware Coverage Report
v2 used by `wafctl`. The unused report was removed. The unused transform-report
API in `internal/coverage/capability.go` was also removed; eligibility continues
to be decided by `internal/capability.Classify`.

### Other unused compatibility helpers

The review removed small no-caller helpers whose live replacements already
exist: `QualificationPassed`, `ActualListenerKey`, `nonAnonymousAuthSamples`,
`sequenceTransitionID`, and the uninstalled `ClientIdentityAuditSink` hook.
These removals do not change a public importable API because the affected root
package is `main` and the TLS helper is under `internal/`.

## Similar code intentionally retained

The following similarities are **not functional duplication** and were left in
place:

- API-6 sequence, API-7 relationship/BOLA and API-8 GraphQL stores each own
  distinct state, persistence and authority boundaries.
- `matchRing` stores recent operator-visible records while `matchLogPlane`
  asynchronously aggregates human-oriented structured log output.
- observation, sequence and relationship workers have similar start/drain
  lifecycle code but process different queues and state machines.
- short list/status handlers are intentionally local to their resource model.
- `notify.go` and `syslog.go` are distinct delivery sinks with different
  semantics; neither replaces the other.
- packaging/update scripts and runtime update controls remain separate because
  one builds/distributes packages while the other governs an installed node.

Merging these only to reduce line count would couple independent authority or
failure domains and is therefore rejected.

## Authority review

No competing request-path enforcement stack was found:

- Coraza/CRS remains deterministic WAF enforcement.
- API-4 Positive Schema may enforce explicit reviewed schema policy.
- API-5 may enforce cryptographically verified identity policy.
- API-6 sequence state is LEARN/DETECT only.
- API-7 learned/inferred BOLA evidence remains non-blocking.
- API-8 may enforce explicit deterministic GraphQL policy.
- OpenAI remains advisory and has no request-path authority.

Admin route registrations remain unique by `METHOD + path`; the cleanup does
not introduce an alternate management route for the same action.

## Console review

The shipping Console has one implementation (`static/admin.html` plus
`static/theme.css`). The removed experimental `web/` tree remains absent.
Within `static/admin.html`, JavaScript function names and static HTML IDs are
unique. Existing role/capability checks and API surfaces remain unchanged by
this consolidation.

## Verification boundary

All dependency-free/API source gates are re-run after this cleanup, including
the dedicated `test-code-duplication-review-source.py` gate. Canonical Go 1.25
`go mod tidy -diff`, build, tests, race and real dependency qualification still
require the pinned Go 1.25 environment. This review must not be promoted to
`TESTED` or `RELEASED` based only on source/static evidence.

## 2026-09-30 — Build and Admin Startup Blocker Fix

The current source fixes two release-blocking defects found during external Go 1.25 review of the 2026-09-28 consolidation baseline. `admin.go` now imports the standard-library `errors` package required by the existing `errors.New(...)` validation paths. Positive-Schema exception state changes now use the unambiguous route `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`; the shipping Console uses the same path. The prior `POST /api/security/schema/enforcement/exceptions/{exception_id}` registration is removed because it conflicts with `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+ `http.ServeMux` specificity rules and can panic during Admin handler construction.

`admin_route_registration_test.go` now makes full Admin handler registration a regression test, and `tools/tests/test-build-startup-blocker-fix-source.py` is wired into `build.sh` and CI. The source gate also registers the complete 139-route Admin/update inventory with the Go standard library in a standalone temporary program; this catches wildcard-pattern ambiguity even when the repository module graph cannot be compiled locally. Current source evidence: blocker-fix gate **13/13 PASS**, route inventory **139/139 accepted by `ServeMux`**, API/source gates through API-8/post-audit/production-hardening/duplication remain PASS, OpenAI source contract and isolated tests PASS, package-source PASS, package-builder **9/9 PASS**, root Go source shape **96 files PASS**, Admin embedded JavaScript PASS, and changed Go files are `gofmt` clean.

This does **not** promote the source to `TESTED` or `RELEASED`. On the current execution host, canonical `GOTOOLCHAIN=local` Go 1.25 `go mod tidy -diff`, root build, vet, targeted Admin/API64 tests, and race remain `BLOCKED_ENVIRONMENT / NOT_RUN` because only Go 1.23.2 is installed and external Go toolchain/module retrieval is unavailable. Packaging/source-manifest integrity is not a substitute for those root compilation gates. No API-security authority changes are introduced: API-6/API-7 remain non-enforcing, API-8 retains explicit deterministic GraphQL ENFORCE authority, and OpenAI remains advisory only.

<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->

## 2026-09-30 addendum — TLS Session Resumption + Handshake Observability

This operational TLS slice was reviewed against the earlier duplication
findings. It adds one `tlsSessionTicketManager` for the built-in Go TLS
termination path and one handshake-observation hook; it does not introduce a
second certificate selector, WAF enforcement engine, API-security store,
background enforcement worker, Admin route, Console implementation, packaging
system, or update path. The Admin/update route inventory remains 139 patterns
and the existing duplication source gate remains 80/80 PASS.

The manager does not own external TLS-frontend ticket state. API-1 through
API-8 stores and enforcement boundaries remain unchanged, and no API-9 is
defined. Similar TLS metrics are consolidated into the existing `metrics`
object and existing `/api/metrics`/shipping Console surfaces rather than adding
parallel telemetry state or UI.
## 2026-09-30 — OWI-1.0 R3 Dashboard Connector

Status remains **IMPLEMENTED_TESTING_DEFERRED**. This is a product integration slice, **not API-9**, and it does not change API-1 through API-8 enforcement authority. The WAF now exposes a separate read-only management reader at `/api/integrations/dashboard/v1` for required ASSET, DETECTION, POLICY and HEALTH lanes using a dedicated digest-only opaque bearer identity. Optional EVENT/ACTION_STATUS remain disabled and `actions=[]`.

R3 controls implemented in source include per-tenant+principal rate/concurrency plus a global ceiling, strict GET-only handling including HEAD rejection, an 8s default request deadline propagated into source sync and context-aware store-lock acquisition, hard snapshot/export/response bounds, durable cursor/snapshot/revision state, non-blocking WAF security export with explicit GAP semantics, fail-closed durability after write errors, opaque-token expiry/rotation/revoke, owner-only non-symlink token/cursor-secret files, TLS >=1.2 on the built-in reader listener, tenant/scope-bound cursors, and allow-listed sanitization. No Dashboard registry/validator/UI code is modified.

Supporting exact connector-source evidence on the available Go 1.23.2 host is **25 PASS / 2 intentional SKIP**, with the same set passing under `-race`; real subprocess SIGKILL→restart/GAP, store contention deadline/retry, injected write failure/recovery, client cancellation, oversize failure, scope reduction, and accelerated retention expiry are covered. The R3 source gate is **38/38 PASS**. The regenerated product return validates **61 files, 18 positive responses, 15 negative cases and 11 transport cases**. Integrated repository Go 1.25 build/vet/test/race remains `BLOCKED_ENVIRONMENT / NOT_RUN`; G2 Dashboard offline acceptance and G3 live TLS/ACL/intended-runtime qualification remain `NOT_RUN`.

