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
	"sync"
	"time"

	"github.com/caas-team/sparrow/internal/helper"
	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	_ checks.Check   = (*DNS)(nil)
	_ checks.Runtime = (*Config)(nil)
)

const CheckName = "dns"

// DNS is a check that resolves the names and addresses
type DNS struct {
	checks.Base[*Config]
	metrics metrics
	client  Resolver
}

// NewCheck creates a new instance of the dns check
func NewCheck() checks.Check {
	return &DNS{
		Base: checks.NewBase(CheckName, &Config{
			Retry: checks.DefaultRetry,
		}),
		metrics: newMetrics(),
		client:  NewResolver(),
	}
}

// result represents the result of a single DNS check for a specific target
type result struct {
	Resolved []string
	Error    *string
	Total    float64
}

// Run starts the dns check
func (d *DNS) Run(ctx context.Context, cResult chan checks.ResultDTO) error {
	return d.StartCheck(ctx, cResult, d.Config.Interval, func(ctx context.Context) any {
		return d.check(ctx)
	})
}

// Schema provides the schema of the data that will be provided
// by the dns check
func (d *DNS) Schema() (*openapi3.SchemaRef, error) {
	return checks.OpenapiFromPerfData(make(map[string]result))
}

// GetMetricCollectors returns all metric collectors of check
func (d *DNS) GetMetricCollectors() []prometheus.Collector {
	return d.metrics.GetCollectors()
}

// check performs DNS checks for all configured targets using a custom net.Resolver.
// Returns a map where each target is associated with its DNS check result.
func (d *DNS) check(ctx context.Context) map[string]result {
	log := logger.FromContext(ctx)
	log.Debug("Checking dns")
	if len(d.Config.Targets) == 0 {
		log.Debug("No targets defined")
		return map[string]result{}
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	results := map[string]result{}

	d.client.SetDialer(&net.Dialer{
		Timeout: d.Config.Timeout,
	})

	log.Debug("Getting dns status for each target in separate routine", "amount", len(d.Config.Targets))
	for _, t := range d.Config.Targets {
		target := t
		wg.Add(1)
		lo := log.With("target", target)

		getDNSRetry := helper.Retry(func(ctx context.Context) error {
			res, err := getDNS(ctx, d.client, target)
			mu.Lock()
			defer mu.Unlock()
			results[target] = res
			if err != nil {
				return err
			}
			return nil
		}, d.Config.Retry)

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
			d.metrics.Set(target, results, float64(status))
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
