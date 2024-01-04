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
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/caas-team/sparrow/pkg/api"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

const (
	successURL string = "http://success.com"
	failURL    string = "http://fail.com"
	timeoutURL string = "http://timeout.com"
)

func stringPointer(s string) *string {
	return &s
}

func TestLatency_Run(t *testing.T) { //nolint:gocyclo
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	tests := []struct {
		name                string
		registeredEndpoints []struct {
			name    string
			status  int
			success bool
		}
		targets []string
		ctx     context.Context
		want    Result
	}{
		{
			name: "success with one target",
			registeredEndpoints: []struct {
				name    string
				status  int
				success bool
			}{
				{
					name:    successURL,
					status:  http.StatusOK,
					success: true,
				},
			},
			targets: []string{successURL},
			ctx:     context.Background(),
			want: Result{
				Data: map[string]LatencyResult{
					successURL: {Code: http.StatusOK, Error: nil, Total: 0},
				},
				Timestamp: time.Time{},
				Err:       "",
			},
		},
		{
			name: "success with multiple targets",
			registeredEndpoints: []struct {
				name    string
				status  int
				success bool
			}{
				{
					name:    successURL,
					status:  http.StatusOK,
					success: true,
				},
				{
					name:    failURL,
					status:  http.StatusInternalServerError,
					success: true,
				},
				{
					name:    timeoutURL,
					status:  0,
					success: false,
				},
			},
			targets: []string{successURL, failURL, timeoutURL},
			ctx:     context.Background(),
			want: Result{
				Data: map[string]LatencyResult{
					successURL: {Code: http.StatusOK, Error: nil, Total: 0},
					failURL:    {Code: http.StatusInternalServerError, Error: nil, Total: 0},
					timeoutURL: {Code: 0, Error: stringPointer(fmt.Sprintf("Get %q: context deadline exceeded", timeoutURL)), Total: 0},
				},
				Timestamp: time.Time{},
				Err:       "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, endpoint := range tt.registeredEndpoints {
				if endpoint.success {
					httpmock.RegisterResponder(http.MethodGet, endpoint.name, httpmock.NewStringResponder(endpoint.status, ""))
				} else {
					httpmock.RegisterResponder(http.MethodGet, endpoint.name, httpmock.NewErrorResponder(context.DeadlineExceeded))
				}
			}

			c := NewLatencyCheck()
			results := make(chan Result, 1)
			err := c.Startup(tt.ctx, results)
			if err != nil {
				t.Fatalf("Latency.Startup() error = %v", err)
			}

			err = c.SetConfig(tt.ctx, map[string]any{
				"targets":  tt.targets,
				"interval": 1,
				"timeout":  5,
			})
			if err != nil {
				t.Fatalf("Latency.SetConfig() error = %v", err)
			}

			go func() {
				err := c.Run(tt.ctx)
				if err != nil {
					t.Errorf("Latency.Run() error = %v", err)
					return
				}
			}()
			defer func() {
				err := c.Shutdown(tt.ctx)
				if err != nil {
					t.Errorf("Latency.Shutdown() error = %v", err)
					return
				}
			}()

			result := <-results

			assert.IsType(t, tt.want.Data, result.Data)

			got := result.Data.(map[string]LatencyResult)
			expected := result.Data.(map[string]LatencyResult)
			if len(got) != len(expected) {
				t.Errorf("Length of Latency.Run() result set (%v) does not match length of expected result set (%v)", len(got), len(expected))
			}

			for key, resultObj := range got {
				if expected[key].Code != resultObj.Code {
					t.Errorf("Result Code of %q = %v, want %v", key, resultObj.Code, expected[key].Code)
				}
				if expected[key].Error != resultObj.Error {
					t.Errorf("Result Error of %q = %v, want %v", key, resultObj.Error, expected[key].Error)
				}
				if key != timeoutURL {
					if resultObj.Total <= 0 || resultObj.Total >= 1 {
						t.Errorf("Result Total time of %q = %v, want in between 0 and 1", key, resultObj.Total)
					}
				} else {
					if resultObj.Total != 0 {
						t.Errorf("Result Total time of %q = %v, want %v since an timeout occurred", key, resultObj.Total, 0)
					}
				}
			}

			if result.Err != tt.want.Err {
				t.Errorf("Latency.Run() = %v, want %v", result.Err, tt.want.Err)
			}
			httpmock.Reset()
		})
	}
}

func TestLatency_check(t *testing.T) {
	httpmock.Activate()
	t.Cleanup(httpmock.DeactivateAndReset)

	tests := []struct {
		name                string
		registeredEndpoints []struct {
			name    string
			status  int
			success bool
		}
		targets []string
		ctx     context.Context
		want    map[string]LatencyResult
	}{
		{
			name:                "no target",
			registeredEndpoints: nil,
			targets:             []string{},
			ctx:                 context.Background(),
			want:                map[string]LatencyResult{},
		},
		{
			name: "one target",
			registeredEndpoints: []struct {
				name    string
				status  int
				success bool
			}{
				{
					name:    successURL,
					status:  200,
					success: true,
				},
			},
			targets: []string{successURL},
			ctx:     context.Background(),
			want: map[string]LatencyResult{
				successURL: {Code: http.StatusOK, Error: nil, Total: 0},
			},
		},
		{
			name: "multiple targets",
			registeredEndpoints: []struct {
				name    string
				status  int
				success bool
			}{
				{
					name:    successURL,
					status:  http.StatusOK,
					success: true,
				},
				{
					name:    failURL,
					status:  http.StatusInternalServerError,
					success: true,
				},
				{
					name:    timeoutURL,
					success: false,
				},
			},
			targets: []string{successURL, failURL, timeoutURL},
			ctx:     context.Background(),
			want: map[string]LatencyResult{
				successURL: {
					Code:  200,
					Error: nil,
					Total: 0,
				},
				failURL: {
					Code:  500,
					Error: nil,
					Total: 0,
				},
				timeoutURL: {
					Code:  0,
					Error: stringPointer(fmt.Sprintf("Get %q: context deadline exceeded", timeoutURL)),
					Total: 0,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, endpoint := range tt.registeredEndpoints {
				if endpoint.success {
					httpmock.RegisterResponder(http.MethodGet, endpoint.name, httpmock.NewStringResponder(endpoint.status, ""))
				} else {
					httpmock.RegisterResponder(http.MethodGet, endpoint.name, httpmock.NewErrorResponder(context.DeadlineExceeded))
				}
			}

			l := &Latency{
				cfg:     LatencyConfig{Targets: tt.targets, Interval: time.Second * 120, Timeout: time.Second * 1},
				metrics: newLatencyMetrics(),
			}

			got := l.check(tt.ctx)

			if len(got) != len(tt.want) {
				t.Errorf("check() got %v results, want %v results", len(got), len(tt.want))
			}

			for k, v := range tt.want {
				if v.Code != got[k].Code {
					t.Errorf("Latency.check() = %v, want %v", got[k].Code, v.Code)
				}
				if got[k].Total < 0 {
					t.Errorf("Latency.check() got negative latency for key %v", k)
				}
				if v.Error != nil && got[k].Error != nil {
					if *v.Error != *got[k].Error {
						t.Errorf("Latency.check() = %v, want %v", *got[k].Error, *v.Error)
					}
				}
			}

			// Resetting httpmock for the next iteration
			httpmock.Reset()
		})
	}
}

func TestLatency_Startup(t *testing.T) {
	c := Latency{}

	if err := c.Startup(context.Background(), make(chan<- Result, 1)); err != nil {
		t.Errorf("Startup() error = %v", err)
	}
}

func TestLatency_Shutdown(t *testing.T) {
	cDone := make(chan bool, 1)
	c := Latency{
		done: cDone,
	}
	err := c.Shutdown(context.Background())
	if err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}

	if !<-cDone {
		t.Error("Shutdown() should be ok")
	}
}

func TestLatency_SetConfig(t *testing.T) {
	c := Latency{}
	wantCfg := LatencyConfig{
		Targets: []string{"http://localhost:9090"},
	}

	err := c.SetConfig(context.Background(), wantCfg)
	if err != nil {
		t.Errorf("SetConfig() error = %v", err)
	}
	if !reflect.DeepEqual(c.cfg, wantCfg) {
		t.Errorf("SetConfig() = %v, want %v", c.cfg, wantCfg)
	}
}

func TestLatency_RegisterHandler(t *testing.T) {
	c := Latency{}

	rt := api.NewRoutingTree()
	c.RegisterHandler(context.Background(), rt)

	h, ok := rt.Get(http.MethodGet, "v1alpha1/latency")

	if !ok {
		t.Error("RegisterHandler() should be ok")
	}
	if h == nil {
		t.Error("RegisterHandler() should not be nil")
	}
	c.DeregisterHandler(context.Background(), rt)
	h, ok = rt.Get(http.MethodGet, "v1alpha1/latency")

	if ok {
		t.Error("DeregisterHandler() should not be ok")
	}

	if h != nil {
		t.Error("DeregisterHandler() should be nil")
	}
}

func TestLatency_Handler(t *testing.T) {
	c := Latency{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1alpha1/latency", http.NoBody)

	c.Handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Handler() should be ok, got %d", rec.Code)
	}
}

func TestNewLatencyCheck(t *testing.T) {
	c := NewLatencyCheck()
	if c == nil {
		t.Error("NewLatencyCheck() should not be nil")
	}
}
