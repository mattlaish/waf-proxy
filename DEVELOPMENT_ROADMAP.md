# Development Roadmap — WAF Proxy

Handover checkpoint: 2026-09-07 (Asia/Taipei)

This roadmap starts from the packaged implementation baseline `waf-proxy-vectorscan-learning-coraza37-2026-09-04.zip`. VectorScan is the selected regex-acceleration direction. Do **not** reopen XDP-vs-VectorScan selection unless the owner explicitly asks to revisit it.

## Status vocabulary

- **DONE** — implemented in the current source line.
- **QUALIFICATION_REQUIRED** — code exists, but a real target/release-host gate is still mandatory before production enablement.
- **PLANNED** — approved next work.
- **DEFERRED** — intentionally postponed; do not start without owner direction.

## Current implementation baseline

| Area | Status | Current truth |
|---|---|---|
| P0 hot-path cleanup | DONE | circular rings, atomic disabled gates, shared request-body prefix, reduced allocations |
| P0-A proxy buffering | DONE | ReverseProxy BufferPool and streaming-safe FlushInterval behavior |
| P0-B load balancing | DONE | zero-allocation member selection and removal of per-request member context handoff |
| P0-C observations | DONE | bounded async observation plane with non-blocking enqueue |
| P0-D match logging | DONE | independent bounded async aggregation/logging plane |
| P1 response inspection | DONE | inherit/on/off response-body inspection plus configurable response-body limit |
| P1 backend transport | DONE | configurable connection pool, shared pool transport, retired idle-connection cleanup |
| P2 non-XDP hardening | DONE | release scripts fixed, AI lazy sampling, AI queue hardening, atomic blocklist, statusRecorder correctness |
| Benchmark harness | DONE | backend/http/coraza/l4/compare modes and JSON/profile output |
| TLS acceleration C1-C3 | DONE / QUALIFICATION_REQUIRED | optional NGINX/OpenSSL frontend, modern HTTP/2 syntax, kTLS/QAT detection/fallback; real QAT hardware remains unqualified |
| Coraza dependency | DONE / QUALIFICATION_REQUIRED | source pinned to Coraza v3.7.0 and Go 1.25.0; real release-host execution still required |
| VectorScan Learning Accelerator | DONE / QUALIFICATION_REQUIRED | conservative grouped acceleration, transaction-final Coraza truth, Learning/FAILSAFE state machine, optional native libhs |
| XDP prefilter | DEFERRED | separate future L3/L4 feature; no longer part of the current selection decision |

## Phase 0 — Mandatory release-host qualification

**Status: QUALIFICATION_REQUIRED — highest priority before production VectorScan enablement.**

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

**Status: QUALIFICATION_REQUIRED — production runner implemented; real release-host/corpus execution remains mandatory.**

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

**Status: QUALIFICATION_REQUIRED — Slices A-C source implementation exists, but production promotion remains blocked until Phase 0/1 qualification passes.**

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

**Status: QUALIFICATION_REQUIRED.**

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

**Status: PLANNED but lower priority than VectorScan qualification.**

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

A new development chat should begin with **Phase 0 real release-host qualification** if it has a suitable Linux host/toolchain. If it does not, it should not fabricate PASS results. It may improve tests/docs or prepare qualification automation, but production enablement of VectorScan remains blocked until the real gates are executed.

## Cross-cutting mandatory gate — Artifact Packaging Integrity

**Status: DONE / REQUIRED FOR EVERY FUTURE RELEASE.**

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

**Status: DONE / QUALIFICATION_REQUIRED.**

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
