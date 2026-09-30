<!-- documentation-review: 2026-09-28; classification: current-generated-return -->
# WAF Proxy OWI-1.0 R3 handoff

Source checkpoint: `2026-09-30-OWI-R3-CONNECTOR-HARDENED`; identity SHA-256 `d131551565cf294fc6a5258436941cde933d35c6397f444e7c9214c63377953c`. This return implements the product-owned Phase A reader only; it does not modify Dashboard, register a production Product Instance, or enable any write/action plane.

## Four states

- PRODUCT_API_LOCAL: **BLOCKED** — exact connector-source tests/race PASS as supporting evidence, but integrated root Go 1.25 qualification is `BLOCKED_ENVIRONMENT / NOT_RUN`
- RETURN_PACKAGE: **PASS**
- DASHBOARD_OFFLINE_ACCEPTANCE: **NOT_RUN** — next owner: Dashboard
- LIVE_DEPLOYMENT: **NOT_RUN** — next owners: WAF deployment + Dashboard worker

Required lanes implemented: ASSET/LIST_ASSETS, DETECTION/LIST_DETECTIONS, POLICY/OBSERVE_POLICY, HEALTH/HEALTH. Optional EVENT and ACTION_STATUS remain disabled and cannot be provisioned as token scopes. `actions=[]`. Dedicated opaque bearer, per-identity/global limits, strict GET-only including HEAD rejection, bounded source/store lock deadline, fail-closed durable state, snapshot/cursor/revision persistence, token expiry/rotation/revoke, sanitization and security-export GAP semantics are implemented.

Re-run fixture generation from exact source: `WAF_OWI_RETURN_FIXTURE_DIR=<dir> go test -run TestGenerateOWIReturnFixture -count=1 .`, then `python3 tools/assemble_waf_dashboard_return.py ...`, then `python3 tools/validate_waf_dashboard_return.py --return-dir return/waf-proxy`. Dashboard should independently hash `binding.fixture.json` and run its fixture qualification CLI; product code must not patch the Dashboard validator or production registry.
