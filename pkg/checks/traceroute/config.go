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
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/caas-team/sparrow/internal/helper"
	"github.com/caas-team/sparrow/pkg/checks"
)

// Config is the configuration for the traceroute check
type Config struct {
	// Targets is a list of targets to traceroute to
	Targets []Target `json:"targets" yaml:"targets" mapstructure:"targets"`
	// Retry defines if and how to retry a target
	Retry helper.RetryConfig `json:"retry" yaml:"retry" mapstructure:"retry"`
	// MaxHops is the maximum number of hops to try before giving up
	MaxHops int `json:"maxHops" yaml:"maxHops" mapstructure:"maxHops"`
	// Interval is the time to wait between check iterations
	Interval time.Duration `json:"interval" yaml:"interval" mapstructure:"interval"`
	// Timeout is the maximum time to wait for a response from a hop
	Timeout time.Duration `json:"timeout" yaml:"timeout" mapstructure:"timeout"`
}

func (c *Config) For() string {
	return CheckName
}

func (c *Config) Validate() error {
	if c.Timeout <= 0 {
		return checks.ErrInvalidConfig{CheckName: CheckName, Field: "traceroute.timeout", Reason: "must be greater than 0"}
	}
	if c.Interval <= 0 {
		return checks.ErrInvalidConfig{CheckName: CheckName, Field: "traceroute.interval", Reason: "must be greater than 0"}
	}

	for i, t := range c.Targets {
		ip := net.ParseIP(t.Addr)
		if ip != nil {
			continue
		}

		_, err := url.Parse(t.Addr)
		if err != nil {
			return checks.ErrInvalidConfig{CheckName: CheckName, Field: fmt.Sprintf("traceroute.targets[%d].addr", i), Reason: "invalid url or ip"}
		}
	}
	return nil
}
