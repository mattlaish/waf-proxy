# WAF Proxy Patch Handover

This file is the working patch ledger for AI handover and manual version
control. Update it whenever source, schema, UI, tests, build behavior, or known
limitations change. Git synchronization remains the repository owner's task.

## Patch Version

- Patch date: 2026-09-03 (Asia/Taipei; request hot-path optimization slice)
- Current merged baseline before this local slice: `95ab964` — `lf modifed`
- Status: hot-path optimizations are implemented locally; changes are not
  committed or deployed
- Remote synchronization was attempted before implementation, but this execution
  environment could not resolve `github.com`; the work therefore uses the
  `origin/main` snapshot bundled in the uploaded repository at `95ab964`.
- Production UI: `static/admin.html`
- Experimental/non-shipping UI: `web/`

## Git Workflow

- GitHub `origin/main` is the source of truth.
- AI may perform normal `pull`, `add`, `commit`, and `push` operations after
  reviewing status/diff, tests, secrets, generated files, and unrelated changes.
- Never force-push, rewrite history, delete branches, or modify remotes without
  explicit approval.
- A task-specific user instruction may narrow these permissions and takes
  precedence for that task.

## Current Patch Scope

### Request hot-path optimization (2026-09-03)

- Replaced the access and WAF-match append/reslice stores with fixed-capacity
  circular buffers. Steady-state writes no longer reallocate/copy the retained
  ring contents under the ring mutex.
- Replaced request-side AI and syslog configuration locks with immutable
  `atomic.Pointer` snapshots. Syslog stream-specific `atomic.Bool` gates return
  before a config snapshot when a stream is disabled, and sites with `ai_mode`
  off now receive the underlying handler directly.
- Added root `passive_discovery_enabled` (default `true` for compatibility). When
  false, passive request-body field discovery adds no body-reading/parsing
  middleware; path/status learning and the explicit crawler continue to work.
- Added one bounded request-body prefix capture shared by passive discovery and
  AI body analysis. A qualifying request is read/restored at most once before
  Coraza/backend processing; AI retains the shared `[]byte` rather than making a
  second read and full `[]byte` -> `string` copy.
- Replaced top-level JSON discovery's `json.RawMessage` value materialization
  with a bounded scanner that records field names while skipping values.
- Removed the unused AI `statusRecorder`. The remaining access-log recorder now
  exposes `Unwrap()` so `http.ResponseController` can reach optional interfaces
  such as `http.Hijacker` through the wrapper.
- Access logging computes the client address once for the ring/syslog record and
  stores `time.Time`; the existing `HH:MM:SS` JSON representation is formatted
  only when the admin API serializes the record. A request-context client-IP
  cache was deliberately not added because the trusted-proxy resolver already
  normalizes XFF once at the outer edge and adding another `WithContext` on every
  request would introduce its own allocation.

### Global trusted-proxy client IP

- Added root `trusted_proxy_cidrs` with strict unique IPv4/IPv6 CIDR validation,
  a 64-network configuration cap, and backward migration from the former
  AI-only field.
- Added one resolver ahead of access logging, AI, Coraza, load balancing, and
  proxying. XFF is honored only from a trusted immediate peer and is walked
  right-to-left to the first untrusted hop.
- Invalid, oversized, or excessive-hop XFF fails back to the TCP peer. The
  default empty trust list preserves existing edge behavior.
- Coraza, `ip_hash`, access/syslog/learner records, AI, and backend XFF/X-Real-IP
  now use the same authoritative address.
- Updated the shipping UI, sample config, README, migration logic, and focused
  tests. No rate limiting or other L7 abuse control was added in this slice.

### Per-field custom policies

- Added `FieldPolicy.allow_pattern` so fields on the same form can use different
  validation expressions.
- Patterns are limited to 512 bytes, compiled before Apply, and cannot contain
  literal double quotes or line breaks in the policy expression.
- This protects SecLang syntax; submitted field values may still contain quotes
  when the selected pattern permits them.
- Existing profile, required, length, and field-only CRS exclusion behavior is
  preserved and composes with the custom pattern.
- The shipping page-policy editor exposes one custom allow-regex input per field.

### Windows update portability and test coverage

- Added Windows `sigupdate.ReExec` support without changing Unix `syscall.Exec`.
- Added focused config migration/full/draft validation tests.
- Added weighted round-robin, least-connections, IP-hash, health filtering, and
  all-members-down fail-open tests.
- Added in-process exact/wildcard/catch-all Host routing and draining health
  tests.
- Marked `web/` explicitly experimental and non-shipping.

### PKI Slice 1: backend CA trust

- Added pool-scoped `BackendTLSConfig`.
- HTTPS pools use the operating-system CA store by default.
- Custom Root/Intermediate CA PEM bundles may be appended, or may form an
  isolated trust store when `use_system_ca` is explicitly false.
- Added server-name override and TLS 1.2 minimum.
- Proxy traffic and HTTPS health monitors share the pool trust policy.
- No `tls_skip_verify` option exists.
- CA files are limited to 8 MiB and must contain valid CA certificates with
  certificate-signing capability.
- Trust-loading failure occurs before atomic runtime swap, preserving the old
  runtime.

### PKI Slice 2: static CRL enforcement

- Supports strict PEM `X509 CRL` bundles and DER CRLs.
- Limits: 8 MiB per file and at most 32 files/lists per pool.
- Apply rejects malformed, future, expired, duplicate-path, or invalid custom
  issuer-signature CRLs.
- CRLs are stored in an immutable snapshot behind `atomic.Pointer`; TLS
  handshakes perform no file or network I/O.
- Leaf and intermediate serials are checked.
- Revoked certificates always fail closed.
- Soft mode permits missing issuer coverage; hard mode requires valid coverage
  for every non-root certificate.
- System-store issuer signatures are checked during the verified handshake
  because Go does not expose the system `CertPool` certificate objects.

## Configuration Schema Added

```json
{
  "backend_tls": {
    "use_system_ca": true,
    "ca_files": ["/etc/waf/pki/company-root.pem"],
    "crl_files": ["/etc/waf/pki/company-root.crl"],
    "server_name": "app.internal.example",
    "revocation_mode": "hard",
    "crl_urls": [],
    "refresh_sec": 0
  }
}
```

`crl_urls` and non-zero `refresh_sec` are reserved and currently rejected by
full Apply. They must not be documented operationally as active until Slice 3.

## Files in the Merged Functional Patch Series

- `AI_HANDOFF.md`
- `AGENTS.md`
- `README.md`
- `config.sample.json`
- `main.go`
- `pool.go`
- `pki.go`
- `pki_test.go`
- `pki_crl_test.go`
- `fieldpolicy_test.go`
- `config_pool_test.go`
- `runtime_smoke_test.go`
- `internal/sigupdate/sigupdate_windows.go`
- `static/admin.html`
- `web/PORTING.md`
- `web/package.json`
- `patch.md`

## Local Files That Should Not Be Included Automatically

- `Codex Image Aug 18, 2026, 02_59_30 PM.png` — unrelated generated image.
- `git-save-push.ps1` — local helper that uses `git add .`; it can accidentally
  stage unrelated files.

The owner or acting AI must still review the working tree before staging. The
normal Git permissions above do not authorize including unrelated local files.

## Validation Evidence

Latest trusted-proxy client-IP checks (Go 1.26.0 Windows/amd64 unless noted):

```text
go test -count=1 ./...                         PASS
go vet ./...                                   PASS
static/admin.html inline JavaScript syntax     PASS
config.sample.json parsing                     PASS
CGO_ENABLED=0 Linux static build               PASS
go test -race ./...                            NOT RUN (no Windows C compiler)
```

Latest Linux/amd64 static binary:

```text
Size:   20,030,558 bytes
SHA256: 7932329870C7331436E6B463ED708E9C8F4CE5154AF0AF2804F5C88197894039
```

No dependency was added. Focused tests use documentation-only IP ranges and an
ephemeral local HTTP backend; no private key, API key, token, production address,
or generated binary is intended for version control.

## Known Limitations / Not Yet Implemented

- CRL URL downloader and SSRF protections.
- Scheduled CRL refresh and refresh deduplication.
- Last-known-good CRL snapshot retention across refresh failures.
- `/api/pki/status` and manual refresh API.
- PKI RBAC/audit coverage.
- Restricted CA import/delete API.
- TLS key-provider abstraction.
- PKCS#11 HSM provider and SoftHSM integration tests.
- Deployed Linux VM smoke tests for the new backend CA/CRL paths.
- No rate limiting, connection capping, or L7 abuse control anywhere in the tree.
- Trusted-proxy client-IP resolution is now implemented locally. Deployment
  behind the real upstream proxy and Linux race verification remain pending.
- No manual IP allow/deny list; only the AI may add a block.
- No custom block page and no request correlation ID, so a user-reported block
  cannot be traced to the rule that caused it.
- Security state is not persisted: AI blocklist, learner aggregates, notification
  queue, sessions, and the audit ring reset on every restart.
- The learner cannot suggest field-scoped rule exclusions. The matched variable is
  discarded in the WAF callback (`main.go:1038`), so it never reaches `ruleAgg`
  (`learn.go:31`) or `SuggestExcl` (`learn.go:142`). Enforcement is unaffected —
  `FieldPolicy` and `PagePolicy.ExcludeTargets` already emit per-field removals.
- `govulncheck` has never been run against the dependency tree; the cloud
  container blocks `vuln.go.dev`. Checksum integrity is verified, CVEs are not.

## Next Slice

Validate trusted-proxy resolution on the Linux VM behind the actual upstream
proxy, including Coraza, access log, syslog, learner, AI advisory, `ip_hash`, and
backend header evidence. Then implement the L7 abuse-control half of approved
roadmap item 1 against the resolved address. PKI Slice 3 remains open after that;
do not begin PKCS#11 dependency work before the key-provider abstraction and
CGO/static-build tradeoff are reviewed explicitly.

## Reviewed Backlog (not scheduled)

Recorded so the decisions are not relitigated. Full reasoning, evidence, and
file references are in `AI_HANDOFF.md`; this list is the ledger's index of them.
The owner explicitly selected roadmap item 1 as the current main-problem work;
the remaining approved and PKI items stay recorded below.

Approved by the owner on 2026-08-21, in priority order:

1. Trusted-proxy client-IP resolution (implemented locally on 2026-08-28), then
   L7 abuse control (rate limiting,
   per-IP connection caps, TLS handshake cap). Scoped explicitly as
   application-layer abuse, **not** volumetric DDoS, which belongs upstream.
   The client-IP half is a prerequisite: limiting on the wrong address would
   throttle the CDN rather than the attacker.
2. Manual IP allow/deny lists, CIDR-aware, allow winning over deny, optional
   TTL, reusing the AI blocklist enforcement path. GeoIP/ASN is a later
   increment.
3. Custom block page plus request correlation ID, stamped onto the match
   record, access log, and syslog event.
4. Persistence for security state, following the `sitemap_persist.go` pattern.

Declined on 2026-08-21:

- Response-side DLP (outbound PHI/PII pattern matching). Do not implement
  without a new decision.

Candidate, undecided, added 2026-08-24:

- Field-scoped learner suggestions: capture the matched variable so the learner
  can propose `ruleRemoveTargetById=RULE;ARGS:field` instead of whole-rule,
  path-scoped exclusions. Sequencing note: the learner's false-positive-versus-
  attack heuristic counts distinct clients, so item 1 should land first or the
  suggestions will key on unreliable counts. Must stay human-reviewed — excluding
  a rule on a field is correct for a hashed password and wrong for any field
  reaching a query.

## Required Update Procedure

For every future patch slice:

1. Update the patch date and scope.
2. List schema/API/UI changes.
3. List intended files and explicitly excluded local files.
4. Record exact test/build results and binary hash when built.
5. Record incomplete work, security decisions, and failed approaches.
6. Set the precise next implementation slice.
7. Update `AI_HANDOFF.md` consistently.
8. End stage-completion feedback with current Taiwan time, formatted exactly as
   `YYYY-MM-DD HH:mm:ss UTC+8 (Taiwan)`.

## Documentation Synchronization on 2026-08-28

- Synchronized this ledger with merged `main` baseline `3186015`.
- Reconciled the Git workflow with current `AGENTS.md` and `AI_HANDOFF.md`.
- Preserved PKI Slice 3 as the next implementation slice; the reviewed security
  backlog does not displace it.
- No functional source, schema, API, UI, test, dependency, or build change was
  made during this synchronization.
## Request Hot-Path Validation on 2026-09-03

- The real `go test -count=1 ./...` could not start because this isolated
  environment cannot resolve/download `github.com/corazawaf/coraza/v3@v3.3.2`
  from `proxy.golang.org`. This is an environment/dependency limitation, not a
  passing product-test claim.
- To catch compile and local-regression errors without network access, a
  temporary copy outside the repository replaced only Coraza with a minimal API
  stub. Against the final source state, `go test -run '^$' ./...`, the focused
  hot-path/discovery tests, the complete repository test suite, `go vet ./...`,
  and `go test -race -count=1 ./...` all passed in that stub harness. These
  results do **not** validate real Coraza behavior.
- Focused coverage includes circular ordering/count, access-log JSON time
  compatibility, ResponseWriter unwrapping/Hijacker reachability, AI-off direct
  handler behavior, disabled syslog access gating, legacy config defaulting,
  disabled passive discovery, shared passive/AI body prefix, large JSON value
  skipping, backend-response discovery flow, and byte-body AI prompting.
- Offline microbenchmarks on the harness host (AMD EPYC 9V74) measured the fixed
  access-ring add at `9.825 ns/op`, `0 B/op`, `0 allocs/op`; the ~60 KiB JSON
  field-name scanner measured `35,846 ns/op`, `1,302 B/op`, `14 allocs/op`. These
  are component benchmarks only and are not an end-to-end latency claim.
- `config.sample.json` parses successfully and the shipping inline admin script
  passes `node --check`. No production binary was built because the real Coraza
  dependency was unavailable.

### Exact next step

Run the unmodified repository with the real Coraza module available and execute
`go test -race ./...`, `go vet ./...`, the normal build/release checks, and
end-to-end benchmarks for representative 1 KiB/16 KiB/64 KiB request bodies with
passive discovery and AI combinations. Verify WebSocket/HTTP Upgrade through the
remaining recorder as part of that real integration run.


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

## P0-B Load-Balancer Hot Path on 2026-09-04

**Scope:** eliminate backend-selection and selected-member handoff allocations;
no schema, API, UI, dependency, or configuration change.

### Source changes

- `pool.go`: removed `healthyMembers()` allocation; all four LB methods now scan
  members directly with zero steady-state allocations while preserving healthy
  filtering and all-down fail-open semantics.
- `pool.go`: direct FNV-1a string hashing replaces the stdlib hash object on the
  IP-hash request path, with compatibility coverage.
- `pool.go` / `main.go`: backend member selection moved into `lbTransport`; the
  former per-request `context.WithValue` handoff and `memberCtxKey` are removed.
  The selected member pointer remains local to the transport for active-request
  accounting through response-body Close.
- `main.go`: `Rewrite` now only handles forwarding headers/PreserveHost setup;
  `lbTransport` applies selected backend scheme/host before the base transport.
- `pool_hotpath_test.go`: added allocation, selection, target, path/query,
  PreserveHost, active-counter, FNV-compatibility, and benchmark coverage.

### Verification

- Real Coraza-dependent test/build: **NOT_RUN**; the isolated environment cannot
  download Coraza v3.3.2 from `proxy.golang.org`.
- Temporary external Coraza API-stub copy: full `go test -count=1 ./...`,
  `go vet ./...`, and `go test -race -count=1 ./...` passed.
- Old 8-member selector benchmark (AMD EPYC 9V74): 42–48 ns/op depending on
  method, `64 B/op`, `1 alloc/op`.
- P0-B selector benchmark: about 12–28 ns/op depending on method,
  `0 B/op`, `0 allocs/op`.
- Removed selected-member context handoff: old component measured
  `32.01 ns/op`, `48 B/op`, `1 alloc/op`.

### Incomplete / release gate

- Real Coraza integration and live backend HTTP/1.1/HTTP/2 concurrency tests are
  still required before production deployment.
- This source archive has no `.git` directory, so branch/remote synchronization
  cannot be performed inside this artifact; reconcile with GitHub/main on a
  networked repository before commit/push.

### Exact next slice

**P0-C: bounded asynchronous observation plane for hostObserver, sitemap,
learner, and signal-store updates, with drop counters and no request blocking.**

### Pre-existing issue observed (not changed by P0-B)

- `install.sh` fails `bash -n` with `line 132: syntax error: unexpected end of file`.
  The untouched P0-A archive has the same failure. This slice leaves it unchanged
  to keep P0-B review-scoped; it remains a required installer/release fix.

## P0-C Observation Plane on 2026-09-04

**Scope:** move non-enforcement host/sitemap/learner/signal telemetry off the
synchronous request path behind a bounded drop-on-full queue; reduce repeated
live-field merge churn; preserve P0-A/P0-B behavior.

### Source changes

- Added `observations.go`: bounded 8,192-event async queue, non-blocking enqueue,
  drop/processed counters, aggregated drop warning, drain-on-stop lifecycle, and
  JSON snapshot stats.
- `main.go`: added `server.observations` and `observeHost` / `observeRequest` /
  `observeMatch` helpers; runtime response recording and Coraza learner match
  recording now enqueue. Observation plane starts before listeners and drains
  after listeners stop. Sitemap persistence loads before listeners and final
  save occurs after observation drain.
- `listeners.go`: host observation now uses the async server observation helper.
- `profiles.go`: added `mergeObservedFields` for in-place low-churn repeated live
  passive-field merging; crawler/general merge behavior remains unchanged.
- `admin.go`: `/api/metrics` adds an `observations` object with queue
  depth/capacity, dropped, processed, and accepting fields.
- Added `observations_test.go`: apply/drain, drop-on-full, zero-allocation enqueue,
  and component/parallel benchmark coverage.
- `README.md`, `AI_HANDOFF.md`, `MANIFEST.md`, and this engineering ledger updated.

### Verification

- Real Coraza-dependent test/build: **NOT_RUN** because the isolated environment
  cannot resolve/download Coraza v3.3.2.
- External temporary Coraza API stub: full `go test -count=1 ./...`, `go vet ./...`,
  and `go test -race -count=1 ./...` pass.
- Original pre-P0-C synchronous request observation measured ~2.56–2.63 µs/op,
  1,664 B/op, 24 allocs/op on AMD EPYC 9V74.
- After low-churn live-field merging, direct store work is ~443–445 ns/op,
  48 B/op, 3 allocs/op; actual async enqueue is ~35–36 ns/op, 0 B/op, 0 allocs/op.
- Host observation moves from ~213–229 ns/op, 48 B/op, 2 allocs/op to ~36–38 ns/op,
  0 B/op, 0 allocs/op on the data-plane side.
- Parallel direct-store work ~0.52–0.66 µs/op vs queue enqueue ~0.13–0.15 µs/op;
  enqueue remains zero-allocation.

### Next slice

**P0-D: bounded/rate-limited asynchronous Coraza match logging/aggregation.**

### Known unrelated issue

Shell validation remains an inherited release issue rather than a P0-C regression:
`install.sh` and `uninstall.sh` have EOF parse errors, and `setup-interfaces.sh`,
`waf-doctor.sh`, and `upgrade.sh` have CRLF-related bash parse failures. The P0-B
input archive has the same failures; this performance slice intentionally does not
normalize or repair those scripts.


## P0-D Bounded Match-Log Aggregation on 2026-09-04

**Scope:** remove synchronous per-match process logging from the Coraza callback and replace
it with bounded, rate-limited background aggregation; preserve enforcement and event sinks.

### Source changes

- Added `matchlog.go`: 8,192-event drop-on-full queue, five-second aggregation window,
  1,024 active-group cap, site/rule/client grouping, drain-on-stop lifecycle, separate
  queue/group drop counters, bounded representative log fields, and JSON snapshot stats.
- `main.go`: removed direct `s.log.Warn("rule match", ...)` from the matched-rule callback;
  callback now non-blocking-enqueues to `server.matchLogs` after unchanged ring, learner,
  syslog, and AI handling. Match-log plane starts before listeners and drains after listener
  shutdown.
- `admin.go`: `/api/metrics` adds additive `match_logging` stats.
- Added `matchlog_test.go` with aggregation, flush/drain, saturation, cardinality-cap,
  no-synchronous-log, truncation, drop-report, allocation, and slog comparison coverage.
- Updated `README.md`, `AI_HANDOFF.md`, `MANIFEST.md`, and this engineering ledger.

### Verification

- Real Coraza-dependent test/build: **NOT_RUN**; isolated DNS cannot reach
  `proxy.golang.org` for Coraza v3.3.2.
- External temporary Coraza API stub: full `go test -count=1 ./...`, `go vet ./...`, and
  `go test -race -count=1 ./...` pass.
- AMD EPYC 9V74: successful match-log enqueue ~39.7–42.7 ns/op, 0 B/op, 0 allocs/op;
  previous per-match JSON slog path ~1.445–1.514 µs/op, 232 B/op, 8 allocs/op.

### Next slice

P1 should focus on response-body inspection policy and backend Transport/connection-pool
tuning, after or alongside repair of the inherited release-script syntax failures.

## P1 Response Inspection + Backend Transport Tuning on 2026-09-04

**Scope:** policy-level response-body inspection controls plus pool-scoped Go
`http.Transport`/connection-pool tuning, on top of P0-D.

### Source/API/UI changes

- `main.go`: added `BackendTransportConfig`, tuned defaults/effective-value
  helpers and validation; added policy response-body access/limit fields and
  SecLang overrides; backend transports now use configurable connection-pool,
  dial, idle, keepalive, and max-connection values; runtime cleanup closes idle
  transports.
- `pool.go`: pool runtime now owns one shared backend `*http.Transport`; pool
  status returns effective transport values; health-monitor transport closes
  idle connections on exit.
- `static/admin.html`: Policies UI exposes response inspection/limit; Pools UI
  exposes backend connection-pool tuning; config cloning/sync/new-object paths
  preserve the new objects.
- `config.sample.json`: documents explicit response inspection and transport
  sizing settings.
- Added `p1_tuning_test.go` for directive/validation/default/transport/UI/runtime
  regression coverage.
- Updated `README.md`, `AI_HANDOFF.md`, `MANIFEST.md`, and this ledger.

### Compatibility/defaults

- Existing policy values with missing/empty response-body fields inherit the
  SecLang file exactly as before.
- Existing pools with no `transport` block receive effective defaults of
  2048 max idle total, 256 max idle per host, unlimited max conns per host,
  90s idle timeout, 5s dial timeout, and 30s keepalive.
- No route/method removal, auth/RBAC change, DB/schema migration, or theme change.

### Verification

- Real Coraza suite: **NOT_RUN** because `proxy.golang.org` remains unreachable.
- External Coraza stub tree: full `go test -count=1 ./...`, `go vet ./...`, and
  `go test -race -count=1 ./...` pass.
- Production inline JavaScript passes `node --check`; `config.sample.json` parses.

### Next

Run real-Coraza plus representative response-body and backend-connection load
benchmarks before production release; choose the next dataplane optimization
from those measurements.

## Benchmark Harness v1.0.0 on 2026-09-04

**Scope:** add a repeatable benchmark/profiling tool to select the next
performance architecture from measurements rather than low production traffic.

### Source/tool changes

- Added `cmd/wafbench/`:
  - `http` full-proxy HTTP/HTTPS benchmark with clean + hostile corpus;
  - `coraza` direct Coraza/CRS benchmark with response inspection controls and
    CPU/heap pprof output;
  - `l4` legal TCP connect/close or TLS-handshake pressure baseline;
  - `backend` deterministic fixed-response backend;
  - `compare` JSON baseline comparison and XDP-vs-regex sizing heuristic.
- Linux metrics collect target process CPU/RSS, host busy/softirq, and optional
  interface RX/TX Mbps/PPS from `/proc`; generator CPU is tracked separately.
- Added low-overhead logarithmic latency histogram and JSON/CSV/table result
  output. JSON records host CPU model/kernel/Go/runtime details.
- Standard corpus includes clean GET, JSON 1/16/64/256 KiB, SQLi 64 KiB, XSS
  64 KiB and traversal. Clean-only scenarios determine the baseline Coraza
  share; hostile cases are retained as worst-case visibility.
- Added `benchmark/README.md` and `benchmark/build.sh`.
- Updated README, AI_HANDOFF, MANIFEST, and this ledger. No production API,
  policy, listener, WAF enforcement, P0/P1, or theme behavior changed.

### Verification

- Real Coraza v3.3.2 build/test: **NOT_RUN** because isolated DNS still cannot
  reach `proxy.golang.org`.
- External Coraza API stub: command-package test/vet/race/build plus full-repository `go test -count=1 ./...`, `go vet ./...`, and `go test -race -count=1 ./...` pass.
- End-to-end stub smoke: deterministic backend + HTTP mode, Coraza mode, L4
  mode, JSON export, compare mode, CPU profile, and heap profile all execute.
- Stub-derived RPS/CPU numbers are validation of the harness only and are not
  Coraza performance claims.

### Next

Use this harness with real Coraza/CRS on the target CPU. Only after that baseline
should the next data-plane POC be selected between XDP and grouped regex
acceleration.

## P2 Non-XDP/VectorScan Hardening on 2026-09-04

**Scope:** close the remaining low-risk performance/release debt that does not
require an XDP or regex-accelerator decision.

### Source changes

- Normalized every top-level release/build `.sh` plus `benchmark/build.sh` to
  LF and restored executable mode. Added `release_scripts_test.go` enforcing
  no carriage returns, executable mode, and `bash -n` syntax.
- `ai.go`: changed pending AI request state from eagerly constructed
  `*analysisJob` values to lightweight live `*http.Request` references keyed by
  `pendingKey{client, uri}`. Full request capture/redaction is lazy on WAF match
  or pre-sampled clean traffic. Clean sampling is decided before capture.
- `ai.go`: queue drops now increment counters and emit at most one warning per
  five seconds instead of synchronously logging every dropped analysis job.
  `/api/metrics` exposes `ai_queue` depth/capacity/logical limit/enqueued/dropped.
- `ai.go`: workers reuse a ticker rather than allocating `time.After` timers in
  the polling loop.
- `ai.go`: AI block-mode reads now use an immutable atomic blocklist snapshot;
  rare add/unblock/expiry updates are copy-on-write under the writer mutex, so
  ordinary request-path lookups do not contend on that mutex.
- `ai.go`: `statusRecorder` suppresses duplicate underlying `WriteHeader` calls
  and records implicit 200 on `Write`.
- Added focused AI lazy-capture/sample/queue-drop tests, status-recorder tests,
  and AI eager-vs-lazy component benchmarks.
- Updated README, INSTALL, MANIFEST and AI_HANDOFF truth boundaries.

### Verification

- All release/build scripts pass `bash -n` and the new repository script gate.
- External Coraza API stub: full `go test -count=1 ./...`, `go vet ./...`, and
  `go test -race -count=1 ./...` pass.
- AI unsampled/no-match component benchmark on AMD EPYC 9V74: new lazy path
  ~169-175 ns/op, 0 B/op, 0 allocs/op; prior eager-capture model ~1.49-1.55 us/op,
  848 B/op, 17 allocs/op.
- Active-block lookup under 8/32-way parallelism: atomic snapshot ~22/~21
  ns/op versus legacy mutex ~93/~115 ns/op; all zero-allocation.
- Real Coraza v3.3.2 build/test remains NOT_RUN because the isolated environment
  cannot resolve/download from `proxy.golang.org`.

### Rejected after measurement

A 16-shard access-ring prototype was not merged. It was slower than the current
fixed ring both serially and at 8/32-way parallelism on the harness host due to
its global sequence atomic plus shard lock. Keep the current ring until a real
profile proves otherwise.

## TLS Acceleration C1-C3 / modern NGINX — 2026-09-04

- Added `internal/tlsfront` capability detection, safe NGINX/OpenSSL renderer,
  kTLS and Intel QAT provider resolution, modern/legacy HTTP/2 syntax selection,
  and preflight.
- Added `cmd/waf-tlsfront` supervisor with atomic generated config, NGINX
  preflight/reload/restart, QAT OpenSSL environment, live status and graceful
  shutdown.
- Added private Unix listener remapping in waf-proxy, trusted frontend metadata
  restoration/sanitization, original HTTPS scheme propagation, Apply preflight,
  live-control publication, API/status and Setup UI.
- Added `waf-tls-frontend.service`; build/install/upgrade/uninstall/doctor now
  know both binaries/services. Go TLS remains the default.
- NGINX >= 1.25.1 renders `listen ... ssl;` + `http2 on;`; older NGINX renders
  the legacy `listen ... ssl http2;` form.
- Real NGINX 1.26.3/OpenSSL 3.5.5 `nginx -t` and HTTPS/HTTP2→Unix smoke passed
  with no HTTP/2 deprecation warning. kTLS/QAT auto fallback was exercised;
  real QAT hardware offload remains NOT_RUN.
- External Coraza API stub full test/vet/race pass; real Coraza v3.3.2 remains
  NOT_RUN because `proxy.golang.org` DNS is unavailable in this environment.


## VectorScan Learning Accelerator / Coraza v3.7 transaction truth — 2026-09-04

**Scope:** implement the selected regex-acceleration direction with a fail-safe Learning Period while keeping Coraza authoritative; upgrade the Coraza dependency to v3.7.0 so transaction `MatchedRules()` can be the learning truth source.

### Source changes

- Added `internal/vectoraccel/` with configuration validation/defaults, conservative SecLang rule/include parser, semantic grouping/fingerprints, grouped scanner abstraction, native libhs CGo adapter and portable unavailable adapter. Native compile uses `hs_compile_multi`; scan/close is synchronized and scratch objects are reused/freed safely.
- Added per-group `CORAZA_ONLY/LEARNING/VALIDATED/ACCELERATED/FAILSAFE` state, persisted samples/Coraza-match/FN counts, verification sampling, immediate false-negative/scan-error fail-safe, reset-to-learning, and rule/native/Coraza semantic fingerprint invalidation.
- `main.go` builds the manager once per runtime, constructs per-site plans, injects private phase-1 rule-removal control directives only for compiled eligible groups, runs VectorScan before Coraza, strips internal control headers at ingress/egress, closes native resources with runtime retirement, and publishes vector config defaults.
- Added `coraza_observer.go`: wraps Coraza WAF transaction creation including `experimental.WAFWithOptions`; binds the request-context VectorScan observation to the exact Coraza transaction; after `ProcessLogging()` reads final `MatchedRules()`, de-duplicates rule IDs and reports them exactly once to the learning plan. ErrorCallback remains for existing logs/AI/syslog but is no longer a Learning truth source.
- Added `coraza_real_gate_test.go` (`realcoraza` tag) proving on a real Coraza build that DetectionOnly+nolog remains silent to ErrorCallback while the matched rule is present in `tx.MatchedRules()`.
- `admin.go` adds vector status/reset endpoints and metrics; `config.sample.json` adds the learning config block.
- Upgraded `go.mod` to Go 1.25.0 / Coraza v3.7.0 and corresponding dependency revisions. `build.sh` rejects Go <1.25.0 and supports `WAF_VECTORSCAN=auto|required|off`, native pkg-config detection, vet/test/race and real-Coraza release gate.
- Updated README/INSTALL/AI_HANDOFF/MANIFEST and this ledger.

### Safety / compatibility boundaries

- Initial acceleration eligibility is deliberately narrow: standalone positive `@rx`, phase 1/2, single `REQUEST_URI`, `REQUEST_FILENAME`, `REQUEST_METHOD`, `REQUEST_PROTOCOL`, or fixed-name `REQUEST_HEADERS:name`, explicit `t:none`, optional `t:lowercase`. Chains, negated operators, ARGS/body, multiple variables, aggregate headers, inherited/unsupported transforms remain Coraza-only.
- `mode=off` is the default. `auto` safely runs Coraza-only without libhs; `required` fails instead of silently omitting the requested native engine.
- VectorScan never blocks/allows directly. A no-hit can skip a group only in `ACCELERATED`; verification samples continue full Coraza evaluation. Any observed false negative or native scan failure moves the group to `FAILSAFE`.

### Verification

- External Coraza v3.7 API stub: full repository test/vet pass and bounded race groups pass. This validates project control-flow/API integration only.
- A local ABI-compatible libhs test library compiles/links the real CGo adapter and passes vectorscan-tag test/vet/race. It specifically caught and led to removal of an unsafe uintptr-to-pointer callback bridge. This is an ABI/lifecycle gate, not real VectorScan matching evidence.
- Real Go 1.25 + Coraza v3.7.0 execution, `realcoraza` DetectionOnly+nolog truth test, and real libvectorscan matching are **NOT_RUN** in the isolated packaging environment and remain explicit release-host gates.

### Release packaging

Final release packaging adds no further product behavior. The final ZIP is produced from this exact source tree after doc-ledger update and diff hygiene, then re-extracted and compared byte-for-byte/file-mode-for-file-mode against the source tree. Patch is generated relative to the TLS Acceleration / modern-NGINX release baseline. SHA-256 values are reported with the delivery artifacts.

## New-chat handover documentation — 2026-09-07

**Scope:** documentation and handover packaging only; no production WAF behavior changed.

- Added `DEVELOPMENT_ROADMAP.md` with the selected VectorScan direction, mandatory real release-host qualification, conservative Learning rollout, coverage-expansion sequence, release-engineering work, older approved security/PKI backlog, and explicit XDP/AF_XDP/DPDK/DPU/GPU deferrals.
- Added `TESTING_RESULTS.md` that separates real NGINX evidence, Coraza API-stub regression, native libhs ABI-only checks, historical component benchmarks, and explicit real Coraza/real VectorScan NOT_RUN gates.
- Added `HANDOVER_STATUS.md` as a concise checkpoint and `HANDOVER_PROMPT.md` as the copy/paste bootstrap prompt for a new development chat.
- Updated `AI_HANDOFF.md` with this checkpoint.
- Synchronized the project completion timestamp convention in `AGENTS.md` to `UTM+8: YYYY-MM-DD HH:MM:SS`.

This handover update intentionally does not change Go source, runtime configuration semantics, API behavior, UI behavior, native build behavior, or dependency versions.

## Phase 0 release-host qualification automation — 2026-09-07

**Scope:** improve the mandatory real-release-host qualification path because the current packaging environment has Go 1.23.2 and no real `libhs`; do not change WAF dataplane architecture or claim unavailable gates.

### Source changes

- Added executable `qualify-release-host.sh` with `--preflight` and `--core` modes. Preflight validates Linux, the installed local Go version without auto-toolchain download, Go/Coraza pins, CGO compiler, `pkg-config libhs`, and real VectorScan provenance. Exit 3 is reserved for **BLOCKED** prerequisites; exit 1 is an executed gate **FAIL**.
- `--core` invokes `WAF_VECTORSCAN=required ./build.sh`, explicitly reruns the native VectorScan semantic gate and real-Coraza truth gate for readable release transcripts, and fails if `go.mod`/`go.sum` drift during qualification.
- Added `internal/vectoraccel/scanner_native_gate_test.go` (`vectorscan && cgo`) that directly exercises native multi-expression compile plus two-hit/clean scans and validates exact match IDs. The test is release evidence only with verified real VectorScan provenance; ABI shims remain ABI-only evidence.
- Hardened `build.sh` Go-version detection to use `GOTOOLCHAIN=local go version`, preventing an underspecified host from silently triggering an implicit Go 1.25 toolchain download before the minimum-version check.
- Updated README, INSTALL, MANIFEST, DEVELOPMENT_ROADMAP, TESTING_RESULTS, AI_HANDOFF, HANDOVER_STATUS and HANDOVER_PROMPT to keep the qualification truth boundary synchronized.

### Verification actually executed

- `bash -n` for all root `*.sh` release scripts and `benchmark/build.sh`: PASS.
- `./qualify-release-host.sh --preflight`: expected BLOCKED, exit 3. Evidence: Linux x86_64; local Go 1.23.2; `pkg-config` 1.8.1; no `libhs`; GCC available; NGINX 1.26.3/OpenSSL 3.5.5 present.
- Synthetic command-shim `--preflight` pass-branch test: PRECHECK_PASS; this validates runner control flow only and is not counted as real Go/VectorScan evidence.
- `GOTOOLCHAIN=local WAF_VECTORSCAN=required ./build.sh`: expected early rejection with `Go >= 1.25.0 ... found 1.23.2`; transcript contained no Go 1.25/proxy.golang.org auto-download attempt.
- Real Coraza v3.7.0, real `realcoraza`, real VectorScan `hs_compile_multi/hs_scan`, native real-libhs race, system install/upgrade/uninstall smoke, and CRS Learning Period: NOT_RUN on this host.

## 2026-09-09 — Artifact Packaging Integrity Gate

### Added

- `RELEASE_PROCESS.md`: mandatory extracted-artifact integrity policy, mini-SIEM packaging incident lesson, release pipeline and WAF-specific automation.
- `DEVELOPMENT.md`: cross-cutting release-artifact engineering rule.
- `TESTING.md`: post-packaging test requirements and evidence boundary.
- `build-release-artifact.sh`: clean source staging, generated `RELEASE_MANIFEST.txt`, deterministic ZIP creation, mandatory post-build artifact verification and SHA-256 output.
- `verify-release-artifact.sh`: archive CRC/path/symlink validation, required-file checks, critical script size/shebang/mode checks, extracted shell syntax validation, complete source/package SHA-256+mode equality and manifest verification.

### Updated

- `DEVELOPMENT_ROADMAP.md`: artifact integrity is a mandatory cross-cutting release gate, not an optional Phase 3 hardening idea.
- `TESTING_RESULTS.md`: records the implemented artifact-gate controls and strict evidence boundary.
- `AI_HANDOFF.md`: carries the release-safety rule and exact new automation into future sessions.
- `MANIFEST.md`: lists the new release-policy documents and packaging/verification scripts.

### Security / truth boundary

- Source-test PASS cannot be used as package-integrity evidence.
- Artifact-integrity PASS cannot be used as real Coraza/VectorScan/CRS/QAT/kTLS evidence.
- Release ZIPs must be independently re-extracted and verified before delivery.

### Validation evidence

- `bash -n` for all root release scripts and `benchmark/build.sh`: PASS.
- normal post-package extraction/integrity verification: PASS.
- 117 packaged source files matched the source tree by SHA-256 and Unix mode; generated manifest hashes matched extracted files.
- negative corruption test replacing `install.sh` with README content: rejected as expected.
- negative path-traversal ZIP test: rejected as expected.
- verifier fix: reapply stored ZIP Unix mode after Python extraction because `zipfile.extractall()` does not restore executable permissions itself.

- Patch-integrity negative finding: initial Git diff omitted five new untracked files; reconstruction failed as designed. Corrected generation includes intent-to-add paths before the binary diff, after which all 117 source files reconstructed byte-for-byte and mode-for-mode.

## 2026-09-10

Added debug_evidence.go and debug_evidence_test.go. Updated development, roadmap, testing and handoff records for Debug Evidence Capture and VectorScan qualification tracking.

## 2026-09-10 Debug evidence integration

Added:
- Coraza transaction evidence hook.
- VectorScan qualification comparison helper.
- Debug export masking foundation.


## 2026-09-11 Supportability Slice

Added wafctl debug export/doctor/support bundle foundation and Phase 1 differential qualification helper. Real Coraza/libvectorscan execution remains NOT_RUN.


## Stage 2 Supportability Implementation
- Added wafctl supportability command foundation.
- Added qualification phase1 differential runner foundation.
- Real Go 1.25/Coraza/libvectorscan qualification remains NOT_RUN.

## 2026-09-11 Phase 0 / Phase 1 qualification evidence update

Documentation/evidence-only update after executing the real release-host preflight against the current baseline. Added `PHASE0_PHASE1_QUALIFICATION_REPORT_2026-09-11.md` and synchronized roadmap, testing and handoff truth. The host is BLOCKED by Go 1.23.2 plus missing real libvectorscan/libhs provenance; Phase 0 core and Phase 1 production qualification remain NOT_RUN. No WAF dataplane source behavior changed.


## 2026-09-11 — Phase 2 conservative VectorScan coverage expansion

Baseline: `waf-proxy-phase0-phase1-qualification-attempt-2026-09-11.zip`.

Production source changes:

- `internal/vectoraccel/rules.go`: ordered transform pipeline, new `QUERY_STRING` / `SERVER_NAME` source classification, semantic adapter version bump.
- `internal/vectoraccel/transforms.go`: exact allow-listed Coraza v3.7.0 transformation semantics.
- `internal/vectoraccel/engine.go`: apply transform pipeline and reconstruct Host/Transfer-Encoding/new single-value sources from the same Go request fields used by Coraza's HTTP connector.
- `internal/vectoraccel/engine_test.go`, `transforms_test.go`: portable Phase 2 coverage tests.
- `internal/vectoraccel/phase2_realcoraza_test.go`: real Coraza v3.7.0 parity gate for the new coverage.

Safety boundaries unchanged: Coraza remains authoritative; false negatives/native scan errors force FAILSAFE; ARGS/body/chains/negation/multi-variable/dynamic-macro/unsupported-transform rules remain Coraza-only. Semantic fingerprint changes force re-Learning.

Executed evidence: isolated `internal/vectoraccel` unit/vet/race PASS on local Go 1.23.2. Full repo Go 1.25, realcoraza, real libvectorscan, and production CRS qualification remain NOT_RUN/BLOCKED and must not be inferred from the isolated checks.


Phase 2 release packaging additionally requires clean ZIP extraction/source comparison, manifest verification, patch reconstruction, secret-pattern hygiene, and exclusion of generated ELF binaries. Artifact integrity PASS does not change the real-host qualification truth boundary.

## 2026-09-11 Phase 2 VectorScan coverage increment 2

Baseline: `waf-proxy-vectorscan-phase2-coverage-2026-09-11.zip`.

Source changes:

- expanded exact single-value source classification/reconstruction with `REQUEST_URI_RAW`, `REQUEST_LINE`, `REQUEST_BASENAME`, `REMOTE_ADDR`, and `REMOTE_PORT`;
- added exact deterministic `base64Encode` and `hexEncode` transforms;
- bumped the VectorScan semantic adapter fingerprint version so affected persisted Learning/ACCELERATED state cannot survive the semantic change;
- expanded portable source/classifier/transform tests and the `realcoraza` parity gate;
- retained Coraza-only handling for ARGS/body/chains/negation/multi-variable/dynamic-macro/decode/hash/normalization semantics.

Executed validation: isolated `internal/vectoraccel` unit/focused/vet/race PASS on the local Go 1.23.2 host. Full repository Go 1.25 execution is BLOCKED; real Coraza/libvectorscan/production CRS qualification remains NOT_RUN.

## 2026-09-11 — Phase 3 release engineering

Added supply-chain/release tooling: SPDX/CycloneDX SBOM generation, version/provenance evidence, govulncheck status/fail-closed option, portable/native flavor separation, SOURCE_DATE_EPOCH reproducibility, optional external-key detached signing, and artifact-side evidence verification. Added a standalone Phase 3 tooling test. Updated roadmap/testing/release/handoff documentation. No dataplane, Coraza, VectorScan learning, or policy semantics were changed.

## 2026-09-11 — Phase 4 security/product implementation

Baseline: `waf-proxy-phase3-release-engineering-2026-09-11.zip`.

Implemented L7 request abuse controls, expiring CIDR allow/deny, WAF-owned request correlation IDs with exact Coraza transaction-ID injection, custom block responses, atomic persistent security state with hashed session tokens, and PKI Slice 3 HTTPS CRL URL refresh/LKG/cache/status/manual-refresh behavior. Added systemd/install state-directory wiring and focused Phase 4 source tests.

Executed evidence: isolated security, PKI and persistence `-race` suites PASS; JSON/shell/systemd-unit syntax checks PASS. Full repository Go 1.25 test/vet, real Coraza request-ID regression, external positive CRL/deployed Linux/HA/load/govulncheck remain BLOCKED/NOT_RUN and must not be inferred from isolated tests.

Phase 4 pre-final complete-source artifact verification: PASS, 143 source files / 151 packaged manifest-tracked files, portable flavor. Final ZIP is rebuilt after evidence/documentation freeze; patch reconstruction remains a mandatory final gate.

## 2026-09-11 — Main build/test blocker hotfix

Baseline: Phase 4 security/product source.

Changes: synchronized the Go module manifests for Coraza v3.7.0 and corrected Git executable modes for every release shell script covered by `TestReleaseScriptsAreLFAndBashSyntaxClean`. This is a release/repository metadata hotfix only; runtime behavior is unchanged. The mode-aware Git patch is intentionally generated from a simulated `main` baseline with the affected scripts at `100644`, so applying it records `100755` rather than relying on archive extraction semantics.

Hotfix pre-freeze artifact evidence: complete-source verifier PASS (`143 source / 151 packaged`, portable); repeated same-input ZIP build was byte-identical; mode-aware patch reconstruction PASS across 152 regular workspace files. Final delivery is rebuilt after evidence freeze.
