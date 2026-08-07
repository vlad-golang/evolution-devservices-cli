package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cloud-ru/evolution-devservices-cli/internal/output"
	"github.com/cloud-ru/evolution-devservices-cli/internal/workflowapi"
)

// newAppCmd creates the parent `eds wf app` command and all its subcommands.
// An "application" (Workflow Studio) wires a repository+branch to a deploy
// pipeline; creating a deployment for it runs that pipeline and publishes
// the result.
func newAppCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "app",
		Short: "Manage Workflow Studio applications (deployed services)",
	}
	cmd.AddCommand(newAppCreateCmd())
	cmd.AddCommand(newAppListCmd())
	cmd.AddCommand(newAppShowCmd())
	cmd.AddCommand(newAppUpdateCmd())
	cmd.AddCommand(newAppDeleteCmd())
	cmd.AddCommand(newAppDeployCmd())
	cmd.AddCommand(newAppDeploymentsCmd())
	cmd.AddCommand(newAppStatusCmd())
	return cmd
}

func newAppCreateCmd() *cobra.Command {
	var (
		repository    string
		repositoryURL string
		branch        string
	)

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a Workflow Studio application from a repository",
		Args:  cobra.ExactArgs(1),
		Long: `create wires a repository + branch to a deploy pipeline.

The repository can be an existing "eds repo" repository (--repository,
accepts either its id or its name) or an external git URL (--repository-url).
Use "eds wf app deploy" afterwards to actually run the pipeline and publish.`,
		Example: `  eds repo create my-site && eds wf app create my-site --repository my-site --branch main
  eds wf app create my-site --repository-url https://github.com/user/my-site --branch main`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := resolveContext(cmd)
			if err != nil {
				return err
			}
			if err := ctx.ensureWorkflowAuth(cmd.Context()); err != nil {
				return err
			}

			if repository == "" && repositoryURL == "" {
				return fmt.Errorf("either --repository or --repository-url is required")
			}

			var repositoryID string
			if repository != "" {
				if !looksLikeUUID(repository) {
					if err := ctx.requireAPIKey(); err != nil {
						return err
					}
				}
				repositoryID, err = resolveRepoID(cmd.Context(), ctx, repository)
				if err != nil {
					return err
				}
			}

			req := workflowapi.CreateApplicationRequest{
				Name:          args[0],
				Branch:        branch,
				RepositoryID:  repositoryID,
				RepositoryURL: repositoryURL,
			}
			app, err := ctx.WorkflowAPI.CreateApplication(cmd.Context(), req)
			if err != nil {
				return err
			}

			if !ctx.Quiet {
				fmt.Fprintf(cmd.OutOrStdout(), "Created application %q (id=%s)\n", app.Name, app.ID)
			}
			return ctx.Printer.PrintJSON(app)
		},
	}

	cmd.Flags().StringVar(&repository, "repository", "", "id or name of an existing `eds repo` repository")
	cmd.Flags().StringVar(&repositoryURL, "repository-url", "", "external git repository URL (alternative to --repository)")
	cmd.Flags().StringVar(&branch, "branch", "main", "branch to deploy from")
	return cmd
}

func newAppListCmd() *cobra.Command {
	var (
		search string
		sort   string
		limit  int
		offset int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List Workflow Studio applications in the configured project",
		Example: `  eds wf app list
  eds wf app list --search my-site
  eds wf app list --json | jq '.applications[].name'`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, err := resolveContext(cmd)
			if err != nil {
				return err
			}
			if err := ctx.ensureWorkflowAuth(cmd.Context()); err != nil {
				return err
			}

			resp, err := ctx.WorkflowAPI.ListApplications(cmd.Context(), workflowapi.ListApplicationsOptions{
				Search: search,
				Sort:   workflowapi.SortOrder(sort),
				Limit:  limit,
				Offset: offset,
			})
			if err != nil {
				return err
			}

			if ctx.Printer.Format == output.FormatJSON {
				return ctx.Printer.PrintJSON(resp)
			}

			headers := []string{"ID", "NAME", "BRANCH", "STATUS", "UPDATED"}
			rows := make([][]string, 0, len(resp.Applications))
			for _, a := range resp.Applications {
				rows = append(rows, []string{a.ID, a.Name, a.Branch, a.Status, output.HumanTime(a.UpdatedAt)})
			}
			ctx.Printer.Table(headers, rows)
			fmt.Fprintf(cmd.OutOrStdout(), "Showing %d of %d\n", len(resp.Applications), resp.Total)
			return nil
		},
	}

	cmd.Flags().StringVar(&search, "search", "", "search term (default: all)")
	cmd.Flags().StringVar(&sort, "sort", "created_at_desc", "sort order: created_at_asc, created_at_desc")
	cmd.Flags().IntVar(&limit, "limit", 50, "page size")
	cmd.Flags().IntVar(&offset, "offset", 0, "offset")
	return cmd
}

func newAppShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <application-id>",
		Short: "Show application details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := resolveContext(cmd)
			if err != nil {
				return err
			}
			if err := ctx.ensureWorkflowAuth(cmd.Context()); err != nil {
				return err
			}

			app, err := ctx.WorkflowAPI.GetApplication(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			if ctx.Printer.Format == output.FormatJSON {
				return ctx.Printer.PrintJSON(app)
			}
			pairs := [][2]string{
				{"id", app.ID},
				{"name", app.Name},
				{"branch", app.Branch},
				{"repository_id", app.RepositoryID},
				{"repository_url", app.RepositoryURL},
				{"status", app.Status},
				{"run_id", app.RunID},
				{"pipeline_id", app.PipelineID},
				{"created_at", output.HumanTime(app.CreatedAt)},
				{"updated_at", output.HumanTime(app.UpdatedAt)},
			}
			ctx.Printer.KeyValue(pairs)
			return nil
		},
	}
	return cmd
}

func newAppUpdateCmd() *cobra.Command {
	var (
		name   string
		branch string
	)

	cmd := &cobra.Command{
		Use:   "update <application-id>",
		Short: "Update an application's name or branch",
		Args:  cobra.ExactArgs(1),
		Example: `  eds wf app update my-app-id --branch release
  eds wf app update my-app-id --name new-name --branch main`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := resolveContext(cmd)
			if err != nil {
				return err
			}
			if err := ctx.ensureWorkflowAuth(cmd.Context()); err != nil {
				return err
			}
			if branch == "" {
				return fmt.Errorf("--branch is required")
			}

			if err := ctx.WorkflowAPI.UpdateApplication(cmd.Context(), args[0], workflowapi.UpdateApplicationRequest{
				Name:   name,
				Branch: branch,
			}); err != nil {
				return err
			}
			if !ctx.Quiet {
				fmt.Fprintf(cmd.OutOrStdout(), "Updated application %s\n", args[0])
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "new application name")
	cmd.Flags().StringVar(&branch, "branch", "", "new branch to deploy from")
	return cmd
}

func newAppDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <application-id>",
		Short: "Delete an application and its deployments (irreversible)",
		Args:  cobra.ExactArgs(1),
		Example: `  eds wf app delete my-app-id --force
  eds wf app delete my-app-id        # prompts for confirmation`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := resolveContext(cmd)
			if err != nil {
				return err
			}
			if err := ctx.ensureWorkflowAuth(cmd.Context()); err != nil {
				return err
			}

			if !force {
				fmt.Fprintf(cmd.ErrOrStderr(),
					"Are you sure you want to delete application %q? Type its id to confirm: ", args[0])
				reader := bufio.NewReader(os.Stdin)
				confirm, _ := reader.ReadString('\n')
				if strings.TrimSpace(confirm) != args[0] {
					return fmt.Errorf("confirmation failed; aborting")
				}
			}

			if err := ctx.WorkflowAPI.DeleteApplication(cmd.Context(), args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted application %s\n", args[0])
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "skip confirmation prompt")
	return cmd
}

func newAppDeployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy <application-id>",
		Short: "Run the application's pipeline and publish it",
		Args:  cobra.ExactArgs(1),
		Long: `deploy triggers a new run of the application's deploy pipeline.
Use the returned run_id with "eds wf run show" or "eds wf app status" to
follow progress, and check the "url" field once it succeeds.`,
		Example: `  eds wf app deploy my-app-id --json | jq -r '.run_id'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := resolveContext(cmd)
			if err != nil {
				return err
			}
			if err := ctx.ensureWorkflowAuth(cmd.Context()); err != nil {
				return err
			}

			dep, err := ctx.WorkflowAPI.CreateDeployment(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if !ctx.Quiet {
				fmt.Fprintf(cmd.OutOrStdout(), "Deploying application %s (run_id=%s)\n", args[0], dep.RunID)
			}
			return ctx.Printer.PrintJSON(dep)
		},
	}
	return cmd
}

func newAppDeploymentsCmd() *cobra.Command {
	var (
		search string
		sort   string
		limit  int
		offset int
	)

	cmd := &cobra.Command{
		Use:   "deployments <application-id>",
		Short: "List deployments (publish history) for an application",
		Args:  cobra.ExactArgs(1),
		Example: `  eds wf app deployments my-app-id
  eds wf app deployments my-app-id --sort created_at_desc --limit 5`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := resolveContext(cmd)
			if err != nil {
				return err
			}
			if err := ctx.ensureWorkflowAuth(cmd.Context()); err != nil {
				return err
			}

			resp, err := ctx.WorkflowAPI.ListDeployments(cmd.Context(), args[0], workflowapi.ListDeploymentsOptions{
				Search: search,
				Sort:   workflowapi.SortOrder(sort),
				Limit:  limit,
				Offset: offset,
			})
			if err != nil {
				return err
			}

			if ctx.Printer.Format == output.FormatJSON {
				return ctx.Printer.PrintJSON(resp)
			}

			headers := []string{"ID", "RUN_ID", "URL", "CREATED"}
			rows := make([][]string, 0, len(resp.Deployments))
			for _, d := range resp.Deployments {
				rows = append(rows, []string{d.ID, d.RunID, d.URL, output.HumanTime(d.CreatedAt)})
			}
			ctx.Printer.Table(headers, rows)
			fmt.Fprintf(cmd.OutOrStdout(), "Showing %d of %d\n", len(resp.Deployments), resp.Total)
			return nil
		},
	}

	cmd.Flags().StringVar(&search, "search", "", "search term (default: all)")
	cmd.Flags().StringVar(&sort, "sort", "created_at_desc", "sort order: created_at_asc, created_at_desc")
	cmd.Flags().IntVar(&limit, "limit", 50, "page size")
	cmd.Flags().IntVar(&offset, "offset", 0, "offset")
	return cmd
}

func newAppStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status <application-id>",
		Short: "Show publish status: current run, stages/jobs and the live URL",
		Args:  cobra.ExactArgs(1),
		Long: `status is a convenience wrapper around "eds wf app show" + the
application's latest run and deployment: it resolves the application's
current run (with its stage/job breakdown) and its most recent deployment
URL in one call. Useful for polling after "eds wf app deploy".`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := resolveContext(cmd)
			if err != nil {
				return err
			}
			if err := ctx.ensureWorkflowAuth(cmd.Context()); err != nil {
				return err
			}

			app, err := ctx.WorkflowAPI.GetApplication(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			var latest *workflowapi.Deployment
			deps, err := ctx.WorkflowAPI.ListDeployments(cmd.Context(), args[0], workflowapi.ListDeploymentsOptions{
				Sort:  workflowapi.SortCreatedAtDesc,
				Limit: 1,
			})
			if err == nil && len(deps.Deployments) > 0 {
				latest = &deps.Deployments[0]
			}

			if ctx.Printer.Format == output.FormatJSON {
				return ctx.Printer.PrintJSON(map[string]any{
					"application":       app,
					"latest_deployment": latest,
				})
			}

			ctx.Printer.KeyValue([][2]string{
				{"id", app.ID},
				{"name", app.Name},
				{"status", app.Status},
				{"run_id", app.RunID},
			})

			if app.Run != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "\nRun %s: %s\n", app.Run.ID, app.Run.Status)
				for _, stage := range app.Run.Stages {
					fmt.Fprintf(cmd.OutOrStdout(), "  stage %-24s %s\n", stage.Name, stage.Status)
					for _, job := range stage.Jobs {
						fmt.Fprintf(cmd.OutOrStdout(), "    job %-22s %s\n", job.Name, job.Status)
					}
				}
			}

			if latest != nil && latest.URL != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "\nurl: %s\n", latest.URL)
			}
			return nil
		},
	}
	return cmd
}
