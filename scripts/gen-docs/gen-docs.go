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

package main

//go:generate go run gen-docs.go gen-docs --path ../../docs

import (
	"fmt"
	"os"

	sparrowcmd "github.com/caas-team/sparrow/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func main() {
	execute()
}

func execute() {
	rootCmd := &cobra.Command{
		Use:   "gen-docs",
		Short: "Generates docs for sparrow",
	}
	rootCmd.AddCommand(NewCmdGenDocs())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// NewCmdGenDocs creates a new gen-docs command
func NewCmdGenDocs() *cobra.Command {
	var docPath string

	cmd := &cobra.Command{
		Use:   "gen-docs",
		Short: "Generate markdown documentation",
		Long:  `Generate the markdown documentation of available CLI flags`,
		Run:   runGenDocs(&docPath),
	}

	cmd.PersistentFlags().StringVar(&docPath, "path", "docs", "directory path where the markdown files will be created")

	return cmd
}

// runGenDocs generates the markdown files for the flag documentation
func runGenDocs(path *string) func(cmd *cobra.Command, args []string) {
	irgendwascmd := sparrowcmd.BuildCmd("")
	irgendwascmd.DisableAutoGenTag = true
	return func(cmd *cobra.Command, args []string) {
		_ = doc.GenMarkdownTree(irgendwascmd, *path)
	}
}
