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

type RuntimeConfig struct {
	Checks Checks `yaml:"checks" json:"checks"`
}

// Empty returns true if no checks are configured
func (c RuntimeConfig) Empty() bool {
	return c.Checks.Empty()
}

// Checks holds the available check configurations
type Checks struct {
	Health  *HealthConfig  `yaml:"health" json:"health"`
	Latency *LatencyConfig `yaml:"latency" json:"latency"`
}

// Iter returns configured checks in an iterable format
func (c Checks) Iter() []Config {
	var configs []Config
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
