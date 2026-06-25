package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/gugahoi/firestore/pkg/cmd/keys"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2/google"
)

const (
	defaultDatabase = "(default)"
	liveBaseURL     = "https://firestore.googleapis.com/v1/"
	datastoreScope  = "https://www.googleapis.com/auth/datastore"
)

func NewApiCmd() *cobra.Command {
	var method, input string

	cmd := &cobra.Command{
		Use:   "api <path>",
		Short: "make an authenticated request to the Firestore REST API",
		Long: `Make an authenticated request to the Firestore REST API.

This is an escape hatch for interactions the CLI does not wrap yet (named
databases, :runQuery, :batchGet, field masks, admin endpoints, ...). It works
like "gh api": you give a path, it makes the request with the project and
credentials you already have configured.

Path resolution:
  - A relative path is expanded under the documents collection of the current
    project, i.e. "users/u1" becomes
      projects/<project>/databases/(default)/documents/users/u1
  - A path starting with "projects/" is sent verbatim (after the /v1/ prefix),
    for admin, query and other non-document endpoints.

Placeholders (substituted in either form):
  - {project}  -> the --project (-p) / PROJECT_ID value
  - {database} -> (default)

Method:
  - Defaults to GET, or POST when a body is supplied with --input.
  - Override with -X/--method.

Body:
  - --input <file> reads the request body from a file.
  - --input -      reads the request body from STDIN.
  - A Content-Type of application/json is set automatically when a body is sent.

Authentication:
  - Uses Application Default Credentials (run 'firestore auth login' first).
  - When --host (or FIRESTORE_EMULATOR_HOST) points at an emulator, requests go
    to that host and no real credentials are required.`,
		Example: `  # Get a document (relative path)
  firestore api users/u1

  # Create a document from STDIN
  echo '{"fields":{"name":{"stringValue":"ada"}}}' | firestore api 'users?documentId=u1' -X POST --input -

  # Run a structured query (verbatim path with placeholders)
  firestore api 'projects/{project}/databases/{database}/documents:runQuery' -X POST --input query.json

  # Delete a document
  firestore api users/u1 -X DELETE`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID, _ := cmd.Context().Value(keys.ProjectIDKey).(string)
			return apiCall(cmd.Context(), projectID, args[0], method, input)
		},
	}

	cmd.Flags().StringVarP(&method, "method", "X", "", "HTTP method (default GET, or POST when --input is given)")
	cmd.Flags().StringVar(&input, "input", "", "request body: a file path, or '-' to read STDIN")

	return cmd
}

func apiCall(ctx context.Context, projectID, path, method, input string) error {
	body, err := readBody(input)
	if err != nil {
		return err
	}

	base, emulator := baseURL()
	url := base + resolvePath(projectID, defaultDatabase, path)

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, defaultMethod(method, body != nil), url, reader)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if emulator {
		req.Header.Set("Authorization", "Bearer owner")
	} else {
		ts, err := google.DefaultTokenSource(ctx, datastoreScope)
		if err != nil {
			return fmt.Errorf("failed to find credentials (run 'firestore auth login'): %w", err)
		}
		tok, err := ts.Token()
		if err != nil {
			return fmt.Errorf("failed to get auth token: %w", err)
		}
		tok.SetAuthHeader(req)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}

	printResponse(respBody)
	return nil
}

// resolvePath expands {project}/{database} placeholders and turns a relative
// path into a fully-qualified documents path. Paths already starting with
// "projects/" are returned verbatim.
func resolvePath(projectID, database, path string) string {
	path = strings.TrimPrefix(path, "/")
	path = strings.ReplaceAll(path, "{project}", projectID)
	path = strings.ReplaceAll(path, "{database}", database)
	if strings.HasPrefix(path, "projects/") {
		return path
	}
	return fmt.Sprintf("projects/%s/databases/%s/documents/%s", projectID, database, path)
}

// defaultMethod returns the explicit method (upper-cased) if given, otherwise
// POST when a body is present and GET otherwise.
func defaultMethod(method string, hasBody bool) string {
	if method != "" {
		return strings.ToUpper(method)
	}
	if hasBody {
		return http.MethodPost
	}
	return http.MethodGet
}

func baseURL() (string, bool) {
	if h := os.Getenv("FIRESTORE_EMULATOR_HOST"); h != "" {
		return "http://" + h + "/v1/", true
	}
	return liveBaseURL, false
}

func readBody(input string) ([]byte, error) {
	switch input {
	case "":
		return nil, nil
	case "-":
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read STDIN: %w", err)
		}
		return b, nil
	default:
		b, err := os.ReadFile(input)
		if err != nil {
			return nil, fmt.Errorf("failed to read input file: %w", err)
		}
		return b, nil
	}
}

// printResponse pretty-prints JSON, falling back to raw output for non-JSON.
func printResponse(b []byte) {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		os.Stdout.Write(b)
		if len(b) > 0 && b[len(b)-1] != '\n' {
			fmt.Println()
		}
		return
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}
