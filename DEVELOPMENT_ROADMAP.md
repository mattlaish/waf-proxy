# Development Roadmap — WAF Proxy

Handover checkpoint: 2026-09-11 (Asia/Taipei)

This roadmap is synchronized to the current packaged development baseline `waf-proxy-supportability-stage2-2026-09-11.zip`. The earlier `waf-proxy-vectorscan-learning-coraza37-2026-09-04.zip` remains the historical VectorScan/Coraza implementation baseline. VectorScan is the selected regex-acceleration direction. Do **not** reopen XDP-vs-VectorScan selection unless the owner explicitly asks to revisit it.

## Status vocabulary

- **DONE** — implemented in the current source line.
- **QUALIFICATION_REQUIRED** — code exists, but a real target/release-host gate is still mandatory before production enablement.
- **IMPLEMENTED_TESTING_DEFERRED** — source/tooling exists in the current baseline, but required execution or production validation has not been completed.
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
| Phase 1 differential qualification harness | IMPLEMENTED_TESTING_DEFERRED / QUALIFICATION_REQUIRED | candidate-vs-Coraza comparator and zero-false-negative result gate exist; real Coraza/libvectorscan/CRS replay has not run |
| Debug & Evidence Capture | IMPLEMENTED_TESTING_DEFERRED | bounded capture, MatchedRules evidence hook, metadata/masking/tenant/TTL foundations exist; live production capture qualification remains deferred |
| Operator supportability (`wafctl`) | IMPLEMENTED_TESTING_DEFERRED | `debug export`, `doctor`, and `support bundle` command foundations exist; runtime dependency/status integration and production bundle qualification remain incomplete |
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

Current 2026-09-11 release-host attempt: `./qualify-release-host.sh --preflight` and `--core` both returned **BLOCKED (exit 3)** because the installed local toolchain is Go 1.23.2, `pkg-config libhs` is unavailable, and real VectorScan provenance cannot be established. External dependency acquisition was also unavailable in this environment. Therefore the Phase 0 core correctness gates remain `NOT_RUN`; no PASS has been promoted from stub/ABI evidence. See `PHASE0_PHASE1_QUALIFICATION_REPORT_2026-09-11.md`.

**Exit criterion:** all mandatory real Coraza + real VectorScan correctness gates PASS. Performance is useful evidence but is not a prerequisite for choosing VectorScan; the choice is already made.

## Phase 1 — VectorScan Learning production qualification

**Status: IMPLEMENTED_TESTING_DEFERRED / QUALIFICATION_REQUIRED.**

The current Stage 2 source includes the differential comparison/zero-false-negative harness foundation. This does **not** constitute production qualification. Real Go 1.25 + Coraza v3.7.0 + libvectorscan + representative CRS/traffic execution remains mandatory before any production qualification claim.

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

**Status: IMPLEMENTED_TESTING_DEFERRED / QUALIFICATION_REQUIRED (owner-directed entry on 2026-09-11).**

The owner explicitly directed Phase 2 implementation even though Phase 0/Phase 1 real production qualification remains blocked on the current host. This authorizes source implementation only; it does **not** waive the production gate. No Phase 2 group may be treated as production-qualified until the real Go 1.25 + Coraza v3.7.0 + libvectorscan gates and representative zero-observed-false-negative qualification pass.

Implemented conservative increments:

1. **Exact transform pipeline expansion — IMPLEMENTED_TESTING_DEFERRED.** In addition to `t:none` and `t:lowercase`, the classifier can reproduce `t:uppercase`, `t:trim`, `t:trimLeft`, `t:trimRight`, `t:removeNulls`, `t:replaceNulls`, `t:compressWhitespace`, `t:removeWhitespace`, `t:length`, `t:base64Encode`, and `t:hexEncode`. Transform order is preserved and included in the group semantic key/fingerprint. Decode transforms, hashes, normalization/decoding transforms and every other non-allow-listed transform remain Coraza-only.
2. **Fixed request-header parity — IMPLEMENTED_TESTING_DEFERRED.** Ordinary fixed-name `REQUEST_HEADERS:name` still use Go header values; `REQUEST_HEADERS:Host` now mirrors Coraza's HTTP connector by using `req.Host`, and `REQUEST_HEADERS:Transfer-Encoding` mirrors the connector's explicit use of `req.TransferEncoding`.
3. **Additional single-value request variables — IMPLEMENTED_TESTING_DEFERRED.** `QUERY_STRING` is reconstructed from `req.URL.RawQuery`; `SERVER_NAME` is reconstructed from the same `req.Host` value that the Coraza HTTP connector passes to `SetServerName`.
4. **Request/connection metadata sources — IMPLEMENTED_TESTING_DEFERRED.** `REQUEST_URI_RAW` and `REQUEST_LINE` use the exact `req.URL.String()`/method/protocol values fed by Coraza's HTTP connector; `REQUEST_BASENAME` mirrors Coraza's parsed-path last-separator behavior; `REMOTE_ADDR` and `REMOTE_PORT` mirror the connector's final-colon `RemoteAddr` split after the existing trusted-proxy resolver has normalized the request.
5. **ARGS/body-oriented groups — NOT IMPLEMENTED.** They remain Coraza-only until parser/normalization parity is formally designed and real-validated.
6. **Chains, negated operators, multi-variable selectors, dynamic macro semantics, and all non-allow-listed transforms — NOT IMPLEMENTED / CORAZA_ONLY.**

The semantic adapter version was bumped for this slice, deliberately invalidating prior persisted learning fingerprints so newly eligible or differently transformed groups must re-enter Learning. Zero observed false negatives remains the hard safety rule; no coverage increase weakens FAILSAFE behavior.

Validation added in source:

- portable unit coverage for transform order/output, source reconstruction, unsupported-transform exclusion, and transform-specific group fingerprint separation;
- a `realcoraza` Phase 2 parity gate that compares the allow-listed transform/source behavior with Coraza v3.7.0 when a qualified Go 1.25 release host is available.

Current packaging host evidence is limited to isolated `internal/vectoraccel` tests/vet/race using the installed Go 1.23.2 toolchain in a temporary module. Full-repository and `realcoraza` execution remain blocked by the project Go 1.25 requirement and are not production evidence.

## Phase 3 — Release engineering and supply-chain hardening

**Status: IMPLEMENTED_TESTING_DEFERRED / RELEASE-HOST GATES OPEN.**

Implemented on 2026-09-11:

- `generate-release-evidence.sh` creates deterministic SPDX 2.3 and CycloneDX 1.5 SBOMs from the pinned Go module graph and, for the native flavor, verified `pkg-config libhs` provenance;
- release evidence records the module Go directive, packaging-host Go runtime, Coraza pin, CRS declaration, VectorScan/libhs provenance, NGINX and OpenSSL versions;
- `govulncheck` has an explicit truth-bearing status file and `--require-govulncheck` fail-closed gate; unavailable or failed scans are never reported as PASS;
- `build-release-artifact.sh` now records `Flavor: portable|native`, generates supply-chain evidence inside the artifact, and honors `SOURCE_DATE_EPOCH` so equal inputs produce byte-identical ZIPs;
- `verify-release-artifact.sh` independently validates the Phase 3 evidence files, SBOM schema/version markers, evidence checksums, flavor, source-byte/mode equality, and release manifest;
- portable and native release flavors are explicit. Native flavor evidence fails closed when verified real `libhs` is unavailable;
- `sign-release-artifact.sh` / `verify-release-signature.sh` provide detached SHA-256 signing/verification without generating, embedding, or retaining an organizational private key. Signing remains `NOT_RUN` unless `WAF_RELEASE_SIGNING_KEY` is explicitly supplied;
- `phase3-release-tooling-test.sh` exercises deterministic evidence generation, SBOM validity markers, portable flavor semantics, and the unsigned signing gate.

Executed on the current packaging host: portable evidence generation PASS; two complete-source builds with identical `SOURCE_DATE_EPOCH` produced identical ZIP SHA-256; artifact verifier PASS; native flavor gate BLOCKED as expected because real `libhs` is unavailable; required `govulncheck` gate BLOCKED because `govulncheck` is unavailable; signing NOT_RUN because no organizational signing key is configured.

Still required before a production release claim: run `govulncheck` on a network/module-cache capable Go 1.25 release host, build/verify the native flavor on a host with real VectorScan/libhs, and perform organizational detached signing if that process is adopted. Phase 0/1/2 real Coraza/VectorScan/CRS qualification remains independent and open.

## Phase 4 — Existing security/product backlog

**Status: IMPLEMENTED_TESTING_DEFERRED / DEPLOYMENT_QUALIFICATION_REQUIRED (owner-prioritized 2026-09-11).**

Implemented in this slice:

- L7 abuse controls: per-normalized-client-IP request rate and in-flight caps plus direct-peer TCP connection/TLS-handshake caps; limiter identity maps are bounded to avoid attacker-driven unbounded memory growth;
- manual IPv4/IPv6 CIDR allow/deny with allow-over-deny precedence, RFC3339 expiry, and audited operator list/upsert/delete API;
- WAF-generated `X-WAF-Request-ID`, client value replacement, response/access/match/syslog propagation, and exact Coraza transaction-ID injection;
- configurable bounded custom HTML block page used by Phase 4 controls and AI blocklist responses;
- atomic persistent security state for AI blocklist, learner aggregates, notifications, SHA-256 session-token hashes/expiry, audit history and cumulative security counters;
- PKI Slice 3: HTTPS-only SSRF-hardened CRL URL retrieval with DNS validation/pinned dial, bounded fetch, refresh scheduling/deduplication, last-known-good memory + disk cache, status/manual refresh API and audit/RBAC;
- deployment wiring with `StateDirectory=waf-proxy` under the existing `ProtectSystem=strict` sandbox.

Executed isolated security/PKI/persistence race suites PASS on the packaging host. Still required: full Go 1.25 repository regression/vet; exact real-Coraza request-ID correlation execution; positive external CRL fetch/rotation; deployed Debian/Ubuntu restart/permissions/HA/load validation; `govulncheck`; and all pre-existing Phase 0/1/2 real Coraza/VectorScan/CRS gates.

## Deferred architecture work

- XDP prefilter: DEFERRED. It is complementary to VectorScan, not an alternative currently under selection.
- AF_XDP / DPDK / SmartNIC/DPU / GPU: DEFERRED unless a future throughput target justifies the architectural cost.
- Deep Coraza fork: do not do this. Keep Coraza authoritative and VectorScan optional/fallback-safe.
- QAT hardware-specific optimization: only qualify on actual supported hardware; software/kTLS/QAT frontend modes must remain optional.

## Current roadmap position — 2026-09-11

The current development line has now entered **Phase 4 by explicit owner direction** while earlier real-host qualification dependencies remain open:

1. **Phase 0 — QUALIFICATION_REQUIRED / BLOCKED on this host.** Real Go 1.25 / Coraza v3.7.0 / libvectorscan execution remains `NOT_RUN`.
2. **Phase 1 implementation — IMPLEMENTED_TESTING_DEFERRED; production qualification — QUALIFICATION_REQUIRED.** Differential/zero-FN harness exists; representative real CRS replay remains `NOT_RUN`.
3. **Phase 2 conservative VectorScan coverage — IMPLEMENTED_TESTING_DEFERRED / QUALIFICATION_REQUIRED.** Two conservative coverage increments are in source; real Coraza/libvectorscan/CRS parity remains open. ARGS/body/chains/negation/multi-variable expansion is still NOT STARTED and Coraza-only.
4. **Phase 3 release engineering — IMPLEMENTED_TESTING_DEFERRED.** SBOM/provenance/reproducibility/flavor/signing tooling exists; release-host govulncheck/native/signing evidence remains open.
5. **Phase 4 security/product backlog — IMPLEMENTED_TESTING_DEFERRED / DEPLOYMENT_QUALIFICATION_REQUIRED.** L7 controls, CIDR governance, exact request correlation, custom block responses, persistent security state and PKI CRL URL lifecycle are implemented. Isolated race suites pass, while full Go 1.25, external CRL and deployed Linux gates remain open.
6. **Debug & Evidence / Supportability — IMPLEMENTED_TESTING_DEFERRED.** Cross-cutting foundations remain in parallel.

## Immediate next development instruction

Treat Phase 4 source as implemented but not deployment-qualified. On a suitable Go 1.25 Linux host, first run the full repository test/vet/race matrix, the request-ID→real-Coraza transaction correlation regression, positive HTTPS CRL refresh/rotation and restart persistence under the shipped systemd sandbox. Keep Phase 0/1/2 real Coraza/VectorScan/CRS qualification independent and open. Do not expand VectorScan into ARGS/body/chains merely because Phase 4 feature work is present.

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

## Cross-cutting implementation status — Debug / Supportability

**Status: IMPLEMENTED_TESTING_DEFERRED.**

Current Stage 2 source contains:

- bounded Debug Evidence capture/export foundations and verdict-path isolation;
- Coraza transaction-final `MatchedRules()` evidence hook after `ProcessLogging()`;
- VectorScan candidate-coverage comparison and zero-false-negative result logic;
- request/response/TLS/proxy evidence schema foundations, masking, tenant-scoping and TTL cleanup foundations;
- `wafctl debug export`, `wafctl doctor`, and `wafctl support bundle` command foundations.

Still required before these are production-qualified:

- full runtime dependency/status wiring for `wafctl doctor`;
- production incident/support bundle content and masking validation;
- live tenant-isolation/TTL behavior qualification;
- real Go 1.25 / Coraza v3.7.0 / libvectorscan execution;
- representative CRS/traffic differential replay with **zero observed false negatives**.

These cross-cutting features do not move the main VectorScan roadmap past Phase 1.
