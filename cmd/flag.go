package cmd

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Flag struct {
	Config string
	Cli    string
}

type StringFlag struct {
	*Flag
}

type IntFlag struct {
	*Flag
}

type DurationFlag struct {
	*Flag
}

type StringPFlag struct {
	*Flag
	sh string
}

type BoolPFlag struct {
	*Flag
	sh string
}

// Bind registers the flag with the command and binds it to the config
func (f *StringFlag) Bind(cmd *cobra.Command, value, usage string) {
	cmd.PersistentFlags().String(f.Cli, value, usage)
	if err := viper.BindPFlag(f.Config, cmd.PersistentFlags().Lookup(f.Cli)); err != nil {
		panic(err)
	}
}

func (f *Flag) String() *StringFlag {
	return &StringFlag{
		Flag: f,
	}
}

func (f *DurationFlag) Bind(cmd *cobra.Command, value time.Duration, usage string) {
	cmd.PersistentFlags().Duration(f.Cli, value, usage)
	if err := viper.BindPFlag(f.Config, cmd.PersistentFlags().Lookup(f.Cli)); err != nil {
		panic(err)
	}
}

func (f *Flag) Duration() *DurationFlag {
	return &DurationFlag{
		Flag: f,
	}
}

// Bind registers the flag with the command and binds it to the config
func (f *IntFlag) Bind(cmd *cobra.Command, value int, usage string) {
	cmd.PersistentFlags().Int(f.Cli, value, usage)
	if err := viper.BindPFlag(f.Config, cmd.PersistentFlags().Lookup(f.Cli)); err != nil {
		panic(err)
	}
}

func (f *Flag) Int() *IntFlag {
	return &IntFlag{
		Flag: f,
	}
}

// Bind registers the flag with the command and binds it to the config
func (f *StringPFlag) Bind(cmd *cobra.Command, value, usage string) {
	cmd.PersistentFlags().StringP(f.Cli, f.sh, value, usage)
	if err := viper.BindPFlag(f.Config, cmd.PersistentFlags().Lookup(f.Cli)); err != nil {
		panic(err)
	}
}

func (f *Flag) StringP(shorthand string) *StringPFlag {
	return &StringPFlag{
		Flag: f,
		sh:   shorthand,
	}
}

// Bind registers the flag with the command and binds it to the config
func (f *BoolPFlag) Bind(cmd *cobra.Command, value bool, usage string) {
	cmd.PersistentFlags().BoolP(f.Cli, f.sh, value, usage)
	if err := viper.BindPFlag(f.Config, cmd.PersistentFlags().Lookup(f.Cli)); err != nil {
		panic(err)
	}
}

func (f *Flag) BoolP(shorthand string) *BoolPFlag {
	return &BoolPFlag{
		Flag: f,
		sh:   shorthand,
	}
}

// NewFlag returns a flag builder
// It serves as a wrapper around cobra and viper, that allows creating and binding typed cli flags to config values
//
// Example:
//
//	NewFlag("config", "c").String().Bind(cmd, "config.yaml", "config file")
func NewFlag(config, cli string) *Flag {
	return &Flag{
		Config: config,
		Cli:    cli,
	}
}
