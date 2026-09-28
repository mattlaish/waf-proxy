# API-3 Test Matrix

## 2026-09-23 executed source/isolated evidence

PASS: API-3 source gate (42 checks), isolated JSON parse/export/diff/drift harness (2/2), compile-only validation of the actual `api3_contract_test.go` under a minimal YAML shim, and API-1→API-2→API-3 live-learning/drift integration. Added coverage scope includes export round-trip, persistence, parameters, enums, security requirements, response diffs, undeclared/missing endpoints, content-type drift and authentication-family drift. Real goccy/go-yaml runtime plus repository-root Go 1.25 tests remain BLOCKED/NOT_RUN.


Status: **PREPARED — Go execution deferred to final qualification gate**

## Parser and validation

| Case | Expected |
|---|---|
| OpenAPI 3.0 YAML | Import succeeds |
| OpenAPI 3.1 YAML/JSON | Import succeeds |
| Local component `$ref` | Resolves with recursion bound |
| Missing `info` / `paths` | Reject |
| Unsupported OpenAPI version | Reject |
| Malformed YAML/JSON | Reject |
| Cyclic/invalid local reference | Fail closed for that reference/import path |

## Import/versioning

| Case | Expected |
|---|---|
| First document import | Contract + immutable version created |
| Same bytes replayed | Same version ID; no duplicate version |
| Changed bytes | New content-hash version |
| Request larger than 8 MiB | HTTP import rejected |

## Operation matching

| Case | Expected |
|---|---|
| Same method + normalized path + fingerprint | `MATCHED`, high confidence |
| Method mismatch | No binding candidate |
| Multiple equivalent candidates | `AMBIGUOUS` |
| No candidate | `UNMATCHED` |

## Contract version comparison

| Case | Expected severity |
|---|---|
| Operation added | INFO |
| Operation removed | BREAKING |
| Optional field added | INFO |
| Required field added | BREAKING |
| Field removed | BREAKING |
| `integer` widened to `number` | WARNING |
| `number` changed to `string` | BREAKING |
| Format changed | WARNING |

## API-2 observed-schema drift

| Case | Expected |
|---|---|
| Declared and observed type compatible | No type drift |
| Declared `number`, observed `string` | BREAKING drift evidence |
| Undeclared observed field | WARNING |
| Required declared field not observed | WARNING |
| No API-2 candidate | No fabricated drift result |

## Security/source boundary

- Contract routes are registered exactly once.
- API operation and schema stores are initialized.
- Production observation plane feeds API-1 operation telemetry.
- `goccy/go-yaml` is a direct module dependency.
- API-3 source contains no block-list, engine-mode, or page-policy activation path.
- Go root build/test/race/real-Coraza remain deferred to final qualification.


## 2026-09-23 final closure refresh

API-3 remains `IMPLEMENTED_TESTING_DEFERRED`. Closure hardening adds explicit external-`$ref` rejection, duplicate-normalized-operation rejection, multiple declared content types, OpenAPI security OR/AND/anonymous-alternative semantics, and matched site/host scoping for `UNDECLARED_ENDPOINT`. Current source gate is **47 checks PASS**. Exact-source compile-only PASSes with a minimal YAML shim, and four deterministic runtime/integration tests that do not require YAML behavior PASS. Real `goccy/go-yaml` runtime and repository-root Go 1.25 qualification remain BLOCKED/NOT_RUN on this host.

<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
