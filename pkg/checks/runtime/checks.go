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
	"slices"
	"sync"

	"github.com/caas-team/sparrow/pkg/checks"
)

// Checks holds all the checks.
type Checks struct {
	mu     sync.RWMutex
	checks []checks.Check
}

// Add adds a new check.
func (c *Checks) Add(check checks.Check) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.checks = append(c.checks, check)
}

// Delete deletes a check.
func (c *Checks) Delete(check checks.Check) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, exist := range c.checks {
		if exist.Name() == check.Name() {
			c.checks = append(c.checks[:i], c.checks[i+1:]...)
			return
		}
	}
}

// Iter returns configured checks in an iterable format
func (c *Checks) Iter() []checks.Check {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return slices.Clone(c.checks)
}
