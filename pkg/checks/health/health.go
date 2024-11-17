// sparrow
// (C) 2024, Deutsche Telekom IT GmbH
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

package health

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/caas-team/sparrow/internal/helper"
	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/getkin/kin-openapi/openapi3"
)

var (
	_            checks.Check   = (*check)(nil)
	_            checks.Runtime = (*Config)(nil)
	stateMapping                = map[int]string{
		0: "unhealthy",
		1: "healthy",
	}
)

const CheckName = "health"

// check is the implementation of the health check.
// It measures the availability of a list of targets.
type check struct {
	checks.Base
	config  Config
	metrics metrics
}

// NewCheck creates a new instance of the health check
func NewCheck() checks.Check {
	return &check{
		Base: checks.NewBase(),
		config: Config{
			Retry: checks.DefaultRetry,
		},
		metrics: newMetrics(),
	}
}

// Run starts the health check
func (ch *check) Run(ctx context.Context, cResult chan checks.ResultDTO) error {
	ctx, cancel := logger.NewContextWithLogger(ctx)
	defer cancel()
	log := logger.FromContext(ctx)

	timer := time.NewTimer(ch.config.Interval)
	log.InfoContext(ctx, "Starting health check", "interval", ch.config.Interval.String())
	for {
		select {
		case <-ctx.Done():
			log.ErrorContext(ctx, "Context canceled", "err", ctx.Err())
			return ctx.Err()
		case <-ch.Done:
			return nil
		case <-ch.Update:
			ch.Mutex.Lock()
			timer.Reset(ch.config.Interval)
			log.DebugContext(ctx, "Interval of health check updated", "interval", ch.config.Interval.String())
			ch.Mutex.Unlock()
		case <-timer.C:
			res := ch.check(ctx)
			cResult <- checks.ResultDTO{
				Name: ch.Name(),
				Result: &checks.Result{
					Data:      res,
					Timestamp: time.Now(),
				},
			}
			log.DebugContext(ctx, "Successfully finished health check run")
			ch.Mutex.Lock()
			timer.Reset(ch.config.Interval)
			ch.Mutex.Unlock()
		}
	}
}

// UpdateConfig sets the configuration for the health check
func (ch *check) UpdateConfig(cfg checks.Runtime) error {
	if c, ok := cfg.(*Config); ok {
		ch.Mutex.Lock()
		defer ch.Mutex.Unlock()
		if reflect.DeepEqual(ch.config, *c) {
			return nil
		}

		for _, target := range ch.config.Targets {
			if !slices.Contains(c.Targets, target) {
				err := ch.metrics.Remove(target)
				if err != nil {
					return err
				}
			}
		}

		ch.config = *c
		ch.Update <- struct{}{}
		return nil
	}

	return checks.ErrConfigMismatch{
		Expected: CheckName,
		Current:  cfg.For(),
	}
}

// GetConfig returns the current configuration of the check
func (ch *check) GetConfig() checks.Runtime {
	ch.Mutex.Lock()
	defer ch.Mutex.Unlock()
	return &ch.config
}

// Name returns the name of the check
func (*check) Name() string {
	return CheckName
}

// Schema provides the schema of the data that will be provided
// by the health check
func (ch *check) Schema() (*openapi3.SchemaRef, error) {
	return checks.OpenapiFromPerfData(map[string]string{})
}

// GetMetricCollectors returns all metric collectors of check
func (ch *check) GetMetricCollectors() []prometheus.Collector {
	return []prometheus.Collector{
		ch.metrics,
	}
}

// RemoveLabelledMetrics removes the metrics which have the passed
// target as a label
func (ch *check) RemoveLabelledMetrics(target string) error {
	return ch.metrics.Remove(target)
}

// check performs a health check using a retry function
// to get the health status for all targets
func (ch *check) check(ctx context.Context) map[string]string {
	log := logger.FromContext(ctx)
	log.Debug("Checking health")
	ch.Mutex.Lock()
	cfg := ch.config
	ch.Mutex.Unlock()

	if len(cfg.Targets) == 0 {
		log.Debug("No targets defined")
		return map[string]string{}
	}
	log.Debug("Getting health status for each target in separate routine", "amount", len(cfg.Targets))

	var wg sync.WaitGroup
	var mu sync.Mutex
	results := map[string]string{}

	client := &http.Client{
		Timeout: cfg.Timeout,
	}
	for _, t := range cfg.Targets {
		target := t
		wg.Add(1)
		l := log.With("target", target)

		getHealthRetry := helper.Retry(func(ctx context.Context) error {
			return getHealth(ctx, client, target)
		}, cfg.Retry)

		go func() {
			defer wg.Done()
			state := 1

			l.Debug("Starting retry routine to get health status")
			if err := getHealthRetry(ctx); err != nil {
				state = 0
				l.Warn(fmt.Sprintf("Health check failed after %d retries", cfg.Retry.Count), "error", err)
			}

			l.Debug("Successfully got health status of target", "status", stateMapping[state])
			mu.Lock()
			defer mu.Unlock()
			results[target] = stateMapping[state]

			ch.metrics.WithLabelValues(target).Set(float64(state))
		}()
	}

	log.Debug("Waiting for all routines to finish")
	wg.Wait()

	log.Debug("Successfully got health status from all targets")
	return results
}

// getHealth performs an HTTP get request and returns ok if status code is 200
func getHealth(ctx context.Context, client *http.Client, url string) error {
	log := logger.FromContext(ctx).With("url", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		log.Error("Error while creating request", "error", err)
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Error("Error while requesting health", "error", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Warn("Health request was not ok (HTTP Status 200)", "status", resp.Status)
		return fmt.Errorf("request failed, status is %s", resp.Status)
	}

	return nil
}
