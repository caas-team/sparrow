// sparrow
// (C) 2023, Deutsche Telekom IT GmbH
//
// Deutsche Telekom IT GmbH and all other contributors /
// copyright owners license this file to you under the Apache
// License, Version 2.0 (the "License"); you may not use this
// file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package config

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/caas-team/sparrow/internal/helper"
	"github.com/caas-team/sparrow/internal/logger"
	"gopkg.in/yaml.v3"
)

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

// Run gets the runtime configuration
// from the http remote endpoint.
// The config is will be loaded periodically defined by the
// loader interval configuration. A failed request will be retried defined
// by the retry configuration
func (gl *HttpLoader) Run(ctx context.Context) {
	ctx, cancel := logger.NewContextWithLogger(ctx, "httpLoader")
	defer cancel()
	log := logger.FromContext(ctx)

	var runtimeCfg *RuntimeConfig
	for {
		getConfigRetry := helper.Retry(func(ctx context.Context) error {
			var err error
			runtimeCfg, err = gl.GetRuntimeConfig(ctx)
			return err
		}, gl.cfg.Loader.http.retryCfg)

		if err := getConfigRetry(ctx); err != nil {
			log.Error("Could not get remote runtime configuration", "error", err)
			return
		}

		log.Info("Successfully got remote runtime configuration")
		gl.cCfgChecks <- runtimeCfg.Checks

		select {
		case <-ctx.Done():
			return
		case <-time.After(gl.cfg.Loader.Interval):
		}
	}
}

// GetRuntimeConfig gets the remote runtime configuration
func (gl *HttpLoader) GetRuntimeConfig(ctx context.Context) (*RuntimeConfig, error) {
	log := logger.FromContext(ctx).With("url", gl.cfg.Loader.http.url)

	client := http.DefaultClient
	client.Timeout = gl.cfg.Loader.http.timeout

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, gl.cfg.Loader.http.url, http.NoBody)
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

	if res.StatusCode != http.StatusOK {
		log.Error("Http get request failed", "status", res.Status)
		return nil, fmt.Errorf("request failed, status is %s", res.Status)
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
