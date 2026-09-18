## Git

AI is allowed to perform normal Git operations:

- git pull
- git add
- git commit
- git push

Before starting work:

- Synchronize with the latest repository state.
- Inspect git status and recent commits.
- Confirm the current branch and remote repository.
- Treat GitHub as the source of truth.

Before pushing:

- Run relevant tests and validation checks.
- Update AI_HANDOFF.md with completed work, verification results, and next recommended step.
- Review git diff and git status.
- Ensure no unrelated files, secrets, credentials, generated binaries, or temporary files are included.

After completing a development stage that updates `README.md`,
`AI_HANDOFF.md`, or `patch.md`, end the final user feedback with the current Taiwan time in this exact format:
`UTM+8: YYYY-MM-DD HH:MM:SS`.

Rules:

- Prefer small meaningful commits.
- Do not overwrite unrelated changes.
- Do not force-push unless explicitly instructed.
- Do not rewrite history without approval.
- Do not delete branches or modify remotes without approval.
- Do not push another AI agent's unreviewed local work.

## Documentation handover

- Keep `AI_HANDOFF.md` and `patch.md` synchronized with the current `main`.
- Update `patch.md` for every source, schema, API, UI, test, or build change.
- Record the current base commit, validation evidence, security decisions,
  incomplete work, and exact next slice.

## Main-branch build integrity

- Do not commit development patches directly to `main`; use a pull request.
- `main` should have branch protection/rulesets requiring the CI build-test check before merge.
- Before describing a source baseline as complete/buildable/PASS, the exact source bytes must pass `GOTOOLCHAIN=local go mod tidy -diff` and `GOTOOLCHAIN=local CGO_ENABLED=0 go build ./...` under the pinned Go version.
- Archive generation, extraction, manifest parity, or packaging tests are not substitutes for root compilation.

## Documentation authority

- Start with `DOCUMENTATION_INDEX.md` for current truth.
- `AI_HANDOFF.md` and `HANDOVER_STATUS.md` define the current continuation point.
- `DEVELOPMENT.md` and `TESTING_RESULTS.md` are append-only historical ledgers;
  older entries do not override the current canonical status block.
- Package/component gate files are scoped evidence only.
- Keep `README.md`, `INSTALL.md`, `DEVELOPMENT.md`, `DEVELOPMENT_ROADMAP.md`,
  `AI_HANDOFF.md`, `HANDOVER_STATUS.md`, `HANDOVER_PROMPT.md`, `TESTING.md`,
  `TESTING_RESULTS.md`, `MANIFEST.md`, `RELEASE_PROCESS.md`, and `patch.md`
  synchronized after meaningful release/build/packaging changes.
- Never use stale "next slice" text from a historical section when a newer
  canonical handover or roadmap entry exists.
