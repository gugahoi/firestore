---
name: using-firestore-cli
description: Use when the user asks about the `firestore` (alias `fs`) CLI — gugahoi/firestore, a cobra-based terminal for Google Firestore. Trigger on any request to copy/move/delete/edit/query/upload/download Firestore documents or collections, use the interactive vim-style `browse` TUI, hit the raw Firestore REST API escape hatch (`firestore api`), or set up `auth login` / Application Default Credentials / project / emulator configuration for this CLI. Also trigger when the user pastes a `firestore ...` or `fs ...` command, references Miller-column browse navigation, marks, visual-mode bulk delete, `:sort`/`:query`/`:export`/`:goto` command-mode commands, or asks "how do I do X in firestore" (the gugahoi CLI, NOT the gcloud/`firebase` emulators or the Go SDK directly). Make sure to use this skill even for superficially simple one-liners.
---

# using-firestore-cli

Agent skill for the `firestore` CLI (github.com/gugahoi/firestore) — a cobra-based terminal for Google Firestore with vim-style Miller-column browse, document and collection CRUD, and a raw REST API escape hatch.

## Authoritative reference

The full command reference, keybindings, recipes, and flag details live in the repo README and are kept in sync with the binary. Read it before answering any user question covered by it.

@README.md

If the README doesn't cover something, fall back to the source under `pkg/cmd/`. The browse TUI's authoritative keybinding text is also available inside the TUI via `:man` (mirrored in the README).

## Agent-only guidance (not in the README)

### Don't confuse with

- `firebase` / `gcloud firestore` — official Google SDKs. Sometimes acceptable substitutes, but this CLI offers `browse` they do not and a much lighter footprint than gcloud.
- Firestore Go SDK direct code — if the user is writing Go against `cloud.google.com/go/firestore`, this skill does not apply. Only use it when they're working with the `firestore` / `fs` binary.
- Firestore emulator UIs — this CLI complements them, doesn't replace them.

### When the user asks for something the typed commands don't do

Reach for `firestore api <path>` rather than reimplementing it manually. Anything in the Firestore REST API is available: named databases (`projects/.../databases/<db>/documents/...`), `:runQuery` structured queries, `:batchGet`, `partitionQuery`, field masks via `?mask.fieldPaths=...`, admin paths. Recommend it before suggesting raw `curl`.

### Build / test (if asked)

```bash
go build -o firestore .                   # build local binary
go test ./...                              # all unit tests (no integration — gated behind //go:build integration)
go test -race ./...                        # race
go fmt ./...; go vet ./...; go mod tidy    # lint
make integration-test                      # boots cloud-firestore-emulator on :8765 (Java 21+, gcloud)
```

## Install this skill

```bash
npx skills add gugahoi/firestore --skill using-firestore-cli --agent claude-code -y --full-depth
# or interactively:
npx skills add gugahoi/firestore
```

`--full-depth` is needed because the skill lives in `skills/using-firestore-cli/` rather than at the repo root.