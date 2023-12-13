package targets

import (
	"context"
	"time"
)

// globalTarget represents a globalTarget that can be checked
type globalTarget struct {
	Url      string    `json:"url"`
	LastSeen time.Time `json:"lastSeen"`
}

// TargetManager handles the management of globalTargets for
// a Sparrow instance
type TargetManager interface {
	Reconcile(ctx context.Context)
	GetTargets() []globalTarget
}
