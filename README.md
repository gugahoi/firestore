# Firestore CLI

Firestore is a command line utility to facilitate operations with Firestore from the command line.

## Usage

Currently both `PROJECT_ID` and `GOOGLE_APPLICATION_CREDENTIALS` are required.

```
export PROJECT_ID=my-project
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/credentials.json

# copying a document
firestore document cp /my-collection/my-document /my-collection/another-document

# querying documents in a collection
firestore -p demo-flux collection query --sort firstName --direction desc -f firstName==Vince -f lastName=Petersen /data
```

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

The `browse` command launches an interactive terminal user interface for navigating your Firestore database.

```bash
# Start browsing from the root
firestore browse

# Start browsing a specific collection or document
firestore browse users
firestore browse users/abc123/preferences
```

**Navigation Controls:**
- `j` / `k` (or `Up`/`Down`): Move selection up and down
- `l` / `h` (or `Enter`/`Left`): Navigate forward (into a collection/doc) or backward
- `Ctrl+d` / `Ctrl+u` (or `PgDn`/`PgUp`): Scroll document content or long lists
- `g` / `G`: Jump to the top or bottom of the list
- `r`: Refresh the current column
- `q`: Quit the browser

## Firestore Emulator

To use this tool with the Firestore Emulator, you can either set the `FIRESTORE_EMULATOR_HOST` environment variable or use the `--host` flag:

```bash
# Using environment variable
export FIRESTORE_EMULATOR_HOST=localhost:9090
firestore document get /my-collection/my-document

# Using --host flag
firestore --host localhost:9090 document get /my-collection/my-document
```

The `--host` flag will override the environment variable if both are set.
