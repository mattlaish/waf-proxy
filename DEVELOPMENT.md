# Development Engineering Ledger

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

## 2026-09-13 — Debug Evidence, Operator CLI and Phase 1 Production Qualification Runner

This slice completes the source-level supportability path without changing the authoritative WAF decision boundary.

Implemented:

- Correlated debug evidence now follows one server-generated transaction ID through request metadata, Coraza transaction-final `MatchedRules()` after `ProcessLogging()`, VectorScan candidates/false-negative comparison, load-balancer/upstream selection, TLS metadata and response status/latency.
- Coraza evidence is captured even when VectorScan is disabled or no eligible plan exists. In that case VectorScan evidence is explicitly `observed:false`; it is never treated as zero-false-negative proof.
- Debug capture is opt-in and site/tenant scoped, bounded by count and TTL, uses an atomic disabled fast path, omits request/response bodies by default, allow-lists request headers and masks sensitive fields before storage/export.
- Added admin endpoints for doctor/debug status/capture/evidence/export and the `wafctl` operator CLI (`doctor`, `debug capture|stop|list|export`, `support bundle`).
- Support bundles contain sanitized API/config/status evidence, optional incident evidence, bounded logs, dependency inventory, SPDX-formatted SBOM evidence, a manifest and SHA-256 list. Bundle creation rejects obvious private-key/admin-token material.
- Added `cmd/wafqualify`, `run-phase1-qualification.sh` and a representative JSONL corpus. The runner requires real libhs plus Go >=1.25, builds the real native VectorScan path, runs real Coraza DetectionOnly transactions and enforces zero observed false negatives over the current eligible rule set.
- Build/install/upgrade/uninstall/doctor/release-artifact scripts now include `wafctl`; generated binaries and qualification output are excluded from source ZIPs.

Truth boundary: source implementation of a real qualification runner is not qualification evidence. This host remains BLOCKED by Go 1.23.2 and absent `pkg-config libhs`; real Coraza/VectorScan/CRS results remain NOT_RUN.


## Phase 2 implementation update (2026-09-13)

Implemented qualification corpus schema foundation and differential false-negative gate helpers. Full runtime qualification remains deferred until real Go 1.25, Coraza v3.7.0 execution and libvectorscan are available.

## Phase 2 Slice B - implementation

Implemented (testing deferred):
- Debug capture lifecycle v2 session model foundation with state tracking and expiry cleanup primitives.
- wafctl qualification report command integration.
- VectorScan transition audit record model.

Validation remains deferred until the final Phase 2 validation stage.


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

## Phase 2 Coverage Expansion Slice B

Implemented CRS rule capability analyzer foundation. CRS metadata can now be normalized into the conservative VectorScan capability model. Analyzer output does not enable acceleration; promotion still requires validation evidence and zero false-negative qualification.

## 2026-09-14 — Phase 2 Coverage Expansion Slice C

Implemented CRS ruleset ingestion and coverage reporting without changing the WAF verdict path or VectorScan promotion state machine.

- Added deterministic CRS `.conf` ruleset loading from either a directory or a configuration entrypoint.
- `Include` and `IncludeOptional` are followed recursively with loop de-duplication, glob ordering, source-size bounds and mandatory-include failure semantics.
- Added normalized SecRule metadata for source file/line, id, phase, explicit operator, selectors, transforms, tags, severity, chain and negation.
- Added duplicate rule-ID detection and a versioned `coverage-inventory.json` model.
- Added Coverage Report v2 with total/eligible/Coraza-only/unsupported counts, duplicate IDs, warning count, coverage percentage and rejection-reason histogram.
- Added local operator command `wafctl coverage analyze --rules PATH [--report FILE] [--inventory FILE] [--json]`.
- Hardened the Slice A/B capability model to match the **current runtime classifier exactly**: explicit positive `@rx`, phase 1/2, exactly one reproducible selector, fixed-name `REQUEST_HEADERS:<name>` or supported scalar source, explicit `t:none`, and only `t:lowercase` after `t:none`. Bare `REQUEST_HEADERS`, `uppercase`, chains, negation, ARGS/body and multi-selector rules remain Coraza-only.
- Analyzer eligibility is inventory metadata only. It cannot move a group into LEARNING/VALIDATED/ACCELERATED and does not weaken the zero-false-negative gate.

Status: **IMPLEMENTED_TESTING_DEFERRED** until the final concentrated validation stage. Real CRS/Coraza/libvectorscan qualification remains NOT_RUN on this packaging host.

Slice C artifact preparation also corrected a baseline packaging defect: the supplied Slice B ZIP had executable release/install shell scripts stored as `0644`. The scripts required by the artifact verifier are restored to `0755`, and mode preservation is part of the final ZIP and patch reconstruction gate.

## 2026-09-14 — Phase 3 Release Engineering / Supply-Chain Hardening

Implemented the formal Phase 3 roadmap scope without changing the WAF verdict path:

- Added machine-readable release provenance plus SPDX 2.3 / CycloneDX 1.5 SBOM generation from the pinned Go module graph.
- Source ZIPs now carry source-release identity; actual portable/native binary identity is written only by `build.sh` after a successful binary build.
- Added a `govulncheck` evidence runner that preserves PASS / FAIL / BLOCKED / NOT_RUN truth; no missing toolchain or network access is promoted to PASS.
- Hardened complete-source ZIP generation with SOURCE_DATE_EPOCH support, deterministic metadata, source manifests, SBOM evidence and optional detached minisign signing. Signing is fail-closed when explicitly required and is otherwise NOT_CONFIGURED.
- Hardened artifact verification against duplicate/encrypted/unsafe paths, links/special files, archive expansion abuse, group/world-writable entries and secret/private-key-like filenames.
- Added automated negative artifact mutation tests and byte-reproducible double-build verification.

Status: **QUALIFICATION_REQUIRED**. Runtime/security qualification that needs Go >=1.25, real Coraza, real VectorScan, representative CRS or production host dependencies remains separate.

## 2026-09-14 — Phase 3 Truth-Boundary Repair

Implemented a repair-only slice after a Markdown↔code audit found release/provenance and coverage-classifier drift.

- Added `internal/capability` as the single eligibility classifier used by both offline coverage analysis and the live `internal/vectoraccel` parser. It owns phase defaults, positive `@rx`, non-empty pattern, chain/negation rejection, selector restrictions, explicit `t:none`, and optional lowercase semantics.
- Added runtime `IncludeOptional` handling and fail-closed mandatory Include behavior so coverage ingestion and runtime rule discovery no longer use different include semantics.
- Added parity regression tests covering previously divergent header punctuation, empty regex, implicit phase, chain-action detection, negation, multi-selector, unsupported transform, and `IncludeOptional`.
- Release evidence schema is now v2. Source archives are always `SOURCE_ARCHIVE`; portable/native binary identity is emitted only for actual binary build provenance.
- Added canonical `tools/source_manifest.py`; govulncheck evidence now binds to its source-manifest digest and includes tool identity/version, execution time, result digest, and raw result text. Invalid or source-mismatched supplied evidence fails packaging rather than degrading silently.
- Added `release-evidence/PROVENANCE.json` cross-digests and verifier checks for exact source/release manifest completeness plus SPDX/CycloneDX dependency parity with `go.mod`.
- Added detached minisign verification via `verify-release-signature.sh` and `wafctl release verify-signature`. Unsigned hash/provenance metadata is explicitly described as integrity-only, not producer authentication.
- `build.sh` now uses `go mod tidy -diff` so release builds fail on dependency-file drift instead of mutating `go.mod`/`go.sum`, and binary provenance records artifact type and native libhs runtime-linkage expectation.

Status: **QUALIFICATION_REQUIRED**. This repair does not create new real Coraza/libvectorscan/Go1.25 qualification evidence.

Truth-boundary validation found and repaired one additional release defect: importing the canonical Python source-manifest helper could create `tools/__pycache__`, changing the staging manifest, and plain `go version` could invoke Go's auto-toolchain download whose network error text contained nondeterministic ephemeral ports. Release source enumeration now excludes Python bytecode/cache artifacts, the builder suppresses bytecode generation, and release Go-version evidence forces `GOTOOLCHAIN=local`. Fixed-epoch reproducibility passed after these repairs.

## Phase 4 Slice A Trusted Client Identity Foundation

Implementation update:
- Added ClientIdentityDecision evidence model.
- Debug evidence now carries client identity decision context.
- Added initial `wafctl proxy identity show` operator visibility.
- Existing trusted proxy resolver remains authoritative; no duplicate resolver was introduced.

Status: IMPLEMENTED_TESTING_DEFERRED


Phase 4 Slice A follow-up: added client identity audit event model constants CLIENT_IDENTITY_RESOLVED and CLIENT_IDENTITY_HEADER_REJECTED.

## 2026-09-16 — Phase 5 Slice F Performance Certification

Slice F reuses the existing `wafbench` traffic generator and measurement model. A second benchmark engine was deliberately rejected because it would create evidence drift between profiling and certification.

Implemented `wafbench certify` as an evidence-binding layer. The command consumes full-proxy JSON results captured under explicit roles (`reverse_proxy_baseline`, `coraza_crs`, optional `vectorscan_assisted`), SHA-256 binds each input, rejects incomparable run shapes, and separates `evidence_status`, `target_status`, and final certification `status`.

The final status is intentionally strict: complete measurements alone do not become a certification PASS. An explicit owner-approved target file is required. VectorScan-assisted evidence additionally requires the real Phase 1 zero-false-negative report; analyzer, mock, ABI-only, or component benchmark evidence cannot satisfy that gate.

Regression output records RPS/p99 deltas only. It does not invent a hidden regression budget. Deployment profile names live in explicit target files so Small/Medium/Large sizing is not populated with guessed numbers.

Source-baseline repair: the input Slice E artifact contained a stale nested `waf-work/` mirror. Canonical source was repaired by restoring the five unique Phase 5 Slice B `qualification/vectorscan` files, correcting `replay.go` to the consistent package, then deleting the duplicate mirror. This is a packaging/source-completeness repair, not a WAF architecture change.

Validation on this host is limited by the pinned Go 1.25 requirement. Focused certification logic and repaired VectorScan qualification helpers were tested in isolated Go 1.23 standard-library-only harnesses; canonical module tests remain BLOCKED because the host cannot download the Go 1.25 toolchain. Real full-proxy performance measurements remain NOT_RUN.

### Slice F packaging-mode repair

The supplied Phase 5 Slice E baseline again stored all release/install shell scripts and the two executable release Python tools as mode `0644`. This broke the mandatory release packager before archive creation. Slice F restores the verifier-required executable mode (`0755`) for the build/install/upgrade/uninstall/qualification/release scripts, `benchmark/build.sh`, `tools/release_evidence.py`, and `tools/source_manifest.py`. Mode restoration is included in the baseline-relative patch and patch-reconstruction gate.

## 2026-09-16 — Phase 5 Slice G External HSM / PKCS#11 Support

Implemented the enterprise external-key-custody slice without changing Coraza, VectorScan, CIDR, L7-abuse, or proxy verdict semantics.

Engineering decisions:

- HSM support is a TLS private-key provider, not a second TLS stack. The existing Go listener still owns TLS; `tls.Certificate.PrivateKey` is a PKCS#11-backed `crypto.Signer`.
- Native PKCS#11 support is controlled by an explicit `pkcs11` build tag. `build.sh` exposes `WAF_HSM_PKCS11=off|auto|required` independently from `WAF_VECTORSCAN`, fixing the prior coupling where Coraza-only builds always used `CGO_ENABLED=0`.
- The provider dynamically loads the operator-approved module with `dlopen`, initializes PKCS#11, resolves one exact slot/token and one exact private key, opens/logs into one session, and closes session/module references on runtime replacement.
- Module loading is fail-closed: absolute clean paths only, approved directory tree only, regular non-symlink files only, no group/world write permission, and Linux root-ownership checks for the module and approved parent path.
- Key selectors are exact. Slot ID/token label and key label/CKA_ID are not fuzzy-matched; zero or multiple token/key matches fail.
- PINs are never accepted inline. Config stores only `env:NAME` or `file:/absolute/path` references; protected files must not be group/world accessible. Resolved byte copies are zeroed immediately after `C_Login`.
- Certificate/key association is verified before runtime swap by signing a random SHA-256 challenge and verifying against the configured certificate public key. A mismatch closes the signer and aborts Apply.
- Runtime signing errors propagate through `crypto/tls` and fail the handshake. HSM-backed sites cannot configure `tls_key`, and `tls_acceleration.mode=frontend` is rejected, so there is no automatic filesystem-key fallback.
- HSM audit events deliberately contain only provider, slot, key reference, operation, and result. Admin config GET/PUT responses redact the PIN secret reference. Qualification reports additionally omit module path and token label.
- Health exposes provider/slot/token/key status through `/api/hsm/status`; HSM audit evidence is reviewer-gated at `/api/hsm/audit` and may be forwarded through the existing audit syslog stream.
- `cmd/hsmqualify` performs the real provider open/login/key lookup, certificate association proof, health check, and in-memory TLS handshake. SoftHSM and real-vendor runners emit distinct evidence classes so software-token PASS cannot be mislabeled as vendor-HSM qualification.

Known/deferred qualification:

- SoftHSM is not installed in the packaging environment; its checked-in report remains `NOT_RUN`.
- Real vendor HSM hardware/service and vendor PKCS#11 libraries were not available; the vendor report remains `NOT_RUN`.
- Canonical repository-wide Go 1.25 tests remain blocked on this host; isolated HSM tests use a temporary Go 1.23 module only to verify the standard-library/CGO implementation shape and must not be promoted to full release qualification.

## 2026-09-16 — Enterprise Linux Distribution Packaging Slice A

Implemented the formal Debian/Ubuntu enterprise package path without changing WAF request/security verdict semantics. The package builder consumes already-built, checksum-verified binaries plus `BUILD_PROVENANCE.json`; it does not compile Go inside package installation and it requires `SOURCE_DATE_EPOCH` for deterministic package bytes.

Fresh install intentionally does not auto-start because the main package does not fetch or bundle OWASP CRS. `postinst` creates service ownership/state directories and one break-glass token only when absent, without echoing the token. Dpkg conffiles preserve local configuration changes. Persistent learner/security state uses `/var/lib/waf-proxy`, now declared as systemd `StateDirectory` so `ProtectSystem=strict` cannot make the configured state path unwritable.

Native VectorScan package dependencies are explicitly target-distribution-specific. The builder refuses a native-vectorscan package unless `WAF_DEB_EXTRA_DEPENDS` is supplied rather than guessing the libhs package name. HSM PKCS#11 remains dlopen-based and is not converted into a package-time vendor-library dependency.

The packaging-host fixture test builds the same package twice with a fixed epoch and verifies identical SHA-256 plus package content/maintainer-script rules. This is packaging-mechanics evidence only. No production `.deb` is claimed because this host cannot build the required Go 1.25 binaries.


## 2026-09-16 — Enterprise Linux Distribution Packaging Slice B

Implemented the formal RHEL-family RPM source path without changing WAF request/security verdict semantics. `packaging/rpm/build-rpm.sh` consumes checksum-verified prebuilt binaries and `BUILD_PROVENANCE.json`, binds ELF architecture to `x86_64`/`aarch64`, stages a deterministic payload, and drives `rpmbuild` with `SOURCE_DATE_EPOCH`, fixed build host, mtime clamping, and build-id suppression. `build-release-rpm.sh` is the qualified-release-host wrapper.

The RPM spec uses `%config(noreplace)` for `config.json`, `coraza.conf`, and the TLS frontend environment template. The generated `/etc/waf/waf-proxy.env` is intentionally not RPM-owned: it is created once when absent, never printed, and preserved across upgrades/erase/reinstall unless an operator explicitly removes it. Persistent learner/security state remains under `/var/lib/waf-proxy`.

Fresh installation never starts or presets the WAF because OWASP CRS is deliberately not downloaded by package scriptlets. Existing active services are `try-restart`ed after an upgrade so new bytes take effect without turning an inactive deployment on. Scriptlets also avoid `dnf`/`yum`/network fetches and do not disable or synthesize SELinux policy. Standard RHEL filesystem locations are used and enforcing-SELinux runtime qualification is deferred to Distribution Slice D.

Native VectorScan runtime dependency names are repository/distribution-specific. Like the DEB path, RPM builds refuse native-vectorscan provenance unless `WAF_RPM_EXTRA_REQUIRES` explicitly names the approved target runtime package rather than guessing. HSM/PKCS#11 vendor modules remain external dlopen targets.

Executed source validation PASSed, but the current Debian 13 host has no `rpmbuild`, `rpm`, or `rpm2cpio`; therefore no fixture/production RPM is claimed. Root Go release-script regression is also BLOCKED because the Go 1.25 toolchain cannot be downloaded in this environment.

Slice B release engineering completed with a reconstructable baseline-relative patch and complete-source artifact gates. Source archive integrity/reproducibility and all 12 mutation-rejection classes passed; this evidence remains separate from the missing real RPM toolchain and target-host qualification.


## 2026-09-16 — Enterprise Linux Distribution Packaging Slice C

Implemented the package upgrade/rollback qualification plane on top of Slice B.
The new `packaging/qualification/package_lifecycle_qualify.py` performs a
non-mutating preflight by default and a guarded real package transaction only
with `--execute`, root, a supported distro, and an exact dedicated-host
acknowledgement. It binds N/N+1/failure artifacts by package metadata + SHA-256,
then checks config, generated-secret, persistent-state and service-state
preservation through upgrade, intentional post-install failure, recovery and
rollback. No admin secret or secret digest is emitted.

Added deterministic DEB/RPM lifecycle fixture generation. Candidate defaults
are deliberately changed to exercise native conffile conflict handling. DEB
failure fixtures receive an intentional failing `postinst`; RPM uses a spec
macro compiled only when `build-rpm.sh --qualification-fail-post` is explicitly
used with a version containing `qualification`. Normal RPMs do not include the
failure branch.

Executed on this host: 7/7 lifecycle unit tests PASS; source/security policy
PASS; deterministic DEB N/N+1/failure fixture builds PASS and are byte-stable
across two fixed-epoch builds; DEB package preflight PASS and confirms all three
operator config defaults differ. Real DEB package-manager execution was not run
on this shared build environment. RPM source validation PASS, while RPM fixture
build is BLOCKED because `rpmbuild`, `rpm`, and `rpm2cpio` are unavailable.

Rejected approach: installing lifecycle fixtures into the shared build host was
not used as substitute evidence for a dedicated qualification VM. This avoids
mutating the shared package database and preserves the Slice C truth boundary.

## 2026-09-16 — Enterprise Linux Distribution Packaging Slice D

Implemented clean-host distribution qualification without changing WAF request
semantics. Added an exact seven-platform distro matrix and a real-host runner
that is non-mutating by default, root/ack gated for execution, binds local
package/CRS digests, refuses pre-existing WAF state, and performs the actual
enterprise operator path: native package install, no-autostart check, local CRS
provisioning, doctor, explicit systemd start, health, break-glass first login,
loopback backend proxy traffic, N+1 upgrade preservation, and non-purge remove.

Security decisions: no apt/dnf/yum/curl/wget/git execution; no secret value or
secret digest in JSON evidence; RHEL-family acceptance requires SELinux
Enforcing; a container/chroot/shared-host run cannot qualify a distro. The
current Debian 13 packaging environment is intentionally unsuitable for the
matrix, so real matrix qualification remains NOT_RUN.

## 2026-09-16 — Project-local DEB/RPM packaging utility

Status: `IMPLEMENTED_TESTING_DEFERRED`.

Implemented `./waf-package` plus `tools/waf_package_builder.py` as the single
project-specific orchestration layer above the canonical `build.sh`, DEB builder,
and RPM builder. The utility reads Go/Coraza requirements from `go.mod`, forces
`GOTOOLCHAIN=local`, computes a source fingerprint and deterministic epoch,
executes the canonical build gates once, packages the exact provenance-bound
binaries, and emits `PACKAGE_BUILD_REPORT.json` with source/package SHA-256
binding. `doctor` performs non-mutating host/toolchain readiness checks.

Hardening during implementation: Coraza pin parsing supports both single-line and
block `require` syntax; Git source identity treats tracked and untracked changes
as dirty; default version dates derive from `SOURCE_DATE_EPOCH`; requested
versions use one conservative DEB/RPM-safe spelling; `all` uses separate Debian
and RPM architecture arguments to prevent `amd64`/`x86_64` cross-format mixups.
The tool never installs or downloads prerequisites and never substitutes fixture
binaries for a failed canonical build.

Current shared host verification executes the source/unit gates, but a real
package build remains deferred because this host has Go 1.23.2 while `go.mod`
requires Go 1.25.0; RPM execution additionally requires the RPM toolchain.

## Root Build Integrity Repair — 2026-09-17

A post-delivery audit proved that `main@1d52d65` was not buildable even though a
source-baseline packaging gate had been reported PASS. The failure was caused by
three partially integrated patches plus module metadata drift: the CRL URL
refresher referenced `crlStore` state that a later `pki.go` patch had removed,
`debug_evidence_ops_v2.go` referenced an obsolete evidence model, and
`debug_bundle.go` referenced a missing TLS version helper. Coraza v3.7.0's
indirect module requirements had also been dropped from `go.mod`.

The repair restores the known-good CRL-store companion model, aligns debug
operator summaries with the current `DebugBundle`, adds `tlsVersionName`, and
restores the known-good Coraza v3.7.0 module metadata. Regression tests cover the
debug summary mapping and TLS-version helper. CI now requires `go mod tidy -diff`
before root `go build ./...`.

Truth-boundary change: archive generation, extraction, file presence, manifest
parity, and packaging integrity can be PASS independently, but the Source
Baseline Gate itself cannot be PASS or say "source tree complete/buildable"
unless the exact source bytes pass the Go 1.25 tidy and root-build gates.

Current local environment cannot execute that final compile gate because only
Go 1.23.2 is available and Go 1.25/module download is blocked. Therefore this
repair remains **IMPLEMENTED_TESTING_DEFERRED / buildability BLOCKED** until the
Go 1.25 CI run succeeds.

## 2026-09-17 — Complete Markdown synchronization

Documentation-only synchronization after the root-build-integrity audit. Added
`DOCUMENTATION_INDEX.md`, replaced stale stacked handover continuation text with
a single current checkpoint, expanded production package operations in
`INSTALL.md`, synchronized package/component qualification docs, and made the
Source Buildability Gate consistently BLOCKED until exact Go 1.25 execution.
No Go/runtime/package implementation semantics changed in this slice.

## 2026-09-17 — OpenAI integration hardening

Implemented native Responses API transport, strict Structured Outputs for WAF verdict/profile review, `env:`/protected `file:` secret references, legacy Chat Completions migration compatibility, admin/UI redaction, and mock HTTP integration coverage. OpenAI error bodies are not reflected into local errors. Provider failures remain fail-open at the AI enforcement boundary.

## 2026-09-23 — API-3 OpenAPI Contract Management integration closure

Implemented real API-3 source: OpenAPI 3.0/3.1 YAML/JSON parsing and validation, bounded local `$ref` resolution, flattened request/response schema metadata, immutable content-hash contract versions, authenticated contract inventory/import/detail/version/match/binding/compare/diff/drift APIs, deterministic API-1 operation matching, version/schema comparison, and API-2 learned-schema drift evidence. Fixed duplicate contract route registration and initialized/wired API-1/API-2 stores so the production asynchronous observation plane records API operations. No enforcement was added. Go qualification remains deferred.


## 2026-09-23 — API Security API-1 → API-3 closure

Status: `IMPLEMENTED_TESTING_DEFERRED`.

Repaired the API-2 `writeJSON` arity defect and completed API-1 durable operation discovery, API-2 live typed schema learning, advisory OpenAI SchemaCandidate review, and API-3 durable OpenAPI contract intelligence. Added API Security admin-console workflows, independent bounded schema body capture, privacy-conservative enum/value handling, semantic review invalidation, backlog body-sample shedding, persistent API state, OpenAPI export, parameter/enum/security parsing, expanded contract diff/drift, and a live API-1 → API-2 → API-3 deterministic integration test. No API-4 enforcement was introduced. Exact root Go 1.25 qualification remains BLOCKED on this host; see `API_SECURITY_CLOSURE_RESULT.md`.


## 2026-09-23 — API Security closure hardening follow-up

Kept API-1 through API-3 at `IMPLEMENTED_TESTING_DEFERRED` and added two fail-safe closure fixes before packaging: credential-header punctuation hardening excludes names such as `X-API-Key` before values can enter the bounded observation queue, and the operator review API now rejects `LEARNING → REVIEWED` lifecycle skips with HTTP 409. Added durable-ID continuity coverage for manual API-1 path reclassification. Re-executed API-1 isolated 4/4, API-2 core isolated 4/4, API-2 lifecycle handler 1/1, API-3 compile-only type gate, API-1→API-2→API-3 live JSON drift integration, and OpenAI SchemaCandidate advisory isolated 1/1. Root Go 1.25 gates remain blocked as recorded in `SOURCE_BASELINE_GATE_RESULT.md`.


## 2026-09-23 — API Security final closure hardening and evidence refresh

Status remains `IMPLEMENTED_TESTING_DEFERRED`; API-4 is `IMPLEMENTED_TESTING_DEFERRED`. Hardened API state persistence to use unique same-directory temporary inodes, fsync, and atomic rename, and added 32-writer deterministic coverage. Bounded API-2 aggregate field growth across requests, retained array-container evidence, invalidated stale review when the dominant observed type changes, and preserved composite authentication-family telemetry without credential values. Hardened API-3 by rejecting external `$ref`, rejecting duplicate normalized operations, modeling multiple content types, preserving OpenAPI security OR/AND/anonymous semantics, and scoping `UNDECLARED_ENDPOINT` drift to traffic from matched application site/host scope. The API-3 source gate was also repaired so its integration-test existence requirement executes before PASS is emitted.

Current local evidence: API-1 5/5 exact-source isolated tests PASS; API-2 8/8 exact-source isolated tests PASS; API-1/API-2 source gate 69 checks PASS; API-3 source gate 47 checks PASS; all Go files syntax-parse PASS via a module-external parser; 19 modified Go files are gofmt-clean; API-3 exact-source compile-only PASS with a minimal YAML shim; four API-3 deterministic runtime/integration tests not requiring YAML behavior PASS; OpenAI source contract 16/16 plus isolated provider/secretref PASS; SchemaCandidate advisory test PASS; package-builder 9/9 PASS; Python compileall, shell syntax, and admin inline-JS syntax PASS. Root Go 1.25 `tidy/build/vet/test/race/real-Coraza` remains BLOCKED before compilation because this host has Go 1.23.2 and cannot retrieve the Go 1.25 toolchain.

## 2026-09-23 — API-4 Positive Schema Enforcement

Implemented API-4 on the build-integrity/L7-hardening baseline. Reviewed SchemaCandidates can now be promoted into immutable positive-schema profile versions only when the operator review is current; AI output remains advisory and cannot activate enforcement. Added atomic runtime policy snapshots, LEARN/DETECT/ENFORCE lifecycle with DETECT as shadow mode and no direct LEARN->ENFORCE jump, required/type/format/enum/content-type/unknown-field validation, bounded body truncation fail-closed handling, scoped/expiring exceptions, rollback history, bounded privacy-conservative violation evidence, durable `api-positive-schema.json` state, admin APIs/UI, and audit entries for control-plane mutations. API-4 deterministic tests 7/7 and targeted race PASS; API-4 source gate 46 checks PASS; whole-repository test-binary compilation PASS in the dependency-stub compile-shape harness. Canonical Go 1.25 real-dependency tidy/build/vet/full-test/race/real-Coraza remains NOT_RUN/BLOCKED, so API-4 is `IMPLEMENTED_TESTING_DEFERRED`, not TESTED/RELEASED.

## API-5 JWT + Identity-aware API Security checkpoint — 2026-09-23

API-5 is implemented in source and remains `IMPLEMENTED_TESTING_DEFERRED`. `identity_api5.go` provides trusted issuer/JWKS verification, verified-claims-only request context, operation-scoped role/scope/tenant/client authorization, DETECT→ENFORCE lifecycle, semantic-change rollback to DETECT, bounded privacy-preserving evidence, and durable `api-identity.json` state. JWKS fetch is HTTPS-only and bounded; unknown-`kid` rotation refresh is serialized and rate-bounded. Raw bearer tokens, unverified claims and JWKS key cache are not persisted. OpenAI has no identity-enforcement authority.

Local evidence: 17 API-5 deterministic test functions PASS; targeted race PASS; `tools/tests/test-api5-source.py` 72 checks PASS; root source-shape PASS for 97 Go files; admin inline JavaScript syntax PASS; and the external-dependency stub harness passes whole-repository `go build ./...`, `go vet ./...`, and all test-binary compilation. These are not substitutes for the canonical Go 1.25 real-dependency gate, which remains `BLOCKED_ENVIRONMENT/NOT_RUN` (`go mod tidy -diff`, build, vet, full tests, full race, real-Coraza). API-6 remains `PLANNED` and has not started.

## API-6.1 Sequence Foundation checkpoint — 2026-09-23

Implemented only the sequence foundation: bounded non-blocking observation, API-1 normalized nodes, API-5 verified-identity-only correlation, HMAC anonymous correlation, capped/TTL session and transition state, immutable atomic snapshots, durable restart state and final drain, plus Reviewer/audit admin visibility. Added five targeted tests and a 33-check source gate wired into `build.sh` and CI. The source gate passes locally. Go deterministic/race execution is `BLOCKED_ENVIRONMENT/NOT_RUN` because no Go toolchain is installed; older deferred gates were not rerun. API-6.2/API-6.3/API-6.4 and API-7 were not started.


## API-6.2 Workflow Learning checkpoint — 2026-09-24

API-6.2 is implemented in source and remains `IMPLEMENTED_TESTING_DEFERRED`. The slice adds LEARN-only workflow cohorts, transition observation/session counts, frequency-derived confidence, `LEARNING`/`MATURE`/`STALE` cold-start semantics, workflow depth plus entry/terminal operation summaries, idle and absolute session lifetime, bounded workflow/session/transition state, API-5 verified-identity-only cohort input, durable v2 state with API-6.1 v1 migration, and Reviewer-gated/audited read-only workflow visibility. It adds no anomaly verdict, BLOCK/DENY/403 behavior, BOLA verdict, reset/relearn control, or sequence ENFORCE mode.

Executed source evidence: API-6.1 gate **33/33 PASS**, API-6.2 gate **56/56 PASS**, and Admin inline JavaScript syntax **PASS**. Seven targeted API-6.2 Go tests are present, including concurrency/race-relevant, restart/privacy and bounded-resource/adversarial cases. Canonical Go 1.25 targeted/race execution is `BLOCKED_ENVIRONMENT/NOT_RUN`: the local toolchain is Go 1.23.2 while `go.mod` requires Go 1.25.0, and external toolchain/dependency retrieval is unavailable. No stub or downgraded-toolchain evidence is used as qualification. API-6.3 is `IMPLEMENTED_TESTING_DEFERRED`; API-6.4 is `IMPLEMENTED_TESTING_DEFERRED`; API-7.1 through API-7.4 are `IMPLEMENTED_TESTING_DEFERRED`; API-8 remains `PLANNED`.

## API-6.3 Sequence Anomaly Detection checkpoint — 2026-09-24

API-6.3 is implemented in source and remains `IMPLEMENTED_TESTING_DEFERRED`. `sequence_api63.go` adds DETECT-only evidence for `UNKNOWN_TRANSITION`, `PREREQUISITE_SKIPPED`, `UNEXPECTED_ENTRY_POINT`, `SEQUENCE_REVERSAL`, `ABNORMAL_REPETITION`, and `WORKFLOW_DIVERGENCE`. Detection requires mature/sample/session/age/confidence evidence as applicable; probability zero alone is insufficient for divergence. Specific skip/reversal evidence suppresses duplicate generic unknown-transition evidence. Bounded persisted `SequenceViolation` and `SequenceException` primitives advance sequence durable state to v3 while retaining v1/v2 restore compatibility. The request path remains bounded non-blocking enqueue; detection has no HTTP response/enforcement path and OpenAI is absent.

Executed evidence: API-1/2 **69**, API-3 **47**, API-4 **46**, API-5 **72**, API-6.1 **33**, API-6.2 **56**, API-6.3 **45** source checks all PASS; OpenAI source contract **16/16 PASS**; package-builder **9/9 PASS**; package-source gate PASS; root source-shape gate PASS (103 root Go files); changed Go formatting PASS; Admin inline JavaScript syntax PASS. Canonical Go 1.25 targeted/race remains `BLOCKED_ENVIRONMENT/NOT_RUN` because the host is Go 1.23.2 and required toolchain/modules cannot be retrieved. API-6.4 is `IMPLEMENTED_TESTING_DEFERRED`; API-7.1 through API-7.4 are `IMPLEMENTED_TESTING_DEFERRED`; API-8 remains `PLANNED`.


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


## 2026-09-24 — API-8 Post-Audit Hardening

Implemented the post-API-8 repository audit fixes without defining a new API-security slice. CIDR runtime now honors Enabled and RFC3339 expiry. The previously dead TLS handshake abuse knob is implemented on the built-in Go TLS ClientHello path with bounded listener/peer token buckets and fails configuration validation with the external TLS frontend. The shipping Console now exposes System/Diagnostics, Debug, HSM, VectorScan, CIDR/L7 traffic controls, complete OpenAPI and Positive Schema lifecycle operations, and inventory-backed selectors for API-security opaque identifiers. Removed the obsolete non-shipping `web/` tree and six model-only/fake-foundation Go files that had no runtime store/API/Console wiring. Phase 5 Slice D truth is corrected to PLANNED / NOT IMPLEMENTED. Authority boundaries are unchanged. See `API8_POST_AUDIT_HARDENING.md`.

## 2026-09-25 — Production Correctness & Control-Plane Hardening

Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.

