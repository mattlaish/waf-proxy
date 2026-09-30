<!-- documentation-review: 2026-09-28; classification: current-generated-return -->
# WAF Proxy OWI reader deployment

The reader is a separate management listener, never the WAF business listener and never the browser/admin bearer plane. Set `WAF_DASHBOARD_READER_ENABLED=true`; choose dedicated management host TCP/443 or same-host fixed TCP/19405; configure TLS cert/key, `WAF_DASHBOARD_SOURCE_INSTANCE_ID`, owner-only non-symlink `WAF_DASHBOARD_CURSOR_SECRET_FILE`, and an owner-only non-symlink digest-only token registry. The built-in reader TLS listener enforces TLS >=1.2. Optional EVENT/ACTION_STATUS scopes cannot be provisioned in R3.

## Bounded resource defaults

- request rate: 10 requests/sec per authenticated tenant+principal; burst 20 (`WAF_DASHBOARD_RATE_PER_SECOND`, `WAF_DASHBOARD_RATE_BURST`)
- concurrency: 2 per principal, 16 global, immediate rejection/no request queue (`WAF_DASHBOARD_PER_PRINCIPAL_CONCURRENCY`, `WAF_DASHBOARD_GLOBAL_CONCURRENCY`)
- server deadline: 8s (`WAF_DASHBOARD_REQUEST_DEADLINE_SECONDS`); source sync and store-lock acquisition consume this same request context; no unbounded worker is spawned on timeout
- Dashboard client contract remains connect 5s/read 10s/hard 30s
- page: max 500; response: max 4 MiB
- retained snapshots: max 256 active, max 10,000 records and 32 MiB per snapshot (`WAF_DASHBOARD_SNAPSHOT_MAX_COUNT`, `_MAX_RECORDS`, `_MAX_BYTES`)
- detection durable journal: max 200,000 rows / 128 MiB plus 30-day time retention (`WAF_DASHBOARD_EXPORT_MAX_ROWS`, `_MAX_BYTES`). If capacity or persistence prevents durable service, the reader fails closed; detection queue/persistence loss marks GAP rather than fake COMPLETE.
- non-blocking security export queue: 4,096 observations; overflow marks GAP and does not block WAF request hot path
- snapshot/cursor TTL: 24h; STATE change journal retention: 7d; enabled DETECTION HISTORY retention target: 30d
- cleanup bound under slow lock contention: request returns when the configured server deadline expires; per-principal/global limiter slots release with handler return. Kernel/filesystem I/O cannot be preempted portably, so G3 must qualify the intended filesystem under fault/load conditions.

Quota counters are process-local. A multi-instance deployment multiplies aggregate request quota unless ingress adds a shared cap; this must be sized/reviewed in G3. The durable state/cursor key is instance-owned; cross-node reader HA is not claimed by this return.

## Token lifecycle

Use `tools/waf_dashboard_token.py`. Tokens are 32-byte CSPRNG values shown only at creation/rotation; registry persists a domain-separated SHA-256 digest. Expiry is at most 90 days; rotation bounds old-token overlap to at most 24h; revoke is explicit. Reader tokens contain only the six enabled R3 read scopes; admin/session/writer credentials are not accepted.
