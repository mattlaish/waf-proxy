# New Chat Handover Prompt

Use the following as the canonical bootstrap prompt for a new development chat.
It supersedes the older date-stacked continuation notes previously kept in this
file.

```text
Continue the waf-proxy project from the 2026-09-17 root-build-integrity repair baseline.

Read, in order:
1. DOCUMENTATION_INDEX.md
2. AGENTS.md
3. AI_HANDOFF.md
4. HANDOVER_STATUS.md
5. DEVELOPMENT_ROADMAP.md
6. TESTING.md and TESTING_RESULTS.md
7. SOURCE_BASELINE_GATE_RESULT.md
8. RELEASE_PROCESS.md

Critical truth:
- Audited GitHub main baseline: 1d52d65a73a802e32f02994e51f0a07beb240177.
- main was found non-buildable because of go.mod drift and half-applied PKI/debug/TLS patches.
- A repair is implemented, but it is NOT TESTED on the required Go 1.25 toolchain in the current environment.
- Source Buildability Gate is BLOCKED, not PASS.
- Do not use archive integrity, package fixtures, ABI stubs, or source-level checks as a substitute for root compilation.
- Do not commit development patches directly to main. Use a PR and require CI before merge.
- Branch protection/ruleset requiring CI is REQUIRED but was not applied by the previous session because GitHub integration writes returned 403.

Mandatory next execution on a Go 1.25 CI/release host, against the exact repair commit:
GOTOOLCHAIN=local go mod tidy -diff
GOTOOLCHAIN=local CGO_ENABLED=0 go build ./...
GOTOOLCHAIN=local CGO_ENABLED=0 go vet ./...
GOTOOLCHAIN=local CGO_ENABLED=0 go test ./...
GOTOOLCHAIN=local CGO_ENABLED=1 go test -race ./...
GOTOOLCHAIN=local CGO_ENABLED=1 go test -tags realcoraza -run 'TestRealCoraza' ./...

If those pass:
- mark the source buildability evidence PASS for that exact commit;
- merge only through the PR after required CI;
- enable branch protection/rulesets requiring the CI build-test check;
- use ./waf-package on qualified hosts to build real DEB/RPM packages;
- then execute Slice C package lifecycle and Slice D clean-host distro gates.

Architecture invariants:
- Coraza v3.7.0 is authoritative.
- VectorScan/libhs is optional acceleration and must fail safe to Coraza.
- PKCS#11/HSM is optional and fail-closed; software-token evidence cannot qualify a real vendor HSM.
- No PostgreSQL backend currently exists; persistence is filesystem-based.
- DEB is the formal Debian/Ubuntu path; RPM is the formal RHEL/Rocky/Alma/Oracle path.
- Package installs must remain offline-safe and must not fetch CRS.
- RHEL-family clean-host PASS requires SELinux Enforcing.
- static/admin.html is the shipping console; web/ is experimental.

Keep DEVELOPMENT.md, AI_HANDOFF.md, DEVELOPMENT_ROADMAP.md, TESTING.md,
TESTING_RESULTS.md, MANIFEST.md, and patch.md synchronized with every meaningful
change. Preserve PASS/FAIL/BLOCKED/NOT_RUN truth exactly.
```

OpenAI connector delta:
- Responses API + Structured Outputs + api_key_ref + mock integration tests are IMPLEMENTED_TESTING_DEFERRED.
- isolated provider tests PASS; root Go 1.25 integration remains BLOCKED.
- read OPENAI_INTEGRATION_GATE_RESULT.md before modifying AI provider behavior.
