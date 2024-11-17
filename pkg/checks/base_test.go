package checks

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBase_Shutdown(t *testing.T) {
	tests := []struct {
		name string
		b    *Base
	}{
		{
			name: "shutdown",
			b: &Base{
				Done: make(chan struct{}, 1),
			},
		},
		{
			name: "already shutdown",
			b: &Base{
				Done:   make(chan struct{}, 1),
				closed: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.b.closed {
				close(tt.b.Done)
			}
			tt.b.Shutdown()

			if !tt.b.closed {
				t.Error("Base.Shutdown() should close DoneChan")
			}

			assert.Panics(t, func() {
				tt.b.Done <- struct{}{}
			}, "Base.DoneChan should be closed")
		})
	}
}
