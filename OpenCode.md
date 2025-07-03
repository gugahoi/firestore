# OpenCode Configuration

## Build/Test Commands
- Build: `go build -o firestore .`
- Test all: `go test ./...`
- Test single package: `go test ./pkg/cmd/document`
- Run with race detection: `go test -race ./...`
- Format code: `go fmt ./...`
- Vet code: `go vet ./...`
- Mod tidy: `go mod tidy`

## Code Style Guidelines
- Use standard Go formatting (gofmt)
- Package names: lowercase, single word (e.g., `document`, `collection`)
- Struct names: PascalCase (e.g., `OrderBy`, `Filter`)
- Function names: camelCase for private, PascalCase for public
- Error handling: wrap errors with context using `fmt.Errorf("message: %w", err)`
- Imports: group standard library, third-party, then local packages
- Use cobra.Command for CLI commands with proper Args validation
- Context passing: use `cmd.Context()` and pass through function calls
- String trimming: use `strings.TrimPrefix(path, "/")` for path normalization
- JSON output: use `json.NewEncoder` with `SetEscapeHTML(false)` and `SetIndent`
- Type assertions: use `cmd.Context().Value(keys.ClientKey).(*firestore.Client)`