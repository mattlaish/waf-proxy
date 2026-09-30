<!-- documentation-review: 2026-09-28; classification: current-generated-return -->
# Open gaps

- PRODUCT_API_LOCAL: BLOCKED. Exact connector source tests, including race, deadline/lock contention, client cancel, oversize fail-closed, real subprocess SIGKILL recovery-to-GAP, accelerated retention expiry and injected write failure/recovery, PASS on the available Go 1.23.2 supporting probe. The integrated product repository requires Go 1.25 and cannot be compiled/qualified here because the required toolchain/modules cannot be obtained; supporting probe PASS is not promoted to product-local qualification.
- RETURN_PACKAGE: PASS for product-owned R3 return construction, schema validation and SHA256SUMS.
- DASHBOARD_OFFLINE_ACCEPTANCE: NOT_RUN. No Dashboard checkout/fixture CLI was available in this execution environment. Dashboard owns consumer issue C-03 if it rejects `bootstrap_retention_days:null` for the disabled EVENT HISTORY lane; WAF deliberately does not claim nonexistent retention for a disabled capability.
- LIVE_DEPLOYMENT: NOT_RUN. Requires real DNS/IP/CA/SNI, worker ACL, opaque secret provisioning, intended OS/filesystem/backend, real HTTPS transport, load/capacity and deployed restart/failure evidence.
- Oversize response and request-context cancel are PASS locally. A real TCP socket disconnect/worker hard-deadline path remains NOT_RUN until G3.
- Optional EVENT/ACTION_STATUS and all Phase B writers are NOT_IMPLEMENTED by design for this round.
