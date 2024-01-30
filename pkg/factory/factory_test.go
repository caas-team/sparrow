package factory

import (
	"reflect"
	"testing"
	"time"

	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/checks/health"
	"github.com/caas-team/sparrow/pkg/checks/latency"
	"github.com/caas-team/sparrow/pkg/checks/runtime"
)

var (
	latencyCfg = &latency.Config{
		Targets:  []string{"http://localhost:8080/health"},
		Interval: 1 * time.Second,
		Timeout:  1 * time.Second,
	}
	healthCfg = &health.Config{
		Targets:  []string{"http://localhost:8080/health"},
		Interval: 1 * time.Second,
		Timeout:  1 * time.Second,
	}
)

func TestNewChecksFromConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     runtime.Config
		want    map[string]checks.Check
		wantErr bool
	}{
		{
			name: "empty config",
			cfg:  runtime.Config{},
			want: map[string]checks.Check{},
		},
		{
			name: "healthcheck",
			cfg: runtime.Config{
				Health: healthCfg,
			},

			want: map[string]checks.Check{
				"health": newHealthCheck(),
			},
		},
		{
			name: "latency",
			cfg: runtime.Config{
				Latency: latencyCfg,
			},

			want: map[string]checks.Check{
				"latency": newLatencyCheck(),
			},
		},
		{
			name: "multiple checks",
			cfg: runtime.Config{
				Health:  healthCfg,
				Latency: latencyCfg,
			},

			want: map[string]checks.Check{
				"health":  newHealthCheck(),
				"latency": newLatencyCheck(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewChecksFromConfig(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewChecksFromConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// assert the configurations of the checks are equal -
			// the checks themselves cannot be compared because of the done channels
			for name, check := range got {
				if !reflect.DeepEqual(check.GetConfig(), tt.want[name].GetConfig()) {
					t.Errorf("NewChecksFromConfig() got = %v, want %v", check.GetConfig(), tt.want[name].GetConfig())
				}
			}
		})
	}
}

// newHealthCheck creates a new health check with a testing configuration
func newHealthCheck() checks.Check {
	res := health.NewCheck()
	_ = res.SetConfig(healthCfg)
	return res
}

// newLatencyCheck creates a new latency check with a testing configuration
func newLatencyCheck() checks.Check {
	res := latency.NewCheck()
	_ = res.SetConfig(latencyCfg)
	return res
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		cfg     checks.Runtime
		want    checks.Check
		wantErr bool
	}{
		{
			name:    "empty config",
			cfg:     nil,
			want:    nil,
			wantErr: true,
		},
		{
			name:    "healthcheck",
			cfg:     healthCfg,
			want:    newHealthCheck(),
			wantErr: false,
		},
		{
			name:    "latency",
			cfg:     latencyCfg,
			want:    newLatencyCheck(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got == nil && tt.want == nil {
				return
			}

			if !reflect.DeepEqual(got.GetConfig(), tt.want.GetConfig()) {
				t.Errorf("Check config got = %v, want %v", got.GetConfig(), tt.want.GetConfig())
			}

			if got.Name() != tt.want.Name() {
				t.Errorf("Check name got = %v, want %v", got.Name(), tt.want.Name())
			}
		})
	}
}
