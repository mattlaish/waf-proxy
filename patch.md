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
