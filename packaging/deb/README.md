# Debian / Ubuntu package builder

> **Documentation baseline — 2026-09-17.** This file documents a component or qualification path. Repository-wide release truth lives in [`DOCUMENTATION_INDEX.md`](../../DOCUMENTATION_INDEX.md), [`SOURCE_BASELINE_GATE_RESULT.md`](../../SOURCE_BASELINE_GATE_RESULT.md), and [`TESTING_RESULTS.md`](../../TESTING_RESULTS.md). Component PASS evidence must not be promoted into a root-build, runtime, package-lifecycle, clean-host, or release PASS outside its stated scope.

This directory implements the formal Debian/Ubuntu enterprise distribution path.
It packages **already-qualified build outputs**; it does not compile Go code and
never downloads OWASP CRS during package installation.

## Build

On the qualified release host, first build the binaries:

```bash
WAF_VECTORSCAN=off WAF_HSM_PKCS11=off ./build.sh
```

Then build the deterministic `.deb`:

```bash
SOURCE_DATE_EPOCH=<approved-epoch> \
  packaging/deb/build-deb.sh --binaries-dir . --output-dir dist/deb
```

A native VectorScan binary additionally requires the distribution-specific
runtime dependency to be supplied explicitly, for example:

```bash
SOURCE_DATE_EPOCH=<approved-epoch> \
WAF_DEB_EXTRA_DEPENDS='libvectorscan5' \
  packaging/deb/build-deb.sh --binaries-dir . --output-dir dist/deb
```

The builder verifies `BUILD_SHA256SUMS.txt`, requires build provenance, refuses
non-ELF inputs outside the fixture test, normalizes package mtimes, and runs
`verify-deb.sh` before reporting success.

## Install

```bash
sudo apt install ./dist/deb/waf-proxy_<version>_<arch>.deb
```

Fresh install intentionally does not auto-start the service. Provision approved
CRS content into `/etc/waf/crs`, run `sudo waf-doctor --check`, then explicitly:

```bash
sudo systemctl enable --now waf-proxy
```

Upgrades preserve dpkg conffiles, `/etc/waf/waf-proxy.env`, certificates/CRS,
and `/var/lib/waf-proxy`. No maintainer script performs network access.
