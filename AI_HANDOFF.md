# AI Development Handoff

## Project
WAF Proxy

## Patch Ledger
- `patch.md` is the required current-patch handover and manual version-control
  ledger. Every future source/schema/UI/test/build change must update both this
  handoff and `patch.md`.
## AI Git Workflow

AI may perform normal Git operations:

- `git pull`
- `git add`
- `git commit`
- `git push`

Before pushing:

- Run relevant tests and validation checks.
- Update `AI_HANDOFF.md` with completed work, verification results, and next recommended steps.
- Review `git diff` and `git status`.
- Ensure no secrets, credentials, generated binaries, temporary files, or unrelated changes are included.

Never:

- Force push.
- Rewrite history unless explicitly approved.
- Delete branches or change remotes without approval.

## Multi-Agent Coordination

- GitHub is the source of truth.
- Before starting work, synchronize with the latest repository state.
- Do not assume another AI agent's uncommitted local changes exist.
- Do not modify the same repository concurrently with another AI agent unless the work is isolated by branch.
## Objective
Continue development and improvement of the WAF proxy without making major
architectural changes until the imported baseline has been verified on a host
with the required toolchains and Linux runtime dependencies.

## Baseline Date
2026-08-20 (Asia/Hong_Kong)

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

## PKI Slice 1: Backend CA Trust on 2026-08-19
- Added pool-scoped `BackendTLSConfig` with `use_system_ca`, custom PEM
  `ca_files`, `server_name`, and reserved CRL fields. An omitted system-CA flag
  defaults to true for backward-compatible HTTPS pools; HTTP pools are unchanged.
- HTTPS proxy traffic and HTTPS health monitors share the pool's validated
  `tls.Config`. TLS 1.2 is the minimum, standard Go chain/hostname verification
  remains enabled, and there is no skip-verify option.
- Custom CA bundles are appended to the OS trust store by default. Operators may
  explicitly disable the OS store, but full Apply validation then requires at
  least one custom CA. Files are limited to 8 MiB and strictly parsed as PEM
  certificate bundles; empty, malformed, unknown-block, trailing-data,
  duplicate-path, and unreadable inputs fail before runtime swap.
- Draft validation checks schema/enums/ranges without requiring files to exist.
  Full validation and runtime construction load the trust material. Failed Apply
  therefore retains the old runtime under the existing atomic Apply model.
- `revocation_mode`, refresh, and CRL source fields are reserved in the schema
  but full Apply rejects them until Slice 2 implements real revocation checking;
  the service never silently claims hard-fail CRL behavior.
- The shipping pool editor exposes OS CA selection, custom CA file paths, and a
  server-name override. `README.md` and `config.sample.json` document the model.
- Tests dynamically generate CA/leaf material and cover custom-root proxying,
  untrusted roots, hostname mismatch, TLS 1.1 rejection, malformed/oversized CA
  files, HTTP-pool misuse, isolated empty stores, system-CA defaults, and old
  HTTP behavior. No private key fixture or production secret is stored.
- Native `go test -count=1 ./...`, native/Linux `go vet ./...`, Linux test
  compilation, shipping UI JavaScript parsing, and sample JSON validation pass.
  `CGO_ENABLED=0` Linux/amd64 static build: 14,000,290 bytes, SHA-256
  `A5FF6CC1607C94796C76D84AD9F6BDE21F67B2A5ED548D7C9DC6AAA82B932862`.
- No PKCS#11 dependency was added, so the static-build assumption is unchanged.
  Next PKI slice: static CRL parsing, issuer/signature/time validation, immutable
  snapshot, and soft/hard handshake enforcement before adding URL refresh or HSM.

## PKI Slice 2: Static CRL Enforcement on 2026-08-20
- Static pool CRLs now support strict PEM `X509 CRL` bundles and DER files, with
  an 8 MiB per-file limit, at most 32 files/lists, duplicate/blank-path checks,
  and rejection of malformed/trailing/unknown PEM data.
- Apply rejects CRLs whose `thisUpdate` is too far in the future or whose
  `nextUpdate` is absent/expired (five-minute clock skew). For custom CA files,
  matching CRL signatures are also checked at Apply. System-store issuer
  certificates are not enumerable through Go's `CertPool`, so their signature
  check occurs during the verified TLS handshake instead.
- CRLs are loaded into an immutable snapshot behind `atomic.Pointer`; TLS
  handshakes perform no file or network I/O. Standard Go chain and hostname
  verification remains enabled and runs before `VerifyConnection`.
- Every non-root certificate in the verified chain is checked. A valid matching
  CRL that lists a leaf or intermediate serial always fails closed. Soft mode
  allows missing issuer coverage; hard mode requires coverage for every leaf and
  intermediate and requires at least one configured static CRL.
- Wrong issuer signatures, expired/future lists, and revoked serials fail the
  connection in both modes. Failed Apply retains the old runtime and snapshot.
- URL CRLs and `refresh_sec` remain explicitly rejected until Slice 3; there is
  no downloader, background updater, status API, or manual refresh API yet.
- The shipping pool editor, README, and sample config now expose static CRL
  files plus soft/hard mode. Tests dynamically generate all keys/certificates/
  CRLs and cover valid/unrevoked, revoked leaf, revoked intermediate, missing
  coverage, wrong signature, expired, future, malformed, and PEM/DER parsing.
- Native `go test -count=1 ./...`, native/Linux `go vet ./...`, Linux test
  compilation, UI JavaScript parsing, sample JSON validation, and the
  `CGO_ENABLED=0` Linux build pass. Static binary: 14,020,770 bytes, SHA-256
  `E0AC425D40250AB0AED46BDBFE23BA4C180B366FEF3C322C3026B7D7AC988A03`.
- Next PKI slice: SSRF-safe URL downloader, refresh deduplication/scheduling,
  atomic last-known-good swap, `/api/pki/status`, manual refresh, RBAC, and audit.
## Feature Roadmap Review on 2026-08-21

Session type: Claude Code Web/cloud, branch `claude/test-e5j6pr`. This was a
read-and-review session. **No application source, configuration, or frontend
files were changed.** The only modified file is this handoff.

### Verification performed this session
- Container toolchain: Go 1.24.7 (linux/amd64), Node v22.22.2, module proxy
  reachable. Note `go.mod` still declares `go 1.22.0`; 1.24.7 builds it cleanly.
- `go vet ./...` — PASS.
- `go test ./...` — PASS (`ok waf-proxy`, `ok waf-proxy/internal/sigupdate`).
- No systemd, no `CAP_NET_ADMIN`, and no privileged ports in this container, so
  the deployed-VM items in the earlier next-steps list remain unverified here.

### Findings confirmed by code inspection
- **Client IP is the TCP peer everywhere except the AI path.** `clientIP()`
  (`main.go:1375`) returns `RemoteAddr` only. It feeds `ip_hash` member
  selection (`main.go:1336`), the access ring (`main.go:1005`), syslog
  forwarding (`main.go:1010`), and the backend error log (`main.go:1355`); the
  Coraza engine likewise sees `RemoteAddr` through `txhttp.WrapHandler`
  (`main.go:865`). Only `ai.go` resolves a real client address, via its own
  `TrustedProxyCIDRs` (`ai.go:73`, chain walk at `ai.go:510`).
  Consequence behind any CDN, upstream load balancer, or SNAT device: `ip_hash`
  collapses onto one member, the learner's distinct-client signal — the basis of
  its false-positive-versus-attack discrimination — degrades to a single client,
  and every log/SIEM record names the upstream instead of the origin. Nothing
  errors; the results are silently wrong. Treated below as a prerequisite, not
  as an isolated defect.
- **`web/vite.config.js` cannot build**, as previously documented. Line 2
  statically imports `@preact/preset-vite`, which is absent from
  `web/package.json`; the `try`/`catch` around `preact()` cannot recover an
  unresolved static import. No lockfile is tracked. Still non-shipping.
- **`MANIFEST.md` status is stale.** It states the code "has never been compiled
  or run by its author", which the 2026-08-17 VM results and this session's
  passing vet/test contradict.
- No rate limiting, connection capping, or throttling exists anywhere in the
  tree. No manual IP allow/deny list exists — only the AI may add a block. No
  block page, request-correlation ID, or GeoIP/ASN support exists.

### Owner decisions on 2026-08-21
The owner reviewed five candidate features and approved four. Recorded here so a
later session does not relitigate them.

1. **Approved — trusted-proxy client IP + L7 abuse control.** Scope named
   honestly: this is **not** DDoS mitigation. Volumetric L3/L4 attacks cannot be
   answered inside this process and belong to the ISP, a scrubbing provider, or
   an upstream firewall; the product should not claim otherwise, consistent with
   its existing refusal to fake VIP failover. What is in scope is application-
   layer abuse where each request is cheap to send and expensive to serve:
   credential stuffing and brute force, scraping, and API abuse.
2. **Approved — manual IP allow/deny lists**, CIDR-aware, allow wins over deny,
   global or per-site, optional TTL, reusing the AI blocklist enforcement path.
   GeoIP/ASN filtering is a later increment, not part of the first delivery.
3. **Approved — custom block page plus request correlation ID**, stamped onto
   the match record, access log, and syslog event so a user-reported block can
   be traced to the rule that caused it.
4. **Approved — persistence for security state.** The AI blocklist, learner
   aggregates, notification queue, sessions, and audit ring are in-memory and
   are discarded on every restart and upgrade. `sitemap_persist.go` is the
   existing pattern to follow.
5. **Declined — response-side DLP.** Outbound PHI/PII pattern matching was
   considered and rejected by the owner for now. Do not implement it without a
   new decision.

### Recommended next step
Implement item 1 as a single change, in this order:

1. Promote trusted-proxy handling to global configuration and resolve one
   authoritative client address per request. Apply it to the Coraza connection
   address, `ip_hash`, the access ring, syslog, and the learner. Fold `ai.go`'s
   per-connector `TrustedProxyCIDRs` into the global setting, preserving its
   existing behaviour and tests. Default with no trusted proxies configured must
   remain today's behaviour: use `RemoteAddr`.
2. Add the limiter in the handler chain ahead of the WAF, keyed on the resolved
   address: token bucket with per-site defaults and per-page overrides on
   `PagePolicy`, action configurable as 429 or a time-boxed block through the
   existing blocklist. Add per-IP concurrent connection caps and a TLS handshake
   rate cap in the same pass. Slowloris already has coverage through
   `read_timeout_sec`/`idle_timeout_sec`.
3. Surface it in the shipping console (`static/admin.html`): limits in the
   page-policy editor, live counters, and notification-bell events on trip.
4. Have discovery suggest limits for authentication endpoints, reusing the
   established learn → suggest → review → apply flow. Nothing auto-applies.

**Constraint carried into implementation:** the limiter is the first component
to sit in the request hot path. Every other subsystem here — AI, syslog,
learner, notifications — is asynchronous and fail-open by deliberate design.
Preserve that discipline: sharded counters, no global lock, and fail-open on any
internal limiter error. Ship the hot-path tests with the feature.

### Deferred, still open
- The integration coverage described in the earlier next-steps list is
  unchanged and still outstanding: admin authentication/RBAC, draft-versus-Apply
  semantics, listener reconciliation over real sockets, WAF blocking, pool
  failover, and drain. Much of it runs in a container without systemd; only
  systemd behaviour, managed-IP assignment, privileged ports, the two-NIC setup,
  and the real-login hybrid-discovery check require the Linux VM.
- `web/` remains experimental and non-shipping; the config defect and missing
  lockfile were not fixed in this session.
- `MANIFEST.md`'s stale status line was not corrected in this session.

## Static and Dynamic Scan on 2026-08-21

Run against the pre-PKI tree on branch `claude/test-e5j6pr` in the Claude Code
cloud container (Go 1.24.7). **No application source was changed.** All scan
scaffolding was created outside the repository; `git status` was clean
afterwards and `go mod verify` reported all modules verified.

### Tools installed and run
`staticcheck`, `gosec`, and `govulncheck` were installed via `go install`;
`golangci-lint` was already present. `semgrep`, `shellcheck`, and `nuclei` were
not available and were not used.

### Static results
- `go vet ./...` — clean.
- `staticcheck ./...` — 2 findings, both unused functions (see below).
- `golangci-lint run ./...` — 20 findings: 17 `errcheck` on `Close()`/`Remove()`
  in cleanup paths, 1 De Morgan style suggestion (`main.go:1309`), 2 unused.
- `gosec -severity=low -confidence=low ./...` — 27 findings over 21 files and
  8,532 lines: 8 HIGH, 11 MEDIUM, 7 LOW (1 additional low-confidence entry).
- `gofmt -l .` — 10 files, all consistent with the project's compact style.
- `bash -n` — all six shipped scripts parse.
- `node --check` on the 83,446 bytes of inline JavaScript extracted from
  `static/admin.html` — parses.
- Secret scan for private keys and provider token patterns — nothing found.
- `go mod verify` — all modules verified.

**Every gosec HIGH was triaged against the source and none is exploitable.** The
triage table is now recorded in README.md under "Security scanning &
verification" so future sessions do not re-litigate it: G115 ×3 are conversions
that cannot overflow, G404 ×2 are non-security uses of `math/rand`, G402 is the
documented `tls_skip_verify` syslog option, G702/G703 flag the sigupdate
path-safety logic its own tests already cover, and G204 is a shell-less
`exec.Command` whose arguments are validated by `net.ParseIP` and
`net.Interfaces()`.

### Static findings worth acting on
- `ipManager.releaseAll` (`ipmanage.go:282`) is unreachable, and its doc comment
  claims it is "called on shutdown". The drain test confirms shutdown never
  releases managed addresses. This matches README's statement that managed IPs
  survive routine restarts, so the behaviour is presumed intentional and **the
  comment is stale**. It is nevertheless dead code in a `CAP_NET_ADMIN` path;
  either wire it up or correct the comment, and say which was intended.
- `signalStore.get` (`profiles.go:410`) is unused dead code.

### Dynamic results
A race-instrumented binary was run live against a throwaway backend on a
separate loopback address, using a minimal SecLang ruleset (no CRS available in
the container). All of the following passed:

- **No data races and no panics** under roughly 1,440 concurrent requests across
  8 workers while five live config Applies swapped the runtime mid-traffic.
  `go test -race -count=1 ./...` also clean. This is the first direct evidence
  that the atomic runtime swap is safe under concurrent load.
- The **management/data-plane separation check** refused startup when the admin
  IP collided with a data-plane listener — verified working, not merely present.
- **Admin authentication**: 401 unauthenticated and with a wrong token across
  `/api/config`, `/api/users`, `/api/audit`, `/api/ai/blocklist`; 200 with the
  correct token.
- **Signed update fails safe**: `/api/update/status` returned 404 with no
  publisher key compiled in.
- **Host routing**: 421 for an undeclared Host and for a missing Host header.
- **WAF enforcement**: SQLi and XSS blocked (403) in query string and POST body;
  benign traffic passed.
- **Draft versus Apply**: draft save returned 200, the live runtime kept serving
  the previous hostname, and the draft-only hostname correctly did not serve.
- **Graceful drain**: `SIGTERM` held `/healthz` at 503 for exactly three seconds
  before listeners closed, logging "stopped cleanly".
- **Request smuggling (CL.TE)**: no desync. The pipelined request appeared in the
  WAF's own access ring, proving it was inspected rather than tunnelled.
- **Protocol abuse**: 8 KB header, 16 KB URL, null bytes, CRLF injection, and an
  unknown method were all handled without error.

### Dynamic evidence for the client-IP finding
A request carrying `X-Forwarded-For: 9.9.9.9` and `X-Real-IP: 8.8.8.8` reached
the backend with both headers rewritten to `127.0.0.1`, the TCP peer. Inbound
XFF is therefore replaced, not appended. Spoofing is correctly neutralised, and
the same mechanism means that behind a CDN or upstream load balancer the backend
also loses the true client address, not only `ip_hash`, the access log, syslog,
and the learner. This is stronger evidence than the earlier code reading and
raises the priority of the trusted-proxy work recorded in the roadmap section
above.

### Open item: dependency vulnerability scan did not run
`govulncheck ./...` **failed**: the container's network policy answers 403 to
`CONNECT vuln.go.dev:443`, so the vulnerability database could not be fetched.
No CVE cross-check has been performed against the dependency tree (Coraza
v3.3.2, libinjection-go v0.2.2, gjson v1.18.0, `golang.org/x/net` v0.34.0 and
the rest). `go mod verify` passing proves checksum integrity only. **Run
`govulncheck ./...` on a host with access to `vuln.go.dev` and treat the release
as unscanned until it passes.**

### Scan coverage limits
Not exercised in this environment: TLS and SNI paths (no certificates), HA,
AI analysis (disabled), managed-IP assignment, systemd behaviour, and privileged
ports. One probe initially looked like a WAF miss — encoded `%2e%2e%2f`
returning 200 — but the cause was the hand-written test rule lacking
`t:urlDecodeUni`, not an engine defect. Recorded so it is not logged as a bug.

## PKI Scan and Rebase on 2026-08-24

The two sections above were written against the pre-PKI tree and are retained
as the record of that session. They are still accurate on this commit: `clientIP`
(`main.go:1388`) continues to return the TCP peer, trusted-proxy resolution
remains confined to `ai.go`, and no rate limiting exists. The approved roadmap
therefore stands unchanged.

One claim in them is now out of date and is corrected here: automated coverage
is no longer narrow across the board. The suite has grown from 20 tests to **43**,
and mTLS/CRL is now the best-covered subsystem in the project.

### Verification of the PKI slices on this commit
- `go build ./...`, `go vet ./...` — clean.
- `go test ./...` — PASS (`waf-proxy`, `internal/sigupdate`).
- `go test -race -count=1 ./...` — PASS, no data races.
- The six static-CRL tests pass, covering revoked and valid serials, soft versus
  hard behaviour on missing coverage, wrong-issuer signatures, expired, future
  and malformed CRLs, leaf plus intermediate checking, and DER parsing.

### Static analysis of the PKI code
- `gosec` reports one finding in `pki.go`: G304 at `readPKIFile` (`pki.go:123`),
  opening an operator-supplied path. Reading operator-specified CA and CRL files
  is the function's purpose; same class as the pre-existing G304 findings.
- `staticcheck` reports SA1019 at `pki.go:256`: `RevocationList.RevokedCertificates`
  is deprecated since Go 1.21. The code deliberately checks both that field and
  `RevokedCertificateEntries`, which is the safe combination; the warning is
  expected, but it will need removing when the field is eventually deleted.
- `ipManager.releaseAll` and `signalStore.get` remain unused on this commit.

### Fuzzing of the untrusted-input parsers
Temporary fuzz targets were written for the parsers that consume attacker- or
operator-supplied bytes, run, and then removed; they are not part of the tree.

- `parseCRLFile` — 47,882 executions, 25 corpus entries, no crash.
- `appendCABundle` — 19,951 executions, 32 corpus entries, no crash.
- `validateBackendServerName` — 419,801 executions, 105 corpus entries, no crash.

No panics, hangs, or crash corpus were produced. Worth making permanent: these
targets are cheap and cover the highest-risk parsing surface in the codebase.

### Review observations on pki.go, none security-critical
- `crlStore.verify` returns after processing the first verified chain, so only
  `VerifiedChains[0]` is checked. Go orders that chain first, so this is
  defensible, but the `return nil` inside the loop makes it read as if all chains
  are checked when only one is.
- `appendCABundle` calls `strings.TrimSpace(string(rest))` each iteration, which
  copies the remaining bundle every time; `parseCRLFile` uses `bytes.TrimSpace`
  on the same pattern and does not. Load-time only and bounded by the 8 MiB cap,
  so this is a consistency cleanup rather than a defect.
- `validateBackendServerName` compiles its regular expression on every call
  instead of once at package level.
- Bounds are sound throughout: 8 MiB per file, 32 CA files, 32 CRL lists, five
  minutes of clock skew enforced in both directions, and TLS 1.2 as the floor.
  No `InsecureSkipVerify` anywhere in this code.

### Branch state
`claude/test-e5j6pr` has twice been rebuilt on the current `main` rather than
rebased commit-by-commit, because this handoff was restructured underneath it
each time. The three review and scan sections are re-applied to the current
documents on each rebuild rather than merged mechanically; treat the newest
copy of this file on `main` as authoritative for structure, and these sections
as the record of what was reviewed, decided, and verified.

The matching `## Security scanning & verification` section in README.md is the
operator-facing half of the 2026-08-21 and 2026-08-24 scans and carries the
gosec triage table those scans refer to. Keep the two together; the scan record
above cites that table by name.

Note for later sessions: `AGENTS.md` and the AI Git Workflow section above now
grant normal Git operations, which the earlier sections predate. The rules that
remain are no force-push, no history rewrite without approval, and no branch or
remote changes without approval.

**Still open:** `govulncheck` has never run against this dependency tree; the
container blocks `vuln.go.dev`. The PKI slices add no new modules, so the gap is
unchanged in scope but now also covers untested certificate-handling paths.
