package config

import "context"

type GitlabLoader struct {
	cfg *Config
}

func NewGitlabLoader(cfg *Config) *GitlabLoader {
	return &GitlabLoader{
		cfg: cfg,
	}
}

func (gl *GitlabLoader) Run(ctx context.Context) {
	// Get cfg from gitlab
	// check cfg has changed
	// send signal or call callbackfunctions

	// cfg has changed
	gl.cfg.Checks = map[string]any{
		"rtt": "bla",
	}
	gl.cfg.Updated <- true
}
