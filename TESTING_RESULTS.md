# Testing Results and Qualification Ledger

Handover checkpoint: 2026-09-07 (Asia/Taipei)

This file separates **real execution**, **stub/API integration**, **ABI-only native checks**, **component benchmarks**, and **NOT_RUN release gates**. Do not promote one evidence class into another.

## Final packaged baseline

- Source artifact: `waf-proxy-vectorscan-learning-coraza37-2026-09-04.zip`
- SHA-256: `64dc3034865d20aedaa38e10af5459da844ae6f6d92960e224223ec0ed359b22`
- Size: 342,234 bytes
- Patch artifact: `waf-proxy-vectorscan-learning-coraza37-2026-09-04.patch`
- SHA-256: `a24645a911caaa986c310509103093ba0cdc5e7bdcd4801251ccadd4e17a8398`
- Size: 77,830 bytes
- Patch baseline: TLS Acceleration / modern-NGINX release line

Previous packaging verification already completed:

- ZIP integrity test: PASS
- ZIP re-extract vs source: 106 files, byte-for-byte identical, file modes identical
- patch reconstruction from TLS/modern-NGINX baseline: byte-for-byte/file-mode identical to final source
- diff hygiene: 20 expected files changed for the VectorScan/Coraza phase; unrelated TLS/AI/LB files did not drift

## Real host evidence already obtained

### TLS frontend / modern NGINX

Host evidence previously recorded:

- NGINX 1.26.3
- OpenSSL 3.5.5
- modern syntax generated as `listen ... ssl;` + `http2 on;`
- `nginx -t`: PASS with no deprecated `listen ... http2` warning
- HTTPS -> NGINX -> private Unix socket -> WAF-side endpoint smoke: PASS
- curl negotiated HTTP/2: PASS
- forged Internet-side internal forwarding/private TLS headers were overwritten/cleared: PASS
- kTLS auto detection fell back to software TLS on the available host because kernel kTLS was not available
- QAT auto detection fell back to software TLS because `qatprovider` was unavailable

This proves frontend/fallback behavior on that host. It does **not** prove real QAT hardware acceleration or active kTLS.

## Portable Coraza API-stub regression

Purpose: validate project control flow, compile compatibility with the intended Coraza API surface, Go logic, and race behavior when the real module cannot be downloaded. It is **not** real Coraza correctness evidence.

Final VectorScan/Coraza integration round:

- full repository `go test`: PASS
- `go vet ./...`: PASS
- bounded `go test -race` groups: PASS
  - root package: PASS (~14.4 s in the recorded run)
  - `internal/vectoraccel`: PASS (~1.1 s)
  - `internal/tlsfront`: PASS (~1.17 s)
  - `internal/sigupdate`: PASS (~4.6 s)
  - `cmd/wafbench`: PASS (~1.15 s)

A single all-packages race invocation exceeded the execution runner timeout; no race detector report was produced. The bounded package runs are the recorded PASS evidence.

## Native libhs ABI-only gate

Purpose: compile/link/exercise the actual CGO adapter shape without claiming real VectorScan regex behavior.

Using a local ABI-compatible libhs test library:

- vectorscan build-tag compile: PASS
- vectorscan-tag tests: PASS
- vectorscan-tag `go vet`: PASS
- vectorscan root race: PASS
- scan/close lifecycle paths compiled and ran

This gate found and caused removal of an unsafe `cgo.Handle` integer -> `unsafe.Pointer` bridge. The final adapter uses C-owned context storage carrying `uintptr_t` handle data instead.

**Evidence boundary:** this does not prove `hs_compile_multi`/`hs_scan` correctness, pattern compatibility, performance, SIMD behavior, or real libvectorscan lifecycle.

## Coraza transaction-truth integration

Implemented and covered by project tests/stub API integration:

- Learning truth source is transaction-final `tx.MatchedRules()` after `ProcessLogging()`;
- ErrorCallback remains available for existing match logging/AI/syslog but is not a Learning truth source;
- same transaction/group rule IDs are de-duplicated before Learning counters are updated;
- request-context transaction correlation replaces client/URI correlation, avoiding HTTP/2 identical-request ambiguity;
- semantic fingerprints include rule/adapter/Coraza/native version inputs and force re-Learning on drift;
- false negative or native scan failure moves the group to `FAILSAFE` and Coraza-only processing;
- no-hit skipping is permitted only for an `ACCELERATED` group and verification samples still execute full Coraza.


## 2026-09-07 Phase 0 qualification-automation evidence

A focused release-host qualification slice was added without changing WAF dataplane behavior:

- added `qualify-release-host.sh --preflight|--core` with explicit PASS / FAIL / BLOCKED semantics;
- preflight checks the installed local Go version with `GOTOOLCHAIN=local` so Go's auto-toolchain feature cannot hide an underspecified release host;
- preflight checks exact Go 1.25.0 / Coraza v3.7.0 pins, CGO compiler, `pkg-config libhs`, and real VectorScan package/operator-attested provenance;
- added `TestNativeVectorScanCompileAndScanGate` (`vectorscan && cgo`) to semantically exercise the native multi-pattern compile and scan path and expected rule IDs;
- `build.sh` now reports Go <1.25 deterministically before any module/toolchain download attempt.

Actually executed in this packaging environment:

- host: Linux 6.18.35 x86_64;
- local Go toolchain: **go1.23.2**;
- `pkg-config`: 1.8.1, but `libhs` is **not present**;
- C compiler: GCC 14.2.0;
- NGINX 1.26.3 and OpenSSL 3.5.5 are present (this does not add new TLS evidence beyond the already-recorded real smoke);
- `bash -n` over every root release shell script plus `benchmark/build.sh`: **PASS** after the change;
- `./qualify-release-host.sh --preflight`: **BLOCKED**, exit 3, for Go 1.23.2 + missing real libhs/provenance — expected behavior;
- synthetic command-shim `--preflight` control-flow test: **PRECHECK_PASS** — validates the pass branch only and is explicitly **not** real Go/VectorScan evidence;
- `GOTOOLCHAIN=local WAF_VECTORSCAN=required ./build.sh`: rejected Go 1.23.2 immediately with the explicit minimum-version message and **did not attempt** a Go 1.25/toolchain network download — expected guard behavior.

Not executed here and still **NOT_RUN**: the new native semantic gate, real Coraza v3.7.0 execution, real `realcoraza` truth gate, real libvectorscan compile/scan/race, install/upgrade/uninstall/doctor clean-VM smoke, and production CRS Learning qualification.

## Real Coraza / real VectorScan release gates

The isolated packaging environment could not obtain/run the required real Go 1.25 + Coraza v3.7.0 + libvectorscan stack. These are therefore explicitly **NOT_RUN**:

| Gate | Status |
|---|---|
| Go 1.25 runtime with project source | NOT_RUN |
| real Coraza v3.7.0 full test suite/integration | NOT_RUN |
| `realcoraza` DetectionOnly + `nolog` `MatchedRules()` truth test | NOT_RUN |
| real libvectorscan `hs_compile_multi` | NOT_RUN |
| real libvectorscan `hs_scan` | NOT_RUN |
| real CRS eligible-rule compilation coverage | NOT_RUN |
| production CRS Learning Period false-negative qualification | NOT_RUN |
| native release-host CGO race with real libvectorscan | NOT_RUN |
| real QAT hardware offload | NOT_RUN |
| active kTLS data-path verification on a kTLS-capable kernel | NOT_RUN |
| `govulncheck` against the final dependency tree | NOT_RUN |

These are mandatory truth boundaries. Do not convert the stub/ABI-only PASS results into these rows.

## Historical component benchmark evidence

These measurements are useful for regression context only; they are not end-to-end capacity claims.

### Hot path / ring / request capture

- fixed circular ring add: ~9.8 ns/op, 0 B/op, 0 allocs/op
- bounded ~60 KiB passive JSON scanner: ~35.8 us/op, ~1.3 KiB/op, 14 allocs/op

### P0-A proxy buffer

- fixed 32 KiB BufferPool operation: ~10.9 ns/op, 0 B/op, 0 allocs/op

### P0-B load balancing, 8-member pool

Recorded before -> after examples:

- round robin: ~47.4 ns / 64 B / 1 alloc -> ~20.3 ns / 0 B / 0 alloc
- least connections: ~43.0 ns -> ~15.6 ns / 0 alloc
- IP hash: ~47.2 ns -> ~12.7 ns / 0 alloc
- random: ~48.1 ns -> ~27.4 ns / 0 alloc
- old member context handoff: ~32 ns / 48 B / 1 alloc, removed

### P0-C async observation plane

- old synchronous request observation: ~2.56-2.63 us, 1664 B, 24 allocs
- optimized direct signal store: ~443-445 ns, 48 B, 3 allocs
- dataplane async enqueue: ~35-36 ns, 0 B, 0 allocs

### P0-D match logging

- old per-match JSON `slog.Warn`: ~1.42-1.51 us, 232 B, 8 allocs
- async callback-front enqueue: ~40 ns, 0 B, 0 allocs

### P2 AI lazy path

- old eager unsampled/no-match path: ~1.53 us, 848 B, 17 allocs
- final lazy unsampled/no-match path: ~174 ns, 0 B, 0 allocs
- AI blocklist lookup after atomic snapshot: ~21 ns at 8/32-way; previous mutex path was roughly 90-107 ns

### Ring sharding experiment

Rejected after measurement:

- existing ring: ~9.5-10.2 ns serial; ~26-36 ns at 8-way; ~43-49 ns at 32-way
- 16-shard prototype: ~14.6-15.1 ns serial; ~42 ns at 8-way; ~61 ns at 32-way

Conclusion: retain the simpler current ring; do not reintroduce sharding without new evidence.

## Benchmark harness

`wafbench` is already implemented with:

- deterministic backend mode;
- full HTTP/reverse-proxy mode;
- Coraza-only mode;
- L4 connect/TLS-handshake mode;
- JSON baseline output;
- compare mode;
- CPU/heap profiling;
- clean GET, clean JSON 1/16/64/256 KiB and malicious SQLi/XSS/traversal scenarios.

The technology selection decision is already VectorScan. Use `wafbench` now for production qualification, regression and sizing—not to reopen XDP vs VectorScan selection.

## Artifact Packaging Integrity Gate — 2026-09-09

A mandatory post-packaging release gate is now implemented following the mini-SIEM installer-ZIP incident in which a generated archive contained README text in place of an executable installer while the source repository and deployed data remained unaffected.

Implemented WAF controls:

- `RELEASE_PROCESS.md` — canonical release-artifact integrity policy and evidence boundary;
- `DEVELOPMENT.md` / `TESTING.md` — cross-cutting engineering and test-policy summaries;
- `build-release-artifact.sh` — clean staging, per-artifact `RELEASE_MANIFEST.txt`, deterministic ZIP construction, mandatory post-build verification;
- `verify-release-artifact.sh` — ZIP CRC/path/symlink checks, required-file checks, critical-script size/shebang/mode/syntax validation, complete source-to-package SHA-256 and mode comparison, manifest file-set/hash verification.

Evidence from this stage is recorded only after execution against the newly generated artifact. Artifact-integrity PASS proves delivery-byte/mode fidelity and implemented package checks; it does not prove any still-`NOT_RUN` real Coraza, real VectorScan, production CRS Learning, QAT, kTLS or target-host qualification gate.

### Artifact-gate execution evidence

Actually executed on 2026-09-09 in the packaging environment:

- `bash -n` across all root release shell scripts plus `benchmark/build.sh`: **PASS** after adding the artifact-gate scripts.
- normal complete-source package created by `build-release-artifact.sh`: **PASS** through clean re-extraction and `verify-release-artifact.sh`.
- extracted package/source equality: **PASS**, 117 packaged source files compared by SHA-256 and Unix mode; `RELEASE_MANIFEST.txt` verified independently as generated artifact metadata.
- tamper simulation replacing extracted `install.sh` bytes with `README.md` content: **REJECTED as expected** (`unexpected shell shebang: install.sh`).
- path-traversal simulation containing `../escape.txt`: **REJECTED as expected** (`unsafe archive path`).

A verifier implementation issue was found during this stage: Python `zipfile.extractall()` does not restore Unix executable modes automatically. The verifier was corrected to reapply the ZIP entry's stored Unix mode from `external_attr` before checking extracted permissions. This is a verifier behavior correction, not WAF dataplane behavior.

Real Go 1.25/Coraza v3.7.0, real libvectorscan, CRS Learning, QAT and active-kTLS qualification remain unchanged and **NOT_RUN** where previously recorded.

Patch reconstruction also caught a release-process defect during this stage: the first baseline-relative `git diff --binary` omitted five newly created untracked files. The reconstruction comparison failed with those five files missing. Patch generation was corrected by intent-adding new paths (`git add -N .`) before diff generation; the corrected patch then reconstructed all 117 source files byte-for-byte and mode-for-mode. The failed first reconstruction is not counted as PASS evidence; only the corrected reconstruction is.

## 2026-09-10 Debug Evidence / Phase 1 Qualification Update

Implemented test coverage:

- Debug evidence bounded capture lifecycle: ADDED
- Debug evidence export lifecycle: ADDED

Not run:

- Real Go 1.25 + Coraza v3.7.0 execution
- Real libvectorscan hs_compile_multi/hs_scan
- Production CRS Learning Period qualification
- Production traffic debug capture qualification

## 2026-09-10 Debug Evidence integration update

Implemented source-level gates:
- transaction-final MatchedRules evidence path: ADDED
- candidate coverage comparison: ADDED
- false-negative zero gate helper: ADDED
- debug masking/export foundation: ADDED

Execution boundary:
- Full Go regression: NOT_RUN (toolchain prerequisite unchanged)
- Real Coraza: NOT_RUN
- Real VectorScan: NOT_RUN

## 2026-09-10 Supportability Slice

Added source-level qualification helpers and debug evidence bundle foundation.
Execution of real Go 1.25 dependency gates remains NOT_RUN in this environment.

## 2026-09-13 — Debug/supportability + Phase 1 runner evidence

### Source implementation completed

- request/response/TLS/proxy-LB evidence is correlated by one server-generated debug transaction ID;
- final Coraza truth is read from `tx.MatchedRules()` after `ProcessLogging()` and captured even when VectorScan has no observation;
- VectorScan evidence distinguishes `observed:false` from an observed zero-FN result;
- debug storage is site/tenant scoped, count/TTL bounded, periodically cleaned and sanitized before export;
- debug capture start/stop/export is audited; evidence listing/export requires reviewer-or-higher and capture control requires operator-or-higher;
- `wafctl` doctor/debug/support commands and admin endpoints are implemented;
- support bundles include sanitized config/status/metrics, bounded logs when available, dependency evidence, SPDX-formatted SBOM evidence, manifest and SHA-256 list;
- `cmd/wafqualify` + `run-phase1-qualification.sh` implement real native-VectorScan vs real-Coraza differential replay with a hard zero-false-negative gate and minimum eligible-match evidence requirement.

### Actually executed on the packaging host

Host prerequisites observed during this run:

- local Go: **go1.23.2 linux/amd64**;
- `pkg-config`: available, but `libhs`/VectorScan: **not resolvable**;
- Phase 0 preflight also observed NGINX 1.26.3 and OpenSSL 3.5.5; this is inventory, not new kTLS/QAT evidence.

Executed gates:

| Gate | Result | Evidence boundary |
|---|---|---|
| `bash -n` all root release scripts | PASS | shell syntax only |
| `GO111MODULE=off GOTOOLCHAIN=local go test ./cmd/wafctl` | PASS | stdlib-only CLI masking/bundle unit tests |
| standalone `wafctl` build + `wafctl version` | PASS | CLI compile/smoke only |
| `GOTOOLCHAIN=local go test ./...` | BLOCKED/NOT_RUN | Go 1.23.2 rejected because module requires Go >=1.25 |
| `./qualify-release-host.sh --preflight` | BLOCKED, exit 3 | missing Go >=1.25 and real libhs/provenance |
| `./run-phase1-qualification.sh --rules ./coraza.conf` | BLOCKED, exit 3 | real libhs unavailable; no corpus qualification executed |

The new root debug tests and VectorScan qualification tests are present in source but cannot be counted as executed repository tests on this host because the module-level Go 1.25 prerequisite prevents the root/package build.

### Still NOT_RUN / mandatory live gates

- real Go 1.25 full repository regression/vet/race for this exact source baseline;
- real Coraza v3.7.0 `realcoraza` DetectionOnly+nolog truth gate;
- real libvectorscan `hs_compile_multi`/`hs_scan` semantic and race gates;
- real Phase 1 corpus differential with representative production CRS/traffic;
- production Learning Period qualification with observed false negatives = 0;
- live installed `wafctl doctor`, bounded debug capture/export and support-bundle smoke;
- production privacy review of captured/exported diagnostics;
- `govulncheck` and independent full SBOM/release-host dependency vulnerability evidence;
- real QAT and active-kTLS qualification where applicable.

### 2026-09-13 packaging rehearsal

After source/doc synchronization, `build-release-artifact.sh` was exercised before the final delivery build. Its clean extraction/verifier returned `ARTIFACT_INTEGRITY_PASS files=131`, proving the current packaging path preserves the staged source byte hashes and Unix modes and verifies the generated release manifest. The final delivery ZIP is rebuilt after this ledger entry and must repeat the same gate; its final checksum is external release metadata and is intentionally not self-embedded here.


## Phase 2 implementation update (2026-09-13)

Implemented qualification corpus schema foundation and differential false-negative gate helpers. Full runtime qualification remains deferred until real Go 1.25, Coraza v3.7.0 execution and libvectorscan are available.

## Phase 2 Slice B

Status: IMPLEMENTED_TESTING_DEFERRED

Added lifecycle/audit/CLI implementation. Full tests, race, vet, real Coraza, real libvectorscan and artifact release gates are deferred to final validation stage.


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

Status: IMPLEMENTED_TESTING_DEFERRED

Added CRS capability parsing/analyzer foundation. Runtime CRS corpus qualification remains deferred.

## 2026-09-14 — Phase 2 Coverage Expansion Slice C

Implemented but broad testing is deferred per the owner's test-last workflow.

Added source-level validation targets (not yet promoted to PASS evidence):

- deterministic CRS ruleset ingestion from directory or config entrypoint;
- recursive Include / IncludeOptional handling and duplicate rule-ID detection;
- normalized SecRule metadata extraction and capability inventory;
- Coverage Report v2 generation;
- `wafctl coverage analyze` CLI integration;
- analyzer/runtime-classifier parity hardening.

Important correction from the earlier Slice A/B model: the coverage analyzer now reflects the runtime's actual conservative scope. Bare `REQUEST_HEADERS` and `uppercase` are **not** reported eligible because the current VectorScan dataplane supports fixed-name `REQUEST_HEADERS:<name>` and lowercase only. This avoids optimistic coverage reporting.

Current truth boundary: real Go 1.25 full regression, real Coraza v3.7 execution, real libvectorscan scan/compile, representative CRS coverage qualification and Phase 1 zero-FN evidence remain NOT_RUN until final validation on a suitable host.

Slice C validation actually executed on the packaging host (2026-09-14):

- root release shell `bash -n` including `benchmark/build.sh`: **PASS**;
- canonical `GOTOOLCHAIN=local go test ./internal/coverage/...`, `./cmd/wafctl`, and `./...`: **BLOCKED before test execution** because installed Go is 1.23.2 while `go.mod` requires Go >=1.25.0;
- real `pkg-config libhs`: **BLOCKED / unavailable**;
- isolated source-compatibility compile/test check using a temporary Go 1.23 module containing only `internal/coverage/...` and `cmd/wafctl`: **PASS**; this is a syntax/logic regression aid only, not canonical Go 1.25/Coraza qualification evidence;
- isolated `go vet` for the same coverage/wafctl subset: **PASS**;
- `wafctl coverage analyze` synthetic ruleset smoke with Include expansion: **PASS**; 5 rules -> 2 eligible, 3 Coraza-only, 0 unsupported, with expected ARGS/multi-selector/uppercase rejection reasons;
- representative real OWASP CRS inventory/coverage run: **NOT_RUN**;
- real Coraza/libvectorscan Phase 0/1 gates: **NOT_RUN/BLOCKED** by host prerequisites.

Artifact-integrity correction discovered during Slice C packaging: the supplied Slice B ZIP stored release/install shell scripts as mode `0644`, so the release builder was not directly executable and the prior claimed executable-mode PASS could not be reproduced from that artifact. Slice C restores the documented executable scripts to `0755`; final ZIP verification and patch reconstruction must prove those modes are preserved. This is treated as a baseline packaging defect, not ignored.

Final Slice C packaging gate (after implementation):

- `build-release-artifact.sh` clean staging/re-extraction verifier: **PASS** (`ARTIFACT_INTEGRITY_PASS`, 152 packaged source files before generated manifest);
- source/package SHA-256 equality: **PASS**;
- source/package Unix mode equality: **PASS**, including restored `0755` release/install scripts;
- generated release manifest file-set/hash verification: **PASS**;
- baseline-relative binary patch reconstruction from the supplied Slice B ZIP: **PASS**, working source bytes and modes identical after apply.

## 2026-09-14 — Phase 3 Release Hardening Implementation

Status at the pre-repair Phase 3 checkpoint: **QUALIFICATION_REQUIRED**.

At this pre-repair checkpoint, implemented gates existed for deterministic source packaging, release provenance, SPDX/CycloneDX SBOMs, govulncheck evidence, archive hardening, negative mutation testing, reproducibility and optional detached minisign signing. Artifact-identity and evidence-binding gaps identified later are superseded by the Truth-Boundary Repair section below.

Final execution evidence for this Phase 3 source slice must be recorded after the concentrated validation run. Until then, govulncheck, real Coraza, real VectorScan, representative production CRS and organizational signing remain **NOT_RUN / NOT_CONFIGURED** unless explicitly proven by that run.

### Phase 3 concentrated validation evidence — 2026-09-14

Executed on the packaging host:

- root/benchmark release shell `bash -n`: **PASS**.
- `tools/release_evidence.py` Python compile: **PASS**.
- `release-security-scan.sh --mode auto`: **NOT_RUN**, exit 3, because `govulncheck` is unavailable. This is not vulnerability-scan PASS evidence.
- deterministic complete-source packaging + independent extraction verification: **PASS** (`ARTIFACT_INTEGRITY_PASS`).
- negative artifact rejection: **PASS** for traversal, secret-like `.env`, symlink, duplicate ZIP entry, executable-mode loss and installer-content replacement.
- fixed-epoch two-build reproducibility: **PASS**; two independently produced source ZIPs were byte-identical.
- canonical `GOTOOLCHAIN=local go test ./...`: **BLOCKED** because local Go is 1.23.2 while `go.mod` requires 1.25.0.
- `qualify-release-host.sh --preflight`: **BLOCKED**, exit 3: Go 1.23.2, no `pkg-config libhs`, and no established real VectorScan provenance. NGINX 1.26.3 and OpenSSL 3.5.5 were detected.
- `GOTOOLCHAIN=local WAF_VECTORSCAN=required ./build.sh`: **BLOCKED** before build because Go >=1.25.0 is required.
- real Coraza v3.7 transaction truth gate: **NOT_RUN**.
- real libvectorscan `hs_compile_multi` / `hs_scan`: **NOT_RUN**.
- representative production CRS differential / zero-FN Phase 1 qualification: **NOT_RUN**.
- detached release signing: **NOT_CONFIGURED** because no approved organizational minisign key/process was supplied; no signing key was generated or bundled.

Conclusion: Phase 3 release-hardening implementation and portable source-artifact gates are **PASS**, but the overall product remains **QUALIFICATION_REQUIRED** for unavailable runtime/security gates. Do not call this a production-qualified VectorScan release.
- Historical pre-repair check: `--variant native-vectorscan` was rejected when real `pkg-config libhs` provenance was absent. The repaired source-archive builder now rejects binary identities categorically because it only emits `SOURCE_ARCHIVE`.

## 2026-09-14 — Phase 3 Truth-Boundary Repair validation

Status: **QUALIFICATION_REQUIRED**. The repair gates below are source/artifact evidence only; they do not qualify real Coraza/libvectorscan behavior.

Executed on the packaging host:

- shared `internal/capability`, `internal/coverage/...`, `internal/vectoraccel`, and `cmd/wafctl` source-compatibility tests in an isolated Go 1.23 stdlib-only harness: **PASS**;
- isolated `go vet` for the same repair surface: **PASS**;
- classifier parity regression covers the previously divergent fixed-header punctuation, empty regex, implicit phase, exact chain action, negation, multi-selector and unsupported-transform cases: **PASS**;
- `IncludeOptional` coverage/runtime parity and dynamic mandatory-Include fail-closed behavior: **PASS**;
- canonical `GOTOOLCHAIN=local go test ./...`: **BLOCKED before execution** because this host is Go 1.23.2 while `go.mod` requires Go 1.25.0;
- `qualify-release-host.sh --preflight`: **BLOCKED**, exit 3 (Go 1.23.2, no real `pkg-config libhs`, no established real VectorScan provenance); NGINX 1.26.3 and OpenSSL 3.5.5 detected;
- `release-security-scan.sh --mode auto`: **NOT_RUN**, exit 3 because `govulncheck` is unavailable; the emitted evidence is nevertheless schema-valid and source-manifest-bound, and is not PASS evidence;
- deterministic source artifact + repaired `verify-release-artifact.sh`: **PASS** on the validation build, including exact SOURCE_MANIFEST/RELEASE_MANIFEST completeness, schema-v2 `SOURCE_ARCHIVE` identity, provenance cross-digests, govuln source binding, and SPDX/CycloneDX dependency parity with `go.mod`;
- truth-boundary/archive negative mutations: **12/12 rejected**: traversal, secret-like file, symlink, duplicate entry, executable-mode loss, installer replacement, release-manifest omission, source-manifest omission with companion rehashing, forged govuln PASS, source→binary artifact-type relabel, provenance/source mismatch, and SBOM dependency drift;
- standalone fabricated external govuln JSON (`{"status":"PASS"}`) supplied to the builder: **REJECTED** for missing required evidence identity/source-binding fields;
- `build-release-artifact.sh --variant native-vectorscan`: **REJECTED** because the source ZIP builder may emit only `SOURCE_ARCHIVE`; binary identity belongs to actual binary build provenance;
- detached signature verification without installed minisign: shell verifier and `wafctl release verify-signature` both return **NOT_CONFIGURED / exit 3**, not VALID;
- fixed-epoch reproducibility initially **FAILED** after the repair because release evidence ran plain `go version`, which attempted an automatic Go 1.25 download and captured an ephemeral UDP source port in the error text. This was treated as a real release defect. `tools/release_evidence.py` now forces `GOTOOLCHAIN=local` for Go version evidence; the repeated fixed-epoch double build then **PASSed byte-for-byte**.

Still **NOT_RUN / BLOCKED**: real Go 1.25 full suite, real Coraza v3.7 transaction truth gate, real libvectorscan `hs_compile_multi`/`hs_scan`, representative production CRS Phase 1 differential/zero-FN qualification, and real organizational minisign signing/verification with an approved key.

The final delivery artifact is rebuilt after this ledger update and must repeat the artifact verifier, negative truth-boundary gates, reproducibility gate, and baseline-relative patch reconstruction. Its final SHA-256 is external delivery metadata and is intentionally not self-embedded here.

## Phase 4 Slice A

Completed source-level changes:
- ClientIdentityDecision model
- debug evidence field
- wafctl proxy identity show

Remaining validation:
- full Go qualification
- runtime proxy deployment tests
- complete audit event pipeline validation


Phase 4 Slice A follow-up: added client identity audit event model constants CLIENT_IDENTITY_RESOLVED and CLIENT_IDENTITY_HEADER_REJECTED.

## Phase 4 Slice B Validation

Implemented but testing deferred.

Executed:
- Source modification completed

Not Run:
- go test ./... (blocked: Go 1.25 toolchain download unavailable in offline environment)
- production proxy traffic validation
- load qualification

## Phase 4 Slice C Roadmap Preparation

Status:
PLANNED

No implementation tests executed.

Not Run:
- CIDR policy functional tests
- policy persistence tests
- enforcement integration tests


## Phase 4 Slice C CIDR Policy
- Source implementation: added
- go test ./...: NOT_RUN/BLOCKED (Go 1.25 toolchain download unavailable)

## Phase 4 Slice D
Status: IMPLEMENTED_TESTING_DEFERRED
Runtime qualification remains separate.


## Phase 4 Slice E — Persistent Security State

Status: IMPLEMENTED_TESTING_DEFERRED

Implemented foundation: security state models and in-memory persistence abstraction. Production database durability, HA replication, retention tuning, and external integrations remain deferred.


## Phase 4 Slice F PKI Hardening
- Source changes added.
- Runtime PKI qualification remains NOT_RUN.
- Production CA/CRL infrastructure validation remains deferred.


## Phase 5 Slice A — Runtime Qualification Closure

Status: IMPLEMENTED_TESTING_DEFERRED

Implemented: runtime qualification evidence schema foundation. Real Go 1.25, Coraza v3.7.0 transaction, and libvectorscan runtime gates remain NOT_RUN until executed on a qualified release host.
