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

func NewLoader(cfg *Config) Loader {
	switch strings.ToUpper("cfg.loaderTyp") {
	case gitlabLoader:
		return NewGitlabLoader(cfg)
	default:
		return NewGitlabLoader(cfg)
	}
}
