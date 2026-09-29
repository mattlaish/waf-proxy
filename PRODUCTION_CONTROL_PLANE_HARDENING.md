# Production Correctness & Control-Plane Hardening

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

Status: **IMPLEMENTED_TESTING_DEFERRED**  
Date: 2026-09-25  
Parent: `waf-proxy-api8-post-audit-hardening-2026-09-24.zip` (`e838ce6bd4f9cca6176cb809eb189978f312b194ae4b7c2f6f13f892a2d85f98`)

This hardening wave closes production correctness and control-plane findings discovered during the post-API-8 full-repository adversarial audit. It does not add API-9 and does not change API-6/API-7 non-enforcement authority.

## Implemented

- Block responses use `html/template`; request correlation IDs accept only a bounded safe character set. HTML block responses add no-store, nosniff, CSP and frame-deny headers.
- Admin JSON responses use no-store/nosniff controls. Notification webhook URLs remain write-only and are redacted from config reads; a separate operator-only clear action is provided.
- Debug evidence masks nested remote/resolved client IPs.
- Login abuse protection is bounded and fail-closed under key saturation; session creation fails closed on CSPRNG error; PBKDF2 parameters are bounded before work begins.
- Mutating admin routes are role-gated and audited. Admin audit records are durably appended to a mode-0600 JSONL log with bounded rotation.
- HA config replication uses a dedicated peer-only HTTPS endpoint and stable replication token. Generic config APIs no longer trust caller-supplied sync headers. Shared config strips node-local HA identity, users and secrets; the receiver restores its own local state. Outbound sync is bounded to one in-flight request plus a latest-wins pending slot.
- Full config mutation is serialized. Config is staged with a same-directory unique temp file, mode 0600 and file fsync; live network/runtime application is checked before atomic rename, and pre-rename commit failure rolls live state back. Directory fsync is performed after rename.
- Listener reconciliation synchronously binds required sockets and rolls back failed protocol transitions. External TLS frontend -> Go TLS waits for frontend stop before public bind. Go TLS -> external frontend now synchronously releases the public listening socket before frontend activation, removing the EADDRINUSE ownership race.
- Sitemap clear persists the cleared snapshot before dependent learning state is discarded and restores the previous snapshot on persistence failure.
- Early API-security stores reject future/unknown durable state versions.
- Notification webhook delivery uses a bounded queue and workers, bounded retries, non-2xx checking and queue-drop telemetry.
- L7 limiter saturation uses a bounded conservative overflow bucket rather than allowing unseen identities.
- Admin file browsing resolves symlinks before enforcing the browse-root boundary.
- In-process updater writes are disabled by default for packaged/systemd deployment. A standalone writable install requires explicit `WAF_UPDATE_INSTALL_DIR`; packaged installations use the OS package manager.
- systemd units explicitly declare device access for hardware watchdog and QAT-related device nodes instead of combining those capabilities with `PrivateDevices=true`.
- Console actions are capability-aware and `/api/learn/clear` is exposed to reviewers. OpenAPI import and webhook-secret mutation remain operator-only.
- Main config sample/documentation includes the post-audit CIDR/L7 controls.

## Validation truth

Static/source gates and artifact gates can validate source structure, security contracts, syntax, packaging and clean extraction. Canonical Go 1.25 compile/test/race remains **BLOCKED_ENVIRONMENT / NOT_RUN** on this host because the installed Go is 1.23.2, `go.mod` requires 1.25.0, and external toolchain retrieval is unavailable. This baseline must not be promoted to TESTED or RELEASED on source-gate evidence alone.

The supplied source archive has no `.git` metadata. Therefore branch creation, remote/main synchronization, commit identity and fresh-clone verification cannot be performed from this artifact. Parent ZIP SHA-256 and package manifests are the source identity for this handoff.

## 2026-09-28 source-topology addendum

The later Code Duplication Review and Consolidation removes superseded/unwired
source layers and centralizes shared SecLang parsing. It does not roll back or
replace the production-correctness behavior documented here. This file remains
the authority for the 2026-09-25 control-plane hardening behavior;
`CODE_DUPLICATION_REVIEW.md` is the authority for the later source-topology
cleanup.

## 2026-09-30 — Build and Admin Startup Blocker Fix

The current source fixes two release-blocking defects found during external Go 1.25 review of the 2026-09-28 consolidation baseline. `admin.go` now imports the standard-library `errors` package required by the existing `errors.New(...)` validation paths. Positive-Schema exception state changes now use the unambiguous route `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`; the shipping Console uses the same path. The prior `POST /api/security/schema/enforcement/exceptions/{exception_id}` registration is removed because it conflicts with `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+ `http.ServeMux` specificity rules and can panic during Admin handler construction.

`admin_route_registration_test.go` now makes full Admin handler registration a regression test, and `tools/tests/test-build-startup-blocker-fix-source.py` is wired into `build.sh` and CI. The source gate also registers the complete 139-route Admin/update inventory with the Go standard library in a standalone temporary program; this catches wildcard-pattern ambiguity even when the repository module graph cannot be compiled locally. Current source evidence: blocker-fix gate **13/13 PASS**, route inventory **139/139 accepted by `ServeMux`**, API/source gates through API-8/post-audit/production-hardening/duplication remain PASS, OpenAI source contract and isolated tests PASS, package-source PASS, package-builder **9/9 PASS**, root Go source shape **96 files PASS**, Admin embedded JavaScript PASS, and changed Go files are `gofmt` clean.

This does **not** promote the source to `TESTED` or `RELEASED`. On the current execution host, canonical `GOTOOLCHAIN=local` Go 1.25 `go mod tidy -diff`, root build, vet, targeted Admin/API64 tests, and race remain `BLOCKED_ENVIRONMENT / NOT_RUN` because only Go 1.23.2 is installed and external Go toolchain/module retrieval is unavailable. Packaging/source-manifest integrity is not a substitute for those root compilation gates. No API-security authority changes are introduced: API-6/API-7 remain non-enforcing, API-8 retains explicit deterministic GraphQL ENFORCE authority, and OpenAI remains advisory only.

<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
