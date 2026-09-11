# Phase 0 / Phase 1 Qualification Report — 2026-09-11

Timestamp: UTM+8: 2026-09-11 09:35:25

## Scope

This report records the real release-host qualification attempt against the current Stage 2 roadmap-synchronized source baseline. It does not promote stub/API or ABI-only evidence into real Coraza or real VectorScan evidence.

## Phase 0 release-host preflight

Command:

`./qualify-release-host.sh --preflight`

Result: **BLOCKED (exit 3)**

Observed host prerequisites:

- Linux 6.18.35 x86_64: PASS
- local Go toolchain: **go1.23.2 — BLOCKED; Go >=1.25.0 required**
- `go.mod` Go 1.25.0 pin: PASS
- `go.mod` Coraza v3.7.0 pin: PASS
- C compiler for CGO: PASS
- `pkg-config libhs`: **BLOCKED / unavailable**
- real VectorScan package/source provenance: **BLOCKED / unavailable**
- NGINX 1.26.3: observed
- OpenSSL 3.5.5: observed

Exact preflight transcript:

```text
== Phase 0 release-host preflight ==
INFO     host=Linux 6.18.35 x86_64
PASS     Linux host
BLOCKED  Go >=1.25.0 required; local toolchain is go1.23.2
PASS     go.mod pins Go 1.25.0
PASS     go.mod pins Coraza v3.7.0
BLOCKED  pkg-config cannot resolve libhs (install real libvectorscan-dev/libhs)
PASS     C compiler available for CGO
BLOCKED  real VectorScan provenance not established; install libvectorscan-dev or set WAF_VECTORSCAN_PROVENANCE_ACK for a reviewed source install
INFO     nginx=nginx/1.26.3
INFO     openssl=OpenSSL 3.5.5 27 Jan 2026 (Library: OpenSSL 3.5.5 27 Jan 2026)
RESULT   BLOCKED — release-host prerequisites are incomplete
```

## Phase 0 core gate

Command:

`./qualify-release-host.sh --core`

Result: **BLOCKED (exit 3)** during prerequisite preflight. The core correctness commands therefore did not execute.

The following remain **NOT_RUN** in this environment:

- `WAF_VECTORSCAN=required ./build.sh` using Go >=1.25 and real libvectorscan
- real Coraza v3.7.0 repository regression
- DetectionOnly + `nolog` transaction-final `MatchedRules()` runtime gate
- real libvectorscan `hs_compile_multi` / `hs_scan` semantic gate
- native real-libhs CGO race/regression
- clean Debian/Ubuntu install/upgrade/uninstall/doctor smoke

## Phase 1 production qualification

Result: **NOT_RUN / BLOCKED BY PHASE 0**.

Production qualification requires real Coraza v3.7.0, real libvectorscan/libhs and a production or representative OWASP CRS ruleset/corpus. The current source package contains only `coraza.conf`; it does not embed the production CRS tree. Because Phase 0 prerequisites did not pass, the production replay/differential gate was not executed.

Hard safety criterion remains unchanged:

`VectorScan candidates ⊇ Coraza transaction-final actual matches`

and observed false negatives must equal zero before any eligible group may be treated as production-qualified for acceleration.

## Dependency acquisition attempt

The execution environment could not resolve external download hosts. A direct request for the official Go 1.25.0 Linux AMD64 archive failed on DNS resolution, and no local Go 1.25 or libvectorscan package/cache was found. No ABI shim was substituted.

## Qualification status

| Gate | Status |
|---|---|
| Phase 0 preflight | BLOCKED |
| Phase 0 core correctness | NOT_RUN |
| Real Coraza v3.7.0 | NOT_RUN |
| Real `MatchedRules()` truth gate | NOT_RUN |
| Real libvectorscan compile/scan | NOT_RUN |
| Native real-libhs race | NOT_RUN |
| Phase 1 production CRS replay | NOT_RUN |
| Phase 1 zero false-negative qualification | NOT_RUN |

**Conclusion:** this host cannot complete Phase 0 or Phase 1 production qualification. The source remains an artifact-verified development baseline with qualification required.
