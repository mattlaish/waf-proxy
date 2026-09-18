# RPM Package Gate Result — Enterprise Linux Distribution Packaging Slice B

> **Documentation baseline — 2026-09-17.** This file documents a component or qualification path. Repository-wide release truth lives in [`DOCUMENTATION_INDEX.md`](DOCUMENTATION_INDEX.md), [`SOURCE_BASELINE_GATE_RESULT.md`](SOURCE_BASELINE_GATE_RESULT.md), and [`TESTING_RESULTS.md`](TESTING_RESULTS.md). Component PASS evidence must not be promoted into a root-build, runtime, package-lifecycle, clean-host, or release PASS outside its stated scope.

Date: 2026-09-17 (documentation synchronized; underlying executed evidence dates are preserved below)
Status: IMPLEMENTED_TESTING_DEFERRED

## Executed on this packaging host

- `packaging/rpm/validate-rpm-source.py`: **PASS**
- RPM source network/SELinux-mutation policy test: **PASS**
- RPM/DEB package script shell syntax: **PASS**
- RPM validator Python compile: **PASS**
- `verify-release-artifact.sh` shell syntax after RPM source-gate integration: **PASS**

The source validator confirms:

- `%config(noreplace)` on the three operator configuration files;
- standard `/usr`, `/etc`, `/var/lib`, `/var/log`, `/usr/lib/systemd/system` layout;
- one-time admin-token creation and preservation;
- fresh-install no-autostart behavior;
- upgrade-only `try-restart` behavior for services already active;
- persistent state path `/var/lib/waf-proxy`;
- no RPM scriptlet `curl`, `wget`, `dnf`, `yum`, `git clone`, `setenforce`, `semanage`, or `audit2allow` path;
- explicit native-VectorScan runtime dependency gate rather than guessed RPM package names;
- reproducible-build controls (`SOURCE_DATE_EPOCH`, fixed build host, mtime clamping, deterministic payload tar ordering);
- checksum/provenance binding of supplied binaries.

## Blocked / not run

- `rpmbuild` fixture package build: **BLOCKED** — this host is Debian 13 and does not provide `rpmbuild`, `rpm`, or `rpm2cpio`.
- repeated fixture-RPM SHA-256 reproducibility comparison: **BLOCKED** by the same missing RPM toolchain.
- extracted real RPM metadata/scriptlet/file-flag verifier: **BLOCKED** by the same missing RPM toolchain.
- production RPM from canonical Go 1.25 WAF binaries: **BLOCKED** — this host cannot obtain the required Go 1.25 toolchain.
- root `go test -run TestReleaseScriptsAreLFAndBashSyntaxClean .`: **BLOCKED** — Go attempted to download Go 1.25 and network/DNS access failed.
- clean RHEL 9 install/start/traffic test: **NOT_RUN**.
- clean Rocky Linux 9 install/start/traffic test: **NOT_RUN**.
- clean AlmaLinux 9 install/start/traffic test: **NOT_RUN**.
- clean Oracle Linux 9 install/start/traffic test: **NOT_RUN**.
- SELinux enforcing runtime qualification: **NOT_RUN**; reserved for Distribution Slice D.
- package upgrade/rollback and `.rpmnew/.rpmsave` behavior: **NOT_RUN**; reserved for Distribution Slice C.

Source-level PASS does not constitute a built-RPM or target-distribution PASS.

## Relation to current root-build gate

This document is scoped package/distribution evidence. The 2026-09-17 audit of
`main@1d52d65` found the root source non-buildable and the prepared repair has
not yet executed on Go 1.25 in this environment. Therefore no PASS in this file
may be interpreted as a current production WAF binary/package or overall release
PASS. See `SOURCE_BASELINE_GATE_RESULT.md` and `DOCUMENTATION_INDEX.md`.
