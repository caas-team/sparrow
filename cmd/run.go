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
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/config"
	"github.com/caas-team/sparrow/pkg/sparrow"
)

// NewCmdRun creates a new run command
func NewCmdRun() *cobra.Command {
	flagMapping := config.RunFlagsNameMapping{
		ApiAddress:           "apiAddress",
		LoaderType:           "loaderType",
		LoaderInterval:       "loaderInterval",
		LoaderHttpUrl:        "loaderHttpUrl",
		LoaderHttpToken:      "loaderHttpToken",
		LoaderHttpTimeout:    "loaderHttpTimeout",
		LoaderHttpRetryCount: "loaderHttpRetryCount",
		LoaderHttpRetryDelay: "loaderHttpRetryDelay",
		LoaderFilePath:       "loaderFilePath",
	}

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run sparrow",
		Long:  `Sparrow will be started with the provided configuration`,
		Run:   run(&flagMapping),
	}

	cmd.PersistentFlags().String(flagMapping.ApiAddress, ":8080", "api: The address the server is listening on")

	cmd.PersistentFlags().StringP(flagMapping.LoaderType, "l", "http",
		"defines the loader type that will load the checks configuration during the runtime. The fallback is the fileLoader")
	cmd.PersistentFlags().Int(flagMapping.LoaderInterval, 300, "defines the interval the loader reloads the configuration in seconds")
	cmd.PersistentFlags().String(flagMapping.LoaderHttpUrl, "", "http loader: The url where to get the remote configuration")
	cmd.PersistentFlags().String(flagMapping.LoaderHttpToken, "", "http loader: Bearer token to authenticate the http endpoint")
	cmd.PersistentFlags().Int(flagMapping.LoaderHttpTimeout, 30, "http loader: The timeout for the http request in seconds")
	cmd.PersistentFlags().Int(flagMapping.LoaderHttpRetryCount, 3, "http loader: Amount of retries trying to load the configuration")
	cmd.PersistentFlags().Int(flagMapping.LoaderHttpRetryDelay, 1, "http loader: The initial delay between retries in seconds")
	cmd.PersistentFlags().String(flagMapping.LoaderFilePath, "config.yaml", "file loader: The path to the file to read the runtime config from")

	viper.BindPFlag(flagMapping.ApiAddress, cmd.PersistentFlags().Lookup(flagMapping.ApiAddress))

	viper.BindPFlag(flagMapping.LoaderType, cmd.PersistentFlags().Lookup(flagMapping.LoaderType))
	viper.BindPFlag(flagMapping.LoaderInterval, cmd.PersistentFlags().Lookup(flagMapping.LoaderInterval))
	viper.BindPFlag(flagMapping.LoaderHttpUrl, cmd.PersistentFlags().Lookup(flagMapping.LoaderHttpUrl))
	viper.BindPFlag(flagMapping.LoaderHttpToken, cmd.PersistentFlags().Lookup(flagMapping.LoaderHttpToken))
	viper.BindPFlag(flagMapping.LoaderHttpTimeout, cmd.PersistentFlags().Lookup(flagMapping.LoaderHttpTimeout))
	viper.BindPFlag(flagMapping.LoaderHttpRetryCount, cmd.PersistentFlags().Lookup(flagMapping.LoaderHttpRetryCount))
	viper.BindPFlag(flagMapping.LoaderHttpRetryDelay, cmd.PersistentFlags().Lookup(flagMapping.LoaderHttpRetryDelay))
	viper.BindPFlag(flagMapping.LoaderFilePath, cmd.PersistentFlags().Lookup(flagMapping.LoaderFilePath))

	return cmd
}

// run is the entry point to start the sparrow
func run(fm *config.RunFlagsNameMapping) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		log := logger.NewLogger()
		ctx := logger.IntoContext(context.Background(), log)

		cfg := config.NewConfig()

		cfg.SetApiAddress(viper.GetString(fm.ApiAddress))

		cfg.SetLoaderType(viper.GetString(fm.LoaderType))
		cfg.SetLoaderInterval(viper.GetInt(fm.LoaderInterval))
		cfg.SetLoaderHttpUrl(viper.GetString(fm.LoaderHttpUrl))
		cfg.SetLoaderHttpToken(viper.GetString(fm.LoaderHttpToken))
		cfg.SetLoaderHttpTimeout(viper.GetInt(fm.LoaderHttpTimeout))
		cfg.SetLoaderHttpRetryCount(viper.GetInt(fm.LoaderHttpRetryCount))
		cfg.SetLoaderHttpRetryDelay(viper.GetInt(fm.LoaderHttpRetryDelay))
		cfg.SetLoaderFilePath(viper.GetString(fm.LoaderFilePath))

		if err := cfg.Validate(ctx, fm); err != nil {
			log.Error("Error while validating the config", "error", err)
			panic(err)
		}

		sparrow := sparrow.New(cfg)

		log.Info("Running sparrow")
		if err := sparrow.Run(ctx); err != nil {
			panic(err)
		}
	}
}
