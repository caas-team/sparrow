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

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewCmdRoot creates a new root command
func NewCmdRoot(version string) *cobra.Command {
	var cfgFile string

	rootCmd := &cobra.Command{
		Use:   "sparrow",
		Short: "Sparrow, the infrastructure monitoring agent",
		Long: "Sparrow is an infrastructure monitoring agent that is able to perform different checks.\n" +
			"The check results are exposed via an API.",
		Version: version,
	}

	cobra.OnInitialize(func() {
		initConfig(cfgFile)
	})

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.sparrow.yaml)")

	return rootCmd
}

// Execute adds all child commands to the root command
// and executes the cmd tree
func Execute(version string) {
	cmd := BuildCmd(version)

	if err := cmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func BuildCmd(version string) *cobra.Command {
	cmd := NewCmdRoot(version)
	cmd.AddCommand(NewCmdRun())
	return cmd
}

func initConfig(cfgFile string) {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".sparrow" (without an extension)
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".sparrow")
	}

	viper.SetEnvPrefix("sparrow")
	dotreplacer := strings.NewReplacer(".", "_")
	viper.EnvKeyReplacer(dotreplacer)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
