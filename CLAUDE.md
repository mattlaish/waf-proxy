# Claude Project Instructions

At the beginning of work:

1. Read `DOCUMENTATION_INDEX.md` for current project/release truth.
2. Read `README.md` for the product.
3. Read `AGENTS.md` and follow its permanent development rules.
4. Read `AI_HANDOFF.md` and `HANDOVER_STATUS.md` for current state and next gate.

Keep changes focused and run the relevant validation. Never infer PASS from a
missing toolchain or environment. In particular, the root source is not
buildable/PASS until the exact bytes pass the Go 1.25 tidy and root-build gate.

Update the canonical documentation set after meaningful work:
`AI_HANDOFF.md`, `DEVELOPMENT.md`, `DEVELOPMENT_ROADMAP.md`, `TESTING.md`,
`TESTING_RESULTS.md`, `MANIFEST.md`, and `patch.md`.

For local/terminal sessions, the user may handle Git operations manually. For
cloud/web sessions, use an isolated branch and pull request. Never commit or
merge development work directly to `main` merely to obtain CI evidence.
