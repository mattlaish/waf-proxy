# Open gates and requested decisions

1. **Approval/binding:** Auto-review blocked wiring the separate HTTPS reader
   into WAF startup. Need explicit user authorization of the read-only metadata
   exposure and actual A/B profile, management IP, TLS hostname/cert/CA,
   source instance and tenant IDs. Do not bypass the rejection. No fabricated
   `binding.json` was created.
2. **Build baseline:** Existing dirty root source has duplicate/missing debug
   symbols; Go 1.25 `tidy -diff` shows `go.sum` drift. Repair in a separate,
   reviewed slice without overwriting unrelated edits.
3. **Runtime wiring:** No detection callback, apply reconciliation, health
   sampler, token-mint CLI, independent default-off listener lifecycle,
   systemd/ACL/TLS deployment, or end-to-end worker connection yet.
4. **Durability/scale:** 64 MiB append journal is intentionally fail-closed
   with GAP; 30-day detection retention, compaction, backpressure throughput,
   race, crash boundaries and multi-principal rotation need qualification.
5. **Contract qualification:** A01–A14 are only partial/NOT_RUN as detailed in
   `test-results.md`. No source-aware Dashboard adapter is supplied here.
   Optional events/action-status and all Phase B actions remain disabled.
6. **Return packet:** Real `binding.json`, deployed integration test logs,
   reviewed fixture corpus and deploy-ready schema copy are withheld until
   environment/approval and full qualification are available.
