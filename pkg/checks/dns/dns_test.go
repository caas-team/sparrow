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
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/caas-team/sparrow/pkg/api"
	"github.com/caas-team/sparrow/pkg/checks/types"
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
		ctx       context.Context
		want      types.Result
	}{
		{
			name: "success with no targets",
			mockSetup: func() *DNS {
				return &DNS{
					CheckBase: types.CheckBase{
						Mu:   sync.Mutex{},
						Done: make(chan bool, 1),
					},
				}
			},
			targets: []string{},
			ctx:     context.Background(),
			want: types.Result{
				Data: map[string]Result{},
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
			ctx:     context.Background(),
			want: types.Result{
				Data: map[string]Result{
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
			ctx:     context.Background(),
			want: types.Result{
				Data: map[string]Result{
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
			ctx:     context.Background(),
			want: types.Result{
				Data: map[string]Result{
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
			ctx:     context.Background(),
			want: types.Result{
				Data: map[string]Result{
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
						time.Sleep(6 * time.Second)
						return nil, fmt.Errorf("context deadline exceeded")
					},
					SetDialerFunc: func(d *net.Dialer) {},
				}
				return c
			},
			targets: []string{exampleURL},
			ctx:     context.Background(),
			want: types.Result{
				Data: map[string]Result{
					exampleURL: {Resolved: nil, Error: stringPointer("context deadline exceeded")},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.mockSetup()

			results := make(chan types.Result, 1)
			err := c.Startup(tt.ctx, results)
			if err != nil {
				t.Fatalf("DNS.Startup() error = %v", err)
			}

			err = c.SetConfig(tt.ctx, map[string]any{
				"targets":  tt.targets,
				"interval": "1s",
				"timeout":  "5s",
			})
			if err != nil {
				t.Fatalf("DNS.SetConfig() error = %v", err)
			}

			go func() {
				err := c.Run(tt.ctx)
				if err != nil {
					t.Errorf("DNS.Run() error = %v", err)
					return
				}
			}()
			defer func() {
				err := c.Shutdown(tt.ctx)
				if err != nil {
					t.Errorf("DNS.Shutdown() error = %v", err)
					return
				}
			}()

			result := <-results

			assert.IsType(t, tt.want.Data, result.Data)

			got := result.Data.(map[string]Result)
			want := tt.want.Data.(map[string]Result)
			if len(got) != len(want) {
				t.Errorf("Length of DNS.Run() result set (%v) does not match length of expected result set (%v)", len(got), len(want))
			}

			for target, result := range got {
				if !reflect.DeepEqual(want[target].Resolved, result.Resolved) {
					t.Errorf("Result Resolved of %s = %v, want %v", target, result.Resolved, want[target].Resolved)
				}
				if want[target].Error != nil {
					if result.Error == nil {
						t.Errorf("Result Error of %s = %v, want %v", target, result.Error, *want[target].Error)
					}
				}
			}

			if result.Err != tt.want.Err {
				t.Errorf("DNS.Run() = %v, want %v", result.Err, tt.want.Err)
			}
		})
	}
}

func TestDNS_Run_Context_Done(t *testing.T) {
	c := NewCheck()
	ctx, cancel := context.WithCancel(context.Background())
	_ = c.SetConfig(ctx, config{
		Interval: time.Second,
	})
	go func() {
		err := c.Run(ctx)
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

func TestDNS_Startup(t *testing.T) {
	c := DNS{}

	if err := c.Startup(context.Background(), make(chan<- types.Result, 1)); err != nil {
		t.Errorf("Startup() error = %v", err)
	}
}

func TestDNS_Shutdown(t *testing.T) {
	cDone := make(chan bool, 1)
	c := DNS{
		CheckBase: types.CheckBase{
			Done: cDone,
		},
	}
	err := c.Shutdown(context.Background())
	if err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}

	if !<-cDone {
		t.Error("Shutdown() should be ok")
	}
}

func TestDNS_SetConfig(t *testing.T) {
	c := DNS{}
	wantCfg := config{
		Targets: []string{"http://localhost:9090"},
	}

	err := c.SetConfig(context.Background(), wantCfg)
	if err != nil {
		t.Errorf("SetConfig() error = %v", err)
	}
	if !reflect.DeepEqual(c.config, wantCfg) {
		t.Errorf("SetConfig() = %v, want %v", c.config, wantCfg)
	}
}

func TestDNS_RegisterHandler(t *testing.T) {
	c := DNS{}

	rt := api.NewRoutingTree()
	c.RegisterHandler(context.Background(), rt)

	h, ok := rt.Get(http.MethodGet, route)

	if !ok {
		t.Error("RegisterHandler() should be ok")
	}
	if h == nil {
		t.Error("RegisterHandler() should not be nil")
	}
	c.DeregisterHandler(context.Background(), rt)
	h, ok = rt.Get(http.MethodGet, "dns")

	if ok {
		t.Error("DeregisterHandler() should not be ok")
	}

	if h != nil {
		t.Error("DeregisterHandler() should be nil")
	}
}

func TestDNS_Handler(t *testing.T) {
	c := DNS{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/%s", route), http.NoBody)

	c.Handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Handler() should be ok, got %d", rec.Code)
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
		CheckBase: types.CheckBase{Mu: sync.Mutex{}, Done: make(chan bool, 1)},
		metrics:   newMetrics(),
	}
}
