# API-3 Acceptance Criteria

## 2026-09-23 closure scope additions

The implemented acceptance surface now also requires durable contract restart recovery, OpenAPI JSON/YAML export, parameter/enum/security requirement preservation, request/response/parameter/security diffs, current-live API-1 rebinding, and end-to-end drift from API-2 live learned candidates. Isolated/source evidence is PASS; promotion to `TESTED` still requires the pinned Go 1.25 root qualification listed below.


Current implementation state: **IMPLEMENTED_TESTING_DEFERRED**

API-3 may be promoted to `TESTED` only when all criteria below are executed successfully against the exact delivered source bytes.

## Source/API criteria

- OpenAPI 3.0.x and 3.1.x import works for JSON and YAML inputs.
- Invalid/unsupported documents fail closed without mutating prior contract state.
- Same-document replay is idempotent; changed content creates a new immutable version.
- API-1 operation bindings are deterministic and ambiguity is surfaced rather than silently selected.
- Version diff classifications match `API3_TEST_MATRIX.md`.
- API-2 drift reporting uses metadata only and does not require raw request values.
- All API-3 endpoints require the existing admin authentication/RBAC wrappers.

## Security criteria

- No request enforcement is introduced by API-3.
- No remote `$ref` fetch occurs.
- No credentials, JWTs, cookies, or runtime request body values are persisted by the contract layer.
- Import size is bounded.

## Final Go qualification criteria — deferred

Run on the pinned Go 1.25 toolchain:

```bash
GOTOOLCHAIN=local go mod tidy -diff
GOTOOLCHAIN=local CGO_ENABLED=0 go build ./...
GOTOOLCHAIN=local CGO_ENABLED=0 go vet ./...
GOTOOLCHAIN=local CGO_ENABLED=0 go test ./...
GOTOOLCHAIN=local CGO_ENABLED=1 go test -race ./...
GOTOOLCHAIN=local CGO_ENABLED=1 go test -tags realcoraza -run 'TestRealCoraza' ./...
```

Until those gates and API-3 tests execute successfully, status remains `IMPLEMENTED_TESTING_DEFERRED`.


## 2026-09-23 final closure refresh

API-3 remains `IMPLEMENTED_TESTING_DEFERRED`. Closure hardening adds explicit external-`$ref` rejection, duplicate-normalized-operation rejection, multiple declared content types, OpenAPI security OR/AND/anonymous-alternative semantics, and matched site/host scoping for `UNDECLARED_ENDPOINT`. Current source gate is **47 checks PASS**. Exact-source compile-only PASSes with a minimal YAML shim, and four deterministic runtime/integration tests that do not require YAML behavior PASS. Real `goccy/go-yaml` runtime and repository-root Go 1.25 qualification remain BLOCKED/NOT_RUN on this host.
