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

package traceroute

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/caas-team/sparrow/internal/helper"
	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var _ checks.Check = (*Traceroute)(nil)

const CheckName = "traceroute"

type Target struct {
	// The address of the target to traceroute to. Can be a DNS name or an IP address
	Addr string `json:"addr" yaml:"addr" mapstructure:"addr"`
	// The port to traceroute to
	Port int `json:"port" yaml:"port" mapstructure:"port"`
}

func (t Target) String() string {
	return fmt.Sprintf("%s:%d", t.Addr, t.Port)
}

func NewCheck() checks.Check {
	c := &Traceroute{
		CheckBase: checks.CheckBase{
			Mu:       sync.Mutex{},
			DoneChan: make(chan struct{}, 1),
		},
		config:     Config{},
		traceroute: TraceRoute,
		metrics:    newMetrics(),
	}
	c.tracer = otel.Tracer(c.Name())
	return c
}

type Traceroute struct {
	checks.CheckBase
	config     Config
	traceroute tracerouteFactory
	metrics    metrics
	tracer     trace.Tracer
}

type tracerouteConfig struct {
	Dest    string
	Port    int
	Timeout time.Duration
	MaxHops int
	Rc      helper.RetryConfig
}

type tracerouteFactory func(ctx context.Context, cfg tracerouteConfig) (map[int][]Hop, error)

type result struct {
	// The minimum number of hops required to reach the target
	MinHops int `json:"min_hops" yaml:"min_hops" mapstructure:"min_hops"`
	// The path taken to the destination
	Hops map[int][]Hop `json:"hops" yaml:"hops" mapstructure:"hops"`
}

// Run runs the check in a loop sending results to the provided channel
func (tr *Traceroute) Run(ctx context.Context, cResult chan checks.ResultDTO) error {
	ctx, cancel := logger.NewContextWithLogger(ctx)
	defer cancel()
	log := logger.FromContext(ctx)

	log.InfoContext(ctx, "Starting traceroute check", "interval", tr.config.Interval.String())
	for {
		select {
		case <-ctx.Done():
			log.ErrorContext(ctx, "Context canceled", "error", ctx.Err())
			return ctx.Err()
		case <-tr.DoneChan:
			return nil
		case <-time.After(tr.config.Interval):
			res := tr.check(ctx)
			tr.metrics.MinHops(res)
			cResult <- checks.ResultDTO{
				Name: tr.Name(),
				Result: &checks.Result{
					Data:      res,
					Timestamp: time.Now(),
				},
			}
			log.DebugContext(ctx, "Successfully finished traceroute check run")
		}
	}
}

// GetConfig returns the current configuration of the check
func (tr *Traceroute) GetConfig() checks.Runtime {
	tr.Mu.Lock()
	defer tr.Mu.Unlock()
	return &tr.config
}

func (tr *Traceroute) check(ctx context.Context) map[string]result {
	res := make(map[string]result)
	log := logger.FromContext(ctx)

	type internalResult struct {
		addr string
		res  result
	}

	cResult := make(chan internalResult, len(tr.config.Targets))
	var wg sync.WaitGroup
	start := time.Now()
	wg.Add(len(tr.config.Targets))

	for _, t := range tr.config.Targets {
		go func(t Target) {
			defer wg.Done()
			l := log.With("target", t.String())
			l.DebugContext(ctx, "Running traceroute")

			c, span := tr.tracer.Start(ctx, t.String(), trace.WithAttributes(
				attribute.String("target.addr", t.Addr),
				attribute.Int("target.port", t.Port),
				attribute.Stringer("config.interval", tr.config.Interval),
				attribute.Stringer("config.timeout", tr.config.Timeout),
				attribute.Int("config.max_hops", tr.config.MaxHops),
				attribute.Int("config.retry.count", tr.config.Retry.Count),
				attribute.Stringer("config.retry.delay", tr.config.Retry.Delay),
			))
			defer span.End()

			s := time.Now()
			hops, err := tr.traceroute(c, tracerouteConfig{
				Dest:    t.Addr,
				Port:    t.Port,
				Timeout: tr.config.Timeout,
				MaxHops: tr.config.MaxHops,
				Rc:      tr.config.Retry,
			})
			elapsed := time.Since(s)

			if err != nil {
				l.ErrorContext(ctx, "Error running traceroute", "error", err)
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			} else {
				span.SetStatus(codes.Ok, "success")
			}

			tr.metrics.CheckDuration(t.Addr, elapsed)
			l.DebugContext(ctx, "Ran traceroute", "result", hops, "duration", elapsed)

			res := result{
				Hops:    hops,
				MinHops: tr.config.MaxHops,
			}
			for ttl, hop := range hops {
				for _, attempt := range hop {
					if attempt.Reached && attempt.Ttl < res.MinHops {
						res.MinHops = ttl
					}
				}
			}

			span.AddEvent("Traceroute completed", trace.WithAttributes(
				attribute.Int("result.min_hops", res.MinHops),
				attribute.Int("result.hop_count", len(hops)),
				attribute.Stringer("result.elapsed_time", elapsed),
			))

			cResult <- internalResult{addr: t.Addr, res: res}
		}(t)
	}

	wg.Wait()
	close(cResult)

	for r := range cResult {
		res[r.addr] = r.res
	}

	elapsed := time.Since(start)
	log.InfoContext(ctx, "Finished traceroute check", "duration", elapsed)
	return res
}

// Shutdown is called once when the check is unregistered or sparrow shuts down
func (tr *Traceroute) Shutdown() {
	tr.DoneChan <- struct{}{}
	close(tr.DoneChan)
}

// UpdateConfig is called once when the check is registered
// This is also called while the check is running, if the remote config is updated
// This should return an error if the config is invalid
func (tr *Traceroute) UpdateConfig(cfg checks.Runtime) error {
	if c, ok := cfg.(*Config); ok {
		tr.Mu.Lock()
		defer tr.Mu.Unlock()

		for _, target := range tr.config.Targets {
			if !slices.Contains(c.Targets, target) {
				err := tr.metrics.Remove(target.Addr)
				if err != nil {
					return err
				}
			}
		}

		tr.config = *c
		return nil
	}

	return checks.ErrConfigMismatch{
		Expected: CheckName,
		Current:  cfg.For(),
	}
}

// Schema returns an openapi3.SchemaRef of the result type returned by the check
func (tr *Traceroute) Schema() (*openapi3.SchemaRef, error) {
	return checks.OpenapiFromPerfData(map[string]result{})
}

// GetMetricCollectors allows the check to provide prometheus metric collectors
func (tr *Traceroute) GetMetricCollectors() []prometheus.Collector {
	return tr.metrics.List()
}

// Name returns the name of the check
func (tr *Traceroute) Name() string {
	return CheckName
}

// RemoveLabelledMetrics removes the metrics which have the passed
// target as a label
func (tr *Traceroute) RemoveLabelledMetrics(target string) error {
	return tr.metrics.Remove(target)
}
