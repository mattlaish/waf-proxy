# Release Process

## Current canonical baseline — 2026-09-28

The current source is the **Code Duplication Review and Consolidation** working
baseline derived byte-for-byte from the Production Correctness & Control-Plane
Hardening parent artifact (`90acc977db2998f4a3c1c4cacc07e19f0087f8bc033bce6b9934b83415957be8`)
before the changes documented in `CODE_DUPLICATION_REVIEW.md`. Status remains
**IMPLEMENTED_TESTING_DEFERRED**. No API-9 is defined.

The review removed only source layers proven to be unwired, superseded or
functionally duplicative, and consolidated the duplicated SecLang action/token
parser into `internal/capability`. Distinct API-6/API-7/API-8 state machines,
workers and authority boundaries remain separate. The shipping Console remains
`static/admin.html` + `static/theme.css`; the previously removed experimental
`web/` tree is not part of the current source.

Current dependency-free evidence: API source gates
**69/47/46/72/33/56/45/58/83/100/110/156/259/134/190 PASS**, code-duplication
source gate **80/80 PASS with 139 unique Admin/update routes**, OpenAI source
contract **16/16 PASS** plus isolated tests PASS, WAF package-source PASS,
package-builder **9/9 PASS**, and root Go source-shape **95 files PASS**. An
isolated dependency-free `internal/capability` test also passes. Canonical Go
1.25 tidy/build/vet/test/race remains **BLOCKED_ENVIRONMENT / NOT_RUN** on this
host; none of these static/source results promotes the product to TESTED or
RELEASED.

Dated sections below are retained as historical engineering/evidence records.
When an older section conflicts with this section, `CODE_DUPLICATION_REVIEW.md`,
`DOCUMENTATION_INDEX.md`, and the current source tree are authoritative.


## Build-integrity and L7 correction — 2026-09-23

An external Go 1.25 CI audit of the previous API-1→API-3 delivery showed that archive integrity and isolated slice tests had not proved package buildability. The current working source repairs the package-level defects found in that audit and an additional duplicate symbol found during follow-up:

- removed the named-result `err` redeclaration in `ai.go`;
- removed the duplicate `testAIEngineWithoutWorkers` test helper;
- consolidated the duplicate root-package `normalizeHost` helper;
- retained the correct two-argument `writeJSON` / three-argument `writeJSONCode` split and added a dependency-free root source-shape gate to reject wrong-arity `writeJSON`, duplicate top-level declarations, and named-result `var` redeclarations;
- corrected the client-identity proxy test so the normal XFF path and the fail-closed conflicting-XFF/X-Real-IP path are tested separately.

Phase 4 Slice B L7 abuse control is no longer a fixed-window/global-lock implementation. It now uses a continuous token bucket keyed by site + trusted client identity, 64 state shards, per-entry locking, bounded state (1024 entries/shard), amortized idle pruning/idle eviction, and deferred active-request release so downstream panic cannot leak concurrency state. The existing configuration remains global; per-site/per-page limit overrides and TLS-handshake-rate enforcement are still not implemented and must not be claimed.

Current local evidence is deliberately split by scope: the dependency-free root source-shape gate passes; a temporary local external-dependency stub harness makes the whole repository pass `go build ./...` and `go test -run '^$' ./...` type/test-binary compilation; targeted client-identity/L7/AI/API tests pass there; L7 isolated unit + race tests pass; and the local 5-worker limiter microbenchmark is 162.5–179.7 ns/op versus 330.6–337.3 ns/op for the previous global-lock implementation. The benchmark is a control-path microbenchmark, not production proxy throughput evidence.

**Source Buildability remains BLOCKED, not PASS.** The canonical Go 1.25 real-dependency `go mod tidy -diff`, build, vet, full tests, race, and real-Coraza gates have not run on these exact bytes in this environment. The prior external CI also reported committed module metadata drift; no guessed `go.mod` edit is accepted as a substitute for the exact Go 1.25 tidy result.

## API Security closure qualification rule — 2026-09-23

API-1 through API-3 are `IMPLEMENTED_TESTING_DEFERRED`. Their isolated/source evidence may be included in release evidence, but it does not satisfy the repository root-build gate. Before API-4 begins or any API Security slice is promoted to `TESTED`, the exact packaged source bytes must pass pinned Go 1.25 `go mod tidy -diff`, root build, vet, full tests, race, and real-Coraza qualification (plus real VectorScan where the selected release variant requires it). After those pass, rerun this document's Artifact Packaging Integrity Gate against the final ZIP. `BLOCKED`/`NOT_RUN` must remain explicit.


## API Security checkpoint — 2026-09-23

The uploaded API-3 baseline was audited against source rather than roadmap labels, then repaired/closed through API-3. Current source truth:

| Slice | State | Current evidence |
|---|---|---|
| API-1 Discovery + Operation Normalization | `IMPLEMENTED_TESTING_DEFERRED` | durable operation inventory, metadata, controls/UI; 5/5 exact-source isolated tests PASS |
| API-2 Typed Schema Learning | `IMPLEMENTED_TESTING_DEFERRED` | live typed collector, lifecycle, privacy controls, persistence; 8/8 exact-source isolated tests PASS |
| OpenAI SchemaCandidate review | `IMPLEMENTED_TESTING_DEFERRED` | Responses API Structured Outputs, advisory-only; OpenAI source/isolated gates PASS |
| API-3 OpenAPI Contract Management | `IMPLEMENTED_TESTING_DEFERRED` | persistence, import/export, params/enums/security, scoped drift; 47-check source gate + deterministic isolated/integration tests PASS |
| API-4 Positive Schema Enforcement | `IMPLEMENTED_TESTING_DEFERRED` | reviewed immutable profiles; LEARN/DETECT/ENFORCE, exceptions, rollback, bounded violation evidence; local API-4 gates PASS |
| API-5 JWT + Identity-aware Policy | `IMPLEMENTED_TESTING_DEFERRED` | verified JWT/JWKS identity context, DETECT/ENFORCE operation policy, bounded durable evidence; API-5 17-test + race + 72-check source gate PASS |
| API-6.1 Sequence Foundation | `IMPLEMENTED_TESTING_DEFERRED` | source gate 33 PASS; Go targeted/race `BLOCKED_ENVIRONMENT/NOT_RUN` |
| API-6.2 Workflow Learning | `IMPLEMENTED_TESTING_DEFERRED` | LEARN-only workflow model; source gate 58 PASS; Go 1.25 targeted/race BLOCKED_ENVIRONMENT/NOT_RUN |
| API-6.3 Sequence Anomaly Detection | `IMPLEMENTED_TESTING_DEFERRED` | DETECT-only bounded anomaly evidence; source gate 45 PASS; Go 1.25 targeted/race `BLOCKED_ENVIRONMENT/NOT_RUN` |
| API-6.4 Sequence Operations + Hardening | `IMPLEMENTED_TESTING_DEFERRED` | explicit bounded LEARN/DETECT controls, exception CRUD, reset/relearn, recent sessions and full operations console; source gate 58 PASS; Go 1.25 targeted/race `BLOCKED_ENVIRONMENT/NOT_RUN` |
| API-7.1 Object Locator Discovery | `IMPLEMENTED_TESTING_DEFERRED` | normalized locator discovery + bounded keyed evidence; source gate 83 PASS; Go 1.25 targeted/race BLOCKED_ENVIRONMENT/NOT_RUN |
| API-7.2–7.4 BOLA Relationship/Detection/Policy | `PLANNED` | not started |
| API-8 GraphQL Security | `IMPLEMENTED_TESTING_DEFERRED` | bounded GraphQL parsing/normalization, SDL contract binding, depth/complexity/field/mutation/subscription/introspection/APQ controls, verified-variable API-7 bridge, staged deterministic LEARN/DETECT/ENFORCE; source gate 259 PASS; Go 1.25 targeted/race `BLOCKED_ENVIRONMENT/NOT_RUN` |

API-1 → API-2 → API-3 deterministic end-to-end learning/drift evidence is PASS. The repository-wide Source Buildability Gate remains **BLOCKED**, not PASS: this host has Go 1.23.2 while `go.mod` requires Go 1.25.0, and network/toolchain acquisition is unavailable. Required Go 1.25 `tidy/build/vet/test/race/real-Coraza` qualification remains `NOT_RUN/BLOCKED`. See `API_SECURITY_CLOSURE_RESULT.md`.

This working baseline came from the user-supplied source archive and contains no `.git` metadata; therefore no new branch/commit/push claim is made for this checkpoint.


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

## Main-branch CI admission rule

The CI workflow running on pushes is detection, not protection. Production/release work must not be committed directly to `main`: use a branch/PR and configure repository branch protection or a ruleset so the `build, vet, test (portable)` job is a required status check before merge. A red post-push CI run on `main` is already too late. If repository permissions do not allow applying the protection, record that as an operational blocker rather than treating the workflow file as a merge gate.

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


## Enterprise Linux binary package release paths

Formal production Linux package release paths are Debian/Ubuntu `.deb` and RHEL-family `.rpm`. Both consume already-built provenance/checksum-bound binaries, require a fixed `SOURCE_DATE_EPOCH`, and keep package installation offline-safe. Package lifecycle code must not download OWASP CRS.

Debian/Ubuntu:

```bash
SOURCE_DATE_EPOCH=<approved-epoch> packaging/deb/build-release-deb.sh --output-dir dist/deb
```

RHEL/Rocky/AlmaLinux/Oracle Linux:

```bash
SOURCE_DATE_EPOCH=<approved-epoch> packaging/rpm/build-release-rpm.sh --output-dir dist/rpm
```

A native VectorScan package must declare the exact approved target-distribution runtime dependency through the package-specific environment variable; builders fail closed rather than guessing repository package names. A built package is not a distro qualification PASS: Slice C owns upgrade/rollback qualification and Slice D owns clean-host/SELinux-enforcing qualification.


## Distribution Slice C release gate

Source releases after Slice C must contain the package-lifecycle runner, fixture
builder, source tests, operator guide, DEB/RPM `NOT_RUN` placeholders and
`PACKAGE_LIFECYCLE_GATE_RESULT.md`. Release integrity validates their executable
modes/syntax and source parity. A release may remain
`IMPLEMENTED_TESTING_DEFERRED` when target package-manager lifecycle execution
has not occurred; source-artifact integrity must not promote that gate.

## Clean-host distribution gate

Distribution Slice D adds `packaging/cleanhost/`. Release acceptance must not
mark a distro PASS from package build success, a container, a chroot, source
installation, or preflight. A distro PASS is valid only when the real local DEB
or RPM executes install → explicit CRS provision → doctor → systemd start →
health/authenticated-admin/proxy-traffic smoke → N+1 upgrade → removal on the
exact clean target OS. RHEL-family evidence additionally requires SELinux
`Enforcing`. Checked-in matrix entries remain `NOT_RUN` until that execution is
performed and its JSON report is retained as release evidence.

## Project-local binary package entry point

Use `./waf-package` as the normal release-host entry point after source changes.
Do not manually package stale binaries. `waf-package` runs the canonical build
first, verifies Go/Coraza/tooling policy, then delegates to the checked-in DEB or
RPM builder/verifier and writes `PACKAGE_BUILD_REPORT.json`.

Typical release-host commands:

```bash
./waf-package doctor --format deb
SOURCE_DATE_EPOCH=<epoch> ./waf-package deb --version <version> --offline

./waf-package doctor --format rpm
SOURCE_DATE_EPOCH=<epoch> ./waf-package rpm --version <version> --rpm-release 1 --offline
```

A `PACKAGE_BUILD_PASS` is package-construction evidence only. It does not replace
Slice C package lifecycle or Slice D clean-host distribution qualification.

## Root-build and merge protection gate

A source release is not eligible merely because it packages and extracts
correctly. Before a source baseline can be called complete/buildable, CI must
pass `go mod tidy -diff` and root `go build ./...` on the exact commit, followed
by the repository test gates.

Production process requirement: `main` changes go through pull requests; direct
patch commits to `main` are not an acceptable release workflow. Repository
branch protection/rulesets should require the CI build-test check before merge.
If branch protection is not enabled, release readiness remains a governance gap
and must be reported rather than assumed.

## Documentation freeze gate

Before packaging a release/source handoff, synchronize every Markdown file with
`DOCUMENTATION_INDEX.md`, the current Source Buildability Gate, and current test
evidence. Current handover files must not contain superseded "next slice"
instructions. Historical ledgers may retain prior evidence. Documentation
consistency is required for delivery but does not replace build/runtime gates.

## OpenAI integration release gate

When OpenAI connector source changes, require `OPENAI_INTEGRATION_GATE_RESULT.md`, isolated mock/secret tests and the root Go 1.25 CI gates. Production packages must not be cut from a source tree where the root gate is BLOCKED even when provider mocks pass. Real external OpenAI calls are optional acceptance evidence and must not replace deterministic mocks.


## API-7.1 Object Locator Discovery checkpoint — 2026-09-24

API-7.1 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. It adds bounded `ObjectLocator` discovery from API-1 normalized path parameters, API-2 typed path/query/body evidence, matched API-3 OpenAPI contracts, and explicit Reviewer-gated operator INCLUDE/SUPPRESS configuration. Locators are keyed by normalized API-1 operation ID plus location/field; raw object values are transient only and durable value evidence is bounded HMAC-SHA256 fingerprint data protected by a separate mode-0600 key. Client-supplied ownership/tenant headers are not authority, and GraphQL `variables.*` discovery is deferred to API-8.

API-7.1 has no identity/object relationship verdict, tenant-boundary verdict, `BOLA_CANDIDATE`, ownership verdict, BLOCK/DENY/403 path, or ENFORCE authority. OpenAI is absent from the locator authority path. State is bounded by global/per-operation/fingerprint/override caps, 30-day learned TTL, versioned restart validation, existing non-blocking observation-plane ingestion, immediate mutation persistence, RBAC and audit.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83 PASS** from API-1/2 through API-7.1; API-7.1 has **9 targeted test functions present**; OpenAI source contract **16/16 PASS** plus isolated tests PASS; package-source PASS; root Go shape **107 files PASS**; package-builder **9/9 PASS**; changed-file gofmt, Admin JavaScript and primary shell syntax PASS. Canonical Go 1.25 targeted/race remains `BLOCKED_ENVIRONMENT/NOT_RUN` because the host has Go 1.23.2 and external toolchain retrieval is unavailable. Next slice: **API-7.2 Identity/Object Relationship**. API-7.3/API-7.4 is `IMPLEMENTED_TESTING_DEFERRED`; API-8 remains `PLANNED`.

## API-7.2 Identity/Object Relationship checkpoint — 2026-09-24

API-7.2 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. It correlates only API-5 cryptographically verified identity context with ACTIVE API-7.1 locator/object evidence. Identity, tenant and client dimensions are persisted only as API-7.2 HMAC-SHA256 pseudonyms protected by a dedicated mode-0600 key; object values remain API-7.1 keyed fingerprints. Raw JWTs, subjects, tenant/client claim values, cookies, caller-supplied owner/tenant headers and raw object values are not durable relationship state.

The relationship plane is bounded and asynchronous: non-blocking queue capacity 2,048; 8,192 total relationships; 512 relationships per verified identity pseudonym; 256 verified identities per keyed object/locator pair; 30-day TTL; bounded path/query/body capture; restart revalidation; autosave/final flush; shutdown drain; Reviewer-gated read-only evidence/status endpoints with audit. SUPPRESSed API-7.1 locators cannot create relationships. GraphQL `variables.*` remains deferred to API-8.

API-7.2 is evidence only. `OBSERVED`/`REPEATED` means recurrence, not ownership. There is no `BOLA_CANDIDATE`, ownership verdict, tenant-boundary verdict, BLOCK/DENY/403 or ENFORCE authority, and OpenAI is absent from this authority path.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83/100 PASS** from API-1/2 through API-7.2; API-7.2 has **9 targeted Go test functions present**; OpenAI source contract **16/16 PASS** plus isolated OpenAI/secretref tests PASS; package-source PASS; root Go source shape **109 files PASS**; package-builder **9/9 PASS**; changed Go files `gofmt` PASS. Canonical Go 1.25 targeted/race remains `BLOCKED_ENVIRONMENT/NOT_RUN` because this host has Go 1.23.2 and external toolchain retrieval is unavailable. Next slice: **API-7.3 BOLA Detection**; API-7.4 is `IMPLEMENTED_TESTING_DEFERRED`; API-8 remains `PLANNED`.



## API-7.3 BOLA Detection checkpoint — 2026-09-24

API-7.3 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. The detector consumes only API-5 cryptographically verified identity pseudonyms, ACTIVE/non-suppressed API-7.1 keyed object locators/fingerprints, and API-7.2 relationship history. It emits bounded evidence-only candidates for `IDENTITY_OBJECT_DIVERGENCE`, `TENANT_OBJECT_DIVERGENCE`, and `OBJECT_ENUMERATION`. A novel object alone is not a candidate; cross-identity and cross-tenant evidence requires repeated historical baseline, while enumeration requires 20 recent distinct keyed objects under the same locator within 10 minutes.

Candidate state persists only pseudonymous/keyed evidence, is capped at 4,096 total / 256 per identity / 128 per object-locator, expires after seven days, is versioned/revalidated on restart, and is saved by the API security autosave/final-flush path. Detection runs inside the API-7.2 background processing plane before current relationship merge. Reviewer-only candidate/status reads are audited. There is no ownership verdict, tenant-boundary verdict, policy mutation API, `BLOCK`, `DENY`, request-path `403`, or `ENFORCE` authority; OpenAI is absent from the detector authority path.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83/100/110 PASS** from API-1/2 through API-7.3; API-7.3 has **9 targeted Go test functions present**; OpenAI source contract **16/16 PASS** plus isolated tests PASS; package-source PASS; root Go source shape **111 files PASS**; package-builder **9/9 PASS**; changed Go `gofmt`, Admin embedded JavaScript and primary shell/Python syntax PASS. Canonical Go 1.25 `go mod tidy -diff`, root build, API-7.3 targeted tests and API-7.3 race tests remain `BLOCKED_ENVIRONMENT/NOT_RUN` because this host has Go 1.23.2 and external toolchain retrieval is unavailable. Next slice: **API-7.4 BOLA Policy/Evidence/Console**; API-8 remains `PLANNED`.

## API-7.4 BOLA Policy/Evidence/Console checkpoint — 2026-09-24

API-7.4 is implemented and remains `IMPLEMENTED_TESTING_DEFERRED`. It completes the API-7 BOLA implementation track with a bounded evidence-handling policy plane and operator workflow. Policies are scoped only by normalized API-1 operation ID, optional API-7.1 locator ID, optional API-7.3 candidate type, and minimum confidence. The only actions are `REVIEW` and `SUPPRESS`; suppression never deletes the underlying API-7.3 candidate. Operator evidence workflow is `OPEN` / `ACKNOWLEDGED` / `DISMISSED` / `RESOLVED` with enumerated reason codes only. New detector evidence after dismissal/resolution reopens the evidence automatically.

API-7.4 is deliberately absent from the API-7.2 relationship processor and API-7.3 detector authority path. It accepts no raw identity/object/tenant/client selectors, arbitrary headers, cookies, Authorization data, or free-text review notes. There is no ownership verdict, request-path `BLOCK`, `DENY`, `403`, or `ENFORCE` authority, and OpenAI is absent from the API-7.4 authority path. State is versioned/revalidated on restart, mutex-protected, TTL/cardinality bounded (1,024 policies; 4,096 reviews; 30-day default policy/review TTL; 180-day policy maximum), immediately persisted on mutation, and included in API security autosave/final flush. Reviewer-only policy/evidence APIs are audited and the embedded console exposes effective policy, evidence workflow, suppression, and reopen status.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83/100/110/156 PASS** from API-1/2 through API-7.4; **10 API-7.4 targeted Go test functions are present**; OpenAI source contract **16/16 PASS** plus isolated package tests PASS; package-source PASS; root Go source shape **113 files PASS**; package-builder **9/9 PASS**; `gofmt`, Admin embedded JavaScript and primary shell/Python syntax PASS. Canonical Go 1.25 `go mod tidy -diff`, root build/test, API-7.4 targeted tests and API-7.4 race tests remain `BLOCKED_ENVIRONMENT/NOT_RUN` because this host has Go 1.23.2 and external toolchain retrieval is unavailable. API-8 GraphQL Security is the next `PLANNED` slice.

## API-8 GraphQL Security checkpoint — 2026-09-24

API-8 is implemented across parsing/normalization, SDL contract binding, deterministic complexity/field/mutation/subscription/introspection policy, Apollo persisted-query support, verified GraphQL-variable integration into API-7 evidence, and Reviewer-gated `LEARN` / `DETECT` / `ENFORCE` policy plus Admin Console. Any policy edit or bound contract content change returns the policy to `LEARN`; a referenced schema contract cannot be deleted. Only explicit deterministic GraphQL policy in `ENFORCE` may reject traffic. API-6/API-7 learned or inferred evidence and OpenAI cannot promote, mutate or bypass GraphQL enforcement.

Durable GraphQL state excludes raw query documents, literals, raw variable values, Authorization/Cookie/JWT data and unverified claims. SDL is reduced to a digest plus normalized topology; GraphQL object values use API-7 keyed fingerprints and API-5 verified identity only. State and parser resources are bounded, versioned, restart-revalidated, autosaved/final-flushed, and strict mutation APIs are Reviewer-gated and audited.

Executed exact-source evidence: API gates **69/47/46/72/33/56/45/58/83/100/110/156/259 PASS** through API-8; **22 targeted API-8 Go test functions are present**; OpenAI source contract **16/16 PASS** plus isolated tests PASS; package-source PASS; package-builder **9/9 PASS**; root Go source shape **115 files PASS**; changed API-8 Go files `gofmt`, Admin embedded JavaScript, relevant shell and Python syntax PASS. Canonical Go 1.25 `go mod tidy -diff`, root build/vet/test, API-8 targeted tests and race tests remain `BLOCKED_ENVIRONMENT/NOT_RUN` because the host has Go 1.23.2 and external toolchain retrieval is unavailable. API-8 remains `IMPLEMENTED_TESTING_DEFERRED`, not `TESTED` or `RELEASED`.

The documented API-security implementation roadmap is now complete through API-8. No API-9 is defined. Next gate: canonical Go 1.25 qualification of the exact source/artifact, followed by release promotion only if all required gates pass.


## API-8 post-audit hardening release gate

A release candidate containing this hardening must pass `tools/tests/test-api8-post-audit-hardening-source.py` and clean-extract checks proving the removed `web/` and model-only foundation files are absent, the SYSTEM/traffic/API lifecycle Console surfaces are present, and source/manifests/modes match the packaged archive. Dependency-free gates do not replace the mandatory pinned Go 1.25 build/vet/test/race and production qualification requirements.

## 2026-09-25 — Production Correctness & Control-Plane Hardening

Current implementation baseline adds the post-audit production/control-plane hardening described in `PRODUCTION_CONTROL_PLANE_HARDENING.md`. Status remains **IMPLEMENTED_TESTING_DEFERRED**. Exact-source source gates pass through API-8 plus post-audit and production-hardening gates (`190/190` for the new hardening gate), but canonical Go 1.25 build/test/race is still BLOCKED_ENVIRONMENT / NOT_RUN. No API-9 is defined and API-6/API-7 learned/inferred signals retain no direct enforcement authority. The source artifact has no `.git` metadata, so source identity is the parent artifact SHA/manifests rather than Git branch/commit provenance.

## 2026-09-28 — Additional release gate

Release candidates derived from the duplication-consolidated source must run
`tools/tests/test-code-duplication-review-source.py` on both source and clean
extract. The gate proves removed superseded layers stay absent, replacement
authorities remain wired, Admin routes remain unique, and API authority
boundaries are unchanged. It is additive to, not a replacement for, pinned Go
1.25 and target-host qualification.

## 2026-09-30 — Build and Admin Startup Blocker Fix

The current source fixes two release-blocking defects found during external Go 1.25 review of the 2026-09-28 consolidation baseline. `admin.go` now imports the standard-library `errors` package required by the existing `errors.New(...)` validation paths. Positive-Schema exception state changes now use the unambiguous route `POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle`; the shipping Console uses the same path. The prior `POST /api/security/schema/enforcement/exceptions/{exception_id}` registration is removed because it conflicts with `POST /api/security/schema/enforcement/{operation_id}/mode` under Go 1.22+ `http.ServeMux` specificity rules and can panic during Admin handler construction.

`admin_route_registration_test.go` now makes full Admin handler registration a regression test, and `tools/tests/test-build-startup-blocker-fix-source.py` is wired into `build.sh` and CI. The source gate also registers the complete 139-route Admin/update inventory with the Go standard library in a standalone temporary program; this catches wildcard-pattern ambiguity even when the repository module graph cannot be compiled locally. Current source evidence: blocker-fix gate **13/13 PASS**, route inventory **139/139 accepted by `ServeMux`**, API/source gates through API-8/post-audit/production-hardening/duplication remain PASS, OpenAI source contract and isolated tests PASS, package-source PASS, package-builder **9/9 PASS**, root Go source shape **96 files PASS**, Admin embedded JavaScript PASS, and changed Go files are `gofmt` clean.

This does **not** promote the source to `TESTED` or `RELEASED`. On the current execution host, canonical `GOTOOLCHAIN=local` Go 1.25 `go mod tidy -diff`, root build, vet, targeted Admin/API64 tests, and race remain `BLOCKED_ENVIRONMENT / NOT_RUN` because only Go 1.23.2 is installed and external Go toolchain/module retrieval is unavailable. Packaging/source-manifest integrity is not a substitute for those root compilation gates. No API-security authority changes are introduced: API-6/API-7 remain non-enforcing, API-8 retains explicit deterministic GraphQL ENFORCE authority, and OpenAI remains advisory only.

Release gate clarification: `SOURCE_BASELINE_GATE_RESULT` and archive integrity prove source/package completeness only. Release/buildability requires the exact artifact bytes to pass the pinned Go 1.25 root tidy/build/vet/test gates. The new Admin route-registration regression and blocker source gate are mandatory pre-compilation gates but are not substitutes for compilation.

<!-- documentation-review: 2026-09-28; classification: current/canonical; current-authority: DOCUMENTATION_INDEX.md -->
