package cmd

import (
	"context"
	"log"

	"github.com/caas-team/sparrow/pkg/config"
	"github.com/caas-team/sparrow/pkg/sparrow"
	"github.com/spf13/cobra"
)

// RunFlags contains the flags for the run command
type RunFlags struct {
	// Loader
	loaderType       string
	loaderReloadTime int
	loaderHttpUrl    string
	loaderHttpToken  string
}

// NewCmdRun creates a new run command
func NewCmdRun() *cobra.Command {
	f := RunFlags{}

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run sparrow",
		Long:  `Sparrow will be started with the provided configuration`,
		Run:   run(&f),
	}

	cmd.PersistentFlags().StringVarP(&f.loaderType, "loader-type", "l", "http", "defines the loader type that will load the checks configuration during the runtime")
	cmd.PersistentFlags().IntVar(&f.loaderReloadTime, "loader-interval", 300, "defines the interval the loader reloads the configuration in seconds")
	cmd.PersistentFlags().StringVar(&f.loaderHttpUrl, "loader-http-url", "", "http loader: The url where to get the remote configuration")
	cmd.PersistentFlags().StringVar(&f.loaderHttpToken, "loader-http-token", "", "http loader: Bearer token to authenticate the http endpoint")

	return cmd
}

// run is the entry point to start the sparrow
func run(f *RunFlags) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		cfg := config.NewConfig()

		cfg.SetLoaderType(f.loaderType)
		cfg.SetLoaderHttpUrl(f.loaderHttpUrl)
		cfg.SetLoaderHttpToken(f.loaderHttpToken)

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
