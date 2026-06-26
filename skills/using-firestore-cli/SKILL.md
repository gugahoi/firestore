---
name: using-firestore-cli
description: Use when the user asks about the `firestore` (alias `fs`) CLI — gugahoi/firestore, a cobra-based terminal for Google Firestore. Trigger on any request to copy/move/delete/edit/query/upload/download Firestore documents or collections, use the interactive vim-style `browse` TUI, hit the raw Firestore REST API escape hatch (`firestore api`), or set up `auth login` / Application Default Credentials / project / emulator configuration for this CLI. Also trigger when the user pastes a `firestore ...` or `fs ...` command, references Miller-column browse navigation, marks, visual-mode bulk delete, `:sort`/`:query`/`:export`/`:goto` command-mode commands, or asks "how do I do X in firestore" (the gugahoi CLI, NOT the gcloud/`firebase` emulators or the Go SDK directly). Make sure to use this skill even for superficially simple one-liners, because the README's browse keybinding table is stale — this skill is the authoritative reference.
---

# using-firestore-cli

## What this CLI is

`firestore` (cobra binary, alias `fs`) — a Go terminal for Google Firestore from `github.com/gugahoi/firestore`. Top-level commands: `auth`, `browse`, `document` (`doc`), `collection` (`col`), `api`. Plus cobra's `completion` and `help`.

Everything except `auth` and `completion` requires a project (`-p <project>` flag or `PROJECT_ID` env var). A live `*firestore.Client` is built in `PersistentPreRunE` before any subcommand runs and stashed in `cmd.Context()` under `keys.ClientKey` — so the user never sees that plumbing; they just set `PROJECT_ID` and run commands.

## Why this skill exists

1. README's browse keybinding table is **out of date** — it omits visual mode, marks, tree folding, preview toggle, rename, copy, filter, jumplist, and the entire `:` command-mode system. The authoritative reference is `:man` inside the TUI, mirrored in the **Browse keybindings** section below.
2. The CLI has sharp edges worth flagging up front: `document mv` refuses docs with subcollections, `api` swaps real ADC for `Bearer owner` when an emulator is set, `-q/--fields` is a CSV mask on `collection query` (not the query alias `q`), `document edit --set` does auto type-coercion.
3. Credential setup is gcloud-free — `firestore auth login` runs an OAuth2 flow with the well-known gcloud ADC client ID. But it shares the same ADC file as gcloud, so they don't conflict.

## First-run setup

```bash
firestore auth login                                    # browser OAuth2 → ~/.config/gcloud/application_default_credentials.json
firestore auth status                                   # reports ADC type/account/quota-project
# OR set GOOGLE_APPLICATION_CREDENTIALS=<service-account-key.json>

export PROJECT_ID=my-project                           # OR per-command: firestore -p my-project ...
# emulator (optional):
export FIRESTORE_EMULATOR_HOST=localhost:9090           # OR: firestore --host localhost:9090 ...
```

`auth` deliberately overrides the root `PersistentPreRunE` so `firestore auth login` works *before* any creds exist. `auth revoke` revokes the refresh token server-side and deletes the ADC file.

## Command map (quick reference)

| Command | Alias | Args | Notes |
|---|---|---|---|
| `firestore auth login` / `status` / `revoke` | — | 0 | gcloud-free ADC; `<5m` timeout |
| `firestore browse [path]` | — | 0–1 | vim-style Miller-column TUI (see **Browse keybindings**) |
| `firestore document add <path>` | — | 1 | `Create` — **reads JSON from STDIN** |
| `firestore document get <path>` | — | 1 | pretty JSON (HTML-escape off, 2-space indent) |
| `firestore document copy <src> <dst>` | `cp` | 2 | full `Set` overwrite |
| `firestore document delete <path>` | `rm` | 1 | |
| `firestore document edit <path>` | `e` | 1 | `--set field=value` (repeatable, dotted-paths ok) OR interactive `$EDITOR` |
| `firestore document list <path>` | `ls` | 1 | lists subcollections of a doc |
| `firestore document move <src> <dst>` | `mv` | 2 | copy+delete — **refuses if src has subcollections** |
| `firestore collection list <col>` | `ls` | 1 | lists doc IDs + create-times |
| `firestore collection query <col>` | `q` | 1 | `-s/-d/-f/-l/-q/--show-id`; operators `== < > <= >=` |
| `firestore collection copy <src> <dst>` | `cp` | 2 | recursive incl. subcollections; aggregates per-doc errors |
| `firestore collection delete <col>` | `rm` | 1 | BulkWriter batches of 10 |
| `firestore collection download <col>` | `dl` | 1 | `-o/-f/-l/-s/-d`; one `<id>.json` per doc |
| `firestore collection upload <col> <folder>` | `up` | 2 | every `*.json` in folder; basename = doc ID |
| `firestore api <path>` | — | 1 | authenticated REST escape hatch (see **REST escape hatch**) |

> **Gotcha:** `collection query`'s `-q/--fields <csv>` is a server-side field mask (return only those fields), not a query alias — the alias for `collection query` itself is `q`. Easy to confuse.
>
> **Flags worth knowing:** `-l 0` means **no limit** (not "zero rows") on `collection query` and `collection download`. `--show-id` defaults to **true** — the doc ID prints even when you set `-q firstName,lastName`. Pass `--show-id=false` to suppress it.

## Common recipes

```bash
# Round-trip export / import to a folder of JSON files
firestore -p demo-flux collection download /users -o ./out --limit 100 -s createdAt -d desc
# edit ./out/*.json ...
firestore -p demo-flux collection upload /seed ./out

# Partial update without replacing the doc
firestore document edit /users/abc --set age=42 --set address.city=NYC
echo '{"name":"ada"}' | firestore document add /users/new

# Server-side query with sort and field mask
firestore -p demo-flux collection query \
    -s firstName --direction desc \
    -f firstName==Vince -f lastName==Petersen \
    -l 50 -q firstName,lastName \
    /data

# Recursive copy between collections (incl. subcollections) — staging → live
firestore -p demo collection cp /staging/orders /live/orders

# Bulk-delete a whole collection
firestore -p demo collection rm /staging/orders

# Visual-mode bulk delete from the TUI (browse, then v/V, then d)
firestore browse /staging/orders
```

## `document edit --set` value typing

`--set` parses each value with auto type detection:

| Input | Parsed as |
|---|---|
| `true` / `false` | bool |
| `42` / `3.14` | number |
| `[1, 2, 3]` | JSON array |
| `'{"a":1}'` (quoted) | JSON object / string |
| anything else | bare string |

Dotted paths (`address.city`) become Firestore field paths; the parent map is created if missing.

## REST escape hatch — `firestore api`

```bash
# Relative path → expanded to projects/<proj>/databases/(default)/documents/<path>
firestore api users/u1
firestore api 'users?documentId=u1' -X POST --input -    # POST on -i (create)
firestore api users/u1 -X DELETE

# Absolute path (starting with projects/) sent verbatim — admin, :runQuery, named DBs
firestore api 'projects/{project}/databases/{database}/documents:runQuery' -X POST --input query.json
```

- `{project}` and `{database}` placeholders substituted in either form.
- Method defaults to GET, or POST when `--input` given.
- JSON responses are pretty-printed; non-JSON passes through raw.
- **Emulator swap:** when `FIRESTORE_EMULATOR_HOST` is set, `api` sends `Authorization: Bearer owner` instead of an ADC token — you do not need real credentials to hit a local emulator.

## Browse keybindings (authoritative — see `:man` in the TUI)

The TUI is launched with `firestore browse [path]` (path optional; odd segments = collection, even = document). Uses Miller columns. Three modes: **NORMAL** (default), **VISUAL** (multi-select + bulk delete), **COMMAND** (`:` prefix, `Tab` autocompletes command names).

### NORMAL mode

**Navigation**
- `j`/`↓`, `k`/`↑` — move cursor
- `g` — top of column; `G` — bottom
- `l`/`→`/`Enter` — forward (open collection / doc)
- `h`/`←`/`Bksp` — back (close column / collapse node)
- `Space` — toggle expand/collapse tree node
- `Ctrl+d`/`PgDn`, `Ctrl+u`/`PgUp` — half-page scroll
- `Ctrl+g` — go to path (dialog)
- `/` — filter current column (substring)
- `Ctrl+o` — jump back in history; `Ctrl+i` — jump forward

**Marks** — `m[a-z]` set, `'[a-z]` jump, `:marks` list.

**Tree folding** — `zM` all in, `zR` all out, `z1`/`z2`/`z3` fold to depth.

**Document ops** — `e` edit (`$EDITOR`), `d` delete (with confirmation), `R` rename/move (copy-then-delete), `c` copy to new id (blank = auto-gen), `r` refresh, `p` toggle preview pane.

**Clipboard** — `yy` copy selected value (field value or path), `ya` copy whole doc as JSON, `Y` copy ID or field key.

**Sort** — `s` open sort dialog (`Tab` switch focus, `Ctrl+d` toggle direction, `Enter` apply, `Esc` cancel); `S` clear sort.

### VISUAL mode
- `v` — toggle item selection / enter visual
- `V` — range select (anchor + move)
- `j`/`k` — move and extend
- `d` — bulk delete selected
- `Esc` — exit visual

### COMMAND mode — `:` prefix, Tab completion
```
:help                :man                :quit               :goto <path>
:refresh             :sort <f> [asc|desc]  :query <f> <op> <v>   :set limit <n>
:add [id]            :rename <path>      :mv <path>          :copy [id|path]
:cp [id|path]        :export json|ndjson [file]  :marks
```

`:query` operators: `==  !=  <  >  <=  >=  array-contains  in`.

`:rename`/`:copy` target resolution: a bare name → same collection; a path containing `/` → absolute path from root. `:copy` with no arg → auto-generated id.

### Quitting
`q`, `:quit`, `Ctrl+c` (in command mode `Ctrl+c` cancels instead). `Esc` shows "Press q to quit" hint.

## Editor resolution (for `document edit` and browse `e`)

`$EDITOR` → `$VISUAL` → `vim`/`vi`/`nano` fallback. The browse package re-implements `detectEditor` deliberately to keep Bubble Tea out of the lightweight document package.

## Build / test (if asked)

```bash
go build -o firestore .                   # build local binary
go test ./...                              # all unit tests (no integration — gated behind //go:build integration)
go test -race ./...                        # race
go fmt ./...; go vet ./...; go mod tidy    # lint
make integration-test                      # boots cloud-firestore-emulator on :8765 (Java 21+, gcloud)
```

## When the user asks for something the typed commands don't do

Reach for `firestore api <path>` rather than reimplementing it manually. Anything in the Firestore REST API is available: named databases (`projects/.../databases/<db>/documents/...`), `:runQuery` structured queries, `:batchGet`, `partitionQuery`, field masks via `?mask.fieldPaths=...`, admin paths. This is the documented escape hatch — recommend it before suggesting raw `curl`.

## Don't confuse with

- `firebase` / `gcloud firestore` — official Google SDKs; sometimes acceptable substitutes, but this CLI offers `browse` they do not, and a much lighter footprint than gcloud.
- Firestore Go SDK direct code — if the user is writing Go against `cloud.google.com/go/firestore`, this skill does **not** apply; only use it when they're working with the `firestore`/`fs` binary.
- Firestore emulator UIs — this CLI complements them, doesn't replace them.

## Publishability

This skill is hosted at `skills/using-firestore-cli/SKILL.md` inside the `gugahoi/firestore` repo. Install with:

```bash
npx skills add gugahoi/firestore --skill using-firestore-cli --agent claude-code -y --full-depth
# or interactively:
npx skills add gugahoi/firestore
```

`--full-depth` is needed because the skill lives in a subdirectory rather than at the repo root.