package gitlabmock

import (
	"context"

	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/checks/config"
)

// MockTargetManager is a mock implementation of the TargetManager interface
type MockTargetManager struct {
	Targets []config.GlobalTarget
}

func (m *MockTargetManager) Reconcile(ctx context.Context) {
	log := logger.FromContext(ctx)
	log.Info("MockReconcile called")
}

func (m *MockTargetManager) Shutdown(ctx context.Context) error {
	log := logger.FromContext(ctx)
	log.Info("MockShutdown called")
	return nil
}

func (m *MockTargetManager) GetTargets() []config.GlobalTarget {
	log := logger.FromContext(context.Background())
	log.Info("MockGetTargets called, returning", "targets", len(m.Targets))
	return m.Targets
}
