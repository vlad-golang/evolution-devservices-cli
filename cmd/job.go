package cmd

import (
	"fmt"
	"time"

	workflowclient "github.com/cloud-ru/evolution-devservices-cli/internal/workflow_client"
	"github.com/spf13/cobra"

	"github.com/cloud-ru/evolution-devservices-cli/internal/output"
	"github.com/cloud-ru/evolution-devservices-cli/internal/workflowapi"
)

// newJobCmd creates the parent `eds wf job` command and all its subcommands.
// A job is a single unit of work within a run's stage (e.g. build, push,
// publish).
func newJobCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "job",
		Short: "Inspect and control Workflow Studio jobs",
	}
	cmd.AddCommand(newJobShowCmd())
	cmd.AddCommand(newJobListCmd())
	cmd.AddCommand(newJobLogsCmd())
	cmd.AddCommand(newJobRetryCmd())
	cmd.AddCommand(newJobStopCmd())
	return cmd
}

func newJobShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <job-id>",
		Short: "Show job details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := resolveContext(cmd)
			if err != nil {
				return err
			}
			if err := ctx.ensureWorkflowAuth(cmd.Context()); err != nil {
				return err
			}

			job, err := ctx.WorkflowAPI.GetJob(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			if ctx.Printer.Format == output.FormatJSON {
				return ctx.Printer.PrintJSON(job)
			}
			ctx.Printer.KeyValue([][2]string{
				{"id", job.ID},
				{"name", job.Name},
				{"run_id", job.RunID},
				{"stage_id", job.StageID},
				{"type", job.Type},
				{"status", string(job.Status)},
				{"updated_at", output.HumanTime(job.UpdatedAt)},
			})
			return nil
		},
	}
	return cmd
}

func newJobListCmd() *cobra.Command {
	var (
		runID  string
		limit  int
		offset int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List jobs for a run",
		Example: `  eds wf job list --run-id my-run-id
  eds wf job list --run-id my-run-id --json | jq '.[].status'`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, err := resolveContext(cmd)
			if err != nil {
				return err
			}
			if err := ctx.ensureWorkflowAuth(cmd.Context()); err != nil {
				return err
			}

			jobs, err := ctx.WorkflowAPI.ListJobs(cmd.Context(), workflowapi.ListJobsOptions{
				RunID:  runID,
				Limit:  limit,
				Offset: offset,
			})
			if err != nil {
				return err
			}

			if ctx.Printer.Format == output.FormatJSON {
				return ctx.Printer.PrintJSON(jobs)
			}

			headers := []string{"ID", "NAME", "STAGE_ID", "STATUS", "UPDATED"}
			rows := make([][]string, 0, len(jobs))
			for _, j := range jobs {
				rows = append(rows, []string{j.ID, j.Name, j.StageID, string(j.Status), output.HumanTime(j.UpdatedAt)})
			}
			ctx.Printer.Table(headers, rows)
			return nil
		},
	}

	cmd.Flags().StringVar(&runID, "run-id", "", "run id to list jobs for (required)")
	cmd.Flags().IntVar(&limit, "limit", 50, "page size")
	cmd.Flags().IntVar(&offset, "offset", 0, "offset")
	return cmd
}

func newJobLogsCmd() *cobra.Command {
	var (
		limit      int
		beforeTime string
	)

	cmd := &cobra.Command{
		Use:   "logs <job-id>",
		Short: "Get logs for a job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := resolveContext(cmd)
			if err != nil {
				return err
			}
			if err := ctx.ensureWorkflowAuth(cmd.Context()); err != nil {
				return err
			}

			resp, _, err := ctx.WorkflowClient.WorkflowsAPI.
				ProjectProjectIdJobJobIdLogListGet(cmd.Context(), ctx.ProjectID, args[0]).
				Request(workflowclient.GitSbercloudTechDsworksServicesPipelineSrcInternalApplicationRequestJobLogList{
					BeforeTime: &beforeTime,
					Limit:      &limit,
				}).Execute()
			if err != nil {
				return fmt.Errorf("workflow client logs: %w", err)
			}

			err = ctx.Printer.PrintJSON(resp)
			if err != nil {
				return fmt.Errorf("print json: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 10, "page size, default is 10")
	cmd.Flags().StringVar(&beforeTime, "before", time.Now().Format(time.RFC3339), "RFC3339 time, default is now")

	return cmd
}

func newJobRetryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "retry <job-id>",
		Short: "Retry a failed or canceled job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := resolveContext(cmd)
			if err != nil {
				return err
			}
			if err := ctx.ensureWorkflowAuth(cmd.Context()); err != nil {
				return err
			}

			if err := ctx.WorkflowAPI.RetryJob(cmd.Context(), args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Retrying job %s\n", args[0])
			return nil
		},
	}
	return cmd
}

func newJobStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop <job-id>",
		Short: "Stop a running job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := resolveContext(cmd)
			if err != nil {
				return err
			}
			if err := ctx.ensureWorkflowAuth(cmd.Context()); err != nil {
				return err
			}

			if err := ctx.WorkflowAPI.StopJob(cmd.Context(), args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Stopped job %s\n", args[0])
			return nil
		},
	}
	return cmd
}
