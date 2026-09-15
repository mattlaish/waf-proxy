# Release Process

## Artifact Packaging Integrity Gate

### Purpose

Automated tests validate source-code correctness, but they do not prove that the final delivery artifact contains the validated files.

A release package can pass regression tests and still fail operationally when files are omitted or overwritten, generated documentation replaces executable content, archive contents drift from the validated source tree, or executable permissions are lost.

For this WAF project, every release package **MUST** pass an artifact inspection phase after packaging and before release. Source-tree tests alone are insufficient release evidence.

### Incident that established this policy

A mini-SIEM delivery ZIP once contained an `install-services.sh` file whose contents had been replaced by README text. The source repository, deployed `/opt/mini_siem` application files, `siem.db`, and running application data were unaffected; the defect existed only in the generated ZIP.

The incident demonstrated that this sequence is incomplete:

```text
source -> tests -> package creation -> release
```

The required sequence is:

```text
source
  -> unit/integration tests
  -> static/security checks
  -> package creation
  -> clean extraction
  -> artifact integrity gate
  -> checksums
  -> release
```

### Mandatory checks

The gate runs against a clean extraction of the delivery artifact, never only against the workspace used to create it.

1. **Archive safety and extraction**
   - verify ZIP CRC/integrity;
   - reject absolute paths and path traversal;
   - reject unexpected symbolic links;
   - extract into a clean temporary directory.

2. **Required-file existence**
   - confirm mandatory WAF source, build, install, qualification, and documentation files exist.

3. **File-size sanity**
   - reject empty or implausibly small critical executable/script files.

4. **Executable/script content**
   - executable shell files must retain an expected shell shebang;
   - executable attributes must survive packaging.

5. **Syntax validation**
   - run `bash -n` against shipped shell scripts;
   - apply equivalent syntax/compile validation to other shipped executable formats when introduced.

6. **Source-to-package byte comparison**
   - compare SHA-256 for every packaged source file against the validated source tree;
   - compare Unix permission modes;
   - a complete source ZIP must not silently omit source files.

7. **Release manifest**
   - every release ZIP must contain `RELEASE_MANIFEST.txt` generated for that artifact;
   - it records release/version label, build timestamp, file list, SHA-256 hashes, and test-summary text;
   - manifest hashes are independently verified against extracted files.

8. **Artifact-level smoke testing**
   - run feasible checks against the extracted artifact itself, not a neighboring workspace copy;
   - release-host/live-environment checks that cannot run must remain explicitly `NOT_RUN` rather than being inferred from packaging validation.

### WAF automation

`build-release-artifact.sh` creates a clean staged complete-source ZIP, generates `RELEASE_MANIFEST.txt`, and then invokes the artifact gate.

```bash
./build-release-artifact.sh /path/to/waf-proxy-release.zip \
  --version "waf-proxy <release-label>" \
  --test-summary TESTING_RESULTS.md
```

`verify-release-artifact.sh` independently validates an already-created ZIP against the source tree:

```bash
./verify-release-artifact.sh /path/to/waf-proxy-release.zip .
```

The verifier must return success before an artifact is reported as releasable.

### Evidence boundary

Passing this gate proves that the delivered ZIP contains the validated source bytes/modes and passes the implemented artifact-level checks. It does **not** convert `NOT_RUN` real Coraza, real VectorScan, CRS Learning, QAT, kTLS, or target-host qualification into PASS evidence.

### Release rule

No ZIP, installer, package, or deployment bundle is complete merely because its source tests passed. Delivery artifacts are part of the production surface and must be independently verified after creation.

## Supportability evidence in releases

The installed release now includes `/usr/local/bin/wafctl`. A release-host operational smoke should run `wafctl doctor`; incident investigations may enable a short site-scoped capture with `wafctl debug capture`, export one transaction with `wafctl debug export`, and generate a sanitized support ZIP with `wafctl support bundle`.

Support/debug ZIPs are diagnostic artifacts, not raw traffic dumps. They must remain bounded, tenant scoped and secret-sanitized. Their manifest/checksum validation is separate from the source-ZIP Artifact Packaging Integrity Gate.

For VectorScan production enablement, run `./run-phase1-qualification.sh --rules <production-or-representative-rules>` only after Phase 0 passes on the same class of host. Archive the generated JSON report with release evidence. A report is acceptable only when it says PASS, has zero false negatives, and contains the required minimum eligible Coraza match events; BLOCKED is not a release PASS.


## Phase 2 implementation update (2026-09-13)

Implemented qualification corpus schema foundation and differential false-negative gate helpers. Full runtime qualification remains deferred until real Go 1.25, Coraza v3.7.0 execution and libvectorscan are available.


## Phase 2 Slice B continuation

Implemented evidence operations, retention policy foundation, support provenance/SBOM evidence models, and VectorScan audit store. Full regression and real qualification remain deferred.

## Phase 3 release engineering and supply-chain hardening

Formal release candidates should use a fixed `SOURCE_DATE_EPOCH` and an explicit release variant. The complete-source package can be produced reproducibly with:

```bash
SOURCE_DATE_EPOCH=<approved-epoch> \
  ./build-release-artifact.sh /outside/tree/waf-source.zip \
  --version '<release-label>' --variant source --require-reproducible \
  --test-summary TESTING_RESULTS.md
```

The artifact contains `release-evidence/RELEASE_EVIDENCE.json`, `SOURCE_MANIFEST.sha256`, `sbom.spdx.json`, and `sbom.cyclonedx.json`. The evidence records the local Go runtime/directive, Coraza pin, VectorScan/libhs discovery, NGINX/OpenSSL discovery and optional CRS tree digest. These are provenance records, not runtime qualification.

`release-security-scan.sh` is the canonical govulncheck evidence runner. Use `--mode required` for a release gate. Missing Go >=1.25, missing govulncheck, unavailable network/module data, or other prerequisites must remain BLOCKED/NOT_RUN rather than PASS.

`build-release-artifact.sh` is a **source-archive** builder and may only emit `SOURCE_ARCHIVE` identity. When binaries are produced, `build.sh` separately records whether the actual WAF binary is `PORTABLE_BINARY` / `portable-coraza` or `NATIVE_BINARY` / `native-vectorscan` and writes `BUILD_PROVENANCE.json` plus `BUILD_SHA256SUMS.txt`. Native VectorScan provenance explicitly records that compatible runtime libhs is required. Never relabel a source archive as a binary artifact or one binary variant as the other.

Run both supply-chain post-package gates:

```bash
./release-artifact-negative-tests.sh /outside/tree/waf-source.zip .
./verify-reproducible-source-release.sh \
  --version '<release-label>' --source-date-epoch <approved-epoch> --variant source
```

Optional artifact signing uses detached minisign signatures only when an organizational signing secret is supplied externally:

```bash
./build-release-artifact.sh /outside/tree/waf-source.zip \
  --version '<release-label>' --variant source \
  --source-date-epoch <approved-epoch> --require-reproducible \
  --minisign-key /secure/offline/path/release.key --require-signature
```

No release signing secret belongs in the repository, installation tree, support bundle, or source artifact. If no approved key/process exists, signing status remains `NOT_CONFIGURED`.

## Phase 3 Truth-Boundary Repair — 2026-09-14

Release evidence is now source-bound and internally cross-digested. `SOURCE_MANIFEST.sha256` must exactly cover the packaged source set; `RELEASE_MANIFEST.txt` must exactly cover every packaged file except itself. `RELEASE_EVIDENCE.json` schema v2 must identify this builder's output as `SOURCE_ARCHIVE`, bind to the source-manifest digest, carry a structured native-VectorScan requirement state, and embed source-bound govulncheck evidence. `PROVENANCE.json` binds the source manifest, release evidence, and both SBOMs. The verifier also checks SPDX/CycloneDX dependency sets against `go.mod`.

This is an **integrity** boundary, not a cryptographic producer-authentication boundary. Anyone able to rewrite an unsigned archive can recompute unsigned hashes. Producer authenticity exists only after a detached minisign signature is verified with an independently trusted public key via `verify-release-signature.sh` or `wafctl release verify-signature`. `--require-signature` therefore requires both signing and public verification inputs.
