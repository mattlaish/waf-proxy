# Build and Admin Startup Blocker Fix Patch

This slice-local patch records source/document differences from the canonical parent `waf-proxy-code-duplication-consolidation-2026-09-28.zip` (SHA-256 `0756e864dd6ef8cc474477665666b2e3000a75cff18c49e627b0553dfd7400c5`) to the 2026-09-30 Build and Admin Startup Blocker Fix baseline. Generated release evidence, delivery manifests, release manifests, checksum lists, runtime/cache outputs, and this patch file itself are intentionally excluded from the diff to avoid self-reference.

diff --git a/.github/workflows/ci.yml b/.github/workflows/ci.yml
index 3db9788..63fe0b4 100644
--- a/.github/workflows/ci.yml
+++ b/.github/workflows/ci.yml
@@ -60,6 +60,8 @@ jobs:
         run: python3 ./tools/tests/test-production-control-plane-hardening-source.py
       - name: Code duplication review source gate
         run: python3 ./tools/tests/test-code-duplication-review-source.py
+      - name: Build/startup blocker fix source gate
+        run: python3 ./tools/tests/test-build-startup-blocker-fix-source.py
       - name: Markdown review source gate
         run: python3 ./tools/tests/test-markdown-review-source.py
 
diff --git a/AI_HANDOFF.md b/AI_HANDOFF.md
index f3d8d6f..e095f0c 100644
--- a/AI_HANDOFF.md
+++ b/AI_HANDOFF.md
@@ -2118,4 +2118,14 @@ VectorScan SecLang parsing use `internal/capability`.
 Older `web/` migration guidance in dated historical sections is superseded: the
 `web/` tree does not exist and must not be treated as a current frontend.
 
+## 2026-09-30 — Build and Admin Startup Blocker Fix
+
+The current source fixes two release-blocking defects found during external Go 1.25 review of the 2026-09-28 consolidation baseline. `admin.go` now imports the standard-library `errors` package required by the existing `errors.New(...)` validation paths. Positive-Schema exception state changes now use the unambiguous route `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`; the shipping Console uses the same path. The prior `POST /api/security/schema/enforcement/exceptions/{exception_id}` registration is removed because it conflicts with `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+ `http.ServeMux` specificity rules and can panic during Admin handler construction.
+
+`admin_route_registration_test.go` now makes full Admin handler registration a regression test, and `tools/tests/test-build-startup-blocker-fix-source.py` is wired into `build.sh` and CI. The source gate also registers the complete 139-route Admin/update inventory with the Go standard library in a standalone temporary program; this catches wildcard-pattern ambiguity even when the repository module graph cannot be compiled locally. Current source evidence: blocker-fix gate **13/13 PASS**, route inventory **139/139 accepted by `ServeMux`**, API/source gates through API-8/post-audit/production-hardening/duplication remain PASS, OpenAI source contract and isolated tests PASS, package-source PASS, package-builder **9/9 PASS**, root Go source shape **96 files PASS**, Admin embedded JavaScript PASS, and changed Go files are `gofmt` clean.
+
+This does **not** promote the source to `TESTED` or `RELEASED`. On the current execution host, canonical `GOTOOLCHAIN=local` Go 1.25 `go mod tidy -diff`, root build, vet, targeted Admin/API64 tests, and race remain `BLOCKED_ENVIRONMENT / NOT_RUN` because only Go 1.23.2 is installed and external Go toolchain/module retrieval is unavailable. Packaging/source-manifest integrity is not a substitute for those root compilation gates. No API-security authority changes are introduced: API-6/API-7 remain non-enforcing, API-8 retains explicit deterministic GraphQL ENFORCE authority, and OpenAI remains advisory only.
+
+Continuation point: use this blocker-fixed tree as the current source, not the 2026-09-28 consolidation ZIP. Do not restore the ambiguous exception-toggle route. The next qualification step is to run the exact final bytes under pinned Go 1.25 with `go mod tidy -diff`, `CGO_ENABLED=0 go build ./...`, `go vet ./...`, the targeted Admin route/API64 tests, full tests, and race tests before any buildable/PASS or release claim.
+
 <!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/API8_GRAPHQL_SECURITY.md b/API8_GRAPHQL_SECURITY.md
index b58dc47..37a1328 100644
--- a/API8_GRAPHQL_SECURITY.md
+++ b/API8_GRAPHQL_SECURITY.md
@@ -114,4 +114,14 @@ The repository-wide duplicate-functionality cleanup does not change GraphQL
 policy semantics, persistence, variable privacy, API-7 evidence integration, or
 LEARN/DETECT/ENFORCE authority. API-8 remains `IMPLEMENTED_TESTING_DEFERRED`.
 
+## 2026-09-30 — Build and Admin Startup Blocker Fix
+
+The current source fixes two release-blocking defects found during external Go 1.25 review of the 2026-09-28 consolidation baseline. `admin.go` now imports the standard-library `errors` package required by the existing `errors.New(...)` validation paths. Positive-Schema exception state changes now use the unambiguous route `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`; the shipping Console uses the same path. The prior `POST /api/security/schema/enforcement/exceptions/{exception_id}` registration is removed because it conflicts with `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+ `http.ServeMux` specificity rules and can panic during Admin handler construction.
+
+`admin_route_registration_test.go` now makes full Admin handler registration a regression test, and `tools/tests/test-build-startup-blocker-fix-source.py` is wired into `build.sh` and CI. The source gate also registers the complete 139-route Admin/update inventory with the Go standard library in a standalone temporary program; this catches wildcard-pattern ambiguity even when the repository module graph cannot be compiled locally. Current source evidence: blocker-fix gate **13/13 PASS**, route inventory **139/139 accepted by `ServeMux`**, API/source gates through API-8/post-audit/production-hardening/duplication remain PASS, OpenAI source contract and isolated tests PASS, package-source PASS, package-builder **9/9 PASS**, root Go source shape **96 files PASS**, Admin embedded JavaScript PASS, and changed Go files are `gofmt` clean.
+
+This does **not** promote the source to `TESTED` or `RELEASED`. On the current execution host, canonical `GOTOOLCHAIN=local` Go 1.25 `go mod tidy -diff`, root build, vet, targeted Admin/API64 tests, and race remain `BLOCKED_ENVIRONMENT / NOT_RUN` because only Go 1.23.2 is installed and external Go toolchain/module retrieval is unavailable. Packaging/source-manifest integrity is not a substitute for those root compilation gates. No API-security authority changes are introduced: API-6/API-7 remain non-enforcing, API-8 retains explicit deterministic GraphQL ENFORCE authority, and OpenAI remains advisory only.
+
+API contract note: this maintenance fix changes only the versionless Admin Console Positive-Schema exception toggle endpoint from `/api/security/schema/enforcement/exceptions/{exception_id}` to `/api/security/schema/enforcement/exceptions/{exception_id}/toggle`. Positive-Schema authority, RBAC, persistence, audit semantics, API-4 enforcement, and API-8 GraphQL authority are unchanged. No API-9 is defined.
+
 <!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/API8_POST_AUDIT_HARDENING.md b/API8_POST_AUDIT_HARDENING.md
index 3943d04..c2d62dd 100644
--- a/API8_POST_AUDIT_HARDENING.md
+++ b/API8_POST_AUDIT_HARDENING.md
@@ -90,4 +90,14 @@ post-audit functional behavior remains in force. Current debug sanitization is
 centralized in `debug_sanitize.go`; the old `debug_evidence.go` compatibility
 layer is no longer present.
 
+## 2026-09-30 — Build and Admin Startup Blocker Fix
+
+The current source fixes two release-blocking defects found during external Go 1.25 review of the 2026-09-28 consolidation baseline. `admin.go` now imports the standard-library `errors` package required by the existing `errors.New(...)` validation paths. Positive-Schema exception state changes now use the unambiguous route `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`; the shipping Console uses the same path. The prior `POST /api/security/schema/enforcement/exceptions/{exception_id}` registration is removed because it conflicts with `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+ `http.ServeMux` specificity rules and can panic during Admin handler construction.
+
+`admin_route_registration_test.go` now makes full Admin handler registration a regression test, and `tools/tests/test-build-startup-blocker-fix-source.py` is wired into `build.sh` and CI. The source gate also registers the complete 139-route Admin/update inventory with the Go standard library in a standalone temporary program; this catches wildcard-pattern ambiguity even when the repository module graph cannot be compiled locally. Current source evidence: blocker-fix gate **13/13 PASS**, route inventory **139/139 accepted by `ServeMux`**, API/source gates through API-8/post-audit/production-hardening/duplication remain PASS, OpenAI source contract and isolated tests PASS, package-source PASS, package-builder **9/9 PASS**, root Go source shape **96 files PASS**, Admin embedded JavaScript PASS, and changed Go files are `gofmt` clean.
+
+This does **not** promote the source to `TESTED` or `RELEASED`. On the current execution host, canonical `GOTOOLCHAIN=local` Go 1.25 `go mod tidy -diff`, root build, vet, targeted Admin/API64 tests, and race remain `BLOCKED_ENVIRONMENT / NOT_RUN` because only Go 1.23.2 is installed and external Go toolchain/module retrieval is unavailable. Packaging/source-manifest integrity is not a substitute for those root compilation gates. No API-security authority changes are introduced: API-6/API-7 remain non-enforcing, API-8 retains explicit deterministic GraphQL ENFORCE authority, and OpenAI remains advisory only.
+
+API contract note: this maintenance fix changes only the versionless Admin Console Positive-Schema exception toggle endpoint from `/api/security/schema/enforcement/exceptions/{exception_id}` to `/api/security/schema/enforcement/exceptions/{exception_id}/toggle`. Positive-Schema authority, RBAC, persistence, audit semantics, API-4 enforcement, and API-8 GraphQL authority are unchanged. No API-9 is defined.
+
 <!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/API_SECURITY_CLOSURE_RESULT.md b/API_SECURITY_CLOSURE_RESULT.md
index ff4559e..37eac1a 100644
--- a/API_SECURITY_CLOSURE_RESULT.md
+++ b/API_SECURITY_CLOSURE_RESULT.md
@@ -249,4 +249,14 @@ blocking remains limited to explicit deterministic ENFORCE policy; OpenAI has no
 request-path authority. The cleanup removes only superseded/unwired source and
 centralizes shared SecLang parsing. Source gate: **80/80 PASS**.
 
+## 2026-09-30 — Build and Admin Startup Blocker Fix
+
+The current source fixes two release-blocking defects found during external Go 1.25 review of the 2026-09-28 consolidation baseline. `admin.go` now imports the standard-library `errors` package required by the existing `errors.New(...)` validation paths. Positive-Schema exception state changes now use the unambiguous route `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`; the shipping Console uses the same path. The prior `POST /api/security/schema/enforcement/exceptions/{exception_id}` registration is removed because it conflicts with `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+ `http.ServeMux` specificity rules and can panic during Admin handler construction.
+
+`admin_route_registration_test.go` now makes full Admin handler registration a regression test, and `tools/tests/test-build-startup-blocker-fix-source.py` is wired into `build.sh` and CI. The source gate also registers the complete 139-route Admin/update inventory with the Go standard library in a standalone temporary program; this catches wildcard-pattern ambiguity even when the repository module graph cannot be compiled locally. Current source evidence: blocker-fix gate **13/13 PASS**, route inventory **139/139 accepted by `ServeMux`**, API/source gates through API-8/post-audit/production-hardening/duplication remain PASS, OpenAI source contract and isolated tests PASS, package-source PASS, package-builder **9/9 PASS**, root Go source shape **96 files PASS**, Admin embedded JavaScript PASS, and changed Go files are `gofmt` clean.
+
+This does **not** promote the source to `TESTED` or `RELEASED`. On the current execution host, canonical `GOTOOLCHAIN=local` Go 1.25 `go mod tidy -diff`, root build, vet, targeted Admin/API64 tests, and race remain `BLOCKED_ENVIRONMENT / NOT_RUN` because only Go 1.23.2 is installed and external Go toolchain/module retrieval is unavailable. Packaging/source-manifest integrity is not a substitute for those root compilation gates. No API-security authority changes are introduced: API-6/API-7 remain non-enforcing, API-8 retains explicit deterministic GraphQL ENFORCE authority, and OpenAI remains advisory only.
+
+API contract note: this maintenance fix changes only the versionless Admin Console Positive-Schema exception toggle endpoint from `/api/security/schema/enforcement/exceptions/{exception_id}` to `/api/security/schema/enforcement/exceptions/{exception_id}/toggle`. Positive-Schema authority, RBAC, persistence, audit semantics, API-4 enforcement, and API-8 GraphQL authority are unchanged. No API-9 is defined.
+
 <!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/API_SECURITY_ROADMAP.md b/API_SECURITY_ROADMAP.md
index 047527c..7da6e3a 100644
--- a/API_SECURITY_ROADMAP.md
+++ b/API_SECURITY_ROADMAP.md
@@ -495,4 +495,14 @@ The code-duplication consolidation is a maintenance wave, not API-9 and not a
 change to the API-security roadmap. API-1 through API-8 remain
 `IMPLEMENTED_TESTING_DEFERRED`; no API-9 is defined.
 
+## 2026-09-30 — Build and Admin Startup Blocker Fix
+
+The current source fixes two release-blocking defects found during external Go 1.25 review of the 2026-09-28 consolidation baseline. `admin.go` now imports the standard-library `errors` package required by the existing `errors.New(...)` validation paths. Positive-Schema exception state changes now use the unambiguous route `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`; the shipping Console uses the same path. The prior `POST /api/security/schema/enforcement/exceptions/{exception_id}` registration is removed because it conflicts with `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+ `http.ServeMux` specificity rules and can panic during Admin handler construction.
+
+`admin_route_registration_test.go` now makes full Admin handler registration a regression test, and `tools/tests/test-build-startup-blocker-fix-source.py` is wired into `build.sh` and CI. The source gate also registers the complete 139-route Admin/update inventory with the Go standard library in a standalone temporary program; this catches wildcard-pattern ambiguity even when the repository module graph cannot be compiled locally. Current source evidence: blocker-fix gate **13/13 PASS**, route inventory **139/139 accepted by `ServeMux`**, API/source gates through API-8/post-audit/production-hardening/duplication remain PASS, OpenAI source contract and isolated tests PASS, package-source PASS, package-builder **9/9 PASS**, root Go source shape **96 files PASS**, Admin embedded JavaScript PASS, and changed Go files are `gofmt` clean.
+
+This does **not** promote the source to `TESTED` or `RELEASED`. On the current execution host, canonical `GOTOOLCHAIN=local` Go 1.25 `go mod tidy -diff`, root build, vet, targeted Admin/API64 tests, and race remain `BLOCKED_ENVIRONMENT / NOT_RUN` because only Go 1.23.2 is installed and external Go toolchain/module retrieval is unavailable. Packaging/source-manifest integrity is not a substitute for those root compilation gates. No API-security authority changes are introduced: API-6/API-7 remain non-enforcing, API-8 retains explicit deterministic GraphQL ENFORCE authority, and OpenAI remains advisory only.
+
+API contract note: this maintenance fix changes only the versionless Admin Console Positive-Schema exception toggle endpoint from `/api/security/schema/enforcement/exceptions/{exception_id}` to `/api/security/schema/enforcement/exceptions/{exception_id}/toggle`. Positive-Schema authority, RBAC, persistence, audit semantics, API-4 enforcement, and API-8 GraphQL authority are unchanged. No API-9 is defined.
+
 <!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/API_SECURITY_SLICE_IMPLEMENTATION_ROADMAP.md b/API_SECURITY_SLICE_IMPLEMENTATION_ROADMAP.md
index 8e9ba21..e028c37 100644
--- a/API_SECURITY_SLICE_IMPLEMENTATION_ROADMAP.md
+++ b/API_SECURITY_SLICE_IMPLEMENTATION_ROADMAP.md
@@ -274,4 +274,14 @@ The repository-wide duplicate-functionality cleanup does not add, remove or
 reorder API-security slices. API-1 through API-8 remain implemented with testing
 deferred. No API-9 has been defined.
 
+## 2026-09-30 — Build and Admin Startup Blocker Fix
+
+The current source fixes two release-blocking defects found during external Go 1.25 review of the 2026-09-28 consolidation baseline. `admin.go` now imports the standard-library `errors` package required by the existing `errors.New(...)` validation paths. Positive-Schema exception state changes now use the unambiguous route `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`; the shipping Console uses the same path. The prior `POST /api/security/schema/enforcement/exceptions/{exception_id}` registration is removed because it conflicts with `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+ `http.ServeMux` specificity rules and can panic during Admin handler construction.
+
+`admin_route_registration_test.go` now makes full Admin handler registration a regression test, and `tools/tests/test-build-startup-blocker-fix-source.py` is wired into `build.sh` and CI. The source gate also registers the complete 139-route Admin/update inventory with the Go standard library in a standalone temporary program; this catches wildcard-pattern ambiguity even when the repository module graph cannot be compiled locally. Current source evidence: blocker-fix gate **13/13 PASS**, route inventory **139/139 accepted by `ServeMux`**, API/source gates through API-8/post-audit/production-hardening/duplication remain PASS, OpenAI source contract and isolated tests PASS, package-source PASS, package-builder **9/9 PASS**, root Go source shape **96 files PASS**, Admin embedded JavaScript PASS, and changed Go files are `gofmt` clean.
+
+This does **not** promote the source to `TESTED` or `RELEASED`. On the current execution host, canonical `GOTOOLCHAIN=local` Go 1.25 `go mod tidy -diff`, root build, vet, targeted Admin/API64 tests, and race remain `BLOCKED_ENVIRONMENT / NOT_RUN` because only Go 1.23.2 is installed and external Go toolchain/module retrieval is unavailable. Packaging/source-manifest integrity is not a substitute for those root compilation gates. No API-security authority changes are introduced: API-6/API-7 remain non-enforcing, API-8 retains explicit deterministic GraphQL ENFORCE authority, and OpenAI remains advisory only.
+
+API contract note: this maintenance fix changes only the versionless Admin Console Positive-Schema exception toggle endpoint from `/api/security/schema/enforcement/exceptions/{exception_id}` to `/api/security/schema/enforcement/exceptions/{exception_id}/toggle`. Positive-Schema authority, RBAC, persistence, audit semantics, API-4 enforcement, and API-8 GraphQL authority are unchanged. No API-9 is defined.
+
 <!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/waf_patch_cur/BUILD_STARTUP_BLOCKER_FIX_TEST_SUMMARY.txt b/BUILD_STARTUP_BLOCKER_FIX_TEST_SUMMARY.txt
new file mode 100644
index 0000000..e6f5d1e
--- /dev/null
+++ b/BUILD_STARTUP_BLOCKER_FIX_TEST_SUMMARY.txt
@@ -0,0 +1,36 @@
+Build and Admin Startup Blocker Fix — 2026-09-30
+Status: IMPLEMENTED_TESTING_DEFERRED
+
+Parent source:
+  waf-proxy-code-duplication-consolidation-2026-09-28.zip
+  SHA-256 0756e864dd6ef8cc474477665666b2e3000a75cff18c49e627b0553dfd7400c5
+
+Implemented fixes:
+  - admin.go imports standard-library errors for existing errors.New paths.
+  - Positive-Schema exception toggle route is now:
+      POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle
+  - Shipping Console calls the same explicit toggle route.
+  - admin_route_registration_test.go fails on any Admin handler registration panic.
+  - dependency-free blocker source gate is wired into build.sh and CI and
+    registers the complete route inventory with stdlib http.ServeMux.
+
+Executed exact-source evidence:
+  BUILD_STARTUP_BLOCKER_FIX_SOURCE_GATE_PASS checks=13 routes=139
+  Standalone stdlib ServeMux registration: PASS, routes=139
+  API/source gates: 69/47/46/72/33/56/45/58/83/100/110/156/259/134/190/80 PASS
+  Markdown review gate: 86 files PASS
+  Root Go source shape: 96 files PASS
+  OpenAI source contract: 16/16 PASS; isolated tests PASS
+  WAF package-source: PASS
+  Package-builder tests: 9/9 PASS
+  Admin embedded JavaScript syntax: PASS
+  Changed Go gofmt: PASS
+
+Canonical Go qualification on this host:
+  GOTOOLCHAIN=local go mod tidy -diff: BLOCKED_ENVIRONMENT / NOT_RUN
+  GOTOOLCHAIN=local CGO_ENABLED=0 go build ./...: BLOCKED_ENVIRONMENT / NOT_RUN
+  GOTOOLCHAIN=local CGO_ENABLED=0 go vet ./...: BLOCKED_ENVIRONMENT / NOT_RUN
+  Targeted route/API64 Go tests: BLOCKED_ENVIRONMENT / NOT_RUN
+  Reason: host Go 1.23.2; go.mod requires Go 1.25.0; external toolchain/module retrieval unavailable.
+
+No TESTED/RELEASED/buildable claim is made from source or packaging gates.
diff --git a/CODE_DUPLICATION_REVIEW.md b/CODE_DUPLICATION_REVIEW.md
index 1dd74e1..0872d7f 100644
--- a/CODE_DUPLICATION_REVIEW.md
+++ b/CODE_DUPLICATION_REVIEW.md
@@ -182,4 +182,12 @@ the dedicated `test-code-duplication-review-source.py` gate. Canonical Go 1.25
 require the pinned Go 1.25 environment. This review must not be promoted to
 `TESTED` or `RELEASED` based only on source/static evidence.
 
+## 2026-09-30 — Build and Admin Startup Blocker Fix
+
+The current source fixes two release-blocking defects found during external Go 1.25 review of the 2026-09-28 consolidation baseline. `admin.go` now imports the standard-library `errors` package required by the existing `errors.New(...)` validation paths. Positive-Schema exception state changes now use the unambiguous route `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`; the shipping Console uses the same path. The prior `POST /api/security/schema/enforcement/exceptions/{exception_id}` registration is removed because it conflicts with `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+ `http.ServeMux` specificity rules and can panic during Admin handler construction.
+
+`admin_route_registration_test.go` now makes full Admin handler registration a regression test, and `tools/tests/test-build-startup-blocker-fix-source.py` is wired into `build.sh` and CI. The source gate also registers the complete 139-route Admin/update inventory with the Go standard library in a standalone temporary program; this catches wildcard-pattern ambiguity even when the repository module graph cannot be compiled locally. Current source evidence: blocker-fix gate **13/13 PASS**, route inventory **139/139 accepted by `ServeMux`**, API/source gates through API-8/post-audit/production-hardening/duplication remain PASS, OpenAI source contract and isolated tests PASS, package-source PASS, package-builder **9/9 PASS**, root Go source shape **96 files PASS**, Admin embedded JavaScript PASS, and changed Go files are `gofmt` clean.
+
+This does **not** promote the source to `TESTED` or `RELEASED`. On the current execution host, canonical `GOTOOLCHAIN=local` Go 1.25 `go mod tidy -diff`, root build, vet, targeted Admin/API64 tests, and race remain `BLOCKED_ENVIRONMENT / NOT_RUN` because only Go 1.23.2 is installed and external Go toolchain/module retrieval is unavailable. Packaging/source-manifest integrity is not a substitute for those root compilation gates. No API-security authority changes are introduced: API-6/API-7 remain non-enforcing, API-8 retains explicit deterministic GraphQL ENFORCE authority, and OpenAI remains advisory only.
+
 <!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/CODE_DUPLICATION_REVIEW_SOURCE_GATE_RESULT.md b/CODE_DUPLICATION_REVIEW_SOURCE_GATE_RESULT.md
index 3e2ef7a..62ca9fc 100644
--- a/CODE_DUPLICATION_REVIEW_SOURCE_GATE_RESULT.md
+++ b/CODE_DUPLICATION_REVIEW_SOURCE_GATE_RESULT.md
@@ -32,4 +32,12 @@ An isolated dependency-free test of `internal/capability` using the host Go
 1.23 toolchain passed after consolidation. That isolated test is advisory and
 is not a substitute for the repository's pinned Go 1.25 qualification.
 
+## 2026-09-30 — Build and Admin Startup Blocker Fix
+
+The current source fixes two release-blocking defects found during external Go 1.25 review of the 2026-09-28 consolidation baseline. `admin.go` now imports the standard-library `errors` package required by the existing `errors.New(...)` validation paths. Positive-Schema exception state changes now use the unambiguous route `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`; the shipping Console uses the same path. The prior `POST /api/security/schema/enforcement/exceptions/{exception_id}` registration is removed because it conflicts with `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+ `http.ServeMux` specificity rules and can panic during Admin handler construction.
+
+`admin_route_registration_test.go` now makes full Admin handler registration a regression test, and `tools/tests/test-build-startup-blocker-fix-source.py` is wired into `build.sh` and CI. The source gate also registers the complete 139-route Admin/update inventory with the Go standard library in a standalone temporary program; this catches wildcard-pattern ambiguity even when the repository module graph cannot be compiled locally. Current source evidence: blocker-fix gate **13/13 PASS**, route inventory **139/139 accepted by `ServeMux`**, API/source gates through API-8/post-audit/production-hardening/duplication remain PASS, OpenAI source contract and isolated tests PASS, package-source PASS, package-builder **9/9 PASS**, root Go source shape **96 files PASS**, Admin embedded JavaScript PASS, and changed Go files are `gofmt` clean.
+
+This does **not** promote the source to `TESTED` or `RELEASED`. On the current execution host, canonical `GOTOOLCHAIN=local` Go 1.25 `go mod tidy -diff`, root build, vet, targeted Admin/API64 tests, and race remain `BLOCKED_ENVIRONMENT / NOT_RUN` because only Go 1.23.2 is installed and external Go toolchain/module retrieval is unavailable. Packaging/source-manifest integrity is not a substitute for those root compilation gates. No API-security authority changes are introduced: API-6/API-7 remain non-enforcing, API-8 retains explicit deterministic GraphQL ENFORCE authority, and OpenAI remains advisory only.
+
 <!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/DEVELOPMENT.md b/DEVELOPMENT.md
index 4729cea..8ced0d1 100644
--- a/DEVELOPMENT.md
+++ b/DEVELOPMENT.md
@@ -523,4 +523,12 @@ reduces the root Go source-shape count to **95**; that reduction is intentional
 and results from removing dead/superseded root files. Canonical Go 1.25
 qualification remains blocked/not run.
 
+## 2026-09-30 — Build and Admin Startup Blocker Fix
+
+The current source fixes two release-blocking defects found during external Go 1.25 review of the 2026-09-28 consolidation baseline. `admin.go` now imports the standard-library `errors` package required by the existing `errors.New(...)` validation paths. Positive-Schema exception state changes now use the unambiguous route `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`; the shipping Console uses the same path. The prior `POST /api/security/schema/enforcement/exceptions/{exception_id}` registration is removed because it conflicts with `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+ `http.ServeMux` specificity rules and can panic during Admin handler construction.
+
+`admin_route_registration_test.go` now makes full Admin handler registration a regression test, and `tools/tests/test-build-startup-blocker-fix-source.py` is wired into `build.sh` and CI. The source gate also registers the complete 139-route Admin/update inventory with the Go standard library in a standalone temporary program; this catches wildcard-pattern ambiguity even when the repository module graph cannot be compiled locally. Current source evidence: blocker-fix gate **13/13 PASS**, route inventory **139/139 accepted by `ServeMux`**, API/source gates through API-8/post-audit/production-hardening/duplication remain PASS, OpenAI source contract and isolated tests PASS, package-source PASS, package-builder **9/9 PASS**, root Go source shape **96 files PASS**, Admin embedded JavaScript PASS, and changed Go files are `gofmt` clean.
+
+This does **not** promote the source to `TESTED` or `RELEASED`. On the current execution host, canonical `GOTOOLCHAIN=local` Go 1.25 `go mod tidy -diff`, root build, vet, targeted Admin/API64 tests, and race remain `BLOCKED_ENVIRONMENT / NOT_RUN` because only Go 1.23.2 is installed and external Go toolchain/module retrieval is unavailable. Packaging/source-manifest integrity is not a substitute for those root compilation gates. No API-security authority changes are introduced: API-6/API-7 remain non-enforcing, API-8 retains explicit deterministic GraphQL ENFORCE authority, and OpenAI remains advisory only.
+
 <!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/DEVELOPMENT_ROADMAP.md b/DEVELOPMENT_ROADMAP.md
index c1281d7..2eb8688 100644
--- a/DEVELOPMENT_ROADMAP.md
+++ b/DEVELOPMENT_ROADMAP.md
@@ -665,4 +665,12 @@ checkpoint on top of it. It removes superseded/unwired implementations and
 centralizes shared parsing without adding a product slice. API-1 through API-8
 remain `IMPLEMENTED_TESTING_DEFERRED`; no API-9 is defined.
 
+## 2026-09-30 — Build and Admin Startup Blocker Fix
+
+The current source fixes two release-blocking defects found during external Go 1.25 review of the 2026-09-28 consolidation baseline. `admin.go` now imports the standard-library `errors` package required by the existing `errors.New(...)` validation paths. Positive-Schema exception state changes now use the unambiguous route `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`; the shipping Console uses the same path. The prior `POST /api/security/schema/enforcement/exceptions/{exception_id}` registration is removed because it conflicts with `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+ `http.ServeMux` specificity rules and can panic during Admin handler construction.
+
+`admin_route_registration_test.go` now makes full Admin handler registration a regression test, and `tools/tests/test-build-startup-blocker-fix-source.py` is wired into `build.sh` and CI. The source gate also registers the complete 139-route Admin/update inventory with the Go standard library in a standalone temporary program; this catches wildcard-pattern ambiguity even when the repository module graph cannot be compiled locally. Current source evidence: blocker-fix gate **13/13 PASS**, route inventory **139/139 accepted by `ServeMux`**, API/source gates through API-8/post-audit/production-hardening/duplication remain PASS, OpenAI source contract and isolated tests PASS, package-source PASS, package-builder **9/9 PASS**, root Go source shape **96 files PASS**, Admin embedded JavaScript PASS, and changed Go files are `gofmt` clean.
+
+This does **not** promote the source to `TESTED` or `RELEASED`. On the current execution host, canonical `GOTOOLCHAIN=local` Go 1.25 `go mod tidy -diff`, root build, vet, targeted Admin/API64 tests, and race remain `BLOCKED_ENVIRONMENT / NOT_RUN` because only Go 1.23.2 is installed and external Go toolchain/module retrieval is unavailable. Packaging/source-manifest integrity is not a substitute for those root compilation gates. No API-security authority changes are introduced: API-6/API-7 remain non-enforcing, API-8 retains explicit deterministic GraphQL ENFORCE authority, and OpenAI remains advisory only.
+
 <!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/DOCUMENTATION_INDEX.md b/DOCUMENTATION_INDEX.md
index 7922473..e48d5a2 100644
--- a/DOCUMENTATION_INDEX.md
+++ b/DOCUMENTATION_INDEX.md
@@ -316,4 +316,12 @@ classification.
 New current documents: `CODE_DUPLICATION_REVIEW.md` and
 `CODE_DUPLICATION_REVIEW_SOURCE_GATE_RESULT.md`.
 
+## 2026-09-30 — Build and Admin Startup Blocker Fix
+
+The current source fixes two release-blocking defects found during external Go 1.25 review of the 2026-09-28 consolidation baseline. `admin.go` now imports the standard-library `errors` package required by the existing `errors.New(...)` validation paths. Positive-Schema exception state changes now use the unambiguous route `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`; the shipping Console uses the same path. The prior `POST /api/security/schema/enforcement/exceptions/{exception_id}` registration is removed because it conflicts with `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+ `http.ServeMux` specificity rules and can panic during Admin handler construction.
+
+`admin_route_registration_test.go` now makes full Admin handler registration a regression test, and `tools/tests/test-build-startup-blocker-fix-source.py` is wired into `build.sh` and CI. The source gate also registers the complete 139-route Admin/update inventory with the Go standard library in a standalone temporary program; this catches wildcard-pattern ambiguity even when the repository module graph cannot be compiled locally. Current source evidence: blocker-fix gate **13/13 PASS**, route inventory **139/139 accepted by `ServeMux`**, API/source gates through API-8/post-audit/production-hardening/duplication remain PASS, OpenAI source contract and isolated tests PASS, package-source PASS, package-builder **9/9 PASS**, root Go source shape **96 files PASS**, Admin embedded JavaScript PASS, and changed Go files are `gofmt` clean.
+
+This does **not** promote the source to `TESTED` or `RELEASED`. On the current execution host, canonical `GOTOOLCHAIN=local` Go 1.25 `go mod tidy -diff`, root build, vet, targeted Admin/API64 tests, and race remain `BLOCKED_ENVIRONMENT / NOT_RUN` because only Go 1.23.2 is installed and external Go toolchain/module retrieval is unavailable. Packaging/source-manifest integrity is not a substitute for those root compilation gates. No API-security authority changes are introduced: API-6/API-7 remain non-enforcing, API-8 retains explicit deterministic GraphQL ENFORCE authority, and OpenAI remains advisory only.
+
 <!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/HANDOVER_PROMPT.md b/HANDOVER_PROMPT.md
index d4b9bd9..550a8d1 100644
--- a/HANDOVER_PROMPT.md
+++ b/HANDOVER_PROMPT.md
@@ -161,4 +161,14 @@ API-6/API-7/API-8 authority boundaries and the single shipping Console. Do not
 reintroduce removed compatibility/place-holder files solely to satisfy stale
 historical references.
 
+## 2026-09-30 — Build and Admin Startup Blocker Fix
+
+The current source fixes two release-blocking defects found during external Go 1.25 review of the 2026-09-28 consolidation baseline. `admin.go` now imports the standard-library `errors` package required by the existing `errors.New(...)` validation paths. Positive-Schema exception state changes now use the unambiguous route `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`; the shipping Console uses the same path. The prior `POST /api/security/schema/enforcement/exceptions/{exception_id}` registration is removed because it conflicts with `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+ `http.ServeMux` specificity rules and can panic during Admin handler construction.
+
+`admin_route_registration_test.go` now makes full Admin handler registration a regression test, and `tools/tests/test-build-startup-blocker-fix-source.py` is wired into `build.sh` and CI. The source gate also registers the complete 139-route Admin/update inventory with the Go standard library in a standalone temporary program; this catches wildcard-pattern ambiguity even when the repository module graph cannot be compiled locally. Current source evidence: blocker-fix gate **13/13 PASS**, route inventory **139/139 accepted by `ServeMux`**, API/source gates through API-8/post-audit/production-hardening/duplication remain PASS, OpenAI source contract and isolated tests PASS, package-source PASS, package-builder **9/9 PASS**, root Go source shape **96 files PASS**, Admin embedded JavaScript PASS, and changed Go files are `gofmt` clean.
+
+This does **not** promote the source to `TESTED` or `RELEASED`. On the current execution host, canonical `GOTOOLCHAIN=local` Go 1.25 `go mod tidy -diff`, root build, vet, targeted Admin/API64 tests, and race remain `BLOCKED_ENVIRONMENT / NOT_RUN` because only Go 1.23.2 is installed and external Go toolchain/module retrieval is unavailable. Packaging/source-manifest integrity is not a substitute for those root compilation gates. No API-security authority changes are introduced: API-6/API-7 remain non-enforcing, API-8 retains explicit deterministic GraphQL ENFORCE authority, and OpenAI remains advisory only.
+
+Continuation point: use this blocker-fixed tree as the current source, not the 2026-09-28 consolidation ZIP. Do not restore the ambiguous exception-toggle route. The next qualification step is to run the exact final bytes under pinned Go 1.25 with `go mod tidy -diff`, `CGO_ENABLED=0 go build ./...`, `go vet ./...`, the targeted Admin route/API64 tests, full tests, and race tests before any buildable/PASS or release claim.
+
 <!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/HANDOVER_STATUS.md b/HANDOVER_STATUS.md
index 8251506..39f1797 100644
--- a/HANDOVER_STATUS.md
+++ b/HANDOVER_STATUS.md
@@ -199,4 +199,14 @@ Admin/update routes are unique, root Go source shape is **95 files PASS**, and
 all prior API/source/package gates listed in the current canonical section remain
 PASS. Canonical Go 1.25 qualification remains blocked/not run.
 
+## 2026-09-30 — Build and Admin Startup Blocker Fix
+
+The current source fixes two release-blocking defects found during external Go 1.25 review of the 2026-09-28 consolidation baseline. `admin.go` now imports the standard-library `errors` package required by the existing `errors.New(...)` validation paths. Positive-Schema exception state changes now use the unambiguous route `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`; the shipping Console uses the same path. The prior `POST /api/security/schema/enforcement/exceptions/{exception_id}` registration is removed because it conflicts with `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+ `http.ServeMux` specificity rules and can panic during Admin handler construction.
+
+`admin_route_registration_test.go` now makes full Admin handler registration a regression test, and `tools/tests/test-build-startup-blocker-fix-source.py` is wired into `build.sh` and CI. The source gate also registers the complete 139-route Admin/update inventory with the Go standard library in a standalone temporary program; this catches wildcard-pattern ambiguity even when the repository module graph cannot be compiled locally. Current source evidence: blocker-fix gate **13/13 PASS**, route inventory **139/139 accepted by `ServeMux`**, API/source gates through API-8/post-audit/production-hardening/duplication remain PASS, OpenAI source contract and isolated tests PASS, package-source PASS, package-builder **9/9 PASS**, root Go source shape **96 files PASS**, Admin embedded JavaScript PASS, and changed Go files are `gofmt` clean.
+
+This does **not** promote the source to `TESTED` or `RELEASED`. On the current execution host, canonical `GOTOOLCHAIN=local` Go 1.25 `go mod tidy -diff`, root build, vet, targeted Admin/API64 tests, and race remain `BLOCKED_ENVIRONMENT / NOT_RUN` because only Go 1.23.2 is installed and external Go toolchain/module retrieval is unavailable. Packaging/source-manifest integrity is not a substitute for those root compilation gates. No API-security authority changes are introduced: API-6/API-7 remain non-enforcing, API-8 retains explicit deterministic GraphQL ENFORCE authority, and OpenAI remains advisory only.
+
+Continuation point: use this blocker-fixed tree as the current source, not the 2026-09-28 consolidation ZIP. Do not restore the ambiguous exception-toggle route. The next qualification step is to run the exact final bytes under pinned Go 1.25 with `go mod tidy -diff`, `CGO_ENABLED=0 go build ./...`, `go vet ./...`, the targeted Admin route/API64 tests, full tests, and race tests before any buildable/PASS or release claim.
+
 <!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/INSTALL.md b/INSTALL.md
index 822142d..46fc0bf 100644
--- a/INSTALL.md
+++ b/INSTALL.md
@@ -953,4 +953,12 @@ The 2026-09-28 duplication consolidation does not change installation paths,
 persistent `/etc/waf`/state preservation requirements, or package lifecycle
 semantics.
 
+## 2026-09-30 — Build and Admin Startup Blocker Fix
+
+The current source fixes two release-blocking defects found during external Go 1.25 review of the 2026-09-28 consolidation baseline. `admin.go` now imports the standard-library `errors` package required by the existing `errors.New(...)` validation paths. Positive-Schema exception state changes now use the unambiguous route `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`; the shipping Console uses the same path. The prior `POST /api/security/schema/enforcement/exceptions/{exception_id}` registration is removed because it conflicts with `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+ `http.ServeMux` specificity rules and can panic during Admin handler construction.
+
+`admin_route_registration_test.go` now makes full Admin handler registration a regression test, and `tools/tests/test-build-startup-blocker-fix-source.py` is wired into `build.sh` and CI. The source gate also registers the complete 139-route Admin/update inventory with the Go standard library in a standalone temporary program; this catches wildcard-pattern ambiguity even when the repository module graph cannot be compiled locally. Current source evidence: blocker-fix gate **13/13 PASS**, route inventory **139/139 accepted by `ServeMux`**, API/source gates through API-8/post-audit/production-hardening/duplication remain PASS, OpenAI source contract and isolated tests PASS, package-source PASS, package-builder **9/9 PASS**, root Go source shape **96 files PASS**, Admin embedded JavaScript PASS, and changed Go files are `gofmt` clean.
+
+This does **not** promote the source to `TESTED` or `RELEASED`. On the current execution host, canonical `GOTOOLCHAIN=local` Go 1.25 `go mod tidy -diff`, root build, vet, targeted Admin/API64 tests, and race remain `BLOCKED_ENVIRONMENT / NOT_RUN` because only Go 1.23.2 is installed and external Go toolchain/module retrieval is unavailable. Packaging/source-manifest integrity is not a substitute for those root compilation gates. No API-security authority changes are introduced: API-6/API-7 remain non-enforcing, API-8 retains explicit deterministic GraphQL ENFORCE authority, and OpenAI remains advisory only.
+
 <!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/MANIFEST.md b/MANIFEST.md
index ec0abe5..3b23563 100644
--- a/MANIFEST.md
+++ b/MANIFEST.md
@@ -605,4 +605,14 @@ Added `debug_sanitize.go`, `internal/capability/seclang.go` plus its test,
 `CODE_DUPLICATION_REVIEW_SOURCE_GATE_RESULT.md`. Root Go source-shape is now
 **95 files** after intentional cleanup.
 
+## 2026-09-30 — Build and Admin Startup Blocker Fix
+
+The current source fixes two release-blocking defects found during external Go 1.25 review of the 2026-09-28 consolidation baseline. `admin.go` now imports the standard-library `errors` package required by the existing `errors.New(...)` validation paths. Positive-Schema exception state changes now use the unambiguous route `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`; the shipping Console uses the same path. The prior `POST /api/security/schema/enforcement/exceptions/{exception_id}` registration is removed because it conflicts with `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+ `http.ServeMux` specificity rules and can panic during Admin handler construction.
+
+`admin_route_registration_test.go` now makes full Admin handler registration a regression test, and `tools/tests/test-build-startup-blocker-fix-source.py` is wired into `build.sh` and CI. The source gate also registers the complete 139-route Admin/update inventory with the Go standard library in a standalone temporary program; this catches wildcard-pattern ambiguity even when the repository module graph cannot be compiled locally. Current source evidence: blocker-fix gate **13/13 PASS**, route inventory **139/139 accepted by `ServeMux`**, API/source gates through API-8/post-audit/production-hardening/duplication remain PASS, OpenAI source contract and isolated tests PASS, package-source PASS, package-builder **9/9 PASS**, root Go source shape **96 files PASS**, Admin embedded JavaScript PASS, and changed Go files are `gofmt` clean.
+
+This does **not** promote the source to `TESTED` or `RELEASED`. On the current execution host, canonical `GOTOOLCHAIN=local` Go 1.25 `go mod tidy -diff`, root build, vet, targeted Admin/API64 tests, and race remain `BLOCKED_ENVIRONMENT / NOT_RUN` because only Go 1.23.2 is installed and external Go toolchain/module retrieval is unavailable. Packaging/source-manifest integrity is not a substitute for those root compilation gates. No API-security authority changes are introduced: API-6/API-7 remain non-enforcing, API-8 retains explicit deterministic GraphQL ENFORCE authority, and OpenAI remains advisory only.
+
+New/changed source surfaces include `admin_route_registration_test.go`, `tools/tests/test-build-startup-blocker-fix-source.py`, `admin.go`, `static/admin.html`, `build.sh`, and `.github/workflows/ci.yml`. The final delivery manifest and release artifact must be regenerated after these changes; historical manifest hashes remain historical evidence only.
+
 <!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/NEXT_CHAT_HANDOVER.md b/NEXT_CHAT_HANDOVER.md
index 37cc79f..611e1b8 100644
--- a/NEXT_CHAT_HANDOVER.md
+++ b/NEXT_CHAT_HANDOVER.md
@@ -161,4 +161,14 @@ SecLang tokenizer/action helper authority shared by coverage and VectorScan.
 No API-9 is defined; next product promotion work remains pinned Go 1.25 and live
 production qualification.
 
+## 2026-09-30 — Build and Admin Startup Blocker Fix
+
+The current source fixes two release-blocking defects found during external Go 1.25 review of the 2026-09-28 consolidation baseline. `admin.go` now imports the standard-library `errors` package required by the existing `errors.New(...)` validation paths. Positive-Schema exception state changes now use the unambiguous route `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`; the shipping Console uses the same path. The prior `POST /api/security/schema/enforcement/exceptions/{exception_id}` registration is removed because it conflicts with `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+ `http.ServeMux` specificity rules and can panic during Admin handler construction.
+
+`admin_route_registration_test.go` now makes full Admin handler registration a regression test, and `tools/tests/test-build-startup-blocker-fix-source.py` is wired into `build.sh` and CI. The source gate also registers the complete 139-route Admin/update inventory with the Go standard library in a standalone temporary program; this catches wildcard-pattern ambiguity even when the repository module graph cannot be compiled locally. Current source evidence: blocker-fix gate **13/13 PASS**, route inventory **139/139 accepted by `ServeMux`**, API/source gates through API-8/post-audit/production-hardening/duplication remain PASS, OpenAI source contract and isolated tests PASS, package-source PASS, package-builder **9/9 PASS**, root Go source shape **96 files PASS**, Admin embedded JavaScript PASS, and changed Go files are `gofmt` clean.
+
+This does **not** promote the source to `TESTED` or `RELEASED`. On the current execution host, canonical `GOTOOLCHAIN=local` Go 1.25 `go mod tidy -diff`, root build, vet, targeted Admin/API64 tests, and race remain `BLOCKED_ENVIRONMENT / NOT_RUN` because only Go 1.23.2 is installed and external Go toolchain/module retrieval is unavailable. Packaging/source-manifest integrity is not a substitute for those root compilation gates. No API-security authority changes are introduced: API-6/API-7 remain non-enforcing, API-8 retains explicit deterministic GraphQL ENFORCE authority, and OpenAI remains advisory only.
+
+Continuation point: use this blocker-fixed tree as the current source, not the 2026-09-28 consolidation ZIP. Do not restore the ambiguous exception-toggle route. The next qualification step is to run the exact final bytes under pinned Go 1.25 with `go mod tidy -diff`, `CGO_ENABLED=0 go build ./...`, `go vet ./...`, the targeted Admin route/API64 tests, full tests, and race tests before any buildable/PASS or release claim.
+
 <!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/PACKAGING_TOOL.md b/PACKAGING_TOOL.md
index 5f5790c..4dcb51d 100644
--- a/PACKAGING_TOOL.md
+++ b/PACKAGING_TOOL.md
@@ -233,4 +233,14 @@ builds must preserve and execute it alongside the existing API, production
 hardening, OpenAI, package-source and artifact-integrity gates. The cleanup does
 not create a second packaging path.
 
+## 2026-09-30 — Build and Admin Startup Blocker Fix
+
+The current source fixes two release-blocking defects found during external Go 1.25 review of the 2026-09-28 consolidation baseline. `admin.go` now imports the standard-library `errors` package required by the existing `errors.New(...)` validation paths. Positive-Schema exception state changes now use the unambiguous route `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`; the shipping Console uses the same path. The prior `POST /api/security/schema/enforcement/exceptions/{exception_id}` registration is removed because it conflicts with `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+ `http.ServeMux` specificity rules and can panic during Admin handler construction.
+
+`admin_route_registration_test.go` now makes full Admin handler registration a regression test, and `tools/tests/test-build-startup-blocker-fix-source.py` is wired into `build.sh` and CI. The source gate also registers the complete 139-route Admin/update inventory with the Go standard library in a standalone temporary program; this catches wildcard-pattern ambiguity even when the repository module graph cannot be compiled locally. Current source evidence: blocker-fix gate **13/13 PASS**, route inventory **139/139 accepted by `ServeMux`**, API/source gates through API-8/post-audit/production-hardening/duplication remain PASS, OpenAI source contract and isolated tests PASS, package-source PASS, package-builder **9/9 PASS**, root Go source shape **96 files PASS**, Admin embedded JavaScript PASS, and changed Go files are `gofmt` clean.
+
+This does **not** promote the source to `TESTED` or `RELEASED`. On the current execution host, canonical `GOTOOLCHAIN=local` Go 1.25 `go mod tidy -diff`, root build, vet, targeted Admin/API64 tests, and race remain `BLOCKED_ENVIRONMENT / NOT_RUN` because only Go 1.23.2 is installed and external Go toolchain/module retrieval is unavailable. Packaging/source-manifest integrity is not a substitute for those root compilation gates. No API-security authority changes are introduced: API-6/API-7 remain non-enforcing, API-8 retains explicit deterministic GraphQL ENFORCE authority, and OpenAI remains advisory only.
+
+Release gate clarification: `SOURCE_BASELINE_GATE_RESULT` and archive integrity prove source/package completeness only. Release/buildability requires the exact artifact bytes to pass the pinned Go 1.25 root tidy/build/vet/test gates. The new Admin route-registration regression and blocker source gate are mandatory pre-compilation gates but are not substitutes for compilation.
+
 <!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/PRODUCTION_CONTROL_PLANE_HARDENING.md b/PRODUCTION_CONTROL_PLANE_HARDENING.md
index b3c2216..36f949a 100644
--- a/PRODUCTION_CONTROL_PLANE_HARDENING.md
+++ b/PRODUCTION_CONTROL_PLANE_HARDENING.md
@@ -70,4 +70,12 @@ the authority for the 2026-09-25 control-plane hardening behavior;
 `CODE_DUPLICATION_REVIEW.md` is the authority for the later source-topology
 cleanup.
 
+## 2026-09-30 — Build and Admin Startup Blocker Fix
+
+The current source fixes two release-blocking defects found during external Go 1.25 review of the 2026-09-28 consolidation baseline. `admin.go` now imports the standard-library `errors` package required by the existing `errors.New(...)` validation paths. Positive-Schema exception state changes now use the unambiguous route `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`; the shipping Console uses the same path. The prior `POST /api/security/schema/enforcement/exceptions/{exception_id}` registration is removed because it conflicts with `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+ `http.ServeMux` specificity rules and can panic during Admin handler construction.
+
+`admin_route_registration_test.go` now makes full Admin handler registration a regression test, and `tools/tests/test-build-startup-blocker-fix-source.py` is wired into `build.sh` and CI. The source gate also registers the complete 139-route Admin/update inventory with the Go standard library in a standalone temporary program; this catches wildcard-pattern ambiguity even when the repository module graph cannot be compiled locally. Current source evidence: blocker-fix gate **13/13 PASS**, route inventory **139/139 accepted by `ServeMux`**, API/source gates through API-8/post-audit/production-hardening/duplication remain PASS, OpenAI source contract and isolated tests PASS, package-source PASS, package-builder **9/9 PASS**, root Go source shape **96 files PASS**, Admin embedded JavaScript PASS, and changed Go files are `gofmt` clean.
+
+This does **not** promote the source to `TESTED` or `RELEASED`. On the current execution host, canonical `GOTOOLCHAIN=local` Go 1.25 `go mod tidy -diff`, root build, vet, targeted Admin/API64 tests, and race remain `BLOCKED_ENVIRONMENT / NOT_RUN` because only Go 1.23.2 is installed and external Go toolchain/module retrieval is unavailable. Packaging/source-manifest integrity is not a substitute for those root compilation gates. No API-security authority changes are introduced: API-6/API-7 remain non-enforcing, API-8 retains explicit deterministic GraphQL ENFORCE authority, and OpenAI remains advisory only.
+
 <!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/README.md b/README.md
index 85c10cc..8b81bb9 100644
--- a/README.md
+++ b/README.md
@@ -1370,4 +1370,12 @@ separate state, persistence and worker boundaries. See
 `CODE_DUPLICATION_REVIEW.md` and
 `CODE_DUPLICATION_REVIEW_SOURCE_GATE_RESULT.md`.
 
+## 2026-09-30 — Build and Admin Startup Blocker Fix
+
+The current source fixes two release-blocking defects found during external Go 1.25 review of the 2026-09-28 consolidation baseline. `admin.go` now imports the standard-library `errors` package required by the existing `errors.New(...)` validation paths. Positive-Schema exception state changes now use the unambiguous route `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`; the shipping Console uses the same path. The prior `POST /api/security/schema/enforcement/exceptions/{exception_id}` registration is removed because it conflicts with `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+ `http.ServeMux` specificity rules and can panic during Admin handler construction.
+
+`admin_route_registration_test.go` now makes full Admin handler registration a regression test, and `tools/tests/test-build-startup-blocker-fix-source.py` is wired into `build.sh` and CI. The source gate also registers the complete 139-route Admin/update inventory with the Go standard library in a standalone temporary program; this catches wildcard-pattern ambiguity even when the repository module graph cannot be compiled locally. Current source evidence: blocker-fix gate **13/13 PASS**, route inventory **139/139 accepted by `ServeMux`**, API/source gates through API-8/post-audit/production-hardening/duplication remain PASS, OpenAI source contract and isolated tests PASS, package-source PASS, package-builder **9/9 PASS**, root Go source shape **96 files PASS**, Admin embedded JavaScript PASS, and changed Go files are `gofmt` clean.
+
+This does **not** promote the source to `TESTED` or `RELEASED`. On the current execution host, canonical `GOTOOLCHAIN=local` Go 1.25 `go mod tidy -diff`, root build, vet, targeted Admin/API64 tests, and race remain `BLOCKED_ENVIRONMENT / NOT_RUN` because only Go 1.23.2 is installed and external Go toolchain/module retrieval is unavailable. Packaging/source-manifest integrity is not a substitute for those root compilation gates. No API-security authority changes are introduced: API-6/API-7 remain non-enforcing, API-8 retains explicit deterministic GraphQL ENFORCE authority, and OpenAI remains advisory only.
+
 <!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/RELEASE_PROCESS.md b/RELEASE_PROCESS.md
index f4b7d2e..aa028c9 100644
--- a/RELEASE_PROCESS.md
+++ b/RELEASE_PROCESS.md
@@ -401,4 +401,14 @@ authorities remain wired, Admin routes remain unique, and API authority
 boundaries are unchanged. It is additive to, not a replacement for, pinned Go
 1.25 and target-host qualification.
 
+## 2026-09-30 — Build and Admin Startup Blocker Fix
+
+The current source fixes two release-blocking defects found during external Go 1.25 review of the 2026-09-28 consolidation baseline. `admin.go` now imports the standard-library `errors` package required by the existing `errors.New(...)` validation paths. Positive-Schema exception state changes now use the unambiguous route `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`; the shipping Console uses the same path. The prior `POST /api/security/schema/enforcement/exceptions/{exception_id}` registration is removed because it conflicts with `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+ `http.ServeMux` specificity rules and can panic during Admin handler construction.
+
+`admin_route_registration_test.go` now makes full Admin handler registration a regression test, and `tools/tests/test-build-startup-blocker-fix-source.py` is wired into `build.sh` and CI. The source gate also registers the complete 139-route Admin/update inventory with the Go standard library in a standalone temporary program; this catches wildcard-pattern ambiguity even when the repository module graph cannot be compiled locally. Current source evidence: blocker-fix gate **13/13 PASS**, route inventory **139/139 accepted by `ServeMux`**, API/source gates through API-8/post-audit/production-hardening/duplication remain PASS, OpenAI source contract and isolated tests PASS, package-source PASS, package-builder **9/9 PASS**, root Go source shape **96 files PASS**, Admin embedded JavaScript PASS, and changed Go files are `gofmt` clean.
+
+This does **not** promote the source to `TESTED` or `RELEASED`. On the current execution host, canonical `GOTOOLCHAIN=local` Go 1.25 `go mod tidy -diff`, root build, vet, targeted Admin/API64 tests, and race remain `BLOCKED_ENVIRONMENT / NOT_RUN` because only Go 1.23.2 is installed and external Go toolchain/module retrieval is unavailable. Packaging/source-manifest integrity is not a substitute for those root compilation gates. No API-security authority changes are introduced: API-6/API-7 remain non-enforcing, API-8 retains explicit deterministic GraphQL ENFORCE authority, and OpenAI remains advisory only.
+
+Release gate clarification: `SOURCE_BASELINE_GATE_RESULT` and archive integrity prove source/package completeness only. Release/buildability requires the exact artifact bytes to pass the pinned Go 1.25 root tidy/build/vet/test gates. The new Admin route-registration regression and blocker source gate are mandatory pre-compilation gates but are not substitutes for compilation.
+
 <!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/TESTING.md b/TESTING.md
index ea24dcb..e660e21 100644
--- a/TESTING.md
+++ b/TESTING.md
@@ -453,4 +453,14 @@ boundaries. The gate must run in both source and clean-extract validation.
 Canonical Go 1.25 `tidy`, build, vet, full tests and race remain mandatory for
 release promotion and are not replaced by the duplication source gate.
 
+## 2026-09-30 — Build and Admin Startup Blocker Fix
+
+The current source fixes two release-blocking defects found during external Go 1.25 review of the 2026-09-28 consolidation baseline. `admin.go` now imports the standard-library `errors` package required by the existing `errors.New(...)` validation paths. Positive-Schema exception state changes now use the unambiguous route `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`; the shipping Console uses the same path. The prior `POST /api/security/schema/enforcement/exceptions/{exception_id}` registration is removed because it conflicts with `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+ `http.ServeMux` specificity rules and can panic during Admin handler construction.
+
+`admin_route_registration_test.go` now makes full Admin handler registration a regression test, and `tools/tests/test-build-startup-blocker-fix-source.py` is wired into `build.sh` and CI. The source gate also registers the complete 139-route Admin/update inventory with the Go standard library in a standalone temporary program; this catches wildcard-pattern ambiguity even when the repository module graph cannot be compiled locally. Current source evidence: blocker-fix gate **13/13 PASS**, route inventory **139/139 accepted by `ServeMux`**, API/source gates through API-8/post-audit/production-hardening/duplication remain PASS, OpenAI source contract and isolated tests PASS, package-source PASS, package-builder **9/9 PASS**, root Go source shape **96 files PASS**, Admin embedded JavaScript PASS, and changed Go files are `gofmt` clean.
+
+This does **not** promote the source to `TESTED` or `RELEASED`. On the current execution host, canonical `GOTOOLCHAIN=local` Go 1.25 `go mod tidy -diff`, root build, vet, targeted Admin/API64 tests, and race remain `BLOCKED_ENVIRONMENT / NOT_RUN` because only Go 1.23.2 is installed and external Go toolchain/module retrieval is unavailable. Packaging/source-manifest integrity is not a substitute for those root compilation gates. No API-security authority changes are introduced: API-6/API-7 remain non-enforcing, API-8 retains explicit deterministic GraphQL ENFORCE authority, and OpenAI remains advisory only.
+
+Release gate clarification: `SOURCE_BASELINE_GATE_RESULT` and archive integrity prove source/package completeness only. Release/buildability requires the exact artifact bytes to pass the pinned Go 1.25 root tidy/build/vet/test gates. The new Admin route-registration regression and blocker source gate are mandatory pre-compilation gates but are not substitutes for compilation.
+
 <!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/TESTING_RESULTS.md b/TESTING_RESULTS.md
index a9cbb85..6bff8ef 100644
--- a/TESTING_RESULTS.md
+++ b/TESTING_RESULTS.md
@@ -1148,4 +1148,14 @@ new shared `internal/capability` package also passes an isolated dependency-free
 unit test under the host Go 1.23 toolchain; this is advisory only. Canonical Go
 1.25 repository qualification remains `BLOCKED_ENVIRONMENT / NOT_RUN`.
 
+## 2026-09-30 — Build and Admin Startup Blocker Fix
+
+The current source fixes two release-blocking defects found during external Go 1.25 review of the 2026-09-28 consolidation baseline. `admin.go` now imports the standard-library `errors` package required by the existing `errors.New(...)` validation paths. Positive-Schema exception state changes now use the unambiguous route `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`; the shipping Console uses the same path. The prior `POST /api/security/schema/enforcement/exceptions/{exception_id}` registration is removed because it conflicts with `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+ `http.ServeMux` specificity rules and can panic during Admin handler construction.
+
+`admin_route_registration_test.go` now makes full Admin handler registration a regression test, and `tools/tests/test-build-startup-blocker-fix-source.py` is wired into `build.sh` and CI. The source gate also registers the complete 139-route Admin/update inventory with the Go standard library in a standalone temporary program; this catches wildcard-pattern ambiguity even when the repository module graph cannot be compiled locally. Current source evidence: blocker-fix gate **13/13 PASS**, route inventory **139/139 accepted by `ServeMux`**, API/source gates through API-8/post-audit/production-hardening/duplication remain PASS, OpenAI source contract and isolated tests PASS, package-source PASS, package-builder **9/9 PASS**, root Go source shape **96 files PASS**, Admin embedded JavaScript PASS, and changed Go files are `gofmt` clean.
+
+This does **not** promote the source to `TESTED` or `RELEASED`. On the current execution host, canonical `GOTOOLCHAIN=local` Go 1.25 `go mod tidy -diff`, root build, vet, targeted Admin/API64 tests, and race remain `BLOCKED_ENVIRONMENT / NOT_RUN` because only Go 1.23.2 is installed and external Go toolchain/module retrieval is unavailable. Packaging/source-manifest integrity is not a substitute for those root compilation gates. No API-security authority changes are introduced: API-6/API-7 remain non-enforcing, API-8 retains explicit deterministic GraphQL ENFORCE authority, and OpenAI remains advisory only.
+
+Release gate clarification: `SOURCE_BASELINE_GATE_RESULT` and archive integrity prove source/package completeness only. Release/buildability requires the exact artifact bytes to pass the pinned Go 1.25 root tidy/build/vet/test gates. The new Admin route-registration regression and blocker source gate are mandatory pre-compilation gates but are not substitutes for compilation.
+
 <!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
diff --git a/admin.go b/admin.go
index c7c9360..c6c2822 100644
--- a/admin.go
+++ b/admin.go
@@ -6,6 +6,7 @@ import (
 	"crypto/subtle"
 	"encoding/hex"
 	"encoding/json"
+	"errors"
 	"fmt"
 	"log/slog"
 	"net"
@@ -357,7 +358,7 @@ func (a *adminServer) handler() http.Handler {
 	mux.HandleFunc("POST /api/security/schema/enforcement/{operation_id}/activate", a.authRole(roleReviewer, a.handlePositiveSchemaActivate))
 	mux.HandleFunc("POST /api/security/schema/enforcement/{operation_id}/rollback", a.authRole(roleReviewer, a.handlePositiveSchemaRollback))
 	mux.HandleFunc("POST /api/security/schema/enforcement/{operation_id}/exceptions", a.authRole(roleReviewer, a.handlePositiveSchemaExceptionCreate))
-	mux.HandleFunc("POST /api/security/schema/enforcement/exceptions/{exception_id}", a.authRole(roleReviewer, a.handlePositiveSchemaExceptionToggle))
+	mux.HandleFunc("POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle", a.authRole(roleReviewer, a.handlePositiveSchemaExceptionToggle))
 	mux.HandleFunc("GET /api/security/schema/{id}", a.auth(a.handleSchemaDetail))
 	mux.HandleFunc("POST /api/security/schema/{id}/review", a.authRole(roleReviewer, a.handleSchemaReview))
 	mux.HandleFunc("POST /api/security/schema/{id}/ai-review", a.authRole(roleReviewer, a.handleSchemaAIReview))
diff --git a/waf_patch_cur/admin_route_registration_test.go b/admin_route_registration_test.go
new file mode 100644
index 0000000..6101dd0
--- /dev/null
+++ b/admin_route_registration_test.go
@@ -0,0 +1,15 @@
+package main
+
+import "testing"
+
+// TestAdminHandlerRouteRegistrationNoPanic protects startup against ambiguous
+// net/http ServeMux patterns. A route conflict panics during handler wiring,
+// before the admin server can start serving requests.
+func TestAdminHandlerRouteRegistrationNoPanic(t *testing.T) {
+	defer func() {
+		if r := recover(); r != nil {
+			t.Fatalf("admin handler route registration panicked: %v", r)
+		}
+	}()
+	_ = (&adminServer{}).handler()
+}
diff --git a/build.sh b/build.sh
index c3ac156..5aea528 100755
--- a/build.sh
+++ b/build.sh
@@ -32,6 +32,7 @@ python3 ./tools/tests/test-api8-source.py
 python3 ./tools/tests/test-api8-post-audit-hardening-source.py
 python3 ./tools/tests/test-production-control-plane-hardening-source.py
 python3 ./tools/tests/test-code-duplication-review-source.py
+python3 ./tools/tests/test-build-startup-blocker-fix-source.py
 python3 ./tools/tests/test-markdown-review-source.py
 
 VERSION="${VERSION:-$(date -u +%Y.%m.%d)}"
diff --git a/static/admin.html b/static/admin.html
index 020f3ba..987c32c 100644
--- a/static/admin.html
+++ b/static/admin.html
@@ -1652,7 +1652,7 @@
       <div class="eyebrow" style="margin-top:14px">Exceptions</div><div style="overflow-x:auto"><table><thead><tr><th>State</th><th>Type</th><th>Field</th><th>Reason</th><th>Expires</th><th></th></tr></thead><tbody>${exceptions.map(x=>`<tr><td>${x.enabled?'<span class="tag src-seen">enabled</span>':'<span class="tag">disabled</span>'}</td><td>${esc(x.violation_type||'all')}</td><td class="mono">${esc(x.field||'all')}</td><td>${esc(x.reason||'—')}</td><td>${esc(x.expires_at||'—')}</td><td><button class="btn sm api-exception-toggle" data-id="${esc(x.id)}" data-enabled="${x.enabled?'1':'0'}">${x.enabled?'Disable':'Enable'}</button></td></tr>`).join('')||'<tr><td colspan="6">No exceptions.</td></tr>'}</tbody></table></div>`;
     $("api_enforce_detail_close").onclick=()=>box.style.display="none";
     box.querySelectorAll(".api-profile-activate").forEach(b=>b.onclick=async()=>{const reason=(prompt("Reason for activating this schema profile version","operator-selected profile version")||"").trim();if(!reason)return;try{await api(`/api/security/schema/enforcement/${encodeURIComponent(operationID)}/activate`,{method:"POST",body:JSON.stringify({profile_version_id:b.dataset.id,reason})});toast("positive schema version activated in LEARN");await loadPositiveSchemaDetail(operationID);loadAPISecurity();}catch(e){toast(e.message,"err");}});
-    box.querySelectorAll(".api-exception-toggle").forEach(b=>b.onclick=async()=>{try{await api(`/api/security/schema/enforcement/exceptions/${encodeURIComponent(b.dataset.id)}`,{method:"POST",body:JSON.stringify({enabled:b.dataset.enabled!=="1"})});toast("exception state updated");await loadPositiveSchemaDetail(operationID);loadAPISecurity();}catch(e){toast(e.message,"err");}});
+    box.querySelectorAll(".api-exception-toggle").forEach(b=>b.onclick=async()=>{try{await api(`/api/security/schema/enforcement/exceptions/${encodeURIComponent(b.dataset.id)}/toggle`,{method:"POST",body:JSON.stringify({enabled:b.dataset.enabled!=="1"})});toast("exception state updated");await loadPositiveSchemaDetail(operationID);loadAPISecurity();}catch(e){toast(e.message,"err");}});
   }
   async function loadOpenAPIContract(contractID){
     const box=$("api_contract_detail"); const [c,versions,bindings,diffs]=await Promise.all([api(`/api/security/contracts/${encodeURIComponent(contractID)}`),api(`/api/security/contracts/${encodeURIComponent(contractID)}/versions`),api(`/api/security/contracts/${encodeURIComponent(contractID)}/bindings`),api(`/api/security/contracts/${encodeURIComponent(contractID)}/diffs`)]);
diff --git a/waf_patch_cur/tools/tests/test-build-startup-blocker-fix-source.py b/tools/tests/test-build-startup-blocker-fix-source.py
new file mode 100755
index 0000000..4d03989
--- /dev/null
+++ b/tools/tests/test-build-startup-blocker-fix-source.py
@@ -0,0 +1,73 @@
+#!/usr/bin/env python3
+from pathlib import Path
+import os
+import re
+import subprocess
+import tempfile
+
+ROOT = Path(__file__).resolve().parents[2]
+read = lambda p: (ROOT / p).read_text(encoding="utf-8")
+checks = []
+
+def req(cond, msg):
+    if not cond:
+        raise SystemExit("BUILD_STARTUP_BLOCKER_FIX_SOURCE_GATE_FAIL: " + msg)
+    checks.append(msg)
+
+admin = read("admin.go")
+ui = read("static/admin.html")
+reg = read("admin_route_registration_test.go")
+build = read("build.sh")
+ci = read(".github/workflows/ci.yml")
+
+# Build blocker: errors.New users must have the standard-library import.
+req(re.search(r'(?m)^\s*"errors"\s*$', admin) is not None, 'admin.go imports standard-library errors')
+req(admin.count('errors.New(') >= 2, 'admin.go errors.New validation paths retained')
+
+# Startup blocker: exception toggle must not overlap operation_id/mode.
+new_route = 'POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle'
+old_route = 'POST /api/security/schema/enforcement/exceptions/{exception_id}'
+req(new_route in admin, 'explicit positive-schema exception toggle route registered')
+req(old_route + '"' not in admin, 'ambiguous legacy exception toggle route absent')
+req('/api/security/schema/enforcement/exceptions/${encodeURIComponent(b.dataset.id)}/toggle' in ui,
+    'shipping Console uses explicit exception toggle route')
+
+# Startup regression: handler construction itself must be tested for panic.
+req('TestAdminHandlerRouteRegistrationNoPanic' in reg, 'admin route registration panic regression test exists')
+req('(&adminServer{}).handler()' in reg, 'route regression constructs full Admin handler')
+req('recover()' in reg, 'route regression fails on ServeMux registration panic')
+
+# Route inventory must be unique at the literal METHOD+path level.
+route_text = admin + "\n" + read("update.go")
+routes = re.findall(r'mux\.HandleFunc\("([A-Z]+) ([^"\\]+)"', route_text)
+req(len(routes) >= 120, 'Admin/update route inventory remains complete')
+req(len(routes) == len(set(routes)), 'Admin/update METHOD+path registrations remain unique')
+
+# Literal uniqueness is not enough for Go 1.22+ ServeMux: two distinct wildcard
+# patterns can still conflict. Register the exact source inventory with the
+# standard library in a standalone temporary program so module dependencies
+# and the repository go.mod cannot mask a startup-fatal ambiguity.
+patterns = [f"{method} {path}" for method, path in routes]
+with tempfile.TemporaryDirectory(prefix="waf-route-gate-") as td:
+    gofile = Path(td) / "main.go"
+    lines = [
+        'package main',
+        'import "net/http"',
+        'func main() {',
+        'm := http.NewServeMux()',
+    ]
+    for pattern in patterns:
+        lines.append(f'm.HandleFunc(`{pattern}`, func(http.ResponseWriter, *http.Request) {{}})')
+    lines.append('}')
+    gofile.write_text("\n".join(lines) + "\n", encoding="utf-8")
+    env = os.environ.copy()
+    env["GOTOOLCHAIN"] = "local"
+    proc = subprocess.run(["go", "run", str(gofile)], cwd=td, env=env,
+                          stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
+    req(proc.returncode == 0, 'Go stdlib ServeMux accepts all Admin/update route patterns without ambiguity')
+
+# Gate is part of both local and hosted build paths.
+req('test-build-startup-blocker-fix-source.py' in build, 'build.sh runs blocker-fix source gate')
+req('test-build-startup-blocker-fix-source.py' in ci, 'CI runs blocker-fix source gate')
+
+print(f"BUILD_STARTUP_BLOCKER_FIX_SOURCE_GATE_PASS checks={len(checks)} routes={len(routes)}")

<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->


# Root Test-Suite Correctness Fix Patch — 2026-10-15

Slice-local diff against `origin/main` (8fbe951) for the L7 fail-open, API-8 GraphQL clone/fingerprint, API-7.2 named path locator, and test-harness fixes. The removed accidental `return/` tree and this patch file itself are excluded from the diff.

diff --git a/api3_contract_test.go b/api3_contract_test.go
index 12d7156..2c0cb51 100644
--- a/api3_contract_test.go
+++ b/api3_contract_test.go
@@ -293,8 +293,11 @@ func TestDriftAgainstLiveLearnedSchemaTaxonomy(t *testing.T) {
 	op.VersionID = "v1"
 	bindings := []OperationBinding{{ID: "binding", ContractVersionID: "v1", ContractOperationID: "cop", APIOperationID: "op-payments", Status: "MATCHED", Confidence: 1}}
 	observed := []apiOperation{
-		{ID: "op-payments", Method: "POST", Path: "/api/payments/{id}", Samples: 100, ContentTypes: map[string]int64{"application/xml": 100}, AuthObserved: map[string]int64{"none": 100}},
-		{ID: "op-shadow", Method: "GET", Path: "/api/internal/debug", Samples: 5},
+		{ID: "op-payments", Site: "payments", Method: "POST", Path: "/api/payments/{id}", Samples: 100, ContentTypes: map[string]int64{"application/xml": 100}, AuthObserved: map[string]int64{"none": 100}},
+		// Shadow endpoints are only reported within the site/host scope of a bound
+		// operation (TestDriftUndeclaredEndpointIsScopedToMatchedApplicationTraffic),
+		// so the fixture places both operations on one site.
+		{ID: "op-shadow", Site: "payments", Method: "GET", Path: "/api/internal/debug", Samples: 5},
 	}
 	events := driftAgainstLearnedSchema("contract", version, []ContractOperation{op}, bindings, learned, observed)
 	seen := map[string]bool{}
diff --git a/graphql_api8.go b/graphql_api8.go
index 6645c24..95c8238 100644
--- a/graphql_api8.go
+++ b/graphql_api8.go
@@ -357,18 +357,20 @@ func cloneGraphQLPersisted(v GraphQLPersistedProfile) GraphQLPersistedProfile {
 	return v
 }
 func cloneGraphQLContract(v GraphQLSchemaContract) GraphQLSchemaContract {
-	v.Types = make(map[string][]string, len(v.Types))
+	types := make(map[string][]string, len(v.Types))
 	for name, fields := range v.Types {
-		v.Types[name] = append([]string(nil), fields...)
+		types[name] = append([]string(nil), fields...)
 	}
-	v.FieldTypes = make(map[string]map[string]string, len(v.FieldTypes))
+	v.Types = types
+	fieldTypes := make(map[string]map[string]string, len(v.FieldTypes))
 	for typ, fields := range v.FieldTypes {
 		cp := make(map[string]string, len(fields))
 		for name, target := range fields {
 			cp[name] = target
 		}
-		v.FieldTypes[typ] = cp
+		fieldTypes[typ] = cp
 	}
+	v.FieldTypes = fieldTypes
 	return v
 }
 
@@ -1723,17 +1725,14 @@ func analyzeGraphQLDocument(src, operationName string) (graphqlAnalysis, error)
 	for k, v := range op.variables {
 		vars[k] = v
 	}
-	// The fingerprint is derived from normalized structural metadata only; raw
-	// literals and the original query text are not durable evidence.
-	varNames := make([]string, 0, len(vars))
-	for k := range vars {
-		varNames = append(varNames, k)
-	}
-	sort.Strings(varNames)
+	// The fingerprint is derived from the normalized selection structure only;
+	// raw literals and the original query text are not durable evidence.
+	// Variable declarations are excluded so an inlined literal and its
+	// parameterized form share one operation identity, and so callers cannot
+	// mint unbounded operation IDs (escaping learned per-operation baselines)
+	// by adding unused variable definitions. Declared variable types remain
+	// available separately via VariableTypes.
 	parts := []string{op.typ, op.name, strconv.Itoa(maxDepth), strconv.Itoa(complexity), strings.Join(fields, "\x1f")}
-	for _, n := range varNames {
-		parts = append(parts, n+":"+vars[n])
-	}
 	h := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
 	return graphqlAnalysis{OperationType: op.typ, OperationName: op.name, DocumentFingerprint: hex.EncodeToString(h[:]), Depth: maxDepth, Complexity: complexity, Fields: fields, VariableTypes: vars, Introspection: introspection}, nil
 }
diff --git a/l7_abuse.go b/l7_abuse.go
index f5d80b2..c497a07 100644
--- a/l7_abuse.go
+++ b/l7_abuse.go
@@ -108,7 +108,7 @@ func (c *l7AbuseController) allowTLSHandshakeAt(scope, identity string, now time
 			c.pruneShardLocked(shard, now)
 		}
 		if len(shard.clients) >= l7AbuseMaxEntriesPerShard {
-			c.evictOldestIdleLocked(shard)
+			c.evictOldestIdleLocked(shard, now)
 		}
 		if len(shard.clients) >= l7AbuseMaxEntriesPerShard {
 			// Saturation must not turn the limiter off. Unknown identities share a
@@ -184,7 +184,7 @@ func (c *l7AbuseController) allowAt(site, identity string, now time.Time) (*abus
 			c.pruneShardLocked(shard, now)
 		}
 		if len(shard.clients) >= l7AbuseMaxEntriesPerShard {
-			c.evictOldestIdleLocked(shard)
+			c.evictOldestIdleLocked(shard, now)
 		}
 		if len(shard.clients) >= l7AbuseMaxEntriesPerShard {
 			// Bound memory without creating an attacker-controlled fail-open bypass.
@@ -257,7 +257,14 @@ func (c *l7AbuseController) pruneShardLocked(shard *l7AbuseShard, now time.Time)
 	}
 }
 
-func (c *l7AbuseController) evictOldestIdleLocked(shard *l7AbuseShard) {
+// evictOldestIdleLocked reclaims at most one entry to make room, but only a
+// genuinely stale one: idle (no in-flight request) AND not seen within the idle
+// TTL. Evicting a recently-seen idle entry would let an attacker cycling many
+// identities reclaim a fresh full-capacity bucket per identity, turning a
+// saturated shard into a fail-open bypass. When every entry is recent the shard
+// stays full and new identities fall to the bounded shared overflow bucket.
+func (c *l7AbuseController) evictOldestIdleLocked(shard *l7AbuseShard, now time.Time) {
+	cutoff := now.Add(-c.idleTTL())
 	var oldestKey abuseClientKey
 	var oldest time.Time
 	found := false
@@ -266,7 +273,7 @@ func (c *l7AbuseController) evictOldestIdleLocked(shard *l7AbuseShard) {
 		active := st.active
 		lastSeen := st.lastSeen
 		st.mu.Unlock()
-		if active != 0 {
+		if active != 0 || !lastSeen.Before(cutoff) {
 			continue
 		}
 		if !found || lastSeen.Before(oldest) {
diff --git a/l7_abuse_test.go b/l7_abuse_test.go
index 156fae2..1925ae5 100644
--- a/l7_abuse_test.go
+++ b/l7_abuse_test.go
@@ -91,19 +91,32 @@ func TestL7AbuseStateIsBoundedPerShard(t *testing.T) {
 	c := newL7AbuseController(L7AbuseConfig{Enabled: true, RequestsPerWindow: 1, WindowSeconds: 60})
 	now := time.Unix(1_700_000_000, 0)
 	const targetShard = uint64(0)
-	inserted := 0
-	for n := 0; inserted < l7AbuseMaxEntriesPerShard+64; n++ {
+	// Drive more distinct identities at a single instant than one shard can
+	// hold. The invariant under test is that per-shard state stays bounded;
+	// admissions past saturation are expected to fall to the shared overflow
+	// bucket and may be refused (fail-closed), which is the hardened contract
+	// exercised by TestProductionL7SaturationUsesBoundedOverflowBucket. The
+	// shard must never evict a recently-seen idle entry to mint a fresh
+	// full-capacity bucket for each new identity, as that is the fail-open
+	// bypass the overflow bucket exists to prevent.
+	attempted := 0
+	admitted := 0
+	for n := 0; attempted < l7AbuseMaxEntriesPerShard+64; n++ {
 		identity := fmt.Sprintf("2001:db8::%x", n)
 		key := abuseClientKey{site: "site-a", identity: identity}
 		if l7AbuseShardIndex(key) != targetShard {
 			continue
 		}
+		attempted++
 		st, ok := c.allowAt(key.site, key.identity, now)
-		if !ok {
-			t.Fatalf("identity %q unexpectedly rejected", identity)
+		if ok {
+			admitted++
+			c.releaseState(st)
 		}
-		c.releaseState(st)
-		inserted++
+	}
+
+	if admitted < l7AbuseMaxEntriesPerShard {
+		t.Fatalf("shard admitted only %d distinct identities before saturation; want >= %d", admitted, l7AbuseMaxEntriesPerShard)
 	}
 
 	shard := &c.shards[targetShard]
diff --git a/object_locator_api71.go b/object_locator_api71.go
index b6fdb03..782d213 100644
--- a/object_locator_api71.go
+++ b/object_locator_api71.go
@@ -633,6 +633,45 @@ func (s *objectLocatorStore) activeLocator(operationID, location, field string)
 	return cp, true
 }
 
+// activePathLocator resolves the locator for one dynamic segment of an
+// observed request path. Raw-path samples are positional (param_N), while
+// contract- and operator-declared path locators carry the declared parameter
+// name (orderId, id). A named path locator is bound to the segment only when
+// the operation has exactly one dynamic segment; with several segments the
+// name-to-position mapping is ambiguous and only the positional locator is
+// trusted.
+func (s *objectLocatorStore) activePathLocator(operationID, positional string, dynamicSegments int) (ObjectLocator, bool) {
+	if loc, ok := s.activeLocator(operationID, "path", positional); ok {
+		return loc, true
+	}
+	if s == nil || dynamicSegments != 1 {
+		return ObjectLocator{}, false
+	}
+	now := s.now()
+	s.mu.Lock()
+	defer s.mu.Unlock()
+	s.pruneLocked(now)
+	var found *ObjectLocator
+	for _, v := range s.locators {
+		if v == nil || v.OperationID != operationID || v.Location != "path" || v.Field == positional {
+			continue
+		}
+		if ov, ok := s.overrides[objectLocatorOverrideID(operationID, "path", v.Field)]; ok && ov.Action == objectLocatorOverrideSuppress {
+			continue
+		}
+		if found != nil {
+			return ObjectLocator{}, false
+		}
+		found = v
+	}
+	if found == nil {
+		return ObjectLocator{}, false
+	}
+	cp := cloneObjectLocator(*found)
+	cp.Status = objectLocatorStatusActive
+	return cp, true
+}
+
 func (s *objectLocatorStore) ensureFingerprintKey(configPath string) error {
 	path := apiSecurityStatePath(configPath, "api-object-locator.key")
 	info, err := os.Lstat(path)
diff --git a/object_relationship_api72.go b/object_relationship_api72.go
index eba9a88..841cd0c 100644
--- a/object_relationship_api72.go
+++ b/object_relationship_api72.go
@@ -303,6 +303,12 @@ func (s *objectRelationshipStore) process(ev objectRelationshipObservation) {
 	}
 	meta := apiObservationMeta{BodyPrefix: ev.BodyPrefix, BodyTruncated: ev.BodyTruncated}
 	samples := collectSchemaSamplesFromObservation(ev.Path, ev.RawQuery, ev.ContentType, meta)
+	pathSegments := 0
+	for _, sample := range samples {
+		if sample.Location == "path" {
+			pathSegments++
+		}
+	}
 	for _, sample := range samples {
 		if sample.Location != "path" && sample.Location != "query" && sample.Location != "body" {
 			continue
@@ -313,11 +319,17 @@ func (s *objectRelationshipStore) process(ev objectRelationshipObservation) {
 		if sample.objectValue == "" {
 			continue
 		}
-		locator, ok := s.locators.activeLocator(ev.OperationID, sample.Location, sample.Path)
+		var locator ObjectLocator
+		var ok bool
+		if sample.Location == "path" {
+			locator, ok = s.locators.activePathLocator(ev.OperationID, sample.Path, pathSegments)
+		} else {
+			locator, ok = s.locators.activeLocator(ev.OperationID, sample.Location, sample.Path)
+		}
 		if !ok {
 			continue
 		}
-		objectFingerprint := s.locators.digestValue(ev.OperationID, sample.Location, sample.Path, sample.objectValue)
+		objectFingerprint := s.locators.digestValue(ev.OperationID, locator.Location, locator.Field, sample.objectValue)
 		if objectFingerprint == "" {
 			continue
 		}
diff --git a/sequence_api62_test.go b/sequence_api62_test.go
index 8c2f068..5d6555d 100644
--- a/sequence_api62_test.go
+++ b/sequence_api62_test.go
@@ -277,6 +277,11 @@ func TestAPI62ConcurrentWorkflowLearningAndSnapshotsRemainBounded(t *testing.T)
 	start := time.Date(2026, 9, 24, 7, 0, 0, 0, time.UTC)
 	a := apiOperationID("race62", http.MethodGet, "/items/{id}")
 	b := apiOperationID("race62", http.MethodPost, "/checkout")
+	// Pin the store clock to simulated time before launching workers: the
+	// concurrent snapshot() calls below each run a prune pass, and against
+	// wall-clock they would evict the historical simulated learning mid-flight.
+	// A single write before goroutine creation is safe for the concurrent reads.
+	s.now = func() time.Time { return start.Add(time.Second) }
 	const workers = 32
 	var wg sync.WaitGroup
 	for i := 0; i < workers; i++ {
diff --git a/sequence_api63_test.go b/sequence_api63_test.go
index 4c65f4c..fedd159 100644
--- a/sequence_api63_test.go
+++ b/sequence_api63_test.go
@@ -30,6 +30,13 @@ func api63TestSettings() sequenceSettings {
 
 func api63TrainPath(s *sequenceStore, workflow, site string, ops []string, sessions int, start time.Time) time.Time {
 	at := start
+	// Couple the store clock to the simulated training clock BEFORE the loop:
+	// process() triggers periodic publishSnapshot()/prune passes mid-training,
+	// and against wall-clock those evict the historical simulated edges while
+	// they are still being learned. The closure tracks the advancing `at` so
+	// pruning uses simulated time throughout; it stays frozen at the final
+	// training time once the loop returns.
+	s.now = func() time.Time { return at }
 	for i := 0; i < sessions; i++ {
 		session := fmt.Sprintf("train-%s-%d", workflow, i)
 		for j, op := range ops {
diff --git a/sequence_api64_test.go b/sequence_api64_test.go
index ed3ba9f..358c121 100644
--- a/sequence_api64_test.go
+++ b/sequence_api64_test.go
@@ -15,6 +15,9 @@ import (
 
 func api64TrainWithoutMode(s *sequenceStore, workflow, site string, ops []string, sessions int, start time.Time) time.Time {
 	at := start
+	// Track the simulated training clock so mid-training prune passes do not
+	// evict historical simulated edges against wall-clock (see api63TrainPath).
+	s.now = func() time.Time { return at }
 	for i := 0; i < sessions; i++ {
 		session := site + "-learn-" + itoa(i+1)
 		for j, op := range ops {
@@ -152,6 +155,10 @@ func TestAPI64RecentSessionsAreBoundedTTLAndPrivacyPreserving(t *testing.T) {
 		s.pruneLocked(at.Add(1500 * time.Millisecond))
 		s.stateMu.Unlock()
 	}
+	// Pin the store clock just past the last simulated observation (well within
+	// RecentSessionTTL) so snapshot() keeps the bounded recent-session evidence
+	// instead of pruning it against wall-clock.
+	s.now = func() time.Time { return start.Add(7 * time.Second) }
 	model := s.snapshot()
 	if len(model.RecentSessions) != 2 || model.EvictedRecentSessions == 0 {
 		t.Fatalf("recent sessions not bounded: len=%d evicted=%d", len(model.RecentSessions), model.EvictedRecentSessions)


## CI source-gate restoration — 2026-10-15

Removed re-introduced dead files (listed in the ledgers), restored `debug_sanitize.go`, and corrected two source gates. Deleted file bodies are omitted; only additions/modifications are shown.

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
diff --git a/tools/tests/test-api12-source.py b/tools/tests/test-api12-source.py
index 0a0e0f3..fc2f040 100644
--- a/tools/tests/test-api12-source.py
+++ b/tools/tests/test-api12-source.py
@@ -98,7 +98,10 @@ require('data-view="api"' in ui, 'API Security console tab missing')
 for dom_id in ['api_ops_body', 'api_schema_body', 'api_contract_body', 'api_contract_import']:
     require(dom_id in ui, f'API Security UI control missing: {dom_id}')
 
-require('writeJSON(w, http.' not in all_go, 'known writeJSON arity compile defect is still present')
+# The package-level writeJSON(w, v) takes no status code; a status argument
+# is the historical arity defect. Methods such as dashboardReader.writeJSON
+# (w, status, v) legitimately take one, so only bare calls are rejected.
+require(re.search(r'(?<![.\w])writeJSON\(w, http\.', all_go) is None, 'known writeJSON arity compile defect is still present')
 
 require('os.CreateTemp' in persist and '.tmp-*' in persist, 'API security persistence still uses a fixed temporary filename')
 require('len(agg.Fields) >= schemaMaxFields' in schema, 'API-2 aggregate field growth is not bounded across requests')
diff --git a/tools/tests/test-markdown-review-source.py b/tools/tests/test-markdown-review-source.py
index e86789a..88ee6c4 100644
--- a/tools/tests/test-markdown-review-source.py
+++ b/tools/tests/test-markdown-review-source.py
@@ -1,19 +1,24 @@
 #!/usr/bin/env python3
 from pathlib import Path
+import re
 import sys
 
 root = Path(__file__).resolve().parents[2]
 mds = sorted(root.rglob('*.md'))
 fail=[]
-marker='<!-- documentation-review: 2026-09-28; classification:'
+# Every Markdown file carries a dated review marker with a classification.
+# Later review passes (e.g. 2026-09-29) stamp their own date, so any dated
+# marker is accepted rather than pinning the original 2026-09-28 pass.
+marker=re.compile(r'<!-- documentation-review: \d{4}-\d{2}-\d{2}; classification:')
 for p in mds:
     text=p.read_text(encoding='utf-8')
-    if marker not in text:
+    if not marker.search(text):
         fail.append(f'missing review marker: {p.relative_to(root)}')
 ledger=root/'MARKDOWN_REVIEW_2026-09-28.md'
 if not ledger.is_file(): fail.append('missing MARKDOWN_REVIEW_2026-09-28.md')
 else:
-    lt=ledger.read_text(encoding='utf-8')
+    # A file must be recorded in at least one dated review ledger.
+    lt='\n'.join(p.read_text(encoding='utf-8') for p in sorted(root.glob('MARKDOWN_REVIEW_*.md')))
     for p in mds:
         rel=str(p.relative_to(root))
         if f'`{rel}`' not in lt:
