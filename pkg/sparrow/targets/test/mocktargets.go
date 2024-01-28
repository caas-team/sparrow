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

package gitlabmock

import (
	"context"

	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/checks/types"
)

// MockTargetManager is a mock implementation of the TargetManager interface
type MockTargetManager struct {
	Targets []types.GlobalTarget
}

func (m *MockTargetManager) Reconcile(ctx context.Context) error {
	log := logger.FromContext(ctx)
	log.Info("MockReconcile called")
	return nil
}

func (m *MockTargetManager) Shutdown(ctx context.Context) error {
	log := logger.FromContext(ctx)
	log.Info("MockShutdown called")
	return nil
}

func (m *MockTargetManager) GetTargets() []types.GlobalTarget {
	log := logger.FromContext(context.Background())
	log.Info("MockGetTargets called, returning", "targets", len(m.Targets))
	return m.Targets
}
