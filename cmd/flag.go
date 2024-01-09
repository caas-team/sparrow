package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Flag struct {
	Config string
	CLI    string
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
	fmt.Println(f.f.CLI)
	fmt.Println(f.f.Config)
	cmd.PersistentFlags().String(f.f.CLI, value, usage)
	viper.BindPFlag(f.f.Config, cmd.PersistentFlags().Lookup(f.f.CLI))
}

func (f *Flag) String() *StringFlag {
	return &StringFlag{
		f: f,
	}
}

func (f *IntFlag) Bind(cmd *cobra.Command, value int, usage string) {
	cmd.PersistentFlags().Int(f.f.CLI, value, usage)
	viper.BindPFlag(f.f.Config, cmd.PersistentFlags().Lookup(f.f.CLI))
}

func (f *Flag) Int() *IntFlag {
	return &IntFlag{
		f: f,
	}
}

func (f *StringPFlag) Bind(cmd *cobra.Command, value, usage string) {
	cmd.PersistentFlags().StringP(f.f.CLI, f.sh, value, usage)
	viper.BindPFlag(f.f.Config, cmd.PersistentFlags().Lookup(f.f.CLI))
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
		CLI:    cli,
	}
}
