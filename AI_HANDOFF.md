# AI Development Handoff


## Current canonical addendum — 2026-09-30 — TLS Session Resumption + Handshake Observability

This source now includes the **TLS Session Resumption + Handshake Observability**
slice on top of `waf-proxy-build-admin-startup-blocker-fix-2026-09-30.zip`
(parent SHA-256 `49450dfffeca2e45f17def136e122fac115c8fba4e8beb70a9307fe5eb6cb0e3`).
Project state remains **IMPLEMENTED_TESTING_DEFERRED**. This is an operational
TLS/performance slice, **not API-9**; API-1 through API-8 enforcement and state
authority boundaries are unchanged.

The built-in Go TLS path can now use a purpose-separated runtime-only shared
session-ticket secret to derive a deterministic rotating current+previous key
ring for restart-stable and HA cross-node resumption. The secret is supplied by
`WAF_TLS_SESSION_TICKET_SECRET` or a private
`WAF_TLS_SESSION_TICKET_SECRET_FILE`; it is not persisted in Config, returned by
Admin APIs, HA-synchronized, or logged, and must not reuse the HA peer-token
value. When unset, Go process-local automatic tickets remain the compatibility
fallback and cross-node/restart resumption is not claimed. External TLS frontend
termination remains outside this key manager.

Handshake observability now records authoritative successful full/resumed state
from `tls.ConnectionState.DidResume`, TLS 1.2/1.3, full-handshake certificate key
algorithm, handshake-processing average/p95, and policy rejects. `/api/metrics`,
`/api/status`, and the shipping Console expose the new data. See
`TLS_SESSION_RESUMPTION_HANDSHAKE_OBSERVABILITY.md`.

Current evidence: dedicated TLS source gate **33/33 PASS**, root Go source-shape
**99 files PASS**, existing API/static source gates remain PASS, duplication and
route gate remains **80/80 with 139 Admin/update patterns**, blocker-fix route
gate remains **13/13 with 139 patterns**, package-source PASS, and package-builder
**9/9 PASS**. A dependency-free isolated exact-core TLS probe on Go 1.23.2 passed all five
repository TLS tests, including TLS 1.2/TLS 1.3 cross-config resumption, and
passed `go test -race`; it is not canonical repository qualification. Exact-source Go 1.25 tidy/build/vet/test/race
and runtime package qualification remain **BLOCKED_ENVIRONMENT / NOT_RUN** here
because the required toolchain/modules cannot be retrieved on this host. No
`TESTED` or `RELEASED` promotion is made.

This section supersedes earlier sections labelled current/canonical when they
conflict; dated historical evidence below remains historical.

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

## Build-integrity and L7 correction — 2026-09-23

An external Go 1.25 CI audit of the previous API-1→API-3 delivery showed that archive integrity and isolated slice tests had not proved package buildability. The current working source repairs the package-level defects found in that audit and an additional duplicate symbol found during follow-up:

- removed the named-result `err` redeclaration in `ai.go`;
- removed the duplicate `testAIEngineWithoutWorkers` test helper;
- consolidated the duplicate root-package `normalizeHost` helper;
- retained the correct two-argument `writeJSON` / three-argument `writeJSONCode` split and added a dependency-free root source-shape gate to reject wrong-arity `writeJSON`, duplicate top-level declarations, and named-result `var` redeclarations;
- corrected the client-identity proxy test so the normal XFF path and the fail-closed conflicting-XFF/X-Real-IP path are tested separately.

Phase 4 Slice B L7 abuse control is no longer a fixed-window/global-lock implementation. It now uses a continuous token bucket keyed by site + trusted client identity, 64 state shards, per-entry locking, bounded state (1024 entries/shard), amortized idle pruning/idle eviction, and deferred active-request release so downstream panic cannot leak concurrency state. The existing configuration remains global; per-site/per-page limit overrides and TLS-handshake-rate enforcement are still not implemented and must not be claimed.

Current local evidence is deliberately split by scope: the dependency-free root source-shape gate passes; a temporary local external-dependency stub harness makes the whole repository pass `go build ./...` and `go test -run '^$' ./...` type/test-binary compilation; targeted client-identity/L7/AI/API tests pass there; L7 isolated unit + race tests pass; and the local 5-worker limiter microbenchmark is 162.5–179.7 ns/op versus 330.6–337.3 ns/op for the previous global-lock implementation. The benchmark is a control-path microbenchmark, not production proxy throughput evidence.

**Source Buildability remains BLOCKED, not PASS.** The canonical Go 1.25 real-dependency `go mod tidy -diff`, build, vet, full tests, race, and real-Coraza gates have not run on these exact bytes in this environment. The prior external CI also reported committed module metadata drift; no guessed `go.mod` edit is accepted as a substitute for the exact Go 1.25 tidy result.

## API Security checkpoint — 2026-09-23

The uploaded API-3 baseline was audited against source rather than roadmap labels, then repaired/closed through API-3. Current source truth:

| Slice | State | Current evidence |
|---|---|---|
| API-1 Discovery + Operation Normalization | `IMPLEMENTED_TESTING_DEFERRED` | durable operation inventory, metadata, controls/UI; 5/5 exact-source isolated tests PASS |
| API-2 Typed Schema Learning | `IMPLEMENTED_TESTING_DEFERRED` | live typed collector, lifecycle, privacy controls, persistence; 8/8 exact-source isolated tests PASS |
| OpenAI SchemaCandidate review | `IMPLEMENTED_TESTING_DEFERRED` | Responses API Structured Outputs, advisory-only; OpenAI source/isolated gates PASS |
| API-3 OpenAPI Contract Management | `IMPLEMENTED_TESTING_DEFERRED` | persistence, import/export, params/enums/security, scoped drift; 47-check source gate + deterministic isolated/integration tests PASS |
| API-4 Positive Schema Enforcement | `IMPLEMENTED_TESTING_DEFERRED` | immutable reviewed profiles; LEARN/DETECT/ENFORCE, exceptions, rollback, bounded durable violation evidence; API-4 7/7 + race + 46-check source gate PASS |
| API-5 JWT + Identity-aware Policy | `IMPLEMENTED_TESTING_DEFERRED` | verified JWT/JWKS identity context, DETECT/ENFORCE operation policy, bounded durable evidence; API-5 17-test + race + 72-check source gate PASS |
| API-6.1 Sequence Foundation | `IMPLEMENTED_TESTING_DEFERRED` | bounded async sessions/transitions, normalized operation IDs, verified/keyed-private correlation, TTL/caps, atomic snapshots, persistence/admin visibility; source gate 33 PASS; Go tests/race `BLOCKED_ENVIRONMENT/NOT_RUN` |
| API-6.2 Workflow Learning | `IMPLEMENTED_TESTING_DEFERRED` | LEARN-only workflow model; source gate 58 PASS; Go 1.25 targeted/race BLOCKED_ENVIRONMENT/NOT_RUN |
| API-6.3 Sequence Anomaly Detection | `IMPLEMENTED_TESTING_DEFERRED` | DETECT-only bounded anomaly evidence; source gate 45 PASS; Go 1.25 targeted/race `BLOCKED_ENVIRONMENT/NOT_RUN` |
| API-6.4 Sequence Operations + Hardening | `IMPLEMENTED_TESTING_DEFERRED` | explicit bounded LEARN/DETECT controls, exception CRUD, reset/relearn, recent sessions and full operations console; source gate 58 PASS; Go 1.25 targeted/race `BLOCKED_ENVIRONMENT/NOT_RUN` |
| API-7.1 Object Locator Discovery | `IMPLEMENTED_TESTING_DEFERRED` | normalized locator discovery + bounded keyed evidence; source gate 83 PASS; Go 1.25 targeted/race BLOCKED_ENVIRONMENT/NOT_RUN |
| API-7.2 Identity/Object Relationship | `IMPLEMENTED_TESTING_DEFERRED` | verified identity pseudonym ↔ keyed object relationship evidence; source gate 100 PASS; Go 1.25 targeted/race `BLOCKED_ENVIRONMENT/NOT_RUN` |
| API-7.3 BOLA Detection | `IMPLEMENTED_TESTING_DEFERRED` | bounded DETECT-only identity/object divergence, tenant divergence and enumeration candidate evidence; source gate 110 PASS; Go 1.25 targeted/race `BLOCKED_ENVIRONMENT/NOT_RUN` |
| API-7.4 BOLA Policy/Evidence/Console | `IMPLEMENTED_TESTING_DEFERRED` | bounded REVIEW/SUPPRESS evidence policy + ACK/DISMISS/RESOLVE/REOPEN workflow + console; source gate 156 PASS; Go 1.25 targeted/race `BLOCKED_ENVIRONMENT/NOT_RUN` |
| API-8 GraphQL Security | `IMPLEMENTED_TESTING_DEFERRED` | bounded GraphQL parsing/normalization, SDL contract binding, depth/complexity/field/mutation/subscription/introspection/APQ controls, verified-variable API-7 bridge, staged deterministic LEARN/DETECT/ENFORCE; source gate 259 PASS; Go 1.25 targeted/race `BLOCKED_ENVIRONMENT/NOT_RUN` |

API-1 → API-2 → API-3 deterministic end-to-end learning/drift evidence is PASS. The repository-wide Source Buildability Gate remains **BLOCKED**, not PASS: this host has Go 1.23.2 while `go.mod` requires Go 1.25.0, and network/toolchain acquisition is unavailable. Required Go 1.25 `tidy/build/vet/test/race/real-Coraza` qualification remains `NOT_RUN/BLOCKED`. See `API_SECURITY_CLOSURE_RESULT.md`.

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
2. The hot-path limiter core is now implemented as a sharded continuous token
   bucket keyed by site + resolved identity, with bounded/pruned state and a
   concurrent-request cap. Remaining work from the original plan is per-site and
   per-PagePolicy configuration overrides, optional time-boxed blocklist action,
   and actual TLS-handshake-rate enforcement. Do not describe those remaining
   items as implemented. Slowloris coverage remains through
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

## Phase 4 Slice A progress

Trusted client identity work extends the existing client IP resolver. It does not create a second identity path. Client identity evidence is attached to debug capture and operator visibility is provided through `wafctl proxy identity show`.

Status: IMPLEMENTED_TESTING_DEFERRED


Phase 4 Slice A follow-up: added client identity audit event model constants CLIENT_IDENTITY_RESOLVED and CLIENT_IDENTITY_HEADER_REJECTED.

## Phase 4 Slice B — L7 Abuse Controls (Implementation Start)

Status: IMPLEMENTED_TESTING_DEFERRED

Implemented foundation:
- trusted client identity context reused from Slice A
- configurable L7 abuse middleware boundary
- continuous token bucket keyed by site + trusted client identity
- 64-way sharded state with per-entry locks; no proxy-wide limiter mutex
- bounded state at 1024 entries/shard with amortized idle pruning and oldest-idle eviction
- panic-safe deferred concurrent-request release
- 429 enforcement evidence boundary
- deterministic unit/race coverage plus a local control-path microbenchmark

Not claimed:
- per-site or per-PagePolicy limit configuration overrides
- TLS handshake-rate enforcement (`TLSHandshakePerWindow` is not an implemented enforcement path)
- production traffic/load qualification
- production end-to-end throughput improvement from the microbenchmark
- release readiness

## Phase 4 Slice C — Manual CIDR Policy (Roadmap Entry)

Status: PLANNED

Scope boundary:
- manual CIDR allow policy
- manual CIDR deny policy
- priority evaluation
- policy match evidence
- optional TTL handling

Dependency:
- reuse Slice A ClientIdentityDecision resolved client identity

Not implemented:
- CIDR enforcement engine
- policy persistence
- cleanup worker
- distributed policy sync


## Phase 4 Slice C — Manual CIDR Policy
Status: IMPLEMENTED_TESTING_DEFERRED
Added CIDR policy engine foundation using trusted client identity context.

## Phase 4 Slice D — Custom Block Page / Correlation
Status: IMPLEMENTED_TESTING_DEFERRED
Added correlation ID foundation, reusable block response generation, and security event model foundation.


## Phase 4 Slice E — Persistent Security State

Status: IMPLEMENTED_TESTING_DEFERRED

Implemented foundation: security state models and in-memory persistence abstraction. Production database durability, HA replication, retention tuning, and external integrations remain deferred.


## Phase 4 Slice F PKI Slice 3 Hardening
- Added CRL retrieval boundary hardening foundation.
- Added SSRF-oriented URL validation and Last Known Good CRL retention helpers.


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


## Phase 5 Slice D — Security Operations Experience

Status: IMPLEMENTED_TESTING_DEFERRED

Implemented foundation: security timeline events, investigation query model, change audit model, and structured evidence export. Production SOC workflow and SIEM integration remain deferred.


## Current Work — Phase 5 Slice E

Reliability Qualification foundation implemented. Future sessions must preserve the boundary between reliability framework implementation and executed production qualification evidence.

## Current Work — Phase 5 Slice F Performance Certification

Status: **IMPLEMENTED_TESTING_DEFERRED**.

Canonical implementation:

- `cmd/wafbench/certify.go` — performance certification evidence binding, target evaluation, comparability checks, VectorScan security gate, and regression deltas.
- `cmd/wafbench/certify_test.go` — focused truth-boundary/unit tests.
- `cmd/wafbench/main.go` — exposes `wafbench certify`.
- `cmd/wafbench/model.go` — wafbench evidence version advanced to 1.1.0.
- `qualification/performance/README.md` — operator evidence contract.
- `qualification/performance/target.example.json` — explicitly non-approved target example.
- `qualification/performance/performance-certification-NOT_RUN.json` — placeholder proving no real benchmark was executed on the packaging host.

Certification semantics:

1. `evidence_status=PASS` means only that real benchmark result files are present, executed, hash-bound, and comparable.
2. `target_status=PASS` requires an explicit `waf-proxy-performance-target-v1` target and measured results meeting every stated requirement.
3. final `status=PASS` requires both of the above. If VectorScan results are supplied, it also requires the real `waf-phase1-vectorscan-differential-v1` zero-false-negative PASS report.
4. No target => final `NOT_RUN`; incompatible run shape/system => `BLOCKED`; target miss or failed VectorScan safety evidence => `FAIL`/`BLOCKED` as applicable.

Do not convert component microbenchmarks, theoretical throughput, or the example target into deployment sizing claims. Application Gbps in the report is measured payload throughput (`AppMbps/1000`), not Ethernet line rate.

Baseline hygiene repair completed in this slice: restored the five unique Phase 5 Slice B `qualification/vectorscan` files from the stale nested `waf-work/` mirror into the canonical path, corrected the replay package mismatch, and removed the duplicate tree.

Focused local validation completed with Go 1.23 in isolated standard-library-only harnesses. Canonical Go 1.25 module validation and all real performance qualification remain deferred/blocked on this packaging host.

Next roadmap item after Slice F implementation is **Phase 5 Slice G — External HSM / PKCS#11 Support**. Do not start it by redesigning TLS/PKI; reuse existing PKI/TLS boundaries and keep private keys non-exportable.

Slice F packaging note: the incoming Slice E ZIP regressed executable modes on release/install tooling to `0644`. Slice F restores all verifier-required shell scripts and executable Python release tools to `0755`. Future packaging must preserve those modes; `verify-release-artifact.sh` treats mode loss as an artifact failure.

## Current Work — Phase 5 Slice G External HSM / PKCS#11 Support

Status: **IMPLEMENTED_TESTING_DEFERRED**.

Canonical implementation now includes:

- `internal/hsm/` — config validation, secret-reference resolution, provider/signer abstractions, module/session lifecycle, Linux PKCS#11 CGO adapter, build-disabled stub, health model and audit schema.
- `hsm_integration.go` — site TLS-provider validation, no-fallback rules, config secret-reference redaction/preservation and restricted audit forwarding.
- `main.go` — HSM runtime config, per-site key-provider config, and HSM-backed `tls.Certificate` construction during runtime build.
- `admin.go` — `/api/hsm/status`, `/api/hsm/audit`, and HSM secret-reference redaction on config GET plus draft/applied PUT responses.
- `syslog.go` — HSM audit forwarding with only the approved fields.
- `cmd/hsmqualify/` and `qualification/hsm/` — explicit SoftHSM vs real-vendor qualification paths with separate evidence classes and checked-in `NOT_RUN` placeholders.
- `build.sh` — independent `WAF_HSM_PKCS11=off|auto|required` capability and `pkcs11` build tag. A Coraza-only binary can now be HSM-capable without enabling VectorScan.

Security invariants that future work must preserve:

1. private keys remain inside the PKCS#11 provider; the WAF receives only signing operations;
2. no automatic fallback from HSM to `tls_key` or the external TLS frontend;
3. certificate ↔ HSM key association must be proven before runtime swap;
4. PINs and PIN references must not appear in logs, audit evidence, qualification reports, or admin config responses;
5. HSM audit payload fields are limited to provider, slot, key reference, operation, and result;
6. module loading is allow-listed and Linux ownership/permission checked;
7. SoftHSM PASS is not real-vendor HSM PASS.

Current execution evidence is focused/local only: HSM isolated tests PASS, but SoftHSM runtime and real vendor HSM qualification are `NOT_RUN`; canonical Go 1.25 repository validation is still blocked on this packaging host.

After Slice G packaging, do not invent a new feature slice automatically. The next engineering focus should be concentrated execution/closure of the already-defined Phase 0/1/5 deferred qualification gates on suitable release/production-like hosts.

Slice G delivery-gate state: baseline-relative patch reconstruction PASS (215 source-tree files byte/mode identical); mandatory release artifact verifier PASS; fixed-epoch reproducible source-release check PASS; all 12 negative artifact mutations were rejected across segmented execution; detached signature remains NOT_CONFIGURED. These are source/supply-chain gates only and do not promote SoftHSM/vendor-HSM qualification.

## Current Work — Enterprise Linux Distribution Packaging Slice A

Slice A — Debian / Ubuntu DEB Packaging is implemented in source and remains `IMPLEMENTED_TESTING_DEFERRED`.

Implemented:

- `packaging/deb/build-release-deb.sh` builds the canonical binaries and passes their provenance-bound bytes to the package builder;
- `packaging/deb/build-deb.sh` creates a deterministic `.deb` from `waf-proxy`, `wafctl`, `waf-tlsfront`, `BUILD_PROVENANCE.json`, and `BUILD_SHA256SUMS.txt`;
- package paths use `/usr/bin`, `/usr/sbin`, `/lib/systemd/system`, `/etc/waf`, `/var/lib/waf-proxy`, and `/usr/share/doc/waf-proxy`;
- `config.json`, `coraza.conf`, and `waf-tls-frontend.env` are dpkg conffiles;
- fresh install creates `waf-proxy.env` once, does not print the token, and never rotates it on upgrade;
- package maintainer scripts contain no network fetch and never obtain CRS automatically;
- fresh install does not auto-start before explicit CRS provisioning;
- systemd now declares `StateDirectory=waf-proxy`, fixing persistent learner/security-state writeability under `ProtectSystem=strict`;
- `waf-doctor` auto-detects source-install versus distro-package binary/unit paths;
- package fixture reproducibility/content/security verifier passes.

Truth boundary:

- production `.deb` from real Go 1.25 / Coraza v3.7.0 binaries is BLOCKED on this host because the local Go toolchain is older than 1.25;
- Debian 12 and Ubuntu 22.04/24.04 clean-host installs remain NOT_RUN;
- fixture package evidence must never be reported as production package/runtime qualification.

Next planned distribution slice: **Slice B — RHEL-family RPM Packaging**.


## 2026-09-16 — Enterprise Linux Distribution Packaging Slice B checkpoint

Current authoritative implementation now includes both formal enterprise Linux package source paths: Debian/Ubuntu `.deb` (Slice A) and RHEL-family `.rpm` (Slice B). Slice B status is **IMPLEMENTED_TESTING_DEFERRED**.

RPM implementation is under `packaging/rpm/` and includes the real spec, provenance/checksum-bound deterministic builder, release-host wrapper, package verifier, source-contract validator, source tests, real-toolchain fixture test, CRS guidance, and SELinux boundary documentation. Configuration uses `%config(noreplace)`; `/etc/waf/waf-proxy.env` is created once and remains unowned/preserved; persistent state remains `/var/lib/waf-proxy`; fresh install stays inactive until explicit CRS provisioning; active services only are `try-restart`ed on upgrade. Scriptlets never fetch CRS/packages and never weaken/generate SELinux policy.

Executed source-level RPM validation PASSed. This host is Debian 13 and lacks `rpmbuild`, `rpm`, and `rpm2cpio`, so actual fixture-RPM/reproducibility/extracted-RPM checks are BLOCKED and must not be reported as PASS. Canonical Go 1.25 binary RPM is also BLOCKED here. RHEL/Rocky/Alma/Oracle clean-host and SELinux enforcing tests remain NOT_RUN.

Next roadmap slice: **Enterprise Linux Distribution Packaging Slice C — Package Upgrade / Rollback Qualification**. Do not redesign either package path; validate their lifecycle semantics on suitable package-manager environments.

Slice B delivery-gate checkpoint: baseline-relative reconstruction PASSed for 238 files with byte/mode identity; source Artifact Integrity PASSed; fixed-epoch source release reproducibility PASSed; clean-extracted RPM source validation PASSed; all 12 artifact mutation classes were rejected across segmented execution. Detached signing remains NOT_CONFIGURED. These gates do not change the actual-RPM BLOCKED or clean-host NOT_RUN status.


## 2026-09-16 — Enterprise Linux Distribution Slice C checkpoint

Slice C Package Upgrade / Rollback Qualification is now
`IMPLEMENTED_TESTING_DEFERRED`. Do not redesign the DEB/RPM package paths. The
canonical lifecycle implementation is under `packaging/qualification/` and must
remain offline-safe, destructive only behind explicit dedicated-host opt-in, and
secret-free in evidence. Local source/unit/DEB-fixture/preflight gates PASS.
Real DEB upgrade/rollback remains NOT_RUN; RPM fixture execution is BLOCKED on
this Debian host because RPM tools are absent; real RPM lifecycle remains
NOT_RUN. The next planned distribution work is Slice D clean-host qualification,
while production Go 1.25 package lifecycle can close Slice C when suitable hosts
are available. Baseline-relative Slice B→Slice C patch reconstruction has been executed and PASSes with 247 files byte/mode identical.

Slice C delivery-gate checkpoint: baseline-relative reconstruction PASSed for 247 files with byte/mode identity; complete-source Artifact Integrity PASSed (241 source / 247 packaged); fixed-epoch source-release reproducibility PASSed; all 12 artifact mutation classes were rejected across segmented execution; clean-extracted lifecycle source tests and DEB fixture/preflight PASSed; detached signing remains NOT_CONFIGURED. These delivery gates do not promote the real package-manager lifecycle gates.

## Current Work — Enterprise Linux Distribution Packaging Slice D

Slice D — Clean-host Distribution Qualification is implemented in source and
remains `IMPLEMENTED_TESTING_DEFERRED`.

Canonical implementation is under `packaging/cleanhost/` with checked-in
`qualification/clean-host/` NOT_RUN evidence. Preserve these invariants:

1. only exact Debian 12, Ubuntu 22.04/24.04, RHEL 9, Rocky 9, AlmaLinux 9 and
   Oracle Linux 9 dedicated clean hosts may earn PASS;
2. RHEL-family acceptance requires SELinux `Enforcing`;
3. the runner never fetches packages/dependencies/CRS from the network;
4. `/etc/waf` and `/var/lib/waf-proxy` must be absent before execution;
5. fresh package install must not auto-start before explicit local CRS setup;
6. PASS includes doctor, systemd start, health, break-glass admin API auth,
   actual reverse-proxy traffic, N+1 upgrade preservation and package removal;
7. admin token bytes are compared only in memory and never written or hashed in
   evidence;
8. container/source/preflight results remain NOT_RUN and cannot promote the
   distro matrix.

Current host source tests PASS, but it is Debian 13 and is correctly BLOCKED for
all exact Slice D targets. All seven real distro rows remain NOT_RUN. Do not
claim clean-host distribution closure until dedicated target-host reports exist.

Slice D local regression note: inherited DEB package regression, RPM source gate
and Slice C lifecycle source gate pass. The repository Go shipped-script test is
BLOCKED under `GOTOOLCHAIN=local` because this packaging host has Go 1.23.2 and
the module requires Go >=1.25.0. The supplied complete-source ZIP has no `.git`
metadata, so Git synchronization/commit/push was not available in this workspace.

Slice D source-delivery gates after source freeze: patch reconstruction PASS
(261 files byte/mode identical), Artifact Integrity PASS (255 source / 261
packaged), fixed-epoch reproducible source release PASS, clean-extracted Slice D
source/unit gate PASS, and all 12 negative artifact mutations rejected across
segmented execution. Detached producer signature remains NOT_CONFIGURED. These
do not change the seven real clean-host platform rows from NOT_RUN.

## Current Work — project-local DEB/RPM packaging utility

`./waf-package` and `tools/waf_package_builder.py` are now the intended operator/
developer entry point for producing this project's binary packages after source
changes. Supported commands are `doctor`, `deb`, `rpm`, and `all`. The utility
is deliberately repository-specific and delegates to `build.sh`,
`packaging/deb/build-deb.sh`, and `packaging/rpm/build-rpm.sh`; do not replace
those canonical paths with a parallel implementation.

Important invariants: Go/Coraza pins come from `go.mod`; `GOTOOLCHAIN=local` is
forced; no toolchain/dependency/CRS auto-install/download is allowed; one build
produces provenance-bound binaries; `all` packages those same bytes in both
formats; requested versions use a shared safe spelling; package results are
recorded in `PACKAGE_BUILD_REPORT.json` with SHA-256. Current source tests are
9/9 PASS plus source contract PASS. Real package execution remains BLOCKED on
the current host by Go 1.23.2 (<1.25.0), and RPM also lacks its native package
build tooling. Next action on an appropriate build host: run `./waf-package
doctor --format deb`, then `./waf-package deb --version <release>`; separately
run RPM doctor/build on an RPM-capable host, then execute Slice C/D gates.

Complete-source packaging for the `waf-package` utility also passed source artifact integrity, fixed-epoch reproducibility, clean-extract source regression, and 12/12 negative mutation rejection. These results cover source delivery only; real WAF `.deb`/`.rpm` production builds still require a qualified Go 1.25+ release host (and RPM tools for RPM).

## 2026-09-17 — Root build integrity repair

GitHub `main@1d52d65a73a802e32f02994e51f0a07beb240177` was independently audited
and found non-buildable. Treat any older `SOURCE_BASELINE_GATE_RESULT.md` PASS
that relied only on archive completeness as superseded for buildability claims.

Repair state:
- `go.mod` / `go.sum` restored to the known-good Coraza v3.7.0 dependency graph
  from the previously green CI branch (`go.mod` blob 7654301..., `go.sum` blob
  7b2ad1b...);
- `pki.go` restored to the CRL URL refresh companion implementation expected by
  `pki_url.go` (`crlStore` mutex/config/issuers/roots/status fields plus runtime
  builder integration);
- debug evidence v2 uses `DebugEvidenceStore.List(tenant, limit)` and current
  `DebugBundle` fields (`Tenant`, `TransactionID`, `CapturedAt`);
- `tlsVersionName` is defined and regression-tested;
- CI contains a mandatory `GOTOOLCHAIN=local go mod tidy -diff` gate before
  `go build ./...`.

Local static/source checks pass, but exact Go 1.25 build/test execution is
BLOCKED on this host. Do not promote this repair to TESTED, and do not restore a
Source Baseline PASS, until CI executes the exact repaired bytes successfully.

Process requirement: changes to `main` must go through a PR with required CI.
Branch protection is REQUIRED but was NOT_APPLIED by this session because the
connected GitHub App returned HTTP 403 for branch-ref mutation/protection-like
operations. Do not bypass this by writing repair commits directly to `main`.

Artifact-integrity follow-up for the root build repair: candidate complete-source packaging passed the project verifier (265 source / 271 packaged), fixed-epoch reproducibility, clean-extract static smoke, byte/mode reconstruction, and all 12 negative artifact mutations. These results must not be used to promote source buildability: Go 1.25 tidy/build/test execution is still BLOCKED locally. Final artifact packaging is rerun after recording this evidence so delivery evidence refers to final bytes.

## 2026-09-17 — Documentation consolidation checkpoint

All Markdown files were synchronized after the root-build-integrity audit. Use
`DOCUMENTATION_INDEX.md` as the entry point. `HANDOVER_PROMPT.md` and
`HANDOVER_STATUS.md` now contain only the current continuation truth rather than
a stack of historical next-slice notes. `INSTALL.md` includes the full package
operations lifecycle and explicitly records that PostgreSQL is not currently a
runtime dependency. No implementation status was promoted: root buildability
remains BLOCKED pending exact Go 1.25 CI.

## 2026-09-17 — OpenAI Responses/Structured Outputs hardening

Status: `IMPLEMENTED_TESTING_DEFERRED`. Native OpenAI now uses `/responses` with strict JSON Schema output, `store=false`, secret references, and deterministic provider mocks. `chat_completions` remains for compatible/self-hosted endpoints and legacy inline-key configs preserve their historical wire contract. Isolated `openaiapi`/`secretref` unit+race+vet PASS; root Go 1.25 integration remains BLOCKED. Next mandatory action remains Go 1.25 PR/CI validation before package production.

### OpenAI hardening verification checkpoint

Executed on this source state: isolated `openaiapi` and `secretref` unit/race/vet PASS; OpenAI source/config/UI contract 16/16 PASS; inherited DEB, RPM-source, lifecycle-source, and clean-host-source gates PASS; source hygiene clean after test cleanup. Baseline-relative reconstruction from `waf-proxy-root-build-integrity-repair-docs-sync-2026-09-17.zip` PASS with 273/273 source files byte-for-byte and Unix-mode identical. Repository-root Go 1.25 build/test remains BLOCKED on the local Go 1.23.2 host.

### 2026-09-18 — OpenAI hardening final delivery freeze

Final delivery evidence after source freeze:
- complete-source artifact verifier: **PASS — 273 source / 279 packaged files**;
- fixed-epoch reproducibility: **PASS — byte-identical release ZIPs**;
- clean-extract OpenAI source/config/UI contract: **16/16 PASS**;
- clean-extract isolated `openaiapi` + `secretref` unit, race, and vet: **PASS**;
- baseline-relative source patch reconstruction: **PASS — 273/273 source files byte + Unix-mode identical**;
- final artifact negative mutation rejection: **12/12 PASS** across segmented execution;
- root `GOTOOLCHAIN=local go test ./...`: **BLOCKED** before compilation because host Go 1.23.2 is below required Go 1.25.0.

Delivery-integrity PASS does not promote Source Buildability. Exact Go 1.25 root tidy/build/vet/test/race/real-Coraza and live provider acceptance remain required.


## Future API Security Direction

The next major product direction is API-aware WAF evolution. See `API_SECURITY_ROADMAP.md`. Existing discovery/profile capabilities are foundation only.


## API-1 Hardening
Status: IMPLEMENTED_TESTING_DEFERRED
- Operation fingerprinting
- Normalization confidence metadata
- ULID/date normalization hardening
- Regression coverage added


## API Security Slice Continuity — 2026-09-23

Current API Security baseline:
- API-1 Operation Discovery + Normalization: `IMPLEMENTED_TESTING_DEFERRED` — durable inventory/controls/UI; 4/4 isolated tests PASS.
- API-2 Typed Schema Learning: `IMPLEMENTED_TESTING_DEFERRED` — live typed collector/lifecycle/privacy/persistence; 4/4 core isolated tests + 1/1 lifecycle-handler test PASS.
- OpenAI SchemaCandidate review: advisory-only Structured Outputs path; source/isolated OpenAI gates PASS.
- API-3 OpenAPI Contract Management: `IMPLEMENTED_TESTING_DEFERRED` — durable import/export, params/enums/security, expanded live drift; source gate 42 checks + isolated harness PASS.
- API-5 and API-6.1 through API-6.3: `IMPLEMENTED_TESTING_DEFERRED`; API-6.4: `IMPLEMENTED_TESTING_DEFERRED`; API-7 and API-8: `PLANNED`.

Executed integration evidence: API-1 → API-2 → API-3 live-learning/drift test PASS. Admin inline JavaScript syntax PASS. Modified Go files are gofmt-clean.

Qualification boundary: repository-root Go 1.25 tidy/build/vet/test/race/real-Coraza is `BLOCKED/NOT_RUN` because this host only has Go 1.23.2 and cannot retrieve the required toolchain/modules. Do not describe API-1/2/3 as `TESTED`, `RELEASED`, or root-buildable from isolated evidence.

Important boundaries:
- Coraza remains enforcement authority; API-1/API-2/API-3 do not block traffic.
- OpenAI remains advisory and cannot mutate enforcement state.
- Do not persist Authorization/Cookie/JWT secret values or complete request bodies.
- Do not fetch remote OpenAPI `$ref` targets.
- API-4 is now implemented with testing deferred; exact-source Go 1.25 qualification is still required before TESTED/RELEASED.

Exact next step: run the delivered bytes on the pinned Go 1.25 CI/release host, then repeat artifact integrity. See `API_SECURITY_CLOSURE_RESULT.md`.


Local executable evidence for this checkpoint is captured in `TESTING_RESULTS.md`; root Go 1.25 blocker evidence is captured in `SOURCE_BASELINE_GATE_RESULT.md`.

## API-4 handoff — 2026-09-23

API-4 Positive Schema Enforcement is implemented and `IMPLEMENTED_TESTING_DEFERRED`. Runtime source is `positive_schema_api4.go`; deterministic tests are `positive_schema_api4_test.go`; source gate is `tools/tests/test-api4-source.py`. Promotion requires a current operator-approved API-2 candidate. Profiles are immutable/versioned; deployments start in LEARN, must pass through DETECT before ENFORCE, support rollback and bounded scoped exceptions, and store bounded violation evidence in `api-positive-schema.json`. Runtime reads use an atomic snapshot and do not call OpenAI. The shared body capture now records truncation by reading at most `passiveBodyLimit+1` and replaying every consumed byte; ENFORCE rejects unvalidated oversized bodies. Do not mark TESTED or start claims of release readiness until exact Go 1.25 `go mod tidy -diff`, build, vet, full tests, race, and real-Coraza gates pass on the delivery bytes. Next planned product slice is API-6 Sequence Analytics.

## API-5 JWT + Identity-aware API Security checkpoint — 2026-09-23

API-5 is implemented in source and remains `IMPLEMENTED_TESTING_DEFERRED`. `identity_api5.go` provides trusted issuer/JWKS verification, verified-claims-only request context, operation-scoped role/scope/tenant/client authorization, DETECT→ENFORCE lifecycle, semantic-change rollback to DETECT, bounded privacy-preserving evidence, and durable `api-identity.json` state. JWKS fetch is HTTPS-only and bounded; unknown-`kid` rotation refresh is serialized and rate-bounded. Raw bearer tokens, unverified claims and JWKS key cache are not persisted. OpenAI has no identity-enforcement authority.

Local evidence: 17 API-5 deterministic test functions PASS; targeted race PASS; `tools/tests/test-api5-source.py` 72 checks PASS; root source-shape PASS for 97 Go files; admin inline JavaScript syntax PASS; and the external-dependency stub harness passes whole-repository `go build ./...`, `go vet ./...`, and all test-binary compilation. These are not substitutes for the canonical Go 1.25 real-dependency gate, which remains `BLOCKED_ENVIRONMENT/NOT_RUN` (`go mod tidy -diff`, build, vet, full tests, full race, real-Coraza). API-6 remains `PLANNED` and has not started.

## API-6.1 Sequence Foundation checkpoint — 2026-09-23

Implemented only API-6.1 in `sequence_api61.go`/`sequence_api61_test.go`. Runtime collection is a bounded non-blocking telemetry queue placed downstream of API-5 identity verification and upstream of schema/Coraza. Nodes are API-1 normalized operation IDs. Identity correlation is possible only from `VerifiedAPIIdentity`; anonymous correlation uses HMAC over short-lived request-derived session material and persists only digests. Sessions, transitions and recent histories have fixed caps and TTLs; persistence is `api-sequence.json` plus a separate 0600 `api-sequence.key`; admin detail is Reviewer-gated and audited; atomic immutable `SequenceModel` snapshots serve persistence/admin readers. No anomaly detection, mode transition, exception, reset/relearn, or enforcement was added.

`tools/tests/test-api61-source.py`: PASS, 33 checks. Five targeted Go test functions cover deterministic identity/normalization, transition state, TTL/cardinality, restart privacy and concurrent non-blocking snapshots. They and the race gate are `BLOCKED_ENVIRONMENT/NOT_RUN` here because no Go toolchain is installed. Next product slice is API-6.2 Workflow Learning; API-7 remains planned.


## API-6.2 Workflow Learning checkpoint — 2026-09-24

API-6.2 is implemented in source and remains `IMPLEMENTED_TESTING_DEFERRED`. The slice adds LEARN-only workflow cohorts, transition observation/session counts, frequency-derived confidence, `LEARNING`/`MATURE`/`STALE` cold-start semantics, workflow depth plus entry/terminal operation summaries, idle and absolute session lifetime, bounded workflow/session/transition state, API-5 verified-identity-only cohort input, durable v2 state with API-6.1 v1 migration, and Reviewer-gated/audited read-only workflow visibility. It adds no anomaly verdict, BLOCK/DENY/403 behavior, BOLA verdict, reset/relearn control, or sequence ENFORCE mode.

Executed source evidence: API-6.1 gate **33/33 PASS**, API-6.2 gate **56/56 PASS**, and Admin inline JavaScript syntax **PASS**. Seven targeted API-6.2 Go tests are present, including concurrency/race-relevant, restart/privacy and bounded-resource/adversarial cases. Canonical Go 1.25 targeted/race execution is `BLOCKED_ENVIRONMENT/NOT_RUN`: the local toolchain is Go 1.23.2 while `go.mod` requires Go 1.25.0, and external toolchain/dependency retrieval is unavailable. No stub or downgraded-toolchain evidence is used as qualification. API-6.3 is `IMPLEMENTED_TESTING_DEFERRED`; API-6.4 is `IMPLEMENTED_TESTING_DEFERRED`; API-7.1 through API-7.4 are `IMPLEMENTED_TESTING_DEFERRED`; API-8 remains `PLANNED`.

## API-6.3 Sequence Anomaly Detection checkpoint — 2026-09-24

API-6.3 is the current canonical implementation baseline and remains `IMPLEMENTED_TESTING_DEFERRED`. The detector is DETECT-only and emits bounded `SequenceViolation` evidence for unknown transition, prerequisite skipped, unexpected entry point, reversal, abnormal repetition and workflow divergence. It requires mature workflow and relevant sample/session/age/confidence evidence, honors bounded persisted `SequenceException` selectors, and never owns BLOCK/DENY/403 or sequence ENFORCE authority. Sequence state is v3 and accepts v1/v2 state on restore. `GET /api/security/sequence/violations` is Reviewer-gated and audited. API-6.4 operator control workflows are now implemented; see the API-6.4 checkpoint below.

Current executed source/static evidence: API-1/2 69 PASS, API-3 47 PASS, API-4 46 PASS, API-5 72 PASS, API-6.1 33 PASS, API-6.2 56 PASS, API-6.3 45 PASS, OpenAI contract 16/16 PASS, package-builder 9/9 PASS, package source PASS, root shape 103 PASS, changed-file gofmt PASS, Admin JS syntax PASS. Canonical Go 1.25 targeted/race: `BLOCKED_ENVIRONMENT/NOT_RUN` on local Go 1.23.2 with no external retrieval. Next slice: API-6.4 Sequence Operations + Hardening.


## API-6.4 Sequence Operations + Hardening checkpoint — 2026-09-24

API-6.4 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. It adds explicit site-scoped `LEARN` / `DETECT` controls with mature-only DETECT promotion, bounded persisted recent-session summaries, bounded exception create/delete operations, site-scoped reset/relearn that returns to LEARN, Reviewer RBAC + audit, and the full Sequence Operations console. Durable sequence state advances to v4 with v1/v2/v3 restore compatibility. Sequence analytics still cannot BLOCK, DENY, return request-path 403, or enter ENFORCE. API-6 is implementation-complete through API-6.4; API-7.1 Object Locator Discovery is the next `PLANNED` slice.

Executed static/source evidence: API gates **69/47/46/72/33/56/45/58 PASS** from API-1/2 through API-6.4; OpenAI source contract **16/16 PASS** plus isolated OpenAI/secretref tests PASS; package-builder **9/9 PASS**; root Go shape **105 files PASS**; Admin JavaScript syntax, shell syntax and changed-file gofmt PASS. Canonical Go 1.25 targeted/race remains `BLOCKED_ENVIRONMENT/NOT_RUN` because the host has Go 1.23.2 and external toolchain/module retrieval is unavailable.


## API-7.1 Object Locator Discovery checkpoint — 2026-09-24

API-7.1 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. It adds bounded `ObjectLocator` discovery from API-1 normalized path parameters, API-2 typed path/query/body evidence, matched API-3 OpenAPI contracts, and explicit Reviewer-gated operator INCLUDE/SUPPRESS configuration. Locators are keyed by normalized API-1 operation ID plus location/field; raw object values are transient only and durable value evidence is bounded HMAC-SHA256 fingerprint data protected by a separate mode-0600 key. Client-supplied ownership/tenant headers are not authority, and GraphQL `variables.*` discovery is deferred to API-8.

API-7.1 has no identity/object relationship verdict, tenant-boundary verdict, `BOLA_CANDIDATE`, ownership verdict, BLOCK/DENY/403 path, or ENFORCE authority. OpenAI is absent from the locator authority path. State is bounded by global/per-operation/fingerprint/override caps, 30-day learned TTL, versioned restart validation, existing non-blocking observation-plane ingestion, immediate mutation persistence, RBAC and audit.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83 PASS** from API-1/2 through API-7.1; API-7.1 has **9 targeted test functions present**; OpenAI source contract **16/16 PASS** plus isolated tests PASS; package-source PASS; root Go shape **107 files PASS**; package-builder **9/9 PASS**; changed-file gofmt, Admin JavaScript and primary shell syntax PASS. Canonical Go 1.25 targeted/race remains `BLOCKED_ENVIRONMENT/NOT_RUN` because the host has Go 1.23.2 and external toolchain retrieval is unavailable. Next slice: **API-7.2 Identity/Object Relationship**. API-7.3/API-7.4 is `IMPLEMENTED_TESTING_DEFERRED`; API-8 remains `PLANNED`.

## API-7.2 Identity/Object Relationship checkpoint — 2026-09-24

API-7.2 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. It correlates only API-5 cryptographically verified identity context with ACTIVE API-7.1 locator/object evidence. Identity, tenant and client dimensions are persisted only as API-7.2 HMAC-SHA256 pseudonyms protected by a dedicated mode-0600 key; object values remain API-7.1 keyed fingerprints. Raw JWTs, subjects, tenant/client claim values, cookies, caller-supplied owner/tenant headers and raw object values are not durable relationship state.

The relationship plane is bounded and asynchronous: non-blocking queue capacity 2,048; 8,192 total relationships; 512 relationships per verified identity pseudonym; 256 verified identities per keyed object/locator pair; 30-day TTL; bounded path/query/body capture; restart revalidation; autosave/final flush; shutdown drain; Reviewer-gated read-only evidence/status endpoints with audit. SUPPRESSed API-7.1 locators cannot create relationships. GraphQL `variables.*` remains deferred to API-8.

API-7.2 is evidence only. `OBSERVED`/`REPEATED` means recurrence, not ownership. There is no `BOLA_CANDIDATE`, ownership verdict, tenant-boundary verdict, BLOCK/DENY/403 or ENFORCE authority, and OpenAI is absent from this authority path.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83/100 PASS** from API-1/2 through API-7.2; API-7.2 has **9 targeted Go test functions present**; OpenAI source contract **16/16 PASS** plus isolated OpenAI/secretref tests PASS; package-source PASS; root Go source shape **109 files PASS**; package-builder **9/9 PASS**; changed Go files `gofmt` PASS. Canonical Go 1.25 targeted/race remains `BLOCKED_ENVIRONMENT/NOT_RUN` because this host has Go 1.23.2 and external toolchain retrieval is unavailable. Next slice: **API-7.4 BOLA Policy/Evidence/Console**; API-7.4 is `IMPLEMENTED_TESTING_DEFERRED`; API-8 remains `PLANNED`.



## API-7.3 BOLA Detection checkpoint — 2026-09-24

API-7.3 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. The detector consumes only API-5 cryptographically verified identity pseudonyms, ACTIVE/non-suppressed API-7.1 keyed object locators/fingerprints, and API-7.2 relationship history. It emits bounded evidence-only candidates for `IDENTITY_OBJECT_DIVERGENCE`, `TENANT_OBJECT_DIVERGENCE`, and `OBJECT_ENUMERATION`. A novel object alone is not a candidate; cross-identity and cross-tenant evidence requires repeated historical baseline, while enumeration requires 20 recent distinct keyed objects under the same locator within 10 minutes.

Candidate state persists only pseudonymous/keyed evidence, is capped at 4,096 total / 256 per identity / 128 per object-locator, expires after seven days, is versioned/revalidated on restart, and is saved by the API security autosave/final-flush path. Detection runs inside the API-7.2 background processing plane before current relationship merge. Reviewer-only candidate/status reads are audited. There is no ownership verdict, tenant-boundary verdict, policy mutation API, `BLOCK`, `DENY`, request-path `403`, or `ENFORCE` authority; OpenAI is absent from the detector authority path.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83/100/110 PASS** from API-1/2 through API-7.3; API-7.3 has **9 targeted Go test functions present**; OpenAI source contract **16/16 PASS** plus isolated tests PASS; package-source PASS; root Go source shape **111 files PASS**; package-builder **9/9 PASS**; changed Go `gofmt`, Admin embedded JavaScript and primary shell/Python syntax PASS. Canonical Go 1.25 `go mod tidy -diff`, root build, API-7.3 targeted tests and API-7.3 race tests remain `BLOCKED_ENVIRONMENT/NOT_RUN` because this host has Go 1.23.2 and external toolchain retrieval is unavailable. Next slice: **API-7.4 BOLA Policy/Evidence/Console**; API-8 remains `PLANNED`.

## API-7.4 BOLA Policy/Evidence/Console checkpoint — 2026-09-24

API-7.4 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. It completes the API-7 BOLA implementation track with a bounded evidence-handling policy plane and operator workflow. Policies are scoped only by normalized API-1 operation ID, optional API-7.1 locator ID, optional API-7.3 candidate type, and minimum confidence. The only actions are `REVIEW` and `SUPPRESS`; suppression never deletes the underlying API-7.3 candidate. Operator evidence workflow is `OPEN` / `ACKNOWLEDGED` / `DISMISSED` / `RESOLVED` with enumerated reason codes only. New detector evidence after dismissal/resolution reopens the evidence automatically.

API-7.4 is deliberately absent from the API-7.2 relationship processor and API-7.3 detector authority path. It accepts no raw identity/object/tenant/client selectors, arbitrary headers, cookies, Authorization data, or free-text review notes. There is no ownership verdict, request-path `BLOCK`, `DENY`, `403`, or `ENFORCE` authority, and OpenAI is absent from the API-7.4 authority path. State is versioned/revalidated on restart, mutex-protected, TTL/cardinality bounded (1,024 policies; 4,096 reviews; 30-day default policy/review TTL; 180-day policy maximum), immediately persisted on mutation, and included in API security autosave/final flush. Reviewer-only policy/evidence APIs are audited and the embedded console exposes effective policy, evidence workflow, suppression, and reopen status.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83/100/110/156 PASS** from API-1/2 through API-7.4; **10 API-7.4 targeted Go test functions are present**; OpenAI source contract **16/16 PASS** plus isolated package tests PASS; package-source PASS; root Go source shape **113 files PASS**; package-builder **9/9 PASS**; `gofmt`, Admin embedded JavaScript and primary shell/Python syntax PASS. Canonical Go 1.25 `go mod tidy -diff`, root build/test, API-7.4 targeted tests and API-7.4 race tests remain `BLOCKED_ENVIRONMENT/NOT_RUN` because this host has Go 1.23.2 and external toolchain retrieval is unavailable. API-8 GraphQL Security is the next `PLANNED` slice.

## API-8 GraphQL Security checkpoint — 2026-09-24

API-8 is implemented across parsing/normalization, SDL contract binding, deterministic complexity/field/mutation/subscription/introspection policy, Apollo persisted-query support, verified GraphQL-variable integration into API-7 evidence, and Reviewer-gated `LEARN` / `DETECT` / `ENFORCE` policy plus Admin Console. Any policy edit or bound contract content change returns the policy to `LEARN`; a referenced schema contract cannot be deleted. Only explicit deterministic GraphQL policy in `ENFORCE` may reject traffic. API-6/API-7 learned or inferred evidence and OpenAI cannot promote, mutate or bypass GraphQL enforcement.

Durable GraphQL state excludes raw query documents, literals, raw variable values, Authorization/Cookie/JWT data and unverified claims. SDL is reduced to a digest plus normalized topology; GraphQL object values use API-7 keyed fingerprints and API-5 verified identity only. State and parser resources are bounded, versioned, restart-revalidated, autosaved/final-flushed, and strict mutation APIs are Reviewer-gated and audited.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83/100/110/156/259 PASS** through API-8; **22 targeted API-8 Go test functions are present**; OpenAI source contract **16/16 PASS** plus isolated tests PASS; package-source PASS; package-builder **9/9 PASS**; root Go source shape **115 files PASS**; changed API-8 Go files `gofmt`, Admin embedded JavaScript, relevant shell and Python syntax PASS. Canonical Go 1.25 `go mod tidy -diff`, root build/vet/test, API-8 targeted tests and race tests remain `BLOCKED_ENVIRONMENT/NOT_RUN` because the host has Go 1.23.2 and external toolchain retrieval is unavailable. API-8 remains `IMPLEMENTED_TESTING_DEFERRED`, not `TESTED` or `RELEASED`.

The documented API-security implementation roadmap is now complete through API-8. No API-9 is defined. Next gate: canonical Go 1.25 qualification of the exact source/artifact, followed by release promotion only if all required gates pass.


## Current Work — API-8 Post-Audit Hardening (2026-09-24)

Status: IMPLEMENTED_TESTING_DEFERRED. This pass is not API-9. It fixes CIDR Enabled/expiry runtime truth, implements the previously dead L7 TLS-handshake limit on built-in Go TLS with external-frontend fail-closed validation, closes Console exposure gaps for System/Doctor/Debug/HSM/Vector/CIDR/L7/OpenAPI/Positive-Schema, replaces Sequence/BOLA/GraphQL opaque-ID text inputs with inventory-backed selectors, removes the obsolete non-shipping `web/` frontend, and removes six model-only Security Operations/debug-lifecycle files that were not wired to runtime/API/Console. Phase 5 Slice D is corrected to PLANNED / NOT IMPLEMENTED. Use `static/admin.html` as the only Console source. Canonical Go 1.25 qualification is still required.

## 2026-09-25 — Production Correctness & Control-Plane Hardening

Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.

## 2026-09-28 handoff delta — duplication consolidation

Continue from the Code Duplication Review and Consolidation source, not from the
2026-09-25 parent tree. Do not restore deleted compatibility/place-holder files
merely because older ledger sections mention them. Current debug authority is
`debug_bundle.go` + `debug_sanitize.go` + `support_api.go`; current API-2 schema
learning is `schema_api2.go`; current CRL URL hardening is `pki.go` +
`pki_url.go`; current VectorScan runtime/qualification authorities are
`internal/vectoraccel` and `cmd/wafqualify`; current coverage reporting is the
ruleset-aware `internal/coverage/crs` path. Both offline CRS coverage and live
VectorScan SecLang parsing use `internal/capability`.

Older `web/` migration guidance in dated historical sections is superseded: the
`web/` tree does not exist and must not be treated as a current frontend.

## 2026-09-30 — Build and Admin Startup Blocker Fix

The current source fixes two release-blocking defects found during external Go 1.25 review of the 2026-09-28 consolidation baseline. `admin.go` now imports the standard-library `errors` package required by the existing `errors.New(...)` validation paths. Positive-Schema exception state changes now use the unambiguous route `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`; the shipping Console uses the same path. The prior `POST /api/security/schema/enforcement/exceptions/{exception_id}` registration is removed because it conflicts with `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+ `http.ServeMux` specificity rules and can panic during Admin handler construction.

`admin_route_registration_test.go` now makes full Admin handler registration a regression test, and `tools/tests/test-build-startup-blocker-fix-source.py` is wired into `build.sh` and CI. The source gate also registers the complete 139-route Admin/update inventory with the Go standard library in a standalone temporary program; this catches wildcard-pattern ambiguity even when the repository module graph cannot be compiled locally. Current source evidence: blocker-fix gate **13/13 PASS**, route inventory **139/139 accepted by `ServeMux`**, API/source gates through API-8/post-audit/production-hardening/duplication remain PASS, OpenAI source contract and isolated tests PASS, package-source PASS, package-builder **9/9 PASS**, root Go source shape **96 files PASS**, Admin embedded JavaScript PASS, and changed Go files are `gofmt` clean.

This does **not** promote the source to `TESTED` or `RELEASED`. On the current execution host, canonical `GOTOOLCHAIN=local` Go 1.25 `go mod tidy -diff`, root build, vet, targeted Admin/API64 tests, and race remain `BLOCKED_ENVIRONMENT / NOT_RUN` because only Go 1.23.2 is installed and external Go toolchain/module retrieval is unavailable. Packaging/source-manifest integrity is not a substitute for those root compilation gates. No API-security authority changes are introduced: API-6/API-7 remain non-enforcing, API-8 retains explicit deterministic GraphQL ENFORCE authority, and OpenAI remains advisory only.

Continuation point: use this blocker-fixed tree as the current source, not the 2026-09-28 consolidation ZIP. Do not restore the ambiguous exception-toggle route. The next qualification step is to run the exact final bytes under pinned Go 1.25 with `go mod tidy -diff`, `CGO_ENABLED=0 go build ./...`, `go vet ./...`, the targeted Admin route/API64 tests, full tests, and race tests before any buildable/PASS or release claim.
## 2026-09-30 — OWI-1.0 R3 Dashboard Connector

Status remains **IMPLEMENTED_TESTING_DEFERRED**. This is a product integration slice, **not API-9**, and it does not change API-1 through API-8 enforcement authority. The WAF now exposes a separate read-only management reader at `/api/integrations/dashboard/v1` for required ASSET, DETECTION, POLICY and HEALTH lanes using a dedicated digest-only opaque bearer identity. Optional EVENT/ACTION_STATUS remain disabled and `actions=[]`.

R3 controls implemented in source include per-tenant+principal rate/concurrency plus a global ceiling, strict GET-only handling including HEAD rejection, an 8s default request deadline propagated into source sync and context-aware store-lock acquisition, hard snapshot/export/response bounds, durable cursor/snapshot/revision state, non-blocking WAF security export with explicit GAP semantics, fail-closed durability after write errors, opaque-token expiry/rotation/revoke, owner-only non-symlink token/cursor-secret files, TLS >=1.2 on the built-in reader listener, tenant/scope-bound cursors, and allow-listed sanitization. No Dashboard registry/validator/UI code is modified.

Supporting exact connector-source evidence on the available Go 1.23.2 host is **25 PASS / 2 intentional SKIP**, with the same set passing under `-race`; real subprocess SIGKILL→restart/GAP, store contention deadline/retry, injected write failure/recovery, client cancellation, oversize failure, scope reduction, and accelerated retention expiry are covered. The R3 source gate is **38/38 PASS**. The regenerated product return validates **61 files, 18 positive responses, 15 negative cases and 11 transport cases**. Integrated repository Go 1.25 build/vet/test/race remains `BLOCKED_ENVIRONMENT / NOT_RUN`; G2 Dashboard offline acceptance and G3 live TLS/ACL/intended-runtime qualification remain `NOT_RUN`.

<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
