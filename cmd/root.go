package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewCmdRoot creates a new root command
func NewCmdRoot(version string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "sparrow",
		Short: "Sparrow, the infrastructure monitoring agent",
		Long: "Sparrow is an infrastructure monitoring agent that is able to perform different checks.\n" +
			"The check results are exposed via an API.",
		Version: version,
	}
	return rootCmd
}

// Execute adds all child commands to the root command
// and executes the cmd tree
func Execute(version string) {
	cmd := NewCmdRoot(version)
	cmd.AddCommand(NewCmdRun())

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
