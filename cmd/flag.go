package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Flag struct {
	Config string
	Cli    string
}

type StringFlag struct {
	f *Flag
}

type IntFlag struct {
	f *Flag
}

type StringPFlag struct {
	f  *Flag
	sh string
}

type BindFN func(cmd *cobra.Command, value, usage string)

func (f *StringFlag) Bind(cmd *cobra.Command, value, usage string) {
	cmd.PersistentFlags().String(f.f.Cli, value, usage)
	if err := viper.BindPFlag(f.f.Config, cmd.PersistentFlags().Lookup(f.f.Cli)); err != nil {
		panic(err)
	}
}

func (f *Flag) String() *StringFlag {
	return &StringFlag{
		f: f,
	}
}

func (f *IntFlag) Bind(cmd *cobra.Command, value int, usage string) {
	cmd.PersistentFlags().Int(f.f.Cli, value, usage)
	if err := viper.BindPFlag(f.f.Config, cmd.PersistentFlags().Lookup(f.f.Cli)); err != nil {
		panic(err)
	}
}

func (f *Flag) Int() *IntFlag {
	return &IntFlag{
		f: f,
	}
}

func (f *StringPFlag) Bind(cmd *cobra.Command, value, usage string) {
	cmd.PersistentFlags().StringP(f.f.Cli, f.sh, value, usage)
	if err := viper.BindPFlag(f.f.Config, cmd.PersistentFlags().Lookup(f.f.Cli)); err != nil {
		panic(err)
	}
}

func (f *Flag) StringP(shorthand string) *StringPFlag {
	return &StringPFlag{
		f:  f,
		sh: shorthand,
	}
}

func NewFlag(config, cli string) *Flag {
	return &Flag{
		Config: config,
		Cli:    cli,
	}
}
