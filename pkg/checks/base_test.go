package checks

import (
	"testing"

	"github.com/caas-team/sparrow/test"
	"github.com/stretchr/testify/assert"
)

func TestBase_Shutdown(t *testing.T) {
	test.MarkAsShort(t)

	tests := []struct {
		name string
		base *Base
	}{
		{
			name: "shutdown",
			base: &Base{Done: make(chan struct{}, 1)},
		},
		{
			name: "already shutdown",
			base: &Base{Done: make(chan struct{}, 1), closed: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.base.closed {
				close(tt.base.Done)
			}
			tt.base.Shutdown()

			if !tt.base.closed {
				t.Error("Base.Shutdown() should close Base.Done")
			}

			assert.Panics(t, func() {
				tt.base.Done <- struct{}{}
			}, "Base.Done should be closed")
		})
	}
}
