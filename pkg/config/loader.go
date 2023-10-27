package config

import (
	"context"
	"strings"
)

const (
	gitlabLoader = "GITLAB"
	localLoader  = "LOCAL"
)

type Loader interface {
	Run(context.Context)
}

func NewLoader(cfg *Config, cCfgChecks chan<- map[string]any) Loader {
	switch strings.ToUpper("cfg.loaderTyp") {
	case gitlabLoader:
		return NewHttpLoader(cfg, cCfgChecks)
	default:
		return NewHttpLoader(cfg, cCfgChecks)
	}
}
