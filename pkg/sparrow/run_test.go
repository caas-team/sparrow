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
	"reflect"
	"testing"
	"time"

	"github.com/caas-team/sparrow/pkg/checks"
	gitlabmock "github.com/caas-team/sparrow/pkg/sparrow/targets/test"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/stretchr/testify/assert"

	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/checks/register"
	"github.com/caas-team/sparrow/pkg/checks/types"
	"github.com/caas-team/sparrow/pkg/config"
	"github.com/caas-team/sparrow/pkg/db"
)

func TestSparrow_ReconcileChecks(t *testing.T) {
	ctx, cancel := logger.NewContextWithLogger(context.Background())
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
		StartupFunc: func(ctx context.Context, cResult chan<- types.Result) error {
			return nil
		},
		GetMetricCollectorsFunc: func() []prometheus.Collector {
			return []prometheus.Collector{}
		},
	}

	register.RegisteredChecks = map[string]func() checks.Check{
		"alpha": func() checks.Check { return &mockCheck },
		"beta":  func() checks.Check { return &mockCheck },
		"gamma": func() checks.Check { return &mockCheck },
	}

	type fields struct {
		checks      map[string]checks.Check
		resultFanIn map[string]chan types.Result

		cResult    chan types.ResultDTO
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
				resultFanIn: make(map[string]chan types.Result),
			},
			newChecksConfig: map[string]any{
				"alpha": "I like sparrows",
			},
		},
		{
			name: "on checks registered and register another",
			fields: fields{
				checks: map[string]checks.Check{
					"alpha": register.RegisteredChecks["alpha"](),
				},
				cfg:         &config.Config{},
				cCfgChecks:  make(chan map[string]any, 1),
				resultFanIn: make(map[string]chan types.Result),
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
					"alpha": register.RegisteredChecks["alpha"](),
				},
				cfg:         &config.Config{},
				cCfgChecks:  make(chan map[string]any, 1),
				resultFanIn: make(map[string]chan types.Result),
			},
			newChecksConfig: map[string]any{},
		},
		{
			name: "two checks registered, register another and unregister one",
			fields: fields{
				checks: map[string]checks.Check{
					"alpha": register.RegisteredChecks["alpha"](),
					"gamma": register.RegisteredChecks["alpha"](),
				},
				cfg:         &config.Config{},
				cCfgChecks:  make(chan map[string]any, 1),
				resultFanIn: make(map[string]chan types.Result),
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
				metrics:     NewMetrics(),
			}

			// Send new config to channel
			s.cfg.Checks = tt.newChecksConfig

			s.ReconcileChecks(ctx)

			for newChecksConfigName := range tt.newChecksConfig {
				check := register.RegisteredChecks[newChecksConfigName]()
				assert.Equal(t, check, s.checks[newChecksConfigName])
			}
		})
	}
}

func Test_fanInResults(t *testing.T) {
	checkChan := make(chan types.Result, 1)
	cResult := make(chan types.ResultDTO, 1)
	name := "check"
	go fanInResults(checkChan, cResult, name)

	result := types.Result{
		Timestamp: time.Time{},
		Err:       "",
		Data:      0,
	}

	checkChan <- result
	output := <-cResult

	want := types.ResultDTO{
		Name:   name,
		Result: &result,
	}

	if !reflect.DeepEqual(output, want) {
		t.Errorf("fanInResults() = %v, want %v", output, want)
	}

	close(checkChan)
}

// TestSparrow_Run_FullComponentStart tests that the Run method starts the API,
// loader and a targetManager all start.
func TestSparrow_Run_FullComponentStart(t *testing.T) {
	// create simple file loader config
	c := &config.Config{
		Api: config.ApiConfig{ListeningAddress: ":9090"},
		Loader: config.LoaderConfig{
			Type:     "file",
			File:     config.FileLoaderConfig{Path: "../config/testdata/config.yaml"},
			Interval: time.Second * 1,
		},
		TargetManager: config.TargetManagerConfig{
			CheckInterval:        time.Second * 1,
			RegistrationInterval: time.Second * 1,
			UnhealthyThreshold:   time.Second * 1,
			Gitlab: config.GitlabTargetManagerConfig{
				BaseURL:   "https://gitlab.com",
				Token:     "my-cool-token",
				ProjectID: 42,
			},
		},
	}

	// start sparrow
	s := New(c)
	ctx := context.Background()
	go func() {
		err := s.Run(ctx)
		if err != nil {
			t.Errorf("Sparrow.Run() error = %v", err)
		}
	}()

	t.Log("Running sparrow for 10ms")
	time.Sleep(time.Millisecond * 10)
}

// TestSparrow_Run_ContextCancel tests that after a context cancels the Run method
// will return an error and all started components will be shut down.
func TestSparrow_Run_ContextCancel(t *testing.T) {
	// create simple file loader config
	c := &config.Config{
		Api: config.ApiConfig{ListeningAddress: ":9090"},
		Loader: config.LoaderConfig{
			Type:     "file",
			File:     config.FileLoaderConfig{Path: "../config/testdata/config.yaml"},
			Interval: time.Second * 1,
		},
	}

	// start sparrow
	s := New(c)
	s.tarMan = &gitlabmock.MockTargetManager{}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		err := s.Run(ctx)
		t.Logf("Sparrow exited with error: %v", err)
		if err == nil {
			t.Error("Sparrow.Run() should have errored out, no error received")
		}
	}()

	t.Log("Running sparrow for 10ms")
	time.Sleep(time.Millisecond * 10)

	t.Log("Canceling context and waiting for shutdown")
	cancel()
	time.Sleep(time.Millisecond * 30)
}

// TestSparrow_updateCheckTargets tests that the updateCheckTargets method
// updates the check targets, if they exist in the config of the checks.
func TestSparrow_updateCheckTargets(t *testing.T) {
	now := time.Now()
	gt := []types.GlobalTarget{
		{
			Url:      "https://localhost.de",
			LastSeen: now,
		},
	}
	tests := []struct {
		name          string
		config        any
		globalTargets []types.GlobalTarget
		expected      any
	}{
		{
			name:          "no config",
			config:        nil,
			globalTargets: gt,
			expected:      nil,
		},
		{
			name: "config with no targets",
			config: map[string]any{
				"targets": nil,
			},
			globalTargets: gt,
			expected: map[string]any{
				"targets": nil,
			},
		},
		{
			name: "config with non-expected targets type",
			config: map[string]any{
				"targets": "not a slice",
			},
			globalTargets: gt,
			expected: map[string]any{
				"targets": "not a slice",
			},
		},
		{
			name: "config with empty targets",
			config: map[string]any{
				"targets": []any{},
			},
			globalTargets: gt,
			expected: map[string]any{
				"targets": []any{},
			},
		},
		{
			name: "config with non string target slice",
			config: map[string]any{
				"targets": []any{1, 2, 3},
			},
			globalTargets: gt,
			expected: map[string]any{
				"targets": []any{1, 2, 3},
			},
		},
		{
			name: "config with mixed target slice",
			config: map[string]any{
				"targets": []any{"https://gitlab.com", 1, 3},
			},
			globalTargets: gt,
			expected: map[string]any{
				"targets": []any{"https://gitlab.com", 1, 3},
			},
		},
		{
			name: "config with targets",
			config: map[string]any{
				"targets": []any{"https://gitlab.com"},
			},
			globalTargets: gt,
			expected: map[string]any{
				"targets": []string{"https://gitlab.com", "https://localhost.de"},
			},
		},
		{
			name: "config has a target already present in global targets - no duplicates",
			config: map[string]any{
				"targets": []any{"https://localhost.de"},
			},
			globalTargets: gt,
			expected: map[string]any{
				"targets": []string{"https://localhost.de"},
			},
		},
		{
			name: "global targets contains self - do not add to config",
			config: map[string]any{
				"targets": []any{"https://localhost.de"},
			},
			globalTargets: append(gt, types.GlobalTarget{
				Url: "https://wonderhost.usa",
			}),
			expected: map[string]any{
				"targets": []string{"https://localhost.de"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Sparrow{
				tarMan: &gitlabmock.MockTargetManager{
					Targets: tt.globalTargets,
				},
				cfg: &config.Config{
					SparrowName: "wonderhost.usa",
				},
			}
			got := s.updateCheckTargets(tt.config)
			assert.Equal(t, tt.expected, got)
		})
	}
}
