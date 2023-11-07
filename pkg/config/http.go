package config

import "context"

type HttpLoader struct {
	cfg        *Config
	cCfgChecks chan<- map[string]any
}

func NewHttpLoader(cfg *Config, cCfgChecks chan<- map[string]any) *HttpLoader {
	return &HttpLoader{
		cfg:        cfg,
		cCfgChecks: cCfgChecks,
	}
}

func (gl *HttpLoader) Run(ctx context.Context) {
	// Get cfg from gitlab
	// check cfg has changed
	// send signal

	// cfg has changed
	gl.cCfgChecks <- map[string]any{
		"rtt": "check cfg to set dynamically",
	}
}
