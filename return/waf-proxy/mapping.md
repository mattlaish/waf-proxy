<!-- documentation-review: 2026-09-28; classification: current-generated-return -->
# WAF Proxy OWI-1.0 R3 mapping and applicability

Source identity SHA-256: `d131551565cf294fc6a5258436941cde933d35c6397f444e7c9214c63377953c`. This table is the O-01 inventory required by the R3 acceptance contract; it maps existing product authority to the thin reader facade and does not create a second WAF engine.

| Case ID | M/C/N/A | Reason | Code/config location | Test command | Environment | Source hash | Result | Evidence path | Gap owner |
|---|---|---|---|---|---|---|---|---|---|
| O-01 | M | Inventory/native→wire mapping is mandatory | `dashboard_connector_*.go`, `main.go`, `return/waf-proxy/` | `python3 tools/dashboard_connector_r3_source_gate.py` | local source gate | `d131551565cf294fc6a5258436941cde933d35c6397f444e7c9214c63377953c` | PASS | `DASHBOARD_CONNECTOR_R3_SOURCE_GATE_RESULT.md` | product |
| O-02 | M | Reader rate/concurrency/capacity protections | `dashboard_connector_http.go`, `dashboard_connector_store.go` | `go test -run 'TestOWI(RateLimit|Limiter|SnapshotCapacity|DetectionExportCapacity)'` | exact isolated connector source | `d131551565cf294fc6a5258436941cde933d35c6397f444e7c9214c63377953c` | PASS supporting | `test-logs/core-probe.txt` | G3 capacity sizing: deployment |
| O-03 | M | Source/lock/page work must be bounded and cancel-safe | `dashboard_connector_http.go`, `dashboard_connector_store.go` | `go test -run 'TestOWI(Deadline|ClientCancel|StoreContention)'` | exact isolated connector source | `d131551565cf294fc6a5258436941cde933d35c6397f444e7c9214c63377953c` | PASS supporting | `test-logs/core-probe.txt` | live socket/TLS: deployment |
| O-04 | M/C | Durable STATE + enabled DETECTION HISTORY | `dashboard_connector_store.go`, `dashboard_connector_export.go` | `go test -run 'TestOWI(RealProcessKill|StoreWriteFailure|CursorAndDetectionRetention|Restart)'` | exact isolated connector source/local filesystem | `d131551565cf294fc6a5258436941cde933d35c6397f444e7c9214c63377953c` | PASS supporting | `test-logs/core-probe.txt` | G3 intended filesystem/runtime |
| O-05 | C/M | WAF profile is opaque bearer; TLS/GET-only/tenant/scope mandatory | `dashboard_connector_http.go`, `main.go`, `tools/waf_dashboard_token.py` | `go test -run 'TestOWI.*(Auth|Token|Scope|GETOnly|Mutation|Cursor)'` | exact isolated connector source | `d131551565cf294fc6a5258436941cde933d35c6397f444e7c9214c63377953c` | PASS supporting | `test-logs/core-probe.txt` | production credential/TLS: deployment |
| O-06 | M | Native WAF identity/security/health mapping | `dashboard_connector_export.go` | `go test -run 'TestOWI.*(Saniti|Detection|Fixture)'` | exact isolated connector source | `d131551565cf294fc6a5258436941cde933d35c6397f444e7c9214c63377953c` | PASS supporting | `fixtures/attention-examples.json` | live capacity thresholds: deployment |
| O-07 | M | Executable return/fixtures/catalogs | `dashboard_connector_fixture_test.go`, `tools/assemble_waf_dashboard_return.py` | `python3 tools/validate_waf_dashboard_return.py --return-dir return/waf-proxy` | local return validator | `d131551565cf294fc6a5258436941cde933d35c6397f444e7c9214c63377953c` | PASS | `test-logs/fixture-generation.txt`; `SHA256SUMS` | Dashboard offline runner |
| O-08 | M | Product-owned return + SHA256SUMS | `return/waf-proxy/` | `sha256sum -c SHA256SUMS` | clean return tree | `d131551565cf294fc6a5258436941cde933d35c6397f444e7c9214c63377953c` | PASS | `SHA256SUMS` | Dashboard review/pin |
| HISTORY-30D | C | DETECTION is enabled HISTORY; EVENT is disabled | `dashboard_connector_store.go`, `/capabilities` | `go test -run TestOWICursorAndDetectionRetentionExpiry` | accelerated local clock | `d131551565cf294fc6a5258436941cde933d35c6397f444e7c9214c63377953c` | PASS supporting | `test-logs/core-probe.txt` | real elapsed 30d: G3 |
| Phase-B | N/A | R3 explicitly excludes product write actions | capability `actions=[]`, optional action-status disabled | source gate | local | `d131551565cf294fc6a5258436941cde933d35c6397f444e7c9214c63377953c` | N/A by contract | `HANDOFF.md` | future governed slice |

## Native → wire authority

| Wire kind | Native authority | Sanitization/null policy | Revision trigger | ID owner | Retention/semantics |
|---|---|---|---|---|---|
| ASSET | WAF runtime/config: node, site, virtual service, origin pool | no raw config export; HTTP peer is not protected app | mapped source-content change | WAF connector stable external ID | STATE; snapshot + delta |
| DETECTION | Coraza/WAF security match stream | query/cookie/Authorization/body/raw sensitive headers excluded; safe path template + rule ID retained | each accepted native security observation | WAF connector durable detection ID | HISTORY; 30-day target + GAP on loss |
| POLICY | native WAF/Positive-Schema policy config | allow-listed active generation/digest/scope only | active mapped policy content change | WAF connector policy ID | STATE; snapshot + delta |
| HEALTH | runtime + measured connector/export state | measured values only; unknown capacity remains null/UNKNOWN | mapped observation change | WAF connector health ID | STATE; snapshot + delta |
| EVENT | none in R3 | disabled | N/A | N/A | disabled; no scope/retention claim |
| ACTION_STATUS | none in R3 | disabled | N/A | N/A | disabled; no scope; Phase A has no actions |

Attention codes are `WAF.PROTECTION_DEGRADED`, `WAF.CAPACITY_PRESSURE`, and `WAF.HIGH_SIGNAL_ATTACK`; positive and evidence-insufficient counterexamples are under `fixtures/attention-examples.json`.
