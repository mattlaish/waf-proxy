# OWASP CRS provisioning for the Debian/Ubuntu package

> **Documentation baseline — 2026-09-17.** This file documents a component or qualification path. Repository-wide release truth lives in [`DOCUMENTATION_INDEX.md`](../../DOCUMENTATION_INDEX.md), [`SOURCE_BASELINE_GATE_RESULT.md`](../../SOURCE_BASELINE_GATE_RESULT.md), and [`TESTING_RESULTS.md`](../../TESTING_RESULTS.md). Component PASS evidence must not be promoted into a root-build, runtime, package-lifecycle, clean-host, or release PASS outside its stated scope.

The `waf-proxy` `.deb` is intentionally **offline-safe**. Its maintainer scripts
never use `curl`, `wget`, `apt`, Git, or any other network fetch to obtain the
OWASP Core Rule Set (CRS).

Before first service start, provision an approved CRS tree into `/etc/waf/crs`
and create `/etc/waf/crs/crs-setup.conf`. The package ships `/etc/waf/coraza.conf`
as a dpkg conffile and preserves local edits across upgrades.

For an Internet-connected administrative workflow you may obtain the approved
CRS version out-of-band, verify it, then copy/extract it into `/etc/waf/crs`.
For air-gapped production, distribute CRS through your organization’s signed
artifact repository, configuration-management system, or a separately governed
package. Package `postinst` must remain network-free.

After provisioning:

```bash
sudo chown -R root:waf /etc/waf/crs
sudo find /etc/waf/crs -type d -exec chmod 0750 {} +
sudo find /etc/waf/crs -type f -exec chmod 0640 {} +
sudo waf-doctor --check
sudo systemctl enable --now waf-proxy
```

Do not treat successful `.deb` installation as WAF readiness when CRS is absent.
