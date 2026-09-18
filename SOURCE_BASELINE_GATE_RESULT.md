# Source Baseline Gate Result

Date: 2026-09-17
Overall result: **BLOCKED**
Implementation state: `IMPLEMENTED_TESTING_DEFERRED`

## Scope

This gate answers one question: **are the exact source bytes being proposed a
complete, buildable Go source baseline?** Archive creation, extraction, manifest
parity, and package-fixture validation are separate gates and cannot answer that
question by themselves.

## Archive / reconstruction evidence

The root-build repair candidate has independent PASS evidence for source archive
construction, extraction, source/manifest parity, patch reconstruction,
fixed-epoch reproducibility, clean-extract static checks, and negative artifact
mutation rejection. Those results prove delivery integrity only.

## Mandatory buildability checks

Before this gate can become PASS, the exact repaired commit must execute
successfully on Go 1.25.x:

```bash
GOTOOLCHAIN=local go mod tidy -diff
GOTOOLCHAIN=local CGO_ENABLED=0 go build ./...
```

The release/CI sequence must then continue with vet, full tests, race, and the
real-Coraza truth gate as defined in `TESTING.md`.

Current host: Go 1.23.2. Go 1.25/module acquisition is unavailable here, so the
mandatory checks are **BLOCKED**, not inferred.

## Repaired failure classes

The prepared repair addresses the failures observed on audited
`main@1d52d65a73a802e32f02994e51f0a07beb240177`:

- Coraza v3.7.0 indirect dependency drift in `go.mod` / `go.sum`;
- `pki_url.go` expecting CRL-refresh state absent from the paired `crlStore`;
- debug-evidence v2 code using stale `List()` arity and stale `DebugBundle`
  fields;
- missing `tlsVersionName` helper;
- CI lacked a mandatory `go mod tidy -diff` step before build.

## Truth boundary

- `source archive generated/extracted`: may be PASS independently.
- `artifact integrity/reproducibility`: may be PASS independently.
- `package fixture/source mechanics`: may be PASS independently.
- `source tree complete/buildable`: **BLOCKED** until the exact repair passes the
  Go 1.25 buildability gate.
- runtime Coraza/VectorScan/HSM, package lifecycle, clean-host, reliability, and
  performance qualification are separate gates and are not promoted by compile
  success alone.

## OpenAI hardening delta

The current source now includes the OpenAI Responses/Structured-Outputs/secret-reference slice. This does not change the overall Source Buildability Gate: it remains **BLOCKED** until these exact bytes pass Go 1.25 `go mod tidy -diff`, root build, vet, tests, race, and real-Coraza CI.
