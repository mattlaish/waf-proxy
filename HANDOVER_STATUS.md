# New-Chat Handover Status

Checkpoint: 2026-09-07 (Asia/Taipei)

## One-line state

The WAF implementation line is **Go 1.25 / Coraza v3.7.0 + optional VectorScan Learning Accelerator + optional NGINX/OpenSSL TLS acceleration**. Portable/stub and native-ABI validation remain complete; Phase 0 now has a strict `qualify-release-host.sh` preflight/core runner and a native semantic compile/scan test, but real Coraza v3.7 / real libvectorscan production qualification is still outstanding because this packaging host has Go 1.23.2 and no real libhs.

## Selected architecture

- Coraza/CRS remains authoritative.
- VectorScan is a candidate-prefilter/skip accelerator for conservatively compatible regex groups.
- Learning Period and continuous verification are permanent product behavior, not temporary development scaffolding.
- Zero observed VectorScan false negatives is the safety objective; mismatch/native scan error -> FAILSAFE -> Coraza-only.
- XDP is deferred and independent.
- TLS acceleration is independent and already implemented as optional NGINX/OpenSSL frontend with fallback-safe kTLS/QAT capability handling.

## What the next chat should not redo

- hot-path ring/config/body work;
- ReverseProxy BufferPool;
- load-balancer zero-allocation work;
- P0-C observation async plane;
- P0-D match-log async plane;
- P1 response-body inspection and backend transport tuning;
- P2 release-script/AI/statusRecorder hardening;
- benchmark harness;
- TLS frontend/modern NGINX implementation;
- VectorScan basic classifier/manager/state machine/native adapter/Coraza transaction observer.

## What blocks production VectorScan enablement

Real release-host execution of Go 1.25 + Coraza v3.7.0 + real libvectorscan/CRS. Run `./qualify-release-host.sh --preflight` then `--core`; this packaging host returns BLOCKED because it has Go 1.23.2 and no real libhs. See `DEVELOPMENT_ROADMAP.md` Phase 0 and `TESTING_RESULTS.md` for the exact gates and evidence boundary.

## Artifacts

Previous final implementation source:

- `waf-proxy-vectorscan-learning-coraza37-2026-09-04.zip`
- SHA-256 `64dc3034865d20aedaa38e10af5459da844ae6f6d92960e224223ec0ed359b22`

Previous implementation patch:

- `waf-proxy-vectorscan-learning-coraza37-2026-09-04.patch`
- SHA-256 `a24645a911caaa986c310509103093ba0cdc5e7bdcd4801251ccadd4e17a8398`

The 2026-09-07 continuation slice adds release-host qualification automation/tests and deterministic build preflight only; it does not alter Coraza/VectorScan authority, eligibility, Learning/FAILSAFE semantics, request processing, or TLS architecture.

## 2026-09-13 update

The source line now includes complete operator-side debug/supportability tooling (`wafctl` doctor/debug/support), transaction-correlated Debug Evidence wiring, and a real Phase 1 native VectorScan-vs-Coraza differential corpus runner with a hard zero-FN gate. Real qualification is still blocked on this packaging host by Go 1.23.2 and missing real libhs. The next live gate is Phase 0 on Go >=1.25 + verified libvectorscan, then the Phase 1 runner with representative CRS/corpus.

## 2026-09-14 Phase 3 Truth-Boundary Repair update

Current development baseline supersedes the older artifact list above. Coverage analysis and live VectorScan parsing now share `internal/capability` as the single eligibility classifier, and runtime Include/IncludeOptional handling is aligned with coverage ingestion. Release source archives identify only as `SOURCE_ARCHIVE`; actual portable/native binary identity is emitted only by successful binary builds. Schema-v2 release evidence binds govulncheck evidence to the canonical source manifest, adds provenance cross-digests, exact manifest completeness checks, SBOM↔go.mod dependency parity, and explicit unauthenticated-until-detached-signature-verified semantics.

The packaging host still has Go 1.23.2, no real libhs, no govulncheck, and no minisign. Canonical Go 1.25/Coraza/libvectorscan qualification remains BLOCKED/NOT_RUN. Artifact/source integrity gates may PASS without changing that runtime qualification status.
