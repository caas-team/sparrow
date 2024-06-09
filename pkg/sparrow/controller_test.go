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

package sparrow

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/checks/dns"
	"github.com/caas-team/sparrow/pkg/checks/health"
	"github.com/caas-team/sparrow/pkg/checks/latency"
	"github.com/caas-team/sparrow/pkg/checks/runtime"
	"github.com/caas-team/sparrow/pkg/db"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestRun_CheckRunError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cc := NewChecksController(db.NewInMemory(), NewMetrics())
	mockCheck := &checks.CheckMock{
		NameFunc: func() string { return "mockCheck" },
		RunFunc: func(ctx context.Context, cResult chan checks.ResultDTO) error {
			return fmt.Errorf("some error")
		},
		GetMetricCollectorsFunc: func() []prometheus.Collector {
			return []prometheus.Collector{}
		},
		ShutdownFunc: func() {},
	}

	cc.RegisterCheck(ctx, mockCheck)

	go func() {
		err := cc.Run(ctx)
		if err != nil {
			t.Errorf("HandleErrors() error = %v", err)
		}
	}()
	// Wait for the error to be processed
	time.Sleep(100 * time.Millisecond)

	found := false
	for _, c := range cc.checks.Iter() {
		if c.Name() == mockCheck.Name() {
			found = true
			break
		}
	}

	if found {
		t.Errorf("Expected check to be unregistered after encountering a run error")
	}
	cc.Shutdown(ctx)
}

func TestRun_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	cc := NewChecksController(db.NewInMemory(), NewMetrics())

	done := make(chan struct{})
	go func() {
		err := cc.Run(ctx)
		if err == nil {
			t.Errorf("HandleErrors() = %v, want %v", nil, err)
		}
		close(done)
	}()

	cancel()

	select {
	case <-done:
		return
	case <-time.After(time.Second):
		t.Fatal("HandleErrors did not exit on context cancellation")
	}
}

func TestChecksController_Reconcile(t *testing.T) {
	ctx, cancel := logger.NewContextWithLogger(context.Background())
	defer cancel()
	rtcfg := &runtime.Config{}
	tests := []struct {
		name             string
		checks           []checks.Check
		newRuntimeConfig runtime.Config
	}{
		{
			name:   "no checks registered yet but register one",
			checks: []checks.Check{},
			newRuntimeConfig: runtime.Config{Health: &health.Config{
				Targets:  []string{"https://gitlab.com"},
				Interval: 1 * time.Second,
				Timeout:  1 * time.Second,
			}},
		},
		{
			name:   "no checks registered, register multiple new ones",
			checks: []checks.Check{},
			newRuntimeConfig: runtime.Config{
				Health: &health.Config{
					Targets:  []string{"https://gitlab.com"},
					Interval: 1 * time.Second,
					Timeout:  1 * time.Second,
				},
				Latency: &latency.Config{
					Targets:  []string{"https://gitlab.com"},
					Interval: 1 * time.Second,
					Timeout:  1 * time.Second,
				},
				Dns: &dns.Config{
					Targets:  []string{"gitlab.com"},
					Interval: 1 * time.Second,
					Timeout:  1 * time.Second,
				},
			},
		},
		{
			name: "one healthcheck registered, register latency check",
			checks: []checks.Check{
				health.NewCheck(),
			},
			newRuntimeConfig: runtime.Config{
				Latency: &latency.Config{
					Targets:  []string{"https://gitlab.com"},
					Interval: 1 * time.Second,
					Timeout:  1 * time.Second,
				},
				Health: &health.Config{
					Targets:  []string{"https://gitlab.com"},
					Interval: 1 * time.Second,
					Timeout:  1 * time.Second,
				},
			},
		},
		{
			name: "no checks registered but unregister all",
			checks: []checks.Check{
				health.NewCheck(),
			},
			newRuntimeConfig: *rtcfg,
		},
		{
			name: "one health check registered, register latency and unregister health",
			checks: []checks.Check{
				health.NewCheck(),
			},
			newRuntimeConfig: runtime.Config{
				Latency: &latency.Config{
					Targets:  []string{"https://gitlab.com"},
					Interval: 1 * time.Second,
					Timeout:  1 * time.Second,
				},
			},
		},
		{
			name: "multiple checks registered, unregister some",
			checks: []checks.Check{
				health.NewCheck(),
				latency.NewCheck(),
			},
			newRuntimeConfig: runtime.Config{
				Health: &health.Config{
					Targets:  []string{"https://gitlab.com"},
					Interval: 1 * time.Second,
					Timeout:  1 * time.Second,
				},
			},
		},
		{
			name: "multiple checks registered, unregister all",
			checks: []checks.Check{
				health.NewCheck(),
				latency.NewCheck(),
			},
			newRuntimeConfig: *rtcfg,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cc := NewChecksController(db.NewInMemory(), NewMetrics())

			for _, c := range tt.checks {
				cc.checks.Add(c)
			}

			cc.Reconcile(ctx, tt.newRuntimeConfig)

			// iterate of the controller's checks and check if they are configured
			for _, c := range cc.checks.Iter() {
				cfg := c.GetConfig()
				assert.NotNil(t, cfg)
				if cfg.For() == health.CheckName {
					assert.Equal(t, tt.newRuntimeConfig.Health, cfg)
				}
				if cfg.For() == latency.CheckName {
					assert.Equal(t, tt.newRuntimeConfig.Latency, cfg)
				}
				if cfg.For() == dns.CheckName {
					assert.Equal(t, tt.newRuntimeConfig.Dns, cfg)
				}
			}

			// check that the number of registered checks is correct
			assert.Equal(t, len(tt.newRuntimeConfig.Iter()), len(cc.checks.Iter()))
		})
	}
}

func TestChecksController_RegisterCheck(t *testing.T) {
	tests := []struct {
		name  string
		setup func() *ChecksController
		check checks.Check
	}{
		{
			name: "register one check",
			setup: func() *ChecksController {
				return NewChecksController(db.NewInMemory(), NewMetrics())
			},
			check: health.NewCheck(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cc := tt.setup()
			cc.RegisterCheck(context.Background(), tt.check)
			if cc.checks.Iter()[0] != tt.check {
				t.Errorf("Expected one check to be registered")
			}
		})
	}
}

func TestChecksController_UnregisterCheck(t *testing.T) {
	tests := []struct {
		name  string
		check checks.Check
	}{
		{
			name:  "valid check",
			check: health.NewCheck(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cc := NewChecksController(db.NewInMemory(), NewMetrics())

			cc.UnregisterCheck(context.Background(), tt.check)

			if len(cc.checks.Iter()) != 0 {
				t.Errorf("Expected check to be unregistered")
			}
		})
	}
}

func TestGenerateCheckSpecs(t *testing.T) {
	tests := []struct {
		name     string
		checks   []checks.Check
		wantErr  bool
		validate func(t *testing.T, doc openapi3.T)
	}{
		{
			name: "successful generation",
			checks: []checks.Check{
				&checks.CheckMock{
					NameFunc: func() string {
						return "check1"
					},
					SchemaFunc: func() (*openapi3.SchemaRef, error) {
						type CheckResultSpec struct {
							name string
						}
						res := CheckResultSpec{name: "check1"}
						return checks.OpenapiFromPerfData(res)
					},
				},
				&checks.CheckMock{
					NameFunc: func() string {
						return "check2"
					},
					SchemaFunc: func() (*openapi3.SchemaRef, error) {
						type CheckResultSpec struct {
							name string
						}
						res := CheckResultSpec{name: "check2"}
						return checks.OpenapiFromPerfData(res)
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, doc openapi3.T) {
				if _, ok := doc.Paths.Extensions["/v1/metrics/check1"]; !ok {
					t.Errorf("Expected path '/v1/metrics/check1' not found")
				}
				if _, ok := doc.Paths.Extensions["/v1/metrics/check2"]; !ok {
					t.Errorf("Expected path '/v1/metrics/check2' not found")
				}
			},
		},
		{
			name: "error in schema generation",
			checks: []checks.Check{
				&checks.CheckMock{
					NameFunc: func() string {
						return ""
					},
					SchemaFunc: func() (*openapi3.SchemaRef, error) {
						return nil, fmt.Errorf("some error")
					},
				},
			},
			wantErr:  true,
			validate: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			cc := &ChecksController{
				checks: runtime.Checks{},
			}

			for _, c := range tt.checks {
				cc.checks.Add(c)
			}

			doc, err := cc.GenerateCheckSpecs(ctx)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GenerateCheckSpecs() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.validate != nil {
				tt.validate(t, doc)
			}

			if tt.wantErr {
				var schemaErr *ErrCreateOpenapiSchema
				t.Logf("Error = %v", err)
				if !errors.As(err, &schemaErr) {
					t.Error("Expected ErrCreateOpenapiSchema")
				}
			}
		})
	}
}
