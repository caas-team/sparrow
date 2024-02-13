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

package health

import (
	"fmt"
	"net/url"
	"time"

	"github.com/caas-team/sparrow/internal/helper"
	"github.com/caas-team/sparrow/pkg/checks"
)

const (
	minInterval = 100 * time.Millisecond
	minTimeout  = 1 * time.Second
)

// Config defines the configuration parameters for a health check
type Config struct {
	Targets  []string           `json:"targets,omitempty" yaml:"targets,omitempty"`
	Interval time.Duration      `json:"interval" yaml:"interval"`
	Timeout  time.Duration      `json:"timeout" yaml:"timeout"`
	Retry    helper.RetryConfig `json:"retry" yaml:"retry"`
}

// For returns the name of the check
func (c *Config) For() string {
	return CheckName
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	for _, t := range c.Targets {
		u, err := url.Parse(t)
		if err != nil {
			return checks.ErrInvalidConfig{CheckName: c.For(), Field: "targets", Reason: "invalid target URL"}
		}

		if u.Scheme != "https" && u.Scheme != "http" {
			return checks.ErrInvalidConfig{CheckName: c.For(), Field: "targets", Reason: "target URLs must start with 'https://' or 'http://'"}
		}
	}

	if c.Interval < minInterval {
		return checks.ErrInvalidConfig{CheckName: c.For(), Field: "interval", Reason: fmt.Sprintf("interval must be at least %v", minInterval)}
	}

	if c.Timeout < minTimeout {
		return checks.ErrInvalidConfig{CheckName: c.For(), Field: "timeout", Reason: fmt.Sprintf("timeout must be at least %v", minTimeout)}
	}

	return nil
}
