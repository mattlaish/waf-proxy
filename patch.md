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
