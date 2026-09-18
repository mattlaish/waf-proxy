# Project-local Package Tool Gate Result

> **Documentation baseline — 2026-09-17.** This file documents a component or qualification path. Repository-wide release truth lives in [`DOCUMENTATION_INDEX.md`](DOCUMENTATION_INDEX.md), [`SOURCE_BASELINE_GATE_RESULT.md`](SOURCE_BASELINE_GATE_RESULT.md), and [`TESTING_RESULTS.md`](TESTING_RESULTS.md). Component PASS evidence must not be promoted into a root-build, runtime, package-lifecycle, clean-host, or release PASS outside its stated scope.

Date: 2026-09-17 (documentation synchronized; underlying executed evidence dates are preserved below)
Status: IMPLEMENTED_TESTING_DEFERRED

## Implemented

- root entry point: `./waf-package`
- implementation: `tools/waf_package_builder.py`
- commands: `doctor`, `deb`, `rpm`, `all`
- canonical `build.sh` composition; no parallel binary build path
- Go/Coraza requirements read from `go.mod`
- `GOTOOLCHAIN=local` fail-closed toolchain policy
- Go module access offline by default (`GOPROXY=off`)
- explicit `--allow-module-network` opt-in
- deterministic source fingerprint / source epoch / default version
- separate DEB/RPM architecture mapping in `all` mode
- provenance-bound package report with SHA-256
- explicit native VectorScan runtime dependency handling

## Executed PASS

- Python unit tests: 9/9 PASS
- package-tool source/CLI contract: PASS
- Python compile + shell syntax: PASS
- inherited DEB packaging fixture/reproducibility regression: PASS
- inherited RPM source/security regression: PASS
- inherited package-lifecycle source regression: PASS
- inherited clean-host source regression: PASS

## Executed BLOCKED (expected fail-closed evidence)

`./waf-package doctor --format deb` returns `PACKAGE_BUILD_BLOCKED` on the current host because the selected Go toolchain is Go 1.23.2 while `go.mod` requires Go 1.25.0.

## Not run / deferred

- real WAF DEB build through `./waf-package deb`: BLOCKED pending Go 1.25+ and required modules
- real WAF RPM build through `./waf-package rpm`: BLOCKED pending Go 1.25+ and RPM build tools
- package-byte reproducibility for real WAF packages: NOT_RUN
- Slice C real package lifecycle: NOT_RUN
- Slice D clean-host distro matrix: NOT_RUN

This gate does not claim binary package qualification.

## Complete-source delivery gates

Candidate complete-source artifact gates executed before final source freeze:

- source-baseline patch reconstruction: PASS — 261 comparable source files, byte + Unix-mode identity
- Artifact Integrity Gate: PASS — 261 source files / 267 packaged files
- fixed-epoch complete-source rebuild: PASS — byte-identical SHA-256
- clean-extract package-tool source gate: PASS — 9/9 unit tests + source contract
- clean-extract RPM source/security gate: PASS
- artifact mutation rejection: PASS — 12/12, executed in timeout-safe segments

These are complete-source artifact integrity results, not real DEB/RPM binary qualification.

## Relation to current root-build gate

This document is scoped package/distribution evidence. The 2026-09-17 audit of
`main@1d52d65` found the root source non-buildable and the prepared repair has
not yet executed on Go 1.25 in this environment. Therefore no PASS in this file
may be interpreted as a current production WAF binary/package or overall release
PASS. See `SOURCE_BASELINE_GATE_RESULT.md` and `DOCUMENTATION_INDEX.md`.
