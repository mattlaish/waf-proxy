# API Security Roadmap — API-aware WAF Evolution

## Purpose

This document defines the future API-aware WAF product line built on the existing
WAF foundation.

The goal is not to replace Coraza/CRS. Coraza, CRS, VectorScan and future CPU
acceleration remain the high-speed dataplane. API Intelligence adds positive
security and application-behavior understanding.

## Existing Foundation (Implemented)

The current baseline already contains:

- passive request observation
- site/path discovery foundation
- crawler-based discovery foundation
- request shape signals
- field discovery
- PagePolicy / FieldPolicy positive validation
- profile suggestion
- OpenAI-assisted profile review

These capabilities are foundation only. They do not represent completion of the
API Security slices below.

## Roadmap Status

All API Security slices are currently:

`PLANNED`

No slice below is implemented unless explicitly promoted by a future release.

---

# API-1 — API Discovery + Operation Normalization

## Goal

Build a canonical API inventory from observed traffic.

## Scope

Implement:

- API operation entity
- HTTP method tracking
- normalized route templates
- path parameter classification
- endpoint lifecycle tracking
- API inventory UI/API

Examples:

```
/api/users/1001
/api/users/1002

becomes

GET /api/users/{id}
```

## Data Model

Planned:

- APIOperation
- PathParameter
- OperationObservation

## Security Boundary

Detection and inventory only.

No blocking.

## Acceptance Criteria

- repeated dynamic paths normalize correctly
- sensitive values are not stored
- tenant/site isolation is preserved

---

# API-2 — Typed Schema Learning

## Goal

Convert observed request traffic into candidate API schemas.

## Scope

Learn:

- JSON fields
- query parameters
- headers
- form fields
- arrays
- nested objects

Generate candidates:

```yaml
amount:
  type: number

currency:
  enum:
    - USD
    - TWD

user_id:
  format: uuid
```

## Data Model

Planned:

- SchemaObservation
- APISchemaCandidate
- APISchemaField
- SchemaVersion

## Security Boundary

Learning only.

No automatic enforcement.

---

# API-3 — OpenAPI Contract Management

## Goal

Connect declared API contracts with observed behavior.

## Scope

Implement:

- OpenAPI 3.x import
- OpenAPI export
- contract comparison
- schema drift detection

Detect:

- undocumented endpoints
- type drift
- enum drift
- missing fields
- auth drift

---

# API-4 — Positive Schema Enforcement

## Goal

Move approved API profiles into enforcement.

Modes:

```
LEARN
DETECT
ENFORCE
```

Support:

- schema versioning
- exceptions
- rollback
- shadow mode
- violation evidence

---

# API-5 — JWT + Identity-aware API Security

## Goal

Add verified identity context.

Scope:

- JWT signature validation
- JWKS
- issuer/audience validation
- expiry validation
- verified claims context

Claims may become policy input only after cryptographic validation.

---

# API-6 — Sequence Analytics

## Goal

Understand API workflows.

Examples:

```
login
 |
account
 |
checkout
 |
payment
```

Implement:

- sequence observation
- transition model
- anomaly detection

Initial mode:

- learn
- detect

---

# API-7 — BOLA Analytics

## Goal

Detect suspicious object-level authorization patterns.

Scope:

- identity/object relationship analysis
- tenant boundary signals
- enumeration detection
- cross-subject access candidates

Initial result:

```
BOLA_CANDIDATE
```

not automatic blocking.

---

# API-8 — GraphQL Security

## Goal

Extend API intelligence to GraphQL.

Scope:

- operation discovery
- query depth
- complexity analysis
- field policy
- mutation controls
- introspection policy
- persisted query support

---

# OpenAI Role

OpenAI is an analysis assistant, not an enforcement authority.

Allowed:

- schema explanation
- candidate review
- anomaly explanation
- operator assistance

Not allowed:

- autonomous policy mutation
- direct blocking authority
- bypass of deterministic validation

---

# Relationship With Dataplane

```
API Intelligence
        |
        v
Positive Security Decisions
        |
        v
Coraza / Runtime Enforcement

Coraza / CRS / VectorScan
        |
        v
High-speed detection path
```

## API-2 Typed Schema Learning

Status: IMPLEMENTATION_IN_PROGRESS

Foundation added for schema candidate workflows. Runtime qualification remains NOT RUN.


## Slice Implementation Blueprint

See `API_SECURITY_SLICE_IMPLEMENTATION_ROADMAP.md` for detailed slice-by-slice implementation scope.

Current sequence:
API-1 → API-2 → API-3 → API-4/API-5 → API-6/API-7 → API-8
