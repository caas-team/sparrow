package cmd

//go:generate go run ../main.go gen-docs --path ../cmd/docs

import (
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// NewCmdRun creates a new gen-docs command
func NewCmdGenDocs(rootCmd *cobra.Command) *cobra.Command {
	var docPath string

	cmd := &cobra.Command{
		Use:   "gen-docs",
		Short: "Generate markdown documentation",
		Long:  `Generate the markdown documentation of available CLI flags`,
		Run:   runGenDocs(rootCmd, &docPath),
	}

	cmd.PersistentFlags().StringVar(&docPath, "path", "docs", "directory path where the markdown files will be created")

	return cmd
}

// run is the entry point to start the sparrow
func runGenDocs(rootCmd *cobra.Command, path *string) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		doc.GenMarkdownTree(rootCmd, *path)
	}
}
