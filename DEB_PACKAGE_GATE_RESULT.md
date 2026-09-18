# Debian / Ubuntu Package Gate Result

> **Documentation baseline — 2026-09-17.** This file documents a component or qualification path. Repository-wide release truth lives in [`DOCUMENTATION_INDEX.md`](DOCUMENTATION_INDEX.md), [`SOURCE_BASELINE_GATE_RESULT.md`](SOURCE_BASELINE_GATE_RESULT.md), and [`TESTING_RESULTS.md`](TESTING_RESULTS.md). Component PASS evidence must not be promoted into a root-build, runtime, package-lifecycle, clean-host, or release PASS outside its stated scope.

## Stage

Enterprise Linux Distribution Packaging — Slice A: Debian / Ubuntu DEB Packaging

Status: `IMPLEMENTED_TESTING_DEFERRED`

## Executed on the packaging host

- maintainer/build/verify shell syntax: PASS
- deterministic fixture `.deb` build: PASS
- fixture `.deb` reproducibility with fixed `SOURCE_DATE_EPOCH`: PASS
  - fixture package SHA-256: `98eba11bd319ac8df190abee8d323250e26645d847a04c3227e526dc6f958564`
- extracted package content verifier: PASS
- dpkg conffile declaration check: PASS
- packaged-admin-secret negative check: PASS
- package-maintainer-script network-fetch negative check: PASS
- package systemd `/usr/bin` path check: PASS
- persistent `StateDirectory=waf-proxy` check: PASS
- native-VectorScan package without explicit distro runtime dependency: correctly rejected

The fixture package is test evidence for the packaging machinery only. It is not a
production WAF package and is not delivered as one.

## Deferred / blocked

- production `.deb` built from the canonical Go 1.25 / Coraza v3.7.0 binaries: BLOCKED on this host (`GOTOOLCHAIN=local go version` = Go 1.23.2; `go.mod` requires Go >=1.25.0)
- clean Debian 12 install: NOT_RUN
- clean Ubuntu 22.04 install: NOT_RUN
- clean Ubuntu 24.04 install: NOT_RUN
- package upgrade / rollback qualification: reserved for Distribution Packaging Slice C
- clean-host distribution qualification: reserved for Distribution Packaging Slice D

No package-build or fixture result is a substitute for runtime WAF qualification.

## Relation to current root-build gate

This document is scoped package/distribution evidence. The 2026-09-17 audit of
`main@1d52d65` found the root source non-buildable and the prepared repair has
not yet executed on Go 1.25 in this environment. Therefore no PASS in this file
may be interpreted as a current production WAF binary/package or overall release
PASS. See `SOURCE_BASELINE_GATE_RESULT.md` and `DOCUMENTATION_INDEX.md`.
