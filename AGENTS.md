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
`AI_HANDOFF.md`, or `patch.md`, end the final user feedback with the current
Taiwan time in this exact format: `YYYY-MM-DD HH:mm:ss UTC+8 (Taiwan)`.

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
