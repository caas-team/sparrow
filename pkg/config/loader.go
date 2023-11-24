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

// Get a new typed runtime configuration loader
func NewLoader(cfg *Config, cCfgChecks chan<- map[string]any) Loader {
	switch cfg.Loader.Type {
	case "http":
		return NewHttpLoader(cfg, cCfgChecks)
	default:
		return NewFileLoader(cfg, cCfgChecks)
	}
}
