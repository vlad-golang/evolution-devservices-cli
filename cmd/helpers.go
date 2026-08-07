package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/cloud-ru/evolution-devservices-cli/internal/config"
	"github.com/cloud-ru/evolution-devservices-cli/internal/iam"
	"github.com/cloud-ru/evolution-devservices-cli/internal/output"
	"github.com/cloud-ru/evolution-devservices-cli/internal/repoapi"
	"github.com/cloud-ru/evolution-devservices-cli/internal/workflowapi"
)

// workflowTokenExpiryBuffer is subtracted from a fetched token's lifetime
// so a cached token is treated as expired slightly before the IAM service
// actually rejects it.
const workflowTokenExpiryBuffer = 30 * time.Second

// runtimeContext holds the resolved configuration, API clients and printer
// for a single command invocation.
type runtimeContext struct {
	Cfg         *config.Config
	API         *repoapi.Client
	WorkflowAPI *workflowapi.Client
	Printer     *output.Printer
	Quiet       bool
}

// resolveContext loads config, applies flag/env overrides and constructs
// the API client and printer.
func resolveContext(cmd *cobra.Command) (*runtimeContext, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	// Apply flag overrides on top of config (flags > env > file > defaults).
	if v, _ := cmd.Flags().GetString("project"); v != "" {
		cfg.ProjectID = v
	}
	if v, _ := cmd.Flags().GetString("iam-url"); v != "" {
		cfg.IAMURL = v
	}
	if v, _ := cmd.Flags().GetString("repo-api-url"); v != "" {
		cfg.APIURL = v
	}
	if v, _ := cmd.Flags().GetString("repo-api-key"); v != "" {
		cfg.APIKey = v
	}
	if v, _ := cmd.Flags().GetString("repo-git-host"); v != "" {
		cfg.GitHost = v
	}
	if v, _ := cmd.Flags().GetString("wf-api-url"); v != "" {
		cfg.WorkflowAPIURL = v
	}
	if v, _ := cmd.Flags().GetString("wf-key-id"); v != "" {
		cfg.WorkflowKeyID = v
	}
	if v, _ := cmd.Flags().GetString("wf-secret"); v != "" {
		cfg.WorkflowSecret = v
	}

	wantJSON, _ := cmd.Flags().GetBool("json")
	quiet, _ := cmd.Flags().GetBool("quiet")

	format := output.FormatAuto
	if wantJSON {
		format = output.FormatJSON
	}

	rt := &runtimeContext{
		Cfg:         cfg,
		API:         repoapi.New(cfg.APIURL, cfg.APIKey, cfg.ProjectID),
		WorkflowAPI: workflowapi.New(cfg.WorkflowAPIURL, cfg.WorkflowAccessToken, cfg.ProjectID),
		Printer:     output.New(format),
		Quiet:       quiet,
	}

	// TEMPORARY: some prod environments don't yet accept X-API-KEY for the
	// Repo product and require a Bearer token exchanged via the same IAM
	// flow as Workflow Studio. Remove this flag once Repo's own X-API-KEY
	// auth works everywhere.
	repoUseWFAuth, _ := cmd.Flags().GetBool("repo-use-wf-auth")
	if !repoUseWFAuth {
		v := os.Getenv("EDS_REPO_USE_WF_AUTH")
		repoUseWFAuth = v == "1" || v == "true"
	}
	if repoUseWFAuth {
		if err := rt.ensureWorkflowAuth(cmd.Context()); err != nil {
			return nil, fmt.Errorf("--repo-use-wf-auth: %w", err)
		}
		rt.API.SetBearerToken(rt.Cfg.WorkflowAccessToken)
	}

	return rt, nil
}

// requireAPIKey returns a friendly error if the Repo product API key is
// missing. Skipped when --repo-use-wf-auth switched the client to Bearer
// auth instead (see resolveContext).
func (r *runtimeContext) requireAPIKey() error {
	if r.API.UsesBearerAuth() {
		return nil
	}
	if r.Cfg.APIKey == "" {
		return fmt.Errorf("repo API key is not set. Run `eds login --repo-api-key <KEY>` or set EDS_REPO_API_KEY")
	}
	return nil
}

// ensureWorkflowAuth makes sure r.WorkflowAPI holds a valid Bearer access
// token before a Workflow Studio call. It reuses the cached token (from
// ~/.config/eds/config.json) while still valid, otherwise exchanges
// WorkflowKeyID/WorkflowSecret for a fresh one via the cloud.ru IAM service
// and caches the result to disk so subsequent commands can reuse it.
func (r *runtimeContext) ensureWorkflowAuth(ctx context.Context) error {
	cacheValid := r.Cfg.WorkflowAccessToken != "" &&
		r.Cfg.WorkflowAccessTokenKeyID == r.Cfg.WorkflowKeyID &&
		!workflowTokenExpired(r.Cfg.WorkflowTokenExpiresAt)
	if cacheValid {
		return nil
	}

	if r.Cfg.WorkflowKeyID == "" || r.Cfg.WorkflowSecret == "" {
		return fmt.Errorf("wf credentials are not set. Run `eds login --wf-key-id <ID> --wf-secret <SECRET>` or set EDS_WF_KEY_ID/EDS_WF_SECRET")
	}

	tok, err := iam.FetchToken(ctx, r.Cfg.IAMURL, r.Cfg.WorkflowKeyID, r.Cfg.WorkflowSecret)
	if err != nil {
		return fmt.Errorf("exchange workflow credentials for an access token: %w", err)
	}

	lifetime := time.Duration(tok.ExpiresIn) * time.Second
	if lifetime > workflowTokenExpiryBuffer {
		lifetime -= workflowTokenExpiryBuffer
	}
	expiresAt := time.Now().Add(lifetime).Format(time.RFC3339)

	r.Cfg.WorkflowAccessToken = tok.AccessToken
	r.Cfg.WorkflowTokenExpiresAt = expiresAt
	r.Cfg.WorkflowAccessTokenKeyID = r.Cfg.WorkflowKeyID
	r.WorkflowAPI.SetToken(tok.AccessToken)

	// Cache the token on disk (best-effort: a caching failure shouldn't
	// fail the command, since we already have a valid token in memory).
	// Reload from disk first so we don't persist any other in-memory
	// overrides that only apply to this invocation (e.g. one-off flags).
	if diskCfg, err := config.Load(); err == nil {
		diskCfg.WorkflowAccessToken = tok.AccessToken
		diskCfg.WorkflowTokenExpiresAt = expiresAt
		diskCfg.WorkflowAccessTokenKeyID = r.Cfg.WorkflowKeyID
		_ = diskCfg.Save()
	}
	return nil
}

func workflowTokenExpired(expiresAt string) bool {
	t, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return true
	}
	return !time.Now().Before(t)
}
