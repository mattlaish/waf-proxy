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

Build & deploy
- `build.sh`            — go mod tidy → vet → test → static binary
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
