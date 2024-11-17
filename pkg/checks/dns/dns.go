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
	"reflect"
	"slices"
	"sync"
	"time"

	"github.com/caas-team/sparrow/internal/helper"
	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	_ checks.Check   = (*check)(nil)
	_ checks.Runtime = (*Config)(nil)
)

const CheckName = "dns"

// check is the implementation of the dns check.
// It resolves DNS names and IP addresses for a list of targets.
type check struct {
	checks.Base
	config  Config
	metrics metrics
	client  Resolver
}

func (ch *check) GetConfig() checks.Runtime {
	ch.Mutex.Lock()
	defer ch.Mutex.Unlock()
	return &ch.config
}

func (*check) Name() string {
	return CheckName
}

// NewCheck creates a new instance of the dns check
func NewCheck() checks.Check {
	return &check{
		Base: checks.NewBase(),
		config: Config{
			Retry: checks.DefaultRetry,
		},
		metrics: newMetrics(),
		client:  newResolver(),
	}
}

// result represents the result of a single DNS check for a specific target
type result struct {
	Resolved []string `json:"resolved"`
	Error    *string  `json:"error"`
	Total    float64  `json:"total"`
}

// Run starts the dns check
func (ch *check) Run(ctx context.Context, cResult chan checks.ResultDTO) error {
	ctx, cancel := logger.NewContextWithLogger(ctx)
	defer cancel()
	log := logger.FromContext(ctx)

	ticker := time.NewTicker(ch.config.Interval)
	log.InfoContext(ctx, "Starting dns check", "interval", ch.config.Interval.String())
	for {
		select {
		case <-ctx.Done():
			log.ErrorContext(ctx, "Context canceled", "err", ctx.Err())
			return ctx.Err()
		case <-ch.Done:
			return nil
		case <-ch.Update:
			ch.Mutex.Lock()
			ticker.Stop()
			ticker = time.NewTicker(ch.config.Interval)
			log.DebugContext(ctx, "Interval of dns check updated", "interval", ch.config.Interval.String())
			ch.Mutex.Unlock()
		case <-ticker.C:
			res := ch.check(ctx)
			cResult <- checks.ResultDTO{
				Name: ch.Name(),
				Result: &checks.Result{
					Data:      res,
					Timestamp: time.Now(),
				},
			}
			log.DebugContext(ctx, "Successfully finished dns check run")
		}
	}
}

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

// Schema provides the schema of the data that will be provided
// by the dns check
func (ch *check) Schema() (*openapi3.SchemaRef, error) {
	return checks.OpenapiFromPerfData(map[string]result{})
}

// GetMetricCollectors returns all metric collectors of check
func (ch *check) GetMetricCollectors() []prometheus.Collector {
	return ch.metrics.GetCollectors()
}

// RemoveLabelledMetrics removes the metrics which have the passed
// target as a label
func (ch *check) RemoveLabelledMetrics(target string) error {
	return ch.metrics.Remove(target)
}

// check performs DNS checks for all configured targets using a custom net.Resolver.
// Returns a map where each target is associated with its DNS check result.
func (ch *check) check(ctx context.Context) map[string]result {
	log := logger.FromContext(ctx)
	log.Debug("Checking dns")
	ch.Mutex.Lock()
	cfg := ch.config
	ch.Mutex.Unlock()

	if len(cfg.Targets) == 0 {
		log.Debug("No targets defined")
		return map[string]result{}
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	results := map[string]result{}

	ch.client.SetDialer(&net.Dialer{
		Timeout: cfg.Timeout,
	})

	log.Debug("Getting dns status for each target in separate routine", "amount", len(cfg.Targets))
	for _, t := range cfg.Targets {
		target := t
		wg.Add(1)
		lo := log.With("target", target)

		getDNSRetry := helper.Retry(func(ctx context.Context) error {
			res, err := getDNS(ctx, ch.client, target)
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
			status := 1

			lo.Debug("Starting retry routine to get dns status")
			if err := getDNSRetry(ctx); err != nil {
				status = 0
				lo.Warn("Error while looking up address", "error", err)
			}
			lo.Debug("DNS check completed for target")

			mu.Lock()
			defer mu.Unlock()
			ch.metrics.Set(target, results, float64(status))
		}()
	}
	wg.Wait()

	log.Debug("Successfully resolved names/addresses from all targets")
	return results
}

// getDNS performs a DNS resolution for the given address using the specified net.Resolver.
// If the address is an IP address, LookupAddr is used to perform a reverse DNS lookup.
// If the address is a hostname, LookupHost is used to find its IP addresses.
// Returns a result struct containing the outcome of the DNS query.
func getDNS(ctx context.Context, c Resolver, address string) (result, error) {
	log := logger.FromContext(ctx).With("address", address)
	var res result

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
		log.Error("Error while looking up address", "error", err)
		errval := err.Error()
		res.Error = &errval
		return res, err
	}
	rtt := time.Since(start).Seconds()

	res.Resolved = resp
	res.Total = rtt

	return res, nil
}
