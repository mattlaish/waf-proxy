# Development Engineering Ledger

The detailed feature/status roadmap remains in `DEVELOPMENT_ROADMAP.md`; `AI_HANDOFF.md` remains the continuation ledger. This file records cross-cutting engineering rules that apply to every development slice.

## Artifact Packaging Integrity Gate

Every development stage that produces a delivery ZIP, installer, package, or deployment bundle must treat the generated artifact as part of the production surface.

After source tests and before release, the artifact must be extracted into a clean location and independently inspected. At minimum validate archive integrity/path safety, required files, critical-file size, executable/script headers, syntax, source-to-package SHA-256 equality, Unix file modes, package manifest contents, and feasible artifact-level smoke checks.

Use `build-release-artifact.sh` for new WAF source ZIPs and `verify-release-artifact.sh` for independent post-build verification. The complete policy and mini-SIEM motivating incident are documented in `RELEASE_PROCESS.md`.

A source-tree PASS is not evidence that a delivery artifact contains the tested source. Artifact verification is a mandatory release gate, separate from real Coraza/VectorScan target-host qualification.

## 2026-09-10 Debug Evidence / VectorScan Qualification Slice

Implemented source baseline additions:

- Added bounded Debug Evidence Capture foundation.
- Capture is independent from the WAF verdict path; capture failure cannot change Coraza decisions.
- Added bounded transaction evidence export foundation.
- Added qualification test coverage for evidence lifecycle.
- Added Phase 1 qualification tracking for VectorScan candidate coverage, false-negative accounting, FAILSAFE transitions, fingerprint invalidation, restart persistence and HTTP/2 transaction correlation requirements.

Real Coraza/libvectorscan execution remains subject to release-host qualification gates and is not promoted from local/stub evidence.

## 2026-09-10 Debug Evidence integration / Phase 1 safety gate

Implemented:
- Coraza transaction-final MatchedRules evidence capture hook after ProcessLogging.
- VectorScan candidate-vs-Coraza qualification helper with zero false-negative gate.
- Debug evidence masking foundation and bounded export support.

Truth boundary:
- Real Coraza execution remains NOT_RUN until qualified host execution.
- Real libvectorscan hs_compile_multi/hs_scan remains NOT_RUN.

## 2026-09-10 Debug Evidence Supportability Slice

Implemented:
- Debug bundle evidence schema foundation for request/response/TLS/proxy/Coraza evidence.
- Tenant-scoped evidence storage foundation with TTL cleanup.
- VectorScan differential qualification helper enforcing Coraza match coverage.

Truth boundary:
- Real Coraza v3.7 execution: NOT_RUN.
- Real libvectorscan execution: NOT_RUN.
- Production CRS Learning qualification: NOT_RUN.


## 2026-09-11 Supportability Slice

Added wafctl debug export/doctor/support bundle foundation and Phase 1 differential qualification helper. Real Coraza/libvectorscan execution remains NOT_RUN.


## Stage 2 Supportability Implementation
- Added wafctl supportability command foundation.
- Added qualification phase1 differential runner foundation.
- Real Go 1.25/Coraza/libvectorscan qualification remains NOT_RUN.


## 2026-09-11 Phase 2 — VectorScan exact-semantics expansion

Implemented an owner-directed conservative coverage slice without changing Coraza authority or FAILSAFE behavior. The rule classifier now stores an ordered transform pipeline rather than a lowercase boolean. Only transforms whose Coraza v3.7.0 behavior can be reproduced locally without transaction/parser state are allowed. The group semantic key includes the transform sequence, and the semantic adapter version was bumped so stale persisted Learning/ACCELERATED state cannot survive this semantic change.

Newly implemented exact transforms: `uppercase`, `trim`, `trimLeft`, `trimRight`, `removeNulls`, `replaceNulls`, `compressWhitespace`, `removeWhitespace`, and `length`, in addition to existing `lowercase`. `QUERY_STRING` and `SERVER_NAME` are added as selected single-value sources. Fixed-header reconstruction now treats `Host` and `Transfer-Encoding` specially to mirror Coraza's Go HTTP connector.

Rejected/deferred in this slice: URL decoding, path normalization, HTML/JS/CSS decoding, ARGS/body variables, chains, negated operators, multi-variable selectors, dynamic macros, and any transform requiring parser or mutable transaction state. These remain Coraza-only.

Validation truth: isolated `internal/vectoraccel` unit/vet/race passes on the available Go 1.23.2 host; the full repository remains blocked by the Go 1.25 module requirement, and the new real-Coraza parity gate is NOT_RUN.

## 2026-09-11 Phase 2 — VectorScan coverage increment 2

Continued the owner-directed conservative Phase 2 expansion without changing Coraza authority, Learning promotion, verification sampling, or FAILSAFE behavior.

Implemented additional exact request/connection sources: `REQUEST_URI_RAW`, `REQUEST_LINE`, `REQUEST_BASENAME`, `REMOTE_ADDR`, and `REMOTE_PORT`. The request-line/raw-URI values intentionally mirror the current Coraza Go HTTP connector input (`req.URL.String()`), basename mirrors Coraza's last `/` or `\\` separator handling over the parsed path, and remote address/port mirrors the connector's final-colon split of `req.RemoteAddr`. Because the existing trusted-proxy resolver runs outside both the VectorScan prefilter and Coraza wrapper, both see the same normalized `RemoteAddr`.

Added deterministic exact transforms `base64Encode` and `hexEncode`, matching Coraza v3.7.0 standard-library encoding behavior. Explicitly did **not** add `base64Decode`, `hexDecode`, `md5`, `sha1`, URL decoders, path normalization, HTML/JS/CSS decoders, ARGS/body collections, chains, negation, or aggregate selectors.

The semantic adapter version was bumped again so previously persisted group fingerprints cannot be reused across the expanded source/transform semantics; affected groups must re-enter Learning. Portable isolated `internal/vectoraccel` unit, focused, vet, and race checks pass on local Go 1.23.2. Full-repository and `realcoraza` execution remain blocked/NOT_RUN because the package requires Go 1.25 and the host still lacks the real release dependency stack.

## 2026-09-11 — Phase 3 release engineering / supply-chain hardening

Implemented release-evidence generation and verification without changing the request dataplane. Added SPDX 2.3 and CycloneDX 1.5 SBOM generation from the pinned module graph, release/version/provenance evidence, explicit portable/native release flavoring, a truth-bearing govulncheck gate, SOURCE_DATE_EPOCH-based reproducible source artifacts, and optional detached OpenSSL signing that requires an externally managed key.

Security/truth decisions: native flavor fails closed without verified `pkg-config libhs`; `--require-govulncheck` fails closed unless govulncheck completes successfully; signing returns NOT_RUN/exit 3 without `WAF_RELEASE_SIGNING_KEY`; no signing key is generated or stored by the project. Artifact evidence is generated metadata and is separately validated rather than treated as source-tree content.

Current host evidence: Go 1.23.2, OpenSSL 3.5.5, NGINX 1.26.3, govulncheck unavailable, libhs unavailable, signing key not configured. Portable Phase 3 tooling and deterministic ZIP checks pass; native/govulncheck/signing production gates remain open. Phase 0/1/2 real WAF qualification remains unchanged.

## 2026-09-11 — Phase 4 security/product backlog implementation

Owner direction moved Phase 4 ahead of the remaining real-host VectorScan qualification gates. The implementation is additive and does not relax the Coraza/VectorScan truth boundary.

Implemented `security.go` with opt-in per-normalized-IP request rate/in-flight controls, direct-peer connection/TLS-handshake caps, bounded limiter identity state, expiring CIDR allow/deny with allow precedence, WAF-generated request IDs and bounded custom block templates. `coraza_observer.go` now injects the WAF request ID as the Coraza transaction ID when no explicit ID was supplied, allowing `MatchedRule.TransactionID()` to be the exact match/access correlation key. Access and WAF-match syslog records carry the same ID.

Implemented `security_state.go` as an atomic local persistence layer for AI blocks, learner aggregates, notifications, hashed sessions, audit history and security counters. Session map keys were changed to SHA-256 hashes so restart persistence never requires storing reusable bearer tokens. `waf-proxy.service` now uses `StateDirectory=waf-proxy`; install/doctor flows provision `/var/lib/waf-proxy`.

Extended the existing static-CRL PKI path instead of replacing it. `crl_urls` now supports SSRF-hardened HTTPS retrieval, DNS-answer validation and pinned dialing, bounded fetches, refresh scheduling/deduplication, last-known-good memory retention, restart cache, per-pool status, manual operator refresh, audit and existing hard/soft verification semantics. Full config validation intentionally performs no network I/O; outbound fetch validation happens during runtime construction before the new config is published.

Executed isolated security, PKI and persistence race suites PASS on the local Go 1.23.2 toolchain. Project-wide Go 1.25 test/vet and real Coraza request-ID regression remain blocked by the packaging host. External CRL positive/deployed Linux testing remains NOT_RUN. See `PHASE4_SECURITY_PRODUCT_REPORT_2026-09-11.md`.

## 2026-09-11 — Release blocker hotfix: module manifest + executable Git modes

Corrected two repository-level breakages found on `main`. `go.mod` had been manually advanced to Coraza v3.7.0 without the full tidied indirect graph, so a clean `go build ./...` requested `go mod tidy`. The indirect block is now synchronized to the Coraza v3.7.0 consumer graph, including `jsonschema`, `go-i18n`, `go-json`, `go-yaml`, the message-format helpers, `x/text`, and the current Mage/Aho-Corasick revisions; `go.sum` carries the corresponding checksums.

The release-script contract was also corrected at the Git metadata boundary: every top-level `*.sh` plus `benchmark/build.sh` must be committed executable (`100755`), not merely packaged with an executable ZIP mode. This directly fixes `TestReleaseScriptsAreLFAndBashSyntaxClean` on a normal Git checkout. No dataplane, policy, Coraza, VectorScan, PKI, or Phase 4 runtime behavior changed.
