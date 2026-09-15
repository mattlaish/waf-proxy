# New Chat Handover Prompt

Copy everything below into the first message of the new development chat after attaching the latest handover/source package.

---

We are continuing development of my Go reverse-proxy WAF project. Treat the attached latest package as the source of truth and continue from it; do not restart the architecture or reconstruct an older version.

Before changing code, read these files in this order:

1. `AGENTS.md`
2. `README.md`
3. `AI_HANDOFF.md`
4. `DEVELOPMENT_ROADMAP.md`
5. `TESTING_RESULTS.md`
6. `MANIFEST.md`
7. `patch.md`
8. `INSTALL.md`

Current baseline and non-negotiable decisions:

- The project is a Go multi-site reverse proxy with embedded Coraza/OWASP CRS, admin UI, load balancing, AI-assisted analysis, discovery/learning, PKI controls, benchmark harness, optional NGINX/OpenSSL TLS frontend, and optional VectorScan Learning Accelerator.
- Current source is pinned to **Go 1.25.0** and **Coraza v3.7.0**.
- **VectorScan is the selected regex-acceleration direction. Do not reopen XDP-vs-VectorScan selection.** XDP is a separate optional/deferred future L3/L4 feature.
- Coraza is always authoritative. VectorScan is optional acceleration only.
- VectorScan eligible groups follow `LEARNING -> VALIDATED -> ACCELERATED`; unsupported rules stay Coraza-only. False negatives or native scan errors force `FAILSAFE` and Coraza-only behavior.
- Learning truth must come from transaction-final `tx.MatchedRules()` after Coraza `ProcessLogging()`, never from ErrorCallback/logging semantics.
- Keep exact request/transaction correlation through Coraza v3.7 context-aware transaction creation; do not go back to client-IP/URI heuristics.
- Do not fork Coraza merely to accelerate regex. Preserve optional/fallback-safe integration.
- The current conservative VectorScan scope intentionally excludes chains, negated regex, ARGS/body groups, multi-variable selectors and transforms whose Coraza input semantics are not reproduced exactly.
- TLS acceleration is already implemented as an optional NGINX/OpenSSL frontend with modern NGINX HTTP/2 syntax and kTLS/QAT capability/fallback logic. Default Go TLS behavior remains supported.
- P0-A/B/C/D, P1, P2 non-XDP hardening and the `wafbench` harness are already implemented. Do not redo them.

Current truth boundary:

- Portable Coraza-API-stub full regression/vet/bounded race: PASS.
- Native libhs ABI-only CGO compile/test/vet/race: PASS.
- Real NGINX 1.26.3/OpenSSL 3.5.5 HTTP/2 TLS-frontend smoke: PASS.
- **Real Go 1.25 + Coraza v3.7.0 execution: NOT_RUN in the packaging environment.**
- **Real `realcoraza` DetectionOnly+nolog `MatchedRules()` gate: NOT_RUN.**
- **Real libvectorscan `hs_compile_multi/hs_scan`: NOT_RUN.**
- **Production CRS Learning Period qualification: NOT_RUN.**
- Never report stub/ABI-only results as real Coraza or real VectorScan evidence.

Immediate next priority is **Phase 0 — real release-host qualification** from `DEVELOPMENT_ROADMAP.md`. Start with `./qualify-release-host.sh --preflight`; exit 3 means the host is BLOCKED rather than a correctness failure. On a host with Go >=1.25 and verified real libvectorscan, run `./qualify-release-host.sh --core`, then complete install/upgrade smoke and document exact PASS/FAIL evidence. If the environment does not support those dependencies, do not fabricate results; improve qualification automation/tests/docs or perform another explicitly approved slice instead.

After real qualification, proceed to VectorScan Learning production qualification in DetectionOnly. Zero observed false negatives is a hard safety criterion. Performance measurements are for optimization/regression/sizing; they are no longer a technology-selection gate.

Engineering rules:

- Keep production changes focused; do not broad-refactor unrelated files.
- Preserve fail-open/fail-safe boundaries already documented.
- Update `AI_HANDOFF.md`, `patch.md`, `DEVELOPMENT_ROADMAP.md` and `TESTING_RESULTS.md` after every meaningful development stage.
- Record what was actually executed vs NOT_RUN.
- Review diff hygiene before packaging.
- For release artifacts, produce a complete source ZIP and patch, SHA-256 both, re-extract the ZIP, reconstruct from the patch when applicable, and compare bytes/file modes.
- Do not include secrets, API keys, local helper files, generated binaries unless explicitly intended, or unrelated workspace files.
- When reporting completion of a development stage to me, end the final response with exactly: `UTM+8: YYYY-MM-DD HH:MM:SS`.

The latest known production source artifact before this handover-document update was:

- `waf-proxy-vectorscan-learning-coraza37-2026-09-04.zip`
- SHA-256 `64dc3034865d20aedaa38e10af5459da844ae6f6d92960e224223ec0ed359b22`

The corresponding patch was:

- `waf-proxy-vectorscan-learning-coraza37-2026-09-04.patch`
- SHA-256 `a24645a911caaa986c310509103093ba0cdc5e7bdcd4801251ccadd4e17a8398`

Start by summarizing the current state and the exact next gate you can execute in your environment, then continue the work directly without asking me to restate information already present in the package.

---

### 2026-09-13 continuation note

The current implementation now includes `wafctl` doctor/debug/support commands, correlated bounded Debug Evidence, and `run-phase1-qualification.sh`/`cmd/wafqualify`. Do not reimplement these. On a suitable release host, execute Phase 0 core first, then Phase 1 differential replay. Keep BLOCKED/NOT_RUN distinct from PASS and retain zero observed false negatives as the production promotion requirement.
