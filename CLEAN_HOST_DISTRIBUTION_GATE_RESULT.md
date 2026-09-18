# Clean-host Distribution Gate Result

> **Documentation baseline — 2026-09-17.** This file documents a component or qualification path. Repository-wide release truth lives in [`DOCUMENTATION_INDEX.md`](DOCUMENTATION_INDEX.md), [`SOURCE_BASELINE_GATE_RESULT.md`](SOURCE_BASELINE_GATE_RESULT.md), and [`TESTING_RESULTS.md`](TESTING_RESULTS.md). Component PASS evidence must not be promoted into a root-build, runtime, package-lifecycle, clean-host, or release PASS outside its stated scope.

Date: 2026-09-17 (documentation synchronized; underlying executed evidence dates are preserved below)
Slice: Enterprise Linux Distribution Packaging Slice D
Status: IMPLEMENTED_TESTING_DEFERRED

## Executed on this packaging host

- clean-host Python unit tests: PASS (7/7)
- clean-host source/security contract: PASS
- shell/Python syntax: PASS
- current-host exact-platform guard: PASS (correctly BLOCKED; Debian 13 is not a supported Slice D target)

## Real target-host acceptance

- Debian 12: NOT_RUN
- Ubuntu 22.04: NOT_RUN
- Ubuntu 24.04: NOT_RUN
- RHEL 9 + SELinux Enforcing: NOT_RUN
- Rocky Linux 9 + SELinux Enforcing: NOT_RUN
- AlmaLinux 9 + SELinux Enforcing: NOT_RUN
- Oracle Linux 9 + SELinux Enforcing: NOT_RUN

## Truth boundary

Source/unit/preflight/container/chroot evidence cannot produce a clean-host PASS.
PASS requires a dedicated clean target host to execute native local package
install, explicit local CRS provisioning, doctor, systemd start, health,
break-glass authenticated admin API access, actual reverse-proxy traffic,
Version N to N+1 upgrade preservation, and non-purge package removal.

## Inherited/regression checks

- Slice C lifecycle source/security gate: PASS
- RPM source/security gate: PASS
- DEB packaging regression: PASS
- release verifier/build script syntax: PASS
- canonical Go shipped-script test: BLOCKED (Go 1.23.2; module requires >=1.25.0)
- Git synchronization: NOT_AVAILABLE (complete-source ZIP contains no `.git` metadata)

## Source delivery gates

- baseline-relative patch reconstruction: PASS (261 files, byte/mode identical)
- complete-source artifact integrity: PASS (255 source / 261 packaged)
- fixed-epoch source reproducibility: PASS
- clean-extracted Slice D source/unit gate: PASS
- artifact negative mutation rejection: PASS (12/12, segmented execution)
- detached producer signature: NOT_CONFIGURED

## Relation to current root-build gate

This document is scoped package/distribution evidence. The 2026-09-17 audit of
`main@1d52d65` found the root source non-buildable and the prepared repair has
not yet executed on Go 1.25 in this environment. Therefore no PASS in this file
may be interpreted as a current production WAF binary/package or overall release
PASS. See `SOURCE_BASELINE_GATE_RESULT.md` and `DOCUMENTATION_INDEX.md`.
