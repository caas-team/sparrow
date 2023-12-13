package targets

import (
	"context"

	"github.com/caas-team/sparrow/pkg/checks"
)

// TargetManager handles the management of globalTargets for
// a Sparrow instance
type TargetManager interface {
	Reconcile(ctx context.Context)
	GetTargets() []checks.GlobalTarget
}
