# waf-proxy — package manifest

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
| API-3 OpenAPI Contract Management | `IMPLEMENTED_TESTING_DEFERRED` | persistence, import/export, params/enums/security, expanded drift; API-3 47-check source gate + deterministic harness PASS |
| API-4 Positive Schema Enforcement | `IMPLEMENTED_TESTING_DEFERRED` | reviewed immutable profiles; LEARN/DETECT/ENFORCE, exceptions, rollback, bounded violation evidence; local API-4 gates PASS |
| API-5 JWT + Identity-aware Policy | `IMPLEMENTED_TESTING_DEFERRED` | verified JWT/JWKS identity context, DETECT/ENFORCE operation policy, bounded durable evidence; API-5 17-test + race + 72-check source gate PASS |
| API-6.1 Sequence Foundation | `IMPLEMENTED_TESTING_DEFERRED` | `sequence_api61.go`, targeted tests and source gate; Go execution blocked by missing toolchain |
| API-6.2 Workflow Learning | `IMPLEMENTED_TESTING_DEFERRED` | LEARN-only workflow model; source gate 56 PASS; Go 1.25 targeted/race BLOCKED_ENVIRONMENT/NOT_RUN |
| API-6.3 Sequence Anomaly Detection | `IMPLEMENTED_TESTING_DEFERRED` | DETECT-only bounded anomaly evidence; source gate 45 PASS; Go 1.25 targeted/race `BLOCKED_ENVIRONMENT/NOT_RUN` |
| API-6.4 Sequence Operations + Hardening | `IMPLEMENTED_TESTING_DEFERRED` | explicit LEARN/DETECT controls, exception CRUD, reset/relearn, recent sessions and operations Console; source gate 58 PASS |
| API-7.1 Object Locator Discovery | `IMPLEMENTED_TESTING_DEFERRED` | normalized locator discovery + bounded keyed evidence; source gate 83 PASS; Go 1.25 targeted/race BLOCKED_ENVIRONMENT/NOT_RUN |
| API-7.2–7.4 BOLA Relationship/Detection/Policy | `IMPLEMENTED_TESTING_DEFERRED` | verified identity/object relationship, bounded BOLA detection, REVIEW/SUPPRESS evidence policy/workflow/console; source gates 100/110/156 PASS |
| API-8 GraphQL Security | `IMPLEMENTED_TESTING_DEFERRED` | bounded GraphQL parsing/normalization, SDL contract binding, depth/complexity/field/mutation/subscription/introspection/APQ controls, verified-variable API-7 bridge, staged deterministic LEARN/DETECT/ENFORCE; source gate 259 PASS; Go 1.25 targeted/race `BLOCKED_ENVIRONMENT/NOT_RUN` |

API-1 → API-2 → API-3 deterministic end-to-end learning/drift evidence is PASS. API-4 adds reviewed positive-schema runtime profiles with local 7/7 deterministic tests, targeted race PASS, and a 46-check source gate. The repository-wide Source Buildability Gate remains **BLOCKED**, not PASS: this host has Go 1.23.2 while `go.mod` requires Go 1.25.0, and network/toolchain acquisition is unavailable. Required Go 1.25 `tidy/build/vet/test/race/real-Coraza` qualification remains `NOT_RUN/BLOCKED`. See `API_SECURITY_CLOSURE_RESULT.md`.

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



### API Security source closure files — 2026-09-23

- `api_operations.go` / `api_operations_test.go` — API-1 operation normalization, metadata, controls and deterministic tests.
- `api_security_persist.go` — durable API Security JSON state persistence/autosave through API-6.1.
- `api_security_handlers.go` — API-1 control and API-2 review endpoints.
- `schema_api2.go` / `schema_api2_test.go` — live typed schema learning, lifecycle, privacy and tests. The unused `schema_api2_continuation.go` helper layer was removed during the 2026-09-28 duplication consolidation.
- `ai_schema_review.go` — advisory-only OpenAI SchemaCandidate review.
- `api3_contract.go` / `api3_export.go` / `api3_contract_test.go` — OpenAPI contract intelligence, export, diff/drift and tests.
- `api_security_integration_test.go` — deterministic API-1 → API-2 → API-3 live-learning/drift test.
- `tools/tests/test-api12-source.py` / `tools/tests/test-api3-source.py` — executable source/integration gates.
- `sequence_api61.go` / `sequence_api61_test.go` — bounded/private sequence session and transition foundation plus targeted deterministic/restart/bounded/concurrency tests.
- `tools/tests/test-api61-source.py` — API-6.1 source contract gate retained by `build.sh` and CI.
- `API61_SEQUENCE_FOUNDATION.md` / `API61_SOURCE_GATE_RESULT.md` — API-6.1 implementation and evidence boundary.
- `API_SECURITY_CLOSURE_RESULT.md` — current implementation and qualification truth boundary.
- `DELIVERY_MANIFEST.json` — generated per-file SHA-256/size/mode manifest plus local qualification summary for the delivered source baseline.

Runtime API Security state files are created adjacent to `config.json`, including `api-operations.json`, `api-schema.json`, `api-contracts.json`, `api-positive-schema.json`, `api-identity.json`, `api-sequence.json`, and `api-sequence.key`. They are runtime state and are not pre-populated source-package files.

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
- `security_event.go` — removed 2026-09-28; the unused generic event placeholder duplicated the wired security-specific evidence paths.


## Phase 4 Slice E — Persistent Security State

Status: IMPLEMENTED_TESTING_DEFERRED

Implemented foundation: security state models and in-memory persistence abstraction. Production database durability, HA replication, retention tuning, and external integrations remain deferred.


Phase 4 Slice F historical additions (superseded):
- `pki_hardening.go` / `pki_hardening_test.go` were isolated helper/test files and were removed 2026-09-28. The current CRL URL hardening authority is `pki.go` + `pki_url.go`.


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
- `debug_bundle.go` / `debug_sanitize.go` — current bounded debug evidence store, export model and sanitization authority.
- The superseded `debug_evidence_ops_v2.go` compatibility wrapper and its test were removed 2026-09-28.

## OpenAI connector hardening files

Added: `internal/openaiapi/responses.go`, `internal/openaiapi/responses_test.go`, `internal/secretref/secretref.go`, `internal/secretref/secretref_test.go`, `ai_openai_integration_test.go`, `tools/tests/test-openai-integration-source.sh`, `OPENAI_INTEGRATION_GATE_RESULT.md`. Modified: `ai.go`, `admin.go`, `config.sample.json`, `static/admin.html`, release verifier/tests, and canonical documentation.

## API-3 OpenAPI Contract Management files

- `api3_contract.go` — OpenAPI parser/import/version store, API-1 binding, version/schema diff, API-2 drift reporting, and authenticated admin handlers.
- `api3_contract_test.go` — prepared parser/import/matching/diff/drift Go tests.
- `tools/tests/test-api3-source.py` — executable source/wiring/security contract gate.
- `API3_OPENAPI_CONTRACT_IMPLEMENTATION.md` — implementation/security boundary.
- `API3_TEST_MATRIX.md` — final Go qualification test matrix.
- `API3_ACCEPTANCE_CRITERIA.md` — promotion requirements.
- `API3_SOURCE_GATE_RESULT.md` — executed source/static evidence.

`github.com/goccy/go-yaml v1.18.0` is now a direct module dependency because API-3 imports YAML OpenAPI documents.


Local executable evidence for this checkpoint is captured in `TESTING_RESULTS.md`; root Go 1.25 blocker evidence is captured in `SOURCE_BASELINE_GATE_RESULT.md`.


## API Security API-1 → API-3 closure files (2026-09-23)

- `api_operations.go` / `api_operations_test.go` — durable API operation discovery, normalization, metadata, controls, and atomic-persistence coverage.
- `schema_api2.go` / `schema_api2_test.go` — live typed schema learning, bounded aggregates, privacy/lifecycle logic, persistence evidence, and deterministic tests; the unused continuation helper was removed 2026-09-28.
- `api_security_persist.go` — versioned API-1/API-2/API-3 state persistence with unique same-directory temp files, fsync, and atomic rename.
- `api_security_handlers.go` / `api_security_handlers_test.go` — API-1/API-2 control-plane handlers and lifecycle checks.
- `ai_schema_review.go` — advisory-only OpenAI SchemaCandidate review using existing Responses API Structured Outputs.
- `api3_contract.go` / `api3_export.go` / `api3_contract_test.go` — durable OpenAPI contract intelligence, export, binding, diff, drift, security/content-type semantics, and hardening tests.
- `api_security_integration_test.go` — API-1 → API-2 → API-3 live-learning/drift deterministic integration coverage.
- `tools/tests/test-api12-source.py` — 69-check API-1/API-2 source/wiring/security gate.
- `tools/tests/test-api3-source.py` — 47-check API-3 source/wiring/security gate.
- `API_SECURITY_CLOSURE_RESULT.md`, `TESTING_RESULTS.md`, `SOURCE_BASELINE_GATE_RESULT.md` — current source closure truth and qualification evidence.

API-1 through API-8 are implemented in source and remain `IMPLEMENTED_TESTING_DEFERRED`. API-6 learned sequence and API-7 inferred BOLA evidence remain non-enforcing; API-8 blocks only under explicit deterministic ENFORCE policy. Repository-root Go 1.25 qualification remains BLOCKED/NOT_RUN as documented; no release-readiness claim is made.

## API-4 Positive Schema Enforcement files

- `positive_schema_api4.go` / `positive_schema_api4_test.go` — reviewed immutable positive-schema profiles, LEARN/DETECT/ENFORCE runtime validation, exceptions, rollback and bounded violation evidence.
- `tools/tests/test-api4-source.py` — 46-check API-4 source/wiring/security gate.
- `static/admin.html` / `admin.go` — API-4 operator workflow and authenticated control-plane routes.

## API-5 JWT + Identity-aware API Security files

- `identity_api5.go` — trusted issuer/JWKS registry, JWT cryptographic verification, verified identity context, operation authorization policy, bounded evidence and durable state.
- `identity_api5_test.go` — 17 deterministic API-5 test functions including EdDSA/RSA/PS256/ES256 verification, issuer/audience/time claims, rotation, unknown-`kid` refresh bounding, policy lifecycle, persistence/privacy and fail-closed semantics.
- `sequence_api61.go` / `sequence_api61_test.go` — API-6.1 bounded async sequence sessions/transitions, private correlation, TTL/caps, atomic snapshots, persistence and targeted tests.
- `tools/tests/test-api61-source.py` — 33-check API-6.1 source gate wired into local build and CI.
- `tools/tests/test-api5-source.py` — 71-check API-5 source/wiring/security/CI gate.
- `admin.go` / `static/admin.html` — trusted issuer, JWKS refresh, operation policy, DETECT/ENFORCE and violation workflow.
- `main.go` / `api_security_persist.go` — data-plane middleware, startup restore and autosave integration.

API-1 through API-8 are `IMPLEMENTED_TESTING_DEFERRED`; no API-9 is defined. Canonical Go 1.25 real-dependency qualification remains `BLOCKED_ENVIRONMENT/NOT_RUN`.

- `sequence_api62.go` / `sequence_api62_test.go` — API-6.2 LEARN-only workflow cohorts, confidence/maturity, cold-start, absolute session lifetime, bounded entry/terminal/depth learning, v1->v2 durable migration and seven targeted tests.
- `tools/tests/test-api62-source.py` — 56-check API-6.2 source/authority/privacy/bounded-state gate retained by build and CI.
- `API62_WORKFLOW_LEARNING.md` / `API62_SOURCE_GATE_RESULT.md` — API-6.2 implementation boundary and current evidence truth.

## API-6.3 Sequence Anomaly Detection files

- `sequence_api63.go` / `sequence_api63_test.go` — DETECT-only six-class sequence anomaly evidence, maturity/sample safeguards, exception matching, bounded persistence and seven targeted tests.
- `tools/tests/test-api63-source.py` — 45-check API-6.3 source/authority/privacy/resource/persistence gate retained by local build and CI.
- `API63_SEQUENCE_ANOMALY_DETECTION.md` / `API63_SOURCE_GATE_RESULT.md` — implementation and qualification truth.
- `admin.go` — Reviewer-gated `GET /api/security/sequence/violations` route.
- `sequence_api61.go` / `sequence_api62.go` — additive v3 state/model integration while preserving API-6.1/API-6.2 behavior and v1/v2 restore compatibility.

- `API64_SEQUENCE_OPERATIONS_HARDENING.md` — API-6.4 implementation and hardening truth.
- `API64_SOURCE_GATE_RESULT.md` — API-6.4 source gate evidence.


## API-7.1 Object Locator Discovery checkpoint — 2026-09-24

API-7.1 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. It adds bounded `ObjectLocator` discovery from API-1 normalized path parameters, API-2 typed path/query/body evidence, matched API-3 OpenAPI contracts, and explicit Reviewer-gated operator INCLUDE/SUPPRESS configuration. Locators are keyed by normalized API-1 operation ID plus location/field; raw object values are transient only and durable value evidence is bounded HMAC-SHA256 fingerprint data protected by a separate mode-0600 key. Client-supplied ownership/tenant headers are not authority, and GraphQL `variables.*` discovery is deferred to API-8.

API-7.1 has no identity/object relationship verdict, tenant-boundary verdict, `BOLA_CANDIDATE`, ownership verdict, BLOCK/DENY/403 path, or ENFORCE authority. OpenAI is absent from the locator authority path. State is bounded by global/per-operation/fingerprint/override caps, 30-day learned TTL, versioned restart validation, existing non-blocking observation-plane ingestion, immediate mutation persistence, RBAC and audit.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83 PASS** from API-1/2 through API-7.1; API-7.1 has **9 targeted test functions present**; OpenAI source contract **16/16 PASS** plus isolated tests PASS; package-source PASS; root Go shape **107 files PASS**; package-builder **9/9 PASS**; changed-file gofmt, Admin JavaScript and primary shell syntax PASS. Canonical Go 1.25 targeted/race remains `BLOCKED_ENVIRONMENT/NOT_RUN` because the host has Go 1.23.2 and external toolchain retrieval is unavailable. Next slice: **API-7.2 Identity/Object Relationship**. API-7.3/API-7.4 and API-8 remain `PLANNED`.

### API-7.2 Identity/Object Relationship — 2026-09-24

Added/changed implementation truth includes `object_relationship_api72.go`, `object_relationship_api72_test.go`, `tools/tests/test-api72-source.py`, `API72_IDENTITY_OBJECT_RELATIONSHIP.md`, and `API72_SOURCE_GATE_RESULT.md`, plus integration changes in `main.go`, `admin.go`, `api_security_persist.go`, `object_locator_api71.go`, `static/admin.html`, build/CI and canonical handover/testing/roadmap documentation. API-7.2 is `IMPLEMENTED_TESTING_DEFERRED`; API-7.3 is next.
- `API72_TEST_SUMMARY.txt` — concise API-7.2 executed/deferred qualification summary used by source-artifact packaging.


## API-7.3 additions — 2026-09-24

- `bola_detection_api73.go` — bounded DETECT-only BOLA candidate engine and Reviewer read APIs.
- `bola_detection_api73_test.go` — nine targeted API-7.3 tests.
- `tools/tests/test-api73-source.py` — dependency-free API-7.3 authority/privacy/bounds source gate.
- `API73_BOLA_DETECTION.md` — implementation/security boundary.
- `API73_SOURCE_GATE_RESULT.md` — source gate evidence.
- `API73_TEST_SUMMARY.txt` — scoped verification summary.
- API security persistence, server wiring, admin routes, build and CI retain all prior slices and include API-7.3.


## API-7.4 additions — 2026-09-24

- `bola_policy_api74.go` — bounded BOLA evidence policy store, workflow/reopen model, persistence, Reviewer APIs and non-blocking policy-status declaration.
- `bola_policy_api74_test.go` — ten targeted API-7.4 policy/workflow/persistence/privacy/concurrency tests.
- `tools/tests/test-api74-source.py` — dependency-free API-7.4 authority/privacy/bounds/console/build-CI source gate (156 checks).
- `API74_BOLA_POLICY_EVIDENCE_CONSOLE.md` — implementation, policy semantics, evidence lifecycle and authority boundary.
- `API74_SOURCE_GATE_RESULT.md` — exact-source gate result.
- `API74_TEST_SUMMARY.txt` — scoped executed/deferred qualification summary used by source-artifact packaging.
- Integration changes include `main.go`, `admin.go`, `api_security_persist.go`, `static/admin.html`, `build.sh`, CI, the additive API-7.3 historical gate, and canonical roadmap/handover/testing documentation.
- API-7.4 remains `IMPLEMENTED_TESTING_DEFERRED`; API-8 GraphQL Security is next.

## API-8 GraphQL Security checkpoint — 2026-09-24

API-8 is implemented across parsing/normalization, SDL contract binding, deterministic complexity/field/mutation/subscription/introspection policy, Apollo persisted-query support, verified GraphQL-variable integration into API-7 evidence, and Reviewer-gated `LEARN` / `DETECT` / `ENFORCE` policy plus Admin Console. Any policy edit or bound contract content change returns the policy to `LEARN`; a referenced schema contract cannot be deleted. Only explicit deterministic GraphQL policy in `ENFORCE` may reject traffic. API-6/API-7 learned or inferred evidence and OpenAI cannot promote, mutate or bypass GraphQL enforcement.

Durable GraphQL state excludes raw query documents, literals, raw variable values, Authorization/Cookie/JWT data and unverified claims. SDL is reduced to a digest plus normalized topology; GraphQL object values use API-7 keyed fingerprints and API-5 verified identity only. State and parser resources are bounded, versioned, restart-revalidated, autosaved/final-flushed, and strict mutation APIs are Reviewer-gated and audited.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83/100/110/156/259 PASS** through API-8; **22 targeted API-8 Go test functions are present**; OpenAI source contract **16/16 PASS** plus isolated tests PASS; package-source PASS; package-builder **9/9 PASS**; root Go source shape **115 files PASS**; changed API-8 Go files `gofmt`, Admin embedded JavaScript, relevant shell and Python syntax PASS. Canonical Go 1.25 `go mod tidy -diff`, root build/vet/test, API-8 targeted tests and race tests remain `BLOCKED_ENVIRONMENT/NOT_RUN` because the host has Go 1.23.2 and external toolchain retrieval is unavailable. API-8 remains `IMPLEMENTED_TESTING_DEFERRED`, not `TESTED` or `RELEASED`.

The documented API-security implementation roadmap is now complete through API-8. No API-9 is defined. Next gate: canonical Go 1.25 qualification of the exact source/artifact, followed by release promotion only if all required gates pass.


## API-8 Post-Audit Hardening additions — 2026-09-24

- `API8_POST_AUDIT_HARDENING.md`
- `API8_POST_AUDIT_HARDENING_SOURCE_GATE_RESULT.md`
- `API8_POST_AUDIT_HARDENING_TEST_SUMMARY.txt`
- `cidr_policy_test.go` — CIDR Enabled/expiry regression coverage.
- `tools/tests/test-api8-post-audit-hardening-source.py` — runtime/UI/debt-cleanup source gate.
- `static/admin.html` — single shipping Console source with System/Diagnostics, HSM/Vector, traffic controls, lifecycle and opaque-ID selector closure.

Removed as non-runtime/misleading debt: `web/`, `investigation.go`, `security_timeline.go`, `change_audit.go`, `security_export.go`, `debug_lifecycle_v2.go`, and `debug_retention_worker_v2.go`.

## 2026-09-25 — Production Correctness & Control-Plane Hardening

Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.

## 2026-09-28 — Duplication consolidation inventory delta

Removed confirmed superseded/unwired source: legacy debug-evidence compatibility
files, generic `security_state.go`/`security_event.go`, standalone
`deployment_readiness.go`/`reliability.go`, root VectorScan placeholder
qualification/audit files, `schema_api2_continuation.go`, the isolated
`pki_hardening.go` helper pair, and legacy `internal/coverage/report.go`.

Added `debug_sanitize.go`, `internal/capability/seclang.go` plus its test,
`tools/tests/test-code-duplication-review-source.py`,
`CODE_DUPLICATION_REVIEW.md`, and
`CODE_DUPLICATION_REVIEW_SOURCE_GATE_RESULT.md`. Root Go source-shape is now
**95 files** after intentional cleanup.

<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
