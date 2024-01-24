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

	"github.com/caas-team/sparrow/pkg/checks/runtime"

	"github.com/caas-team/sparrow/internal/helper"
	"github.com/caas-team/sparrow/internal/logger"
	"gopkg.in/yaml.v3"
)

type HttpLoader struct {
	cfg         *Config
	cConfigCfgs chan<- runtime.Config
	done        chan struct{}
	client      *http.Client
}

func NewHttpLoader(cfg *Config, cCfgChecks chan<- runtime.Config) *HttpLoader {
	return &HttpLoader{
		cfg:         cfg,
		cConfigCfgs: cCfgChecks,
		client: &http.Client{
			Timeout: cfg.Loader.Http.Timeout,
		},
	}
}

// Run gets the runtime configuration
// from the http remote endpoint.
// The config is will be loaded periodically defined by the
// loader interval configuration. A failed request will be retried defined
// by the retry configuration
func (hl *HttpLoader) Run(ctx context.Context) error {
	ctx, cancel := logger.NewContextWithLogger(ctx)
	defer cancel()
	log := logger.FromContext(ctx)
	var runtimeCfg *runtime.Config
	tick := time.NewTicker(hl.cfg.Loader.Interval)
	defer tick.Stop()

	for {
		select {
		case <-hl.done:
			log.Info("HTTP Loader terminated")
			return nil

		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			getConfigRetry := helper.Retry(func(ctx context.Context) error {
				var err error
				runtimeCfg, err = hl.GetRuntimeConfig(ctx)
				return err
			}, hl.cfg.Loader.Http.RetryCfg)

			if err := getConfigRetry(ctx); err != nil {
				log.Warn("Could not get remote runtime configuration", "error", err)
				tick.Reset(hl.cfg.Loader.Interval)
				continue
			}

			log.Info("Successfully got remote runtime configuration")
			hl.cConfigCfgs <- *runtimeCfg
			tick.Reset(hl.cfg.Loader.Interval)
		}
	}
}

// GetRuntimeConfig gets the remote runtime configuration
func (hl *HttpLoader) GetRuntimeConfig(ctx context.Context) (*runtime.Config, error) {
	log := logger.FromContext(ctx).With("url", hl.cfg.Loader.Http.Url)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, hl.cfg.Loader.Http.Url, http.NoBody)
	if err != nil {
		log.Error("Could not create http GET request", "error", err.Error())
		return nil, err
	}
	if hl.cfg.Loader.Http.Token != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", hl.cfg.Loader.Http.Token))
	}

	res, err := hl.client.Do(req) //nolint:bodyclose
	if err != nil {
		log.Error("Http get request failed", "error", err.Error())
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			log.Error("Failed to close response body", "error", err.Error())
		}
	}(res.Body)

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

	runtimeCfg := &runtime.Config{}
	if err := yaml.Unmarshal(body, &runtimeCfg); err != nil {
		log.Error("Could not unmarshal response", "error", err.Error())
		return nil, err
	}

	return runtimeCfg, nil
}

// Shutdown stops the loader
func (hl *HttpLoader) Shutdown(ctx context.Context) {
	log := logger.FromContext(ctx)
	// make sure that only one shutdown is processed
	select {
	case hl.done <- struct{}{}:
		log.Info("Sent shutdown signal")
	default:
	}
}
