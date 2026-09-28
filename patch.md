# Code Duplication Consolidation Patch

This slice-local patch records source/document differences from the canonical parent `waf-proxy-production-control-plane-hardening-2026-09-25.zip` (SHA-256 `90acc977db2998f4a3c1c4cacc07e19f0087f8bc033bce6b9934b83415957be8`) to the 2026-09-28 Code Duplication Review and Consolidation baseline. Generated release evidence, delivery manifests, and this patch file itself are intentionally excluded from the diff to avoid self-reference.

diff --git a/.github/workflows/ci.yml b/.github/workflows/ci.yml
index 2eb0c81..3db9788 100644
--- a/.github/workflows/ci.yml
+++ b/.github/workflows/ci.yml
@@ -58,6 +58,10 @@ jobs:
         run: python3 ./tools/tests/test-api8-post-audit-hardening-source.py
       - name: Production correctness and control-plane hardening source gate
         run: python3 ./tools/tests/test-production-control-plane-hardening-source.py
+      - name: Code duplication review source gate
+        run: python3 ./tools/tests/test-code-duplication-review-source.py
+      - name: Markdown review source gate
+        run: python3 ./tools/tests/test-markdown-review-source.py
 
       # Fail before compilation if committed module metadata is not canonical.
       # -diff never mutates go.mod/go.sum and exits non-zero when tidy would.
diff --git a/AGENTS.md b/AGENTS.md
index e86b386..7f5dcc9 100644
--- a/AGENTS.md
+++ b/AGENTS.md
@@ -61,3 +61,5 @@ Rules:
   synchronized after meaningful release/build/packaging changes.
 - Never use stale "next slice" text from a historical section when a newer
   canonical handover or roadmap entry exists.
+
+<!-- documentation-review: 2026-09-28; classification: repository policy; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/AI_HANDOFF.md b/AI_HANDOFF.md
index b86de25..f3d8d6f 100644
--- a/AI_HANDOFF.md
+++ b/AI_HANDOFF.md
@@ -1,5 +1,34 @@
 # AI Development Handoff
 
+## Current canonical baseline — 2026-09-28
+
+The current source is the **Code Duplication Review and Consolidation** working
+baseline derived byte-for-byte from the Production Correctness & Control-Plane
+Hardening parent artifact (`90acc977db2998f4a3c1c4cacc07e19f0087f8bc033bce6b9934b83415957be8`)
+before the changes documented in `CODE_DUPLICATION_REVIEW.md`. Status remains
+**IMPLEMENTED_TESTING_DEFERRED**. No API-9 is defined.
+
+The review removed only source layers proven to be unwired, superseded or
+functionally duplicative, and consolidated the duplicated SecLang action/token
+parser into `internal/capability`. Distinct API-6/API-7/API-8 state machines,
+workers and authority boundaries remain separate. The shipping Console remains
+`static/admin.html` + `static/theme.css`; the previously removed experimental
+`web/` tree is not part of the current source.
+
+Current dependency-free evidence: API source gates
+**69/47/46/72/33/56/45/58/83/100/110/156/259/134/190 PASS**, code-duplication
+source gate **80/80 PASS with 139 unique Admin/update routes**, OpenAI source
+contract **16/16 PASS** plus isolated tests PASS, WAF package-source PASS,
+package-builder **9/9 PASS**, and root Go source-shape **95 files PASS**. An
+isolated dependency-free `internal/capability` test also passes. Canonical Go
+1.25 tidy/build/vet/test/race remains **BLOCKED_ENVIRONMENT / NOT_RUN** on this
+host; none of these static/source results promotes the product to TESTED or
+RELEASED.
+
+Dated sections below are retained as historical engineering/evidence records.
+When an older section conflicts with this section, `CODE_DUPLICATION_REVIEW.md`,
+`DOCUMENTATION_INDEX.md`, and the current source tree are authoritative.
+
 ## Build-integrity and L7 correction — 2026-09-23
 
 An external Go 1.25 CI audit of the previous API-1→API-3 delivery showed that archive integrity and isolated slice tests had not proved package buildability. The current working source repairs the package-level defects found in that audit and an additional duplicate symbol found during follow-up:
@@ -2074,3 +2103,19 @@ Status: IMPLEMENTED_TESTING_DEFERRED. This pass is not API-9. It fixes CIDR Enab
 
 Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.
 
+## 2026-09-28 handoff delta — duplication consolidation
+
+Continue from the Code Duplication Review and Consolidation source, not from the
+2026-09-25 parent tree. Do not restore deleted compatibility/place-holder files
+merely because older ledger sections mention them. Current debug authority is
+`debug_bundle.go` + `debug_sanitize.go` + `support_api.go`; current API-2 schema
+learning is `schema_api2.go`; current CRL URL hardening is `pki.go` +
+`pki_url.go`; current VectorScan runtime/qualification authorities are
+`internal/vectoraccel` and `cmd/wafqualify`; current coverage reporting is the
+ruleset-aware `internal/coverage/crs` path. Both offline CRS coverage and live
+VectorScan SecLang parsing use `internal/capability`.
+
+Older `web/` migration guidance in dated historical sections is superseded: the
+`web/` tree does not exist and must not be treated as a current frontend.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/API1_FINAL_QUALIFICATION_REPORT.md b/API1_FINAL_QUALIFICATION_REPORT.md
index d0fb3f5..7a2f7ed 100644
--- a/API1_FINAL_QUALIFICATION_REPORT.md
+++ b/API1_FINAL_QUALIFICATION_REPORT.md
@@ -68,3 +68,4 @@ Run on qualified Go 1.25 build environment:
 5. artifact reconstruction gate
 6. final source baseline packaging
 
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/API1_HARDENING_RESULT.md b/API1_HARDENING_RESULT.md
index b2f9b0a..3c5e5eb 100644
--- a/API1_HARDENING_RESULT.md
+++ b/API1_HARDENING_RESULT.md
@@ -23,3 +23,5 @@ Root qualification remains blocked by Go 1.25 environment requirement.
 ## Final Qualification Closure
 
 See `API1_FINAL_QUALIFICATION_REPORT.md`. Runtime qualification remains NOT_RUN / BLOCKED until executed on qualified environment.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/API2_CHANGE_SUMMARY.md b/API2_CHANGE_SUMMARY.md
index 762eef5..3e5049a 100644
--- a/API2_CHANGE_SUMMARY.md
+++ b/API2_CHANGE_SUMMARY.md
@@ -10,3 +10,5 @@ Added:
 The implementation intentionally keeps schema learning separate from request blocking.
 
 Qualification: NOT RUN.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/API2_CONTINUATION_CHANGE_SUMMARY.md b/API2_CONTINUATION_CHANGE_SUMMARY.md
index 53ac78c..ce0885a 100644
--- a/API2_CONTINUATION_CHANGE_SUMMARY.md
+++ b/API2_CONTINUATION_CHANGE_SUMMARY.md
@@ -22,3 +22,5 @@ Implementation continuation added.
 
 Runtime qualification:
 NOT RUN
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/API2_DOCUMENTATION_SYNC_CHANGE_SUMMARY.md b/API2_DOCUMENTATION_SYNC_CHANGE_SUMMARY.md
index 43a63db..7559527 100644
--- a/API2_DOCUMENTATION_SYNC_CHANGE_SUMMARY.md
+++ b/API2_DOCUMENTATION_SYNC_CHANGE_SUMMARY.md
@@ -10,3 +10,5 @@ Current truth:
 - API-2 Typed Schema Learning: IMPLEMENTATION COMPLETE
 - Runtime qualification: NOT RUN
 - Release qualification: NOT CLAIMED
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/API2_FINAL_HANDOVER_STATUS.md b/API2_FINAL_HANDOVER_STATUS.md
index 87d324f..2a83fae 100644
--- a/API2_FINAL_HANDOVER_STATUS.md
+++ b/API2_FINAL_HANDOVER_STATUS.md
@@ -32,3 +32,5 @@ Not claimed:
 ## 2026-09-23 closure refresh
 
 API-2 is now source-complete for the agreed Slice-2 scope and remains `IMPLEMENTED_TESTING_DEFERRED`: live bounded observations feed typed schema aggregates/candidates; persistence/restart continuation, lifecycle controls, privacy filtering, aggregate field caps, array-container evidence, dominant-type review invalidation, and composite auth-family telemetry are implemented. Exact-source isolated API-2 tests are **8/8 PASS** and the shared API-1/API-2 source gate is **69 checks PASS**. Repository-root Go 1.25 qualification is still BLOCKED/NOT_RUN.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/API2_IMPLEMENTATION_REPORT.md b/API2_IMPLEMENTATION_REPORT.md
index c79f540..8aa0935 100644
--- a/API2_IMPLEMENTATION_REPORT.md
+++ b/API2_IMPLEMENTATION_REPORT.md
@@ -28,3 +28,5 @@ Runtime qualification was not executed.
 ## 2026-09-23 closure refresh
 
 API-2 is now source-complete for the agreed Slice-2 scope and remains `IMPLEMENTED_TESTING_DEFERRED`: live bounded observations feed typed schema aggregates/candidates; persistence/restart continuation, lifecycle controls, privacy filtering, aggregate field caps, array-container evidence, dominant-type review invalidation, and composite auth-family telemetry are implemented. Exact-source isolated API-2 tests are **8/8 PASS** and the shared API-1/API-2 source gate is **69 checks PASS**. Repository-root Go 1.25 qualification is still BLOCKED/NOT_RUN.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/API2_TESTING_STATUS.md b/API2_TESTING_STATUS.md
index 3b0c2fa..f818f54 100644
--- a/API2_TESTING_STATUS.md
+++ b/API2_TESTING_STATUS.md
@@ -22,3 +22,5 @@ NOT RUN in this artifact generation step.
 ## 2026-09-23 closure refresh
 
 API-2 is now source-complete for the agreed Slice-2 scope and remains `IMPLEMENTED_TESTING_DEFERRED`: live bounded observations feed typed schema aggregates/candidates; persistence/restart continuation, lifecycle controls, privacy filtering, aggregate field caps, array-container evidence, dominant-type review invalidation, and composite auth-family telemetry are implemented. Exact-source isolated API-2 tests are **8/8 PASS** and the shared API-1/API-2 source gate is **69 checks PASS**. Repository-root Go 1.25 qualification is still BLOCKED/NOT_RUN.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/API2_TYPED_SCHEMA_IMPLEMENTATION.md b/API2_TYPED_SCHEMA_IMPLEMENTATION.md
index e457c59..603eb99 100644
--- a/API2_TYPED_SCHEMA_IMPLEMENTATION.md
+++ b/API2_TYPED_SCHEMA_IMPLEMENTATION.md
@@ -20,3 +20,5 @@ Sensitive request values must not be persisted.
 ## 2026-09-23 closure refresh
 
 API-2 is now source-complete for the agreed Slice-2 scope and remains `IMPLEMENTED_TESTING_DEFERRED`: live bounded observations feed typed schema aggregates/candidates; persistence/restart continuation, lifecycle controls, privacy filtering, aggregate field caps, array-container evidence, dominant-type review invalidation, and composite auth-family telemetry are implemented. Exact-source isolated API-2 tests are **8/8 PASS** and the shared API-1/API-2 source gate is **69 checks PASS**. Repository-root Go 1.25 qualification is still BLOCKED/NOT_RUN.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/API3_ACCEPTANCE_CRITERIA.md b/API3_ACCEPTANCE_CRITERIA.md
index 41b676f..2900db7 100644
--- a/API3_ACCEPTANCE_CRITERIA.md
+++ b/API3_ACCEPTANCE_CRITERIA.md
@@ -45,3 +45,5 @@ Until those gates and API-3 tests execute successfully, status remains `IMPLEMEN
 ## 2026-09-23 final closure refresh
 
 API-3 remains `IMPLEMENTED_TESTING_DEFERRED`. Closure hardening adds explicit external-`$ref` rejection, duplicate-normalized-operation rejection, multiple declared content types, OpenAPI security OR/AND/anonymous-alternative semantics, and matched site/host scoping for `UNDECLARED_ENDPOINT`. Current source gate is **47 checks PASS**. Exact-source compile-only PASSes with a minimal YAML shim, and four deterministic runtime/integration tests that do not require YAML behavior PASS. Real `goccy/go-yaml` runtime and repository-root Go 1.25 qualification remain BLOCKED/NOT_RUN on this host.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/API3_OPENAPI_CONTRACT_IMPLEMENTATION.md b/API3_OPENAPI_CONTRACT_IMPLEMENTATION.md
index a0e1bbc..db8467d 100644
--- a/API3_OPENAPI_CONTRACT_IMPLEMENTATION.md
+++ b/API3_OPENAPI_CONTRACT_IMPLEMENTATION.md
@@ -65,3 +65,5 @@ See `API3_TEST_MATRIX.md` and `API3_ACCEPTANCE_CRITERIA.md`.
 ## 2026-09-23 final closure refresh
 
 API-3 remains `IMPLEMENTED_TESTING_DEFERRED`. Closure hardening adds explicit external-`$ref` rejection, duplicate-normalized-operation rejection, multiple declared content types, OpenAPI security OR/AND/anonymous-alternative semantics, and matched site/host scoping for `UNDECLARED_ENDPOINT`. Current source gate is **47 checks PASS**. Exact-source compile-only PASSes with a minimal YAML shim, and four deterministic runtime/integration tests that do not require YAML behavior PASS. Real `goccy/go-yaml` runtime and repository-root Go 1.25 qualification remain BLOCKED/NOT_RUN on this host.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/API3_SOURCE_GATE_RESULT.md b/API3_SOURCE_GATE_RESULT.md
index c45dcd3..0a724e9 100644
--- a/API3_SOURCE_GATE_RESULT.md
+++ b/API3_SOURCE_GATE_RESULT.md
@@ -25,3 +25,5 @@ Those are deferred to the final qualification gate. This PASS must not be used t
 ## 2026-09-23 final closure refresh
 
 API-3 remains `IMPLEMENTED_TESTING_DEFERRED`. Closure hardening adds explicit external-`$ref` rejection, duplicate-normalized-operation rejection, multiple declared content types, OpenAPI security OR/AND/anonymous-alternative semantics, and matched site/host scoping for `UNDECLARED_ENDPOINT`. Current source gate is **47 checks PASS**. Exact-source compile-only PASSes with a minimal YAML shim, and four deterministic runtime/integration tests that do not require YAML behavior PASS. Real `goccy/go-yaml` runtime and repository-root Go 1.25 qualification remain BLOCKED/NOT_RUN on this host.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/API3_TEST_MATRIX.md b/API3_TEST_MATRIX.md
index 6ba3eca..afef965 100644
--- a/API3_TEST_MATRIX.md
+++ b/API3_TEST_MATRIX.md
@@ -73,3 +73,5 @@ Status: **PREPARED — Go execution deferred to final qualification gate**
 ## 2026-09-23 final closure refresh
 
 API-3 remains `IMPLEMENTED_TESTING_DEFERRED`. Closure hardening adds explicit external-`$ref` rejection, duplicate-normalized-operation rejection, multiple declared content types, OpenAPI security OR/AND/anonymous-alternative semantics, and matched site/host scoping for `UNDECLARED_ENDPOINT`. Current source gate is **47 checks PASS**. Exact-source compile-only PASSes with a minimal YAML shim, and four deterministic runtime/integration tests that do not require YAML behavior PASS. Real `goccy/go-yaml` runtime and repository-root Go 1.25 qualification remain BLOCKED/NOT_RUN on this host.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/API61_SEQUENCE_FOUNDATION.md b/API61_SEQUENCE_FOUNDATION.md
index 6d8f28e..4af8ffe 100644
--- a/API61_SEQUENCE_FOUNDATION.md
+++ b/API61_SEQUENCE_FOUNDATION.md
@@ -29,3 +29,5 @@ API-6.1 cannot influence API-1 through API-5 enforcement results.
 - `sequence_api61_test.go` contains five targeted tests for normalized/verified correlation, deterministic transitions, TTL/cardinality bounds, restart/privacy persistence and concurrent non-blocking snapshots.
 - The current host has no Go or gofmt executable. Targeted Go tests and `go test -race` are therefore `BLOCKED_ENVIRONMENT/NOT_RUN`, not PASS.
 - Repository-wide historical deferred gates were not rerun for this slice, per scope. Canonical Go 1.25 real-dependency qualification remains separate.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/API61_SOURCE_GATE_RESULT.md b/API61_SOURCE_GATE_RESULT.md
index 3364b64..e46b930 100644
--- a/API61_SOURCE_GATE_RESULT.md
+++ b/API61_SOURCE_GATE_RESULT.md
@@ -17,3 +17,5 @@ The source gate confirms the three foundation models, API-1 normalized operation
 | Go 1.25 real-dependency build/vet/full test | `NOT_RUN` | outside this API-6.1-only request and unavailable on this host |
 
 No blocked/not-run row is represented as PASS.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/API62_SOURCE_GATE_RESULT.md b/API62_SOURCE_GATE_RESULT.md
index aa60293..3c02651 100644
--- a/API62_SOURCE_GATE_RESULT.md
+++ b/API62_SOURCE_GATE_RESULT.md
@@ -18,3 +18,5 @@ Additional executed evidence:
 The source gate verifies the LEARN-only authority boundary, normalized operation nodes, API-5 verified-identity-only cohort input, cold-start/maturity semantics, frequency and unique-session evidence, idle/absolute session lifecycle, bounded state, non-blocking request-path handoff, atomic snapshots, durable v2 learning state with v1 migration, read-only Reviewer RBAC/audit visibility, absence of sequence anomaly/enforcement logic, and the presence of seven targeted API-6.2 tests.
 
 No result in this file upgrades API-6.2 to `TESTED` or `RELEASED`.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/API62_WORKFLOW_LEARNING.md b/API62_WORKFLOW_LEARNING.md
index bbdeac6..1661bff 100644
--- a/API62_WORKFLOW_LEARNING.md
+++ b/API62_WORKFLOW_LEARNING.md
@@ -51,3 +51,5 @@ The embedded console shows workflow maturity, observations, sessions, depth, tra
 - A Go 1.23 compatibility compile attempt with module lookup disabled was also blocked by unavailable real dependencies and is not qualification evidence.
 
 No stub or downgraded-toolchain result is used to promote the slice. Status therefore remains `IMPLEMENTED_TESTING_DEFERRED`.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/API63_SEQUENCE_ANOMALY_DETECTION.md b/API63_SEQUENCE_ANOMALY_DETECTION.md
index bd0eb53..7aebf49 100644
--- a/API63_SEQUENCE_ANOMALY_DETECTION.md
+++ b/API63_SEQUENCE_ANOMALY_DETECTION.md
@@ -78,3 +78,5 @@ Seven API-6.3 Go test functions are present for cold-start/mature unknown transi
 ## Next slice
 
 `API-6.4 Sequence Operations + Hardening` remains `PLANNED`.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/API63_SOURCE_GATE_RESULT.md b/API63_SOURCE_GATE_RESULT.md
index c080ebb..1127971 100644
--- a/API63_SOURCE_GATE_RESULT.md
+++ b/API63_SOURCE_GATE_RESULT.md
@@ -29,3 +29,5 @@ Canonical Go 1.25 targeted and race commands were attempted with `GOTOOLCHAIN=lo
 The API-6.3 source gate verifies all six anomaly classes, mature/sample/session/age/confidence safeguards, exception evaluation, bounded TTL evidence, v2-to-v3 additive persistence compatibility, privacy-preserving normalized/keyed evidence, Reviewer RBAC/audit visibility, absence of sequence enforcement/OpenAI request-path authority, retained prior source gates and seven targeted API-6.3 Go tests.
 
 No result in this file upgrades API-6.3 to `TESTED` or `RELEASED`.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/API64_SEQUENCE_OPERATIONS_HARDENING.md b/API64_SEQUENCE_OPERATIONS_HARDENING.md
index 4940d7a..98ca6d8 100644
--- a/API64_SEQUENCE_OPERATIONS_HARDENING.md
+++ b/API64_SEQUENCE_OPERATIONS_HARDENING.md
@@ -49,3 +49,5 @@ Local source/static evidence for this slice includes the dedicated API-6.4 sourc
 ## Next slice
 
 API-6 is now complete at implementation level through API-6.4. The next roadmap slice is `API-7.1 Object Locator Discovery`, still `PLANNED`.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/API64_SOURCE_GATE_RESULT.md b/API64_SOURCE_GATE_RESULT.md
index 4e7e32b..aac3c33 100644
--- a/API64_SOURCE_GATE_RESULT.md
+++ b/API64_SOURCE_GATE_RESULT.md
@@ -11,3 +11,5 @@ Status: `IMPLEMENTED_TESTING_DEFERRED`
 The gate verifies explicit bounded `LEARN`/`DETECT` site controls, mature-only DETECT promotion, v4 persistence with v1/v2/v3 restore compatibility, bounded recent-session summaries, site-scoped reset/relearn, bounded exception CRUD, Reviewer RBAC and audit/persistence hooks, complete Sequence Operations console surfaces, absence of sequence ENFORCE authority, retained prior source gates, and targeted API-6.4 test presence.
 
 Canonical Go 1.25 targeted/race execution is separately classified `BLOCKED_ENVIRONMENT / NOT_RUN`; source-gate success is not runtime qualification.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/API71_OBJECT_LOCATOR_DISCOVERY.md b/API71_OBJECT_LOCATOR_DISCOVERY.md
index f790929..8d27a87 100644
--- a/API71_OBJECT_LOCATOR_DISCOVERY.md
+++ b/API71_OBJECT_LOCATOR_DISCOVERY.md
@@ -72,3 +72,5 @@ The dedicated API-7.1 source gate and retained API-1→API-6.4 compatibility/sou
 ## Next slice
 
 `API-7.2 — Identity/Object Relationship` is the next `PLANNED` slice. API-7.3 BOLA Detection, API-7.4 BOLA Policy/Evidence/Console, and API-8 remain `PLANNED`.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/API71_SOURCE_GATE_RESULT.md b/API71_SOURCE_GATE_RESULT.md
index f1e1bc3..3fb61f0 100644
--- a/API71_SOURCE_GATE_RESULT.md
+++ b/API71_SOURCE_GATE_RESULT.md
@@ -31,3 +31,5 @@ Canonical Go qualification attempt:
 - `GOTOOLCHAIN=local GOPROXY=off go test -race ./... -run '^TestAPI71' -count=1`: **BLOCKED_ENVIRONMENT / NOT_RUN** for the same prerequisite
 
 No downgraded-toolchain or stub result is used to promote API-7.1. Production/live qualification remains deferred.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/API72_IDENTITY_OBJECT_RELATIONSHIP.md b/API72_IDENTITY_OBJECT_RELATIONSHIP.md
index 04a202c..0852276 100644
--- a/API72_IDENTITY_OBJECT_RELATIONSHIP.md
+++ b/API72_IDENTITY_OBJECT_RELATIONSHIP.md
@@ -78,3 +78,5 @@ OpenAI is absent from API-7.2 relationship authority.
 Exact-source and clean-extract source/static gates are used for this slice. Canonical Go 1.25 targeted/race execution is `BLOCKED_ENVIRONMENT / NOT_RUN` on the current host because installed Go is 1.23.2 and `go.mod` requires Go 1.25.0. External toolchain retrieval is unavailable. This limitation prevents promotion to `TESTED` or `RELEASED`.
 
 Next slice: **API-7.3 — BOLA Detection**.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/API72_SOURCE_GATE_RESULT.md b/API72_SOURCE_GATE_RESULT.md
index 93fe1ad..0bbfc8f 100644
--- a/API72_SOURCE_GATE_RESULT.md
+++ b/API72_SOURCE_GATE_RESULT.md
@@ -45,3 +45,5 @@ Additional executed local evidence:
 - changed Go files: `gofmt` PASS
 
 Canonical Go targeted/race execution remains `BLOCKED_ENVIRONMENT / NOT_RUN`: host Go is 1.23.2, `go.mod` requires >=1.25.0, and external Go toolchain retrieval is unavailable.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/API73_BOLA_DETECTION.md b/API73_BOLA_DETECTION.md
index 54db5b6..f0afe90 100644
--- a/API73_BOLA_DETECTION.md
+++ b/API73_BOLA_DETECTION.md
@@ -115,3 +115,5 @@ Therefore API-7.3 remains `IMPLEMENTED_TESTING_DEFERRED` and is not `TESTED` or
 ## Next slice
 
 API-7.4 — BOLA Policy/Evidence/Console.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/API73_SOURCE_GATE_RESULT.md b/API73_SOURCE_GATE_RESULT.md
index ff6d087..3963e30 100644
--- a/API73_SOURCE_GATE_RESULT.md
+++ b/API73_SOURCE_GATE_RESULT.md
@@ -17,3 +17,5 @@ API73_SOURCE_GATE_PASS checks=110
 The gate verifies the API-7.3 candidate model, mature-baseline requirements, privacy boundaries, API-5/API-7.1/API-7.2 evidence chain, bounded detector state, persistence/restart validation, pre-merge async placement, Reviewer read-only APIs, audit, absence of BOLA enforcement/ownership authority, prior API-7 source-gate retention, and presence of nine targeted API-7.3 Go tests.
 
 This PASS is not root buildability evidence. Canonical Go 1.25 execution remains `BLOCKED_ENVIRONMENT / NOT_RUN` on this host.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/API74_BOLA_POLICY_EVIDENCE_CONSOLE.md b/API74_BOLA_POLICY_EVIDENCE_CONSOLE.md
index d67095f..fb7a2bb 100644
--- a/API74_BOLA_POLICY_EVIDENCE_CONSOLE.md
+++ b/API74_BOLA_POLICY_EVIDENCE_CONSOLE.md
@@ -128,3 +128,5 @@ Therefore API-7.4 remains `IMPLEMENTED_TESTING_DEFERRED`, not `TESTED` or `RELEA
 ## Next slice
 
 API-8 — GraphQL Security remains `PLANNED`.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/API74_SOURCE_GATE_RESULT.md b/API74_SOURCE_GATE_RESULT.md
index 6bc6292..dfef6cd 100644
--- a/API74_SOURCE_GATE_RESULT.md
+++ b/API74_SOURCE_GATE_RESULT.md
@@ -28,3 +28,5 @@ Date: 2026-09-24
 Historical API-7.3 source-gate semantics were updated additively: API-7.4 is allowed to add BOLA mutation routes to the shared router, while the API-7.3 detector source itself is still required to have no policy/evidence mutation handlers. API-7.3 remains 110/110 PASS.
 
 Canonical Go 1.25 test/race execution is not represented by this static/source gate.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/API8_GRAPHQL_SECURITY.md b/API8_GRAPHQL_SECURITY.md
index fa942c0..b58dc47 100644
--- a/API8_GRAPHQL_SECURITY.md
+++ b/API8_GRAPHQL_SECURITY.md
@@ -1,5 +1,34 @@
 # API-8 — GraphQL Security
 
+## Current canonical baseline — 2026-09-28
+
+The current source is the **Code Duplication Review and Consolidation** working
+baseline derived byte-for-byte from the Production Correctness & Control-Plane
+Hardening parent artifact (`90acc977db2998f4a3c1c4cacc07e19f0087f8bc033bce6b9934b83415957be8`)
+before the changes documented in `CODE_DUPLICATION_REVIEW.md`. Status remains
+**IMPLEMENTED_TESTING_DEFERRED**. No API-9 is defined.
+
+The review removed only source layers proven to be unwired, superseded or
+functionally duplicative, and consolidated the duplicated SecLang action/token
+parser into `internal/capability`. Distinct API-6/API-7/API-8 state machines,
+workers and authority boundaries remain separate. The shipping Console remains
+`static/admin.html` + `static/theme.css`; the previously removed experimental
+`web/` tree is not part of the current source.
+
+Current dependency-free evidence: API source gates
+**69/47/46/72/33/56/45/58/83/100/110/156/259/134/190 PASS**, code-duplication
+source gate **80/80 PASS with 139 unique Admin/update routes**, OpenAI source
+contract **16/16 PASS** plus isolated tests PASS, WAF package-source PASS,
+package-builder **9/9 PASS**, and root Go source-shape **95 files PASS**. An
+isolated dependency-free `internal/capability` test also passes. Canonical Go
+1.25 tidy/build/vet/test/race remains **BLOCKED_ENVIRONMENT / NOT_RUN** on this
+host; none of these static/source results promotes the product to TESTED or
+RELEASED.
+
+Dated sections below are retained as historical engineering/evidence records.
+When an older section conflicts with this section, `CODE_DUPLICATION_REVIEW.md`,
+`DOCUMENTATION_INDEX.md`, and the current source tree are authoritative.
+
 Status: `IMPLEMENTED_TESTING_DEFERRED`
 
 API-8 closes the documented API-security implementation roadmap with deterministic GraphQL security while preserving the authority boundaries established by API-1 through API-7.
@@ -78,3 +107,11 @@ Mutation bodies are bounded and strict-decoded. Mutations persist immediately an
 Exact-source source/static evidence is PASS through API-8. API-8 has 22 targeted Go test functions present and a 259-check source gate. Canonical Go 1.25 `go mod tidy -diff`, root build/test, targeted API-8 tests and race tests remain `BLOCKED_ENVIRONMENT/NOT_RUN` because the host has Go 1.23.2, `go.mod` requires Go 1.25.0, and external toolchain retrieval is unavailable.
 
 Therefore API-8 is not `TESTED` or `RELEASED`.
+
+## 2026-09-28 maintenance addendum
+
+The repository-wide duplicate-functionality cleanup does not change GraphQL
+policy semantics, persistence, variable privacy, API-7 evidence integration, or
+LEARN/DETECT/ENFORCE authority. API-8 remains `IMPLEMENTED_TESTING_DEFERRED`.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/API8_POST_AUDIT_HARDENING.md b/API8_POST_AUDIT_HARDENING.md
index 4af7b5e..3943d04 100644
--- a/API8_POST_AUDIT_HARDENING.md
+++ b/API8_POST_AUDIT_HARDENING.md
@@ -1,5 +1,34 @@
 # API-8 Post-Audit Hardening
 
+## Current canonical baseline — 2026-09-28
+
+The current source is the **Code Duplication Review and Consolidation** working
+baseline derived byte-for-byte from the Production Correctness & Control-Plane
+Hardening parent artifact (`90acc977db2998f4a3c1c4cacc07e19f0087f8bc033bce6b9934b83415957be8`)
+before the changes documented in `CODE_DUPLICATION_REVIEW.md`. Status remains
+**IMPLEMENTED_TESTING_DEFERRED**. No API-9 is defined.
+
+The review removed only source layers proven to be unwired, superseded or
+functionally duplicative, and consolidated the duplicated SecLang action/token
+parser into `internal/capability`. Distinct API-6/API-7/API-8 state machines,
+workers and authority boundaries remain separate. The shipping Console remains
+`static/admin.html` + `static/theme.css`; the previously removed experimental
+`web/` tree is not part of the current source.
+
+Current dependency-free evidence: API source gates
+**69/47/46/72/33/56/45/58/83/100/110/156/259/134/190 PASS**, code-duplication
+source gate **80/80 PASS with 139 unique Admin/update routes**, OpenAI source
+contract **16/16 PASS** plus isolated tests PASS, WAF package-source PASS,
+package-builder **9/9 PASS**, and root Go source-shape **95 files PASS**. An
+isolated dependency-free `internal/capability` test also passes. Canonical Go
+1.25 tidy/build/vet/test/race remains **BLOCKED_ENVIRONMENT / NOT_RUN** on this
+host; none of these static/source results promotes the product to TESTED or
+RELEASED.
+
+Dated sections below are retained as historical engineering/evidence records.
+When an older section conflicts with this section, `CODE_DUPLICATION_REVIEW.md`,
+`DOCUMENTATION_INDEX.md`, and the current source tree are authoritative.
+
 Status: **IMPLEMENTED_TESTING_DEFERRED**
 
 This hardening pass follows the API-8 GraphQL Security implementation. It does not define API-9 and does not change the API-6/API-7/API-8 authority model.
@@ -36,7 +65,7 @@ The following model-only files were removed because they had no runtime store/AP
 - `debug_lifecycle_v2.go`
 - `debug_retention_worker_v2.go`
 
-The real bounded debug implementation remains `debug_bundle.go`, `debug_evidence.go`, `support_api.go`, CLI integration and the SYSTEM Console. Phase 5 Slice D is therefore corrected to PLANNED / NOT IMPLEMENTED rather than claiming model-only structs as implementation.
+The real bounded debug implementation remains `debug_bundle.go`, shared `debug_sanitize.go`, `support_api.go`, CLI integration and the SYSTEM Console. The superseded `debug_evidence.go` compatibility layer was later removed by the 2026-09-28 duplication consolidation. Phase 5 Slice D is therefore corrected to PLANNED / NOT IMPLEMENTED rather than claiming model-only structs as implementation.
 
 ## Authority boundary
 
@@ -52,3 +81,13 @@ The real bounded debug implementation remains `debug_bundle.go`, `debug_evidence
 ## Qualification truth
 
 Dependency-free source/static/package gates pass on the exact hardening source. Canonical Go 1.25 `tidy/build/test/race` remains **BLOCKED_ENVIRONMENT / NOT_RUN** on the current host because only Go 1.23.2 is locally installed and external toolchain retrieval is unavailable. This hardening must not be promoted to TESTED or RELEASED from source/static evidence alone.
+
+## 2026-09-28 maintenance addendum
+
+The later duplication consolidation removes additional legacy/unwired helper
+layers identified by a whole-repository liveness/overlap review. The API-8
+post-audit functional behavior remains in force. Current debug sanitization is
+centralized in `debug_sanitize.go`; the old `debug_evidence.go` compatibility
+layer is no longer present.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/API8_POST_AUDIT_HARDENING_SOURCE_GATE_RESULT.md b/API8_POST_AUDIT_HARDENING_SOURCE_GATE_RESULT.md
index 3decd55..3bf3d95 100644
--- a/API8_POST_AUDIT_HARDENING_SOURCE_GATE_RESULT.md
+++ b/API8_POST_AUDIT_HARDENING_SOURCE_GATE_RESULT.md
@@ -17,3 +17,5 @@ API8_POST_AUDIT_HARDENING_SOURCE_GATE_PASS checks=134
 The gate verifies CIDR expiry/enabled runtime wiring, implemented TLS handshake rate limiting and external-frontend fail-closed validation, SYSTEM/HSM/Vector/Debug/Doctor Console exposure, CIDR/L7 traffic controls, OpenAPI and Positive Schema lifecycle surfaces, opaque-ID inventory selectors, removal of misleading model-only foundation code and the obsolete dual frontend, backend route retention, API-5/L7 middleware authority ordering, Console ID wiring, and build/CI gate integration.
 
 This is a dependency-free source/static gate, not canonical Go 1.25 runtime qualification.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/API8_SOURCE_GATE_RESULT.md b/API8_SOURCE_GATE_RESULT.md
index 68d6879..ce4b6c4 100644
--- a/API8_SOURCE_GATE_RESULT.md
+++ b/API8_SOURCE_GATE_RESULT.md
@@ -58,3 +58,5 @@ The local toolchain is Go 1.23.2 while `go.mod` requires Go 1.25.0. `GOTOOLCHAIN
 - downstream real-dependency/production qualification.
 
 Source/static PASS does not promote API-8 to `TESTED` or `RELEASED`.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/API_SECURITY_CLOSURE_RESULT.md b/API_SECURITY_CLOSURE_RESULT.md
index 0ffbc6e..ff4559e 100644
--- a/API_SECURITY_CLOSURE_RESULT.md
+++ b/API_SECURITY_CLOSURE_RESULT.md
@@ -1,5 +1,34 @@
 # API Security API-1 → API-3 Closure Result
 
+## Current canonical baseline — 2026-09-28
+
+The current source is the **Code Duplication Review and Consolidation** working
+baseline derived byte-for-byte from the Production Correctness & Control-Plane
+Hardening parent artifact (`90acc977db2998f4a3c1c4cacc07e19f0087f8bc033bce6b9934b83415957be8`)
+before the changes documented in `CODE_DUPLICATION_REVIEW.md`. Status remains
+**IMPLEMENTED_TESTING_DEFERRED**. No API-9 is defined.
+
+The review removed only source layers proven to be unwired, superseded or
+functionally duplicative, and consolidated the duplicated SecLang action/token
+parser into `internal/capability`. Distinct API-6/API-7/API-8 state machines,
+workers and authority boundaries remain separate. The shipping Console remains
+`static/admin.html` + `static/theme.css`; the previously removed experimental
+`web/` tree is not part of the current source.
+
+Current dependency-free evidence: API source gates
+**69/47/46/72/33/56/45/58/83/100/110/156/259/134/190 PASS**, code-duplication
+source gate **80/80 PASS with 139 unique Admin/update routes**, OpenAI source
+contract **16/16 PASS** plus isolated tests PASS, WAF package-source PASS,
+package-builder **9/9 PASS**, and root Go source-shape **95 files PASS**. An
+isolated dependency-free `internal/capability` test also passes. Canonical Go
+1.25 tidy/build/vet/test/race remains **BLOCKED_ENVIRONMENT / NOT_RUN** on this
+host; none of these static/source results promotes the product to TESTED or
+RELEASED.
+
+Dated sections below are retained as historical engineering/evidence records.
+When an older section conflicts with this section, `CODE_DUPLICATION_REVIEW.md`,
+`DOCUMENTATION_INDEX.md`, and the current source tree are authoritative.
+
 Date: 2026-09-23 (Asia/Taipei)
 
 Status: `IMPLEMENTED_TESTING_DEFERRED`
@@ -211,3 +240,13 @@ The API-1 through API-8 authority roadmap remains implementation-complete, but a
 
 Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.
 
+## 2026-09-28 — Duplication review closure addendum
+
+The repository-wide duplication consolidation does not change API-security
+policy authority. API-1 through API-8 remain `IMPLEMENTED_TESTING_DEFERRED`;
+API-6 and API-7 learned/inferred evidence remains non-blocking; API-8 request
+blocking remains limited to explicit deterministic ENFORCE policy; OpenAI has no
+request-path authority. The cleanup removes only superseded/unwired source and
+centralizes shared SecLang parsing. Source gate: **80/80 PASS**.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/API_SECURITY_ROADMAP.md b/API_SECURITY_ROADMAP.md
index 5a01118..047527c 100644
--- a/API_SECURITY_ROADMAP.md
+++ b/API_SECURITY_ROADMAP.md
@@ -1,5 +1,34 @@
 # API Security Roadmap — API-aware WAF Evolution
 
+## Current canonical baseline — 2026-09-28
+
+The current source is the **Code Duplication Review and Consolidation** working
+baseline derived byte-for-byte from the Production Correctness & Control-Plane
+Hardening parent artifact (`90acc977db2998f4a3c1c4cacc07e19f0087f8bc033bce6b9934b83415957be8`)
+before the changes documented in `CODE_DUPLICATION_REVIEW.md`. Status remains
+**IMPLEMENTED_TESTING_DEFERRED**. No API-9 is defined.
+
+The review removed only source layers proven to be unwired, superseded or
+functionally duplicative, and consolidated the duplicated SecLang action/token
+parser into `internal/capability`. Distinct API-6/API-7/API-8 state machines,
+workers and authority boundaries remain separate. The shipping Console remains
+`static/admin.html` + `static/theme.css`; the previously removed experimental
+`web/` tree is not part of the current source.
+
+Current dependency-free evidence: API source gates
+**69/47/46/72/33/56/45/58/83/100/110/156/259/134/190 PASS**, code-duplication
+source gate **80/80 PASS with 139 unique Admin/update routes**, OpenAI source
+contract **16/16 PASS** plus isolated tests PASS, WAF package-source PASS,
+package-builder **9/9 PASS**, and root Go source-shape **95 files PASS**. An
+isolated dependency-free `internal/capability` test also passes. Canonical Go
+1.25 tidy/build/vet/test/race remains **BLOCKED_ENVIRONMENT / NOT_RUN** on this
+host; none of these static/source results promotes the product to TESTED or
+RELEASED.
+
+Dated sections below are retained as historical engineering/evidence records.
+When an older section conflicts with this section, `CODE_DUPLICATION_REVIEW.md`,
+`DOCUMENTATION_INDEX.md`, and the current source tree are authoritative.
+
 ## Purpose
 
 This document defines the future API-aware WAF product line built on the existing
@@ -27,7 +56,7 @@ API Security slices below.
 
 ## Roadmap Status
 
-Current source status is recorded in the table below. API-1 through API-5, API-6.1 and API-6.2 are implemented in source and remain `IMPLEMENTED_TESTING_DEFERRED`. API-6.3 and API-6.4 are `IMPLEMENTED_TESTING_DEFERRED`; API-7.1 Object Locator Discovery, API-7.2 Identity/Object Relationship, API-7.3 BOLA Detection and API-7.4 BOLA Policy/Evidence/Console are `IMPLEMENTED_TESTING_DEFERRED`; API-8 remains `PLANNED`.
+Current source status is recorded in the table below. API-1 through API-8 are implemented in source and remain `IMPLEMENTED_TESTING_DEFERRED`. The API-security implementation roadmap is complete through API-8; no API-9 is defined. Canonical Go 1.25 runtime/race qualification is still required before release promotion.
 
 ---
 
@@ -460,3 +489,10 @@ No API-9 is defined. API-8 post-audit hardening closes runtime correctness and o
 
 Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.
 
+## 2026-09-28 maintenance note
+
+The code-duplication consolidation is a maintenance wave, not API-9 and not a
+change to the API-security roadmap. API-1 through API-8 remain
+`IMPLEMENTED_TESTING_DEFERRED`; no API-9 is defined.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/API_SECURITY_ROADMAP_CHANGE.md b/API_SECURITY_ROADMAP_CHANGE.md
index b583d85..10dfea1 100644
--- a/API_SECURITY_ROADMAP_CHANGE.md
+++ b/API_SECURITY_ROADMAP_CHANGE.md
@@ -22,3 +22,5 @@ The original “feature implementation: NOT STARTED / all slices PLANNED” entr
 ## API-7.3 BOLA Detection implementation checkpoint — 2026-09-24
 
 API-7.3 moved from `PLANNED` to `IMPLEMENTED_TESTING_DEFERRED`. It adds bounded, non-enforcing BOLA candidate evidence over API-5/API-7.1/API-7.2 trusted/pseudonymous inputs. API-7.4 BOLA Policy/Evidence/Console is now the next `PLANNED` slice; API-8 remains `PLANNED`.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/API_SECURITY_SLICE_IMPLEMENTATION_ROADMAP.md b/API_SECURITY_SLICE_IMPLEMENTATION_ROADMAP.md
index e4e26c1..8e9ba21 100644
--- a/API_SECURITY_SLICE_IMPLEMENTATION_ROADMAP.md
+++ b/API_SECURITY_SLICE_IMPLEMENTATION_ROADMAP.md
@@ -1,5 +1,34 @@
 # API Security Slice Implementation Roadmap
 
+## Current canonical baseline — 2026-09-28
+
+The current source is the **Code Duplication Review and Consolidation** working
+baseline derived byte-for-byte from the Production Correctness & Control-Plane
+Hardening parent artifact (`90acc977db2998f4a3c1c4cacc07e19f0087f8bc033bce6b9934b83415957be8`)
+before the changes documented in `CODE_DUPLICATION_REVIEW.md`. Status remains
+**IMPLEMENTED_TESTING_DEFERRED**. No API-9 is defined.
+
+The review removed only source layers proven to be unwired, superseded or
+functionally duplicative, and consolidated the duplicated SecLang action/token
+parser into `internal/capability`. Distinct API-6/API-7/API-8 state machines,
+workers and authority boundaries remain separate. The shipping Console remains
+`static/admin.html` + `static/theme.css`; the previously removed experimental
+`web/` tree is not part of the current source.
+
+Current dependency-free evidence: API source gates
+**69/47/46/72/33/56/45/58/83/100/110/156/259/134/190 PASS**, code-duplication
+source gate **80/80 PASS with 139 unique Admin/update routes**, OpenAI source
+contract **16/16 PASS** plus isolated tests PASS, WAF package-source PASS,
+package-builder **9/9 PASS**, and root Go source-shape **95 files PASS**. An
+isolated dependency-free `internal/capability` test also passes. Canonical Go
+1.25 tidy/build/vet/test/race remains **BLOCKED_ENVIRONMENT / NOT_RUN** on this
+host; none of these static/source results promotes the product to TESTED or
+RELEASED.
+
+Dated sections below are retained as historical engineering/evidence records.
+When an older section conflicts with this section, `CODE_DUPLICATION_REVIEW.md`,
+`DOCUMENTATION_INDEX.md`, and the current source tree are authoritative.
+
 ## Purpose
 
 This document is the canonical implementation blueprint for API-aware WAF security evolution.
@@ -239,3 +268,10 @@ The documented API-security implementation roadmap is now complete through API-8
 
 Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.
 
+## 2026-09-28 maintenance note
+
+The repository-wide duplicate-functionality cleanup does not add, remove or
+reorder API-security slices. API-1 through API-8 remain implemented with testing
+deferred. No API-9 has been defined.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/CLAUDE.md b/CLAUDE.md
index 5296162..3fbebc4 100644
--- a/CLAUDE.md
+++ b/CLAUDE.md
@@ -18,3 +18,5 @@ Update the canonical documentation set after meaningful work:
 For local/terminal sessions, the user may handle Git operations manually. For
 cloud/web sessions, use an isolated branch and pull request. Never commit or
 merge development work directly to `main` merely to obtain CI evidence.
+
+<!-- documentation-review: 2026-09-28; classification: repository policy; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/CLEAN_HOST_DISTRIBUTION_GATE_RESULT.md b/CLEAN_HOST_DISTRIBUTION_GATE_RESULT.md
index 4363e39..e70c170 100644
--- a/CLEAN_HOST_DISTRIBUTION_GATE_RESULT.md
+++ b/CLEAN_HOST_DISTRIBUTION_GATE_RESULT.md
@@ -56,3 +56,5 @@ This document is scoped package/distribution evidence. The 2026-09-17 audit of
 not yet executed on Go 1.25 in this environment. Therefore no PASS in this file
 may be interpreted as a current production WAF binary/package or overall release
 PASS. See `SOURCE_BASELINE_GATE_RESULT.md` and `DOCUMENTATION_INDEX.md`.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/CODE_DUPLICATION_REVIEW.md b/CODE_DUPLICATION_REVIEW.md
new file mode 100644
index 0000000..1dd74e1
--- /dev/null
+++ b/CODE_DUPLICATION_REVIEW.md
@@ -0,0 +1,185 @@
+# Code Duplication and Functional-Overlap Review
+
+**Review date:** 2026-09-28 (Asia/Taipei)  
+**Parent source:** `waf-proxy-production-control-plane-hardening-2026-09-25.zip`  
+**Parent SHA-256:** `90acc977db2998f4a3c1c4cacc07e19f0087f8bc033bce6b9934b83415957be8`  
+**State:** `IMPLEMENTED_TESTING_DEFERRED`
+
+## Scope
+
+This review covers the complete source tree, including the root WAF service,
+API-1 through API-8 security slices, control-plane/Admin API, shipping Console,
+PKI, HA, debug/support evidence, VectorScan/coverage tooling, qualification
+packages, packaging, service units, and release tooling.
+
+The review distinguished three different cases:
+
+1. **Functional overlap** — two implementations claim the same runtime or
+   operator responsibility. These must be consolidated or one must be removed.
+2. **Code-shape similarity** — small lifecycle/list/snapshot helpers are similar
+   but operate on different bounded stores with different authority. These are
+   not merged merely for DRY style.
+3. **Historical/qualification models** — code under explicit qualification
+   scope may intentionally model evidence without participating in the runtime.
+   It is retained when it has a real qualification responsibility.
+
+The source archive contains no `.git` metadata. No branch, commit, pull, push,
+or fresh-clone claim is made. Source identity is the parent ZIP checksum plus
+its manifests.
+
+## Review methods
+
+- enumerated all Go source files and package directories;
+- checked Admin `METHOD + path` registrations for collisions;
+- compared top-level declarations and substantial exact function bodies;
+- scanned production symbols for definitions with no production caller;
+- traced runtime stores, persistence, request-path and Admin/API callers;
+- compared shipping Console function names and HTML IDs for duplicates;
+- reviewed API security enforcement boundaries for competing authority;
+- compared offline CRS coverage parsing with live VectorScan parsing;
+- re-ran all existing source gates after consolidation.
+
+## Consolidated duplicate or superseded functionality
+
+### Debug evidence
+
+The repository carried four overlapping layers:
+
+- the active tenant-scoped `DebugEvidenceStore` in `debug_bundle.go`;
+- a never-initialized compatibility `DebugEvidenceCapture` in
+  `debug_evidence.go`;
+- a test-only `DebugEvidenceOpsV2` wrapper;
+- unused support-bundle provenance structs.
+
+The compatibility capture was never installed by production code, so its
+Coraza callback branch was always inactive. The active store is now the only
+debug evidence authority. Shared sanitization moved to `debug_sanitize.go`.
+The obsolete compatibility wrapper, V2 wrapper and unused support-bundle model
+were removed. `DebugEvidenceStore.TenantExport`, which had no caller and merely
+wrapped `ExportIncident`, was also removed.
+
+### Generic security-state placeholders
+
+`security_state.go` and `security_event.go` defined a second generic in-memory
+indicator/audit/event model with no runtime caller. The product already has
+concrete bounded stores and audit/notification/evidence models. The unused
+generic layer and its isolated test were removed so there is no second implied
+security-state authority.
+
+### Deployment/reliability foundations
+
+`deployment_readiness.go` duplicated the responsibility already served by the
+wired `/api/doctor`, `waf-doctor.sh`, package qualification and release-host
+qualification flows. `reliability.go` was a standalone `NOT_RUN` report model
+used only by its own test. Both were removed; the real diagnostic and
+qualification paths remain unchanged.
+
+### VectorScan qualification/audit placeholders
+
+Root-level `vectorscan_qualification.go`, `vectorscan_audit_store.go` and
+`vectorscan_transition_audit.go` were not wired to runtime or the real
+qualification command. The authoritative paths are:
+
+- `internal/vectoraccel` for runtime planning/learning and native observation;
+- `cmd/wafqualify` for real Coraza-versus-VectorScan differential execution;
+- `qualification/vectorscan` for qualification evidence schema/tests.
+
+The unused root placeholder models were removed. In
+`internal/vectoraccel/qualification.go`, the test-only `QualificationResult` /
+`CompareCandidates` duplicate was removed while the live
+`QualificationCandidates`, `EligibleRuleIDs`, `CandidateRuleIDs` and
+`EligibleMatchedRuleIDs` methods remain.
+
+### API-2 continuation placeholder
+
+`schema_api2_continuation.go` had no caller. The live API-2 schema collector,
+review lifecycle, persistence and Admin surface are in `schema_api2.go` and its
+existing integrations. The unreferenced continuation model was removed.
+
+### PKI CRL hardening duplicate
+
+`pki_hardening.go` was an isolated helper/test pair and was not used by the
+backend TLS path. The wired CRL implementation in `pki.go` + `pki_url.go` is
+stricter: HTTPS-only URL syntax, DNS/address SSRF controls, response size
+bounds, parsing/signature validation, cache handling and last-known-good
+refresh behavior. The unused earlier helper/test pair was removed.
+
+### CRS/VectorScan SecLang parsing
+
+Offline CRS coverage analysis and the live VectorScan rule classifier carried
+separate implementations of token parsing, action-list splitting,
+`name:value` splitting and quoted-action trimming. These were semantically the
+same and could drift independently.
+
+`internal/capability/seclang.go` is now the single parser-helper source used by
+both `internal/coverage/crs/parser.go` and `internal/vectoraccel/rules.go`.
+Focused tests lock quote/comma/action behavior.
+
+### Legacy coverage report
+
+`internal/coverage/report.go` was the unused Slice-A summary while
+`internal/coverage/crs/report.go` is the current ruleset-aware Coverage Report
+v2 used by `wafctl`. The unused report was removed. The unused transform-report
+API in `internal/coverage/capability.go` was also removed; eligibility continues
+to be decided by `internal/capability.Classify`.
+
+### Other unused compatibility helpers
+
+The review removed small no-caller helpers whose live replacements already
+exist: `QualificationPassed`, `ActualListenerKey`, `nonAnonymousAuthSamples`,
+`sequenceTransitionID`, and the uninstalled `ClientIdentityAuditSink` hook.
+These removals do not change a public importable API because the affected root
+package is `main` and the TLS helper is under `internal/`.
+
+## Similar code intentionally retained
+
+The following similarities are **not functional duplication** and were left in
+place:
+
+- API-6 sequence, API-7 relationship/BOLA and API-8 GraphQL stores each own
+  distinct state, persistence and authority boundaries.
+- `matchRing` stores recent operator-visible records while `matchLogPlane`
+  asynchronously aggregates human-oriented structured log output.
+- observation, sequence and relationship workers have similar start/drain
+  lifecycle code but process different queues and state machines.
+- short list/status handlers are intentionally local to their resource model.
+- `notify.go` and `syslog.go` are distinct delivery sinks with different
+  semantics; neither replaces the other.
+- packaging/update scripts and runtime update controls remain separate because
+  one builds/distributes packages while the other governs an installed node.
+
+Merging these only to reduce line count would couple independent authority or
+failure domains and is therefore rejected.
+
+## Authority review
+
+No competing request-path enforcement stack was found:
+
+- Coraza/CRS remains deterministic WAF enforcement.
+- API-4 Positive Schema may enforce explicit reviewed schema policy.
+- API-5 may enforce cryptographically verified identity policy.
+- API-6 sequence state is LEARN/DETECT only.
+- API-7 learned/inferred BOLA evidence remains non-blocking.
+- API-8 may enforce explicit deterministic GraphQL policy.
+- OpenAI remains advisory and has no request-path authority.
+
+Admin route registrations remain unique by `METHOD + path`; the cleanup does
+not introduce an alternate management route for the same action.
+
+## Console review
+
+The shipping Console has one implementation (`static/admin.html` plus
+`static/theme.css`). The removed experimental `web/` tree remains absent.
+Within `static/admin.html`, JavaScript function names and static HTML IDs are
+unique. Existing role/capability checks and API surfaces remain unchanged by
+this consolidation.
+
+## Verification boundary
+
+All dependency-free/API source gates are re-run after this cleanup, including
+the dedicated `test-code-duplication-review-source.py` gate. Canonical Go 1.25
+`go mod tidy -diff`, build, tests, race and real dependency qualification still
+require the pinned Go 1.25 environment. This review must not be promoted to
+`TESTED` or `RELEASED` based only on source/static evidence.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/CODE_DUPLICATION_REVIEW_SOURCE_GATE_RESULT.md b/CODE_DUPLICATION_REVIEW_SOURCE_GATE_RESULT.md
new file mode 100644
index 0000000..3e2ef7a
--- /dev/null
+++ b/CODE_DUPLICATION_REVIEW_SOURCE_GATE_RESULT.md
@@ -0,0 +1,35 @@
+# Code Duplication Review Source Gate Result
+
+## Current result — 2026-09-28
+
+Status: **PASS (dependency-free source gate)**.
+
+Command:
+
+```bash
+python3 tools/tests/test-code-duplication-review-source.py
+```
+
+Observed result:
+
+```text
+CODE_DUPLICATION_REVIEW_SOURCE_GATE_PASS checks=80 routes=139
+```
+
+The gate verifies that the confirmed superseded/dead duplicate layers removed by
+`CODE_DUPLICATION_REVIEW.md` remain absent, the shared SecLang parsing helpers
+are the single source used by both CRS coverage and live VectorScan
+classification, current replacement authorities remain present, Admin route
+registrations are unique, API-6/API-7/API-8 authority boundaries remain
+separate, and the gate itself is retained by build/CI wiring.
+
+This is **source/static evidence only**. It does not replace the mandatory
+pinned Go 1.25 `tidy`, build, vet, test and race qualification. Those canonical
+Go gates remain `BLOCKED_ENVIRONMENT / NOT_RUN` on the current host because the
+host provides Go 1.23.2 and external toolchain/module retrieval is unavailable.
+
+An isolated dependency-free test of `internal/capability` using the host Go
+1.23 toolchain passed after consolidation. That isolated test is advisory and
+is not a substitute for the repository's pinned Go 1.25 qualification.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/CODE_DUPLICATION_REVIEW_TEST_SUMMARY.txt b/CODE_DUPLICATION_REVIEW_TEST_SUMMARY.txt
new file mode 100644
index 0000000..4f6ba37
--- /dev/null
+++ b/CODE_DUPLICATION_REVIEW_TEST_SUMMARY.txt
@@ -0,0 +1,11 @@
+Code Duplication Review — 2026-09-28
+Status: IMPLEMENTED_TESTING_DEFERRED
+Source gate: CODE_DUPLICATION_REVIEW_SOURCE_GATE_PASS checks=80 routes=139
+API source gates: 69/47/46/72/33/56/45/58/83/100/110/156/259/134/190 PASS
+OpenAI source contract: 16/16 PASS
+OpenAI isolated packages: PASS
+WAF package-source: PASS
+Package builder: 9/9 PASS
+Root Go source shape: 95 files PASS
+internal/capability isolated dependency-free test: PASS (advisory Go 1.23 only)
+Canonical Go 1.25 tidy/build/vet/test/race: BLOCKED_ENVIRONMENT / NOT_RUN
diff --git a/DEB_PACKAGE_GATE_RESULT.md b/DEB_PACKAGE_GATE_RESULT.md
index 8c830d1..d8401bb 100644
--- a/DEB_PACKAGE_GATE_RESULT.md
+++ b/DEB_PACKAGE_GATE_RESULT.md
@@ -43,3 +43,5 @@ This document is scoped package/distribution evidence. The 2026-09-17 audit of
 not yet executed on Go 1.25 in this environment. Therefore no PASS in this file
 may be interpreted as a current production WAF binary/package or overall release
 PASS. See `SOURCE_BASELINE_GATE_RESULT.md` and `DOCUMENTATION_INDEX.md`.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/DEVELOPMENT.md b/DEVELOPMENT.md
index e7ad330..4729cea 100644
--- a/DEVELOPMENT.md
+++ b/DEVELOPMENT.md
@@ -1,5 +1,34 @@
 # Development Engineering Ledger
 
+## Current canonical baseline — 2026-09-28
+
+The current source is the **Code Duplication Review and Consolidation** working
+baseline derived byte-for-byte from the Production Correctness & Control-Plane
+Hardening parent artifact (`90acc977db2998f4a3c1c4cacc07e19f0087f8bc033bce6b9934b83415957be8`)
+before the changes documented in `CODE_DUPLICATION_REVIEW.md`. Status remains
+**IMPLEMENTED_TESTING_DEFERRED**. No API-9 is defined.
+
+The review removed only source layers proven to be unwired, superseded or
+functionally duplicative, and consolidated the duplicated SecLang action/token
+parser into `internal/capability`. Distinct API-6/API-7/API-8 state machines,
+workers and authority boundaries remain separate. The shipping Console remains
+`static/admin.html` + `static/theme.css`; the previously removed experimental
+`web/` tree is not part of the current source.
+
+Current dependency-free evidence: API source gates
+**69/47/46/72/33/56/45/58/83/100/110/156/259/134/190 PASS**, code-duplication
+source gate **80/80 PASS with 139 unique Admin/update routes**, OpenAI source
+contract **16/16 PASS** plus isolated tests PASS, WAF package-source PASS,
+package-builder **9/9 PASS**, and root Go source-shape **95 files PASS**. An
+isolated dependency-free `internal/capability` test also passes. Canonical Go
+1.25 tidy/build/vet/test/race remains **BLOCKED_ENVIRONMENT / NOT_RUN** on this
+host; none of these static/source results promotes the product to TESTED or
+RELEASED.
+
+Dated sections below are retained as historical engineering/evidence records.
+When an older section conflicts with this section, `CODE_DUPLICATION_REVIEW.md`,
+`DOCUMENTATION_INDEX.md`, and the current source tree are authoritative.
+
 ## Build-integrity and L7 correction — 2026-09-23
 
 An external Go 1.25 CI audit of the previous API-1→API-3 delivery showed that archive integrity and isolated slice tests had not proved package buildability. The current working source repairs the package-level defects found in that audit and an additional duplicate symbol found during follow-up:
@@ -477,3 +506,21 @@ Implemented the post-API-8 repository audit fixes without defining a new API-sec
 
 Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.
 
+## 2026-09-28 — Code Duplication Review and Consolidation
+
+Completed a source-wide duplicate-functionality review using route inventory,
+Go AST/function-body comparison, symbol liveness checks, runtime caller tracing,
+Console/API mapping, and source-manifest comparison against the 2026-09-25
+Production Correctness & Control-Plane Hardening parent artifact. Confirmed
+dead/superseded layers were removed and the duplicate SecLang tokenizer/action
+parser was centralized in `internal/capability`. Similar but authority-distinct
+API-6/API-7/API-8 stores/workers were explicitly retained.
+
+The dedicated source gate passes **80/80** with **139 unique Admin/update
+routes**. Existing API source gates, production-hardening gate, OpenAI gates,
+package-source, package-builder and root source-shape remain PASS. The cleanup
+reduces the root Go source-shape count to **95**; that reduction is intentional
+and results from removing dead/superseded root files. Canonical Go 1.25
+qualification remains blocked/not run.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/DEVELOPMENT_ROADMAP.md b/DEVELOPMENT_ROADMAP.md
index c8e6dc3..c1281d7 100644
--- a/DEVELOPMENT_ROADMAP.md
+++ b/DEVELOPMENT_ROADMAP.md
@@ -1,5 +1,34 @@
 # Development Roadmap — WAF Proxy
 
+## Current canonical baseline — 2026-09-28
+
+The current source is the **Code Duplication Review and Consolidation** working
+baseline derived byte-for-byte from the Production Correctness & Control-Plane
+Hardening parent artifact (`90acc977db2998f4a3c1c4cacc07e19f0087f8bc033bce6b9934b83415957be8`)
+before the changes documented in `CODE_DUPLICATION_REVIEW.md`. Status remains
+**IMPLEMENTED_TESTING_DEFERRED**. No API-9 is defined.
+
+The review removed only source layers proven to be unwired, superseded or
+functionally duplicative, and consolidated the duplicated SecLang action/token
+parser into `internal/capability`. Distinct API-6/API-7/API-8 state machines,
+workers and authority boundaries remain separate. The shipping Console remains
+`static/admin.html` + `static/theme.css`; the previously removed experimental
+`web/` tree is not part of the current source.
+
+Current dependency-free evidence: API source gates
+**69/47/46/72/33/56/45/58/83/100/110/156/259/134/190 PASS**, code-duplication
+source gate **80/80 PASS with 139 unique Admin/update routes**, OpenAI source
+contract **16/16 PASS** plus isolated tests PASS, WAF package-source PASS,
+package-builder **9/9 PASS**, and root Go source-shape **95 files PASS**. An
+isolated dependency-free `internal/capability` test also passes. Canonical Go
+1.25 tidy/build/vet/test/race remains **BLOCKED_ENVIRONMENT / NOT_RUN** on this
+host; none of these static/source results promotes the product to TESTED or
+RELEASED.
+
+Dated sections below are retained as historical engineering/evidence records.
+When an older section conflicts with this section, `CODE_DUPLICATION_REVIEW.md`,
+`DOCUMENTATION_INDEX.md`, and the current source tree are authoritative.
+
 ## Build-integrity and L7 correction — 2026-09-23
 
 An external Go 1.25 CI audit of the previous API-1→API-3 delivery showed that archive integrity and isolated slice tests had not proved package buildability. The current working source repairs the package-level defects found in that audit and an additional duplicate symbol found during follow-up:
@@ -567,7 +596,7 @@ Implemented Responses API, strict Structured Outputs, secret references, and moc
 
 ## Future API Security Product Line
 
-The API-aware WAF roadmap is documented in `API_SECURITY_ROADMAP.md` and `API_SECURITY_SLICE_IMPLEMENTATION_ROADMAP.md`. Current source status: API-1 through API-5, API-6.1 through API-6.4, and API-7.1 through API-7.4 are IMPLEMENTED_TESTING_DEFERRED; API-8 is PLANNED. Go qualification remains deferred to the applicable gate.
+The API-aware WAF roadmap is documented in `API_SECURITY_ROADMAP.md` and `API_SECURITY_SLICE_IMPLEMENTATION_ROADMAP.md`. Current source status: API-1 through API-8 are `IMPLEMENTED_TESTING_DEFERRED`; no API-9 is defined. Go 1.25 qualification remains deferred to the applicable gate.
 
 
 ## API-1 Hardening
@@ -588,7 +617,7 @@ Status: IMPLEMENTED_TESTING_DEFERRED
 - API-6.2 Workflow Learning: `IMPLEMENTED_TESTING_DEFERRED`.
 - API-6.3 Sequence Anomaly Detection: `IMPLEMENTED_TESTING_DEFERRED`.
 - API-6.4 and API-7.1 through API-7.4: `IMPLEMENTED_TESTING_DEFERRED`.
-- API-8: `PLANNED`.
+- API-8: `IMPLEMENTED_TESTING_DEFERRED`; implementation roadmap complete through API-8, with Go 1.25 production qualification still required.
 
 API-3 source includes OpenAPI 3.0/3.1 parsing, local references, contract import/versioning, API-1 operation binding, version/schema comparison, and API-2 drift evidence. No enforcement is introduced. Go 1.25 build/test qualification is deferred to the final qualification gate.
 
@@ -628,3 +657,12 @@ The documented API-security implementation roadmap is now complete through API-8
 
 Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.
 
+## 2026-09-28 — Maintenance consolidation checkpoint
+
+Production Correctness & Control-Plane Hardening remains the functional
+baseline; Code Duplication Review and Consolidation is a source-maintenance
+checkpoint on top of it. It removes superseded/unwired implementations and
+centralizes shared parsing without adding a product slice. API-1 through API-8
+remain `IMPLEMENTED_TESTING_DEFERRED`; no API-9 is defined.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/DOCUMENTATION_INDEX.md b/DOCUMENTATION_INDEX.md
index dfc3243..7922473 100644
--- a/DOCUMENTATION_INDEX.md
+++ b/DOCUMENTATION_INDEX.md
@@ -1,5 +1,34 @@
 # Documentation Index and Current Project Truth
 
+## Current canonical baseline — 2026-09-28
+
+The current source is the **Code Duplication Review and Consolidation** working
+baseline derived byte-for-byte from the Production Correctness & Control-Plane
+Hardening parent artifact (`90acc977db2998f4a3c1c4cacc07e19f0087f8bc033bce6b9934b83415957be8`)
+before the changes documented in `CODE_DUPLICATION_REVIEW.md`. Status remains
+**IMPLEMENTED_TESTING_DEFERRED**. No API-9 is defined.
+
+The review removed only source layers proven to be unwired, superseded or
+functionally duplicative, and consolidated the duplicated SecLang action/token
+parser into `internal/capability`. Distinct API-6/API-7/API-8 state machines,
+workers and authority boundaries remain separate. The shipping Console remains
+`static/admin.html` + `static/theme.css`; the previously removed experimental
+`web/` tree is not part of the current source.
+
+Current dependency-free evidence: API source gates
+**69/47/46/72/33/56/45/58/83/100/110/156/259/134/190 PASS**, code-duplication
+source gate **80/80 PASS with 139 unique Admin/update routes**, OpenAI source
+contract **16/16 PASS** plus isolated tests PASS, WAF package-source PASS,
+package-builder **9/9 PASS**, and root Go source-shape **95 files PASS**. An
+isolated dependency-free `internal/capability` test also passes. Canonical Go
+1.25 tidy/build/vet/test/race remains **BLOCKED_ENVIRONMENT / NOT_RUN** on this
+host; none of these static/source results promotes the product to TESTED or
+RELEASED.
+
+Dated sections below are retained as historical engineering/evidence records.
+When an older section conflicts with this section, `CODE_DUPLICATION_REVIEW.md`,
+`DOCUMENTATION_INDEX.md`, and the current source tree are authoritative.
+
 ## Build-integrity and L7 correction — 2026-09-23
 
 An external Go 1.25 CI audit of the previous API-1→API-3 delivery showed that archive integrity and isolated slice tests had not proved package buildability. The current working source repairs the package-level defects found in that audit and an additional duplicate symbol found during follow-up:
@@ -274,3 +303,17 @@ The documented API-security implementation roadmap is now complete through API-8
 
 Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.
 
+## Documentation review — 2026-09-28
+
+Every Markdown file in the source tree was reviewed for the duplication
+consolidation. Current/canonical documents were synchronized to the 2026-09-28
+source truth. Historical qualification/result documents intentionally retain
+their original dated results; they carry repository review metadata and are not
+current-state authority when they conflict with this index or the top current
+canonical sections. `MARKDOWN_REVIEW_2026-09-28.md` records the complete file
+classification.
+
+New current documents: `CODE_DUPLICATION_REVIEW.md` and
+`CODE_DUPLICATION_REVIEW_SOURCE_GATE_RESULT.md`.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/HANDOVER_PROMPT.md b/HANDOVER_PROMPT.md
index dce68b9..d4b9bd9 100644
--- a/HANDOVER_PROMPT.md
+++ b/HANDOVER_PROMPT.md
@@ -1,5 +1,34 @@
 # New Chat Handover Prompt
 
+## Current canonical baseline — 2026-09-28
+
+The current source is the **Code Duplication Review and Consolidation** working
+baseline derived byte-for-byte from the Production Correctness & Control-Plane
+Hardening parent artifact (`90acc977db2998f4a3c1c4cacc07e19f0087f8bc033bce6b9934b83415957be8`)
+before the changes documented in `CODE_DUPLICATION_REVIEW.md`. Status remains
+**IMPLEMENTED_TESTING_DEFERRED**. No API-9 is defined.
+
+The review removed only source layers proven to be unwired, superseded or
+functionally duplicative, and consolidated the duplicated SecLang action/token
+parser into `internal/capability`. Distinct API-6/API-7/API-8 state machines,
+workers and authority boundaries remain separate. The shipping Console remains
+`static/admin.html` + `static/theme.css`; the previously removed experimental
+`web/` tree is not part of the current source.
+
+Current dependency-free evidence: API source gates
+**69/47/46/72/33/56/45/58/83/100/110/156/259/134/190 PASS**, code-duplication
+source gate **80/80 PASS with 139 unique Admin/update routes**, OpenAI source
+contract **16/16 PASS** plus isolated tests PASS, WAF package-source PASS,
+package-builder **9/9 PASS**, and root Go source-shape **95 files PASS**. An
+isolated dependency-free `internal/capability` test also passes. Canonical Go
+1.25 tidy/build/vet/test/race remains **BLOCKED_ENVIRONMENT / NOT_RUN** on this
+host; none of these static/source results promotes the product to TESTED or
+RELEASED.
+
+Dated sections below are retained as historical engineering/evidence records.
+When an older section conflicts with this section, `CODE_DUPLICATION_REVIEW.md`,
+`DOCUMENTATION_INDEX.md`, and the current source tree are authoritative.
+
 ## API Security checkpoint — 2026-09-23
 
 The uploaded API-3 baseline was audited against source rather than roadmap labels, then repaired/closed through API-3. Current source truth:
@@ -122,3 +151,14 @@ API-8 post-audit hardening is the current implementation baseline once final pac
 
 Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.
 
+## 2026-09-28 continuation instruction
+
+Start from the Code Duplication Review and Consolidation complete-source
+baseline. Read `CODE_DUPLICATION_REVIEW.md`,
+`CODE_DUPLICATION_REVIEW_SOURCE_GATE_RESULT.md`, `DOCUMENTATION_INDEX.md`, and
+`PRODUCTION_CONTROL_PLANE_HARDENING.md` before editing. Preserve separate
+API-6/API-7/API-8 authority boundaries and the single shipping Console. Do not
+reintroduce removed compatibility/place-holder files solely to satisfy stale
+historical references.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/HANDOVER_STATUS.md b/HANDOVER_STATUS.md
index a1256af..8251506 100644
--- a/HANDOVER_STATUS.md
+++ b/HANDOVER_STATUS.md
@@ -1,5 +1,34 @@
 # New-Chat Handover Status
 
+## Current canonical baseline — 2026-09-28
+
+The current source is the **Code Duplication Review and Consolidation** working
+baseline derived byte-for-byte from the Production Correctness & Control-Plane
+Hardening parent artifact (`90acc977db2998f4a3c1c4cacc07e19f0087f8bc033bce6b9934b83415957be8`)
+before the changes documented in `CODE_DUPLICATION_REVIEW.md`. Status remains
+**IMPLEMENTED_TESTING_DEFERRED**. No API-9 is defined.
+
+The review removed only source layers proven to be unwired, superseded or
+functionally duplicative, and consolidated the duplicated SecLang action/token
+parser into `internal/capability`. Distinct API-6/API-7/API-8 state machines,
+workers and authority boundaries remain separate. The shipping Console remains
+`static/admin.html` + `static/theme.css`; the previously removed experimental
+`web/` tree is not part of the current source.
+
+Current dependency-free evidence: API source gates
+**69/47/46/72/33/56/45/58/83/100/110/156/259/134/190 PASS**, code-duplication
+source gate **80/80 PASS with 139 unique Admin/update routes**, OpenAI source
+contract **16/16 PASS** plus isolated tests PASS, WAF package-source PASS,
+package-builder **9/9 PASS**, and root Go source-shape **95 files PASS**. An
+isolated dependency-free `internal/capability` test also passes. Canonical Go
+1.25 tidy/build/vet/test/race remains **BLOCKED_ENVIRONMENT / NOT_RUN** on this
+host; none of these static/source results promotes the product to TESTED or
+RELEASED.
+
+Dated sections below are retained as historical engineering/evidence records.
+When an older section conflicts with this section, `CODE_DUPLICATION_REVIEW.md`,
+`DOCUMENTATION_INDEX.md`, and the current source tree are authoritative.
+
 ## Build-integrity and L7 correction — 2026-09-23
 
 An external Go 1.25 CI audit of the previous API-1→API-3 delivery showed that archive integrity and isolated slice tests had not proved package buildability. The current working source repairs the package-level defects found in that audit and an additional duplicate symbol found during follow-up:
@@ -161,3 +190,13 @@ The documented API-security implementation roadmap is now complete through API-8
 
 Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.
 
+## 2026-09-28 — Current handover status
+
+Code Duplication Review and Consolidation is implemented with testing deferred.
+Confirmed superseded/unwired layers were removed; API authority separation and
+shipping Console behavior are unchanged. Source gate **80/80 PASS**, 139
+Admin/update routes are unique, root Go source shape is **95 files PASS**, and
+all prior API/source/package gates listed in the current canonical section remain
+PASS. Canonical Go 1.25 qualification remains blocked/not run.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/INSTALL.md b/INSTALL.md
index 34a821b..822142d 100644
--- a/INSTALL.md
+++ b/INSTALL.md
@@ -1,5 +1,34 @@
 # waf-proxy — install guide
 
+## Current canonical baseline — 2026-09-28
+
+The current source is the **Code Duplication Review and Consolidation** working
+baseline derived byte-for-byte from the Production Correctness & Control-Plane
+Hardening parent artifact (`90acc977db2998f4a3c1c4cacc07e19f0087f8bc033bce6b9934b83415957be8`)
+before the changes documented in `CODE_DUPLICATION_REVIEW.md`. Status remains
+**IMPLEMENTED_TESTING_DEFERRED**. No API-9 is defined.
+
+The review removed only source layers proven to be unwired, superseded or
+functionally duplicative, and consolidated the duplicated SecLang action/token
+parser into `internal/capability`. Distinct API-6/API-7/API-8 state machines,
+workers and authority boundaries remain separate. The shipping Console remains
+`static/admin.html` + `static/theme.css`; the previously removed experimental
+`web/` tree is not part of the current source.
+
+Current dependency-free evidence: API source gates
+**69/47/46/72/33/56/45/58/83/100/110/156/259/134/190 PASS**, code-duplication
+source gate **80/80 PASS with 139 unique Admin/update routes**, OpenAI source
+contract **16/16 PASS** plus isolated tests PASS, WAF package-source PASS,
+package-builder **9/9 PASS**, and root Go source-shape **95 files PASS**. An
+isolated dependency-free `internal/capability` test also passes. Canonical Go
+1.25 tidy/build/vet/test/race remains **BLOCKED_ENVIRONMENT / NOT_RUN** on this
+host; none of these static/source results promotes the product to TESTED or
+RELEASED.
+
+Dated sections below are retained as historical engineering/evidence records.
+When an older section conflicts with this section, `CODE_DUPLICATION_REVIEW.md`,
+`DOCUMENTATION_INDEX.md`, and the current source tree are authoritative.
+
 > **Documentation baseline — 2026-09-17.** This file documents a component or qualification path. Repository-wide release truth lives in [`DOCUMENTATION_INDEX.md`](DOCUMENTATION_INDEX.md), [`SOURCE_BASELINE_GATE_RESULT.md`](SOURCE_BASELINE_GATE_RESULT.md), and [`TESTING_RESULTS.md`](TESTING_RESULTS.md). Component PASS evidence must not be promoted into a root-build, runtime, package-lifecycle, clean-host, or release PASS outside its stated scope.
 
 Coraza-based reverse-proxy WAF with an embedded admin console. The console is compiled in (`go:embed`) with no CDN dependency. The portable Coraza-only build is pure Go; optional VectorScan acceleration uses CGO/libhs.
@@ -532,8 +561,8 @@ an IP you didn't expect = it's up, you're looking at the wrong address.
   failover is keepalived/VRRP or your LB reading `/healthz`.
 - **Fail-open when powered off requires bypass hardware.** Software can't do it;
   `WAF_WATCHDOG_DEVICE` only feeds/withholds a heartbeat for such hardware.
-- The `web/` Vite console is a **scaffold**, not a replacement; the shipping
-  console is the embedded single file.
+- The obsolete experimental `web/` Vite/Preact console has been removed. The
+  shipping and authoritative console is `static/admin.html` with `static/theme.css`.
 
 ## 11. Upgrading / replacing files
 
@@ -915,3 +944,13 @@ validation errors, backend health failures, filesystem ownership, stale systemd
 drop-ins, and on RHEL-family systems SELinux AVCs. Do not "fix" startup by
 disabling SELinux, weakening file permissions, bypassing TLS validation, or
 silently falling back from HSM to a filesystem key.\n\n## 14. OpenAI Responses API credentials\n\nFor native OpenAI, configure `provider=openai`, `api_style=responses`, and use a secret reference rather than putting the key in `config.json`:\n\n```json\n"api_key_ref": "env:OPENAI_API_KEY"\n```\n\nThe packaged systemd unit already reads `/etc/waf/waf-proxy.env` before dropping privileges, so an operator may add the variable to that root-owned `0600` file without exposing it to the admin API:\n\n```bash\nsudo sh -c 'printf "\\nOPENAI_API_KEY=%s\\n" "YOUR_KEY" >> /etc/waf/waf-proxy.env'\nsudo chown root:root /etc/waf/waf-proxy.env\nsudo chmod 0600 /etc/waf/waf-proxy.env\nsudo systemctl restart waf-proxy\n```\n\nFor file references, use a dedicated absolute regular file such as `/etc/waf/secrets/openai.key`, mode `0600`, with no symlinked path component. The admin API never returns the key or stored reference. Existing historical inline `api_key` configs remain migration-compatible but should be replaced with `api_key_ref`. Native Responses requires HTTPS; self-hosted/OpenAI-compatible endpoints that still implement Chat Completions should use `api_style=chat_completions`.\n
+
+## 2026-09-28 source-layout note
+
+The shipping UI has a single source: `static/admin.html` with
+`static/theme.css`. The old non-shipping `web/` migration scaffold is absent.
+The 2026-09-28 duplication consolidation does not change installation paths,
+persistent `/etc/waf`/state preservation requirements, or package lifecycle
+semantics.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/MANIFEST.md b/MANIFEST.md
index 8aa4afe..ec0abe5 100644
--- a/MANIFEST.md
+++ b/MANIFEST.md
@@ -1,5 +1,34 @@
 # waf-proxy — package manifest
 
+## Current canonical baseline — 2026-09-28
+
+The current source is the **Code Duplication Review and Consolidation** working
+baseline derived byte-for-byte from the Production Correctness & Control-Plane
+Hardening parent artifact (`90acc977db2998f4a3c1c4cacc07e19f0087f8bc033bce6b9934b83415957be8`)
+before the changes documented in `CODE_DUPLICATION_REVIEW.md`. Status remains
+**IMPLEMENTED_TESTING_DEFERRED**. No API-9 is defined.
+
+The review removed only source layers proven to be unwired, superseded or
+functionally duplicative, and consolidated the duplicated SecLang action/token
+parser into `internal/capability`. Distinct API-6/API-7/API-8 state machines,
+workers and authority boundaries remain separate. The shipping Console remains
+`static/admin.html` + `static/theme.css`; the previously removed experimental
+`web/` tree is not part of the current source.
+
+Current dependency-free evidence: API source gates
+**69/47/46/72/33/56/45/58/83/100/110/156/259/134/190 PASS**, code-duplication
+source gate **80/80 PASS with 139 unique Admin/update routes**, OpenAI source
+contract **16/16 PASS** plus isolated tests PASS, WAF package-source PASS,
+package-builder **9/9 PASS**, and root Go source-shape **95 files PASS**. An
+isolated dependency-free `internal/capability` test also passes. Canonical Go
+1.25 tidy/build/vet/test/race remains **BLOCKED_ENVIRONMENT / NOT_RUN** on this
+host; none of these static/source results promotes the product to TESTED or
+RELEASED.
+
+Dated sections below are retained as historical engineering/evidence records.
+When an older section conflicts with this section, `CODE_DUPLICATION_REVIEW.md`,
+`DOCUMENTATION_INDEX.md`, and the current source tree are authoritative.
+
 ## API Security checkpoint — 2026-09-23
 
 The uploaded API-3 baseline was audited against source rather than roadmap labels, then repaired/closed through API-3. Current source truth:
@@ -15,7 +44,7 @@ The uploaded API-3 baseline was audited against source rather than roadmap label
 | API-6.1 Sequence Foundation | `IMPLEMENTED_TESTING_DEFERRED` | `sequence_api61.go`, targeted tests and source gate; Go execution blocked by missing toolchain |
 | API-6.2 Workflow Learning | `IMPLEMENTED_TESTING_DEFERRED` | LEARN-only workflow model; source gate 56 PASS; Go 1.25 targeted/race BLOCKED_ENVIRONMENT/NOT_RUN |
 | API-6.3 Sequence Anomaly Detection | `IMPLEMENTED_TESTING_DEFERRED` | DETECT-only bounded anomaly evidence; source gate 45 PASS; Go 1.25 targeted/race `BLOCKED_ENVIRONMENT/NOT_RUN` |
-| API-6.4 Sequence Operations + Hardening | `PLANNED` | mode controls, exception CRUD, reset/relearn and full operations console not started |
+| API-6.4 Sequence Operations + Hardening | `IMPLEMENTED_TESTING_DEFERRED` | explicit LEARN/DETECT controls, exception CRUD, reset/relearn, recent sessions and operations Console; source gate 58 PASS |
 | API-7.1 Object Locator Discovery | `IMPLEMENTED_TESTING_DEFERRED` | normalized locator discovery + bounded keyed evidence; source gate 83 PASS; Go 1.25 targeted/race BLOCKED_ENVIRONMENT/NOT_RUN |
 | API-7.2–7.4 BOLA Relationship/Detection/Policy | `IMPLEMENTED_TESTING_DEFERRED` | verified identity/object relationship, bounded BOLA detection, REVIEW/SUPPRESS evidence policy/workflow/console; source gates 100/110/156 PASS |
 | API-8 GraphQL Security | `IMPLEMENTED_TESTING_DEFERRED` | bounded GraphQL parsing/normalization, SDL contract binding, depth/complexity/field/mutation/subscription/introspection/APQ controls, verified-variable API-7 bridge, staged deterministic LEARN/DETECT/ENFORCE; source gate 259 PASS; Go 1.25 targeted/race `BLOCKED_ENVIRONMENT/NOT_RUN` |
@@ -121,7 +150,7 @@ Go source (module `waf-proxy`; Coraza plus optional native VectorScan/libhs)
 - `api_operations.go` / `api_operations_test.go` — API-1 operation normalization, metadata, controls and deterministic tests.
 - `api_security_persist.go` — durable API Security JSON state persistence/autosave through API-6.1.
 - `api_security_handlers.go` — API-1 control and API-2 review endpoints.
-- `schema_api2.go` / `schema_api2_continuation.go` / `schema_api2_test.go` — live typed schema learning, lifecycle, privacy and tests.
+- `schema_api2.go` / `schema_api2_test.go` — live typed schema learning, lifecycle, privacy and tests. The unused `schema_api2_continuation.go` helper layer was removed during the 2026-09-28 duplication consolidation.
 - `ai_schema_review.go` — advisory-only OpenAI SchemaCandidate review.
 - `api3_contract.go` / `api3_export.go` / `api3_contract_test.go` — OpenAPI contract intelligence, export, diff/drift and tests.
 - `api_security_integration_test.go` — deterministic API-1 → API-2 → API-3 live-learning/drift test.
@@ -244,7 +273,7 @@ Phase 4 Slice C source additions:
 Phase 4 Slice D artifacts:
 - block_response.go
 - correlation.go
-- security_event.go
+- `security_event.go` — removed 2026-09-28; the unused generic event placeholder duplicated the wired security-specific evidence paths.
 
 
 ## Phase 4 Slice E — Persistent Security State
@@ -254,9 +283,8 @@ Status: IMPLEMENTED_TESTING_DEFERRED
 Implemented foundation: security state models and in-memory persistence abstraction. Production database durability, HA replication, retention tuning, and external integrations remain deferred.
 
 
-Phase 4 Slice F additions:
-- pki_hardening.go
-- pki_hardening_test.go
+Phase 4 Slice F historical additions (superseded):
+- `pki_hardening.go` / `pki_hardening_test.go` were isolated helper/test files and were removed 2026-09-28. The current CRL URL hardening authority is `pki.go` + `pki_url.go`.
 
 
 ## Phase 5 Slice A — Runtime Qualification Closure
@@ -429,8 +457,8 @@ Repair-relevant source:
 - `pki.go` / `pki_url.go` — matched CRL URL refresh/store implementation;
 - `pki_url_phase4_test.go` — URL/SSRF/LKG/concurrency/cache regression coverage;
 - `debug_bundle.go` — current evidence model plus TLS version naming helper;
-- `debug_evidence_ops_v2.go` — current store API/model mapping;
-- `debug_evidence_ops_v2_test.go` — integration regression tests.
+- `debug_bundle.go` / `debug_sanitize.go` — current bounded debug evidence store, export model and sanitization authority.
+- The superseded `debug_evidence_ops_v2.go` compatibility wrapper and its test were removed 2026-09-28.
 
 ## OpenAI connector hardening files
 
@@ -455,7 +483,7 @@ Local executable evidence for this checkpoint is captured in `TESTING_RESULTS.md
 ## API Security API-1 → API-3 closure files (2026-09-23)
 
 - `api_operations.go` / `api_operations_test.go` — durable API operation discovery, normalization, metadata, controls, and atomic-persistence coverage.
-- `schema_api2.go` / `schema_api2_continuation.go` / `schema_api2_test.go` — live typed schema learning, bounded aggregates, privacy/lifecycle logic, persistence evidence, and deterministic tests.
+- `schema_api2.go` / `schema_api2_test.go` — live typed schema learning, bounded aggregates, privacy/lifecycle logic, persistence evidence, and deterministic tests; the unused continuation helper was removed 2026-09-28.
 - `api_security_persist.go` — versioned API-1/API-2/API-3 state persistence with unique same-directory temp files, fsync, and atomic rename.
 - `api_security_handlers.go` / `api_security_handlers_test.go` — API-1/API-2 control-plane handlers and lifecycle checks.
 - `ai_schema_review.go` — advisory-only OpenAI SchemaCandidate review using existing Responses API Structured Outputs.
@@ -465,7 +493,7 @@ Local executable evidence for this checkpoint is captured in `TESTING_RESULTS.md
 - `tools/tests/test-api3-source.py` — 47-check API-3 source/wiring/security gate.
 - `API_SECURITY_CLOSURE_RESULT.md`, `TESTING_RESULTS.md`, `SOURCE_BASELINE_GATE_RESULT.md` — current source closure truth and qualification evidence.
 
-API-4 Positive Schema Enforcement is implemented with testing deferred; API-5 is implemented with testing deferred; API-6.1 through API-6.3 are implemented with testing deferred; API-6.4, API-7 and API-8 are not implemented in this package. Repository-root Go 1.25 qualification remains BLOCKED/NOT_RUN as documented; no release-readiness claim is made.
+API-1 through API-8 are implemented in source and remain `IMPLEMENTED_TESTING_DEFERRED`. API-6 learned sequence and API-7 inferred BOLA evidence remain non-enforcing; API-8 blocks only under explicit deterministic ENFORCE policy. Repository-root Go 1.25 qualification remains BLOCKED/NOT_RUN as documented; no release-readiness claim is made.
 
 ## API-4 Positive Schema Enforcement files
 
@@ -483,7 +511,7 @@ API-4 Positive Schema Enforcement is implemented with testing deferred; API-5 is
 - `admin.go` / `static/admin.html` — trusted issuer, JWKS refresh, operation policy, DETECT/ENFORCE and violation workflow.
 - `main.go` / `api_security_persist.go` — data-plane middleware, startup restore and autosave integration.
 
-API-1 through API-5 and API-6.1 through API-6.3 are `IMPLEMENTED_TESTING_DEFERRED`; API-6.4, API-7 and API-8 remain `PLANNED`. Canonical Go 1.25 real-dependency qualification remains `BLOCKED_ENVIRONMENT/NOT_RUN`.
+API-1 through API-8 are `IMPLEMENTED_TESTING_DEFERRED`; no API-9 is defined. Canonical Go 1.25 real-dependency qualification remains `BLOCKED_ENVIRONMENT/NOT_RUN`.
 
 - `sequence_api62.go` / `sequence_api62_test.go` — API-6.2 LEARN-only workflow cohorts, confidence/maturity, cold-start, absolute session lifetime, bounded entry/terminal/depth learning, v1->v2 durable migration and seven targeted tests.
 - `tools/tests/test-api62-source.py` — 56-check API-6.2 source/authority/privacy/bounded-state gate retained by build and CI.
@@ -563,3 +591,18 @@ Removed as non-runtime/misleading debt: `web/`, `investigation.go`, `security_ti
 
 Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.
 
+## 2026-09-28 — Duplication consolidation inventory delta
+
+Removed confirmed superseded/unwired source: legacy debug-evidence compatibility
+files, generic `security_state.go`/`security_event.go`, standalone
+`deployment_readiness.go`/`reliability.go`, root VectorScan placeholder
+qualification/audit files, `schema_api2_continuation.go`, the isolated
+`pki_hardening.go` helper pair, and legacy `internal/coverage/report.go`.
+
+Added `debug_sanitize.go`, `internal/capability/seclang.go` plus its test,
+`tools/tests/test-code-duplication-review-source.py`,
+`CODE_DUPLICATION_REVIEW.md`, and
+`CODE_DUPLICATION_REVIEW_SOURCE_GATE_RESULT.md`. Root Go source-shape is now
+**95 files** after intentional cleanup.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/MARKDOWN_REVIEW_2026-09-28.md b/MARKDOWN_REVIEW_2026-09-28.md
new file mode 100644
index 0000000..d3d3218
--- /dev/null
+++ b/MARKDOWN_REVIEW_2026-09-28.md
@@ -0,0 +1,117 @@
+# Markdown Review Ledger — 2026-09-28
+
+This ledger records the requested review of **every Markdown file** in the source
+tree after the Code Duplication Review and Consolidation. Current/canonical
+documents were synchronized to current source truth. Historical result,
+qualification and slice-evidence documents intentionally preserve their original
+dated observations; changing those old PASS/FAIL/NOT_RUN statements would
+falsify evidence. Repository-policy files were reviewed without altering their
+behavioral instructions.
+
+Current truth precedence is: current source tree → `DOCUMENTATION_INDEX.md` →
+`CODE_DUPLICATION_REVIEW.md` / current canonical root documents → dated
+historical evidence.
+
+| Markdown file | Classification | 2026-09-28 action |
+|---|---|---|
+| `AGENTS.md` | repository policy | Reviewed; policy/instruction content preserved. |
+| `AI_HANDOFF.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+| `API1_FINAL_QUALIFICATION_REPORT.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `API1_HARDENING_RESULT.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `API2_CHANGE_SUMMARY.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `API2_CONTINUATION_CHANGE_SUMMARY.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `API2_DOCUMENTATION_SYNC_CHANGE_SUMMARY.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `API2_FINAL_HANDOVER_STATUS.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `API2_IMPLEMENTATION_REPORT.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `API2_TESTING_STATUS.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `API2_TYPED_SCHEMA_IMPLEMENTATION.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `API3_ACCEPTANCE_CRITERIA.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `API3_OPENAPI_CONTRACT_IMPLEMENTATION.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `API3_SOURCE_GATE_RESULT.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `API3_TEST_MATRIX.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `API61_SEQUENCE_FOUNDATION.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `API61_SOURCE_GATE_RESULT.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `API62_SOURCE_GATE_RESULT.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `API62_WORKFLOW_LEARNING.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `API63_SEQUENCE_ANOMALY_DETECTION.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `API63_SOURCE_GATE_RESULT.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `API64_SEQUENCE_OPERATIONS_HARDENING.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `API64_SOURCE_GATE_RESULT.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `API71_OBJECT_LOCATOR_DISCOVERY.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `API71_SOURCE_GATE_RESULT.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `API72_IDENTITY_OBJECT_RELATIONSHIP.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `API72_SOURCE_GATE_RESULT.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `API73_BOLA_DETECTION.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `API73_SOURCE_GATE_RESULT.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `API74_BOLA_POLICY_EVIDENCE_CONSOLE.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `API74_SOURCE_GATE_RESULT.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `API8_GRAPHQL_SECURITY.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+| `API8_POST_AUDIT_HARDENING.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+| `API8_POST_AUDIT_HARDENING_SOURCE_GATE_RESULT.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `API8_SOURCE_GATE_RESULT.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `API_SECURITY_CLOSURE_RESULT.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+| `API_SECURITY_ROADMAP.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+| `API_SECURITY_ROADMAP_CHANGE.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `API_SECURITY_SLICE_IMPLEMENTATION_ROADMAP.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+| `CLAUDE.md` | repository policy | Reviewed; policy/instruction content preserved. |
+| `CLEAN_HOST_DISTRIBUTION_GATE_RESULT.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `CODE_DUPLICATION_REVIEW.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+| `CODE_DUPLICATION_REVIEW_SOURCE_GATE_RESULT.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+| `DEB_PACKAGE_GATE_RESULT.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `DEVELOPMENT.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+| `DEVELOPMENT_ROADMAP.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+| `DOCUMENTATION_INDEX.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+| `HANDOVER_PROMPT.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+| `HANDOVER_STATUS.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+| `INSTALL.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+| `MANIFEST.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+| `MARKDOWN_REVIEW_2026-09-28.md` | current/canonical | Created as the complete Markdown review ledger. |
+| `NEXT_CHAT_HANDOVER.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+| `OPENAI_INTEGRATION_GATE_RESULT.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `PACKAGE_LIFECYCLE_GATE_RESULT.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `PACKAGE_TOOL_GATE_RESULT.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `PACKAGING_TOOL.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+| `PRODUCTION_CONTROL_PLANE_HARDENING.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+| `PRODUCTION_CONTROL_PLANE_HARDENING_SOURCE_GATE_RESULT.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `README.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+| `RELEASE_PROCESS.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+| `RPM_PACKAGE_GATE_RESULT.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `SOURCE_BASELINE_GATE_RESULT.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `TESTING.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+| `TESTING_RESULTS.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+| `benchmark/README.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+| `handover/API_SECURITY_SLICE_ROADMAP.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `handover/CHANGE_SUMMARY.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `handover/NEW_CHAT_HANDOVER_PROMPT.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `handover/NEXT_CHAT_HANDOVER.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `handover/NEXT_CHAT_HANDOVER_API2.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `packaging/cleanhost/README.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+| `packaging/deb/CRS-PROVISIONING.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+| `packaging/deb/README.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+| `packaging/qualification/README.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+| `packaging/rpm/CRS-PROVISIONING.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+| `packaging/rpm/README.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+| `packaging/rpm/SELINUX.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+| `patch.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `qualification/API2_FINAL_QUALIFICATION_REPORT.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `qualification/API2_QUALIFICATION_STATUS.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `qualification/README.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+| `qualification/RELEASE_DECISION.md` | historical evidence | Reviewed; dated evidence preserved without rewriting past results. |
+| `qualification/corpus/README.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+| `qualification/hsm/README.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+| `qualification/performance/README.md` | current/canonical | Reviewed and synchronized where current source truth changed. |
+
+## Review outcome
+
+- Confirmed current/canonical docs no longer require the removed source layers
+  for runtime authority.
+- Stale current inventory claims in `MANIFEST.md`, `INSTALL.md`, API-8/current
+  roadmap material were corrected.
+- Historical references to deleted files or earlier PLANNED states remain only
+  where they describe the actual state at that historical checkpoint; the top
+  current-baseline sections and this ledger prevent them from being interpreted
+  as current truth.
+- No API-9 was introduced. Status remains `IMPLEMENTED_TESTING_DEFERRED`.
+- Canonical Go 1.25 qualification remains `BLOCKED_ENVIRONMENT / NOT_RUN`.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/NEXT_CHAT_HANDOVER.md b/NEXT_CHAT_HANDOVER.md
index 476bdaf..37cc79f 100644
--- a/NEXT_CHAT_HANDOVER.md
+++ b/NEXT_CHAT_HANDOVER.md
@@ -1,5 +1,34 @@
 # Next Chat Handover
 
+## Current canonical baseline — 2026-09-28
+
+The current source is the **Code Duplication Review and Consolidation** working
+baseline derived byte-for-byte from the Production Correctness & Control-Plane
+Hardening parent artifact (`90acc977db2998f4a3c1c4cacc07e19f0087f8bc033bce6b9934b83415957be8`)
+before the changes documented in `CODE_DUPLICATION_REVIEW.md`. Status remains
+**IMPLEMENTED_TESTING_DEFERRED**. No API-9 is defined.
+
+The review removed only source layers proven to be unwired, superseded or
+functionally duplicative, and consolidated the duplicated SecLang action/token
+parser into `internal/capability`. Distinct API-6/API-7/API-8 state machines,
+workers and authority boundaries remain separate. The shipping Console remains
+`static/admin.html` + `static/theme.css`; the previously removed experimental
+`web/` tree is not part of the current source.
+
+Current dependency-free evidence: API source gates
+**69/47/46/72/33/56/45/58/83/100/110/156/259/134/190 PASS**, code-duplication
+source gate **80/80 PASS with 139 unique Admin/update routes**, OpenAI source
+contract **16/16 PASS** plus isolated tests PASS, WAF package-source PASS,
+package-builder **9/9 PASS**, and root Go source-shape **95 files PASS**. An
+isolated dependency-free `internal/capability` test also passes. Canonical Go
+1.25 tidy/build/vet/test/race remains **BLOCKED_ENVIRONMENT / NOT_RUN** on this
+host; none of these static/source results promotes the product to TESTED or
+RELEASED.
+
+Dated sections below are retained as historical engineering/evidence records.
+When an older section conflicts with this section, `CODE_DUPLICATION_REVIEW.md`,
+`DOCUMENTATION_INDEX.md`, and the current source tree are authoritative.
+
 ## API Security checkpoint — 2026-09-23
 
 The uploaded API-3 baseline was audited against source rather than roadmap labels, then repaired/closed through API-3. Current source truth:
@@ -122,3 +151,14 @@ Continue from the post-audit hardening complete-source artifact, not the pre-har
 
 Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.
 
+## 2026-09-28 — Next-chat authority
+
+Use the Code Duplication Review and Consolidation artifact once its clean-extract
+gate is recorded. Do not restore deleted legacy debug/security-state/reliability/
+VectorScan/API-2/PKI/coverage helper layers without a new runtime requirement and
+explicit architecture decision. Treat `internal/capability` as the single
+SecLang tokenizer/action helper authority shared by coverage and VectorScan.
+No API-9 is defined; next product promotion work remains pinned Go 1.25 and live
+production qualification.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/OPENAI_INTEGRATION_GATE_RESULT.md b/OPENAI_INTEGRATION_GATE_RESULT.md
index a891a0d..d9fa261 100644
--- a/OPENAI_INTEGRATION_GATE_RESULT.md
+++ b/OPENAI_INTEGRATION_GATE_RESULT.md
@@ -84,3 +84,5 @@ Buildability Gate.
 - Baseline-relative source patch reconstruction: **PASS — 273/273 source files byte + Unix-mode identical**.
 - Final artifact negative mutations: **12/12 rejected PASS**.
 - Root Go 1.25 buildability remains **BLOCKED** on this Go 1.23.2 host.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/PACKAGE_LIFECYCLE_GATE_RESULT.md b/PACKAGE_LIFECYCLE_GATE_RESULT.md
index cb16051..9044121 100644
--- a/PACKAGE_LIFECYCLE_GATE_RESULT.md
+++ b/PACKAGE_LIFECYCLE_GATE_RESULT.md
@@ -59,3 +59,5 @@ This document is scoped package/distribution evidence. The 2026-09-17 audit of
 not yet executed on Go 1.25 in this environment. Therefore no PASS in this file
 may be interpreted as a current production WAF binary/package or overall release
 PASS. See `SOURCE_BASELINE_GATE_RESULT.md` and `DOCUMENTATION_INDEX.md`.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/PACKAGE_TOOL_GATE_RESULT.md b/PACKAGE_TOOL_GATE_RESULT.md
index e806171..ae94c82 100644
--- a/PACKAGE_TOOL_GATE_RESULT.md
+++ b/PACKAGE_TOOL_GATE_RESULT.md
@@ -64,3 +64,5 @@ This document is scoped package/distribution evidence. The 2026-09-17 audit of
 not yet executed on Go 1.25 in this environment. Therefore no PASS in this file
 may be interpreted as a current production WAF binary/package or overall release
 PASS. See `SOURCE_BASELINE_GATE_RESULT.md` and `DOCUMENTATION_INDEX.md`.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/PACKAGING_TOOL.md b/PACKAGING_TOOL.md
index 466db31..5f5790c 100644
--- a/PACKAGING_TOOL.md
+++ b/PACKAGING_TOOL.md
@@ -1,5 +1,34 @@
 # WAF Project Package Builder
 
+## Current canonical baseline — 2026-09-28
+
+The current source is the **Code Duplication Review and Consolidation** working
+baseline derived byte-for-byte from the Production Correctness & Control-Plane
+Hardening parent artifact (`90acc977db2998f4a3c1c4cacc07e19f0087f8bc033bce6b9934b83415957be8`)
+before the changes documented in `CODE_DUPLICATION_REVIEW.md`. Status remains
+**IMPLEMENTED_TESTING_DEFERRED**. No API-9 is defined.
+
+The review removed only source layers proven to be unwired, superseded or
+functionally duplicative, and consolidated the duplicated SecLang action/token
+parser into `internal/capability`. Distinct API-6/API-7/API-8 state machines,
+workers and authority boundaries remain separate. The shipping Console remains
+`static/admin.html` + `static/theme.css`; the previously removed experimental
+`web/` tree is not part of the current source.
+
+Current dependency-free evidence: API source gates
+**69/47/46/72/33/56/45/58/83/100/110/156/259/134/190 PASS**, code-duplication
+source gate **80/80 PASS with 139 unique Admin/update routes**, OpenAI source
+contract **16/16 PASS** plus isolated tests PASS, WAF package-source PASS,
+package-builder **9/9 PASS**, and root Go source-shape **95 files PASS**. An
+isolated dependency-free `internal/capability` test also passes. Canonical Go
+1.25 tidy/build/vet/test/race remains **BLOCKED_ENVIRONMENT / NOT_RUN** on this
+host; none of these static/source results promotes the product to TESTED or
+RELEASED.
+
+Dated sections below are retained as historical engineering/evidence records.
+When an older section conflicts with this section, `CODE_DUPLICATION_REVIEW.md`,
+`DOCUMENTATION_INDEX.md`, and the current source tree are authoritative.
+
 > **Documentation baseline — 2026-09-17.** This file documents a component or qualification path. Repository-wide release truth lives in [`DOCUMENTATION_INDEX.md`](DOCUMENTATION_INDEX.md), [`SOURCE_BASELINE_GATE_RESULT.md`](SOURCE_BASELINE_GATE_RESULT.md), and [`TESTING_RESULTS.md`](TESTING_RESULTS.md). Component PASS evidence must not be promoted into a root-build, runtime, package-lifecycle, clean-host, or release PASS outside its stated scope.
 
 ## Current buildability precondition
@@ -196,3 +225,12 @@ PACKAGE_BUILD_BLOCKED: <reason>
 A canonical build/test/package command failure also stops the run. The tool does
 not downgrade Go/Coraza, skip tests, substitute fixture binaries, or manufacture
 PASS evidence.
+
+## 2026-09-28 source-gate addition
+
+The canonical source now includes the code-duplication review gate. Package
+builds must preserve and execute it alongside the existing API, production
+hardening, OpenAI, package-source and artifact-integrity gates. The cleanup does
+not create a second packaging path.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/PRODUCTION_CONTROL_PLANE_HARDENING.md b/PRODUCTION_CONTROL_PLANE_HARDENING.md
index 7572a49..b3c2216 100644
--- a/PRODUCTION_CONTROL_PLANE_HARDENING.md
+++ b/PRODUCTION_CONTROL_PLANE_HARDENING.md
@@ -1,5 +1,34 @@
 # Production Correctness & Control-Plane Hardening
 
+## Current canonical baseline — 2026-09-28
+
+The current source is the **Code Duplication Review and Consolidation** working
+baseline derived byte-for-byte from the Production Correctness & Control-Plane
+Hardening parent artifact (`90acc977db2998f4a3c1c4cacc07e19f0087f8bc033bce6b9934b83415957be8`)
+before the changes documented in `CODE_DUPLICATION_REVIEW.md`. Status remains
+**IMPLEMENTED_TESTING_DEFERRED**. No API-9 is defined.
+
+The review removed only source layers proven to be unwired, superseded or
+functionally duplicative, and consolidated the duplicated SecLang action/token
+parser into `internal/capability`. Distinct API-6/API-7/API-8 state machines,
+workers and authority boundaries remain separate. The shipping Console remains
+`static/admin.html` + `static/theme.css`; the previously removed experimental
+`web/` tree is not part of the current source.
+
+Current dependency-free evidence: API source gates
+**69/47/46/72/33/56/45/58/83/100/110/156/259/134/190 PASS**, code-duplication
+source gate **80/80 PASS with 139 unique Admin/update routes**, OpenAI source
+contract **16/16 PASS** plus isolated tests PASS, WAF package-source PASS,
+package-builder **9/9 PASS**, and root Go source-shape **95 files PASS**. An
+isolated dependency-free `internal/capability` test also passes. Canonical Go
+1.25 tidy/build/vet/test/race remains **BLOCKED_ENVIRONMENT / NOT_RUN** on this
+host; none of these static/source results promotes the product to TESTED or
+RELEASED.
+
+Dated sections below are retained as historical engineering/evidence records.
+When an older section conflicts with this section, `CODE_DUPLICATION_REVIEW.md`,
+`DOCUMENTATION_INDEX.md`, and the current source tree are authoritative.
+
 Status: **IMPLEMENTED_TESTING_DEFERRED**  
 Date: 2026-09-25  
 Parent: `waf-proxy-api8-post-audit-hardening-2026-09-24.zip` (`e838ce6bd4f9cca6176cb809eb189978f312b194ae4b7c2f6f13f892a2d85f98`)
@@ -31,3 +60,14 @@ This hardening wave closes production correctness and control-plane findings dis
 Static/source gates and artifact gates can validate source structure, security contracts, syntax, packaging and clean extraction. Canonical Go 1.25 compile/test/race remains **BLOCKED_ENVIRONMENT / NOT_RUN** on this host because the installed Go is 1.23.2, `go.mod` requires 1.25.0, and external toolchain retrieval is unavailable. This baseline must not be promoted to TESTED or RELEASED on source-gate evidence alone.
 
 The supplied source archive has no `.git` metadata. Therefore branch creation, remote/main synchronization, commit identity and fresh-clone verification cannot be performed from this artifact. Parent ZIP SHA-256 and package manifests are the source identity for this handoff.
+
+## 2026-09-28 source-topology addendum
+
+The later Code Duplication Review and Consolidation removes superseded/unwired
+source layers and centralizes shared SecLang parsing. It does not roll back or
+replace the production-correctness behavior documented here. This file remains
+the authority for the 2026-09-25 control-plane hardening behavior;
+`CODE_DUPLICATION_REVIEW.md` is the authority for the later source-topology
+cleanup.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/PRODUCTION_CONTROL_PLANE_HARDENING_SOURCE_GATE_RESULT.md b/PRODUCTION_CONTROL_PLANE_HARDENING_SOURCE_GATE_RESULT.md
index b126bc1..6fb405b 100644
--- a/PRODUCTION_CONTROL_PLANE_HARDENING_SOURCE_GATE_RESULT.md
+++ b/PRODUCTION_CONTROL_PLANE_HARDENING_SOURCE_GATE_RESULT.md
@@ -9,3 +9,5 @@ Status: **PASS (source/static gate only)**
 The gate checks request-ID/block-page hardening, JSON privacy headers, webhook secret handling, debug privacy, login/session bounds, RBAC/audit, durable audit, dedicated HA replication and node-local preservation, atomic config persistence, synchronous listener ownership transitions including Go-TLS -> external frontend release, sitemap durability, state-version guards, bounded notification delivery, L7 saturation behavior, update/deployment constraints, systemd device declarations, Console capability mapping, test presence and build/CI wiring.
 
 This is not a substitute for canonical Go 1.25 build/test/race execution.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/README.md b/README.md
index f7f09a4..85c10cc 100644
--- a/README.md
+++ b/README.md
@@ -1,5 +1,34 @@
 # waf-proxy
 
+## Current canonical baseline — 2026-09-28
+
+The current source is the **Code Duplication Review and Consolidation** working
+baseline derived byte-for-byte from the Production Correctness & Control-Plane
+Hardening parent artifact (`90acc977db2998f4a3c1c4cacc07e19f0087f8bc033bce6b9934b83415957be8`)
+before the changes documented in `CODE_DUPLICATION_REVIEW.md`. Status remains
+**IMPLEMENTED_TESTING_DEFERRED**. No API-9 is defined.
+
+The review removed only source layers proven to be unwired, superseded or
+functionally duplicative, and consolidated the duplicated SecLang action/token
+parser into `internal/capability`. Distinct API-6/API-7/API-8 state machines,
+workers and authority boundaries remain separate. The shipping Console remains
+`static/admin.html` + `static/theme.css`; the previously removed experimental
+`web/` tree is not part of the current source.
+
+Current dependency-free evidence: API source gates
+**69/47/46/72/33/56/45/58/83/100/110/156/259/134/190 PASS**, code-duplication
+source gate **80/80 PASS with 139 unique Admin/update routes**, OpenAI source
+contract **16/16 PASS** plus isolated tests PASS, WAF package-source PASS,
+package-builder **9/9 PASS**, and root Go source-shape **95 files PASS**. An
+isolated dependency-free `internal/capability` test also passes. Canonical Go
+1.25 tidy/build/vet/test/race remains **BLOCKED_ENVIRONMENT / NOT_RUN** on this
+host; none of these static/source results promotes the product to TESTED or
+RELEASED.
+
+Dated sections below are retained as historical engineering/evidence records.
+When an older section conflicts with this section, `CODE_DUPLICATION_REVIEW.md`,
+`DOCUMENTATION_INDEX.md`, and the current source tree are authoritative.
+
 ## Build-integrity and L7 correction — 2026-09-23
 
 An external Go 1.25 CI audit of the previous API-1→API-3 delivery showed that archive integrity and isolated slice tests had not proved package buildability. The current working source repairs the package-level defects found in that audit and an additional duplicate symbol found during follow-up:
@@ -1325,3 +1354,20 @@ The current development baseline includes the post-audit hardening documented in
 
 Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.
 
+## Code Duplication Consolidation — 2026-09-28
+
+A whole-repository functional-overlap review removed only code proven to be
+unwired, superseded, or duplicated by a current implementation. The cleanup
+consolidates SecLang token/action parsing into `internal/capability`, removes
+legacy debug-evidence compatibility layers, unused generic security-state/event
+placeholders, stale deployment/reliability models, unused VectorScan
+qualification/audit placeholders, the unused API-2 continuation helper, the
+isolated legacy PKI hardening helper, and the superseded legacy coverage report.
+
+No API-security authority was merged. API-6 sequence learning/detection, API-7
+inferred BOLA evidence, and API-8 deterministic GraphQL enforcement keep their
+separate state, persistence and worker boundaries. See
+`CODE_DUPLICATION_REVIEW.md` and
+`CODE_DUPLICATION_REVIEW_SOURCE_GATE_RESULT.md`.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/RELEASE_PROCESS.md b/RELEASE_PROCESS.md
index 94ab924..f4b7d2e 100644
--- a/RELEASE_PROCESS.md
+++ b/RELEASE_PROCESS.md
@@ -1,5 +1,34 @@
 # Release Process
 
+## Current canonical baseline — 2026-09-28
+
+The current source is the **Code Duplication Review and Consolidation** working
+baseline derived byte-for-byte from the Production Correctness & Control-Plane
+Hardening parent artifact (`90acc977db2998f4a3c1c4cacc07e19f0087f8bc033bce6b9934b83415957be8`)
+before the changes documented in `CODE_DUPLICATION_REVIEW.md`. Status remains
+**IMPLEMENTED_TESTING_DEFERRED**. No API-9 is defined.
+
+The review removed only source layers proven to be unwired, superseded or
+functionally duplicative, and consolidated the duplicated SecLang action/token
+parser into `internal/capability`. Distinct API-6/API-7/API-8 state machines,
+workers and authority boundaries remain separate. The shipping Console remains
+`static/admin.html` + `static/theme.css`; the previously removed experimental
+`web/` tree is not part of the current source.
+
+Current dependency-free evidence: API source gates
+**69/47/46/72/33/56/45/58/83/100/110/156/259/134/190 PASS**, code-duplication
+source gate **80/80 PASS with 139 unique Admin/update routes**, OpenAI source
+contract **16/16 PASS** plus isolated tests PASS, WAF package-source PASS,
+package-builder **9/9 PASS**, and root Go source-shape **95 files PASS**. An
+isolated dependency-free `internal/capability` test also passes. Canonical Go
+1.25 tidy/build/vet/test/race remains **BLOCKED_ENVIRONMENT / NOT_RUN** on this
+host; none of these static/source results promotes the product to TESTED or
+RELEASED.
+
+Dated sections below are retained as historical engineering/evidence records.
+When an older section conflicts with this section, `CODE_DUPLICATION_REVIEW.md`,
+`DOCUMENTATION_INDEX.md`, and the current source tree are authoritative.
+
 
 ## Build-integrity and L7 correction — 2026-09-23
 
@@ -363,3 +392,13 @@ A release candidate containing this hardening must pass `tools/tests/test-api8-p
 
 Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.
 
+## 2026-09-28 — Additional release gate
+
+Release candidates derived from the duplication-consolidated source must run
+`tools/tests/test-code-duplication-review-source.py` on both source and clean
+extract. The gate proves removed superseded layers stay absent, replacement
+authorities remain wired, Admin routes remain unique, and API authority
+boundaries are unchanged. It is additive to, not a replacement for, pinned Go
+1.25 and target-host qualification.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/RPM_PACKAGE_GATE_RESULT.md b/RPM_PACKAGE_GATE_RESULT.md
index c53ae0e..71f9463 100644
--- a/RPM_PACKAGE_GATE_RESULT.md
+++ b/RPM_PACKAGE_GATE_RESULT.md
@@ -49,3 +49,5 @@ This document is scoped package/distribution evidence. The 2026-09-17 audit of
 not yet executed on Go 1.25 in this environment. Therefore no PASS in this file
 may be interpreted as a current production WAF binary/package or overall release
 PASS. See `SOURCE_BASELINE_GATE_RESULT.md` and `DOCUMENTATION_INDEX.md`.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/SOURCE_BASELINE_GATE_RESULT.md b/SOURCE_BASELINE_GATE_RESULT.md
index 0b6305a..f999f60 100644
--- a/SOURCE_BASELINE_GATE_RESULT.md
+++ b/SOURCE_BASELINE_GATE_RESULT.md
@@ -72,3 +72,5 @@ The current source now includes the OpenAI Responses/Structured-Outputs/secret-r
 The exact API-5 working source adds `identity_api5.go` / `identity_api5_test.go`, API/Admin UI wiring, durable `api-identity.json` state, and `tools/tests/test-api5-source.py`. Dependency-free/source evidence is PASS: root source-shape 97 Go files, API12 69 checks, API3 47 checks, API4 46 checks, API5 72 checks, admin inline JavaScript syntax, 17 deterministic API-5 test functions and targeted race under the external-dependency stub harness. The stub harness also passes whole-repository build, vet and test-binary compilation; this is compile-shape evidence only.
 
 The mandatory canonical gate was re-run after API-5 changes. On this host, `GOTOOLCHAIN=local go mod tidy -diff`, `CGO_ENABLED=0 go build ./...`, `go vet ./...`, `go test ./...`, and `CGO_ENABLED=1 go test -race ./...` all stop before compilation because `go.mod` requires Go 1.25.0 while the installed toolchain is Go 1.23.2. Automatic Go 1.25 acquisition also fails at `proxy.golang.org` DNS/network resolution. Classification remains **BLOCKED_ENVIRONMENT / NOT_RUN**, not PASS and not a source defect. API-1 through API-5 remain `IMPLEMENTED_TESTING_DEFERRED`; API-6 has not started.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/TESTING.md b/TESTING.md
index d7d9cd1..ea24dcb 100644
--- a/TESTING.md
+++ b/TESTING.md
@@ -1,5 +1,34 @@
 # Testing Policy
 
+## Current canonical baseline — 2026-09-28
+
+The current source is the **Code Duplication Review and Consolidation** working
+baseline derived byte-for-byte from the Production Correctness & Control-Plane
+Hardening parent artifact (`90acc977db2998f4a3c1c4cacc07e19f0087f8bc033bce6b9934b83415957be8`)
+before the changes documented in `CODE_DUPLICATION_REVIEW.md`. Status remains
+**IMPLEMENTED_TESTING_DEFERRED**. No API-9 is defined.
+
+The review removed only source layers proven to be unwired, superseded or
+functionally duplicative, and consolidated the duplicated SecLang action/token
+parser into `internal/capability`. Distinct API-6/API-7/API-8 state machines,
+workers and authority boundaries remain separate. The shipping Console remains
+`static/admin.html` + `static/theme.css`; the previously removed experimental
+`web/` tree is not part of the current source.
+
+Current dependency-free evidence: API source gates
+**69/47/46/72/33/56/45/58/83/100/110/156/259/134/190 PASS**, code-duplication
+source gate **80/80 PASS with 139 unique Admin/update routes**, OpenAI source
+contract **16/16 PASS** plus isolated tests PASS, WAF package-source PASS,
+package-builder **9/9 PASS**, and root Go source-shape **95 files PASS**. An
+isolated dependency-free `internal/capability` test also passes. Canonical Go
+1.25 tidy/build/vet/test/race remains **BLOCKED_ENVIRONMENT / NOT_RUN** on this
+host; none of these static/source results promotes the product to TESTED or
+RELEASED.
+
+Dated sections below are retained as historical engineering/evidence records.
+When an older section conflicts with this section, `CODE_DUPLICATION_REVIEW.md`,
+`DOCUMENTATION_INDEX.md`, and the current source tree are authoritative.
+
 ## Build-integrity and L7 correction — 2026-09-23
 
 An external Go 1.25 CI audit of the previous API-1→API-3 delivery showed that archive integrity and isolated slice tests had not proved package buildability. The current working source repairs the package-level defects found in that audit and an additional duplicate symbol found during follow-up:
@@ -413,3 +442,15 @@ Required dependency-free gates include all API-1 through API-8 source gates, `to
 
 Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.
 
+## Code Duplication Consolidation qualification — 2026-09-28
+
+Required dependency-free checks now include
+`tools/tests/test-code-duplication-review-source.py`. It must verify removal of
+confirmed superseded layers, retention of replacement authorities, shared
+SecLang parsing, route uniqueness, and unchanged API-6/API-7/API-8 authority
+boundaries. The gate must run in both source and clean-extract validation.
+
+Canonical Go 1.25 `tidy`, build, vet, full tests and race remain mandatory for
+release promotion and are not replaced by the duplication source gate.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/TESTING_RESULTS.md b/TESTING_RESULTS.md
index f0bfb06..a9cbb85 100644
--- a/TESTING_RESULTS.md
+++ b/TESTING_RESULTS.md
@@ -1,5 +1,34 @@
 # Testing Results and Qualification Ledger
 
+## Current canonical baseline — 2026-09-28
+
+The current source is the **Code Duplication Review and Consolidation** working
+baseline derived byte-for-byte from the Production Correctness & Control-Plane
+Hardening parent artifact (`90acc977db2998f4a3c1c4cacc07e19f0087f8bc033bce6b9934b83415957be8`)
+before the changes documented in `CODE_DUPLICATION_REVIEW.md`. Status remains
+**IMPLEMENTED_TESTING_DEFERRED**. No API-9 is defined.
+
+The review removed only source layers proven to be unwired, superseded or
+functionally duplicative, and consolidated the duplicated SecLang action/token
+parser into `internal/capability`. Distinct API-6/API-7/API-8 state machines,
+workers and authority boundaries remain separate. The shipping Console remains
+`static/admin.html` + `static/theme.css`; the previously removed experimental
+`web/` tree is not part of the current source.
+
+Current dependency-free evidence: API source gates
+**69/47/46/72/33/56/45/58/83/100/110/156/259/134/190 PASS**, code-duplication
+source gate **80/80 PASS with 139 unique Admin/update routes**, OpenAI source
+contract **16/16 PASS** plus isolated tests PASS, WAF package-source PASS,
+package-builder **9/9 PASS**, and root Go source-shape **95 files PASS**. An
+isolated dependency-free `internal/capability` test also passes. Canonical Go
+1.25 tidy/build/vet/test/race remains **BLOCKED_ENVIRONMENT / NOT_RUN** on this
+host; none of these static/source results promotes the product to TESTED or
+RELEASED.
+
+Dated sections below are retained as historical engineering/evidence records.
+When an older section conflicts with this section, `CODE_DUPLICATION_REVIEW.md`,
+`DOCUMENTATION_INDEX.md`, and the current source tree are authoritative.
+
 ## Build-integrity and L7 correction — 2026-09-23
 
 An external Go 1.25 CI audit of the previous API-1→API-3 delivery showed that archive integrity and isolated slice tests had not proved package buildability. The current working source repairs the package-level defects found in that audit and an additional duplicate symbol found during follow-up:
@@ -1108,3 +1137,15 @@ Current exact-source dependency-free results: API source gates 69/47/46/72/33/56
 
 Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.
 
+## 2026-09-28 — Code Duplication Review results
+
+Dependency-free exact-source results after consolidation: API gates
+**69/47/46/72/33/56/45/58/83/100/110/156/259/134/190 PASS**; duplication
+review gate **80/80 PASS** with **139 unique Admin/update routes**; OpenAI source
+contract **16/16 PASS** plus isolated tests PASS; WAF package-source PASS;
+package-builder **9/9 PASS**; root Go source-shape **95 files PASS**. The
+new shared `internal/capability` package also passes an isolated dependency-free
+unit test under the host Go 1.23 toolchain; this is advisory only. Canonical Go
+1.25 repository qualification remains `BLOCKED_ENVIRONMENT / NOT_RUN`.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/api3_contract.go b/api3_contract.go
index 31f566b..e022c1f 100644
--- a/api3_contract.go
+++ b/api3_contract.go
@@ -1266,16 +1266,6 @@ func hasEnumOutside(observed, declared []string) bool {
 	return false
 }
 
-func nonAnonymousAuthSamples(m map[string]int64) int64 {
-	var n int64
-	for k, v := range m {
-		if k != "none" && k != "" {
-			n += v
-		}
-	}
-	return n
-}
-
 func authRequirementDrift(schemes map[string]map[string]any, reqs []ContractSecurityRequirement, observed map[string]int64) bool {
 	type groupState struct {
 		required    map[string]bool
diff --git a/benchmark/README.md b/benchmark/README.md
index 41a1716..0af41bc 100644
--- a/benchmark/README.md
+++ b/benchmark/README.md
@@ -168,3 +168,5 @@ For the full proxy, combine `GOMAXPROCS`/CPU affinity on the waf-proxy process w
 ## Safety / test-environment boundary
 
 The hostile corpus is intended for a dedicated benchmark WAF/site and deterministic backend. Do not point this suite at a production application unless you explicitly intend to generate SQLi/XSS/traversal requests against it. The L4 mode uses ordinary TCP connections only; it does not spoof addresses or emit raw SYN floods.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/build.sh b/build.sh
index 48dd739..c3ac156 100755
--- a/build.sh
+++ b/build.sh
@@ -31,6 +31,8 @@ python3 ./tools/tests/test-api74-source.py
 python3 ./tools/tests/test-api8-source.py
 python3 ./tools/tests/test-api8-post-audit-hardening-source.py
 python3 ./tools/tests/test-production-control-plane-hardening-source.py
+python3 ./tools/tests/test-code-duplication-review-source.py
+python3 ./tools/tests/test-markdown-review-source.py
 
 VERSION="${VERSION:-$(date -u +%Y.%m.%d)}"
 COMMIT="${COMMIT:-unknown}"
diff --git a/clientip.go b/clientip.go
index 913d8ea..c67060d 100644
--- a/clientip.go
+++ b/clientip.go
@@ -6,7 +6,6 @@ import (
 	"net"
 	"net/http"
 	"strings"
-	"sync/atomic"
 )
 
 const (
@@ -20,34 +19,6 @@ const (
 // peer is trusted, then walked from right to left until the first untrusted
 // hop. This prevents an external client from choosing its own identity by
 // prepending an address to X-Forwarded-For.
-const (
-	AuditClientIdentityResolved       = "CLIENT_IDENTITY_RESOLVED"
-	AuditClientIdentityHeaderRejected = "CLIENT_IDENTITY_HEADER_REJECTED"
-)
-
-type ClientIdentityAuditEvent struct {
-	Action   string                 `json:"action"`
-	Decision ClientIdentityDecision `json:"decision"`
-}
-
-type ClientIdentityAuditSink func(ClientIdentityAuditEvent)
-
-var clientIdentityAuditSink atomic.Value
-
-func SetClientIdentityAuditSink(sink ClientIdentityAuditSink) {
-	clientIdentityAuditSink.Store(sink)
-}
-
-func emitClientIdentityAudit(event ClientIdentityAuditEvent) {
-	v := clientIdentityAuditSink.Load()
-	if v == nil {
-		return
-	}
-	if sink, ok := v.(ClientIdentityAuditSink); ok && sink != nil {
-		sink(event)
-	}
-}
-
 type ClientIdentityDecision struct {
 	RemoteAddr       string `json:"remote_addr"`
 	ResolvedClientIP string `json:"resolved_client_ip"`
@@ -124,14 +95,6 @@ func (r *clientIPResolver) isTrusted(ip net.IP) bool {
 	return false
 }
 
-func (d ClientIdentityDecision) AuditEvent() ClientIdentityAuditEvent {
-	action := AuditClientIdentityResolved
-	if d.Decision == "REJECTED" {
-		action = AuditClientIdentityHeaderRejected
-	}
-	return ClientIdentityAuditEvent{Action: action, Decision: d}
-}
-
 func (r *clientIPResolver) ResolveDecision(req *http.Request) ClientIdentityDecision {
 	d := ClientIdentityDecision{RemoteAddr: clientIP(req), ResolvedClientIP: clientIP(req), Source: "REMOTE_ADDR", Decision: "ACCEPTED"}
 	peer := net.ParseIP(d.RemoteAddr)
@@ -204,7 +167,6 @@ func (r *clientIPResolver) wrap(next http.Handler) http.Handler {
 			req.RemoteAddr = resolved
 		}
 		req = req.WithContext(context.WithValue(req.Context(), clientIdentityContextKey{}, decision))
-		emitClientIdentityAudit(decision.AuditEvent())
 		next.ServeHTTP(w, req)
 	})
 }
diff --git a/coraza_observer.go b/coraza_observer.go
index d6bf140..38b2988 100644
--- a/coraza_observer.go
+++ b/coraza_observer.go
@@ -105,9 +105,6 @@ func (t *observedCorazaTx) observe() {
 				})
 			}
 		}
-		if c := currentDebugEvidenceCapture(); c != nil {
-			c.Capture("coraza-tx", map[string]any{"matched_rules": ids, "source": "tx.MatchedRules-after-ProcessLogging"})
-		}
 	})
 }
 func (t *observedCorazaTx) ProcessLogging() { t.Transaction.ProcessLogging(); t.observe() }
diff --git a/debug_bundle.go b/debug_bundle.go
index 9ad824d..46fa4ef 100644
--- a/debug_bundle.go
+++ b/debug_bundle.go
@@ -321,53 +321,6 @@ func (s *DebugEvidenceStore) ExportIncident(tenant, txid string, out io.Writer)
 	return nil
 }
 
-// TenantExport is kept for compatibility with the earlier foundation API. It
-// exports the newest incident for exactly one tenant and never includes another
-// tenant's evidence.
-func (s *DebugEvidenceStore) TenantExport(tenant string, out io.Writer) error {
-	items := s.List(tenant, 1)
-	if len(items) == 0 {
-		return fmt.Errorf("no evidence for tenant %q", tenant)
-	}
-	return s.ExportIncident(tenant, items[0].TransactionID, out)
-}
-
-type debugRequestContext struct {
-	Tenant        string
-	TransactionID string
-	Started       time.Time
-}
-
-type debugRequestContextKey struct{}
-
-func debugContextFrom(ctx context.Context) (debugRequestContext, bool) {
-	v, ok := ctx.Value(debugRequestContextKey{}).(debugRequestContext)
-	return v, ok
-}
-
-func newDebugTransactionID() string {
-	var b [16]byte
-	if _, err := rand.Read(b[:]); err == nil {
-		return hex.EncodeToString(b[:])
-	}
-	return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
-}
-
-func tlsVersionName(version uint16) string {
-	switch version {
-	case tls.VersionTLS10:
-		return "TLS1.0"
-	case tls.VersionTLS11:
-		return "TLS1.1"
-	case tls.VersionTLS12:
-		return "TLS1.2"
-	case tls.VersionTLS13:
-		return "TLS1.3"
-	default:
-		return fmt.Sprintf("0x%04x", version)
-	}
-}
-
 func debugEvidenceWrap(store *DebugEvidenceStore, tenant string, next http.Handler) http.Handler {
 	if store == nil {
 		return next
diff --git a/debug_evidence.go b/debug_evidence.go
deleted file mode 100644
index 1abaee2..0000000
--- a/debug_evidence.go
+++ /dev/null
@@ -1,159 +0,0 @@
-package main
-
-import (
-	"archive/zip"
-	"crypto/sha256"
-	"encoding/hex"
-	"encoding/json"
-	"os"
-	"path/filepath"
-	"sort"
-	"strings"
-	"sync"
-	"sync/atomic"
-	"time"
-)
-
-// DebugEvidenceCapture is the compatibility wrapper retained for callers from
-// the first debug-evidence slice. New request-scoped capture uses
-// DebugEvidenceStore in debug_bundle.go.
-var activeDebugEvidenceCapture atomic.Pointer[DebugEvidenceCapture]
-
-func SetDebugEvidenceCapture(c *DebugEvidenceCapture)    { activeDebugEvidenceCapture.Store(c) }
-func currentDebugEvidenceCapture() *DebugEvidenceCapture { return activeDebugEvidenceCapture.Load() }
-
-type DebugEvidenceCapture struct {
-	mu      sync.RWMutex
-	enabled bool
-	max     int
-	ttl     time.Duration
-	items   map[string]map[string]any
-}
-
-func NewDebugEvidenceCapture(max int) *DebugEvidenceCapture {
-	if max < 1 {
-		max = 100
-	}
-	return &DebugEvidenceCapture{max: max, items: map[string]map[string]any{}}
-}
-func (d *DebugEvidenceCapture) Enable(v bool)          { d.mu.Lock(); d.enabled = v; d.mu.Unlock() }
-func (d *DebugEvidenceCapture) SetTTL(v time.Duration) { d.mu.Lock(); d.ttl = v; d.mu.Unlock() }
-
-func isSensitiveDebugKey(k string) bool {
-	k = strings.ToLower(strings.TrimSpace(k))
-	for _, part := range []string{"authorization", "proxy-authorization", "cookie", "set-cookie", "password", "passwd", "secret", "api_key", "apikey", "token", "private_key", "peer_token"} {
-		if strings.Contains(k, part) {
-			return true
-		}
-	}
-	return false
-}
-
-func sanitizeDebugValue(key string, v any) any {
-	if isSensitiveDebugKey(key) {
-		return "[masked]"
-	}
-	switch x := v.(type) {
-	case string:
-		return truncateDebugString(x, 4096)
-	case []string:
-		out := make([]string, len(x))
-		for i := range x {
-			out[i] = truncateDebugString(x[i], 4096)
-		}
-		return out
-	case map[string]any:
-		return sanitizeDebugMap(x)
-	case map[string]string:
-		out := make(map[string]any, len(x))
-		for k, v := range x {
-			out[k] = sanitizeDebugValue(k, v)
-		}
-		return out
-	case []any:
-		out := make([]any, len(x))
-		for i := range x {
-			out[i] = sanitizeDebugValue("", x[i])
-		}
-		return out
-	default:
-		return v
-	}
-}
-
-func sanitizeDebugMap(in map[string]any) map[string]any {
-	if in == nil {
-		return nil
-	}
-	out := make(map[string]any, len(in))
-	for k, v := range in {
-		out[k] = sanitizeDebugValue(k, v)
-	}
-	return out
-}
-
-func (d *DebugEvidenceCapture) Capture(id string, evidence map[string]any) {
-	d.mu.Lock()
-	defer d.mu.Unlock()
-	if !d.enabled || id == "" {
-		return
-	}
-	now := time.Now().UTC()
-	if d.ttl > 0 {
-		for k, v := range d.items {
-			if raw, ok := v["captured_unix_nano"].(int64); ok && now.Sub(time.Unix(0, raw)) >= d.ttl {
-				delete(d.items, k)
-			}
-		}
-	}
-	if len(d.items) >= d.max {
-		return
-	}
-	e := sanitizeDebugMap(evidence)
-	e["captured_at"] = now.Format(time.RFC3339Nano)
-	e["captured_unix_nano"] = now.UnixNano()
-	d.items[id] = e
-}
-
-func (d *DebugEvidenceCapture) Export(path string) error {
-	d.mu.RLock()
-	defer d.mu.RUnlock()
-	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
-		return err
-	}
-	f, err := os.Create(path)
-	if err != nil {
-		return err
-	}
-	defer f.Close()
-	z := zip.NewWriter(f)
-	defer z.Close()
-	b, err := json.MarshalIndent(map[string]any{"files": len(d.items), "evidence": d.items}, "", "  ")
-	if err != nil {
-		return err
-	}
-	w, err := z.Create("debug-evidence.json")
-	if err != nil {
-		return err
-	}
-	_, err = w.Write(b)
-	return err
-}
-
-func HashEvidence(path string) (string, error) {
-	b, err := os.ReadFile(path)
-	if err != nil {
-		return "", err
-	}
-	h := sha256.Sum256(b)
-	return hex.EncodeToString(h[:]), nil
-}
-
-func sortedDebugKeys(m map[string]any) []string {
-	keys := make([]string, 0, len(m))
-	for k := range m {
-		keys = append(keys, k)
-	}
-	sort.Strings(keys)
-	return keys
-}
diff --git a/debug_evidence_ops_v2.go b/debug_evidence_ops_v2.go
deleted file mode 100644
index 3eb58ee..0000000
--- a/debug_evidence_ops_v2.go
+++ /dev/null
@@ -1,35 +0,0 @@
-package main
-
-import (
-	"sort"
-	"time"
-)
-
-// DebugEvidenceOpsV2 provides bounded operator-facing operations.
-// Storage backends can implement persistence without changing WAF dataplane.
-type DebugEvidenceOpsV2 struct {
-	Store *DebugEvidenceStore
-}
-
-type DebugEvidenceSummary struct {
-	ID        string    `json:"id"`
-	Tenant    string    `json:"tenant"`
-	CreatedAt time.Time `json:"created_at"`
-	Size      int64     `json:"size"`
-}
-
-func (o DebugEvidenceOpsV2) List(tenant string) []DebugEvidenceSummary {
-	if o.Store == nil {
-		return nil
-	}
-	out := []DebugEvidenceSummary{}
-	for _, e := range o.Store.List(tenant, 500) {
-		out = append(out, DebugEvidenceSummary{
-			ID:        e.TransactionID,
-			Tenant:    e.Tenant,
-			CreatedAt: e.CapturedAt,
-		})
-	}
-	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
-	return out
-}
diff --git a/debug_evidence_ops_v2_test.go b/debug_evidence_ops_v2_test.go
deleted file mode 100644
index d4d3ed1..0000000
--- a/debug_evidence_ops_v2_test.go
+++ /dev/null
@@ -1,38 +0,0 @@
-package main
-
-import (
-	"testing"
-	"time"
-)
-
-func TestDebugEvidenceOpsV2ListUsesCurrentBundleModel(t *testing.T) {
-	store := NewDebugEvidenceStore(10, time.Hour)
-	store.Put(DebugBundle{Tenant: "site-a", TransactionID: "tx-a"})
-	store.Put(DebugBundle{Tenant: "site-b", TransactionID: "tx-b"})
-
-	got := (DebugEvidenceOpsV2{Store: store}).List("site-a")
-	if len(got) != 1 {
-		t.Fatalf("List(site-a) returned %d entries, want 1", len(got))
-	}
-	if got[0].ID != "tx-a" || got[0].Tenant != "site-a" || got[0].CreatedAt.IsZero() {
-		t.Fatalf("unexpected summary: %#v", got[0])
-	}
-}
-
-func TestTLSVersionName(t *testing.T) {
-	tests := []struct {
-		version uint16
-		want    string
-	}{
-		{0x0301, "TLS1.0"},
-		{0x0302, "TLS1.1"},
-		{0x0303, "TLS1.2"},
-		{0x0304, "TLS1.3"},
-		{0x9999, "0x9999"},
-	}
-	for _, tc := range tests {
-		if got := tlsVersionName(tc.version); got != tc.want {
-			t.Fatalf("tlsVersionName(%#x)=%q, want %q", tc.version, got, tc.want)
-		}
-	}
-}
diff --git a/debug_evidence_test.go b/debug_evidence_test.go
deleted file mode 100644
index 7344cc2..0000000
--- a/debug_evidence_test.go
+++ /dev/null
@@ -1,14 +0,0 @@
-package main
-
-import (
- "testing"
- "time"
-)
-
-func TestDebugEvidenceCaptureMaskTTLFoundation(t *testing.T){
- d:=NewDebugEvidenceCapture(2)
- d.Enable(true)
- d.SetTTL(time.Minute)
- d.Capture("tx1", map[string]any{"authorization":"secret","rule":"942100"})
- if err:=d.Export(t.TempDir()+"/bundle.zip"); err!=nil { t.Fatal(err) }
-}
diff --git a/debug_sanitize.go b/debug_sanitize.go
new file mode 100644
index 0000000..353e534
--- /dev/null
+++ b/debug_sanitize.go
@@ -0,0 +1,56 @@
+package main
+
+import "strings"
+
+func isSensitiveDebugKey(k string) bool {
+	k = strings.ToLower(strings.TrimSpace(k))
+	for _, part := range []string{"authorization", "proxy-authorization", "cookie", "set-cookie", "password", "passwd", "secret", "api_key", "apikey", "token", "private_key", "peer_token"} {
+		if strings.Contains(k, part) {
+			return true
+		}
+	}
+	return false
+}
+
+func sanitizeDebugValue(key string, v any) any {
+	if isSensitiveDebugKey(key) {
+		return "[masked]"
+	}
+	switch x := v.(type) {
+	case string:
+		return truncateDebugString(x, 4096)
+	case []string:
+		out := make([]string, len(x))
+		for i := range x {
+			out[i] = truncateDebugString(x[i], 4096)
+		}
+		return out
+	case map[string]any:
+		return sanitizeDebugMap(x)
+	case map[string]string:
+		out := make(map[string]any, len(x))
+		for k, v := range x {
+			out[k] = sanitizeDebugValue(k, v)
+		}
+		return out
+	case []any:
+		out := make([]any, len(x))
+		for i := range x {
+			out[i] = sanitizeDebugValue("", x[i])
+		}
+		return out
+	default:
+		return v
+	}
+}
+
+func sanitizeDebugMap(in map[string]any) map[string]any {
+	if in == nil {
+		return nil
+	}
+	out := make(map[string]any, len(in))
+	for k, v := range in {
+		out[k] = sanitizeDebugValue(k, v)
+	}
+	return out
+}
diff --git a/deployment_readiness.go b/deployment_readiness.go
deleted file mode 100644
index 5f969ff..0000000
--- a/deployment_readiness.go
+++ /dev/null
@@ -1,67 +0,0 @@
-package main
-
-import (
-	"crypto/x509"
-	"encoding/json"
-	"os"
-	"time"
-)
-
-// DeploymentReadinessCheck represents a deployment preflight result.
-type DeploymentReadinessCheck struct {
-	Name   string `json:"name"`
-	Status string `json:"status"`
-	Detail string `json:"detail"`
-}
-
-// DeploymentReadinessReport is the enterprise deployment readiness evidence model.
-type DeploymentReadinessReport struct {
-	GeneratedAt time.Time                  `json:"generated_at"`
-	Status      string                     `json:"status"`
-	Checks      []DeploymentReadinessCheck `json:"checks"`
-}
-
-func RunDeploymentPreflight(certPath string) DeploymentReadinessReport {
-	report := DeploymentReadinessReport{GeneratedAt: time.Now().UTC(), Status: "PASS"}
-
-	report.Checks = append(report.Checks, DeploymentReadinessCheck{
-		Name:   "configuration",
-		Status: "PASS",
-		Detail: "configuration validation boundary available",
-	})
-
-	if certPath == "" {
-		report.Checks = append(report.Checks, DeploymentReadinessCheck{
-			Name:   "tls_certificate",
-			Status: "WARN",
-			Detail: "no certificate path supplied",
-		})
-		report.Status = "WARN"
-		return report
-	}
-
-	b, err := os.ReadFile(certPath)
-	if err != nil {
-		report.Checks = append(report.Checks, DeploymentReadinessCheck{
-			Name:   "tls_certificate",
-			Status: "FAIL",
-			Detail: err.Error(),
-		})
-		report.Status = "FAIL"
-		return report
-	}
-
-	if _, err := x509.ParseCertificate(b); err != nil {
-		report.Checks = append(report.Checks, DeploymentReadinessCheck{
-			Name:   "tls_certificate",
-			Status: "WARN",
-			Detail: "certificate parsing requires decoded DER/PEM handling path",
-		})
-		report.Status = "WARN"
-	}
-	return report
-}
-
-func EncodeDeploymentReadinessReport(report DeploymentReadinessReport) ([]byte, error) {
-	return json.MarshalIndent(report, "", "  ")
-}
diff --git a/handover/API_SECURITY_SLICE_ROADMAP.md b/handover/API_SECURITY_SLICE_ROADMAP.md
index 12f11a1..512a2e6 100644
--- a/handover/API_SECURITY_SLICE_ROADMAP.md
+++ b/handover/API_SECURITY_SLICE_ROADMAP.md
@@ -43,3 +43,5 @@ PLANNED
 
 ### API-8 GraphQL Security
 PLANNED
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/handover/CHANGE_SUMMARY.md b/handover/CHANGE_SUMMARY.md
index 515ce14..7912c24 100644
--- a/handover/CHANGE_SUMMARY.md
+++ b/handover/CHANGE_SUMMARY.md
@@ -11,3 +11,5 @@ Added handover package documentation:
 No new feature implementation was added in this handover packaging step.
 
 Qualification status remains deferred because runtime qualification was skipped.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/handover/NEW_CHAT_HANDOVER_PROMPT.md b/handover/NEW_CHAT_HANDOVER_PROMPT.md
index 7019862..2859304 100644
--- a/handover/NEW_CHAT_HANDOVER_PROMPT.md
+++ b/handover/NEW_CHAT_HANDOVER_PROMPT.md
@@ -19,3 +19,5 @@ Next planned implementation:
 API-2 Typed Schema Learning
 
 Before starting API-2, review API-1 data model and architecture boundaries.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/handover/NEXT_CHAT_HANDOVER.md b/handover/NEXT_CHAT_HANDOVER.md
index 949bb57..07ff80e 100644
--- a/handover/NEXT_CHAT_HANDOVER.md
+++ b/handover/NEXT_CHAT_HANDOVER.md
@@ -24,3 +24,5 @@ Do not claim:
 
 Next focus:
 complete API-2 integration and qualification preparation.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/handover/NEXT_CHAT_HANDOVER_API2.md b/handover/NEXT_CHAT_HANDOVER_API2.md
index dff88cb..a3d31f1 100644
--- a/handover/NEXT_CHAT_HANDOVER_API2.md
+++ b/handover/NEXT_CHAT_HANDOVER_API2.md
@@ -14,3 +14,5 @@ Continue with:
 3. final release readiness decision
 
 Do not claim TESTED/RELEASED without execution evidence.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/internal/capability/seclang.go b/internal/capability/seclang.go
new file mode 100644
index 0000000..a99113f
--- /dev/null
+++ b/internal/capability/seclang.go
@@ -0,0 +1,93 @@
+package capability
+
+import "strings"
+
+// NextToken returns the next SecLang token and the remaining input. Quoted
+// tokens preserve backslash escapes exactly so callers can make conservative
+// semantic decisions without inventing a second tokenizer.
+func NextToken(s string) (string, string, bool) {
+	if s == "" {
+		return "", "", false
+	}
+	if s[0] == '"' || s[0] == '\'' {
+		q := s[0]
+		var b strings.Builder
+		escaped := false
+		for i := 1; i < len(s); i++ {
+			c := s[i]
+			if escaped {
+				b.WriteByte(c)
+				escaped = false
+				continue
+			}
+			if c == '\\' {
+				escaped = true
+				b.WriteByte(c)
+				continue
+			}
+			if c == q {
+				return b.String(), s[i+1:], true
+			}
+			b.WriteByte(c)
+		}
+		return "", "", false
+	}
+	if i := strings.IndexAny(s, " \t\r\n"); i >= 0 {
+		return s[:i], s[i:], true
+	}
+	return s, "", true
+}
+
+// SplitActions separates a SecLang action list without splitting commas inside
+// quoted values. It is shared by offline CRS coverage analysis and the live
+// VectorScan rule classifier so both paths classify the same source text.
+func SplitActions(s string) []string {
+	var out []string
+	start := 0
+	quote := byte(0)
+	escaped := false
+	for i := 0; i < len(s); i++ {
+		c := s[i]
+		if escaped {
+			escaped = false
+			continue
+		}
+		if c == '\\' {
+			escaped = true
+			continue
+		}
+		if quote != 0 {
+			if c == quote {
+				quote = 0
+			}
+			continue
+		}
+		if c == '\'' || c == '"' {
+			quote = c
+			continue
+		}
+		if c == ',' {
+			out = append(out, s[start:i])
+			start = i + 1
+		}
+	}
+	return append(out, s[start:])
+}
+
+// SplitAction separates an action name from its optional value.
+func SplitAction(s string) (string, string) {
+	if i := strings.IndexByte(s, ':'); i >= 0 {
+		return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1:])
+	}
+	return strings.TrimSpace(s), ""
+}
+
+// TrimActionValue removes one matching quote pair after trimming surrounding
+// whitespace. It deliberately does not unescape or reinterpret content.
+func TrimActionValue(s string) string {
+	s = strings.TrimSpace(s)
+	if len(s) >= 2 && ((s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '"' && s[len(s)-1] == '"')) {
+		s = s[1 : len(s)-1]
+	}
+	return strings.TrimSpace(s)
+}
diff --git a/internal/capability/seclang_test.go b/internal/capability/seclang_test.go
new file mode 100644
index 0000000..1034256
--- /dev/null
+++ b/internal/capability/seclang_test.go
@@ -0,0 +1,22 @@
+package capability
+
+import "testing"
+
+func TestSecLangTokenAndActionHelpers(t *testing.T) {
+	tok, rest, ok := NextToken(`"ARGS:id" "@rx ^[0-9]+$" "id:1,phase:2,t:none,msg:'a,b'"`)
+	if !ok || tok != "ARGS:id" || rest == "" {
+		t.Fatalf("unexpected first token: %q %q %v", tok, rest, ok)
+	}
+	actions := SplitActions(`id:1,phase:2,msg:'a,b',t:"lowercase"`)
+	if len(actions) != 4 {
+		t.Fatalf("actions=%#v", actions)
+	}
+	name, value := SplitAction(actions[2])
+	if name != "msg" || TrimActionValue(value) != "a,b" {
+		t.Fatalf("action=%q value=%q", name, value)
+	}
+	name, value = SplitAction("chain")
+	if name != "chain" || value != "" {
+		t.Fatalf("flag action=%q value=%q", name, value)
+	}
+}
diff --git a/internal/coverage/capability.go b/internal/coverage/capability.go
index f6cc2ba..2c07915 100644
--- a/internal/coverage/capability.go
+++ b/internal/coverage/capability.go
@@ -1,10 +1,6 @@
 package coverage
 
-import (
-	"strings"
-
-	shared "waf-proxy/internal/capability"
-)
+import shared "waf-proxy/internal/capability"
 
 // RuleCapability describes whether a rule can be safely considered for the
 // current runtime adapter. The eligibility decision itself lives in
@@ -22,21 +18,6 @@ type RuleCapability struct {
 	Reason     string   `json:"reason,omitempty"`
 }
 
-type TransformCapability struct {
-	Name      string `json:"name"`
-	Supported bool   `json:"supported"`
-	Exact     bool   `json:"exact"`
-	Reason    string `json:"reason,omitempty"`
-}
-
-var transforms = map[string]TransformCapability{
-	"none":             {Name: "none", Supported: true, Exact: true},
-	"lowercase":        {Name: "lowercase", Supported: true, Exact: true},
-	"uppercase":        {Name: "uppercase", Supported: false, Exact: false, Reason: "runtime adapter does not reproduce uppercase yet"},
-	"urldecode":        {Name: "urldecode", Supported: false, Exact: false, Reason: "requires exact Coraza decoding semantics"},
-	"removewhitespace": {Name: "removewhitespace", Supported: false, Exact: false, Reason: "requires exact Coraza normalization semantics"},
-}
-
 func EvaluateRule(rule RuleCapability) RuleCapability {
 	result := shared.Classify(shared.Input{
 		RuleID:     rule.RuleID,
@@ -55,12 +36,3 @@ func EvaluateRule(rule RuleCapability) RuleCapability {
 	}
 	return rule
 }
-
-func TransformCapabilities() []TransformCapability {
-	order := []string{"none", "lowercase", "uppercase", "urldecode", "removewhitespace"}
-	out := make([]TransformCapability, 0, len(order))
-	for _, name := range order {
-		out = append(out, transforms[strings.ToLower(name)])
-	}
-	return out
-}
diff --git a/internal/coverage/crs/parser.go b/internal/coverage/crs/parser.go
index 8664891..67c379e 100644
--- a/internal/coverage/crs/parser.go
+++ b/internal/coverage/crs/parser.go
@@ -3,6 +3,8 @@ package crs
 import (
 	"strconv"
 	"strings"
+
+	shared "waf-proxy/internal/capability"
 )
 
 // ParseRule preserves the Slice B API for focused callers.
@@ -19,15 +21,15 @@ func ParseRuleAt(statement, file string, line int) Rule {
 		return r
 	}
 	s = strings.TrimSpace(s[len("SecRule"):])
-	targets, rest, ok := nextToken(s)
+	targets, rest, ok := shared.NextToken(s)
 	if !ok {
 		return r
 	}
-	op, rest, ok := nextToken(strings.TrimSpace(rest))
+	op, rest, ok := shared.NextToken(strings.TrimSpace(rest))
 	if !ok {
 		return r
 	}
-	actions, _, ok := nextToken(strings.TrimSpace(rest))
+	actions, _, ok := shared.NextToken(strings.TrimSpace(rest))
 	if !ok {
 		return r
 	}
@@ -55,115 +57,34 @@ func ParseRuleAt(statement, file string, line int) Rule {
 		}
 	}
 
-	for _, rawAction := range splitActions(actions) {
+	for _, rawAction := range shared.SplitActions(actions) {
 		a := strings.TrimSpace(rawAction)
 		if a == "" {
 			continue
 		}
 		r.Actions = append(r.Actions, a)
-		name, value := splitAction(a)
+		name, value := shared.SplitAction(a)
 		switch strings.ToLower(name) {
 		case "id":
-			r.ID = trimActionValue(value)
+			r.ID = shared.TrimActionValue(value)
 		case "phase":
-			if n, err := strconv.Atoi(trimActionValue(value)); err == nil {
+			if n, err := strconv.Atoi(shared.TrimActionValue(value)); err == nil {
 				r.Phase = n
 			}
 		case "t":
-			v := strings.ToLower(trimActionValue(value))
+			v := strings.ToLower(shared.TrimActionValue(value))
 			if v != "" {
 				r.Transforms = append(r.Transforms, v)
 			}
 		case "tag":
-			if v := trimActionValue(value); v != "" {
+			if v := shared.TrimActionValue(value); v != "" {
 				r.Tags = append(r.Tags, v)
 			}
 		case "severity":
-			r.Severity = trimActionValue(value)
+			r.Severity = shared.TrimActionValue(value)
 		case "chain":
 			r.HasChain = true
 		}
 	}
 	return r
 }
-
-func nextToken(s string) (string, string, bool) {
-	if s == "" {
-		return "", "", false
-	}
-	if s[0] == '"' || s[0] == '\'' {
-		q := s[0]
-		var b strings.Builder
-		escaped := false
-		for i := 1; i < len(s); i++ {
-			c := s[i]
-			if escaped {
-				b.WriteByte(c)
-				escaped = false
-				continue
-			}
-			if c == '\\' {
-				escaped = true
-				b.WriteByte(c)
-				continue
-			}
-			if c == q {
-				return b.String(), s[i+1:], true
-			}
-			b.WriteByte(c)
-		}
-		return "", "", false
-	}
-	if i := strings.IndexAny(s, " \t\r\n"); i >= 0 {
-		return s[:i], s[i:], true
-	}
-	return s, "", true
-}
-
-func splitActions(s string) []string {
-	var out []string
-	start := 0
-	quote := byte(0)
-	escaped := false
-	for i := 0; i < len(s); i++ {
-		c := s[i]
-		if escaped {
-			escaped = false
-			continue
-		}
-		if c == '\\' {
-			escaped = true
-			continue
-		}
-		if quote != 0 {
-			if c == quote {
-				quote = 0
-			}
-			continue
-		}
-		if c == '\'' || c == '"' {
-			quote = c
-			continue
-		}
-		if c == ',' {
-			out = append(out, s[start:i])
-			start = i + 1
-		}
-	}
-	return append(out, s[start:])
-}
-
-func splitAction(s string) (string, string) {
-	if i := strings.IndexByte(s, ':'); i >= 0 {
-		return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1:])
-	}
-	return strings.TrimSpace(s), ""
-}
-
-func trimActionValue(s string) string {
-	s = strings.TrimSpace(s)
-	if len(s) >= 2 && ((s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '"' && s[len(s)-1] == '"')) {
-		s = s[1 : len(s)-1]
-	}
-	return strings.TrimSpace(s)
-}
diff --git a/internal/coverage/report.go b/internal/coverage/report.go
deleted file mode 100644
index 569311a..0000000
--- a/internal/coverage/report.go
+++ /dev/null
@@ -1,23 +0,0 @@
-package coverage
-
-// CoverageReport summarizes conservative acceleration coverage. It is retained
-// for callers from Slice A; Slice C's ruleset report lives in coverage/crs.
-type CoverageReport struct {
-	EligibleRules    int `json:"eligible_rules"`
-	AcceleratedRules int `json:"accelerated_rules"`
-	CorazaOnlyRules  int `json:"coraza_only_rules"`
-	UnsafeRules      int `json:"unsafe_rules"`
-}
-
-// BuildReport creates a report from evaluated rule capabilities.
-func BuildReport(rules []RuleCapability) CoverageReport {
-	var r CoverageReport
-	for _, rule := range rules {
-		if rule.Eligible {
-			r.EligibleRules++
-			continue
-		}
-		r.CorazaOnlyRules++
-	}
-	return r
-}
diff --git a/internal/tlsfront/tlsfront.go b/internal/tlsfront/tlsfront.go
index 8a86f93..5f2cc40 100644
--- a/internal/tlsfront/tlsfront.go
+++ b/internal/tlsfront/tlsfront.go
@@ -177,13 +177,6 @@ func PublicTLSListeners(sites []Site) map[string]bool {
 	return out
 }
 
-func ActualListenerKey(cfg AccelerationConfig, sites []Site, publicListen string) string {
-	if FrontendEnabled(cfg) && PublicTLSListeners(sites)[publicListen] {
-		return InternalListenerKey(publicListen)
-	}
-	return publicListen
-}
-
 func PublicListenForKey(cfg AccelerationConfig, sites []Site, key string) string {
 	if !FrontendEnabled(cfg) || !IsInternalListenerKey(key) {
 		return key
diff --git a/internal/vectoraccel/qualification.go b/internal/vectoraccel/qualification.go
index 9f8b972..6ec2827 100644
--- a/internal/vectoraccel/qualification.go
+++ b/internal/vectoraccel/qualification.go
@@ -6,34 +6,6 @@ import (
 	"sort"
 )
 
-// QualificationResult is used by Phase 1 learning qualification. It records
-// the safety property without changing the authoritative Coraza path.
-type QualificationResult struct {
-	Samples          int64
-	CandidateMatches int64
-	CorazaMatches    int64
-	FalseNegatives   int64
-}
-
-func (r QualificationResult) Passed() bool {
-	return r.FalseNegatives == 0 && r.CandidateMatches >= r.CorazaMatches
-}
-
-// CompareCandidates records the strict VectorScan safety gate:
-// candidates must cover all transaction-final Coraza matches.
-func CompareCandidates(candidates, coraza map[int]struct{}) QualificationResult {
-	r := QualificationResult{}
-	r.CandidateMatches = int64(len(candidates))
-	r.CorazaMatches = int64(len(coraza))
-	for id := range coraza {
-		if _, ok := candidates[id]; !ok {
-			r.FalseNegatives++
-		}
-	}
-	r.Samples = 1
-	return r
-}
-
 // QualificationCandidates executes the same native group scans used by the
 // runtime Learning path, but never creates a skip header and never suppresses
 // Coraza. It is intended for the offline Phase 1 differential runner.
diff --git a/internal/vectoraccel/qualification_test.go b/internal/vectoraccel/qualification_test.go
index 66d8a85..1dafdb5 100644
--- a/internal/vectoraccel/qualification_test.go
+++ b/internal/vectoraccel/qualification_test.go
@@ -2,17 +2,6 @@ package vectoraccel
 
 import "testing"
 
-func TestCompareCandidatesRequiresSuperset(t *testing.T) {
-	got := CompareCandidates(map[int]struct{}{1: {}, 2: {}, 3: {}}, map[int]struct{}{1: {}, 3: {}})
-	if !got.Passed() || got.FalseNegatives != 0 {
-		t.Fatalf("expected superset to pass: %#v", got)
-	}
-	got = CompareCandidates(map[int]struct{}{1: {}}, map[int]struct{}{1: {}, 3: {}})
-	if got.Passed() || got.FalseNegatives != 1 {
-		t.Fatalf("expected missing candidate to fail: %#v", got)
-	}
-}
-
 func TestEligibleMatchedRuleIDsFiltersAndDeduplicates(t *testing.T) {
 	p := &SitePlan{byRule: map[int]*groupRuntime{100: {}, 200: {}}}
 	got := p.EligibleMatchedRuleIDs([]int{300, 200, 100, 200})
diff --git a/internal/vectoraccel/rules.go b/internal/vectoraccel/rules.go
index fbf87c0..999977f 100644
--- a/internal/vectoraccel/rules.go
+++ b/internal/vectoraccel/rules.go
@@ -138,12 +138,12 @@ func classifyRule(id int, vars, op, actions string) (RuleSpec, bool) {
 	}
 	transforms := []string{}
 	hasChain := false
-	for _, raw := range splitActions(actions) {
+	for _, raw := range shared.SplitActions(actions) {
 		a := strings.TrimSpace(raw)
-		name, value := splitActionForCapability(a)
+		name, value := shared.SplitAction(a)
 		switch strings.ToLower(name) {
 		case "t":
-			if v := strings.ToLower(trimQuotedActionValue(value)); v != "" {
+			if v := strings.ToLower(shared.TrimActionValue(value)); v != "" {
 				transforms = append(transforms, v)
 			}
 		case "chain":
@@ -183,109 +183,26 @@ func classifyRule(id int, vars, op, actions string) (RuleSpec, bool) {
 	return spec, true
 }
 
-func splitActionForCapability(s string) (string, string) {
-	if i := strings.IndexByte(s, ':'); i >= 0 {
-		return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1:])
-	}
-	return strings.TrimSpace(s), ""
-}
-
-func trimQuotedActionValue(s string) string {
-	s = strings.TrimSpace(s)
-	if len(s) >= 2 && ((s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '"' && s[len(s)-1] == '"')) {
-		s = s[1 : len(s)-1]
-	}
-	return strings.TrimSpace(s)
-}
-
-func splitActions(s string) []string {
-	var out []string
-	start := 0
-	quote := byte(0)
-	esc := false
-	for i := 0; i < len(s); i++ {
-		c := s[i]
-		if esc {
-			esc = false
-			continue
-		}
-		if c == '\\' {
-			esc = true
-			continue
-		}
-		if quote != 0 {
-			if c == quote {
-				quote = 0
-			}
-			continue
-		}
-		if c == '\'' || c == '"' {
-			quote = c
-			continue
-		}
-		if c == ',' {
-			out = append(out, s[start:i])
-			start = i + 1
-		}
-	}
-	out = append(out, s[start:])
-	return out
-}
-
 func splitSecRule(st string) (string, string, string, bool) {
 	s := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(st), "SecRule"))
 	if s == "" {
 		return "", "", "", false
 	}
-	vars, rest, ok := nextToken(s)
+	vars, rest, ok := shared.NextToken(s)
 	if !ok {
 		return "", "", "", false
 	}
-	op, rest, ok := nextToken(strings.TrimSpace(rest))
+	op, rest, ok := shared.NextToken(strings.TrimSpace(rest))
 	if !ok {
 		return "", "", "", false
 	}
-	actions, _, ok := nextToken(strings.TrimSpace(rest))
+	actions, _, ok := shared.NextToken(strings.TrimSpace(rest))
 	if !ok {
 		return "", "", "", false
 	}
 	return vars, op, actions, true
 }
 
-func nextToken(s string) (string, string, bool) {
-	if s == "" {
-		return "", "", false
-	}
-	if s[0] == '"' || s[0] == '\'' {
-		q := s[0]
-		var b strings.Builder
-		esc := false
-		for i := 1; i < len(s); i++ {
-			c := s[i]
-			if esc {
-				b.WriteByte(c)
-				esc = false
-				continue
-			}
-			if c == '\\' {
-				esc = true
-				b.WriteByte(c)
-				continue
-			}
-			if c == q {
-				return b.String(), s[i+1:], true
-			}
-			b.WriteByte(c)
-		}
-		return "", "", false
-	}
-	i := strings.IndexAny(s, " \t\r\n")
-	if i < 0 {
-		return s, "", true
-	}
-	return s[:i], s[i:], true
-}
-
 func loadStatements(path string) ([]string, error) {
 	seen := map[string]bool{}
 	var total int64
diff --git a/packaging/cleanhost/README.md b/packaging/cleanhost/README.md
index e6b2595..ec6b9e6 100644
--- a/packaging/cleanhost/README.md
+++ b/packaging/cleanhost/README.md
@@ -74,3 +74,5 @@ sudo ./packaging/cleanhost/run-clean-host-qualification.sh \
 Use the corresponding RPM packages on RHEL-family hosts. Slice C remains the
 more exhaustive rollback/failed-upgrade qualification; Slice D intentionally
 focuses on the end-to-end clean-machine operator experience.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/packaging/deb/CRS-PROVISIONING.md b/packaging/deb/CRS-PROVISIONING.md
index 1c94486..d070e6b 100644
--- a/packaging/deb/CRS-PROVISIONING.md
+++ b/packaging/deb/CRS-PROVISIONING.md
@@ -27,3 +27,5 @@ sudo systemctl enable --now waf-proxy
 ```
 
 Do not treat successful `.deb` installation as WAF readiness when CRS is absent.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/packaging/deb/README.md b/packaging/deb/README.md
index 1564f26..251b16a 100644
--- a/packaging/deb/README.md
+++ b/packaging/deb/README.md
@@ -49,3 +49,5 @@ sudo systemctl enable --now waf-proxy
 
 Upgrades preserve dpkg conffiles, `/etc/waf/waf-proxy.env`, certificates/CRS,
 and `/var/lib/waf-proxy`. No maintainer script performs network access.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/packaging/qualification/README.md b/packaging/qualification/README.md
index e9517f0..3197322 100644
--- a/packaging/qualification/README.md
+++ b/packaging/qualification/README.md
@@ -73,3 +73,5 @@ Use `--format rpm` on RHEL/Rocky/AlmaLinux/Oracle Linux with the corresponding
 RPM fixture set. Real production package lifecycle qualification should rerun
 the same runner with qualified Go 1.25 packages; fixture PASS alone is not
 production distro acceptance.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/packaging/rpm/CRS-PROVISIONING.md b/packaging/rpm/CRS-PROVISIONING.md
index fe00f23..d5959c3 100644
--- a/packaging/rpm/CRS-PROVISIONING.md
+++ b/packaging/rpm/CRS-PROVISIONING.md
@@ -27,3 +27,5 @@ sudo systemctl enable --now waf-proxy
 ```
 
 Do not treat successful RPM installation as WAF readiness when CRS is absent.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/packaging/rpm/README.md b/packaging/rpm/README.md
index 40690ed..4a54e0e 100644
--- a/packaging/rpm/README.md
+++ b/packaging/rpm/README.md
@@ -55,3 +55,5 @@ sudo systemctl enable --now waf-proxy
 not download packages, CRS, Git repositories, or other network content.
 
 See `SELINUX.md` for the enforcing-SELinux qualification boundary.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/packaging/rpm/SELINUX.md b/packaging/rpm/SELINUX.md
index bbcebc8..80809ca 100644
--- a/packaging/rpm/SELINUX.md
+++ b/packaging/rpm/SELINUX.md
@@ -18,3 +18,5 @@ This is deliberate:
 The systemd units continue to use capability bounding and filesystem sandboxing.
 Any RHEL-family AVC observed during Slice D is release evidence and must be
 resolved before that distribution is promoted from NOT_RUN/DEFERRED.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/pki_hardening.go b/pki_hardening.go
deleted file mode 100644
index 2c7b0e5..0000000
--- a/pki_hardening.go
+++ /dev/null
@@ -1,77 +0,0 @@
-package main
-
-import (
-	"errors"
-	"net"
-	"net/url"
-	"strings"
-	"time"
-)
-
-// PKI Slice 3 hardening helpers. These are deliberately independent from
-// certificate/key management. They only protect CRL retrieval boundaries.
-
-type crlFetchPolicy struct {
-	AllowHTTP bool
-}
-
-func validateCRLFetchURL(raw string, policy crlFetchPolicy) error {
-	u, err := url.Parse(raw)
-	if err != nil {
-		return errors.New("invalid CRL URL")
-	}
-	if u.Scheme != "https" && !(policy.AllowHTTP && u.Scheme == "http") {
-		return errors.New("CRL URL scheme is not allowed")
-	}
-	if u.Host == "" || u.User != nil {
-		return errors.New("CRL URL host is invalid")
-	}
-	host := u.Hostname()
-	if host == "" {
-		return errors.New("CRL URL hostname is empty")
-	}
-	if strings.EqualFold(host, "localhost") {
-		return errors.New("CRL URL localhost is blocked")
-	}
-	if ip := net.ParseIP(host); ip != nil {
-		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
-			return errors.New("CRL URL private or local address is blocked")
-		}
-	}
-	return nil
-}
-
-type crlRefreshState string
-
-const (
-	crlRefreshPending crlRefreshState = "PENDING"
-	crlRefreshRunning crlRefreshState = "RUNNING"
-	crlRefreshSuccess crlRefreshState = "SUCCESS"
-	crlRefreshFailed  crlRefreshState = "FAILED"
-)
-
-type crlRefreshRecord struct {
-	URL        string
-	State      crlRefreshState
-	Attempts   int
-	LastError  string
-	UpdatedAt  time.Time
-	ActiveHash string
-}
-
-// lastKnownGoodCRL retains the previous accepted CRL material when a refresh
-// candidate fails validation.
-type lastKnownGoodCRL struct {
-	LoadedAt time.Time
-	Data     []byte
-}
-
-func retainLastKnownGood(previous *lastKnownGoodCRL, candidate []byte, now time.Time, valid bool) *lastKnownGoodCRL {
-	if valid {
-		return &lastKnownGoodCRL{LoadedAt: now, Data: append([]byte(nil), candidate...)}
-	}
-	if previous == nil {
-		return nil
-	}
-	return previous
-}
diff --git a/pki_hardening_test.go b/pki_hardening_test.go
deleted file mode 100644
index 1844579..0000000
--- a/pki_hardening_test.go
+++ /dev/null
@@ -1,28 +0,0 @@
-package main
-
-import (
-	"testing"
-	"time"
-)
-
-func TestValidateCRLFetchURLBlocksLocalTargets(t *testing.T) {
-	for _, u := range []string{
-		"http://127.0.0.1/crl",
-		"https://localhost/crl",
-		"http://169.254.169.254/latest",
-	} {
-		if err := validateCRLFetchURL(u, crlFetchPolicy{AllowHTTP: true}); err == nil {
-			t.Fatalf("expected blocked URL: %s", u)
-		}
-	}
-}
-
-func TestLastKnownGoodCRLRetention(t *testing.T) {
-	old := &lastKnownGoodCRL{Data: []byte("old")}
-	got := retainLastKnownGood(old, []byte("bad"), timeNow(), false)
-	if string(got.Data) != "old" {
-		t.Fatal("previous CRL was not retained")
-	}
-}
-
-func timeNow() time.Time { return time.Now() }
diff --git a/qualification/API2_FINAL_QUALIFICATION_REPORT.md b/qualification/API2_FINAL_QUALIFICATION_REPORT.md
index 0ab80e1..11773c9 100644
--- a/qualification/API2_FINAL_QUALIFICATION_REPORT.md
+++ b/qualification/API2_FINAL_QUALIFICATION_REPORT.md
@@ -21,3 +21,5 @@ Go 1.25 build and Go test execution are deferred to the final Go qualification g
 | Production runtime validation | DEFERRED |
 
 Release decision: NOT CLAIMED.
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/qualification/API2_QUALIFICATION_STATUS.md b/qualification/API2_QUALIFICATION_STATUS.md
index 4f6e4ff..6b0d7fa 100644
--- a/qualification/API2_QUALIFICATION_STATUS.md
+++ b/qualification/API2_QUALIFICATION_STATUS.md
@@ -7,3 +7,5 @@ Functional qualification: COMPLETE
 Go qualification: DEFERRED
 
 Release qualification: PENDING
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/qualification/README.md b/qualification/README.md
index d074ffe..cccc77c 100644
--- a/qualification/README.md
+++ b/qualification/README.md
@@ -26,3 +26,5 @@ traffic before using this as production qualification evidence.
 ## Phase 5 Slice A Runtime Qualification Framework
 
 `runtime_report.go` provides the evidence schema used by release-host qualification. It does not mark runtime gates as passed; reports must be populated by executed Go 1.25, Coraza v3.7.0, and libvectorscan release-host runs.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/qualification/RELEASE_DECISION.md b/qualification/RELEASE_DECISION.md
index 1186b88..5bc2479 100644
--- a/qualification/RELEASE_DECISION.md
+++ b/qualification/RELEASE_DECISION.md
@@ -5,3 +5,5 @@ API-2 Typed Schema Learning is functionally prepared but not released.
 Blocking items:
 - Go 1.25 qualified build/test
 - production runtime validation
+
+<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
diff --git a/qualification/corpus/README.md b/qualification/corpus/README.md
index 203a509..3acf560 100644
--- a/qualification/corpus/README.md
+++ b/qualification/corpus/README.md
@@ -3,3 +3,5 @@
 > **Documentation baseline — 2026-09-17.** This file documents a component or qualification path. Repository-wide release truth lives in [`DOCUMENTATION_INDEX.md`](../../DOCUMENTATION_INDEX.md), [`SOURCE_BASELINE_GATE_RESULT.md`](../../SOURCE_BASELINE_GATE_RESULT.md), and [`TESTING_RESULTS.md`](../../TESTING_RESULTS.md). Component PASS evidence must not be promoted into a root-build, runtime, package-lifecycle, clean-host, or release PASS outside its stated scope.
 
 Replay samples are metadata driven. Production CRS qualification remains deferred until real Coraza and libvectorscan are available.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/qualification/hsm/README.md b/qualification/hsm/README.md
index 1dd2f08..5724cf4 100644
--- a/qualification/hsm/README.md
+++ b/qualification/hsm/README.md
@@ -51,3 +51,5 @@ The runner writes `vendor-hsm-qualification.json` with
 `vendor_class=real_vendor_hsm`. Do not run this against SoftHSM or a mock. The
 checked-in file intentionally remains `NOT_RUN` until real production-class
 hardware/service infrastructure is exercised.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/qualification/performance/README.md b/qualification/performance/README.md
index f9e50f0..3207225 100644
--- a/qualification/performance/README.md
+++ b/qualification/performance/README.md
@@ -72,3 +72,5 @@ Application throughput is payload throughput from `wafbench`; it is not Ethernet
 line rate. No throughput figure in this repository should be treated as a
 production sizing claim unless the corresponding certification report is bound
 to the tested artifact and approved target.
+
+<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/qualification_diff.go b/qualification_diff.go
index 5327e96..6edd271 100644
--- a/qualification_diff.go
+++ b/qualification_diff.go
@@ -12,6 +12,3 @@ func VectorScanDifferential(candidates []int, coraza []int) (falseNeg []int) {
 	}
 	return
 }
-func QualificationPassed(candidates []int, coraza []int) bool {
-	return len(VectorScanDifferential(candidates, coraza)) == 0
-}
diff --git a/reliability.go b/reliability.go
deleted file mode 100644
index a95b453..0000000
--- a/reliability.go
+++ /dev/null
@@ -1,43 +0,0 @@
-package main
-
-import (
-	"encoding/json"
-	"time"
-)
-
-// ReliabilityScenario describes a controlled failure or recovery qualification case.
-// It records expected behavior without claiming that execution happened.
-type ReliabilityScenario struct {
-	Name        string `json:"name"`
-	Category    string `json:"category"`
-	Expected    string `json:"expected"`
-	Status      string `json:"status"`
-	Evidence    string `json:"evidence,omitempty"`
-	GeneratedAt string `json:"generated_at"`
-}
-
-type ReliabilityReport struct {
-	Format    string               `json:"format"`
-	Status    string               `json:"status"`
-	Scenarios []ReliabilityScenario `json:"scenarios"`
-}
-
-func NewReliabilityReport() ReliabilityReport {
-	now := time.Now().UTC().Format(time.RFC3339)
-	return ReliabilityReport{
-		Format: "waf-proxy-reliability-qualification-v1",
-		Status: "NOT_RUN",
-		Scenarios: []ReliabilityScenario{
-			{Name: "crs_unavailable", Category: "dependency_failure", Expected: "fail_closed_no_bypass", Status: "NOT_RUN", GeneratedAt: now},
-			{Name: "crl_unavailable", Category: "pki_failure", Expected: "use_last_known_good", Status: "NOT_RUN", GeneratedAt: now},
-			{Name: "storage_unavailable", Category: "storage_failure", Expected: "degraded_visible_state", Status: "NOT_RUN", GeneratedAt: now},
-			{Name: "process_restart", Category: "recovery", Expected: "recover_configuration_and_state", Status: "NOT_RUN", GeneratedAt: now},
-			{Name: "resource_exhaustion", Category: "resource", Expected: "controlled_failure", Status: "NOT_RUN", GeneratedAt: now},
-		},
-	}
-}
-
-func MarshalReliabilityReport() ([]byte, error) {
-	r := NewReliabilityReport()
-	return json.MarshalIndent(r, "", "  ")
-}
diff --git a/reliability_test.go b/reliability_test.go
deleted file mode 100644
index 3affdfa..0000000
--- a/reliability_test.go
+++ /dev/null
@@ -1,13 +0,0 @@
-package main
-
-import "testing"
-
-func TestReliabilityReportBoundary(t *testing.T) {
-	report := NewReliabilityReport()
-	if report.Status != "NOT_RUN" {
-		t.Fatalf("unexpected status: %s", report.Status)
-	}
-	if len(report.Scenarios) == 0 {
-		t.Fatal("expected scenarios")
-	}
-}
diff --git a/schema_api2_continuation.go b/schema_api2_continuation.go
deleted file mode 100644
index f7b5529..0000000
--- a/schema_api2_continuation.go
+++ /dev/null
@@ -1,54 +0,0 @@
-package main
-
-import "time"
-
-// Compatibility input used by deterministic unit tests and migration helpers.
-type SchemaFieldLearning struct {
-	Path              string
-	Location          string
-	Types             []SchemaTypeEvidence
-	Formats           []string
-	SampleCount       int64
-	PresenceRate      float64
-	RequiredCandidate bool
-	EnumCandidate     []string
-	Sensitive         bool
-}
-
-type SchemaObservationLearning struct {
-	OperationID string
-	Version     int
-	SampleCount int64
-	Fields      []SchemaFieldLearning
-	UpdatedAt   time.Time
-}
-
-// buildSchemaLearningCandidate remains deterministic and does not invoke AI.
-func buildSchemaLearningCandidate(obs SchemaObservationLearning) SchemaCandidate {
-	fields := make([]SchemaFieldObservation, 0, len(obs.Fields))
-	var confidence float64
-	for _, field := range obs.Fields {
-		typeCounts := map[string]int64{}
-		for _, t := range field.Types {
-			typeCounts[t.Type] = t.Count
-			confidence += t.Confidence
-		}
-		fields = append(fields, SchemaFieldObservation{
-			Path: field.Path, Location: field.Location, TypeCounts: typeCounts, Types: append([]SchemaTypeEvidence(nil), field.Types...),
-			Formats: append([]string(nil), field.Formats...), Samples: field.SampleCount, PresenceRate: field.PresenceRate,
-			RequiredCandidate: field.RequiredCandidate, EnumCandidate: append([]string(nil), field.EnumCandidate...), Sensitive: field.Sensitive,
-		})
-	}
-	status := "LEARNING"
-	if obs.SampleCount >= schemaCandidateMinSamples {
-		status = "CANDIDATE"
-	}
-	if obs.UpdatedAt.IsZero() {
-		obs.UpdatedAt = time.Now().UTC()
-	}
-	c := SchemaCandidate{ID: schemaCandidateID(obs.OperationID), OperationID: obs.OperationID, Status: status, SampleCount: obs.SampleCount, Fields: fields, CreatedAt: obs.UpdatedAt, UpdatedAt: obs.UpdatedAt, Version: obs.Version}
-	if len(fields) > 0 {
-		c.Confidence = confidence / float64(len(fields))
-	}
-	return c
-}
diff --git a/security_event.go b/security_event.go
deleted file mode 100644
index 732127f..0000000
--- a/security_event.go
+++ /dev/null
@@ -1,11 +0,0 @@
-package main
-
-type securityEvent struct {
-	Type      string `json:"type"`
-	RequestID string `json:"request_id,omitempty"`
-	Client    string `json:"client,omitempty"`
-	Reason    string `json:"reason,omitempty"`
-	Action    string `json:"action,omitempty"`
-}
-
-type securityEventExporter interface{ ExportSecurityEvent(securityEvent) error }
diff --git a/security_state.go b/security_state.go
deleted file mode 100644
index 2c3f7eb..0000000
--- a/security_state.go
+++ /dev/null
@@ -1,99 +0,0 @@
-package main
-
-import (
-	"sync"
-	"time"
-)
-
-type SecurityIndicator struct {
-	Type string `json:"type"`
-	Value string `json:"value"`
-	Confidence float64 `json:"confidence"`
-	Source string `json:"source"`
-	State string `json:"state"`
-	CreatedAt time.Time `json:"created_at"`
-	ExpiresAt *time.Time `json:"expires_at,omitempty"`
-}
-
-type LearningAggregate struct {
-	RuleID int `json:"rule_id"`
-	ObservationCount uint64 `json:"observation_count"`
-	MatchCount uint64 `json:"match_count"`
-	Confidence float64 `json:"confidence"`
-	State string `json:"state"`
-	LastSeen time.Time `json:"last_seen"`
-}
-
-type SecuritySession struct {
-	ID string `json:"id"`
-	Client string `json:"client"`
-	CorrelationID string `json:"correlation_id"`
-	FirstSeen time.Time `json:"first_seen"`
-	LastSeen time.Time `json:"last_seen"`
-	RequestCount uint64 `json:"request_count"`
-	RiskState string `json:"risk_state"`
-}
-
-type SecurityAuditEvent struct {
-	ID string `json:"id"`
-	Type string `json:"type"`
-	RequestID string `json:"request_id,omitempty"`
-	Client string `json:"client,omitempty"`
-	Action string `json:"action,omitempty"`
-	Evidence map[string]string `json:"evidence,omitempty"`
-	CreatedAt time.Time `json:"created_at"`
-}
-
-type NotificationState struct {
-	ID string `json:"id"`
-	Type string `json:"type"`
-	Severity string `json:"severity"`
-	State string `json:"state"`
-	RetryCount int `json:"retry_count"`
-	CreatedAt time.Time `json:"created_at"`
-}
-
-type securityStateStore interface {
-	PutIndicator(SecurityIndicator)
-	ListIndicators() []SecurityIndicator
-	PutAudit(SecurityAuditEvent)
-	ListAudits() []SecurityAuditEvent
-}
-
-type memorySecurityStateStore struct {
-	mu sync.RWMutex
-	indicators []SecurityIndicator
-	audits []SecurityAuditEvent
-}
-
-func newMemorySecurityStateStore() *memorySecurityStateStore {
-	return &memorySecurityStateStore{}
-}
-
-func (s *memorySecurityStateStore) PutIndicator(v SecurityIndicator) {
-	s.mu.Lock()
-	defer s.mu.Unlock()
-	s.indicators = append(s.indicators, v)
-}
-
-func (s *memorySecurityStateStore) ListIndicators() []SecurityIndicator {
-	s.mu.RLock()
-	defer s.mu.RUnlock()
-	out := make([]SecurityIndicator, len(s.indicators))
-	copy(out, s.indicators)
-	return out
-}
-
-func (s *memorySecurityStateStore) PutAudit(v SecurityAuditEvent) {
-	s.mu.Lock()
-	defer s.mu.Unlock()
-	s.audits = append(s.audits, v)
-}
-
-func (s *memorySecurityStateStore) ListAudits() []SecurityAuditEvent {
-	s.mu.RLock()
-	defer s.mu.RUnlock()
-	out := make([]SecurityAuditEvent, len(s.audits))
-	copy(out, s.audits)
-	return out
-}
diff --git a/security_state_test.go b/security_state_test.go
deleted file mode 100644
index 1c21d04..0000000
--- a/security_state_test.go
+++ /dev/null
@@ -1,15 +0,0 @@
-package main
-
-import (
-	"testing"
-	"time"
-)
-
-func TestMemorySecurityStateStore(t *testing.T) {
-	s := newMemorySecurityStateStore()
-	s.PutIndicator(SecurityIndicator{Type:"ip", Value:"192.0.2.1", State:"active"})
-	s.PutAudit(SecurityAuditEvent{ID:"1", Type:"blocked", CreatedAt:time.Now()})
-	if len(s.ListIndicators()) != 1 || len(s.ListAudits()) != 1 {
-		t.Fatal("security state persistence failed")
-	}
-}
diff --git a/sequence_api61.go b/sequence_api61.go
index 40ddafb..65e0880 100644
--- a/sequence_api61.go
+++ b/sequence_api61.go
@@ -440,11 +440,6 @@ func (s *sequenceStore) digest(material string) string {
 	return hex.EncodeToString(h.Sum(nil)[:20])
 }
 
-func sequenceTransitionID(site, from, to string) string {
-	h := sha256.Sum256([]byte(site + "\x00" + from + "\x00" + to))
-	return hex.EncodeToString(h[:20])
-}
-
 func sequenceWorkflowTransitionID(site, workflowID, from, to string) string {
 	h := sha256.Sum256([]byte(site + "\x00" + workflowID + "\x00" + from + "\x00" + to))
 	return hex.EncodeToString(h[:20])
diff --git a/support_bundle_v2.go b/support_bundle_v2.go
deleted file mode 100644
index 4004799..0000000
--- a/support_bundle_v2.go
+++ /dev/null
@@ -1,17 +0,0 @@
-package main
-
-import "time"
-
-type SupportBundleProvenance struct {
-	Version string `json:"version"`
-	CreatedAt time.Time `json:"created_at"`
-	GoVersion string `json:"go_version,omitempty"`
-	CorazaVersion string `json:"coraza_version,omitempty"`
-	VectorScanEnabled bool `json:"vectorscan_enabled"`
-}
-
-type DependencyEvidence struct {
-	Name string `json:"name"`
-	Version string `json:"version"`
-	Source string `json:"source,omitempty"`
-}
diff --git a/tools/tests/test-code-duplication-review-source.py b/tools/tests/test-code-duplication-review-source.py
new file mode 100755
index 0000000..0fccba1
--- /dev/null
+++ b/tools/tests/test-code-duplication-review-source.py
@@ -0,0 +1,105 @@
+#!/usr/bin/env python3
+from pathlib import Path
+import re
+
+ROOT = Path(__file__).resolve().parents[2]
+read = lambda p: (ROOT / p).read_text(encoding="utf-8")
+checks = []
+
+def req(cond, msg):
+    if not cond:
+        raise SystemExit("CODE_DUPLICATION_REVIEW_SOURCE_GATE_FAIL: " + msg)
+    checks.append(msg)
+
+# Removed duplicate/dead compatibility layers must not reappear.
+removed = (
+    "debug_evidence.go", "debug_evidence_ops_v2.go", "support_bundle_v2.go",
+    "security_state.go", "security_event.go", "deployment_readiness.go",
+    "reliability.go", "vectorscan_qualification.go", "vectorscan_audit_store.go",
+    "vectorscan_transition_audit.go", "schema_api2_continuation.go",
+    "pki_hardening.go", "internal/coverage/report.go",
+)
+for path in removed:
+    req(not (ROOT / path).exists(), "removed duplicate/dead layer stays absent: " + path)
+
+removed_tests = (
+    "debug_evidence_test.go", "debug_evidence_ops_v2_test.go", "security_state_test.go",
+    "reliability_test.go", "vectorscan_qualification_test.go", "pki_hardening_test.go",
+)
+for path in removed_tests:
+    req(not (ROOT / path).exists(), "obsolete test for removed layer stays absent: " + path)
+
+# Debug evidence has one runtime store and one sanitization path.
+debug = read("debug_bundle.go")
+sanitize = read("debug_sanitize.go")
+observer = read("coraza_observer.go")
+support = read("support_api.go")
+for token in ("type DebugEvidenceStore struct", "func (s *DebugEvidenceStore) Put", "func (s *DebugEvidenceStore) ExportIncident"):
+    req(token in debug, "authoritative debug store retained: " + token)
+for token in ("isSensitiveDebugKey", "sanitizeDebugValue", "sanitizeDebugMap"):
+    req(token in sanitize, "single debug sanitization path retained: " + token)
+for token in ("DebugEvidenceCapture", "currentDebugEvidenceCapture", "DebugEvidenceOpsV2", "TenantExport"):
+    req(token not in debug + sanitize + observer + support, "legacy debug compatibility path absent: " + token)
+req("currentDebugEvidenceStore" in observer, "Coraza debug evidence uses authoritative store")
+
+# SecLang token/action parsing is shared by offline coverage and live VectorScan.
+shared = read("internal/capability/seclang.go")
+coverage = read("internal/coverage/crs/parser.go")
+vector = read("internal/vectoraccel/rules.go")
+for token in ("func NextToken", "func SplitActions", "func SplitAction", "func TrimActionValue"):
+    req(token in shared, "shared SecLang helper exists: " + token)
+for token in ("shared.NextToken", "shared.SplitActions", "shared.SplitAction", "shared.TrimActionValue"):
+    req(token in coverage, "coverage parser uses shared helper: " + token)
+    req(token in vector, "VectorScan parser uses shared helper: " + token)
+for token in ("func nextToken", "func splitActions", "func splitActionForCapability", "func trimQuotedActionValue"):
+    req(token not in coverage + vector, "duplicate SecLang helper removed: " + token)
+
+# Superseding implementations remain present.
+req("func (a *adminServer) handleDoctor" in support, "Doctor remains the deployment readiness surface")
+req("func validateCRLURLSyntax" in read("pki_url.go"), "wired PKI URL hardening remains")
+req("func prepareCRLStore" in read("pki_url.go"), "wired PKI CRL store remains")
+req("func (p *SitePlan) QualificationCandidates" in read("internal/vectoraccel/qualification.go"), "live VectorScan qualification path remains")
+req("BuildCoverageReportV2" in read("internal/coverage/crs/report.go"), "current CRS coverage report remains")
+req("type schemaStore struct" in read("schema_api2.go"), "live API-2 schema store remains")
+
+# Removed placeholder models/symbols must not exist anywhere in production Go source.
+prod = "\n".join(p.read_text(encoding="utf-8") for p in ROOT.rglob("*.go") if not p.name.endswith("_test.go"))
+for token in (
+    "DebugEvidenceCapture", "DebugEvidenceOpsV2", "SupportBundleProvenance",
+    "securityStateStore", "securityEventExporter", "DeploymentReadinessReport",
+    "ReliabilityReport", "VectorScanQualificationRecord", "VectorScanAuditStore",
+    "VectorScanTransitionAudit", "SchemaObservationLearning", "crlFetchPolicy",
+    "QualificationPassed", "TransformCapabilities", "CompareCandidates",
+    "QualificationResult", "ActualListenerKey", "nonAnonymousAuthSamples",
+    "sequenceTransitionID", "ClientIdentityAuditSink",
+):
+    req(re.search(r"\b" + re.escape(token) + r"\b", prod) is None, "obsolete symbol absent: " + token)
+
+# Admin route authority must remain unique. Duplicate METHOD+pattern registrations
+# would mean two handlers competing for the same control-plane function.
+route_text = read("admin.go") + "\n" + read("update.go")
+routes = re.findall(r'mux\.HandleFunc\("([A-Z]+) ([^"\\]+)"', route_text)
+seen = set()
+dups = []
+for route in routes:
+    if route in seen:
+        dups.append(route)
+    seen.add(route)
+req(not dups, "Admin METHOD+path registrations remain unique")
+req(len(routes) >= 120, "Admin route inventory remains complete after consolidation")
+
+# Authority separation is unchanged: learned sequence/BOLA evidence is not a
+# second enforcement stack; deterministic GraphQL/positive-schema/identity stay separate.
+api6 = read("sequence_api63.go") + read("sequence_api64.go")
+api7 = read("bola_detection_api73.go") + read("bola_policy_api74.go")
+api8 = read("graphql_api8.go")
+req('Mode string `json:"mode"`' in api6 and 'sequenceDetectionDetect' in api6, "sequence control remains LEARN/DETECT authority")
+req("InferenceBlocking" in api7 and "false" in api7, "BOLA inferred evidence remains non-blocking")
+req("graphqlModeEnforce" in api8, "GraphQL explicit deterministic ENFORCE authority remains")
+
+# Documentation and gate wiring.
+req((ROOT / "CODE_DUPLICATION_REVIEW.md").exists(), "duplication review document exists")
+req("test-code-duplication-review-source.py" in read("build.sh"), "build.sh runs duplication review gate")
+req("test-code-duplication-review-source.py" in read(".github/workflows/ci.yml"), "CI runs duplication review gate")
+
+print(f"CODE_DUPLICATION_REVIEW_SOURCE_GATE_PASS checks={len(checks)} routes={len(routes)}")
diff --git a/tools/tests/test-markdown-review-source.py b/tools/tests/test-markdown-review-source.py
new file mode 100755
index 0000000..e86789a
--- /dev/null
+++ b/tools/tests/test-markdown-review-source.py
@@ -0,0 +1,40 @@
+#!/usr/bin/env python3
+from pathlib import Path
+import sys
+
+root = Path(__file__).resolve().parents[2]
+mds = sorted(root.rglob('*.md'))
+fail=[]
+marker='<!-- documentation-review: 2026-09-28; classification:'
+for p in mds:
+    text=p.read_text(encoding='utf-8')
+    if marker not in text:
+        fail.append(f'missing review marker: {p.relative_to(root)}')
+ledger=root/'MARKDOWN_REVIEW_2026-09-28.md'
+if not ledger.is_file(): fail.append('missing MARKDOWN_REVIEW_2026-09-28.md')
+else:
+    lt=ledger.read_text(encoding='utf-8')
+    for p in mds:
+        rel=str(p.relative_to(root))
+        if f'`{rel}`' not in lt:
+            fail.append(f'ledger missing: {rel}')
+required_current=[
+ 'README.md','DEVELOPMENT.md','AI_HANDOFF.md','TESTING.md','TESTING_RESULTS.md',
+ 'HANDOVER_STATUS.md','NEXT_CHAT_HANDOVER.md','HANDOVER_PROMPT.md','API_SECURITY_ROADMAP.md',
+ 'API_SECURITY_SLICE_IMPLEMENTATION_ROADMAP.md','DEVELOPMENT_ROADMAP.md','DOCUMENTATION_INDEX.md',
+ 'MANIFEST.md','RELEASE_PROCESS.md','API_SECURITY_CLOSURE_RESULT.md','PRODUCTION_CONTROL_PLANE_HARDENING.md',
+ 'INSTALL.md','PACKAGING_TOOL.md','API8_GRAPHQL_SECURITY.md','API8_POST_AUDIT_HARDENING.md'
+]
+for rel in required_current:
+    text=(root/rel).read_text(encoding='utf-8')
+    if '## Current canonical baseline — 2026-09-28' not in text:
+        fail.append(f'current baseline section missing: {rel}')
+if 'The `web/` Vite console is a **scaffold**' in (root/'INSTALL.md').read_text():
+    fail.append('INSTALL.md still advertises removed web scaffold')
+if '`debug_bundle.go`, `debug_evidence.go`, `support_api.go`' in (root/'API8_POST_AUDIT_HARDENING.md').read_text():
+    fail.append('API8 post-audit doc still claims removed debug_evidence.go as current')
+if fail:
+    print('MARKDOWN_REVIEW_SOURCE_GATE_FAIL')
+    for item in fail: print(' -',item)
+    sys.exit(1)
+print(f'MARKDOWN_REVIEW_SOURCE_GATE_PASS files={len(mds)} current={len(required_current)}')
diff --git a/vectorscan_audit_store.go b/vectorscan_audit_store.go
deleted file mode 100644
index 2dcbcbe..0000000
--- a/vectorscan_audit_store.go
+++ /dev/null
@@ -1,22 +0,0 @@
-package main
-
-import "sync"
-
-type VectorScanAuditStore struct {
-	mu sync.RWMutex
-	Events []VectorScanTransitionAudit
-}
-
-func (s *VectorScanAuditStore) Add(e VectorScanTransitionAudit) {
-	s.mu.Lock()
-	defer s.mu.Unlock()
-	s.Events = append(s.Events, e)
-}
-
-func (s *VectorScanAuditStore) List() []VectorScanTransitionAudit {
-	s.mu.RLock()
-	defer s.mu.RUnlock()
-	out := make([]VectorScanTransitionAudit,len(s.Events))
-	copy(out,s.Events)
-	return out
-}
diff --git a/vectorscan_qualification.go b/vectorscan_qualification.go
deleted file mode 100644
index e919ebe..0000000
--- a/vectorscan_qualification.go
+++ /dev/null
@@ -1,32 +0,0 @@
-package main
-
-import "time"
-
-// VectorScanQualificationRecord is an evidence model for Phase 5 Slice B.
-// It records qualification state without making analyzer-only results runtime PASS.
-type VectorScanQualificationRecord struct {
-	Corpus string    `json:"corpus"`
-	RuleID string    `json:"rule_id,omitempty"`
-	Status string    `json:"status"`
-	FalseNegatives []int `json:"false_negatives,omitempty"`
-	Evidence string  `json:"evidence,omitempty"`
-	CreatedAt time.Time `json:"created_at"`
-}
-
-func NewVectorScanQualificationRecord(corpus, status string) VectorScanQualificationRecord {
-	return VectorScanQualificationRecord{Corpus: corpus, Status: status, CreatedAt: time.Now().UTC()}
-}
-
-// VectorScanQualificationGate keeps Coraza authoritative.
-// Any false negative blocks acceleration eligibility.
-func VectorScanQualificationGate(corazaRules, vectorRules []int) VectorScanQualificationRecord {
-	missed := VectorScanDifferential(vectorRules, corazaRules)
-	if len(missed) != 0 {
-		return VectorScanQualificationRecord{
-			Status: "FAILSAFE_CORAZA_ONLY",
-			FalseNegatives: missed,
-			CreatedAt: time.Now().UTC(),
-		}
-	}
-	return VectorScanQualificationRecord{Status: "VALIDATED", CreatedAt: time.Now().UTC()}
-}
diff --git a/vectorscan_qualification_test.go b/vectorscan_qualification_test.go
deleted file mode 100644
index 7959087..0000000
--- a/vectorscan_qualification_test.go
+++ /dev/null
@@ -1,17 +0,0 @@
-package main
-
-import "testing"
-
-func TestVectorScanQualificationGateZeroFalseNegative(t *testing.T) {
-	r := VectorScanQualificationGate([]int{1,2}, []int{1,2})
-	if r.Status != "VALIDATED" {
-		t.Fatalf("expected validated, got %s", r.Status)
-	}
-}
-
-func TestVectorScanQualificationGateFailsSafe(t *testing.T) {
-	r := VectorScanQualificationGate([]int{1,2}, []int{1})
-	if r.Status != "FAILSAFE_CORAZA_ONLY" {
-		t.Fatalf("expected failsafe, got %s", r.Status)
-	}
-}
diff --git a/vectorscan_transition_audit.go b/vectorscan_transition_audit.go
deleted file mode 100644
index 868bc30..0000000
--- a/vectorscan_transition_audit.go
+++ /dev/null
@@ -1,16 +0,0 @@
-package main
-
-import "time"
-
-type VectorScanTransitionAudit struct {
-	Component   string    `json:"component"`
-	From        string    `json:"from"`
-	To          string    `json:"to"`
-	Reason      string    `json:"reason"`
-	EvidenceRef string    `json:"evidence_ref,omitempty"`
-	At          time.Time `json:"timestamp"`
-}
-
-func NewVectorScanTransitionAudit(from, to, reason, evidence string) VectorScanTransitionAudit {
-	return VectorScanTransitionAudit{Component: "vectorscan", From: from, To: to, Reason: reason, EvidenceRef: evidence, At: time.Now().UTC()}
-}

<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
