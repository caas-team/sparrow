package targets

import (
	"context"
	"time"
)

// globalTarget represents a globalTarget that can be checked
type globalTarget struct {
	url      string
	lastSeen time.Time
}

// TargetManager handles the management of globalTargets for
// a Sparrow instance
type TargetManager interface {
	Reconcile(ctx context.Context)
	Register(ctx context.Context)
	GetTargets() []globalTarget
}
