# OpenAI Integration Gate Result

## Status

`IMPLEMENTED_TESTING_DEFERRED`

The OpenAI connector hardening slice is implemented in source. The provider-level
Responses API and secret-reference packages have executed deterministic mock/unit,
race, and vet validation in an isolated standard-library-only Go module on the
current host. Repository-root integration remains `BLOCKED` because the exact
source requires Go 1.25.x and this host provides Go 1.23.2.

## Implemented contract

- Native OpenAI path: `POST {base_url}/responses`.
- Structured Outputs: `text.format.type=json_schema`, named schema,
  `strict=true`, and `additionalProperties=false`.
- WAF verdict schema: `verdict`, `score`, `category`, `reason`.
- Page-profile review schema: `agree`, `confidence`, `reason`.
- `store=false` on Responses requests.
- `max_output_tokens` uses the configured AI token limit.
- Bearer API-key authentication.
- Responses refusal, incomplete/non-completed status, malformed JSON, oversized
  responses, HTTP errors, and timeouts return errors; AI enforcement remains
  fail-open at the existing engine boundary.
- OpenAI-compatible Chat Completions remains available with
  `api_style=chat_completions`.
- Historical configs with omitted `api_style` plus a legacy inline `api_key`
  retain the historical Chat Completions wire contract until explicitly
  migrated.
- New secrets use `api_key_ref=env:NAME` or `file:/absolute/path`; the admin API
  returns neither secret values nor stored secret references.
- File secret references reject relative/unclean paths, symlinked path
  components, non-regular files, group/world-accessible files, empty/oversized
  values, and replacement detected between metadata validation and open.

## Executed evidence

Executed against exact copies of `internal/openaiapi` and `internal/secretref`
in an isolated temporary module using the host Go 1.23.2 toolchain. These
packages use only the Go standard library, so this is real package execution,
not parser-only evidence.

- `go test -count=1 ./openaiapi ./secretref`: **PASS**.
  - `openaiapi`: 4 top-level tests / 13 passing test+subtest events.
  - `secretref`: 4/4 top-level tests PASS.
- `go test -race -count=1 ./openaiapi ./secretref`: **PASS**.
- `go vet ./openaiapi ./secretref`: **PASS**.
- `tools/tests/test-openai-integration-source.sh`: **PASS**.
- OpenAI source/config/UI contract audit: **16/16 PASS**.
- `gofmt` on all Go files changed by this slice: **PASS**.

Mock coverage includes exact `/responses` path, Bearer auth, JSON content type,
model/store/token envelope, strict JSON Schema request shape, output-text parsing,
refusal/incomplete/malformed handling, HTTP 401/429/500 status-only errors,
timeout, response-size limit, env/file secret resolution, permission/symlink
rejection, and zeroing of resolved secret byte slices.

Repository-root integration tests additionally encode verdict/profile Structured
Outputs, Chat Completions compatibility, secret preservation/redaction, HTTPS
validation, and legacy inline-key migration semantics.

## Deferred / blocked evidence

- `GOTOOLCHAIN=local go test ./...`: **BLOCKED** before compilation because
  `go.mod` requires Go >=1.25.0 and the host is Go 1.23.2.
- Therefore `ai_openai_integration_test.go` repository-root execution is
  **BLOCKED**, not PASS.
- Full Go 1.25 root `tidy/build/vet/test/race/realcoraza` remains mandatory.
- No real OpenAI API key or external provider call was used; live provider
  acceptance is `NOT_RUN` and does not substitute for deterministic mock tests.

## Promotion rule

Do not promote this slice to `TESTED` until the exact complete-source commit
passes the mandatory Go 1.25 repository gates, including the root OpenAI
integration tests. Provider-level isolated PASS does not override the root Source
Buildability Gate.

## Final delivery freeze evidence — 2026-09-18

- Complete-source artifact verifier: **PASS — 273 source / 279 packaged files**.
- Fixed-epoch reproducibility: **PASS — byte-identical ZIPs**.
- Baseline-relative source patch reconstruction: **PASS — 273/273 source files byte + Unix-mode identical**.
- Final artifact negative mutations: **12/12 rejected PASS**.
- Root Go 1.25 buildability remains **BLOCKED** on this Go 1.23.2 host.
