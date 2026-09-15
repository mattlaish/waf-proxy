# Testing Policy

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
