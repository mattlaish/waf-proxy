# API-8 Source Gate Result

Status: **PASS — 259 checks**

Implementation state: `IMPLEMENTED_TESTING_DEFERRED`

`tools/tests/test-api8-source.py` validates the exact API-8 source architecture without substituting for canonical Go 1.25 execution. Coverage includes:

- bounded GraphQL request capture, lexer/parser and normalized operation identity;
- privacy-preserving query/literal/variable handling;
- fragment expansion/cycle rejection and operation selection;
- depth, complexity, field and variable resource limits;
- Apollo persisted-query hash validation, bounded structural registry and no raw-query persistence;
- bounded SDL import and normalized schema topology;
- endpoint-scoped `schema_contract_id` policy binding and `SCHEMA_FIELD_MISMATCH` evaluation;
- dependent-policy fallback to `LEARN` when a bound contract changes;
- referenced-contract deletion rejection;
- deterministic `LEARN -> DETECT -> ENFORCE` lifecycle and required ENFORCE promotion reason;
- policy edits returning to `LEARN`;
- mutation/subscription/introspection/persisted-query/field policy controls;
- deterministic malformed-endpoint fallback selection;
- bounded, revalidated durable violation evidence;
- `json.Decoder.UseNumber()` precision protection for numeric GraphQL object values;
- GraphQL variable integration into API-7.1 keyed locator evidence and API-7.2/API-7.3 historical relationship/detection ordering;
- verified API-5 identity boundary for relationship evidence;
- absence of raw JWT/Cookie/Authorization/object/query authority in durable GraphQL state;
- absence of API-6/API-7/OpenAI authority over GraphQL ENFORCE decisions;
- Reviewer RBAC, audit, strict mutation handling, persistence and Admin Console surfaces;
- build/CI retention of historical API gates and the API-8 gate;
- 22 targeted API-8 Go test functions present.

Exact-source compatibility gates retained:

- API-1/API-2: 69 PASS
- API-3: 47 PASS
- API-4: 46 PASS
- API-5: 72 PASS
- API-6.1: 33 PASS
- API-6.2: 56 PASS
- API-6.3: 45 PASS
- API-6.4: 58 PASS
- API-7.1: 83 PASS
- API-7.2: 100 PASS
- API-7.3: 110 PASS
- API-7.4: 156 PASS
- API-8: 259 PASS

Other exact-source gates: OpenAI source contract 16/16 plus isolated tests PASS; package-source PASS; package-builder 9/9 PASS; root Go source shape PASS; changed API-8 Go files gofmt-clean; embedded Admin JavaScript and relevant shell/Python syntax PASS.

## Deferred canonical qualification

The local toolchain is Go 1.23.2 while `go.mod` requires Go 1.25.0. `GOTOOLCHAIN=local` fails closed before compilation, and external Go 1.25 retrieval is unavailable. Consequently the following remain `BLOCKED_ENVIRONMENT/NOT_RUN`:

- Go 1.25 `go mod tidy -diff`;
- repository root Go 1.25 build/test/vet gates;
- API-8 targeted Go tests;
- API-8 race tests;
- downstream real-dependency/production qualification.

Source/static PASS does not promote API-8 to `TESTED` or `RELEASED`.
