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

package sparrow

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"

	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/api"
	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/config"
	"github.com/caas-team/sparrow/pkg/db"
)

func TestSparrow_ReconcileChecks(t *testing.T) {
	ctx, cancel := logger.NewContextWithLogger(context.Background(), "sparrow-test")
	defer cancel()

	mockCheck := checks.CheckMock{
		RunFunc: func(ctx context.Context) error {
			return nil
		},
		SchemaFunc: func() (*openapi3.SchemaRef, error) {
			return nil, nil
		},
		SetConfigFunc: func(ctx context.Context, config any) error {
			return nil
		},
		ShutdownFunc: func(ctx context.Context) error {
			return nil
		},
		StartupFunc: func(ctx context.Context, cResult chan<- checks.Result) error {
			return nil
		},
		RegisterHandlerFunc:   func(ctx context.Context, router *api.RoutingTree) {},
		DeregisterHandlerFunc: func(ctx context.Context, router *api.RoutingTree) {},
	}

	checks.RegisteredChecks = map[string]func() checks.Check{
		"alpha": func() checks.Check { return &mockCheck },
		"beta":  func() checks.Check { return &mockCheck },
		"gamma": func() checks.Check { return &mockCheck },
	}

	type fields struct {
		checks      map[string]checks.Check
		resultFanIn map[string]chan checks.Result

		cResult    chan checks.ResultDTO
		loader     config.Loader
		cfg        *config.Config
		cCfgChecks chan map[string]any
		db         db.DB
	}

	tests := []struct {
		name            string
		fields          fields
		newChecksConfig map[string]any
	}{
		{
			name: "no checks registered yet but register one",
			fields: fields{
				checks:      map[string]checks.Check{},
				cfg:         &config.Config{},
				cCfgChecks:  make(chan map[string]any, 1),
				resultFanIn: make(map[string]chan checks.Result),
			},
			newChecksConfig: map[string]any{
				"alpha": "I like sparrows",
			},
		},
		{
			name: "on checks registered and register another",
			fields: fields{
				checks: map[string]checks.Check{
					"alpha": checks.RegisteredChecks["alpha"](),
				},
				cfg:         &config.Config{},
				cCfgChecks:  make(chan map[string]any, 1),
				resultFanIn: make(map[string]chan checks.Result),
			},
			newChecksConfig: map[string]any{
				"alpha": "I like sparrows",
				"beta":  "I like them more",
			},
		},
		{
			name: "on checks registered but unregister all",
			fields: fields{
				checks: map[string]checks.Check{
					"alpha": checks.RegisteredChecks["alpha"](),
				},
				cfg:         &config.Config{},
				cCfgChecks:  make(chan map[string]any, 1),
				resultFanIn: make(map[string]chan checks.Result),
			},
			newChecksConfig: map[string]any{},
		},
		{
			name: "two checks registered, register another and unregister one",
			fields: fields{
				checks: map[string]checks.Check{
					"alpha": checks.RegisteredChecks["alpha"](),
					"gamma": checks.RegisteredChecks["alpha"](),
				},
				cfg:         &config.Config{},
				cCfgChecks:  make(chan map[string]any, 1),
				resultFanIn: make(map[string]chan checks.Result),
			},
			newChecksConfig: map[string]any{
				"alpha": "I like sparrows",
				"beta":  "I like them more",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Sparrow{
				checks:      tt.fields.checks,
				resultFanIn: tt.fields.resultFanIn,
				cResult:     tt.fields.cResult,
				loader:      tt.fields.loader,
				cfg:         tt.fields.cfg,
				cCfgChecks:  tt.fields.cCfgChecks,
				db:          tt.fields.db,
			}

			// Send new config to channel
			s.cfg.Checks = tt.newChecksConfig

			s.ReconcileChecks(ctx)

			for newChecksConfigName := range tt.newChecksConfig {
				check := checks.RegisteredChecks[newChecksConfigName]()
				assert.Equal(t, check, s.checks[newChecksConfigName])
			}
		})
	}
}

func Test_fanInResults(t *testing.T) {
	checkChan := make(chan checks.Result, 1)
	cResult := make(chan checks.ResultDTO, 1)
	name := "check"
	go fanInResults(checkChan, cResult, name)

	result := checks.Result{
		Timestamp: time.Time{},
		Err:       "",
		Data:      0,
	}

	checkChan <- result
	output := <-cResult

	want := checks.ResultDTO{
		Name:   name,
		Result: &result,
	}

	if !reflect.DeepEqual(output, want) {
		t.Errorf("fanInResults() = %v, want %v", output, want)
	}

	close(checkChan)
}

func TestSparrow_Run(t *testing.T) {
	c := &config.Config{
		Api: config.ApiConfig{ListeningAddress: ":9090"},
		Loader: config.LoaderConfig{
			Type: "http",
		},
	}

	s := New(c)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := s.Run(ctx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Sparrow.Run() error = %v", err)
	}
}
