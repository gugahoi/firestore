---
description: Draft a ready-to-paste prompt to hand a scoped task to a new agent on a new branch
argument-hint: "<task description>"
---

Produce a single, copy-pasteable prompt block that a fresh agent can execute with no prior context.
Take the task from `$ARGUMENTS`. If `$ARGUMENTS` is empty, ask the user: "What task do you want to hand off?"

**Do NOT start implementing the task.** This command only produces the handoff prompt.

---

### Step 1 — Understand the task

Parse `$ARGUMENTS` to identify:
- The goal (what should be built or fixed)
- Which packages or files are likely involved (browse, cmd, Firestore client, TUI, tests, etc.)
- Whether it touches the TUI browse package or emulator/integration code (affects the test requirement)

---

### Step 2 — Gather repo context

Read just enough to make the prompt self-contained:
- Relevant file paths and package names (e.g. `pkg/cmd/browse/`, `main.go`, `Makefile`)
- Any existing types, functions, or patterns the new agent should follow or extend
- Whether an integration test is required for this change

Keep this lightweight — the goal is orientation for the new agent, not a full audit.

---

### Step 3 — Write the handoff prompt

Output **only** a fenced code block (` ```text `) containing the ready-to-paste prompt. Structure it as follows:

```
## Goal
<One-sentence description of what to build or fix.>

## Acceptance criteria
- <Concrete, testable bullet points — what does "done" look like?>

## Files / packages involved
- <List the specific files or packages to create or modify>

## Conventions
- Branch off `main`; the PR target is `main`. Create a new branch — never commit to main.
- Commit messages follow Conventional Commits with a scope:
    feat(browse): copy documents in TUI
    fix(browse): hide preview pane when active column is a document
    chore(test): gate integration tests behind build tag
- After making changes, verify with:
    go build -o firestore .
    go test ./...
<If the change touches pkg/cmd/browse or emulator/integration code, add:>
    make integration-test   # requires gcloud + cloud-firestore-emulator + Java 21+
- Open a PR with `gh pr create --base main` when done.

## Scope
Keep this a small, focused PR. Do not bundle unrelated changes.

## Additional context
<Any patterns, types, or implementation notes the new agent must know to avoid
re-inventing or diverging from existing conventions — e.g. how documents are
copied today, what TUI model fields exist, relevant key bindings, etc.>
```

Fill every section from the task description and the repo context gathered in Step 2. Be specific about file paths. Do not leave placeholder text.

---

### Guardrails

- Output only the fenced prompt block (plus a one-line intro like "Here is your handoff prompt:"). Do not add commentary after the block.
- Do not begin implementing the task.
- If the task is too vague to write concrete acceptance criteria, ask the user one clarifying question before proceeding.
