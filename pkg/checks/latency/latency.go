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

package latency

import (
	"context"
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
	_ checks.Check   = (*check)(nil)
	_ checks.Runtime = (*Config)(nil)
)

const CheckName = "latency"

// check is the implementation of the latency check.
// It measures the latency to a list of targets.
type check struct {
	checks.Base
	config  Config
	metrics metrics
}

// NewCheck creates a new instance of the latency check
func NewCheck() checks.Check {
	return &check{
		Base: checks.NewBase(),
		config: Config{
			Retry: checks.DefaultRetry,
		},
		metrics: newMetrics(),
	}
}

// result represents the result of a single latency check for a specific target
type result struct {
	Code  int     `json:"code"`
	Error *string `json:"error"`
	Total float64 `json:"total"`
}

// Run starts the latency check
func (ch *check) Run(ctx context.Context, cResult chan checks.ResultDTO) error {
	ctx, cancel := logger.NewContextWithLogger(ctx)
	defer cancel()
	log := logger.FromContext(ctx)

	timer := time.NewTimer(ch.config.Interval)
	log.InfoContext(ctx, "Starting latency check", "interval", ch.config.Interval.String())
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
			log.DebugContext(ctx, "Interval of latency check updated", "interval", ch.config.Interval.String())
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
			log.DebugContext(ctx, "Successfully finished latency check run")
			ch.Mutex.Lock()
			timer.Reset(ch.config.Interval)
			ch.Mutex.Unlock()
		}
	}
}

// UpdateConfig sets the configuration for the latency check
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

// GetConfig returns the current configuration of the latency Check
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
// by the latency check
func (ch *check) Schema() (*openapi3.SchemaRef, error) {
	return checks.OpenapiFromPerfData(map[string]result{})
}

// GetMetricCollectors returns all metric collectors of check
func (ch *check) GetMetricCollectors() []prometheus.Collector {
	return []prometheus.Collector{
		ch.metrics.totalDuration,
		ch.metrics.count,
		ch.metrics.histogram,
	}
}

// RemoveLabelledMetrics removes the metrics which have the passed target as a label
func (ch *check) RemoveLabelledMetrics(target string) error {
	return ch.metrics.Remove(target)
}

// check performs a latency check using a retry function
// to get the latency to all targets
func (ch *check) check(ctx context.Context) map[string]result {
	log := logger.FromContext(ctx)
	log.Debug("Checking latency")
	ch.Mutex.Lock()
	cfg := ch.config
	ch.Mutex.Unlock()

	if len(cfg.Targets) == 0 {
		log.Debug("No targets defined")
		return map[string]result{}
	}
	log.Debug("Getting latency status for each target in separate routine", "amount", len(cfg.Targets))

	var mu sync.Mutex
	var wg sync.WaitGroup
	results := map[string]result{}

	client := &http.Client{
		Timeout: cfg.Timeout,
	}
	for _, t := range cfg.Targets {
		target := t
		wg.Add(1)
		lo := log.With("target", target)

		getLatencyRetry := helper.Retry(func(ctx context.Context) error {
			res, err := getLatency(ctx, client, target)
			mu.Lock()
			defer mu.Unlock()
			results[target] = res
			if err != nil {
				return err
			}
			return nil
		}, cfg.Retry)

		go func() {
			defer wg.Done()

			lo.Debug("Starting retry routine to get latency status")
			if err := getLatencyRetry(ctx); err != nil {
				lo.Error("Error while checking latency", "error", err)
			}

			lo.Debug("Successfully got latency status of target")
			mu.Lock()
			defer mu.Unlock()

			ch.metrics.totalDuration.WithLabelValues(target).Set(results[target].Total)
			ch.metrics.count.WithLabelValues(target).Inc()
			ch.metrics.histogram.WithLabelValues(target).Observe(results[target].Total)
		}()
	}

	log.Debug("Waiting for all routines to finish")
	wg.Wait()

	log.Debug("Successfully got latency status from all targets")
	return results
}

// getLatency performs an HTTP get request and returns ok if request succeeds
func getLatency(ctx context.Context, c *http.Client, url string) (result, error) {
	log := logger.FromContext(ctx).With("url", url)
	var res result

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		log.Error("Error while creating request", "error", err)
		errval := err.Error()
		res.Error = &errval
		return res, err
	}

	start := time.Now()
	resp, err := c.Do(req)
	if err != nil {
		log.Error("Error while checking latency", "error", err)
		errval := err.Error()
		res.Error = &errval
		return res, err
	}
	end := time.Now()
	defer resp.Body.Close()

	res.Code = resp.StatusCode
	res.Total = end.Sub(start).Seconds()
	return res, nil
}
