# WAF Project Package Builder

> **Documentation baseline — 2026-09-17.** This file documents a component or qualification path. Repository-wide release truth lives in [`DOCUMENTATION_INDEX.md`](DOCUMENTATION_INDEX.md), [`SOURCE_BASELINE_GATE_RESULT.md`](SOURCE_BASELINE_GATE_RESULT.md), and [`TESTING_RESULTS.md`](TESTING_RESULTS.md). Component PASS evidence must not be promoted into a root-build, runtime, package-lifecycle, clean-host, or release PASS outside its stated scope.

## Current buildability precondition

As of 2026-09-17, the root-build-integrity repair is implemented but its exact
bytes remain BLOCKED pending the mandatory Go 1.25 tidy/root-build CI gate.
`./waf-package` must only produce production packages from a commit that has
passed that gate. The tool intentionally fails closed when the selected Go
toolchain or dependency state is unsuitable.

`./waf-package` is the project-local package build entry point for WAF Reverse Proxy.
It is intentionally specific to this repository. It composes the checked-in
`build.sh`, DEB builder/verifier, and RPM builder/verifier instead of maintaining
a second build implementation.

## What it does

For every real package build it:

1. reads the required Go version and Coraza pin from `go.mod`;
2. refuses a Go toolchain older than the project requirement;
3. checks the selected DEB/RPM host tools;
4. computes a source fingerprint and reproducible `SOURCE_DATE_EPOCH`;
5. invokes the canonical `build.sh` (tidy diff, vet, tests, race test, real Coraza gate, binaries, provenance);
6. packages the exact generated binaries through the existing DEB/RPM builder;
7. relies on the package verifier embedded in those builders;
8. writes `PACKAGE_BUILD_REPORT.json` with source identity, toolchain/provenance, package paths, sizes, and SHA-256 values.

It never installs Go, package-manager tooling, CRS, native libraries, or other
dependencies. Missing prerequisites fail closed as `PACKAGE_BUILD_BLOCKED`.

## Fast path

From the repository root:

```bash
./waf-package doctor --format deb
./waf-package deb --version 2026.09.16.1
```

For RHEL-family RPM:

```bash
./waf-package doctor --format rpm
./waf-package rpm --version 2026.09.16.1 --rpm-release 1
```

Build both from one canonical binary build:

```bash
./waf-package all \
  --version 2026.09.16.1 \
  --deb-arch amd64 \
  --rpm-arch x86_64
```

Package builds are offline with respect to Go modules by default (`GOPROXY=off`).
For a connected build host where the operator explicitly permits normal Go module
resolution, use:

```bash
./waf-package deb --allow-module-network --version 2026.09.16.1
```

`--offline` may be supplied explicitly but is already the default. Neither mode
automatically installs a Go toolchain, operating-system packages, CRS, or native
libraries.

## Host prerequisites

Common requirements:

- Python 3.10+
- Go version satisfying `go.mod` (currently Go 1.25.0+)
- bash
- gcc / working C toolchain (the canonical race/real-Coraza gates use CGO)
- sha256sum
- Go module dependencies already available locally or reachable through the configured Go module proxy

DEB additionally requires `dpkg` and `dpkg-deb`.

RPM additionally requires `rpmbuild`, `rpm`, `rpm2cpio`, `cpio`, and `tar`.
Install the distribution packages that provide those commands before running the
tool; `waf-package` deliberately does not modify the host.

## Native capabilities

Portable package (default):

```bash
./waf-package deb --vectorscan off --hsm off
```

Require VectorScan:

```bash
./waf-package deb \
  --vectorscan required \
  --deb-extra-dep '<approved Debian libhs runtime package>'
```

```bash
./waf-package rpm \
  --vectorscan required \
  --rpm-extra-require '<approved RHEL-family libhs runtime requirement>'
```

The tool never guesses distribution-specific VectorScan runtime package names.

Require the PKCS#11 build path:

```bash
./waf-package deb --hsm required
```

HSM module/provider runtime qualification remains separate from package creation.

## Selecting a Go toolchain

If the correct Go is not first in `PATH`:

```bash
./waf-package doctor --format deb --go-bin /opt/go1.25/bin/go
./waf-package deb --go-bin /opt/go1.25/bin/go --version 2026.09.16.1
```

`GOTOOLCHAIN=local` is forced so a build cannot silently switch toolchains.

## Version and source identity

`--version` uses a deliberately conservative cross-DEB/RPM character set:

```text
[A-Za-z0-9][A-Za-z0-9._+~]*
```

This prevents DEB and RPM from silently normalizing the same requested version
to different strings.

If `--version` is omitted, the version is derived from the reproducible epoch and
source fingerprint:

```text
YYYY.MM.DD.src<8-hex-source-fingerprint>
```

If Git metadata exists, the source identity is the current short commit. Dirty
tracked or untracked changes add a source-fingerprint suffix. Delivery source
archives without `.git` use `source-<fingerprint>`.

## Reproducibility

Recommended release invocation:

```bash
SOURCE_DATE_EPOCH=<approved-epoch> \
./waf-package deb \
  --version <release-version> \
  --offline
```

The DEB/RPM builders perform their own deterministic timestamp normalization and
package verification. Run the same build twice on an equivalent qualified host
when package-byte reproducibility is a release requirement.

## Outputs

Typical DEB output:

```text
dist/deb/waf-proxy_<version>_<arch>.deb
dist/deb/waf-proxy_<version>_<arch>.deb.sha256
dist/deb/PACKAGE_BUILD_REPORT.json
```

Typical RPM output:

```text
dist/rpm/waf-proxy-<version>-<release>.<arch>.rpm
dist/rpm/waf-proxy-<version>-<release>.<arch>.rpm.sha256
dist/rpm/PACKAGE_BUILD_REPORT.json
```

`all` can use one output directory for both package formats.

## Failure semantics

A missing or unsuitable prerequisite returns exit code `2` with:

```text
PACKAGE_BUILD_BLOCKED: <reason>
```

A canonical build/test/package command failure also stops the run. The tool does
not downgrade Go/Coraza, skip tests, substitute fixture binaries, or manufacture
PASS evidence.
