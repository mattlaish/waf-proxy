# AI Development Handoff

## Project
WAF Proxy

## Objective
Continue development and improvement of the WAF proxy without making major
architectural changes until the imported baseline has been verified on a host
with the required toolchains and Linux runtime dependencies.

## Baseline Date
2026-08-19 (Asia/Hong_Kong)

## Repository State
- Branch: `main`, tracking `origin/main`.
- HEAD at this handoff: `8d9c003` (`Implement hybrid form discovery`).
- The current intended uncommitted source changes add per-field custom allow
  patterns, Windows update portability, config/pool/runtime coverage, explicit
  experimental frontend labeling, documentation, and this handoff update.
- Two unrelated untracked local files were found during push preparation:
  `Codex Image Aug 18, 2026, 02_59_30 PM.png` and `git-save-push.ps1`. They are
  not required by the WAF build and should remain unstaged unless the owner
  explicitly wants them in the repository.
- The repository contains the imported implementation, documentation baseline,
  and committed dependency reproducibility baseline.
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
- Reproducibility baseline on a Windows development host using the checksum-
  verified official Go 1.26.0 toolchain:
  - `go mod tidy` completed and generated `go.sum`; it normalized the `go`
    directive to `1.22.0` and recorded Coraza's indirect dependencies in
    `go.mod`.
  - `go mod verify` — PASS (`all modules verified`).
  - `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go vet ./...` — PASS.
  - Linux root and `internal/sigupdate` test binaries compiled successfully via
    `go test -c`; they cannot be executed on the Windows host.
  - A Linux/amd64 static release-style binary built successfully with
    `-trimpath` and stripped ldflags (13,955,234 bytes).
  - Native Windows `go test ./...` — FAIL at root package compile because
    `update.go` references `sigupdate.ReExec`, which has no Windows
    implementation. `internal/sigupdate` tests themselves passed. This is a
    portability defect, not a Linux deployment regression.
- Linux VM verification on 2026-08-17 using Go 1.26.0:
  - `VERSION=2026.08.17-ai2 COMMIT=local-ai-safety ./build.sh` — PASS through
    `go mod tidy`, `go vet`, `go test`, and static binary build.
  - `go test` passed for both `waf-proxy` and `waf-proxy/internal/sigupdate`.
  - The service started successfully under systemd after allowing `AF_NETLINK`
    in `RestrictAddressFamilies`, which Go requires for interface discovery.
  - Managed-IP reconciliation was verified on data interface `ens37`: existing
    `192.168.1.71/21` plus managed `192.168.1.80/21`; the service was active and
    listening on both addresses at port 443.
  - `/etc/waf/config.json` was verified as valid JSON and retained ownership
    `waf:waf` with mode `0600`.
- Frontend build was not run; the migration remains non-shipping and has known
  dependency/lockfile gaps.
- Hybrid Form Discovery validation on 2026-08-18:
  - New tests cover URL-encoded/JSON/multipart name-only parsing, value
    non-retention, body restoration, passive+crawled merging, action indexing,
    safe observed-GET crawl seeds, and same-origin form-action normalization.
  - `go test -count=1 ./...` — PASS in an isolated Windows test copy using a
    test-only Windows `ReExec` shim for the separately documented platform gap.
  - `go vet ./...` — PASS in the same isolated test copy.
  - `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go vet ./...` — PASS against the
    actual source tree.
  - Linux root and `internal/sigupdate` test binaries compiled successfully.
  - Linux/amd64 static release-style build — PASS (13,979,810 bytes; SHA-256
    `9C7AB8597E7A9B910548745EB3F65A33A938D37670874F78F28097FCB7802317`).
  - Shipping `static/admin.html` inline JavaScript parsed successfully with
    Node (`new Function` syntax check).

## Known Incomplete or Broken Areas
- Windows now has a `ReExec` implementation that starts the replacement binary
  with inherited arguments/environment/standard streams and exits the parent.
  Native Windows compile, tests, and vet pass; production deployment remains
  Linux/systemd, where the existing atomic `syscall.Exec` behavior is unchanged.
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
- Linux/systemd validation now covers service startup, the systemd netlink
  sandbox requirement, managed IP assignment, and two TLS listen sockets on a
  real data interface. It does **not** yet constitute a full integration suite:
  Host/SNI routing, WAF detection/blocking, pool failover, draft/live Apply,
  graceful drain, HA, watchdog, and signed update still need repeatable tests.
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
- Hybrid passive discovery captures at most 64 KiB for POST/PUT/PATCH supported
  content types, restores the request body before Coraza/backend handling, and
  persists only allow-listed field names after a backend response. Field values
  and uploaded content are never placed in discovery state.
- Crawler and passive fields merge by method/action/name with HTML metadata
  authoritative for type and required flags. Discovery provenance is exposed as
  `discovery_source` (`passive`, `crawled`, or `both`) so it cannot collide with
  Page Policy's existing `source` (`ARGS_POST`/`ARGS`).
- Active discovery seeds previously observed GET paths and follows same-origin
  form actions with GET only; it still never submits forms or authenticates.

## Recommended Next Steps
1. Turn the successful local checks into a deployed Linux integration smoke test
   with temporary ports/backend and a local CRS checkout: startup, health
   endpoint, Host routing, WAF detection,
   pool failover, draft save, live Apply, and graceful drain.
2. Add focused tests for admin authentication/RBAC, draft persistence/live Apply,
   monitor rise/fall behavior, and listener reconciliation across real sockets.
3. Keep `web/` experimental and non-shipping until every tab is ported and a
   lockfile plus feature-parity tests make its build reproducible.
4. After the deployed checks, choose the
   next focused feature or defect; avoid broad refactoring before test coverage
   protects runtime Apply and listener behavior.

## Changes Made During This Baseline
- Generated and reviewed `go.sum` using `go mod tidy`; `go.mod` now includes the
  tidy-normalized Go version and indirect dependency graph.
- Updated this handoff with reproducibility, VM/systemd, managed-IP, and Windows
  portability results.
- No application source, runtime configuration, frontend, or architecture
  changes were made in the committed baseline.

## Hybrid Form Discovery
- Added privacy-safe passive field-name discovery for URL-encoded, top-level
  JSON, and multipart write requests, with bounded capture and body restoration.
- Passive observations are committed only after the request reaches a backend
  and receives a response; WAF-blocked requests do not seed field suggestions.
- Merged passive and crawler fields without retaining values, and indexed HTML
  form metadata by resolved action path so `/login` policies can use forms
  rendered on another page.
- Extended crawler seeds with previously observed GET paths and same-origin form
  actions; added completed pages/forms/fields counters.
- Updated the shipping console to show completion summaries and discovery
  provenance, and to seed POST/PUT/PATCH actions without confusing provenance
  with Page Policy request source.
- Added `hybrid_discovery_test.go` and updated the README.

## Recommended Next Step for Hybrid Discovery
1. Build/deploy on the Linux VM and submit a real login once. Confirm `/login`
   exposes passive `username`/`password` fields, preserves backend request body,
   and upgrades provenance to `both` after a successful Discover crawl.
2. Keep policy application manual; do not auto-apply field suggestions during
   the first runtime smoke test.

## Local Coverage Expansion on 2026-08-19
- Added a Windows `sigupdate.ReExec` implementation; native Windows root and
  `internal/sigupdate` tests plus `go vet ./...` now pass without a test shim.
- Added focused tests for legacy config migration, draft/full validation,
  managed-IP shape, weighted round robin, least connections, IP hash, health
  filtering, and all-members-down fail-open behavior.
- Added an in-process runtime smoke test for exact/wildcard/catch-all Host
  routing and `/healthz` transition from 200 to draining 503.
- Marked `web/` explicitly experimental and non-shipping. The production UI
  remains `static/admin.html`; no incomplete Vite output is served.
- Verified a clean source copy with `go mod verify`, native `go test -count=1
  ./...`, native and Linux-target `go vet ./...`, Linux test compilation, and a
  stripped Linux/amd64 build (13,979,810 bytes; SHA-256
  `3DCB0E4B78912BDDFD96933098B8D21873F48B18F657B0B0C7D3405E919FA257`).
- Corrected the stale `main.go` comment: listener/TLS changes reconcile live.
- Deployed systemd, real-socket Apply/drain, WAF blocking, and failover smoke
  checks still require the Linux VM; local tests do not claim those results.

## Per-Field Custom Policies on 2026-08-19
- `FieldPolicy` now supports an optional `allow_pattern` per field. It is
  combined with that field's profile, required flag, length limits, and
  field-only CRS exclusions; sibling fields on the same form are unaffected.
- Patterns are limited to 512 bytes, compiled with Go's regexp validator, and
  rejected if the policy expression contains literal double quotes or line
  breaks before SecLang generation. This protects policy syntax; quotes in
  submitted field values remain governed by the selected pattern (for example,
  `[[:graph:]]` permits visible quote characters).
- The shipping `static/admin.html` editor exposes one custom allow-regex input
  per field. Existing configurations remain compatible because an empty pattern
  preserves the previous profile-only behavior.
- Focused tests prove different directives for `user_id` and `password`, reject
  malformed/injection-shaped patterns, and compile the generated directives
  with Coraza. Native tests/vet, Linux vet/test compilation/build, and shipping
  UI JavaScript syntax all pass. Linux build SHA-256:
  `246151E7BF958EE097A3726B728D90EF271CA9E317243512C1E6AF73434688BE`.
