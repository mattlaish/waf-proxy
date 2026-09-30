# Documentation Index and Current Project Truth


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

### TLS session-resumption documents

- `TLS_SESSION_RESUMPTION_HANDSHAKE_OBSERVABILITY.md` — design, operational
  configuration, metrics semantics, scope and qualification boundary.
- `TLS_SESSION_RESUMPTION_HANDSHAKE_OBSERVABILITY_SOURCE_GATE_RESULT.md` —
  dependency-free 33-check source gate result.
- `TLS_SESSION_RESUMPTION_HANDSHAKE_OBSERVABILITY_TEST_SUMMARY.txt` — exact
  executed-versus-blocked test truth for this slice.

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
| API-4 Positive Schema Enforcement | `IMPLEMENTED_TESTING_DEFERRED` | reviewed immutable profiles; LEARN/DETECT/ENFORCE, exceptions, rollback, bounded violation evidence; local API-4 gates PASS |
| API-5 JWT + Identity-aware Policy | `IMPLEMENTED_TESTING_DEFERRED` | verified JWT/JWKS identity context, DETECT/ENFORCE operation policy, bounded durable evidence; API-5 17-test + race + 72-check source gate PASS |
| API-6.1 Sequence Foundation | `IMPLEMENTED_TESTING_DEFERRED` | bounded async session/transition state, verified/keyed-private correlation, TTL/caps, atomic snapshots, persistence/admin visibility; 33-check source gate PASS; Go tests/race blocked by missing Go toolchain |
| API-6.2 Workflow Learning | `IMPLEMENTED_TESTING_DEFERRED` | LEARN-only workflow model; source gate 58 PASS; Go 1.25 targeted/race BLOCKED_ENVIRONMENT/NOT_RUN |
| API-6.3 Sequence Anomaly Detection | `IMPLEMENTED_TESTING_DEFERRED` | DETECT-only bounded anomaly evidence; source gate 45 PASS; Go 1.25 targeted/race `BLOCKED_ENVIRONMENT/NOT_RUN` |
| API-6.4 Sequence Operations + Hardening | `IMPLEMENTED_TESTING_DEFERRED` | explicit bounded LEARN/DETECT controls, exception CRUD, reset/relearn, recent sessions and full operations console; source gate 58 PASS; Go 1.25 targeted/race `BLOCKED_ENVIRONMENT/NOT_RUN` |
| API-7.1 Object Locator Discovery | `IMPLEMENTED_TESTING_DEFERRED` | normalized locator discovery + bounded keyed evidence; source gate 83 PASS; Go 1.25 targeted/race BLOCKED_ENVIRONMENT/NOT_RUN |
| API-7.2 Identity/Object Relationship | `IMPLEMENTED_TESTING_DEFERRED` | verified identity pseudonym ↔ keyed object relationship evidence; source gate 100 PASS; Go 1.25 targeted/race `BLOCKED_ENVIRONMENT/NOT_RUN` |
| API-7.3 BOLA Detection | `IMPLEMENTED_TESTING_DEFERRED` | bounded DETECT-only identity/object divergence, tenant divergence and enumeration candidate evidence; source gate 110 PASS; Go 1.25 targeted/race `BLOCKED_ENVIRONMENT/NOT_RUN` |
| API-7.4 BOLA Policy/Evidence/Console | `IMPLEMENTED_TESTING_DEFERRED` | bounded REVIEW/SUPPRESS evidence policy, structured evidence workflow, recurrence reopen and complete console; source gate 156 PASS; Go 1.25 targeted/race `BLOCKED_ENVIRONMENT/NOT_RUN` |
| API-8 GraphQL Security | `IMPLEMENTED_TESTING_DEFERRED` | bounded GraphQL parsing/normalization, SDL contract binding, depth/complexity/field/mutation/subscription/introspection/APQ controls, verified-variable API-7 bridge, staged deterministic LEARN/DETECT/ENFORCE; source gate 259 PASS; Go 1.25 targeted/race `BLOCKED_ENVIRONMENT/NOT_RUN` |

API-1 → API-2 → API-3 deterministic end-to-end learning/drift evidence is PASS. The repository-wide Source Buildability Gate remains **BLOCKED**, not PASS: this host has Go 1.23.2 while `go.mod` requires Go 1.25.0, and network/toolchain acquisition is unavailable. Required Go 1.25 `tidy/build/vet/test/race/real-Coraza` qualification remains `NOT_RUN/BLOCKED`. See `API_SECURITY_CLOSURE_RESULT.md`.

This working baseline came from the user-supplied source archive and contains no `.git` metadata; therefore no new branch/commit/push claim is made for this checkpoint.


> **Canonical documentation baseline: 2026-09-23 (Asia/Taipei).**
>
> This file is the starting point for operators, developers, release engineers,
> and future AI sessions. Historical ledgers remain valuable evidence, but when
> older text conflicts with this page and the current gate files, this page plus
> the current gate files are authoritative.

## Current canonical state

- Audited GitHub baseline: `main@1d52d65a73a802e32f02994e51f0a07beb240177`.
- A root-build-integrity repair has been prepared for the broken main baseline.
- Repair status: `IMPLEMENTED_TESTING_DEFERRED`.
- **Source Buildability Gate: BLOCKED.** The exact repaired bytes still require a
  Go 1.25.x execution of `GOTOOLCHAIN=local go mod tidy -diff` and
  `GOTOOLCHAIN=local CGO_ENABLED=0 go build ./...`, followed by vet/tests/race
  and the real-Coraza gate.
- Current documentation/packaging host: Go 1.23.2; Go 1.25/module acquisition is
  unavailable here. A missing prerequisite is BLOCKED, never PASS.
- Artifact integrity/reproducibility and package-fixture tests have independent
  PASS evidence, but they are not evidence that the WAF root binary builds or
  that a production DEB/RPM is qualified.
- GitHub `main` was observed unprotected with required status checks disabled.
  Branch protection/rulesets requiring CI before merge are REQUIRED but were
  not applied by the connected integration because GitHub write/ref operations
  returned HTTP 403.

## API Security closure evidence

- `API8_GRAPHQL_SECURITY.md` — API-8 implementation/authority/privacy/resource-boundary truth.
- `API8_SOURCE_GATE_RESULT.md` — API-8 259-check exact-source gate and deferred Go 1.25 qualification truth.

- `API_SECURITY_CLOSURE_RESULT.md` — authoritative API-1 → API-3 source-closure scope and boundary.
- `TESTING_RESULTS.md` — current executable scoped PASS evidence.
- `SOURCE_BASELINE_GATE_RESULT.md` — exact root Go 1.25 blocker evidence.

## Status vocabulary

Implementation state uses only:

- `PLANNED` — approved but not implemented.
- `IMPLEMENTED_TESTING_DEFERRED` — implementation exists, but one or more
  required validation gates are not executed or are blocked.
- `TESTED` — the defined mandatory tests for that scope have executed and passed.
- `RELEASED` — explicitly approved and released after all release-blocking gates.

Evidence state uses `PASS`, `FAIL`, `BLOCKED`, `NOT_RUN`, and where applicable
`NOT_CONFIGURED`. Never translate `BLOCKED` or `NOT_RUN` into PASS.

## Release blockers and required order

1. Apply the exact root-build-integrity repair to a branch based on the audited
   `main` baseline; do not patch directly to `main`.
2. On Go 1.25.x CI, run dependency-drift, root build, vet, full tests, race, and
   real-Coraza truth gates against the exact proposed commit.
3. Require the CI build/test check through branch protection/rulesets before
   merging to `main`.
4. Only from a green buildable commit, produce real WAF `.deb` and `.rpm`
   packages with `./waf-package` on qualified build hosts.
5. Run package lifecycle qualification (Slice C) on dedicated hosts.
6. Run clean-host distribution qualification (Slice D) on the exact supported
   distro matrix; RHEL-family acceptance requires SELinux `Enforcing`.
7. Complete remaining real Coraza/VectorScan, HSM, reliability, and performance
   gates before any production release decision.

## Distribution state

| Area | State | Current truth |
|---|---|---|
| Debian/Ubuntu DEB builder | `IMPLEMENTED_TESTING_DEFERRED` | fixture mechanics/reproducibility PASS; no production WAF DEB from the repaired source yet |
| RHEL-family RPM builder | `IMPLEMENTED_TESTING_DEFERRED` | source/security validation PASS; actual RPM build is BLOCKED on this host by missing RPM toolchain and root Go build |
| Package upgrade/rollback | `IMPLEMENTED_TESTING_DEFERRED` | harness and DEB fixtures PASS; real package-manager transactions remain NOT_RUN |
| Clean-host qualification | `IMPLEMENTED_TESTING_DEFERRED` | harness/source tests PASS; Debian 12, Ubuntu 22.04/24.04, RHEL/Rocky/Alma/Oracle 9 rows remain NOT_RUN |
| Project package utility | `IMPLEMENTED_TESTING_DEFERRED` | `./waf-package` source/unit gates PASS; real package production remains blocked by build-host prerequisites |

## Runtime and persistence truth

- Coraza v3.7.0 is authoritative for WAF decisions.
- VectorScan/libhs is optional acceleration and must fail safe to Coraza.
- PKCS#11/HSM support is optional and fail-closed; SoftHSM and real-vendor
  qualification remain separate evidence classes.
- Current WAF persistence is filesystem-based under `/etc/waf`,
  `/var/lib/waf-proxy`, and `/var/log/waf`.
- **PostgreSQL is not a current runtime dependency or supported persistence
  backend.** Do not invent `DATABASE_URL`, PostgreSQL migrations, or a database
  provisioning requirement in deployment instructions.
- The shipping admin console is the single authoritative UI: embedded `static/admin.html` plus `static/theme.css`. The obsolete experimental `web/` migration tree was removed during API-8 post-audit hardening.

## Canonical documents

| Document | Purpose |
|---|---|
| `README.md` | product overview and current release truth |
| `INSTALL.md` | production/package installation and operations |
| `DEVELOPMENT.md` | chronological engineering ledger |
| `DEVELOPMENT_ROADMAP.md` | implementation states, blockers, and next order |
| `AI_HANDOFF.md` | current architecture/state for another AI/session |
| `HANDOVER_STATUS.md` | concise human-readable checkpoint |
| `HANDOVER_PROMPT.md` | copy/paste prompt for a new development chat |
| `TESTING.md` | required test/gate policy |
| `TESTING_RESULTS.md` | executed evidence and deferred/blocked truth |
| `SOURCE_BASELINE_GATE_RESULT.md` | root source-buildability gate |
| `RELEASE_PROCESS.md` | release sequencing and artifact rules |
| `MANIFEST.md` | source/delivery file map |
| `patch.md` | implementation patch ledger |

## Package and distribution gate documents

- `DEB_PACKAGE_GATE_RESULT.md`
- `RPM_PACKAGE_GATE_RESULT.md`
- `PACKAGE_LIFECYCLE_GATE_RESULT.md`
- `CLEAN_HOST_DISTRIBUTION_GATE_RESULT.md`
- `PACKAGE_TOOL_GATE_RESULT.md`
- `PACKAGING_TOOL.md`

These documents are scoped evidence. Their PASS entries do not automatically
promote the Source Buildability Gate or overall release status.

## Component documentation

- `benchmark/README.md` — benchmark and performance-certification harness.
- `packaging/deb/README.md` and `packaging/deb/CRS-PROVISIONING.md` — DEB build
  and offline CRS provisioning.
- `packaging/rpm/README.md`, `packaging/rpm/CRS-PROVISIONING.md`, and
  `packaging/rpm/SELINUX.md` — RPM build, offline CRS, and SELinux boundary.
- `packaging/qualification/README.md` — upgrade/rollback qualification.
- `packaging/cleanhost/README.md` — clean-host distro acceptance.
- `qualification/README.md` and subdirectory READMEs — runtime, corpus, HSM,
  and performance qualification.

## Documentation maintenance rule

Every meaningful source/package/release change must update the relevant current
state in `AI_HANDOFF.md`, `DEVELOPMENT.md`, `DEVELOPMENT_ROADMAP.md`,
`TESTING.md`, `TESTING_RESULTS.md`, `MANIFEST.md`, and `patch.md`. Update
`README.md`, `INSTALL.md`, package gate files, and component READMEs whenever
operator behavior or that component's truth changes. Historical evidence may be
retained, but stale "next step" text must be marked historical or removed from
current handover files.

## OpenAI connector hardening — current state

- Status: `IMPLEMENTED_TESTING_DEFERRED`.
- Native OpenAI uses Responses API + strict Structured Outputs; OpenAI-compatible Chat Completions remains available.
- New credentials use `api_key_ref=env:NAME|file:/absolute/path`; legacy inline `api_key` is migration-only.
- Isolated provider/secret tests, race, vet, and 16/16 source contract checks PASS.
- Root Go 1.25 build/test remains BLOCKED; see `OPENAI_INTEGRATION_GATE_RESULT.md`.


## API Security Roadmap

See `API_SECURITY_ROADMAP.md` for the planned API-aware WAF evolution slices.

## API Security implementation documents — current

- `API_SECURITY_ROADMAP.md` — API-1 through API-8 product sequence and current statuses.
- `API_SECURITY_SLICE_IMPLEMENTATION_ROADMAP.md` — canonical detailed slice blueprint.
- `API3_OPENAPI_CONTRACT_IMPLEMENTATION.md` — current API-3 source architecture and security boundary.
- `API3_TEST_MATRIX.md` / `API3_ACCEPTANCE_CRITERIA.md` — API-3 final qualification plan.
- `API3_SOURCE_GATE_RESULT.md` — current executed API-3 source/static evidence.

- `API62_WORKFLOW_LEARNING.md` — API-6.2 Workflow Learning scope, architecture, privacy/cardinality and authority boundary.
- `API62_SOURCE_GATE_RESULT.md` — API-6.2 source-gate and blocked-environment evidence.

### API-6.3 Sequence Anomaly Detection

- `API63_SEQUENCE_ANOMALY_DETECTION.md` — anomaly semantics, maturity safeguards, privacy/resource/authority boundaries and next-slice truth.
- `API63_SOURCE_GATE_RESULT.md` — 45-check API-6.3 source gate plus retained prerequisite/static and blocked-environment evidence.


### API-6.4 Sequence Operations + Hardening

- `API64_SEQUENCE_OPERATIONS_HARDENING.md` — implementation boundary, controls, evidence privacy and console.
- `API64_SOURCE_GATE_RESULT.md` — 58-check source-gate result and qualification boundary.


## API-7.1 Object Locator Discovery checkpoint — 2026-09-24

API-7.1 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. It adds bounded `ObjectLocator` discovery from API-1 normalized path parameters, API-2 typed path/query/body evidence, matched API-3 OpenAPI contracts, and explicit Reviewer-gated operator INCLUDE/SUPPRESS configuration. Locators are keyed by normalized API-1 operation ID plus location/field; raw object values are transient only and durable value evidence is bounded HMAC-SHA256 fingerprint data protected by a separate mode-0600 key. Client-supplied ownership/tenant headers are not authority, and GraphQL `variables.*` discovery is deferred to API-8.

API-7.1 has no identity/object relationship verdict, tenant-boundary verdict, `BOLA_CANDIDATE`, ownership verdict, BLOCK/DENY/403 path, or ENFORCE authority. OpenAI is absent from the locator authority path. State is bounded by global/per-operation/fingerprint/override caps, 30-day learned TTL, versioned restart validation, existing non-blocking observation-plane ingestion, immediate mutation persistence, RBAC and audit.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83 PASS** from API-1/2 through API-7.1; API-7.1 has **9 targeted test functions present**; OpenAI source contract **16/16 PASS** plus isolated tests PASS; package-source PASS; root Go shape **107 files PASS**; package-builder **9/9 PASS**; changed-file gofmt, Admin JavaScript and primary shell syntax PASS. Canonical Go 1.25 targeted/race remains `BLOCKED_ENVIRONMENT/NOT_RUN` because the host has Go 1.23.2 and external toolchain retrieval is unavailable. Next slice: **API-7.2 Identity/Object Relationship**. API-7.3/API-7.4 and API-8 remain `PLANNED`.

### API-7.1 current slice documents

- `API71_OBJECT_LOCATOR_DISCOVERY.md` — API-7.1 architecture, privacy/resource boundaries, admin surface and authority limits.
- `API71_SOURCE_GATE_RESULT.md` — exact-source/static evidence and blocked Go 1.25 qualification truth.

## API-7.2 Identity/Object Relationship

- `API72_IDENTITY_OBJECT_RELATIONSHIP.md` — implementation boundary, privacy model, async architecture, resource limits, persistence, operator visibility and non-authority semantics.
- `API72_SOURCE_GATE_RESULT.md` — API-7.2 100-check source gate, retained compatibility gates and deferred Go 1.25 runtime/race truth.
- `API72_TEST_SUMMARY.txt` — concise API-7.2 executed/deferred qualification summary used by source-artifact packaging.


## API-7.3 BOLA Detection checkpoint — 2026-09-24

API-7.3 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. The detector consumes only API-5 cryptographically verified identity pseudonyms, ACTIVE/non-suppressed API-7.1 keyed object locators/fingerprints, and API-7.2 relationship history. It emits bounded evidence-only candidates for `IDENTITY_OBJECT_DIVERGENCE`, `TENANT_OBJECT_DIVERGENCE`, and `OBJECT_ENUMERATION`. A novel object alone is not a candidate; cross-identity and cross-tenant evidence requires repeated historical baseline, while enumeration requires 20 recent distinct keyed objects under the same locator within 10 minutes.

Candidate state persists only pseudonymous/keyed evidence, is capped at 4,096 total / 256 per identity / 128 per object-locator, expires after seven days, is versioned/revalidated on restart, and is saved by the API security autosave/final-flush path. Detection runs inside the API-7.2 background processing plane before current relationship merge. Reviewer-only candidate/status reads are audited. There is no ownership verdict, tenant-boundary verdict, policy mutation API, `BLOCK`, `DENY`, request-path `403`, or `ENFORCE` authority; OpenAI is absent from the detector authority path.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83/100/110 PASS** from API-1/2 through API-7.3; API-7.3 has **9 targeted Go test functions present**; OpenAI source contract **16/16 PASS** plus isolated tests PASS; package-source PASS; root Go source shape **111 files PASS**; package-builder **9/9 PASS**; changed Go `gofmt`, Admin embedded JavaScript and primary shell/Python syntax PASS. Canonical Go 1.25 `go mod tidy -diff`, root build, API-7.3 targeted tests and API-7.3 race tests remain `BLOCKED_ENVIRONMENT/NOT_RUN` because this host has Go 1.23.2 and external toolchain retrieval is unavailable. Next slice: **API-7.4 BOLA Policy/Evidence/Console**; API-8 remains `PLANNED`.


## API-7.4 BOLA Policy/Evidence/Console checkpoint — 2026-09-24

API-7.4 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. It adds bounded evidence-routing policy with only `REVIEW` and `SUPPRESS`, structured `OPEN` / `ACKNOWLEDGED` / `DISMISSED` / `RESOLVED` review workflow, recurrence-driven reopen behavior, Reviewer RBAC/audit, restart-safe bounded persistence, and the complete BOLA evidence/policy console. Inferred API-7.3 BOLA candidates remain non-enforcing and are never deleted or converted into ownership truth by policy handling.

Primary API-7.4 files are `bola_policy_api74.go`, `bola_policy_api74_test.go`, `tools/tests/test-api74-source.py`, `API74_BOLA_POLICY_EVIDENCE_CONSOLE.md`, `API74_SOURCE_GATE_RESULT.md`, and `API74_TEST_SUMMARY.txt`, plus lifecycle/admin/persistence/console/build/CI integration. Exact-source evidence is API gates **69/47/46/72/33/56/45/58/83/100/110/156 PASS**, **10 API-7.4 targeted Go test functions present**, OpenAI source contract **16/16 PASS** plus isolated package tests, package-source PASS, root Go source shape **113 files PASS**, package-builder **9/9 PASS**, and gofmt/Admin-JS/shell/Python syntax PASS. Canonical Go 1.25 tidy/build/targeted/race qualification remains `BLOCKED_ENVIRONMENT/NOT_RUN`. API-8 GraphQL Security is the next `PLANNED` slice.

## API-8 GraphQL Security checkpoint — 2026-09-24

API-8 is implemented across parsing/normalization, SDL contract binding, deterministic complexity/field/mutation/subscription/introspection policy, Apollo persisted-query support, verified GraphQL-variable integration into API-7 evidence, and Reviewer-gated `LEARN` / `DETECT` / `ENFORCE` policy plus Admin Console. Any policy edit or bound contract content change returns the policy to `LEARN`; a referenced schema contract cannot be deleted. Only explicit deterministic GraphQL policy in `ENFORCE` may reject traffic. API-6/API-7 learned or inferred evidence and OpenAI cannot promote, mutate or bypass GraphQL enforcement.

Durable GraphQL state excludes raw query documents, literals, raw variable values, Authorization/Cookie/JWT data and unverified claims. SDL is reduced to a digest plus normalized topology; GraphQL object values use API-7 keyed fingerprints and API-5 verified identity only. State and parser resources are bounded, versioned, restart-revalidated, autosaved/final-flushed, and strict mutation APIs are Reviewer-gated and audited.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83/100/110/156/259 PASS** through API-8; **22 targeted API-8 Go test functions are present**; OpenAI source contract **16/16 PASS** plus isolated tests PASS; package-source PASS; package-builder **9/9 PASS**; root Go source shape **115 files PASS**; changed API-8 Go files `gofmt`, Admin embedded JavaScript, relevant shell and Python syntax PASS. Canonical Go 1.25 `go mod tidy -diff`, root build/vet/test, API-8 targeted tests and race tests remain `BLOCKED_ENVIRONMENT/NOT_RUN` because the host has Go 1.23.2 and external toolchain retrieval is unavailable. API-8 remains `IMPLEMENTED_TESTING_DEFERRED`, not `TESTED` or `RELEASED`.

The documented API-security implementation roadmap is now complete through API-8. No API-9 is defined. Next gate: canonical Go 1.25 qualification of the exact source/artifact, followed by release promotion only if all required gates pass.


## API-8 post-audit hardening

- `API8_POST_AUDIT_HARDENING.md` — runtime correctness, Console exposure closure, debt cleanup and authority boundaries.
- `API8_POST_AUDIT_HARDENING_SOURCE_GATE_RESULT.md` — dependency-free 134-check hardening source gate.
- `API8_POST_AUDIT_HARDENING_TEST_SUMMARY.txt` — exact-source qualification summary and canonical Go blocker truth.

## 2026-09-25 — Production Correctness & Control-Plane Hardening

Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.

## Documentation review — 2026-09-28

Every Markdown file in the source tree was reviewed for the duplication
consolidation. Current/canonical documents were synchronized to the 2026-09-28
source truth. Historical qualification/result documents intentionally retain
their original dated results; they carry repository review metadata and are not
current-state authority when they conflict with this index or the top current
canonical sections. `MARKDOWN_REVIEW_2026-09-28.md` records the complete file
classification.

New current documents: `CODE_DUPLICATION_REVIEW.md` and
`CODE_DUPLICATION_REVIEW_SOURCE_GATE_RESULT.md`.

## 2026-09-30 — Build and Admin Startup Blocker Fix

The current source fixes two release-blocking defects found during external Go 1.25 review of the 2026-09-28 consolidation baseline. `admin.go` now imports the standard-library `errors` package required by the existing `errors.New(...)` validation paths. Positive-Schema exception state changes now use the unambiguous route `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`; the shipping Console uses the same path. The prior `POST /api/security/schema/enforcement/exceptions/{exception_id}` registration is removed because it conflicts with `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+ `http.ServeMux` specificity rules and can panic during Admin handler construction.

`admin_route_registration_test.go` now makes full Admin handler registration a regression test, and `tools/tests/test-build-startup-blocker-fix-source.py` is wired into `build.sh` and CI. The source gate also registers the complete 139-route Admin/update inventory with the Go standard library in a standalone temporary program; this catches wildcard-pattern ambiguity even when the repository module graph cannot be compiled locally. Current source evidence: blocker-fix gate **13/13 PASS**, route inventory **139/139 accepted by `ServeMux`**, API/source gates through API-8/post-audit/production-hardening/duplication remain PASS, OpenAI source contract and isolated tests PASS, package-source PASS, package-builder **9/9 PASS**, root Go source shape **96 files PASS**, Admin embedded JavaScript PASS, and changed Go files are `gofmt` clean.

This does **not** promote the source to `TESTED` or `RELEASED`. On the current execution host, canonical `GOTOOLCHAIN=local` Go 1.25 `go mod tidy -diff`, root build, vet, targeted Admin/API64 tests, and race remain `BLOCKED_ENVIRONMENT / NOT_RUN` because only Go 1.23.2 is installed and external Go toolchain/module retrieval is unavailable. Packaging/source-manifest integrity is not a substitute for those root compilation gates. No API-security authority changes are introduced: API-6/API-7 remain non-enforcing, API-8 retains explicit deterministic GraphQL ENFORCE authority, and OpenAI remains advisory only.
## 2026-09-30 — OWI-1.0 R3 Dashboard Connector

Status remains **IMPLEMENTED_TESTING_DEFERRED**. This is a product integration slice, **not API-9**, and it does not change API-1 through API-8 enforcement authority. The WAF now exposes a separate read-only management reader at `/api/integrations/dashboard/v1` for required ASSET, DETECTION, POLICY and HEALTH lanes using a dedicated digest-only opaque bearer identity. Optional EVENT/ACTION_STATUS remain disabled and `actions=[]`.

R3 controls implemented in source include per-tenant+principal rate/concurrency plus a global ceiling, strict GET-only handling including HEAD rejection, an 8s default request deadline propagated into source sync and context-aware store-lock acquisition, hard snapshot/export/response bounds, durable cursor/snapshot/revision state, non-blocking WAF security export with explicit GAP semantics, fail-closed durability after write errors, opaque-token expiry/rotation/revoke, owner-only non-symlink token/cursor-secret files, TLS >=1.2 on the built-in reader listener, tenant/scope-bound cursors, and allow-listed sanitization. No Dashboard registry/validator/UI code is modified.

Supporting exact connector-source evidence on the available Go 1.23.2 host is **25 PASS / 2 intentional SKIP**, with the same set passing under `-race`; real subprocess SIGKILL→restart/GAP, store contention deadline/retry, injected write failure/recovery, client cancellation, oversize failure, scope reduction, and accelerated retention expiry are covered. The R3 source gate is **38/38 PASS**. The regenerated product return validates **61 files, 18 positive responses, 15 negative cases and 11 transport cases**. Integrated repository Go 1.25 build/vet/test/race remains `BLOCKED_ENVIRONMENT / NOT_RUN`; G2 Dashboard offline acceptance and G3 live TLS/ACL/intended-runtime qualification remain `NOT_RUN`.

<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->

- `DASHBOARD_CONNECTOR_R3.md` — current OWI-1.0 R3 connector architecture, controls and qualification truth.
- `DASHBOARD_CONNECTOR_R3_SOURCE_GATE_RESULT.md` — dependency-free R3 source-gate evidence.
