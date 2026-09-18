# Clean-host Distribution Qualification

> **Documentation baseline — 2026-09-17.** This file documents a component or qualification path. Repository-wide release truth lives in [`DOCUMENTATION_INDEX.md`](../../DOCUMENTATION_INDEX.md), [`SOURCE_BASELINE_GATE_RESULT.md`](../../SOURCE_BASELINE_GATE_RESULT.md), and [`TESTING_RESULTS.md`](../../TESTING_RESULTS.md). Component PASS evidence must not be promoted into a root-build, runtime, package-lifecycle, clean-host, or release PASS outside its stated scope.

This directory implements Enterprise Linux Distribution Packaging **Slice D**.
It validates that an approved local WAF package can be installed on a clean,
dedicated OS instance and progress through first start, authenticated management
access, proxy traffic, package upgrade, and package removal without network
fetches or hidden setup steps.

## Supported matrix

- Debian 12
- Ubuntu 22.04 LTS
- Ubuntu 24.04 LTS
- RHEL 9
- Rocky Linux 9
- AlmaLinux 9
- Oracle Linux 9

RHEL-family acceptance additionally requires SELinux to be **Enforcing**. A
container, chroot, shared build host, or source-level simulation cannot produce
a clean-host PASS.

## Truth boundary

The runner defaults to non-mutating preflight. Real execution requires root,
`--execute`, and this exact acknowledgement:

```text
I_UNDERSTAND_THIS_MUTATES_A_DEDICATED_CLEAN_HOST
```

The host must begin with `waf-proxy` uninstalled and without an existing
`/etc/waf` or `/var/lib/waf-proxy`. The runner never invokes `apt`, `apt-get`,
`dnf`, `yum`, `curl`, `wget`, `git`, or any other dependency/network fetcher.
Dependencies, Version N/N+1 packages, and an approved CRS directory must already
be present locally.

## Qualification flow

1. verify exact distro/version and clean-host preconditions;
2. bind baseline/candidate package metadata and SHA-256 into evidence;
3. install Version N with the native offline package command (`dpkg -i` or
   `rpm -Uvh`);
4. verify fresh install did not auto-start the WAF;
5. provision the supplied local CRS directory and create a loopback-only
   qualification config/backend;
6. run `waf-doctor --check`;
7. explicitly enable/start `waf-proxy`;
8. verify `/healthz`, break-glass first-login API access, and real reverse-proxy
   traffic to the local backend;
9. create persistent-state evidence, upgrade to Version N+1, and repeat health,
   login, traffic, secret/config/state checks;
10. remove (not purge) the package and verify binaries/service are gone while
    break-glass secret, CRS and persistent state remain preserved.

The temporary qualification backend and systemd drop-in are removed on exit.
The report never contains the administrator token or any derivative of it.

Example on a dedicated Debian 12 VM:

```bash
sudo ./packaging/cleanhost/run-clean-host-qualification.sh \
  --platform debian-12 \
  --baseline /qualification/waf-proxy_N_amd64.deb \
  --candidate /qualification/waf-proxy_Nplus1_amd64.deb \
  --crs-dir /qualification/coreruleset \
  --output /qualification/debian-12-clean-host.json \
  --execute \
  --ack I_UNDERSTAND_THIS_MUTATES_A_DEDICATED_CLEAN_HOST
```

Use the corresponding RPM packages on RHEL-family hosts. Slice C remains the
more exhaustive rollback/failed-upgrade qualification; Slice D intentionally
focuses on the end-to-end clean-machine operator experience.
