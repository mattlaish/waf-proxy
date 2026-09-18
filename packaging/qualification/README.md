# Package Upgrade / Rollback Qualification

> **Documentation baseline — 2026-09-17.** This file documents a component or qualification path. Repository-wide release truth lives in [`DOCUMENTATION_INDEX.md`](../../DOCUMENTATION_INDEX.md), [`SOURCE_BASELINE_GATE_RESULT.md`](../../SOURCE_BASELINE_GATE_RESULT.md), and [`TESTING_RESULTS.md`](../../TESTING_RESULTS.md). Component PASS evidence must not be promoted into a root-build, runtime, package-lifecycle, clean-host, or release PASS outside its stated scope.

This directory implements Enterprise Linux Distribution Packaging **Slice C**.
It qualifies package-manager lifecycle behavior; it does not qualify WAF
request detection, performance, or the clean-host distro matrix from Slice D.

## Safety model

`package_lifecycle_qualify.py` is non-mutating unless `--execute` is supplied.
Real execution is intentionally restricted to a **dedicated disposable VM** and
requires root plus the exact acknowledgement:

```text
I_UNDERSTAND_THIS_MUTATES_A_DEDICATED_HOST
```

The runner never invokes `apt`, `dnf`, `yum`, `curl`, or `wget`. Dependencies
must already be present. Package qualification is therefore offline-safe.

## What is tested

For both DEB and RPM the runner binds evidence to Version N, Version N+1 and an
optional intentional-failure fixture by SHA-256 and native package metadata. It
then validates:

1. baseline installation;
2. operator-modified configuration preservation;
3. generated administrator secret preservation without writing the secret or a
   secret digest into evidence;
4. `/var/lib/waf-proxy` persistent-state preservation;
5. upgrade to N+1;
6. service active/inactive state preservation;
7. native conffile conflict behavior (`.dpkg-dist` for `--force-confold`,
   `.rpmnew` for `%config(noreplace)`) when package defaults actually changed;
8. intentional post-install failure and preservation after that failure;
9. recovery from the failed transaction;
10. rollback/downgrade to Version N with config, secret and state preserved.

`.dpkg-old` / `.rpmsave` files are captured if the native package manager emits
them, but they are not fabricated or required when the package manager's chosen
semantics do not create them.

## Fixture packages

`build-lifecycle-fixtures.sh` creates N / N+1 / intentional-failure fixtures.
The payload uses `/bin/true`, so these artifacts validate package lifecycle only
and **must never be treated as WAF runtime qualification**. Candidate defaults
are intentionally changed so `.dpkg-dist` / `.rpmnew` paths can be exercised.
The RPM failure package is built with a compile-time spec macro that injects a
`%post` failure only into the qualification fixture; normal RPM builds do not
contain that branch.

Example on Debian/Ubuntu:

```bash
SOURCE_DATE_EPOCH=1700000000 \
  ./packaging/qualification/build-lifecycle-fixtures.sh \
  --format deb --output-dir /tmp/waf-lifecycle-deb

sudo ./packaging/qualification/run-package-lifecycle-qualification.sh \
  --format deb \
  --baseline /tmp/waf-lifecycle-deb/baseline/*.deb \
  --candidate /tmp/waf-lifecycle-deb/candidate/*.deb \
  --failure-fixture /tmp/waf-lifecycle-deb/failure/*.deb \
  --output /tmp/deb-upgrade-rollback.json \
  --execute \
  --ack I_UNDERSTAND_THIS_MUTATES_A_DEDICATED_HOST
```

Use `--format rpm` on RHEL/Rocky/AlmaLinux/Oracle Linux with the corresponding
RPM fixture set. Real production package lifecycle qualification should rerun
the same runner with qualified Go 1.25 packages; fixture PASS alone is not
production distro acceptance.
