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

package factory

import (
	"errors"

	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/checks/dns"
	"github.com/caas-team/sparrow/pkg/checks/health"
	"github.com/caas-team/sparrow/pkg/checks/latency"
	"github.com/caas-team/sparrow/pkg/checks/runtime"
)

// newCheck creates a new check instance from the given name
func newCheck(cfg checks.Runtime) (checks.Check, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}

	if f, ok := registry[cfg.For()]; ok {
		c := f()
		err := c.SetConfig(cfg)
		return c, err
	}
	return nil, errors.New("unknown check type")
}

// NewChecksFromConfig creates all checks defined provided config
func NewChecksFromConfig(cfg runtime.Config) (map[string]checks.Check, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	result := make(map[string]checks.Check)
	for _, c := range cfg.Iter() {
		check, err := newCheck(c)
		if err != nil {
			return nil, err
		}
		result[check.Name()] = check
	}
	return result, nil
}

// registry is a convenience map to create new checks
var registry = map[string]func() checks.Check{
	health.CheckName:  health.NewCheck,
	latency.CheckName: latency.NewCheck,
	dns.CheckName:     dns.NewCheck,
}
