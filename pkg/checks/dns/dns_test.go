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

package dns

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/checks/health"

	"github.com/stretchr/testify/assert"
)

const (
	exampleURL = "www.example.com"
	sparrowURL = "www.sparrow.com"
	exampleIP  = "1.2.3.4"
	sparrowIP  = "4.3.2.1"
)

func TestDNS_Run(t *testing.T) {
	tests := []struct {
		name      string
		mockSetup func() *DNS
		targets   []string
		want      checks.Result
	}{
		{
			name: "success with no targets",
			mockSetup: func() *DNS {
				return &DNS{
					CheckBase: checks.CheckBase{
						Mu:       sync.Mutex{},
						DoneChan: make(chan struct{}, 1),
					},
				}
			},
			targets: []string{},
			want: checks.Result{
				Data: map[string]result{},
			},
		},
		{
			name: "success with one target lookup",
			mockSetup: func() *DNS {
				c := newCommonDNS()
				c.client = &ResolverMock{
					LookupHostFunc: func(ctx context.Context, addr string) ([]string, error) {
						return []string{exampleIP}, nil
					},
					SetDialerFunc: func(d *net.Dialer) {},
				}
				return c
			},
			targets: []string{exampleURL},
			want: checks.Result{
				Data: map[string]result{
					exampleURL: {Resolved: []string{exampleIP}},
				},
			},
		},
		{ //nolint:dupl // normal lookup
			name: "success with multiple target lookups",
			mockSetup: func() *DNS {
				c := newCommonDNS()
				c.client = &ResolverMock{
					LookupHostFunc: func(ctx context.Context, addr string) ([]string, error) {
						return []string{exampleIP, sparrowIP}, nil
					},
					SetDialerFunc: func(d *net.Dialer) {},
				}
				return c
			},
			targets: []string{exampleURL, sparrowURL},
			want: checks.Result{
				Data: map[string]result{
					exampleURL: {Resolved: []string{exampleIP, sparrowIP}},
					sparrowURL: {Resolved: []string{exampleIP, sparrowIP}},
				},
			},
		},
		{ //nolint:dupl // reverse lookup
			name: "success with multiple target reverse lookups",
			mockSetup: func() *DNS {
				c := newCommonDNS()
				c.client = &ResolverMock{
					LookupAddrFunc: func(ctx context.Context, addr string) ([]string, error) {
						return []string{exampleURL, sparrowURL}, nil
					},
					SetDialerFunc: func(d *net.Dialer) {},
				}
				return c
			},
			targets: []string{exampleIP, sparrowIP},
			want: checks.Result{
				Data: map[string]result{
					exampleIP: {Resolved: []string{exampleURL, sparrowURL}},
					sparrowIP: {Resolved: []string{exampleURL, sparrowURL}},
				},
			},
		},
		{
			name: "error - lookup failure for a target",
			mockSetup: func() *DNS {
				c := newCommonDNS()
				c.client = &ResolverMock{
					LookupHostFunc: func(ctx context.Context, addr string) ([]string, error) {
						return nil, fmt.Errorf("lookup failed")
					},
					SetDialerFunc: func(d *net.Dialer) {},
				}
				return c
			},
			targets: []string{exampleURL},
			want: checks.Result{
				Data: map[string]result{
					exampleURL: {Error: stringPointer("lookup failed")},
				},
			},
		},
		{
			name: "error - timeout scenario for a target",
			mockSetup: func() *DNS {
				c := newCommonDNS()
				c.client = &ResolverMock{
					LookupHostFunc: func(ctx context.Context, addr string) ([]string, error) {
						return nil, fmt.Errorf("context deadline exceeded")
					},
					SetDialerFunc: func(d *net.Dialer) {},
				}
				return c
			},
			targets: []string{exampleURL},
			want: checks.Result{
				Data: map[string]result{
					exampleURL: {Resolved: nil, Error: stringPointer("context deadline exceeded")},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			c := tt.mockSetup()

			cResult := make(chan checks.ResultDTO, 1)
			defer close(cResult)

			err := c.SetConfig(&Config{
				Targets:  tt.targets,
				Interval: 1 * time.Second,
				Timeout:  5 * time.Millisecond,
			})
			if err != nil {
				t.Fatalf("DNS.SetConfig() error = %v", err)
			}

			go func() {
				err := c.Run(ctx, cResult)
				if err != nil {
					t.Errorf("DNS.Run() error = %v", err)
					return
				}
			}()
			defer func() {
				c.Shutdown()
			}()

			r := <-cResult
			assert.IsType(t, tt.want.Data, r.Result.Data)

			got := r.Result.Data.(map[string]result)
			want := tt.want.Data.(map[string]result)
			if len(got) != len(want) {
				t.Errorf("Length of DNS.Run() result set (%v) does not match length of expected result set (%v)", len(got), len(want))
			}

			for tar, res := range got {
				if !reflect.DeepEqual(want[tar].Resolved, res.Resolved) {
					t.Errorf("Result Resolved of %s = %v, want %v", tar, res.Resolved, want[tar].Resolved)
				}
				if want[tar].Error != nil {
					if res.Error == nil {
						t.Errorf("Result Error of %s = %v, want %v", tar, res.Error, *want[tar].Error)
					}
				}
			}
		})
	}
}

func TestDNS_Run_Context_Done(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	c := NewCheck()
	cResult := make(chan checks.ResultDTO, 1)
	defer close(cResult)

	err := c.SetConfig(&Config{
		Interval: time.Second,
	})
	if err != nil {
		t.Fatalf("DNS.SetConfig() error = %v", err)
	}

	go func() {
		err := c.Run(ctx, cResult)
		t.Logf("DNS.Run() exited with error: %v", err)
		if err == nil {
			t.Error("DNS.Run() should have errored out, no error received")
		}
	}()

	t.Log("Running dns check for 10ms")
	time.Sleep(time.Millisecond * 10)

	t.Log("Canceling context and waiting for shutdown")
	cancel()
	time.Sleep(time.Millisecond * 30)
}

func TestDNS_Shutdown(t *testing.T) {
	cDone := make(chan struct{}, 1)
	c := DNS{
		CheckBase: checks.CheckBase{
			DoneChan: cDone,
		},
	}
	c.Shutdown()

	_, ok := <-cDone
	if !ok {
		t.Error("Shutdown() should be ok")
	}
}

func TestDNS_SetConfig(t *testing.T) {
	tests := []struct {
		name    string
		input   checks.Runtime
		want    Config
		wantErr bool
	}{
		{
			name: "simple config",
			input: &Config{
				Targets: []string{
					exampleURL,
					sparrowURL,
				},
				Interval: 10 * time.Second,
				Timeout:  30 * time.Second,
			},
			want: Config{
				Targets:  []string{exampleURL, sparrowURL},
				Interval: 10 * time.Second,
				Timeout:  30 * time.Second,
			},
			wantErr: false,
		},
		{
			name:    "empty config",
			input:   &Config{},
			want:    Config{},
			wantErr: false,
		},
		{
			name: "wrong type",
			input: &health.Config{
				Targets: []string{
					exampleURL,
				},
			},
			want:    Config{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &DNS{}

			if err := c.SetConfig(tt.input); (err != nil) != tt.wantErr {
				t.Errorf("DNS.SetConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.want, c.config, "Config is not equal")
		})
	}
}

func TestNewCheck(t *testing.T) {
	c := NewCheck()
	if c == nil {
		t.Error("NewLatencyCheck() should not be nil")
	}
}

func stringPointer(s string) *string {
	return &s
}

func newCommonDNS() *DNS {
	return &DNS{
		CheckBase: checks.CheckBase{
			Mu:       sync.Mutex{},
			DoneChan: make(chan struct{}, 1),
		},
		metrics: newMetrics(),
	}
}
