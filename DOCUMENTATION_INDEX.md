# Documentation Index and Current Project Truth

> **Canonical documentation baseline: 2026-09-17 (Asia/Taipei).**
>
> This file is the starting point for operators, developers, release engineers,
> and future AI sessions. Historical ledgers remain valuable evidence, but when
> older text conflicts with this page and the current gate files, this page plus
> the current gate files are authoritative.

## Current canonical state

- Audited GitHub baseline: `main@1d52d65a73a802e32f02994e51f0a07beb240177`.
- A root-build-integrity repair has been prepared for the broken main baseline.
- Repair status: `IMPLEMENTED_TESTING_DEFERRED`.
- **Source Buildability Gate: BLOCKED.** The exact repaired bytes still require a
  Go 1.25.x execution of `GOTOOLCHAIN=local go mod tidy -diff` and
  `GOTOOLCHAIN=local CGO_ENABLED=0 go build ./...`, followed by vet/tests/race
  and the real-Coraza gate.
- Current documentation/packaging host: Go 1.23.2; Go 1.25/module acquisition is
  unavailable here. A missing prerequisite is BLOCKED, never PASS.
- Artifact integrity/reproducibility and package-fixture tests have independent
  PASS evidence, but they are not evidence that the WAF root binary builds or
  that a production DEB/RPM is qualified.
- GitHub `main` was observed unprotected with required status checks disabled.
  Branch protection/rulesets requiring CI before merge are REQUIRED but were
  not applied by the connected integration because GitHub write/ref operations
  returned HTTP 403.

## Status vocabulary

Implementation state uses only:

- `PLANNED` — approved but not implemented.
- `IMPLEMENTED_TESTING_DEFERRED` — implementation exists, but one or more
  required validation gates are not executed or are blocked.
- `TESTED` — the defined mandatory tests for that scope have executed and passed.
- `RELEASED` — explicitly approved and released after all release-blocking gates.

Evidence state uses `PASS`, `FAIL`, `BLOCKED`, `NOT_RUN`, and where applicable
`NOT_CONFIGURED`. Never translate `BLOCKED` or `NOT_RUN` into PASS.

## Release blockers and required order

1. Apply the exact root-build-integrity repair to a branch based on the audited
   `main` baseline; do not patch directly to `main`.
2. On Go 1.25.x CI, run dependency-drift, root build, vet, full tests, race, and
   real-Coraza truth gates against the exact proposed commit.
3. Require the CI build/test check through branch protection/rulesets before
   merging to `main`.
4. Only from a green buildable commit, produce real WAF `.deb` and `.rpm`
   packages with `./waf-package` on qualified build hosts.
5. Run package lifecycle qualification (Slice C) on dedicated hosts.
6. Run clean-host distribution qualification (Slice D) on the exact supported
   distro matrix; RHEL-family acceptance requires SELinux `Enforcing`.
7. Complete remaining real Coraza/VectorScan, HSM, reliability, and performance
   gates before any production release decision.

## Distribution state

| Area | State | Current truth |
|---|---|---|
| Debian/Ubuntu DEB builder | `IMPLEMENTED_TESTING_DEFERRED` | fixture mechanics/reproducibility PASS; no production WAF DEB from the repaired source yet |
| RHEL-family RPM builder | `IMPLEMENTED_TESTING_DEFERRED` | source/security validation PASS; actual RPM build is BLOCKED on this host by missing RPM toolchain and root Go build |
| Package upgrade/rollback | `IMPLEMENTED_TESTING_DEFERRED` | harness and DEB fixtures PASS; real package-manager transactions remain NOT_RUN |
| Clean-host qualification | `IMPLEMENTED_TESTING_DEFERRED` | harness/source tests PASS; Debian 12, Ubuntu 22.04/24.04, RHEL/Rocky/Alma/Oracle 9 rows remain NOT_RUN |
| Project package utility | `IMPLEMENTED_TESTING_DEFERRED` | `./waf-package` source/unit gates PASS; real package production remains blocked by build-host prerequisites |

## Runtime and persistence truth

- Coraza v3.7.0 is authoritative for WAF decisions.
- VectorScan/libhs is optional acceleration and must fail safe to Coraza.
- PKCS#11/HSM support is optional and fail-closed; SoftHSM and real-vendor
  qualification remain separate evidence classes.
- Current WAF persistence is filesystem-based under `/etc/waf`,
  `/var/lib/waf-proxy`, and `/var/log/waf`.
- **PostgreSQL is not a current runtime dependency or supported persistence
  backend.** Do not invent `DATABASE_URL`, PostgreSQL migrations, or a database
  provisioning requirement in deployment instructions.
- The shipping admin console remains embedded `static/admin.html`; `web/` is an
  experimental non-shipping migration tree.

## Canonical documents

| Document | Purpose |
|---|---|
| `README.md` | product overview and current release truth |
| `INSTALL.md` | production/package installation and operations |
| `DEVELOPMENT.md` | chronological engineering ledger |
| `DEVELOPMENT_ROADMAP.md` | implementation states, blockers, and next order |
| `AI_HANDOFF.md` | current architecture/state for another AI/session |
| `HANDOVER_STATUS.md` | concise human-readable checkpoint |
| `HANDOVER_PROMPT.md` | copy/paste prompt for a new development chat |
| `TESTING.md` | required test/gate policy |
| `TESTING_RESULTS.md` | executed evidence and deferred/blocked truth |
| `SOURCE_BASELINE_GATE_RESULT.md` | root source-buildability gate |
| `RELEASE_PROCESS.md` | release sequencing and artifact rules |
| `MANIFEST.md` | source/delivery file map |
| `patch.md` | implementation patch ledger |

## Package and distribution gate documents

- `DEB_PACKAGE_GATE_RESULT.md`
- `RPM_PACKAGE_GATE_RESULT.md`
- `PACKAGE_LIFECYCLE_GATE_RESULT.md`
- `CLEAN_HOST_DISTRIBUTION_GATE_RESULT.md`
- `PACKAGE_TOOL_GATE_RESULT.md`
- `PACKAGING_TOOL.md`

These documents are scoped evidence. Their PASS entries do not automatically
promote the Source Buildability Gate or overall release status.

## Component documentation

- `benchmark/README.md` — benchmark and performance-certification harness.
- `packaging/deb/README.md` and `packaging/deb/CRS-PROVISIONING.md` — DEB build
  and offline CRS provisioning.
- `packaging/rpm/README.md`, `packaging/rpm/CRS-PROVISIONING.md`, and
  `packaging/rpm/SELINUX.md` — RPM build, offline CRS, and SELinux boundary.
- `packaging/qualification/README.md` — upgrade/rollback qualification.
- `packaging/cleanhost/README.md` — clean-host distro acceptance.
- `qualification/README.md` and subdirectory READMEs — runtime, corpus, HSM,
  and performance qualification.
- `web/PORTING.md` — experimental console migration only.

## Documentation maintenance rule

Every meaningful source/package/release change must update the relevant current
state in `AI_HANDOFF.md`, `DEVELOPMENT.md`, `DEVELOPMENT_ROADMAP.md`,
`TESTING.md`, `TESTING_RESULTS.md`, `MANIFEST.md`, and `patch.md`. Update
`README.md`, `INSTALL.md`, package gate files, and component READMEs whenever
operator behavior or that component's truth changes. Historical evidence may be
retained, but stale "next step" text must be marked historical or removed from
current handover files.

## OpenAI connector hardening — current state

- Status: `IMPLEMENTED_TESTING_DEFERRED`.
- Native OpenAI uses Responses API + strict Structured Outputs; OpenAI-compatible Chat Completions remains available.
- New credentials use `api_key_ref=env:NAME|file:/absolute/path`; legacy inline `api_key` is migration-only.
- Isolated provider/secret tests, race, vet, and 16/16 source contract checks PASS.
- Root Go 1.25 build/test remains BLOCKED; see `OPENAI_INTEGRATION_GATE_RESULT.md`.


## API Security Roadmap

See `API_SECURITY_ROADMAP.md` for the planned API-aware WAF evolution slices.
