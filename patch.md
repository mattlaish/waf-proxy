# Production Correctness & Control-Plane Hardening patch

Parent: waf-proxy-api8-post-audit-hardening-2026-09-24.zip
Parent SHA-256: e838ce6bd4f9cca6176cb809eb189978f312b194ae4b7c2f6f13f892a2d85f98

```diff
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/.github/workflows/ci.yml ./.github/workflows/ci.yml
--- /mnt/data/waf_prod_hardening_base/.github/workflows/ci.yml	2026-09-25 11:11:25.266162571 +0000
+++ ./.github/workflows/ci.yml	2026-09-25 11:32:52.799182430 +0000
@@ -56,6 +56,8 @@
         run: python3 ./tools/tests/test-api8-source.py
       - name: API-8 post-audit hardening source gate
         run: python3 ./tools/tests/test-api8-post-audit-hardening-source.py
+      - name: Production correctness and control-plane hardening source gate
+        run: python3 ./tools/tests/test-production-control-plane-hardening-source.py
 
       # Fail before compilation if committed module metadata is not canonical.
       # -diff never mutates go.mod/go.sum and exits non-zero when tidy would.
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/AI_HANDOFF.md ./AI_HANDOFF.md
--- /mnt/data/waf_prod_hardening_base/AI_HANDOFF.md	2026-09-25 12:02:55.515522420 +0000
+++ ./AI_HANDOFF.md	2026-09-25 13:33:09.159713010 +0000
@@ -2069,3 +2069,8 @@
 ## Current Work — API-8 Post-Audit Hardening (2026-09-24)
 
 Status: IMPLEMENTED_TESTING_DEFERRED. This pass is not API-9. It fixes CIDR Enabled/expiry runtime truth, implements the previously dead L7 TLS-handshake limit on built-in Go TLS with external-frontend fail-closed validation, closes Console exposure gaps for System/Doctor/Debug/HSM/Vector/CIDR/L7/OpenAPI/Positive-Schema, replaces Sequence/BOLA/GraphQL opaque-ID text inputs with inventory-backed selectors, removes the obsolete non-shipping `web/` frontend, and removes six model-only Security Operations/debug-lifecycle files that were not wired to runtime/API/Console. Phase 5 Slice D is corrected to PLANNED / NOT IMPLEMENTED. Use `static/admin.html` as the only Console source. Canonical Go 1.25 qualification is still required.
+
+## 2026-09-25 — Production Correctness & Control-Plane Hardening
+
+Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.
+
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/API_SECURITY_CLOSURE_RESULT.md ./API_SECURITY_CLOSURE_RESULT.md
--- /mnt/data/waf_prod_hardening_base/API_SECURITY_CLOSURE_RESULT.md	2026-09-25 12:02:58.951598308 +0000
+++ ./API_SECURITY_CLOSURE_RESULT.md	2026-09-25 13:33:09.166248227 +0000
@@ -206,3 +206,8 @@
 ## Post-closure correction — API-8 Post-Audit Hardening (2026-09-24)
 
 The API-1 through API-8 authority roadmap remains implementation-complete, but a repository-wide audit found and corrected two runtime truth gaps (CIDR Enabled/expiry and an unused TLS-handshake abuse knob) plus operator Console exposure gaps. This hardening does not create API-9 or change API-6/API-7/API-8 authority. It also removes an obsolete non-shipping frontend and model-only Security Operations placeholders that had no runtime/API/Console implementation. Overall release status remains IMPLEMENTED_TESTING_DEFERRED pending canonical Go 1.25 and live qualification.
+
+## 2026-09-25 — Production Correctness & Control-Plane Hardening
+
+Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.
+
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/API_SECURITY_ROADMAP.md ./API_SECURITY_ROADMAP.md
--- /mnt/data/waf_prod_hardening_base/API_SECURITY_ROADMAP.md	2026-09-25 12:02:59.407608379 +0000
+++ ./API_SECURITY_ROADMAP.md	2026-09-25 13:33:09.166759517 +0000
@@ -455,3 +455,8 @@
 ## 2026-09-24 post-API-8 hardening note
 
 No API-9 is defined. API-8 post-audit hardening closes runtime correctness and operator-surface gaps discovered after API-8 implementation and preserves all established authority boundaries. Release promotion remains a qualification task, not a new API-security feature slice.
+
+## 2026-09-25 — Production Correctness & Control-Plane Hardening
+
+Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.
+
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/API_SECURITY_SLICE_IMPLEMENTATION_ROADMAP.md ./API_SECURITY_SLICE_IMPLEMENTATION_ROADMAP.md
--- /mnt/data/waf_prod_hardening_base/API_SECURITY_SLICE_IMPLEMENTATION_ROADMAP.md	2026-09-25 12:02:59.811617302 +0000
+++ ./API_SECURITY_SLICE_IMPLEMENTATION_ROADMAP.md	2026-09-25 13:33:09.167181463 +0000
@@ -235,3 +235,7 @@
 
 The documented API-security implementation roadmap is now complete through API-8. No API-9 is defined. Next gate: canonical Go 1.25 qualification of the exact source/artifact, followed by release promotion only if all required gates pass.
 
+## 2026-09-25 — Production Correctness & Control-Plane Hardening
+
+Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.
+
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/DEVELOPMENT.md ./DEVELOPMENT.md
--- /mnt/data/waf_prod_hardening_base/DEVELOPMENT.md	2026-09-25 12:02:59.383607850 +0000
+++ ./DEVELOPMENT.md	2026-09-25 13:33:09.157907389 +0000
@@ -472,3 +472,8 @@
 ## 2026-09-24 — API-8 Post-Audit Hardening
 
 Implemented the post-API-8 repository audit fixes without defining a new API-security slice. CIDR runtime now honors Enabled and RFC3339 expiry. The previously dead TLS handshake abuse knob is implemented on the built-in Go TLS ClientHello path with bounded listener/peer token buckets and fails configuration validation with the external TLS frontend. The shipping Console now exposes System/Diagnostics, Debug, HSM, VectorScan, CIDR/L7 traffic controls, complete OpenAPI and Positive Schema lifecycle operations, and inventory-backed selectors for API-security opaque identifiers. Removed the obsolete non-shipping `web/` tree and six model-only/fake-foundation Go files that had no runtime store/API/Console wiring. Phase 5 Slice D truth is corrected to PLANNED / NOT IMPLEMENTED. Authority boundaries are unchanged. See `API8_POST_AUDIT_HARDENING.md`.
+
+## 2026-09-25 — Production Correctness & Control-Plane Hardening
+
+Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.
+
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/DEVELOPMENT_ROADMAP.md ./DEVELOPMENT_ROADMAP.md
--- /mnt/data/waf_prod_hardening_base/DEVELOPMENT_ROADMAP.md	2026-09-25 12:03:00.243626844 +0000
+++ ./DEVELOPMENT_ROADMAP.md	2026-09-25 13:33:09.163576151 +0000
@@ -624,3 +624,7 @@
 
 The documented API-security implementation roadmap is now complete through API-8. No API-9 is defined. Next gate: canonical Go 1.25 qualification of the exact source/artifact, followed by release promotion only if all required gates pass.
 
+## 2026-09-25 — Production Correctness & Control-Plane Hardening
+
+Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.
+
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/DOCUMENTATION_INDEX.md ./DOCUMENTATION_INDEX.md
--- /mnt/data/waf_prod_hardening_base/DOCUMENTATION_INDEX.md	2026-09-25 12:03:00.243626844 +0000
+++ ./DOCUMENTATION_INDEX.md	2026-09-25 13:33:09.164382916 +0000
@@ -269,3 +269,8 @@
 - `API8_POST_AUDIT_HARDENING.md` — runtime correctness, Console exposure closure, debt cleanup and authority boundaries.
 - `API8_POST_AUDIT_HARDENING_SOURCE_GATE_RESULT.md` — dependency-free 134-check hardening source gate.
 - `API8_POST_AUDIT_HARDENING_TEST_SUMMARY.txt` — exact-source qualification summary and canonical Go blocker truth.
+
+## 2026-09-25 — Production Correctness & Control-Plane Hardening
+
+Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.
+
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/HANDOVER_PROMPT.md ./HANDOVER_PROMPT.md
--- /mnt/data/waf_prod_hardening_base/HANDOVER_PROMPT.md	2026-09-25 12:03:00.407630466 +0000
+++ ./HANDOVER_PROMPT.md	2026-09-25 13:33:09.163083188 +0000
@@ -117,3 +117,8 @@
 
 
 API-8 post-audit hardening is the current implementation baseline once final packaging evidence is attached. Preserve: CIDR RFC3339 expiry and Enabled semantics; built-in-Go-TLS bounded handshake limiting with external TLS frontend fail-closed validation; SYSTEM/Doctor/Debug/HSM/Vector and CIDR/L7 Console surfaces; OpenAPI/Positive Schema lifecycle controls; inventory-backed opaque selectors; one shipping Console source under static/; and corrected Phase 5 Slice D truth. Do not claim TESTED/RELEASED until canonical Go 1.25 and required live gates actually pass.
+
+## 2026-09-25 — Production Correctness & Control-Plane Hardening
+
+Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.
+
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/HANDOVER_STATUS.md ./HANDOVER_STATUS.md
--- /mnt/data/waf_prod_hardening_base/HANDOVER_STATUS.md	2026-09-25 12:03:00.659636032 +0000
+++ ./HANDOVER_STATUS.md	2026-09-25 13:33:09.161849820 +0000
@@ -156,3 +156,8 @@
 ## API-8 Post-Audit Hardening — 2026-09-24
 
 **IMPLEMENTED_TESTING_DEFERRED.** CIDR expiry/enable and L7 TLS handshake abuse control are implemented; SYSTEM/HSM/Vector/Debug/Doctor, CIDR/L7 traffic controls, OpenAPI/Positive Schema lifecycle and opaque-ID selectors are exposed in the shipping Console; obsolete `web/` and unwired model-only foundation files are removed. Existing API authority separation remains unchanged. Canonical Go 1.25 runtime/race qualification is still blocked by the local environment.
+
+## 2026-09-25 — Production Correctness & Control-Plane Hardening
+
+Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.
+
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/MANIFEST.md ./MANIFEST.md
--- /mnt/data/waf_prod_hardening_base/MANIFEST.md	2026-09-25 12:03:00.095623576 +0000
+++ ./MANIFEST.md	2026-09-25 13:33:09.164971223 +0000
@@ -558,3 +558,8 @@
 - `static/admin.html` — single shipping Console source with System/Diagnostics, HSM/Vector, traffic controls, lifecycle and opaque-ID selector closure.
 
 Removed as non-runtime/misleading debt: `web/`, `investigation.go`, `security_timeline.go`, `change_audit.go`, `security_export.go`, `debug_lifecycle_v2.go`, and `debug_retention_worker_v2.go`.
+
+## 2026-09-25 — Production Correctness & Control-Plane Hardening
+
+Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.
+
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/NEXT_CHAT_HANDOVER.md ./NEXT_CHAT_HANDOVER.md
--- /mnt/data/waf_prod_hardening_base/NEXT_CHAT_HANDOVER.md	2026-09-25 12:03:00.667636209 +0000
+++ ./NEXT_CHAT_HANDOVER.md	2026-09-25 13:33:09.162287189 +0000
@@ -117,3 +117,8 @@
 ## API-8 Post-Audit Hardening checkpoint
 
 Continue from the post-audit hardening complete-source artifact, not the pre-hardening API-8 ZIP, once its final artifact gate is recorded. Do not reintroduce `web/` or the deleted model-only Security Operations/debug lifecycle files as if they were implemented features. Phase 5 Slice D remains PLANNED / NOT IMPLEMENTED until a real store/API/Console workflow is built. Before release promotion, run the exact artifact under pinned Go 1.25 and execute required build/vet/test/race plus production dependency qualification.
+
+## 2026-09-25 — Production Correctness & Control-Plane Hardening
+
+Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.
+
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/PRODUCTION_CONTROL_PLANE_HARDENING.md ./PRODUCTION_CONTROL_PLANE_HARDENING.md
--- /mnt/data/waf_prod_hardening_base/PRODUCTION_CONTROL_PLANE_HARDENING.md	1970-01-01 00:00:00.000000000 +0000
+++ ./PRODUCTION_CONTROL_PLANE_HARDENING.md	2026-09-25 13:33:08.692065342 +0000
@@ -0,0 +1,33 @@
+# Production Correctness & Control-Plane Hardening
+
+Status: **IMPLEMENTED_TESTING_DEFERRED**  
+Date: 2026-09-25  
+Parent: `waf-proxy-api8-post-audit-hardening-2026-09-24.zip` (`e838ce6bd4f9cca6176cb809eb189978f312b194ae4b7c2f6f13f892a2d85f98`)
+
+This hardening wave closes production correctness and control-plane findings discovered during the post-API-8 full-repository adversarial audit. It does not add API-9 and does not change API-6/API-7 non-enforcement authority.
+
+## Implemented
+
+- Block responses use `html/template`; request correlation IDs accept only a bounded safe character set. HTML block responses add no-store, nosniff, CSP and frame-deny headers.
+- Admin JSON responses use no-store/nosniff controls. Notification webhook URLs remain write-only and are redacted from config reads; a separate operator-only clear action is provided.
+- Debug evidence masks nested remote/resolved client IPs.
+- Login abuse protection is bounded and fail-closed under key saturation; session creation fails closed on CSPRNG error; PBKDF2 parameters are bounded before work begins.
+- Mutating admin routes are role-gated and audited. Admin audit records are durably appended to a mode-0600 JSONL log with bounded rotation.
+- HA config replication uses a dedicated peer-only HTTPS endpoint and stable replication token. Generic config APIs no longer trust caller-supplied sync headers. Shared config strips node-local HA identity, users and secrets; the receiver restores its own local state. Outbound sync is bounded to one in-flight request plus a latest-wins pending slot.
+- Full config mutation is serialized. Config is staged with a same-directory unique temp file, mode 0600 and file fsync; live network/runtime application is checked before atomic rename, and pre-rename commit failure rolls live state back. Directory fsync is performed after rename.
+- Listener reconciliation synchronously binds required sockets and rolls back failed protocol transitions. External TLS frontend -> Go TLS waits for frontend stop before public bind. Go TLS -> external frontend now synchronously releases the public listening socket before frontend activation, removing the EADDRINUSE ownership race.
+- Sitemap clear persists the cleared snapshot before dependent learning state is discarded and restores the previous snapshot on persistence failure.
+- Early API-security stores reject future/unknown durable state versions.
+- Notification webhook delivery uses a bounded queue and workers, bounded retries, non-2xx checking and queue-drop telemetry.
+- L7 limiter saturation uses a bounded conservative overflow bucket rather than allowing unseen identities.
+- Admin file browsing resolves symlinks before enforcing the browse-root boundary.
+- In-process updater writes are disabled by default for packaged/systemd deployment. A standalone writable install requires explicit `WAF_UPDATE_INSTALL_DIR`; packaged installations use the OS package manager.
+- systemd units explicitly declare device access for hardware watchdog and QAT-related device nodes instead of combining those capabilities with `PrivateDevices=true`.
+- Console actions are capability-aware and `/api/learn/clear` is exposed to reviewers. OpenAPI import and webhook-secret mutation remain operator-only.
+- Main config sample/documentation includes the post-audit CIDR/L7 controls.
+
+## Validation truth
+
+Static/source gates and artifact gates can validate source structure, security contracts, syntax, packaging and clean extraction. Canonical Go 1.25 compile/test/race remains **BLOCKED_ENVIRONMENT / NOT_RUN** on this host because the installed Go is 1.23.2, `go.mod` requires 1.25.0, and external toolchain retrieval is unavailable. This baseline must not be promoted to TESTED or RELEASED on source-gate evidence alone.
+
+The supplied source archive has no `.git` metadata. Therefore branch creation, remote/main synchronization, commit identity and fresh-clone verification cannot be performed from this artifact. Parent ZIP SHA-256 and package manifests are the source identity for this handoff.
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/PRODUCTION_CONTROL_PLANE_HARDENING_SOURCE_GATE_RESULT.md ./PRODUCTION_CONTROL_PLANE_HARDENING_SOURCE_GATE_RESULT.md
--- /mnt/data/waf_prod_hardening_base/PRODUCTION_CONTROL_PLANE_HARDENING_SOURCE_GATE_RESULT.md	1970-01-01 00:00:00.000000000 +0000
+++ ./PRODUCTION_CONTROL_PLANE_HARDENING_SOURCE_GATE_RESULT.md	2026-09-25 13:33:08.693025248 +0000
@@ -0,0 +1,11 @@
+# Production Correctness & Control-Plane Hardening — Source Gate Result
+
+Status: **PASS (source/static gate only)**
+
+`tools/tests/test-production-control-plane-hardening-source.py`:
+
+- `PRODUCTION_CONTROL_PLANE_HARDENING_SOURCE_GATE_PASS checks=190`
+
+The gate checks request-ID/block-page hardening, JSON privacy headers, webhook secret handling, debug privacy, login/session bounds, RBAC/audit, durable audit, dedicated HA replication and node-local preservation, atomic config persistence, synchronous listener ownership transitions including Go-TLS -> external frontend release, sitemap durability, state-version guards, bounded notification delivery, L7 saturation behavior, update/deployment constraints, systemd device declarations, Console capability mapping, test presence and build/CI wiring.
+
+This is not a substitute for canonical Go 1.25 build/test/race execution.
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/PRODUCTION_CONTROL_PLANE_HARDENING_TEST_SUMMARY.txt ./PRODUCTION_CONTROL_PLANE_HARDENING_TEST_SUMMARY.txt
--- /mnt/data/waf_prod_hardening_base/PRODUCTION_CONTROL_PLANE_HARDENING_TEST_SUMMARY.txt	1970-01-01 00:00:00.000000000 +0000
+++ ./PRODUCTION_CONTROL_PLANE_HARDENING_TEST_SUMMARY.txt	2026-09-25 13:37:42.521268527 +0000
@@ -0,0 +1,39 @@
+PRODUCTION CORRECTNESS & CONTROL-PLANE HARDENING TEST SUMMARY
+Status: IMPLEMENTED_TESTING_DEFERRED
+
+Exact-source source gates:
+API-1/2 69 PASS
+API-3 47 PASS
+API-4 46 PASS
+API-5 72 PASS
+API-6.1 33 PASS
+API-6.2 56 PASS
+API-6.3 45 PASS
+API-6.4 58 PASS
+API-7.1 83 PASS
+API-7.2 100 PASS
+API-7.3 110 PASS
+API-7.4 156 PASS
+API-8 259 PASS
+API-8 post-audit hardening 134 PASS
+Production control-plane hardening 190 PASS
+
+Additional exact-source gates:
+OpenAI source contract 16/16 PASS
+OpenAI isolated tests PASS
+WAF package-source PASS
+WAF package-builder 9/9 PASS
+Root Go source shape 112 files PASS
+Changed Go gofmt PASS
+Admin embedded JavaScript syntax PASS
+Shell syntax PASS
+Python syntax PASS
+Runtime-junk hygiene after validation: 0
+Production targeted Go test functions present: 22
+
+Canonical Go 1.25 qualification:
+go mod tidy -diff: BLOCKED_ENVIRONMENT / NOT_RUN
+go build ./...: BLOCKED_ENVIRONMENT / NOT_RUN
+go test -run '^TestProduction': BLOCKED_ENVIRONMENT / NOT_RUN
+go test -race -run '^TestProduction': BLOCKED_ENVIRONMENT / NOT_RUN
+Reason: go.mod requires Go >=1.25.0; host is Go 1.23.2; external toolchain retrieval unavailable.
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/README.md ./README.md
--- /mnt/data/waf_prod_hardening_base/README.md	2026-09-25 12:03:01.563655998 +0000
+++ ./README.md	2026-09-25 13:33:09.156808784 +0000
@@ -1320,3 +1320,8 @@
 ## API-8 Post-Audit Hardening — 2026-09-24
 
 The current development baseline includes the post-audit hardening documented in `API8_POST_AUDIT_HARDENING.md`: CIDR Enabled/expiry correctness, real bounded TLS-handshake abuse limiting on built-in Go TLS, SYSTEM/Doctor/Debug/HSM/Vector operator surfaces, CIDR/L7 traffic controls, full OpenAPI/Positive-Schema lifecycle controls, inventory-backed API-security selectors, and removal of obsolete dual-frontend/model-only foundation code. Status remains **IMPLEMENTED_TESTING_DEFERRED** pending canonical Go 1.25 and required live qualification.
+
+## 2026-09-25 — Production Correctness & Control-Plane Hardening
+
+Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.
+
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/RELEASE_PROCESS.md ./RELEASE_PROCESS.md
--- /mnt/data/waf_prod_hardening_base/RELEASE_PROCESS.md	2026-09-25 12:03:01.539655467 +0000
+++ ./RELEASE_PROCESS.md	2026-09-25 13:33:09.165627822 +0000
@@ -358,3 +358,8 @@
 ## API-8 post-audit hardening release gate
 
 A release candidate containing this hardening must pass `tools/tests/test-api8-post-audit-hardening-source.py` and clean-extract checks proving the removed `web/` and model-only foundation files are absent, the SYSTEM/traffic/API lifecycle Console surfaces are present, and source/manifests/modes match the packaged archive. Dependency-free gates do not replace the mandatory pinned Go 1.25 build/vet/test/race and production qualification requirements.
+
+## 2026-09-25 — Production Correctness & Control-Plane Hardening
+
+Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.
+
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/TESTING.md ./TESTING.md
--- /mnt/data/waf_prod_hardening_base/TESTING.md	2026-09-25 12:03:01.975665097 +0000
+++ ./TESTING.md	2026-09-25 13:33:09.160111570 +0000
@@ -408,3 +408,8 @@
 ## API-8 Post-Audit Hardening gates
 
 Required dependency-free gates include all API-1 through API-8 source gates, `tools/tests/test-api8-post-audit-hardening-source.py`, OpenAI source/isolated tests, WAF package-source, package-builder tests, root Go source-shape, changed-Go gofmt, embedded Admin Console JavaScript syntax, shell syntax, Python syntax, and clean-extract artifact integrity. Canonical Go 1.25 tidy/build/targeted/race remains required before promotion beyond IMPLEMENTED_TESTING_DEFERRED.
+
+## 2026-09-25 — Production Correctness & Control-Plane Hardening
+
+Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.
+
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/TESTING_RESULTS.md ./TESTING_RESULTS.md
--- /mnt/data/waf_prod_hardening_base/TESTING_RESULTS.md	2026-09-25 12:03:02.107668012 +0000
+++ ./TESTING_RESULTS.md	2026-09-25 13:33:09.160950775 +0000
@@ -1103,3 +1103,8 @@
 ## 2026-09-24 — API-8 Post-Audit Hardening
 
 Current exact-source dependency-free results: API source gates 69/47/46/72/33/56/45/58/83/100/110/156/259 PASS, post-audit hardening source gate 134 PASS, OpenAI source contract 16/16 + isolated tests PASS, WAF package-source PASS, package-builder 9/9 PASS, and root Go source-shape 110 files PASS. Canonical Go 1.25 qualification remains BLOCKED_ENVIRONMENT / NOT_RUN because the host has Go 1.23.2 and external toolchain retrieval is unavailable.
+
+## 2026-09-25 — Production Correctness & Control-Plane Hardening
+
+Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.
+
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/admin.go ./admin.go
--- /mnt/data/waf_prod_hardening_base/admin.go	2026-09-25 11:11:25.272560330 +0000
+++ ./admin.go	2026-09-25 12:48:48.660540415 +0000
@@ -164,16 +164,20 @@
 // ── admin server ────────────────────────────────────────────────────────
 
 type adminServer struct {
-	srv      *server
-	token    string
-	sessions *sessionStore
-	audit    *auditLog
-	staged   stagedUpdate
-	log      *slog.Logger
-	started  time.Time
+	srv               *server
+	token             string
+	haPeerToken       string
+	haPeerTokenPinned bool
+	sessions          *sessionStore
+	loginLimiter      *loginAttemptLimiter
+	audit             *auditLog
+	staged            stagedUpdate
+	log               *slog.Logger
+	started           time.Time
 }
 
-func newAdminServer(s *server, token string, log *slog.Logger) *adminServer {
+func newAdminServer(s *server, token, haPeerToken string, log *slog.Logger) *adminServer {
+	haPeerToken = strings.TrimSpace(haPeerToken)
 	if token == "" {
 		b := make([]byte, 24)
 		if _, err := rand.Read(b); err != nil {
@@ -184,8 +188,17 @@
 		// Printed once at startup; not logged again.
 		fmt.Fprintf(os.Stderr, "\n  admin token: %s\n  (set WAF_ADMIN_TOKEN or -admin-token to pin one)\n\n", token)
 	}
-	as := &adminServer{srv: s, token: token, sessions: newSessionStore(), audit: newAuditLog(), log: log, started: time.Now()}
+	as := &adminServer{srv: s, token: token, haPeerToken: haPeerToken, haPeerTokenPinned: haPeerToken != "", sessions: newSessionStore(), loginLimiter: newLoginAttemptLimiter(), audit: newAuditLog(), log: log, started: time.Now()}
 	as.audit.sink = s.syslog.forwardAudit // fan audit entries out to syslog
+	auditPath := strings.TrimSpace(os.Getenv("WAF_ADMIN_AUDIT_FILE"))
+	if auditPath == "" && s.configPath != "" {
+		auditPath = filepath.Join(filepath.Dir(s.configPath), "admin-audit.jsonl")
+	}
+	if auditPath != "" {
+		if err := as.audit.configurePersistence(auditPath, log); err != nil {
+			log.Error("admin audit persistence unavailable", "path", auditPath, "err", err)
+		}
+	}
 	return as
 }
 
@@ -205,6 +218,30 @@
 	return a.authRole("", next)
 }
 
+// haPeerAuth is intentionally narrower than normal admin authentication. HA
+// config replication must use the receiver's break-glass peer token and the
+// dedicated sync protocol marker; browser/admin sessions cannot masquerade as
+// replication traffic merely by setting a header.
+func (a *adminServer) haPeerAuth(next http.HandlerFunc) http.HandlerFunc {
+	return func(w http.ResponseWriter, r *http.Request) {
+		if !a.haPeerTokenPinned {
+			http.Error(w, "HA config sync requires WAF_HA_PEER_TOKEN or -ha-peer-token", http.StatusServiceUnavailable)
+			return
+		}
+		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
+		if got == "" || subtle.ConstantTimeCompare([]byte(got), []byte(a.haPeerToken)) != 1 {
+			http.Error(w, "unauthorized", http.StatusUnauthorized)
+			return
+		}
+		if r.Header.Get("X-WAF-HA-Sync") != "v1" {
+			http.Error(w, "forbidden", http.StatusForbidden)
+			return
+		}
+		id := identity{user: "(ha-peer)", role: roleAdmin}
+		next(w, r.WithContext(context.WithValue(r.Context(), identityKey, id)))
+	}
+}
+
 // authRole is auth plus a minimum-role requirement ("" = any authenticated).
 func (a *adminServer) authRole(minRole string, next http.HandlerFunc) http.HandlerFunc {
 	return func(w http.ResponseWriter, r *http.Request) {
@@ -225,7 +262,19 @@
 			http.Error(w, "forbidden: requires "+minRole, http.StatusForbidden)
 			return
 		}
-		next(w, r.WithContext(context.WithValue(r.Context(), identityKey, id)))
+		r = r.WithContext(context.WithValue(r.Context(), identityKey, id))
+		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
+			next(w, r)
+			return
+		}
+		aw := &mutationAuditWriter{ResponseWriter: w}
+		next(aw, r)
+		if aw.status == 0 {
+			aw.status = http.StatusOK
+		}
+		if aw.status < 400 {
+			a.audit.add(id.user, "http.mutation", r.Method+" "+r.URL.Path)
+		}
 	}
 }
 
@@ -259,21 +308,22 @@
 	mux.HandleFunc("GET /api/config", a.auth(a.handleGetConfig))
 	mux.HandleFunc("GET /api/interfaces", a.auth(a.handleInterfaces))
 	mux.HandleFunc("PUT /api/config", a.authRole(roleOperator, a.handlePutConfig))
-	mux.HandleFunc("POST /api/reload", a.auth(a.handleReload))
+	mux.HandleFunc("POST /api/reload", a.authRole(roleOperator, a.handleReload))
 	mux.HandleFunc("GET /api/matches", a.auth(a.handleMatches))
 	mux.HandleFunc("GET /api/access", a.auth(a.handleAccess))
 	mux.HandleFunc("GET /api/pools", a.auth(a.handlePools))
 	mux.HandleFunc("GET /api/ai/verdicts", a.auth(a.handleAIVerdicts))
 	mux.HandleFunc("GET /api/ai/blocklist", a.auth(a.handleAIBlocklist))
-	mux.HandleFunc("POST /api/ai/unblock", a.auth(a.handleAIUnblock))
-	mux.HandleFunc("POST /api/ai/test", a.auth(a.handleAITest))
+	mux.HandleFunc("POST /api/ai/unblock", a.authRole(roleOperator, a.handleAIUnblock))
+	mux.HandleFunc("POST /api/ai/test", a.authRole(roleOperator, a.handleAITest))
 	mux.HandleFunc("POST /api/syslog/test", a.authRole(roleOperator, a.handleSyslogTest))
 	mux.HandleFunc("GET /api/learn", a.auth(a.handleLearn))
 	mux.HandleFunc("POST /api/learn/apply", a.authRole(roleReviewer, a.handleLearnApply))
-	mux.HandleFunc("POST /api/learn/clear", a.auth(a.handleLearnClear))
+	mux.HandleFunc("POST /api/learn/clear", a.authRole(roleReviewer, a.handleLearnClear))
 	mux.HandleFunc("GET /api/notifications", a.auth(a.handleNotifications))
-	mux.HandleFunc("POST /api/notifications/read", a.auth(a.handleNotifyRead))
-	mux.HandleFunc("POST /api/notifications/dismiss", a.auth(a.handleNotifyDismiss))
+	mux.HandleFunc("POST /api/notifications/read", a.authRole(roleReviewer, a.handleNotifyRead))
+	mux.HandleFunc("POST /api/notifications/dismiss", a.authRole(roleReviewer, a.handleNotifyDismiss))
+	mux.HandleFunc("DELETE /api/notifications/webhook", a.authRole(roleOperator, a.handleNotifyWebhookClear))
 	mux.HandleFunc("POST /api/notifications/apply", a.authRole(roleReviewer, a.handleNotifyApply))
 	mux.HandleFunc("POST /api/login", a.handleLogin) // unauthenticated
 	mux.HandleFunc("POST /api/logout", a.auth(a.handleLogout))
@@ -286,6 +336,7 @@
 	mux.HandleFunc("GET /api/audit", a.auth(a.handleAudit))
 	mux.HandleFunc("GET /api/ha", a.auth(a.handleHA))
 	mux.HandleFunc("POST /api/ha/sync", a.authRole(roleOperator, a.handleHASync))
+	mux.HandleFunc("PUT /api/ha/peer-config", a.haPeerAuth(a.handleHAPeerConfig))
 	mux.HandleFunc("GET /api/pagepolicy", a.auth(a.handlePagePolicies))
 	mux.HandleFunc("GET /api/forms", a.auth(a.handleDiscoveredForms))
 	mux.HandleFunc("GET /api/security/contracts", a.auth(a.handleContracts))
@@ -364,8 +415,8 @@
 	mux.HandleFunc("POST /api/profiles/auto", a.authRole(roleReviewer, a.handleProfileAuto))
 	mux.HandleFunc("GET /api/fs", a.auth(a.handleFS))
 	mux.HandleFunc("GET /api/sitemap", a.auth(a.handleSitemap))
-	mux.HandleFunc("POST /api/crawl", a.auth(a.handleCrawl))
-	mux.HandleFunc("POST /api/sitemap/clear", a.auth(a.handleSitemapClear))
+	mux.HandleFunc("POST /api/crawl", a.authRole(roleOperator, a.handleCrawl))
+	mux.HandleFunc("POST /api/sitemap/clear", a.authRole(roleReviewer, a.handleSitemapClear))
 	mux.HandleFunc("GET /api/discovered", a.auth(a.handleDiscovered))
 	mux.HandleFunc("GET /api/security/operations", a.auth(a.handleAPIOperations))
 	mux.HandleFunc("GET /api/security/operations/{id}", a.auth(a.handleAPIOperationDetail))
@@ -386,8 +437,15 @@
 	return mux
 }
 
-func writeJSON(w http.ResponseWriter, v any) {
+func setJSONSecurityHeaders(w http.ResponseWriter) {
 	w.Header().Set("Content-Type", "application/json")
+	w.Header().Set("Cache-Control", "no-store")
+	w.Header().Set("Pragma", "no-cache")
+	w.Header().Set("X-Content-Type-Options", "nosniff")
+}
+
+func writeJSON(w http.ResponseWriter, v any) {
+	setJSONSecurityHeaders(w)
 	_ = json.NewEncoder(w).Encode(v)
 }
 
@@ -503,13 +561,31 @@
 		http.Error(w, "bad json", http.StatusBadRequest)
 		return
 	}
+	now := time.Now()
+	keys := loginAttemptKeys(r, req.Username)
+	if ok, retry := a.loginLimiter.allow(keys, now); !ok {
+		secs := int(retry.Seconds())
+		if secs < 1 {
+			secs = 1
+		}
+		w.Header().Set("Retry-After", strconv.Itoa(secs))
+		a.audit.add("(login)", "login.throttled", "remote="+adminRemoteHost(r))
+		http.Error(w, "too many login attempts", http.StatusTooManyRequests)
+		return
+	}
 	cfg := a.srv.rt.Load().cfg
 	for _, u := range cfg.Users {
 		if strings.EqualFold(u.Username, req.Username) {
 			if u.Disabled || !verifyPassword(req.Password, u.PasswordHash) {
 				break
 			}
-			tok := a.sessions.create(u.Username, u.Role)
+			tok, err := a.sessions.createWithError(u.Username, u.Role)
+			if err != nil {
+				a.log.Error("session token generation failed", "err", err)
+				http.Error(w, "login unavailable", http.StatusServiceUnavailable)
+				return
+			}
+			a.loginLimiter.success(keys)
 			a.audit.add(u.Username, "login", "")
 			writeJSON(w, map[string]any{"token": tok, "user": u.Username, "role": u.Role})
 			return
@@ -517,6 +593,8 @@
 	}
 	// constant-ish: run a verify against a dummy to blunt user enumeration timing
 	verifyPassword(req.Password, "pbkdf2$sha256$210000$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
+	a.loginLimiter.failure(keys, now)
+	a.audit.add("(login)", "login.failed", "remote="+adminRemoteHost(r))
 	http.Error(w, "invalid credentials", http.StatusUnauthorized)
 }
 
@@ -546,18 +624,22 @@
 	writeJSON(w, out)
 }
 
-// mutateUsers applies a mutation to a copy of the user list, then apply+save.
+// mutateUsers derives the change from the newest runtime under the server's
+// config transaction lock so simultaneous page-policy/config operations cannot
+// be overwritten by a stale user snapshot.
 func (a *adminServer) mutateUsers(fn func(users []UserConfig) ([]UserConfig, error)) error {
-	cfg := a.srv.rt.Load().cfg
-	next, err := fn(append([]UserConfig(nil), cfg.Users...))
+	_, err := a.srv.mutatePersisted(false, func(cfg *Config) error {
+		next, err := fn(append([]UserConfig(nil), cfg.Users...))
+		if err != nil {
+			return err
+		}
+		cfg.Users = next
+		return nil
+	})
 	if err != nil {
-		return err
-	}
-	cfg.Users = next
-	if err := a.srv.apply(cfg); err != nil {
-		return fmt.Errorf("apply failed: %w", err)
+		return fmt.Errorf("apply/persist failed: %w", err)
 	}
-	return saveConfig(a.srv.configPath, cfg)
+	return nil
 }
 
 func (a *adminServer) handleUserCreate(w http.ResponseWriter, r *http.Request) {
@@ -712,8 +794,9 @@
 		limit = 100
 	}
 	writeJSON(w, map[string]any{
-		"unread": a.srv.notify.unreadCount(),
-		"items":  a.srv.notify.list(limit),
+		"unread":           a.srv.notify.unreadCount(),
+		"items":            a.srv.notify.list(limit),
+		"webhook_delivery": a.srv.notify.webhookDeliveryStats(),
 	})
 }
 
@@ -735,6 +818,7 @@
 		return
 	}
 	a.srv.notify.markRead(id, all)
+	a.audit.add(who(r).user, "notification.read", fmt.Sprintf("id=%d all=%t", id, all))
 	writeJSON(w, map[string]any{"ok": true})
 }
 
@@ -744,11 +828,24 @@
 		return
 	}
 	a.srv.notify.dismiss(id, all)
+	a.audit.add(who(r).user, "notification.dismiss", fmt.Sprintf("id=%d all=%t", id, all))
 	writeJSON(w, map[string]any{"ok": true})
 }
 
 // handleNotifyApply executes a notification's attached action (currently only
 // apply_exclusion, which writes a learned exclusion into the site's policy).
+func (a *adminServer) handleNotifyWebhookClear(w http.ResponseWriter, r *http.Request) {
+	if _, err := a.srv.mutatePersisted(false, func(cfg *Config) error {
+		cfg.Notify.WebhookURL = ""
+		return nil
+	}); err != nil {
+		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
+		return
+	}
+	a.audit.add(who(r).user, "notification.webhook_clear", "")
+	writeJSON(w, map[string]any{"ok": true})
+}
+
 func (a *adminServer) handleNotifyApply(w http.ResponseWriter, r *http.Request) {
 	id, _, ok := decodeIDReq(w, r)
 	if !ok {
@@ -803,12 +900,38 @@
 	writeJSON(w, a.srv.ha.status())
 }
 
-func (a *adminServer) handleHASync(w http.ResponseWriter, _ *http.Request) {
+func (a *adminServer) handleHASync(w http.ResponseWriter, r *http.Request) {
 	if !a.srv.ha.snapshotCfg().Enabled {
 		http.Error(w, "HA is disabled", http.StatusBadRequest)
 		return
 	}
 	a.srv.ha.pushConfig(a.srv.rt.Load().cfg)
+	a.audit.add(who(r).user, "ha.sync_requested", "")
+	writeJSON(w, map[string]any{"ok": true})
+}
+
+func (a *adminServer) handleHAPeerConfig(w http.ResponseWriter, r *http.Request) {
+	var env haSyncEnvelope
+	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
+	dec.DisallowUnknownFields()
+	if err := dec.Decode(&env); err != nil {
+		http.Error(w, "bad peer config: "+err.Error(), http.StatusBadRequest)
+		return
+	}
+	if env.Version != haSyncEnvelopeVersion {
+		http.Error(w, "unsupported HA sync version", http.StatusConflict)
+		return
+	}
+	next, err := a.srv.mutatePersisted(true, func(local *Config) error {
+		merged := mergePeerConfig(*local, env.Config)
+		*local = merged
+		return nil
+	})
+	if err != nil {
+		http.Error(w, "peer apply/persist failed: "+err.Error(), http.StatusUnprocessableEntity)
+		return
+	}
+	a.audit.add("(ha-peer)", "ha.peer_config_applied", fmt.Sprintf("%d sites, %d pools, %d policies", len(next.Sites), len(next.Pools), len(next.Policies)))
 	writeJSON(w, map[string]any{"ok": true})
 }
 
@@ -834,6 +957,7 @@
 		return
 	}
 	a.srv.learn.clear(site)
+	a.audit.add(who(r).user, "learner.clear", "site="+site)
 	writeJSON(w, map[string]any{"ok": true})
 }
 
@@ -888,41 +1012,40 @@
 	if pp.Match == "" {
 		pp.Match = "prefix"
 	}
-	cfg := a.srv.rt.Load().cfg // copy
-	si := -1
-	for i := range cfg.Sites {
-		if cfg.Sites[i].Name == siteName {
-			si = i
-			break
+	_, err := a.srv.mutatePersisted(false, func(cfg *Config) error {
+		si := -1
+		for i := range cfg.Sites {
+			if cfg.Sites[i].Name == siteName {
+				si = i
+				break
+			}
 		}
-	}
-	if si < 0 {
-		return fmt.Errorf("unknown site: %s", siteName)
-	}
-	found := -1
-	for i, e := range cfg.Sites[si].PagePolicies {
-		if e.Path == pp.Path && (e.Match == pp.Match || (e.Match == "" && pp.Match == "prefix")) {
-			found = i
-			break
+		if si < 0 {
+			return fmt.Errorf("unknown site: %s", siteName)
 		}
-	}
-	if found >= 0 && mergeExclusions {
-		cfg.Sites[si].PagePolicies[found].ExcludeRuleIDs = unionInts(
-			cfg.Sites[si].PagePolicies[found].ExcludeRuleIDs, pp.ExcludeRuleIDs)
-		if pp.Note != "" {
-			cfg.Sites[si].PagePolicies[found].Note = pp.Note
-		}
-	} else if found >= 0 {
-		pp.Source = firstNonEmpty(pp.Source, cfg.Sites[si].PagePolicies[found].Source)
-		cfg.Sites[si].PagePolicies[found] = pp
-	} else {
-		cfg.Sites[si].PagePolicies = append(cfg.Sites[si].PagePolicies, pp)
-	}
-	if err := a.srv.apply(cfg); err != nil {
-		return fmt.Errorf("apply failed: %w", err)
-	}
-	if err := saveConfig(a.srv.configPath, cfg); err != nil {
-		return fmt.Errorf("applied but not persisted: %w", err)
+		found := -1
+		for i, e := range cfg.Sites[si].PagePolicies {
+			if e.Path == pp.Path && (e.Match == pp.Match || (e.Match == "" && pp.Match == "prefix")) {
+				found = i
+				break
+			}
+		}
+		if found >= 0 && mergeExclusions {
+			cfg.Sites[si].PagePolicies[found].ExcludeRuleIDs = unionInts(
+				cfg.Sites[si].PagePolicies[found].ExcludeRuleIDs, pp.ExcludeRuleIDs)
+			if pp.Note != "" {
+				cfg.Sites[si].PagePolicies[found].Note = pp.Note
+			}
+		} else if found >= 0 {
+			pp.Source = firstNonEmpty(pp.Source, cfg.Sites[si].PagePolicies[found].Source)
+			cfg.Sites[si].PagePolicies[found] = pp
+		} else {
+			cfg.Sites[si].PagePolicies = append(cfg.Sites[si].PagePolicies, pp)
+		}
+		return nil
+	})
+	if err != nil {
+		return fmt.Errorf("apply/persist failed: %w", err)
 	}
 	a.log.Info("page policy saved", "site", siteName, "path", pp.Path)
 	return nil
@@ -1131,34 +1254,35 @@
 		http.Error(w, "bad json", http.StatusBadRequest)
 		return
 	}
-	cfg := a.srv.rt.Load().cfg
-	si := -1
-	for i := range cfg.Sites {
-		if cfg.Sites[i].Name == req.Site {
-			si = i
-			break
+	if _, err := a.srv.mutatePersisted(false, func(cfg *Config) error {
+		si := -1
+		for i := range cfg.Sites {
+			if cfg.Sites[i].Name == req.Site {
+				si = i
+				break
+			}
 		}
-	}
-	if si < 0 {
-		http.Error(w, "unknown site", http.StatusNotFound)
-		return
-	}
-	out := cfg.Sites[si].PagePolicies[:0]
-	for _, e := range cfg.Sites[si].PagePolicies {
-		if e.Path == req.Path {
-			continue
+		if si < 0 {
+			return fmt.Errorf("unknown site")
+		}
+		out := make([]PagePolicy, 0, len(cfg.Sites[si].PagePolicies))
+		for _, e := range cfg.Sites[si].PagePolicies {
+			if e.Path == req.Path && (req.Match == "" || e.Match == req.Match) {
+				continue
+			}
+			out = append(out, e)
+		}
+		cfg.Sites[si].PagePolicies = out
+		return nil
+	}); err != nil {
+		if strings.Contains(err.Error(), "unknown site") {
+			http.Error(w, "unknown site", http.StatusNotFound)
+		} else {
+			http.Error(w, "apply/persist failed: "+err.Error(), http.StatusUnprocessableEntity)
 		}
-		out = append(out, e)
-	}
-	cfg.Sites[si].PagePolicies = out
-	if err := a.srv.apply(cfg); err != nil {
-		http.Error(w, "apply failed: "+err.Error(), http.StatusUnprocessableEntity)
-		return
-	}
-	if err := saveConfig(a.srv.configPath, cfg); err != nil {
-		http.Error(w, "applied but not persisted: "+err.Error(), http.StatusInternalServerError)
 		return
 	}
+	a.audit.add(who(r).user, "pagepolicy.delete", "site="+req.Site+" path="+req.Path)
 	writeJSON(w, map[string]any{"ok": true})
 }
 
@@ -1187,6 +1311,7 @@
 		return
 	}
 	a.srv.ai.unblock(req.Site, req.IP)
+	a.audit.add(who(r).user, "ai.unblock", "site="+req.Site)
 	writeJSON(w, map[string]any{"ok": true})
 }
 
@@ -1204,6 +1329,7 @@
 		http.Error(w, err.Error(), http.StatusBadGateway)
 		return
 	}
+	a.audit.add(who(r).user, "ai.connector_test", "")
 	writeJSON(w, v)
 }
 
@@ -1235,19 +1361,30 @@
 	if root == "" {
 		root = "/etc"
 	}
+	root = filepath.Clean(root)
+	resolvedRoot, err := filepath.EvalSymlinks(root)
+	if err != nil {
+		http.Error(w, "cannot resolve browse root: "+err.Error(), http.StatusInternalServerError)
+		return
+	}
 	p := r.URL.Query().Get("path")
 	if p == "" {
-		p = root
+		p = resolvedRoot
 	}
 	if !filepath.IsAbs(p) {
-		p = filepath.Join(root, p)
+		p = filepath.Join(resolvedRoot, p)
 	}
 	p = filepath.Clean(p)
-
-	// Clamp inside root: reject anything that resolves above it.
-	if rel, err := filepath.Rel(root, p); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
-		p = root
+	resolved, err := filepath.EvalSymlinks(p)
+	if err != nil {
+		http.Error(w, "cannot resolve path: "+err.Error(), http.StatusBadRequest)
+		return
 	}
+	if rel, err := filepath.Rel(resolvedRoot, resolved); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
+		http.Error(w, "path escapes browse root", http.StatusForbidden)
+		return
+	}
+	p = resolved
 
 	info, err := os.Stat(p)
 	if err != nil {
@@ -1300,10 +1437,28 @@
 	})
 }
 
-func (a *adminServer) handleGetConfig(w http.ResponseWriter, _ *http.Request) {
-	c := redactAISecrets(a.srv.rt.Load().cfg)
+func (a *adminServer) handleGetConfig(w http.ResponseWriter, r *http.Request) {
+	rt := a.srv.rt.Load()
+	if rt == nil {
+		http.Error(w, "runtime unavailable", http.StatusServiceUnavailable)
+		return
+	}
+	c := rt.cfg
+	if r.URL.Query().Get("draft") == "1" {
+		if draft, ok, err := loadDraftConfig(a.srv.configPath); err != nil {
+			http.Error(w, "load draft: "+err.Error(), http.StatusInternalServerError)
+			return
+		} else if ok {
+			c = draft
+			w.Header().Set("X-WAF-Config-Source", "draft")
+		} else {
+			w.Header().Set("X-WAF-Config-Source", "live")
+		}
+	}
+	c = redactAISecrets(c)
 	c.HA.PeerToken = ""
 	c = redactHSMSecretRefs(c)
+	c = redactNotifySecrets(c)
 	users := make([]UserConfig, len(c.Users)) // copy with hashes blanked
 	for i, u := range c.Users {
 		u.PasswordHash = ""
@@ -1352,36 +1507,80 @@
 	// Preserve the historical passive-discovery behavior for older API clients
 	// that submit a full config without the newly added toggle.
 	c := Config{PassiveDiscoveryEnabled: true}
-	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&c); err != nil {
+	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
+	dec.DisallowUnknownFields()
+	if err := dec.Decode(&c); err != nil {
 		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
 		return
 	}
 	migrateTrustedProxyConfig(&c)
-	// Blank secrets on submit mean "keep the current ones" — the UI never
-	// receives stored secrets, so it can't echo them back.
-	cur := a.srv.rt.Load().cfg
-	preserveAISecrets(cur, &c)
-	if c.HA.PeerToken == "" {
-		c.HA.PeerToken = cur.HA.PeerToken
-	}
-	preserveHSMSecretRefs(cur, &c)
-	// Users are managed only via the dedicated user endpoints; a general config
-	// save never touches them (the console can't see the hashes anyway).
-	c.Users = cur.Users
-
-	// Draft save: persist work-in-progress to disk WITHOUT applying it to the
-	// live engine. Lets you build a config incrementally (add a node, save; add
-	// a pool, save) without every intermediate state being fully consistent.
-	// The running WAF keeps serving the last APPLIED config untouched.
-	if r.URL.Query().Get("draft") == "1" {
-		if err := c.validateDraft(); err != nil {
-			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
-			return
+	draft := r.URL.Query().Get("draft") == "1"
+
+	// Full-config submissions still contain redacted secret fields and no user
+	// password hashes. Preserve those node-local values from the newest runtime
+	// while holding applyMu, then validate and persist in the same transaction.
+	// This prevents a concurrent user/secret mutation from being lost between a
+	// stale GET /api/config snapshot and this PUT.
+	errKind := ""
+	txnErr := func() error {
+		a.srv.applyMu.Lock()
+		defer a.srv.applyMu.Unlock()
+		rt := a.srv.rt.Load()
+		if rt == nil {
+			errKind = "runtime"
+			return errors.New("runtime unavailable")
+		}
+		cur := rt.cfg
+		preserveAISecrets(cur, &c)
+		preserveNotifySecrets(cur, &c)
+		if c.HA.PeerToken == "" {
+			c.HA.PeerToken = cur.HA.PeerToken
+		}
+		preserveHSMSecretRefs(cur, &c)
+		// Users are managed only through the dedicated user endpoints; a general
+		// config save can never replace password hashes with redacted UI data.
+		c.Users = append([]UserConfig(nil), cur.Users...)
+		if c.HA.Enabled && c.HA.SyncConfig && !a.haPeerTokenPinned {
+			errKind = "validation"
+			return errors.New("HA config sync requires a dedicated WAF_HA_PEER_TOKEN or -ha-peer-token before enabling sync")
+		}
+		if draft {
+			if err := c.validateDraft(); err != nil {
+				errKind = "validation"
+				return err
+			}
+			// A draft is durable operator work-in-progress, not startup authority.
+			// Persist it separately so a restart cannot silently promote an
+			// unapplied Console draft to live traffic.
+			if err := saveConfig(draftConfigPath(a.srv.configPath), c); err != nil {
+				errKind = "persist"
+				return err
+			}
+			return nil
 		}
-		if err := saveConfig(a.srv.configPath, c); err != nil {
-			http.Error(w, "not persisted: "+err.Error(), http.StatusInternalServerError)
-			return
+		if err := c.validate(); err != nil {
+			errKind = "validation"
+			return err
+		}
+		if err := a.srv.applyPersistedLocked(c, false); err != nil {
+			errKind = "apply"
+			return err
+		}
+		return nil
+	}()
+	if txnErr != nil {
+		switch errKind {
+		case "runtime":
+			http.Error(w, txnErr.Error(), http.StatusServiceUnavailable)
+		case "persist":
+			http.Error(w, "not persisted: "+txnErr.Error(), http.StatusInternalServerError)
+		default:
+			http.Error(w, txnErr.Error(), http.StatusUnprocessableEntity)
 		}
+		return
+	}
+
+	if draft {
 		a.audit.add(who(r).user, "config.draft_saved", fmt.Sprintf("%d sites, %d pools, %d nodes", len(c.Sites), len(c.Pools), len(c.Nodes)))
 		applyErr := ""
 		if err := c.validate(); err != nil {
@@ -1390,46 +1589,38 @@
 		resp := redactAISecrets(c)
 		resp.HA.PeerToken = ""
 		resp = redactHSMSecretRefs(resp)
+		resp = redactNotifySecrets(resp)
 		writeJSON(w, map[string]any{"config": resp, "draft": true, "apply_ready": applyErr == "", "apply_error": applyErr})
 		return
 	}
 
-	if err := c.validate(); err != nil {
-		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
-		return
-	}
-	// A PUT carrying X-WAF-Sync came from the HA peer: apply without pushing
-	// it back (loop guard).
-	fromSync := r.Header.Get("X-WAF-Sync") == "1"
-	if err := a.srv.applyEx(c, fromSync); err != nil {
-		http.Error(w, "apply failed: "+err.Error(), http.StatusUnprocessableEntity)
-		return
-	}
-	if err := saveConfig(a.srv.configPath, c); err != nil {
-		http.Error(w, "applied but not persisted: "+err.Error(), http.StatusInternalServerError)
-		return
-	}
-	a.log.Info("config applied via admin API", "engine_mode", c.EngineMode, "sites", len(c.Sites), "from_sync", fromSync)
-	if !fromSync {
-		a.audit.add(who(r).user, "config.apply", fmt.Sprintf("%d sites, %d pools, %d policies", len(c.Sites), len(c.Pools), len(c.Policies)))
+	draftCleanupWarning := ""
+	if err := removeConfigFileDurable(draftConfigPath(a.srv.configPath)); err != nil {
+		draftCleanupWarning = err.Error()
+		a.log.Warn("applied config but stale draft cleanup failed", "err", err)
 	}
+	a.log.Info("config applied via admin API", "engine_mode", c.EngineMode, "sites", len(c.Sites))
+	a.audit.add(who(r).user, "config.apply", fmt.Sprintf("%d sites, %d pools, %d policies", len(c.Sites), len(c.Pools), len(c.Policies)))
 	resp := redactAISecrets(c)
 	resp.HA.PeerToken = ""
 	resp = redactHSMSecretRefs(resp)
+	resp = redactNotifySecrets(resp)
 	writeJSON(w, map[string]any{
-		"config":           resp,
-		"applied":          true,
-		"restart_required": a.srv.restartPending(c),
+		"config":                resp,
+		"applied":               true,
+		"restart_required":      a.srv.restartPending(c),
+		"draft_cleanup_warning": draftCleanupWarning,
 	})
 }
 
-func (a *adminServer) handleReload(w http.ResponseWriter, _ *http.Request) {
+func (a *adminServer) handleReload(w http.ResponseWriter, r *http.Request) {
 	cfg := a.srv.rt.Load().cfg
 	if err := a.srv.apply(cfg); err != nil {
 		http.Error(w, "reload failed: "+err.Error(), http.StatusUnprocessableEntity)
 		return
 	}
 	a.log.Info("rules reloaded via admin API", "rules", cfg.Rules)
+	a.audit.add(who(r).user, "config.reload", "rules="+cfg.Rules)
 	writeJSON(w, map[string]any{"ok": true})
 }
 
@@ -1593,6 +1784,7 @@
 	}
 	a.log.Info("crawl started", "site", site.Name, "backend", backend.String(),
 		"max_pages", opts.maxPages, "max_depth", opts.maxDepth)
+	a.audit.add(who(r).user, "sitemap.crawl", fmt.Sprintf("site=%s max_pages=%d max_depth=%d", site.Name, opts.maxPages, opts.maxDepth))
 	writeJSON(w, map[string]any{"started": true, "site": site.Name})
 }
 
@@ -1602,10 +1794,22 @@
 		http.Error(w, "site query param required", http.StatusBadRequest)
 		return
 	}
+	before := a.srv.maps.snapshot(site)
+	if before.Crawl.Running {
+		http.Error(w, "cannot clear site map while crawl is running", http.StatusConflict)
+		return
+	}
 	a.srv.maps.clear(site)
-	_ = a.srv.maps.save(a.srv.configPath) // persist the cleared state
+	if err := a.srv.maps.save(a.srv.configPath); err != nil {
+		a.srv.maps.restoreSnapshot(before)
+		http.Error(w, "clear not persisted; in-memory map restored: "+err.Error(), http.StatusInternalServerError)
+		return
+	}
+	// Only clear dependent in-memory learning after the authoritative map
+	// snapshot has durably accepted the destructive change.
 	a.srv.signals.clear(site)
 	a.srv.learn.clear(site)
+	a.audit.add(who(r).user, "sitemap.clear", "site="+site)
 	writeJSON(w, map[string]any{"ok": true})
 }
 
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/admin_login_hardening.go ./admin_login_hardening.go
--- /mnt/data/waf_prod_hardening_base/admin_login_hardening.go	1970-01-01 00:00:00.000000000 +0000
+++ ./admin_login_hardening.go	2026-09-25 12:12:59.771683747 +0000
@@ -0,0 +1,142 @@
+package main
+
+import (
+	"net"
+	"net/http"
+	"strings"
+	"sync"
+	"time"
+)
+
+const (
+	loginAttemptWindow = 5 * time.Minute
+	loginBlockDuration = 5 * time.Minute
+	loginAttemptLimit  = 8
+	loginAttemptMaxKey = 4096
+)
+
+type loginAttempt struct {
+	Count        int
+	WindowStart  time.Time
+	BlockedUntil time.Time
+}
+
+type loginAttemptLimiter struct {
+	mu       sync.Mutex
+	entries  map[string]loginAttempt
+	overflow loginAttempt
+}
+
+func newLoginAttemptLimiter() *loginAttemptLimiter {
+	return &loginAttemptLimiter{entries: map[string]loginAttempt{}}
+}
+
+func adminRemoteHost(r *http.Request) string {
+	host := strings.TrimSpace(r.RemoteAddr)
+	if h, _, err := net.SplitHostPort(host); err == nil {
+		host = h
+	}
+	if len(host) > 128 {
+		host = host[:128]
+	}
+	return host
+}
+
+func loginAttemptKeys(r *http.Request, username string) []string {
+	u := strings.ToLower(strings.TrimSpace(username))
+	if len(u) > 128 {
+		u = u[:128]
+	}
+	return []string{"ip:" + adminRemoteHost(r), "user:" + u}
+}
+
+func (l *loginAttemptLimiter) pruneLocked(now time.Time) {
+	for k, v := range l.entries {
+		if now.After(v.BlockedUntil) && now.Sub(v.WindowStart) > loginAttemptWindow {
+			delete(l.entries, k)
+		}
+	}
+	if !l.overflow.WindowStart.IsZero() && now.After(l.overflow.BlockedUntil) && now.Sub(l.overflow.WindowStart) > loginAttemptWindow {
+		l.overflow = loginAttempt{}
+	}
+}
+
+func (l *loginAttemptLimiter) stateLocked(key string) (loginAttempt, bool) {
+	if v, ok := l.entries[key]; ok {
+		return v, false
+	}
+	if len(l.entries) >= loginAttemptMaxKey {
+		return l.overflow, true
+	}
+	return loginAttempt{}, false
+}
+
+func (l *loginAttemptLimiter) storeLocked(key string, v loginAttempt, overflow bool) {
+	if overflow {
+		l.overflow = v
+		return
+	}
+	l.entries[key] = v
+}
+
+func (l *loginAttemptLimiter) allow(keys []string, now time.Time) (bool, time.Duration) {
+	l.mu.Lock()
+	defer l.mu.Unlock()
+	l.pruneLocked(now)
+	var retry time.Duration
+	for _, k := range keys {
+		v, _ := l.stateLocked(k)
+		if now.Before(v.BlockedUntil) {
+			r := v.BlockedUntil.Sub(now)
+			if r > retry {
+				retry = r
+			}
+		}
+	}
+	return retry <= 0, retry
+}
+
+func (l *loginAttemptLimiter) failure(keys []string, now time.Time) {
+	l.mu.Lock()
+	defer l.mu.Unlock()
+	l.pruneLocked(now)
+	for _, k := range keys {
+		v, overflow := l.stateLocked(k)
+		if v.WindowStart.IsZero() || now.Sub(v.WindowStart) > loginAttemptWindow {
+			v = loginAttempt{WindowStart: now}
+		}
+		v.Count++
+		if v.Count >= loginAttemptLimit {
+			v.BlockedUntil = now.Add(loginBlockDuration)
+		}
+		l.storeLocked(k, v, overflow)
+	}
+}
+
+func (l *loginAttemptLimiter) success(keys []string) {
+	l.mu.Lock()
+	defer l.mu.Unlock()
+	for _, k := range keys {
+		delete(l.entries, k)
+	}
+}
+
+// mutationAuditWriter captures status without changing Admin API response bodies.
+type mutationAuditWriter struct {
+	http.ResponseWriter
+	status int
+}
+
+func (w *mutationAuditWriter) WriteHeader(code int) {
+	if w.status == 0 {
+		w.status = code
+	}
+	w.ResponseWriter.WriteHeader(code)
+}
+
+func (w *mutationAuditWriter) Write(b []byte) (int, error) {
+	if w.status == 0 {
+		w.status = http.StatusOK
+	}
+	return w.ResponseWriter.Write(b)
+}
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/api_security_persist.go ./api_security_persist.go
--- /mnt/data/waf_prod_hardening_base/api_security_persist.go	2026-09-25 11:11:25.273722422 +0000
+++ ./api_security_persist.go	2026-09-25 11:36:44.234542758 +0000
@@ -3,6 +3,7 @@
 import (
 	"encoding/json"
 	"errors"
+	"fmt"
 	"log/slog"
 	"os"
 	"path/filepath"
@@ -12,6 +13,16 @@
 
 const apiSecurityStateVersion = 1
 
+func validateAPISecurityStateVersion(kind string, version int) error {
+	// Version 0 is the pre-versioning legacy form and remains readable for
+	// upgrade compatibility. Any future/non-canonical version fails closed so
+	// a downgraded binary cannot silently reinterpret newer durable state.
+	if version != 0 && version != apiSecurityStateVersion {
+		return fmt.Errorf("%s state version %d is unsupported (expected %d or legacy 0)", kind, version, apiSecurityStateVersion)
+	}
+	return nil
+}
+
 type apiOperationsStateFile struct {
 	Version int            `json:"version"`
 	Saved   time.Time      `json:"saved"`
@@ -113,6 +124,9 @@
 	if err := readJSONIfExists(apiSecurityStatePath(configPath, "api-operations.json"), &state); err != nil {
 		return err
 	}
+	if err := validateAPISecurityStateVersion("api-operations", state.Version); err != nil {
+		return err
+	}
 	if len(state.Ops) == 0 {
 		return nil
 	}
@@ -158,6 +172,9 @@
 	if err := readJSONIfExists(apiSecurityStatePath(configPath, "api-schema.json"), &state); err != nil {
 		return err
 	}
+	if err := validateAPISecurityStateVersion("api-schema", state.Version); err != nil {
+		return err
+	}
 	return s.restorePersistence(state)
 }
 
@@ -188,6 +205,9 @@
 	if err := readJSONIfExists(apiSecurityStatePath(configPath, "api-contracts.json"), &state); err != nil {
 		return err
 	}
+	if err := validateAPISecurityStateVersion("api-contracts", state.Version); err != nil {
+		return err
+	}
 	if len(state.Contracts) == 0 && len(state.Versions) == 0 {
 		return nil
 	}
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/block_response.go ./block_response.go
--- /mnt/data/waf_prod_hardening_base/block_response.go	2026-09-25 11:11:25.273889845 +0000
+++ ./block_response.go	2026-09-25 11:15:31.083901753 +0000
@@ -2,6 +2,7 @@
 
 import (
 	"encoding/json"
+	"html/template"
 	"net/http"
 	"time"
 )
@@ -13,11 +14,15 @@
 	Timestamp string `json:"timestamp"`
 }
 
+var blockPageTemplate = template.Must(template.New("block-page").Parse(`<!doctype html><html><head><meta charset="utf-8"><title>Request blocked</title></head><body><h1>Request blocked</h1><p>{{.Reason}}</p><p>Request ID: {{.RequestID}}</p></body></html>`))
+
 func writeBlockResponse(w http.ResponseWriter, r *http.Request, status int, reason string, jsonMode bool) {
 	id := requestCorrelationID(r)
 	if id != "" {
 		w.Header().Set("X-Request-ID", id)
 	}
+	w.Header().Set("Cache-Control", "no-store")
+	w.Header().Set("X-Content-Type-Options", "nosniff")
 	if jsonMode || r.Header.Get("Accept") == "application/json" {
 		w.Header().Set("Content-Type", "application/json")
 		w.WriteHeader(status)
@@ -25,6 +30,8 @@
 		return
 	}
 	w.Header().Set("Content-Type", "text/html; charset=utf-8")
+	w.Header().Set("Content-Security-Policy", "default-src 'none'; base-uri 'none'; frame-ancestors 'none'")
+	w.Header().Set("X-Frame-Options", "DENY")
 	w.WriteHeader(status)
-	_, _ = w.Write([]byte("<html><body><h1>Request blocked</h1><p>" + reason + "</p><p>Request ID: " + id + "</p></body></html>"))
+	_ = blockPageTemplate.Execute(w, blockResponse{Reason: reason, RequestID: id})
 }
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/build.sh ./build.sh
--- /mnt/data/waf_prod_hardening_base/build.sh	2026-09-25 11:11:25.274302827 +0000
+++ ./build.sh	2026-09-25 11:32:52.799051363 +0000
@@ -30,6 +30,7 @@
 python3 ./tools/tests/test-api74-source.py
 python3 ./tools/tests/test-api8-source.py
 python3 ./tools/tests/test-api8-post-audit-hardening-source.py
+python3 ./tools/tests/test-production-control-plane-hardening-source.py
 
 VERSION="${VERSION:-$(date -u +%Y.%m.%d)}"
 COMMIT="${COMMIT:-unknown}"
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/config.sample.json ./config.sample.json
--- /mnt/data/waf_prod_hardening_base/config.sample.json	2026-09-25 11:11:25.275791402 +0000
+++ ./config.sample.json	2026-09-25 11:37:39.411230718 +0000
@@ -6,6 +6,17 @@
   "backend_timeout_sec": 30,
   "passive_discovery_enabled": true,
   "trusted_proxy_cidrs": [],
+  "l7_abuse": {
+    "enabled": false,
+    "requests_per_window": 0,
+    "window_seconds": 60,
+    "max_concurrent_connections": 0,
+    "tls_handshake_per_window": 0
+  },
+  "cidr_policy": {
+    "enabled": false,
+    "rules": []
+  },
   "hsm": {
     "allowed_module_dirs": [
       "/usr/lib",
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/correlation.go ./correlation.go
--- /mnt/data/waf_prod_hardening_base/correlation.go	2026-09-25 11:11:25.276282633 +0000
+++ ./correlation.go	2026-09-25 11:15:31.083811057 +0000
@@ -5,16 +5,41 @@
 	"crypto/rand"
 	"encoding/hex"
 	"net/http"
+	"strconv"
+	"time"
 )
 
 type requestCorrelationKey struct{}
 
+func validRequestCorrelationID(id string) bool {
+	if len(id) < 8 || len(id) > 128 {
+		return false
+	}
+	for i := 0; i < len(id); i++ {
+		c := id[i]
+		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == ':' {
+			continue
+		}
+		return false
+	}
+	return true
+}
+
+func newRequestCorrelationID() string {
+	var b [16]byte
+	if _, err := rand.Read(b[:]); err == nil {
+		return hex.EncodeToString(b[:])
+	}
+	// Correlation IDs are not authentication tokens. A monotonic-ish local
+	// fallback is safer than accepting attacker-controlled bytes when the CSPRNG
+	// is unavailable.
+	return "req-" + strconv.FormatInt(time.Now().UnixNano(), 36)
+}
+
 func withRequestCorrelationID(r *http.Request) *http.Request {
 	id := r.Header.Get("X-Request-ID")
-	if len(id) < 8 || len(id) > 128 {
-		var b [16]byte
-		_, _ = rand.Read(b[:])
-		id = hex.EncodeToString(b[:])
+	if !validRequestCorrelationID(id) {
+		id = newRequestCorrelationID()
 	}
 	return r.WithContext(context.WithValue(r.Context(), requestCorrelationKey{}, id))
 }
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/debug_bundle.go ./debug_bundle.go
--- /mnt/data/waf_prod_hardening_base/debug_bundle.go	2026-09-25 11:11:25.276357877 +0000
+++ ./debug_bundle.go	2026-09-25 11:15:31.084815702 +0000
@@ -449,7 +449,7 @@
 
 func clientIdentityEvidence(r *http.Request) map[string]any {
 	if d, ok := clientIdentityFromContext(r.Context()); ok {
-		return map[string]any{"remote_addr": d.RemoteAddr, "resolved_client_ip": d.ResolvedClientIP, "source": d.Source, "trusted_proxy": d.TrustedProxy, "decision": d.Decision, "rejection_reason": d.RejectionReason}
+		return map[string]any{"remote_addr": maskedDebugClientIP(d.RemoteAddr), "resolved_client_ip": maskedDebugClientIP(d.ResolvedClientIP), "source": d.Source, "trusted_proxy": d.TrustedProxy, "decision": d.Decision, "rejection_reason": d.RejectionReason}
 	}
-	return map[string]any{"remote_addr": clientIP(r), "source": "REMOTE_ADDR"}
+	return map[string]any{"remote_addr": maskedDebugClientIP(clientIP(r)), "source": "REMOTE_ADDR"}
 }
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/ha.go ./ha.go
--- /mnt/data/waf_prod_hardening_base/ha.go	2026-09-25 11:11:25.277468932 +0000
+++ ./ha.go	2026-09-25 12:30:03.240256496 +0000
@@ -1,227 +1,305 @@
-package main
-
-// High availability: config sync + failover role.
-//
-// Scope, stated honestly:
-//   * waf-proxy coordinates STATE and ROLE between two instances. On every
-//     config apply it pushes the config to the peer, which validates+applies
-//     it. It polls the peer's health and computes whether it should consider
-//     itself active or standby.
-//   * waf-proxy does NOT move IP addresses. Actual packet failover (who
-//     answers on the VIP) belongs to keepalived/VRRP or a load balancer, which
-//     can consume this instance's role via GET /api/ha (or /healthz). Building
-//     an in-process VIP grab would be a dishonest half-solution.
-//
-// Blocklist state is intentionally NOT synced (per design): each node makes its
-// own AI decisions.
-
-import (
-	"bytes"
-	"context"
-	"encoding/json"
-	"fmt"
-	"log/slog"
-	"net/http"
-	"sync"
-	"time"
-)
-
-type HAConfig struct {
-	Enabled   bool   `json:"enabled"`
-	Role      string `json:"role"`       // primary | secondary — tie-breaker for split-brain
-	PeerURL   string `json:"peer_url"`   // e.g. https://10.0.0.6:9090
-	PeerToken string `json:"peer_token"` // admin bearer token of the peer (masked on read)
-	SyncConfig bool  `json:"sync_config"`
-}
-
-func defaultHAConfig() HAConfig {
-	return HAConfig{Role: "primary", SyncConfig: true}
-}
-
-func (c HAConfig) validate() error {
-	if !c.Enabled {
-		return nil
-	}
-	if c.Role != "primary" && c.Role != "secondary" {
-		return fmt.Errorf("ha: role must be primary or secondary")
-	}
-	if c.PeerURL == "" {
-		return fmt.Errorf("ha: peer_url is required when enabled")
-	}
-	return nil
-}
-
-type haState struct {
-	PeerUp    bool      `json:"peer_up"`
-	Role      string    `json:"role"`      // active | standby | solo
-	LastSync  string    `json:"last_sync"` // result text
-	LastSeen  time.Time `json:"-"`
-	LastError string    `json:"last_error,omitempty"`
-}
-
-type haEngine struct {
-	mu     sync.Mutex
-	cfg    HAConfig
-	state  haState
-	client *http.Client
-	log    *slog.Logger
-	notify *notifier
-
-	syncGate gate
-}
-
-func newHAEngine(log *slog.Logger, n *notifier) *haEngine {
-	return &haEngine{
-		cfg:    defaultHAConfig(),
-		state:  haState{Role: "solo"},
-		client: &http.Client{Timeout: 5 * time.Second},
-		log:    log,
-		notify: n,
-	}
-}
-
-func (h *haEngine) configure(c HAConfig) {
-	h.mu.Lock()
-	defer h.mu.Unlock()
-	h.cfg = c
-	if !c.Enabled {
-		h.state = haState{Role: "solo"}
-	}
-}
-
-func (h *haEngine) snapshotCfg() HAConfig {
-	h.mu.Lock()
-	defer h.mu.Unlock()
-	return h.cfg
-}
-
-func (h *haEngine) status() haState {
-	h.mu.Lock()
-	defer h.mu.Unlock()
-	return h.state
-}
-
-// run polls peer health and updates role until ctx is cancelled.
-func (h *haEngine) run(ctx context.Context) {
-	t := time.NewTicker(3 * time.Second)
-	defer t.Stop()
-	for {
-		select {
-		case <-ctx.Done():
-			return
-		case <-t.C:
-			h.tick(ctx)
-		}
-	}
-}
-
-func (h *haEngine) tick(ctx context.Context) {
-	cfg := h.snapshotCfg()
-	if !cfg.Enabled {
-		return
-	}
-	up := h.pingPeer(ctx, cfg)
-
-	h.mu.Lock()
-	prevUp := h.state.PeerUp
-	h.state.PeerUp = up
-	if up {
-		h.state.LastSeen = time.Now()
-		// Peer alive: primary is active, secondary is standby.
-		if cfg.Role == "primary" {
-			h.state.Role = "active"
-		} else {
-			h.state.Role = "standby"
-		}
-	} else {
-		// Peer down: we take over regardless of role.
-		h.state.Role = "active"
-	}
-	role := h.state.Role
-	h.mu.Unlock()
-
-	if prevUp != up {
-		if up {
-			h.notify.push(notifyPeer, "info", "HA peer is up", "peer reachable; role="+role, "ha_peer", "", nil)
-		} else {
-			h.notify.push(notifyPeer, "alert", "HA peer is DOWN", "peer unreachable — this node is now active", "ha_peer", "", nil)
-		}
-	}
-}
-
-func (h *haEngine) pingPeer(ctx context.Context, cfg HAConfig) bool {
-	req, err := http.NewRequestWithContext(ctx, http.MethodGet, trimSlash(cfg.PeerURL)+"/healthz", nil)
-	if err != nil {
-		return false
-	}
-	resp, err := h.client.Do(req)
-	if err != nil {
-		return false
-	}
-	defer resp.Body.Close()
-	return resp.StatusCode == http.StatusOK
-}
-
-// pushConfig sends the given config to the peer's admin API. Called after a
-// successful local apply. Best-effort: a peer error is surfaced, not fatal.
-func (h *haEngine) pushConfig(cfg Config) {
-	hc := h.snapshotCfg()
-	if !hc.Enabled || !hc.SyncConfig || hc.PeerURL == "" {
-		return
-	}
-	if !h.syncGate.enter() {
-		return // a sync is already in flight
-	}
-	go func() {
-		defer h.syncGate.leave()
-		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
-		defer cancel()
-
-		// Mark the payload so the peer doesn't echo it back to us (loop guard).
-		body, _ := json.Marshal(cfg)
-		req, err := http.NewRequestWithContext(ctx, http.MethodPut,
-			trimSlash(hc.PeerURL)+"/api/config", bytes.NewReader(body))
-		if err != nil {
-			h.recordSync("error: "+err.Error(), true)
-			return
-		}
-		req.Header.Set("Content-Type", "application/json")
-		req.Header.Set("Authorization", "Bearer "+hc.PeerToken)
-		req.Header.Set("X-WAF-Sync", "1") // peer treats this as a sync, won't re-push
-		resp, err := h.client.Do(req)
-		if err != nil {
-			h.recordSync("error: "+err.Error(), true)
-			return
-		}
-		defer resp.Body.Close()
-		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
-			h.recordSync(fmt.Sprintf("peer http %d", resp.StatusCode), true)
-			return
-		}
-		h.recordSync("ok "+time.Now().Format("15:04:05"), false)
-	}()
-}
-
-func (h *haEngine) recordSync(text string, isErr bool) {
-	h.mu.Lock()
-	h.state.LastSync = text
-	if isErr {
-		h.state.LastError = text
-	} else {
-		h.state.LastError = ""
-	}
-	h.mu.Unlock()
-	if isErr {
-		h.notify.push(notifySync, "warn", "Config sync failed", text, "ha_sync_err", "", nil)
-		h.log.Warn("ha config sync failed", "detail", text)
-	} else {
-		h.notify.push(notifySync, "info", "Config synced to peer", text, "", "", nil)
-	}
-}
-
-func trimSlash(s string) string {
-	for len(s) > 0 && s[len(s)-1] == '/' {
-		s = s[:len(s)-1]
-	}
-	return s
-}
+package main
+
+// High availability: config sync + failover role.
+//
+// Scope, stated honestly:
+//   * waf-proxy coordinates STATE and ROLE between two instances. On every
+//     config apply it pushes the config to the peer, which validates+applies
+//     it. It polls the peer's health and computes whether it should consider
+//     itself active or standby.
+//   * waf-proxy does NOT move IP addresses. Actual packet failover (who
+//     answers on the VIP) belongs to keepalived/VRRP or a load balancer, which
+//     can consume this instance's role via GET /api/ha (or /healthz). Building
+//     an in-process VIP grab would be a dishonest half-solution.
+//
+// Blocklist state is intentionally NOT synced (per design): each node makes its
+// own AI decisions.
+
+import (
+	"bytes"
+	"context"
+	"encoding/json"
+	"fmt"
+	"log/slog"
+	"net/http"
+	"net/url"
+	"strings"
+	"sync"
+	"time"
+)
+
+type HAConfig struct {
+	Enabled    bool   `json:"enabled"`
+	Role       string `json:"role"`       // primary | secondary — tie-breaker for split-brain
+	PeerURL    string `json:"peer_url"`   // e.g. https://10.0.0.6:9090
+	PeerToken  string `json:"peer_token"` // admin bearer token of the peer (masked on read)
+	SyncConfig bool   `json:"sync_config"`
+}
+
+func defaultHAConfig() HAConfig {
+	return HAConfig{Role: "primary", SyncConfig: true}
+}
+
+func (c HAConfig) validate() error {
+	if !c.Enabled {
+		return nil
+	}
+	if c.Role != "primary" && c.Role != "secondary" {
+		return fmt.Errorf("ha: role must be primary or secondary")
+	}
+	if c.PeerURL == "" {
+		return fmt.Errorf("ha: peer_url is required when enabled")
+	}
+	u, err := url.Parse(c.PeerURL)
+	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
+		return fmt.Errorf("ha: peer_url must be an https origin without userinfo, query, fragment, or path")
+	}
+	if c.SyncConfig && strings.TrimSpace(c.PeerToken) == "" {
+		return fmt.Errorf("ha: peer_token is required when sync_config is enabled")
+	}
+	return nil
+}
+
+type haState struct {
+	PeerUp    bool      `json:"peer_up"`
+	Role      string    `json:"role"`      // active | standby | solo
+	LastSync  string    `json:"last_sync"` // result text
+	LastSeen  time.Time `json:"-"`
+	LastError string    `json:"last_error,omitempty"`
+}
+
+type haEngine struct {
+	mu     sync.Mutex
+	cfg    HAConfig
+	state  haState
+	client *http.Client
+	log    *slog.Logger
+	notify *notifier
+
+	syncMu      sync.Mutex
+	syncRunning bool
+	syncPending *haSyncJob
+}
+
+type haSyncJob struct {
+	peerURL   string
+	peerToken string
+	body      []byte
+}
+
+func newHAEngine(log *slog.Logger, n *notifier) *haEngine {
+	return &haEngine{
+		cfg:    defaultHAConfig(),
+		state:  haState{Role: "solo"},
+		client: &http.Client{Timeout: 5 * time.Second},
+		log:    log,
+		notify: n,
+	}
+}
+
+func (h *haEngine) configure(c HAConfig) {
+	h.mu.Lock()
+	defer h.mu.Unlock()
+	h.cfg = c
+	if !c.Enabled {
+		h.state = haState{Role: "solo"}
+	}
+}
+
+func (h *haEngine) snapshotCfg() HAConfig {
+	h.mu.Lock()
+	defer h.mu.Unlock()
+	return h.cfg
+}
+
+func (h *haEngine) status() haState {
+	h.mu.Lock()
+	defer h.mu.Unlock()
+	return h.state
+}
+
+// run polls peer health and updates role until ctx is cancelled.
+func (h *haEngine) run(ctx context.Context) {
+	t := time.NewTicker(3 * time.Second)
+	defer t.Stop()
+	for {
+		select {
+		case <-ctx.Done():
+			return
+		case <-t.C:
+			h.tick(ctx)
+		}
+	}
+}
+
+func (h *haEngine) tick(ctx context.Context) {
+	cfg := h.snapshotCfg()
+	if !cfg.Enabled {
+		return
+	}
+	up := h.pingPeer(ctx, cfg)
+
+	h.mu.Lock()
+	prevUp := h.state.PeerUp
+	h.state.PeerUp = up
+	if up {
+		h.state.LastSeen = time.Now()
+		// Peer alive: primary is active, secondary is standby.
+		if cfg.Role == "primary" {
+			h.state.Role = "active"
+		} else {
+			h.state.Role = "standby"
+		}
+	} else {
+		// Peer down: we take over regardless of role.
+		h.state.Role = "active"
+	}
+	role := h.state.Role
+	h.mu.Unlock()
+
+	if prevUp != up {
+		if up {
+			h.notify.push(notifyPeer, "info", "HA peer is up", "peer reachable; role="+role, "ha_peer", "", nil)
+		} else {
+			h.notify.push(notifyPeer, "alert", "HA peer is DOWN", "peer unreachable — this node is now active", "ha_peer", "", nil)
+		}
+	}
+}
+
+func (h *haEngine) pingPeer(ctx context.Context, cfg HAConfig) bool {
+	req, err := http.NewRequestWithContext(ctx, http.MethodGet, trimSlash(cfg.PeerURL)+"/healthz", nil)
+	if err != nil {
+		return false
+	}
+	resp, err := h.client.Do(req)
+	if err != nil {
+		return false
+	}
+	defer resp.Body.Close()
+	return resp.StatusCode == http.StatusOK
+}
+
+type haSyncEnvelope struct {
+	Version int    `json:"version"`
+	Config  Config `json:"config"`
+}
+
+const haSyncEnvelopeVersion = 1
+
+// sharedConfigForPeer removes node-local control-plane identity and secret
+// material before HA synchronization. The receiver always merges its own HA
+// identity, users and secret references back in before validation/apply.
+func sharedConfigForPeer(cfg Config) Config {
+	out := cfg
+	out.HA = HAConfig{}
+	out.Users = nil
+	out = redactAISecrets(out)
+	out = redactHSMSecretRefs(out)
+	out = redactNotifySecrets(out)
+	return out
+}
+
+func mergePeerConfig(local, incoming Config) Config {
+	incoming.HA = local.HA
+	incoming.Users = local.Users
+	preserveAISecrets(local, &incoming)
+	preserveNotifySecrets(local, &incoming)
+	preserveHSMSecretRefs(local, &incoming)
+	return incoming
+}
+
+// pushConfig sends the given config to the peer's dedicated replication API.
+// There is at most one network request in flight. If more local Applies commit
+// while it is running, they replace a single pending slot so the peer always
+// converges to the newest committed shared config without an unbounded queue.
+func (h *haEngine) pushConfig(cfg Config) {
+	hc := h.snapshotCfg()
+	if !hc.Enabled || !hc.SyncConfig || hc.PeerURL == "" {
+		return
+	}
+	payload := haSyncEnvelope{Version: haSyncEnvelopeVersion, Config: sharedConfigForPeer(cfg)}
+	body, err := json.Marshal(payload)
+	if err != nil {
+		h.recordSync("error: encode peer config: "+err.Error(), true)
+		return
+	}
+	job := haSyncJob{peerURL: hc.PeerURL, peerToken: hc.PeerToken, body: body}
+
+	h.syncMu.Lock()
+	if h.syncRunning {
+		// Latest wins: older unsent pending state is obsolete once a newer local
+		// config has durably committed.
+		pending := job
+		h.syncPending = &pending
+		h.syncMu.Unlock()
+		return
+	}
+	h.syncRunning = true
+	h.syncMu.Unlock()
+	go h.runSyncLoop(job)
+}
+
+func (h *haEngine) runSyncLoop(job haSyncJob) {
+	for {
+		h.sendSyncJob(job)
+		h.syncMu.Lock()
+		if h.syncPending != nil {
+			job = *h.syncPending
+			h.syncPending = nil
+			h.syncMu.Unlock()
+			continue
+		}
+		h.syncRunning = false
+		h.syncMu.Unlock()
+		return
+	}
+}
+
+func (h *haEngine) sendSyncJob(job haSyncJob) {
+	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
+	defer cancel()
+	req, err := http.NewRequestWithContext(ctx, http.MethodPut,
+		trimSlash(job.peerURL)+"/api/ha/peer-config", bytes.NewReader(job.body))
+	if err != nil {
+		h.recordSync("error: "+err.Error(), true)
+		return
+	}
+	req.Header.Set("Content-Type", "application/json")
+	req.Header.Set("Authorization", "Bearer "+job.peerToken)
+	req.Header.Set("X-WAF-HA-Sync", "v1")
+	resp, err := h.client.Do(req)
+	if err != nil {
+		h.recordSync("error: "+err.Error(), true)
+		return
+	}
+	defer resp.Body.Close()
+	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
+		h.recordSync(fmt.Sprintf("peer http %d", resp.StatusCode), true)
+		return
+	}
+	h.recordSync("ok "+time.Now().Format("15:04:05"), false)
+}
+
+func (h *haEngine) recordSync(text string, isErr bool) {
+	h.mu.Lock()
+	h.state.LastSync = text
+	if isErr {
+		h.state.LastError = text
+	} else {
+		h.state.LastError = ""
+	}
+	h.mu.Unlock()
+	if isErr {
+		h.notify.push(notifySync, "warn", "Config sync failed", text, "ha_sync_err", "", nil)
+		h.log.Warn("ha config sync failed", "detail", text)
+	} else {
+		h.notify.push(notifySync, "info", "Config synced to peer", text, "", "", nil)
+	}
+}
+
+func trimSlash(s string) string {
+	for len(s) > 0 && s[len(s)-1] == '/' {
+		s = s[:len(s)-1]
+	}
+	return s
+}
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/identity_api5.go ./identity_api5.go
--- /mnt/data/waf_prod_hardening_base/identity_api5.go	2026-09-25 11:11:25.278059523 +0000
+++ ./identity_api5.go	2026-09-25 11:36:44.234734527 +0000
@@ -1426,6 +1426,9 @@
 	if err := readJSONIfExists(apiSecurityStatePath(configPath, "api-identity.json"), &state); err != nil {
 		return err
 	}
+	if err := validateAPISecurityStateVersion("api-identity", state.Version); err != nil {
+		return err
+	}
 	s.mu.Lock()
 	defer s.mu.Unlock()
 	if state.Issuers != nil {
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/l7_abuse.go ./l7_abuse.go
--- /mnt/data/waf_prod_hardening_base/l7_abuse.go	2026-09-25 11:11:25.280863691 +0000
+++ ./l7_abuse.go	2026-09-25 11:24:23.507760410 +0000
@@ -44,9 +44,10 @@
 }
 
 type l7AbuseShard struct {
-	mu      sync.Mutex
-	clients map[abuseClientKey]*abuseClientState
-	ops     uint64
+	mu       sync.Mutex
+	clients  map[abuseClientKey]*abuseClientState
+	overflow abuseClientState
+	ops      uint64
 }
 
 type l7AbuseController struct {
@@ -110,12 +111,13 @@
 			c.evictOldestIdleLocked(shard)
 		}
 		if len(shard.clients) >= l7AbuseMaxEntriesPerShard {
-			// Preserve the existing bounded fail-open behavior when every slot is
-			// occupied. Never grow attacker-controlled state without bound.
-			return true
+			// Saturation must not turn the limiter off. Unknown identities share a
+			// bounded overflow bucket until an idle per-identity slot is available.
+			st = &shard.overflow
+		} else {
+			st = &abuseClientState{tokens: float64(c.cfg.TLSHandshakePerWindow), lastRefill: now, lastSeen: now}
+			shard.clients[key] = st
 		}
-		st = &abuseClientState{tokens: float64(c.cfg.TLSHandshakePerWindow), lastRefill: now, lastSeen: now}
-		shard.clients[key] = st
 	}
 
 	st.mu.Lock()
@@ -185,17 +187,17 @@
 			c.evictOldestIdleLocked(shard)
 		}
 		if len(shard.clients) >= l7AbuseMaxEntriesPerShard {
-			// The bounded shard is saturated with active entries and there is no
-			// safe idle entry to evict. Preserve the documented fail-open hot-path
-			// behavior rather than growing memory without bound.
-			return nil, true
-		}
-		st = &abuseClientState{lastSeen: now}
-		if c.cfg.RequestsPerWindow > 0 {
-			st.tokens = float64(c.cfg.RequestsPerWindow)
-			st.lastRefill = now
+			// Bound memory without creating an attacker-controlled fail-open bypass.
+			// Saturated unknown identities share one conservative shard bucket.
+			st = &shard.overflow
+		} else {
+			st = &abuseClientState{lastSeen: now}
+			if c.cfg.RequestsPerWindow > 0 {
+				st.tokens = float64(c.cfg.RequestsPerWindow)
+				st.lastRefill = now
+			}
+			shard.clients[key] = st
 		}
-		shard.clients[key] = st
 	}
 
 	// The shard lock pins the map entry until active has been incremented, so
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/listeners.go ./listeners.go
--- /mnt/data/waf_prod_hardening_base/listeners.go	2026-09-25 11:11:25.281077304 +0000
+++ ./listeners.go	2026-09-25 13:30:24.479744709 +0000
@@ -19,6 +19,7 @@
 import (
 	"context"
 	"crypto/tls"
+	"encoding/json"
 	"errors"
 	"fmt"
 	"log/slog"
@@ -38,6 +39,8 @@
 	srv        *http.Server
 	isTLS      bool
 	socketPath string
+	ln         net.Listener
+	done       chan struct{}
 }
 
 type originalSchemeContextKey struct{}
@@ -139,6 +142,18 @@
 		}
 		lr := rt.listeners[addr]
 		if lr == nil {
+			// During an atomic Go-TLS <-> external-frontend ownership handoff,
+			// the newly-bound socket can become reachable a fraction before the
+			// runtime pointer swaps. Fall back to the equivalent logical listener
+			// in the previous runtime so the transition never exposes a spurious
+			// 421 solely because the ownership key changed.
+			if frontendListener {
+				lr = rt.listeners[logicalAddr]
+			} else {
+				lr = rt.listeners[tlsfront.InternalListenerKey(addr)]
+			}
+		}
+		if lr == nil {
 			http.Error(w, "listener not configured", http.StatusMisdirectedRequest)
 			return
 		}
@@ -189,18 +204,12 @@
 // start launches a server in the background. Bind failures are retried while
 // the socket remains desired. This matters during a Go-TLS <-> external-TLS
 // mode transition, where the old owner may hold the public port briefly.
-func (m *listenerManager) start(addr string, isTLS bool, cfg Config) {
+func (m *listenerManager) prepare(addr string, isTLS bool, cfg Config) (*managedListener, error) {
 	srv := m.buildServer(addr, isTLS, cfg)
-	ml := &managedListener{srv: srv, isTLS: isTLS}
+	ml := &managedListener{srv: srv, isTLS: isTLS, done: make(chan struct{})}
 	if p, ok := tlsfront.SocketPathFromKey(addr); ok {
 		ml.socketPath = p
 	}
-	m.live[addr] = ml
-	go m.serve(addr, ml, cfg)
-}
-
-func (m *listenerManager) serve(addr string, ml *managedListener, cfg Config) {
-	m.log.Info("listener up", "addr", addr, "public_addr", tlsfront.PublicListenForKey(cfg.TLSAcceleration, tlsFrontendSites(cfg), addr), "tls", ml.isTLS, "tls_frontend_internal", ml.socketPath != "")
 	var ln net.Listener
 	var err error
 	if ml.socketPath != "" {
@@ -226,26 +235,50 @@
 		ln, err = lc.Listen(context.Background(), "tcp", addr)
 	}
 	if err != nil {
-		m.listenerFailed(addr, ml, err)
-		return
+		if ln != nil {
+			_ = ln.Close()
+		}
+		if ml.socketPath != "" {
+			_ = os.Remove(ml.socketPath)
+		}
+		return nil, err
 	}
+	ml.ln = ln
+	return ml, nil
+}
+
+func (m *listenerManager) serveBound(addr string, ml *managedListener, cfg Config) {
+	defer close(ml.done)
+	m.log.Info("listener up", "addr", addr, "public_addr", tlsfront.PublicListenForKey(cfg.TLSAcceleration, tlsFrontendSites(cfg), addr), "tls", ml.isTLS, "tls_frontend_internal", ml.socketPath != "")
 	defer func() {
-		_ = ln.Close()
+		if ml.ln != nil {
+			_ = ml.ln.Close()
+		}
 		if ml.socketPath != "" {
 			_ = os.Remove(ml.socketPath)
 		}
 	}()
 	var serveErr error
 	if ml.isTLS {
-		serveErr = ml.srv.ServeTLS(ln, "", "")
+		serveErr = ml.srv.ServeTLS(ml.ln, "", "")
 	} else {
-		serveErr = ml.srv.Serve(ln)
+		serveErr = ml.srv.Serve(ml.ln)
 	}
 	if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
 		m.listenerFailed(addr, ml, serveErr)
 	}
 }
 
+func (m *listenerManager) startLocked(addr string, isTLS bool, cfg Config) error {
+	ml, err := m.prepare(addr, isTLS, cfg)
+	if err != nil {
+		return err
+	}
+	m.live[addr] = ml
+	go m.serveBound(addr, ml, cfg)
+	return nil
+}
+
 func (m *listenerManager) listenerFailed(addr string, ml *managedListener, err error) {
 	m.log.Error("listener error", "addr", addr, "err", err)
 	m.mu.Lock()
@@ -270,46 +303,157 @@
 	if !ok || desired != isTLS {
 		return
 	}
-	m.start(addr, isTLS, rt.cfg)
+	if err := m.startLocked(addr, isTLS, rt.cfg); err != nil {
+		m.log.Error("listener retry bind failed", "addr", addr, "err", err)
+		time.AfterFunc(time.Second, func() { m.retry(addr, isTLS) })
+	}
 }
 
-// reconcile brings the running listeners in line with the desired set from cfg.
-// Added addresses are opened, removed ones gracefully closed, TLS flips
-// reopened; unchanged addresses are untouched (their connections persist).
-func (m *listenerManager) reconcile(cfg Config) {
-	desired := listenerSet(cfg) // addr -> isTLS
-	m.mu.Lock()
-	defer m.mu.Unlock()
+func waitListenerStopped(ml *managedListener) {
+	if ml == nil || ml.done == nil {
+		return
+	}
+	select {
+	case <-ml.done:
+	case <-time.After(2 * time.Second):
+	}
+}
 
-	// close removed or TLS-changed
+func closePrepared(ml *managedListener) {
+	if ml == nil {
+		return
+	}
+	if ml.ln != nil {
+		_ = ml.ln.Close()
+	}
+	if ml.socketPath != "" {
+		_ = os.Remove(ml.socketPath)
+	}
+}
+
+func (m *listenerManager) restoreOldLocked(oldCfg Config) error {
+	oldDesired := listenerSet(oldCfg)
+	var errs []string
 	for addr, ml := range m.live {
-		wantTLS, keep := desired[addr]
-		if !keep {
-			m.log.Info("listener closing", "addr", addr, "reason", "removed")
-			go gracefulClose(ml.srv) // drain removed address
-			delete(m.live, addr)
-		} else if wantTLS != ml.isTLS {
-			// Same address, http<->tls flip: free the port synchronously so the
-			// reopen below can rebind immediately (graceful drain would hold it).
-			m.log.Info("listener reopening", "addr", addr, "reason", "tls-change", "tls", wantTLS)
+		wantTLS, keep := oldDesired[addr]
+		if !keep || wantTLS != ml.isTLS {
 			_ = ml.srv.Close()
+			waitListenerStopped(ml)
 			delete(m.live, addr)
 		}
 	}
-	// open added (or reopen after a TLS flip)
+	for addr, isTLS := range oldDesired {
+		if _, ok := m.live[addr]; ok {
+			continue
+		}
+		if err := m.startLocked(addr, isTLS, oldCfg); err != nil {
+			errs = append(errs, fmt.Sprintf("%s: %v", addr, err))
+		}
+	}
+	if len(errs) > 0 {
+		return fmt.Errorf("listener rollback failed: %s", strings.Join(errs, "; "))
+	}
+	return nil
+}
+
+// reconcile synchronously binds every newly-required socket before reporting
+// success. Destructive TLS-mode flips are rolled back to the old listener set
+// on bind failure; removed sockets are closed only after all additions/flips
+// are known-good. This prevents Apply from claiming success when the data plane
+// cannot actually own the configured addresses.
+func (m *listenerManager) reconcile(cfg Config) error {
+	desired := listenerSet(cfg)
+	m.mu.Lock()
+	defer m.mu.Unlock()
+
+	oldCfg := cfg
+	if rt := m.srv.rt.Load(); rt != nil {
+		oldCfg = rt.cfg
+	}
+
+	prepared := map[string]*managedListener{}
 	for addr, isTLS := range desired {
-		if _, running := m.live[addr]; !running {
-			m.start(addr, isTLS, cfg)
+		if cur, ok := m.live[addr]; ok {
+			if cur.isTLS == isTLS {
+				continue
+			}
+			continue // protocol flips are handled below after closing the old socket
+		}
+		ml, err := m.prepare(addr, isTLS, cfg)
+		if err != nil {
+			for _, p := range prepared {
+				closePrepared(p)
+			}
+			return fmt.Errorf("bind listener %s: %w", addr, err)
+		}
+		prepared[addr] = ml
+	}
+
+	// TLS-mode flips require releasing the old socket. If any replacement bind
+	// fails, restore the complete old listener topology before returning.
+	for addr, cur := range m.live {
+		wantTLS, keep := desired[addr]
+		if !keep || wantTLS == cur.isTLS {
+			continue
+		}
+		m.log.Info("listener reopening", "addr", addr, "reason", "tls-change", "tls", wantTLS)
+		_ = cur.srv.Close()
+		waitListenerStopped(cur)
+		delete(m.live, addr)
+		ml, err := m.prepare(addr, wantTLS, cfg)
+		if err != nil {
+			for _, p := range prepared {
+				closePrepared(p)
+			}
+			rbErr := m.restoreOldLocked(oldCfg)
+			if rbErr != nil {
+				return fmt.Errorf("bind listener %s after TLS change: %v; %v", addr, err, rbErr)
+			}
+			return fmt.Errorf("bind listener %s after TLS change: %w", addr, err)
 		}
+		m.live[addr] = ml
+		go m.serveBound(addr, ml, cfg)
 	}
+
+	for addr, ml := range prepared {
+		m.live[addr] = ml
+		go m.serveBound(addr, ml, cfg)
+	}
+
+	frontendOwnedPublic := map[string]struct{}{}
+	if tlsfront.FrontendEnabled(cfg.TLSAcceleration) {
+		for addr, isTLS := range publicListenerSet(cfg) {
+			if isTLS {
+				frontendOwnedPublic[addr] = struct{}{}
+			}
+		}
+	}
+	for addr, ml := range m.live {
+		if _, keep := desired[addr]; keep {
+			continue
+		}
+		if _, handoff := frontendOwnedPublic[addr]; handoff {
+			m.log.Info("listener releasing", "addr", addr, "reason", "tls-frontend-ownership")
+			releaseForFrontendOwnership(ml)
+		} else {
+			m.log.Info("listener closing", "addr", addr, "reason", "removed")
+			go gracefulClose(ml.srv)
+		}
+		delete(m.live, addr)
+	}
+	return nil
 }
 
-// startAll is the initial bind at boot.
+// startAll is the initial bind at boot. Startup keeps the historical retry
+// behavior, but each initial bind attempt is synchronous and visible in logs.
 func (m *listenerManager) startAll(cfg Config) {
 	m.mu.Lock()
 	defer m.mu.Unlock()
 	for addr, isTLS := range listenerSet(cfg) {
-		m.start(addr, isTLS, cfg)
+		if err := m.startLocked(addr, isTLS, cfg); err != nil {
+			m.log.Error("initial listener bind failed", "addr", addr, "err", err)
+			time.AfterFunc(time.Second, func() { m.retry(addr, isTLS) })
+		}
 	}
 }
 
@@ -333,6 +477,23 @@
 	return len(m.live)
 }
 
+func releaseForFrontendOwnership(ml *managedListener) {
+	if ml == nil || ml.srv == nil {
+		return
+	}
+	// Shutdown closes the listening socket before waiting for active requests,
+	// which is the property the external TLS frontend needs before it can bind
+	// the public address. Keep the drain bounded; Close is the fail-safe that
+	// guarantees ownership is released even if a handler ignores cancellation.
+	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
+	err := ml.srv.Shutdown(ctx)
+	cancel()
+	if err != nil {
+		_ = ml.srv.Close()
+	}
+	waitListenerStopped(ml)
+}
+
 func gracefulClose(srv *http.Server) {
 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
 	defer cancel()
@@ -340,17 +501,13 @@
 }
 
 func writeHealth(w http.ResponseWriter, ok bool, draining bool, role string) {
-	if ok {
-		_, _ = w.Write([]byte(`{"status":"ok","role":"` + role + `"}` + "\n"))
-		return
-	}
 	status := "unavailable"
-	_, _ = w.Write([]byte(`{"status":"` + status + `","draining":` + boolStr(draining) + `,"role":"` + role + `"}` + "\n"))
-}
-
-func boolStr(b bool) string {
-	if b {
-		return "true"
+	if ok {
+		status = "ok"
 	}
-	return "false"
+	_ = json.NewEncoder(w).Encode(map[string]any{
+		"status":   status,
+		"draining": draining,
+		"role":     role,
+	})
 }
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/main.go ./main.go
--- /mnt/data/waf_prod_hardening_base/main.go	2026-09-25 11:11:25.281337146 +0000
+++ ./main.go	2026-09-25 12:47:31.445612251 +0000
@@ -820,16 +820,124 @@
 	return ""
 }
 
-func saveConfig(path string, c Config) error {
+type stagedConfigWrite struct {
+	path string
+	tmp  string
+}
+
+func stageConfig(path string, c Config) (*stagedConfigWrite, error) {
 	b, err := json.MarshalIndent(c, "", "  ")
 	if err != nil {
+		return nil, err
+	}
+	b = append(b, '\n')
+	dir := filepath.Dir(path)
+	f, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
+	if err != nil {
+		return nil, err
+	}
+	tmp := f.Name()
+	ok := false
+	defer func() {
+		if !ok {
+			_ = f.Close()
+			_ = os.Remove(tmp)
+		}
+	}()
+	if err := f.Chmod(0o600); err != nil {
+		return nil, err
+	}
+	if _, err := f.Write(b); err != nil {
+		return nil, err
+	}
+	if err := f.Sync(); err != nil {
+		return nil, err
+	}
+	if err := f.Close(); err != nil {
+		return nil, err
+	}
+	ok = true
+	return &stagedConfigWrite{path: path, tmp: tmp}, nil
+}
+
+func (s *stagedConfigWrite) abort() {
+	if s != nil && s.tmp != "" {
+		_ = os.Remove(s.tmp)
+	}
+}
+
+// commit returns committed=true once rename succeeded. A directory fsync error
+// after rename is a durability warning, not a reason to roll live state back to
+// an older config while the filesystem already exposes the new one.
+func (s *stagedConfigWrite) commit() (committed bool, err error) {
+	if s == nil {
+		return false, errors.New("nil staged config")
+	}
+	if err := os.Rename(s.tmp, s.path); err != nil {
+		return false, err
+	}
+	s.tmp = ""
+	d, err := os.Open(filepath.Dir(s.path))
+	if err != nil {
+		return true, err
+	}
+	defer d.Close()
+	if err := d.Sync(); err != nil {
+		return true, err
+	}
+	return true, nil
+}
+
+func saveConfig(path string, c Config) error {
+	staged, err := stageConfig(path, c)
+	if err != nil {
 		return err
 	}
-	tmp := path + ".tmp"
-	if err := os.WriteFile(tmp, b, 0o600); err != nil {
+	defer staged.abort()
+	_, err = staged.commit()
+	return err
+}
+
+func draftConfigPath(path string) string {
+	return path + ".draft"
+}
+
+// removeConfigFileDurable removes an auxiliary control-plane state file and
+// fsyncs the parent directory so a successful Apply cannot resurrect a stale
+// draft after a crash/restart. Missing files are already converged.
+func removeConfigFileDurable(path string) error {
+	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
+		return err
+	}
+	d, err := os.Open(filepath.Dir(path))
+	if err != nil {
 		return err
 	}
-	return os.Rename(tmp, path)
+	defer d.Close()
+	return d.Sync()
+}
+
+// loadDraftConfig returns a draft only when it is newer than the authoritative
+// config. This makes best-effort cleanup safe: an old draft left behind after a
+// committed Apply is ignored on the next Console load and is never startup
+// authority.
+func loadDraftConfig(configPath string) (Config, bool, error) {
+	draftPath := draftConfigPath(configPath)
+	draftInfo, err := os.Stat(draftPath)
+	if err != nil {
+		if os.IsNotExist(err) {
+			return Config{}, false, nil
+		}
+		return Config{}, false, err
+	}
+	if liveInfo, liveErr := os.Stat(configPath); liveErr == nil && !draftInfo.ModTime().After(liveInfo.ModTime()) {
+		return Config{}, false, nil
+	}
+	c, err := loadConfig(draftPath)
+	if err != nil {
+		return Config{}, false, err
+	}
+	return c, true, nil
 }
 
 func normalizeHost(h string) string {
@@ -959,6 +1067,7 @@
 }
 
 type server struct {
+	applyMu             sync.Mutex
 	rt                  atomic.Pointer[runtimeState]
 	matches             *matchRing
 	access              *accessRing
@@ -1004,6 +1113,8 @@
 }
 
 func (s *server) apply(cfg Config) error {
+	s.applyMu.Lock()
+	defer s.applyMu.Unlock()
 	return s.applyEx(cfg, false)
 }
 
@@ -1023,38 +1134,199 @@
 			return fmt.Errorf("TLS frontend preflight: %w", err)
 		}
 	}
-	// Network ownership must succeed before the runtime becomes live. Otherwise
-	// the API could report Apply failure after already swapping the config.
+
+	oldCfg := cfg
+	if old := s.rt.Load(); old != nil {
+		oldCfg = old.cfg
+	}
+	oldFrontend := tlsfront.FrontendEnabled(oldCfg.TLSAcceleration)
+	newFrontend := tlsfront.FrontendEnabled(cfg.TLSAcceleration)
+	frontendPublished := false
+
+	// The external frontend owns the public TLS port. When handing ownership
+	// back to Go crypto/tls, stop it and wait for acknowledgement before the
+	// listener manager attempts to bind the public socket. Without this ordering,
+	// a valid frontend->go transition deterministically fails with EADDRINUSE.
+	if oldFrontend && !newFrontend {
+		if s.tlsFrontend == nil || !s.tlsFrontend.enabled() {
+			rt.close()
+			return errors.New("cannot disable TLS frontend: control publisher is unavailable")
+		}
+		if err := s.tlsFrontend.publishAndWait(cfg, 8*time.Second); err != nil {
+			// Best effort restore of the previously active frontend specification.
+			_ = s.tlsFrontend.publishAndWait(oldCfg, 8*time.Second)
+			rt.close()
+			return fmt.Errorf("disable TLS frontend: %w", err)
+		}
+		frontendPublished = true
+	}
+
+	listenersChanged := false
+	frontendRollback := func() {
+		if !frontendPublished || s.tlsFrontend == nil || !s.tlsFrontend.enabled() {
+			return
+		}
+		// If the attempted config enabled the external frontend while the old
+		// config used Go TLS, stop the frontend first so the old public socket can
+		// be rebound. If the old config used the frontend, restore it only after
+		// the old internal listener topology is back in place.
+		if !oldFrontend && newFrontend {
+			if rbErr := s.tlsFrontend.publishAndWait(oldCfg, 8*time.Second); rbErr != nil {
+				s.log.Error("TLS frontend rollback/stop failed", "err", rbErr)
+			}
+		}
+	}
+	frontendRestoreAfterListeners := func() {
+		if frontendPublished && oldFrontend && s.tlsFrontend != nil && s.tlsFrontend.enabled() {
+			if rbErr := s.tlsFrontend.publishAndWait(oldCfg, 8*time.Second); rbErr != nil {
+				s.log.Error("TLS frontend rollback/restore failed", "err", rbErr)
+			}
+		}
+	}
+	rollbackNetwork := func() {
+		frontendRollback()
+		if s.ipmgr != nil {
+			if rbErr := s.ipmgr.reconcile(oldCfg); rbErr != nil {
+				s.log.Error("managed IP rollback failed", "err", rbErr)
+			}
+		}
+		if listenersChanged && s.listenMgr != nil {
+			if rbErr := s.listenMgr.reconcile(oldCfg); rbErr != nil {
+				s.log.Error("listener rollback failed", "err", rbErr)
+			}
+		}
+		frontendRestoreAfterListeners()
+	}
+
+	if s.listenMgr != nil {
+		if err := s.listenMgr.reconcile(cfg); err != nil {
+			frontendRestoreAfterListeners()
+			rt.close()
+			return err
+		}
+		listenersChanged = true
+	}
 	if s.ipmgr != nil {
 		if err := s.ipmgr.reconcile(cfg); err != nil {
+			rollbackNetwork()
 			rt.close()
 			return err
 		}
 	}
-	// Publish only the successfully built/live TLS frontend specification. Draft
-	// saves never touch this file, so the companion cannot bind public TLS ports
-	// before the WAF runtime is actually applied.
-	if s.tlsFrontend != nil {
-		if err := s.tlsFrontend.publish(cfg); err != nil {
+	if s.tlsFrontend != nil && !frontendPublished {
+		if newFrontend {
+			if err := s.tlsFrontend.publishAndWait(cfg, 8*time.Second); err != nil {
+				// The frontend may have partially activated. Restore old ownership
+				// before returning Apply failure.
+				frontendPublished = true
+				rollbackNetwork()
+				rt.close()
+				return fmt.Errorf("publish TLS frontend config: %w", err)
+			}
+			frontendPublished = true
+		} else if err := s.tlsFrontend.publish(cfg); err != nil {
+			rollbackNetwork()
 			rt.close()
 			return fmt.Errorf("publish TLS frontend config: %w", err)
 		}
 	}
+
 	old := s.rt.Swap(rt)
-	old.close() // stop previous monitors and release idle backend connections
+	if old != nil {
+		old.close()
+	}
 	s.ai.configure(cfg.AI)
 	s.notify.configure(cfg.Notify)
 	s.ha.configure(cfg.HA)
 	s.syslog.configure(cfg.Syslog)
-	if s.listenMgr != nil {
-		s.listenMgr.reconcile(cfg) // open/close data-plane sockets live — no restart
+	_ = fromSync // loop prevention is handled by applyPersisted after durable commit.
+	return nil
+}
+
+// applyPersisted stages the config durably before changing live state, applies
+// the full runtime/listener transaction, then atomically renames the staged
+// config. If commit fails before rename, live state is rolled back. HA sync is
+// emitted only after local persistence succeeded.
+func (s *server) applyPersisted(cfg Config, fromSync bool) error {
+	s.applyMu.Lock()
+	defer s.applyMu.Unlock()
+	return s.applyPersistedLocked(cfg, fromSync)
+}
+
+// applyPersistedLocked is the lock-assumed implementation shared by direct
+// full-config Apply and read-modify-write control-plane mutations. Keeping the
+// lock across stage -> live reconcile -> rename -> HA enqueue prevents a second
+// mutation from rolling back or publishing across the first transaction.
+func (s *server) applyPersistedLocked(cfg Config, fromSync bool) error {
+	if err := cfg.validate(); err != nil {
+		return err
+	}
+	staged, err := stageConfig(s.configPath, cfg)
+	if err != nil {
+		return fmt.Errorf("stage config: %w", err)
+	}
+	defer staged.abort()
+	oldCfg := cfg
+	if old := s.rt.Load(); old != nil {
+		oldCfg = old.cfg
+	}
+	if err := s.applyEx(cfg, fromSync); err != nil {
+		return err
+	}
+	committed, err := staged.commit()
+	if err != nil && !committed {
+		if rbErr := s.applyEx(oldCfg, true); rbErr != nil {
+			return fmt.Errorf("persist config: %v; live rollback also failed: %v", err, rbErr)
+		}
+		return fmt.Errorf("persist config: %w", err)
+	}
+	if err != nil && committed {
+		s.log.Warn("config renamed but directory fsync failed", "err", err)
 	}
 	if !fromSync {
-		s.ha.pushConfig(cfg) // propagate local changes to the peer
+		s.ha.pushConfig(cfg)
 	}
 	return nil
 }
 
+// cloneConfig creates an ownership-independent copy suitable for control-plane
+// read-modify-write operations. Config contains nested slices/maps; assigning it
+// by value is not enough and can mutate the active runtime before Apply.
+func cloneConfig(cfg Config) (Config, error) {
+	b, err := json.Marshal(cfg)
+	if err != nil {
+		return Config{}, fmt.Errorf("clone config: %w", err)
+	}
+	var out Config
+	if err := json.Unmarshal(b, &out); err != nil {
+		return Config{}, fmt.Errorf("clone config: %w", err)
+	}
+	return out, nil
+}
+
+// mutatePersisted serializes a partial config change and derives it from the
+// newest live runtime while holding applyMu, preventing shallow-copy mutation
+// of live state and avoiding lost updates between independent partial handlers.
+func (s *server) mutatePersisted(fromSync bool, mutate func(*Config) error) (Config, error) {
+	s.applyMu.Lock()
+	defer s.applyMu.Unlock()
+	rt := s.rt.Load()
+	if rt == nil {
+		return Config{}, errors.New("runtime unavailable")
+	}
+	cfg, err := cloneConfig(rt.cfg)
+	if err != nil {
+		return Config{}, err
+	}
+	if err := mutate(&cfg); err != nil {
+		return Config{}, err
+	}
+	if err := s.applyPersistedLocked(cfg, fromSync); err != nil {
+		return Config{}, err
+	}
+	return cfg, nil
+}
+
 func (s *server) buildRuntime(cfg Config) (*runtimeState, error) {
 	l7Abuse := newL7AbuseController(cfg.L7Abuse)
 	cidrEngine, err := newCIDRPolicyEngine(cfg.CIDRPolicy)
@@ -1830,8 +2102,18 @@
 func (s *server) getCertificate(addr string) func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
 	return func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
 		rt := s.rt.Load()
+		if rt == nil {
+			return nil, fmt.Errorf("runtime unavailable")
+		}
 		lr := rt.listeners[addr]
 		if lr == nil {
+			// When ownership is moving from the external TLS frontend back to
+			// Go crypto/tls, the public socket can be bound just before the
+			// runtime pointer swaps. Reuse the equivalent internal-listener
+			// certificate table from the previous runtime for that handoff.
+			lr = rt.listeners[tlsfront.InternalListenerKey(addr)]
+		}
+		if lr == nil {
 			return nil, fmt.Errorf("no listener for %s", addr)
 		}
 		if site := lr.lookup(hello.ServerName); site != nil && site.cert != nil {
@@ -1848,12 +2130,13 @@
 
 func main() {
 	var (
-		configPath = flag.String("config", "config.json", "path to JSON config (created with defaults if missing)")
-		adminAddr  = flag.String("admin", envOr("WAF_ADMIN_ADDR", "127.0.0.1:9090"), "admin console listen address, e.g. 127.0.0.1:9090, 0.0.0.0:9090, or a mgmt IP")
-		adminCert  = flag.String("admin-cert", os.Getenv("WAF_ADMIN_TLS_CERT"), "TLS cert for the admin console (recommended when binding off-loopback)")
-		adminKey   = flag.String("admin-key", os.Getenv("WAF_ADMIN_TLS_KEY"), "TLS key for the admin console")
-		adminToken = flag.String("admin-token", os.Getenv("WAF_ADMIN_TOKEN"), "bearer token for the admin API (random if empty)")
-		browseRoot = flag.String("tls-browse-root", "/etc", "directory the console's cert/key file browser is allowed to list (read-only)")
+		configPath  = flag.String("config", "config.json", "path to JSON config (created with defaults if missing)")
+		adminAddr   = flag.String("admin", envOr("WAF_ADMIN_ADDR", "127.0.0.1:9090"), "admin console listen address, e.g. 127.0.0.1:9090, 0.0.0.0:9090, or a mgmt IP")
+		adminCert   = flag.String("admin-cert", os.Getenv("WAF_ADMIN_TLS_CERT"), "TLS cert for the admin console (recommended when binding off-loopback)")
+		adminKey    = flag.String("admin-key", os.Getenv("WAF_ADMIN_TLS_KEY"), "TLS key for the admin console")
+		adminToken  = flag.String("admin-token", os.Getenv("WAF_ADMIN_TOKEN"), "bearer token for the admin API (random if empty)")
+		haPeerToken = flag.String("ha-peer-token", os.Getenv("WAF_HA_PEER_TOKEN"), "dedicated bearer token accepted only by the HA replication endpoint")
+		browseRoot  = flag.String("tls-browse-root", "/etc", "directory the console's cert/key file browser is allowed to list (read-only)")
 	)
 	flag.Parse()
 
@@ -1872,6 +2155,10 @@
 		log.Error("could not load config", "path", *configPath, "err", err)
 		os.Exit(1)
 	}
+	if cfg.HA.Enabled && cfg.HA.SyncConfig && strings.TrimSpace(*haPeerToken) == "" {
+		log.Error("HA config sync requires a stable dedicated replication token", "hint", "set WAF_HA_PEER_TOKEN or -ha-peer-token")
+		os.Exit(1)
+	}
 
 	root, err := filepath.Abs(*browseRoot)
 	if err != nil {
@@ -2001,7 +2288,7 @@
 	go s.debug.RunCleanup(debugStop, time.Minute)
 
 	// ── admin listener ──
-	admin := newAdminServer(s, *adminToken, log)
+	admin := newAdminServer(s, *adminToken, *haPeerToken, log)
 	adminSrv := &http.Server{
 		Addr:              *adminAddr,
 		Handler:           admin.handler(),
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/notify.go ./notify.go
--- /mnt/data/waf_prod_hardening_base/notify.go	2026-09-25 11:11:25.281612230 +0000
+++ ./notify.go	2026-09-25 12:49:39.652294106 +0000
@@ -1,232 +1,305 @@
-package main
-
-// Notifications.
-//
-// A lightweight in-memory queue of noteworthy events (learner suggestions,
-// AI blocks, pool member down, config sync results, peer state changes). Each
-// is surfaced in the console bell and optionally POSTed to a webhook
-// (Slack/Teams/generic JSON). Some carry an "apply" action payload so the
-// operator can act on them with one click — nothing is ever auto-applied.
-//
-// State is in-memory and resets on restart (a clean seam exists to back it
-// with a JSON snapshot later).
-
-import (
-	"bytes"
-	"context"
-	"encoding/json"
-	"log/slog"
-	"net/http"
-	"sync"
-	"sync/atomic"
-	"time"
-)
-
-type NotifyConfig struct {
-	WebhookURL  string `json:"webhook_url,omitempty"`
-	WebhookKind string `json:"webhook_kind,omitempty"` // slack | generic
-	// event toggles
-	OnSuggestion bool `json:"on_suggestion"`
-	OnAIBlock    bool `json:"on_ai_block"`
-	OnMemberDown bool `json:"on_member_down"`
-	OnSync       bool `json:"on_sync"`
-	OnPeer       bool `json:"on_peer"`
-}
-
-func defaultNotifyConfig() NotifyConfig {
-	return NotifyConfig{
-		WebhookKind:  "slack",
-		OnSuggestion: true, OnAIBlock: true, OnMemberDown: true, OnSync: true, OnPeer: true,
-	}
-}
-
-// notification kinds
-const (
-	notifySuggestion = "suggestion"
-	notifyAIBlock    = "ai_block"
-	notifyMemberDown = "member_down"
-	notifySync       = "sync"
-	notifyPeer       = "peer"
-	notifyInfo       = "info"
-)
-
-type notification struct {
-	ID      int64          `json:"id"`
-	Time    string         `json:"time"`
-	Kind    string         `json:"kind"`
-	Level   string         `json:"level"` // info | warn | alert
-	Title   string         `json:"title"`
-	Body    string         `json:"body"`
-	Read    bool           `json:"read"`
-	Action  string         `json:"action,omitempty"` // e.g. "apply_exclusion"
-	Payload map[string]any `json:"payload,omitempty"`
-}
-
-type notifier struct {
-	mu     sync.Mutex
-	cfg    NotifyConfig
-	items  []notification
-	nextID int64
-	cap    int
-	client *http.Client
-	log    *slog.Logger
-
-	// dedupe map + optional external sink (e.g. syslog)
-	dedupe map[string]time.Time
-	sink   func(level, kind, title, body string)
-}
-
-func newNotifier(log *slog.Logger) *notifier {
-	return &notifier{
-		cfg:    defaultNotifyConfig(),
-		cap:    200,
-		client: &http.Client{Timeout: 6 * time.Second},
-		log:    log,
-		dedupe: map[string]time.Time{},
-	}
-}
-
-func (n *notifier) configure(c NotifyConfig) {
-	n.mu.Lock()
-	defer n.mu.Unlock()
-	n.cfg = c
-}
-
-func (n *notifier) enabledFor(kind string) bool {
-	n.mu.Lock()
-	defer n.mu.Unlock()
-	switch kind {
-	case notifySuggestion:
-		return n.cfg.OnSuggestion
-	case notifyAIBlock:
-		return n.cfg.OnAIBlock
-	case notifyMemberDown:
-		return n.cfg.OnMemberDown
-	case notifySync:
-		return n.cfg.OnSync
-	case notifyPeer:
-		return n.cfg.OnPeer
-	}
-	return true
-}
-
-// push adds a notification (respecting per-kind toggles and a dedupe window)
-// and fires the webhook asynchronously.
-func (n *notifier) push(kind, level, title, body, dedupeKey string, action string, payload map[string]any) {
-	if !n.enabledFor(kind) {
-		return
-	}
-	n.mu.Lock()
-	if dedupeKey != "" {
-		if exp, ok := n.dedupe[dedupeKey]; ok && time.Now().Before(exp) {
-			n.mu.Unlock()
-			return
-		}
-		n.dedupe[dedupeKey] = time.Now().Add(60 * time.Second)
-	}
-	n.nextID++
-	item := notification{
-		ID: n.nextID, Time: time.Now().Format("15:04:05"), Kind: kind, Level: level,
-		Title: title, Body: body, Action: action, Payload: payload,
-	}
-	n.items = append(n.items, item)
-	if len(n.items) > n.cap {
-		n.items = n.items[len(n.items)-n.cap:]
-	}
-	hook := n.cfg.WebhookURL
-	kindHook := n.cfg.WebhookKind
-	sink := n.sink
-	n.mu.Unlock()
-
-	if sink != nil {
-		sink(level, kind, title, body)
-	}
-	if hook != "" {
-		go n.sendWebhook(hook, kindHook, item)
-	}
-}
-
-func (n *notifier) sendWebhook(url, kind string, item notification) {
-	var payload any
-	text := "[" + item.Level + "] " + item.Title + " — " + item.Body
-	if kind == "slack" {
-		payload = map[string]any{"text": text}
-	} else {
-		payload = map[string]any{
-			"level": item.Level, "kind": item.Kind, "title": item.Title,
-			"body": item.Body, "time": item.Time,
-		}
-	}
-	b, _ := json.Marshal(payload)
-	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
-	defer cancel()
-	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
-	if err != nil {
-		return
-	}
-	req.Header.Set("Content-Type", "application/json")
-	resp, err := n.client.Do(req)
-	if err != nil {
-		n.log.Warn("notify webhook failed", "err", err)
-		return
-	}
-	_ = resp.Body.Close()
-}
-
-func (n *notifier) list(limit int) []notification {
-	n.mu.Lock()
-	defer n.mu.Unlock()
-	total := len(n.items)
-	if limit <= 0 || limit > total {
-		limit = total
-	}
-	out := make([]notification, limit)
-	for i := 0; i < limit; i++ {
-		out[i] = n.items[total-1-i] // newest first
-	}
-	return out
-}
-
-func (n *notifier) unreadCount() int {
-	n.mu.Lock()
-	defer n.mu.Unlock()
-	c := 0
-	for _, it := range n.items {
-		if !it.Read {
-			c++
-		}
-	}
-	return c
-}
-
-func (n *notifier) markRead(id int64, all bool) {
-	n.mu.Lock()
-	defer n.mu.Unlock()
-	for i := range n.items {
-		if all || n.items[i].ID == id {
-			n.items[i].Read = true
-		}
-	}
-}
-
-func (n *notifier) dismiss(id int64, all bool) {
-	n.mu.Lock()
-	defer n.mu.Unlock()
-	if all {
-		n.items = nil
-		return
-	}
-	out := n.items[:0]
-	for _, it := range n.items {
-		if it.ID != id {
-			out = append(out, it)
-		}
-	}
-	n.items = out
-}
-
-// pending atomic guard so periodic scanners don't stack.
-type gate struct{ busy int32 }
-
-func (g *gate) enter() bool { return atomic.CompareAndSwapInt32(&g.busy, 0, 1) }
-func (g *gate) leave()      { atomic.StoreInt32(&g.busy, 0) }
+package main
+
+// Notifications.
+//
+// A lightweight in-memory queue of noteworthy events (learner suggestions,
+// AI blocks, pool member down, config sync results, peer state changes). Each
+// is surfaced in the console bell and optionally POSTed to a webhook
+// (Slack/Teams/generic JSON). Some carry an "apply" action payload so the
+// operator can act on them with one click — nothing is ever auto-applied.
+//
+// State is in-memory and resets on restart (a clean seam exists to back it
+// with a JSON snapshot later).
+
+import (
+	"bytes"
+	"context"
+	"encoding/json"
+	"fmt"
+	"log/slog"
+	"net/http"
+	"strings"
+	"sync"
+	"sync/atomic"
+	"time"
+)
+
+type NotifyConfig struct {
+	WebhookURL  string `json:"webhook_url,omitempty"`
+	WebhookKind string `json:"webhook_kind,omitempty"` // slack | generic
+	// event toggles
+	OnSuggestion bool `json:"on_suggestion"`
+	OnAIBlock    bool `json:"on_ai_block"`
+	OnMemberDown bool `json:"on_member_down"`
+	OnSync       bool `json:"on_sync"`
+	OnPeer       bool `json:"on_peer"`
+}
+
+func redactNotifySecrets(c Config) Config {
+	if c.Notify.WebhookURL != "" {
+		c.Notify.WebhookURL = ""
+	}
+	return c
+}
+
+func preserveNotifySecrets(cur Config, next *Config) {
+	if next == nil {
+		return
+	}
+	if strings.TrimSpace(next.Notify.WebhookURL) == "" {
+		next.Notify.WebhookURL = cur.Notify.WebhookURL
+	}
+}
+
+func defaultNotifyConfig() NotifyConfig {
+	return NotifyConfig{
+		WebhookKind:  "slack",
+		OnSuggestion: true, OnAIBlock: true, OnMemberDown: true, OnSync: true, OnPeer: true,
+	}
+}
+
+// notification kinds
+const (
+	notifySuggestion = "suggestion"
+	notifyAIBlock    = "ai_block"
+	notifyMemberDown = "member_down"
+	notifySync       = "sync"
+	notifyPeer       = "peer"
+	notifyInfo       = "info"
+)
+
+type notification struct {
+	ID      int64          `json:"id"`
+	Time    string         `json:"time"`
+	Kind    string         `json:"kind"`
+	Level   string         `json:"level"` // info | warn | alert
+	Title   string         `json:"title"`
+	Body    string         `json:"body"`
+	Read    bool           `json:"read"`
+	Action  string         `json:"action,omitempty"` // e.g. "apply_exclusion"
+	Payload map[string]any `json:"payload,omitempty"`
+}
+
+type webhookJob struct {
+	url  string
+	kind string
+	item notification
+}
+
+const notificationDedupeMax = 4096
+
+type notifier struct {
+	mu             sync.Mutex
+	cfg            NotifyConfig
+	items          []notification
+	nextID         int64
+	cap            int
+	client         *http.Client
+	log            *slog.Logger
+	webhookQ       chan webhookJob
+	webhookDropped uint64
+
+	// dedupe map + optional external sink (e.g. syslog)
+	dedupe map[string]time.Time
+	sink   func(level, kind, title, body string)
+}
+
+func newNotifier(log *slog.Logger) *notifier {
+	n := &notifier{
+		cfg:      defaultNotifyConfig(),
+		cap:      200,
+		client:   &http.Client{Timeout: 6 * time.Second},
+		log:      log,
+		dedupe:   map[string]time.Time{},
+		webhookQ: make(chan webhookJob, 256),
+	}
+	for i := 0; i < 2; i++ {
+		go n.webhookWorker()
+	}
+	return n
+}
+
+func (n *notifier) configure(c NotifyConfig) {
+	n.mu.Lock()
+	defer n.mu.Unlock()
+	n.cfg = c
+}
+
+func (n *notifier) enabledFor(kind string) bool {
+	n.mu.Lock()
+	defer n.mu.Unlock()
+	switch kind {
+	case notifySuggestion:
+		return n.cfg.OnSuggestion
+	case notifyAIBlock:
+		return n.cfg.OnAIBlock
+	case notifyMemberDown:
+		return n.cfg.OnMemberDown
+	case notifySync:
+		return n.cfg.OnSync
+	case notifyPeer:
+		return n.cfg.OnPeer
+	}
+	return true
+}
+
+// push adds a notification (respecting per-kind toggles and a dedupe window)
+// and fires the webhook asynchronously.
+func (n *notifier) push(kind, level, title, body, dedupeKey string, action string, payload map[string]any) {
+	if !n.enabledFor(kind) {
+		return
+	}
+	n.mu.Lock()
+	if dedupeKey != "" {
+		now := time.Now()
+		if exp, ok := n.dedupe[dedupeKey]; ok && now.Before(exp) {
+			n.mu.Unlock()
+			return
+		}
+		// Dedupe is convenience state, not notification authority. Prune expired
+		// keys and cap retained identities so attacker-controlled IP/path churn
+		// cannot grow the process indefinitely. At saturation we still emit the
+		// notification; we simply stop remembering additional dedupe keys.
+		for key, exp := range n.dedupe {
+			if !now.Before(exp) {
+				delete(n.dedupe, key)
+			}
+		}
+		if len(n.dedupe) < notificationDedupeMax {
+			n.dedupe[dedupeKey] = now.Add(60 * time.Second)
+		}
+	}
+	n.nextID++
+	item := notification{
+		ID: n.nextID, Time: time.Now().Format("15:04:05"), Kind: kind, Level: level,
+		Title: title, Body: body, Action: action, Payload: payload,
+	}
+	n.items = append(n.items, item)
+	if len(n.items) > n.cap {
+		n.items = n.items[len(n.items)-n.cap:]
+	}
+	hook := n.cfg.WebhookURL
+	kindHook := n.cfg.WebhookKind
+	sink := n.sink
+	n.mu.Unlock()
+
+	if sink != nil {
+		sink(level, kind, title, body)
+	}
+	if hook != "" {
+		select {
+		case n.webhookQ <- webhookJob{url: hook, kind: kindHook, item: item}:
+		default:
+			atomic.AddUint64(&n.webhookDropped, 1)
+			n.log.Warn("notification webhook queue full; dropping delivery")
+		}
+	}
+}
+
+func (n *notifier) webhookWorker() {
+	for job := range n.webhookQ {
+		for attempt := 0; attempt < 3; attempt++ {
+			retry, err := n.sendWebhook(job.url, job.kind, job.item)
+			if err == nil || !retry {
+				break
+			}
+			time.Sleep(time.Duration(1<<attempt) * 250 * time.Millisecond)
+		}
+	}
+}
+
+func (n *notifier) sendWebhook(url, kind string, item notification) (bool, error) {
+	var payload any
+	text := "[" + item.Level + "] " + item.Title + " — " + item.Body
+	if kind == "slack" {
+		payload = map[string]any{"text": text}
+	} else {
+		payload = map[string]any{
+			"level": item.Level, "kind": item.Kind, "title": item.Title,
+			"body": item.Body, "time": item.Time,
+		}
+	}
+	b, _ := json.Marshal(payload)
+	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
+	defer cancel()
+	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
+	if err != nil {
+		return false, err
+	}
+	req.Header.Set("Content-Type", "application/json")
+	resp, err := n.client.Do(req)
+	if err != nil {
+		n.log.Warn("notify webhook failed", "err", err)
+		return true, err
+	}
+	_ = resp.Body.Close()
+	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
+		return false, nil
+	}
+	err = fmt.Errorf("webhook HTTP %d", resp.StatusCode)
+	n.log.Warn("notify webhook rejected", "status", resp.StatusCode)
+	return resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500, err
+}
+
+func (n *notifier) list(limit int) []notification {
+	n.mu.Lock()
+	defer n.mu.Unlock()
+	total := len(n.items)
+	if limit <= 0 || limit > total {
+		limit = total
+	}
+	out := make([]notification, limit)
+	for i := 0; i < limit; i++ {
+		out[i] = n.items[total-1-i] // newest first
+	}
+	return out
+}
+
+func (n *notifier) unreadCount() int {
+	n.mu.Lock()
+	defer n.mu.Unlock()
+	c := 0
+	for _, it := range n.items {
+		if !it.Read {
+			c++
+		}
+	}
+	return c
+}
+
+func (n *notifier) webhookDeliveryStats() map[string]any {
+	if n == nil || n.webhookQ == nil {
+		return map[string]any{"queue_depth": 0, "queue_capacity": 0, "dropped": uint64(0)}
+	}
+	return map[string]any{
+		"queue_depth":    len(n.webhookQ),
+		"queue_capacity": cap(n.webhookQ),
+		"dropped":        atomic.LoadUint64(&n.webhookDropped),
+	}
+}
+
+func (n *notifier) markRead(id int64, all bool) {
+	n.mu.Lock()
+	defer n.mu.Unlock()
+	for i := range n.items {
+		if all || n.items[i].ID == id {
+			n.items[i].Read = true
+		}
+	}
+}
+
+func (n *notifier) dismiss(id int64, all bool) {
+	n.mu.Lock()
+	defer n.mu.Unlock()
+	if all {
+		n.items = nil
+		return
+	}
+	out := n.items[:0]
+	for _, it := range n.items {
+		if it.ID != id {
+			out = append(out, it)
+		}
+	}
+	n.items = out
+}
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/positive_schema_api4.go ./positive_schema_api4.go
--- /mnt/data/waf_prod_hardening_base/positive_schema_api4.go	2026-09-25 11:11:25.285043065 +0000
+++ ./positive_schema_api4.go	2026-09-25 11:36:44.234811603 +0000
@@ -857,6 +857,9 @@
 	if err := readJSONIfExists(apiSecurityStatePath(configPath, "api-positive-schema.json"), &state); err != nil {
 		return err
 	}
+	if err := validateAPISecurityStateVersion("api-positive-schema", state.Version); err != nil {
+		return err
+	}
 	if len(state.Profiles) == 0 && len(state.Deployments) == 0 && len(state.Exceptions) == 0 && len(state.Violations) == 0 {
 		return nil
 	}
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/production_control_plane_hardening_test.go ./production_control_plane_hardening_test.go
--- /mnt/data/waf_prod_hardening_base/production_control_plane_hardening_test.go	1970-01-01 00:00:00.000000000 +0000
+++ ./production_control_plane_hardening_test.go	2026-09-25 12:49:39.820169554 +0000
@@ -0,0 +1,527 @@
+package main
+
+import (
+	"context"
+	"encoding/json"
+	"io"
+	"log/slog"
+	"net/http"
+	"net/http/httptest"
+	"os"
+	"path/filepath"
+	"strconv"
+	"strings"
+	"sync"
+	"testing"
+	"time"
+)
+
+func TestProductionCorrelationIDValidation(t *testing.T) {
+	valid := []string{"request-1234", "abc_DEF.123:xyz", "01234567"}
+	for _, id := range valid {
+		if !validRequestCorrelationID(id) {
+			t.Fatalf("expected valid request id %q", id)
+		}
+	}
+	invalid := []string{"short", "<script>alert(1)</script>", "contains space", strings.Repeat("a", 129)}
+	for _, id := range invalid {
+		if validRequestCorrelationID(id) {
+			t.Fatalf("expected invalid request id %q", id)
+		}
+	}
+}
+
+func TestProductionBlockResponseEscapesReasonAndRequestID(t *testing.T) {
+	req := httptest.NewRequest(http.MethodGet, "https://example.test/", nil)
+	req = req.WithContext(context.WithValue(req.Context(), requestCorrelationKey{}, `<img src=x onerror=alert(1)>`))
+	rr := httptest.NewRecorder()
+	writeBlockResponse(rr, req, http.StatusForbidden, `<script>alert("reason")</script>`, false)
+	body := rr.Body.String()
+	if strings.Contains(body, "<script>") || strings.Contains(body, "<img src=x") {
+		t.Fatalf("block page reflected raw HTML: %s", body)
+	}
+	if !strings.Contains(body, "&lt;script&gt;") || !strings.Contains(body, "&lt;img") {
+		t.Fatalf("block page did not HTML-escape untrusted values: %s", body)
+	}
+	if got := rr.Header().Get("Cache-Control"); got != "no-store" {
+		t.Fatalf("Cache-Control=%q", got)
+	}
+}
+
+func TestProductionHASharedConfigStripsNodeLocalAndSecrets(t *testing.T) {
+	cfg := defaultConfig()
+	cfg.HA = HAConfig{Enabled: true, Role: "primary", PeerURL: "https://peer.example", PeerToken: "peer-secret", SyncConfig: true}
+	cfg.Users = []UserConfig{{Username: "admin", PasswordHash: "secret-hash", Role: roleAdmin}}
+	cfg.AI.APIKey = "inline-ai-secret"
+	cfg.AI.APIKeyRef = "env:AI_SECRET"
+	cfg.Notify.WebhookURL = "https://hooks.example/secret"
+	out := sharedConfigForPeer(cfg)
+	if out.HA != (HAConfig{}) {
+		t.Fatalf("HA node identity leaked into shared payload: %+v", out.HA)
+	}
+	if len(out.Users) != 0 {
+		t.Fatalf("users leaked into shared payload")
+	}
+	if out.AI.APIKey != "" || out.AI.APIKeyRef != "" || out.Notify.WebhookURL != "" {
+		t.Fatalf("secret material leaked into shared payload")
+	}
+}
+
+func TestProductionHAMergeRestoresReceiverLocalState(t *testing.T) {
+	local := defaultConfig()
+	local.HA = HAConfig{Enabled: true, Role: "secondary", PeerURL: "https://primary.example", PeerToken: "receiver-secret", SyncConfig: true}
+	local.Users = []UserConfig{{Username: "receiver", PasswordHash: "receiver-hash", Role: roleAdmin}}
+	local.AI.APIKeyRef = "env:RECEIVER_AI_KEY"
+	local.Notify.WebhookURL = "https://hooks.example/receiver"
+	incoming := defaultConfig()
+	incoming.EngineMode = "On"
+	incoming.HA = HAConfig{Enabled: true, Role: "primary", PeerURL: "https://wrong.example", PeerToken: "wrong", SyncConfig: true}
+	merged := mergePeerConfig(local, incoming)
+	if merged.HA != local.HA {
+		t.Fatalf("receiver HA identity was not preserved: %+v", merged.HA)
+	}
+	if len(merged.Users) != 1 || merged.Users[0].Username != "receiver" {
+		t.Fatalf("receiver users were not preserved")
+	}
+	if merged.AI.APIKeyRef != local.AI.APIKeyRef || merged.Notify.WebhookURL != local.Notify.WebhookURL {
+		t.Fatalf("receiver secrets were not preserved")
+	}
+	if merged.EngineMode != "On" {
+		t.Fatalf("non-local shared config did not merge")
+	}
+}
+
+func TestProductionHARequiresHTTPSOrigin(t *testing.T) {
+	bad := []string{"http://peer.example", "https://user@peer.example", "https://peer.example/path", "https://peer.example?q=1"}
+	for _, peer := range bad {
+		cfg := HAConfig{Enabled: true, Role: "primary", PeerURL: peer, PeerToken: "x", SyncConfig: true}
+		if err := cfg.validate(); err == nil {
+			t.Fatalf("expected HA peer URL rejection for %q", peer)
+		}
+	}
+	good := HAConfig{Enabled: true, Role: "primary", PeerURL: "https://peer.example:9443", PeerToken: "x", SyncConfig: true}
+	if err := good.validate(); err != nil {
+		t.Fatalf("valid HA peer rejected: %v", err)
+	}
+}
+
+func TestProductionHAPeerAuthUsesDedicatedToken(t *testing.T) {
+	a := &adminServer{token: "admin-secret", haPeerToken: "replica-secret", haPeerTokenPinned: true}
+	h := a.haPeerAuth(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
+
+	req := httptest.NewRequest(http.MethodPut, "/api/ha/peer-config", nil)
+	req.Header.Set("Authorization", "Bearer admin-secret")
+	req.Header.Set("X-WAF-HA-Sync", "v1")
+	rr := httptest.NewRecorder()
+	h(rr, req)
+	if rr.Code != http.StatusUnauthorized {
+		t.Fatalf("break-glass admin token unexpectedly authenticated HA replication: %d", rr.Code)
+	}
+
+	req = httptest.NewRequest(http.MethodPut, "/api/ha/peer-config", nil)
+	req.Header.Set("Authorization", "Bearer replica-secret")
+	rr = httptest.NewRecorder()
+	h(rr, req)
+	if rr.Code != http.StatusForbidden {
+		t.Fatalf("HA protocol marker must be required: %d", rr.Code)
+	}
+
+	req = httptest.NewRequest(http.MethodPut, "/api/ha/peer-config", nil)
+	req.Header.Set("Authorization", "Bearer replica-secret")
+	req.Header.Set("X-WAF-HA-Sync", "v1")
+	rr = httptest.NewRecorder()
+	h(rr, req)
+	if rr.Code != http.StatusNoContent {
+		t.Fatalf("dedicated HA token rejected: %d", rr.Code)
+	}
+}
+
+func TestProductionHASyncCoalescesLatestPendingConfig(t *testing.T) {
+	var mu sync.Mutex
+	modes := []string{}
+	firstStarted := make(chan struct{})
+	releaseFirst := make(chan struct{})
+	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
+		defer r.Body.Close()
+		var env haSyncEnvelope
+		if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
+			t.Errorf("decode sync envelope: %v", err)
+			w.WriteHeader(http.StatusBadRequest)
+			return
+		}
+		mu.Lock()
+		modes = append(modes, env.Config.EngineMode)
+		idx := len(modes)
+		mu.Unlock()
+		if idx == 1 {
+			close(firstStarted)
+			<-releaseFirst
+		}
+		w.WriteHeader(http.StatusNoContent)
+	}))
+	defer server.Close()
+
+	log := slog.New(slog.NewTextHandler(io.Discard, nil))
+	notify := newNotifier(log)
+	h := newHAEngine(log, notify)
+	h.client = server.Client()
+	h.configure(HAConfig{Enabled: true, Role: "primary", PeerURL: server.URL, PeerToken: "peer-secret", SyncConfig: true})
+
+	first := defaultConfig()
+	first.EngineMode = "DetectionOnly"
+	latest := first
+	latest.EngineMode = "On"
+	h.pushConfig(first)
+	select {
+	case <-firstStarted:
+	case <-time.After(2 * time.Second):
+		t.Fatal("first HA sync did not start")
+	}
+	h.pushConfig(latest)
+	close(releaseFirst)
+
+	deadline := time.Now().Add(3 * time.Second)
+	for time.Now().Before(deadline) {
+		mu.Lock()
+		count := len(modes)
+		mu.Unlock()
+		h.syncMu.Lock()
+		running := h.syncRunning
+		h.syncMu.Unlock()
+		if count >= 2 && !running {
+			break
+		}
+		time.Sleep(10 * time.Millisecond)
+	}
+	mu.Lock()
+	defer mu.Unlock()
+	if len(modes) != 2 || modes[0] != "DetectionOnly" || modes[1] != "On" {
+		t.Fatalf("HA coalescing did not converge to latest config: %#v", modes)
+	}
+}
+
+func TestProductionCloneConfigDoesNotShareNestedSlices(t *testing.T) {
+	cfg := defaultConfig()
+	cfg.Sites[0].Hostnames = []string{"original.example"}
+	cloned, err := cloneConfig(cfg)
+	if err != nil {
+		t.Fatal(err)
+	}
+	cloned.Sites[0].Hostnames[0] = "changed.example"
+	if cfg.Sites[0].Hostnames[0] != "original.example" {
+		t.Fatal("cloneConfig shared nested slice storage with live config")
+	}
+}
+
+func TestProductionLoginLimiterBlocksAndResets(t *testing.T) {
+	l := newLoginAttemptLimiter()
+	now := time.Unix(1_700_000_000, 0)
+	keys := []string{"ip:192.0.2.10", "user:admin"}
+	for i := 0; i < loginAttemptLimit; i++ {
+		l.failure(keys, now.Add(time.Duration(i)*time.Second))
+	}
+	if ok, retry := l.allow(keys, now.Add(time.Duration(loginAttemptLimit)*time.Second)); ok || retry <= 0 {
+		t.Fatalf("expected blocked login, ok=%v retry=%v", ok, retry)
+	}
+	l.success(keys)
+	if ok, _ := l.allow(keys, now.Add(time.Minute)); !ok {
+		t.Fatal("successful login did not reset limiter")
+	}
+}
+
+func TestProductionLoginLimiterSaturationDoesNotEvictBlockedKeys(t *testing.T) {
+	l := newLoginAttemptLimiter()
+	now := time.Unix(1_700_000_000, 0)
+	blocked := []string{"ip:192.0.2.44"}
+	for i := 0; i < loginAttemptLimit; i++ {
+		l.failure(blocked, now.Add(time.Duration(i)*time.Millisecond))
+	}
+	for i := 0; i < loginAttemptMaxKey+128; i++ {
+		l.failure([]string{"user:flood-" + strconv.Itoa(i)}, now)
+	}
+	if ok, _ := l.allow(blocked, now.Add(time.Second)); ok {
+		t.Fatal("bounded login limiter evicted a blocked identity under key-space saturation")
+	}
+	// Once the key map is full, unseen identities share the bounded overflow
+	// bucket rather than creating unbounded state or flushing established bans.
+	for i := 0; i < loginAttemptLimit; i++ {
+		l.failure([]string{"user:overflow-" + strconv.Itoa(i)}, now)
+	}
+	if ok, _ := l.allow([]string{"user:another-unseen"}, now.Add(time.Second)); ok {
+		t.Fatal("saturated login limiter failed open for a new identity")
+	}
+}
+
+func TestProductionHealthJSONEncodesRole(t *testing.T) {
+	rr := httptest.NewRecorder()
+	writeHealth(rr, true, false, `active\"<script>`)
+	var body map[string]any
+	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
+		t.Fatalf("health response is not valid JSON: %v body=%q", err, rr.Body.String())
+	}
+	if body["role"] != `active\"<script>` {
+		t.Fatalf("role round-trip mismatch: %#v", body["role"])
+	}
+}
+
+func TestProductionPBKDF2RejectsAbsurdIterationCount(t *testing.T) {
+	encoded := "pbkdf2$sha256$1000001$c2FsdHNhbHQ=$AAAAAAAAAAAAAAAAAAAAAA=="
+	if verifyPassword("password", encoded) {
+		t.Fatal("absurd PBKDF2 iteration count must be rejected")
+	}
+}
+
+func TestProductionDirectUpdateDisabledWithoutExplicitDirectory(t *testing.T) {
+	t.Setenv("WAF_UPDATE_INSTALL_DIR", "")
+	if directUpdateInstallEnabled() {
+		t.Fatal("direct update install must be disabled by default")
+	}
+	t.Setenv("WAF_UPDATE_INSTALL_DIR", "relative/path")
+	if directUpdateInstallEnabled() {
+		t.Fatal("relative update install directory must not enable direct install")
+	}
+	exe, err := os.Executable()
+	if err != nil {
+		t.Fatal(err)
+	}
+	t.Setenv("WAF_UPDATE_INSTALL_DIR", filepath.Dir(exe))
+	if !directUpdateInstallEnabled() {
+		t.Fatal("current executable directory should enable explicit standalone install")
+	}
+	t.Setenv("WAF_UPDATE_INSTALL_DIR", t.TempDir())
+	if directUpdateInstallEnabled() {
+		t.Fatal("a different directory would install bytes that ReExec cannot activate")
+	}
+}
+
+func TestProductionUpdateLocalhostRejectsForwardedRequests(t *testing.T) {
+	r := httptest.NewRequest(http.MethodPost, "http://localhost/api/update/install", nil)
+	r.RemoteAddr = "127.0.0.1:12345"
+	if !updateLocalhost(r) {
+		t.Fatal("direct loopback request should be accepted")
+	}
+	r.Header.Set("X-Forwarded-For", "203.0.113.7")
+	if updateLocalhost(r) {
+		t.Fatal("forwarded loopback request must not satisfy localhost-only guard")
+	}
+}
+
+func TestProductionL7SaturationUsesBoundedOverflowBucket(t *testing.T) {
+	c := newL7AbuseController(L7AbuseConfig{Enabled: true, RequestsPerWindow: 1, WindowSeconds: 60})
+	now := time.Unix(1_700_000_000, 0)
+	const targetShard = 0
+	filled := 0
+	for i := 0; filled < l7AbuseMaxEntriesPerShard; i++ {
+		identity := "198.51.100." + strconv.Itoa(i)
+		key := abuseClientKey{site: "site", identity: identity}
+		if l7AbuseShardIndex(key) != targetShard {
+			continue
+		}
+		state, ok := c.allowAt("site", identity, now)
+		if !ok || state == nil {
+			t.Fatalf("failed filling bounded shard at %d", filled)
+		}
+		c.releaseState(state)
+		filled++
+	}
+	var firstOverflow, secondOverflow string
+	for i := 100000; secondOverflow == ""; i++ {
+		identity := "203.0.113." + strconv.Itoa(i)
+		key := abuseClientKey{site: "site", identity: identity}
+		if l7AbuseShardIndex(key) != targetShard {
+			continue
+		}
+		if firstOverflow == "" {
+			firstOverflow = identity
+		} else {
+			secondOverflow = identity
+		}
+	}
+	state, ok := c.allowAt("site", firstOverflow, now)
+	if !ok || state == nil {
+		t.Fatal("first overflow admission should consume the shared bounded token")
+	}
+	c.releaseState(state)
+	if state, ok = c.allowAt("site", secondOverflow, now); ok || state != nil {
+		t.Fatal("saturated unknown identity unexpectedly failed open")
+	}
+}
+
+func TestProductionStageConfigUsesPrivateAtomicFile(t *testing.T) {
+	path := filepath.Join(t.TempDir(), "config.json")
+	cfg := defaultConfig()
+	staged, err := stageConfig(path, cfg)
+	if err != nil {
+		t.Fatal(err)
+	}
+	defer staged.abort()
+	if staged.tmp == path || !strings.HasPrefix(filepath.Base(staged.tmp), ".config.json.tmp-") {
+		t.Fatalf("unexpected staging file %q", staged.tmp)
+	}
+	if info, err := os.Stat(staged.tmp); err != nil || info.Mode().Perm() != 0o600 {
+		t.Fatalf("staged config mode/stat: %v %v", info, err)
+	}
+	committed, err := staged.commit()
+	if err != nil || !committed {
+		t.Fatalf("commit: committed=%v err=%v", committed, err)
+	}
+	if _, err := os.Stat(path); err != nil {
+		t.Fatalf("committed config missing: %v", err)
+	}
+}
+
+func TestProductionAPISecurityStateVersionFailsClosed(t *testing.T) {
+	if err := validateAPISecurityStateVersion("test", 0); err != nil {
+		t.Fatalf("legacy v0 must remain readable: %v", err)
+	}
+	if err := validateAPISecurityStateVersion("test", apiSecurityStateVersion); err != nil {
+		t.Fatalf("current state version rejected: %v", err)
+	}
+	if err := validateAPISecurityStateVersion("test", apiSecurityStateVersion+1); err == nil {
+		t.Fatal("future state version must fail closed")
+	}
+}
+
+func TestProductionSitemapPersistenceUsesUniqueDurableTemp(t *testing.T) {
+	dir := t.TempDir()
+	configPath := filepath.Join(dir, "config.json")
+	maps := newSiteMaps(128)
+	maps.record("site-a", http.MethodGet, "/products/42", http.StatusOK, srcObserved)
+
+	var wg sync.WaitGroup
+	errCh := make(chan error, 8)
+	for i := 0; i < 8; i++ {
+		wg.Add(1)
+		go func() {
+			defer wg.Done()
+			errCh <- maps.save(configPath)
+		}()
+	}
+	wg.Wait()
+	close(errCh)
+	for err := range errCh {
+		if err != nil {
+			t.Fatalf("concurrent sitemap save failed: %v", err)
+		}
+	}
+
+	b, err := os.ReadFile(filepath.Join(dir, "sitemap.json"))
+	if err != nil {
+		t.Fatal(err)
+	}
+	var state siteMapsFile
+	if err := json.Unmarshal(b, &state); err != nil {
+		t.Fatalf("persisted sitemap is not valid JSON: %v", err)
+	}
+	if len(state.Sites) != 1 || state.Sites[0].Site != "site-a" {
+		t.Fatalf("unexpected sitemap state: %+v", state.Sites)
+	}
+	leftovers, err := filepath.Glob(filepath.Join(dir, ".sitemap.json.tmp-*"))
+	if err != nil {
+		t.Fatal(err)
+	}
+	if len(leftovers) != 0 {
+		t.Fatalf("temporary sitemap files leaked: %v", leftovers)
+	}
+}
+
+func TestProductionSitemapRestoreSnapshotAfterFailedClearPath(t *testing.T) {
+	maps := newSiteMaps(128)
+	maps.record("site-a", http.MethodGet, "/orders/123", http.StatusOK, srcObserved)
+	before := maps.snapshot("site-a")
+	maps.clear("site-a")
+	maps.restoreSnapshot(before)
+	after := maps.snapshot("site-a")
+	if after.Site != before.Site || after.Nodes != before.Nodes {
+		t.Fatalf("restored sitemap metadata differs: before=%+v after=%+v", before, after)
+	}
+	beforeJSON, err := json.Marshal(before.Tree)
+	if err != nil {
+		t.Fatal(err)
+	}
+	afterJSON, err := json.Marshal(after.Tree)
+	if err != nil {
+		t.Fatal(err)
+	}
+	if string(beforeJSON) != string(afterJSON) {
+		t.Fatalf("restored sitemap tree differs: before=%s after=%s", beforeJSON, afterJSON)
+	}
+}
+
+func TestProductionDraftConfigIsSeparateFromStartupAuthority(t *testing.T) {
+	dir := t.TempDir()
+	livePath := filepath.Join(dir, "config.json")
+	live := defaultConfig()
+	live.EngineMode = "DetectionOnly"
+	if err := saveConfig(livePath, live); err != nil {
+		t.Fatal(err)
+	}
+
+	draft := live
+	draft.EngineMode = "On"
+	if err := saveConfig(draftConfigPath(livePath), draft); err != nil {
+		t.Fatal(err)
+	}
+	// Ensure the draft is observably newer even on coarse timestamp filesystems.
+	future := time.Now().Add(2 * time.Second)
+	if err := os.Chtimes(draftConfigPath(livePath), future, future); err != nil {
+		t.Fatal(err)
+	}
+
+	startup, err := loadConfig(livePath)
+	if err != nil {
+		t.Fatal(err)
+	}
+	if startup.EngineMode != "DetectionOnly" {
+		t.Fatalf("startup authority unexpectedly read draft: %q", startup.EngineMode)
+	}
+	resumed, ok, err := loadDraftConfig(livePath)
+	if err != nil {
+		t.Fatal(err)
+	}
+	if !ok || resumed.EngineMode != "On" {
+		t.Fatalf("newer draft was not resumable: ok=%t mode=%q", ok, resumed.EngineMode)
+	}
+}
+
+func TestProductionStaleDraftIgnoredAfterNewerLiveCommit(t *testing.T) {
+	dir := t.TempDir()
+	livePath := filepath.Join(dir, "config.json")
+	base := defaultConfig()
+	if err := saveConfig(livePath, base); err != nil {
+		t.Fatal(err)
+	}
+	draft := base
+	draft.EngineMode = "DetectionOnly"
+	if err := saveConfig(draftConfigPath(livePath), draft); err != nil {
+		t.Fatal(err)
+	}
+	past := time.Now().Add(-2 * time.Second)
+	if err := os.Chtimes(draftConfigPath(livePath), past, past); err != nil {
+		t.Fatal(err)
+	}
+	live := base
+	live.EngineMode = "On"
+	if err := saveConfig(livePath, live); err != nil {
+		t.Fatal(err)
+	}
+	if _, ok, err := loadDraftConfig(livePath); err != nil || ok {
+		t.Fatalf("stale draft should be ignored after newer live commit: ok=%t err=%v", ok, err)
+	}
+}
+
+func TestProductionNotificationDedupeStateIsBounded(t *testing.T) {
+	log := slog.New(slog.NewTextHandler(io.Discard, nil))
+	n := newNotifier(log)
+	for i := 0; i < notificationDedupeMax+1024; i++ {
+		n.push(notifySuggestion, "info", "test", "body", "key-"+strconv.Itoa(i), "", nil)
+	}
+	n.mu.Lock()
+	defer n.mu.Unlock()
+	if len(n.dedupe) > notificationDedupeMax {
+		t.Fatalf("notification dedupe state exceeded cap: %d", len(n.dedupe))
+	}
+	if len(n.items) != n.cap {
+		t.Fatalf("notification ring cap not preserved: got %d want %d", len(n.items), n.cap)
+	}
+}
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/sitemap.go ./sitemap.go
--- /mnt/data/waf_prod_hardening_base/sitemap.go	2026-09-25 11:11:25.288629253 +0000
+++ ./sitemap.go	2026-09-25 12:42:22.373811609 +0000
@@ -74,9 +74,10 @@
 }
 
 type siteMaps struct {
-	mu       sync.Mutex
-	byName   map[string]*siteMap
-	maxNodes int
+	mu        sync.Mutex
+	persistMu sync.Mutex
+	byName    map[string]*siteMap
+	maxNodes  int
 }
 
 func newSiteMaps(maxNodes int) *siteMaps {
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/sitemap_persist.go ./sitemap_persist.go
--- /mnt/data/waf_prod_hardening_base/sitemap_persist.go	2026-09-25 11:11:25.288690886 +0000
+++ ./sitemap_persist.go	2026-09-25 12:42:22.374027585 +0000
@@ -1,139 +1,190 @@
-package main
-
-// Site-map persistence.
-//
-// The site content map is built in memory as traffic flows (and by the
-// crawler). Without persistence it resets on every restart — and since a
-// binary upgrade requires a restart, rebuilding the tool throws the map away.
-// This saves all site maps to a JSON file next to config.json and reloads them
-// at startup, so the observed structure survives restarts and upgrades.
-//
-// The file is written atomically (temp + rename) into the same dir as the
-// config, which is already group-writable by the waf user.
-
-import (
-	"encoding/json"
-	"os"
-	"path/filepath"
-	"sort"
-	"strings"
-	"sync"
-	"time"
-)
-
-type siteMapsFile struct {
-	Saved time.Time      `json:"saved"`
-	Sites []siteMapJSON  `json:"sites"`
-}
-
-func (m *siteMaps) statePath(configPath string) string {
-	return filepath.Join(filepath.Dir(configPath), "sitemap.json")
-}
-
-// save writes every site's tree to disk atomically.
-func (m *siteMaps) save(configPath string) error {
-	m.mu.Lock()
-	names := make([]string, 0, len(m.byName))
-	for n := range m.byName {
-		names = append(names, n)
-	}
-	m.mu.Unlock()
-	sort.Strings(names)
-
-	out := siteMapsFile{Saved: time.Now()}
-	for _, n := range names {
-		out.Sites = append(out.Sites, m.snapshot(n))
-	}
-	b, err := json.MarshalIndent(out, "", "  ")
-	if err != nil {
-		return err
-	}
-	path := m.statePath(configPath)
-	tmp := path + ".tmp"
-	if err := os.WriteFile(tmp, b, 0o640); err != nil {
-		return err
-	}
-	return os.Rename(tmp, path)
-}
-
-// load rebuilds site maps from disk. Missing file is not an error.
-func (m *siteMaps) load(configPath string) error {
-	b, err := os.ReadFile(m.statePath(configPath))
-	if err != nil {
-		if os.IsNotExist(err) {
-			return nil
-		}
-		return err
-	}
-	var in siteMapsFile
-	if err := json.Unmarshal(b, &in); err != nil {
-		return err
-	}
-	m.mu.Lock()
-	defer m.mu.Unlock()
-	for _, sj := range in.Sites {
-		sm := &siteMap{root: newNode("", "/")}
-		sm.nodes = sj.Nodes
-		fromJSON(sm.root, sj.Tree)
-		m.byName[sj.Site] = sm
-	}
-	return nil
-}
-
-// fromJSON rebuilds a pathNode subtree from its serialized form (inverse of
-// pathNode.toJSON).
-func fromJSON(n *pathNode, j nodeJSON) {
-	n.full = j.Full
-	n.hits = j.Hits
-	n.lastCode = j.LastCode
-	n.methods = map[string]struct{}{}
-	for _, mth := range j.Methods {
-		n.methods[mth] = struct{}{}
-	}
-	switch j.Source {
-	case "both":
-		n.seen, n.crawled = true, true
-	case srcObserved:
-		n.seen = true
-	case srcCrawled:
-		n.crawled = true
-	}
-	if j.LastSeen != "" {
-		// Stored as clock time only; anchor to today so ordering is sane.
-		if t, err := time.Parse("15:04:05", j.LastSeen); err == nil {
-			now := time.Now()
-			n.lastSeen = time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), t.Second(), 0, now.Location())
-		}
-	}
-	n.children = map[string]*pathNode{}
-	for _, cj := range j.Children {
-		name := cj.Name
-		if name == "/" {
-			name = ""
-		}
-		child := newNode(name, cj.Full)
-		fromJSON(child, cj)
-		n.children[strings.ToLower(cj.Name)] = child
-	}
-}
-
-// startAutosave periodically flushes site maps to disk until ctx is done, and
-// writes a final snapshot on exit. Interval is generous — this is observed
-// state, not critical data, so we trade freshness for negligible I/O.
-func (m *siteMaps) startAutosave(configPath string, every time.Duration, stop <-chan struct{}, wg *sync.WaitGroup) {
-	wg.Add(1)
-	go func() {
-		defer wg.Done()
-		t := time.NewTicker(every)
-		defer t.Stop()
-		for {
-			select {
-			case <-stop:
-				_ = m.save(configPath)
-				return
-			case <-t.C:
-				_ = m.save(configPath)
-			}
-		}
-	}()
-}
+package main
+
+// Site-map persistence.
+//
+// The site content map is built in memory as traffic flows (and by the
+// crawler). Without persistence it resets on every restart — and since a
+// binary upgrade requires a restart, rebuilding the tool throws the map away.
+// This saves all site maps to a JSON file next to config.json and reloads them
+// at startup, so the observed structure survives restarts and upgrades.
+//
+// The file is written atomically (temp + rename) into the same dir as the
+// config, which is already group-writable by the waf user.
+
+import (
+	"encoding/json"
+	"os"
+	"path/filepath"
+	"sort"
+	"strings"
+	"sync"
+	"time"
+)
+
+type siteMapsFile struct {
+	Saved time.Time     `json:"saved"`
+	Sites []siteMapJSON `json:"sites"`
+}
+
+func (m *siteMaps) statePath(configPath string) string {
+	return filepath.Join(filepath.Dir(configPath), "sitemap.json")
+}
+
+// save writes every site's tree to disk atomically. Persistence is serialized
+// independently from the hot-path tree mutex so periodic autosave and explicit
+// operator clears cannot race on the same state file or temporary path.
+func (m *siteMaps) save(configPath string) error {
+	m.persistMu.Lock()
+	defer m.persistMu.Unlock()
+
+	m.mu.Lock()
+	names := make([]string, 0, len(m.byName))
+	for n := range m.byName {
+		names = append(names, n)
+	}
+	m.mu.Unlock()
+	sort.Strings(names)
+
+	out := siteMapsFile{Saved: time.Now()}
+	for _, n := range names {
+		out.Sites = append(out.Sites, m.snapshot(n))
+	}
+	b, err := json.MarshalIndent(out, "", "  ")
+	if err != nil {
+		return err
+	}
+	return writeSiteMapsFile(m.statePath(configPath), b)
+}
+
+func writeSiteMapsFile(path string, b []byte) error {
+	dir := filepath.Dir(path)
+	tmp, err := os.CreateTemp(dir, ".sitemap.json.tmp-*")
+	if err != nil {
+		return err
+	}
+	tmpName := tmp.Name()
+	keep := false
+	defer func() {
+		_ = tmp.Close()
+		if !keep {
+			_ = os.Remove(tmpName)
+		}
+	}()
+	if err := tmp.Chmod(0o640); err != nil {
+		return err
+	}
+	if _, err := tmp.Write(b); err != nil {
+		return err
+	}
+	if err := tmp.Sync(); err != nil {
+		return err
+	}
+	if err := tmp.Close(); err != nil {
+		return err
+	}
+	if err := os.Rename(tmpName, path); err != nil {
+		return err
+	}
+	keep = true
+	d, err := os.Open(dir)
+	if err != nil {
+		return err
+	}
+	defer d.Close()
+	return d.Sync()
+}
+
+// restoreSnapshot reinstates one site map after a failed destructive persistence
+// operation. It is intentionally narrow: clear handlers use it to avoid leaving
+// in-memory state cleared when the authoritative snapshot could not be updated.
+func (m *siteMaps) restoreSnapshot(sj siteMapJSON) {
+	sm := &siteMap{root: newNode("", "/")}
+	sm.nodes = sj.Nodes
+	sm.crawl = sj.Crawl
+	fromJSON(sm.root, sj.Tree)
+	m.mu.Lock()
+	m.byName[sj.Site] = sm
+	m.mu.Unlock()
+}
+
+// load rebuilds site maps from disk. Missing file is not an error.
+func (m *siteMaps) load(configPath string) error {
+	b, err := os.ReadFile(m.statePath(configPath))
+	if err != nil {
+		if os.IsNotExist(err) {
+			return nil
+		}
+		return err
+	}
+	var in siteMapsFile
+	if err := json.Unmarshal(b, &in); err != nil {
+		return err
+	}
+	m.mu.Lock()
+	defer m.mu.Unlock()
+	for _, sj := range in.Sites {
+		sm := &siteMap{root: newNode("", "/")}
+		sm.nodes = sj.Nodes
+		fromJSON(sm.root, sj.Tree)
+		m.byName[sj.Site] = sm
+	}
+	return nil
+}
+
+// fromJSON rebuilds a pathNode subtree from its serialized form (inverse of
+// pathNode.toJSON).
+func fromJSON(n *pathNode, j nodeJSON) {
+	n.full = j.Full
+	n.hits = j.Hits
+	n.lastCode = j.LastCode
+	n.methods = map[string]struct{}{}
+	for _, mth := range j.Methods {
+		n.methods[mth] = struct{}{}
+	}
+	switch j.Source {
+	case "both":
+		n.seen, n.crawled = true, true
+	case srcObserved:
+		n.seen = true
+	case srcCrawled:
+		n.crawled = true
+	}
+	if j.LastSeen != "" {
+		// Stored as clock time only; anchor to today so ordering is sane.
+		if t, err := time.Parse("15:04:05", j.LastSeen); err == nil {
+			now := time.Now()
+			n.lastSeen = time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), t.Second(), 0, now.Location())
+		}
+	}
+	n.children = map[string]*pathNode{}
+	for _, cj := range j.Children {
+		name := cj.Name
+		if name == "/" {
+			name = ""
+		}
+		child := newNode(name, cj.Full)
+		fromJSON(child, cj)
+		n.children[strings.ToLower(cj.Name)] = child
+	}
+}
+
+// startAutosave periodically flushes site maps to disk until ctx is done, and
+// writes a final snapshot on exit. Interval is generous — this is observed
+// state, not critical data, so we trade freshness for negligible I/O.
+func (m *siteMaps) startAutosave(configPath string, every time.Duration, stop <-chan struct{}, wg *sync.WaitGroup) {
+	wg.Add(1)
+	go func() {
+		defer wg.Done()
+		t := time.NewTicker(every)
+		defer t.Stop()
+		for {
+			select {
+			case <-stop:
+				_ = m.save(configPath)
+				return
+			case <-t.C:
+				_ = m.save(configPath)
+			}
+		}
+	}()
+}
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/static/admin.html ./static/admin.html
--- /mnt/data/waf_prod_hardening_base/static/admin.html	2026-09-25 11:11:25.289339374 +0000
+++ ./static/admin.html	2026-09-25 12:48:49.256912933 +0000
@@ -681,12 +681,12 @@
             <div class="field"><label>Role</label><select id="ha_role"><option value="primary">primary</option><option value="secondary">secondary</option></select></div>
             <div class="field" style="display:flex;align-items:flex-end"><div class="check"><input type="checkbox" id="ha_sync_config"><label for="ha_sync_config" style="margin:0">Sync config to peer</label></div></div>
           </div>
-          <div class="field"><label>Peer admin URL</label><input type="text" id="ha_peer_url" spellcheck="false" placeholder="https://10.0.0.6:9090"></div>
-          <div class="field"><label>Peer admin token <span id="ha_tokenset" class="restart" style="border-color:var(--line);color:var(--mut)">not set</span></label>
+          <div class="field"><label>Peer HA endpoint URL</label><input type="text" id="ha_peer_url" spellcheck="false" placeholder="https://10.0.0.6:9090"></div>
+          <div class="field"><label>Peer dedicated HA token <span id="ha_tokenset" class="restart" style="border-color:var(--line);color:var(--mut)">not set</span></label>
             <input type="password" id="ha_peer_token" spellcheck="false" placeholder="leave blank to keep current"></div>
           <div class="strip" id="ha_status" style="margin-top:4px"></div>
           <div class="actions"><button class="btn primary" id="ha_save">Save &amp; apply</button><button class="btn" id="ha_syncnow">Sync now</button></div>
-          <div class="hint">waf-proxy syncs config and computes an <b>active/standby</b> role from peer health. Actual VIP failover belongs to keepalived/VRRP or your load balancer — point its health check at <code>/healthz</code> and read role from <code>/api/ha</code>. Blocklist state is intentionally not shared.</div>
+          <div class="hint">HA sync uses a dedicated peer-only HTTPS endpoint and a separate replication credential. Configure this node's <code>WAF_HA_PEER_TOKEN</code> locally; enter the peer node's dedicated HA token above. Shared WAF config is replicated, while each node keeps its own HA role/peer/token, users, AI/HSM secret references, and notification webhook secret. Actual VIP failover belongs to keepalived/VRRP or your load balancer — point its health check at <code>/healthz</code> and read role from <code>/api/ha</code>.</div>
         </div>
         <div>
           <div class="eyebrow">Notifications</div>
@@ -697,8 +697,8 @@
           <div class="field check"><input type="checkbox" id="nt_on_member_down"><label for="nt_on_member_down" style="margin:0">Pool member down</label></div>
           <div class="field check"><input type="checkbox" id="nt_on_sync"><label for="nt_on_sync" style="margin:0">Config sync results</label></div>
           <div class="field check"><input type="checkbox" id="nt_on_peer"><label for="nt_on_peer" style="margin:0">HA peer up/down</label></div>
-          <div class="actions"><button class="btn primary" id="nt_save">Save &amp; apply</button></div>
-          <div class="hint">Events always show in the bell; the webhook is an extra fan-out. Suggestions never auto-apply — they arrive with a one-click apply button.</div>
+          <div class="actions"><button class="btn primary" id="nt_save">Save &amp; apply</button><button class="btn danger" id="nt_clear_webhook">Clear stored webhook</button></div>
+          <div class="hint">Events always show in the bell; the webhook is an extra fan-out. Stored webhook URLs are never returned by the config API; leave this field blank to keep the current secret, or use <b>Clear stored webhook</b> to remove it explicitly. Suggestions never auto-apply.</div>
         </div>
       </div>
     </section>
@@ -740,7 +740,7 @@
       <div class="eyebrow">Signed self-update</div>
       <div id="upd_off" class="hint" style="display:none">Signed updates aren't enabled in this build (no publisher key baked in, or not on localhost). Rebuild with a publisher public key, or set <code>WAF_PUBLISHER_KEY_FILE</code>, and access the console over <code>ssh -L</code>.</div>
       <div id="upd_on" style="display:none">
-        <div class="strip" id="upd_status" style="margin-top:0"></div>
+        <div class="strip" id="upd_status" style="margin-top:0"></div><div class="hint" id="upd_mode_hint" style="margin-top:8px"></div>
         <div class="row" style="margin-top:12px">
           <div class="field"><label>Upload signed package (.wafupdate)</label><input type="file" id="upd_file" accept=".wafupdate"></div>
           <div class="field" style="display:flex;align-items:flex-end"><button class="btn" id="upd_stage">Verify &amp; stage</button></div>
@@ -761,7 +761,7 @@
           <button class="btn" id="upd_rollback">Roll back</button>
           <button class="btn danger" id="upd_restart" style="display:none">Restart into new binary</button>
         </div>
-        <div class="hint"><b style="color:var(--amber)">How trust works:</b> packages are verified against the baked-in publisher <b>public</b> key (RSA/SHA-256) before anything is written — a bad signature, checksum, or unsafe path is rejected. The private key stays offline, so a stolen admin session still can't forge an update. Installing replaces the binary and requires a <b>restart</b>; the previous version is kept for one-click rollback. Admin-only, localhost-only.</div>
+        <div class="hint"><b style="color:var(--amber)">How trust works:</b> packages are verified against the baked-in publisher <b>public</b> key (RSA/SHA-256) before anything is written — a bad signature, checksum, or unsafe path is rejected. The private key stays offline, so a stolen admin session still can't forge an update. Hardened DEB/RPM/systemd deployments intentionally use the external package manager for privileged installation; direct install is only available with an explicit standalone <code>WAF_UPDATE_INSTALL_DIR</code>. Admin-only, localhost-only.</div>
       </div>
     </section>
 
@@ -962,7 +962,45 @@
 
   // ── auth ──
   function getToken(){ return sessionStorage.getItem("waf_token")||""; }
+  function requiredCapability(path, method){
+    method=(method||"GET").toUpperCase();
+    if(["GET","HEAD","OPTIONS"].includes(method)) return "";
+    if(path==="/api/login" || path==="/api/logout" || path==="/api/users/password") return "";
+    if(path.startsWith("/api/users/") || path.startsWith("/api/update/")) return "manage";
+    if(path.startsWith("/api/config") || path==="/api/reload" || path.startsWith("/api/ai/") || path.startsWith("/api/syslog/") || path.startsWith("/api/ha/") || path==="/api/crawl" || path.startsWith("/api/debug/capture") || path==="/api/notifications/webhook" || path==="/api/security/contracts/import") return "edit";
+    return "review";
+  }
+  function hasCapability(cap){
+    if(!cap || !curUser) return true;
+    if(cap==="manage") return !!curUser.can_manage_users;
+    if(cap==="edit") return !!curUser.can_edit_config;
+    if(cap==="review") return !!curUser.can_review;
+    return false;
+  }
+  const roleControlIDs={
+    edit:["addsite","save","applybtn","reload","addnode","nodes_save","addpool","pools_save","pools_apply","addpolicy","pol_save","pol_apply","cidr_add","traffic_save","traffic_apply","api_contract_import","tls_accel_save","ai_test","ai_save","ha_save","ha_syncnow","nt_save","nt_clear_webhook","sl_save","sl_test","debug_start","debug_stop","hsm_save","hsm_apply","vector_save","vector_apply"],
+    review:["bellread","bellclear","api_idp_save","api_sequence_mode_apply","api_sequence_relearn","api_sequence_exception_create","api_object_override_save","api_bola_policy_save","api_graphql_contract_import","api_graphql_policy_save","vector_reset","pp_save","pp_delete"],
+    manage:["upd_stage","upd_catalog","upd_install","upd_discard","upd_rollback","upd_restart","nu_add"]
+  };
+  function applyRoleUI(){
+    if(!curUser) return;
+    for(const [cap,ids] of Object.entries(roleControlIDs)) for(const id of ids){
+      const el=$(id); if(!el) continue;
+      const denied=!hasCapability(cap);
+      el.disabled=denied;
+      if(denied) el.title=`${curUser.role||"viewer"} role cannot perform this action`;
+      else if(el.title&&el.title.includes("role cannot perform")) el.removeAttribute("title");
+    }
+    document.querySelectorAll("[data-required-capability]").forEach(el=>{
+      const cap=el.dataset.requiredCapability||"";
+      const denied=!hasCapability(cap);
+      el.disabled=denied;
+      if(denied) el.title=`${curUser.role||"viewer"} role cannot perform this action`;
+    });
+  }
   async function api(path, opts={}){
+    const cap=requiredCapability(path, opts.method||"GET");
+    if(curUser && !hasCapability(cap)) throw new Error(`role ${curUser.role||"viewer"} cannot perform this action`);
     opts.headers = Object.assign({"Authorization":"Bearer "+getToken(),"Content-Type":"application/json"}, opts.headers||{});
     const r = await fetch(path, opts);
     if (r.status===401){ sessionStorage.removeItem("waf_token"); showGate(); throw new Error("session expired — sign in again"); }
@@ -2254,6 +2292,7 @@
       const rec=await api("/api/learn?site="+encodeURIComponent(site));
       const plNote = rec.suggest_paranoia>0 ? ` · suggested paranoia <b>PL${rec.suggest_paranoia}</b>` : "";
       let html=`<div class="hint" style="margin-bottom:8px">${esc(rec.summary)}${plNote}</div>`;
+      if(curUser&&curUser.can_review) html+=`<div class="actions" style="margin:0 0 8px"><button class="btn danger sm clearlearn" data-required-capability="review" data-site="${esc(site)}">Clear learning state</button></div>`;
       if(!rec.pages||!rec.pages.length){ html+=`<div class="empty">No per-page findings yet — send more traffic through the WAF.</div>`; box.innerHTML=html; return; }
       html+=`<table><thead><tr><th>Page</th><th>Risk</th><th>Hits</th><th>Findings</th><th>Suggestion</th><th></th></tr></thead><tbody>`;
       rec.pages.forEach((pg,idx)=>{
@@ -2264,6 +2303,12 @@
       });
       html+=`</tbody></table>`;
       box.innerHTML=html;
+      const clearBtn=box.querySelector(".clearlearn");
+      if(clearBtn) clearBtn.onclick=async()=>{
+        if(!confirm(`Clear learned state for ${site}? This does not remove already-applied policy changes.`)) return;
+        try{ await api("/api/learn/clear?site="+encodeURIComponent(site),{method:"POST",body:"{}"}); toast("learning state cleared for "+site); loadLearn(site,box); }
+        catch(e){ toast(e.message,"err"); }
+      };
       box.querySelectorAll(".applyx").forEach(b=>b.onclick=async()=>{
         const ids=b.dataset.ids.split(",").map(x=>+x).filter(Boolean);
         try{
@@ -2345,6 +2390,11 @@
   }
   $("ha_save").onclick=()=>saveConfig().then(()=>toast("HA settings — draft saved")).catch(e=>toast(e.message,"err"));
   $("nt_save").onclick=()=>saveConfig().then(()=>toast("notifications — draft saved")).catch(e=>toast(e.message,"err"));
+  $("nt_clear_webhook").onclick=async()=>{
+    if(!confirm("Remove the stored notification webhook URL? This cannot be recovered from the Console.")) return;
+    try{ await api("/api/notifications/webhook",{method:"DELETE"}); $("nt_webhook_url").value=""; toast("stored notification webhook cleared"); }
+    catch(e){ toast(e.message,"err"); }
+  };
   $("ha_syncnow").onclick=()=>api("/api/ha/sync",{method:"POST"}).then(()=>toast("sync triggered")).catch(e=>toast(e.message,"err"));
 
   // ── notification bell ──
@@ -2357,6 +2407,9 @@
     try{
       const r=await api("/api/notifications?limit=100");
       const items=r.items||[];
+      const wd=r.webhook_delivery||{};
+      const dropped=Number(wd.dropped||0), depth=Number(wd.queue_depth||0), cap=Number(wd.queue_capacity||0);
+      $("bell").title=dropped?`Notifications — webhook delivery dropped ${dropped}; queue ${depth}/${cap}`:`Notifications — webhook queue ${depth}/${cap}`;
       $("bellempty").style.display=items.length?"none":"";
       $("belllist").innerHTML=items.map(n=>{
         const applyBtn=n.action==="apply_exclusion"?`<button class="btn sm applyn" data-id="${n.id}">Apply</button>`:"";
@@ -2534,8 +2587,13 @@
   async function loadWhoami(){
     try{ curUser=await api("/api/whoami"); }catch(e){ curUser=null; }
     const hdr=$("whoami-hdr"); if(hdr) hdr.textContent=curUser?curUser.user:"—";
-    const w=$("whoami"); if(w && curUser){ w.innerHTML=`<div>signed in as <b>${esc(curUser.user)}</b></div><div>role <b>${esc(curUser.role||"—")}</b></div>`; }
+    const w=$("whoami"); if(w && curUser){
+      const caps=[curUser.can_edit_config?"config edit":"read-only config",curUser.can_review?"security review":"no review actions",curUser.can_manage_users?"user admin":"no user admin"];
+      w.innerHTML=`<div>signed in as <b>${esc(curUser.user)}</b></div><div>role <b>${esc(curUser.role||"—")}</b></div><div class="muted">${esc(caps.join(" · "))}</div>`;
+    }
     const um=$("usermgmt"); if(um) um.style.display=(curUser&&curUser.can_manage_users)?"":"none";
+    document.body.dataset.role=curUser?curUser.role:"";
+    applyRoleUI();
   }
   const roleOpts=(sel)=>["viewer","reviewer","operator","admin"].map(r=>`<option value="${r}" ${r===sel?"selected":""}>${r}</option>`).join("");
   async function loadUsers(){
@@ -2617,6 +2675,7 @@
   };
 
   // ── signed self-update ──
+  let updateInstallSupported=false;
   async function loadUpdate(){
     try{
       const st=await api("/api/update/status");
@@ -2626,8 +2685,14 @@
         `<div>publisher key <b>${esc(fp)}</b></div>`+
         `<div>catalog <b>${st.catalog_url?esc(st.catalog_url):"upload-only"}</b></div>`+
         `<div>backup <b>${st.has_backup?"available":"none"}</b></div>`;
+      updateInstallSupported=!!st.install_supported;
       $("upd_catalogwrap").style.display=st.catalog_url?"":"none";
+      $("upd_install").disabled=!updateInstallSupported;
+      $("upd_rollback").disabled=!updateInstallSupported;
       $("upd_rollback").style.display=st.has_backup?"":"none";
+      $("upd_mode_hint").innerHTML=updateInstallSupported
+        ? "Install mode: <b>explicit standalone directory</b>. In-process install/rollback is enabled only for this operator-configured writable directory."
+        : "Install mode: <b>external package manager</b>. Verification/staging remains available, but install/rollback is intentionally disabled in the hardened service; deploy the verified DEB/RPM/package through the privileged package manager.";
       if(st.staged){ renderStaged(st.staged, null); } else { $("upd_staged").style.display="none"; }
     }catch(e){
       // 404 = feature disabled (no key / not localhost); anything else = show off state too
@@ -2646,6 +2711,7 @@
     }
     if(summary&&summary.needs_restart) html+=`<div class="hint" style="color:var(--amber);margin-top:6px">Installing requires a restart.</div>`;
     $("upd_staged_body").innerHTML=html;
+    $("upd_install").disabled=!updateInstallSupported;
   }
   $("upd_stage").onclick=async()=>{
     const f=$("upd_file").files[0];
@@ -2832,7 +2898,7 @@
 
   async function startConsole(){
     try{
-      const loaded=await Promise.all([api("/api/config"),api("/api/interfaces").catch(()=>[])]);
+      const loaded=await Promise.all([api("/api/config?draft=1"),api("/api/interfaces").catch(()=>[])]);
       interfaces=loaded[1]||[];
       fill(loaded[0]);
     }catch(e){ toast(e.message,"err"); }
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/tls_frontend_control.go ./tls_frontend_control.go
--- /mnt/data/waf_prod_hardening_base/tls_frontend_control.go	2026-09-25 11:11:25.289854981 +0000
+++ ./tls_frontend_control.go	2026-09-25 12:13:13.011991559 +0000
@@ -62,6 +62,40 @@
 	return err
 }
 
+func (p *tlsFrontendPublisher) publishAndWait(cfg Config, timeout time.Duration) error {
+	if p == nil || !p.enabled() {
+		if tlsfront.FrontendEnabled(cfg.TLSAcceleration) {
+			return errors.New("TLS frontend publisher is not configured")
+		}
+		return nil
+	}
+	started := time.Now().UTC()
+	if err := p.publish(cfg); err != nil {
+		return err
+	}
+	if timeout <= 0 {
+		timeout = 6 * time.Second
+	}
+	wantMode := cfg.TLSAcceleration.Effective().Mode
+	wantActive := tlsfront.FrontendEnabled(cfg.TLSAcceleration)
+	deadline := time.Now().Add(timeout)
+	for {
+		st := p.status()
+		if !st.UpdatedAt.IsZero() && !st.UpdatedAt.Before(started) && st.Mode == wantMode {
+			if st.LastError != "" {
+				return fmt.Errorf("TLS frontend apply failed: %s", st.LastError)
+			}
+			if st.Active == wantActive {
+				return nil
+			}
+		}
+		if time.Now().After(deadline) {
+			return fmt.Errorf("timed out waiting for TLS frontend mode=%s active=%t", wantMode, wantActive)
+		}
+		time.Sleep(100 * time.Millisecond)
+	}
+}
+
 func (p *tlsFrontendPublisher) publish(cfg Config) error {
 	if !p.enabled() {
 		return nil
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/tools/tests/test-production-control-plane-hardening-source.py ./tools/tests/test-production-control-plane-hardening-source.py
--- /mnt/data/waf_prod_hardening_base/tools/tests/test-production-control-plane-hardening-source.py	1970-01-01 00:00:00.000000000 +0000
+++ ./tools/tests/test-production-control-plane-hardening-source.py	2026-09-25 13:30:25.144496935 +0000
@@ -0,0 +1,185 @@
+#!/usr/bin/env python3
+from pathlib import Path
+import re
+ROOT=Path(__file__).resolve().parents[2]
+read=lambda p:(ROOT/p).read_text(encoding='utf-8')
+admin=read('admin.go'); corr=read('correlation.go'); block=read('block_response.go'); ha=read('ha.go')
+main=read('main.go'); listeners=read('listeners.go'); users=read('users.go'); login=read('admin_login_hardening.go')
+notify=read('notify.go'); l7=read('l7_abuse.go'); debug=read('debug_bundle.go'); update=read('update.go'); tlsctl=read('tls_frontend_control.go')
+ui=read('static/admin.html'); svc=read('waf-proxy.service'); tlssvc=read('waf-tls-frontend.service')
+build=read('build.sh'); ci=read('.github/workflows/ci.yml'); tests=read('production_control_plane_hardening_test.go'); persist=read('api_security_persist.go'); identity=read('identity_api5.go'); positive=read('positive_schema_api4.go'); sample=read('config.sample.json'); sitemap=read('sitemap.go'); sitemap_persist=read('sitemap_persist.go')
+checks=[]
+def req(cond,msg):
+    if not cond: raise SystemExit('PRODUCTION_CONTROL_PLANE_HARDENING_SOURCE_GATE_FAIL: '+msg)
+    checks.append(msg)
+
+# Request-path output encoding/correlation.
+for t in ('validRequestCorrelationID','len(id) < 8','len(id) > 128',"c == '-'","c == '_'","c == '.'","c == ':'"):
+    req(t in corr,'request correlation validation contains '+t)
+req('html/template' in block,'block page uses html/template')
+req('{{.Reason}}' in block and '{{.RequestID}}' in block,'block page interpolates only template fields')
+req('Content-Security-Policy' in block and 'X-Frame-Options' in block and 'Cache-Control' in block,'block page emits anti-cache/browser hardening headers')
+
+# JSON/admin secret privacy.
+req('setJSONSecurityHeaders' in admin and 'Cache-Control", "no-store"' in admin and 'X-Content-Type-Options", "nosniff"' in admin,'Admin JSON responses are no-store/nosniff')
+req('redactNotifySecrets' in notify and 'preserveNotifySecrets' in notify,'notification webhook secret has redact/preserve helpers')
+req(admin.count('redactNotifySecrets') >= 3,'config/draft responses redact notification webhook secret')
+req('maskedDebugClientIP(d.RemoteAddr)' in debug and 'maskedDebugClientIP(d.ResolvedClientIP)' in debug,'nested debug identity evidence masks client IPs')
+
+req('DELETE /api/notifications/webhook' in admin and 'handleNotifyWebhookClear' in admin,'stored webhook secret has explicit operator clear route')
+req('notification.webhook_clear' in admin,'webhook clear action is semantically audited')
+req('nt_clear_webhook' in ui and '/api/notifications/webhook' in ui,'Console can explicitly clear stored webhook secret')
+req('"l7_abuse"' in sample and '"cidr_policy"' in sample,'sample config exposes current L7/CIDR controls')
+
+# Authentication abuse resistance.
+for t in ('loginAttemptWindow','loginBlockDuration','loginAttemptLimit','loginAttemptMaxKey','newLoginAttemptLimiter','failure(keys','success(keys'):
+    req(t in login,'login limiter contains '+t)
+req(re.search(r'loginLimiter\s+\*loginAttemptLimiter', admin) is not None and 'newLoginAttemptLimiter()' in admin,'Admin server owns login limiter')
+req('http.StatusTooManyRequests' in admin and 'Retry-After' in admin,'login throttle returns 429 with Retry-After')
+req('pbkdf2MaxIter' in users and 'iter > pbkdf2MaxIter' in users,'PBKDF2 iteration count has upper bound')
+req('len(want) < 16 || len(want) > 64' in users,'PBKDF2 derived-key length bounded')
+req('len(salt) < 8 || len(salt) > 64' in users,'PBKDF2 salt length bounded')
+req('createWithError' in users and 'rand.Read(b)' in users,'session token CSPRNG errors can fail closed')
+req('a.sessions.createWithError(u.Username, u.Role)' in admin,'production login uses error-returning session creation')
+req('overflow loginAttempt' in login and 'stateLocked' in login and 'storeLocked' in login,'login limiter saturation uses bounded shared overflow state')
+req('delete(l.entries, k)' in login and 'len(l.entries) >= loginAttemptMaxKey' in login,'login limiter prunes expired keys without arbitrary saturation eviction')
+
+# Viewer read-only backend authority and mutation audit.
+role_routes={
+'POST /api/reload':'roleOperator','POST /api/ai/unblock':'roleOperator','POST /api/ai/test':'roleOperator',
+'POST /api/learn/clear':'roleReviewer','POST /api/notifications/read':'roleReviewer','POST /api/notifications/dismiss':'roleReviewer',
+'POST /api/crawl':'roleOperator','POST /api/sitemap/clear':'roleReviewer'}
+for route,role in role_routes.items():
+    req(route in admin and role in admin[admin.index(route):admin.index(route)+180],f'{route} requires {role}')
+req('mutationAuditWriter' in login and 'http.mutation' in admin,'successful authenticated mutations receive generic audit coverage')
+for action in ('config.reload','ai.unblock','ai.connector_test','learner.clear','notification.read','notification.dismiss','sitemap.crawl','sitemap.clear'):
+    req(action in admin,'semantic audit retained for '+action)
+
+# Durable bounded local audit.
+for t in ('configurePersistence','adminAuditRotateBytes','os.O_CREATE|os.O_WRONLY|os.O_APPEND','f.Sync()'):
+    req(t in users,'durable audit implementation contains '+t)
+req('WAF_ADMIN_AUDIT_FILE' in admin and 'admin-audit.jsonl' in admin,'Admin audit path has env + local fallback')
+req('WAF_ADMIN_AUDIT_FILE=/var/lib/waf-proxy/admin-audit.jsonl' in svc,'systemd configures durable audit path')
+
+# HA dedicated replication protocol and node-local separation.
+req('X-WAF-Sync' not in ''.join([admin,ha]),'legacy caller-controlled HA sync header removed')
+req('PUT /api/ha/peer-config' in admin and 'haPeerAuth' in admin,'HA config uses dedicated peer-only endpoint')
+req('X-WAF-HA-Sync' in admin and 'X-WAF-HA-Sync' in ha,'dedicated HA protocol marker required/sent')
+req('subtle.ConstantTimeCompare' in admin[admin.index('func (a *adminServer) haPeerAuth'):admin.index('// authRole')],'HA peer token checked constant-time')
+for t in ('sharedConfigForPeer','out.HA = HAConfig{}','out.Users = nil','redactAISecrets','redactHSMSecretRefs','redactNotifySecrets'):
+    req(t in ha,'HA shared payload strips node-local/secret state: '+t)
+for t in ('mergePeerConfig','incoming.HA = local.HA','incoming.Users = local.Users','preserveAISecrets','preserveNotifySecrets','preserveHSMSecretRefs'):
+    req(t in ha,'HA receiver restores local state: '+t)
+req('u.Scheme != "https"' in ha and 'u.User != nil' in ha and 'u.RawQuery != ""' in ha,'HA peer URL restricted to HTTPS origin')
+req('haSyncEnvelopeVersion = 1' in ha and 'DisallowUnknownFields' in admin[admin.index('func (a *adminServer) handleHAPeerConfig'):admin.index('func (a *adminServer) handleHAPeerConfig')+1400],'HA replication envelope versioned and strict')
+req('syncRunning bool' in ha and 'syncPending *haSyncJob' in ha,'HA outbound sync has one in-flight request plus one bounded latest pending slot')
+req('Latest wins' in ha and 'h.syncPending = &pending' in ha and 'runSyncLoop' in ha,'HA sync coalesces concurrent Apply commits instead of dropping the newest config')
+req('haPeerToken       string' in admin and 'haPeerTokenPinned bool' in admin,'HA replication has a credential distinct from the break-glass admin token')
+req('subtle.ConstantTimeCompare([]byte(got), []byte(a.haPeerToken))' in admin,'HA peer endpoint authenticates the dedicated replication token')
+req('if !a.haPeerTokenPinned' in admin[admin.index('func (a *adminServer) haPeerAuth'):admin.index('func (a *adminServer) haPeerAuth')+900],'HA peer endpoint fails closed when dedicated replication token is absent')
+req('c.HA.Enabled && c.HA.SyncConfig && !a.haPeerTokenPinned' in admin,'config draft/apply cannot enable HA sync without local dedicated replication token')
+req('WAF_HA_PEER_TOKEN' in main and 'strings.TrimSpace(*haPeerToken) == ""' in main,'startup refuses persisted HA sync without stable dedicated replication token')
+req('WAF_HA_PEER_TOKEN' in svc,'systemd documentation names the dedicated HA replication token')
+
+# Transactional config persistence/listener ownership.
+for t in ('type stagedConfigWrite','os.CreateTemp','f.Sync()','os.Rename','d.Sync()'):
+    req(t in main,'durable staged config contains '+t)
+req('func (s *server) applyPersisted' in main,'transactional applyPersisted exists')
+req('applyMu             sync.Mutex' in main,'server has a single config-apply serialization mutex')
+req('func (s *server) apply(cfg Config) error' in main and 's.applyMu.Lock()' in main[main.index('func (s *server) apply(cfg Config) error'):main.index('func (s *server) apply(cfg Config) error')+180],'non-persisted Apply is serialized')
+req('func (s *server) applyPersisted' in main and 's.applyMu.Lock()' in main[main.index('func (s *server) applyPersisted'):main.index('func (s *server) applyPersisted')+220],'persisted Apply transaction is serialized')
+req('a.srv.applyMu.Lock()' in admin[admin.index('func (a *adminServer) handlePutConfig'):admin.index('func (a *adminServer) handlePutConfig')+1800],'full config submit preserves secrets/users under the Apply serialization lock')
+req('a.srv.applyPersistedLocked(c, false)' in admin,'full config Apply reuses the lock-assumed transaction without recursive locking')
+req('func cloneConfig(cfg Config) (Config, error)' in main and 'json.Marshal(cfg)' in main and 'json.Unmarshal(b, &out)' in main,'partial mutations deep-clone Config instead of sharing live slice backing arrays')
+req('func (s *server) mutatePersisted' in main and 'applyPersistedLocked(cfg, fromSync)' in main,'partial config mutations derive from the newest runtime inside the serialized transaction')
+req(admin.count('mutatePersisted(') >= 5,'user/webhook/HA/page-policy partial mutations use serialized read-modify-write helper')
+segment=main[main.index('func (s *server) applyPersisted'):main.index('func (s *server) buildRuntime')]
+req(segment.index('stageConfig') < segment.index('s.applyEx') < segment.index('staged.commit'),'config staged before live apply and committed afterward')
+req('s.applyEx(oldCfg, true)' in segment,'commit failure rolls live config back')
+req(segment.index('staged.commit') < segment.index('s.ha.pushConfig'),'HA sync occurs only after durable commit')
+req('func (m *listenerManager) reconcile(cfg Config) error' in listeners,'listener reconcile reports errors')
+for t in ('func (m *listenerManager) prepare','net.Listen','restoreOldLocked','closePrepared','bind listener'):
+    req(t in listeners,'listener transaction contains '+t)
+req('s.listenMgr.reconcile(cfg); err != nil' in main,'runtime apply checks synchronous listener reconciliation')
+req('publishAndWait' in tlsctl and 'timed out waiting for TLS frontend' in tlsctl,'TLS frontend ownership changes wait for manager acknowledgement')
+req('oldFrontend && !newFrontend' in main and 'disable TLS frontend' in main,'frontend-to-Go ownership releases external public port before Go bind')
+req('frontendRollback' in main and 'frontendRestoreAfterListeners' in main,'TLS frontend ownership rollback is ordered around listener restoration')
+req('rt.listeners[tlsfront.InternalListenerKey(addr)]' in listeners,'public listener can bridge to prior internal runtime during ownership handoff')
+req('rt.listeners[logicalAddr]' in listeners,'internal listener can bridge to prior public runtime during ownership handoff')
+req('rt.listeners[tlsfront.InternalListenerKey(addr)]' in main[main.index('func (s *server) getCertificate'):],'TLS certificate lookup bridges external-frontend to Go handoff')
+req('releaseForFrontendOwnership' in listeners and 'tls-frontend-ownership' in listeners,'Go TLS public listener is synchronously released before external frontend ownership')
+req('Shutdown(ctx)' in listeners[listeners.index('func releaseForFrontendOwnership'):listeners.index('func gracefulClose')],'frontend ownership release closes the listener synchronously before returning')
+req('json.NewEncoder(w).Encode' in listeners[listeners.index('func writeHealth'):],'health response uses JSON encoder instead of string concatenation')
+req('draftConfigPath(a.srv.configPath)' in admin and 'saveConfig(draftConfigPath(a.srv.configPath), c)' in admin,'Save draft persists to a separate non-startup config file')
+req('func loadDraftConfig' in main and 'draftInfo.ModTime().After(liveInfo.ModTime())' in main,'Console resumes only drafts newer than authoritative live config')
+req('removeConfigFileDurable(draftConfigPath(a.srv.configPath))' in admin,'successful Apply cleans the auxiliary draft durably')
+req('api("/api/config?draft=1")' in ui,'Console initial load resumes a durable draft without changing startup authority')
+
+# Site-map destructive operations and autosave use serialized durable persistence.
+req('persistMu sync.Mutex' in sitemap,'site-map persistence has a dedicated serialization mutex')
+req('m.persistMu.Lock()' in sitemap_persist and 'defer m.persistMu.Unlock()' in sitemap_persist,'site-map save serializes periodic and operator-triggered writes')
+for t in ('os.CreateTemp(dir, ".sitemap.json.tmp-*")','tmp.Sync()','os.Rename(tmpName, path)','d.Sync()'):
+    req(t in sitemap_persist,'durable site-map persistence contains '+t)
+req('restoreSnapshot(before)' in admin and 'clear not persisted; in-memory map restored' in admin,'destructive sitemap clear restores memory if persistence fails')
+req('before.Crawl.Running' in admin and 'cannot clear site map while crawl is running' in admin,'sitemap clear refuses to race an active crawl')
+
+# Durable API-security version compatibility fails closed on future versions.
+req('func validateAPISecurityStateVersion' in persist and 'version != 0 && version != apiSecurityStateVersion' in persist,'shared API-security state version guard accepts only current/legacy')
+for kind in ('api-operations','api-schema','api-contracts'):
+    req(f'validateAPISecurityStateVersion("{kind}"' in persist,'durable state load checks version: '+kind)
+req('validateAPISecurityStateVersion("api-identity"' in identity,'identity durable state checks version')
+req('validateAPISecurityStateVersion("api-positive-schema"' in positive,'positive-schema durable state checks version')
+
+# Bounded notification delivery.
+for t in ('webhookQ       chan webhookJob','make(chan webhookJob, 256)','webhookWorker','webhookDropped','attempt < 3'):
+    req(t in notify,'bounded webhook delivery contains '+t)
+req('resp.StatusCode >= 200 && resp.StatusCode < 300' in notify,'webhook success requires HTTP 2xx')
+req('webhookDeliveryStats' in notify and 'atomic.LoadUint64(&n.webhookDropped)' in notify,'bounded webhook queue exposes drop/depth telemetry')
+req('notificationDedupeMax = 4096' in notify and 'len(n.dedupe) < notificationDedupeMax' in notify,'notification dedupe memory is explicitly bounded')
+req('webhook_delivery' in admin and 'webhook_delivery||{}' in ui,'notification delivery telemetry is exposed to operators')
+req('http.StatusTooManyRequests' in notify and 'resp.StatusCode >= 500' in notify,'webhook retries 429/5xx only')
+
+# L7 saturation cannot be attacker-controlled fail-open.
+req('overflow abuseClientState' in l7,'L7 shard has bounded overflow bucket')
+req(l7.count('st = &shard.overflow') >= 2,'HTTP and TLS saturation use overflow bucket')
+req('attacker-controlled fail-open' in l7,'source documents no fail-open saturation behavior')
+
+# Filesystem/update/deployment controls.
+req(admin.count('filepath.EvalSymlinks') >= 2,'admin file browser resolves root and requested symlinks')
+req('directUpdateInstallEnabled' in update and 'WAF_UPDATE_INSTALL_DIR' in update,'direct updater requires explicit standalone directory')
+req('filepath.Clean(filepath.Dir(exe)) == filepath.Clean(dir)' in update,'direct updater only enables when install directory is the running binary directory')
+req(update.count('http.StatusConflict') >= 2,'install/rollback fail closed when direct update unsupported')
+for h in ('Forwarded','X-Forwarded-For','X-Real-IP'):
+    req(h in update,'localhost update guard rejects '+h)
+req('DevicePolicy=closed' in svc and 'DeviceAllow=/dev/watchdog rw' in svc,'main systemd service explicitly allows watchdog device under closed policy')
+req('DevicePolicy=closed' in tlssvc and 'DeviceAllow=/dev/qat_adf_ctl rw' in tlssvc and 'DeviceAllow=/dev/qat_dev_processes rw' in tlssvc,'TLS frontend systemd service explicitly allows QAT control devices')
+
+# Console role awareness/operator lifecycle.
+for t in ('function requiredCapability','function hasCapability','Clear learning state','/api/learn/clear'):
+    req(t in ui,'Console capability/lifecycle contains '+t)
+req('can_review' in ui and 'can_edit_config' in ui and 'can_manage_users' in ui,'Console consumes server role capabilities')
+req('path==="/api/notifications/webhook"' in ui and 'path==="/api/security/contracts/import"' in ui,'operator-only webhook clear and contract import are role-gated in Console')
+req('dedicated peer-only HTTPS' in ui and 'webhook URLs are never returned' in ui,'Console explains HA/webhook secret semantics')
+
+req('install_supported' in ui and 'external package manager' in ui and 'updateInstallSupported' in ui,'Console reflects hardened updater install mode instead of implying in-process install')
+req('const roleControlIDs' in ui and 'function applyRoleUI' in ui,'Console proactively disables fixed controls the active role cannot use')
+req('data-required-capability="review"' in ui,'dynamic review controls declare required capability')
+
+# Targeted tests committed.
+for name in (
+'TestProductionCorrelationIDValidation','TestProductionBlockResponseEscapesReasonAndRequestID',
+'TestProductionHASharedConfigStripsNodeLocalAndSecrets','TestProductionHAMergeRestoresReceiverLocalState',
+'TestProductionHARequiresHTTPSOrigin','TestProductionHAPeerAuthUsesDedicatedToken','TestProductionHASyncCoalescesLatestPendingConfig','TestProductionCloneConfigDoesNotShareNestedSlices','TestProductionLoginLimiterBlocksAndResets',
+'TestProductionLoginLimiterSaturationDoesNotEvictBlockedKeys','TestProductionHealthJSONEncodesRole',
+'TestProductionPBKDF2RejectsAbsurdIterationCount','TestProductionDirectUpdateDisabledWithoutExplicitDirectory',
+'TestProductionUpdateLocalhostRejectsForwardedRequests','TestProductionL7SaturationUsesBoundedOverflowBucket',
+'TestProductionStageConfigUsesPrivateAtomicFile','TestProductionAPISecurityStateVersionFailsClosed',
+'TestProductionSitemapPersistenceUsesUniqueDurableTemp','TestProductionSitemapRestoreSnapshotAfterFailedClearPath',
+'TestProductionDraftConfigIsSeparateFromStartupAuthority','TestProductionStaleDraftIgnoredAfterNewerLiveCommit',
+'TestProductionNotificationDedupeStateIsBounded'):
+    req(name in tests,'targeted regression test exists: '+name)
+
+# Build/CI gate wiring.
+req('test-production-control-plane-hardening-source.py' in build,'build.sh runs production hardening gate')
+req('test-production-control-plane-hardening-source.py' in ci,'CI runs production hardening gate')
+print(f'PRODUCTION_CONTROL_PLANE_HARDENING_SOURCE_GATE_PASS checks={len(checks)}')
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/update.go ./update.go
--- /mnt/data/waf_prod_hardening_base/update.go	2026-09-25 11:11:25.291293601 +0000
+++ ./update.go	2026-09-25 12:33:14.030828989 +0000
@@ -1,281 +1,318 @@
-package main
-
-// Signed self-update for waf-proxy, built on the shared sigupdate engine
-// (internal/sigupdate — copied unchanged, never forked).
-//
-// Trust model: the publisher signs update packages with an RSA private key held
-// OFFLINE; this binary embeds only the PUBLIC key and verifies. A stolen admin
-// session cannot forge an update — without the private key, nothing installs.
-// If no publisher key is baked in, the whole subsystem is disabled (fail-safe),
-// and every endpoint 404s.
-//
-// Guards (kept strict, per the standard): admin role + localhost-only + a
-// feature flag. Applying an update is an admin-only action and always requires
-// a restart for a compiled binary; the operator triggers the restart explicitly.
-
-import (
-	"encoding/json"
-	"io"
-	"net"
-	"net/http"
-	"os"
-	"path/filepath"
-	"sync"
-
-	"waf-proxy/internal/sigupdate"
-)
-
-const wafUpdateExtension = ".wafupdate"
-
-// PublisherKeyPEM is the baked-in publisher PUBLIC key (PEM). Empty ⇒ updates
-// disabled. Set this before `go build` to enable signed updates. It can also be
-// supplied at runtime via WAF_PUBLISHER_KEY_FILE (a path to the public-key PEM)
-// so operators can enable updates without a rebuild.
-var PublisherKeyPEM = ``
-
-// CatalogURL is the baked-in online-catalog base (e.g.
-// "https://updates.example.com/waf/"). Empty ⇒ upload-only. Overridable at
-// runtime via WAF_UPDATE_CATALOG_URL.
-var CatalogURL = ``
-
-func loadPublisherKey() string {
-	if PublisherKeyPEM != "" {
-		return PublisherKeyPEM
-	}
-	if p := os.Getenv("WAF_PUBLISHER_KEY_FILE"); p != "" {
-		if b, err := os.ReadFile(p); err == nil {
-			return string(b)
-		}
-	}
-	return ""
-}
-
-func catalogURL() string {
-	if v := os.Getenv("WAF_UPDATE_CATALOG_URL"); v != "" {
-		return v
-	}
-	return CatalogURL
-}
-
-// updateInstallDir is the directory holding the running binary.
-func updateInstallDir() string {
-	exe, err := os.Executable()
-	if err != nil {
-		return "."
-	}
-	return filepath.Dir(exe)
-}
-
-func (a *adminServer) updateConfig() *sigupdate.Config {
-	dir := updateInstallDir()
-	return &sigupdate.Config{
-		PublisherKeyPEM: loadPublisherKey(),
-		CatalogURL:      catalogURL(),
-		InstallDir:      dir,
-		BackupDir:       filepath.Join(dir, ".wafupdate-backup"),
-		// waf-proxy ships as a single binary plus optional rules/config assets.
-		AllowedExts:  map[string]bool{".json": true, ".conf": true, ".html": true},
-		AllowedNames: map[string]bool{"waf-proxy": true}, // the binary itself
-	}
-}
-
-// staged package (single-process state).
-type stagedUpdate struct {
-	mu       sync.Mutex
-	manifest *sigupdate.Manifest
-	payload  map[string][]byte
-}
-
-func (a *adminServer) registerUpdateRoutes(mux *http.ServeMux) {
-	mux.HandleFunc("GET /api/update/status", a.guardedUpdate(a.handleUpdateStatus))
-	mux.HandleFunc("GET /api/update/catalog", a.guardedUpdate(a.handleUpdateCatalog))
-	mux.HandleFunc("POST /api/update/stage", a.guardedUpdate(a.handleUpdateStage))
-	mux.HandleFunc("POST /api/update/download", a.guardedUpdate(a.handleUpdateDownload))
-	mux.HandleFunc("POST /api/update/install", a.guardedUpdate(a.handleUpdateInstall))
-	mux.HandleFunc("POST /api/update/discard", a.guardedUpdate(a.handleUpdateDiscard))
-	mux.HandleFunc("POST /api/update/rollback", a.guardedUpdate(a.handleUpdateRollback))
-	mux.HandleFunc("POST /api/update/restart", a.guardedUpdate(a.handleUpdateRestart))
-}
-
-// guardedUpdate: feature-enabled (404 when no key) + localhost-only + admin.
-// It resolves identity via the same auth path as the rest of the admin API, so
-// the caller must present the master token or an admin session.
-func (a *adminServer) guardedUpdate(h http.HandlerFunc) http.HandlerFunc {
-	inner := a.authRole(roleAdmin, func(w http.ResponseWriter, r *http.Request) {
-		if !updateLocalhost(r) {
-			writeJSONCode(w, http.StatusForbidden, map[string]string{"error": "updates are localhost-only; tunnel with ssh -L"})
-			return
-		}
-		h(w, r)
-	})
-	return func(w http.ResponseWriter, r *http.Request) {
-		// Feature flag: 404 when no publisher key is present at all.
-		if loadPublisherKey() == "" {
-			http.NotFound(w, r)
-			return
-		}
-		inner(w, r)
-	}
-}
-
-func (a *adminServer) handleUpdateStatus(w http.ResponseWriter, _ *http.Request) {
-	cfg := a.updateConfig()
-	fp, _ := sigupdate.KeyFingerprint(cfg.PublisherKeyPEM)
-	a.staged.mu.Lock()
-	var staged any
-	if a.staged.manifest != nil {
-		staged = map[string]any{"version": a.staged.manifest.Version, "notes": a.staged.manifest.Notes}
-	}
-	a.staged.mu.Unlock()
-	writeJSON(w, map[string]any{
-		"enabled":         cfg.Enabled(),
-		"key_fingerprint": fp,
-		"catalog_url":     cfg.CatalogURL,
-		"extension":       wafUpdateExtension,
-		"has_backup":      cfg.HasBackup(),
-		"staged":          staged,
-		"current_version": buildVersion,
-	})
-}
-
-func (a *adminServer) handleUpdateCatalog(w http.ResponseWriter, _ *http.Request) {
-	cfg := a.updateConfig()
-	if cfg.CatalogURL == "" {
-		writeJSON(w, map[string]any{"enabled": false, "reason": "no catalog URL in this build"})
-		return
-	}
-	cat, err := cfg.FetchCatalog()
-	if err != nil {
-		writeJSON(w, map[string]any{"enabled": true, "error": err.Error()})
-		return
-	}
-	writeJSON(w, map[string]any{"enabled": true, "packages": cat.Packages})
-}
-
-func (a *adminServer) handleUpdateStage(w http.ResponseWriter, r *http.Request) {
-	cfg := a.updateConfig()
-	cfg.MaxPackageBytes = 200 << 20
-	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, cfg.MaxPackageBytes+1))
-	if err != nil {
-		writeJSONCode(w, http.StatusBadRequest, map[string]string{"error": "read failed"})
-		return
-	}
-	m, payloads, err := cfg.Inspect(data)
-	if err != nil {
-		a.audit.add(who(r).user, "update.stage_rejected", err.Error())
-		writeJSONCode(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
-		return
-	}
-	a.stageUpdate(m, payloads)
-	a.audit.add(who(r).user, "update.staged", m.Version)
-	writeJSON(w, a.stagedSummary(cfg, m))
-}
-
-func (a *adminServer) handleUpdateDownload(w http.ResponseWriter, r *http.Request) {
-	cfg := a.updateConfig()
-	var body struct {
-		Filename string `json:"filename"`
-	}
-	_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&body)
-	m, payloads, err := cfg.Download(body.Filename, wafUpdateExtension)
-	if err != nil {
-		a.audit.add(who(r).user, "update.download_rejected", err.Error())
-		writeJSONCode(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
-		return
-	}
-	a.stageUpdate(m, payloads)
-	a.audit.add(who(r).user, "update.staged", m.Version)
-	writeJSON(w, a.stagedSummary(cfg, m))
-}
-
-func (a *adminServer) handleUpdateInstall(w http.ResponseWriter, r *http.Request) {
-	cfg := a.updateConfig()
-	a.staged.mu.Lock()
-	m, payloads := a.staged.manifest, a.staged.payload
-	a.staged.mu.Unlock()
-	if m == nil {
-		writeJSONCode(w, http.StatusConflict, map[string]string{"error": "nothing staged"})
-		return
-	}
-	applied, err := cfg.Apply(payloads)
-	if err != nil {
-		a.audit.add(who(r).user, "update.install_failed", err.Error())
-		writeJSONCode(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
-		return
-	}
-	restart := cfg.NeedsRestart(m, applied)
-	a.audit.add(who(r).user, "update.installed", m.Version)
-	a.clearStaged()
-	writeJSON(w, map[string]any{"ok": true, "version": m.Version, "restart_required": restart})
-}
-
-func (a *adminServer) handleUpdateDiscard(w http.ResponseWriter, _ *http.Request) {
-	a.clearStaged()
-	writeJSON(w, map[string]bool{"ok": true})
-}
-
-func (a *adminServer) handleUpdateRollback(w http.ResponseWriter, r *http.Request) {
-	cfg := a.updateConfig()
-	restored, err := cfg.Rollback()
-	if err != nil {
-		writeJSONCode(w, http.StatusConflict, map[string]string{"error": err.Error()})
-		return
-	}
-	a.audit.add(who(r).user, "update.rolledback", "")
-	writeJSON(w, map[string]any{"ok": true, "restored": restored, "restart_required": true})
-}
-
-func (a *adminServer) handleUpdateRestart(w http.ResponseWriter, r *http.Request) {
-	a.audit.add(who(r).user, "update.restart", "")
-	writeJSON(w, map[string]string{"message": "restarting; reconnect shortly"})
-	// Drain first so upstreams fail over, then re-exec into the new binary.
-	go func() {
-		a.srv.draining.Store(true)
-		_ = sigupdate.ReExec()
-	}()
-}
-
-// ── staged-package helpers ──
-
-func (a *adminServer) stageUpdate(m *sigupdate.Manifest, p map[string][]byte) {
-	a.staged.mu.Lock()
-	a.staged.manifest, a.staged.payload = m, p
-	a.staged.mu.Unlock()
-}
-
-func (a *adminServer) clearStaged() {
-	a.staged.mu.Lock()
-	a.staged.manifest, a.staged.payload = nil, nil
-	a.staged.mu.Unlock()
-}
-
-func (a *adminServer) stagedSummary(cfg *sigupdate.Config, m *sigupdate.Manifest) map[string]any {
-	var files []map[string]string
-	for _, f := range m.Files {
-		action := "add"
-		if _, err := os.Stat(filepath.Join(cfg.InstallDir, filepath.FromSlash(f.Path))); err == nil {
-			action = "replace"
-		}
-		files = append(files, map[string]string{"path": f.Path, "action": action})
-	}
-	return map[string]any{
-		"ok": true, "version": m.Version, "notes": m.Notes,
-		"files": files, "needs_restart": cfg.NeedsRestart(m, nil),
-	}
-}
-
-func updateLocalhost(r *http.Request) bool {
-	host, _, err := net.SplitHostPort(r.RemoteAddr)
-	if err != nil {
-		host = r.RemoteAddr
-	}
-	ip := net.ParseIP(host)
-	return ip != nil && ip.IsLoopback()
-}
-
-// writeJSONCode is writeJSON with an explicit status code.
-func writeJSONCode(w http.ResponseWriter, code int, v any) {
-	w.Header().Set("Content-Type", "application/json")
-	w.WriteHeader(code)
-	_ = json.NewEncoder(w).Encode(v)
-}
+package main
+
+// Signed self-update for waf-proxy, built on the shared sigupdate engine
+// (internal/sigupdate — copied unchanged, never forked).
+//
+// Trust model: the publisher signs update packages with an RSA private key held
+// OFFLINE; this binary embeds only the PUBLIC key and verifies. A stolen admin
+// session cannot forge an update — without the private key, nothing installs.
+// If no publisher key is baked in, the whole subsystem is disabled (fail-safe),
+// and every endpoint 404s.
+//
+// Guards (kept strict, per the standard): admin role + localhost-only + a
+// feature flag. Applying an update is an admin-only action and always requires
+// a restart for a compiled binary; the operator triggers the restart explicitly.
+
+import (
+	"encoding/json"
+	"io"
+	"net"
+	"net/http"
+	"os"
+	"path/filepath"
+	"strings"
+	"sync"
+
+	"waf-proxy/internal/sigupdate"
+)
+
+const wafUpdateExtension = ".wafupdate"
+
+// PublisherKeyPEM is the baked-in publisher PUBLIC key (PEM). Empty ⇒ updates
+// disabled. Set this before `go build` to enable signed updates. It can also be
+// supplied at runtime via WAF_PUBLISHER_KEY_FILE (a path to the public-key PEM)
+// so operators can enable updates without a rebuild.
+var PublisherKeyPEM = ``
+
+// CatalogURL is the baked-in online-catalog base (e.g.
+// "https://updates.example.com/waf/"). Empty ⇒ upload-only. Overridable at
+// runtime via WAF_UPDATE_CATALOG_URL.
+var CatalogURL = ``
+
+func loadPublisherKey() string {
+	if PublisherKeyPEM != "" {
+		return PublisherKeyPEM
+	}
+	if p := os.Getenv("WAF_PUBLISHER_KEY_FILE"); p != "" {
+		if b, err := os.ReadFile(p); err == nil {
+			return string(b)
+		}
+	}
+	return ""
+}
+
+func catalogURL() string {
+	if v := os.Getenv("WAF_UPDATE_CATALOG_URL"); v != "" {
+		return v
+	}
+	return CatalogURL
+}
+
+// updateInstallDir is used for package inspection. Direct in-process writes
+// are disabled by default for systemd/DEB/RPM deployments; operators must use
+// the package manager or explicitly opt a standalone/dev install into a
+// dedicated writable directory with WAF_UPDATE_INSTALL_DIR.
+func updateInstallDir() string {
+	if dir := strings.TrimSpace(os.Getenv("WAF_UPDATE_INSTALL_DIR")); dir != "" && filepath.IsAbs(dir) {
+		return filepath.Clean(dir)
+	}
+	exe, err := os.Executable()
+	if err != nil {
+		return "."
+	}
+	return filepath.Dir(exe)
+}
+
+func directUpdateInstallEnabled() bool {
+	dir := strings.TrimSpace(os.Getenv("WAF_UPDATE_INSTALL_DIR"))
+	if dir == "" || !filepath.IsAbs(dir) {
+		return false
+	}
+	exe, err := os.Executable()
+	if err != nil {
+		return false
+	}
+	// ReExec restarts the current executable path. Treat a different writable
+	// directory as unsupported rather than reporting a successful install that
+	// would restart into the old binary.
+	return filepath.Clean(filepath.Dir(exe)) == filepath.Clean(dir)
+}
+
+func (a *adminServer) updateConfig() *sigupdate.Config {
+	dir := updateInstallDir()
+	return &sigupdate.Config{
+		PublisherKeyPEM: loadPublisherKey(),
+		CatalogURL:      catalogURL(),
+		InstallDir:      dir,
+		BackupDir:       filepath.Join(dir, ".wafupdate-backup"),
+		// waf-proxy ships as a single binary plus optional rules/config assets.
+		AllowedExts:  map[string]bool{".json": true, ".conf": true, ".html": true},
+		AllowedNames: map[string]bool{"waf-proxy": true}, // the binary itself
+	}
+}
+
+// staged package (single-process state).
+type stagedUpdate struct {
+	mu       sync.Mutex
+	manifest *sigupdate.Manifest
+	payload  map[string][]byte
+}
+
+func (a *adminServer) registerUpdateRoutes(mux *http.ServeMux) {
+	mux.HandleFunc("GET /api/update/status", a.guardedUpdate(a.handleUpdateStatus))
+	mux.HandleFunc("GET /api/update/catalog", a.guardedUpdate(a.handleUpdateCatalog))
+	mux.HandleFunc("POST /api/update/stage", a.guardedUpdate(a.handleUpdateStage))
+	mux.HandleFunc("POST /api/update/download", a.guardedUpdate(a.handleUpdateDownload))
+	mux.HandleFunc("POST /api/update/install", a.guardedUpdate(a.handleUpdateInstall))
+	mux.HandleFunc("POST /api/update/discard", a.guardedUpdate(a.handleUpdateDiscard))
+	mux.HandleFunc("POST /api/update/rollback", a.guardedUpdate(a.handleUpdateRollback))
+	mux.HandleFunc("POST /api/update/restart", a.guardedUpdate(a.handleUpdateRestart))
+}
+
+// guardedUpdate: feature-enabled (404 when no key) + localhost-only + admin.
+// It resolves identity via the same auth path as the rest of the admin API, so
+// the caller must present the master token or an admin session.
+func (a *adminServer) guardedUpdate(h http.HandlerFunc) http.HandlerFunc {
+	inner := a.authRole(roleAdmin, func(w http.ResponseWriter, r *http.Request) {
+		if !updateLocalhost(r) {
+			writeJSONCode(w, http.StatusForbidden, map[string]string{"error": "updates are localhost-only; tunnel with ssh -L"})
+			return
+		}
+		h(w, r)
+	})
+	return func(w http.ResponseWriter, r *http.Request) {
+		// Feature flag: 404 when no publisher key is present at all.
+		if loadPublisherKey() == "" {
+			http.NotFound(w, r)
+			return
+		}
+		inner(w, r)
+	}
+}
+
+func (a *adminServer) handleUpdateStatus(w http.ResponseWriter, _ *http.Request) {
+	cfg := a.updateConfig()
+	fp, _ := sigupdate.KeyFingerprint(cfg.PublisherKeyPEM)
+	a.staged.mu.Lock()
+	var staged any
+	if a.staged.manifest != nil {
+		staged = map[string]any{"version": a.staged.manifest.Version, "notes": a.staged.manifest.Notes}
+	}
+	a.staged.mu.Unlock()
+	writeJSON(w, map[string]any{
+		"enabled":           cfg.Enabled(),
+		"install_supported": directUpdateInstallEnabled(),
+		"install_mode":      map[bool]string{true: "explicit_standalone_directory", false: "external_package_manager"}[directUpdateInstallEnabled()],
+		"key_fingerprint":   fp,
+		"catalog_url":       cfg.CatalogURL,
+		"extension":         wafUpdateExtension,
+		"has_backup":        cfg.HasBackup(),
+		"staged":            staged,
+		"current_version":   buildVersion,
+	})
+}
+
+func (a *adminServer) handleUpdateCatalog(w http.ResponseWriter, _ *http.Request) {
+	cfg := a.updateConfig()
+	if cfg.CatalogURL == "" {
+		writeJSON(w, map[string]any{"enabled": false, "reason": "no catalog URL in this build"})
+		return
+	}
+	cat, err := cfg.FetchCatalog()
+	if err != nil {
+		writeJSON(w, map[string]any{"enabled": true, "error": err.Error()})
+		return
+	}
+	writeJSON(w, map[string]any{"enabled": true, "packages": cat.Packages})
+}
+
+func (a *adminServer) handleUpdateStage(w http.ResponseWriter, r *http.Request) {
+	cfg := a.updateConfig()
+	cfg.MaxPackageBytes = 200 << 20
+	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, cfg.MaxPackageBytes+1))
+	if err != nil {
+		writeJSONCode(w, http.StatusBadRequest, map[string]string{"error": "read failed"})
+		return
+	}
+	m, payloads, err := cfg.Inspect(data)
+	if err != nil {
+		a.audit.add(who(r).user, "update.stage_rejected", err.Error())
+		writeJSONCode(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
+		return
+	}
+	a.stageUpdate(m, payloads)
+	a.audit.add(who(r).user, "update.staged", m.Version)
+	writeJSON(w, a.stagedSummary(cfg, m))
+}
+
+func (a *adminServer) handleUpdateDownload(w http.ResponseWriter, r *http.Request) {
+	cfg := a.updateConfig()
+	var body struct {
+		Filename string `json:"filename"`
+	}
+	_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&body)
+	m, payloads, err := cfg.Download(body.Filename, wafUpdateExtension)
+	if err != nil {
+		a.audit.add(who(r).user, "update.download_rejected", err.Error())
+		writeJSONCode(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
+		return
+	}
+	a.stageUpdate(m, payloads)
+	a.audit.add(who(r).user, "update.staged", m.Version)
+	writeJSON(w, a.stagedSummary(cfg, m))
+}
+
+func (a *adminServer) handleUpdateInstall(w http.ResponseWriter, r *http.Request) {
+	if !directUpdateInstallEnabled() {
+		writeJSONCode(w, http.StatusConflict, map[string]string{"error": "direct in-process install is disabled for packaged/systemd deployments; use the OS package manager or set WAF_UPDATE_INSTALL_DIR for a standalone writable install"})
+		return
+	}
+	cfg := a.updateConfig()
+	a.staged.mu.Lock()
+	m, payloads := a.staged.manifest, a.staged.payload
+	a.staged.mu.Unlock()
+	if m == nil {
+		writeJSONCode(w, http.StatusConflict, map[string]string{"error": "nothing staged"})
+		return
+	}
+	applied, err := cfg.Apply(payloads)
+	if err != nil {
+		a.audit.add(who(r).user, "update.install_failed", err.Error())
+		writeJSONCode(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
+		return
+	}
+	restart := cfg.NeedsRestart(m, applied)
+	a.audit.add(who(r).user, "update.installed", m.Version)
+	a.clearStaged()
+	writeJSON(w, map[string]any{"ok": true, "version": m.Version, "restart_required": restart})
+}
+
+func (a *adminServer) handleUpdateDiscard(w http.ResponseWriter, _ *http.Request) {
+	a.clearStaged()
+	writeJSON(w, map[string]bool{"ok": true})
+}
+
+func (a *adminServer) handleUpdateRollback(w http.ResponseWriter, r *http.Request) {
+	if !directUpdateInstallEnabled() {
+		writeJSONCode(w, http.StatusConflict, map[string]string{"error": "direct rollback is disabled for packaged/systemd deployments; use the OS package manager"})
+		return
+	}
+	cfg := a.updateConfig()
+	restored, err := cfg.Rollback()
+	if err != nil {
+		writeJSONCode(w, http.StatusConflict, map[string]string{"error": err.Error()})
+		return
+	}
+	a.audit.add(who(r).user, "update.rolledback", "")
+	writeJSON(w, map[string]any{"ok": true, "restored": restored, "restart_required": true})
+}
+
+func (a *adminServer) handleUpdateRestart(w http.ResponseWriter, r *http.Request) {
+	a.audit.add(who(r).user, "update.restart", "")
+	writeJSON(w, map[string]string{"message": "restarting; reconnect shortly"})
+	// Drain first so upstreams fail over, then re-exec into the new binary.
+	go func() {
+		a.srv.draining.Store(true)
+		_ = sigupdate.ReExec()
+	}()
+}
+
+// ── staged-package helpers ──
+
+func (a *adminServer) stageUpdate(m *sigupdate.Manifest, p map[string][]byte) {
+	a.staged.mu.Lock()
+	a.staged.manifest, a.staged.payload = m, p
+	a.staged.mu.Unlock()
+}
+
+func (a *adminServer) clearStaged() {
+	a.staged.mu.Lock()
+	a.staged.manifest, a.staged.payload = nil, nil
+	a.staged.mu.Unlock()
+}
+
+func (a *adminServer) stagedSummary(cfg *sigupdate.Config, m *sigupdate.Manifest) map[string]any {
+	var files []map[string]string
+	for _, f := range m.Files {
+		action := "add"
+		if _, err := os.Stat(filepath.Join(cfg.InstallDir, filepath.FromSlash(f.Path))); err == nil {
+			action = "replace"
+		}
+		files = append(files, map[string]string{"path": f.Path, "action": action})
+	}
+	return map[string]any{
+		"ok": true, "version": m.Version, "notes": m.Notes,
+		"files": files, "needs_restart": cfg.NeedsRestart(m, nil),
+	}
+}
+
+func updateLocalhost(r *http.Request) bool {
+	// A local reverse proxy makes RemoteAddr loopback even for remote callers.
+	// Refuse forwarded requests so localhost-only cannot be silently weakened.
+	if r.Header.Get("Forwarded") != "" || r.Header.Get("X-Forwarded-For") != "" || r.Header.Get("X-Real-IP") != "" {
+		return false
+	}
+	host, _, err := net.SplitHostPort(r.RemoteAddr)
+	if err != nil {
+		host = r.RemoteAddr
+	}
+	ip := net.ParseIP(host)
+	return ip != nil && ip.IsLoopback()
+}
+
+// writeJSONCode is writeJSON with an explicit status code.
+func writeJSONCode(w http.ResponseWriter, code int, v any) {
+	setJSONSecurityHeaders(w)
+	w.WriteHeader(code)
+	_ = json.NewEncoder(w).Encode(v)
+}
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/users.go ./users.go
--- /mnt/data/waf_prod_hardening_base/users.go	2026-09-25 11:11:25.291403516 +0000
+++ ./users.go	2026-09-25 11:24:23.644820251 +0000
@@ -1,260 +1,351 @@
-package main
-
-// Multi-user auth + role-based access + audit trail.
-//
-// Modeled on the mini-SIEM's RBAC idea, with one deliberate upgrade: passwords
-// are stored as PBKDF2-HMAC-SHA256 (iterated, salted) rather than a single
-// salted SHA-256 — the correct version of that idea, brute-force resistant,
-// and stdlib-only.
-//
-// The startup admin token remains a break-glass credential (always full admin)
-// so you can never lock yourself out. Named users are layered on top for
-// day-to-day access and audit attribution. Sessions are in-memory (re-login
-// after a restart); a persistence seam exists for later.
-
-import (
-	"crypto/hmac"
-	"crypto/rand"
-	"crypto/sha256"
-	"crypto/subtle"
-	"encoding/base64"
-	"encoding/binary"
-	"fmt"
-	"strconv"
-	"strings"
-	"sync"
-	"time"
-)
-
-// ── roles ───────────────────────────────────────────────────────────────
-
-// Role hierarchy: admin > operator > viewer. A higher role satisfies any lower
-// requirement. reviewer sits beside operator: it may apply suggestions/page
-// policies but not edit global config or manage users.
-const (
-	roleAdmin    = "admin"
-	roleOperator = "operator"
-	roleReviewer = "reviewer"
-	roleViewer   = "viewer"
-)
-
-func roleRank(r string) int {
-	switch r {
-	case roleAdmin:
-		return 4
-	case roleOperator:
-		return 3
-	case roleReviewer:
-		return 2
-	case roleViewer:
-		return 1
-	}
-	return 0
-}
-
-func validRole(r string) bool { return roleRank(r) > 0 }
-
-// capabilities each identity is allowed. Kept explicit so the UI and server
-// agree. "review" = apply page policies / profiles / learned suggestions.
-func canManageUsers(role string) bool { return role == roleAdmin }
-func canEditConfig(role string) bool  { return roleRank(role) >= roleRank(roleOperator) }
-func canReview(role string) bool      { return role == roleReviewer || roleRank(role) >= roleRank(roleOperator) }
-
-// ── user model ──────────────────────────────────────────────────────────
-
-type UserConfig struct {
-	Username     string `json:"username"`
-	PasswordHash string `json:"password_hash"` // pbkdf2$sha256$iter$salt$hash ; masked on read
-	Role         string `json:"role"`
-	Disabled     bool   `json:"disabled,omitempty"`
-}
-
-func (c Config) validateUsers() error {
-	seen := map[string]bool{}
-	for i, u := range c.Users {
-		if strings.TrimSpace(u.Username) == "" {
-			return fmt.Errorf("user %d: username required", i+1)
-		}
-		if seen[strings.ToLower(u.Username)] {
-			return fmt.Errorf("duplicate user %q", u.Username)
-		}
-		seen[strings.ToLower(u.Username)] = true
-		if !validRole(u.Role) {
-			return fmt.Errorf("user %q: role must be admin, operator, reviewer, or viewer", u.Username)
-		}
-		if u.PasswordHash == "" {
-			return fmt.Errorf("user %q: password not set", u.Username)
-		}
-	}
-	return nil
-}
-
-// ── password hashing (PBKDF2-HMAC-SHA256, stdlib only) ───────────────────
-
-const pbkdf2Iter = 210000
-
-func pbkdf2SHA256(password, salt []byte, iter, keyLen int) []byte {
-	prf := func(block int) []byte {
-		mac := hmac.New(sha256.New, password)
-		mac.Write(salt)
-		var b [4]byte
-		binary.BigEndian.PutUint32(b[:], uint32(block))
-		mac.Write(b[:])
-		u := mac.Sum(nil)
-		out := make([]byte, len(u))
-		copy(out, u)
-		for i := 1; i < iter; i++ {
-			mac = hmac.New(sha256.New, password)
-			mac.Write(u)
-			u = mac.Sum(nil)
-			for j := range out {
-				out[j] ^= u[j]
-			}
-		}
-		return out
-	}
-	var dk []byte
-	blocks := (keyLen + sha256.Size - 1) / sha256.Size
-	for i := 1; i <= blocks; i++ {
-		dk = append(dk, prf(i)...)
-	}
-	return dk[:keyLen]
-}
-
-func hashPassword(pw string) (string, error) {
-	if len(pw) < 8 {
-		return "", fmt.Errorf("password must be at least 8 characters")
-	}
-	salt := make([]byte, 16)
-	if _, err := rand.Read(salt); err != nil {
-		return "", err
-	}
-	dk := pbkdf2SHA256([]byte(pw), salt, pbkdf2Iter, 32)
-	return fmt.Sprintf("pbkdf2$sha256$%d$%s$%s", pbkdf2Iter,
-		base64.RawStdEncoding.EncodeToString(salt),
-		base64.RawStdEncoding.EncodeToString(dk)), nil
-}
-
-func verifyPassword(pw, encoded string) bool {
-	parts := strings.Split(encoded, "$")
-	if len(parts) != 5 || parts[0] != "pbkdf2" || parts[1] != "sha256" {
-		return false
-	}
-	iter, err := strconv.Atoi(parts[2])
-	if err != nil || iter < 1 {
-		return false
-	}
-	salt, err1 := base64.RawStdEncoding.DecodeString(parts[3])
-	want, err2 := base64.RawStdEncoding.DecodeString(parts[4])
-	if err1 != nil || err2 != nil {
-		return false
-	}
-	got := pbkdf2SHA256([]byte(pw), salt, iter, len(want))
-	return subtle.ConstantTimeCompare(got, want) == 1
-}
-
-// ── sessions (in-memory) ────────────────────────────────────────────────
-
-type session struct {
-	user    string
-	role    string
-	expires time.Time
-}
-
-type sessionStore struct {
-	mu   sync.Mutex
-	byID map[string]session
-	ttl  time.Duration
-}
-
-func newSessionStore() *sessionStore {
-	return &sessionStore{byID: map[string]session{}, ttl: 12 * time.Hour}
-}
-
-func (s *sessionStore) create(user, role string) string {
-	b := make([]byte, 24)
-	_, _ = rand.Read(b)
-	id := base64.RawURLEncoding.EncodeToString(b)
-	s.mu.Lock()
-	s.byID[id] = session{user: user, role: role, expires: time.Now().Add(s.ttl)}
-	s.mu.Unlock()
-	return id
-}
-
-func (s *sessionStore) lookup(id string) (session, bool) {
-	s.mu.Lock()
-	defer s.mu.Unlock()
-	sess, ok := s.byID[id]
-	if !ok {
-		return session{}, false
-	}
-	if time.Now().After(sess.expires) {
-		delete(s.byID, id)
-		return session{}, false
-	}
-	return sess, true
-}
-
-func (s *sessionStore) destroy(id string) {
-	s.mu.Lock()
-	delete(s.byID, id)
-	s.mu.Unlock()
-}
-
-// revoke drops all sessions for a user (on disable/delete/role change).
-func (s *sessionStore) revoke(user string) {
-	s.mu.Lock()
-	defer s.mu.Unlock()
-	for id, sess := range s.byID {
-		if strings.EqualFold(sess.user, user) {
-			delete(s.byID, id)
-		}
-	}
-}
-
-// ── audit trail (in-memory ring) ─────────────────────────────────────────
-
-type auditEntry struct {
-	Time   string `json:"time"`
-	User   string `json:"user"`
-	Action string `json:"action"`
-	Detail string `json:"detail"`
-}
-
-type auditLog struct {
-	mu   sync.Mutex
-	recs []auditEntry
-	cap  int
-	sink func(user, action, detail string)
-}
-
-func newAuditLog() *auditLog { return &auditLog{cap: 500} }
-
-func (a *auditLog) add(user, action, detail string) {
-	a.mu.Lock()
-	a.recs = append(a.recs, auditEntry{
-		Time: time.Now().Format("2006-01-02 15:04:05"), User: user, Action: action, Detail: detail,
-	})
-	if len(a.recs) > a.cap {
-		a.recs = a.recs[len(a.recs)-a.cap:]
-	}
-	sink := a.sink
-	a.mu.Unlock()
-	if sink != nil {
-		sink(user, action, detail)
-	}
-}
-
-func (a *auditLog) list(limit int) []auditEntry {
-	a.mu.Lock()
-	defer a.mu.Unlock()
-	n := len(a.recs)
-	if limit <= 0 || limit > n {
-		limit = n
-	}
-	out := make([]auditEntry, limit)
-	for i := 0; i < limit; i++ {
-		out[i] = a.recs[n-1-i]
-	}
-	return out
-}
+package main
+
+// Multi-user auth + role-based access + audit trail.
+//
+// Modeled on the mini-SIEM's RBAC idea, with one deliberate upgrade: passwords
+// are stored as PBKDF2-HMAC-SHA256 (iterated, salted) rather than a single
+// salted SHA-256 — the correct version of that idea, brute-force resistant,
+// and stdlib-only.
+//
+// The startup admin token remains a break-glass credential (always full admin)
+// so you can never lock yourself out. Named users are layered on top for
+// day-to-day access and audit attribution. Sessions are in-memory (re-login
+// after a restart); a persistence seam exists for later.
+
+import (
+	"bufio"
+	"crypto/hmac"
+	"crypto/rand"
+	"crypto/sha256"
+	"crypto/subtle"
+	"encoding/base64"
+	"encoding/binary"
+	"encoding/json"
+	"fmt"
+	"log/slog"
+	"os"
+	"path/filepath"
+	"strconv"
+	"strings"
+	"sync"
+	"time"
+)
+
+// ── roles ───────────────────────────────────────────────────────────────
+
+// Role hierarchy: admin > operator > viewer. A higher role satisfies any lower
+// requirement. reviewer sits beside operator: it may apply suggestions/page
+// policies but not edit global config or manage users.
+const (
+	roleAdmin    = "admin"
+	roleOperator = "operator"
+	roleReviewer = "reviewer"
+	roleViewer   = "viewer"
+)
+
+func roleRank(r string) int {
+	switch r {
+	case roleAdmin:
+		return 4
+	case roleOperator:
+		return 3
+	case roleReviewer:
+		return 2
+	case roleViewer:
+		return 1
+	}
+	return 0
+}
+
+func validRole(r string) bool { return roleRank(r) > 0 }
+
+// capabilities each identity is allowed. Kept explicit so the UI and server
+// agree. "review" = apply page policies / profiles / learned suggestions.
+func canManageUsers(role string) bool { return role == roleAdmin }
+func canEditConfig(role string) bool  { return roleRank(role) >= roleRank(roleOperator) }
+func canReview(role string) bool {
+	return role == roleReviewer || roleRank(role) >= roleRank(roleOperator)
+}
+
+// ── user model ──────────────────────────────────────────────────────────
+
+type UserConfig struct {
+	Username     string `json:"username"`
+	PasswordHash string `json:"password_hash"` // pbkdf2$sha256$iter$salt$hash ; masked on read
+	Role         string `json:"role"`
+	Disabled     bool   `json:"disabled,omitempty"`
+}
+
+func (c Config) validateUsers() error {
+	seen := map[string]bool{}
+	for i, u := range c.Users {
+		if strings.TrimSpace(u.Username) == "" {
+			return fmt.Errorf("user %d: username required", i+1)
+		}
+		if seen[strings.ToLower(u.Username)] {
+			return fmt.Errorf("duplicate user %q", u.Username)
+		}
+		seen[strings.ToLower(u.Username)] = true
+		if !validRole(u.Role) {
+			return fmt.Errorf("user %q: role must be admin, operator, reviewer, or viewer", u.Username)
+		}
+		if u.PasswordHash == "" {
+			return fmt.Errorf("user %q: password not set", u.Username)
+		}
+	}
+	return nil
+}
+
+// ── password hashing (PBKDF2-HMAC-SHA256, stdlib only) ───────────────────
+
+const (
+	pbkdf2Iter    = 210000
+	pbkdf2MaxIter = 1000000
+)
+
+func pbkdf2SHA256(password, salt []byte, iter, keyLen int) []byte {
+	prf := func(block int) []byte {
+		mac := hmac.New(sha256.New, password)
+		mac.Write(salt)
+		var b [4]byte
+		binary.BigEndian.PutUint32(b[:], uint32(block))
+		mac.Write(b[:])
+		u := mac.Sum(nil)
+		out := make([]byte, len(u))
+		copy(out, u)
+		for i := 1; i < iter; i++ {
+			mac = hmac.New(sha256.New, password)
+			mac.Write(u)
+			u = mac.Sum(nil)
+			for j := range out {
+				out[j] ^= u[j]
+			}
+		}
+		return out
+	}
+	var dk []byte
+	blocks := (keyLen + sha256.Size - 1) / sha256.Size
+	for i := 1; i <= blocks; i++ {
+		dk = append(dk, prf(i)...)
+	}
+	return dk[:keyLen]
+}
+
+func hashPassword(pw string) (string, error) {
+	if len(pw) < 8 {
+		return "", fmt.Errorf("password must be at least 8 characters")
+	}
+	salt := make([]byte, 16)
+	if _, err := rand.Read(salt); err != nil {
+		return "", err
+	}
+	dk := pbkdf2SHA256([]byte(pw), salt, pbkdf2Iter, 32)
+	return fmt.Sprintf("pbkdf2$sha256$%d$%s$%s", pbkdf2Iter,
+		base64.RawStdEncoding.EncodeToString(salt),
+		base64.RawStdEncoding.EncodeToString(dk)), nil
+}
+
+func verifyPassword(pw, encoded string) bool {
+	parts := strings.Split(encoded, "$")
+	if len(parts) != 5 || parts[0] != "pbkdf2" || parts[1] != "sha256" {
+		return false
+	}
+	iter, err := strconv.Atoi(parts[2])
+	if err != nil || iter < 1 || iter > pbkdf2MaxIter {
+		return false
+	}
+	salt, err1 := base64.RawStdEncoding.DecodeString(parts[3])
+	want, err2 := base64.RawStdEncoding.DecodeString(parts[4])
+	if err1 != nil || err2 != nil || len(salt) < 8 || len(salt) > 64 || len(want) < 16 || len(want) > 64 {
+		return false
+	}
+	got := pbkdf2SHA256([]byte(pw), salt, iter, len(want))
+	return subtle.ConstantTimeCompare(got, want) == 1
+}
+
+// ── sessions (in-memory) ────────────────────────────────────────────────
+
+type session struct {
+	user    string
+	role    string
+	expires time.Time
+}
+
+type sessionStore struct {
+	mu   sync.Mutex
+	byID map[string]session
+	ttl  time.Duration
+}
+
+func newSessionStore() *sessionStore {
+	return &sessionStore{byID: map[string]session{}, ttl: 12 * time.Hour}
+}
+
+func (s *sessionStore) createWithError(user, role string) (string, error) {
+	b := make([]byte, 24)
+	if _, err := rand.Read(b); err != nil {
+		return "", fmt.Errorf("generate session token: %w", err)
+	}
+	id := base64.RawURLEncoding.EncodeToString(b)
+	s.mu.Lock()
+	s.byID[id] = session{user: user, role: role, expires: time.Now().Add(s.ttl)}
+	s.mu.Unlock()
+	return id, nil
+}
+
+// create is retained for test helpers. Production login uses createWithError
+// so a CSPRNG failure can never create an empty/predictable authenticated session.
+func (s *sessionStore) create(user, role string) string {
+	id, _ := s.createWithError(user, role)
+	return id
+}
+
+func (s *sessionStore) lookup(id string) (session, bool) {
+	s.mu.Lock()
+	defer s.mu.Unlock()
+	sess, ok := s.byID[id]
+	if !ok {
+		return session{}, false
+	}
+	if time.Now().After(sess.expires) {
+		delete(s.byID, id)
+		return session{}, false
+	}
+	return sess, true
+}
+
+func (s *sessionStore) destroy(id string) {
+	s.mu.Lock()
+	delete(s.byID, id)
+	s.mu.Unlock()
+}
+
+// revoke drops all sessions for a user (on disable/delete/role change).
+func (s *sessionStore) revoke(user string) {
+	s.mu.Lock()
+	defer s.mu.Unlock()
+	for id, sess := range s.byID {
+		if strings.EqualFold(sess.user, user) {
+			delete(s.byID, id)
+		}
+	}
+}
+
+// ── audit trail (in-memory ring) ─────────────────────────────────────────
+
+type auditEntry struct {
+	Time   string `json:"time"`
+	User   string `json:"user"`
+	Action string `json:"action"`
+	Detail string `json:"detail"`
+}
+
+type auditLog struct {
+	mu          sync.Mutex
+	recs        []auditEntry
+	cap         int
+	sink        func(user, action, detail string)
+	persistMu   sync.Mutex
+	persistPath string
+	persistLog  *slog.Logger
+}
+
+func newAuditLog() *auditLog { return &auditLog{cap: 500} }
+
+const adminAuditRotateBytes int64 = 4 << 20
+
+func (a *auditLog) configurePersistence(path string, log *slog.Logger) error {
+	path = filepath.Clean(path)
+	if !filepath.IsAbs(path) {
+		return fmt.Errorf("audit path must be absolute")
+	}
+	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
+		return err
+	}
+	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
+	if err != nil {
+		return err
+	}
+	defer f.Close()
+	_ = f.Chmod(0o600)
+	scanner := bufio.NewScanner(f)
+	scanner.Buffer(make([]byte, 4096), 1<<20)
+	loaded := make([]auditEntry, 0, a.cap)
+	for scanner.Scan() {
+		var e auditEntry
+		if json.Unmarshal(scanner.Bytes(), &e) == nil {
+			loaded = append(loaded, e)
+			if len(loaded) > a.cap {
+				loaded = loaded[len(loaded)-a.cap:]
+			}
+		}
+	}
+	if err := scanner.Err(); err != nil {
+		return err
+	}
+	a.mu.Lock()
+	a.recs = loaded
+	a.mu.Unlock()
+	a.persistPath = path
+	a.persistLog = log
+	return nil
+}
+
+func (a *auditLog) persist(entry auditEntry) {
+	if a == nil || a.persistPath == "" {
+		return
+	}
+	a.persistMu.Lock()
+	defer a.persistMu.Unlock()
+	if st, err := os.Stat(a.persistPath); err == nil && st.Size() >= adminAuditRotateBytes {
+		_ = os.Remove(a.persistPath + ".1")
+		if err := os.Rename(a.persistPath, a.persistPath+".1"); err != nil && a.persistLog != nil {
+			a.persistLog.Error("rotate admin audit failed", "err", err)
+		}
+	}
+	f, err := os.OpenFile(a.persistPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
+	if err != nil {
+		if a.persistLog != nil {
+			a.persistLog.Error("persist admin audit failed", "err", err)
+		}
+		return
+	}
+	_ = f.Chmod(0o600)
+	err = json.NewEncoder(f).Encode(entry)
+	if err == nil {
+		err = f.Sync()
+	}
+	_ = f.Close()
+	if err != nil && a.persistLog != nil {
+		a.persistLog.Error("persist admin audit failed", "err", err)
+	}
+}
+
+func (a *auditLog) add(user, action, detail string) {
+	entry := auditEntry{Time: time.Now().Format("2006-01-02 15:04:05"), User: user, Action: action, Detail: detail}
+	a.mu.Lock()
+	a.recs = append(a.recs, entry)
+	if len(a.recs) > a.cap {
+		a.recs = a.recs[len(a.recs)-a.cap:]
+	}
+	sink := a.sink
+	a.mu.Unlock()
+	if sink != nil {
+		sink(user, action, detail)
+	}
+	a.persist(entry)
+}
+
+func (a *auditLog) list(limit int) []auditEntry {
+	a.mu.Lock()
+	defer a.mu.Unlock()
+	n := len(a.recs)
+	if limit <= 0 || limit > n {
+		limit = n
+	}
+	out := make([]auditEntry, limit)
+	for i := 0; i < limit; i++ {
+		out[i] = a.recs[n-1-i]
+	}
+	return out
+}
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/waf-proxy.service ./waf-proxy.service
--- /mnt/data/waf_prod_hardening_base/waf-proxy.service	2026-09-25 11:11:25.291883771 +0000
+++ ./waf-proxy.service	2026-09-25 12:34:36.547357149 +0000
@@ -15,10 +15,13 @@
 # Admin token: REQUIRED for a stable token across restarts.
 # Generate: openssl rand -hex 24
 # Prefer an EnvironmentFile (0600, root-owned) over inlining the secret here.
+# HA config replication uses a separate WAF_HA_PEER_TOKEN; do not reuse the break-glass admin token.
+# Generate both independently (for example: openssl rand -hex 24).
 EnvironmentFile=-/etc/waf/waf-proxy.env
 EnvironmentFile=-/etc/waf/waf-tls-frontend.env
 Environment=WAF_TLS_FRONTEND_CONTROL=/run/waf-proxy/tls-frontend-live.json
 Environment=WAF_TLS_FRONTEND_STATUS=/run/waf-tls-frontend/status.json
+Environment=WAF_ADMIN_AUDIT_FILE=/var/lib/waf-proxy/admin-audit.jsonl
 
 # Admin console bind address. Default is loopback-only (reach it via ssh -L).
 # To expose it to a management network, set an IP or 0.0.0.0 here AND set TLS
@@ -49,7 +52,18 @@
 # ── hardening ──
 NoNewPrivileges=true
 PrivateTmp=true
-PrivateDevices=true
+# Host devices remain hidden by DevicePolicy except the minimal runtime set and
+# optional watchdog nodes. This makes WAF_WATCHDOG_DEVICE usable without
+# exposing arbitrary accelerator/storage devices to the process.
+PrivateDevices=false
+DevicePolicy=closed
+DeviceAllow=/dev/null rw
+DeviceAllow=/dev/zero rw
+DeviceAllow=/dev/full rw
+DeviceAllow=/dev/random r
+DeviceAllow=/dev/urandom r
+DeviceAllow=/dev/watchdog rw
+DeviceAllow=/dev/watchdog0 rw
 ProtectSystem=strict
 ProtectHome=true
 ProtectKernelTunables=true
diff -ruN '--exclude=DELIVERY_MANIFEST.json' '--exclude=patch.md' '--exclude=RELEASE_MANIFEST.txt' '--exclude=SHA256SUMS.txt' '--exclude=release-evidence' '--exclude=__pycache__' /mnt/data/waf_prod_hardening_base/waf-tls-frontend.service ./waf-tls-frontend.service
--- /mnt/data/waf_prod_hardening_base/waf-tls-frontend.service	2026-09-25 11:11:25.291920786 +0000
+++ ./waf-tls-frontend.service	2026-09-25 11:26:22.624588666 +0000
@@ -25,6 +25,18 @@
 # still constrained by ordinary Unix device permissions and the service user.
 NoNewPrivileges=true
 PrivateTmp=true
+# QAT/OpenSSL acceleration requires host device nodes. Keep /dev visible but
+# close device access at the cgroup boundary and allow only the minimum common
+# runtime/QAT control devices.
+PrivateDevices=false
+DevicePolicy=closed
+DeviceAllow=/dev/null rw
+DeviceAllow=/dev/zero rw
+DeviceAllow=/dev/full rw
+DeviceAllow=/dev/random r
+DeviceAllow=/dev/urandom r
+DeviceAllow=/dev/qat_adf_ctl rw
+DeviceAllow=/dev/qat_dev_processes rw
 ProtectSystem=strict
 ProtectHome=true
 ProtectKernelTunables=true
```
