package cmd

import (
	"bufio"
	"fmt"
	"strings"

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
	cmd := &cobra.Command{
		Use:   "logs <job-id>",
		Short: "Stream logs for a job",
		Args:  cobra.ExactArgs(1),
		Long: `logs opens the job's log stream (server-sent events) and prints
each log line to stdout as it arrives. For a finished job the stream ends
as soon as the buffered log has been sent; for a running job it stays open
until the job finishes or the command is interrupted (Ctrl-C).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := resolveContext(cmd)
			if err != nil {
				return err
			}
			if err := ctx.ensureWorkflowAuth(cmd.Context()); err != nil {
				return err
			}

			body, err := ctx.WorkflowAPI.StreamJobLogs(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			defer body.Close()

			scanner := bufio.NewScanner(body)
			scanner.Buffer(make([]byte, 64*1024), 1024*1024)
			for scanner.Scan() {
				line := scanner.Text()
				if !strings.HasPrefix(line, "data:") {
					// Skip SSE "event:" lines, ":" heartbeat comments and blank separators.
					continue
				}
				fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			}
			return scanner.Err()
		},
	}
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
