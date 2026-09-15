# waf-proxy — package manifest

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
