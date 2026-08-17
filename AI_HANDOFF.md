# AI Development Handoff

## Project
WAF Proxy

## Objective
Continue development and improvement of the WAF proxy without making major
architectural changes until the imported baseline has been verified on a host
with the required toolchains and Linux runtime dependencies.

## Baseline Date
2026-08-17 (Asia/Hong_Kong)

## Repository State
- Branch: `main`, tracking `origin/main`.
- HEAD: `d4dba8d` (`Initial import of WAF proxy project`).
- The worktree was clean before this handoff update.
- The repository contains one imported commit; no later implementation history
  is available to distinguish recently completed work from older code.
- No credentials, private keys, generated binary, dependency directories, or
  local config were found tracked.

## Architecture and Main Components
- **Core service and configuration (`main.go`)**: a single Go 1.22 module. It
  defines the node -> member -> pool -> site model, configuration migration and
  validation, per-site Coraza WAF construction, reverse proxies, request logging,
  runtime construction, and atomic runtime swaps on Apply.
- **Data-plane listeners (`listeners.go`, `freebind_*.go`, `ipmanage.go`)**:
  live reconciliation of HTTP/TLS listen sockets; host/SNI routing; Linux
  `IP_FREEBIND`; optional interface address management using `CAP_NET_ADMIN`.
- **Load balancing (`pool.go`)**: weighted round robin, least connections,
  client-IP hash, and random selection; TCP/HTTP/none health monitors with
  rise/fall state and fail-open behavior when all members are down.
- **WAF policy layer (`main.go`, `profiles.go`, `fieldpolicy_test.go`)**:
  Coraza/SecLang rules, named base policies, URL-scoped page policies, virtual
  patches, allow-listed per-field validation, and content-derived profiles.
- **Learning and discovery (`sitemap.go`, `sitemap_persist.go`, `discovery.go`,
  `learn.go`, `profiles.go`)**: passive site maps, an opt-in crawler, discovered
  form shape, traffic/rule aggregation, and reviewed tuning suggestions.
- **AI analysis (`ai.go`)**: asynchronous OpenAI-compatible/Anthropic request
  analysis, redaction, structured verdict validation, site-scoped expiring
  blocklists, and fail-open worker queues. AI calls are not in the proxy hot path.
- **Management plane (`admin.go`, `users.go`)**: localhost-default HTTP API and
  embedded `static/admin.html`, bearer/session authentication, RBAC, users,
  audit records, draft versus live configuration, logs, and operational APIs.
- **Operations/integrations**: HA config synchronization (`ha.go`), syslog
  (`syslog.go`), notifications/webhooks (`notify.go`), metrics (`metrics.go`),
  watchdog support (`watchdog.go`), and signed self-update/rollback
  (`update.go`, `internal/sigupdate`).
- **Deployment**: Debian/Ubuntu systemd installation scripts, a hardened unit,
  interface setup, diagnostics, upgrade/uninstall scripts, and external OWASP
  CRS rules expected under `/etc/waf/crs`.
- **Frontend migration (`web/`)**: an optional Vite/Preact migration target.
  It is not the shipping console. Only the shell, notification bell, and HA tab
  are ported; Config, Pools, Policies, AI Setup, Site Map, and file browser are
  explicit stubs/TODOs. Go currently embeds and serves `static/admin.html`.

## Build, Test, and Development Commands
- Full release build: `./build.sh`
  - runs `go mod tidy`, `go vet ./...`, `go test ./...`, then a static
    `CGO_ENABLED=0 go build` with version/commit ldflags.
- Direct backend checks: `go test ./...`, `go vet ./...`, and `go build ./...`.
- Frontend migration development: `cd web && npm install && npm run dev`.
- Frontend migration build: `cd web && npm install && npm run build`; output is
  intended for `static/app`, but it is not yet embedded or served.
- Deployment diagnostics: `sudo /opt/waf-proxy/waf-doctor.sh` after installation
  (or `sudo ./waf-doctor.sh` from the source checkout where appropriate).
- There is no Makefile and no configured CI workflow in this import.

## Verification Performed
- `bash -n build.sh install.sh setup-interfaces.sh uninstall.sh upgrade.sh waf-doctor.sh`
  - PASS: all shipped shell scripts parse successfully.
- `jq empty config.sample.json`
  - PASS: sample configuration is valid JSON.
- Confirmed both the sample's global rules path and default policy rules path
  resolve to `/etc/waf/coraza.conf`.
- Inspected test inventory: 20 Go tests covering AI redaction/config/verdict and
  blocklist behavior, field-policy directive/validation/discovery behavior, and
  signed-update verification/path safety/apply/rollback.
- Go tests, vet, build, and frontend build were **not run** because this host has
  no `go`, `node`, or `npm` installed.

## Known Incomplete or Broken Areas
- **Build/test baseline remains unproven** until Go >= 1.22 is available and
  `go test ./...`, `go vet ./...`, and a build complete successfully.
- **Dependency reproducibility is incomplete**: `go.mod` pins Coraza, but no
  `go.sum` is tracked. `build.sh` runs `go mod tidy`, so the first build needs
  network access and will modify the checkout.
- **The Vite/Preact migration is intentionally incomplete** and must not replace
  `static/admin.html` yet. Most application tabs are stubs.
- **The frontend manifest/config are inconsistent**: `web/vite.config.js`
  statically imports `@preact/preset-vite`, but that package is absent from
  `web/package.json`. The documented try/catch cannot recover from an unresolved
  static import, so a clean `npm install && npm run build` is expected to fail.
- **No frontend lockfile is tracked**, so the optional migration build is not
  reproducible.
- **Automated coverage is narrow for the service's scope**: there are no direct
  tests for config validation/migration, listener reconciliation and host/SNI
  routing, load-balancer algorithms and monitor state, admin auth/RBAC and draft
  Apply behavior, HA synchronization, or end-to-end WAF proxying.
- Runtime validation requiring Linux/systemd, capabilities, real interfaces,
  TLS files, backends, and OWASP CRS has not been performed. macOS cannot fully
  validate `IP_FREEBIND`, managed IPs, the systemd sandbox, or watchdog behavior.
- The source comment near the top of `main.go` still says listener-set/TLS
  changes require restart, while the implementation and later documentation say
  listener changes reconcile live. This is a documentation inconsistency, not a
  demonstrated runtime defect.

## Security and Operational Notes
- Admin defaults to `127.0.0.1:9090`; off-loopback binds require TLS and startup
  checks prevent obvious management/data-plane overlap.
- The systemd service runs as user `waf`, with privileged-port and optional
  managed-IP capabilities. Review whether `CAP_NET_ADMIN` is needed per deployment.
- AI is asynchronous/fail-open and redacts credential headers, but its configured
  API key and the HA peer token are stored in the `0600` config file; protect it
  as a secret.
- Signed updates are disabled unless a publisher public key is configured, and
  update actions are admin-only and localhost-only.
- The packaged `coraza.conf` requires an external OWASP CRS installation before
  a production config using it can compile.

## Important Decisions
- Preserve the current single-process architecture and atomic runtime model for
  now; no architectural refactor was made during baseline review.
- Treat `static/admin.html` as the production console until all Vite/Preact tabs
  are ported, built, embedded/served, and functionally tested.
- Treat draft persistence and live Apply as separate operator workflows; changes
  to this behavior require focused tests because Apply rebuilds WAFs and swaps
  the running runtime.

## Recommended Next Steps
1. On a Go 1.22+ development host, run `go mod tidy`, review and commit the
   resulting `go.sum`, then run `go test ./...`, `go vet ./...`, and
   `CGO_ENABLED=0 go build ./...`.
2. Add focused tests around configuration validation/migration and pool selection;
   these are high-value, low-architecture-risk seams in the request path.
3. Decide whether the frontend migration is active. If yes, add
   `@preact/preset-vite`, create/commit a lockfile, prove a clean build, and keep
   it non-shipping until feature parity. If no, clearly mark `web/` experimental.
4. Perform a Linux integration smoke test with temporary ports/backend and a
   local CRS checkout: startup, health endpoint, Host routing, WAF detection,
   pool failover, draft save, live Apply, and graceful drain.
5. After those checks, reconcile documentation inconsistencies and choose the
   next focused feature or defect; avoid broad refactoring before test coverage
   protects runtime Apply and listener behavior.

## Changes Made During This Review
- Updated only `AI_HANDOFF.md` with this technical baseline.
- No source, configuration, dependency, or architecture changes were made.
