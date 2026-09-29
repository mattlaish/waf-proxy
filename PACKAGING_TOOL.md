# WAF Project Package Builder

## Current canonical baseline — 2026-09-28

The current source is the **Code Duplication Review and Consolidation** working
baseline derived byte-for-byte from the Production Correctness & Control-Plane
Hardening parent artifact (`90acc977db2998f4a3c1c4cacc07e19f0087f8bc033bce6b9934b83415957be8`)
before the changes documented in `CODE_DUPLICATION_REVIEW.md`. Status remains
**IMPLEMENTED_TESTING_DEFERRED**. No API-9 is defined.

The review removed only source layers proven to be unwired, superseded or
functionally duplicative, and consolidated the duplicated SecLang action/token
parser into `internal/capability`. Distinct API-6/API-7/API-8 state machines,
workers and authority boundaries remain separate. The shipping Console remains
`static/admin.html` + `static/theme.css`; the previously removed experimental
`web/` tree is not part of the current source.

Current dependency-free evidence: API source gates
**69/47/46/72/33/56/45/58/83/100/110/156/259/134/190 PASS**, code-duplication
source gate **80/80 PASS with 139 unique Admin/update routes**, OpenAI source
contract **16/16 PASS** plus isolated tests PASS, WAF package-source PASS,
package-builder **9/9 PASS**, and root Go source-shape **95 files PASS**. An
isolated dependency-free `internal/capability` test also passes. Canonical Go
1.25 tidy/build/vet/test/race remains **BLOCKED_ENVIRONMENT / NOT_RUN** on this
host; none of these static/source results promotes the product to TESTED or
RELEASED.

Dated sections below are retained as historical engineering/evidence records.
When an older section conflicts with this section, `CODE_DUPLICATION_REVIEW.md`,
`DOCUMENTATION_INDEX.md`, and the current source tree are authoritative.

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

## 2026-09-28 source-gate addition

The canonical source now includes the code-duplication review gate. Package
builds must preserve and execute it alongside the existing API, production
hardening, OpenAI, package-source and artifact-integrity gates. The cleanup does
not create a second packaging path.

## 2026-09-30 — Build and Admin Startup Blocker Fix

The current source fixes two release-blocking defects found during external Go 1.25 review of the 2026-09-28 consolidation baseline. `admin.go` now imports the standard-library `errors` package required by the existing `errors.New(...)` validation paths. Positive-Schema exception state changes now use the unambiguous route `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`; the shipping Console uses the same path. The prior `POST /api/security/schema/enforcement/exceptions/{exception_id}` registration is removed because it conflicts with `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+ `http.ServeMux` specificity rules and can panic during Admin handler construction.

`admin_route_registration_test.go` now makes full Admin handler registration a regression test, and `tools/tests/test-build-startup-blocker-fix-source.py` is wired into `build.sh` and CI. The source gate also registers the complete 139-route Admin/update inventory with the Go standard library in a standalone temporary program; this catches wildcard-pattern ambiguity even when the repository module graph cannot be compiled locally. Current source evidence: blocker-fix gate **13/13 PASS**, route inventory **139/139 accepted by `ServeMux`**, API/source gates through API-8/post-audit/production-hardening/duplication remain PASS, OpenAI source contract and isolated tests PASS, package-source PASS, package-builder **9/9 PASS**, root Go source shape **96 files PASS**, Admin embedded JavaScript PASS, and changed Go files are `gofmt` clean.

This does **not** promote the source to `TESTED` or `RELEASED`. On the current execution host, canonical `GOTOOLCHAIN=local` Go 1.25 `go mod tidy -diff`, root build, vet, targeted Admin/API64 tests, and race remain `BLOCKED_ENVIRONMENT / NOT_RUN` because only Go 1.23.2 is installed and external Go toolchain/module retrieval is unavailable. Packaging/source-manifest integrity is not a substitute for those root compilation gates. No API-security authority changes are introduced: API-6/API-7 remain non-enforcing, API-8 retains explicit deterministic GraphQL ENFORCE authority, and OpenAI remains advisory only.

Release gate clarification: `SOURCE_BASELINE_GATE_RESULT` and archive integrity prove source/package completeness only. Release/buildability requires the exact artifact bytes to pass the pinned Go 1.25 root tidy/build/vet/test gates. The new Admin route-registration regression and blocker source gate are mandatory pre-compilation gates but are not substitutes for compilation.

<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
