package cmd

import (
	"context"

	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/config"
	"github.com/caas-team/sparrow/pkg/sparrow"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type RunFlagsNameMapping struct {
	loaderType           string
	loaderInterval       string
	loaderHttpUrl        string
	loaderHttpToken      string
	loaderHttpTimeout    string
	loaderHttpRetryCount string
	loaderHttpRetryDelay string
}

// NewCmdRun creates a new run command
func NewCmdRun() *cobra.Command {
	flagMapping := RunFlagsNameMapping{
		loaderType:           "loaderType",
		loaderInterval:       "loaderInterval",
		loaderHttpUrl:        "loaderHttpUrl",
		loaderHttpToken:      "loaderHttpToken",
		loaderHttpTimeout:    "loaderHttpTimeout",
		loaderHttpRetryCount: "loaderHttpRetryCount",
		loaderHttpRetryDelay: "loaderHttpRetryDelay",
	}

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run sparrow",
		Long:  `Sparrow will be started with the provided configuration`,
		Run:   run(&flagMapping),
	}

	cmd.PersistentFlags().StringP(flagMapping.loaderType, "l", "http", "defines the loader type that will load the checks configuration during the runtime")
	cmd.PersistentFlags().Int(flagMapping.loaderInterval, 300, "defines the interval the loader reloads the configuration in seconds")
	cmd.PersistentFlags().String(flagMapping.loaderHttpUrl, "", "http loader: The url where to get the remote configuration")
	cmd.PersistentFlags().String(flagMapping.loaderHttpToken, "", "http loader: Bearer token to authenticate the http endpoint")
	cmd.PersistentFlags().Int(flagMapping.loaderHttpTimeout, 30, "http loader: The timeout for the http request in seconds")
	cmd.PersistentFlags().Int(flagMapping.loaderHttpRetryCount, 3, "http loader: Amount of retries trying to load the configuration")
	cmd.PersistentFlags().Int(flagMapping.loaderHttpRetryDelay, 1, "http loader: The initial delay between retries in seconds")

	viper.BindPFlag(flagMapping.loaderType, cmd.PersistentFlags().Lookup(flagMapping.loaderType))
	viper.BindPFlag(flagMapping.loaderInterval, cmd.PersistentFlags().Lookup(flagMapping.loaderInterval))
	viper.BindPFlag(flagMapping.loaderHttpUrl, cmd.PersistentFlags().Lookup(flagMapping.loaderHttpUrl))
	viper.BindPFlag(flagMapping.loaderHttpToken, cmd.PersistentFlags().Lookup(flagMapping.loaderHttpToken))
	viper.BindPFlag(flagMapping.loaderHttpTimeout, cmd.PersistentFlags().Lookup(flagMapping.loaderHttpTimeout))
	viper.BindPFlag(flagMapping.loaderHttpRetryCount, cmd.PersistentFlags().Lookup(flagMapping.loaderHttpRetryCount))
	viper.BindPFlag(flagMapping.loaderHttpRetryDelay, cmd.PersistentFlags().Lookup(flagMapping.loaderHttpRetryDelay))

	return cmd
}

// run is the entry point to start the sparrow
func run(fm *RunFlagsNameMapping) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		log := logger.GetLogger()
		ctx := logger.IntoContext(context.Background(), log)

		cfg := config.NewConfig()
		cfg.SetLoaderType(viper.GetString(fm.loaderType))
		cfg.SetLoaderInterval(viper.GetInt(fm.loaderInterval))
		cfg.SetLoaderHttpUrl(viper.GetString(fm.loaderHttpUrl))
		cfg.SetLoaderHttpToken(viper.GetString(fm.loaderHttpToken))
		cfg.SetLoaderHttpTimeout(viper.GetInt(fm.loaderHttpTimeout))
		cfg.SetLoaderHttpRetryCount(viper.GetInt(fm.loaderHttpRetryCount))
		cfg.SetLoaderHttpRetryDelay(viper.GetInt(fm.loaderHttpRetryDelay))

		if err := cfg.Validate(); err != nil {
			log.Error("Error while validating the config", "error", err)
			panic(err)
		}

		sparrow := sparrow.New(ctx, cfg)

		log.Info("Running sparrow")
		if err := sparrow.Run(ctx); err != nil {
			panic(err)
		}
	}
}
