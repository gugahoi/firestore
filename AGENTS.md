# Agent Instructions

## Build / Test / Run

- Build: `go build -o firestore .`
- Test all: `go test ./...`
- Test single package: `go test ./pkg/cmd/document`
- Race detection: `go test -race ./...`
- Format: `go fmt ./...`
- Vet: `go vet ./...`
- Tidy: `go mod tidy`

## Integration Tests

Integration tests are gated behind the `integration` Go build tag — `go test ./...` excludes them entirely.

- Run: `make integration-test` (wraps `scripts/run-integration-tests.sh`)
- Override port: `PORT=<port> make integration-test`
- Pass extra go test args: `make integration-test ARGS="-run TestFoo"`
- Default port: `8765`

**Prerequisites:**
- `gcloud` SDK installed
- `cloud-firestore-emulator` gcloud component: `gcloud components install cloud-firestore-emulator`
- **Java 21+** — Java 17 is too old and will fail (already broke CI once; the integration-test GitHub workflow pins `java-version: '21'`)

**Verify build-tag exclusion:**
```sh
# Without tag — integration files absent:
go list -f '{{.TestGoFiles}}' ./pkg/cmd/browse/
# With tag — integration files present:
go list -tags integration -f '{{.TestGoFiles}}' ./pkg/cmd/browse/
```

**CI:**
- Integration tests: `.github/workflows/integration-test.yml`
- Regular tests + goreleaser check: `.github/workflows/test.yml`

## Code Style

- Standard `gofmt` formatting
- Package names: lowercase single word (`document`, `collection`)
- Struct names: PascalCase (`OrderBy`, `Filter`)
- Public symbols: PascalCase; private: camelCase
- Error wrapping: `fmt.Errorf("message: %w", err)`
- Imports grouped: stdlib → third-party → local
- CLI commands: `cobra.Command` with proper `Args` validation
- Context: pass via `cmd.Context()` through all call chains
- Path normalization: `strings.TrimPrefix(path, "/")`
- JSON output: `json.NewEncoder` with `SetEscapeHTML(false)` and `SetIndent`
- Client retrieval: `cmd.Context().Value(keys.ClientKey).(*firestore.Client)`

## Workflow Conventions

- **Grill first:** For feature requests, stress-test the idea before writing a plan — ask sharp clarifying/challenge questions about scope, edge cases, and overwrite behavior. Use `/grill-with-docs` or ask directly.
- **Standard delivery loop:** new branch off `main` → implement → verify (`go build` + `go test ./...`, plus `make integration-test` when touching browse/emulator code) → commit → open PR via `gh` → watch CI → merge when green.
- **Small, focused PRs:** split unrelated changes into separate branches/PRs.
- **Commit messages:** Conventional Commits with a scope — e.g. `feat(browse): copy documents in TUI`, `fix(browse): hide preview pane when active column is a document`, `chore(test): gate integration tests behind build tag`.
- **Project commands:** `/ship` runs the delivery loop end-to-end; `/handoff` drafts a new-branch agent prompt to hand a slice of work to a fresh agent.
- **Repo layout:** checked out as git worktrees from a bare repo; branches like `feat/duplicate-command` are sibling directories under the bare repo root.
