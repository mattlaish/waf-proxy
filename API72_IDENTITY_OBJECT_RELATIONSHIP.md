# API-7.2 — Identity/Object Relationship

Status: `IMPLEMENTED_TESTING_DEFERRED`

## Purpose

API-7.2 correlates API-5 cryptographically verified identity context with API-7.1 keyed object-selector evidence. It creates bounded relationship evidence for later API-7.3 BOLA analytics without deciding ownership, tenant authorization, or request blocking.

## Durable relationship model

`IdentityObjectRelationship` retains only pseudonymous evidence:

- API-1 normalized `operation_id`
- API-7.1 `locator_id`, location, and field
- `identity_kind` (`subject` or `client`)
- keyed `identity_fingerprint`
- optional keyed `tenant_fingerprint`
- optional keyed `client_fingerprint`
- API-7.1 keyed `object_fingerprint`
- observational evidence level (`OBSERVED` or `REPEATED`)
- observation count and bounded timestamps/expiry
- evidence sources `API5_VERIFIED_IDENTITY` and `API71_KEYED_OBJECT`

No raw JWT, Authorization header, subject, tenant claim, client_id, cookie, object value, owner value, or caller-supplied tenant/ownership header is durable relationship state.

## Identity authority

Only `VerifiedAPIIdentity` produced by API-5 after cryptographic JWT verification is accepted. Requests without verified identity context do not create API-7.2 relationship observations. Subject is preferred as the identity dimension; verified client_id is used only when subject is absent.

API-7.2 has its own 32-byte `api-object-relationship.key`, persisted as a regular mode-0600 file. Identity/tenant/client pseudonyms are HMAC-SHA256 digests truncated to 20 bytes and hex encoded. Raw verified claims remain request-memory input only.

## Object authority

A relationship is created only when the observed selector position resolves to an ACTIVE API-7.1 locator. SUPPRESS overrides therefore prevent API-7.2 relationship creation. Object values are never copied into the relationship model; they are converted through the existing API-7.1 keyed `digestValue` primitive.

GraphQL `variables.*` selectors remain deferred to API-8. Headers and cookies are not object-selector authority for API-7.2.

## Async hot-path design

The API-7.2 wrapper runs after bounded request-body capture and downstream of API-5 identity verification. The request path creates only a compact bounded observation and performs a non-blocking queue send. Queue saturation increments a drop counter instead of applying backpressure or altering the WAF verdict.

The background worker parses bounded path/query/body evidence, validates the active API-7.1 locator, derives the keyed object fingerprint, and mutates relationship state.

## Resource limits

- relationship TTL: 30 days
- total relationships: 8,192
- relationships per verified identity pseudonym: 512
- verified identities per keyed object/locator pair: 256
- async queue: 2,048 observations
- raw path queued per observation: maximum 4,096 bytes
- raw query queued per observation: maximum 8,192 bytes
- body prefix: existing bounded API schema capture limit

Expired evidence is pruned. The same limits are reapplied when persisted state is restored.

## Persistence and restart

Relationship state is stored in `api-object-relationships.json`, schema version 1. Restore rejects unknown versions and revalidates every row, digest shape, TTL, source set, and cardinality bound. The API autosave/final-flush path includes API-7.2 state, and shutdown drains accepted relationship observations before final persistence.

## Operator visibility

Reviewer-or-higher read-only endpoints:

- `GET /api/security/object-relationships`
- `GET /api/security/object-relationships/status`

Both reads are audited. API-7.2 deliberately exposes no POST/DELETE ownership mutation endpoint. The Admin Console shows only pseudonymous relationship evidence and queue/resource status.

## Authority boundary

API-7.2 is relationship learning/visibility only. It does **not** emit `BOLA_CANDIDATE`, `OWNER_MATCH`, `OWNER_MISMATCH`, tenant-boundary verdicts, `BLOCK`, `DENY`, `403`, or `ENFORCE`. Repeated observation means only that the same verified pseudonym/object fingerprint pair was observed again; it is not proof of ownership or authorization.

OpenAI is absent from API-7.2 relationship authority.

## Testing status

Exact-source and clean-extract source/static gates are used for this slice. Canonical Go 1.25 targeted/race execution is `BLOCKED_ENVIRONMENT / NOT_RUN` on the current host because installed Go is 1.23.2 and `go.mod` requires Go 1.25.0. External toolchain retrieval is unavailable. This limitation prevents promotion to `TESTED` or `RELEASED`.

Next slice: **API-7.3 — BOLA Detection**.
