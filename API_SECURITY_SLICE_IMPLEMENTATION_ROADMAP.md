# API Security Slice Implementation Roadmap

## Purpose

This document is the canonical implementation blueprint for API-aware WAF security evolution.

Each slice defines scope, architecture, components, APIs, tests, dependencies, and security boundaries.

## Slice Status

| Slice | Name | Status |
|---|---|---|
| API-1 | API Discovery + Operation Normalization | IMPLEMENTED |
| API-2 | Typed Schema Learning | IMPLEMENTATION COMPLETE / QUALIFICATION NOT RUN |
| API-3 | OpenAPI Contract Management | PLANNED |
| API-4 | Positive Schema Enforcement | PLANNED |
| API-5 | JWT + Identity-aware API Security | PLANNED |
| API-6 | Sequence Analytics | PLANNED |
| API-7 | BOLA Analytics | PLANNED |
| API-8 | GraphQL Security | PLANNED |

## API-1 — API Discovery + Operation Normalization

Goal:
- discover API operations
- normalize paths
- maintain stable operation identity

Components:
- operation discovery
- normalization engine
- fingerprinting
- operation inventory

Tests:
- normalization corpus
- fingerprint stability
- operation persistence

## API-2 — Typed Schema Learning

Goal:
Learn application request schema characteristics from observations without storing sensitive values.

Components:
- schema observation collector
- recursive type inference
- confidence calculator
- enum detection
- required-field detection
- schema candidate lifecycle
- OpenAI review adapter

Data:
- SchemaObservation
- SchemaFieldObservation
- SchemaCandidate
- SchemaVersion
- SchemaReview

Boundary:
- learning only
- no automatic enforcement
- no automatic blocking
- no raw secret/value storage

## Future Slices

API-3 introduces OpenAPI contract import, contract versions, and drift analysis.

API-4 introduces positive schema validation after appropriate approval controls.

API-5 introduces JWT and identity-aware API security.

API-6 introduces sequence and workflow analytics.

API-7 introduces BOLA-oriented relationship and authorization analytics.

API-8 introduces GraphQL-specific security analysis.

## Design Principles

1. Learn before enforce.
2. AI assists review; AI does not become policy authority.
3. Sensitive values are not stored as evidence.
4. Every slice maintains source, tests, documentation, and handover state.
