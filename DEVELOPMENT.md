# Development Engineering Ledger

The detailed feature/status roadmap remains in `DEVELOPMENT_ROADMAP.md`; `AI_HANDOFF.md` remains the continuation ledger. This file records cross-cutting engineering rules that apply to every development slice.

## Artifact Packaging Integrity Gate

Every development stage that produces a delivery ZIP, installer, package, or deployment bundle must treat the generated artifact as part of the production surface.

After source tests and before release, the artifact must be extracted into a clean location and independently inspected. At minimum validate archive integrity/path safety, required files, critical-file size, executable/script headers, syntax, source-to-package SHA-256 equality, Unix file modes, package manifest contents, and feasible artifact-level smoke checks.

Use `build-release-artifact.sh` for new WAF source ZIPs and `verify-release-artifact.sh` for independent post-build verification. The complete policy and mini-SIEM motivating incident are documented in `RELEASE_PROCESS.md`.

A source-tree PASS is not evidence that a delivery artifact contains the tested source. Artifact verification is a mandatory release gate, separate from real Coraza/VectorScan target-host qualification.

## 2026-09-10 Debug Evidence / VectorScan Qualification Slice

Implemented source baseline additions:

- Added bounded Debug Evidence Capture foundation.
- Capture is independent from the WAF verdict path; capture failure cannot change Coraza decisions.
- Added bounded transaction evidence export foundation.
- Added qualification test coverage for evidence lifecycle.
- Added Phase 1 qualification tracking for VectorScan candidate coverage, false-negative accounting, FAILSAFE transitions, fingerprint invalidation, restart persistence and HTTP/2 transaction correlation requirements.

Real Coraza/libvectorscan execution remains subject to release-host qualification gates and is not promoted from local/stub evidence.

## 2026-09-10 Debug Evidence integration / Phase 1 safety gate

Implemented:
- Coraza transaction-final MatchedRules evidence capture hook after ProcessLogging.
- VectorScan candidate-vs-Coraza qualification helper with zero false-negative gate.
- Debug evidence masking foundation and bounded export support.

Truth boundary:
- Real Coraza execution remains NOT_RUN until qualified host execution.
- Real libvectorscan hs_compile_multi/hs_scan remains NOT_RUN.

## 2026-09-10 Debug Evidence Supportability Slice

Implemented:
- Debug bundle evidence schema foundation for request/response/TLS/proxy/Coraza evidence.
- Tenant-scoped evidence storage foundation with TTL cleanup.
- VectorScan differential qualification helper enforcing Coraza match coverage.

Truth boundary:
- Real Coraza v3.7 execution: NOT_RUN.
- Real libvectorscan execution: NOT_RUN.
- Production CRS Learning qualification: NOT_RUN.

## 2026-09-13 — Debug Evidence, Operator CLI and Phase 1 Production Qualification Runner

This slice completes the source-level supportability path without changing the authoritative WAF decision boundary.

Implemented:

- Correlated debug evidence now follows one server-generated transaction ID through request metadata, Coraza transaction-final `MatchedRules()` after `ProcessLogging()`, VectorScan candidates/false-negative comparison, load-balancer/upstream selection, TLS metadata and response status/latency.
- Coraza evidence is captured even when VectorScan is disabled or no eligible plan exists. In that case VectorScan evidence is explicitly `observed:false`; it is never treated as zero-false-negative proof.
- Debug capture is opt-in and site/tenant scoped, bounded by count and TTL, uses an atomic disabled fast path, omits request/response bodies by default, allow-lists request headers and masks sensitive fields before storage/export.
- Added admin endpoints for doctor/debug status/capture/evidence/export and the `wafctl` operator CLI (`doctor`, `debug capture|stop|list|export`, `support bundle`).
- Support bundles contain sanitized API/config/status evidence, optional incident evidence, bounded logs, dependency inventory, SPDX-formatted SBOM evidence, a manifest and SHA-256 list. Bundle creation rejects obvious private-key/admin-token material.
- Added `cmd/wafqualify`, `run-phase1-qualification.sh` and a representative JSONL corpus. The runner requires real libhs plus Go >=1.25, builds the real native VectorScan path, runs real Coraza DetectionOnly transactions and enforces zero observed false negatives over the current eligible rule set.
- Build/install/upgrade/uninstall/doctor/release-artifact scripts now include `wafctl`; generated binaries and qualification output are excluded from source ZIPs.

Truth boundary: source implementation of a real qualification runner is not qualification evidence. This host remains BLOCKED by Go 1.23.2 and absent `pkg-config libhs`; real Coraza/VectorScan/CRS results remain NOT_RUN.


## Phase 2 implementation update (2026-09-13)

Implemented qualification corpus schema foundation and differential false-negative gate helpers. Full runtime qualification remains deferred until real Go 1.25, Coraza v3.7.0 execution and libvectorscan are available.

## Phase 2 Slice B - implementation

Implemented (testing deferred):
- Debug capture lifecycle v2 session model foundation with state tracking and expiry cleanup primitives.
- wafctl qualification report command integration.
- VectorScan transition audit record model.

Validation remains deferred until the final Phase 2 validation stage.


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

## Phase 2 Coverage Expansion Slice B

Implemented CRS rule capability analyzer foundation. CRS metadata can now be normalized into the conservative VectorScan capability model. Analyzer output does not enable acceleration; promotion still requires validation evidence and zero false-negative qualification.

## 2026-09-14 — Phase 2 Coverage Expansion Slice C

Implemented CRS ruleset ingestion and coverage reporting without changing the WAF verdict path or VectorScan promotion state machine.

- Added deterministic CRS `.conf` ruleset loading from either a directory or a configuration entrypoint.
- `Include` and `IncludeOptional` are followed recursively with loop de-duplication, glob ordering, source-size bounds and mandatory-include failure semantics.
- Added normalized SecRule metadata for source file/line, id, phase, explicit operator, selectors, transforms, tags, severity, chain and negation.
- Added duplicate rule-ID detection and a versioned `coverage-inventory.json` model.
- Added Coverage Report v2 with total/eligible/Coraza-only/unsupported counts, duplicate IDs, warning count, coverage percentage and rejection-reason histogram.
- Added local operator command `wafctl coverage analyze --rules PATH [--report FILE] [--inventory FILE] [--json]`.
- Hardened the Slice A/B capability model to match the **current runtime classifier exactly**: explicit positive `@rx`, phase 1/2, exactly one reproducible selector, fixed-name `REQUEST_HEADERS:<name>` or supported scalar source, explicit `t:none`, and only `t:lowercase` after `t:none`. Bare `REQUEST_HEADERS`, `uppercase`, chains, negation, ARGS/body and multi-selector rules remain Coraza-only.
- Analyzer eligibility is inventory metadata only. It cannot move a group into LEARNING/VALIDATED/ACCELERATED and does not weaken the zero-false-negative gate.

Status: **IMPLEMENTED_TESTING_DEFERRED** until the final concentrated validation stage. Real CRS/Coraza/libvectorscan qualification remains NOT_RUN on this packaging host.

Slice C artifact preparation also corrected a baseline packaging defect: the supplied Slice B ZIP had executable release/install shell scripts stored as `0644`. The scripts required by the artifact verifier are restored to `0755`, and mode preservation is part of the final ZIP and patch reconstruction gate.

## 2026-09-14 — Phase 3 Release Engineering / Supply-Chain Hardening

Implemented the formal Phase 3 roadmap scope without changing the WAF verdict path:

- Added machine-readable release provenance plus SPDX 2.3 / CycloneDX 1.5 SBOM generation from the pinned Go module graph.
- Source ZIPs now carry source-release identity; actual portable/native binary identity is written only by `build.sh` after a successful binary build.
- Added a `govulncheck` evidence runner that preserves PASS / FAIL / BLOCKED / NOT_RUN truth; no missing toolchain or network access is promoted to PASS.
- Hardened complete-source ZIP generation with SOURCE_DATE_EPOCH support, deterministic metadata, source manifests, SBOM evidence and optional detached minisign signing. Signing is fail-closed when explicitly required and is otherwise NOT_CONFIGURED.
- Hardened artifact verification against duplicate/encrypted/unsafe paths, links/special files, archive expansion abuse, group/world-writable entries and secret/private-key-like filenames.
- Added automated negative artifact mutation tests and byte-reproducible double-build verification.

Status: **QUALIFICATION_REQUIRED**. Runtime/security qualification that needs Go >=1.25, real Coraza, real VectorScan, representative CRS or production host dependencies remains separate.

## 2026-09-14 — Phase 3 Truth-Boundary Repair

Implemented a repair-only slice after a Markdown↔code audit found release/provenance and coverage-classifier drift.

- Added `internal/capability` as the single eligibility classifier used by both offline coverage analysis and the live `internal/vectoraccel` parser. It owns phase defaults, positive `@rx`, non-empty pattern, chain/negation rejection, selector restrictions, explicit `t:none`, and optional lowercase semantics.
- Added runtime `IncludeOptional` handling and fail-closed mandatory Include behavior so coverage ingestion and runtime rule discovery no longer use different include semantics.
- Added parity regression tests covering previously divergent header punctuation, empty regex, implicit phase, chain-action detection, negation, multi-selector, unsupported transform, and `IncludeOptional`.
- Release evidence schema is now v2. Source archives are always `SOURCE_ARCHIVE`; portable/native binary identity is emitted only for actual binary build provenance.
- Added canonical `tools/source_manifest.py`; govulncheck evidence now binds to its source-manifest digest and includes tool identity/version, execution time, result digest, and raw result text. Invalid or source-mismatched supplied evidence fails packaging rather than degrading silently.
- Added `release-evidence/PROVENANCE.json` cross-digests and verifier checks for exact source/release manifest completeness plus SPDX/CycloneDX dependency parity with `go.mod`.
- Added detached minisign verification via `verify-release-signature.sh` and `wafctl release verify-signature`. Unsigned hash/provenance metadata is explicitly described as integrity-only, not producer authentication.
- `build.sh` now uses `go mod tidy -diff` so release builds fail on dependency-file drift instead of mutating `go.mod`/`go.sum`, and binary provenance records artifact type and native libhs runtime-linkage expectation.

Status: **QUALIFICATION_REQUIRED**. This repair does not create new real Coraza/libvectorscan/Go1.25 qualification evidence.

Truth-boundary validation found and repaired one additional release defect: importing the canonical Python source-manifest helper could create `tools/__pycache__`, changing the staging manifest, and plain `go version` could invoke Go's auto-toolchain download whose network error text contained nondeterministic ephemeral ports. Release source enumeration now excludes Python bytecode/cache artifacts, the builder suppresses bytecode generation, and release Go-version evidence forces `GOTOOLCHAIN=local`. Fixed-epoch reproducibility passed after these repairs.
