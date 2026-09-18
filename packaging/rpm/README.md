# RHEL-family RPM package builder

> **Documentation baseline — 2026-09-17.** This file documents a component or qualification path. Repository-wide release truth lives in [`DOCUMENTATION_INDEX.md`](../../DOCUMENTATION_INDEX.md), [`SOURCE_BASELINE_GATE_RESULT.md`](../../SOURCE_BASELINE_GATE_RESULT.md), and [`TESTING_RESULTS.md`](../../TESTING_RESULTS.md). Component PASS evidence must not be promoted into a root-build, runtime, package-lifecycle, clean-host, or release PASS outside its stated scope.

This directory implements the formal enterprise RPM distribution path for RHEL,
Rocky Linux, AlmaLinux and Oracle Linux. It packages **already-qualified build
outputs**; it does not compile Go in an RPM install transaction and never fetches
OWASP CRS from package scriptlets.

## Release-host build

A qualified RHEL-family release host needs `rpm-build`, `rpm`, `rpm2cpio`,
`cpio`, `tar`, `python3`, `openssl` and the normal Go build prerequisites.

```bash
SOURCE_DATE_EPOCH=<approved-epoch> \
  packaging/rpm/build-release-rpm.sh --output-dir dist/rpm
```

For an already-built binary set:

```bash
SOURCE_DATE_EPOCH=<approved-epoch> \
  packaging/rpm/build-rpm.sh --binaries-dir . --output-dir dist/rpm
```

A native VectorScan binary must declare the exact target-distribution runtime
package explicitly. The builder intentionally refuses to guess it:

```bash
SOURCE_DATE_EPOCH=<approved-epoch> \
WAF_RPM_EXTRA_REQUIRES='<approved vectorscan runtime RPM name>' \
  packaging/rpm/build-rpm.sh --binaries-dir . --output-dir dist/rpm
```

PKCS#11/HSM vendor modules remain operator-provided dlopen targets and are not
silently converted into an RPM dependency.

## Install

```bash
sudo dnf install ./dist/rpm/waf-proxy-<version>-1.<arch>.rpm
```

Fresh installation does **not** start the WAF automatically. Provision approved
CRS content under `/etc/waf/crs`, run `sudo waf-doctor --check`, then explicitly:

```bash
sudo systemctl enable --now waf-proxy
```

`/etc/waf/config.json`, `coraza.conf`, and `waf-tls-frontend.env` use
`%config(noreplace)`. `/etc/waf/waf-proxy.env`, certificates/CRS and
`/var/lib/waf-proxy` are preserved across package upgrades. RPM scriptlets do
not download packages, CRS, Git repositories, or other network content.

See `SELINUX.md` for the enforcing-SELinux qualification boundary.
