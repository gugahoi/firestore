package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/spf13/cobra"
)

func NewRevokeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "revoke",
		Short: "Revoke Application Default Credentials",
		Long:  `Revoke the locally stored Application Default Credentials and remove the credential file.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			adcPath := wellKnownADCPath()

			data, err := os.ReadFile(adcPath)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("No credentials to revoke.")
					return nil
				}
				return fmt.Errorf("failed to read credentials: %w", err)
			}

			// Extract the refresh token so we can revoke it server-side.
			var creds struct {
				RefreshToken string `json:"refresh_token"`
			}
			if err := json.Unmarshal(data, &creds); err == nil && creds.RefreshToken != "" {
				resp, err := http.PostForm("https://oauth2.googleapis.com/revoke", url.Values{
					"token": {creds.RefreshToken},
				})
				if err != nil {
					fmt.Printf("Warning: could not reach revocation endpoint: %v\n", err)
				} else {
					resp.Body.Close()
					if resp.StatusCode != http.StatusOK {
						fmt.Printf("Warning: token revocation returned status %d (token may already be expired)\n", resp.StatusCode)
					}
				}
			}

			if err := os.Remove(adcPath); err != nil {
				return fmt.Errorf("failed to remove credentials file: %w", err)
			}

			fmt.Println("Credentials revoked.")
			return nil
		},
	}
}
