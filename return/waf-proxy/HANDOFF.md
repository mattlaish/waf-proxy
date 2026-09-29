# WAF Proxy → Operator Workspace handoff (partial)

Status: **UNWIRED_PROTOTYPE / NO_GO**, 2026-09-29. Source baseline observed at
`main@0931e75d6659a5636692678575053d355aae5a70`; the repository already
had substantial unrelated local changes. No Git write or deployment occurred.
The OWI-1.0 packet is a target contract, not proof of an existing API.

Added, untracked source files (SHA-256):

| File | SHA-256 |
|---|---|
| `dashboard_reader.go` | `b50d5e095744772bc197cf761145eb72705505cc81707f8d239c6539f8831b6a` |
| `dashboard_reader_store.go` | `c75e727ccb5b4d03882990282652732f52b259cc097088cc2fef284a7a0bf3fc` |
| `dashboard_reader_isolated_test.go` | `c4bee1fa341136f055c77d5f93f852a26de0d98e3b75e9f2e579a72b65a0926d` |

The prototype has a separate machine-token digest and scope check, closed
allowlist projections for the four required resource kinds, a durable JSONL
read model, signed cursor, snapshot restart and explicit GAP behavior. It has
no database migration and no active route. `/events`, `/action-status`, and
Phase B actions are unsupported. Token-mint helper and TLS server constructor
are not connected to a CLI or lifecycle.

Security review rejected the attempted `main.go` wiring of a new HTTPS
listener because concrete destination/data exposure had not been explicitly
authorized and startup failure could affect WAF. No equivalent wiring was
attempted. Deployment and rollback are therefore **NOT_APPLICABLE** yet;
source rollback is removing only these three new files after review.

Test environment: Windows host, portable official Go 1.25.0, isolated
named-file harness against supplied OWI-1.0 page schema. See
`test-results.md`. Next owner must obtain exact binding/approval, repair the
unrelated root build blockers, safely wire the reader default-off, and perform
the full A01–A14 and real VM/TLS qualification before enabling it.

