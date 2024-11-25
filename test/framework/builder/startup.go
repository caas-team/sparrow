package builder

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/caas-team/sparrow/internal/helper"
	"github.com/caas-team/sparrow/pkg/api"
	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/config"
	"github.com/caas-team/sparrow/pkg/sparrow/metrics"
	"github.com/caas-team/sparrow/pkg/sparrow/targets"
	"github.com/caas-team/sparrow/pkg/sparrow/targets/interactor"
	"github.com/caas-team/sparrow/pkg/sparrow/targets/remote/gitlab"
	"github.com/goccy/go-yaml"
)

type SparrowConfig struct{ cfg config.Config }

func NewSparrowConfig() *SparrowConfig {
	return &SparrowConfig{
		cfg: config.Config{
			SparrowName: "sparrow.telekom.com",
			Loader:      NewLoaderConfig().Build(),
			Api:         NewAPIConfig("localhost:8080"),
		},
	}
}

func (b *SparrowConfig) WithName(n string) *SparrowConfig {
	b.cfg.SparrowName = n
	return b
}

func (b *SparrowConfig) WithLoader(cfg config.LoaderConfig) *SparrowConfig { //nolint:gocritic // Performance is not a concern here
	b.cfg.Loader = cfg
	return b
}

func (b *SparrowConfig) WithAPI(cfg api.Config) *SparrowConfig {
	b.cfg.Api = cfg
	return b
}

func (b *SparrowConfig) WithTargetManager(cfg targets.TargetManagerConfig) *SparrowConfig { //nolint:gocritic // Performance is not a concern here
	b.cfg.TargetManager = cfg
	return b
}

func (b *SparrowConfig) WithTelemetry(cfg metrics.Config) *SparrowConfig { //nolint:gocritic // Performance is not a concern here
	b.cfg.Telemetry = cfg
	return b
}

func (b *SparrowConfig) Config(t *testing.T) *config.Config {
	t.Helper()
	if err := b.cfg.Validate(context.Background()); err != nil {
		t.Fatalf("config is not valid: %v", err)
	}
	return &b.cfg
}

func (b *SparrowConfig) YAML(t *testing.T) []byte {
	t.Helper()
	out, err := yaml.Marshal(b.cfg)
	if err != nil {
		t.Fatalf("[%T] failed to marshal config: %v", b.cfg, err)
		return []byte{}
	}
	return out
}

type LoaderConfigBuilder struct{ cfg config.LoaderConfig }

func NewLoaderConfig() *LoaderConfigBuilder {
	return &LoaderConfigBuilder{
		cfg: config.LoaderConfig{
			Type:     "file",
			Interval: 0,
			File: config.FileLoaderConfig{
				Path: "testdata/checks.yaml",
			},
		},
	}
}

func (b *LoaderConfigBuilder) WithInterval(i time.Duration) *LoaderConfigBuilder {
	b.cfg.Interval = i
	return b
}

func (b *LoaderConfigBuilder) FromFile(path string) *LoaderConfigBuilder {
	b.cfg.Type = "file"
	b.cfg.File.Path = path
	return b
}

func (b *LoaderConfigBuilder) FromHTTP(cfg config.HttpLoaderConfig) *LoaderConfigBuilder {
	if cfg.RetryCfg == (helper.RetryConfig{}) {
		cfg.RetryCfg = checks.DefaultRetry
	}

	b.cfg.Type = "http"
	b.cfg.Http = cfg
	return b
}

func (b *LoaderConfigBuilder) Build() config.LoaderConfig {
	return b.cfg
}

func NewAPIConfig(address string) api.Config {
	return api.Config{ListeningAddress: address}
}

type TargetManagerConfigBuilder struct{ cfg targets.TargetManagerConfig }

func NewTargetManagerConfig() *TargetManagerConfigBuilder {
	id, _ := strconv.Atoi(os.Getenv("SPARROW_TARGETMANAGER_GITLAB_PROJECTID"))
	return &TargetManagerConfigBuilder{
		cfg: targets.TargetManagerConfig{
			Enabled: true,
			Type:    interactor.Gitlab,
			General: targets.General{
				CheckInterval:        60 * time.Second,
				RegistrationInterval: 0,
				UpdateInterval:       0,
				UnhealthyThreshold:   0,
				Scheme:               "http",
			},
			Config: interactor.Config{
				Gitlab: gitlab.Config{
					BaseURL:   os.Getenv("SPARROW_TARGETMANAGER_GITLAB_BASEURL"),
					Token:     os.Getenv("SPARROW_TARGETMANAGER_GITLAB_TOKEN"),
					ProjectID: id,
				},
			},
		},
	}
}

func (b *TargetManagerConfigBuilder) WithScheme(s string) *TargetManagerConfigBuilder {
	b.cfg.Scheme = s
	return b
}

func (b *TargetManagerConfigBuilder) WithCheckInterval(i time.Duration) *TargetManagerConfigBuilder {
	b.cfg.CheckInterval = i
	return b
}

func (b *TargetManagerConfigBuilder) WithRegistrationInterval(i time.Duration) *TargetManagerConfigBuilder {
	b.cfg.RegistrationInterval = i
	return b
}

func (b *TargetManagerConfigBuilder) WithUpdateInterval(i time.Duration) *TargetManagerConfigBuilder {
	b.cfg.UpdateInterval = i
	return b
}

func (b *TargetManagerConfigBuilder) WithUnhealthyThreshold(t time.Duration) *TargetManagerConfigBuilder {
	b.cfg.UnhealthyThreshold = t
	return b
}

func (b *TargetManagerConfigBuilder) Build() targets.TargetManagerConfig {
	return b.cfg
}
