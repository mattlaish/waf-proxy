# API-6.3 Source Gate Result

Date: 2026-09-24  
Slice: API-6.3 Sequence Anomaly Detection  
Status: `IMPLEMENTED_TESTING_DEFERRED`

`python3 tools/tests/test-api63-source.py`: **PASS — 45 checks**.

Retained prerequisite source gates:

- API-1/API-2: **69 PASS**;
- API-3: **47 PASS**;
- API-4: **46 PASS**;
- API-5: **72 PASS**;
- API-6.1: **33 PASS**;
- API-6.2: **56 PASS**.

Additional executed evidence:

- `tools/tests/test-go-root-shape.sh`: **PASS — 103 root Go files**;
- OpenAI integration source contract: **16/16 PASS**;
- package-builder Python tests: **9/9 PASS**;
- package source gate: **PASS**;
- changed sequence/admin Go files checked with `gofmt -l`: **PASS**;
- embedded Admin Console JavaScript checked by `node --check`: **PASS**.

Canonical Go 1.25 targeted and race commands were attempted with `GOTOOLCHAIN=local` and stopped at the required toolchain boundary because the host is Go 1.23.2 while `go.mod` requires Go 1.25.0. Their status is **`BLOCKED_ENVIRONMENT / NOT_RUN`**, not PASS. A local downgraded-toolchain compile was also unable to proceed because required modules are not locally available and `GOPROXY=off` was used to prevent network substitution. This auxiliary attempt is not qualification evidence.

The API-6.3 source gate verifies all six anomaly classes, mature/sample/session/age/confidence safeguards, exception evaluation, bounded TTL evidence, v2-to-v3 additive persistence compatibility, privacy-preserving normalized/keyed evidence, Reviewer RBAC/audit visibility, absence of sequence enforcement/OpenAI request-path authority, retained prior source gates and seven targeted API-6.3 Go tests.

No result in this file upgrades API-6.3 to `TESTED` or `RELEASED`.

<!-- documentation-review: 2026-09-28; classification: historical evidence; current-authority: DOCUMENTATION_INDEX.md; historical-evidence-preserved: true -->
