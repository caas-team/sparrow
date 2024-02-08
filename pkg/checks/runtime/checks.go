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
	"github.com/caas-team/sparrow/pkg/checks"
)

// Checks holds all the checks.
type Checks struct {
	checks []checks.Check
}

// Add adds a new check.
func (c *Checks) Add(check checks.Check) {
	c.checks = append(c.checks, check)
}

// Delete deletes a check.
func (c *Checks) Delete(index int) {
	c.checks = append(c.checks[:index], c.checks[index+1:]...)
}

// Iter returns configured checks in an iterable format
func (c *Checks) Iter() []checks.Check {
	return c.checks
}
