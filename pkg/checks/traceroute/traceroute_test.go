package traceroute

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/caas-team/sparrow/internal/traceroute"
	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/google/go-cmp/cmp"
)

func TestTraceroute_Check(t *testing.T) {
	tests := []struct {
		name       string
		targets    []Target
		tracerFunc func(ctx context.Context, addr string, port uint16) ([]traceroute.Hop, error)
		want       map[string]result
		wantErr    bool
	}{
		{
			name: "single successful traceroute",
			targets: []Target{
				{Addr: "example.com", Port: 80},
			},
			tracerFunc: func(ctx context.Context, addr string, port uint16) ([]traceroute.Hop, error) {
				return []traceroute.Hop{{IP: net.ParseIP("192.168.1.1"), Duration: (10 * time.Millisecond).Seconds()}}, nil
			},
			want: map[string]result{
				"example.com:80": {
					Hops: []traceroute.Hop{{IP: net.ParseIP("192.168.1.1"), Duration: (10 * time.Millisecond).Seconds()}},
				},
			},
		},
		{
			name: "traceroute with error",
			targets: []Target{
				{Addr: "example.com", Port: 80},
			},
			tracerFunc: func(ctx context.Context, addr string, port uint16) ([]traceroute.Hop, error) {
				return nil, errors.New("traceroute error")
			},
			want: map[string]result{
				"example.com:80": {
					Hops: nil,
				},
			},
			wantErr: true,
		},
		{
			name: "multiple traceroutes",
			targets: []Target{
				{Addr: "example.com", Port: 80},
				{Addr: "test.com", Port: 80},
			},
			tracerFunc: func(ctx context.Context, addr string, port uint16) ([]traceroute.Hop, error) {
				if addr == "example.com" {
					return []traceroute.Hop{{IP: net.ParseIP("192.168.1.1"), Duration: (10 * time.Millisecond).Seconds()}}, nil
				}
				return []traceroute.Hop{{IP: net.ParseIP("192.168.1.2"), Duration: (20 * time.Millisecond).Seconds()}}, nil
			},
			want: map[string]result{
				"example.com:80": {
					Hops: []traceroute.Hop{{IP: net.ParseIP("192.168.1.1"), Duration: (10 * time.Millisecond).Seconds()}},
				},
				"test.com:80": {
					Hops: []traceroute.Hop{{IP: net.ParseIP("192.168.1.2"), Duration: (20 * time.Millisecond).Seconds()}},
				},
			},
		},
		{
			name:    "no targets defined",
			targets: []Target{},
			tracerFunc: func(ctx context.Context, addr string, port uint16) ([]traceroute.Hop, error) {
				t.Error("traceroute.Run should not be called")
				return nil, nil
			},
			want: map[string]result{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &Traceroute{
				Base: checks.NewBase(CheckName, &Config{
					Targets: tt.targets,
					Retry:   checks.DefaultRetry,
				}),
				tracer: &traceroute.TracerMock{
					RunFunc: tt.tracerFunc,
				},
				metrics: newMetrics(),
			}

			results := tr.check(context.Background())
			for k, r := range results {
				if r.Duration == 0 {
					t.Error("expected duration to be set")
				}
				if !cmp.Equal(tt.want[k].Hops, r.Hops) {
					t.Error(cmp.Diff(tt.want[k].Hops, r.Hops))
				}
			}

			wantCalls := len(tt.targets)
			if tt.wantErr {
				wantCalls *= (tr.Config.Retry.Count + 1)
			}
			if len(tr.tracer.(*traceroute.TracerMock).RunCalls()) != wantCalls {
				t.Errorf("expected %d calls to tracer.Run, got %d", wantCalls, len(tr.tracer.(*traceroute.TracerMock).RunCalls()))
			}
		})
	}
}
