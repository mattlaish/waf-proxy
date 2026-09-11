# waf-proxy — package manifest

A Coraza-based reverse-proxy WAF with an embedded admin console, delivered as
source you build into `waf-proxy` plus the optional `waf-tlsfront` companion. **Start with `INSTALL.md`.**

## Status

Current source baseline: Coraza v3.7.0 / Go 1.25.0 with optional VectorScan Learning Accelerator. Portable Coraza API-stub full regression/vet/race and native libhs ABI compile/vet/race gates pass. Real Go 1.25 + Coraza v3.7.0 execution and real libvectorscan matching are **NOT_RUN** in the isolated packaging environment and remain release-host gates. Coraza is always authoritative; VectorScan false negatives/scan errors fail the affected group to Coraza-only `FAILSAFE`.

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
- `build.sh`            — go mod tidy → vet → test → static binary; deterministic local-Go minimum check
- `qualify-release-host.sh` — Phase 0 preflight/core real-host qualification; BLOCKED vs FAIL truth boundary
- `build-release-artifact.sh` — clean complete-source ZIP builder with generated release manifest and mandatory verification
- `verify-release-artifact.sh` — post-packaging extracted-artifact integrity gate
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
- `internal/vectoraccel/` — conservative rule classifier, grouped VectorScan scanner, Learning/VALIDATED/ACCELERATED/FAILSAFE state machine, persistence and native/stub adapters
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
tar xzf waf-proxy-install.tar.gz
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


## 2026-09-11 Supportability Slice

Added wafctl debug export/doctor/support bundle foundation and Phase 1 differential qualification helper. Real Coraza/libvectorscan execution remains NOT_RUN.


## Stage 2 Supportability Implementation
- Added wafctl supportability command foundation.
- Added qualification phase1 differential runner foundation.
- Real Go 1.25/Coraza/libvectorscan qualification remains NOT_RUN.

## 2026-09-11 qualification evidence

- `PHASE0_PHASE1_QUALIFICATION_REPORT_2026-09-11.md` — exact release-host Phase 0 preflight/core BLOCKED evidence and Phase 1 NOT_RUN boundary.


## 2026-09-11 Phase 2 source additions

- `internal/vectoraccel/transforms.go` — ordered exact-semantics transform allow-list and local implementations.
- `internal/vectoraccel/transforms_test.go` — portable transform parity/allow-list tests.
- `internal/vectoraccel/phase2_realcoraza_test.go` — release-host Coraza v3.7.0 transform/source parity gate (`realcoraza` build tag).
- `internal/vectoraccel/rules.go` — Phase 2 classifier/source/semantic-key expansion.
- `internal/vectoraccel/engine.go` — applies ordered transforms and reconstructs the new/special request sources.
- `internal/vectoraccel/engine_test.go` — Phase 2 classifier/source/fingerprint tests.

Truth boundary: Phase 2 source is implemented but real Go 1.25 + Coraza v3.7.0 + libvectorscan + production CRS qualification remains NOT_RUN.


### Phase 2 delivery artifact policy

The Phase 2 complete-source ZIP is built with `build-release-artifact.sh`, contains a regenerated `RELEASE_MANIFEST.txt`, and is independently verified with `verify-release-artifact.sh`. The accompanying patch is relative to `waf-proxy-phase0-phase1-qualification-attempt-2026-09-11.zip` and must reconstruct the source workspace byte-for-byte and mode-for-mode.

## 2026-09-11 Phase 2 coverage increment 2

Modified production/test source:

- `internal/vectoraccel/rules.go` — semantic adapter v4; classification for `REQUEST_URI_RAW`, `REQUEST_LINE`, `REQUEST_BASENAME`, `REMOTE_ADDR`, and `REMOTE_PORT`.
- `internal/vectoraccel/engine.go` — exact reconstruction of the new request/connection sources from the same request fields/connector semantics used by Coraza.
- `internal/vectoraccel/transforms.go` — exact `base64Encode` and `hexEncode` transformations.
- `internal/vectoraccel/engine_test.go` / `transforms_test.go` — expanded positive and negative classifier/source/transform coverage.
- `internal/vectoraccel/phase2_realcoraza_test.go` — real-Coraza parity cases for the newly eligible transforms/sources; remains release-host `NOT_RUN` here.

Still intentionally excluded: ARGS/body, chains, negation, aggregate selectors, decode transforms, hash transforms, URL/path/HTML/JS/CSS normalization/decoding, dynamic macros and every other unqualified semantic.

## 2026-09-11 Phase 3 release/supply-chain files

- `generate-release-evidence.sh` — portable/native release evidence, SPDX 2.3 + CycloneDX 1.5 SBOM, version/provenance and govulncheck truth status.
- `sign-release-artifact.sh` — optional detached SHA-256 signing using an externally supplied private key; no key generation/storage.
- `verify-release-signature.sh` — detached signature verification using the organizational public key.
- `phase3-release-tooling-test.sh` — deterministic evidence/SBOM/signing-gate regression.
- `build-release-artifact.sh` — now emits release flavor/evidence and supports `SOURCE_DATE_EPOCH` reproducibility.
- `verify-release-artifact.sh` — now verifies generated Phase 3 evidence, checksums and SBOM format markers after extraction.
- `PHASE3_RELEASE_ENGINEERING_REPORT_2026-09-11.md` — implementation and executed-evidence report.

## 2026-09-11 Phase 4 security/product files

- `security.go` — request correlation, CIDR allow/deny, rate/in-flight and direct transport/TLS abuse controls, custom block response and bounded limiter state.
- `security_state.go` — atomic persistence/restoration for AI/learner/notification/session-hash/audit/security state.
- `security_phase4_test.go` — security policy/request-ID/rate/in-flight/connection/TLS/bounded-map regressions.
- `security_state_phase4_test.go` — persistence/no-raw-token round-trip regression.
- `request_id_coraza_phase4_test.go` — qualified-host exact WAF request-ID → Coraza transaction-ID regression.
- `pki_url.go` — SSRF-hardened HTTPS CRL retrieval, refresh/LKG/cache/status logic.
- `pki_url_phase4_test.go` — CRL URL/SSRF/LKG/dedupe/cache regressions.
- `PHASE4_SECURITY_PRODUCT_REPORT_2026-09-11.md` — implementation and executed-evidence boundary.
- `waf-proxy.service` / `install.sh` / `waf-doctor.sh` — `/var/lib/waf-proxy` deployment-state wiring.

## 2026-09-11 release-blocker hotfix

- `go.mod` / `go.sum` — synchronized Coraza v3.7.0 indirect module graph/checksums.
- top-level `*.sh` and `benchmark/build.sh` — Git executable mode is part of the release contract and must be `100755`.
- No production Go behavior changed in this hotfix.
