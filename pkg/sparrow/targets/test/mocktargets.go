package gitlabmock

import (
	"context"

	"github.com/caas-team/sparrow/pkg/checks"

	"github.com/caas-team/sparrow/internal/logger"
)

// MockTargetManager is a mock implementation of the TargetManager interface
type MockTargetManager struct {
	Targets []checks.GlobalTarget
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

func (m *MockTargetManager) GetTargets() []checks.GlobalTarget {
	log := logger.FromContext(context.Background())
	log.Info("MockGetTargets called, returning", "targets", len(m.Targets))
	return m.Targets
}
