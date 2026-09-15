# AI Development Handoff

## Project
WAF Proxy

## Patch Ledger
- `patch.md` is the required current-patch handover and manual version-control
  ledger. Every future source/schema/UI/test/build change must update both this
  handoff and `patch.md`.
- Stage-completion feedback after documentation/handoff updates must end with
  current Taiwan time: `YYYY-MM-DD HH:mm:ss UTC+8 (Taiwan)`.
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
2026-09-03 (Asia/Taipei; request hot-path optimization slice)

## Repository State
- Branch: `main`, tracking `origin/main`.
- Current local baseline before this slice: `95ab964` (`lf modifed`); the
  uploaded repository also recorded `origin/main` at that commit.
- The worktree was clean before this hot-path slice. Source, UI, configuration,
  tests, README, and handoff documentation are now intentionally modified but
  remain uncommitted and undeployed.
- A `git pull --ff-only` synchronization attempt was made before implementation,
  but the execution environment could not resolve `github.com`. Do not infer
  from that failure that the public remote still points to `95ab964`; synchronize
  on a networked host before merging or pushing.
- No credentials, private keys, generated binaries, dependency directories, or
  local production config were added by this slice.

## Documentation Synchronization on 2026-08-28
- Updated the stale repository base from `8d9c003` to the post-PR-merge main
  baseline `3186015`.
- Reconciled Git policy with `AGENTS.md`: normal pull/add/commit/push operations
  are allowed, while force-push, history rewrite, branch deletion, and remote
  changes remain approval-gated. A user instruction for a particular task may
  still narrow those permissions.
- `patch.md` remains the authoritative concise patch/version ledger; this file
  retains the detailed review, scan evidence, decisions, and implementation
  history.
- No functional implementation or verification claim changed in this sync.

## Architecture and Main Components
- **Core service and configuration (`main.go`)**: a Go 1.25 module pinned to Coraza v3.7.0. It
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
  - requires Go >=1.25, runs `go mod tidy -diff`, vet/tests/race and the real-Coraza gate before producing binaries; native VectorScan builds use CGO/libhs and are not self-contained static binaries.
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

## Candidate: Field-Scoped Learner Suggestions on 2026-08-24

**Status: candidate, not approved.** This is a fifth item proposed after the four
approved on 2026-08-21 and is recorded here for a decision before any build. Do
not start it on the strength of this section alone.

### The requirement
Page policies scope by path, so the finest automatic unit today is "exclude rule
942100 on `/login`". The operator need is finer: a password field may legitimately
contain a quote while a comment field on the same path must stay fully inspected.
The correct unit is therefore **path plus field name**, not path alone. True
per-form scoping is not achievable at the HTTP layer — two forms POSTing to one
URL are indistinguishable unless their field names differ — so field name is both
the practical and the correct key.

### Already built; do not rebuild
A prior analysis concluded this capability was merely "underexposed". That
understates the current state. Verified on this commit:

- The engine emits per-field removal in three places: `PagePolicy.ExcludeTargets`
  at `main.go:1114`, per-field `FieldPolicy` exclusions at `main.go:1119`, and
  policy-level exclusions at `main.go:1264` (global `SecRuleUpdateTargetById`
  or path-gated `ctl:ruleRemoveTargetById`).
- `PageExcludeTarget{RuleID, Target}` exists at `main.go:104`, wired into
  `PagePolicy.ExcludeTargets` at `main.go:134`.
- `FieldPolicy` (`main.go:109`) already carries `Name`, `Source`, `Profile`,
  `AllowPattern`, `Required`, `MinLength`, `MaxLength`, and `ExcludeRuleIDs`,
  with a per-field editor in `static/admin.html` and coverage in
  `fieldpolicy_test.go` proving `user_id` and `password` compile to different
  directives on the same form.

**An operator can already achieve the stated goal through the console today.**
The gap is not enforcement and not the UI for hand-authored policy.

### The actual gap
The learner cannot *suggest* a field-scoped exclusion, because it never records
which field tripped a rule:

- `matchRec` (`admin.go:31`) carries Time, Site, RuleID, Severity, Phase, Client,
  URI, Msg, and Data. There is no matched-variable field.
- The WAF error callback (`main.go:1038`) reads `rule.Rule().ID()`, `Severity()`,
  `Phase()`, `ClientIPAddress()`, `URI()`, `Message()`, and `Data()`. `Data()` is
  the matched *value*, not the variable name; the variable is discarded.
- `ruleAgg` (`learn.go:31`) aggregates count, clients, and severity only, and
  `noteMatch(site, uri, ruleID, client, severity)` (`learn.go:101`) has no field
  parameter.
- `pageRecommendation.SuggestExcl` (`learn.go:142`) is `[]int` — rule IDs alone.
- `handleLearnApply` (`admin.go:658`) accepts `RuleIDs []int` and writes
  whole-rule, path-scoped exclusions.

### The API needed is available
Coraza exposes `types.MatchedRule.MatchedDatas() []MatchData`
(`types/rule_match.go:43`), and `MatchData` exposes `Variable()`, `Key()`, and
`Value()` (`types/rule_match.go:10`). `Key()` is the argument name. The callback
already holds the `MatchedRule`, so the data is one method call away.

### Implementation sketch
1. Capture the matched variable and key in the WAF callback; add a field to
   `matchRec`. Record the name only, never the value — the same rule the hybrid
   discovery work follows.
2. Thread it through `noteMatch` and key `ruleAgg` by rule plus field.
3. Change `SuggestExcl` to carry rule/target pairs rather than bare rule IDs, and
   propose `ruleRemoveTargetById=RULE;ARGS:field` when a rule fires repeatedly on
   one field from many distinct clients.
4. Extend `handleLearnApply` to write `ExcludeTargets` instead of whole-rule
   exclusions, and surface the rule/field pair in the Site Map and the bell.

### Security caveat, and why this must stay review-gated
Excluding a rule on a field is a real security decision. `ARGS:password
!942100` is correct for a value that is hashed and never concatenated into SQL,
and wrong for any field that reaches a query. Suggestions must remain
human-reviewed with no auto-apply, consistent with every other learner
suggestion in this product.

### Dependency worth noting
The learner's false-positive-versus-attack discrimination rests on counting
distinct clients, which is degraded by the client-IP defect recorded in the
2026-08-21 roadmap section. Field-scoped suggestions inherit that weakness, so
the trusted-proxy work should land first or the suggestions will be keyed on
unreliable client counts.

### Decision needed
Approve as a fifth roadmap item, defer behind the four approved on 2026-08-21,
or decline. Until then this section is a record of investigation only.

## Trusted-Proxy Client-IP Resolution on 2026-08-28

**Status: implemented locally; not committed or deployed.** This completes the
client-IP prerequisite in approved roadmap item 1. L7 abuse controls remain a
separate follow-up and were not added in this slice.

### Implementation

- Added root config `trusted_proxy_cidrs`. An empty list preserves the former
  edge behavior: the immediate TCP peer is authoritative and inbound XFF is
  ignored.
- Added one compiled request-path resolver in `clientip.go`. It accepts XFF only
  when the immediate peer is trusted, walks the chain right-to-left, and stops
  at the first untrusted hop. Invalid, over-4-KiB, or over-32-hop chains fail
  safely to the immediate peer. Configuration accepts at most 64 unique IPv4 or
  IPv6 CIDRs.
- The resolver runs outside logging, AI, Coraza, and proxying, so Coraza match
  records, `ip_hash`, access logs, syslog, learner client counts, AI block keys,
  and backend forwarding headers share one address.
- Backend `X-Forwarded-For` is rebuilt from the resolved address and
  `X-Real-IP` is replaced; client-supplied forwarding identity is never appended.
- Existing `ai.trusted_proxy_cidrs` is imported once into the global field on
  config load or admin PUT, then removed from the AI object on the next save.
- The shipping UI retains the control in Setup / AI but labels it explicitly as
  a global upstream-proxy trust setting. `config.sample.json` includes the new
  root field.

### Verification

- `go test -count=1 ./...`: pass on Go 1.26.0 Windows/amd64.
- `go vet ./...`: pass.
- Focused tests cover untrusted spoofing, one and multiple trusted proxies,
  right-to-left trust boundaries, IPv6, malformed/oversized fallback, legacy
  migration, draft validation, and canonical backend XFF/X-Real-IP.
- Shipping inline JavaScript parses with Node `--check`; sample JSON parses with
  `python -m json.tool`.
- `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath`: pass; binary size
  20,030,558 bytes, SHA-256
  `7932329870C7331436E6B463ED708E9C8F4CE5154AF0AF2804F5C88197894039`.
- Race tests were not run on this Windows host because no C compiler is
  installed (`go test -race` requires CGO). The ordinary suite and static Linux
  build passed; rerun `go test -race ./...` on the Linux VM before release.

### Recommended next step

Deploy this build behind the real test proxy/CDN path and verify that one request
produces the same public client address in the Coraza event, access log, syslog,
learner input, backend XFF/X-Real-IP, and AI advisory record. After that, design
the L7 limiter against this resolver; do not key rate limits directly from raw
forwarding headers.
## Request Hot-Path Optimization on 2026-09-03

**Status: implemented locally; not committed, deployed, or real-Coraza verified.**
The goal was to implement the reviewed hot-path wins while explicitly avoiding
the rejected aggregate "everything is under 1 microsecond" characterization.

### Implementation

- `accessRing` and `matchRing` are fixed circular buffers with preallocated
  storage, write index, and size. Snapshot ordering remains newest-first and the
  admin API shape is unchanged.
- `accessRec` retains a `time.Time` internally and custom-marshals the historical
  `HH:MM:SS` field on admin serialization, moving that format allocation off the
  traffic path. `logWrap` also resolves the client once and reuses one record for
  the access ring and syslog.
- AI config/client publication now uses immutable `atomic.Pointer` snapshots and
  an atomic enabled gate. `ai_mode: off` sites receive `next` directly; globally
  disabled AI on an advisory/block site returns after a single atomic load. The
  obsolete AI `statusRecorder` was removed.
- The remaining `statusRecorder` implements `Unwrap()` for compatibility with
  `http.ResponseController` and optional interfaces such as `http.Hijacker`.
- Syslog config is published through `atomic.Pointer[SyslogConfig]`; global and
  per-stream atomic gates let disabled WAF/access/audit/notify hooks return before
  copying a config snapshot. Connection lifecycle remains serialized by the
  existing mutex.
- Added root `passive_discovery_enabled`, default `true`. The admin PUT path also
  defaults it true before decoding so older full-config clients that omit the
  new field preserve historical behavior. The shipping UI and sample config
  expose the toggle.
- `requestBodyPrefixWrap` is now the sole bounded body-prefix reader for passive
  discovery and AI. Both consumers share one `[]byte` through one request-context
  capture object and the request body is restored only once before Coraza/backend
  handling. Passive discovery disabled at build time returns the underlying
  handler and adds no request-body parsing cost.
- AI jobs retain the shared byte prefix and `buildPrompt` writes at most the
  first 2,000 body bytes directly to the builder instead of creating a full body
  string copy.
- Passive JSON discovery uses a bounded top-level scanner that skips values
  lexically instead of decoding each value into `json.RawMessage`; only field
  names are retained.
- No new client-IP context cache was added. Trusted-proxy XFF processing already
  occurs once in `clientIPs.wrap`; caching the normalized address again would add
  a `WithContext`/context value on every request merely to avoid a small number
  of `SplitHostPort` calls.

### Verification

- Real `go test -count=1 ./...` is **NOT_RUN**: the environment cannot resolve
  `proxy.golang.org`, so `github.com/corazawaf/coraza/v3@v3.3.2` cannot be
  downloaded. No real-Coraza pass is claimed.
- A temporary external compile harness using a minimal Coraza API stub passed
  `go test -run '^$' ./...`, all focused hot-path/discovery tests, the full
  repository suite, `go vet ./...`, and `go test -race -count=1 ./...` against
  the final source snapshot. This checks Go compilation and non-Coraza
  regressions only; it is not a substitute for real WAF tests.
- Added focused tests for circular buffer ordering/count and JSON compatibility,
  status-recorder unwrap/Hijacker reachability, AI-off direct wrapping, disabled
  syslog gating, legacy passive-toggle defaulting, disabled passive discovery,
  shared AI/passive body capture, large JSON-value skipping, discovery commit
  flow, and byte-body prompt generation.
- Offline component benchmarks on AMD EPYC 9V74: fixed access-ring add
  `9.825 ns/op`, `0 B/op`, `0 allocs/op`; ~60 KiB JSON scanner `35,846 ns/op`,
  `1,302 B/op`, `14 allocs/op`. These are not end-to-end Coraza benchmarks.
- `config.sample.json` parses and the inline production admin JavaScript passes
  `node --check`. No release binary/hash is recorded because the real dependency
  is unavailable in this environment.

### Recommended next step

On a host with the real Coraza dependency/cache, first synchronize `origin/main`,
then rebase/merge this working slice as appropriate and run `go test -race ./...`,
`go vet ./...`, normal Linux build/release checks, WebSocket/HTTP Upgrade
integration, and representative 1 KiB/16 KiB/64 KiB request-body benchmarks for
passive discovery on/off and AI body capture on/off. Only after those gates pass
should this slice be committed/deployed.


## P0-A ReverseProxy Buffering + Shared Console Theme on 2026-09-04

**Status: implemented locally; real-Coraza validation remains NOT_RUN in this isolated environment.**

### Implementation

- Added `proxy_buffer.go`: one shared 32 KiB `httputil.BufferPool` backed by
  `sync.Pool` of fixed-array pointers. Unexpected buffer capacities are not
  retained. The pointer-backed pool avoids the slice-header allocation observed
  when storing `[]byte` directly in `sync.Pool`.
- `buildProxy` now sets `BufferPool: reverseProxyCopyBuffers` and
  `FlushInterval: 0`. This removes the default per-response 32 KiB copy-buffer
  allocation and avoids opting normal responses into the periodic flush
  timer/mutex path. Streaming behavior remains delegated to Go's
  `httputil.ReverseProxy` streaming detection.
- Added focused P0-A tests for pool shape and `buildProxy` settings, an end-to-end
  proxy copy test that confirms BufferPool Get/Put use on a 64 KiB response, an
  SSE test confirming event-stream flushing still works with `FlushInterval=0`,
  plus a component benchmark.
- Imported the user-supplied `theme.css` byte-for-byte as both
  `static/theme.css` and `web/src/theme.css`. The shipping console embeds and
  serves `/theme.css`; its CSP now allows same-origin styles, and existing
  console tokens alias the canonical `--cg-*` variables. The Preact migration
  tree imports the same theme before its compatibility/layout stylesheet.
- Added a console test that verifies `/theme.css`, the canonical green token,
  the root stylesheet link, and the updated CSP.

### Verification

- Real `go test` with Coraza v3.3.2: **NOT_RUN** because this environment still
  cannot resolve `proxy.golang.org`; no real-Coraza pass is claimed.
- Temporary external Coraza API-stub harness: focused P0-A/theme tests passed,
  full repository `go test -count=1 ./...` passed, `go vet ./...` passed, and
  `go test -race -count=1 ./...` passed. This validates Go integration and local
  regressions only, not Coraza semantics.
- `BenchmarkFixedBufferPoolGetPut` on AMD EPYC 9V74: approximately
  `10.90 ns/op`, `0 B/op`, `0 allocs/op`.
- User-supplied CSS and both repository copies have identical SHA-256
  `83aae56abd54b9837adffcb93138becd01d1bd2941664f1948a13f56a418f753`.
- Shipping inline admin JavaScript passes `node --check`;
  `config.sample.json` parses successfully.

### Release gate / next slice

Before production release, rerun the unmodified tree with real Coraza available,
including race/vet/build and an HTTP streaming/WebSocket integration pass. The
next performance implementation slice is **P0-B: zero-allocation backend member
selection and removal of the per-request member `context.WithValue` handoff**.

## P0-B Zero-Allocation Load-Balancer Hot Path on 2026-09-04

**Status: implemented locally on top of P0-A; real-Coraza validation remains NOT_RUN in this isolated environment.**

### Implementation

- Removed `healthyMembers()` and its temporary `[]*memberRuntime` allocation from
  every backend selection. `poolRuntime.pick` now performs direct zero-allocation
  scans for round-robin, least-connections, IP-hash, and random selection.
- Preserved existing health semantics: healthy members are preferred; if all
  members are marked down, selection fails open across the configured pool.
  Concurrent health transitions between the small counting/selection passes have
  explicit configured-member fallbacks instead of transient nil/503 results.
- Replaced `hash/fnv` object construction in the IP-hash path with an equivalent
  direct FNV-1a loop over the client-IP string; a regression test checks parity
  against `fnv.New32a`.
- Removed the per-request `context.WithValue`/`Request.WithContext` handoff used
  to pass the chosen member from `ReverseProxy.Rewrite` to `lbTransport`.
  `lbTransport` now selects the member itself and keeps that pointer local while
  incrementing/decrementing the member's active-request counter.
- `lbTransport` applies only member scheme/host; pool targets are deliberately
  scheme://host:port with no path/query, so the original request path/query remain
  unchanged. `preserve_host=true` remains honored; otherwise Host is cleared so
  `http.Transport` uses the selected backend host, matching `ProxyRequest.SetURL`.
- `Rewrite` remains responsible for authoritative client forwarding headers and
  no longer mutates the backend target or request context.
- Added `pool_hotpath_test.go` with zero-allocation assertions for all four LB
  methods, FNV compatibility, backend target/path/query behavior, active-count
  lifetime through response-body close, PreserveHost behavior, and benchmarks.

### Verification

- Real `go test` with Coraza v3.3.2: **NOT_RUN** because this environment cannot
  retrieve `github.com/corazawaf/coraza/v3@v3.3.2`; no real-Coraza pass is claimed.
- A temporary external minimal Coraza API-stub harness passed the full repository
  `go test -count=1 ./...`, `go vet ./...`, and `go test -race -count=1 ./...`
  against the P0-B source. This validates Go integration/local regressions only.
- Pre-P0-B 8-member selector benchmark on AMD EPYC 9V74:
  round-robin `47.44 ns/op`, least_conn `42.99 ns/op`, ip_hash `47.16 ns/op`,
  random `48.14 ns/op`; all `64 B/op`, `1 alloc/op`.
- P0-B selector benchmark on the same host/harness (representative run range):
  round-robin ~`19.5–20.6 ns/op`, least_conn ~`15.3–16.1 ns/op`, ip_hash
  ~`12.1–12.5 ns/op`, random ~`27.7–28.2 ns/op`; all `0 B/op`, `0 allocs/op`.
- The removed pre-P0-B member context handoff independently measured
  `32.01 ns/op`, `48 B/op`, `1 alloc/op`.

### Release gate / next slice

Before production release, rerun the unmodified source with the real Coraza
module available and execute race/vet/build plus live HTTP/1.1 and HTTP/2 backend
proxy tests, including preserve-host, backend failures, health transitions, and
least-connections under concurrency. The next planned performance slice is
**P0-C: move host observation, sitemap, learner, and signal telemetry off the
synchronous data plane behind a bounded drop-on-full observation queue**.

### Pre-existing release issue observed during P0-B validation

- `bash -n install.sh` fails at line 132 with `syntax error: unexpected end of file`.
  The identical failure is present in the untouched P0-A source archive, so it
  was not introduced by P0-B. P0-B intentionally does not mix in this unrelated
  installer repair. Fix and validate the installer before calling a production
  release gate fully green.

## P0-C Bounded Asynchronous Observation Plane on 2026-09-04

**Status: implemented locally on top of P0-B; real-Coraza validation remains NOT_RUN in this isolated environment.**

### Implementation

- Added `observations.go` with one bounded 8,192-event non-blocking observation
  queue. Host discovery, live sitemap updates, learner request accounting,
  learner rule-match accounting, and request-shape signals are processed by a
  background worker rather than while a request is waiting.
- Queue saturation never blocks data-plane traffic. Full-queue events are
  dropped, counted atomically, and reported by an aggregated background warning
  at most once per five-second reporting interval.
- Added observation stats (`queue_depth`, `queue_capacity`, `dropped`,
  `processed`, `accepting`) as an additive `observations` object in `/api/metrics`.
- Production `server.observeHost`, `observeRequest`, and `observeMatch` route to
  the async plane; minimal/unit-test servers without a plane preserve the old
  synchronous fallback semantics.
- `hostObserver.note`, `siteMaps.record`, `learnStore.noteRequest/noteMatch`, and
  `signalStore.noteRequestShape` no longer execute from the production request or
  Coraza match callback path. Crawler/admin workflows continue to update their
  stores directly because they are already off the data plane.
- Optimized repeated live passive-field merging with `mergeObservedFields`: the
  signal store mutates its owned field slice in place and avoids rebuilding the
  general crawl merge index/string keys on unchanged requests.
- Lifecycle ordering was hardened: persisted sitemap state now loads before
  listeners are exposed; graceful shutdown stops listeners, drains accepted
  observations, then triggers the final sitemap save.
- Added `observations_test.go` covering eventual host/request/match application,
  drain semantics, drop-on-full behavior, zero-allocation enqueue, and sync vs
  async component/parallel benchmarks.

### Verification

- Real `go test -count=1 ./...`: **NOT_RUN** because `proxy.golang.org` remains
  unreachable and Coraza v3.3.2 cannot be downloaded. No real-Coraza pass is
  claimed.
- Temporary external minimal Coraza API-stub copy: full repository
  `go test -count=1 ./...`, `go vet ./...`, and `go test -race -count=1 ./...`
  passed against the final P0-C source snapshot.
- AMD EPYC 9V74 component benchmark, original pre-P0-C synchronous request
  observation before the live-field merge: ~2.56–2.63 µs/op, 1,664 B/op,
  24 allocs/op.
- After `mergeObservedFields`, direct synchronous request observation is
  ~443–445 ns/op, 48 B/op, 3 allocs/op. The actual P0-C enqueue is ~35–36 ns/op,
  0 B/op, 0 allocs/op.
- Host observation changes from ~213–229 ns/op, 48 B/op, 2 allocs/op to
  ~36–38 ns/op, 0 B/op, 0 allocs/op on the request path.
- Parallel request observation: optimized direct stores ~0.52–0.66 µs/op with
  48 B/op / 3 allocs; queue enqueue ~0.13–0.15 µs/op with 0 B/op / 0 allocs.

### Semantics / risk boundary

- Observation data is explicitly eventual and lossy under overload. It is used
  for visibility, discovery, recommendations, and learning only; WAF verdicts,
  backend routing, authentication, syslog forwarding, and AI enforcement are not
  dependent on successful observation enqueue.
- A single consumer deliberately serializes telemetry work off the request path.
  If a future sustained workload outpaces it, drops are visible in `/api/metrics`
  and the queue can later be sharded without changing the request contract.
- This archive still has no `.git` metadata, so repository synchronization with
  GitHub/main cannot be performed inside the artifact.

### Release gate / next slice

Before production release, rerun the unmodified source with the real Coraza
module available, including race/vet/build and live concurrency/load tests that
verify observation drops do not affect request success. The next performance
slice is **P0-D: remove synchronous per-match structured logging from the Coraza
callback using bounded/rate-limited aggregation so attack floods cannot turn
stdout/log I/O into the data-plane bottleneck.**

### Pre-existing release issue (unchanged)

- Shell release validation is inherited-red rather than P0-C-red: `install.sh` and
  `uninstall.sh` have EOF parse errors; `setup-interfaces.sh`, `waf-doctor.sh`, and
  `upgrade.sh` have CRLF-related bash parse failures. The identical failures are
  present in the P0-B input archive. P0-C intentionally leaves them unchanged,
  but they must be normalized/repaired before a production release gate is green.


## P0-D Bounded Match-Log Aggregation on 2026-09-04

**Status: implemented locally on top of P0-C; real-Coraza validation remains NOT_RUN in this isolated environment.**

### Implementation

- Added `matchlog.go`: an independent bounded 8,192-event non-blocking logging queue so
  rule-match log floods cannot compete with P0-C observation telemetry.
- Coraza's matched-rule callback no longer calls synchronous `slog.Warn`. It preserves
  match-ring recording, asynchronous learner observation, syslog forwarding, AI match
  handling, and WAF verdict/block behavior, then performs only a non-blocking match-log
  enqueue when the plane is available.
- Background aggregation keys on `site + rule_id + client` for a five-second window and
  caps active groups at 1,024. Queue or group saturation drops logging telemetry rather
  than applying backpressure; queue/group drop counters are tracked separately.
- Each emitted `rule match summary` includes count, first/last-seen times, severity/phase,
  and one representative URI/message/data sample. Process-log samples are bounded to
  1 KiB URI, 1 KiB message, and 2 KiB matched data with explicit truncation flags. The
  match ring and syslog record path remain unchanged.
- Drop telemetry is itself rate-limited/aggregated to at most one background warning per
  flush interval.
- `/api/metrics` now includes additive `match_logging` stats: queue depth/capacity, total
  drops, queue drops, group drops, processed events, emitted summaries, active/max
  groups, and accepting state.
- Shutdown ordering drains the match-log plane only after data-plane listeners are down,
  ensuring accepted events are flushed without allowing new request-path enqueue.
- Added `matchlog_test.go` covering same-key aggregation, client separation, no synchronous
  logger call on enqueue, queue saturation/drop accounting, group-cap behavior, accepted
  event draining, log-field truncation, aggregated drop reporting, and benchmarks.

### Verification

- Real `go test -count=1 ./...`: **NOT_RUN** because this environment still cannot resolve
  `proxy.golang.org` and cannot retrieve Coraza v3.3.2. No real-Coraza pass is claimed.
- Temporary external minimal Coraza API-stub copy: full repository `go test -count=1 ./...`,
  `go vet ./...`, and `go test -race -count=1 ./...` all pass against the final P0-D tree.
- AMD EPYC 9V74 component benchmark, successful non-blocking enqueue with a consumer:
  ~39.7–42.7 ns/op, 0 B/op, 0 allocs/op.
- Old-style per-match JSON `slog.Warn` to `io.Discard`: ~1.445–1.514 µs/op, 232 B/op,
  8 allocs/op. This is roughly a 34–38x callback-front CPU reduction before considering
  real stdout/filesystem/journald I/O, which would make synchronous logging more costly.

### Semantics / risk boundary

- Only process-log visibility is lossy under overload. WAF enforcement, match ring, syslog,
  AI match handling, and other security decisions do not depend on the match-log queue.
- Aggregation intentionally changes process-log cardinality from one line per matched rule
  event to one summary per site/rule/client/window. Operators needing every event should use
  the existing match ring/syslog path rather than stdout.
- This archive still contains no `.git` metadata, so GitHub/main synchronization cannot be
  performed from the artifact.

### Release gate / next slice

Before production release, rerun the unmodified source with real Coraza available and run
race/vet/build plus sustained clean-traffic and hostile-match-flood tests. Verify that match
log queue/group drops are visible without affecting request success. The next performance
slice should move to **P1 response-body inspection policy and backend Transport tuning**,
unless the inherited release-script syntax issues are repaired first.

### Pre-existing release issue (unchanged)

The inherited shell-script validation failures from P0-B/P0-C remain out of scope: installer
EOF parse errors and CRLF-related helper-script bash failures must be repaired before a full
production release gate can be green.

## P1 Response Inspection + Backend Transport Tuning on 2026-09-04

**Status: implemented locally on top of P0-D; real-Coraza validation remains NOT_RUN in this isolated environment.**

### Implementation

- `PolicyConfig` adds `response_body_inspection` (`""`/`inherit`/`on`/`off`) and
  `response_body_limit`. `policyDirectives` emits allow-listed
  `SecResponseBodyAccess On|Off` and integer `SecResponseBodyLimit` overrides
  after the policy rules file. Empty/inherit emits no override, preserving old
  configs and the rules-file default.
- `PoolConfig` adds a `transport` object with max-idle-total, max-idle-per-host,
  max-conns-per-host, idle timeout, dial timeout, and TCP keepalive controls.
  Validation rejects negative/excessive values. Zero uses tuned defaults except
  `max_conns_per_host=0`, which remains intentionally unlimited.
- Effective defaults are 2048 total idle, 256 idle per host, unlimited max
  connections per host, 90s idle timeout, 5s dial timeout, and 30s keepalive.
- Backend `http.Transport` ownership moved to `poolRuntime`; sites sharing one
  pool now share one backend keepalive/HTTP2 transport rather than creating a
  separate connection pool per site.
- Runtime replacement/error cleanup/shutdown closes idle connections on retired
  pool transports. Health-monitor transports also close idle connections when
  their goroutine exits; active requests remain unaffected by
  `CloseIdleConnections`.
- `/api/pools` now exposes effective transport values.
- Shipping admin UI adds response-body controls in Policies and backend
  connection-pool controls in Pools & Nodes. `config.sample.json` documents both
  features. Existing theme/P0-A/B/C/D behavior remains intact.
- Added `p1_tuning_test.go` covering response-body directive generation and
  validation, transport defaults/overrides/validation, concrete
  `http.Transport` field application, pool-scoped transport reuse, effective
  pool-status output, default config, and production UI wiring.

### Verification

- Real Coraza `go test -count=1 ./...`: **NOT_RUN** because the isolated
  environment still cannot resolve `proxy.golang.org` to download Coraza
  v3.3.2. No real-Coraza pass is claimed.
- Temporary external minimal Coraza API-stub copy: full repository
  `go test -count=1 ./...`, `go vet ./...`, and
  `go test -race -count=1 ./...` passed against the P1 source tree.
- Production inline admin JavaScript passes `node --check`; sample JSON parses.

### Semantics / risk boundary

- Response-body inspection defaults to the previous rules-file behavior for
  legacy configs. Operators must explicitly choose `off` to trade response WAF
  coverage for lower CPU/memory/latency.
- Pool transport tuning affects connection reuse/capacity only; it does not
  change backend TLS trust, load-balancer choice, health eligibility, request
  routing, or enforcement semantics.
- The pool-scoped transport is safe to share because backend scheme/TLS trust is
  already pool-scoped; site-specific Host preservation remains in each site's
  `lbTransport` wrapper.
- This source archive has no `.git` metadata, so GitHub/main synchronization
  cannot be performed inside the artifact.

### Release gate / next slice

Before production release, rerun the unmodified source with real Coraza and
perform load tests that compare response inspection on/off for representative
HTML/JSON response sizes and verify backend connection reuse under bursty
HTTP/1.1 and HTTP/2 traffic. Also verify tuned pool sizes against backend
connection limits. After that, the next performance work should be selected from
measured bottlenecks (for example XDP prefiltering or regex acceleration) rather
than another generic micro-optimization.

### Pre-existing release issue (unchanged)

The inherited shell-script syntax/CRLF failures from P0-B/P0-C remain out of
scope for this performance slice and still block a fully green production
release gate.

## Benchmark Harness v1.0.0 on 2026-09-04

**Status: implemented on top of P1; real-Coraza build/test remains NOT_RUN in
this isolated environment because `proxy.golang.org` is unreachable.**

### Purpose

Production traffic is currently too low to expose a stable next data-plane
bottleneck. `cmd/wafbench` provides a repeatable artificial-load baseline so the
next architectural change can be selected from measured Coraza, full-proxy, and
L3/L4 costs rather than from traffic volume.

### Implementation

- Added `cmd/wafbench` with five subcommands:
  - `backend`: deterministic fixed-size local backend;
  - `http`: full WAF HTTP/HTTPS load generator with clean/malicious corpus,
    latency/RPS/application-Mbps, HTTP status classes, Linux target-process
    CPU/RSS, host busy/softirq, NIC Mbps/PPS, and generator CPU;
  - `coraza`: direct transaction benchmark against an operator-supplied
    SecLang/CRS file, DetectionOnly by default, audit I/O off by default,
    optional response inspection/limit overrides and CPU/heap profiles;
  - `l4`: legal TCP connect/close or TLS-handshake pressure with the same
    process/host/NIC measurements; no raw SYN generation or spoofing;
  - `compare`: matches HTTP and direct-Coraza scenarios and optionally L4 data.
- Standard corpus: clean GET; clean JSON 1/16/64/256 KiB; SQLi 64 KiB; XSS
  64 KiB; traversal query. Generated JSON corpus is size-stable and validated.
- Coraza-share heuristic is computed from clean GET/JSON scenarios only;
  malicious cases are shown but intentionally excluded from the baseline median.
- Comparison heuristic: >=35% direct-Coraza share favors a regex-acceleration
  feasibility experiment; >=5% softirq plus >=100k aggregate sampled PPS favors
  XDP when the Coraza signal is below threshold; if both qualify, the tool
  explicitly reports both rather than hiding the tradeoff.
- JSON result files include system identity fields (CPU model, kernel, GOOS,
  GOARCH, Go version, CPU count) so baselines remain comparable across hosts.
- Added `benchmark/README.md` and `benchmark/build.sh`.

### Verification

- `cmd/wafbench` unit tests cover fixed-size/valid JSON corpus, histogram
  quantiles, and comparison math.
- Temporary external Coraza API stub: command package `go test`, `go vet`,
  `go test -race`, and build pass.
- Temporary external stub: full repository `go test -count=1 ./...`, `go vet ./...`, and `go test -race -count=1 ./...` pass.
- End-to-end stub smoke passed for `backend -> http`, direct `coraza`, `l4`,
  JSON result generation, `compare`, and CPU/heap pprof file generation.
- Real Coraza v3.3.2 dependency build remains NOT_RUN; do not treat stub
  throughput values as Coraza performance evidence.

### Usage / interpretation boundary

- `--pid`/`--iface` are local `/proc` measurements. For CPU-share comparison, run the tool on the WAF host and pin generator/WAF to disjoint CPU sets. A remote generator can still measure RPS/latency but cannot sample the WAF PID with this version.
- `app_mbps` is application payload throughput, not Ethernet line rate.
- Plain `l4` connect/close against a TLS listener may induce handshake-EOF logs;
  use `--tls` for a TLS listener or a dedicated plaintext benchmark listener.
- Direct Coraza and HTTP comparison must use the same CRS/policy assumptions and
  representative response size; otherwise the Coraza-share ratio is not valid.
- `compare` is a sizing heuristic, not an automatic architecture decision.

### Next step

Run a real-Coraza baseline on the target hardware with the deterministic backend,
collect 1/4/8-core direct Coraza results plus full WAF HTTP and L4 results, then
use `wafbench compare` and CPU profiles to decide whether the next POC is XDP or
VectorScan/Hyperscan-compatible regex acceleration.

## P2 Non-XDP/VectorScan Hardening on 2026-09-04

**Status: implemented on top of the benchmark-harness/P1 tree; real Coraza v3.3.2 validation remains NOT_RUN because the isolated environment cannot resolve `proxy.golang.org`.**

### Implemented

- Repaired the inherited release-script blocker. `build.sh`, `install.sh`,
  `setup-interfaces.sh`, `uninstall.sh`, `upgrade.sh`, `waf-doctor.sh`, and
  `benchmark/build.sh` are LF-normalized, executable, and pass `bash -n`.
  Added `release_scripts_test.go` to enforce no carriage returns, executable
  mode, and Bash syntax in future changes.
- AI request capture is now lazy. The pending map stores only the live
  `*http.Request` under a zero-allocation `pendingKey{client, uri}` key while
  Coraza runs. Full header/query/body/context capture happens only on an actual
  Coraza match or on a request that was pre-selected by the clean sample rate.
  Match analysis remains available even at `sample_rate=0`.
- Clean-request sampling is decided before expensive AI job construction.
  Unsampled/no-match requests no longer allocate header maps, parsed/redacted
  query state, timestamps, or `analysisJob` objects.
- AI queue saturation remains fail-open but is no longer a synchronous log
  amplifier: cumulative drop/enqueue counters were added, saturation warnings
  are limited to one per five seconds, and `/api/metrics` exposes `ai_queue`
  depth/capacity/logical-limit/enqueued/dropped.
- AI workers reuse one ticker each instead of allocating `time.After` timers in
  the worker polling loop.
- AI block-mode reads now use an immutable `atomic.Pointer` blocklist snapshot.
  Add/unblock/janitor writes are copy-on-write under the existing update mutex;
  request-path misses/hits never take that mutex, and expired entries are ignored
  immediately then pruned off-path.
- `statusRecorder.WriteHeader` now suppresses duplicate underlying
  `WriteHeader` calls; implicit `Write` records status 200. Existing `Unwrap`
  behavior for `http.ResponseController`/Hijacker remains.

### Verification

- Every release/build shell script listed above passes `bash -n` after LF
  normalization.
- External minimal Coraza API-stub harness: `go test -count=1 ./...`,
  `go vet ./...`, and `go test -race -count=1 ./...` pass.
- AMD EPYC 9V74 component benchmark for AI-enabled unsampled/no-match traffic:
  lazy path ~169-175 ns/op, 0 B/op, 0 allocs/op; test-only model of the
  previous eager-capture path ~1.49-1.55 us/op, 848 B/op, 17 allocs/op.
- Active-block lookup benchmark on the same host: atomic snapshot ~22 ns/op at
  8-way and ~21 ns/op at 32-way parallelism versus the former mutex model at
  ~93 ns/op and ~115 ns/op respectively; all zero-allocation.
- Real `go test -count=1 ./...` with Coraza v3.3.2 is NOT_RUN: DNS access to
  `proxy.golang.org` is still refused. No real-Coraza pass is claimed.

### Deliberately rejected P2 change

Access-ring sharding was prototyped and benchmarked rather than assumed to be a
win. The current fixed ring is ~9.5-10.2 ns/op serial and ~26-36 ns/op at
8-way / ~43-49 ns/op at 32-way parallelism on the harness host. A 16-shard
prototype regressed to ~14.6-15.1 ns/op serial, ~42 ns/op at 8-way, and ~61
ns/op at 32-way because the global sequence atomic plus shard lock outweighed
the short existing mutex critical section. Production stays on the simple fixed
ring until real profiling says otherwise.

### Remaining non-XDP/VectorScan work

No further host-independent low-risk dataplane change is currently justified by
code inspection alone. TLS/kTLS/QAT/external TLS termination is still a separate
deployment/hardware decision and must not be enabled speculatively. Real Coraza
and `wafbench` results should drive any further dataplane work.

## 2026-09-04 — TLS Acceleration C1-C3 + modern NGINX HTTP/2

Implemented an optional OpenSSL/NGINX TLS acceleration plane without changing
the default Go TLS behavior. New `waf-tlsfront` owns public TLS listeners only
when `tls_acceleration.mode=frontend`; waf-proxy remaps those listeners to
private Unix-domain HTTP sockets and keeps Coraza/policy/backend processing in
the existing Go data plane. Private client IP/port/proto metadata is accepted
only on those Unix listeners and all Internet-supplied forwarding metadata is
cleared before the authoritative chain is rebuilt.

Configuration supports `mode=go|frontend`, `ktls=off|auto|required`,
`qat=off|auto|required`, NGINX worker settings, and HTTP/2. Apply performs
capability detection plus `nginx -t` before changing the live runtime. Auto
modes fail open to OpenSSL software TLS; required modes fail closed. QAT uses
OpenSSL 3 provider configuration (`qatprovider` plus `default`) and is not
claimed real-validated without QAT hardware/provider on the host.

Modern NGINX compatibility: version detection selects standalone `http2 on;`
for NGINX >= 1.25.1 and legacy `listen ... http2` only for older releases. Real
NGINX 1.26.3/OpenSSL 3.5.5 `nginx -t` passed with no deprecation warning. A real
HTTPS/HTTP2 smoke to the generated NGINX config and Unix backend passed and
showed NGINX-overwritten client metadata. On the current host kTLS kernel
capability and Intel `qatprovider` are unavailable, so `auto` correctly fell
back to software TLS.

Verification: direct `internal/tlsfront` tests pass; external minimal Coraza API
stub full `go test -count=1 ./...`, `go vet ./...`, and `go test -race
-count=1 ./...` pass. Real Coraza v3.3.2 remains NOT_RUN because this isolated
environment cannot resolve `proxy.golang.org`. Next performance decision remains
independent: use real wafbench data before selecting XDP or VectorScan.


## 2026-09-04 — VectorScan Learning Accelerator + Coraza v3.7 transaction truth

**Current implementation line:** TLS/modern-NGINX baseline plus optional VectorScan Learning Accelerator. Coraza is authoritative. Source `go.mod` is now Go 1.25.0 with `github.com/corazawaf/coraza/v3 v3.7.0`.

### Architecture / invariants

- `internal/vectoraccel` parses SecLang sources conservatively and groups only eligible standalone positive `@rx` rules whose request input can be reconstructed exactly. Initial sources: `REQUEST_URI`, `REQUEST_FILENAME`, `REQUEST_METHOD`, `REQUEST_PROTOCOL`, fixed-name `REQUEST_HEADERS:name`; transforms require explicit `t:none` and may add only `t:lowercase`. Chains, negation, ARGS/body, multi/aggregate variables and other transforms remain Coraza-only.
- Native acceleration is build-tagged `vectorscan && cgo`, uses libhs `hs_compile_multi`, reusable scratch, cgo.Handle callback IDs carried in C-owned `uintptr_t` context, and RWMutex-protected scan/close lifecycle. Portable builds use the stub factory and never claim native acceleration.
- Per group state is `CORAZA_ONLY`, `LEARNING`, `VALIDATED`, `ACCELERATED`, or `FAILSAFE`. Learning state persists to the configured state file. Runtime fingerprint combines rule fingerprint, native VectorScan version and `coraza-v3.7.0-matchedrules-v1`; drift invalidates stale learning.
- VectorScan executes before Coraza. Only an already-ACCELERATED group with a no-hit result may be skipped, and verification sampling periodically forces full Coraza evaluation. Skip control uses an internal request header stripped at Internet ingress and reverse-proxy egress.
- Learning truth does **not** use ErrorCallback. `coraza_observer.go` preserves Coraza's WAF interfaces, including `experimental.WAFWithOptions`, captures the request-specific Observation from the request context, and observes transaction-final `MatchedRules()` exactly once after `ProcessLogging()` (with `Close()` fallback). This avoids HTTP/2 same-client/same-URI correlation ambiguity.
- Coraza matches are counted once per transaction per group. Any actual matched rule missing from the VectorScan candidate set increments false negatives and immediately sets that group to `FAILSAFE`; native scan errors also fail safe. Reviewer-or-higher may explicitly reset a site's FAILSAFE groups to Learning.

### API/config/build

- Config object: `vector_acceleration.mode=off|auto|required`, `state_path`, `min_samples`, `min_coraza_matches`, `min_learning_sec`, `verification_sample_rate`. Defaults keep acceleration off.
- `GET /api/vector-acceleration`; `POST /api/vector-acceleration/reset`; `/api/metrics` includes `vector_acceleration`.
- `build.sh` requires Go >=1.25.0 and supports `WAF_VECTORSCAN=auto|required|off`; native detection uses `pkg-config libhs`. The build script runs vet/test/race and a `realcoraza` transaction-truth gate on a capable release host.

### Verification truth boundary

Passed in the packaging environment:
- external Coraza v3.7 API-stub: full repository `go test`, `go vet`, and bounded `go test -race` groups;
- native libhs ABI-only test library: vectorscan-tag compile/test/vet/race, including the CGo callback/pointer and scan/close paths;
- release shell syntax/line endings and source hygiene checks.

**NOT_RUN / not claimed:** real Go 1.25 toolchain execution of Coraza v3.7.0, real Coraza DetectionOnly+nolog `MatchedRules()` gate, real libvectorscan `hs_compile_multi/hs_scan`, production CRS coverage/performance. The current container has an older Go toolchain and cannot obtain the required real modules/toolchain. These are release-host gates, not reasons to substitute stub results.

### Next engineering step

Run `WAF_VECTORSCAN=required ./build.sh` on a Go 1.25+ host with real `libvectorscan-dev`, then run DetectionOnly Learning with production CRS/corpus. Do not enable or preserve `ACCELERATED` state across a semantic fingerprint change; the implementation will re-learn automatically. XDP remains separate and deferred.

## 2026-09-07 — New-chat handover checkpoint

This checkpoint is documentation-only and does not change production behavior. Added `DEVELOPMENT_ROADMAP.md`, `TESTING_RESULTS.md`, `HANDOVER_STATUS.md`, and `HANDOVER_PROMPT.md` so a new chat can continue without reconstructing the implementation history. VectorScan remains the selected acceleration direction; XDP is deferred and must not be reintroduced as a selection prerequisite. The immediate release engineering priority is real Go 1.25 + Coraza v3.7.0 + real libvectorscan qualification on a capable Linux host, followed by DetectionOnly Learning Period qualification with zero observed false negatives. Stub/API and ABI-only gates remain explicitly non-production evidence. The project completion timestamp convention in `AGENTS.md` is synchronized to the owner's required `UTM+8: YYYY-MM-DD HH:MM:SS` format.

## 2026-09-07 — Phase 0 release-host qualification automation

A focused qualification slice was added because this packaging host still cannot execute the real Go 1.25 + Coraza v3.7.0 + real VectorScan gates. Production WAF request-processing semantics were not broadened or refactored.

Changes:

- Added `qualify-release-host.sh` with `--preflight` and `--core`. It distinguishes **BLOCKED** release-host prerequisites (exit 3) from a correctness **FAIL** (exit 1), checks the installed local Go toolchain without auto-download, validates exact Go/Coraza pins, CGO/compiler and `pkg-config libhs`, and requires real VectorScan provenance through Debian `libvectorscan-dev` metadata or an explicit reviewed source-install attestation.
- Added `internal/vectoraccel/scanner_native_gate_test.go` under `vectorscan && cgo`. It compiles two native expressions, scans two-hit/clean inputs, and verifies the exact rule-ID set, directly covering the `hs_compile_multi`/`hs_scan` semantic path. This must only be counted as real VectorScan evidence after provenance is established; ABI shims remain non-production evidence.
- Changed `build.sh` to inspect the installed toolchain with `GOTOOLCHAIN=local go version`; an old host now fails with the explicit Go >=1.25 requirement instead of first attempting an implicit Go toolchain download.
- Updated README, INSTALL, MANIFEST, DEVELOPMENT_ROADMAP and TESTING_RESULTS with the new qualification workflow and truth boundary.

Evidence executed on this host:

- release shell `bash -n`: PASS;
- `qualify-release-host.sh --preflight`: expected BLOCKED/exit 3 because local Go is 1.23.2 and real `libhs`/VectorScan provenance is unavailable;
- synthetic command-shim preflight branch: PRECHECK_PASS; control-flow validation only, not real release-host evidence;
- `GOTOOLCHAIN=local WAF_VECTORSCAN=required ./build.sh`: expected early rejection for Go 1.23.2 with no toolchain/network download attempt.

Still NOT_RUN: real Coraza v3.7.0 tests, DetectionOnly+nolog `MatchedRules()` truth gate, real VectorScan semantic test/race, clean-VM install/upgrade/uninstall/doctor smoke, and production CRS Learning Period qualification.

Exact next step: on a Linux release host with Go >=1.25.0 and verified real libvectorscan, run `./qualify-release-host.sh --preflight` and then `./qualify-release-host.sh --core`. Only after `CORE_PASS` should Phase 1 DetectionOnly Learning qualification begin; zero observed false negatives remains mandatory.

## 2026-09-09 — Mandatory Artifact Packaging Integrity Gate

A cross-project release-safety rule is now implemented in this WAF source line. The motivating mini-SIEM incident was a generated ZIP in which an installer script was accidentally replaced by README content even though the source repository and live application data were intact. The WAF release process must therefore validate the extracted artifact itself, not infer package correctness from source tests.

New files:

- `RELEASE_PROCESS.md` — canonical packaging-integrity policy, incident summary and release sequence;
- `DEVELOPMENT.md` — cross-cutting engineering ledger entry;
- `TESTING.md` — artifact test-policy entry;
- `build-release-artifact.sh` — stages a clean complete-source tree, generates `RELEASE_MANIFEST.txt`, creates a deterministic ZIP, invokes the verifier and emits the ZIP SHA-256;
- `verify-release-artifact.sh` — validates ZIP CRC/path safety, rejects symlinks, checks required files and critical scripts, runs shell syntax checks against extracted scripts, compares source/package SHA-256 and modes, and verifies manifest hashes.

This is now a mandatory release gate. It does not change the Coraza/VectorScan architecture or Phase 0 truth boundary. Real Go 1.25/Coraza v3.7.0 and real libvectorscan release-host qualification remains required independently.

Artifact-gate validation executed on 2026-09-09:

- all shipped shell scripts: `bash -n` PASS;
- clean generated complete-source ZIP: artifact verifier PASS;
- 117 source files: extracted ZIP SHA-256 and Unix modes matched the source tree;
- generated `RELEASE_MANIFEST.txt`: file-set/hash verification PASS;
- simulated mini-SIEM-class corruption (`install.sh` replaced with README bytes): correctly rejected;
- simulated ZIP path traversal (`../escape.txt`): correctly rejected.

During validation, the verifier was corrected to restore stored Unix modes from ZIP metadata after Python extraction before permission checks. Final packaging must use the corrected verifier and rebuild the ZIP after documentation/evidence updates.

The 2026-09-09 packaging run also demonstrated why patch reconstruction is mandatory: an initial `git diff --binary` omitted five new untracked files. Reconstruction failed, the patch-generation procedure was corrected to include intent-to-add paths before diffing, and only the corrected reconstruction is valid release evidence.

## 2026-09-10 Continuation

Added Debug Evidence Capture foundation and Phase 1 qualification tracking. Existing Coraza authoritative boundary remains unchanged. Real release-host qualification remains mandatory.

## Latest slice

Debug Evidence now consumes transaction-final Coraza MatchedRules evidence and keeps VectorScan qualification based on candidate coverage with zero false-negative requirement.

## Current Slice

Debug evidence supportability foundation added. Continue with CLI integration, full capture wiring, and release qualification when real dependencies are available.

## 2026-09-13 — Debug/supportability and Phase 1 qualification-runner completion

Current source baseline now contains the operator/debug/qualification implementation requested after the 2026-09-10 supportability foundation.

Implemented source behavior:

- `debugEvidenceWrap` is opt-in per configured site and creates a server-side transaction ID only while capture is enabled. It records bounded request metadata (no body; allow-listed headers), TLS metadata and response status/size/latency.
- The same request context flows through VectorScan, Coraza and the load-balancer transport. `coraza_observer.go` captures transaction-final `tx.MatchedRules()` after `ProcessLogging()` even when VectorScan is disabled/no eligible plan exists. When VectorScan did run, the bundle also records candidates, eligible Coraza matches and FN IDs; otherwise it says `observed:false`.
- Debug evidence is exact-site scoped, count/TTL bounded and periodically cleaned. Sensitive map keys are masked. Debug start/stop/export are audit events. Evidence listing/export requires reviewer-or-higher; enabling/stopping capture requires operator-or-higher.
- Admin API adds `/api/doctor` plus debug status/capture/evidence/export endpoints.
- `cmd/wafctl` implements `doctor`, `debug capture|stop|list|export` and `support bundle`. Incident/support ZIPs are deterministic-entry diagnostic artifacts with manifests/checksums and 0600 local output. Support bundle collects sanitized config/status/metrics, bounded logs when available, dependency evidence and SPDX-formatted SBOM evidence.
- `cmd/wafqualify`, `run-phase1-qualification.sh` and `qualification/corpus/default.jsonl` implement the real Phase 1 differential runner. On a qualified host it builds the actual `vectorscan` CGo path, uses the same native plan scanner, runs real Coraza DetectionOnly transactions, reads final `MatchedRules()` and compares only currently eligible rules. PASS requires zero observed FNs plus a minimum eligible-match count; native scan/FN is FAIL and missing prerequisites/eligible evidence is BLOCKED (exit 3).
- `build.sh`, install/upgrade/uninstall, `waf-doctor.sh`, source-package build/verify scripts are synchronized for the new `wafctl` binary and qualification sources.

Executed on this packaging host on 2026-09-13:

- host local Go: 1.23.2; real `pkg-config libhs`: unavailable;
- root release shell `bash -n`: PASS;
- `GO111MODULE=off GOTOOLCHAIN=local go test ./cmd/wafctl`: PASS;
- standalone stdlib-only `wafctl` build/version smoke: PASS;
- full `GOTOOLCHAIN=local go test ./...`: BLOCKED before execution because go.mod requires Go >=1.25.0;
- `./qualify-release-host.sh --preflight`: BLOCKED/exit 3 for Go 1.23.2 and missing real libhs/provenance;
- `./run-phase1-qualification.sh --rules ./coraza.conf`: BLOCKED/exit 3 because real libhs is absent.

Do not promote any source-level runner/test or BLOCKED transcript into real Coraza/VectorScan/CRS qualification. Exact next live step remains Phase 0 `--core` on Go >=1.25 + verified real libvectorscan, followed by the Phase 1 corpus runner with representative CRS/traffic and zero observed false negatives.

Packaging note: the 2026-09-13 source/doc tree passed a pre-final `build-release-artifact.sh` clean-extraction rehearsal with `ARTIFACT_INTEGRITY_PASS files=131`. The final delivery is rebuilt after ledger synchronization and independently verified/reconstructed before handoff.


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

## 2026-09-14 — Phase 2 Coverage Expansion Slice C handoff

Slice C is implemented on top of the Slice B baseline.

Current source now has a CRS ruleset coverage analyzer rather than only single-rule metadata helpers. `internal/coverage/crs` can load a directory of `.conf` files or follow `Include`/`IncludeOptional` from a config entrypoint, normalize SecRule metadata, identify duplicate IDs, build a versioned capability inventory and produce Coverage Report v2. `wafctl coverage analyze` exposes this locally without requiring the admin API.

Critical semantic decision: coverage eligibility must never be broader than `internal/vectoraccel`'s real dataplane classifier. Slice C therefore corrects the earlier analyzer model to require explicit positive `@rx`, phase 1/2, exactly one supported value (`REQUEST_URI`, `REQUEST_FILENAME`, `REQUEST_METHOD`, `REQUEST_PROTOCOL`, or fixed-name `REQUEST_HEADERS:<name>`), explicit `t:none`, and only optional `t:lowercase`. Bare header collections, uppercase, chains, negation, ARGS/body and multi-selector rules remain Coraza-only.

Analyzer `ELIGIBLE` means only "compatible with the current parser/runtime scope". It is not LEARNING/VALIDATED/ACCELERATED evidence. Phase 0 real release-host qualification and Phase 1 representative CRS differential qualification with zero observed false negatives are still mandatory and remain NOT_RUN on the packaging host.

Next recommended Phase 2 work after final validation: use representative real CRS inventory results to select the next exact transform or fixed-input expansion, implement byte-for-byte Coraza input parity tests first, then extend the runtime classifier and analyzer together. Do not widen the analyzer ahead of the dataplane.

Slice C packaging-host verification: shell syntax passed; canonical Go tests are BLOCKED by local Go 1.23.2 versus the required 1.25.0 and real libhs is absent. A temporary isolated Go 1.23 module containing only the new coverage packages plus `cmd/wafctl` passed focused tests and vet, and a synthetic Include-based `wafctl coverage analyze` smoke produced the expected 2/5 eligible classification. Treat those only as source-level regression evidence; representative CRS, real Coraza and real VectorScan qualification remain NOT_RUN.

Slice C artifact gate also exposed and fixed a supplied-baseline mode defect: release/install scripts in the Slice B ZIP were stored as `0644`. Slice C restores the expected `0755` modes. Clean ZIP verification and baseline-relative patch reconstruction pass with byte-and-mode equality. The final ZIP remains a development source baseline, not evidence that Phase 0/1 real qualification passed.

## Phase 3 release-hardening handoff — 2026-09-14

Formal roadmap Phase 3 is implemented in source and remains qualification-bound.

Canonical release-hardening tools:

- `build-release-artifact.sh`: deterministic complete-source packaging, explicit build variant, generated release evidence/SBOMs, optional minisign.
- `verify-release-artifact.sh`: mandatory clean-extraction integrity/supply-chain gate.
- `release-security-scan.sh`: govulncheck evidence with PASS/FAIL/BLOCKED/NOT_RUN semantics.
- `release-artifact-negative-tests.sh`: corrupt/malicious ZIP rejection suite.
- `verify-reproducible-source-release.sh`: two-build byte reproducibility gate.
- `tools/release_evidence.py`: standard-library provenance + SPDX 2.3 + CycloneDX 1.5 generator.

Do not call Phase 3 fully qualified until the required release host can execute govulncheck and the relevant Phase 0/1 live/runtime gates. Organizational artifact signing is NOT_CONFIGURED unless an approved minisign secret key is supplied externally; never add a signing private key to this repository or release ZIP.

Phase 3 local validation result: shell/Python syntax, complete-source artifact integrity, six negative archive mutations, and fixed-epoch byte reproducibility passed. Packaging host remains BLOCKED for canonical Go tests and native Phase 0/1 gates because it has Go 1.23.2 and no real libhs; govulncheck is unavailable and signing is NOT_CONFIGURED. NGINX 1.26.3 / OpenSSL 3.5.5 were detected by preflight. Preserve these truth boundaries in the next session.

## Current Override — 2026-09-14 Phase 3 Truth-Boundary Repair

This section supersedes older build/test wording above where it conflicts. The current source requires Go 1.25.0 and `build.sh` uses `go mod tidy -diff` rather than mutating dependency files. The exact current baseline has not rerun the full canonical Go 1.25 suite in the packaging environment (Go 1.23.2); historical stub/ABI PASS evidence must stay labeled historical.

Coverage and live VectorScan eligibility now share `internal/capability`; do not reintroduce independent classifier logic. Runtime and coverage both handle `IncludeOptional` consistently. Analyzer eligibility is not promotion evidence.

The source release builder emits only `SOURCE_ARCHIVE`. Actual binary type/variant belongs to `BUILD_PROVENANCE.json` produced after a real build. Release evidence schema v2, source-bound govulncheck evidence, complete source/release manifests, `PROVENANCE.json` cross-digests, and SBOM↔`go.mod` checks are mandatory artifact gates. Unsigned hashes are not producer authentication; only a detached signature verified against an approved public key establishes that boundary.

Phase 0/1 real Go 1.25 + Coraza v3.7 + libvectorscan qualification remains NOT_RUN/BLOCKED on the packaging host and must not be inferred from source/artifact hardening PASS results.
