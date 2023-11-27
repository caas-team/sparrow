// sparrow
// (C) 2023, Deutsche Telekom IT GmbH
//
// Deutsche Telekom IT GmbH and all other contributors /
// copyright owners license this file to you under the Apache
// License, Version 2.0 (the "License"); you may not use this
// file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package cmd

//go:generate go run ../main.go gen-docs --path ../docs

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
