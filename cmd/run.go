package cmd

import (
	"context"
	"log"

	"github.com/caas-team/sparrow/pkg/config"
	"github.com/caas-team/sparrow/pkg/sparrow"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type RunFlagsNameMapping struct {
	loaderType      string
	loaderInterval  string
	loaderHttpUrl   string
	loaderHttpToken string
}

// NewCmdRun creates a new run command
func NewCmdRun() *cobra.Command {
	flagMapping := RunFlagsNameMapping{
		loaderType:      "loaderType",
		loaderInterval:  "loaderInterval",
		loaderHttpUrl:   "loaderHttpUrl",
		loaderHttpToken: "loaderHttpToken",
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

	viper.BindPFlag(flagMapping.loaderType, cmd.PersistentFlags().Lookup(flagMapping.loaderType))
	viper.BindPFlag(flagMapping.loaderInterval, cmd.PersistentFlags().Lookup(flagMapping.loaderInterval))
	viper.BindPFlag(flagMapping.loaderHttpUrl, cmd.PersistentFlags().Lookup(flagMapping.loaderHttpUrl))
	viper.BindPFlag(flagMapping.loaderHttpToken, cmd.PersistentFlags().Lookup(flagMapping.loaderHttpToken))

	return cmd
}

// run is the entry point to start the sparrow
func run(fm *RunFlagsNameMapping) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		cfg := config.NewConfig()

		cfg.SetLoaderType(viper.GetString(fm.loaderType))
		cfg.SetLoaderInterval(viper.GetInt(fm.loaderInterval))
		cfg.SetLoaderHttpUrl(viper.GetString(fm.loaderHttpUrl))
		cfg.SetLoaderHttpToken(viper.GetString(fm.loaderHttpToken))

		if err := cfg.Validate(); err != nil {
			log.Panic(err)
		}

		sparrow := sparrow.New(cfg)

		log.Println("running sparrow")
		if err := sparrow.Run(context.Background()); err != nil {
			panic(err)
		}
	}
}
