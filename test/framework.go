package test

import (
	"context"
	"testing"

	"github.com/caas-team/sparrow/pkg/config"
	"github.com/caas-team/sparrow/pkg/sparrow"
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
func NewFramework(t *testing.T) *Framework {
	t.Helper()
	return &Framework{t: t}
}

// Unit creates a new unit test.
// If the test is not run in short mode, it will be skipped.
func (f *Framework) Unit(t *testing.T, run func(context.Context) error) *Unit {
	if !testing.Short() {
		f.t.Skip("skipping unit tests")
		return nil
	}

	return &Unit{t: t, run: run}
}

// E2E creates a new end-to-end test.
// If the test is run in short mode, it will be skipped.
func (f *Framework) E2E(t *testing.T, cfg *config.Config) *E2E {
	if testing.Short() {
		f.t.Skip("skipping e2e tests")
		return nil
	}

	if cfg == nil {
		cfg = NewSparrowConfig().Config(f.t)
	}

	return &E2E{
		t:       t,
		config:  *cfg,
		sparrow: sparrow.New(cfg),
		checks:  map[string]CheckBuilder{},
	}
}
