<!-- documentation-review: 2026-09-28; classification: current -->
# Dashboard / Operator Workspace Connector — OWI-1.0 R3

Status: `IMPLEMENTED_TESTING_DEFERRED`. This is an integration slice, not API-9. It adds a dedicated read-only management connector and preserves all existing API-1 through API-8 authority boundaries.

## Contract

- prefix: `/api/integrations/dashboard/v1`
- auth: dedicated `opaque_bearer`, digest-only registry, expiry/rotation/revoke; never admin/browser session
- listener: dedicated management hostname TCP/443 or same-host fixed TCP/19405; plane overlap fails closed
- required lanes: ASSET/LIST_ASSETS, DETECTION/LIST_DETECTIONS, POLICY/OBSERVE_POLICY, HEALTH/HEALTH
- optional lanes: EVENT and ACTION_STATUS disabled/501; no scopes advertised; `actions=[]`

## R3 controls

Reader requests are bounded by per-tenant+principal rate and concurrency plus a global concurrency ceiling. Server processing has an 8s default deadline. Stable snapshots are persisted for cursor continuation across clean restart and are bounded by active count, records and bytes. DETECTION HISTORY uses the existing WAF match authority through a bounded non-blocking export queue and durable journal with 30-day time retention plus explicit row/byte caps. Queue/persistence/capacity loss marks coverage GAP. Context-aware store lock acquisition consumes the request deadline; durable-state write failure latches fail-closed until a successful persist clears it. Reader secret/token files must be owner-only regular non-symlink files, and the built-in management TLS listener requires TLS 1.2 or newer.

Native mapping deliberately remains thin: no duplicate WAF engine is introduced. Query strings and sensitive request material are excluded; identifier-like path segments are templated. WAF.PROTECTION_DEGRADED, WAF.CAPACITY_PRESSURE and WAF.HIGH_SIGNAL_ATTACK are evidence-driven and have positive/counterexample fixtures.

## Qualification truth

Exact isolated connector source: **25 tests PASS / 2 intentional SKIP** and the same set passes under `go test -race` on the available Go 1.23.2 supporting probe. The added coverage includes strict HEAD rejection, disabled-scope provisioning rejection, owner-only/non-symlink secret files, oversize fail-closed behavior, client cancel cleanup, scope-reduction cursor reset, accelerated cursor/HISTORY expiry, independent subprocess SIGKILL restart→GAP, context-bounded store-lock contention/retry, and injected durable-state write failure/recovery. R3 dependency-free source gate: **38/38 PASS**. Product return validator: **PASS (61 files / 18 positive responses / 15 negative cases / 11 transport cases)**. Root repository Go 1.25 qualification is still `BLOCKED_ENVIRONMENT / NOT_RUN`; Dashboard offline acceptance and live deployment are NOT_RUN. These boundaries prevent supporting probe/fixture success from being promoted to TESTED or RELEASED.
