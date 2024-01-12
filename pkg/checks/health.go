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

package checks

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/caas-team/sparrow/internal/helper"
	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/api"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	_            Check = (*Health)(nil)
	stateMapping       = map[int]string{
		0: "unhealthy",
		1: "healthy",
	}
)

// Health is a check that measures the availability of an endpoint
type Health struct {
	CheckBase
	route   string
	config  HealthConfig
	metrics healthMetrics
}

// NewHealthCheck creates a new instance of the health check
func NewHealthCheck() Check {
	return &Health{
		CheckBase: CheckBase{
			mu:      sync.Mutex{},
			cResult: nil,
			done:    make(chan bool, 1),
		},
		route: "health",
		config: HealthConfig{
			Retry: DefaultRetry,
		},
		metrics: newHealthMetrics(),
	}
}

// HealthConfig defines the configuration parameters for a health check
type HealthConfig struct {
	Targets  []string           `json:"targets,omitempty" yaml:"targets,omitempty" mapstructure:"targets"`
	Interval time.Duration      `json:"interval" yaml:"interval" mapstructure:"interval"`
	Timeout  time.Duration      `json:"timeout" yaml:"timeout" mapstructure:"timeout"`
	Retry    helper.RetryConfig `json:"retry" yaml:"retry" mapstructure:"retry"`
}

// Defined metric collectors of health check
type healthMetrics struct {
	*prometheus.GaugeVec
}

// Run starts the health check
func (h *Health) Run(ctx context.Context) error {
	ctx, cancel := logger.NewContextWithLogger(ctx, "health")
	defer cancel()
	log := logger.FromContext(ctx)
	log.Info(fmt.Sprintf("Using health check interval of %s", h.config.Interval.String()))

	for {
		select {
		case <-ctx.Done():
			log.Error("Context canceled", "err", ctx.Err())
			return ctx.Err()
		case <-h.done:
			log.Debug("Soft shut down")
			return nil
		case <-time.After(h.config.Interval):
			res := h.check(ctx)
			errval := ""
			r := Result{
				Data:      res,
				Err:       errval,
				Timestamp: time.Now(),
			}

			h.cResult <- r
			log.Debug("Successfully finished health check run")
		}
	}
}

// Startup is called once when the health check is registered
func (h *Health) Startup(ctx context.Context, cResult chan<- Result) error {
	log := logger.FromContext(ctx).WithGroup("latency")
	log.Debug("Starting latency check")

	h.cResult = cResult
	return nil
}

// Shutdown is called once when the check is unregistered or sparrow shuts down
func (h *Health) Shutdown(_ context.Context) error {
	h.done <- true
	close(h.done)

	return nil
}

// SetConfig sets the configuration for the health check
func (h *Health) SetConfig(ctx context.Context, config any) error {
	log := logger.FromContext(ctx)

	c, err := helper.Decode[HealthConfig](config)
	if err != nil {
		log.Error("Failed to decode health config", "error", err)
		return ErrInvalidConfig
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	h.config = c

	return nil
}

// Schema provides the schema of the data that will be provided
// by the health check
func (h *Health) Schema() (*openapi3.SchemaRef, error) {
	return OpenapiFromPerfData[map[string]string](map[string]string{})
}

// RegisterHandler dynamically registers a server handler
func (h *Health) RegisterHandler(ctx context.Context, router *api.RoutingTree) {
	log := logger.FromContext(ctx)
	router.Add(http.MethodGet, h.route, func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte("ok"))
		if err != nil {
			log.Error("Could not write response", "error", err)
		}
	})
}

// DeregisterHandler dynamically deletes the server handler
func (h *Health) DeregisterHandler(_ context.Context, router *api.RoutingTree) {
	router.Remove(http.MethodGet, h.route)
}

// NewHealthMetrics initializes metric collectors of the health check
func newHealthMetrics() healthMetrics {
	return healthMetrics{
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
	log := logger.FromContext(ctx).WithGroup("check")
	log.Debug("Checking health")
	if len(h.config.Targets) == 0 {
		log.Debug("No targets defined")
		return map[string]string{}
	}
	log.Debug("Getting health status for each target in separate routine", "amount", len(h.config.Targets))

	var wg sync.WaitGroup
	var mu sync.Mutex
	results := map[string]string{}

	client := &http.Client{
		Timeout: h.config.Timeout,
	}
	for _, t := range h.config.Targets {
		target := t
		wg.Add(1)
		l := log.With("target", target)

		getHealthRetry := helper.Retry(func(ctx context.Context) error {
			return getHealth(ctx, client, target)
		}, h.config.Retry)

		go func() {
			defer wg.Done()
			state := 1

			l.Debug("Starting retry routine to get health status")
			if err := getHealthRetry(ctx); err != nil {
				state = 0
				l.Warn(fmt.Sprintf("Health check failed after %d retries", h.config.Retry.Count), "error", err)
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
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Warn("Health request was not ok (HTTP Status 200)", "status", resp.Status)
		return fmt.Errorf("request failed, status is %s", resp.Status)
	}

	return nil
}
