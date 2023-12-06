// sparrow
// (C) 2023, Deutsche Telekom IT GmbH
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

package checks

import (
	"context"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

// Ensure that Health implements the Check interface
var _ Check = (*Health)(nil)

func TestHealth_SetConfig(t *testing.T) {
	tests := []struct {
		name           string
		inputConfig    any
		expectedConfig HealthConfig
		wantErr        bool
	}{
		{
			name: "simple config",
			inputConfig: map[string]any{
				"enabled": true,
				"targets": []any{
					"test",
				},
				"healthEndpoint": true,
			},
			expectedConfig: HealthConfig{
				Enabled: true,
				Targets: []string{
					"test",
				},
				HealthEndpoint: true,
			},
			wantErr: false,
		},
		{
			name: "missing config field",
			inputConfig: map[string]any{
				"healthEndpoint": true,
			},
			expectedConfig: HealthConfig{
				Enabled:        false,
				Targets:        nil,
				HealthEndpoint: true,
			},
			wantErr: false,
		},
		{
			name: "wrong type",
			inputConfig: map[string]any{
				"enabled":        "not bool",
				"target":         "not a slice",
				"healthEndpoint": true,
			},
			expectedConfig: HealthConfig{},
			wantErr:        true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Health{}

			if err := h.SetConfig(context.Background(), tt.inputConfig); (err != nil) != tt.wantErr {
				t.Errorf("Health.SetConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.expectedConfig, h.config, "Config is not equal")
		})
	}
}

func Test_getHealth(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	endpoint := "https://api.test.com/test"

	type args struct {
		ctx context.Context
		url string
	}
	tests := []struct {
		name string
		args args

		httpResponder httpmock.Responder
		wantErr       bool
	}{
		{
			name: "status 200",
			args: args{
				ctx: context.Background(),
				url: endpoint,
			},
			httpResponder: httpmock.NewStringResponder(200, ""),
			wantErr:       false,
		},
		{
			name: "status not 200",
			args: args{
				ctx: context.Background(),
				url: endpoint,
			},
			httpResponder: httpmock.NewStringResponder(400, ""),
			wantErr:       true,
		},
		{
			name: "ctx is nil",
			args: args{
				ctx: nil,
				url: endpoint,
			},
			httpResponder: httpmock.NewStringResponder(200, ""),
			wantErr:       true,
		},
		{
			name: "unknown url",
			args: args{
				ctx: context.Background(),
				url: "unknown url",
			},
			httpResponder: httpmock.NewStringResponder(200, ""),
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		httpmock.RegisterResponder(http.MethodGet, endpoint, tt.httpResponder)
		t.Run(tt.name, func(t *testing.T) {
			if err := getHealth(tt.args.ctx, tt.args.url); (err != nil) != tt.wantErr {
				t.Errorf("getHealth() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHealth_Check(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	tests := []struct {
		name               string
		registerdEndpoints map[string]int
		targets            []string
		ctx                context.Context
		want               healthData
	}{
		{
			name:               "no target",
			registerdEndpoints: nil,
			targets:            []string{},
			ctx:                context.Background(),
			want:               healthData{},
		},
		{
			name: "one target healthy",
			registerdEndpoints: map[string]int{
				"https://api.test.com": 200,
			},
			targets: []string{
				"https://api.test.com",
			},
			ctx: context.Background(),
			want: healthData{
				Targets: []Target{
					{
						Target: "https://api.test.com",
						Status: "healthy",
					},
				},
			},
		},
		{
			name: "one target unhealthy",
			registerdEndpoints: map[string]int{
				"https://api.test.com": 400,
			},
			targets: []string{
				"https://api.test.com",
			},
			ctx: context.Background(),
			want: healthData{
				Targets: []Target{
					{
						Target: "https://api.test.com",
						Status: "unhealthy",
					},
				},
			},
		},
		{
			name: "many targets",
			registerdEndpoints: map[string]int{
				"https://api1.test.com": 200,
				"https://api2.test.com": 400,
				"https://api3.test.com": 200,
				"https://api4.test.com": 300,
				"https://api5.test.com": 200,
			},
			targets: []string{
				"https://api1.test.com",
				"https://api2.test.com",
				"https://api3.test.com",
				"https://api4.test.com",
				"https://api5.test.com",
			},
			ctx: context.Background(),
			want: healthData{
				Targets: []Target{
					{
						Target: "https://api1.test.com",
						Status: "healthy",
					},
					{
						Target: "https://api2.test.com",
						Status: "unhealthy",
					},
					{
						Target: "https://api3.test.com",
						Status: "healthy",
					},
					{
						Target: "https://api4.test.com",
						Status: "unhealthy",
					},
					{
						Target: "https://api5.test.com",
						Status: "healthy",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for endpoint, statuscode := range tt.registerdEndpoints {
				httpmock.RegisterResponder(http.MethodGet, endpoint,
					httpmock.NewStringResponder(statuscode, ""),
				)
			}

			h := &Health{
				config: HealthConfig{
					Targets: tt.targets,
				},
			}
			got := h.check(tt.ctx)
			assert.Equal(t, len(got.Targets), len(tt.want.Targets), "Amount of targets is not equal")
			for _, target := range tt.want.Targets {
				helperStatus := "unhealthy"
				if tt.registerdEndpoints[target.Target] == 200 {
					helperStatus = "healthy"
				}
				assert.Equal(t, helperStatus, target.Status, "Target does not map with expected target")
			}
		})
	}
}
