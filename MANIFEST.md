# waf-proxy — package manifest

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

A Coraza-based reverse-proxy WAF with an embedded admin console, delivered as
source you build into `waf-proxy` plus the optional `waf-tlsfront` companion. **Start with `INSTALL.md`.**

## Status

Current source baseline: Coraza v3.7.0 / Go 1.25.0 with optional VectorScan Learning Accelerator. Earlier baselines recorded portable Coraza API-stub regression/vet/race and native libhs ABI compile/vet/race PASS evidence; those are historical and are not re-labeled as exact-current-baseline PASS. Real Go 1.25 + Coraza v3.7.0 execution and real libvectorscan matching are **NOT_RUN** in the isolated packaging environment and remain release-host gates. Coraza is always authoritative; VectorScan false negatives/scan errors fail the affected group to Coraza-only `FAILSAFE`.

## Read in this order

1. `INSTALL.md` — prerequisites, build, install, CRS, first login, verification, rollout.
2. `README.md` — feature reference (all tabs and subsystems).
3. `config.sample.json` — annotated starting config.

## Contents

Docs
- `INSTALL.md`      — install & operations guide
- `README.md`       — full feature reference
- `MANIFEST.md`     — this file
- `RELEASE_PROCESS.md` — mandatory release artifact packaging-integrity gate
- `DEVELOPMENT.md`    — cross-cutting engineering ledger and release rules
- `TESTING.md`        — test-policy requirements; executed evidence remains in TESTING_RESULTS.md

Build & deploy
- `build.sh`            — go mod tidy -diff → vet → test/race → binary build; deterministic local-Go minimum check; native VectorScan requires runtime libhs
- `qualify-release-host.sh` — Phase 0 preflight/core real-host qualification; BLOCKED vs FAIL truth boundary
- `build-release-artifact.sh` — deterministic SOURCE_ARCHIVE ZIP builder with source-bound evidence, generated manifest and mandatory verification
- `verify-release-artifact.sh` — post-packaging integrity gate including manifest completeness, provenance binding and SBOM↔go.mod parity
- `install.sh`         — idempotent installer (user, dirs, unit, admin token)
- `uninstall.sh`       — removal (`--purge` for config/user)
- `setup-interfaces.sh`— interactive mgmt/data NIC separation (self-signed admin cert, drop-in)
- `waf-doctor.sh`      — one-shot repair+verify of ownership/perms/unit/capability (run if anything misbehaves)
- `upgrade.sh`         — routine upgrade: rebuild + swap binary, auto-detects if unit/rules changed
- `waf-proxy.service`  — hardened systemd unit
- `waf-tls-frontend.service` — hardened optional NGINX/OpenSSL TLS-frontend systemd unit
- `cmd/waf-tlsfront/`  — TLS frontend supervisor/preflight/render CLI
- `internal/tlsfront/` — TLS frontend config, capability probe, kTLS/QAT resolution, NGINX renderer
- `tls_frontend_control.go` — waf-proxy live-control publisher and frontend runtime status reader
- `tls_acceleration_test.go` — TLS listener remap/metadata/proxy/control regression tests
- `config.sample.json` — sample production config
- `coraza.conf`        — engine config (includes OWASP CRS from /etc/waf/crs)
- `go.mod`             — Go 1.25 module + pinned Coraza v3.7.0

Go source (module `waf-proxy`; Coraza plus optional native VectorScan/libhs)
- `main.go`       — config schema, runtime build, listeners, health, drain, shutdown
- `coraza_observer.go` — transaction wrapper that observes final `MatchedRules()` after `ProcessLogging()` for Learning truth
- `coraza_real_gate_test.go` — real-Coraza DetectionOnly+nolog `MatchedRules()` release gate (`realcoraza` tag)
- `internal/vectoraccel/scanner_native_gate_test.go` — native vectorscan+cgo multi-pattern compile/scan semantic release gate
- `internal/capability/` — single source of truth for VectorScan eligibility shared by coverage analysis and runtime parsing
- `internal/vectoraccel/` — grouped VectorScan scanner, Learning/VALIDATED/ACCELERATED/FAILSAFE state machine, persistence and native/stub adapters; eligibility delegates to `internal/capability`
- `proxy_buffer.go` — fixed 32 KiB ReverseProxy response-copy buffer pool (P0-A)
- `observations.go` — bounded drop-on-full async host/sitemap/learner/signal observation plane (P0-C)
- `observations_test.go` — P0-C queue/drain/allocation/regression tests + benchmarks
- `matchlog.go`     — bounded/rate-limited asynchronous Coraza match-log aggregation plane (P0-D)
- `matchlog_test.go` — P0-D aggregation/saturation/no-sync-log tests + benchmarks
- `pool_hotpath_test.go` — P0-B LB allocation/accounting/target regression tests + benchmarks
- `p1_tuning_test.go` — P1 response-body policy + backend Transport/config/UI regression tests
- `release_scripts_test.go` — LF/executable/Bash syntax release-script regression gate
- `admin.go`      — admin API, auth middleware, match/access rings, console serving
- `ai.go`         — async AI traffic analysis (fail-open, lazy match/sample capture, atomic blocklist reads, redaction, queue-drop telemetry, per-site modes)
- `pool.go`       — zero-allocation load balancing + active accounting + shared pool-scoped backend Transport + health monitors (P0-B/P1)
- `sitemap.go`    — passive path discovery + polite crawler
- `learn.go`      — policy-fit learner (false-positive vs attack, exclusion suggestions)
- `profiles.go`   — content-driven page-profile catalog + classifier
- `notify.go`     — notifications (bell + webhook) with syslog sink
- `ha.go`         — config sync + active/standby role (no VIP movement)
- `syslog.go`     — async fail-open syslog forwarding to a SIEM (UDP/TCP/TLS)
- `users.go`      — users, RBAC, PBKDF2 auth, sessions, audit ring
- `watchdog.go`   — opt-in hardware watchdog / bypass-NIC heartbeat feeder
- `update.go`     — signed self-update wiring (admin + localhost)
- `discovery.go`  — passive observed-hostnames flag (undeclared-host visibility)
- `internal/sigupdate/` — shared signed-update engine (copied unchanged; RSA/SHA-256)
- `static/admin.html`   — embedded shipping admin console shell
- `static/theme.css`    — embedded canonical console theme, served at `/theme.css`

Optional
- `web/`          — Vite+preact console SCAFFOLD (not the shipping UI; imports the same theme tokens)

## Quick start

```bash
unzip waf-proxy-<release>.zip -d waf-proxy
cd waf-proxy
./build.sh          # needs Go >= 1.25.0; WAF_VECTORSCAN=auto|off|required
sudo ./install.sh   # prints the admin token — save it
sudo ./setup-interfaces.sh   # two-NIC appliance: pin mgmt/data planes (optional)
# then fetch OWASP CRS and start — see INSTALL.md §4–§5
```

## Performance benchmark tooling

- `cmd/wafbench/`       — repeatable full-WAF, direct-Coraza, L4, backend and comparison CLI
- `benchmark/README.md` — benchmark matrix, pprof workflow and interpretation guide
- `benchmark/build.sh`  — builds `bin/wafbench` without changing the production WAF binary

Handover / continuation documentation:

- `DEVELOPMENT_ROADMAP.md` — selected VectorScan direction, release-host qualification and future development phases
- `TESTING_RESULTS.md` — evidence ledger separating real/stub/ABI-only/NOT_RUN gates
- `HANDOVER_STATUS.md` — concise current-state checkpoint for a new chat
- `HANDOVER_PROMPT.md` — copy/paste bootstrap prompt for a new development chat

## 2026-09-13 supportability / qualification additions

- `debug_bundle.go` / `debug_bundle_test.go` — bounded tenant/site-scoped transaction evidence, TTL cleanup, incident ZIP export and tests.
- `support_api.go` — authenticated doctor and debug-control/export admin endpoints.
- `cmd/wafctl/` — operator CLI for doctor, bounded debug capture/export and sanitized support bundles.
- `cmd/wafqualify/` — real Coraza vs native VectorScan differential corpus runner.
- `qualification/corpus/default.jsonl` / `qualification/README.md` — representative starter corpus and qualification semantics.
- `run-phase1-qualification.sh` — release-host wrapper enforcing Go/libhs prerequisites and PASS/FAIL/BLOCKED semantics.
- `coraza_observer.go` — final Coraza evidence is captured even without a VectorScan observation; VectorScan absence remains explicit.


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

## Phase 2 Coverage Expansion Slice C additions

Source additions/changes:

- `internal/coverage/capability.go` — runtime-parity conservative eligibility rules.
- `internal/coverage/crs/model.go` — normalized ruleset/rule/warning/duplicate metadata.
- `internal/coverage/crs/parser.go` — SecRule metadata parser.
- `internal/coverage/crs/loader.go` — bounded directory/config ingestion with Include/IncludeOptional handling.
- `internal/coverage/crs/inventory.go` — versioned capability inventory generation.
- `internal/coverage/crs/report.go` — Coverage Report v2.
- `internal/coverage/crs/analyzer.go` — current-runtime capability mapping.
- `cmd/wafctl/coverage.go` — offline `wafctl coverage analyze` operator command.
- focused parser/loader/capability tests are included for the final validation stage.

These files do not enable acceleration by themselves; Coraza remains authoritative and Phase 1 zero-FN qualification remains mandatory.

## Phase 3 release-hardening files

- `tools/source_manifest.py` — canonical source-file enumeration and source-manifest digest helper.
- `tools/release_evidence.py` — schema-v2 release evidence, SOURCE_ARCHIVE identity, source-bound provenance, SPDX 2.3 and CycloneDX 1.5 SBOM generator.
- `release-security-scan.sh` — source-manifest-bound govulncheck PASS/FAIL/BLOCKED/NOT_RUN evidence runner.
- `release-artifact-negative-tests.sh` — automated corrupt/malicious ZIP plus truth-boundary mutation rejection tests.
- `verify-reproducible-source-release.sh` — fixed-epoch two-build byte-reproducibility gate.
- `build-release-artifact.sh` — hardened deterministic SOURCE_ARCHIVE packager; binary identities are refused; optional detached minisign signing is external to the archive identity.
- `verify-release-artifact.sh` — hardened post-package archive/source/evidence verifier with exact source/release manifest coverage, provenance cross-digests and SBOM dependency checks.
- `verify-release-signature.sh` — detached minisign verification with an independently supplied approved public key; returns NOT_CONFIGURED when minisign is unavailable.
- `BUILD_PROVENANCE.json` / `BUILD_SHA256SUMS.txt` — generated only after a successful binary build and intentionally excluded from complete-source ZIP staging.


Phase 4 Slice A follow-up: added client identity audit event model constants CLIENT_IDENTITY_RESOLVED and CLIENT_IDENTITY_HEADER_REJECTED.

## Phase 4 Slice B Added Files

- l7_abuse.go
- l7_abuse_test.go

Modified:
- main.go

## Phase 4 Slice C Roadmap Documentation Update

Added roadmap references:
- Manual CIDR Policy boundary
- Slice C scope definition

No Slice C source files added.


Phase 4 Slice C source additions:
- cidr_policy.go

Phase 4 Slice D artifacts:
- block_response.go
- correlation.go
- security_event.go


## Phase 4 Slice E — Persistent Security State

Status: IMPLEMENTED_TESTING_DEFERRED

Implemented foundation: security state models and in-memory persistence abstraction. Production database durability, HA replication, retention tuning, and external integrations remain deferred.


Phase 4 Slice F additions:
- pki_hardening.go
- pki_hardening_test.go


## Phase 5 Slice A — Runtime Qualification Closure

Status: IMPLEMENTED_TESTING_DEFERRED

Implemented: runtime qualification evidence schema foundation. Real Go 1.25, Coraza v3.7.0 transaction, and libvectorscan runtime gates remain NOT_RUN until executed on a qualified release host.


Phase 5 Slice B — VectorScan Production Qualification
Status: IMPLEMENTED_TESTING_DEFERRED
Documentation update: qualification boundary recorded.


## Phase 5 Slice C — Enterprise Deployment Readiness (IMPLEMENTED_TESTING_DEFERRED)

Added deployment readiness evidence foundation:
- preflight report model
- health/readiness evidence boundary
- deployment diagnostics foundation

Runtime deployment qualification remains NOT_RUN until executed on target environments.


## Phase 5 Slice D Update

Added security operations foundation source artifacts and roadmap documentation.


## Phase 5 Slice E Update

Added reliability qualification source and evidence boundary documentation.

## Phase 5 Slice F — Performance Certification

Added:

- `cmd/wafbench/certify.go` — hash-bound performance evidence, target evaluation, comparability and regression model.
- `cmd/wafbench/certify_test.go` — certification truth-boundary tests.
- `qualification/performance/README.md` — certification evidence contract.
- `qualification/performance/target.example.json` — explicitly non-approved target schema example.
- `qualification/performance/performance-certification-NOT_RUN.json` — no-measurement placeholder.

Modified:

- `cmd/wafbench/main.go` — `certify` command wiring.
- `cmd/wafbench/model.go` — wafbench tool evidence version 1.1.0.
- `benchmark/README.md`, `README.md` — certification workflow documentation.
- canonical handoff/roadmap/testing/release Markdown for Slice F status and truth boundary.

Source baseline repair:

- restored `qualification/vectorscan/{differential.go,lifecycle.go,replay.go,report.go,vectorscan_test.go}` from the stale nested baseline copy;
- corrected `replay.go` to the canonical `vectorscan` package;
- removed obsolete nested `waf-work/` source mirror from the deliverable source tree.

No XDP, DPDK, kernel-bypass, hardware-offload, Coraza-verdict, VectorScan-promotion, or TLS dataplane behavior was added by Slice F.

Slice F mode repair:

- restored executable mode `0755` for `build.sh`, `build-release-artifact.sh`, `verify-release-artifact.sh`, `verify-release-signature.sh`, `release-security-scan.sh`, `release-artifact-negative-tests.sh`, `verify-reproducible-source-release.sh`, `qualify-release-host.sh`, `run-phase1-qualification.sh`, `install.sh`, `upgrade.sh`, `uninstall.sh`, `waf-doctor.sh`, `setup-interfaces.sh`, `benchmark/build.sh`, `tools/release_evidence.py`, and `tools/source_manifest.py`.

## Phase 5 Slice G — External HSM / PKCS#11 additions

Source/runtime:

- `internal/hsm/config.go` — PKCS#11 runtime/key configuration and module-path policy.
- `internal/hsm/provider.go` — provider/signer interfaces, TLS certificate binding, health and restricted audit model.
- `internal/hsm/pkcs11.go` — session/login/key lookup/sign lifecycle and RSA/ECDSA signing plans.
- `internal/hsm/pkcs11_linux_cgo.go` — Linux native PKCS#11 loader (`pkcs11` build tag).
- `internal/hsm/pkcs11_stub.go` — non-native/disabled fail-closed provider stub.
- `internal/hsm/module_security_linux.go` — Linux root-ownership/module-path checks.
- `internal/hsm/secret.go` — env/file secret-reference validation and bounded secret loading.
- `hsm_integration.go` / `hsm_integration_test.go` — WAF config/runtime/audit integration and no-fallback/redaction tests.
- `cmd/hsmqualify/main.go` — real PKCS#11 provider/TLS qualification runner.
- `qualification/hsm/` — SoftHSM and real-vendor qualification scripts, documentation and explicit `NOT_RUN` evidence placeholders.

Build/runtime truth:

- native HSM capability requires Linux + CGO + the `pkcs11` build tag; `build.sh` exposes this as `WAF_HSM_PKCS11=off|auto|required`;
- the vendor PKCS#11 module is runtime-loaded and is not vendored into the source artifact;
- no private key, PIN, vendor library, token database, generated certificate, or production secret is included in the source package;
- SoftHSM and real vendor qualification are not claimed unless their corresponding runners are actually executed.

Slice G build/release integration:

- `build.sh` — independent `WAF_HSM_PKCS11=off|auto|required` build capability and PKCS#11 provenance fields.
- `verify-release-artifact.sh` — mandatory HSM source/qualification file presence and executable-mode checks.
- `release_scripts_test.go` — both HSM qualification runners included in shipped-script regression checks.
- `admin.go` / `syslog.go` — HSM status/audit operator surfaces and restricted audit forwarding.
- `config.sample.json` — global HSM module allow-list and per-site key-provider schema example.

## Enterprise Linux Distribution Packaging — Slice A additions

- `packaging/deb/build-release-deb.sh` — qualified-host binary build + deterministic DEB wrapper.
- `packaging/deb/build-deb.sh` — provenance/checksum-bound deterministic binary `.deb` builder.
- `packaging/deb/verify-deb.sh` — extracted-content, conffile, service-path, state-directory, secret and network-fetch verifier.
- `packaging/deb/tests/test-deb-packaging.sh` — reproducible fixture package and negative native-dependency test.
- `packaging/deb/maintainer/postinst` — offline-safe service user/directory/secret lifecycle; no fresh autostart.
- `packaging/deb/maintainer/prerm`, `postrm` — safe remove/upgrade lifecycle preserving persistent state.
- `packaging/deb/config/waf-tls-frontend.env` — packaged non-secret frontend environment template.
- `packaging/deb/CRS-PROVISIONING.md`, `packaging/deb/README.md` — operator/package build guidance.
- `DEB_PACKAGE_GATE_RESULT.md` — package-slice executed/deferred gate ledger.
- `waf-proxy.service` — persistent `StateDirectory=waf-proxy`.
- `install.sh`, `waf-doctor.sh` — persistent state/source-vs-package path compatibility updates.

No production secret, generated token, CRS runtime tree, `.deb` fixture, or compiled WAF binary is included in the complete source baseline.


## Enterprise Linux Distribution Packaging — Slice B additions

- `packaging/rpm/waf-proxy.spec` — binary RPM spec, `%config(noreplace)`, service-user lifecycle, controlled service restart, persistent state and offline scriptlets.
- `packaging/rpm/build-release-rpm.sh` — qualified-host Go build + RPM wrapper.
- `packaging/rpm/build-rpm.sh` — provenance/checksum/ELF-architecture-bound deterministic RPM builder.
- `packaging/rpm/verify-rpm.sh` — extracted content, scriptlets, config flags, service paths, state path, secret and CRS verifier.
- `packaging/rpm/validate-rpm-source.py` — RPM source-contract verifier usable even without an installed RPM toolchain.
- `packaging/rpm/tests/test-rpm-source.sh` — shell/source/security policy test.
- `packaging/rpm/tests/test-rpm-packaging.sh` — real rpmbuild fixture reproducibility and negative native-dependency test; reports BLOCKED when RPM tools are unavailable.
- `packaging/rpm/config/waf-tls-frontend.env` — non-secret packaged frontend template.
- `packaging/rpm/CRS-PROVISIONING.md` — network-free CRS provisioning boundary.
- `packaging/rpm/SELINUX.md` — enforcing-SELinux qualification and no-scriptlet-policy-mutation boundary.
- `packaging/rpm/README.md` — RHEL-family package build/install guidance.
- `RPM_PACKAGE_GATE_RESULT.md` — executed/BLOCKED/NOT_RUN gate ledger.
- `verify-release-artifact.sh` / `release_scripts_test.go` — DEB/RPM package builders/verifiers/tests are now critical shipped release files.

No RPM binary, generated token, vendor HSM library, CRS runtime tree, or compiled WAF binary is included in the complete source baseline.


## Enterprise Linux Distribution Packaging — Slice C additions

- `packaging/qualification/package_lifecycle_qualify.py` — guarded DEB/RPM upgrade, failed-upgrade, recovery and rollback evidence runner.
- `packaging/qualification/run-package-lifecycle-qualification.sh` — operator wrapper.
- `packaging/qualification/build-lifecycle-fixtures.sh` — deterministic N/N+1/failure semantic fixture builder.
- `packaging/qualification/tests/test_package_lifecycle.py` — lifecycle helper tests.
- `packaging/qualification/tests/test-qualification-source.sh` — source/security/truth-boundary gate.
- `packaging/qualification/README.md` — dedicated-host operating guide and truth boundaries.
- `qualification/package-lifecycle/*-NOT_RUN.json` — separate DEB/RPM real-execution placeholders.
- `PACKAGE_LIFECYCLE_GATE_RESULT.md` — current executed/BLOCKED/NOT_RUN Slice C gate ledger.
- `packaging/rpm/waf-proxy.spec` / `build-rpm.sh` — qualification-only `%post` failpoint compiled only into explicitly guarded fixture builds.
- `verify-release-artifact.sh` / `release_scripts_test.go` — new lifecycle qualification sources are now required/mode/syntax-checked release files.

No lifecycle `.deb`/`.rpm` fixture, package-manager database, generated admin
token, persistent-state marker, or qualification-host local artifact is included
in the complete source baseline.

## Enterprise Linux Distribution Slice D — Clean-host qualification

- `packaging/cleanhost/README.md` — operator/truth-boundary guide.
- `packaging/cleanhost/clean_host_qualify.py` — exact-distro clean-host install/start/auth/traffic/upgrade/remove runner.
- `packaging/cleanhost/run-clean-host-qualification.sh` — operator entrypoint.
- `packaging/cleanhost/tests/test_clean_host_qualify.py` — unit tests.
- `packaging/cleanhost/tests/test-clean-host-source.sh` — source/security contract gate.
- `qualification/clean-host/matrix.json` — seven-platform NOT_RUN matrix.
- `qualification/clean-host/*-NOT_RUN.json` — per-platform truth-boundary placeholders.
- `CLEAN_HOST_DISTRIBUTION_GATE_RESULT.md` — executed/deferred clean-host gate ledger.

## Project-local package builder additions

- `waf-package` — executable project-root entry point for `doctor`, `deb`, `rpm`, and `all`.
- `tools/waf_package_builder.py` — fail-closed package build orchestrator.
- `tools/tests/test_waf_package_builder.py` — Python unit tests for source/version/package orchestration logic.
- `tools/tests/test-waf-package-source.sh` — shell/source/CLI contract gate.
- `PACKAGING_TOOL.md` — operator/developer guide for rebuilding DEB/RPM after source changes.

## Root Build Integrity Repair (2026-09-17)

Repair-relevant source:
- `.github/workflows/ci.yml` — mandatory module-tidy drift + root build/test CI;
- `go.mod` / `go.sum` — canonical Coraza v3.7.0 dependency graph;
- `pki.go` / `pki_url.go` — matched CRL URL refresh/store implementation;
- `pki_url_phase4_test.go` — URL/SSRF/LKG/concurrency/cache regression coverage;
- `debug_bundle.go` — current evidence model plus TLS version naming helper;
- `debug_evidence_ops_v2.go` — current store API/model mapping;
- `debug_evidence_ops_v2_test.go` — integration regression tests.

## OpenAI connector hardening files

Added: `internal/openaiapi/responses.go`, `internal/openaiapi/responses_test.go`, `internal/secretref/secretref.go`, `internal/secretref/secretref_test.go`, `ai_openai_integration_test.go`, `tools/tests/test-openai-integration-source.sh`, `OPENAI_INTEGRATION_GATE_RESULT.md`. Modified: `ai.go`, `admin.go`, `config.sample.json`, `static/admin.html`, release verifier/tests, and canonical documentation.
