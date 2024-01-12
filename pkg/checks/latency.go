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
	"strconv"
	"sync"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/mitchellh/mapstructure"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/caas-team/sparrow/internal/helper"
	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/api"
)

var _ Check = (*Latency)(nil)

// Latency is a check that measures the latency to an endpoint
type Latency struct {
	CheckBase
	config  LatencyConfig
	metrics latencyMetrics
}

// NewLatencyCheck creates a new instance of the latency check
func NewLatencyCheck() Check {
	return &Latency{
		CheckBase: CheckBase{
			mu:      sync.Mutex{},
			cResult: nil,
			done:    make(chan bool, 1),
		},
		config: LatencyConfig{
			Retry: DefaultRetry,
		},
		metrics: newLatencyMetrics(),
	}
}

// LatencyConfig defines the configuration parameters for a latency check
type LatencyConfig struct {
	Targets  []string           `json:"targets" yaml:"targets"`
	Interval time.Duration      `json:"interval" yaml:"interval"`
	Timeout  time.Duration      `json:"timeout" yaml:"timeout"`
	Retry    helper.RetryConfig `json:"retry" yaml:"retry"`
}

// LatencyResult represents the result of a single latency check for a specific target
type LatencyResult struct {
	Code  int     `json:"code"`
	Error *string `json:"error"`
	Total float64 `json:"total"`
}

// Defined metric collectors of latency check
type latencyMetrics struct {
	latencyDuration  *prometheus.GaugeVec
	latencyCount     *prometheus.CounterVec
	latencyHistogram *prometheus.HistogramVec
}

// Run starts the latency check
func (l *Latency) Run(ctx context.Context) error {
	ctx, cancel := logger.NewContextWithLogger(ctx, "latency")
	defer cancel()
	log := logger.FromContext(ctx)
	log.Info(fmt.Sprintf("Using latency check interval of %s", l.config.Interval.String()))

	for {
		select {
		case <-ctx.Done():
			log.Error("Context canceled", "err", ctx.Err())
			return ctx.Err()
		case <-l.done:
			return nil
		case <-time.After(l.config.Interval):
			res := l.check(ctx)
			errval := ""
			r := Result{
				Data:      res,
				Err:       errval,
				Timestamp: time.Now(),
			}

			l.cResult <- r
			log.Debug("Successfully finished latency check run")
		}
	}
}

func (l *Latency) Startup(ctx context.Context, cResult chan<- Result) error {
	log := logger.FromContext(ctx).WithGroup("latency")
	log.Debug("Starting latency check")

	l.cResult = cResult
	return nil
}

func (l *Latency) Shutdown(_ context.Context) error {
	l.done <- true
	close(l.done)

	return nil
}

func (l *Latency) SetConfig(_ context.Context, config any) error {
	var c LatencyConfig
	if err := mapstructure.Decode(config, &c); err != nil {
		return ErrInvalidConfig
	}
	c.Interval *= time.Second
	c.Retry.Delay *= time.Second
	l.mu.Lock()
	defer l.mu.Unlock()
	l.config = c

	return nil
}

// Schema provides the schema of the data that will be provided
// by the latency check
func (l *Latency) Schema() (*openapi3.SchemaRef, error) {
	return OpenapiFromPerfData(make(map[string]LatencyResult))
}

// RegisterHandler registers a server handler
func (l *Latency) RegisterHandler(_ context.Context, router *api.RoutingTree) {
	router.Add(http.MethodGet, "v1alpha1/latency", l.Handler)
}

// DeregisterHandler deletes the server handler
func (l *Latency) DeregisterHandler(_ context.Context, router *api.RoutingTree) {
	router.Remove(http.MethodGet, "v1alpha1/latency")
}

// Handler defines the server handler for the latency check
func (l *Latency) Handler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// NewLatencyMetrics initializes metric collectors of the latency check
func newLatencyMetrics() latencyMetrics {
	return latencyMetrics{
		latencyDuration: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "sparrow_latency_duration_seconds",
				Help: "Latency with status information of targets",
			},
			[]string{
				"target",
				"status",
			},
		),
		latencyCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "sparrow_latency_count",
				Help: "Count of latency checks done",
			},
			[]string{
				"target",
			},
		),
		latencyHistogram: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "sparrow_latency_duration",
				Help: "Latency of targets in seconds",
			},
			[]string{
				"target",
			},
		),
	}
}

// GetMetricCollectors returns all metric collectors of check
func (l *Latency) GetMetricCollectors() []prometheus.Collector {
	return []prometheus.Collector{
		l.metrics.latencyDuration,
		l.metrics.latencyCount,
		l.metrics.latencyHistogram,
	}
}

// check performs a latency check using a retry function
// to get the latency to all targets
func (l *Latency) check(ctx context.Context) map[string]LatencyResult {
	log := logger.FromContext(ctx).WithGroup("check")
	log.Debug("Checking latency")
	if len(l.config.Targets) == 0 {
		log.Debug("No targets defined")
		return map[string]LatencyResult{}
	}
	log.Debug("Getting latency status for each target in separate routine", "amount", len(l.config.Targets))

	var mu sync.Mutex
	var wg sync.WaitGroup
	results := map[string]LatencyResult{}

	client := &http.Client{
		Timeout: l.config.Timeout * time.Second,
	}
	for _, t := range l.config.Targets {
		target := t
		wg.Add(1)
		lo := log.With("target", target)

		getLatencyRetry := helper.Retry(func(ctx context.Context) error {
			res := getLatency(ctx, client, target)
			mu.Lock()
			defer mu.Unlock()
			results[target] = res

			return nil
		}, l.config.Retry)

		go func() {
			defer wg.Done()

			lo.Debug("Starting retry routine to get latency status")
			if err := getLatencyRetry(ctx); err != nil {
				lo.Error("Error while checking latency", "error", err)
			}

			lo.Debug("Successfully got latency status of target")
			mu.Lock()
			defer mu.Unlock()
			l.metrics.latencyDuration.WithLabelValues(target, strconv.Itoa(results[target].Code)).Set(results[target].Total)
			l.metrics.latencyHistogram.WithLabelValues(target).Observe(results[target].Total)
			l.metrics.latencyCount.WithLabelValues(target).Inc()
		}()
	}

	log.Debug("Waiting for all routines to finish")
	wg.Wait()

	log.Debug("Successfully got latency status from all targets")
	return results
}

// getLatency performs an HTTP get request and returns ok if request succeeds
func getLatency(ctx context.Context, c *http.Client, url string) LatencyResult {
	log := logger.FromContext(ctx).With("url", url)
	var res LatencyResult

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		log.Error("Error while creating request", "error", err)
		errval := err.Error()
		res.Error = &errval
		return res
	}

	start := time.Now()
	resp, err := c.Do(req) //nolint:bodyclose // Closed in defer below
	if err != nil {
		log.Error("Error while checking latency", "error", err)
		errval := err.Error()
		res.Error = &errval
		return res
	}
	end := time.Now()

	res.Code = resp.StatusCode
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	res.Total = end.Sub(start).Seconds()
	return res
}
