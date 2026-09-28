# API-8 Post-Audit Hardening

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

This hardening pass follows the API-8 GraphQL Security implementation. It does not define API-9 and does not change the API-6/API-7/API-8 authority model.

## Runtime correctness

- CIDR policy now honors `cidr_policy.enabled` and parses `expires_at` as RFC3339 into the runtime rule. Expired rules remain valid configuration but are skipped at evaluation time; malformed expiry is rejected during configuration validation.
- `l7_abuse.tls_handshake_per_window` is now an implemented bounded control on the built-in Go TLS ClientHello path. Buckets are keyed by direct TCP peer and public listener, use the existing bounded sharded state discipline, and execute before certificate/signature work.
- TLS handshake limiting fails closed at configuration validation when the external TLS frontend is selected, because those handshakes do not enter the Go TLS ClientHello path. HTTP request/concurrency abuse controls remain unchanged.

## Console exposure closure

The shipping `static/admin.html` now exposes operator-visible backend capability that already existed but was not surfaced:

- SYSTEM tab: Doctor/runtime truth, bounded debug capture/evidence/export, HSM status/audit and module-directory policy, VectorScan state/groups/reset.
- POLICIES traffic controls: L7 request/concurrency/TLS-handshake limits plus CIDR allow/deny rules, priorities, reasons and RFC3339 expiry.
- SITES: per-site PKCS#11 provider/module/slot/token/key selectors and secret-reference-only PIN configuration. Redacted secret references remain preserved server-side when the operator leaves the field blank.
- OpenAPI contracts: detail, immutable versions, operation match/bindings, version compare/diffs, learned-schema drift refresh and JSON/YAML export.
- Positive Schema: candidate return-to-candidate, deployment detail, profile-version activation, exception expiry and enable/disable lifecycle.
- API Security opaque identifiers: Sequence workflow, BOLA locator and GraphQL SDL contract selectors are inventory-backed selects and carry their associated site/operation/endpoint scope automatically.

Backend RBAC remains authoritative. The Console may render controls that a lower role cannot invoke; forbidden operations remain rejected server-side.

## Source/debt cleanup

The obsolete non-shipping `web/` Preact migration tree was removed. `static/admin.html` plus `static/theme.css` is now the single authoritative Console source.

The following model-only files were removed because they had no runtime store/API/Console wiring and could be mistaken for implemented Security Operations functionality:

- `investigation.go`
- `security_timeline.go`
- `change_audit.go`
- `security_export.go`
- `debug_lifecycle_v2.go`
- `debug_retention_worker_v2.go`

The real bounded debug implementation remains `debug_bundle.go`, shared `debug_sanitize.go`, `support_api.go`, CLI integration and the SYSTEM Console. The superseded `debug_evidence.go` compatibility layer was later removed by the 2026-09-28 duplication consolidation. Phase 5 Slice D is therefore corrected to PLANNED / NOT IMPLEMENTED rather than claiming model-only structs as implementation.

## Authority boundary

- Coraza/CRS remains the deterministic WAF authority.
- API-4 explicit Positive Schema policy may enforce only through its explicit staged policy path.
- API-5 uses only cryptographically verified identity for identity policy.
- API-6 learned sequence evidence does not block.
- API-7 inferred BOLA evidence does not block.
- API-8 GraphQL blocks only under explicit deterministic Reviewer-gated ENFORCE policy.
- OpenAI has no request-path enforcement authority.
- VectorScan remains an accelerator and Coraza remains authoritative.

## Qualification truth

Dependency-free source/static/package gates pass on the exact hardening source. Canonical Go 1.25 `tidy/build/test/race` remains **BLOCKED_ENVIRONMENT / NOT_RUN** on the current host because only Go 1.23.2 is locally installed and external toolchain retrieval is unavailable. This hardening must not be promoted to TESTED or RELEASED from source/static evidence alone.

## 2026-09-28 maintenance addendum

The later duplication consolidation removes additional legacy/unwired helper
layers identified by a whole-repository liveness/overlap review. The API-8
post-audit functional behavior remains in force. Current debug sanitization is
centralized in `debug_sanitize.go`; the old `debug_evidence.go` compatibility
layer is no longer present.

<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
