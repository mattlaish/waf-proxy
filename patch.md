# WAF Proxy Patch Handover

This file is the working patch ledger for AI handover and manual version
control. Update it whenever source, schema, UI, tests, build behavior, or known
limitations change. Git synchronization remains the repository owner's task.

## Patch Version

- Patch date: 2026-08-20 (Asia/Hong_Kong)
- Base visible before the current local patch series: `8d9c003` — Implement
  hybrid form discovery
- Status: local working changes; owner decides staging, commit, and push
- Production UI: `static/admin.html`
- Experimental/non-shipping UI: `web/`

## Current Patch Scope

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

## Files Intended for the Current Patch Series

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

The owner must review the working tree manually. AI must not execute Git
operations unless the owner explicitly changes that instruction.

## Validation Evidence

Latest PKI Slice 2 checks:

```text
go test -count=1 ./...                         PASS
go vet ./...                                   PASS
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go vet  PASS
Linux root test compilation                    PASS
static/admin.html inline JavaScript syntax     PASS
config.sample.json parsing                     PASS
CGO_ENABLED=0 Linux static build               PASS
```

Latest Linux/amd64 static binary:

```text
Size:   14,020,770 bytes
SHA256: E0AC425D40250AB0AED46BDBFE23BA4C180B366FEF3C322C3026B7D7AC988A03
```

All PKI tests dynamically generate keys, certificates, and CRLs. No private-key
fixture, HSM PIN, API key, token, or production secret is required or intended
for version control.

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
- Client IP is the TCP peer everywhere except `ai.go`: `clientIP` (`main.go:1388`)
  returns `RemoteAddr`, so `ip_hash`, the access log, syslog, and the learner's
  distinct-client signal are wrong behind a CDN, upstream load balancer, or SNAT.
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

PKI Slice 3:

1. SSRF-safe HTTPS CRL downloader with DNS/IP checks, redirect revalidation,
   response limit, timeout, and no environment proxy.
2. Atomic last-known-good refresh with concurrency deduplication and runtime
   cancellation.
3. Scheduled refresh using `refresh_sec` and jitter.
4. `GET /api/pki/status` for authenticated viewers.
5. `POST /api/pki/crl/refresh` for operators/admins with audit records.
6. Focused downloader, refresh, RBAC, audit, and race tests.

Do not begin PKCS#11 dependency work before the key-provider abstraction and
CGO/static-build tradeoff are reviewed explicitly.

## Reviewed Backlog (not scheduled)

Recorded so the decisions are not relitigated. Full reasoning, evidence, and
file references are in `AI_HANDOFF.md`; this list is the ledger's index of them.
None of these displace the Next Slice above, which remains PKI Slice 3.

Approved by the owner on 2026-08-21, in priority order:

1. Trusted-proxy client-IP resolution, then L7 abuse control (rate limiting,
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

