// sparrow
// (C) 2024, Deutsche Telekom IT GmbH
//
// Deutsche Telekom IT GmbH and all other contributors /
// copyright owners license this file to you under the Apache
// License, Version 2.0 (the "License"); you may not use this
// file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package factory

import (
	"reflect"
	"testing"
	"time"

	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/checks/health"
	"github.com/caas-team/sparrow/pkg/checks/latency"
	"github.com/caas-team/sparrow/pkg/checks/runtime"
	"github.com/caas-team/sparrow/test"
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
	test.MarkAsShort(t)

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
	_ = res.UpdateConfig(healthCfg)
	return res
}

// newLatencyCheck creates a new latency check with a testing configuration
func newLatencyCheck() checks.Check {
	res := latency.NewCheck()
	_ = res.UpdateConfig(latencyCfg)
	return res
}

func TestNewCheck(t *testing.T) {
	test.MarkAsShort(t)

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
			got, err := newCheck(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("newCheck() error = %v, wantErr %v", err, tt.wantErr)
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
