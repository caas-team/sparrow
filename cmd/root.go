package cmd

import (
	"fmt"
	"os"

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
	cmd := NewCmdRoot(version)
	cmd.AddCommand(NewCmdRun())

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func initConfig(cfgFile string) {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".sparrow" (without extension)
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".sparrow")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
