package config

import (
	"context"
)

const (
	gitlabLoader = "GITLAB"
	localLoader  = "LOCAL"
)

type Loader interface {
	Run(context.Context)
}

func NewLoader(cfg *Config, cCfgChecks chan<- map[string]any) Loader {
	return NewHttpLoader(cfg, cCfgChecks)
}
