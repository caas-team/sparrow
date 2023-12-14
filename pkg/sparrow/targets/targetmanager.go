package targets

import (
	"context"

	"github.com/caas-team/sparrow/pkg/checks"
)

// TargetManager handles the management of globalTargets for
// a Sparrow instance
type TargetManager interface {
	// Reconcile fetches the global targets from the configured
	// endpoint and updates the local state
	Reconcile(ctx context.Context)
	// GetTargets returns the current global targets
	GetTargets() []checks.GlobalTarget
}
