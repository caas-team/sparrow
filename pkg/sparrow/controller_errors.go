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

package sparrow

import (
	"fmt"

	"github.com/caas-team/sparrow/pkg/checks"
)

type ErrRunningCheck struct {
	Check checks.Check
	Err   error
}

func (e *ErrRunningCheck) Error() string {
	return fmt.Sprintf("check %s failed: %v", e.Check.Name(), e.Err)
}

type ErrCreateOpenapiSchema struct {
	name string
	err  error
}

func (e ErrCreateOpenapiSchema) Error() string {
	return fmt.Sprintf("failed to get schema for check %s: %v", e.name, e.err)
}
