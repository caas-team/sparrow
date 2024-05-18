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
	"io"
	"net/http"
	"sync"

	"github.com/caas-team/sparrow/internal/helper"
	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	_ checks.Check   = (*Health)(nil)
	_ checks.Runtime = (*Config)(nil)
	// stateMapping maps the health state to a human readable string
	stateMapping = map[int]string{
		0: "unhealthy",
		1: "healthy",
	}
)

const CheckName = "health"

// Health is a check that measures the availability of an endpoint
type Health struct {
	checks.Base[*Config]
	metrics metrics
}

// NewCheck creates a new instance of the health check
func NewCheck() checks.Check {
	return &Health{
		Base: checks.NewBase(CheckName, &Config{
			Retry: checks.DefaultRetry,
		}),
		metrics: newMetrics(),
	}
}

// metrics contains the metric collectors for the Health check
type metrics struct {
	*prometheus.GaugeVec
}

// Run starts the health check
func (h *Health) Run(ctx context.Context, cResult chan checks.ResultDTO) error {
	return h.StartCheck(ctx, cResult, h.Config.Interval, func(ctx context.Context) any {
		return h.check(ctx)
	})
}

// Schema provides the schema of the data that will be provided
// by the health check
func (h *Health) Schema() (*openapi3.SchemaRef, error) {
	return checks.OpenapiFromPerfData(map[string]string{})
}

// newMetrics initializes metric collectors of the health check
func newMetrics() metrics {
	return metrics{
		GaugeVec: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "sparrow_health_up",
				Help: "Health of targets",
			},
			[]string{
				"target",
			},
		),
	}
}

// GetMetricCollectors returns all metric collectors of check
func (h *Health) GetMetricCollectors() []prometheus.Collector {
	return []prometheus.Collector{
		h.metrics,
	}
}

// check performs a health check using a retry function
// to get the health status for all targets
func (h *Health) check(ctx context.Context) map[string]string {
	log := logger.FromContext(ctx)
	log.Debug("Checking health")
	if len(h.Config.Targets) == 0 {
		log.Debug("No targets defined")
		return map[string]string{}
	}
	log.Debug("Getting health status for each target in separate routine", "amount", len(h.Config.Targets))

	var wg sync.WaitGroup
	var mu sync.Mutex
	results := map[string]string{}

	client := &http.Client{
		Timeout: h.Config.Timeout,
	}
	for _, t := range h.Config.Targets {
		target := t
		wg.Add(1)
		l := log.With("target", target)

		getHealthRetry := helper.Retry(func(ctx context.Context) error {
			return getHealth(ctx, client, target)
		}, h.Config.Retry)

		go func() {
			defer wg.Done()
			state := 1

			l.Debug("Starting retry routine to get health status")
			if err := getHealthRetry(ctx); err != nil {
				state = 0
				l.Warn(fmt.Sprintf("Health check failed after %d retries", h.Config.Retry.Count), "error", err)
			}

			l.Debug("Successfully got health status of target", "status", stateMapping[state])
			mu.Lock()
			defer mu.Unlock()
			results[target] = stateMapping[state]

			h.metrics.WithLabelValues(target).Set(float64(state))
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

	resp, err := client.Do(req) //nolint:bodyclose // Closed in defer below
	if err != nil {
		log.Error("Error while requesting health", "error", err)
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Error("Failed to close response body", "error", err.Error())
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Warn("Health request was not ok (HTTP Status 200)", "status", resp.Status)
		return fmt.Errorf("request failed, status is %s", resp.Status)
	}

	return nil
}
