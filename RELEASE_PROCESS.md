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


## 2026-09-11 Supportability Slice

Added wafctl debug export/doctor/support bundle foundation and Phase 1 differential qualification helper. Real Coraza/libvectorscan execution remains NOT_RUN.


## Stage 2 Supportability Implementation
- Added wafctl supportability command foundation.
- Added qualification phase1 differential runner foundation.
- Real Go 1.25/Coraza/libvectorscan qualification remains NOT_RUN.

## Phase 3 supply-chain release flow

Use an explicit release flavor:

```bash
SOURCE_DATE_EPOCH=<unix-epoch> ./build-release-artifact.sh /tmp/waf-source.zip \
  --version '<release-label>' --flavor portable \
  --test-summary TESTING_RESULTS.md
```

Use `--flavor native` only on a host with verified real `pkg-config libhs`; the generator exits 3 otherwise. Production release runs should also use `--require-govulncheck`. The artifact contains a generated `release-evidence/` directory with SPDX/CycloneDX SBOMs, version/provenance information, govulncheck truth status, and its own checksum file. `verify-release-artifact.sh` validates these independently after clean extraction.

For reproducibility, the exact same source tree, version, flavor and `SOURCE_DATE_EPOCH` must produce a byte-identical source ZIP. Any differing release input is expected to change the hash.

Optional organizational signing is external-key only:

```bash
WAF_RELEASE_SIGNING_KEY=/secure/path/release-key.pem \
  ./sign-release-artifact.sh /tmp/waf-source.zip
./verify-release-signature.sh /tmp/waf-source.zip /tmp/waf-source.zip.sig release-public-key.pem
```

Never add the private key to the repository or release ZIP. If no organizational signing key/process exists, record signing as `NOT_RUN`; do not generate an ad-hoc key merely to turn the gate green.

## Phase 4 deployment-state release checks

A Phase 4 release additionally treats local state paths as production surface. The shipped systemd unit must keep `ProtectSystem=strict` while declaring `StateDirectory=waf-proxy`; extracted artifacts must retain that directive. Deploy/upgrade tests must verify `/var/lib/waf-proxy` ownership/permissions and that `security-state.json` and CRL cache files are service-writable but not world-readable.

Do not package runtime security-state or CRL-cache contents into a source/release artifact. They are mutable deployment state. Release qualification must separately exercise restart persistence, corrupt/expired cache handling, CRL refresh failure with last-known-good retention, and absence of raw session bearer tokens in persisted state.
