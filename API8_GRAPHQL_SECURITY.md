# API-8 — GraphQL Security

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

Status: `IMPLEMENTED_TESTING_DEFERRED`

API-8 closes the documented API-security implementation roadmap with deterministic GraphQL security while preserving the authority boundaries established by API-1 through API-7.

## Implementation map

- **API-8.1 — Parsing / normalization:** bounded GraphQL lexer/parser for GET, JSON POST, `application/graphql`, fragments, directives, variables, operation selection, structural operation fingerprints, depth, complexity and field-path analysis. Raw query text, string/number literals and raw variable values are not durable state.
- **API-8.2 — Schema contract:** bounded authenticated SDL import. Durable contracts retain only the SDL SHA-256 digest and normalized type/field topology. Explicit GraphQL policies may bind to an endpoint-scoped schema contract and deterministically emit `SCHEMA_FIELD_MISMATCH` evidence. Changing a bound contract returns dependent policies to `LEARN`; a referenced contract cannot be deleted until its policy reference is removed.
- **API-8.3 — Complexity / abuse controls:** absolute parser limits plus explicit per-policy depth, complexity, field allow/deny, mutation and subscription controls.
- **API-8.4 — Introspection / persisted queries:** explicit introspection policy and bounded Apollo-style persisted-query (APQ) support. Persisted state stores normalized structure keyed by SHA-256, never raw query text.
- **API-8.5 — Identity + BOLA integration:** GraphQL variables may become API-7.1 `graphql_variable` locators. Raw values are transient only and are converted to existing keyed object fingerprints; only API-5 cryptographically verified identity can enter API-7.2 relationship evidence. API-7.3 sees the keyed GraphQL object observation before current relationship merge. API-7 learned/inferred evidence remains non-enforcing.
- **API-8.6 — Enforcement + console:** Reviewer-gated deterministic policy lifecycle `LEARN -> DETECT -> ENFORCE`, strict bounded mutation APIs, durable violations/status, SDL/policy/operation/APQ/violation console, audit and immediate persistence. Any policy edit returns it to `LEARN`; `ENFORCE` promotion requires an explicit reason.

## Enforcement boundary

Only an explicit operator-reviewed GraphQL policy in `ENFORCE` may reject a GraphQL request. `LEARN` and `DETECT` never block. API-6 sequence evidence and API-7 BOLA evidence cannot promote, mutate or bypass GraphQL policy. OpenAI is not in the parser, policy-selection, schema-validation or enforcement authority path.

Deterministic violation classes include:

- `MALFORMED_GRAPHQL`
- `ABSOLUTE_DEPTH_LIMIT`
- `ABSOLUTE_COMPLEXITY_LIMIT`
- `DEPTH_LIMIT`
- `COMPLEXITY_LIMIT`
- `INTROSPECTION_DISABLED`
- `MUTATION_DISABLED`
- `SUBSCRIPTION_DISABLED`
- `PERSISTED_QUERY_REQUIRED`
- `SCHEMA_FIELD_MISMATCH`
- `FIELD_NOT_ALLOWED`
- `FIELD_DENIED`

## Privacy boundary

Durable GraphQL state does not contain raw GraphQL query documents, literal values, raw GraphQL variable values, Authorization/Cookie data, JWTs or unverified identity claims. SDL source is reduced to a digest plus normalized topology. Numeric JSON variables are decoded with `UseNumber()` before keyed fingerprinting so large integer identity/object values are not silently changed by float64 conversion.

## Bounds

- observation queue: 1,024
- operations: 4,096
- persisted queries: 4,096
- policies: 1,024
- schema contracts: 128
- violations: 4,096
- GraphQL request: 64 KiB
- GraphQL tokens: 8,192
- fields: 2,048
- variables: 256
- absolute depth: 64
- absolute complexity: 10,000
- policy field selectors: 512
- policy reason: 512 bytes
- SDL: 256 KiB / 32,768 tokens / 1,024 types / 8,192 fields
- operation and persisted-query TTL: 30 days

## Operational APIs

Reviewer-gated API-8 endpoints:

- `GET /api/security/graphql/operations`
- `GET /api/security/graphql/persisted-queries`
- `GET /api/security/graphql/schema-contracts`
- `POST /api/security/graphql/schema-contracts`
- `DELETE /api/security/graphql/schema-contracts/{contract_id}`
- `GET /api/security/graphql/policies`
- `POST /api/security/graphql/policies`
- `POST /api/security/graphql/policies/{policy_id}/mode`
- `DELETE /api/security/graphql/policies/{policy_id}`
- `GET /api/security/graphql/violations`
- `GET /api/security/graphql/status`

Mutation bodies are bounded and strict-decoded. Mutations persist immediately and are audited. GraphQL state is also included in API-security autosave/final flush and revalidated on restart.

## Verification status

Exact-source source/static evidence is PASS through API-8. API-8 has 22 targeted Go test functions present and a 259-check source gate. Canonical Go 1.25 `go mod tidy -diff`, root build/test, targeted API-8 tests and race tests remain `BLOCKED_ENVIRONMENT/NOT_RUN` because the host has Go 1.23.2, `go.mod` requires Go 1.25.0, and external toolchain retrieval is unavailable.

Therefore API-8 is not `TESTED` or `RELEASED`.

## 2026-09-28 maintenance addendum

The repository-wide duplicate-functionality cleanup does not change GraphQL
policy semantics, persistence, variable privacy, API-7 evidence integration, or
LEARN/DETECT/ENFORCE authority. API-8 remains `IMPLEMENTED_TESTING_DEFERRED`.

## 2026-09-30 — Build and Admin Startup Blocker Fix

The current source fixes two release-blocking defects found during external Go 1.25 review of the 2026-09-28 consolidation baseline. `admin.go` now imports the standard-library `errors` package required by the existing `errors.New(...)` validation paths. Positive-Schema exception state changes now use the unambiguous route `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`; the shipping Console uses the same path. The prior `POST /api/security/schema/enforcement/exceptions/{exception_id}` registration is removed because it conflicts with `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+ `http.ServeMux` specificity rules and can panic during Admin handler construction.

`admin_route_registration_test.go` now makes full Admin handler registration a regression test, and `tools/tests/test-build-startup-blocker-fix-source.py` is wired into `build.sh` and CI. The source gate also registers the complete 139-route Admin/update inventory with the Go standard library in a standalone temporary program; this catches wildcard-pattern ambiguity even when the repository module graph cannot be compiled locally. Current source evidence: blocker-fix gate **13/13 PASS**, route inventory **139/139 accepted by `ServeMux`**, API/source gates through API-8/post-audit/production-hardening/duplication remain PASS, OpenAI source contract and isolated tests PASS, package-source PASS, package-builder **9/9 PASS**, root Go source shape **96 files PASS**, Admin embedded JavaScript PASS, and changed Go files are `gofmt` clean.

This does **not** promote the source to `TESTED` or `RELEASED`. On the current execution host, canonical `GOTOOLCHAIN=local` Go 1.25 `go mod tidy -diff`, root build, vet, targeted Admin/API64 tests, and race remain `BLOCKED_ENVIRONMENT / NOT_RUN` because only Go 1.23.2 is installed and external Go toolchain/module retrieval is unavailable. Packaging/source-manifest integrity is not a substitute for those root compilation gates. No API-security authority changes are introduced: API-6/API-7 remain non-enforcing, API-8 retains explicit deterministic GraphQL ENFORCE authority, and OpenAI remains advisory only.

API contract note: this maintenance fix changes only the versionless Admin Console Positive-Schema exception toggle endpoint from `/api/security/schema/enforcement/exceptions/{exception_id}` to `/api/security/schema/enforcement/exceptions/{exception_id}/toggle`. Positive-Schema authority, RBAC, persistence, audit semantics, API-4 enforcement, and API-8 GraphQL authority are unchanged. No API-9 is defined.

<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
