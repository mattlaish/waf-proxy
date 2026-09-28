# Source Baseline Gate Result

Date: 2026-09-23
Overall result: **BLOCKED**
Implementation state: `IMPLEMENTED_TESTING_DEFERRED`

## 2026-09-23 external-CI correction delta

The previous API-1→API-3 delivery was externally exercised with Go 1.25 and was not buildable as committed. The audit reported module metadata drift plus package/test compile failures. This working source fixes the confirmed local package defects (`ai.go` named-result redeclaration, duplicate AI test helper, and an additional duplicate `normalizeHost` symbol found during follow-up). The API-2/API-3 wrong-helper calls reported from the audited CI revision are not present in the supplied closure archive; the current tree uses `writeJSONCode` for explicit status codes, and the new root source-shape gate rejects future wrong-arity `writeJSON` calls.

A dependency-free AST/root-shape gate passes, and a temporary external-dependency stub harness now compiles every production package and every test binary. This is stronger local compile-shape evidence than the previous isolated gates, but it is not real-dependency or pinned-toolchain evidence. The exact Go 1.25 `go mod tidy -diff` result is still unavailable locally, so the externally reported module drift remains an open canonical buildability blocker until normalized and re-run on the real dependency graph.

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

## API-5 checkpoint — 2026-09-23

The exact API-5 working source adds `identity_api5.go` / `identity_api5_test.go`, API/Admin UI wiring, durable `api-identity.json` state, and `tools/tests/test-api5-source.py`. Dependency-free/source evidence is PASS: root source-shape 97 Go files, API12 69 checks, API3 47 checks, API4 46 checks, API5 72 checks, admin inline JavaScript syntax, 17 deterministic API-5 test functions and targeted race under the external-dependency stub harness. The stub harness also passes whole-repository build, vet and test-binary compilation; this is compile-shape evidence only.

The mandatory canonical gate was re-run after API-5 changes. On this host, `GOTOOLCHAIN=local go mod tidy -diff`, `CGO_ENABLED=0 go build ./...`, `go vet ./...`, `go test ./...`, and `CGO_ENABLED=1 go test -race ./...` all stop before compilation because `go.mod` requires Go 1.25.0 while the installed toolchain is Go 1.23.2. Automatic Go 1.25 acquisition also fails at `proxy.golang.org` DNS/network resolution. Classification remains **BLOCKED_ENVIRONMENT / NOT_RUN**, not PASS and not a source defect. API-1 through API-5 remain `IMPLEMENTED_TESTING_DEFERRED`; API-6 has not started.
