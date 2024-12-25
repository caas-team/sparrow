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
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/config"
	"github.com/caas-team/sparrow/pkg/sparrow"
)

const (
	defaultLoaderHttpTimeout = 30 * time.Second
	defaultLoaderInterval    = 300 * time.Second
	defaultHttpRetryCount    = 3
	defaultHttpRetryDelay    = 1 * time.Second
)

// NewCmdRun creates a new run command
func NewCmdRun() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run sparrow",
		Long:  `Sparrow will be started with the provided configuration`,
		RunE:  run(),
	}

	NewFlag("api.address", "apiAddress").String().Bind(cmd, ":8080", "api: The address the server is listening on")
	NewFlag("name", "sparrowName").String().Bind(cmd, "", "The DNS name of the sparrow")
	NewFlag("loader.type", "loaderType").StringP("l").Bind(cmd, "http", "Defines the loader type that will load the checks configuration during the runtime. The fallback is the fileLoader")
	NewFlag("loader.interval", "loaderInterval").Duration().Bind(cmd, defaultLoaderInterval, "defines the interval the loader reloads the configuration in seconds")
	NewFlag("loader.http.url", "loaderHttpUrl").String().Bind(cmd, "", "http loader: The url where to get the remote configuration")
	NewFlag("loader.http.token", "loaderHttpToken").String().Bind(cmd, "", "http loader: Bearer token to authenticate the http endpoint")
	NewFlag("loader.http.timeout", "loaderHttpTimeout").Duration().Bind(cmd, defaultLoaderHttpTimeout, "http loader: The timeout for the http request in seconds")
	NewFlag("loader.http.retry.count", "loaderHttpRetryCount").Int().Bind(cmd, defaultHttpRetryCount, "http loader: Amount of retries trying to load the configuration")
	NewFlag("loader.http.retry.delay", "loaderHttpRetryDelay").Duration().Bind(cmd, defaultHttpRetryDelay, "http loader: The initial delay between retries in seconds")
	NewFlag("loader.file.path", "loaderFilePath").String().Bind(cmd, "config.yaml", "file loader: The path to the file to read the runtime config from")

	return cmd
}

// run is the entry point to start the sparrow
func run() func(cmd *cobra.Command, args []string) error {
	return func(_ *cobra.Command, _ []string) error {
		cfg := &config.Config{}
		err := viper.Unmarshal(cfg)
		if err != nil {
			return fmt.Errorf("failed to parse config: %w", err)
		}

		ctx, cancel := logger.NewContextWithLogger(context.Background())
		log := logger.FromContext(ctx)
		defer cancel()

		if err = cfg.Validate(ctx); err != nil {
			return fmt.Errorf("error while validating the config: %w", err)
		}

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		s := sparrow.New(cfg)
		cErr := make(chan error, 1)
		log.InfoContext(ctx, "Running sparrow")
		go func() {
			cErr <- s.Run(ctx)
		}()

		select {
		case <-sigChan:
			log.InfoContext(ctx, "Signal received, shutting down")
			cancel()
			<-cErr
		case err = <-cErr:
			log.InfoContext(ctx, "Sparrow was shut down")
			return err
		}

		return nil
	}
}
