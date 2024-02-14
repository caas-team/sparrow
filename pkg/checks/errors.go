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

package checks

import (
	"fmt"
)

// ErrConfigMismatch is returned when a configuration is of the wrong type
type ErrConfigMismatch struct {
	Expected string
	Current  string
}

func (e ErrConfigMismatch) Error() string {
	return fmt.Sprintf("config mismatch: expected type %v, got %v", e.Expected, e.Current)
}

// ErrInvalidConfig is returned when a configuration is invalid
type ErrInvalidConfig struct {
	CheckName string
	Field     string
	Reason    string
}

func (e ErrInvalidConfig) Error() string {
	return fmt.Sprintf("invalid configuration field %q in check %q: %s", e.Field, e.CheckName, e.Reason)
}
