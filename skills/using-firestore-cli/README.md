# Firestore CLI

Firestore is a command line utility to facilitate operations with Firestore from the command line.

## Installation

### Prebuilt binary (recommended)

Download the archive for your platform from the [latest release](https://github.com/gugahoi/firestore/releases/latest). Archives are named `firestore_<OS>_<arch>` (e.g. `firestore_Darwin_arm64.tar.gz`, `firestore_Linux_x86_64.tar.gz`, `firestore_Windows_x86_64.zip`).

```bash
# macOS / Linux example — adjust the URL for your OS and architecture
curl -L -o firestore.tar.gz https://github.com/gugahoi/firestore/releases/latest/download/firestore_Darwin_arm64.tar.gz
tar -xzf firestore.tar.gz firestore
sudo mv firestore /usr/local/bin/
firestore --help
```

On Windows, extract the `.zip` and move `firestore.exe` to a directory on your `PATH`.

### With `go install`

Requires Go 1.24 or newer.

```bash
go install github.com/gugahoi/firestore@latest
```

This places the `firestore` binary in `$(go env GOBIN)` (or `$(go env GOPATH)/bin` if `GOBIN` is unset). Make sure that directory is on your `PATH`.

### From source

Requires Go 1.24 or newer.

```bash
git clone https://github.com/gugahoi/firestore.git
cd firestore
go build -o firestore .
./firestore --help
```

Move the resulting binary somewhere on your `PATH` (e.g. `sudo mv firestore /usr/local/bin/`) to use it from anywhere.

## Usage

### Authentication

Log in with your Google account:

```bash
firestore auth login
```

This opens a browser-based OAuth2 flow and saves Application Default Credentials locally. No external tools (like `gcloud`) are required.

You can check your authentication status or revoke credentials at any time:

```bash
firestore auth status
firestore auth revoke
```

Alternatively, you can set `GOOGLE_APPLICATION_CREDENTIALS` to point to a service account key file.

### Project

Set the project via the `--project` flag or the `PROJECT_ID` environment variable:

```bash
export PROJECT_ID=my-project
# or
firestore -p my-project <command>
```

### Command reference

| Command | Alias | Args | Notes |
|---|---|---|---|
| `firestore auth login` / `status` / `revoke` | — | 0 | gcloud-free ADC; 5-min login timeout |
| `firestore browse [path]` | — | 0–1 | vim-style Miller-column TUI (see [Interactive Browser](#interactive-browser-tui)) |
| `firestore document add <path>` | — | 1 | `Create` — reads JSON from STDIN |
| `firestore document get <path>` | — | 1 | pretty JSON (HTML-escape off, 2-space indent) |
| `firestore document copy <src> <dst>` | `cp` | 2 | full `Set` overwrite |
| `firestore document delete <path>` | `rm` | 1 | |
| `firestore document edit <path>` | `e` | 1 | `--set field=value` (repeatable, dotted paths ok) OR interactive `$EDITOR` |
| `firestore document list <path>` | `ls` | 1 | lists subcollections of a doc |
| `firestore document move <src> <dst>` | `mv` | 2 | copy+delete — refuses if src has subcollections |
| `firestore collection list <col>` | `ls` | 1 | lists doc IDs + create times |
| `firestore collection query <col>` | `q` | 1 | `-s/-d/-f/-l/-q/--show-id`; operators `== < > <= >=` |
| `firestore collection copy <src> <dst>` | `cp` | 2 | recursive incl. subcollections; aggregates per-doc errors |
| `firestore collection delete <col>` | `rm` | 1 | BulkWriter in batches of 10 |
| `firestore collection download <col>` | `dl` | 1 | `-o/-f/-l/-s/-d`; one `<id>.json` per doc |
| `firestore collection upload <col> <folder>` | `up` | 2 | every `*.json` in folder; basename = doc ID |
| `firestore api <path>` | — | 1 | authenticated REST escape hatch (see [REST escape hatch](#rest-escape-hatch-firestore-api)) |

Notes on `collection query`:

- `-q/--fields <csv>` is a server-side field mask (return only those fields), not a query alias. The alias for `collection query` itself is `q`.
- `-l 0` means no limit (not "zero rows") on `collection query` and `collection download`.
- `--show-id` defaults to `true`; the doc ID prints even with a field mask. Pass `--show-id=false` to suppress it.

### Examples

```bash
# Copy a document
firestore document cp /my-collection/my-document /my-collection/another-document

# Partial update without replacing the doc
firestore document edit /users/abc --set age=42 --set address.city=NYC
echo '{"name":"ada"}' | firestore document add /users/new

# Server-side query with sort and field mask
firestore -p demo-flux collection query -s firstName --direction desc -f firstName==Vince -f lastName==Petersen -l 50 -q firstName,lastName /data

# Recursive copy between collections (incl. subcollections)
firestore -p demo collection cp /staging/orders /live/orders

# Round-trip export / import through a folder of JSON files
firestore -p demo-flux collection download /users -o ./out --limit 100 -s createdAt -d desc
# edit ./out/*.json ...
firestore -p demo-flux collection upload /seed ./out
```

### `document edit --set` value typing

`--set` parses each value with auto type detection:

| Input | Parsed as |
|---|---|
| `true` / `false` | bool |
| `42` / `3.14` | number |
| `[1, 2, 3]` | JSON array |
| `'{"a":1}'` (quoted) | JSON object / string |
| anything else | bare string |

Dotted paths (e.g. `address.city`) become Firestore field paths; the parent map is created if missing.

### REST escape hatch (`firestore api`)

`firestore api` is an authenticated escape hatch to the raw Firestore REST API, similar in spirit to `gh api`:

```bash
# Relative path → expanded to projects/<proj>/databases/(default)/documents/<path>
firestore api users/u1
firestore api 'users?documentId=u1' -X POST --input -    # POST on -i (create)
firestore api users/u1 -X DELETE

# Absolute path (starting with projects/) sent verbatim — admin, :runQuery, named DBs
firestore api 'projects/{project}/databases/{database}/documents:runQuery' -X POST --input query.json
```

- `{project}` and `{database}` placeholders substituted in either form.
- Method defaults to GET, or POST when `--input` is given.
- JSON responses are pretty-printed; non-JSON passes through raw.
- Under the Firestore emulator, `api` sends `Authorization: Bearer owner` instead of an ADC token, so no real credentials are required to hit a local emulator.

## Supported Features

### Documents

- [x] delete
- [x] move
- [x] copy
- [x] download
- [x] add

### Collections

- [x] copy
- [x] delete
- [x] list
- [x] download
- [x] upload
- [x] query

### Interactive Browser (TUI)

The `browse` command launches an interactive terminal user interface for navigating your Firestore database. It uses Miller columns and is vim-style, with three modes: **NORMAL** (default), **VISUAL** (multi-select + bulk delete), and **COMMAND** (`:` prefix, `Tab` autocompletes command names).

```bash
# Start browsing from the root
firestore browse

# Start browsing a specific collection or document
firestore browse users
firestore browse users/abc123/preferences
```

Path segments alternate: odd segments = a collection, even = a document.

The full keybinding reference is also available inside the TUI via `:man`.

#### NORMAL mode

**Navigation**
- `j` / `↓`, `k` / `↑` — move cursor
- `g` — top of column; `G` — bottom
- `l` / `→` / `Enter` — navigate forward (open collection / doc)
- `h` / `←` / `Bksp` — navigate back (close column / collapse node)
- `Space` — toggle expand/collapse tree node
- `Ctrl+d` / `PgDn`, `Ctrl+u` / `PgUp` — half-page scroll
- `Ctrl+g` — go to path (dialog)
- `/` — filter current column (substring)
- `Ctrl+o` — jump back in history; `Ctrl+i` — jump forward

**Marks** — `m[a-z]` set, `'[a-z]` jump, `:marks` list.

**Tree folding** — `zM` all in, `zR` all out, `z1` / `z2` / `z3` fold to depth.

**Document ops** — `e` edit (`$EDITOR`), `d` delete (with confirmation), `R` rename / move (copy-then-delete), `c` copy to a new id (blank = auto-generate), `r` refresh, `p` toggle preview pane.

**Clipboard** — `yy` copy selected value (field value or path), `ya` copy whole doc as JSON, `Y` copy ID or field key.

**Sort** — `s` open sort dialog (`Tab` switch focus, `Ctrl+d` toggle direction, `Enter` apply, `Esc` cancel); `S` clear sort.

#### VISUAL mode
- `v` — toggle item selection / enter visual
- `V` — range select (anchor + move)
- `j` / `k` — move and extend
- `d` — bulk delete selected
- `Esc` — exit visual

#### COMMAND mode — `:` prefix, Tab completion
```
:help                :man                :quit               :goto <path>
:refresh             :sort <f> [asc|desc]  :query <f> <op> <v>   :set limit <n>
:add [id]            :rename <path>      :mv <path>          :copy [id|path]
:cp [id|path]        :export json|ndjson [file]  :marks
```

`:query` operators: `==  !=  <  >  <=  >=  array-contains  in`.

`:rename` / `:copy` target resolution: a bare name → same collection; a path containing `/` → absolute path from root. `:copy` with no arg → auto-generated id.

#### Quitting
- `q` — quit
- `:quit` — quit
- `Ctrl+c` — quit (in command mode `Ctrl+c` cancels instead)
- `Esc` — shows "Press q to quit" hint

### Editor resolution (for `document edit` and TUI `e`)

`$EDITOR` → `$VISUAL` → `vim` / `vi` / `nano` fallback.

## Firestore Emulator

To use this tool with the Firestore Emulator, you can either set the `FIRESTORE_EMULATOR_HOST` environment variable or use the `--host` flag:

```bash
# Using environment variable
export FIRESTORE_EMULATOR_HOST=localhost:9090
firestore document get /my-collection/my-document

# Using --host flag
firestore --host localhost:9090 document get /my-collection/my-document
```

The `--host` flag will override the environment variable if both are set. When `firestore api` runs against the emulator it sends `Authorization: Bearer owner` instead of an ADC token, so no real credentials are required to hit a local emulator (see [REST escape hatch](#rest-escape-hatch-firestore-api)).

## Running Integration Tests

Integration tests live in `pkg/cmd/browse/rename_integration_test.go` and are gated by the `integration` build tag, so they are excluded from the normal `go test ./...` run.

**Prerequisites:** `gcloud` CLI with the `cloud-firestore-emulator` component and Java installed.

```bash
# Install the emulator component once
gcloud components install cloud-firestore-emulator

# Run all integration tests (boots the emulator automatically on port 8765)
make integration-test

# Use a different port
make integration-test PORT=9000
```

The `make integration-test` target starts the emulator in the background, exports `FIRESTORE_EMULATOR_HOST`, runs `go test -tags integration -v ./...`, and tears the emulator down on exit — even if the tests fail.
