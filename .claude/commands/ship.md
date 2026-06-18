---
description: Commit, push, open a PR, watch CI, and merge when green
argument-hint: "[optional PR title / notes]"
---

Ship the current changes: build, test, commit, push, open a PR, watch CI, and merge when green.
Use `$ARGUMENTS` as optional title context or notes for the PR message.

**Guardrail:** only auto-merge if the user explicitly said "watch CI and merge if it passes" (or equivalent). Otherwise stop after CI is green and ask before merging.

---

### Step 1 — Pre-flight check

1. Run `git status` and `git diff --stat` so the user sees exactly what will ship. Summarize the changed files.
2. Run `go build -o firestore .` — stop and report if it fails.
3. Run `go test ./...` — stop and report if any tests fail.
4. If the diff touches `pkg/cmd/browse` or any emulator/integration test files, also run `make integration-test`. Stop and report if it fails. (This requires gcloud, the `cloud-firestore-emulator` component, and Java 21+; warn the user if the environment looks wrong rather than silently skipping.)
5. Do **not** proceed past this step if any check fails.

---

### Step 2 — Branch guard

If currently on `main`, create a descriptive feature branch before committing:

```
git checkout -b <scope>/<short-description>
```

Derive the scope and description from the staged diff. Never push directly to `main`.

---

### Step 3 — Commit

Stage relevant changed files (be explicit — avoid `git add -A` if it would sweep in unrelated files or secrets).

Write a Conventional Commits message with a scope, e.g.:

```
feat(browse): copy documents in TUI
fix(browse): hide preview pane when active column is a document
chore(test): gate integration tests behind build tag
```

Derive the type (`feat`, `fix`, `refactor`, `chore`, `docs`, `test`), scope (usually the package or area, e.g. `browse`, `cmd`, `test`), and summary from the diff. Incorporate `$ARGUMENTS` if the user supplied title/notes.

Use a HEREDOC to pass the message:

```bash
git commit -m "$(cat <<'EOF'
<type>(<scope>): <summary>
EOF
)"
```

---

### Step 4 — Push

```bash
git push -u origin HEAD
```

---

### Step 5 — Open PR

```bash
gh pr create \
  --base main \
  --title "<type>(<scope>): <summary>" \
  --body "$(cat <<'EOF'
## Summary
- <bullet points from the diff / $ARGUMENTS>

## Test plan
- `go build -o firestore .` passes
- `go test ./...` passes
- [ ] Manual smoke test in TUI (if browse/UI changes)
EOF
)"
```

Fill the title and body from the commit message and `$ARGUMENTS`. Keep the PR small and focused.

---

### Step 6 — Watch CI

Poll until all checks finish:

```bash
gh pr checks <PR-NUMBER> --watch
```

If `--watch` is unavailable, loop `gh pr checks <PR-NUMBER>` every 30 seconds until all checks show `pass` or at least one shows `fail`.

---

### Step 7 — Merge decision

- If **any check fails**: STOP. Summarize the failing job name and logs. Do **not** merge. Ask the user how to proceed.
- If **all checks pass**:
  - If the user asked for auto-merge ("watch CI and merge if it passes"): run `gh pr merge <PR-NUMBER> --squash --delete-branch` and report the result.
  - Otherwise: report that CI is green and ask "All checks passed — shall I merge with squash?"

---

### Step 8 — Report

Output the PR URL and final status (merged / green-awaiting-confirmation / failed).
