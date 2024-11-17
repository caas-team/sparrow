package test

import (
	"context"
	"testing"
)

var _ Runner = (*Unit)(nil)

// Unit is a unit test.
type Unit struct {
	t   *testing.T
	run func(context.Context) error
}

// Run runs the test.
func (t *Unit) Run(ctx context.Context) error {
	return t.run(ctx)
}
