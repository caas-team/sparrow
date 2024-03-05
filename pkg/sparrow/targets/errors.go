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

package targets

import "errors"

var (
	// ErrInvalidCheckInterval is returned when the check interval is invalid
	ErrInvalidCheckInterval = errors.New("invalid check interval")
	// ErrInvalidRegistrationInterval is returned when the registration interval is invalid
	ErrInvalidRegistrationInterval = errors.New("invalid registration interval")
	// ErrInvalidUnhealthyThreshold is returned when the unhealthy threshold is invalid
	ErrInvalidUnhealthyThreshold = errors.New("invalid unhealthy threshold")
	// ErrInvalidUpdateInterval is returned when the update interval is invalid
	ErrInvalidUpdateInterval = errors.New("invalid update interval")
)
