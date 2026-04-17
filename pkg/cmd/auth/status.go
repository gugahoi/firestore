package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

func NewStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check authentication status",
		Long:  `Check whether Application Default Credentials are configured and display the credential type and account info.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check GOOGLE_APPLICATION_CREDENTIALS env var first
			if sa := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); sa != "" {
				info, err := readCredentialFile(sa)
				if err != nil {
					return fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS is set to %s but cannot be read: %w", sa, err)
				}
				printCredentialInfo("service_account (via GOOGLE_APPLICATION_CREDENTIALS)", sa, info)
				return nil
			}

			// Check ADC well-known location
			adcPath := wellKnownADCPath()
			info, err := readCredentialFile(adcPath)
			if err != nil {
				fmt.Println("Not authenticated.")
				fmt.Println()
				fmt.Println("Run: firestore auth login")
				return nil
			}

			printCredentialInfo("application_default_credentials", adcPath, info)
			return nil
		},
	}
}

// wellKnownADCPath returns the path where gcloud stores ADC credentials.
func wellKnownADCPath() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("APPDATA"), "gcloud", "application_default_credentials.json")
	}
	return filepath.Join(os.Getenv("HOME"), ".config", "gcloud", "application_default_credentials.json")
}

// credentialInfo holds the fields we care about from a credential JSON file.
type credentialInfo struct {
	Type     string `json:"type"`
	ClientID string `json:"client_id"`
	Account  string `json:"client_email"`
	Project  string `json:"quota_project_id"`
}

func readCredentialFile(path string) (credentialInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return credentialInfo{}, err
	}
	var info credentialInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return credentialInfo{}, fmt.Errorf("invalid credential file: %w", err)
	}
	return info, nil
}

func printCredentialInfo(source, path string, info credentialInfo) {
	fmt.Printf("Authenticated (%s)\n", source)
	fmt.Printf("  Credential file: %s\n", path)
	if info.Type != "" {
		fmt.Printf("  Type:            %s\n", info.Type)
	}
	if info.Account != "" {
		fmt.Printf("  Account:         %s\n", info.Account)
	}
	if info.Project != "" {
		fmt.Printf("  Quota project:   %s\n", info.Project)
	}
}
