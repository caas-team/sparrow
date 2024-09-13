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

package runtime

import (
	"errors"

	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/checks/dns"
	"github.com/caas-team/sparrow/pkg/checks/health"
	"github.com/caas-team/sparrow/pkg/checks/latency"
	"github.com/caas-team/sparrow/pkg/checks/traceroute"
)

// Config holds the runtime configuration
// for the various checks the sparrow supports
type Config struct {
	Health     *health.Config     `yaml:"health" json:"health"`
	Latency    *latency.Config    `yaml:"latency" json:"latency"`
	Dns        *dns.Config        `yaml:"dns" json:"dns"`
	Traceroute *traceroute.Config `yaml:"traceroute" json:"traceroute"`
}

// Empty returns true if no checks are configured
func (c Config) Empty() bool {
	return c.size() == 0
}

func (c Config) Validate() (err error) {
	for _, cfg := range c.Iter() {
		if vErr := cfg.Validate(); vErr != nil {
			err = errors.Join(err, vErr)
		}
	}

	return err
}

// Iter returns configured checks in an iterable format
func (c Config) Iter() []checks.ConfigProvider {
	var configs []checks.ConfigProvider
	if c.Health != nil {
		configs = append(configs, c.Health)
	}
	if c.Latency != nil {
		configs = append(configs, c.Latency)
	}
	if c.Dns != nil {
		configs = append(configs, c.Dns)
	}
	if c.Traceroute != nil {
		configs = append(configs, c.Traceroute)
	}
	return configs
}

// size returns the number of checks configured
func (c Config) size() int {
	size := 0
	if c.HasHealthCheck() {
		size++
	}
	if c.HasLatencyCheck() {
		size++
	}
	if c.HasDNSCheck() {
		size++
	}
	if c.HasTracerouteCheck() {
		size++
	}
	return size
}

// HasHealthCheck returns true if the check has a health check configured
func (c Config) HasHealthCheck() bool {
	return c.Health != nil
}

// HasLatencyCheck returns true if the check has a latency check configured
func (c Config) HasLatencyCheck() bool {
	return c.Latency != nil
}

// HasDNSCheck returns true if the check has a dns check configured
func (c Config) HasDNSCheck() bool {
	return c.Dns != nil
}

// HasTracerouteCheck returns true if the check has a traceroute check configured
func (c Config) HasTracerouteCheck() bool {
	return c.Traceroute != nil
}

// HasCheck returns true if the check has a check with the given name configured
func (c Config) HasCheck(name string) bool {
	switch name {
	case health.CheckName:
		return c.HasHealthCheck()
	case latency.CheckName:
		return c.HasLatencyCheck()
	case dns.CheckName:
		return c.HasDNSCheck()
	case traceroute.CheckName:
		return c.HasTracerouteCheck()
	default:
		return false
	}
}

// For returns the runtime configuration for the check with the given name
func (c Config) For(name string) checks.ConfigProvider {
	switch name {
	case health.CheckName:
		if c.HasHealthCheck() {
			return c.Health
		}
	case latency.CheckName:
		if c.HasLatencyCheck() {
			return c.Latency
		}
	case dns.CheckName:
		if c.HasDNSCheck() {
			return c.Dns
		}
	case traceroute.CheckName:
		if c.HasTracerouteCheck() {
			return c.Traceroute
		}
	}
	return nil
}
