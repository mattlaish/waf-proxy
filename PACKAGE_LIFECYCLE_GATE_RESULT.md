# Package Upgrade / Rollback Qualification Gate — Slice C

> **Documentation baseline — 2026-09-17.** This file documents a component or qualification path. Repository-wide release truth lives in [`DOCUMENTATION_INDEX.md`](DOCUMENTATION_INDEX.md), [`SOURCE_BASELINE_GATE_RESULT.md`](SOURCE_BASELINE_GATE_RESULT.md), and [`TESTING_RESULTS.md`](TESTING_RESULTS.md). Component PASS evidence must not be promoted into a root-build, runtime, package-lifecycle, clean-host, or release PASS outside its stated scope.

Status: **IMPLEMENTED_TESTING_DEFERRED**

Slice C adds a destructive-on-opt-in lifecycle qualification harness for both
formal enterprise Linux package formats. It does not convert package fixture or
preflight results into target-distro lifecycle PASS.

## Executed on this build host

- package-lifecycle Python unit tests: **7/7 PASS**
- lifecycle source/security policy test: **PASS**
- RPM qualification-only failpoint source guard: **PASS**
- DEB lifecycle fixture build (Version N / N+1 / intentional failure): **PASS**
- DEB fixture reproducibility with fixed `SOURCE_DATE_EPOCH`: **PASS** for all three packages
- DEB lifecycle package preflight / metadata / changed-default check: **PASS**
- DEB real package-manager upgrade / failed-upgrade / rollback transaction: **NOT_RUN**
- RPM lifecycle fixture build: **BLOCKED** because this Debian host lacks `rpmbuild`, `rpm`, and `rpm2cpio`
- RPM real package-manager upgrade / failed-upgrade / rollback transaction: **NOT_RUN**

DEB fixture SHA-256 values from the executed reproducibility gate:

- baseline `5.0.0.qualification1`: `f853c78478f06b1dd5df36aa3cb4ff46067096f867bcdbd554b082179565b908`
- candidate `5.0.0.qualification2`: `c9f26751a57ecb9edb6af5121fe865c3533c6f496d61481e56f109f3be24ff2e`
- intentional failure `5.0.0.qualification3`: `b69380a63fac985153269e0934056601cd3b91df3a302b5650559f46334f1fe8`

The preflight confirmed that all three packaged operator-config defaults differ
between N and N+1, so the real DEB lifecycle run can exercise native
`--force-confold` / `.dpkg-dist` behavior rather than merely checking unchanged
files.

## Qualification truth boundary

A full Slice C PASS requires actual package-manager execution on a dedicated
supported host. The runner compares the generated admin secret only in memory;
no token, secret-reference value, token hash, or other secret derivative is
written into evidence. Persistent state is tested using a non-secret marker.

The lifecycle runner never calls dependency resolvers or network tools (`apt`,
`dnf`, `yum`, `curl`, `wget`). Dependencies must already be installed. This
preserves the enterprise offline-safe packaging contract.

Clean-host OS matrix acceptance remains Slice D and is not implied by Slice C.

## Source delivery gates

The frozen Slice C candidate also passed baseline patch reconstruction, complete
source artifact integrity, fixed-epoch source-release reproducibility, 12/12
negative artifact mutations, and clean-extraction source/DEB-preflight smoke.
Those delivery gates validate the implementation artifact only and do not change
the real package-manager transaction states above.

## Relation to current root-build gate

This document is scoped package/distribution evidence. The 2026-09-17 audit of
`main@1d52d65` found the root source non-buildable and the prepared repair has
not yet executed on Go 1.25 in this environment. Therefore no PASS in this file
may be interpreted as a current production WAF binary/package or overall release
PASS. See `SOURCE_BASELINE_GATE_RESULT.md` and `DOCUMENTATION_INDEX.md`.
