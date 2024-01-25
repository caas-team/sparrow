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

package dns

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/caas-team/sparrow/internal/helper"
	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/api"
	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/checks/errors"
	"github.com/caas-team/sparrow/pkg/checks/types"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/prometheus/client_golang/prometheus"
)

const route = "dns"

var _ checks.Check = (*DNS)(nil)

// DNS is a check that resolves the names and addresses
type DNS struct {
	types.CheckBase
	config  config
	metrics metrics
	client  Resolver
}

// NewCheck creates a new instance of the dns check
func NewCheck() checks.Check {
	return &DNS{
		CheckBase: types.CheckBase{
			Mu:      sync.Mutex{},
			CResult: nil,
			Done:    make(chan bool, 1),
		},
		config: config{
			Retry: types.DefaultRetry,
		},
		metrics: newMetrics(),
		client:  NewResolver(),
	}
}

// config defines the configuration parameters for a dns check
type config struct {
	Targets  []string           `json:"targets" yaml:"targets" mapstructure:"targets"`
	Interval time.Duration      `json:"interval" yaml:"interval" mapstructure:"interval"`
	Timeout  time.Duration      `json:"timeout" yaml:"timeout" mapstructure:"timeout"`
	Retry    helper.RetryConfig `json:"retry" yaml:"retry" mapstructure:"retry"`
}

// Result represents the result of a single DNS check for a specific target
type Result struct {
	Resolved []string
	Error    *string
	Total    float64
}

// metrics defines the metric collectors of the DNS check
type metrics struct {
	// GaugeVec for tracking the duration of each DNS resolution
	duration *prometheus.GaugeVec
	// CounterVec for counting the total number of DNS checks
	count *prometheus.CounterVec
	// HistogramVec for detailed distribution of resolution times
	histogram *prometheus.HistogramVec
}

// Run starts the dns check
func (d *DNS) Run(ctx context.Context) error {
	ctx, cancel := logger.NewContextWithLogger(ctx)
	defer cancel()
	log := logger.FromContext(ctx)
	log.Info("Starting dns check", "interval", d.config.Interval.String())

	for {
		select {
		case <-ctx.Done():
			log.Error("Context canceled", "err", ctx.Err())
			return ctx.Err()
		case <-d.Done:
			return nil
		case <-time.After(d.config.Interval):
			res := d.check(ctx)
			errval := ""
			r := types.Result{
				Data:      res,
				Err:       errval,
				Timestamp: time.Now(),
			}

			d.CResult <- r
			log.Debug("Successfully finished dns check run")
		}
	}
}

func (d *DNS) Startup(ctx context.Context, cResult chan<- types.Result) error {
	log := logger.FromContext(ctx)
	log.Debug("Initializing dns check")

	d.CResult = cResult
	return nil
}

func (d *DNS) Shutdown(_ context.Context) error {
	d.Done <- true
	close(d.Done)

	return nil
}

func (d *DNS) SetConfig(ctx context.Context, conf any) error {
	log := logger.FromContext(ctx)

	c, err := helper.Decode[config](conf)
	if err != nil {
		log.Error("Failed to decode dns config", "error", err)
		return errors.ErrInvalidConfig
	}

	d.Mu.Lock()
	defer d.Mu.Unlock()
	d.config = c

	return nil
}

// Schema provides the schema of the data that will be provided
// by the dns check
func (d *DNS) Schema() (*openapi3.SchemaRef, error) {
	return checks.OpenapiFromPerfData(make(map[string]Result))
}

// RegisterHandler registers a server handler
func (d *DNS) RegisterHandler(_ context.Context, router *api.RoutingTree) {
	router.Add(http.MethodGet, route, d.Handler)
}

// DeregisterHandler deletes the server handler
func (d *DNS) DeregisterHandler(_ context.Context, router *api.RoutingTree) {
	router.Remove(http.MethodGet, route)
}

// Handler defines the server handler for the dns check
func (d *DNS) Handler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// newMetrics initializes metric collectors of the dns check
func newMetrics() metrics {
	return metrics{
		duration: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "sparrow_dns_duration_seconds",
				Help: "Duration of DNS resolution attempts in seconds.",
			},
			[]string{"target"},
		),
		count: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "sparrow_dns_check_count",
				Help: "Total number of DNS checks performed.",
			},
			[]string{"target", "status"},
		),
		histogram: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "sparrow_dns_response_time_seconds",
				Help: "Histogram of response times for DNS checks in seconds.",
			},
			[]string{"target"},
		),
	}
}

// GetMetricCollectors returns all metric collectors of check
func (d *DNS) GetMetricCollectors() []prometheus.Collector {
	return []prometheus.Collector{
		d.metrics.duration,
		d.metrics.count,
		d.metrics.histogram,
	}
}

// check performs DNS checks for all configured targets using a custom net.Resolver.
// Returns a map where each target is associated with its DNS check Result.
func (d *DNS) check(ctx context.Context) map[string]Result {
	log := logger.FromContext(ctx)
	log.Debug("Checking dns")
	if len(d.config.Targets) == 0 {
		log.Debug("No targets defined")
		return map[string]Result{}
	}
	log.Debug("Getting dns status for each target in separate routine", "amount", len(d.config.Targets))

	var mu sync.Mutex
	var wg sync.WaitGroup
	results := map[string]Result{}

	d.client.SetDialer(&net.Dialer{
		Timeout: d.config.Timeout,
	})

	for _, t := range d.config.Targets {
		target := t
		wg.Add(1)
		lo := log.With("target", target)

		getDNSRetry := helper.Retry(func(ctx context.Context) error {
			res := getDNS(ctx, d.client, target)
			mu.Lock()
			defer mu.Unlock()
			results[target] = res

			d.metrics.duration.WithLabelValues(target).Set(results[target].Total)
			d.metrics.histogram.WithLabelValues(target).Observe(results[target].Total)
			d.metrics.count.WithLabelValues(target, "success").Inc()

			return nil
		}, d.config.Retry)

		go func() {
			defer wg.Done()

			lo.Debug("Starting retry routine to get dns status")
			if err := getDNSRetry(ctx); err != nil {
				lo.Error("Error while checking dns", "error", err)
				d.metrics.count.WithLabelValues(target, "failure").Inc()
			}
			lo.Debug("Successfully got dns status of target")
		}()
	}

	log.Debug("Waiting for all routines to finish")
	wg.Wait()

	log.Debug("Successfully resolved names/addresses from all targets")
	return results
}

// getDNS performs a DNS resolution for the given address using the specified net.Resolver.
// If the address is an IP address, LookupAddr is used to perform a reverse DNS lookup.
// If the address is a hostname, LookupHost is used to find its IP addresses.
// Returns a Result struct containing the outcome of the DNS query.
func getDNS(ctx context.Context, c Resolver, address string) Result {
	log := logger.FromContext(ctx).With("url", address)
	var res Result

	var lookupFunc func(context.Context, string) ([]string, error)
	ip := net.ParseIP(address)
	if ip != nil {
		lookupFunc = c.LookupAddr
	} else {
		lookupFunc = c.LookupHost
	}

	start := time.Now()
	resp, err := lookupFunc(ctx, address)
	if err != nil {
		log.Error("Error while checking dns", "error", err)
		errval := err.Error()
		res.Error = &errval
		return res
	}
	rtt := time.Since(start).Seconds()

	res.Resolved = resp
	res.Total = rtt

	return res
}