# Development Roadmap — WAF Proxy


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
| API-1 Discovery + Operation Normalization | `IMPLEMENTED_TESTING_DEFERRED` | durable operation inventory, metadata, controls/UI; 4/4 exact-source isolated tests PASS |
| API-2 Typed Schema Learning | `IMPLEMENTED_TESTING_DEFERRED` | live typed collector, lifecycle, privacy controls, persistence; 4/4 core isolated tests + 1/1 lifecycle-handler test PASS |
| OpenAI SchemaCandidate review | `IMPLEMENTED_TESTING_DEFERRED` | Responses API Structured Outputs, advisory-only; OpenAI source/isolated gates PASS |
| API-3 OpenAPI Contract Management | `IMPLEMENTED_TESTING_DEFERRED` | persistence, import/export, params/enums/security, expanded drift; API-3 source gate + isolated harness PASS |
| API-4 Positive Schema Enforcement | `IMPLEMENTED_TESTING_DEFERRED` | reviewed immutable profiles; LEARN/DETECT/ENFORCE, exceptions, rollback, bounded violation evidence; local API-4 gates PASS |
| API-5 JWT + Identity-aware Policy | `IMPLEMENTED_TESTING_DEFERRED` | verified JWT/JWKS identity context, DETECT/ENFORCE operation policy, bounded durable evidence; API-5 17-test + race + 72-check source gate PASS |
| API-6.1 Sequence Foundation | `IMPLEMENTED_TESTING_DEFERRED` | bounded/private sessions and transitions, TTL/caps, atomic snapshots and persistence; 33-check source gate PASS; Go tests/race blocked by missing toolchain |
| API-6.2 Workflow Learning | `IMPLEMENTED_TESTING_DEFERRED` | LEARN-only workflow model; source gate 56 PASS; Go 1.25 targeted/race BLOCKED_ENVIRONMENT/NOT_RUN |
| API-6.3 Sequence Anomaly Detection | `IMPLEMENTED_TESTING_DEFERRED` | DETECT-only bounded anomaly evidence; source gate 45 PASS; Go 1.25 targeted/race `BLOCKED_ENVIRONMENT/NOT_RUN` |
| API-6.4 Sequence Operations + Hardening | `IMPLEMENTED_TESTING_DEFERRED` | site-scoped LEARN/DETECT, exception CRUD, reset/relearn, recent-session evidence and full operations console |
| API-7.1 Object Locator Discovery | `IMPLEMENTED_TESTING_DEFERRED` | normalized locator discovery + bounded keyed evidence; source gate 83 PASS; Go 1.25 targeted/race BLOCKED_ENVIRONMENT/NOT_RUN |
| API-7.2 Identity/Object Relationship | `IMPLEMENTED_TESTING_DEFERRED` | API-5 verified identity pseudonyms correlated with API-7.1 keyed object evidence; bounded async persistence; no ownership/BOLA/enforcement verdict |
| API-7.3 BOLA Detection | `IMPLEMENTED_TESTING_DEFERRED` | bounded non-enforcing BOLA candidate evidence; source gate 110 PASS |
| API-7.4 BOLA Policy/Evidence/Console | `IMPLEMENTED_TESTING_DEFERRED` | REVIEW/SUPPRESS evidence policy, structured workflow/reopen semantics and complete console; source gate 156 PASS |
| API-8 GraphQL Security | `IMPLEMENTED_TESTING_DEFERRED` | bounded GraphQL parsing/normalization, SDL contract binding, depth/complexity/field/mutation/subscription/introspection/APQ controls, verified-variable API-7 bridge, staged deterministic LEARN/DETECT/ENFORCE; source gate 259 PASS; Go 1.25 targeted/race `BLOCKED_ENVIRONMENT/NOT_RUN` |

API-1 → API-2 → API-3 deterministic end-to-end learning/drift evidence is PASS. The repository-wide Source Buildability Gate remains **BLOCKED**, not PASS: this host has Go 1.23.2 while `go.mod` requires Go 1.25.0, and network/toolchain acquisition is unavailable. Required Go 1.25 `tidy/build/vet/test/race/real-Coraza` qualification remains `NOT_RUN/BLOCKED`. See `API_SECURITY_CLOSURE_RESULT.md`.

This working baseline came from the user-supplied source archive and contains no `.git` metadata; therefore no new branch/commit/push claim is made for this checkpoint.


## Current canonical status — 2026-09-17

The audited GitHub `main@1d52d65a73a802e32f02994e51f0a07beb240177`
was found non-buildable. A root-build-integrity repair is implemented, but the
exact repaired bytes have not executed the mandatory Go 1.25 dependency-drift
and root-build gates in this environment. Current source state is therefore
`IMPLEMENTED_TESTING_DEFERRED` and the Source Buildability Gate is `BLOCKED`.

Artifact integrity, source reconstruction, package fixtures, and component
source checks retain their separately scoped PASS evidence. They do not promote
root buildability or release readiness. Read `DOCUMENTATION_INDEX.md` and
`SOURCE_BASELINE_GATE_RESULT.md` before relying on older historical sections in
this ledger.

Handover checkpoint: 2026-09-17 (Asia/Taipei)

Historical roadmap lineage starts from `waf-proxy-vectorscan-learning-coraza37-2026-09-04.zip`; the current continuation point is the 2026-09-17 root-build-integrity repair described above. VectorScan remains the selected regex-acceleration direction. Do **not** reopen XDP-vs-VectorScan selection unless the owner explicitly asks to revisit it.

## Status vocabulary

Implementation states used for current roadmap decisions:

- **PLANNED** — approved but not implemented.
- **IMPLEMENTED_TESTING_DEFERRED** — implementation exists, but required validation is blocked or not yet run.
- **TESTED** — mandatory tests for that scope have executed successfully.
- **RELEASED** — explicitly approved after all release-blocking gates.

Evidence within a state uses `PASS`, `FAIL`, `BLOCKED`, `NOT_RUN`, and
`NOT_CONFIGURED`. Historical sections may contain the legacy labels `DONE` and
`QUALIFICATION_REQUIRED`; interpret `DONE` as implementation presence only and
`QUALIFICATION_REQUIRED` as `IMPLEMENTED_TESTING_DEFERRED` unless a later gate
explicitly promotes the scope.

## Current implementation baseline

| Area | Status | Current truth |
|---|---|---|
| P0 hot-path cleanup | IMPLEMENTED_TESTING_DEFERRED | circular rings, atomic disabled gates, shared request-body prefix, reduced allocations |
| P0-A proxy buffering | IMPLEMENTED_TESTING_DEFERRED | ReverseProxy BufferPool and streaming-safe FlushInterval behavior |
| P0-B load balancing | IMPLEMENTED_TESTING_DEFERRED | zero-allocation member selection and removal of per-request member context handoff |
| P0-C observations | IMPLEMENTED_TESTING_DEFERRED | bounded async observation plane with non-blocking enqueue |
| P0-D match logging | IMPLEMENTED_TESTING_DEFERRED | independent bounded async aggregation/logging plane |
| P1 response inspection | IMPLEMENTED_TESTING_DEFERRED | inherit/on/off response-body inspection plus configurable response-body limit |
| P1 backend transport | IMPLEMENTED_TESTING_DEFERRED | configurable connection pool, shared pool transport, retired idle-connection cleanup |
| P2 non-XDP hardening | IMPLEMENTED_TESTING_DEFERRED | release scripts fixed, AI lazy sampling, AI queue hardening, atomic blocklist, statusRecorder correctness |
| Benchmark harness | IMPLEMENTED_TESTING_DEFERRED | backend/http/coraza/l4/compare modes and JSON/profile output |
| TLS acceleration C1-C3 | IMPLEMENTED_TESTING_DEFERRED | optional NGINX/OpenSSL frontend, modern HTTP/2 syntax, kTLS/QAT detection/fallback; real QAT hardware remains unqualified |
| Coraza dependency | IMPLEMENTED_TESTING_DEFERRED | source pinned to Coraza v3.7.0 and Go 1.25.0; real release-host execution still required |
| VectorScan Learning Accelerator | IMPLEMENTED_TESTING_DEFERRED | conservative grouped acceleration, transaction-final Coraza truth, Learning/FAILSAFE state machine, optional native libhs |
| XDP prefilter | PLANNED | separate future L3/L4 feature; no longer part of the current selection decision |

## Phase 0 — Mandatory release-host qualification

**Status: IMPLEMENTED_TESTING_DEFERRED — release-host qualification is not yet complete.**

Run on a Linux release/target host with:

- Go >= 1.25.0
- real Coraza v3.7.0 module/dependencies
- `pkg-config`
- real `libvectorscan-dev` / libhs
- the production or representative OWASP CRS ruleset

Qualification automation added 2026-09-07:

- `./qualify-release-host.sh --preflight` validates Linux, the **installed local** Go toolchain without auto-download, exact Go/Coraza pins, CGO compiler, `pkg-config libhs`, and real VectorScan provenance. Exit code 3 means the host is **BLOCKED**, not that a WAF correctness gate failed.
- `./qualify-release-host.sh --core` runs the required native build/test/race path, the explicit real-Coraza truth gate, the native multi-pattern compile/scan semantic gate, and a `go.mod`/`go.sum` drift check.
- `internal/vectoraccel/scanner_native_gate_test.go` is built only with `vectorscan && cgo` and semantically exercises the native `hs_compile_multi` + `hs_scan` path. It only counts as real VectorScan evidence when release provenance is established; ABI-only shims remain non-production evidence.

Required gates:

1. `WAF_VECTORSCAN=required ./build.sh` succeeds without stubs or ABI shims.
2. `go test -tags realcoraza -run 'TestRealCoraza' ./...` passes against real Coraza v3.7.0.
3. DetectionOnly + `nolog` truth gate proves `tx.MatchedRules()` contains the match while ErrorCallback remains silent.
4. Real libhs `hs_compile_multi` and `hs_scan` execute and return the expected pattern IDs in `TestNativeVectorScanCompileAndScanGate`.
5. Native CGO race/regression passes on the release architecture.
6. Install/upgrade/uninstall/doctor smoke passes on a clean Debian/Ubuntu VM.
7. NGINX/OpenSSL frontend `nginx -t` and HTTPS/HTTP2 end-to-end smoke passes.
8. If the target advertises kTLS/QAT, verify ACTIVE behavior separately; do not infer activation from capability detection.

Current 2026-09-07 packaging-host execution: the new preflight ran and correctly returned **BLOCKED** because the installed local toolchain is Go 1.23.2 and no `pkg-config libhs` / verified real VectorScan installation is present. Therefore gates 1-5 remain `NOT_RUN`; no PASS has been promoted from stub/ABI evidence.

**Exit criterion:** all mandatory real Coraza + real VectorScan correctness gates PASS. Performance is useful evidence but is not a prerequisite for choosing VectorScan; the choice is already made.

## Phase 1 — VectorScan Learning production qualification

**Status: IMPLEMENTED_TESTING_DEFERRED — production runner exists; real release-host/corpus execution remains mandatory.**

Start in DetectionOnly with representative CRS and traffic/corpus. Keep Coraza authoritative.

Required validation:

- confirm every eligible group starts/re-enters Learning when its semantic fingerprint changes;
- verify `VectorScan candidates ⊇ Coraza actual matches` for all observed eligible groups;
- hard requirement: observed false negatives = 0;
- verify native scan error -> affected group `FAILSAFE` -> Coraza-only;
- verify persisted learning state survives normal restart but invalidates on rule/Coraza/VectorScan/adapter fingerprint change;
- verify HTTP/2 concurrent identical requests remain transaction-correlated through request context;
- verify verification-sampled no-hit requests still run full Coraza and can force FAILSAFE;
- verify private internal control headers cannot be supplied by clients and never reach backends;
- exercise config reload while requests are in flight to validate native DB/scratch retirement;
- collect `wafbench` RPS/latency/CPU/profile data only as optimization evidence, not as a technology-selection gate.

**Promotion rule:** a group may enter ACCELERATED only after configured Learning thresholds and zero observed false negatives. Any mismatch returns the group to FAILSAFE/Coraza-only.

## Phase 2 — Expand VectorScan coverage conservatively

**Status: IMPLEMENTED_TESTING_DEFERRED — Slices A-C source implementation exists, but production promotion remains blocked until Phase 0/1 qualification passes.**

Current eligible scope is intentionally narrow: standalone positive `@rx` over reproducible request sources and supported transforms. Expand only when exact Coraza input semantics can be reproduced and regression-tested.

Candidate increments, in order:

1. additional exact transforms with byte-for-byte parity tests against Coraza;
2. more fixed-name request-header cases;
3. selected additional single-value request variables;
4. only then investigate ARGS or body-oriented groups, with explicit normalization/parser parity tests;
5. chains, negated operators, multi-variable selectors, dynamic macro semantics and unsupported transforms remain Coraza-only until a formal semantic design exists.

Do not increase coverage by weakening the zero-false-negative rule.

Implementation progress (2026-09-14):

- Slice A: conservative capability matrix/report foundation.
- Slice B: CRS single-rule metadata parser/analyzer foundation.
- Slice C: deterministic CRS ruleset ingestion, Include/IncludeOptional expansion, duplicate-ID inventory, Coverage Report v2, and `wafctl coverage analyze`.
- Slice C also narrows analyzer eligibility to the exact **current runtime** VectorScan classifier semantics; the analyzer does not claim broader coverage than the dataplane actually supports.
- No Slice A-C analyzer result is a promotion signal by itself. Real Phase 1 differential evidence and zero observed false negatives are still mandatory before ACCELERATED.

## Phase 3 — Release engineering and supply-chain hardening

**Status: IMPLEMENTED_TESTING_DEFERRED.**

Implemented in source on 2026-09-14:

- `release-security-scan.sh` records `govulncheck` as PASS/FAIL/BLOCKED/NOT_RUN without fabricating evidence; `required` mode can make scan availability a release gate.
- `tools/release_evidence.py` generates deterministic SPDX 2.3 and CycloneDX 1.5 SBOMs from the pinned Go module graph plus release provenance.
- release evidence records Go directive/runtime, Coraza pin, CRS tree digest when supplied, VectorScan/libhs provenance, NGINX and OpenSSL versions.
- `build-release-artifact.sh` supports `SOURCE_DATE_EPOCH` / `--source-date-epoch`, truthful `SOURCE_ARCHIVE` identity, generated source manifests/SBOM/provenance evidence, and optional detached minisign signing. Binary portable/native identity is reserved for actual outputs of `build.sh`.
- `build.sh` records the realized binary build variant plus binary SHA-256 provenance in `BUILD_PROVENANCE.json` / `BUILD_SHA256SUMS.txt`.
- `verify-release-artifact.sh` rejects duplicate/encrypted/unsafe/symlink/special/oversized/group-writable/world-writable entries and secret/private-key-like filenames; verifies source bytes/modes, complete source/release manifests, source-bound evidence, provenance cross-digests, and SBOM dependency parity with `go.mod`.
- `release-artifact-negative-tests.sh` automates malicious/corrupt archive rejection for traversal, secret files, symlinks, duplicate entries, executable-mode loss and installer replacement.
- `verify-reproducible-source-release.sh` rebuilds the same source release twice with a fixed epoch and requires byte-identical ZIP SHA-256.
- organizational release signing remains **NOT_CONFIGURED** unless an approved minisign key/process is explicitly supplied; no signing key is bundled or generated by the project.
- Phase 3 Truth-Boundary Repair consolidates coverage/runtime eligibility in `internal/capability`, aligns `IncludeOptional` handling, binds govulncheck evidence to the source manifest, removes source-as-binary identity, adds manifest-completeness checks, and adds detached signature verification commands.
- Unsigned release evidence is explicitly integrity-bound but **not authenticated**; authenticity requires detached signature verification with an approved public key.

Qualification boundary: implementation of release tooling does not turn an unavailable `govulncheck`, real Coraza, real VectorScan, production CRS, kTLS, QAT, install/upgrade or target-host gate into PASS evidence.

## Phase 4 — Existing security/product backlog

**Status: IMPLEMENTED_TESTING_DEFERRED.**

The bullets below are the original backlog description. Their source foundations were subsequently implemented in the Phase 4 Slice A-F entries later in this roadmap; production/live qualification remains deferred.

The older backlog remains valid unless the owner reprioritizes it:

- L7 abuse controls using the already normalized trusted client IP: per-IP rate limits, connection caps, TLS-handshake caps;
- manual CIDR allow/deny lists with allow-over-deny precedence and optional TTL;
- custom block page + request correlation ID carried through match/access/syslog evidence;
- persistent security state for AI blocklist, learner/security aggregates, notification/session/audit state;
- PKI Slice 3: SSRF-safe CRL URL retrieval, refresh scheduling/deduplication, last-known-good retention, PKI status/manual refresh and related audit/RBAC;
- dependency vulnerability scanning and deployed Linux validation for remaining PKI paths.

## Deferred architecture work

- XDP prefilter: DEFERRED. It is complementary to VectorScan, not an alternative currently under selection.
- AF_XDP / DPDK / SmartNIC/DPU / GPU: DEFERRED unless a future throughput target justifies the architectural cost.
- Deep Coraza fork: do not do this. Keep Coraza authoritative and VectorScan optional/fallback-safe.
- QAT hardware-specific optimization: only qualify on actual supported hardware; software/kTLS/QAT frontend modes must remain optional.

## Immediate next development instruction

**Priority 0 is root build integrity, not another feature slice.** Apply the
prepared repair to a branch based on current/audited `main`, run the exact Go
1.25 CI tidy/build/vet/test/race/real-Coraza sequence, and merge only through a
PR with required CI. After root buildability is proven, build real DEB/RPM
artifacts with `./waf-package`, then execute package lifecycle and clean-host
qualification. Do not start a new feature phase while the root build gate is
BLOCKED.

## Cross-cutting mandatory gate — Artifact Packaging Integrity

**Implementation state: TESTED; gate is REQUIRED FOR EVERY FUTURE RELEASE.**

The release artifact itself is a production surface. Source tests do not prove that a ZIP, installer, or deployment bundle contains the validated files.

Every future release must therefore run the post-packaging gate documented in `RELEASE_PROCESS.md` and `TESTING.md`. For WAF source ZIPs:

1. create the ZIP with `build-release-artifact.sh` or an equivalent controlled process;
2. generate `RELEASE_MANIFEST.txt` inside the artifact;
3. extract into a clean temporary directory;
4. reject CRC errors, unsafe paths and symlinks;
5. validate required files, critical script sizes, shebangs, executable modes and shell syntax;
6. compare the complete packaged source set against the validated source tree by SHA-256 and Unix mode;
7. independently verify every manifest hash;
8. run feasible artifact-level smoke checks;
9. keep any unavailable real Coraza/VectorScan/CRS/QAT/kTLS gates explicitly `NOT_RUN`.

No artifact is releasable solely because source-tree tests passed. This gate is mandatory and is independent of Phase 0 real release-host qualification.

## Debug & Evidence Capture / Supportability Slice

**Status: IMPLEMENTED_TESTING_DEFERRED.**

Implemented in source:
- bounded, opt-in per-site/tenant debug capture with an atomic disabled fast path;
- server-generated transaction/request IDs and exact request-context correlation;
- request/response/TLS/proxy-LB metadata plus transaction-final Coraza `MatchedRules()` evidence;
- VectorScan candidate/eligible-match/false-negative evidence when an observation exists; absence of a VectorScan observation is recorded as `observed:false`, never as zero-FN evidence;
- TTL cleanup, entry limits, exact-tenant lookup/export, sensitive-field masking and body-free default metadata capture;
- admin debug APIs and `wafctl debug capture|stop|list|export`;
- `wafctl doctor` backed by runtime/dependency/TLS/VectorScan status;
- `wafctl support bundle` with sanitized config, logs, metrics, debug status, dependency evidence, SPDX-formatted SBOM evidence, manifest and SHA-256 list;
- install/upgrade/uninstall/doctor integration for `/usr/local/bin/wafctl`.

Still qualification-required:
- full repository Go regression on Go 1.25;
- live production debug-capture/privacy validation;
- real release-host Phase 1 differential corpus run;
- operational smoke of `wafctl` against a running installed appliance.

## Phase 1 runner implementation

`run-phase1-qualification.sh` and `cmd/wafqualify` now execute the mandatory real differential gate on a qualified host. They use real Coraza DetectionOnly transactions and transaction-final `MatchedRules()`, the same native VectorScan plan/scanner path, and compare only the deliberately eligible rule set. PASS requires zero observed false negatives and a configurable minimum of eligible Coraza match evidence. Missing Go/libhs/rules/eligible evidence returns BLOCKED/NOT_RUN; any native scan error or false negative returns FAIL.


## Phase 2 implementation update (2026-09-13)

Implemented qualification corpus schema foundation and differential false-negative gate helpers. Full runtime qualification remains deferred until real Go 1.25, Coraza v3.7.0 execution and libvectorscan are available.


## Phase 2 Slice B continuation

Implemented evidence operations, retention policy foundation, support provenance/SBOM evidence models, and VectorScan audit store. Full regression and real qualification remain deferred.


## Phase 2 Coverage Expansion Slice A

Implemented source slice:
- conservative VectorScan coverage capability matrix
- rule eligibility evaluation
- transform capability registry
- coverage report model
- unsupported scopes remain Coraza-only

Status: IMPLEMENTED_TESTING_DEFERRED


## Phase 4 Slice A Trusted Client Identity Foundation
Status: IMPLEMENTED_TESTING_DEFERRED



Phase 4 Slice A follow-up: added client identity audit event model constants CLIENT_IDENTITY_RESOLVED and CLIENT_IDENTITY_HEADER_REJECTED.

---

# Phase 4 Slice B — L7 Abuse Controls

Status: IMPLEMENTED_TESTING_DEFERRED

Implemented:
- L7 abuse control boundary
- trusted client identity reuse
- continuous token bucket keyed by site + trusted identity
- 64 sharded state maps and per-entry locking; no global limiter mutex
- hard per-shard state bound with amortized idle pruning/eviction
- panic-safe deferred active-request release
- concurrent request tracking and 429 response boundary

Deferred:
- per-site/per-PagePolicy configured limit overrides
- TLS handshake-rate enforcement
- production tuning and traffic/load qualification

---

# Phase 4 Slice C — Manual CIDR Policy

Status: PLANNED

Scope:
- CIDR allow list
- CIDR deny list
- priority evaluation
- policy evidence
- optional TTL lifecycle

Boundary:
- consumes trusted client identity from Slice A
- does not replace Coraza authority
- does not introduce distributed policy storage


## Slice C Update
Manual CIDR Policy Engine implementation completed. Status: IMPLEMENTED_TESTING_DEFERRED.

## Phase 4 Slice D — Custom Block Page / Correlation
Status: IMPLEMENTED_TESTING_DEFERRED
Completed implementation foundation:
- request correlation ID
- HTML/JSON block response abstraction
- security event model foundation


## Phase 4 Slice E — Persistent Security State

Status: IMPLEMENTED_TESTING_DEFERRED

Implemented foundation: security state models and in-memory persistence abstraction. Production database durability, HA replication, retention tuning, and external integrations remain deferred.


## Phase 4 Slice F — PKI Slice 3 Hardening
Status: IMPLEMENTED_TESTING_DEFERRED

Implemented foundation: CRL retrieval validation, refresh state model, and last-known-good retention boundary.


## Phase 5 Slice A — Runtime Qualification Closure

Status: IMPLEMENTED_TESTING_DEFERRED

Implemented: runtime qualification evidence schema foundation. Real Go 1.25, Coraza v3.7.0 transaction, and libvectorscan runtime gates remain NOT_RUN until executed on a qualified release host.


## Phase 5 Slice B — VectorScan Production Qualification

Status: IMPLEMENTED_TESTING_DEFERRED

Implemented: qualification evidence model, Coraza vs VectorScan differential gate foundation, and zero false-negative failsafe boundary. Real CRS corpus replay, real libvectorscan runtime execution, and production qualification remain NOT_RUN.


## Phase 5 Slice C — Enterprise Deployment Readiness (IMPLEMENTED_TESTING_DEFERRED)

Added deployment readiness evidence foundation:
- preflight report model
- health/readiness evidence boundary
- deployment diagnostics foundation

Runtime deployment qualification remains NOT_RUN until executed on target environments.


## Phase 5 Slice D — Security Operations Experience

Status: PLANNED / NOT IMPLEMENTED

The earlier repository contained model-only placeholder files for Security Event Timeline, Investigation Search, Change Audit, Security Evidence Export, and a second debug lifecycle/retention design. They had no runtime store/API/Console wiring and were removed during API-8 post-audit hardening rather than being misrepresented as implemented functionality. Existing real debug capture/evidence/export remains implemented through `debug_bundle.go`, `support_api.go`, `wafctl`, and the SYSTEM Console.


## Phase 5 Slice E — Reliability Qualification

Status: IMPLEMENTED_TESTING_DEFERRED

Implemented reliability qualification foundation:
- failure scenario evidence model
- recovery qualification boundary
- dependency/resource failure scenario ledger

Real failure injection and production reliability qualification remain NOT_RUN.

## Phase 5 Slice F — Performance Certification

**Status: IMPLEMENTED_TESTING_DEFERRED.**

Implemented in source on 2026-09-16:

- extended the existing `cmd/wafbench` harness instead of creating a second benchmark stack;
- added `wafbench certify` with hash-bound evidence inputs for `reverse_proxy_baseline`, `coraza_crs`, and optional `vectorscan_assisted` full-proxy runs;
- added system/run-shape comparability checks covering OS/arch/CPU/Go identity, host identity when recorded, worker/concurrency count, GOMAXPROCS, and average request/response workload size;
- added target-aware certification using explicit `waf-proxy-performance-target-v1` SLO files. Missing target keeps certification `NOT_RUN` even if benchmark inputs are complete;
- added RPS, application throughput (Mbps/Gbps), p50/p95/p99, process CPU, CPU-us/request, RSS, softirq and optional network Gbps evidence snapshots;
- added load-generator saturation warnings so generator-limited results are not treated as WAF ceilings;
- added optional prior-report regression deltas for RPS and p99 without inventing an implicit pass/fail threshold;
- VectorScan-assisted certification is blocked unless a real Phase 1 `wafqualify` report is supplied with `result=PASS`, `zero_false_negatives=true`, and `false_negative_count=0`;
- added `qualification/performance/` operator documentation, target example, and explicit `NOT_RUN` placeholder evidence.

Truth boundary:

- no end-to-end performance benchmark was executed on this packaging host;
- no production sizing target was approved or evaluated;
- no Gbps/RPS number from this slice is a production sizing claim;
- real Performance Certification remains `NOT_RUN` until the same artifact is measured on a qualified host and an approved target is supplied;
- VectorScan remains optional acceleration and Coraza remains authoritative.

Source-baseline hygiene repair performed while entering Slice F: the supplied Slice E ZIP contained a stale nested `waf-work/` source mirror. Five Phase 5 Slice B `qualification/vectorscan` files existed only in that mirror, and `replay.go` had a mismatched package declaration. Those files were restored to the canonical source path, the package mismatch was repaired, and the stale duplicate source tree was removed. This repair changes no dataplane decision semantics.

## Phase 5 Slice G — External HSM / PKCS#11 Support

**Status: IMPLEMENTED_TESTING_DEFERRED.**

Implemented in source on 2026-09-16:

- `internal/hsm` provider abstraction with an explicit native `pkcs11` build tag and a fail-closed non-native stub;
- independent `WAF_HSM_PKCS11=off|auto|required` build policy so HSM support does not depend on enabling VectorScan;
- configurable approved module directories plus exact module path, slot/token, key label and/or CKA_ID selectors;
- Linux module hardening: absolute clean path, regular non-symlink file, root-owned approved path, and no group/world write permission;
- PKCS#11 module/session/login/private-key lifecycle with exact one-key lookup and bounded session/module reference management;
- Go TLS `crypto.Signer` integration for RSA PKCS#1 v1.5, RSA-PSS and ECDSA signing paths;
- certificate/public-key association proof by challenge signature before runtime swap;
- PIN secret references limited to `env:NAME` or protected `file:/absolute/path`; inline PINs are rejected and resolved byte copies are zeroed after login;
- admin/API secret-reference redaction, including both GET and config PUT responses;
- provider/slot/token/key health model and `/api/hsm/status` runtime endpoint;
- fail-closed TLS signing behavior and explicit rejection of filesystem-key fallback or external TLS frontend fallback;
- HSM audit events restricted to `provider`, `slot`, `key_reference`, `operation`, and `result`, with matching syslog export;
- mock/native-fixture tests for signer/TLS association, session/login/key lookup, secret handling, audit fields, and signing failure;
- `cmd/hsmqualify` plus separate SoftHSM and real-vendor qualification runners/evidence classes;
- checked-in SoftHSM and real-vendor evidence placeholders remain `NOT_RUN` until those environments are actually exercised.

Truth boundary:

- isolated HSM package tests on the packaging host are source-level/mock-native evidence only;
- SoftHSM tooling/module is not installed on this packaging host, so SoftHSM qualification remains `NOT_RUN`;
- real vendor HSM qualification remains `NOT_RUN` until the exact vendor module/token/key/login/failover/TLS path is executed on real production-class infrastructure;
- no mock or SoftHSM result may upgrade the real-vendor gate;
- canonical repository-wide Go 1.25 validation remains separate from the isolated local Go 1.23 HSM harness.

## Enterprise Linux Distribution Packaging

This stage follows Phase 5 and formalizes Linux package delivery. It is a deployment/distribution stage, not a new WAF detection phase.

### Slice A — Debian / Ubuntu DEB Packaging

**Status: IMPLEMENTED_TESTING_DEFERRED.**

Implemented on 2026-09-16:

- deterministic binary `.deb` builder under `packaging/deb/`;
- formal `/usr/bin` and systemd package layout;
- dpkg conffile preservation for WAF configuration;
- one-time admin-token creation with upgrade preservation and no secret value in package-manager output;
- persistent `/var/lib/waf-proxy` systemd `StateDirectory`;
- network-free maintainer scripts and explicit CRS provisioning boundary;
- package verifier and reproducible fixture-package test;
- native VectorScan packages fail closed unless the target distribution runtime dependency is explicitly supplied.

Truth boundary: the local host cannot build the canonical Go 1.25 binaries, so a production `.deb` from the real WAF binaries is BLOCKED here. Fixture `.deb` PASS validates package mechanics only. Clean Debian/Ubuntu install qualification remains NOT_RUN.

### Slice B — RHEL-family RPM Packaging

**Status: IMPLEMENTED_TESTING_DEFERRED.**

Implemented on 2026-09-16 for RHEL, Rocky Linux, AlmaLinux, and Oracle Linux:

- provenance/checksum-bound binary RPM builder under `packaging/rpm/`;
- real `waf-proxy.spec` with standard `/usr`/`/etc`/`/var`/systemd paths;
- `%config(noreplace)` for operator configuration and explicit `.rpmnew` preservation semantics;
- one-time break-glass admin secret creation with no upgrade regeneration or package-manager secret output;
- persistent `/var/lib/waf-proxy` and `/var/log/waf` ownership/state model;
- fresh-install no-autostart plus controlled `try-restart` only for services already active on upgrade;
- offline-safe scriptlets with no CRS/package/network fetch;
- SELinux-aware policy boundary: no `setenforce`, `audit2allow`, `semanage`, or ad-hoc policy module injection from RPM scriptlets;
- exact x86_64/aarch64 ELF-to-RPM architecture binding;
- deterministic payload ordering/build-time controls and source-level package verifier;
- native VectorScan package builds fail closed unless the exact target-distribution runtime dependency is supplied explicitly.

Truth boundary: this Debian packaging host lacks `rpmbuild`, `rpm`, and `rpm2cpio`, so an actual fixture RPM and RPM-byte reproducibility gate are BLOCKED here. The production Go 1.25 RPM is also BLOCKED by the unavailable toolchain. Clean RHEL-family and SELinux-enforcing qualification remain NOT_RUN.

### Slice C — Package Upgrade / Rollback Qualification

**Status: IMPLEMENTED_TESTING_DEFERRED.**

Implemented on 2026-09-16:

- one offline-safe lifecycle qualification runner for both DEB and RPM package managers;
- destructive execution is opt-in, root-only and guarded by an exact dedicated-host acknowledgement;
- SHA-256/native-metadata binding of Version N, Version N+1 and an intentional post-install-failure package;
- operator-modified config, generated admin secret and `/var/lib/waf-proxy` state preservation checks across upgrade, failed upgrade, recovery and rollback;
- service active/inactive-state preservation checks;
- Debian `--force-confold` / `.dpkg-dist` and RPM `%config(noreplace)` / `.rpmnew` conflict evidence when packaged defaults actually change; `.dpkg-old` / `.rpmsave` are captured when emitted but never fabricated;
- deterministic lifecycle fixture builder using `/bin/true` payloads strictly for package semantics, not WAF runtime evidence;
- qualification-only RPM `%post` failure macro compiled only into explicit `qualification` fixture versions; normal RPM builds do not contain the failure branch;
- separate DEB/RPM `NOT_RUN` evidence placeholders and an operator guide.

Executed here: lifecycle unit/source tests PASS; deterministic three-package DEB fixture build and preflight PASS; real DEB lifecycle transaction NOT_RUN. RPM source path PASS, while RPM fixture/lifecycle execution is BLOCKED/NOT_RUN because this Debian host lacks the RPM toolchain. Clean-host distribution acceptance remains Slice D.

### Slice D — Clean-host Distribution Qualification

**Status: IMPLEMENTED_TESTING_DEFERRED.**

Implemented on 2026-09-16:

- dedicated-host, root-only, explicit-ack qualification runner under `packaging/cleanhost/`;
- exact distro/version matrix for Debian 12, Ubuntu 22.04/24.04, RHEL 9, Rocky 9, AlmaLinux 9 and Oracle Linux 9;
- RHEL-family acceptance requires SELinux `Enforcing`;
- strict clean-host preconditions reject a pre-existing package, `/etc/waf`, or `/var/lib/waf-proxy`;
- local package SHA/native metadata and local CRS tree digest binding;
- offline native package install/upgrade/remove only (`dpkg` / `rpm`), with no apt/dnf/yum/curl/wget/git dependency or CRS fetch;
- fresh-install no-autostart verification, explicit local CRS provisioning, `waf-doctor --check`, explicit systemd enable/start, `/healthz`, break-glass first-login API and real loopback reverse-proxy traffic smoke;
- upgrade preservation checks for admin secret, operator config and `/var/lib/waf-proxy` state plus repeated health/auth/traffic;
- non-purge removal verification for package/service removal while secret, CRS, operator config (or RPM `.rpmsave`) and persistent state remain preserved;
- checked-in per-platform `NOT_RUN` evidence and a matrix file.

Executed here: clean-host unit/source policy tests PASS; the current Debian 13 shared packaging host is correctly BLOCKED for the Debian 12 matrix and no real clean-host transaction is claimed. All seven supported platform acceptance runs remain `NOT_RUN` until executed on dedicated clean VMs/hosts with real qualified packages and local approved CRS content.

## Project-local package builder utility

Status: `IMPLEMENTED_TESTING_DEFERRED`.

`./waf-package` is the supported developer/release entry point for rebuilding the
project's DEB and RPM after source changes. It must continue to compose the
canonical build and format-specific package verifiers rather than creating an
independent build path. Remaining qualification is execution on a Go 1.25+
release host for a real DEB and on an RPM-capable Go 1.25+ host for a real RPM,
followed by the existing Slice C/D lifecycle and clean-host gates.

## Root Build Integrity Repair

Status: **IMPLEMENTED_TESTING_DEFERRED**

This is a release-integrity repair, not a new feature Slice. Promotion requires
a green Go 1.25 CI run for `go mod tidy -diff`, root build, vet, tests, race, and
real-Coraza gate on the exact repaired commit. Branch protection requiring CI
before `main` merge remains an operational prerequisite.

## OpenAI Connector Hardening

**Status: IMPLEMENTED_TESTING_DEFERRED.**

Implemented Responses API, strict Structured Outputs, secret references, and mock integration tests. Promotion requires the exact source bytes to pass the repository Go 1.25 root gates; live external-provider acceptance remains `NOT_RUN`.


## Future API Security Product Line

The API-aware WAF roadmap is documented in `API_SECURITY_ROADMAP.md` and `API_SECURITY_SLICE_IMPLEMENTATION_ROADMAP.md`. Current source status: API-1 through API-8 are `IMPLEMENTED_TESTING_DEFERRED`; no API-9 is defined. Go 1.25 qualification remains deferred to the applicable gate.


## API-1 Hardening
Status: IMPLEMENTED_TESTING_DEFERRED
- Operation fingerprinting
- Normalization confidence metadata
- ULID/date normalization hardening
- Regression coverage added

## API Security Current State — 2026-09-24

- API-1 — API Discovery + Operation Normalization: `IMPLEMENTED_TESTING_DEFERRED`.
- API-2 — Typed Schema Learning: `IMPLEMENTED_TESTING_DEFERRED`.
- API-3 — OpenAPI Contract Management: `IMPLEMENTED_TESTING_DEFERRED`.
- API-4 — Positive Schema Enforcement: `IMPLEMENTED_TESTING_DEFERRED`.
- API-5 — JWT + Identity-aware API Security: `IMPLEMENTED_TESTING_DEFERRED`.
- API-6.1 — Sequence Foundation: `IMPLEMENTED_TESTING_DEFERRED`.
- API-6.2 Workflow Learning: `IMPLEMENTED_TESTING_DEFERRED`.
- API-6.3 Sequence Anomaly Detection: `IMPLEMENTED_TESTING_DEFERRED`.
- API-6.4 and API-7.1 through API-7.4: `IMPLEMENTED_TESTING_DEFERRED`.
- API-8: `IMPLEMENTED_TESTING_DEFERRED`; implementation roadmap complete through API-8, with Go 1.25 production qualification still required.

API-3 source includes OpenAPI 3.0/3.1 parsing, local references, contract import/versioning, API-1 operation binding, version/schema comparison, and API-2 drift evidence. No enforcement is introduced. Go 1.25 build/test qualification is deferred to the final qualification gate.


## API-7.1 Object Locator Discovery checkpoint — 2026-09-24

API-7.1 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. It adds bounded `ObjectLocator` discovery from API-1 normalized path parameters, API-2 typed path/query/body evidence, matched API-3 OpenAPI contracts, and explicit Reviewer-gated operator INCLUDE/SUPPRESS configuration. Locators are keyed by normalized API-1 operation ID plus location/field; raw object values are transient only and durable value evidence is bounded HMAC-SHA256 fingerprint data protected by a separate mode-0600 key. Client-supplied ownership/tenant headers are not authority, and GraphQL `variables.*` discovery is deferred to API-8.

API-7.1 has no identity/object relationship verdict, tenant-boundary verdict, `BOLA_CANDIDATE`, ownership verdict, BLOCK/DENY/403 path, or ENFORCE authority. OpenAI is absent from the locator authority path. State is bounded by global/per-operation/fingerprint/override caps, 30-day learned TTL, versioned restart validation, existing non-blocking observation-plane ingestion, immediate mutation persistence, RBAC and audit.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83 PASS** from API-1/2 through API-7.1; API-7.1 has **9 targeted test functions present**; OpenAI source contract **16/16 PASS** plus isolated tests PASS; package-source PASS; root Go shape **107 files PASS**; package-builder **9/9 PASS**; changed-file gofmt, Admin JavaScript and primary shell syntax PASS. Canonical Go 1.25 targeted/race remains `BLOCKED_ENVIRONMENT/NOT_RUN` because the host has Go 1.23.2 and external toolchain retrieval is unavailable. Next slice: **API-7.2 Identity/Object Relationship**. API-7.3/API-7.4 is `IMPLEMENTED_TESTING_DEFERRED`; API-8 remains `PLANNED`.


## API-7.3 BOLA Detection checkpoint — 2026-09-24

API-7.3 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. It adds bounded detection-only candidate evidence for identity/object divergence, verified-tenant/object divergence and recent per-locator object enumeration. It consumes only API-5 verified identity pseudonyms, API-7.1 keyed object evidence and API-7.2 relationship history. Candidate state is capped, TTL-bounded, restart-validated and read-only to Reviewer-or-higher operators. No ownership verdict or request blocking authority is introduced. Source gate 110 PASS; nine targeted Go tests are present; canonical Go 1.25 targeted/race is `BLOCKED_ENVIRONMENT/NOT_RUN`. Next: API-7.4 BOLA Policy/Evidence/Console.

## API-7.4 BOLA Policy/Evidence/Console checkpoint — 2026-09-24

API-7.4 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. It completes the API-7 BOLA implementation track with a bounded evidence-handling policy plane and operator workflow. Policies are scoped only by normalized API-1 operation ID, optional API-7.1 locator ID, optional API-7.3 candidate type, and minimum confidence. The only actions are `REVIEW` and `SUPPRESS`; suppression never deletes the underlying API-7.3 candidate. Operator evidence workflow is `OPEN` / `ACKNOWLEDGED` / `DISMISSED` / `RESOLVED` with enumerated reason codes only. New detector evidence after dismissal/resolution reopens the evidence automatically.

API-7.4 is deliberately absent from the API-7.2 relationship processor and API-7.3 detector authority path. It accepts no raw identity/object/tenant/client selectors, arbitrary headers, cookies, Authorization data, or free-text review notes. There is no ownership verdict, request-path `BLOCK`, `DENY`, `403`, or `ENFORCE` authority, and OpenAI is absent from the API-7.4 authority path. State is versioned/revalidated on restart, mutex-protected, TTL/cardinality bounded (1,024 policies; 4,096 reviews; 30-day default policy/review TTL; 180-day policy maximum), immediately persisted on mutation, and included in API security autosave/final flush. Reviewer-only policy/evidence APIs are audited and the embedded console exposes effective policy, evidence workflow, suppression, and reopen status.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83/100/110/156 PASS** from API-1/2 through API-7.4; **10 API-7.4 targeted Go test functions are present**; OpenAI source contract **16/16 PASS** plus isolated package tests PASS; package-source PASS; root Go source shape **113 files PASS**; package-builder **9/9 PASS**; `gofmt`, Admin embedded JavaScript and primary shell/Python syntax PASS. Canonical Go 1.25 `go mod tidy -diff`, root build/test, API-7.4 targeted tests and API-7.4 race tests remain `BLOCKED_ENVIRONMENT/NOT_RUN` because this host has Go 1.23.2 and external toolchain retrieval is unavailable. API-8 GraphQL Security is the next `PLANNED` slice.

## API-8 GraphQL Security checkpoint — 2026-09-24

API-8 is implemented across parsing/normalization, SDL contract binding, deterministic complexity/field/mutation/subscription/introspection policy, Apollo persisted-query support, verified GraphQL-variable integration into API-7 evidence, and Reviewer-gated `LEARN` / `DETECT` / `ENFORCE` policy plus Admin Console. Any policy edit or bound contract content change returns the policy to `LEARN`; a referenced schema contract cannot be deleted. Only explicit deterministic GraphQL policy in `ENFORCE` may reject traffic. API-6/API-7 learned or inferred evidence and OpenAI cannot promote, mutate or bypass GraphQL enforcement.

Durable GraphQL state excludes raw query documents, literals, raw variable values, Authorization/Cookie/JWT data and unverified claims. SDL is reduced to a digest plus normalized topology; GraphQL object values use API-7 keyed fingerprints and API-5 verified identity only. State and parser resources are bounded, versioned, restart-revalidated, autosaved/final-flushed, and strict mutation APIs are Reviewer-gated and audited.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83/100/110/156/259 PASS** through API-8; **22 targeted API-8 Go test functions are present**; OpenAI source contract **16/16 PASS** plus isolated tests PASS; package-source PASS; package-builder **9/9 PASS**; root Go source shape **115 files PASS**; changed API-8 Go files `gofmt`, Admin embedded JavaScript, relevant shell and Python syntax PASS. Canonical Go 1.25 `go mod tidy -diff`, root build/vet/test, API-8 targeted tests and race tests remain `BLOCKED_ENVIRONMENT/NOT_RUN` because the host has Go 1.23.2 and external toolchain retrieval is unavailable. API-8 remains `IMPLEMENTED_TESTING_DEFERRED`, not `TESTED` or `RELEASED`.

The documented API-security implementation roadmap is now complete through API-8. No API-9 is defined. Next gate: canonical Go 1.25 qualification of the exact source/artifact, followed by release promotion only if all required gates pass.

## 2026-09-25 — Production Correctness & Control-Plane Hardening

Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.

## 2026-09-28 — Maintenance consolidation checkpoint

Production Correctness & Control-Plane Hardening remains the functional
baseline; Code Duplication Review and Consolidation is a source-maintenance
checkpoint on top of it. It removes superseded/unwired implementations and
centralizes shared parsing without adding a product slice. API-1 through API-8
remain `IMPLEMENTED_TESTING_DEFERRED`; no API-9 is defined.

## 2026-09-30 — Build and Admin Startup Blocker Fix

The current source fixes two release-blocking defects found during external Go 1.25 review of the 2026-09-28 consolidation baseline. `admin.go` now imports the standard-library `errors` package required by the existing `errors.New(...)` validation paths. Positive-Schema exception state changes now use the unambiguous route `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`; the shipping Console uses the same path. The prior `POST /api/security/schema/enforcement/exceptions/{exception_id}` registration is removed because it conflicts with `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+ `http.ServeMux` specificity rules and can panic during Admin handler construction.

`admin_route_registration_test.go` now makes full Admin handler registration a regression test, and `tools/tests/test-build-startup-blocker-fix-source.py` is wired into `build.sh` and CI. The source gate also registers the complete 139-route Admin/update inventory with the Go standard library in a standalone temporary program; this catches wildcard-pattern ambiguity even when the repository module graph cannot be compiled locally. Current source evidence: blocker-fix gate **13/13 PASS**, route inventory **139/139 accepted by `ServeMux`**, API/source gates through API-8/post-audit/production-hardening/duplication remain PASS, OpenAI source contract and isolated tests PASS, package-source PASS, package-builder **9/9 PASS**, root Go source shape **96 files PASS**, Admin embedded JavaScript PASS, and changed Go files are `gofmt` clean.

This does **not** promote the source to `TESTED` or `RELEASED`. On the current execution host, canonical `GOTOOLCHAIN=local` Go 1.25 `go mod tidy -diff`, root build, vet, targeted Admin/API64 tests, and race remain `BLOCKED_ENVIRONMENT / NOT_RUN` because only Go 1.23.2 is installed and external Go toolchain/module retrieval is unavailable. Packaging/source-manifest integrity is not a substitute for those root compilation gates. No API-security authority changes are introduced: API-6/API-7 remain non-enforcing, API-8 retains explicit deterministic GraphQL ENFORCE authority, and OpenAI remains advisory only.
## 2026-09-30 — OWI-1.0 R3 Dashboard Connector

Status remains **IMPLEMENTED_TESTING_DEFERRED**. This is a product integration slice, **not API-9**, and it does not change API-1 through API-8 enforcement authority. The WAF now exposes a separate read-only management reader at `/api/integrations/dashboard/v1` for required ASSET, DETECTION, POLICY and HEALTH lanes using a dedicated digest-only opaque bearer identity. Optional EVENT/ACTION_STATUS remain disabled and `actions=[]`.

R3 controls implemented in source include per-tenant+principal rate/concurrency plus a global ceiling, strict GET-only handling including HEAD rejection, an 8s default request deadline propagated into source sync and context-aware store-lock acquisition, hard snapshot/export/response bounds, durable cursor/snapshot/revision state, non-blocking WAF security export with explicit GAP semantics, fail-closed durability after write errors, opaque-token expiry/rotation/revoke, owner-only non-symlink token/cursor-secret files, TLS >=1.2 on the built-in reader listener, tenant/scope-bound cursors, and allow-listed sanitization. No Dashboard registry/validator/UI code is modified.

Supporting exact connector-source evidence on the available Go 1.23.2 host is **25 PASS / 2 intentional SKIP**, with the same set passing under `-race`; real subprocess SIGKILL→restart/GAP, store contention deadline/retry, injected write failure/recovery, client cancellation, oversize failure, scope reduction, and accelerated retention expiry are covered. The R3 source gate is **38/38 PASS**. The regenerated product return validates **61 files, 18 positive responses, 15 negative cases and 11 transport cases**. Integrated repository Go 1.25 build/vet/test/race remains `BLOCKED_ENVIRONMENT / NOT_RUN`; G2 Dashboard offline acceptance and G3 live TLS/ACL/intended-runtime qualification remain `NOT_RUN`.

<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
