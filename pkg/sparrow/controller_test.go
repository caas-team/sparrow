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

	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/checks/health"
	"github.com/caas-team/sparrow/pkg/checks/runtime"
	"github.com/caas-team/sparrow/pkg/db"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/prometheus/client_golang/prometheus"
)

func TestListenErrors_CheckRunError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cc := NewChecksController(db.NewInMemory(), NewMetrics())
	mockCheck := &checks.CheckMock{
		NameFunc: func() string { return "mockCheck" },
		RunFunc: func(ctx context.Context) error {
			return fmt.Errorf("some error")
		},
		GetMetricCollectorsFunc: func() []prometheus.Collector {
			return []prometheus.Collector{}
		},
		ResultChanFunc: func() chan checks.Result {
			return make(chan checks.Result, 1)
		},
		ShutdownFunc: func(ctx context.Context) error {
			return nil
		},
	}

	err := cc.RegisterCheck(ctx, mockCheck)
	if err != nil {
		t.Fatalf("RegisterCheck() error = %v", err)
	}

	go cc.ListenErrors(ctx)

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
		t.Errorf("Expected mockCheck to be unregistered after encountering a run error")
	}
	if err := cc.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}

func TestListenErrors_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	cc := NewChecksController(db.NewInMemory(), NewMetrics())

	done := make(chan struct{})
	go func() {
		cc.ListenErrors(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
		return
	case <-time.After(time.Second):
		t.Fatal("ListenErrors did not exit on context cancellation")
	}
}

func TestChecksController_RegisterCheck(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) *ChecksController
		check   checks.Check
		wantErr bool
	}{
		{
			name: "valid check",
			setup: func(_ *testing.T) *ChecksController {
				return NewChecksController(db.NewInMemory(), NewMetrics())
			},
			check:   health.NewCheck(),
			wantErr: false,
		},
		{
			name: "duplicate check registration",
			setup: func(t *testing.T) *ChecksController {
				cc := NewChecksController(db.NewInMemory(), NewMetrics())
				check := health.NewCheck()
				err := cc.RegisterCheck(context.Background(), check)
				if err != nil {
					t.Fatalf("RegisterCheck() error = %v", err)
				}
				return cc
			},
			check:   health.NewCheck(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cc := tt.setup(t)

			err := cc.RegisterCheck(context.Background(), tt.check)
			if (err != nil) != tt.wantErr {
				t.Errorf("ChecksController.RegisterCheck() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChecksController_UnregisterCheck(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) *ChecksController
		check   checks.Check
		wantErr bool
	}{
		{
			name: "valid check",
			setup: func(_ *testing.T) *ChecksController {
				return NewChecksController(db.NewInMemory(), NewMetrics())
			},
			check:   health.NewCheck(),
			wantErr: false,
		},
		{
			name: "unregister non-existent check",
			setup: func(_ *testing.T) *ChecksController {
				return NewChecksController(db.NewInMemory(), NewMetrics())
			},
			check:   health.NewCheck(),
			wantErr: false,
		},
		{
			name: "error during check shutdown",
			setup: func(_ *testing.T) *ChecksController {
				cc := NewChecksController(db.NewInMemory(), NewMetrics())
				check := &checks.CheckMock{
					NameFunc:                func() string { return "check" },
					GetMetricCollectorsFunc: func() []prometheus.Collector { return []prometheus.Collector{} },
					RunFunc:                 func(ctx context.Context) error { return nil },
					ResultChanFunc:          func() chan checks.Result { return make(chan checks.Result) },
					ShutdownFunc: func(ctx context.Context) error {
						return fmt.Errorf("some error")
					},
				}
				err := cc.RegisterCheck(context.Background(), check)
				if err != nil {
					t.Fatalf("RegisterCheck() error = %v", err)
				}
				return cc
			},
			check: &checks.CheckMock{
				NameFunc: func() string { return "check" },
				GetMetricCollectorsFunc: func() []prometheus.Collector {
					return []prometheus.Collector{}
				},
				ShutdownFunc: func(ctx context.Context) error {
					return fmt.Errorf("some error")
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cc := tt.setup(t)

			err := cc.UnregisterCheck(context.Background(), tt.check)
			if (err != nil) != tt.wantErr {
				t.Errorf("ChecksController.RegisterCheck() error = %v, wantErr %v", err, tt.wantErr)
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
						return checks.OpenapiFromPerfData[CheckResultSpec](res)
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
						return checks.OpenapiFromPerfData[CheckResultSpec](res)
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, doc openapi3.T) {
				if _, ok := doc.Paths["/v1/metrics/check1"]; !ok {
					t.Errorf("Expected path '/v1/metrics/check1' not found")
				}
				if _, ok := doc.Paths["/v1/metrics/check2"]; !ok {
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
