package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

// Well-known OAuth2 client credentials for Application Default Credentials.
// These are the same public credentials used by the gcloud CLI and are
// documented in Google's source code.
const (
	adcClientID     = "764086051850-6qr4p6gpi6hn506pt8ejuq83di341hur.apps.googleusercontent.com"
	adcClientSecret = "d-FL95Q19q7MQmFpd7hHD0Ty"
)

var adcScopes = []string{
	"https://www.googleapis.com/auth/cloud-platform",
	"https://www.googleapis.com/auth/userinfo.email",
	"openid",
}

// adcCredentials is the JSON format for Application Default Credentials files.
type adcCredentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"refresh_token"`
	Type         string `json:"type"`
}

func NewLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Google Cloud",
		Long: `Set up Application Default Credentials (ADC) for accessing Firestore.

This command runs an OAuth2 login flow in your browser and stores credentials
locally. No external tools (like gcloud) are required.

After logging in, set your project with:
  export PROJECT_ID=my-project
or pass it with the --project flag.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Start a local HTTP server on a random port to receive the OAuth callback.
			listener, err := net.Listen("tcp", "localhost:0")
			if err != nil {
				return fmt.Errorf("failed to start local server: %w", err)
			}
			defer listener.Close()

			port := listener.Addr().(*net.TCPAddr).Port
			redirectURL := fmt.Sprintf("http://localhost:%d", port)

			config := &oauth2.Config{
				ClientID:     adcClientID,
				ClientSecret: adcClientSecret,
				Endpoint: oauth2.Endpoint{
					AuthURL:  "https://accounts.google.com/o/oauth2/auth",
					TokenURL: "https://oauth2.googleapis.com/token",
				},
				RedirectURL: redirectURL,
				Scopes:      adcScopes,
			}

			state, err := generateState()
			if err != nil {
				return fmt.Errorf("failed to generate state: %w", err)
			}

			codeCh := make(chan string, 1)
			errCh := make(chan error, 1)

			mux := http.NewServeMux()
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				q := r.URL.Query()
				// Ignore requests without OAuth parameters (e.g., favicon).
				if q.Get("code") == "" && q.Get("error") == "" {
					return
				}

				if q.Get("state") != state {
					errCh <- fmt.Errorf("state mismatch in OAuth callback")
					http.Error(w, "State mismatch", http.StatusBadRequest)
					return
				}
				if errMsg := q.Get("error"); errMsg != "" {
					errCh <- fmt.Errorf("authorization denied: %s", errMsg)
					fmt.Fprintf(w, "<html><body><h2>Authorization failed</h2><p>%s</p><p>You can close this window.</p></body></html>", errMsg)
					return
				}

				code := q.Get("code")
				fmt.Fprint(w, "<html><body><h2>Authentication successful!</h2><p>You can close this window and return to the terminal.</p></body></html>")
				codeCh <- code
			})

			server := &http.Server{Handler: mux}
			go server.Serve(listener)
			defer server.Shutdown(context.Background())

			authURL := config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent"))

			fmt.Println("Your browser will open to authenticate with Google Cloud.")
			fmt.Println()

			if err := openBrowser(authURL); err != nil {
				fmt.Println("Could not open browser automatically. Please visit this URL:")
				fmt.Println()
				fmt.Println("  " + authURL)
				fmt.Println()
			}

			fmt.Println("Waiting for authentication...")

			var code string
			select {
			case code = <-codeCh:
			case err := <-errCh:
				return err
			case <-time.After(5 * time.Minute):
				return fmt.Errorf("authentication timed out after 5 minutes")
			}

			token, err := config.Exchange(cmd.Context(), code)
			if err != nil {
				return fmt.Errorf("token exchange failed: %w", err)
			}

			if err := saveADCCredentials(token.RefreshToken); err != nil {
				return err
			}

			fmt.Println()
			fmt.Println("Authentication successful!")
			fmt.Printf("Credentials saved to: %s\n", wellKnownADCPath())
			fmt.Println()
			fmt.Println("Make sure to set your project:")
			fmt.Println("  export PROJECT_ID=my-project")
			fmt.Println("  # or use: firestore -p my-project <command>")

			return nil
		},
	}
}

func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("cmd", "/c", "start", url).Start()
	default:
		return fmt.Errorf("unsupported platform")
	}
}

func saveADCCredentials(refreshToken string) error {
	creds := adcCredentials{
		ClientID:     adcClientID,
		ClientSecret: adcClientSecret,
		RefreshToken: refreshToken,
		Type:         "authorized_user",
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	path := wellKnownADCPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("failed to create credentials directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write credentials: %w", err)
	}

	return nil
}
