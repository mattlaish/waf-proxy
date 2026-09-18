# Testing Policy

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
