# Build and Admin Startup Blocker Fix Patch

## 2026-11-25 — Complete operator-workflow pass

Approved implementation of the 2026-11-24 workflow backlog and all six
additional operator-experience proposals. Active source remains
`C:\Users\mattlai\Projects\waf-proxy`; no Git operations or relocation.
This entry supersedes the earlier NOT IMPLEMENTED UI backlog, not historical
security qualification or the still-missing SOC reader.

- Webhook clear is a cancellable/undoable draft intent
  `notify.clear_webhook`, saved with the draft and resolved on reviewed Apply.
  Blank URL without that intent still preserves the stored secret. Live config
  is not touched by the UI clear button; unrelated edits survive. The intent is
  consumed before live persistence and stripped/ignored across HA. The legacy
  DELETE endpoint remains immediate for compatibility; UI no longer calls it.
- Syslog Facility 0 is serialized as 0, not 16. Opening a group no longer
  materializes optional monitor defaults and invents unsaved edits.
- Delivery tests confirm the active destination, explicitly distinguish pending
  edits/drafts and send only a fixed synthetic message. Test results update
  transport observations, including failures/short writes. New operator-only
  `POST /api/notifications/test`; existing `POST /api/syslog/test` retained.
  Status exposes configuration generation plus an opaque per-process/config
  token; new UI sends `expected_generation` and `expected_token`, with 409 on
  mismatches. This prevents stale/different-node confirmations. Older API
  clients may omit these fields for compatibility. Tokens are not credentials
  or endpoint hashes. A successful send is not receipt/indexing/ticket proof.
- New Reviewer-gated/audited `POST /api/notifications/handle` separates handled
  from read. Opening the bell is read-only; explicit mark-read/all-read remains.
  Attack notifications carry structured site/kind/start/end context, with a
  read-only related-logs shortcut. Match/access JSON includes `occurred_at`.
  Logs apply site/period scope and label sampling/legacy timestamp limitations;
  access status alone does not prove a WAF block. Alerts never auto-edit policy.
- Dashboard Needs attention includes enabled webhook/SIEM delivery failure,
  drop and stale observations, linking to Setup. Intentional disable is not an
  outage. Counts remain process-lifetime observations.
- Site workspace links to existing site/group/policy/map/log editors; no second
  config authority or duplicate editors. Site context and shared-resource impact
  are shown. Unsaved/page edits guard context changes. Bounded memory-only task
  history (20 entries) retains selections, log mode/filter/severity and scroll;
  page close also restores the originating log position. No persisted secrets.
- Field workflow identifies site/submission URL/configured fields and supports
  observed/manual paths. Discovery remains opt-in: searchable multi-selection,
  explicit selected-rule preview, then existing page review/Apply; no default
  selection, no sibling-field activation or enforcement/method semantics change.
- Guided setup leads to draft review, Apply, readiness and actual-request
  verification. Readiness re-reads active config, health and recent access data
  without sending an application request. Operator manual verification is a
  session-local note bound to the active-config signature, invalidated by
  changes; it is never presented as an automated test or full application proof.
- Setup task navigation opens existing TLS/AI/HA/update advanced groups without
  losing inputs. Notifications/SIEM stay directly available. On phones the site
  action panel defaults collapsed and honors explicit user expansion.

Source scope: `static/admin.html`, `notify.go`, `attack_alerts.go`,
`admin.go`, `syslog.go`, `delivery_health.go`, `ha.go`, `main.go`;
new `console_workflow.go` and `console_workflow_test.go`; both Console suites
expanded. No dependency, package, deployment or WAF enforcement-engine change.

Verification: pinned portable Go 1.26.9, real dependencies. **32 targeted Go
workflow/security/L7/HA/notification tests PASS** (7 new workflow tests);
**26/26 Node Console**, **40/40 intercepted Edge/Playwright workflows** PASS
(3 new Node, 8 new browser tests). Desktop/390px rendering inspected with
synthetic fixtures. Tidy-diff clean; native Windows build/vet and Linux
cross-build PASS. **19/19 Python source/document gates PASS**, including the
Go-backed route gate; Admin/update inventory is **142** patterns.

The complete Windows `go test ./...` was attempted and **FAILED**, including
Unix private modes, directory fsync, Unix paths/shell/executable bits and
concurrent-rename checks; no new workflow/security test failure was observed.
Race attempt is BLOCKED because gcc is absent; WSL is not installed. Linux
full test/race, VM/HA, real SIEM/webhook/provider/CRS and hardware acceptance
remain NOT_RUN. State remains **IMPLEMENTED_TESTING_DEFERRED**; no release
promotion, generated artifact, deployment or Git action.

Initial browser expansion found a real selector/input redraw race and phantom
monitor-default edits; both were fixed. A test also read the confirmation before
its async status fetch completed; it now waits for the visible dialog without
weakening destination assertions. No tests were skipped to claim success.

## 2026-11-24 — UI workflow audit backlog (recorded, NOT implemented)

User approved recording these findings only. No source/API/UI implementation or
Git operation is authorized by this entry; obtain explicit approval before fixes.
The 2026-11-19 implementation/qualification status below is unchanged.

1. High: Clear stored webhook immediately mutates live configuration, clears a
   newly entered draft URL, and does not refresh the editor baselines. Prefer a
   staged clear through review/Apply; if retaining immediate deletion, guard
   unsaved work and reconcile active/draft state without losing unrelated edits.
2. High: collectSyslog serializes legal Facility 0 as 16 via a truthy fallback.
   Preserve numeric zero and add request-payload round-trip regression coverage.
3. Medium: Send test uses active settings, not the current form; its result does
   not update delivery observations. Make test target/config source explicit,
   handle pending drafts, and join test results to transport health without
   claiming SIEM receipt/indexing. Setup's existing active-settings hint is not
   a replacement for an explicit test workflow.
4. Medium: Attack notifications expose only Dismiss, with no investigation
   shortcut; opening the bell marks all notifications read. Add structured
   site/time/event context for safe log navigation and distinguish read from
   handled; never auto-create exceptions or change enforcement from an alert.
5. Medium: Dashboard Needs attention ignores failed webhook/SIEM delivery.
   Include failed/dropped/stale observations with links to the relevant settings;
   intentionally disabled integrations must not be labelled broken.
6. Navigation proposal: group the long Setup / AI page by task (TLS, AI, HA,
   notifications, SIEM, updates) with section navigation. Proposal, not a verified
   functional bug or approved implementation.

Evidence from the read-only audit: two isolated in-memory browser cases confirmed
draft URL loss/direct DELETE and Dismiss-only attack actions; two executions of
the actual console JavaScript reproduced Facility 0 -> 16 and an empty attention
list when delivery alone is degraded. Other findings were confirmed by source
inspection. No full suite or live WAF/VM acceptance was performed for this audit.
This documentation update does not fix or qualify these pending findings.


## 2026-11-19 — Rate limiter, alert delivery and webhook security

Approved report-driven security changes in the fixed source root
`C:\Users\mattlai\Projects\waf-proxy`; no Git operations, relocation,
production changes, dependency edits, or archive regeneration.

- L7 HTTP/TLS identities default to IPv6 /64 aggregation, canonicalize mapped
  IPv4, and ignore host ports/zones for limiter keys only. Config
  `l7_abuse.ipv6_prefix_length` accepts 0 (default /64) or 64..128. /128 is an
  explicit rotation-vs-shared-network trade-off, not a recommended bypass fix.
  Overflow is now one HTTP bucket per configured site / TLS bucket per listener
  across all shards. Retained overflow states are capped at 8192; never evicted
  to mint fresh tokens. Capacity exhaustion refuses new scopes. Known client
  entries remain bounded and cannot be evicted while recently active.
  Normal table capacity remains globally bounded: one site's saturation may
  force another's new visitors into that site's aggregate overflow bucket.
  Overflow allowance/concurrency isolation does not promise fair per-client
  admission or eliminate this indirect shared-capacity pressure.
- Notifications add `on_attack`, `attack_threshold` (0 defaults to 10),
  `attack_window_seconds` (0 defaults to 60), and `attack_min_severity`
  (default CRITICAL). Matching severity or higher is counted; WAF block events
  have their own counter. Either threshold triggers one notification per site/period,
  bounded 4096 aggregation states. Rules are event counts, not unique requests.
  Coraza blocks use authoritative final transaction interruptions (including
  response phases), not HTTP status; explicit WAF rejection helpers also mark
  blocks. Application 403s are not blocks. TLS-handshake rejections have no
  HTTP transaction and are not counted here. No request bodies, query strings,
  client addresses or credentials enter the aggregate alert payload.
- Webhooks require HTTPS and public destinations by default.
  `notify.webhook_allow_http` and `notify.webhook_allow_private` are explicit,
  separate insecure/internal exceptions. URL syntax, credentials/fragments and
  literal addresses are checked at configuration validation and send time.
  Every resolved DNS address is checked inside DialContext, then a validated
  numeric address is dialled directly; environment proxies are disabled and
  redirects are refused. Link-local/metadata, multicast, unspecified and
  reserved/transition destinations remain forbidden even with private opt-in.
  TLS certificate verification is unchanged. Invalid legacy HTTP/internal
  webhook configs must be adjusted before startup/Apply; do not silently grant
  exceptions during migration.
- `GET /api/status` now exposes `syslog_delivery` and `webhook_delivery`;
  notification API also retains webhook telemetry. Console Setup displays
  last send/failure, failure count, queue depth/capacity and drops, with
  disabled/unverified/degraded/stale/last-send-success states, never an
  unqualified green "working" claim. Syslog drops include queue-full,
  disabled queued entries and failed/short writes. Webhook failure counts are
  per attempt; webhook dropped counts queue-full and obsolete queued jobs.
  Endpoint/config changes invalidate observations and old in-flight completion
  cannot verify a new endpoint. Counters are process-lifetime, not durable.
  UDP/TCP send success does not prove SIEM receipt/indexing.
- Current Notify/Syslog settings are HA-shared; webhook URL remains node-local.
  Console explains this. No SOC/ticketing reader implementation was present in
  this workspace: report-specific off-switch/auth ordering and SOC switch
  isolation cannot be changed or claimed fixed. No replacement reader was
  invented. Supply that work before closing report items 6–8.

### Required residual-risk statements

- **`/64`＋每站 overflow 不能完全消除 lockout**：能限制同 `/64` 輪換及跨站誤傷，但同站大量來源仍可能耗盡配額；也需處理同 `/64` 合法用戶共用額度的取捨。
- **HTTPS 不等於防 SSRF**：目的地址、重新導向與 DNS 解析後的地址也要檢查；允許內部 webhook 應採明確例外。

The underlying controls above implement the second statement; neither statement
is a claim of zero exposure or complete distributed-DoS protection.

Validation on this host using verified portable Go 1.26.9 (windows/amd64):
targeted Go security/L7/notification tests **23 PASS** (13 new security tests,
including actual Coraza inline-rule request/response tests); Node console
**23/23 PASS**, intercepted Edge/Playwright workflows **32/32 PASS**; Python
source gates **18/18 PASS** plus Go-backed route gate **13 checks / 140 routes
PASS**. Root tidy-diff clean, native Windows build/vet and Linux cross-build PASS.
These are not Linux runtime/CRS/hardware qualification.

Full `go test ./...` was actually attempted on Windows and **FAILED**, including
Unix-only permissions/path/fsync, concurrent rename, release executable bits
and unavailable Linux command checks; do not relabel that run PASS or skip
those assertions. Race was attempted but **BLOCKED**: CGO=0 is unsupported;
with CGO=1 gcc is absent. WSL is not installed, so Linux full tests/race,
live VM/HA/webhook/SIEM/CRS/VectorScan/HSM acceptance remain pending.
State: **IMPLEMENTED_TESTING_DEFERRED**. No deployment or new release ZIP.

Next: run exact-source Go 1.26.9 tidy/build/vet/full tests/race on Linux,
real CRS and VM saturation/HA/delivery acceptance; bring the missing SOC
work into this same root before fixing its auth/off-switch semantics.

## 2026-11-11 — Keep underlying logic aligned with changes

Added the user's implementation rule to `AGENTS.md`:

> 修改不能只修表面, 底層邏輯要跟著修改一併同步改

Documentation only; the existing approval requirement remains unchanged.

## 2026-11-09 — Explicit approval before AI file changes

The user requested an approval gate in `AGENTS.md`: project files are read-only
unless the user explicitly authorizes the specific changes. This applies to
source, tests, settings, Markdown, and generated artifacts; additional work
outside the approved scope requires new approval. Git operations still require
separate, task-specific authorization. This is a documentation-only change;
no source, build, deployment, or Git operation is part of this update.

## 2026-10-24 — Five operation-workflow fixes

Base: the existing local tree with the 2026-10-18 qualification/review notes;
no Git identity was newly inspected or changed. This dated record supersedes
older workflow descriptions, without rewriting historical evidence.

The five issues reproduced in the 2026-10-23 user-operation review are fixed in
the shipping source at `C:\Users\mattlai\Projects\waf-proxy`:

1. `Configure this form` explicitly opens a submission path even when the
   only option was already selected. Candidates observed on the source page are
   carried to the target editor, merged with target observations, and deduplicated.
2. Discovered fields are read-only candidates until `Add suggested rule` is
   pressed. Preparing a password exception changes only the selected target;
   existing field rules are retained and discovered siblings are not activated.
   Candidate buttons preserve permission/already-configured disabled state.
3. Wizard Back captures the current step's raw inputs without validating them,
   then restores them on return. Forward still validates before advancing.
4. Page undo is found by recorded site/path/scope, including deleted prefix rules,
   rather than the new editor's default exact scope. With two histories, the
   operator chooses which scope to undo. Other policies and concurrency checks
   are retained.
5. Full-config actions distinguish `Load active settings` (read-only),
   `Continue saved draft` (read-only), and `Discard saved draft` (confirmed
   shared-draft deletion). Cancel does not write; failure does not claim success.
   Reload after a successful discard cannot resume that old draft.

`DELETE /api/config?draft=1` is the only new Admin operation. It requires
Operator/Admin, rejects other/missing draft selectors, rejects unavailable storage,
serializes with config writes via `applyMu`, and durably removes only
`configPath + ".draft"` using the existing remove/fsync helper. Missing drafts
are idempotently converged. Live config bytes/runtime, listeners, managed IPs,
page policies and credentials are not rewritten. Success is audited. The browser
warns that this removes the one shared server draft, including others' saved work,
and local unsaved config edits; this is not a per-user draft or draft undo feature.
There is no atomic draft revision/CAS token; the operation intentionally discards
the shared draft present when the serialized request runs.


Executed on this host: **21/21** dependency-free Node checks and **27/27** isolated
headless Edge/Playwright browser tests PASS; **18/18** Python source/document
gates PASS. Source route inventory is now **140** Admin/update patterns.
Browser tests use intercepted fixtures, never a live WAF. The 12 new browser
cases cover all five fixes plus candidate approval/permissions, existing-field
preservation, dual-scope undo, concurrent edits, cancellation, reload and failure.

Added four Go tests in `console_draft_test.go` for durable/idempotent deletion,
active-file/runtime preservation, selector/role checks, failure and missing storage.
They and root Go tidy/build/vet/full tests/race/gofmt and the Go-executing route
gate are **NOT_RUN/BLOCKED on these changed bytes**: Go/gofmt are not installed on
this Windows host. Earlier 2026-10-18 Go PASS is historical, not proof of this slice.
Run pinned Go 1.26.9+ qualification and real VM acceptance before release.
State remains `IMPLEMENTED_TESTING_DEFERRED`. No Git operation, project relocation,
live mutation, deployment, binary build or archive regeneration was performed.

Changed implementation/tests: `static/admin.html`, `admin.go`,
`console_draft_test.go`, `tools/tests/test-console-copy.mjs`,
`tools/tests/test-console-workflows.mjs`. Operating/handover/testing documentation
is synchronized. No Go dependency or protection-engine setting was changed.


## 2026-10-18 — Console workflow UX maintenance

Base: the existing local tree containing the 2026-10-17 wording and 2026-10-15
root fixes; no commit hash was newly inspected because no Git operation was used.

Active root: `C:\Users\mattlai\Projects\waf-proxy`. No Git operations,
source relocation, deployment, binary rebuild or ZIP regeneration. This is a
console workflow slice, not a new protection engine/API-security capability.
Verified locally: **19/19** dependency-free Node tests, **15/15** isolated
headless Edge/Playwright workflow tests, and **18/18** Python source/document
gates PASS. Browser requests are intercepted fixtures, not a live WAF/VM.

Go/gofmt and WSL remain unavailable. The added active-certificate status helper
and its Go tests, root tidy/build/vet/full tests/race, and the Go-executing route
gate are **NOT_RUN/BLOCKED** on these bytes. Historical Go PASS does not qualify
this slice. State remains **IMPLEMENTED_TESTING_DEFERRED**.

Implemented the approved UX improvements in the shipping `static/admin.html`:

- Unsaved full-config and page-edit guards; save/discard/cancel navigation,
  before-unload warnings, draft/active state, and explicit sign-out confirmation.
- All full-config Apply controls now share an escaped, nested-secret-redacted
  change preview, affected-site list, and active-config preflight. Cancellation
  sends no write; named-list order changes are shown; previews cap at 200
  differences and conservatively list all sites when capped. No browser draft
  or entered secret is stored in local/session storage (only the existing login
  session token). Repeated mutations are guarded and never blindly auto-retried.
- API Security loads all 31 reads independently/progressively, including when one
  stalls. Successful results survive other failures; stale data, last success,
  permission denial, persistent errors and read Retry are visible. Searchable
  endpoint selectors replace seven raw-ID inputs; existing workflow, field and
  schema choices remain. Native prompt chains/password prompts become validated
  dialogs with reason/field/expiry controls.
- Four-step site wizard: address, application servers, HTTPS/network, protection,
  then explicit review/save as draft. Management NICs are excluded. Existing
  groups and a new single-server group are supported without disabling upstream
  TLS verification. Readiness validation is not a backend connectivity test.
- Event detail opens the page or prepares a named field-only rule exception.
  Old events lack authoritative method/field metadata, so both are chosen by the
  operator, never inferred from match text or data. Page editor reloads current
  rules, displays inheritance and overlapping rules in execution order, preserves
  hidden target exceptions, validates inputs, previews changes, and supports
  session-local undo with preflight. Exact/prefix policies sharing one path are
  selected/deleted separately. Existing policy scope/path is not silently renamed.
- Direct page actions/presets/learner shortcuts require a clean full-config draft
  and explicit review, refresh the active baseline and record session undo.
  This prevents a known stale full draft from later overwriting a page change.
- Scan progress is polled per site without rebuilding open page editors.
  Completion shows pages/forms/field observations and reviewable new field names.
  Field listing uses at most four parallel requests over 200 known page paths;
  partial results/baselines are labelled. Resumed scans with no baseline do not
  claim that every discovered field is new. Scan counts are observations, not
  guaranteed distinct fields. Clear discovery now confirms and uses the existing
  registered POST method (the old console incorrectly sent GET).
- Dashboard attention list: unavailable servers, recent response-error samples,
  recent matches, pending field suggestions, restart/config activation needs,
  and certificate expiry/unknown status. 403/429 samples are explicitly not
  labelled WAF-block rates, nor are match counts labelled blocked attacks.
- Inline numeric/form/page errors, linked error summaries, modal Escape/Tab/focus
  return, labelled dialogs and responsive narrow-screen layout.

Backend scope: `admin.go`, `console_status.go`, `console_status_test.go` add
only read-only `tls_certificate_expires_at` per site to GET /api/status, using
the earliest active Go-listener certificate for that site. No private key/file
reads; unavailable/external TLS expiry remains unknown. Route count remains 139.

Important boundaries: preflight detects already-visible concurrent edits but is
not an atomic revision/CAS API, and redacted secrets cannot be compared for
revision identity. Undo exists only in the current browser session and refuses
known intervening page changes. Page rule exceptions apply to all HTTP methods
matching the URL; the method selector gates field validation, not these existing
phase-1 exceptions. The UI states this explicitly; enforcement semantics were not
changed. The effective-policy display is explanatory, not a full CRS simulation.

Next: on a Go 1.26.9+ host, format/review the three touched Go files and run
`GOTOOLCHAIN=local go mod tidy -diff`, `CGO_ENABLED=0 go build ./...`,
`go vet ./...`, `go test ./...`, `go test -race ./...`, and the existing
Go-backed route gate. Include `TestConsoleRuntimeCertificateExpiry` and the
previous slice's `TestBlockPageProductLanguagePreservesAPIReason`.
Then qualify real VM save/apply/restart, managed-IP, certificate modes, crawler
and page-rule behavior before rebuilding/reissuing a package. Do not use
unchanged binaries/archives as evidence for these source changes. Continue in
the same source root; no Git operations unless the user explicitly authorizes them.


## 2026-10-17 — Product language maintenance record

Working root: `C:\Users\mattlai\Projects\waf-proxy`. Base is the existing local
source with the 2026-10-15 root/toolchain fixes documented in AI_HANDOFF.md;
no Git command was used to resolve or alter a commit.

- `static/admin.html`: replace exposed engineering stage names and terms with
  operator wording; add display-only mode/finding labels; preserve all 132 option
  values, API route literals, and element IDs; allow navigation labels to wrap.
- Correct draft-only save labels and explain activation. Hoist `liveApply` to
  repair the existing console startup initialization failure.
- `profiles.go`, `learn.go`: update displayed descriptions/suggestions only;
  `observations_test.go` expects the new wording for the same observed fields.
- `block_response.go`, `block_response_copy_test.go`: human-readable known HTML
  rejection reasons, unchanged JSON reason codes, and a focused regression test.
- `tools/tests/test-console-copy.mjs` and `.github/workflows/ci.yml`: execute the
  actual console script and verify display/request compatibility under Node.
- Update wording assertions in the API/source gates without removing authority
  checks; normalize Markdown ledger paths for Windows/Linux parity.
- Add OPERATOR_GUIDE.md and synchronize README/INSTALL, current handover,
  documentation index, verification policy, manifest, and ledgers.

Executed: console tests 9/9 PASS; Python source/document gates 18/18 PASS;
embedded JavaScript syntax PASS; before/after route/ID/option parity PASS.
Go/gofmt and WSL are unavailable here. Root Go build/vet/test/race, block-page
test, and the Go-executing route gate are NOT_RUN/BLOCKED on these bytes.
No package ZIP, binary, checksums, Git state, or live deployment was changed.
Historical patch content below describes its original dated source, not a complete
diff for this maintenance pass. Current next step is exact-source Go qualification.

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


## Go 1.26 toolchain migration — 2026-10-15

Moves the project from end-of-life Go 1.25 to Go 1.26 (go1.26.9) to remediate 11 reachable standard-library vulnerabilities reported by govulncheck; bumps golang.org/x/net to v0.60.0 and updates CI, release scripts and the package builder.

diff --git a/.github/workflows/ci.yml b/.github/workflows/ci.yml
index f2600a1..2d45d79 100644
--- a/.github/workflows/ci.yml
+++ b/.github/workflows/ci.yml
@@ -18,7 +18,7 @@ jobs:
 
       - uses: actions/setup-go@v5
         with:
-          go-version: '1.25'
+          go-version: '1.26'
           check-latest: true
 
       - name: dependency-free root source-shape gate
@@ -95,9 +95,9 @@ jobs:
       - uses: actions/checkout@v4
       - uses: actions/setup-go@v5
         with:
-          go-version: '1.25'
+          go-version: '1.26'
           check-latest: true
       - name: Install govulncheck
-        run: GOTOOLCHAIN=local go install golang.org/x/vuln/cmd/govulncheck@v1.7.0
+        run: GOTOOLCHAIN=local go install golang.org/x/vuln/cmd/govulncheck@v1.9.0
       - name: Run govulncheck
         run: GOTOOLCHAIN=local govulncheck ./...
diff --git a/build.sh b/build.sh
index 8884130..8ac5307 100755
--- a/build.sh
+++ b/build.sh
@@ -1,7 +1,8 @@
 #!/usr/bin/env bash
 # Build waf-proxy and companion binaries.
 #
-# Requires Go >= 1.25 because Coraza v3.7.0 declares go 1.25.0.
+# Requires Go >= 1.26: go.mod declares go 1.26.0 (Go 1.25 is end-of-life and
+# no longer receives standard-library security fixes).
 # VectorScan build policy:
 #   WAF_VECTORSCAN=auto      build native acceleration when libhs is available (default)
 #   WAF_VECTORSCAN=required  require libhs/pkg-config and fail otherwise
@@ -45,8 +46,8 @@ version_ge() {
   [ "$(printf '%s\n%s\n' "$2" "$1" | sort -V | head -n1)" = "$2" ]
 }
 GO_VERSION="$(GOTOOLCHAIN=local go version 2>/dev/null | awk '{print $3}' | sed 's/^go//' || true)"
-if [ -z "$GO_VERSION" ] || ! version_ge "$GO_VERSION" "1.25.0"; then
-  echo "!! Go >= 1.25.0 is required for Coraza v3.7.0 (found ${GO_VERSION:-unavailable})" >&2
+if [ -z "$GO_VERSION" ] || ! version_ge "$GO_VERSION" "1.26.0"; then
+  echo "!! Go >= 1.26.0 is required by go.mod (found ${GO_VERSION:-unavailable})" >&2
   exit 1
 fi
 
diff --git a/go.mod b/go.mod
index 4d8f73b..b4c9ecb 100644
--- a/go.mod
+++ b/go.mod
@@ -1,6 +1,8 @@
 module waf-proxy
 
-go 1.25.0
+go 1.26.0
+
+toolchain go1.26.9
 
 require (
 	github.com/corazawaf/coraza/v3 v3.7.0
@@ -20,9 +22,9 @@ require (
 	github.com/tidwall/match v1.1.1 // indirect
 	github.com/tidwall/pretty v1.2.1 // indirect
 	github.com/valllabh/ocsf-schema-golang v1.0.3 // indirect
-	golang.org/x/net v0.52.0 // indirect
-	golang.org/x/sync v0.20.0 // indirect
-	golang.org/x/text v0.35.0 // indirect
+	golang.org/x/net v0.60.0 // indirect
+	golang.org/x/sync v0.23.0 // indirect
+	golang.org/x/text v0.42.0 // indirect
 	google.golang.org/protobuf v1.36.11 // indirect
 	rsc.io/binaryregexp v0.2.0 // indirect
 )
diff --git a/qualify-release-host.sh b/qualify-release-host.sh
index 4cec607..9d28e9e 100755
--- a/qualify-release-host.sh
+++ b/qualify-release-host.sh
@@ -2,7 +2,7 @@
 # Phase 0 release-host qualification runner.
 #
 # This script intentionally distinguishes BLOCKED from FAIL. A host that lacks
-# Go >=1.25 or verified real libvectorscan provenance has not failed WAF
+# Go >=1.26 or verified real libvectorscan provenance has not failed WAF
 # correctness; it is simply not a valid host on which to claim the real gates.
 #
 # Usage:
@@ -61,17 +61,17 @@ else
   gov="$(local_go_version || true)"
   if [ -z "$gov" ]; then
     block "could not determine local Go version without toolchain auto-download"
-  elif ! version_ge "$gov" "1.25.0"; then
-    block "Go >=1.25.0 required; local toolchain is go${gov}"
+  elif ! version_ge "$gov" "1.26.0"; then
+    block "Go >=1.26.0 required; local toolchain is go${gov}"
   else
     pass "local Go toolchain go${gov}"
   fi
 fi
 
-if grep -Eq '^go 1\.25\.0$' go.mod; then
-  pass "go.mod pins Go 1.25.0"
+if grep -Eq '^go 1\.26\.0$' go.mod; then
+  pass "go.mod pins Go 1.26.0"
 else
-  fail "go.mod no longer pins Go 1.25.0"
+  fail "go.mod no longer pins Go 1.26.0"
 fi
 if grep -Eq '^require[[:space:]]+github\.com/corazawaf/coraza/v3[[:space:]]+v3\.7\.0[[:space:]]*$' go.mod; then
   pass "go.mod pins Coraza v3.7.0"
diff --git a/release-security-scan.sh b/release-security-scan.sh
index 6a286ae..5a0f246 100755
--- a/release-security-scan.sh
+++ b/release-security-scan.sh
@@ -68,8 +68,8 @@ if ! command -v go >/dev/null 2>&1; then
   [[ "$mode" == "required" ]] && exit 1 || exit 3
 fi
 gov=$(GOTOOLCHAIN=local go version 2>/dev/null | awk '{print $3}' | sed 's/^go//' || true)
-if [[ -z "$gov" ]] || [[ "$(printf '%s\n%s\n' 1.25.0 "$gov" | sort -V | head -n1)" != "1.25.0" ]]; then
-  write_evidence BLOCKED "Go >=1.25.0 required by project; local toolchain is ${gov:-unknown}" "$govuln_version"
+if [[ -z "$gov" ]] || [[ "$(printf '%s\n%s\n' 1.26.0 "$gov" | sort -V | head -n1)" != "1.26.0" ]]; then
+  write_evidence BLOCKED "Go >=1.26.0 required by project; local toolchain is ${gov:-unknown}" "$govuln_version"
   [[ "$mode" == "required" ]] && exit 1 || exit 3
 fi
 
diff --git a/run-phase1-qualification.sh b/run-phase1-qualification.sh
index bb30228..30338cf 100755
--- a/run-phase1-qualification.sh
+++ b/run-phase1-qualification.sh
@@ -38,8 +38,8 @@ pkg-config --exists libhs || { echo "BLOCKED: real libhs/VectorScan is unavailab
 
 ver=$(GOTOOLCHAIN=local go version 2>/dev/null | awk '{print $3}' | sed 's/^go//' || true)
 version_ge() { [ "$(printf '%s\n%s\n' "$2" "$1" | sort -V | head -n1)" = "$2" ]; }
-if [[ -z "$ver" ]] || ! version_ge "$ver" 1.25.0; then
-  echo "BLOCKED: Go >=1.25.0 required (found ${ver:-unavailable})" >&2
+if [[ -z "$ver" ]] || ! version_ge "$ver" 1.26.0; then
+  echo "BLOCKED: Go >=1.26.0 required (found ${ver:-unavailable})" >&2
   exit 3
 fi
 
diff --git a/tools/tests/test_waf_package_builder.py b/tools/tests/test_waf_package_builder.py
index 989b963..ff64c45 100644
--- a/tools/tests/test_waf_package_builder.py
+++ b/tools/tests/test_waf_package_builder.py
@@ -26,7 +26,7 @@ class BuilderTests(unittest.TestCase):
 
     def test_project_pins(self):
         go, coraza = mod.read_project_requirements(ROOT)
-        self.assertEqual(go, "1.25.0")
+        self.assertEqual(go, "1.26.0")
         self.assertEqual(coraza, "v3.7.0")
 
     def test_fingerprint_is_stable_and_ignores_dist(self):
diff --git a/tools/waf_package_builder.py b/tools/waf_package_builder.py
index fa575dd..43287ba 100644
--- a/tools/waf_package_builder.py
+++ b/tools/waf_package_builder.py
@@ -381,7 +381,7 @@ def add_common(p: argparse.ArgumentParser) -> None:
     p.add_argument("--version", help="package version; default is YYYY.MM.DD.src<sourcehash>")
     p.add_argument("--commit", help="build source identity; default git SHA or source fingerprint")
     p.add_argument("--source-date-epoch", help="reproducible epoch; default env, git commit, then newest source mtime")
-    p.add_argument("--go-bin", help="Go binary to use (must satisfy go.mod, currently >=1.25.0)")
+    p.add_argument("--go-bin", help="Go binary to use (must satisfy go.mod, currently >=1.26.0)")
     p.add_argument("--vectorscan", choices=("off", "auto", "required"), default="off",
                    help="native VectorScan build policy (default: off for portable Coraza package)")
     p.add_argument("--hsm", choices=("off", "auto", "required"), default="off",


## Console UX qualification + review fixes — 2026-10-18

Rate-limit visitor page for browsers, no mid-token wrapping in event/access tables, scrollable tab row on phones, and Playwright console workflow tests in CI.

diff --git a/.github/workflows/ci.yml b/.github/workflows/ci.yml
index 6d96ee1..a4acb29 100644
--- a/.github/workflows/ci.yml
+++ b/.github/workflows/ci.yml
@@ -28,6 +28,16 @@ jobs:
       - name: Console startup, product wording, and unchanged setting values
         run: node --test tools/tests/test-console-copy.mjs
 
+      - name: Install pinned Playwright + Chromium for console workflow tests
+        run: |
+          mkdir -p "$RUNNER_TEMP/pw" && cd "$RUNNER_TEMP/pw"
+          npm init -y >/dev/null
+          npm install --no-audit --no-fund playwright@1.56.1
+          npx playwright install --with-deps chromium
+
+      - name: Console browser workflows (draft/apply, guided setup, permissions, keyboard)
+        run: WAF_PLAYWRIGHT_MODULE="$RUNNER_TEMP/pw/node_modules/playwright" node --test tools/tests/test-console-workflows.mjs
+
       - name: dependency-free root source-shape gate
         run: ./tools/tests/test-go-root-shape.sh
 
diff --git a/block_response.go b/block_response.go
index c52d78e..9460395 100644
--- a/block_response.go
+++ b/block_response.go
@@ -27,6 +27,13 @@ func blockPageReason(reason string) string {
 	return reason
 }
 
+// acceptsHTML reports whether the client explicitly accepts an HTML page,
+// which browsers do on navigation. API clients that send no Accept header or
+// only application/json are not treated as browsers.
+func acceptsHTML(r *http.Request) bool {
+	return r != nil && strings.Contains(r.Header.Get("Accept"), "text/html")
+}
+
 func writeBlockResponse(w http.ResponseWriter, r *http.Request, status int, reason string, jsonMode bool) {
 	id := requestCorrelationID(r)
 	if id != "" {
diff --git a/block_response_copy_test.go b/block_response_copy_test.go
index 4cd8be4..c10244d 100644
--- a/block_response_copy_test.go
+++ b/block_response_copy_test.go
@@ -35,3 +35,32 @@ func TestBlockPageProductLanguagePreservesAPIReason(t *testing.T) {
 		})
 	}
 }
+
+func TestRateLimitPageForBrowsersJSONForAPIClients(t *testing.T) {
+	c := newL7AbuseController(L7AbuseConfig{Enabled: true, RequestsPerWindow: 1, WindowSeconds: 60})
+	h := c.wrap("site-a", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
+	for _, tc := range []struct {
+		name, accept, contentType, text string
+	}{
+		{"browser", "text/html,application/xhtml+xml,*/*;q=0.8", "text/html", "Please wait and try again"},
+		{"api-no-accept", "", "application/json", `"reason":"l7_abuse_limit"`},
+		{"api-json", "application/json", "application/json", `"reason":"l7_abuse_limit"`},
+	} {
+		t.Run(tc.name, func(t *testing.T) {
+			req := httptest.NewRequest(http.MethodGet, "https://example.test/", nil)
+			req.RemoteAddr = "192.0.2.50:1234"
+			if tc.accept != "" {
+				req.Header.Set("Accept", tc.accept)
+			}
+			h.ServeHTTP(httptest.NewRecorder(), req) // consume the single token
+			rr := httptest.NewRecorder()
+			h.ServeHTTP(rr, req)
+			if rr.Code != http.StatusTooManyRequests {
+				t.Fatalf("status=%d want 429", rr.Code)
+			}
+			if !strings.HasPrefix(rr.Header().Get("Content-Type"), tc.contentType) || !strings.Contains(rr.Body.String(), tc.text) {
+				t.Fatalf("content-type=%q body=%s", rr.Header().Get("Content-Type"), rr.Body.String())
+			}
+		})
+	}
+}
diff --git a/l7_abuse.go b/l7_abuse.go
index c497a07..d269890 100644
--- a/l7_abuse.go
+++ b/l7_abuse.go
@@ -154,7 +154,9 @@ func (c *l7AbuseController) wrap(site string, next http.Handler) http.Handler {
 		}
 		state, allowed := c.allowAt(site, identity, time.Now())
 		if !allowed {
-			writeBlockResponse(w, r, http.StatusTooManyRequests, "l7_abuse_limit", true)
+			// API clients keep the machine-readable JSON body; browsers get the
+			// visitor page explaining the limit.
+			writeBlockResponse(w, r, http.StatusTooManyRequests, "l7_abuse_limit", !acceptsHTML(r))
 			return
 		}
 		if state != nil {
diff --git a/static/admin.html b/static/admin.html
index 0c2d2d1..eb67143 100644
--- a/static/admin.html
+++ b/static/admin.html
@@ -153,6 +153,7 @@
   table{width:100%;border-collapse:collapse;font:12px var(--mono)}
   th{color:var(--mut);text-align:left;font-weight:500;padding:7px 8px;border-bottom:1px solid var(--line);white-space:nowrap}
   td{padding:7px 8px;border-bottom:1px solid var(--line);vertical-align:top;word-break:break-all}
+  td.nw{white-space:nowrap;word-break:normal}
   tr:hover td{background:var(--raised)}
   .sev{border-radius:4px;padding:1px 6px;font-size:11px;white-space:nowrap}
   .sev.critical,.sev.emergency,.sev.alert{background:color-mix(in srgb,var(--red) 20%,transparent);color:var(--red)}
@@ -232,6 +233,7 @@
     .metgrid{grid-template-columns:minmax(0,1fr)}.feedhead,.dashrow,.cardhead,.crawlbar{flex-wrap:wrap}
     .grid2{grid-template-columns:minmax(0,1fr)}.field{min-width:0}.fpolicy{overflow:auto}
     .fbpanel{max-height:90vh}.ux-dialog-body{padding:10px}.tab{padding:10px}
+    .tabs{flex-wrap:nowrap;overflow-x:auto;scrollbar-width:thin}.tab{flex:0 0 auto;white-space:nowrap}
     .dashrow .nm,.dashrow .meta{overflow-wrap:anywhere}.attention-item{overflow-wrap:anywhere}
   }
 </style>
@@ -1145,6 +1147,10 @@
   let mapActive=false;
   function showView(v){
     document.querySelectorAll(".tab").forEach(x=>x.classList.toggle("sel",x.dataset.view===v));
+    // On narrow screens the tab row scrolls sideways; keep the selected tab visible
+    // without scrolling the page itself.
+    const selTab=document.querySelector(".tab.sel"),tabBar=selTab&&selTab.parentElement;
+    if(tabBar&&tabBar.scrollWidth>tabBar.clientWidth) tabBar.scrollLeft+=selTab.getBoundingClientRect().left-tabBar.getBoundingClientRect().left-16;
     document.querySelectorAll(".view").forEach(x=>x.classList.toggle("sel", x.id==="view-"+v));
     mapActive = v==="map";
     if (v==="config") loadDashboard();      // landing = dashboard
@@ -2548,9 +2554,9 @@
       $("homecount").textContent=rows.length?rows.length+" recent":"";
       $("home-empty").style.display=rows.length?"none":"";
       $("homefeed").innerHTML=rows.map(r=>`<tr>
-        <td>${esc(r.time)}</td><td>${esc(r.site)}</td><td>${esc(r.client)}</td>
-        <td><span class="sev ${sevClass(r.severity)}">${esc(uiLabel(r.severity||""))}</span></td>
-        <td>${r.rule_id}</td><td class="mono">${esc(r.uri)}</td><td>${esc(r.msg||"")}</td></tr>`).join("");
+        <td class="nw">${esc(r.time)}</td><td class="nw">${esc(r.site)}</td><td class="nw">${esc(r.client)}</td>
+        <td class="nw"><span class="sev ${sevClass(r.severity)}">${esc(uiLabel(r.severity||""))}</span></td>
+        <td class="nw">${Number(r.rule_id)||0}</td><td class="mono">${esc(r.uri)}</td><td>${esc(r.msg||"")}</td></tr>`).join("");
     }catch(e){}
   }
   function modePill(m){
@@ -2641,9 +2647,9 @@
         rows=rows.filter(r=>logMatchesFilter(r,q));
         $("loghead").innerHTML=`<tr><th>Time</th><th>Site</th><th>Client</th><th>Severity</th><th>Rule</th><th>Inspection step</th><th>URL</th><th>Message</th><th>Review</th></tr>`;
         $("logbody").innerHTML=rows.map(r=>`<tr>
-          <td>${esc(r.time)}</td><td>${esc(r.site)}</td><td>${esc(r.client)}</td>
-          <td><span class="sev ${sevClass(r.severity)}">${esc(uiLabel(r.severity||""))}</span></td>
-          <td>${Number(r.rule_id)||0}</td><td>${esc(inspectionStep(r.phase))}</td><td class="mono">${esc(r.uri)}</td><td>${esc(r.msg||"")}</td><td><button class="btn sm" data-event="${uxState.events.indexOf(r)}">Details</button></td></tr>`).join("");
+          <td class="nw">${esc(r.time)}</td><td class="nw">${esc(r.site)}</td><td class="nw">${esc(r.client)}</td>
+          <td class="nw"><span class="sev ${sevClass(r.severity)}">${esc(uiLabel(r.severity||""))}</span></td>
+          <td class="nw">${Number(r.rule_id)||0}</td><td>${esc(inspectionStep(r.phase))}</td><td class="mono">${esc(r.uri)}</td><td>${esc(r.msg||"")}</td><td><button class="btn sm" data-event="${uxState.events.indexOf(r)}">Details</button></td></tr>`).join("");
         $("logbody").querySelectorAll("[data-event]").forEach(b=>b.onclick=()=>showEvent(uxState.events[+b.dataset.event]).catch(e=>toast(e.message,"err")));
         $("logcount").textContent=rows.length+" events";
         $("log-empty").style.display=rows.length?"none":"";
@@ -2653,8 +2659,8 @@
         rows=rows.filter(r=>logMatchesFilter(r,q));
         $("loghead").innerHTML=`<tr><th>Time</th><th>Site</th><th>Client</th><th>Method</th><th>Path</th><th>Status</th></tr>`;
         $("logbody").innerHTML=rows.map(r=>`<tr>
-          <td>${esc(r.time)}</td><td>${esc(r.site)}</td><td>${esc(r.client)}</td>
-          <td>${esc(r.method)}</td><td class="mono">${esc(r.path)}</td>
+          <td class="nw">${esc(r.time)}</td><td class="nw">${esc(r.site)}</td><td class="nw">${esc(r.client)}</td>
+          <td class="nw">${esc(r.method)}</td><td class="mono">${esc(r.path)}</td>
           <td><b style="color:${statusColor(r.status)}">${r.status}</b></td></tr>`).join("");
         $("logcount").textContent=rows.length+" requests";
         $("log-empty").style.display=rows.length?"none":"";


## Saved-draft control visibility — 2026-10-24

Hide Continue/Discard saved draft when the server reports no saved draft; new Playwright workflow test.

diff --git a/static/admin.html b/static/admin.html
index 72c959e..3b49714 100644
--- a/static/admin.html
+++ b/static/admin.html
@@ -154,6 +154,7 @@
   th{color:var(--mut);text-align:left;font-weight:500;padding:7px 8px;border-bottom:1px solid var(--line);white-space:nowrap}
   td{padding:7px 8px;border-bottom:1px solid var(--line);vertical-align:top;word-break:break-all}
   td.nw{white-space:nowrap;word-break:normal}
+  .workbar .btn[hidden]{display:none}
   tr:hover td{background:var(--raised)}
   .sev{border-radius:4px;padding:1px 6px;font-size:11px;white-space:nowrap}
   .sev.critical,.sev.emergency,.sev.alert{background:color-mix(in srgb,var(--red) 20%,transparent);color:var(--red)}
@@ -1134,10 +1135,27 @@
       const r=await fetch(path,{...opts,headers,signal:controller.signal});
       if(r.status===401){sessionStorage.removeItem("waf_token");showGate();}
       if(!r.ok){const e=new Error(r.status===401?"Session expired — sign in again":r.status===403?"This account does not have access.":(await r.text()).trim()||("HTTP "+r.status));e.status=r.status;throw e;}
-      const result=await r.json();resourceSuccess(path);return result;
+      const result=await r.json();resourceSuccess(path);noteServerDraft(path,method,r);return result;
     }catch(error){const e=error.name==="AbortError"?new Error("The request timed out. Check the connection and retry."):error;resourceFailure(path,e,mutation?"ACTION":"GET");throw e;}
     finally{clearTimeout(timer);if(registered)uxState.pending.delete(key);}
   }
+  // The server returns a saved draft only when it is newer than the active
+  // configuration and labels the response with X-WAF-Config-Source. Track that
+  // so the draft controls are hidden only when no saved draft exists.
+  function noteServerDraft(path,method,response){
+    let known;
+    if(path==="/api/config?draft=1"){
+      if(method==="GET"){const src=response.headers.get("X-WAF-Config-Source");if(src==="draft"||src==="live")known=src==="draft";}
+      else if(method==="PUT")known=true;
+      else if(method==="DELETE")known=false;
+    }else if(path==="/api/config"&&method==="PUT")known=false;
+    if(known===undefined)return;
+    uxState.serverDraft=known;updateDraftControls();
+  }
+  function updateDraftControls(){
+    const none=uxState.serverDraft===false;
+    $("ux_resume_draft").hidden=none;$("ux_discard_draft").hidden=none;
+  }
   async function downloadAPI(path, filename){
     const r=await fetch(path,{headers:{"Authorization":"Bearer "+getToken()}});
     if(r.status===401){sessionStorage.removeItem("waf_token");showGate();throw new Error("session expired — sign in again");}
@@ -3061,7 +3079,7 @@
   // secrets are never written to browser storage or shown in change previews.
   const uxState={ready:false,saved:null,active:null,activeRaw:null,baseline:"",writing:false,
     resources:new Map(),cache:new Map(),pending:new Set(),endpoints:[],events:[],access:[],scan:new Map(),scanTimers:new Map(),scanResults:new Map(),
-    pageHistory:new Map(),pageOriginal:null,dialogResolve:null,dialogReturn:null,attention:[],previousErrorShare:null};
+    pageHistory:new Map(),pageOriginal:null,serverDraft:null,dialogResolve:null,dialogReturn:null,attention:[],previousErrorShare:null};
   const copyJSON=value=>JSON.parse(JSON.stringify(value));
   function stableConfig(value){
     if(Array.isArray(value))return value.map(stableConfig);
diff --git a/tools/tests/test-console-workflows.mjs b/tools/tests/test-console-workflows.mjs
index 0c0a3c4..ed5b191 100644
--- a/tools/tests/test-console-workflows.mjs
+++ b/tools/tests/test-console-workflows.mjs
@@ -505,6 +505,27 @@ test('discard failure keeps the saved draft available and never claims success',
   } finally { await s.close(); }
 });
 
+test('saved-draft controls hide when the server reports no saved draft and return after saving', async () => {
+  const s = await session();
+  try {
+    await s.page.route('**/api/config?draft=1', route => route.request().method() === 'GET'
+      ? route.fulfill({ status: 200, contentType: 'application/json', headers: { 'X-WAF-Config-Source': 'live' }, body: JSON.stringify(s.active) })
+      : route.fallback());
+    await s.page.locator('#ux_resume_draft').click();
+    await s.page.locator('#ux_discard_draft').waitFor({ state: 'hidden' });
+    assert.equal(await s.page.locator('#ux_resume_draft').isHidden(), true);
+    await tab(s.page, 'sites');await s.page.locator('#site_list [data-sel]').first().click();
+    await s.page.locator('#sites input[data-f="listen"]').fill(':8081');await s.page.locator('#ux_save').click();
+    await s.page.locator('#ux_change_state').filter({ hasText: 'Draft saved' }).waitFor();
+    assert.equal(await s.page.locator('#ux_discard_draft').isVisible(), true);
+    assert.equal(await s.page.locator('#ux_resume_draft').isVisible(), true);
+    await s.page.locator('#ux_review').click();await choice(s.page, 'apply');
+    await s.page.locator('#ux_change_state').filter({ hasText: 'matches active settings' }).waitFor();
+    assert.equal(await s.page.locator('#ux_discard_draft').isHidden(), true);
+    assert.deepEqual(s.errors, []);
+  } finally { await s.close(); }
+});
+
 test('reviewer and viewer cannot discard the shared saved draft', async () => {
   for (const role of ['reviewer', 'viewer']) {
     const s = await session({ role });


## Console change bar and API section navigation — 2026-10-29

Disabled-button styling, Review disabled with no pending changes, collapsing clean change bar on phones, pinned and scroll-synced API section menu, section-jump overshoot fix, and secondary "Add another site" call to action.

diff --git a/static/admin.html b/static/admin.html
index 3b49714..76c6b8b 100644
--- a/static/admin.html
+++ b/static/admin.html
@@ -111,6 +111,7 @@
   .btn.primary:hover{filter:brightness(1.08)}
   .btn.danger{color:var(--red)}
   .btn.danger:hover{border-color:var(--red)}
+  .btn:disabled{opacity:.45;cursor:not-allowed}.btn:disabled:hover{border-color:var(--line);filter:none}
 
   .card{border:1px solid var(--line);border-radius:8px;padding:14px;margin-bottom:14px;background:var(--bg)}
   .cardhead{display:flex;align-items:center;gap:10px;margin-bottom:12px}
@@ -155,6 +156,15 @@
   td{padding:7px 8px;border-bottom:1px solid var(--line);vertical-align:top;word-break:break-all}
   td.nw{white-space:nowrap;word-break:normal}
   .workbar .btn[hidden]{display:none}
+  /* The API section jump stays reachable below the pinned change bar while scrolling. */
+  .api-jump{position:sticky;top:var(--sticky-top,0px);z-index:30;background:var(--bg);padding:8px 0}
+  #view-api section{scroll-margin-top:calc(var(--sticky-top,0px) + 96px)}
+  @media(max-width:600px){
+    /* With nothing to save or apply, the change bar scrolls away and shows only
+       its status and remaining actions; it pins again once there are edits. */
+    .workbar.clean{position:static}
+    .workbar.clean #ux_review,.workbar.clean #ux_save,.workbar.clean #ux_discard{display:none}
+  }
   tr:hover td{background:var(--raised)}
   .sev{border-radius:4px;padding:1px 6px;font-size:11px;white-space:nowrap}
   .sev.critical,.sev.emergency,.sev.alert{background:color-mix(in srgb,var(--red) 20%,transparent);color:var(--red)}
@@ -443,7 +453,7 @@
 
   <!-- ══ API SECURITY ══ -->
   <div id="view-api" class="view">
-    <div class="row" style="margin-bottom:14px"><div class="field"><label for="ux_api_section">Go to API security section</label><select id="ux_api_section"><option value="api_ops_body">Discovered endpoints</option><option value="api_schema_body">Suggested field definitions</option><option value="api_enforce_body">Field protection</option><option value="api_identity_issuers">Sign-in providers</option><option value="api_sequence_states">API usage patterns</option><option value="api_object_locators">Record ID fields</option><option value="api_bola_evidence">Record-access findings</option><option value="api_graphql_operations">GraphQL</option><option value="api_contract_body">OpenAPI definitions</option></select></div><div class="hint">Each section loads independently. Endpoint selectors show the site, method, and URL.</div></div>
+    <div class="row api-jump" style="margin-bottom:14px"><div class="field"><label for="ux_api_section">Go to API security section</label><select id="ux_api_section"><option value="api_ops_body">Discovered endpoints</option><option value="api_schema_body">Suggested field definitions</option><option value="api_enforce_body">Field protection</option><option value="api_identity_issuers">Sign-in providers</option><option value="api_sequence_states">API usage patterns</option><option value="api_object_locators">Record ID fields</option><option value="api_bola_evidence">Record-access findings</option><option value="api_graphql_operations">GraphQL</option><option value="api_contract_body">OpenAPI definitions</option></select></div><div class="hint">Each section loads independently. Endpoint selectors show the site, method, and URL.</div></div>
     <section>
       <div class="feedhead"><div><div class="eyebrow">API endpoints</div><h3 style="margin:2px 0 0">Discovered endpoints</h3></div><div class="count" id="api_ops_count"></div><button class="btn sm" id="api_refresh">Refresh</button></div>
       <div class="hint" style="margin:6px 0 12px">API endpoints are discovered from live traffic. Organizing, hiding, or reclassifying an endpoint does not change its protection settings.</div>
@@ -3114,6 +3124,16 @@
     const current=snapshotEditors(),pending=configChanges(uxState.active,current),dirty=configText(current)!==uxState.baseline;
     $("ux_change_state").textContent=`${dirty?"Unsaved edits":pending.length?"Draft saved":"Active settings"} · ${pending.length}${pending.length>=200?"+":""} changes compared with active settings${pending.length?" — not active yet":" — matches active settings"}`;
     $("ux_save").disabled=!hasCapability("edit")||!dirty;$("ux_discard").disabled=!dirty;
+    $("ux_review").disabled=!pending.length;
+    $("ux_workbar").classList.toggle("clean",!dirty&&!pending.length);
+    // The dashboard call to action becomes a secondary "add" once a site is active.
+    const hasSite=((uxState.active&&uxState.active.sites)||[]).length>0,wizard=$("ux_wizard");
+    wizard.textContent=hasSite?"Add another site":"Set up a site";wizard.classList.toggle("primary",!hasSite);
+    updateStickyOffset();
+  }
+  function updateStickyOffset(){
+    const bar=$("ux_workbar"),pinned=getComputedStyle(bar).position==="sticky";
+    document.documentElement.style.setProperty("--sticky-top",(pinned?bar.offsetHeight:0)+"px");
   }
   function syncVisibleEditors(){if(!cfg)return;syncSites();syncNodes();syncPools();syncPolicies();syncCIDRRules();}
   function resourceLabel(path){
@@ -3566,7 +3586,33 @@
     $("ux_page_preview").onclick=()=>chooseAction("Page change preview",changeTable(configChanges(uxState.pageOriginal,readPagePolicy())),[['close','Return to editor']]);
     $("ux_dialog_close").onclick=()=>closeUXDialog();
     $("ux_dialog").onclick=e=>{if(e.target===$("ux_dialog"))closeUXDialog();};
-    $("ux_api_section").onchange=()=>{const target=$($("ux_api_section").value)?.closest("section");if(target){target.setAttribute("tabindex","-1");target.scrollIntoView({behavior:"smooth",block:"start"});target.focus();}};
+    if("ResizeObserver" in window)new ResizeObserver(updateStickyOffset).observe($("ux_workbar"));
+    window.addEventListener("resize",updateStickyOffset);
+    (()=>{
+      // Keep the section menu on the section the operator is reading: the last
+      // section whose top has passed just below the pinned menu.
+      const menu=$("ux_api_section");if(!menu)return;
+      const sections=[...menu.options].map(o=>[$(o.value)?.closest("section"),o.value]).filter(([section])=>section)
+        .sort(([a],[b])=>a.compareDocumentPosition(b)&Node.DOCUMENT_POSITION_FOLLOWING?-1:1);
+      // A section chosen from the menu stays selected until the operator scrolls
+      // by hand; otherwise a jump near the end of the page would be relabelled.
+      let queued=false,chosen=null;
+      menu.addEventListener("change",()=>{chosen=menu.value;});
+      for(const type of ["wheel","touchmove","keydown"])
+        window.addEventListener(type,event=>{if(event.target!==menu)chosen=null;},{passive:true});
+      const sync=()=>{
+        queued=false;
+        if(!sections.length||menu.offsetParent===null||document.activeElement===menu)return;
+        if(chosen){menu.value=chosen;return;}
+        const line=menu.closest(".api-jump").getBoundingClientRect().bottom+40;
+        let current=sections[0][1];
+        for(const [section,value] of sections)if(section.getBoundingClientRect().top<=line)current=value;
+        if(innerHeight+scrollY>=document.documentElement.scrollHeight-2)current=sections[sections.length-1][1];
+        menu.value=current;
+      };
+      window.addEventListener("scroll",()=>{if(!queued){queued=true;requestAnimationFrame(sync);}},{passive:true});
+    })();
+    $("ux_api_section").onchange=()=>{const target=$($("ux_api_section").value)?.closest("section");if(target){target.setAttribute("tabindex","-1");target.focus({preventScroll:true});target.scrollIntoView({behavior:"smooth",block:"start"});}};
     $("ux_errors").onclick=async e=>{const retry=e.target.closest?.("[data-resource-retry]");if(!retry)return;const path=retry.dataset.resourceRetry;
       retry.disabled=true;
       try{if(!uxState.ready){await startConsole();return;}
diff --git a/tools/tests/test-console-workflows.mjs b/tools/tests/test-console-workflows.mjs
index ed5b191..136f076 100644
--- a/tools/tests/test-console-workflows.mjs
+++ b/tools/tests/test-console-workflows.mjs
@@ -526,6 +526,61 @@ test('saved-draft controls hide when the server reports no saved draft and retur
   } finally { await s.close(); }
 });
 
+test('change bar: review needs pending changes, disabled actions look disabled, phone bar collapses when clean', async () => {
+  const s = await session();
+  try {
+    const review = s.page.locator('#ux_review');
+    assert.equal(await review.isDisabled(), true);
+    assert.ok(Number(await review.evaluate(el => getComputedStyle(el).opacity)) < 1);
+    assert.equal(await s.page.locator('#ux_wizard').innerText(), 'Add another site');
+    assert.equal(await s.page.locator('#ux_wizard').evaluate(el => el.classList.contains('primary')), false);
+    await s.page.setViewportSize({ width: 390, height: 844 });
+    assert.equal(await s.page.locator('#ux_workbar').evaluate(el => getComputedStyle(el).position), 'static');
+    assert.equal(await review.isHidden(), true);
+    assert.equal(await s.page.locator('#ux_save').isHidden(), true);
+    await tab(s.page, 'sites');await s.page.locator('#site_list [data-sel]').first().click();
+    await s.page.locator('#sites input[data-f="listen"]').fill(':8081');
+    await s.page.locator('#ux_change_state').filter({ hasText: 'Unsaved edits' }).waitFor();
+    assert.equal(await s.page.locator('#ux_workbar').evaluate(el => getComputedStyle(el).position), 'sticky');
+    assert.equal(await review.isVisible(), true);
+    assert.equal(await review.isDisabled(), false);
+    assert.equal(await s.page.locator('#ux_save').isVisible(), true);
+    assert.equal(await s.page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth), true);
+    assert.deepEqual(s.errors, []);
+  } finally { await s.close(); }
+});
+
+test('API section menu stays pinned below the change bar and follows scrolling', async () => {
+  const s = await session();
+  try {
+    await tab(s.page, 'api');
+    const jump = s.page.locator('.api-jump');
+    assert.equal(await jump.evaluate(el => getComputedStyle(el).position), 'sticky');
+    const barBottom = await s.page.locator('#ux_workbar').evaluate(el => el.getBoundingClientRect().height);
+    assert.ok(parseFloat(await jump.evaluate(el => getComputedStyle(el).top)) >= barBottom);
+    const options = await s.page.locator('#ux_api_section option').evaluateAll(list => list.map(o => o.value));
+    const target = options[Math.min(3, options.length - 1)];
+    await s.page.evaluate(id => document.getElementById(id).closest('section').scrollIntoView({ block: 'start' }), target);
+    await s.page.waitForFunction(id => document.getElementById('ux_api_section').value === id, target);
+    const heading = await s.page.evaluate(id => document.getElementById(id).closest('section').getBoundingClientRect().top, target);
+    const jumpBottom = await jump.evaluate(el => el.getBoundingClientRect().bottom);
+    assert.ok(heading >= jumpBottom - 1, `section top ${heading} hidden under pinned menu ending at ${jumpBottom}`);
+    // Choosing a section from the menu lands it just below the pinned menu and the
+    // menu keeps the choice, even when the page cannot scroll that section to the top.
+    for (const chosen of [options[1], options[Math.min(5, options.length - 1)]]) {
+      await s.page.selectOption('#ux_api_section', chosen);
+      await s.page.waitForFunction(id => {
+        const top = document.getElementById(id).closest('section').getBoundingClientRect().top;
+        const bottom = document.querySelector('.api-jump').getBoundingClientRect().bottom;
+        return top >= bottom - 1 && top < bottom + 140;
+      }, chosen);
+      await s.page.waitForTimeout(400);
+      assert.equal(await s.page.locator('#ux_api_section').inputValue(), chosen);
+    }
+    assert.deepEqual(s.errors, []);
+  } finally { await s.close(); }
+});
+
 test('reviewer and viewer cannot discard the shared saved draft', async () => {
   for (const role of ['reviewer', 'viewer']) {
     const s = await session({ role });


## Review fixes: CI, Coraza v3.8.1, attack alerts, delivery health — 2026-11-26

Base `e371979`. See AI_HANDOFF.md (2026-11-26). Source/test/CI diff (documentation excluded):

diff --git a/.github/workflows/ci.yml b/.github/workflows/ci.yml
index a4acb29..51ffe8d 100644
--- a/.github/workflows/ci.yml
+++ b/.github/workflows/ci.yml
@@ -12,7 +12,7 @@ permissions:
 jobs:
   build-test:
     name: build, vet, test (portable)
-    runs-on: ubuntu-latest
+    runs-on: ubuntu-24.04
     steps:
       - uses: actions/checkout@v4
 
@@ -28,16 +28,6 @@ jobs:
       - name: Console startup, product wording, and unchanged setting values
         run: node --test tools/tests/test-console-copy.mjs
 
-      - name: Install pinned Playwright + Chromium for console workflow tests
-        run: |
-          mkdir -p "$RUNNER_TEMP/pw" && cd "$RUNNER_TEMP/pw"
-          npm init -y >/dev/null
-          npm install --no-audit --no-fund playwright@1.56.1
-          npx playwright install --with-deps chromium
-
-      - name: Console browser workflows (draft/apply, guided setup, permissions, keyboard)
-        run: WAF_PLAYWRIGHT_MODULE="$RUNNER_TEMP/pw/node_modules/playwright" node --test tools/tests/test-console-workflows.mjs
-
       - name: dependency-free root source-shape gate
         run: ./tools/tests/test-go-root-shape.sh
 
@@ -105,9 +95,28 @@ jobs:
       - name: real Coraza transaction truth gate
         run: GOTOOLCHAIN=local CGO_ENABLED=1 go test -tags realcoraza -run 'TestRealCoraza' ./...
 
+  # Browser workflow tests run in their own job so a browser-install problem can
+  # never skip the Go build, test and source gates above.
+  console-browser:
+    name: console browser workflows
+    runs-on: ubuntu-24.04
+    steps:
+      - uses: actions/checkout@v4
+      - uses: actions/setup-node@v4
+        with:
+          node-version: '22'
+      - name: Install pinned Playwright + Chromium
+        run: |
+          mkdir -p "$RUNNER_TEMP/pw" && cd "$RUNNER_TEMP/pw"
+          npm init -y >/dev/null
+          npm install --no-audit --no-fund playwright@1.56.1
+          npx playwright install --with-deps chromium
+      - name: Console browser workflows (draft/apply, guided setup, permissions, keyboard)
+        run: WAF_PLAYWRIGHT_MODULE="$RUNNER_TEMP/pw/node_modules/playwright" node --test tools/tests/test-console-workflows.mjs
+
   govulncheck:
     name: dependency vulnerabilities
-    runs-on: ubuntu-latest
+    runs-on: ubuntu-24.04
     steps:
       - uses: actions/checkout@v4
       - uses: actions/setup-go@v5
diff --git a/admin.go b/admin.go
index 171ed02..7b0e0e4 100644
--- a/admin.go
+++ b/admin.go
@@ -506,6 +506,7 @@ func (a *adminServer) handleStatus(w http.ResponseWriter, _ *http.Request) {
 		"notify_unread":       a.srv.notify.unreadCount(),
 		"webhook_delivery":    a.srv.notify.webhookDeliveryStats(),
 		"syslog_delivery":     a.srv.syslog.deliveryStats(),
+		"l7_saturation":       rt.l7Abuse.saturationStats(),
 	})
 }
 
diff --git a/attack_alerts.go b/attack_alerts.go
index d3d2750..c315eb2 100644
--- a/attack_alerts.go
+++ b/attack_alerts.go
@@ -6,32 +6,73 @@ import (
 	"fmt"
 	"net/http"
 	"strings"
+	"sync"
 	"time"
 )
 
 type attackSignalKey struct{}
+
+// attackSignals is per-request evidence gathered while the request is served:
+// whether a security control blocked it, and the highest severity of the WAF
+// rules it matched. It lets the alert engine count requests, not rule hits.
 type attackSignals struct {
-	blocked bool
+	mu       sync.Mutex
+	blocked  bool
+	severity string // highest matched rule severity; "" when nothing matched
 }
 
+// markAttackBlock records a block by a security control (WAF interruption or
+// API schema/identity/GraphQL enforcement). Rate limiting and IP allowlists are
+// access controls and deliberately do not call this.
 func markAttackBlock(r *http.Request) {
+	if r == nil {
+		return
+	}
 	if signals, ok := r.Context().Value(attackSignalKey{}).(*attackSignals); ok {
+		signals.mu.Lock()
 		signals.blocked = true
+		signals.mu.Unlock()
+	}
+}
+
+func (a *attackSignals) noteSeverity(severity string) {
+	rank := severityRank(severity)
+	if a == nil || rank < 0 {
+		return
+	}
+	a.mu.Lock()
+	defer a.mu.Unlock()
+	if current := severityRank(a.severity); current < 0 || rank < current {
+		a.severity = strings.ToUpper(severity)
 	}
 }
 
+func (a *attackSignals) snapshot() (blocked bool, severity string) {
+	a.mu.Lock()
+	defer a.mu.Unlock()
+	return a.blocked, a.severity
+}
+
 func requestAttackSignals(r *http.Request) (*http.Request, *attackSignals) {
 	signals := &attackSignals{}
 	return r.WithContext(context.WithValue(r.Context(), attackSignalKey{}, signals)), signals
 }
 
+// attackWindow tracks one site. Requests are counted inside a window; once an
+// alert fires, the site is in an incident until activity stops for one full
+// window, with at most one follow-up per attackReminderInterval while it lasts.
 type attackWindow struct {
-	start   time.Time
-	matches int
-	blocks  int
-	emitted bool
+	start         time.Time
+	requests      int
+	blocks        int
+	lastEvent     time.Time
+	incidentStart time.Time
+	alertedAt     time.Time
+	incidentTotal int
 }
 
+const attackReminderInterval = time.Hour
+
 func severityRank(value string) int {
 	switch strings.ToUpper(value) {
 	case "EMERGENCY":
@@ -58,6 +99,9 @@ func (c NotifyConfig) validate() error {
 	if err := validateWebhookURL(c.WebhookURL, c); err != nil {
 		return err
 	}
+	if err := validateWebhookProxyURL(c.WebhookProxyURL); err != nil {
+		return err
+	}
 	if c.AttackThreshold < 0 || c.AttackThreshold > 1000000 {
 		return errors.New("notify.attack_threshold must be 0..1000000 (0 defaults to 10)")
 	}
@@ -70,18 +114,20 @@ func (c NotifyConfig) validate() error {
 	return nil
 }
 
-// Counts matches and blocks independently, with one emitted alert per site/window.
-// There are no client/URL/rule-ID keys, raw request values or network calls here.
+// observeAttack records one suspicious request for a site: kind "block" when
+// a security control blocked it, or "match" with the request's highest matched
+// rule severity. It is called at most once per request, so the threshold
+// counts requests, not rule hits. No client, URL, rule ID or request content
+// is kept, and nothing here makes network calls.
 func (n *notifier) observeAttack(site, kind, severity string, now time.Time) {
-	if n == nil {
+	if n == nil || !n.attackOn.Load() {
 		return
 	}
-	n.mu.Lock()
-	cfg := n.cfg
-	if !cfg.OnAttack {
-		n.mu.Unlock()
+	if kind != "match" && kind != "block" {
 		return
 	}
+	n.mu.Lock()
+	cfg := n.cfg
 	minimum := cfg.AttackMinSeverity
 	if minimum == "" {
 		minimum = "CRITICAL"
@@ -90,10 +136,6 @@ func (n *notifier) observeAttack(site, kind, severity string, now time.Time) {
 		n.mu.Unlock()
 		return
 	}
-	if kind != "match" && kind != "block" {
-		n.mu.Unlock()
-		return
-	}
 	window := time.Duration(cfg.AttackWindowSeconds) * time.Second
 	if window == 0 {
 		window = time.Minute
@@ -105,11 +147,10 @@ func (n *notifier) observeAttack(site, kind, severity string, now time.Time) {
 	if n.attacks == nil {
 		n.attacks = make(map[string]*attackWindow)
 	}
-	key := site
-	state := n.attacks[key]
+	state := n.attacks[site]
 	if state == nil {
 		for k, old := range n.attacks {
-			if !now.Before(old.start.Add(window)) {
+			if !now.Before(old.lastEvent.Add(window)) {
 				delete(n.attacks, k)
 			}
 		}
@@ -117,34 +158,60 @@ func (n *notifier) observeAttack(site, kind, severity string, now time.Time) {
 			n.mu.Unlock()
 			return
 		}
-		state = &attackWindow{start: now}
-		n.attacks[key] = state
-	}
-	if !now.Before(state.start.Add(window)) {
-		*state = attackWindow{start: now}
+		state = &attackWindow{start: now, lastEvent: now}
+		n.attacks[site] = state
 	}
-	if state.emitted {
+
+	// An open incident ends after one full window without suspicious requests.
+	if !state.alertedAt.IsZero() && !now.Before(state.lastEvent.Add(window)) {
+		*state = attackWindow{start: now, lastEvent: now}
+	}
+	if !state.alertedAt.IsZero() {
+		state.lastEvent = now
+		state.incidentTotal++
+		if kind == "block" {
+			state.blocks++
+		}
+		if now.Sub(state.alertedAt) < attackReminderInterval {
+			n.mu.Unlock()
+			return
+		}
+		state.alertedAt = now
+		total, blocks, startedAt := state.incidentTotal, state.blocks, state.incidentStart
 		n.mu.Unlock()
+		n.pushContext(notifyAttack, "alert", "WAF activity still ongoing",
+			fmt.Sprintf("Site %s: %d suspicious requests (%d blocked) since this incident started; matches at or above %s. Matches are not necessarily blocked requests.", site, total, blocks, minimum), "", "", nil,
+			&notificationContext{Site: site, Kind: attackContextKind(blocks), StartedAt: startedAt.UTC(), EndedAt: now.UTC()})
 		return
 	}
-	count := 0
-	if kind == "match" {
-		state.matches++
-		count = state.matches
-	} else {
+
+	if !now.Before(state.start.Add(window)) {
+		*state = attackWindow{start: now}
+	}
+	state.lastEvent = now
+	state.requests++
+	if kind == "block" {
 		state.blocks++
-		count = state.blocks
 	}
-	if count < threshold {
+	if state.requests < threshold {
 		n.mu.Unlock()
 		return
 	}
-	state.emitted = true
-	startedAt := state.start
+	state.alertedAt = now
+	state.incidentStart = state.start
+	state.incidentTotal = state.requests
+	requests, blocks, startedAt := state.requests, state.blocks, state.start
 	n.mu.Unlock()
-	// Window-level suppression above is authoritative; do not use the generic
-	// dedupe map which can shorten a deliberately configured alert window.
+	// Incident-level suppression above is authoritative; do not use the generic
+	// dedupe map, which could shorten a deliberately configured window.
 	n.pushContext(notifyAttack, "alert", "WAF activity threshold reached",
-		fmt.Sprintf("Site %s: %d %s events in %ds; minimum match severity %s. Matches are not necessarily blocked requests.", site, count, kind, int(window.Seconds()), minimum), "", "", nil,
-		&notificationContext{Site: site, Kind: kind, StartedAt: startedAt.UTC(), EndedAt: now.UTC()})
+		fmt.Sprintf("Site %s: %d suspicious requests in %ds (%d blocked); matches at or above %s. One alert per incident; a follow-up is sent hourly while it continues. Matches are not necessarily blocked requests.", site, requests, int(window.Seconds()), blocks, minimum), "", "", nil,
+		&notificationContext{Site: site, Kind: attackContextKind(blocks), StartedAt: startedAt.UTC(), EndedAt: now.UTC()})
+}
+
+func attackContextKind(blocks int) string {
+	if blocks > 0 {
+		return "block"
+	}
+	return "match"
 }
diff --git a/block_response.go b/block_response.go
index 1c8130f..9460395 100644
--- a/block_response.go
+++ b/block_response.go
@@ -35,9 +35,6 @@ func acceptsHTML(r *http.Request) bool {
 }
 
 func writeBlockResponse(w http.ResponseWriter, r *http.Request, status int, reason string, jsonMode bool) {
-	if status >= 400 && status < 500 {
-		markAttackBlock(r)
-	}
 	id := requestCorrelationID(r)
 	if id != "" {
 		w.Header().Set("X-Request-ID", id)
diff --git a/build.sh b/build.sh
index 8ac5307..6aae7f1 100755
--- a/build.sh
+++ b/build.sh
@@ -163,7 +163,7 @@ links=[]
 if "vectorscan" in variant: links.append("DYNAMIC_LIBHS_REQUIRED_AT_RUNTIME")
 else: links.append("NO_LIBHS_LINK")
 if hsm == "enabled": links.append("PKCS11_DLOPEN_RUNTIME_WHEN_CONFIGURED")
-obj={"schema_version":2,"version":version,"commit":commit,"artifact_type":artifact_type,"build_variant":variant,"go_version":gov,"coraza_version":"v3.7.0","vectorscan_version":vs,"hsm_pkcs11_enabled":hsm=="enabled","linkage_expectation":";".join(links),"binaries":files}
+obj={"schema_version":2,"version":version,"commit":commit,"artifact_type":artifact_type,"build_variant":variant,"go_version":gov,"coraza_version":"v3.8.1","vectorscan_version":vs,"hsm_pkcs11_enabled":hsm=="enabled","linkage_expectation":";".join(links),"binaries":files}
 pathlib.Path("BUILD_PROVENANCE.json").write_text(json.dumps(obj,indent=2,sort_keys=True)+"\n",encoding="utf-8")
 pathlib.Path("BUILD_SHA256SUMS.txt").write_text("".join(f"{x['sha256']}  {x['name']}\n" for x in files),encoding="utf-8")
 PY2
diff --git a/coraza_observer.go b/coraza_observer.go
index 8c729d5..257bec8 100644
--- a/coraza_observer.go
+++ b/coraza_observer.go
@@ -68,9 +68,16 @@ func (t *observedCorazaTx) observe() {
 		// Use the authoritative interruption, not response status: an application
 		// 403 is not a WAF block; response-phase denies count as well as requests.
 		if in := t.Transaction.Interruption(); in != nil && t.attack != nil && (in.Action == "deny" || in.Action == "drop") {
+			t.attack.mu.Lock()
 			t.attack.blocked = true
+			t.attack.mu.Unlock()
 		}
 		matches := t.Transaction.MatchedRules()
+		if t.attack != nil {
+			for _, m := range matches {
+				t.attack.noteSeverity(m.Rule().Severity().String())
+			}
+		}
 		ids := make([]int, 0, len(matches))
 		seen := make(map[int]struct{}, len(matches))
 		for _, m := range matches {
diff --git a/delivery_health.go b/delivery_health.go
index e045ccf..6f68bca 100644
--- a/delivery_health.go
+++ b/delivery_health.go
@@ -3,11 +3,18 @@ package main
 import (
 	"crypto/rand"
 	"encoding/hex"
+	"errors"
+	"net/url"
 	"strconv"
+	"strings"
 	"sync"
 	"time"
 )
 
+// recentDropWindow is how long a dropped message keeps the delivery flagged
+// for review. A brief outage stops warning on its own once it is over.
+const recentDropWindow = time.Hour
+
 // Transport observations, not a promise that a SIEM received/indexed a message.
 type deliveryHealth struct {
 	mu          sync.Mutex
@@ -15,10 +22,50 @@ type deliveryHealth struct {
 	failed      uint64
 	lastSuccess time.Time
 	lastFailure time.Time
+	lastError   string
+	lastDrop    time.Time
 	generation  uint64
 	epoch       string
 }
 
+// deliveryErrorReason turns a delivery error into a short reason that is safe
+// to show and store: URL wrappers are removed because a webhook URL or its
+// query can carry credentials.
+func deliveryErrorReason(err error) string {
+	if err == nil {
+		return ""
+	}
+	var ue *url.Error
+	if errors.As(err, &ue) {
+		err = ue.Err
+	}
+	reason := err.Error()
+	if strings.Contains(reason, "://") {
+		reason = "request failed"
+	}
+	if len(reason) > 160 {
+		reason = reason[:160]
+	}
+	return reason
+}
+
+// fail records a failed delivery attempt with its reason.
+func (h *deliveryHealth) fail(generation uint64, err error) {
+	h.recordAt(generation, false)
+	h.mu.Lock()
+	defer h.mu.Unlock()
+	if generation == h.generation {
+		h.lastError = deliveryErrorReason(err)
+	}
+}
+
+// noteDrop records that a message was dropped (queue full, backoff, failure).
+func (h *deliveryHealth) noteDrop() {
+	h.mu.Lock()
+	defer h.mu.Unlock()
+	h.lastDrop = time.Now().UTC()
+}
+
 // A process-specific nonce prevents a stale confirmation from matching another
 // node/restart that happens to have the same numeric configuration generation.
 func (h *deliveryHealth) configToken() string {
@@ -50,6 +97,8 @@ func (h *deliveryHealth) invalidate() {
 	h.generation++
 	h.lastSuccess = time.Time{}
 	h.lastFailure = time.Time{}
+	h.lastError = ""
+	h.lastDrop = time.Time{}
 }
 
 func (h *deliveryHealth) recordAt(generation uint64, success bool) {
@@ -59,6 +108,7 @@ func (h *deliveryHealth) recordAt(generation uint64, success bool) {
 		h.sent++
 		if generation == h.generation {
 			h.lastSuccess = time.Now().UTC()
+			h.lastError = ""
 		}
 	} else {
 		h.failed++
@@ -77,10 +127,9 @@ func (h *deliveryHealth) snapshot(enabled bool, depth, capacity int, dropped uin
 	} else if h.lastFailure.After(h.lastSuccess) {
 		state = "degraded"
 	} else if !h.lastSuccess.IsZero() {
+		// No age-based "stale" state: a healthy integration with nothing to
+		// send is not a fault. The last_success timestamp shows its age.
 		state = "last_send_succeeded"
-		if time.Since(h.lastSuccess) > 5*time.Minute {
-			state = "stale"
-		}
 	}
 	stamp := func(t time.Time) string {
 		if t.IsZero() {
@@ -88,7 +137,9 @@ func (h *deliveryHealth) snapshot(enabled bool, depth, capacity int, dropped uin
 		}
 		return t.Format(time.RFC3339Nano)
 	}
+	recentDrops := !h.lastDrop.IsZero() && time.Since(h.lastDrop) < recentDropWindow
 	return map[string]any{"enabled": enabled, "state": state, "sent": h.sent, "failures": h.failed,
-		"last_success": stamp(h.lastSuccess), "last_failure": stamp(h.lastFailure),
+		"last_success": stamp(h.lastSuccess), "last_failure": stamp(h.lastFailure), "last_error": h.lastError,
+		"last_drop": stamp(h.lastDrop), "recent_drops": recentDrops,
 		"queue_depth": depth, "queue_capacity": capacity, "dropped": dropped}
 }
diff --git a/go.mod b/go.mod
index b4c9ecb..fd2d65d 100644
--- a/go.mod
+++ b/go.mod
@@ -5,13 +5,13 @@ go 1.26.0
 toolchain go1.26.9
 
 require (
-	github.com/corazawaf/coraza/v3 v3.7.0
-	github.com/goccy/go-yaml v1.18.0
+	github.com/corazawaf/coraza/v3 v3.8.1
+	github.com/goccy/go-yaml v1.19.2
 	github.com/kaptinlin/jsonschema v0.4.6
 )
 
 require (
-	github.com/corazawaf/libinjection-go v0.3.2 // indirect
+	github.com/corazawaf/libinjection-go v0.3.3 // indirect
 	github.com/goccy/go-json v0.10.5 // indirect
 	github.com/gotnospirit/makeplural v0.0.0-20180622080156-a5f48d94d976 // indirect
 	github.com/gotnospirit/messageformat v0.0.0-20221001023931-dfe49f1eb092 // indirect
diff --git a/graphql_api8.go b/graphql_api8.go
index 95c8238..0f6440d 100644
--- a/graphql_api8.go
+++ b/graphql_api8.go
@@ -2132,6 +2132,7 @@ func (s *graphqlStore) wrap(site string, next http.Handler) http.Handler {
 		w.Header().Set("Content-Type", "application/json")
 		w.Header().Set("Cache-Control", "no-store")
 		w.Header().Set("X-Content-Type-Options", "nosniff")
+		markAttackBlock(r)
 		w.WriteHeader(status)
 		_, _ = w.Write([]byte(`{"errors":[{"message":"request rejected by GraphQL security policy"}]}`))
 	})
diff --git a/identity_api5.go b/identity_api5.go
index fbeca0b..61dfe3e 100644
--- a/identity_api5.go
+++ b/identity_api5.go
@@ -1316,10 +1316,15 @@ func identityAction(mode string) string {
 	return "detect"
 }
 
-func writeIdentityReject(w http.ResponseWriter, status int) {
+func writeIdentityReject(w http.ResponseWriter, r *http.Request, status int) {
 	if status != http.StatusUnauthorized && status != http.StatusForbidden && status != http.StatusServiceUnavailable {
 		status = http.StatusForbidden
 	}
+	// A policy rejection is security enforcement. 503 means the issuer or key
+	// set is unreachable: an operational fault, not attack evidence.
+	if status != http.StatusServiceUnavailable {
+		markAttackBlock(r)
+	}
 	w.Header().Set("Content-Type", "application/json")
 	w.Header().Set("Cache-Control", "no-store")
 	w.Header().Set("X-Content-Type-Options", "nosniff")
@@ -1356,7 +1361,7 @@ func (s *apiIdentityStore) wrap(site string, next http.Handler) http.Handler {
 			v := APIIdentityViolation{OperationID: opID, Mode: rp.Policy.Mode, Action: identityAction(rp.Policy.Mode), Type: vType, IssuerID: rp.Policy.IssuerID, Expected: "cryptographically verified Bearer JWT", Observed: observed}
 			s.recordViolation(v)
 			if rp.Policy.Mode == apiIdentityModeEnforce {
-				writeIdentityReject(w, rp.Policy.UnauthorizedCode)
+				writeIdentityReject(w, r, rp.Policy.UnauthorizedCode)
 				return
 			}
 			next.ServeHTTP(w, r)
@@ -1365,7 +1370,7 @@ func (s *apiIdentityStore) wrap(site string, next http.Handler) http.Handler {
 		if rp.Issuer == nil {
 			s.recordViolation(APIIdentityViolation{OperationID: opID, Mode: rp.Policy.Mode, Action: identityAction(rp.Policy.Mode), Type: "JWT_ISSUER_UNAVAILABLE", IssuerID: rp.Policy.IssuerID, Expected: "enabled trusted issuer runtime", Observed: "issuer configuration unavailable"})
 			if rp.Policy.Mode == apiIdentityModeEnforce {
-				writeIdentityReject(w, http.StatusServiceUnavailable)
+				writeIdentityReject(w, r, http.StatusServiceUnavailable)
 				return
 			}
 			next.ServeHTTP(w, r)
@@ -1383,7 +1388,7 @@ func (s *apiIdentityStore) wrap(site string, next http.Handler) http.Handler {
 				if code == "JWKS_UNAVAILABLE" {
 					status = http.StatusServiceUnavailable
 				}
-				writeIdentityReject(w, status)
+				writeIdentityReject(w, r, status)
 				return
 			}
 			next.ServeHTTP(w, r)
@@ -1394,7 +1399,7 @@ func (s *apiIdentityStore) wrap(site string, next http.Handler) http.Handler {
 		if typ, expected, observed := evaluateIdentityRequirements(rp.Policy, id); typ != "" {
 			s.recordViolation(APIIdentityViolation{OperationID: opID, Mode: rp.Policy.Mode, Action: identityAction(rp.Policy.Mode), Type: typ, IssuerID: rp.Policy.IssuerID, SubjectHash: identitySubjectHash(id), Expected: expected, Observed: observed})
 			if rp.Policy.Mode == apiIdentityModeEnforce {
-				writeIdentityReject(w, rp.Policy.ForbiddenCode)
+				writeIdentityReject(w, r, rp.Policy.ForbiddenCode)
 				return
 			}
 		}
diff --git a/internal/vectoraccel/engine.go b/internal/vectoraccel/engine.go
index ed55094..0684e6a 100644
--- a/internal/vectoraccel/engine.go
+++ b/internal/vectoraccel/engine.go
@@ -112,7 +112,7 @@ func NewManager(cfg Config) (*Manager, error) {
 func (m *Manager) NativeAvailable() bool { return m.native }
 func (m *Manager) NativeVersion() string { return m.version }
 
-const corazaLearningSemanticVersion = "coraza-v3.7.0-matchedrules-v1"
+const corazaLearningSemanticVersion = "coraza-v3.8.1-matchedrules-v1"
 
 func groupRuntimeFingerprint(ruleFingerprint, nativeVersion string) string {
 	h := sha256.Sum256([]byte(ruleFingerprint + "|" + nativeVersion + "|" + corazaLearningSemanticVersion))
diff --git a/l7_abuse.go b/l7_abuse.go
index f629788..6501cd7 100644
--- a/l7_abuse.go
+++ b/l7_abuse.go
@@ -6,6 +6,7 @@ import (
 	"net/http"
 	"net/netip"
 	"sync"
+	"sync/atomic"
 	"time"
 
 	"waf-proxy/internal/tlsfront"
@@ -58,6 +59,38 @@ type l7AbuseController struct {
 	tlsShards  [l7AbuseShardCount]l7AbuseShard
 	overflowMu sync.Mutex
 	overflows  map[abuseClientKey]*abuseClientState
+
+	// Saturation telemetry: while the client table is full, new identities
+	// share a per-site overflow allowance. Operators need to see that.
+	overflowAdmitted atomic.Uint64
+	overflowRefused  atomic.Uint64
+	lastOverflow     atomic.Int64 // unix nanoseconds
+}
+
+func (c *l7AbuseController) noteOverflow(allowed bool) {
+	c.lastOverflow.Store(time.Now().UnixNano())
+	if allowed {
+		c.overflowAdmitted.Add(1)
+	} else {
+		c.overflowRefused.Add(1)
+	}
+}
+
+// saturationStats reports overflow use. "active" means overflow was used in
+// the last ten minutes, i.e. new clients are currently sharing an allowance.
+func (c *l7AbuseController) saturationStats() map[string]any {
+	if c == nil || !c.cfg.Enabled {
+		return map[string]any{"enabled": false, "active": false}
+	}
+	last := c.lastOverflow.Load()
+	stamp := ""
+	if last != 0 {
+		stamp = time.Unix(0, last).UTC().Format(time.RFC3339)
+	}
+	return map[string]any{"enabled": true,
+		"active":            last != 0 && time.Since(time.Unix(0, last)) < 10*time.Minute,
+		"overflow_admitted": c.overflowAdmitted.Load(), "overflow_refused": c.overflowRefused.Load(),
+		"last_overflow": stamp}
 }
 
 func newL7AbuseController(cfg L7AbuseConfig) *l7AbuseController {
@@ -96,10 +129,16 @@ func validateL7AbuseConfig(cfg L7AbuseConfig, accel tlsfront.AccelerationConfig)
 // because no HTTP forwarding metadata exists before the TLS handshake. The
 // scope is the public listener rather than SNI so rotating hostnames cannot
 // evade the bucket.
-func (c *l7AbuseController) allowTLSHandshakeAt(scope, identity string, now time.Time) bool {
+func (c *l7AbuseController) allowTLSHandshakeAt(scope, identity string, now time.Time) (allowed bool) {
 	if c == nil || !c.cfg.Enabled || c.cfg.TLSHandshakePerWindow <= 0 {
 		return true
 	}
+	overflow := false
+	defer func() {
+		if overflow {
+			c.noteOverflow(allowed)
+		}
+	}()
 	key := abuseClientKey{site: scope, identity: c.limiterIdentity(identity)}
 	shard := &c.tlsShards[l7AbuseShardIndex(key)]
 	shard.mu.Lock()
@@ -121,6 +160,7 @@ func (c *l7AbuseController) allowTLSHandshakeAt(scope, identity string, now time
 			// Saturation must not turn the limiter off. Unknown identities share a
 			// bounded overflow bucket until an idle per-identity slot is available.
 			st = c.overflowState(scope, "tls")
+			overflow = true
 			if st == nil {
 				return false
 			}
@@ -179,7 +219,13 @@ func (c *l7AbuseController) wrap(site string, next http.Handler) http.Handler {
 	})
 }
 
-func (c *l7AbuseController) allowAt(site, identity string, now time.Time) (*abuseClientState, bool) {
+func (c *l7AbuseController) allowAt(site, identity string, now time.Time) (state *abuseClientState, allowed bool) {
+	overflow := false
+	defer func() {
+		if overflow {
+			c.noteOverflow(allowed)
+		}
+	}()
 	key := abuseClientKey{site: site, identity: c.limiterIdentity(identity)}
 	shard := &c.shards[l7AbuseShardIndex(key)]
 	shard.mu.Lock()
@@ -202,6 +248,7 @@ func (c *l7AbuseController) allowAt(site, identity string, now time.Time) (*abus
 			// Bound memory without creating an attacker-controlled fail-open bypass.
 			// One bucket per configured site, shared across shards, not sites.
 			st = c.overflowState(site, "http")
+			overflow = true
 			if st == nil {
 				return nil, false
 			}
diff --git a/main.go b/main.go
index 5bf3674..ba870c5 100644
--- a/main.go
+++ b/main.go
@@ -1693,8 +1693,12 @@ func (s *server) logWrap(siteName string, next http.Handler) http.Handler {
 		sw := &statusRecorder{ResponseWriter: w, code: 200}
 		r, signals := requestAttackSignals(r)
 		next.ServeHTTP(sw, r)
-		if signals.blocked {
+		// One observation per request: blocked requests count as blocks, other
+		// requests by their highest matched rule severity.
+		if blocked, severity := signals.snapshot(); blocked {
 			s.notify.observeAttack(siteName, "block", "", time.Now().UTC())
+		} else if severity != "" {
+			s.notify.observeAttack(siteName, "match", severity, time.Now().UTC())
 		}
 		// metrics: count the request, approximate in/out bytes
 		s.metrics.addReq()
@@ -1740,7 +1744,6 @@ func (s *server) buildWAF(policy PolicyConfig, pages []PagePolicy, mode, aiMode,
 				Data:       rule.Data(),
 			}
 			s.matches.add(rec)
-			s.notify.observeAttack(siteName, "match", rec.Severity, matchAt)
 			s.observeMatch(siteName, rec.URI, rec.RuleID, rec.Client, rec.Severity)
 			s.syslog.forwardMatch(rec)
 			if s.matchLogs != nil {
diff --git a/notify.go b/notify.go
index 6babab8..fdecc45 100644
--- a/notify.go
+++ b/notify.go
@@ -15,6 +15,7 @@ import (
 	"bytes"
 	"context"
 	"encoding/json"
+	"errors"
 	"fmt"
 	"log/slog"
 	"net/http"
@@ -30,6 +31,10 @@ type NotifyConfig struct {
 	ClearWebhook        bool   `json:"clear_webhook,omitempty"` // draft intent, consumed on Apply; never HA-shared
 	WebhookAllowHTTP    bool   `json:"webhook_allow_http,omitempty"`
 	WebhookAllowPrivate bool   `json:"webhook_allow_private,omitempty"`
+	// WebhookProxyURL is an optional egress proxy (http:// or https://, no
+	// credentials) for networks that reach chat or ticketing services only
+	// through a proxy. Environment proxy variables are never used.
+	WebhookProxyURL     string `json:"webhook_proxy_url,omitempty"`
 	OnAttack            bool   `json:"on_attack"`
 	AttackThreshold     int    `json:"attack_threshold,omitempty"`
 	AttackWindowSeconds int    `json:"attack_window_seconds,omitempty"`
@@ -115,6 +120,7 @@ type notifier struct {
 	webhookDropped uint64
 	delivery       deliveryHealth
 	attacks        map[string]*attackWindow
+	attackOn       atomic.Bool // fast path: skip the notifier lock when attack alerts are off
 
 	// dedupe map + optional external sink (e.g. syslog)
 	dedupe map[string]time.Time
@@ -143,6 +149,7 @@ func (n *notifier) configure(c NotifyConfig) {
 		n.delivery.invalidate()
 		n.attacks = make(map[string]*attackWindow)
 	}
+	n.attackOn.Store(c.OnAttack)
 	n.cfg = c
 }
 
@@ -227,7 +234,8 @@ func (n *notifier) pushContext(kind, level, title, body, dedupeKey string, actio
 		case n.webhookQ <- webhookJob{url: hook, kind: kindHook, item: item, policy: policy}:
 		default:
 			atomic.AddUint64(&n.webhookDropped, 1)
-			n.delivery.record(false)
+			n.delivery.fail(n.delivery.current(), errors.New("delivery queue full"))
+			n.delivery.noteDrop()
 			n.log.Warn("notification webhook queue full; dropping delivery")
 		}
 	}
@@ -240,6 +248,8 @@ func (n *notifier) webhookWorker() {
 		generation := n.delivery.current()
 		n.mu.Unlock()
 		if !current {
+			// Queued under settings that have since changed: discarded on purpose,
+			// counted, but not flagged against the new settings.
 			atomic.AddUint64(&n.webhookDropped, 1)
 			continue
 		}
@@ -260,13 +270,18 @@ func (n *notifier) webhookWorker() {
 	}
 }
 
-func (n *notifier) sendWebhook(url, kind string, item notification, policy NotifyConfig, versions ...uint64) (bool, error) {
-	success := false
+func (n *notifier) sendWebhook(url, kind string, item notification, policy NotifyConfig, versions ...uint64) (retry bool, err error) {
 	generation := n.delivery.current()
 	if len(versions) > 0 {
 		generation = versions[0]
 	}
-	defer func() { n.delivery.recordAt(generation, success) }()
+	defer func() {
+		if err == nil {
+			n.delivery.recordAt(generation, true)
+		} else {
+			n.delivery.fail(generation, err)
+		}
+	}()
 	if err := validateWebhookURL(url, policy); err != nil {
 		return false, err
 	}
@@ -297,7 +312,6 @@ func (n *notifier) sendWebhook(url, kind string, item notification, policy Notif
 	}
 	_ = resp.Body.Close()
 	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
-		success = true
 		return false, nil
 	}
 	err = fmt.Errorf("webhook HTTP %d", resp.StatusCode)
diff --git a/packaging/deb/tests/test-deb-packaging.sh b/packaging/deb/tests/test-deb-packaging.sh
index 9892d7a..802b40a 100755
--- a/packaging/deb/tests/test-deb-packaging.sh
+++ b/packaging/deb/tests/test-deb-packaging.sh
@@ -25,7 +25,7 @@ cat > "$BIN/BUILD_PROVENANCE.json" <<'EOF2'
   "artifact_type": "PORTABLE_BINARY",
   "build_variant": "portable-coraza",
   "go_version": "1.25.0",
-  "coraza_version": "v3.7.0",
+  "coraza_version": "v3.8.1",
   "vectorscan_version": "not-linked",
   "hsm_pkcs11_enabled": false,
   "binaries": []
diff --git a/packaging/qualification/build-lifecycle-fixtures.sh b/packaging/qualification/build-lifecycle-fixtures.sh
index 8d3dad4..0082feb 100755
--- a/packaging/qualification/build-lifecycle-fixtures.sh
+++ b/packaging/qualification/build-lifecycle-fixtures.sh
@@ -31,7 +31,7 @@ make_tree(){
   python3 - "$bin/BUILD_PROVENANCE.json" "$version" <<'PY'
 import json, pathlib, sys
 out,version=sys.argv[1:]
-obj={"schema_version":2,"version":version,"artifact_type":"PORTABLE_BINARY","build_variant":"portable-coraza","go_version":"1.25.0","coraza_version":"v3.7.0","vectorscan_version":"not-linked","hsm_pkcs11_enabled":False,"binaries":[]}
+obj={"schema_version":2,"version":version,"artifact_type":"PORTABLE_BINARY","build_variant":"portable-coraza","go_version":"1.25.0","coraza_version":"v3.8.1","vectorscan_version":"not-linked","hsm_pkcs11_enabled":False,"binaries":[]}
 pathlib.Path(out).write_text(json.dumps(obj,sort_keys=True)+"\n")
 PY
   if [ "$changed" = 1 ]; then
diff --git a/packaging/rpm/tests/test-rpm-packaging.sh b/packaging/rpm/tests/test-rpm-packaging.sh
index 3eb775c..455769a 100755
--- a/packaging/rpm/tests/test-rpm-packaging.sh
+++ b/packaging/rpm/tests/test-rpm-packaging.sh
@@ -23,7 +23,7 @@ cat > "$BIN/BUILD_PROVENANCE.json" <<'JSON'
   "artifact_type": "PORTABLE_BINARY",
   "build_variant": "portable-coraza",
   "go_version": "1.25.0",
-  "coraza_version": "v3.7.0",
+  "coraza_version": "v3.8.1",
   "vectorscan_version": "not-linked",
   "hsm_pkcs11_enabled": false,
   "binaries": []
diff --git a/positive_schema_api4.go b/positive_schema_api4.go
index 49080cd..b802cab 100644
--- a/positive_schema_api4.go
+++ b/positive_schema_api4.go
@@ -821,6 +821,7 @@ func (s *positiveSchemaStore) wrap(site string, next http.Handler) http.Handler
 		w.Header().Set("Content-Type", "application/json")
 		w.Header().Set("Cache-Control", "no-store")
 		w.Header().Set("X-Content-Type-Options", "nosniff")
+		markAttackBlock(r)
 		w.WriteHeader(status)
 		_, _ = w.Write([]byte(`{"error":"request rejected by API schema policy"}`))
 	})
diff --git a/qualify-release-host.sh b/qualify-release-host.sh
index 9d28e9e..dcd542a 100755
--- a/qualify-release-host.sh
+++ b/qualify-release-host.sh
@@ -73,10 +73,11 @@ if grep -Eq '^go 1\.26\.0$' go.mod; then
 else
   fail "go.mod no longer pins Go 1.26.0"
 fi
-if grep -Eq '^require[[:space:]]+github\.com/corazawaf/coraza/v3[[:space:]]+v3\.7\.0[[:space:]]*$' go.mod; then
-  pass "go.mod pins Coraza v3.7.0"
+# Accept both the single-line form and the usual require ( ... ) block form.
+if grep -Eq '^(require[[:space:]]+)?[[:space:]]*github\.com/corazawaf/coraza/v3[[:space:]]+v3\.8\.1[[:space:]]*$' go.mod; then
+  pass "go.mod pins Coraza v3.8.1"
 else
-  fail "go.mod no longer pins Coraza v3.7.0"
+  fail "go.mod no longer pins Coraza v3.8.1"
 fi
 
 if ! command -v pkg-config >/dev/null 2>&1; then
diff --git a/review_fixes_test.go b/review_fixes_test.go
new file mode 100644
index 0000000..cb4ac26
--- /dev/null
+++ b/review_fixes_test.go
@@ -0,0 +1,273 @@
+package main
+
+import (
+	"context"
+	"errors"
+	"fmt"
+	"io"
+	"log/slog"
+	"net"
+	"net/http"
+	"net/http/httptest"
+	"net/netip"
+	"net/url"
+	"strings"
+	"sync/atomic"
+	"testing"
+	"time"
+)
+
+func quietNotifier(cfg NotifyConfig) *notifier {
+	n := newNotifier(slog.New(slog.NewTextHandler(io.Discard, nil)))
+	n.configure(cfg)
+	return n
+}
+
+func attackAlerts(n *notifier) []notification {
+	var out []notification
+	for _, it := range n.list(1000) {
+		if it.Kind == notifyAttack {
+			out = append(out, it)
+		}
+	}
+	return out
+}
+
+func TestReviewRateLimitAndAllowlistResponsesDoNotRaiseAttackAlerts(t *testing.T) {
+	n := quietNotifier(NotifyConfig{OnAttack: true, AttackThreshold: 3, AttackWindowSeconds: 60})
+	s := &server{notify: n, metrics: newMetrics(), access: newAccessRing(16), syslog: newSyslogEngine(nil)}
+	l7 := newL7AbuseController(L7AbuseConfig{Enabled: true, RequestsPerWindow: 1, WindowSeconds: 60})
+	h := s.logWrap("shop", l7.wrap("shop", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})))
+	for i := 0; i < 10; i++ { // one ordinary client bursting: 1 allowed, 9 rate-limited
+		req := httptest.NewRequest(http.MethodGet, "http://shop/", nil)
+		req.RemoteAddr = "198.51.100.7:1000"
+		h.ServeHTTP(httptest.NewRecorder(), req)
+	}
+	if got := attackAlerts(n); len(got) != 0 {
+		t.Fatalf("rate limiting raised an attack alert: %+v", got)
+	}
+}
+
+func TestReviewAlertThresholdCountsRequestsNotRuleHits(t *testing.T) {
+	n := quietNotifier(NotifyConfig{OnAttack: true, AttackThreshold: 3, AttackWindowSeconds: 60, AttackMinSeverity: "CRITICAL"})
+	s := &server{notify: n, metrics: newMetrics(), access: newAccessRing(16), syslog: newSyslogEngine(nil)}
+	// Each request matches five critical rules, as a typical attack request does.
+	h := s.logWrap("shop", http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
+		signals := r.Context().Value(attackSignalKey{}).(*attackSignals)
+		for i := 0; i < 5; i++ {
+			signals.noteSeverity("CRITICAL")
+		}
+		signals.noteSeverity("WARNING")
+	}))
+	serve := func() { h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "http://shop/", nil)) }
+	serve()
+	serve()
+	if got := attackAlerts(n); len(got) != 0 {
+		t.Fatalf("2 requests (10 rule hits) crossed a 3-request threshold: %+v", got)
+	}
+	serve()
+	got := attackAlerts(n)
+	if len(got) != 1 || !strings.Contains(got[0].Body, "3 suspicious requests") {
+		t.Fatalf("third request should raise exactly one alert: %+v", got)
+	}
+}
+
+func TestReviewOneAlertPerIncidentWithHourlyFollowUp(t *testing.T) {
+	n := quietNotifier(NotifyConfig{OnAttack: true, AttackThreshold: 2, AttackWindowSeconds: 60})
+	start := time.Now()
+	n.observeAttack("shop", "block", "", start)
+	n.observeAttack("shop", "block", "", start)
+	if len(attackAlerts(n)) != 1 {
+		t.Fatal("first alert missing")
+	}
+	// A sustained attack: one request every 30s for 90 minutes, never quiet.
+	at := start
+	for i := 0; i < 180; i++ {
+		at = at.Add(30 * time.Second)
+		n.observeAttack("shop", "block", "", at)
+	}
+	alerts := attackAlerts(n)
+	if len(alerts) != 2 || alerts[0].Title != "WAF activity still ongoing" {
+		t.Fatalf("sustained attack should give the alert plus one hourly follow-up, got %d: %+v", len(alerts), alerts)
+	}
+	// Quiet for a full window ends the incident; the next burst is a new alert.
+	at = at.Add(2 * time.Minute)
+	n.observeAttack("shop", "block", "", at)
+	n.observeAttack("shop", "block", "", at)
+	if got := attackAlerts(n); len(got) != 3 || got[0].Title != "WAF activity threshold reached" {
+		t.Fatalf("new incident after a quiet window should alert again: %+v", got)
+	}
+}
+
+func TestReviewQuietIntegrationIsNotStaleAndDropsAgeOut(t *testing.T) {
+	var h deliveryHealth
+	h.record(true)
+	h.lastSuccess = time.Now().Add(-6 * time.Hour) // healthy, simply nothing to send
+	if got := h.snapshot(true, 0, 256, 0)["state"]; got != "last_send_succeeded" {
+		t.Fatalf("quiet healthy integration flagged: %v", got)
+	}
+	h.noteDrop()
+	if h.snapshot(true, 0, 256, 1)["recent_drops"] != true {
+		t.Fatal("a fresh drop should be flagged")
+	}
+	h.lastDrop = time.Now().Add(-2 * recentDropWindow)
+	if h.snapshot(true, 0, 256, 1)["recent_drops"] != false {
+		t.Fatal("an old drop should stop flagging after the recent-drop window")
+	}
+	h.noteDrop()
+	h.invalidate()
+	if h.snapshot(true, 0, 256, 1)["recent_drops"] != false {
+		t.Fatal("a configuration change should clear the drop flag")
+	}
+}
+
+func TestReviewFailureReasonIsRecordedWithoutURL(t *testing.T) {
+	var h deliveryHealth
+	secret := &url.Error{Op: "Post", URL: "https://hooks.example.test/services/T000/SECRET?token=abc", Err: errors.New("dial tcp 203.0.113.9:443: connect: connection refused")}
+	h.fail(h.current(), secret)
+	got := h.snapshot(true, 0, 1, 0)
+	reason, _ := got["last_error"].(string)
+	if !strings.Contains(reason, "connection refused") || strings.Contains(reason, "SECRET") || strings.Contains(reason, "token=") {
+		t.Fatalf("reason=%q", reason)
+	}
+	h.record(true)
+	if h.snapshot(true, 0, 1, 0)["last_error"] != "" {
+		t.Fatal("a successful send should clear the last error")
+	}
+}
+
+func TestReviewSyslogBacksOffWhileSIEMIsDown(t *testing.T) {
+	ln, err := net.Listen("tcp", "127.0.0.1:0")
+	if err != nil {
+		t.Fatal(err)
+	}
+	port := ln.Addr().(*net.TCPAddr).Port
+	ln.Close() // nothing listens: every dial is refused
+	s := newSyslogEngine(slog.New(slog.NewTextHandler(io.Discard, nil)))
+	cfg := defaultSyslogConfig()
+	cfg.Enabled, cfg.Protocol, cfg.Host, cfg.Port = true, "tcp", "127.0.0.1", port
+	s.storeConfig(cfg)
+	s.deliver("first")
+	s.mu.Lock()
+	failures, retryAt := s.dialFailures, s.retryAt
+	s.mu.Unlock()
+	if failures != 1 || !retryAt.After(time.Now()) {
+		t.Fatalf("first failure should start a backoff: failures=%d retryAt=%v", failures, retryAt)
+	}
+	for i := 0; i < 50; i++ {
+		s.deliver("during backoff")
+	}
+	s.mu.Lock()
+	failures = s.dialFailures
+	s.mu.Unlock()
+	stats := s.deliveryStats()
+	if failures != 1 {
+		t.Fatalf("lines during backoff must not re-dial: dial failures=%d", failures)
+	}
+	if stats["dropped"] != uint64(51) || stats["recent_drops"] != true || !strings.Contains(fmt.Sprint(stats["last_error"]), "refused") {
+		t.Fatalf("backoff drops must be counted and keep the real reason: %v", stats)
+	}
+	if stats["failures"] != uint64(1) {
+		t.Fatalf("backoff drops are not new delivery attempts: %v", stats["failures"])
+	}
+}
+
+func TestReviewUDPSyslogIsReportedUnconfirmed(t *testing.T) {
+	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
+	if err != nil {
+		t.Fatal(err)
+	}
+	defer pc.Close()
+	s := newSyslogEngine(slog.New(slog.NewTextHandler(io.Discard, nil)))
+	cfg := defaultSyslogConfig()
+	cfg.Enabled, cfg.Protocol, cfg.Host, cfg.Port = true, "udp", "127.0.0.1", pc.LocalAddr().(*net.UDPAddr).Port
+	s.storeConfig(cfg)
+	s.deliver("hello")
+	if got := s.deliveryStats()["state"]; got != "sent_unconfirmed" {
+		t.Fatalf("UDP state=%v", got)
+	}
+}
+
+func TestReviewWebhookProxyValidation(t *testing.T) {
+	for _, bad := range []string{"ftp://proxy:21", "http://user:pass@proxy:3128", "http://proxy:3128/path", "http://proxy:3128?x=1", "proxy:3128"} {
+		if validateWebhookProxyURL(bad) == nil {
+			t.Fatalf("accepted %q", bad)
+		}
+	}
+	for _, good := range []string{"", "http://proxy.internal:3128", "https://10.0.0.5:8443/"} {
+		if err := validateWebhookProxyURL(good); err != nil {
+			t.Fatalf("rejected %q: %v", good, err)
+		}
+	}
+}
+
+func TestReviewWebhookProxyIsUsedAndDestinationStillChecked(t *testing.T) {
+	var proxied atomic.Int32
+	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
+		// A forward proxy receives the absolute destination URL.
+		if r.URL.Host == "hooks.example.test" {
+			proxied.Add(1)
+		}
+		w.WriteHeader(http.StatusNoContent)
+	}))
+	defer proxy.Close()
+	public := func(context.Context, string, string) ([]netip.Addr, error) {
+		return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
+	}
+	internal := func(context.Context, string, string) ([]netip.Addr, error) {
+		return []netip.Addr{netip.MustParseAddr("10.1.2.3")}, nil
+	}
+	var dialer net.Dialer
+	policy := NotifyConfig{WebhookProxyURL: proxy.URL, WebhookAllowHTTP: true}
+	send := func(lookup func(context.Context, string, string) ([]netip.Addr, error)) error {
+		client, transport := webhookClientWithNetwork(policy, lookup, dialer.DialContext)
+		defer transport.CloseIdleConnections()
+		req, _ := http.NewRequest(http.MethodPost, "http://hooks.example.test/notify", strings.NewReader("{}"))
+		resp, err := client.Do(req)
+		if err == nil {
+			resp.Body.Close()
+		}
+		return err
+	}
+	if err := send(public); err != nil || proxied.Load() != 1 {
+		t.Fatalf("delivery through the proxy failed: err=%v proxied=%d", err, proxied.Load())
+	}
+	if err := send(internal); err == nil || proxied.Load() != 1 {
+		t.Fatalf("a destination resolving to a private address must be refused before the proxy: err=%v proxied=%d", err, proxied.Load())
+	}
+}
+
+func TestReviewRateLimiterSaturationIsVisible(t *testing.T) {
+	c := newL7AbuseController(L7AbuseConfig{Enabled: true, RequestsPerWindow: 1, WindowSeconds: 60})
+	if c.saturationStats()["active"] != false {
+		t.Fatal("fresh limiter reported saturation")
+	}
+	now := time.Now()
+	const target = uint64(0)
+	filled := 0
+	for n := 0; filled < l7AbuseMaxEntriesPerShard; n++ {
+		id := fmt.Sprintf("198.51.%d.%d", n/256, n%256)
+		if l7AbuseShardIndex(abuseClientKey{site: "shop", identity: c.limiterIdentity(id)}) != target {
+			continue
+		}
+		if st, ok := c.allowAt("shop", id, now); ok {
+			c.releaseState(st)
+		}
+		filled++
+	}
+	tried := 0
+	for n := 0; tried < 3; n++ {
+		id := fmt.Sprintf("203.0.%d.%d", n/256, n%256)
+		if l7AbuseShardIndex(abuseClientKey{site: "shop", identity: c.limiterIdentity(id)}) != target {
+			continue
+		}
+		if st, ok := c.allowAt("shop", id, now); ok {
+			c.releaseState(st)
+		}
+		tried++
+	}
+	stats := c.saturationStats()
+	if stats["active"] != true || stats["overflow_admitted"] != uint64(1) || stats["overflow_refused"] != uint64(2) {
+		t.Fatalf("saturation not visible: %v", stats)
+	}
+}
diff --git a/security_report_test.go b/security_report_test.go
index 8e58402..3a8c267 100644
--- a/security_report_test.go
+++ b/security_report_test.go
@@ -185,9 +185,20 @@ func TestSecurityBlockAlertsExcludeApplicationResponses(t *testing.T) {
 	if signals.blocked {
 		t.Fatal("application 403 misclassified")
 	}
+	// Rate limiting and IP allowlists are access controls, not attack evidence.
 	writeBlockResponse(httptest.NewRecorder(), r, 429, "l7_abuse_limit", true)
-	if !signals.blocked {
-		t.Fatal("explicit WAF block missed")
+	writeBlockResponse(httptest.NewRecorder(), r, 403, "cidr_policy_denied:office", false)
+	if blocked, _ := signals.snapshot(); blocked {
+		t.Fatal("rate limit or IP allowlist response counted as an attack block")
+	}
+	// An issuer/JWKS outage is operational, not attack evidence.
+	writeIdentityReject(httptest.NewRecorder(), r, http.StatusServiceUnavailable)
+	if blocked, _ := signals.snapshot(); blocked {
+		t.Fatal("identity issuer outage counted as an attack block")
+	}
+	writeIdentityReject(httptest.NewRecorder(), r, http.StatusForbidden)
+	if blocked, _ := signals.snapshot(); !blocked {
+		t.Fatal("API identity policy block missed")
 	}
 }
 
diff --git a/static/admin.html b/static/admin.html
index c6ebf61..6fa2a01 100644
--- a/static/admin.html
+++ b/static/admin.html
@@ -757,6 +757,7 @@
           <div class="eyebrow">Notifications</div>
           <div class="field"><label>Webhook URL (optional)</label><input type="text" id="nt_webhook_url" spellcheck="false" placeholder="https://hooks.slack.com/..."></div>
           <div class="field"><label>Webhook format</label><select id="nt_webhook_kind"><option value="slack">Slack / Teams text</option><option value="generic">Generic JSON</option></select></div>
+          <div class="field"><label for="nt_proxy_url">Outbound proxy (optional)</label><input type="text" id="nt_proxy_url" spellcheck="false" placeholder="http://proxy.internal:3128"><div class="hint">Use when this server reaches chat or ticketing services only through a proxy. No user name or password in the address. The webhook address is still checked before each send.</div></div>
           <div class="field check"><input type="checkbox" id="nt_allow_private"><label for="nt_allow_private">Allow private-network / loopback webhook destinations (explicit exception)</label></div>
           <div class="field check"><input type="checkbox" id="nt_allow_http"><label for="nt_allow_http">Allow unencrypted HTTP webhook delivery (insecure exception)</label></div>
           <div class="hint">HTTPS and public destinations are required by default. DNS answers are checked when connecting; redirects are refused. Link-local/metadata addresses remain forbidden. Existing internal or HTTP hooks need explicit exceptions before Apply.</div>
@@ -2619,6 +2620,7 @@
     notifyClearPending=!!nt.clear_webhook;notifyClearPrevious="";
     $("nt_webhook_url").value=nt.webhook_url||"";
     $("nt_webhook_kind").value=nt.webhook_kind||"slack";
+    $("nt_proxy_url").value=nt.webhook_proxy_url||"";
     $("nt_allow_private").checked=!!nt.webhook_allow_private;$("nt_allow_http").checked=!!nt.webhook_allow_http;
     $("nt_on_attack").checked=nt.on_attack!==false;$("nt_attack_threshold").value=nt.attack_threshold||10;$("nt_attack_window").value=nt.attack_window_seconds||60;$("nt_attack_severity").value=nt.attack_min_severity||"CRITICAL";
     $("nt_on_suggestion").checked=nt.on_suggestion!==false;
@@ -2631,7 +2633,7 @@
   function collectNotify(){
     return { webhook_url:$("nt_webhook_url").value.trim(), webhook_kind:$("nt_webhook_kind").value,
       ...(notifyClearPending?{clear_webhook:true}:{}),
-      webhook_allow_http:$("nt_allow_http").checked,webhook_allow_private:$("nt_allow_private").checked,on_attack:$("nt_on_attack").checked,
+      webhook_proxy_url:$("nt_proxy_url").value.trim(),webhook_allow_http:$("nt_allow_http").checked,webhook_allow_private:$("nt_allow_private").checked,on_attack:$("nt_on_attack").checked,
       attack_threshold:Number($("nt_attack_threshold").value)||10,attack_window_seconds:Number($("nt_attack_window").value)||60,attack_min_severity:$("nt_attack_severity").value,
       on_suggestion:$("nt_on_suggestion").checked, on_ai_block:$("nt_on_ai_block").checked,
       on_member_down:$("nt_on_member_down").checked, on_sync:$("nt_on_sync").checked, on_peer:$("nt_on_peer").checked };
@@ -2660,8 +2662,8 @@
   $("bellclear").onclick=async()=>{ await api("/api/notifications/dismiss",{method:"POST",body:JSON.stringify({all:true})}); loadBell(); };
   function deliveryText(label, stats){
     if(!stats||!stats.state)return label+": unavailable — delivery not verified";
-    const states={disabled:"Off",unverified:"Not verified",degraded:"Delivery failures",last_send_succeeded:"Last send succeeded (not a receipt)",stale:"Previous result is stale"};
-    return label+": "+(states[stats.state]||"Unknown")+" · last successful send "+(stats.last_success||"none")+" · last failure "+(stats.last_failure||"none")+" · failures "+Number(stats.failures||0)+" · dropped "+Number(stats.dropped||0)+" · pending "+Number(stats.queue_depth||0)+"/"+Number(stats.queue_capacity||0);
+    const states={disabled:"Off",unverified:"Not verified",degraded:"Delivery failures",last_send_succeeded:"Last send succeeded (not a receipt)",sent_unconfirmed:"Sent over UDP — delivery cannot be confirmed",stale:"Previous result is stale"};
+    return label+": "+(states[stats.state]||"Unknown")+(stats.last_error?" · last error: "+stats.last_error:"")+" · last successful send "+(stats.last_success||"none")+" · last failure "+(stats.last_failure||"none")+" · failures "+Number(stats.failures||0)+" · dropped "+Number(stats.dropped||0)+" · pending "+Number(stats.queue_depth||0)+"/"+Number(stats.queue_capacity||0);
   }
   async function loadDeliveryHealth(){
     try{const status=await api("/api/status");$("nt_delivery").textContent=deliveryText("Webhook",status.webhook_delivery);$("sl_delivery").textContent=deliveryText("SIEM",status.syslog_delivery);}
@@ -3608,7 +3610,8 @@
   }
   function attentionItems(st,poolRows,matches,access,candidates){
     const out=[],add=(text,view)=>out.push({text,view});
-    for(const [key,label] of [["syslog_delivery","SIEM"],["webhook_delivery","Webhook"]]){const state=st[key];if(state?.enabled&&(state.state==="degraded"||state.state==="stale"||Number(state.dropped)>0))add(label+" delivery needs review: "+state.state+"; "+Number(state.failures||0)+" failures, "+Number(state.dropped||0)+" dropped.","setup");}
+    for(const [key,label] of [["syslog_delivery","SIEM"],["webhook_delivery","Webhook"]]){const state=st[key];if(state?.enabled&&(state.state==="degraded"||state.recent_drops))add(label+" delivery needs review"+(state.last_error?": "+state.last_error:"")+" ("+Number(state.failures||0)+" failures, "+Number(state.dropped||0)+" dropped in total).","setup");}
+    const sat=st.l7_saturation;if(sat?.active)add("Rate limiter is at capacity: new clients share a small per-site allowance ("+Number(sat.overflow_refused||0)+" refused). This usually means a distributed flood; check Logs and the limits under Policies.","policies");
     for(const p of poolRows)for(const m of p.members||[])if(!m.healthy)add("Application server unavailable: "+m.node+" in "+p.name,"pools");
     const failures=access.filter(r=>r.status>=500).length,rejections=access.filter(r=>r.status===403||r.status===429).length;
     if(failures)add(failures+" server-error responses in the latest "+access.length+" stored requests.","logs");
diff --git a/syslog.go b/syslog.go
index dbfdab4..fd68403 100644
--- a/syslog.go
+++ b/syslog.go
@@ -16,6 +16,7 @@ package main
 
 import (
 	"crypto/tls"
+	"errors"
 	"fmt"
 	"io"
 	"log/slog"
@@ -112,6 +113,11 @@ type syslogEngine struct {
 	cancel   chan struct{}
 	dropped  uint64
 	delivery deliveryHealth
+
+	// Reconnect backoff while the SIEM is unreachable (guarded by mu): lines
+	// are dropped without dialling until retryAt, and the outage is logged once.
+	dialFailures int
+	retryAt      time.Time
 }
 
 func newSyslogEngine(log *slog.Logger) *syslogEngine {
@@ -147,6 +153,7 @@ func (s *syslogEngine) configure(c SyslogConfig) {
 	// Publish the immutable snapshot while the connection is serialized, then
 	// force a reconnect so the next background delivery uses the new endpoint.
 	s.storeConfig(c)
+	s.dialFailures, s.retryAt = 0, time.Time{}
 	if s.conn != nil {
 		s.conn.Close()
 		s.conn = nil
@@ -171,7 +178,8 @@ func (s *syslogEngine) enqueue(line string) {
 	case s.queue <- line:
 	default:
 		atomic.AddUint64(&s.dropped, 1)
-		s.delivery.record(false)
+		s.delivery.fail(s.delivery.current(), errors.New("delivery queue full"))
+		s.delivery.noteDrop()
 	}
 }
 
@@ -212,17 +220,48 @@ func (s *syslogEngine) deliver(line string) {
 		atomic.AddUint64(&s.dropped, 1)
 		return
 	}
-	success := false
+	var failure error
+	backingOff := false
 	defer func() {
-		s.delivery.recordAt(generation, success)
-		if !success {
+		switch {
+		case backingOff:
+			// Dropped without an attempt: counted, but neither a success nor a
+			// new failure, so the last real failure reason stays visible.
+			atomic.AddUint64(&s.dropped, 1)
+			s.delivery.noteDrop()
+		case failure == nil:
+			s.delivery.recordAt(generation, true)
+		default:
+			s.delivery.fail(generation, failure)
+			s.delivery.noteDrop()
 			atomic.AddUint64(&s.dropped, 1)
 		}
 	}()
 	if conn == nil {
+		s.mu.Lock()
+		waiting := time.Now().Before(s.retryAt)
+		s.mu.Unlock()
+		if waiting {
+			// Still backing off: drop without dialling (fail-open) and without
+			// logging every line.
+			backingOff = true
+			return
+		}
 		c, err := s.dial(cfg)
 		if err != nil {
-			s.log.Warn("syslog dial failed", "err", err) // fail-open: drop this line
+			failure = err
+			s.mu.Lock()
+			s.dialFailures++
+			delay := time.Second << min(s.dialFailures-1, 6)
+			if delay > time.Minute {
+				delay = time.Minute
+			}
+			s.retryAt = time.Now().Add(delay)
+			first := s.dialFailures == 1
+			s.mu.Unlock()
+			if first {
+				s.log.Warn("syslog dial failed; backing off and dropping lines until the SIEM is reachable", "err", err)
+			}
 			return
 		}
 		s.mu.Lock()
@@ -231,6 +270,10 @@ func (s *syslogEngine) deliver(line string) {
 			c.Close()
 			return
 		}
+		if s.dialFailures > 0 {
+			s.log.Info("syslog connection restored", "failed_attempts", s.dialFailures)
+		}
+		s.dialFailures, s.retryAt = 0, time.Time{}
 		s.conn = c
 		conn = c
 		s.mu.Unlock()
@@ -247,6 +290,7 @@ func (s *syslogEngine) deliver(line string) {
 		err = io.ErrShortWrite
 	}
 	if err != nil {
+		failure = err
 		s.log.Warn("syslog write failed; will reconnect", "err", err)
 		s.mu.Lock()
 		if s.conn != nil {
@@ -256,7 +300,6 @@ func (s *syslogEngine) deliver(line string) {
 		s.mu.Unlock()
 		return
 	}
-	success = true
 }
 
 func (s *syslogEngine) deliveryStats() map[string]any {
@@ -267,6 +310,10 @@ func (s *syslogEngine) deliveryStats() map[string]any {
 	defer s.mu.Unlock()
 	stats := s.delivery.snapshot(s.snapshotCfg().Enabled, len(s.queue), cap(s.queue), atomic.LoadUint64(&s.dropped))
 	stats["receipt_verified"] = false
+	// UDP has no transport acknowledgement: a send never proves delivery.
+	if s.snapshotCfg().Protocol == "udp" && stats["state"] == "last_send_succeeded" {
+		stats["state"] = "sent_unconfirmed"
+	}
 	stats["config_generation"] = s.delivery.current()
 	stats["config_token"] = s.delivery.configToken()
 	cfg := s.snapshotCfg()
@@ -450,10 +497,17 @@ func (s *syslogEngine) testForTarget(expected *uint64, token string) error {
 	if !cfg.Enabled {
 		return fmt.Errorf("syslog is disabled — enable and Apply first")
 	}
-	success := false
-	defer func() { s.delivery.recordAt(generation, success) }()
+	var failure error
+	defer func() {
+		if failure == nil {
+			s.delivery.recordAt(generation, true)
+		} else {
+			s.delivery.fail(generation, failure)
+		}
+	}()
 	conn, err := s.dial(cfg)
 	if err != nil {
+		failure = err
 		return err
 	}
 	defer conn.Close()
@@ -476,6 +530,6 @@ func (s *syslogEngine) testForTarget(expected *uint64, token string) error {
 	if err == nil && n != len(line) {
 		err = io.ErrShortWrite
 	}
-	success = err == nil
+	failure = err
 	return err
 }
diff --git a/tools/tests/test-api3-source.py b/tools/tests/test-api3-source.py
index 3b9e547..3dcb5e6 100644
--- a/tools/tests/test-api3-source.py
+++ b/tools/tests/test-api3-source.py
@@ -78,9 +78,12 @@ require('matchedSites' in api3 and 'UNDECLARED_ENDPOINT' in api3, 'undeclared-en
 require('authObservationFamilies' in api3 and 'anonymous' in api3, 'OpenAPI auth OR/AND/anonymous semantics hardening missing')
 require('RequestMediaTypes' in api3 and 'observedContentTypeMatches' in api3, 'multi-content-type contract drift support missing')
 
-require('github.com/goccy/go-yaml v1.18.0' in gomod, 'OpenAPI YAML parser dependency missing from go.mod')
+# Any version: the parser must stay a direct dependency (checked below), but an
+# exact pin here would block security upgrades pulled in by Coraza.
+require(re.search(r'(?m)^\s*github\.com/goccy/go-yaml v1\.\d+\.\d+\s*$', gomod) is not None, 'OpenAPI YAML parser dependency missing from go.mod')
 # Direct dependency: it must appear before the second require block and must not be marked indirect.
-yaml_line = next((line for line in gomod.splitlines() if 'github.com/goccy/go-yaml v1.18.0' in line), '')
+yaml_line = next((line for line in gomod.splitlines() if 'github.com/goccy/go-yaml v' in line), '')
+require(yaml_line != '', 'goccy/go-yaml requirement line not found')
 require('// indirect' not in yaml_line, 'goccy/go-yaml is used directly but still marked indirect')
 
 # Security boundary: API-3 source must not invoke known blocking/enforcement entry points.
diff --git a/tools/tests/test_waf_package_builder.py b/tools/tests/test_waf_package_builder.py
index ff64c45..c4cef83 100644
--- a/tools/tests/test_waf_package_builder.py
+++ b/tools/tests/test_waf_package_builder.py
@@ -27,7 +27,7 @@ class BuilderTests(unittest.TestCase):
     def test_project_pins(self):
         go, coraza = mod.read_project_requirements(ROOT)
         self.assertEqual(go, "1.26.0")
-        self.assertEqual(coraza, "v3.7.0")
+        self.assertEqual(coraza, "v3.8.1")
 
     def test_fingerprint_is_stable_and_ignores_dist(self):
         with tempfile.TemporaryDirectory() as td:
diff --git a/tools/waf_package_builder.py b/tools/waf_package_builder.py
index 43287ba..ed0dbd6 100644
--- a/tools/waf_package_builder.py
+++ b/tools/waf_package_builder.py
@@ -306,7 +306,7 @@ def load_provenance(root: Path) -> dict:
     if not p.is_file():
         raise BuildError("canonical build did not produce BUILD_PROVENANCE.json")
     obj = json.loads(p.read_text(encoding="utf-8"))
-    if obj.get("coraza_version") != "v3.7.0":
+    if obj.get("coraza_version") != "v3.8.1":
         raise BuildError(f"unexpected built Coraza version: {obj.get('coraza_version')!r}")
     return obj
 
diff --git a/webhook_policy.go b/webhook_policy.go
index b12af82..f8e75f0 100644
--- a/webhook_policy.go
+++ b/webhook_policy.go
@@ -33,6 +33,55 @@ func validateWebhookURL(raw string, policy NotifyConfig) error {
 	return nil
 }
 
+// validateWebhookProxyURL accepts an optional http(s) egress proxy address.
+// Credentials are refused so the value is never a secret.
+func validateWebhookProxyURL(raw string) error {
+	if raw == "" {
+		return nil
+	}
+	u, err := url.Parse(raw)
+	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil ||
+		(u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
+		return errors.New("notify.webhook_proxy_url must be http:// or https:// host[:port] without credentials or path")
+	}
+	return nil
+}
+
+// checkWebhookDestination applies the destination address policy by resolving
+// the webhook host. It is used when a proxy dials on our behalf; the proxy
+// resolves the name again, so this narrows but cannot fully close that gap.
+func checkWebhookDestination(ctx context.Context, host string, policy NotifyConfig, lookup func(context.Context, string, string) ([]netip.Addr, error)) error {
+	if ip, err := netip.ParseAddr(host); err == nil {
+		if !webhookIPAllowed(ip, policy.WebhookAllowPrivate) {
+			return errors.New("webhook destination is not allowed")
+		}
+		return nil
+	}
+	ips, err := lookup(ctx, "ip", host)
+	if err != nil || len(ips) == 0 {
+		return errors.New("webhook DNS resolution failed")
+	}
+	for _, ip := range ips {
+		if !webhookIPAllowed(ip, policy.WebhookAllowPrivate) {
+			return errors.New("webhook DNS resolved to a forbidden address")
+		}
+	}
+	return nil
+}
+
+type webhookDestinationGuard struct {
+	next   http.RoundTripper
+	policy NotifyConfig
+	lookup func(context.Context, string, string) ([]netip.Addr, error)
+}
+
+func (g webhookDestinationGuard) RoundTrip(req *http.Request) (*http.Response, error) {
+	if err := checkWebhookDestination(req.Context(), req.URL.Hostname(), g.policy, g.lookup); err != nil {
+		return nil, err
+	}
+	return g.next.RoundTrip(req)
+}
+
 func webhookIPAllowed(ip netip.Addr, allowPrivate bool) bool {
 	ip = ip.Unmap()
 	if !ip.IsValid() || ip.Zone() != "" || !ip.IsGlobalUnicast() && !ip.IsLoopback() {
@@ -87,7 +136,15 @@ func webhookClientWithNetwork(policy NotifyConfig, lookup func(context.Context,
 		}
 		return nil, errors.New("webhook connection failed")
 	}
-	client := &http.Client{Timeout: 6 * time.Second, Transport: transport,
+	var roundTripper http.RoundTripper = transport
+	if proxy, err := url.Parse(policy.WebhookProxyURL); policy.WebhookProxyURL != "" && err == nil {
+		// Through a proxy we dial only the operator-chosen proxy (which may be
+		// on a private network) and check the destination before each send.
+		transport.Proxy = http.ProxyURL(proxy)
+		transport.DialContext = dial
+		roundTripper = webhookDestinationGuard{next: transport, policy: policy, lookup: lookup}
+	}
+	client := &http.Client{Timeout: 6 * time.Second, Transport: roundTripper,
 		// Never forward alert bodies or credentials to a redirected destination.
 		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
 	}
