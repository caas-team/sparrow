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

package latency

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/caas-team/sparrow/pkg/checks"

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

func TestLatency_Run(t *testing.T) {
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
		want    checks.Result
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
			want: checks.Result{
				Data: map[string]result{
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
			want: checks.Result{
				Data: map[string]result{
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

			c := NewCheck()
			err := c.SetConfig(&Config{
				Targets:  tt.targets,
				Interval: time.Millisecond * 120,
				Timeout:  time.Second * 1,
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

			res := <-c.ResultChan()

			assert.IsType(t, tt.want.Data, res.Data)

			got := res.Data.(map[string]result)
			expected := tt.want.Data.(map[string]result)
			if len(got) != len(expected) {
				t.Errorf("Length of Latency.Run() result set (%v) does not match length of expected result set (%v)", len(got), len(expected))
			}

			for key, resultObj := range got {
				if expected[key].Code != resultObj.Code {
					t.Errorf("Result Code of %q = %v, want %v", key, resultObj.Code, expected[key].Code)
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

			if res.Err != tt.want.Err {
				t.Errorf("Latency.Run() = %v, want %v", res.Err, tt.want.Err)
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
		want    map[string]result
	}{
		{
			name:                "no target",
			registeredEndpoints: nil,
			targets:             []string{},
			ctx:                 context.Background(),
			want:                map[string]result{},
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
			want: map[string]result{
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
			want: map[string]result{
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
				config:  Config{Targets: tt.targets, Interval: time.Second * 120, Timeout: time.Second * 1},
				metrics: newMetrics(),
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

func TestDNS_ResultChan(t *testing.T) {
	c := Latency{
		CheckBase: checks.CheckBase{
			ResChan: make(chan checks.Result, 1),
		},
	}

	rc := c.ResultChan()
	if rc != c.ResChan {
		t.Errorf("ResultChan() got = %v, want %v", rc, c.ResChan)
	}
}

func TestLatency_Shutdown(t *testing.T) {
	cDone := make(chan struct{}, 1)
	c := Latency{
		CheckBase: checks.CheckBase{
			ResChan:  make(chan checks.Result, 1),
			DoneChan: cDone,
		},
	}
	err := c.Shutdown(context.Background())
	if err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}

	_, ok := <-cDone
	if !ok {
		t.Error("Shutdown() should be ok")
	}
}

func TestLatency_SetConfig(t *testing.T) {
	c := Latency{}
	wantCfg := Config{
		Targets: []string{"http://localhost:9090"},
	}

	err := c.SetConfig(&wantCfg)
	if err != nil {
		t.Errorf("SetConfig() error = %v", err)
	}
	if !reflect.DeepEqual(c.config, wantCfg) {
		t.Errorf("SetConfig() = %v, want %v", c.config, wantCfg)
	}
}

func TestNewLatencyCheck(t *testing.T) {
	c := NewCheck()
	if c == nil {
		t.Error("NewLatencyCheck() should not be nil")
	}
}
