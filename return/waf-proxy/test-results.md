# OWI-1.0 WAF reader verification (2026-09-29)

Environment: Windows host, portable Go 1.25.0; source was not deployed.
Named-file command (with `DASHBOARD_PAGE_SCHEMA_PATH` set to supplied
`outputs/operator-workspace-integration-2026-09-21/schemas/page.schema.json`):

```text
go test -v dashboard_reader.go dashboard_reader_store.go dashboard_reader_isolated_test.go
go vet dashboard_reader.go dashboard_reader_store.go dashboard_reader_isolated_test.go
```

Both **PASS**. Four isolated tests cover scoped/revocable machine auth,
cookie-only/admin-token and mutation rejection, >2-page snapshot with a
restart and concurrent insertion, signed cursor GAP/scope reset, detection
delta/privacy, and supplied shared-schema validation for ASSET, DETECTION,
POLICY, HEALTH plus ASSET DELETE. The shared schema is read from the packet,
not reproduced as a private variant.

`go test -run '^TestDashboardReader' ./...` **FAIL** at root compilation on
pre-existing duplicate `isSensitiveDebugKey`/`sanitizeDebugValue`/
`sanitizeDebugMap` and missing `debugContextFrom`/`debugRequestContext`/
`newDebugTransactionID`/`tlsVersionName` symbols. `go mod tidy -diff`
**FAIL** on current `go.sum` drift. Neither issue was changed in this slice.
`go test -race` is **NOT_RUN**: Windows race requires CGO and no C compiler
was available. Real TLS, VM, and production traffic tests are **NOT_RUN**.

| Gate | State | Scope |
|---|---|---|
| A01 | PARTIAL | Token/scope/revoke/cookie/admin/mutation tested; live disabled listener and TLS not tested. |
| A02 | PARTIAL | Cursor binds tenant and scope; true multi-tenant deployment not tested. |
| A03 | NOT_RUN | No live network/TLS binding. |
| A04 | PARTIAL | Four UPSERT kinds and ASSET DELETE pass shared schema; negative fixture matrix incomplete. |
| A05 | PARTIAL | Three-page/restart/insertion and empty checkpoint behavior tested; concurrency/load incomplete. |
| A06 | PARTIAL | Revision/sequence and detection delta exercised; replay conflict matrix incomplete. |
| A07 | NOT_RUN | Real crash/rotation/backpressure qualification absent. |
| A08 | PARTIAL | GAP and scope-change return RESET_REQUIRED; retention expiry not tested. |
| A09 | PARTIAL | Seeded site/query secret excluded from journal; full headers/body/config/log corpus absent. |
| A10 | PARTIAL | Distinct asset IDs and site association schema checked; NAT/DHCP cases absent. |
| A11 | PARTIAL | Rule hit yields no high-signal attention; pool evidence edge cases incomplete. |
| A12 | PARTIAL | Health/capability handler exists; live deadlines, lag, quota absent. |
| A13 | PARTIAL | ASSET DELETE schema checked; lifecycle/reopen matrix absent. |
| A14 | PARTIAL | POST rejected in isolated handler; no live routing proof. |

