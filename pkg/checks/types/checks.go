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

package types

import (
	"errors"
	"time"

	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/checks/health"
	"github.com/caas-team/sparrow/pkg/checks/latency"
)

// GlobalTarget includes the basic information regarding
// other Sparrow instances, which this Sparrow can communicate with.
type GlobalTarget struct {
	Url      string    `json:"url"`
	LastSeen time.Time `json:"lastSeen"`
}

// New creates a new check instance from the given name
func New(cfg checks.Runtime) (checks.Check, error) {
	if f, ok := RegisteredChecks[cfg.For()]; ok {
		c := f()
		err := c.SetConfig(cfg)
		return f(), err
	}
	return nil, errors.New("unknown check type")
}

// NewChecksFromConfig creates all checks defined provided config
func NewChecksFromConfig(cfg RuntimeConfig) (map[string]checks.Check, error) {
	result := make(map[string]checks.Check)
	for _, c := range cfg.Checks.Iter() {
		check, err := New(c)
		if err != nil {
			return nil, err
		}
		result[check.Name()] = check
	}
	return result, nil
}

// RegisteredChecks will be registered in this map
// The key is the name of the Check
// The name needs to map the configuration item key
var RegisteredChecks = map[string]func() checks.Check{
	"health":  health.NewCheck,
	"latency": latency.NewCheck,
}

type RuntimeConfig struct {
	Checks Checks `yaml:"checks" json:"checks"`
}

// Empty returns true if no checks are configured
func (c RuntimeConfig) Empty() bool {
	return c.Checks.Empty()
}

// Checks holds the available check configurations
type Checks struct {
	Health  *health.Config  `yaml:"health" json:"health"`
	Latency *latency.Config `yaml:"latency" json:"latency"`
}

// Iter returns configured checks in an iterable format
func (c Checks) Iter() []checks.Runtime {
	var configs []checks.Runtime
	if c.Health != nil {
		configs = append(configs, c.Health)
	}
	if c.Latency != nil {
		configs = append(configs, c.Latency)
	}
	return configs
}

// Validate validates the checks configuration
func (c Checks) Validate() error {
	if c.Health != nil {
		if err := c.Health.Validate(); err != nil {
			return err
		}
	}
	if c.Latency != nil {
		if err := c.Latency.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// Empty returns true if no checks are configured
func (c Checks) Empty() bool {
	return c.Size() == 0
}

// Size returns the number of checks configured
func (c Checks) Size() int {
	size := 0
	if c.Health != nil {
		size++
	}
	if c.Latency != nil {
		size++
	}
	return size
}

// HasHealthCheck returns true if the check has a health check configured
func (c Checks) HasHealthCheck() bool {
	return c.Health != nil
}

// HasLatencyCheck returns true if the check has a latency check configured
func (c Checks) HasLatencyCheck() bool {
	return c.Latency != nil
}

// HasCheck returns true if the check has a check with the given name configured
func (c Checks) HasCheck(name string) bool {
	switch name {
	case "health":
		return c.HasHealthCheck()
	case "latency":
		return c.HasLatencyCheck()
	default:
		return false
	}
}
