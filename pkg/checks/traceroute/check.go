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
	"reflect"
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

var _ checks.Check = (*check)(nil)

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

// check is the implementation of the traceroute check.
// It traces the path to a list of targets.
type check struct {
	checks.Base
	config     Config
	traceroute tracerouteFactory
	metrics    metrics
	tracer     trace.Tracer
}

func NewCheck() checks.Check {
	c := &check{
		Base:       checks.NewBase(),
		config:     Config{Retry: checks.DefaultRetry},
		traceroute: TraceRoute,
		metrics:    newMetrics(),
	}
	c.tracer = otel.Tracer(c.Name())
	return c
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
func (ch *check) Run(ctx context.Context, cResult chan checks.ResultDTO) error {
	ctx, cancel := logger.NewContextWithLogger(ctx)
	defer cancel()
	log := logger.FromContext(ctx)

	timer := time.NewTimer(ch.config.Interval)
	log.InfoContext(ctx, "Starting traceroute check", "interval", ch.config.Interval.String())
	for {
		select {
		case <-ctx.Done():
			log.ErrorContext(ctx, "Context canceled", "error", ctx.Err())
			return ctx.Err()
		case <-ch.Done:
			return nil
		case <-timer.C:
			res := ch.check(ctx)
			ch.metrics.MinHops(res)
			cResult <- checks.ResultDTO{
				Name: ch.Name(),
				Result: &checks.Result{
					Data:      res,
					Timestamp: time.Now(),
				},
			}
			log.DebugContext(ctx, "Successfully finished traceroute check run")
			ch.Mutex.Lock()
			timer.Reset(ch.config.Interval)
			ch.Mutex.Unlock()
		}
	}
}

// GetConfig returns the current configuration of the check
func (ch *check) GetConfig() checks.Runtime {
	ch.Mutex.Lock()
	defer ch.Mutex.Unlock()
	return &ch.config
}

func (ch *check) check(ctx context.Context) map[string]result {
	res := make(map[string]result)
	log := logger.FromContext(ctx)
	ch.Mutex.Lock()
	cfg := ch.config
	ch.Mutex.Unlock()

	type internalResult struct {
		addr string
		res  result
	}

	cResult := make(chan internalResult, len(cfg.Targets))
	var wg sync.WaitGroup
	start := time.Now()
	wg.Add(len(cfg.Targets))

	for _, t := range cfg.Targets {
		go func(t Target) {
			defer wg.Done()
			l := log.With("target", t.String())
			l.DebugContext(ctx, "Running traceroute")

			c, span := ch.tracer.Start(ctx, t.String(), trace.WithAttributes(
				attribute.String("target.addr", t.Addr),
				attribute.Int("target.port", t.Port),
				attribute.Stringer("config.interval", cfg.Interval),
				attribute.Stringer("config.timeout", cfg.Timeout),
				attribute.Int("config.max_hops", cfg.MaxHops),
				attribute.Int("config.retry.count", cfg.Retry.Count),
				attribute.Stringer("config.retry.delay", cfg.Retry.Delay),
			))
			defer span.End()

			s := time.Now()
			hops, err := ch.traceroute(c, tracerouteConfig{
				Dest:    t.Addr,
				Port:    t.Port,
				Timeout: cfg.Timeout,
				MaxHops: cfg.MaxHops,
				Rc:      cfg.Retry,
			})
			elapsed := time.Since(s)

			if err != nil {
				l.ErrorContext(ctx, "Error running traceroute", "error", err)
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			} else {
				span.SetStatus(codes.Ok, "success")
			}

			ch.metrics.CheckDuration(t.Addr, elapsed)
			l.DebugContext(ctx, "Ran traceroute", "result", hops, "duration", elapsed)

			res := result{
				Hops:    hops,
				MinHops: cfg.MaxHops,
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

// UpdateConfig updates the configuration of the check.
func (ch *check) UpdateConfig(cfg checks.Runtime) error {
	if c, ok := cfg.(*Config); ok {
		ch.Mutex.Lock()
		defer ch.Mutex.Unlock()
		if c == nil || reflect.DeepEqual(&ch.config, c) {
			return nil
		}

		for _, target := range ch.config.Targets {
			if !slices.Contains(c.Targets, target) {
				err := ch.metrics.Remove(target.Addr)
				if err != nil {
					return err
				}
			}
		}

		ch.config = *c
		return nil
	}

	return checks.ErrConfigMismatch{
		Expected: CheckName,
		Current:  cfg.For(),
	}
}

// Schema returns an openapi3.SchemaRef of the result type returned by the check
func (ch *check) Schema() (*openapi3.SchemaRef, error) {
	return checks.OpenapiFromPerfData(map[string]result{})
}

// GetMetricCollectors allows the check to provide prometheus metric collectors
func (ch *check) GetMetricCollectors() []prometheus.Collector {
	return ch.metrics.List()
}

// Name returns the name of the check
func (*check) Name() string {
	return CheckName
}

// RemoveLabelledMetrics removes the metrics which have the passed
// target as a label
func (ch *check) RemoveLabelledMetrics(target string) error {
	return ch.metrics.Remove(target)
}
