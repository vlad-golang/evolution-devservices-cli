package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cloud-ru/evolution-devservices-cli/internal/output"
)

// newRunCmd creates the parent `eds wf run` command and all its subcommands.
// A run is a single execution of a Workflow Studio pipeline (e.g. the
// pipeline behind an application's deploy).
func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Inspect and control Workflow Studio pipeline runs",
	}
	cmd.AddCommand(newRunShowCmd())
	cmd.AddCommand(newRunStopCmd())
	return cmd
}

func newRunShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <run-id>",
		Short: "Show a run's status, stages and jobs",
		Args:  cobra.ExactArgs(1),
		Example: `  eds wf run show my-run-id
  eds wf run show my-run-id --json | jq '.status'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := resolveContext(cmd)
			if err != nil {
				return err
			}
			if err := ctx.ensureWorkflowAuth(cmd.Context()); err != nil {
				return err
			}

			run, err := ctx.WorkflowAPI.GetRun(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			if ctx.Printer.Format == output.FormatJSON {
				return ctx.Printer.PrintJSON(run)
			}

			ctx.Printer.KeyValue([][2]string{
				{"id", run.ID},
				{"pipeline_id", run.PipelineID},
				{"branch", run.Branch},
				{"status", string(run.Status)},
				{"status_message", run.StatusMessage},
				{"updated_at", output.HumanTime(run.UpdatedAt)},
			})
			for _, stage := range run.Stages {
				fmt.Fprintf(cmd.OutOrStdout(), "\nstage %-24s %s\n", stage.Name, stage.Status)
				for _, job := range stage.Jobs {
					fmt.Fprintf(cmd.OutOrStdout(), "  job %-22s %s\n", job.Name, job.Status)
				}
			}
			return nil
		},
	}
	return cmd
}

func newRunStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop <run-id>",
		Short: "Stop a running run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := resolveContext(cmd)
			if err != nil {
				return err
			}
			if err := ctx.ensureWorkflowAuth(cmd.Context()); err != nil {
				return err
			}

			if err := ctx.WorkflowAPI.StopRun(cmd.Context(), args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Stopped run %s\n", args[0])
			return nil
		},
	}
	return cmd
}
