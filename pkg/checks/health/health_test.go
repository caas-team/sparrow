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

package health

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/checks/latency"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestHealth_UpdateConfig(t *testing.T) {
	tests := []struct {
		name           string
		inputConfig    checks.Runtime
		expectedConfig Config
		wantErr        bool
	}{
		{
			name: "simple config",
			inputConfig: &Config{
				Targets:  []string{"test"},
				Interval: 10 * time.Second,
				Timeout:  30 * time.Second,
			},
			expectedConfig: Config{
				Targets:  []string{"test"},
				Interval: 10 * time.Second,
				Timeout:  30 * time.Second,
			},
			wantErr: false,
		},
		{
			name:           "empty config",
			inputConfig:    &Config{},
			expectedConfig: Config{},
			wantErr:        false,
		},
		{
			name: "wrong type",
			inputConfig: &latency.Config{
				Targets: []string{"test"},
			},
			expectedConfig: Config{},
			wantErr:        true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Health{
				metrics: newMetrics(),
			}

			if err := h.UpdateConfig(tt.inputConfig); (err != nil) != tt.wantErr {
				t.Errorf("Health.UpdateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.expectedConfig, h.config, "Runtime config is not equal")
		})
	}
}

func Test_getHealth(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	endpoint := "https://api.test.com/test"

	type args struct {
		ctx    context.Context
		client *http.Client
		url    string
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
				ctx:    context.Background(),
				client: &http.Client{},
				url:    endpoint,
			},
			httpResponder: httpmock.NewStringResponder(200, ""),
			wantErr:       false,
		},
		{
			name: "status not 200",
			args: args{
				ctx:    context.Background(),
				client: &http.Client{},
				url:    endpoint,
			},
			httpResponder: httpmock.NewStringResponder(400, ""),
			wantErr:       true,
		},
		{
			name: "ctx is nil",
			args: args{
				ctx:    nil,
				client: &http.Client{},
				url:    endpoint,
			},
			httpResponder: httpmock.NewStringResponder(200, ""),
			wantErr:       true,
		},
		{
			name: "unknown url",
			args: args{
				ctx:    context.Background(),
				client: &http.Client{},
				url:    "unknown url",
			},
			httpResponder: httpmock.NewStringResponder(200, ""),
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		httpmock.RegisterResponder(http.MethodGet, endpoint, tt.httpResponder)
		t.Run(tt.name, func(t *testing.T) {
			if err := getHealth(tt.args.ctx, tt.args.client, tt.args.url); (err != nil) != tt.wantErr {
				t.Errorf("getHealth() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHealth_Check(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	tests := []struct {
		name                string
		registeredEndpoints map[string]int
		targets             []string
		ctx                 context.Context
		want                map[string]string
	}{
		{
			name:                "no target",
			registeredEndpoints: nil,
			targets:             []string{},
			ctx:                 context.Background(),
			want:                map[string]string{},
		},
		{
			name: "one target healthy",
			registeredEndpoints: map[string]int{
				"https://api.test.com": 200,
			},
			targets: []string{
				"https://api.test.com",
			},
			ctx: context.Background(),
			want: map[string]string{
				"https://api.test.com": "healthy",
			},
		},
		{
			name: "one target unhealthy",
			registeredEndpoints: map[string]int{
				"https://api.test.com": 400,
			},
			targets: []string{
				"https://api.test.com",
			},
			ctx: context.Background(),
			want: map[string]string{
				"https://api.test.com": "unhealthy",
			},
		},
		{
			name: "many targets",
			registeredEndpoints: map[string]int{
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
			want: map[string]string{
				"https://api1.test.com": "healthy",
				"https://api2.test.com": "unhealthy",
				"https://api3.test.com": "healthy",
				"https://api4.test.com": "unhealthy",
				"https://api5.test.com": "healthy",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for endpoint, statuscode := range tt.registeredEndpoints {
				httpmock.RegisterResponder(http.MethodGet, endpoint,
					httpmock.NewStringResponder(statuscode, ""),
				)
			}

			h := &Health{
				config: Config{
					Targets: tt.targets,
					Timeout: 30,
					Retry:   checks.DefaultRetry,
				},
				metrics: newMetrics(),
			}
			got := h.check(tt.ctx)
			assert.Equal(t, len(got), len(tt.want), "Amount of targets is not equal")
			for target, status := range tt.want {
				helperStatus := "unhealthy"
				if tt.registeredEndpoints[target] == 200 {
					helperStatus = "healthy"
				}
				assert.Equal(t, helperStatus, status, "Target does not map with expected target")
			}
		})
	}
}
