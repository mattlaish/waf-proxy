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


## 2026-09-11 Supportability Slice

Added wafctl debug export/doctor/support bundle foundation and Phase 1 differential qualification helper. Real Coraza/libvectorscan execution remains NOT_RUN.


## Stage 2 Supportability Implementation
- Added wafctl supportability command foundation.
- Added qualification phase1 differential runner foundation.
- Real Go 1.25/Coraza/libvectorscan qualification remains NOT_RUN.

## 2026-09-11 Phase 0 / Phase 1 real qualification attempt

Executed against the current roadmap-synchronized Stage 2 baseline:

- `./qualify-release-host.sh --preflight`: **BLOCKED (exit 3)**.
  - Linux host: PASS.
  - local Go: 1.23.2 -> BLOCKED; >=1.25.0 required.
  - Go 1.25.0 pin: PASS.
  - Coraza v3.7.0 pin: PASS.
  - C compiler: PASS.
  - `pkg-config libhs`: BLOCKED / unavailable.
  - real VectorScan provenance: BLOCKED / unavailable.
- `./qualify-release-host.sh --core`: **BLOCKED (exit 3)** at preflight; core correctness gates did not execute.
- Real Coraza v3.7.0 runtime, real DetectionOnly+nolog `MatchedRules()` truth test, real `hs_compile_multi`/`hs_scan`, native real-libhs race, production CRS replay and Phase 1 zero-false-negative qualification: **NOT_RUN**.
- External dependency acquisition was unavailable and no local Go 1.25/libvectorscan cache was found.
- No ABI-only shim result was promoted.

See `PHASE0_PHASE1_QUALIFICATION_REPORT_2026-09-11.md` for the exact transcript and boundary.


## 2026-09-11 Phase 2 conservative coverage validation

Source changes under test: ordered exact-transform pipelines, `QUERY_STRING`, `SERVER_NAME`, special `Host`/`Transfer-Encoding` request-header reconstruction, and conservative exclusions.

Executed on the packaging host:

- installed local toolchain: Go 1.23.2;
- `GOTOOLCHAIN=local go test ./...`: **BLOCKED** before test execution because `go.mod` requires Go >=1.25.0;
- temporary isolated module containing `internal/vectoraccel`, `go test ./internal/vectoraccel -count=1`: **PASS**;
- focused Phase 2 transform/source/grouping tests: **PASS**;
- isolated `go vet ./internal/vectoraccel`: **PASS**;
- isolated `go test -race ./internal/vectoraccel -count=1`: **PASS**.

The isolated module is a source-level unit/race check only. It is not real Go 1.25/Coraza/libvectorscan release evidence.

New real gate added but **NOT_RUN** on this host:

- `internal/vectoraccel/phase2_realcoraza_test.go` under the `realcoraza` build tag validates the allow-listed transform output and the `QUERY_STRING` / `SERVER_NAME` / `Host` / `Transfer-Encoding` source mapping against Coraza v3.7.0.

Still **NOT_RUN**: full Go 1.25 repository regression, real Coraza v3.7.0 Phase 0/1/2 execution, real libvectorscan scan, production CRS differential replay, and production zero-observed-false-negative qualification.


### Phase 2 artifact packaging gate

The Phase 2 delivery is required to pass, after the documentation synchronization above: ZIP CRC/path/symlink checks, required-file/script validation, complete source-to-extracted-package SHA-256 and Unix-mode equality, release-manifest hash verification, baseline-relative patch reconstruction byte/mode equality, secret-pattern hygiene, and generated-ELF exclusion. These packaging checks are independent of the still-NOT_RUN real Coraza/libvectorscan/CRS qualification gates.

## 2026-09-11 Phase 2 coverage increment 2 validation

Implemented scope: `REQUEST_URI_RAW`, `REQUEST_LINE`, `REQUEST_BASENAME`, `REMOTE_ADDR`, `REMOTE_PORT`, plus exact `base64Encode` and `hexEncode` transforms. The semantic adapter version was bumped so persisted group state must re-enter Learning.

Executed on the available packaging host using a temporary isolated module with local Go 1.23.2:

- baseline isolated `internal/vectoraccel` unit/vet/race before changes: **PASS**;
- expanded isolated `go test ./internal/vectoraccel -count=1`: **PASS**;
- focused expanded transform/source/classifier tests: **PASS**;
- expanded isolated `go vet ./internal/vectoraccel`: **PASS**;
- expanded isolated `go test -race ./internal/vectoraccel -count=1`: **PASS**.

Project-wide `GOTOOLCHAIN=local go test ./...`: **BLOCKED** because `go.mod` requires Go >=1.25.0 while the host has Go 1.23.2. The expanded `realcoraza` parity tests, real libvectorscan execution and representative production CRS differential/zero-false-negative qualification are therefore **NOT_RUN**. Isolated test PASS is source-level evidence only.

## 2026-09-11 — Phase 3 release engineering evidence

Executed on this packaging host:

- `bash -n` for all new/modified Phase 3 release scripts: PASS.
- `./phase3-release-tooling-test.sh`: PASS.
- portable `generate-release-evidence.sh`: PASS; emitted SPDX 2.3, CycloneDX 1.5, versions/provenance, govulncheck status and evidence checksums.
- repeated complete-source builds with identical `SOURCE_DATE_EPOCH=1789092000`, version and portable flavor: PASS; both ZIPs SHA-256 `9888418b78846f27e9cc9cc2d328c1613f7dd497814d4cfdf22b687b1f127d50` before final documentation synchronization.
- `verify-release-artifact.sh` on the reproducibility artifact: PASS (`source_files=134 packaged_files=142 flavor=portable`) before final documentation synchronization.
- native evidence generation: BLOCKED / exit 3 because real `pkg-config libhs` is unavailable.
- required govulncheck gate: BLOCKED / exit 3 because `govulncheck` is unavailable.
- organizational artifact signing: NOT_RUN because `WAF_RELEASE_SIGNING_KEY` is not configured.

Host facts: Go `go1.23.2 linux/amd64`; OpenSSL `3.5.5`; NGINX `1.26.3`; libhs unavailable; govulncheck unavailable. Full Go 1.25 / real Coraza / real VectorScan / production CRS qualification remains NOT_RUN/BLOCKED exactly as documented by the prior phases.
- signing implementation self-test with an ephemeral test-only RSA key: PASS (sign + verify path only; this is not organizational release signing evidence).
- full repository `GOTOOLCHAIN=local go test ./...`: BLOCKED as expected because `go.mod` requires Go 1.25.0 and the host has Go 1.23.2.

## 2026-09-11 — Phase 4 security/product implementation validation

Executed on the packaging host:

- isolated `security.go` + Phase 4 tests with `GOTOOLCHAIN=local go test -race -v ./...`: **PASS** (`ok security-isolated 1.146s`), covering allow-over-deny, request-ID replacement, custom/expired CIDR behavior, rate/in-flight caps, TCP connection cap, TLS-handshake cap and bounded limiter buckets;
- isolated static + URL CRL suite with `GOTOOLCHAIN=local go test -race -v ./...`: **PASS** (`ok pki-isolated 4.777s`), including prior static CRL behavior, URL/SSRF rejection, LKG retention, refresh dedupe and persistent CRL cache round trip;
- isolated persistent security-state round trip with `GOTOOLCHAIN=local go test -race -v ./...`: **PASS** (`ok state-isolated 1.013s`), including 0600 state file, no raw bearer token, restored hashed session, AI block, learner, notification, audit and security counters;
- `python3 -m json.tool config.sample.json`: **PASS**;
- `bash -n` across root shell scripts: **PASS**;
- `systemd-analyze verify` on an otherwise identical temporary unit with `ExecStart=/bin/true`: **PASS**. Direct verification of the source unit reports only the expected absent `/usr/local/bin/waf-proxy` executable in this source-only packaging workspace.

Blocked/not run:

- `GOTOOLCHAIN=local go test ./...`: **BLOCKED**, `go.mod requires go >= 1.25.0 (running go 1.23.2)`;
- `GOTOOLCHAIN=local go vet ./...`: **BLOCKED** for the same reason;
- `request_id_coraza_phase4_test.go`: **NOT_RUN** here because the qualified Go 1.25/real Coraza module environment is unavailable;
- positive external HTTPS CRL retrieval/rotation: **NOT_RUN** because this environment has no usable external DNS/network path;
- deployed Debian/Ubuntu service/restart/HA/load qualification and `govulncheck`: **NOT_RUN/BLOCKED**.

These results establish implementation-level evidence only. They do not change the Phase 0/1/2 real Coraza/VectorScan/CRS qualification boundary.

### Phase 4 pre-final artifact gate

A controlled complete-source portable build using the Phase 3 release tooling returned `ARTIFACT_INTEGRITY_PASS source_files=143 packaged_files=151 flavor=portable`. This verifies clean staging, regenerated Phase 4 release evidence, extracted source byte/mode equality and manifest/evidence validation before the final documentation freeze. The final artifact is rebuilt from the frozen source with the same `SOURCE_DATE_EPOCH` and independently rechecked below/outside the source tree.

## 2026-09-11 — Main-branch build/test blocker hotfix

External main-branch reproduction supplied by the owner established two failures before this hotfix: `go build ./...` exited 1 with `updates to go.mod needed; to update it: go mod tidy`, and `go test ./...` otherwise passed except `TestReleaseScriptsAreLFAndBashSyntaxClean`, because the tested scripts were Git mode `100644`. The owner also confirmed that `go mod tidy` made the code build cleanly.

This hotfix synchronizes `go.mod`/`go.sum` to the Coraza v3.7.0 indirect graph and makes all scripts covered by `TestReleaseScriptsAreLFAndBashSyntaxClean` executable in the Git-mode patch. Packaging-host checks executed here: all covered scripts are mode 0755 in the fixed source tree and `bash -n` clean. Full Go 1.25 `go build ./...` / `go test ./...` cannot be independently rerun in this packaging host because only Go 1.23.2 is available and external toolchain/module acquisition is blocked; do not relabel the owner's external execution as local release-host evidence.

### Hotfix artifact gate

Pre-freeze packaging verification passed with `source_files=143 packaged_files=151 flavor=portable`. Two builds using identical source, version, flavor and `SOURCE_DATE_EPOCH=1789110000` were byte-identical (pre-freeze SHA-256 `b0d90e41dbe47b2df83160c10e1aa5605176da60b900e6323c99534e724d3795`). The mode-aware Git patch reconstructed the fixed workspace from the reported main state with 152 regular files byte-for-byte and mode-for-mode identical. Final delivery is rebuilt after this evidence text is frozen.
