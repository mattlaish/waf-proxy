# API-7.1 — Object Locator Discovery

Status: `IMPLEMENTED_TESTING_DEFERRED`

## Purpose

API-7.1 discovers **where an object selector exists** for a normalized API operation. It does not decide who owns the object, whether the caller may access it, whether a tenant boundary was crossed, or whether a request is a BOLA attack. Those relationship and anomaly decisions remain API-7.2/API-7.3 work.

## Discovery sources

`ObjectLocator` merges evidence from four bounded sources:

- **API-1 normalized path parameters** — identifier/opaque dynamic path segments are attached to the normalized API-1 operation ID, so `/orders/1` and `/orders/999999` do not create separate locator nodes.
- **API-2 typed schema observations** — identifier-like query and JSON/body fields are discovered only from bounded typed evidence. Header values are excluded.
- **API-3 matched OpenAPI contracts** — only `MATCHED` contract-operation bindings may seed declared path/query/request-body locators. Contract header/cookie parameters are excluded.
- **Operator configuration** — Reviewer-or-higher operators may explicitly `INCLUDE` a locator or `SUPPRESS` a false positive. These controls define locator position only, never ownership or authorization.

GraphQL `variables.*` fields are deliberately deferred to API-8 so REST object-locator heuristics do not silently become GraphQL policy.

## Durable model

Each locator contains at least:

- `operation_id`
- `location` (`path`, `query`, or `body`)
- `field`
- `schema_type`
- `semantic_name`
- `confidence`
- `first_seen`
- `last_seen`

It also carries bounded source metadata, observation count, status, expiry, and up to eight keyed value fingerprints.

## Privacy and trust boundary

Raw object values are transient only. They are never exported by the schema sample type and are never written to `api-object-locators.json`. Value evidence is HMAC-SHA256 keyed with a separate `api-object-locator.key` file created mode `0600`; only a bounded 20-byte digest is retained.

Client-supplied ownership/tenant headers are not locator authority. API-7.1 ignores header/cookie fields entirely. API-5 verified identity is not yet correlated with objects in this slice.

## Resource and restart hardening

- 4,096 total locators maximum.
- 32 locators maximum per normalized operation.
- 8 keyed value fingerprints maximum per locator.
- 1,024 operator overrides maximum.
- Learned locator TTL: 30 days.
- Discovery runs in the existing bounded non-blocking observation plane; request traffic never waits on the locator store.
- Durable state is versioned and revalidated on restore.
- OpenAPI-declared sources are refreshed after contract matching and at startup.
- Operator mutations persist immediately and are audited.

## Admin/API

Reviewer-or-higher endpoints:

- `GET /api/security/object-locators`
- `GET /api/security/object-locators/overrides`
- `POST /api/security/object-locators/overrides`
- `DELETE /api/security/object-locators/overrides/{override_id}`

The embedded Admin Console exposes the normalized locator inventory and INCLUDE/SUPPRESS controls while explicitly documenting that API-7.1 has no ownership, BOLA, or blocking authority.

## Authority boundary

API-7.1 adds **no** `BOLA_CANDIDATE`, owner-match/owner-mismatch verdict, request-path `403`, `BLOCK`, `DENY`, or `ENFORCE` primitive. OpenAI is absent from the locator discovery/authorization path. Coraza/CRS, API-4 explicit positive-schema policy, and API-5 explicit verified identity policy retain their existing independent authority boundaries.

## Qualification status

The dedicated API-7.1 source gate and retained API-1→API-6.4 compatibility/source gates pass on the exact working tree. Canonical Go 1.25 targeted and race execution is `BLOCKED_ENVIRONMENT / NOT_RUN` because this host provides Go 1.23.2 while `go.mod` requires Go 1.25.0 and external toolchain acquisition is unavailable. This slice is therefore not `TESTED` or `RELEASED`.

## Next slice

`API-7.2 — Identity/Object Relationship` is the next `PLANNED` slice. API-7.3 BOLA Detection, API-7.4 BOLA Policy/Evidence/Console, and API-8 remain `PLANNED`.
