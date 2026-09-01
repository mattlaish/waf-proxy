# waf-proxy — package manifest

A Coraza-based reverse-proxy WAF with an embedded admin console, delivered as
source you build into a single static binary. **Start with `INSTALL.md`.**

## Status

Verified **statically only** — this code has never been compiled or run by its
author (no Go toolchain was available at build time). `./build.sh` runs
`go vet` + `go test` and is the real gate. Treat the first deployment as a
bring-up in **DetectionOnly**, not a production cutover. See INSTALL.md §Status.

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
- `config.sample.json` — sample production config
- `coraza.conf`        — engine config (includes OWASP CRS from /etc/waf/crs)
- `go.mod`             — module + pinned Coraza

Go source (module `waf-proxy`, stdlib + Coraza only)
- `main.go`       — config schema, runtime build, listeners, health, drain, shutdown
- `admin.go`      — admin API, auth middleware, match/access rings, console serving
- `ai.go`         — async AI traffic analysis (fail-open, redaction, per-site modes)
- `pool.go`       — load balancing + health monitors
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
- `static/admin.html`   — single-file admin console (go:embed-ed)

Optional
- `web/`          — Vite+preact console SCAFFOLD (not the shipping UI; source-modularization only)

## Quick start

```bash
tar xzf waf-proxy-install.tar.gz
cd waf-proxy
./build.sh          # needs Go >= 1.22 and network once (Coraza + tests)
sudo ./install.sh   # prints the admin token — save it
sudo ./setup-interfaces.sh   # two-NIC appliance: pin mgmt/data planes (optional)
# then fetch OWASP CRS and start — see INSTALL.md §4–§5
```
