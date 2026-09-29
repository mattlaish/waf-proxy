# Code Duplication Review Source Gate Result

## Current result — 2026-09-28

Status: **PASS (dependency-free source gate)**.

Command:

```bash
python3 tools/tests/test-code-duplication-review-source.py
```

Observed result:

```text
CODE_DUPLICATION_REVIEW_SOURCE_GATE_PASS checks=80 routes=139
```

The gate verifies that the confirmed superseded/dead duplicate layers removed by
`CODE_DUPLICATION_REVIEW.md` remain absent, the shared SecLang parsing helpers
are the single source used by both CRS coverage and live VectorScan
classification, current replacement authorities remain present, Admin route
registrations are unique, API-6/API-7/API-8 authority boundaries remain
separate, and the gate itself is retained by build/CI wiring.

This is **source/static evidence only**. It does not replace the mandatory
pinned Go 1.25 `tidy`, build, vet, test and race qualification. Those canonical
Go gates remain `BLOCKED_ENVIRONMENT / NOT_RUN` on the current host because the
host provides Go 1.23.2 and external toolchain/module retrieval is unavailable.

An isolated dependency-free test of `internal/capability` using the host Go
1.23 toolchain passed after consolidation. That isolated test is advisory and
is not a substitute for the repository's pinned Go 1.25 qualification.

## 2026-09-30 — Build and Admin Startup Blocker Fix

The current source fixes two release-blocking defects found during external Go 1.25 review of the 2026-09-28 consolidation baseline. `admin.go` now imports the standard-library `errors` package required by the existing `errors.New(...)` validation paths. Positive-Schema exception state changes now use the unambiguous route `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`; the shipping Console uses the same path. The prior `POST /api/security/schema/enforcement/exceptions/{exception_id}` registration is removed because it conflicts with `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+ `http.ServeMux` specificity rules and can panic during Admin handler construction.

`admin_route_registration_test.go` now makes full Admin handler registration a regression test, and `tools/tests/test-build-startup-blocker-fix-source.py` is wired into `build.sh` and CI. The source gate also registers the complete 139-route Admin/update inventory with the Go standard library in a standalone temporary program; this catches wildcard-pattern ambiguity even when the repository module graph cannot be compiled locally. Current source evidence: blocker-fix gate **13/13 PASS**, route inventory **139/139 accepted by `ServeMux`**, API/source gates through API-8/post-audit/production-hardening/duplication remain PASS, OpenAI source contract and isolated tests PASS, package-source PASS, package-builder **9/9 PASS**, root Go source shape **96 files PASS**, Admin embedded JavaScript PASS, and changed Go files are `gofmt` clean.

This does **not** promote the source to `TESTED` or `RELEASED`. On the current execution host, canonical `GOTOOLCHAIN=local` Go 1.25 `go mod tidy -diff`, root build, vet, targeted Admin/API64 tests, and race remain `BLOCKED_ENVIRONMENT / NOT_RUN` because only Go 1.23.2 is installed and external Go toolchain/module retrieval is unavailable. Packaging/source-manifest integrity is not a substitute for those root compilation gates. No API-security authority changes are introduced: API-6/API-7 remain non-enforcing, API-8 retains explicit deterministic GraphQL ENFORCE authority, and OpenAI remains advisory only.

<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
