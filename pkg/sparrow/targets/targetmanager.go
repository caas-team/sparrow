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

import (
	"context"

	"github.com/caas-team/sparrow/pkg/checks/types"
)

// TargetManager handles the management of globalTargets for
// a Sparrow instance
type TargetManager interface {
	// Reconcile fetches the global targets from the configured
	// endpoint and updates the local state
	Reconcile(ctx context.Context) error
	// GetTargets returns the current global targets
	GetTargets() []types.GlobalTarget
	// Shutdown shuts down the target manager
	// and unregisters the instance as a global target
	Shutdown(ctx context.Context) error
}
