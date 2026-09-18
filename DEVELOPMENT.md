# Development Engineering Ledger

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
