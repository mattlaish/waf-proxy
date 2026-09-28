# API-6.4 Sequence Operations + Hardening

Status: `IMPLEMENTED_TESTING_DEFERRED`  
Date: 2026-09-24

## Scope

API-6.4 completes the API-6 operational layer without giving learned sequence analytics enforcement authority. The runtime now exposes explicit site-scoped `LEARN` / `DETECT` controls, bounded recent-session evidence, learning-state summaries, exception create/delete operations, reset/relearn, and the full embedded Sequence Operations console.

There is no sequence `ENFORCE` mode. API-6 sequence findings never own `BLOCK`, `DENY`, or request-path `403` authority.

## Operational controls

- Default site state is `LEARN`.
- Promotion to `DETECT` requires at least one `MATURE` workflow for that site.
- A v3 state restored into the v4 runtime defaults to `LEARN` because no explicit control record existed previously.
- Reset/relearn is site-scoped, clears learned/runtime sequence state and violations for that site, preserves explicit exceptions, increments the control generation, and forces the site back to `LEARN`.
- Site-control cardinality is bounded.

Reviewer-gated mutations are persisted before success is returned and emit audit events for mode changes, reset/relearn, exception creation, and exception deletion.

## Exceptions

Exception creation accepts only bounded selectors over site, anomaly type, keyed workflow ID, and normalized operation IDs. Site-only broad exceptions are rejected. Exception TTL is bounded and defaults to 24 hours. Raw URLs, object values, JWTs, cookies, claim values, or other sensitive request values are not selectors.

## Sessions and evidence

Expired or capacity-evicted active sessions may become bounded recent-session summaries containing only keyed session/workflow fingerprints, normalized operation IDs, correlation kind, timestamps, and request count. Recent-session evidence has TTL and cardinality limits and is persisted in sequence state v4. State v4 retains restore compatibility with v1/v2/v3.

## Console

The embedded Admin Console now provides:

- Sequence Overview / Learning State
- Workflow Models
- Transitions
- Active and Recent Sessions
- Sequence Violations
- Exceptions
- LEARN / DETECT mode control
- Reset / Relearn

The console resolves normalized operation IDs to operation labels when inventory data is available; the underlying durable evidence remains normalized/keyed rather than raw sensitive values.

## Evidence boundary

Local source/static evidence for this slice includes the dedicated API-6.4 source gate plus all retained API source gates, OpenAI source/isolated checks, root-source shape, package-builder tests, JavaScript syntax, shell syntax, and formatting checks. Canonical Go 1.25 targeted and race execution remains `BLOCKED_ENVIRONMENT / NOT_RUN` on the available Go 1.23.2 host with no external toolchain/module retrieval. No source/static/advisory result upgrades the slice to `TESTED` or `RELEASED`.

## Next slice

API-6 is now complete at implementation level through API-6.4. The next roadmap slice is `API-7.1 Object Locator Discovery`, still `PLANNED`.
