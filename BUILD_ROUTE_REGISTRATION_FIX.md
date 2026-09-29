# Build and Admin Route Registration Fix — 2026-09-29

<!-- documentation-review: 2026-09-29; classification: current/canonical; precedence: current-source -->

## Status

**IMPLEMENTED_TESTING_DEFERRED**.

This micro-slice fixes two release-blocking defects found in the 2026-09-28 Code
Duplication Review and Consolidation artifact. It does not add a new API-security
slice and does not change API-4 enforcement authority.

## Fixes

1. `admin.go` now imports `errors`, matching the existing `errors.New(...)` uses
   in control-plane validation paths.
2. The positive-schema exception toggle route is now:
   `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`.
   The previous `POST /api/security/schema/enforcement/exceptions/{exception_id}`
   pattern overlapped with
   `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+
   `http.ServeMux` matching rules and could panic while `adminServer.handler()`
   registered routes.
3. The shipping Console uses the new `/toggle` route.
4. `TestAdminHandlerRouteRegistrationDoesNotPanic` explicitly protects handler
   construction.
5. The code-duplication source gate now checks not only exact METHOD+path
   duplicates but also simple single-segment ServeMux patterns whose match sets
   overlap while neither pattern is more specific. The gate also requires the
   new toggle route and forbids the ambiguous legacy route.

## Verification truth

Dependency-free source gates pass after the fix, including the updated code-
duplication gate at **83 checks / 139 routes**. `gofmt` passes for changed Go
files. The local execution environment still has Go 1.23.2 while `go.mod`
requires Go 1.25.0, so canonical `go build`, `go vet`, full `go test`, the
explicit handler-registration regression, and the API-6.4 targeted regression
remain **BLOCKED_ENVIRONMENT / NOT_RUN locally** before compilation.

The defect report supplied with this slice states that adding the missing import
made build and vet clean in the reporter's Go 1.25-capable environment. That is
recorded as external/user-supplied verification, not as local canonical PASS.

## Authority boundary

The route change is an administrative URI disambiguation only. Positive Schema
exception creation/toggle RBAC, audit and persistence semantics are unchanged.
API-6 and API-7 remain non-enforcing, API-8 retains explicit deterministic
GraphQL ENFORCE authority, and OpenAI remains advisory only.
