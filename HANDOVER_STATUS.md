# New-Chat Handover Status

> **Canonical checkpoint: 2026-09-17.** Read `DOCUMENTATION_INDEX.md` first.

## One-line state

The WAF feature/package source line is broad, but the current root-build-integrity
repair is `IMPLEMENTED_TESTING_DEFERRED`: artifact integrity is independently
validated, while exact Go 1.25 source buildability remains `BLOCKED` until CI
runs the repaired bytes successfully.

## Authoritative baseline

- GitHub baseline audited: `main@1d52d65a73a802e32f02994e51f0a07beb240177`.
- Known main failures before repair: `go.mod` drift plus root compile errors in
  PKI CRL-refresh/debug-evidence/TLS-version integration.
- Repair restores the Coraza v3.7.0 dependency graph, CRL store/runtime
  companion state, current DebugBundle/List APIs, and TLS version helper.
- CI repair adds `GOTOOLCHAIN=local go mod tidy -diff` before root build.
- Do not describe the source tree as buildable/PASS until the exact repaired
  commit passes the Go 1.25 gate.

## Current implementation states

- Phase 0 runtime qualification: `IMPLEMENTED_TESTING_DEFERRED`.
- Phase 1 VectorScan production qualification: `IMPLEMENTED_TESTING_DEFERRED`.
- Phase 2 coverage expansion: `IMPLEMENTED_TESTING_DEFERRED`.
- Phase 3 release/supply-chain hardening: `IMPLEMENTED_TESTING_DEFERRED`.
- Phase 4 security/product controls: implemented source foundations; live
  qualification remains deferred.
- Phase 5 deployment/security-ops/reliability/performance/HSM foundations:
  `IMPLEMENTED_TESTING_DEFERRED`.
- Enterprise Linux Distribution Slices A-D: all
  `IMPLEMENTED_TESTING_DEFERRED`.
- Project-local `./waf-package` utility: `IMPLEMENTED_TESTING_DEFERRED`.

## What is actually PASS

Scoped source/package evidence includes DEB fixture mechanics/reproducibility,
RPM source/security checks, package-tool unit/source checks, lifecycle source and
DEB fixture checks, clean-host harness/source checks, and complete-source
artifact integrity/reproducibility/negative mutation checks recorded in
`TESTING_RESULTS.md` and the package gate files.

These PASS results do **not** prove the repaired WAF root binary builds.

## What remains BLOCKED / NOT_RUN

- Exact repaired source: Go 1.25 `go mod tidy -diff` and `go build ./...` —
  `BLOCKED` on this host.
- Go 1.25 vet/full tests/race/real-Coraza gate on repaired bytes — `BLOCKED`.
- Real production `.deb` and `.rpm` from the repaired source — not built.
- Real package-manager lifecycle transactions — `NOT_RUN`.
- Seven clean-host distro rows — `NOT_RUN`.
- Real libvectorscan production gate — `NOT_RUN`.
- SoftHSM and real vendor HSM qualification — `NOT_RUN`.
- Full production performance certification — `NOT_RUN`.
- Branch protection requiring CI — `REQUIRED / NOT_APPLIED` because the
  connected GitHub integration returned HTTP 403 for write/ref operations.

## Persistence and deployment invariants

- Coraza remains authoritative; VectorScan is optional acceleration only.
- PostgreSQL is not currently used. State is filesystem-based under `/etc/waf`,
  `/var/lib/waf-proxy`, and `/var/log/waf`.
- Formal production distribution is DEB for Debian/Ubuntu and RPM for RHEL,
  Rocky, AlmaLinux, and Oracle Linux. `install.sh` is generic/development/recovery.
- Package install is offline-safe and does not fetch CRS.
- RHEL-family clean-host acceptance requires SELinux `Enforcing`.
- Shipping admin UI is `static/admin.html`; `web/` remains experimental.

## Exact next step

Create a PR branch from the audited/current `main`, apply the root-build repair,
and let Go 1.25 CI run, in order:

```bash
GOTOOLCHAIN=local go mod tidy -diff
GOTOOLCHAIN=local CGO_ENABLED=0 go build ./...
GOTOOLCHAIN=local CGO_ENABLED=0 go vet ./...
GOTOOLCHAIN=local CGO_ENABLED=0 go test ./...
GOTOOLCHAIN=local CGO_ENABLED=1 go test -race ./...
GOTOOLCHAIN=local CGO_ENABLED=1 go test -tags realcoraza -run 'TestRealCoraza' ./...
```

Only after those are green should `main` be merged/protected and real DEB/RPM
production builds resume.

## OpenAI hardening checkpoint

Responses API + strict Structured Outputs + secret references + provider mock tests are now implemented. Isolated provider evidence PASS; exact root Go 1.25 integration remains BLOCKED. Do not start another feature slice before the root PR/CI gate is green.
