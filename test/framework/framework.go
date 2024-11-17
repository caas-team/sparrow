package framework

import (
	"context"
	"testing"

	"github.com/caas-team/sparrow/pkg/config"
	"github.com/caas-team/sparrow/pkg/sparrow"
	"github.com/caas-team/sparrow/test/framework/builder"
)

// Runner is a test runner.
type Runner interface {
	// Run runs the test.
	Run(ctx context.Context) error
}

// Framework is a test framework.
// It provides a way to run various tests.
type Framework struct {
	t *testing.T
}

// NewFramework creates a new test framework.
func New(t *testing.T) *Framework {
	t.Helper()
	return &Framework{t: t}
}

// E2E creates a new end-to-end test.
// If the test is run in short mode, it will be skipped.
func (f *Framework) E2E(t *testing.T, cfg *config.Config) *E2E {
	if cfg == nil {
		cfg = builder.NewSparrowConfig().Config(f.t)
	}

	return &E2E{
		t:       t,
		config:  *cfg,
		sparrow: sparrow.New(cfg),
		checks:  map[string]builder.Check{},
	}
}
