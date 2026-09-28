# API-3 OpenAPI Contract Management Implementation

## 2026-09-23 closure update

API-3 now additionally includes durable `api-contracts.json` state, OpenAPI JSON/YAML export, path/query/header/cookie parameters, enums, security scheme definitions/requirements, request+response/parameter/security diffs, live rebinding against API-1 inventory, and drift types `MISSING_ENDPOINT`, `UNDECLARED_ENDPOINT`, `UNKNOWN_FIELD`, `MISSING_REQUIRED_FIELD`, `TYPE_DRIFT`, `ENUM_DRIFT`, `CONTENT_TYPE_DRIFT`, and `AUTH_DRIFT`. A deterministic API-1→API-2→API-3 live-learning/drift integration test passes in the isolated harness. Root Go 1.25 qualification remains BLOCKED/NOT_RUN.


Status: **IMPLEMENTED_TESTING_DEFERRED**

API-3 adds declared-contract intelligence on top of API-1 operation identity and API-2 learned schema metadata. It is intentionally non-enforcing: no request blocking, automatic policy activation, or autonomous AI decision is introduced by this slice.

## Implemented source

- OpenAPI 3.0.x / 3.1.x YAML-or-JSON import.
- Validation of `openapi`, `info.title`, `info.version`, and `paths`.
- Local `#/...` reference resolution with recursion bounds.
- Request/response schema flattening for objects, arrays, primitive fields, `required`, `format`, and `allOf`.
- Immutable content-hash contract versions with replay idempotency.
- Contract inventory, detail, version history, and imported operation inventory.
- API-1 operation binding using method, normalized path, and operation fingerprint evidence.
- Ambiguous/unmatched binding states; no arbitrary auto-selection of ambiguous matches.
- Contract version comparison for operation/field/type/format/required changes.
- API-2 learned-schema drift reporting for type mismatch, undeclared fields, and required declared fields not yet observed.
- Production observation-plane wiring so API-1 operation inventory receives asynchronous request telemetry.

## Admin API

- `GET /api/security/contracts`
- `POST /api/security/contracts/import`
- `GET /api/security/contracts/{id}`
- `GET /api/security/contracts/{id}/versions`
- `POST /api/security/contracts/{id}/match`
- `GET /api/security/contracts/{id}/bindings`
- `POST /api/security/contracts/{id}/compare`
- `GET /api/security/contracts/{id}/diffs`
- `GET /api/security/contracts/{id}/drift`

Import requests are capped at 8 MiB at the HTTP boundary. Source imports are metadata/contract documents; API-3 does not persist runtime request bodies, cookies, authorization headers, JWTs, or credentials.

## Change severity model

- `INFO`: additive/non-breaking declared contract change.
- `WARNING`: compatible widening, format difference, or observational mismatch that needs review.
- `BREAKING`: operation removal, incompatible type change, required-field addition, or field removal.

Observed-vs-declared drift is evidence only. A `BREAKING` label does **not** block traffic in API-3.

## Security boundaries

- Coraza remains the request enforcement authority.
- API-3 never modifies `EngineMode`, page policy, block lists, or enforcement state.
- OpenAPI import cannot execute remote references; only local `#/...` references are resolved in this slice.
- URL/repository OpenAPI retrieval is deferred; imports are supplied directly to the authenticated admin API.
- Binding confidence is deterministic and evidence-based; ambiguous matches remain `AMBIGUOUS`.

## Verification status

Implemented tests are in `api3_contract_test.go`. The executable non-Go source gate is `tools/tests/test-api3-source.py`.

Per project instruction, Go 1.25 build/test execution is deferred to the final project qualification gate. Therefore API-3 remains `IMPLEMENTED_TESTING_DEFERRED` and must not be described as `TESTED` or `RELEASED`.

See `API3_TEST_MATRIX.md` and `API3_ACCEPTANCE_CRITERIA.md`.


## 2026-09-23 final closure refresh

API-3 remains `IMPLEMENTED_TESTING_DEFERRED`. Closure hardening adds explicit external-`$ref` rejection, duplicate-normalized-operation rejection, multiple declared content types, OpenAPI security OR/AND/anonymous-alternative semantics, and matched site/host scoping for `UNDECLARED_ENDPOINT`. Current source gate is **47 checks PASS**. Exact-source compile-only PASSes with a minimal YAML shim, and four deterministic runtime/integration tests that do not require YAML behavior PASS. Real `goccy/go-yaml` runtime and repository-root Go 1.25 qualification remain BLOCKED/NOT_RUN on this host.
