package config

import (
	"context"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/caas-team/sparrow/internal/logger"
)

var _ Loader = (*FileLoader)(nil)

type FileLoader struct {
	path string
	c    chan<- map[string]any
}

func NewFileLoader(cfg *Config, cCfgChecks chan<- map[string]any) *FileLoader {
	return &FileLoader{
		path: cfg.Loader.file.path,
		c:    cCfgChecks,
	}
}

func (f *FileLoader) Run(ctx context.Context) {
	log := logger.FromContext(ctx).WithGroup("FileLoader")
	log.Info("Reading config from file", "file", f.path)
	// TODO refactor this to use fs.FS
	b, err := os.ReadFile(f.path)
	if err != nil {
		log.Error("Failed to read config file", "path", f.path, "error", err)
		panic("failed to read config file " + err.Error())
	}

	var cfg RuntimeConfig

	if err := yaml.Unmarshal(b, &cfg); err != nil {
		log.Error("Failed to parse config file", "error", err)
		panic("failed to parse config file: " + err.Error())
	}

	f.c <- cfg.Checks
}
