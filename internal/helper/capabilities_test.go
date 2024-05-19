package helper

import "testing"

func TestHasCapabilities(t *testing.T) {
	tests := []struct {
		name string
		cap  Capability
		want bool
	}{
		{
			name: "CAP_NET_RAW",
			cap:  CAP_NET_RAW,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasCapabilities(tt.cap); got != tt.want {
				t.Errorf("HasCapabilities() = %v, want %v", got, tt.want)
			}
		})
	}
}
