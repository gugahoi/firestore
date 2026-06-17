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

### Examples

```bash
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
- `Ctrl+g`: Open path input to jump to a specific path
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
