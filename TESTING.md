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


## 2026-09-11 Supportability Slice

Added wafctl debug export/doctor/support bundle foundation and Phase 1 differential qualification helper. Real Coraza/libvectorscan execution remains NOT_RUN.


## Stage 2 Supportability Implementation
- Added wafctl supportability command foundation.
- Added qualification phase1 differential runner foundation.
- Real Go 1.25/Coraza/libvectorscan qualification remains NOT_RUN.


## Phase 2 VectorScan coverage test matrix

For every newly allow-listed transform/source, testing must preserve the rule that VectorScan candidates are only a superset prefilter and Coraza remains authoritative.

Required tests:

- ordered transform-pipeline unit tests, including Unicode case conversion, ASCII trim cutset, NUL handling, Latin-1 whitespace compression, Unicode whitespace removal, byte-length semantics, standard Base64 encoding, and lowercase hexadecimal byte encoding;
- classifier tests proving non-allow-listed transforms, ARGS/body, chains, negation and multi-variable selectors remain ineligible;
- Decode transforms (`base64Decode` / `hexDecode`), hashes and parser/normalization-dependent transforms remain negative eligibility cases until separately qualified;
- source reconstruction tests for `QUERY_STRING`, `SERVER_NAME`, `REQUEST_URI_RAW`, `REQUEST_LINE`, `REQUEST_BASENAME`, `REMOTE_ADDR`, `REMOTE_PORT`, ordinary fixed headers, `Host`, and `Transfer-Encoding`;
- fingerprint separation tests for different transform pipelines;
- `realcoraza` parity tests against Coraza v3.7.0 for every allow-listed transform/source before production qualification;
- real libvectorscan + representative CRS differential replay with observed false negatives = 0 before promotion;
- stale persisted state must invalidate after semantic adapter/fingerprint changes.

A portable or isolated test PASS never substitutes for the `realcoraza`/real-libvectorscan/CRS gates.

## Phase 3 release evidence gates

Every candidate release must distinguish `portable` from `native`. The artifact must contain `release-evidence/versions.json`, `provenance.json`, `sbom.spdx.json`, `sbom.cdx.json`, `govulncheck-status.json`, and `SHA256SUMS.txt`. The verifier must independently validate those checksums and the declared SPDX/CycloneDX formats.

For a production release, run `--require-govulncheck` on a suitable Go 1.25 host. A native release additionally requires real `pkg-config libhs` and must never derive native provenance from the ABI-only shim. Reproducibility is tested by running two builds from identical source/version/flavor with the same `SOURCE_DATE_EPOCH` and requiring byte-identical ZIPs. Detached signing is required only when an organizational key/process is explicitly configured; absence of a key is `NOT_RUN`, never PASS.

## Phase 4 security/product qualification gates

Phase 4 must be evaluated in layers; isolated stdlib tests are useful implementation evidence but are not a substitute for the project-wide Go 1.25 and deployed-host gates.

Required implementation tests include: HTTP CIDR allow-over-deny and expiry; request-ID replacement and correlation; rate/in-flight/connection/TLS limits including bounded identity-state behavior; custom block-page rendering; persistent-state round trip without raw bearer-token storage; static CRL regression; CRL URL syntax/SSRF rejection; refresh deduplication; last-known-good retention/cache; admin RBAC/audit routes; and configuration validation.

Release-host/deployed validation still required: `go test ./...`, `go vet ./...`, bounded race on the production module, real Coraza request-ID transaction correlation, external HTTPS CRL success/rotation/failure recovery, service restart state persistence under `ProtectSystem=strict`, concurrent abuse/load behavior, and HA/failover behavior where applicable. A packaging host lacking Go 1.25 or network access must record those as `BLOCKED/NOT_RUN` rather than infer PASS from isolated tests.
