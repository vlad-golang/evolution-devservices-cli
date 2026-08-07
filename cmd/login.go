package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cloud-ru/evolution-devservices-cli/internal/config"
)

// newLoginCmd creates the `eds login` command.
func newLoginCmd() *cobra.Command {
	var (
		repoAPIKey  string
		repoAPIURL  string
		project     string
		repoGitHost string
		wfKeyID     string
		wfSecret    string
		wfAPIURL    string
		iamURL      string
		fromStdin   bool
	)

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Save credentials to the local config",
		Long: `login stores credentials and (optionally) API URLs, project id
and git host in the user's local config (~/.config/eds/config.json
by default, configurable via EDS_CONFIG or XDG_CONFIG_HOME).

The Repo product credential (--repo-api-key) and the Workflow Studio
credentials (--wf-key-id/--wf-secret) are independent; provide either or
both. Workflow Studio key id + secret are exchanged for a short-lived
Bearer access token via the cloud.ru IAM service on first use; the CLI
caches that token in the config file and refreshes it automatically once
it expires.

It is safe to call this from a script: the file is written with 0600
permissions.`,
		Example: `  # interactive (you'll be prompted for the key)
  eds login --project 3232b2d0-1063-41e6-b2fa-13df767f4a0a

  # non-interactive (preferred for automation / agents)
  eds login --repo-api-key $EDS_REPO_API_KEY --project <project-id>

  # dev environment
  eds login --repo-api-url https://devtools.dev.api.internal.cloud.ru/repo/api/v1 \
             --repo-api-key <KEY> --project <project-id>

  # add Workflow Studio access (separate credential pair)
  eds login --wf-key-id "$WF_KEY_ID" --wf-secret "$WF_SECRET" \
             --project <project-id>`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			// Read from stdin if requested.
			if fromStdin {
				if _, err := fmt.Scanln(&repoAPIKey); err != nil {
					return fmt.Errorf("read api key from stdin: %w", err)
				}
			}

			if repoAPIKey == "" && wfKeyID == "" && wfSecret == "" {
				return fmt.Errorf("at least one credential is required: --repo-api-key/--stdin or --wf-key-id/--wf-secret")
			}
			if repoAPIKey != "" {
				cfg.APIKey = repoAPIKey
			}
			if wfKeyID != "" {
				cfg.WorkflowKeyID = wfKeyID
			}
			if wfSecret != "" {
				cfg.WorkflowSecret = wfSecret
			}
			if wfKeyID != "" || wfSecret != "" {
				// Credentials changed: drop any cached token from the old pair.
				cfg.WorkflowAccessToken = ""
				cfg.WorkflowTokenExpiresAt = ""
				cfg.WorkflowAccessTokenKeyID = ""
			}

			if repoAPIURL != "" {
				cfg.APIURL = repoAPIURL
			}
			if project != "" {
				cfg.ProjectID = project
			}
			if repoGitHost != "" {
				cfg.GitHost = repoGitHost
			}
			if wfAPIURL != "" {
				cfg.WorkflowAPIURL = wfAPIURL
			}
			if iamURL != "" {
				cfg.IAMURL = iamURL
			}

			if err := cfg.Save(); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Credentials saved.")
			return nil
		},
	}

	cmd.Flags().StringVar(&repoAPIKey, "repo-api-key", "", "Repo product API key (X-API-KEY)")
	cmd.Flags().StringVar(&repoAPIURL, "repo-api-url", "", "Repo product API base URL")
	cmd.Flags().StringVar(&project, "project", "", "default project id (shared across products)")
	cmd.Flags().StringVar(&repoGitHost, "repo-git-host", "", "Repo product git smart-HTTP host")
	cmd.Flags().StringVar(&wfKeyID, "wf-key-id", "", "Workflow Studio key id (separate credential from --repo-api-key)")
	cmd.Flags().StringVar(&wfSecret, "wf-secret", "", "Workflow Studio secret")
	cmd.Flags().StringVar(&wfAPIURL, "wf-api-url", "", "Workflow Studio API base URL")
	cmd.Flags().StringVar(&iamURL, "iam-url", "", "cloud.ru IAM token endpoint (shared)")
	cmd.Flags().BoolVar(&fromStdin, "stdin", false, "read the Repo API key from stdin")

	return cmd
}
