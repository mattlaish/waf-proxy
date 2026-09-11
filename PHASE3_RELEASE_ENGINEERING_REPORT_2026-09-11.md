# Phase 3 Release Engineering / Supply-Chain Report — 2026-09-11

## Scope implemented

Phase 3 hardens release evidence without changing WAF request processing. The release path now emits SPDX 2.3 and CycloneDX 1.5 SBOMs, component/version/provenance evidence, explicit portable/native flavor identity, govulncheck truth state, deterministic checksums, SOURCE_DATE_EPOCH reproducibility, and an optional detached-signature workflow using an external organizational key.

## Truth and security boundaries

- Portable flavor explicitly records VectorScan as not included.
- Native flavor fails closed unless real `pkg-config libhs` is available; ABI-only fixtures are not accepted as native provenance.
- `--require-govulncheck` exits 3 unless the scan completes successfully.
- Signing exits 3/NOT_RUN without `WAF_RELEASE_SIGNING_KEY`; no private key is generated, embedded, or retained.
- SBOM generation from `go.mod` is source/dependency evidence; it is not mislabeled as a binary-observed SBOM.
- Phase 3 does not waive Phase 0/1/2 real Coraza/VectorScan/CRS qualification.

## Executed evidence on current host

- Go: go1.23.2 linux/amd64.
- OpenSSL: 3.5.5.
- NGINX: 1.26.3.
- libhs: unavailable.
- govulncheck: unavailable.
- organizational signing key: not configured.
- Phase 3 shell syntax: PASS.
- Phase 3 standalone tooling test: PASS.
- Portable evidence generation: PASS.
- Same-input SOURCE_DATE_EPOCH reproducibility: PASS; two pre-final-doc ZIPs were byte-identical.
- Native flavor gate: BLOCKED / exit 3 as designed.
- Required govulncheck gate: BLOCKED / exit 3 as designed.
- Detached signing: NOT_RUN.

Final artifact integrity and patch reconstruction are recorded in the delivery verification report and `TESTING_RESULTS.md` after final packaging.
- Detached-signature implementation self-test using an ephemeral test-only RSA key: PASS. This validates the mechanism only; organizational release signing remains NOT_RUN.
- Full repository Go test: BLOCKED because the host has Go 1.23.2 and `go.mod` requires Go 1.25.0.
