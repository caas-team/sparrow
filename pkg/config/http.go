package config

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/caas-team/sparrow/internal/helper"
	"gopkg.in/yaml.v3"
)

type HttpLoader struct {
	log        *slog.Logger
	cfg        *Config
	cCfgChecks chan<- map[string]any
}

func NewHttpLoader(cfg *Config, cCfgChecks chan<- map[string]any) *HttpLoader {
	return &HttpLoader{
		// TODO: set logger from cfg
		log:        slog.Default().WithGroup("httpLoader"),
		cfg:        cfg,
		cCfgChecks: cCfgChecks,
	}
}

// GetRuntimeConfig gets the runtime configuration
// from the http remote endpoint.
// The config is will be loaded periodically defined by the
// loader interval configuration. A failed request will be retried defined
// by the retry configuration
func (gl *HttpLoader) Run(ctx context.Context) {
	var runtimeCfg *RuntimeConfig

	for {
		getConfigRetry := helper.Retry(func(ctx context.Context) error {
			var err error
			runtimeCfg, err = gl.GetRuntimeConfig(ctx)
			return err

		}, gl.cfg.Loader.http.retryCfg)

		if err := getConfigRetry(ctx); err != nil {
			gl.log.Error("Could not get remote runtime configuration", "error", err)
			return
		}

		gl.log.Info("Successfully got remote runtime configuration")
		gl.cCfgChecks <- runtimeCfg.Checks

		timer := time.NewTimer(gl.cfg.Loader.Interval)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
	}
}

// GetRuntimeConfig gets the remote runtime configuration
func (gl *HttpLoader) GetRuntimeConfig(ctx context.Context) (*RuntimeConfig, error) {
	log := gl.log.With("url", gl.cfg.Loader.http.url)

	client := http.DefaultClient
	client.Timeout = gl.cfg.Loader.http.timeout

	req, err := http.NewRequestWithContext(ctx, "GET", gl.cfg.Loader.http.url, nil)
	if err != nil {
		log.Error("Could not create http GET request", "error", err.Error())
		return nil, err
	}
	if gl.cfg.Loader.http.token != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", gl.cfg.Loader.http.token))
	}

	res, err := client.Do(req)
	if err != nil {
		log.Error("Http get request failed", "error", err.Error())
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Error("Http get request failed", "status", res.Status)
		return nil, fmt.Errorf("request fail, status is %s", res.Status)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error("Could not read response body", "error", err.Error())
		return nil, err
	}
	log.Debug("Successfully got response")

	runtimeCfg := &RuntimeConfig{}
	if err := yaml.Unmarshal(body, &runtimeCfg); err != nil {
		log.Error("Could not unmarshal response", "error", err.Error())
		return nil, err
	}

	return runtimeCfg, nil
}
