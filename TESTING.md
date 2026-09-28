# Testing Policy

## Current canonical baseline — 2026-09-28

The current source is the **Code Duplication Review and Consolidation** working
baseline derived byte-for-byte from the Production Correctness & Control-Plane
Hardening parent artifact (`90acc977db2998f4a3c1c4cacc07e19f0087f8bc033bce6b9934b83415957be8`)
before the changes documented in `CODE_DUPLICATION_REVIEW.md`. Status remains
**IMPLEMENTED_TESTING_DEFERRED**. No API-9 is defined.

The review removed only source layers proven to be unwired, superseded or
functionally duplicative, and consolidated the duplicated SecLang action/token
parser into `internal/capability`. Distinct API-6/API-7/API-8 state machines,
workers and authority boundaries remain separate. The shipping Console remains
`static/admin.html` + `static/theme.css`; the previously removed experimental
`web/` tree is not part of the current source.

Current dependency-free evidence: API source gates
**69/47/46/72/33/56/45/58/83/100/110/156/259/134/190 PASS**, code-duplication
source gate **80/80 PASS with 139 unique Admin/update routes**, OpenAI source
contract **16/16 PASS** plus isolated tests PASS, WAF package-source PASS,
package-builder **9/9 PASS**, and root Go source-shape **95 files PASS**. An
isolated dependency-free `internal/capability` test also passes. Canonical Go
1.25 tidy/build/vet/test/race remains **BLOCKED_ENVIRONMENT / NOT_RUN** on this
host; none of these static/source results promotes the product to TESTED or
RELEASED.

Dated sections below are retained as historical engineering/evidence records.
When an older section conflicts with this section, `CODE_DUPLICATION_REVIEW.md`,
`DOCUMENTATION_INDEX.md`, and the current source tree are authoritative.

## Build-integrity and L7 correction — 2026-09-23

An external Go 1.25 CI audit of the previous API-1→API-3 delivery showed that archive integrity and isolated slice tests had not proved package buildability. The current working source repairs the package-level defects found in that audit and an additional duplicate symbol found during follow-up:

- removed the named-result `err` redeclaration in `ai.go`;
- removed the duplicate `testAIEngineWithoutWorkers` test helper;
- consolidated the duplicate root-package `normalizeHost` helper;
- retained the correct two-argument `writeJSON` / three-argument `writeJSONCode` split and added a dependency-free root source-shape gate to reject wrong-arity `writeJSON`, duplicate top-level declarations, and named-result `var` redeclarations;
- corrected the client-identity proxy test so the normal XFF path and the fail-closed conflicting-XFF/X-Real-IP path are tested separately.

Phase 4 Slice B L7 abuse control is no longer a fixed-window/global-lock implementation. It now uses a continuous token bucket keyed by site + trusted client identity, 64 state shards, per-entry locking, bounded state (1024 entries/shard), amortized idle pruning/idle eviction, and deferred active-request release so downstream panic cannot leak concurrency state. The existing configuration remains global; per-site/per-page limit overrides and TLS-handshake-rate enforcement are still not implemented and must not be claimed.

Current local evidence is deliberately split by scope: the dependency-free root source-shape gate passes; a temporary local external-dependency stub harness makes the whole repository pass `go build ./...` and `go test -run '^$' ./...` type/test-binary compilation; targeted client-identity/L7/AI/API tests pass there; L7 isolated unit + race tests pass; and the local 5-worker limiter microbenchmark is 162.5–179.7 ns/op versus 330.6–337.3 ns/op for the previous global-lock implementation. The benchmark is a control-path microbenchmark, not production proxy throughput evidence.

**Source Buildability remains BLOCKED, not PASS.** The canonical Go 1.25 real-dependency `go mod tidy -diff`, build, vet, full tests, race, and real-Coraza gates have not run on these exact bytes in this environment. The prior external CI also reported committed module metadata drift; no guessed `go.mod` edit is accepted as a substitute for the exact Go 1.25 tidy result.

## API Security checkpoint — 2026-09-23

The uploaded API-3 baseline was audited against source rather than roadmap labels, then repaired/closed through API-3. Current source truth:

| Slice | State | Current evidence |
|---|---|---|
| API-1 Discovery + Operation Normalization | `IMPLEMENTED_TESTING_DEFERRED` | durable operation inventory, metadata, controls/UI; 5/5 exact-source isolated tests PASS |
| API-2 Typed Schema Learning | `IMPLEMENTED_TESTING_DEFERRED` | live typed collector, lifecycle, privacy controls, persistence; 8/8 exact-source isolated tests PASS |
| OpenAI SchemaCandidate review | `IMPLEMENTED_TESTING_DEFERRED` | Responses API Structured Outputs, advisory-only; OpenAI source/isolated gates PASS |
| API-3 OpenAPI Contract Management | `IMPLEMENTED_TESTING_DEFERRED` | persistence, import/export, params/enums/security, scoped drift; 47-check source gate + deterministic isolated/integration tests PASS |
| API-4 Positive Schema Enforcement | `IMPLEMENTED_TESTING_DEFERRED` | immutable reviewed profiles; LEARN/DETECT/ENFORCE, exceptions, rollback, bounded durable violation evidence; API-4 7/7 + race + 46-check source gate PASS |
| API-5 JWT + Identity-aware Policy | `IMPLEMENTED_TESTING_DEFERRED` | verified JWT/JWKS identity context, DETECT/ENFORCE operation policy, bounded durable evidence; API-5 17-test + race + 72-check source gate PASS |
| API-6.1 Sequence Foundation | `IMPLEMENTED_TESTING_DEFERRED` | five deterministic/restart/bounded/concurrency tests implemented; 33-check source gate PASS; Go targeted/race `BLOCKED_ENVIRONMENT/NOT_RUN` |
| API-6.2 Workflow Learning | `IMPLEMENTED_TESTING_DEFERRED` | LEARN-only workflow model; source gate 58 PASS; Go 1.25 targeted/race BLOCKED_ENVIRONMENT/NOT_RUN |
| API-6.3 Sequence Anomaly Detection | `IMPLEMENTED_TESTING_DEFERRED` | DETECT-only bounded anomaly evidence; source gate 45 PASS; Go 1.25 targeted/race `BLOCKED_ENVIRONMENT/NOT_RUN` |
| API-6.4 Sequence Operations + Hardening | `IMPLEMENTED_TESTING_DEFERRED` | explicit bounded LEARN/DETECT controls, exception CRUD, reset/relearn, recent sessions and full operations console; source gate 58 PASS; Go 1.25 targeted/race `BLOCKED_ENVIRONMENT/NOT_RUN` |
| API-7.1 Object Locator Discovery | `IMPLEMENTED_TESTING_DEFERRED` | normalized locator discovery + bounded keyed evidence; source gate 83 PASS; Go 1.25 targeted/race BLOCKED_ENVIRONMENT/NOT_RUN |
| API-7.2–7.4 BOLA Relationship/Detection/Policy | `IMPLEMENTED_TESTING_DEFERRED` | verified identity/object relationships, bounded BOLA candidate detection, REVIEW/SUPPRESS evidence policy/workflow/console; source gates 100/110/156 PASS; Go 1.25 targeted/race `BLOCKED_ENVIRONMENT/NOT_RUN` |
| API-8 GraphQL Security | `IMPLEMENTED_TESTING_DEFERRED` | bounded GraphQL parsing/normalization, SDL contract binding, depth/complexity/field/mutation/subscription/introspection/APQ controls, verified-variable API-7 bridge, staged deterministic LEARN/DETECT/ENFORCE; source gate 259 PASS; Go 1.25 targeted/race `BLOCKED_ENVIRONMENT/NOT_RUN` |

API-1 → API-2 → API-3 deterministic end-to-end learning/drift evidence is PASS. The repository-wide Source Buildability Gate remains **BLOCKED**, not PASS: this host has Go 1.23.2 while `go.mod` requires Go 1.25.0, and network/toolchain acquisition is unavailable. Required Go 1.25 `tidy/build/vet/test/race/real-Coraza` qualification remains `NOT_RUN/BLOCKED`. See `API_SECURITY_CLOSURE_RESULT.md`.

This working baseline came from the user-supplied source archive and contains no `.git` metadata; therefore no new branch/commit/push claim is made for this checkpoint.


## Current canonical status — 2026-09-17

The audited GitHub `main@1d52d65a73a802e32f02994e51f0a07beb240177`
was found non-buildable. A root-build-integrity repair is implemented, but the
exact repaired bytes have not executed the mandatory Go 1.25 dependency-drift
and root-build gates in this environment. Current source state is therefore
`IMPLEMENTED_TESTING_DEFERRED` and the Source Buildability Gate is `BLOCKED`.

Artifact integrity, source reconstruction, package fixtures, and component
source checks retain their separately scoped PASS evidence. They do not promote
root buildability or release readiness. Read `DOCUMENTATION_INDEX.md` and
`SOURCE_BASELINE_GATE_RESULT.md` before relying on older historical sections in
this ledger.

Detailed executed evidence remains in `TESTING_RESULTS.md`. This file defines test-policy requirements that apply to future releases.

## Artifact Packaging Integrity Gate

Every generated release artifact must be tested after packaging from a clean extraction. Required checks are:

- ZIP CRC/integrity and path-traversal/symlink rejection;
- required-file existence;
- critical-file size sanity;
- executable/script shebang and file-mode validation;
- syntax validation against extracted scripts;
- complete source-to-extracted-package SHA-256 comparison;
- source-to-package permission-mode comparison;
- `RELEASE_MANIFEST.txt` existence and checksum verification;
- feasible smoke tests against the extracted artifact itself.

`verify-release-artifact.sh` implements the WAF source-ZIP gate. `build-release-artifact.sh` stages a clean package, generates the release manifest, builds the ZIP, and invokes the verifier.

Artifact-integrity PASS must never be promoted into real Coraza, real VectorScan, CRS Learning, QAT, kTLS, or production-host evidence. Any unexecuted live gate remains `NOT_RUN`.

## Debug / supportability / Phase 1 gates

Future release validation must include these additional checks:

- debug disabled path adds no request ID/evidence and does not alter WAF decisions;
- enabled debug evidence remains exact-site/tenant scoped and bounded by TTL/count;
- incident/support bundles contain no Authorization/Cookie/password/API-token/private-key material and include independently verifiable manifests/checksums;
- Coraza debug truth is transaction-final `MatchedRules()` after `ProcessLogging()` regardless of whether VectorScan produced an observation;
- VectorScan differential qualification compares only the implementation's conservative eligible rule set and requires `candidates ⊇ eligible Coraza matches` for every observed sample;
- Phase 1 PASS requires observed false negatives = 0 and at least the configured minimum eligible-match evidence; missing prerequisites/evidence is BLOCKED/NOT_RUN, never PASS;
- operator smoke must exercise `wafctl doctor`, temporary debug capture, incident export and support-bundle generation against an installed instance.

`run-phase1-qualification.sh` is the canonical real Phase 1 runner. Its PASS is valid only on a release/target host satisfying Phase 0 provenance/toolchain requirements and using production or representative CRS/corpus inputs.


## Phase 2 implementation update (2026-09-13)

Implemented qualification corpus schema foundation and differential false-negative gate helpers. Full runtime qualification remains deferred until real Go 1.25, Coraza v3.7.0 execution and libvectorscan are available.


## Phase 2 Slice B continuation

Implemented evidence operations, retention policy foundation, support provenance/SBOM evidence models, and VectorScan audit store. Full regression and real qualification remain deferred.


## Phase 2 Coverage Expansion Slice A

Implemented source slice:
- conservative VectorScan coverage capability matrix
- rule eligibility evaluation
- transform capability registry
- coverage report model
- unsupported scopes remain Coraza-only

Status: IMPLEMENTED_TESTING_DEFERRED

## Phase 2 Coverage Expansion Slice C deferred validation matrix

Source implementation is complete; broad execution is intentionally deferred to the final concentrated validation stage.

Required tests before production use:

- parser tests for multiline SecRule metadata, fixed-header selectors, transforms, tags, severity, chain and negation;
- ruleset loader tests for Include/IncludeOptional, glob ordering, include loops, mandatory missing includes, byte/statement limits and directory ingestion;
- duplicate rule-ID inventory and report behavior;
- analyzer/runtime-classifier parity for every eligible selector/transform combination;
- real representative OWASP CRS ingestion and inventory sanity review;
- `wafctl coverage analyze` report/inventory file smoke;
- full Go 1.25 repository regression, vet and bounded race;
- Phase 0 real Coraza/libvectorscan core gate;
- Phase 1 real differential corpus and zero-FN gate;
- final Artifact Packaging Integrity Gate and patch reconstruction.

Until those execute, Slice C is `IMPLEMENTED_TESTING_DEFERRED`; coverage percentages are descriptive analyzer output, not production acceleration qualification.

## Phase 3 Release Hardening Gates

For a formal release candidate, run the following in addition to the existing Artifact Packaging Integrity Gate:

1. `./release-security-scan.sh --output <evidence>/govulncheck.json --mode required` on a network-enabled Go >=1.25 release host. `BLOCKED`/`NOT_RUN` is not PASS.
2. Build portable and native variants separately when both are shipped; their provenance must say `portable-coraza` and `native-vectorscan` respectively. Native evidence must show real `libhs` provenance.
3. Generate release evidence/SBOMs and verify SPDX 2.3 / CycloneDX 1.5 JSON parse and release component/dependency inventory.
4. Provide representative CRS path to release evidence when CRS is part of qualification and retain its tree SHA-256/file count.
5. Use a fixed `SOURCE_DATE_EPOCH` and run `verify-reproducible-source-release.sh`; both ZIPs must be byte-identical.
6. Run `release-artifact-negative-tests.sh` and require rejection of traversal, secret-like file, symlink, duplicate-entry, executable-mode-loss and installer-replacement mutations.
7. If organizational signing is mandatory, use an approved external minisign key plus `--require-signature`; absence/failure is a release blocker. Never store the secret key in source or the ZIP.
8. Keep real Coraza/VectorScan/CRS/kTLS/QAT/install/upgrade gates separate; release-hardening PASS cannot substitute for runtime qualification.

## Phase 3 Truth-Boundary Repair Gates

Before a source release can be accepted, additionally require:

- coverage analyzer and runtime VectorScan classifier parity over boundary cases;
- `Include` / `IncludeOptional` ingestion parity;
- source-bound govulncheck evidence schema and digest validation;
- `SOURCE_MANIFEST.sha256` exact file-set completeness;
- `RELEASE_MANIFEST.txt` exact packaged-file completeness;
- `RELEASE_EVIDENCE.json` must identify source ZIPs only as `SOURCE_ARCHIVE`;
- provenance cross-digest validation for release evidence and both SBOMs;
- SPDX and CycloneDX dependency sets must match `go.mod`;
- negative mutation rejection for forged evidence, artifact-type drift, manifest omission, and provenance/source mismatch;
- detached signature verification when a release claims authenticated producer identity. Without successful public-key verification, provenance is integrity-only.

## Phase 5 Slice F — Performance Certification Gate

A production performance certification requires full-proxy `wafbench http` JSON captured against the same qualified host, deterministic backend, workload, concurrency and GOMAXPROCS for:

1. TLS reverse proxy with WAF rules disabled (`reverse_proxy_baseline`);
2. TLS reverse proxy with Coraza + production/representative CRS (`coraza_crs`);
3. optional VectorScan-assisted Coraza/CRS configuration (`vectorscan_assisted`).

Run `wafbench certify` against those files and an owner-approved `waf-proxy-performance-target-v1` target. The certification gate must remain `NOT_RUN` without an explicit target. Incomparable hardware/run shape is `BLOCKED`.

If VectorScan evidence is included, provide the real Phase 1 `wafqualify` report. `result=PASS`, `zero_false_negatives=true`, and `false_negative_count=0` are mandatory; otherwise the performance certification cannot PASS.

Record at least RPS, application Gbps, p50/p95/p99, process CPU, CPU-us/request, RSS, generator CPU and (where available) network Gbps/softirq. Treat application Gbps as payload throughput, not line rate. Do not derive production sizing from component benchmarks or theoretical calculations.

## Phase 5 Slice G — External HSM / PKCS#11 required gates

Source-level gates:

- PKCS#11-disabled stub build and native `pkcs11` CGO build both compile/test.
- TLS handshake uses the supplied `crypto.Signer` rather than any filesystem key.
- certificate public key ↔ HSM signer association succeeds for matching keys and fails closed for mismatches.
- forced signer failure causes TLS handshake failure; no filesystem fallback is permitted.
- inline PINs and weak secret-file permissions are rejected; audit JSON is limited to approved non-secret fields.
- session open/login/exact-key lookup/sign/close lifecycle is covered with a native-interface fixture.
- config rejects simultaneous `tls_key` + `tls_key_provider` and rejects external TLS frontend mode for HSM-backed sites.
- qualification shell runners pass Bash syntax validation and never accept a raw PIN CLI argument.

Environment qualification gates:

- SoftHSM: run `qualification/hsm/run-softhsm-qualification.sh` against a pre-provisioned token/private key and matching certificate. PASS proves only the software PKCS#11 path.
- Real vendor HSM: run `qualification/hsm/run-vendor-hsm-qualification.sh` with the exact production-class provider/module/token/key/login policy. This gate remains `NOT_RUN` until actually executed.
- Vendor qualification must include representative TLS signing and operational failure behavior; mock or SoftHSM evidence cannot satisfy it.
- Repository-wide Go 1.25 build/test/race and artifact gates remain separate mandatory release evidence.

## Enterprise Linux Distribution Packaging qualification matrix

Slice A requires deterministic DEB build verification, conffile/state/secret preservation checks, maintainer-script network isolation, and package content verification. Production package qualification additionally requires canonical Go 1.25 binaries plus clean Debian/Ubuntu hosts.

Slice B will add RPM-specific equivalents. Slice C owns upgrade/rollback lifecycle qualification. Slice D owns clean-host distribution qualification. No package build result alone qualifies Coraza, VectorScan, HSM, or throughput behavior.


## Enterprise Linux Distribution Packaging — Slice B required gates

Source/package construction gates:

- RPM spec/source invariant validation;
- shell/Python syntax;
- exact ELF architecture -> RPM architecture binding;
- `BUILD_SHA256SUMS.txt` and `BUILD_PROVENANCE.json` binding;
- `%config(noreplace)` metadata for all operator config;
- scriptlet secret-preservation and no-secret-output checks;
- no package/CRS/network fetch in `%pre/%post/%preun/%postun`;
- no `setenforce`, `audit2allow`, or ad-hoc SELinux policy mutation;
- native VectorScan dependency must be explicitly supplied;
- repeated fixed-epoch RPM build must have identical SHA-256;
- extracted RPM content/scriptlet/file-flag verifier.

Target-host gates reserved for later slices:

- Slice C: upgrade/rollback, `.rpmnew/.rpmsave`, state and admin-secret preservation, failed-upgrade behavior;
- Slice D: clean RHEL/Rocky/Alma/Oracle install/start/traffic/remove under SELinux enforcing.

A missing RPM toolchain is BLOCKED, never PASS. A source/spec PASS never promotes a RHEL-family distribution to TESTED.


## Enterprise Linux Distribution Packaging Slice C test matrix

- lifecycle helper/unit tests: required and locally executable;
- DEB N/N+1/failure fixture reproducibility: required;
- RPM N/N+1/failure fixture reproducibility: required on an RPM build host;
- non-mutating package preflight and changed-default detection: required;
- real DEB upgrade + `.dpkg-dist` + failed-postinst recovery + rollback: required on dedicated Debian/Ubuntu host;
- real RPM upgrade + `.rpmnew` + failed-%post recovery + rollback: required on dedicated RHEL-family host;
- admin secret preservation: required; evidence must contain boolean only, no value/hash;
- persistent `/var/lib/waf-proxy` marker preservation: required;
- service active/inactive state preservation: required;
- `.dpkg-old` / `.rpmsave`: capture when emitted, never fabricate or force when native semantics do not produce them;
- production Go 1.25 package lifecycle: deferred until qualified binaries/package hosts are available;
- clean-host distro matrix: Slice D, not Slice C.

## Enterprise Linux Distribution Slice D — Clean-host qualification

Source gate:

```bash
./packaging/cleanhost/tests/test-clean-host-source.sh
```

The unit tests cover exact matrix definitions, distro/version validation, CRS
shape/digest validation, secret-free report structure, NOT_RUN preflight
semantics and the exact mutation acknowledgement. The source-policy gate checks
that the operator contract and all seven target platforms are present.

Real qualification must be executed on a dedicated clean target with qualified
Version N/N+1 packages and local approved CRS content. It is not acceptable to
substitute the current shared packaging host, a container, package extraction,
or a source-level fixture for this gate. RHEL-family runs require SELinux
Enforcing. Preserve the generated JSON report and only change that platform's
matrix state after the real run completes PASS.

## Project-local package builder gates

Required gates for `waf-package` changes:

- Python unit tests for Go/Coraza pin parsing, source fingerprinting, deterministic epoch/version handling, cross-package-safe version validation, package command construction, and package report hashing.
- shell/Python syntax validation and CLI help contract.
- source contract proving the utility invokes canonical `build.sh` and checked-in DEB/RPM builders and forces `GOTOOLCHAIN=local`.
- `doctor` must fail closed on an insufficient Go toolchain or missing package-host commands.
- real DEB: must be built only from canonical binaries and pass the existing DEB verifier.
- real RPM: must be built only from the same canonical provenance path and pass the existing RPM verifier.
- real package reproducibility remains a release-host gate; source/unit PASS does not substitute for package-byte evidence.

## Mandatory source-buildability gate

Before any source baseline may be labeled PASS/complete/buildable, the exact
source bytes must execute successfully under the pinned Go toolchain:

```bash
GOTOOLCHAIN=local go mod tidy -diff
GOTOOLCHAIN=local CGO_ENABLED=0 go build ./...
```

The CI-required test gates then run `go vet`, `go test ./...`, race tests, and
the real-Coraza transaction gate as defined in `.github/workflows/ci.yml`.
Archive extraction, source-manifest parity, SBOM/provenance verification, or
package-layout tests never substitute for this compile gate.

## Documentation consistency gate

Before delivery, inspect every tracked Markdown file. Current-state documents
must agree on: audited baseline, implementation state, Source Buildability Gate,
Go/Coraza pins, distribution matrix, PostgreSQL non-dependency, package/runtime
truth boundaries, and the exact next release gate. Historical ledgers may retain
older evidence but must not present stale next steps as current instructions.
A documentation consistency PASS never substitutes for source build/test PASS.

## OpenAI connector gate

For changes to AI provider code, require `tools/tests/test-openai-integration-source.sh`, isolated `internal/openaiapi` + `internal/secretref` unit/vet, race execution when supported, and repository-root `ai_openai_integration_test.go` under the mandatory Go 1.25 CI gate. Mock/provider-package PASS never substitutes for root buildability.

## API-3 OpenAPI Contract Management

Prepared Go coverage is in `api3_contract_test.go`; final execution is deferred to the project final Go 1.25 qualification gate. The executable source/static gate is `tools/tests/test-api3-source.py`. Acceptance rules are in `API3_ACCEPTANCE_CRITERIA.md` and cases in `API3_TEST_MATRIX.md`.


## API Security final closure evidence — 2026-09-23

Current authoritative scoped evidence for the API-1 → API-3 closure:

- API-1/API-2 source gate: **69 checks PASS**.
- API-3 source gate: **47 checks PASS**.
- All repository Go files: **syntax parse PASS** using a module-external parser (no dependency/type claim).
- Modified Go files: **19 files, gofmt-clean**.
- API-1 exact-source isolated tests: **5/5 PASS**, including 32 concurrent writers against atomic persistence.
- API-2 exact-source isolated tests: **8/8 PASS**.
- API-3 exact-source compile-only gate: **PASS** with a minimal YAML dependency shim; this does **not** claim real YAML runtime behavior.
- API-3 deterministic runtime/integration tests not requiring YAML behavior: **4 PASS**.
- OpenAI source contract: **16/16 PASS**; isolated provider/secretref tests PASS; SchemaCandidate advisory test PASS.
- Package builder: **9/9 PASS**; package source gate PASS.
- Python compileall, shell syntax, and admin inline-JavaScript syntax: **PASS**.
- Repository-root Go 1.25 tidy/build/vet/full-test/race/real-Coraza: **BLOCKED/NOT_RUN before compilation** on this Go 1.23.2 host; automatic Go 1.25 acquisition fails due network/DNS.

API-1 through API-8 remain `IMPLEMENTED_TESTING_DEFERRED`. API-8 exact-source evidence is documented in `API8_SOURCE_GATE_RESULT.md`; canonical Go 1.25 execution remains blocked. See `TESTING_RESULTS.md` and `SOURCE_BASELINE_GATE_RESULT.md`.

## API-4 local evidence — 2026-09-23

- `python3 tools/tests/test-api4-source.py`: PASS, 46 checks.
- dependency-stub whole-repository `go test -run '^$' ./...`: PASS (test binaries compile).
- targeted `TestAPI4*`: 7/7 PASS.
- targeted `go test -race -run '^TestAPI4' .`: PASS.
- shared request-body capture regression tests: PASS.
- Admin JavaScript syntax: PASS.

These are local implementation/compile-shape results, not canonical release qualification. Go 1.25 real-dependency `go mod tidy -diff`, `go build ./...`, `go vet ./...`, full `go test ./...`, full race, and real-Coraza remain NOT_RUN/BLOCKED here.

## API-5 JWT + Identity-aware API Security checkpoint — 2026-09-23

API-5 is implemented in source and remains `IMPLEMENTED_TESTING_DEFERRED`. `identity_api5.go` provides trusted issuer/JWKS verification, verified-claims-only request context, operation-scoped role/scope/tenant/client authorization, DETECT→ENFORCE lifecycle, semantic-change rollback to DETECT, bounded privacy-preserving evidence, and durable `api-identity.json` state. JWKS fetch is HTTPS-only and bounded; unknown-`kid` rotation refresh is serialized and rate-bounded. Raw bearer tokens, unverified claims and JWKS key cache are not persisted. OpenAI has no identity-enforcement authority.

Local evidence: 17 API-5 deterministic test functions PASS; targeted race PASS; `tools/tests/test-api5-source.py` 72 checks PASS; root source-shape PASS for 97 Go files; admin inline JavaScript syntax PASS; and the external-dependency stub harness passes whole-repository `go build ./...`, `go vet ./...`, and all test-binary compilation. These are not substitutes for the canonical Go 1.25 real-dependency gate, which remains `BLOCKED_ENVIRONMENT/NOT_RUN` (`go mod tidy -diff`, build, vet, full tests, full race, real-Coraza). API-6 remains `PLANNED` and has not started.


## API-6.2 Workflow Learning checkpoint — 2026-09-24

API-6.2 is implemented in source and remains `IMPLEMENTED_TESTING_DEFERRED`. The slice adds LEARN-only workflow cohorts, transition observation/session counts, frequency-derived confidence, `LEARNING`/`MATURE`/`STALE` cold-start semantics, workflow depth plus entry/terminal operation summaries, idle and absolute session lifetime, bounded workflow/session/transition state, API-5 verified-identity-only cohort input, durable v2 state with API-6.1 v1 migration, and Reviewer-gated/audited read-only workflow visibility. It adds no anomaly verdict, BLOCK/DENY/403 behavior, BOLA verdict, reset/relearn control, or sequence ENFORCE mode.

Executed source evidence: API-6.1 gate **33/33 PASS**, API-6.2 gate **56/56 PASS**, and Admin inline JavaScript syntax **PASS**. Seven targeted API-6.2 Go tests are present, including concurrency/race-relevant, restart/privacy and bounded-resource/adversarial cases. Canonical Go 1.25 targeted/race execution is `BLOCKED_ENVIRONMENT/NOT_RUN`: the local toolchain is Go 1.23.2 while `go.mod` requires Go 1.25.0, and external toolchain/dependency retrieval is unavailable. No stub or downgraded-toolchain evidence is used as qualification. API-6.3 is `IMPLEMENTED_TESTING_DEFERRED`; API-6.4 is `IMPLEMENTED_TESTING_DEFERRED`; API-7.1 through API-7.4 are `IMPLEMENTED_TESTING_DEFERRED`; API-8 remains `PLANNED`.

## API-6.3 Sequence Anomaly Detection checkpoint — 2026-09-24

API-6.3 source status is `IMPLEMENTED_TESTING_DEFERRED`. Seven deterministic Go tests are present for cold-start/mature unknown transitions, prerequisite/reversal semantics, unexpected-entry exceptions, probability-aware repetition, non-zero-probability workflow divergence, bounded restart/privacy persistence and concurrent bounded snapshots. The dedicated source gate passes **45/45** and all prior API source gates remain passing. Canonical Go 1.25 targeted and race execution remains `BLOCKED_ENVIRONMENT/NOT_RUN`; no source/static/advisory result is treated as runtime qualification.


## API-6.4 Sequence Operations + Hardening checkpoint — 2026-09-24

API-6.4 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. It adds explicit site-scoped `LEARN` / `DETECT` controls with mature-only DETECT promotion, bounded persisted recent-session summaries, bounded exception create/delete operations, site-scoped reset/relearn that returns to LEARN, Reviewer RBAC + audit, and the full Sequence Operations console. Durable sequence state advances to v4 with v1/v2/v3 restore compatibility. Sequence analytics still cannot BLOCK, DENY, return request-path 403, or enter ENFORCE. API-6 is implementation-complete through API-6.4; API-7.1 Object Locator Discovery is the next `PLANNED` slice.

Executed static/source evidence: API gates **69/47/46/72/33/56/45/58 PASS** from API-1/2 through API-6.4; OpenAI source contract **16/16 PASS** plus isolated OpenAI/secretref tests PASS; package-builder **9/9 PASS**; root Go shape **105 files PASS**; Admin JavaScript syntax, shell syntax and changed-file gofmt PASS. Canonical Go 1.25 targeted/race remains `BLOCKED_ENVIRONMENT/NOT_RUN` because the host has Go 1.23.2 and external toolchain/module retrieval is unavailable.


## API-7.1 Object Locator Discovery checkpoint — 2026-09-24

API-7.1 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. It adds bounded `ObjectLocator` discovery from API-1 normalized path parameters, API-2 typed path/query/body evidence, matched API-3 OpenAPI contracts, and explicit Reviewer-gated operator INCLUDE/SUPPRESS configuration. Locators are keyed by normalized API-1 operation ID plus location/field; raw object values are transient only and durable value evidence is bounded HMAC-SHA256 fingerprint data protected by a separate mode-0600 key. Client-supplied ownership/tenant headers are not authority, and GraphQL `variables.*` discovery is deferred to API-8.

API-7.1 has no identity/object relationship verdict, tenant-boundary verdict, `BOLA_CANDIDATE`, ownership verdict, BLOCK/DENY/403 path, or ENFORCE authority. OpenAI is absent from the locator authority path. State is bounded by global/per-operation/fingerprint/override caps, 30-day learned TTL, versioned restart validation, existing non-blocking observation-plane ingestion, immediate mutation persistence, RBAC and audit.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83 PASS** from API-1/2 through API-7.1; API-7.1 has **9 targeted test functions present**; OpenAI source contract **16/16 PASS** plus isolated tests PASS; package-source PASS; root Go shape **107 files PASS**; package-builder **9/9 PASS**; changed-file gofmt, Admin JavaScript and primary shell syntax PASS. Canonical Go 1.25 targeted/race remains `BLOCKED_ENVIRONMENT/NOT_RUN` because the host has Go 1.23.2 and external toolchain retrieval is unavailable. Next slice: **API-7.2 Identity/Object Relationship**. API-7.3/API-7.4 is `IMPLEMENTED_TESTING_DEFERRED`; API-8 remains `PLANNED`.

## API-7.2 Identity/Object Relationship checkpoint — 2026-09-24

API-7.2 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. It correlates only API-5 cryptographically verified identity context with ACTIVE API-7.1 locator/object evidence. Identity, tenant and client dimensions are persisted only as API-7.2 HMAC-SHA256 pseudonyms protected by a dedicated mode-0600 key; object values remain API-7.1 keyed fingerprints. Raw JWTs, subjects, tenant/client claim values, cookies, caller-supplied owner/tenant headers and raw object values are not durable relationship state.

The relationship plane is bounded and asynchronous: non-blocking queue capacity 2,048; 8,192 total relationships; 512 relationships per verified identity pseudonym; 256 verified identities per keyed object/locator pair; 30-day TTL; bounded path/query/body capture; restart revalidation; autosave/final flush; shutdown drain; Reviewer-gated read-only evidence/status endpoints with audit. SUPPRESSed API-7.1 locators cannot create relationships. GraphQL `variables.*` remains deferred to API-8.

API-7.2 is evidence only. `OBSERVED`/`REPEATED` means recurrence, not ownership. There is no `BOLA_CANDIDATE`, ownership verdict, tenant-boundary verdict, BLOCK/DENY/403 or ENFORCE authority, and OpenAI is absent from this authority path.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83/100 PASS** from API-1/2 through API-7.2; API-7.2 has **9 targeted Go test functions present**; OpenAI source contract **16/16 PASS** plus isolated OpenAI/secretref tests PASS; package-source PASS; root Go source shape **109 files PASS**; package-builder **9/9 PASS**; changed Go files `gofmt` PASS. Canonical Go 1.25 targeted/race remains `BLOCKED_ENVIRONMENT/NOT_RUN` because this host has Go 1.23.2 and external toolchain retrieval is unavailable. Next slice: **API-7.3 BOLA Detection**; API-7.4 is `IMPLEMENTED_TESTING_DEFERRED`; API-8 remains `PLANNED`.



## API-7.3 BOLA Detection checkpoint — 2026-09-24

API-7.3 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. The detector consumes only API-5 cryptographically verified identity pseudonyms, ACTIVE/non-suppressed API-7.1 keyed object locators/fingerprints, and API-7.2 relationship history. It emits bounded evidence-only candidates for `IDENTITY_OBJECT_DIVERGENCE`, `TENANT_OBJECT_DIVERGENCE`, and `OBJECT_ENUMERATION`. A novel object alone is not a candidate; cross-identity and cross-tenant evidence requires repeated historical baseline, while enumeration requires 20 recent distinct keyed objects under the same locator within 10 minutes.

Candidate state persists only pseudonymous/keyed evidence, is capped at 4,096 total / 256 per identity / 128 per object-locator, expires after seven days, is versioned/revalidated on restart, and is saved by the API security autosave/final-flush path. Detection runs inside the API-7.2 background processing plane before current relationship merge. Reviewer-only candidate/status reads are audited. There is no ownership verdict, tenant-boundary verdict, policy mutation API, `BLOCK`, `DENY`, request-path `403`, or `ENFORCE` authority; OpenAI is absent from the detector authority path.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83/100/110 PASS** from API-1/2 through API-7.3; API-7.3 has **9 targeted Go test functions present**; OpenAI source contract **16/16 PASS** plus isolated tests PASS; package-source PASS; root Go source shape **111 files PASS**; package-builder **9/9 PASS**; changed Go `gofmt`, Admin embedded JavaScript and primary shell/Python syntax PASS. Canonical Go 1.25 `go mod tidy -diff`, root build, API-7.3 targeted tests and API-7.3 race tests remain `BLOCKED_ENVIRONMENT/NOT_RUN` because this host has Go 1.23.2 and external toolchain retrieval is unavailable. Next slice: **API-7.4 BOLA Policy/Evidence/Console**; API-8 remains `PLANNED`.

## API-7.4 BOLA Policy/Evidence/Console checkpoint — 2026-09-24

API-7.4 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. It completes the API-7 BOLA implementation track with a bounded evidence-handling policy plane and operator workflow. Policies are scoped only by normalized API-1 operation ID, optional API-7.1 locator ID, optional API-7.3 candidate type, and minimum confidence. The only actions are `REVIEW` and `SUPPRESS`; suppression never deletes the underlying API-7.3 candidate. Operator evidence workflow is `OPEN` / `ACKNOWLEDGED` / `DISMISSED` / `RESOLVED` with enumerated reason codes only. New detector evidence after dismissal/resolution reopens the evidence automatically.

API-7.4 is deliberately absent from the API-7.2 relationship processor and API-7.3 detector authority path. It accepts no raw identity/object/tenant/client selectors, arbitrary headers, cookies, Authorization data, or free-text review notes. There is no ownership verdict, request-path `BLOCK`, `DENY`, `403`, or `ENFORCE` authority, and OpenAI is absent from the API-7.4 authority path. State is versioned/revalidated on restart, mutex-protected, TTL/cardinality bounded (1,024 policies; 4,096 reviews; 30-day default policy/review TTL; 180-day policy maximum), immediately persisted on mutation, and included in API security autosave/final flush. Reviewer-only policy/evidence APIs are audited and the embedded console exposes effective policy, evidence workflow, suppression, and reopen status.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83/100/110/156 PASS** from API-1/2 through API-7.4; **10 API-7.4 targeted Go test functions are present**; OpenAI source contract **16/16 PASS** plus isolated package tests PASS; package-source PASS; root Go source shape **113 files PASS**; package-builder **9/9 PASS**; `gofmt`, Admin embedded JavaScript and primary shell/Python syntax PASS. Canonical Go 1.25 `go mod tidy -diff`, root build/test, API-7.4 targeted tests and API-7.4 race tests remain `BLOCKED_ENVIRONMENT/NOT_RUN` because this host has Go 1.23.2 and external toolchain retrieval is unavailable. API-8 GraphQL Security is the next `PLANNED` slice.

## API-8 GraphQL Security checkpoint — 2026-09-24

API-8 is implemented across parsing/normalization, SDL contract binding, deterministic complexity/field/mutation/subscription/introspection policy, Apollo persisted-query support, verified GraphQL-variable integration into API-7 evidence, and Reviewer-gated `LEARN` / `DETECT` / `ENFORCE` policy plus Admin Console. Any policy edit or bound contract content change returns the policy to `LEARN`; a referenced schema contract cannot be deleted. Only explicit deterministic GraphQL policy in `ENFORCE` may reject traffic. API-6/API-7 learned or inferred evidence and OpenAI cannot promote, mutate or bypass GraphQL enforcement.

Durable GraphQL state excludes raw query documents, literals, raw variable values, Authorization/Cookie/JWT data and unverified claims. SDL is reduced to a digest plus normalized topology; GraphQL object values use API-7 keyed fingerprints and API-5 verified identity only. State and parser resources are bounded, versioned, restart-revalidated, autosaved/final-flushed, and strict mutation APIs are Reviewer-gated and audited.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83/100/110/156/259 PASS** through API-8; **22 targeted API-8 Go test functions are present**; OpenAI source contract **16/16 PASS** plus isolated tests PASS; package-source PASS; package-builder **9/9 PASS**; root Go source shape **115 files PASS**; changed API-8 Go files `gofmt`, Admin embedded JavaScript, relevant shell and Python syntax PASS. Canonical Go 1.25 `go mod tidy -diff`, root build/vet/test, API-8 targeted tests and race tests remain `BLOCKED_ENVIRONMENT/NOT_RUN` because the host has Go 1.23.2 and external toolchain retrieval is unavailable. API-8 remains `IMPLEMENTED_TESTING_DEFERRED`, not `TESTED` or `RELEASED`.

The documented API-security implementation roadmap is now complete through API-8. No API-9 is defined. Next gate: canonical Go 1.25 qualification of the exact source/artifact, followed by release promotion only if all required gates pass.


## API-8 Post-Audit Hardening gates

Required dependency-free gates include all API-1 through API-8 source gates, `tools/tests/test-api8-post-audit-hardening-source.py`, OpenAI source/isolated tests, WAF package-source, package-builder tests, root Go source-shape, changed-Go gofmt, embedded Admin Console JavaScript syntax, shell syntax, Python syntax, and clean-extract artifact integrity. Canonical Go 1.25 tidy/build/targeted/race remains required before promotion beyond IMPLEMENTED_TESTING_DEFERRED.

## 2026-09-25 — Production Correctness & Control-Plane Hardening

Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.

## Code Duplication Consolidation qualification — 2026-09-28

Required dependency-free checks now include
`tools/tests/test-code-duplication-review-source.py`. It must verify removal of
confirmed superseded layers, retention of replacement authorities, shared
SecLang parsing, route uniqueness, and unchanged API-6/API-7/API-8 authority
boundaries. The gate must run in both source and clean-extract validation.

Canonical Go 1.25 `tidy`, build, vet, full tests and race remain mandatory for
release promotion and are not replaced by the duplication source gate.

<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
