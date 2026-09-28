# Production Correctness & Control-Plane Hardening

Status: **IMPLEMENTED_TESTING_DEFERRED**  
Date: 2026-09-25  
Parent: `waf-proxy-api8-post-audit-hardening-2026-09-24.zip` (`e838ce6bd4f9cca6176cb809eb189978f312b194ae4b7c2f6f13f892a2d85f98`)

This hardening wave closes production correctness and control-plane findings discovered during the post-API-8 full-repository adversarial audit. It does not add API-9 and does not change API-6/API-7 non-enforcement authority.

## Implemented

- Block responses use `html/template`; request correlation IDs accept only a bounded safe character set. HTML block responses add no-store, nosniff, CSP and frame-deny headers.
- Admin JSON responses use no-store/nosniff controls. Notification webhook URLs remain write-only and are redacted from config reads; a separate operator-only clear action is provided.
- Debug evidence masks nested remote/resolved client IPs.
- Login abuse protection is bounded and fail-closed under key saturation; session creation fails closed on CSPRNG error; PBKDF2 parameters are bounded before work begins.
- Mutating admin routes are role-gated and audited. Admin audit records are durably appended to a mode-0600 JSONL log with bounded rotation.
- HA config replication uses a dedicated peer-only HTTPS endpoint and stable replication token. Generic config APIs no longer trust caller-supplied sync headers. Shared config strips node-local HA identity, users and secrets; the receiver restores its own local state. Outbound sync is bounded to one in-flight request plus a latest-wins pending slot.
- Full config mutation is serialized. Config is staged with a same-directory unique temp file, mode 0600 and file fsync; live network/runtime application is checked before atomic rename, and pre-rename commit failure rolls live state back. Directory fsync is performed after rename.
- Listener reconciliation synchronously binds required sockets and rolls back failed protocol transitions. External TLS frontend -> Go TLS waits for frontend stop before public bind. Go TLS -> external frontend now synchronously releases the public listening socket before frontend activation, removing the EADDRINUSE ownership race.
- Sitemap clear persists the cleared snapshot before dependent learning state is discarded and restores the previous snapshot on persistence failure.
- Early API-security stores reject future/unknown durable state versions.
- Notification webhook delivery uses a bounded queue and workers, bounded retries, non-2xx checking and queue-drop telemetry.
- L7 limiter saturation uses a bounded conservative overflow bucket rather than allowing unseen identities.
- Admin file browsing resolves symlinks before enforcing the browse-root boundary.
- In-process updater writes are disabled by default for packaged/systemd deployment. A standalone writable install requires explicit `WAF_UPDATE_INSTALL_DIR`; packaged installations use the OS package manager.
- systemd units explicitly declare device access for hardware watchdog and QAT-related device nodes instead of combining those capabilities with `PrivateDevices=true`.
- Console actions are capability-aware and `/api/learn/clear` is exposed to reviewers. OpenAPI import and webhook-secret mutation remain operator-only.
- Main config sample/documentation includes the post-audit CIDR/L7 controls.

## Validation truth

Static/source gates and artifact gates can validate source structure, security contracts, syntax, packaging and clean extraction. Canonical Go 1.25 compile/test/race remains **BLOCKED_ENVIRONMENT / NOT_RUN** on this host because the installed Go is 1.23.2, `go.mod` requires 1.25.0, and external toolchain retrieval is unavailable. This baseline must not be promoted to TESTED or RELEASED on source-gate evidence alone.

The supplied source archive has no `.git` metadata. Therefore branch creation, remote/main synchronization, commit identity and fresh-clone verification cannot be performed from this artifact. Parent ZIP SHA-256 and package manifests are the source identity for this handoff.
