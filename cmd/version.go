package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// version is the CLI version, set from main via SetVersion.
var version = "dev"

// SetVersion overrides the version string at runtime. Called from main
// after the build injects it via -ldflags.
func SetVersion(v string) {
	if v != "" {
		version = v
	}
}

// newVersionCmd creates the `eds version` command.
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the CLI version",
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintln(cmd.OutOrStdout(), version)
		},
	}
}
