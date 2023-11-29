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
		LoaderType:           "loaderType",
		LoaderInterval:       "loaderInterval",
		LoaderHttpUrl:        "loaderHttpUrl",
		LoaderHttpToken:      "loaderHttpToken",
		LoaderHttpTimeout:    "loaderHttpTimeout",
		LoaderHttpRetryCount: "loaderHttpRetryCount",
		LoaderHttpRetryDelay: "loaderHttpRetryDelay",
		LoaderFile:           "loaderFile",
		ApiPort:              "apiPort",
	}

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run sparrow",
		Long:  `Sparrow will be started with the provided configuration`,
		Run:   run(&flagMapping),
	}

	cmd.PersistentFlags().String(flagMapping.ApiPort, ":8080", "Specifies the address to bind the api to")
	cmd.PersistentFlags().StringP(flagMapping.LoaderType, "l", "http", "defines the loader type that will load the checks configuration during the runtime")
	cmd.PersistentFlags().Int(flagMapping.LoaderInterval, 300, "defines the interval the loader reloads the configuration in seconds")
	cmd.PersistentFlags().String(flagMapping.LoaderHttpUrl, "", "http loader: The url where to get the remote configuration")
	cmd.PersistentFlags().String(flagMapping.LoaderHttpToken, "", "http loader: Bearer token to authenticate the http endpoint")
	cmd.PersistentFlags().Int(flagMapping.LoaderHttpTimeout, 30, "http loader: The timeout for the http request in seconds")
	cmd.PersistentFlags().Int(flagMapping.LoaderHttpRetryCount, 3, "http loader: Amount of retries trying to load the configuration")
	cmd.PersistentFlags().Int(flagMapping.LoaderHttpRetryDelay, 1, "http loader: The initial delay between retries in seconds")
	cmd.PersistentFlags().String(flagMapping.LoaderFile, "config.yaml", "file loader: The file to read the runtime config from")

	viper.BindPFlag(flagMapping.ApiPort, cmd.PersistentFlags().Lookup(flagMapping.ApiPort))
	viper.BindPFlag(flagMapping.LoaderType, cmd.PersistentFlags().Lookup(flagMapping.LoaderType))
	viper.BindPFlag(flagMapping.LoaderInterval, cmd.PersistentFlags().Lookup(flagMapping.LoaderInterval))
	viper.BindPFlag(flagMapping.LoaderHttpUrl, cmd.PersistentFlags().Lookup(flagMapping.LoaderHttpUrl))
	viper.BindPFlag(flagMapping.LoaderHttpToken, cmd.PersistentFlags().Lookup(flagMapping.LoaderHttpToken))
	viper.BindPFlag(flagMapping.LoaderHttpTimeout, cmd.PersistentFlags().Lookup(flagMapping.LoaderHttpTimeout))
	viper.BindPFlag(flagMapping.LoaderHttpRetryCount, cmd.PersistentFlags().Lookup(flagMapping.LoaderHttpRetryCount))
	viper.BindPFlag(flagMapping.LoaderHttpRetryDelay, cmd.PersistentFlags().Lookup(flagMapping.LoaderHttpRetryDelay))
	viper.BindPFlag(flagMapping.LoaderFile, cmd.PersistentFlags().Lookup(flagMapping.LoaderFile))

	return cmd
}

// run is the entry point to start the sparrow
func run(fm *config.RunFlagsNameMapping) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		log := logger.NewLogger()
		ctx := logger.IntoContext(context.Background(), log)

		cfg := config.NewConfig()

		cfg.SetApiPort(viper.GetString(fm.ApiPort))
		cfg.SetLoaderType(viper.GetString(fm.LoaderType))
		cfg.SetLoaderInterval(viper.GetInt(fm.LoaderInterval))
		cfg.SetLoaderHttpUrl(viper.GetString(fm.LoaderHttpUrl))
		cfg.SetLoaderHttpToken(viper.GetString(fm.LoaderHttpToken))
		cfg.SetLoaderHttpTimeout(viper.GetInt(fm.LoaderHttpTimeout))
		cfg.SetLoaderHttpRetryCount(viper.GetInt(fm.LoaderHttpRetryCount))
		cfg.SetLoaderHttpRetryDelay(viper.GetInt(fm.LoaderHttpRetryDelay))
		cfg.SetLoaderFile(viper.GetString(fm.LoaderFile))

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
