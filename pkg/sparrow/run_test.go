package sparrow

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/caas-team/sparrow/pkg/db"

	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/config"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestSparrow_getOpenapi(t *testing.T) {
	type fields struct {
		checks  map[string]checks.Check
		config  *config.Config
		cResult chan checks.ResultDTO
	}
	type test struct {
		name    string
		fields  fields
		want    openapi3.T
		wantErr bool
	}
	tests := []test{

		{name: "no checks registered", fields: fields{checks: map[string]checks.Check{}, config: config.NewConfig()}, want: oapiBoilerplate, wantErr: false},
		{name: "check registered", fields: fields{checks: map[string]checks.Check{"rtt": checks.GetRoundtripCheck()}, config: config.NewConfig()}, want: oapiBoilerplate, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Sparrow{
				checks:      tt.fields.checks,
				cfg:         tt.fields.config,
				cResult:     tt.fields.cResult,
				resultFanIn: make(map[string]chan checks.Result),
			}
			got, err := s.Openapi()
			if (err != nil) != tt.wantErr {
				t.Errorf("Sparrow.getOpenapi() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Sparrow.getOpenapi() = %v, want %v", got, tt.want)
			}

			bgot, err := yaml.Marshal(got)
			if err != nil {
				t.Errorf("OpenapiFromPerfData() error = %v", err)
				return
			}
			t.Logf("\nGot:\n%s", string(bgot))

			bwant, err := yaml.Marshal(tt.want)
			if err != nil {
				t.Errorf("OpenapiFromPerfData() error = %v", err)
				return
			}

			if !reflect.DeepEqual(bgot, bwant) {
				t.Errorf("Sparrow.getOpenapi() = %v, want %v", bgot, bwant)
			}
		})
	}
}

func TestSparrow_ReconcileChecks(t *testing.T) {
	mockCheck := checks.CheckMock{
		RunFunc: func(ctx context.Context) (checks.Result, error) {
			return checks.Result{}, nil
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
				cCfgChecks:  make(chan map[string]any),
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
				cCfgChecks:  make(chan map[string]any),
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
				cCfgChecks:  make(chan map[string]any),
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
				cCfgChecks:  make(chan map[string]any),
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

			s.ReconcileChecks(context.Background())

			for newChecksConfigName := range tt.newChecksConfig {
				check := checks.RegisteredChecks[newChecksConfigName]()
				assert.Equal(t, check, s.checks[newChecksConfigName])
			}
		})
	}
}

func Test_fanInResults(t *testing.T) {

	checkChan := make(chan checks.Result)
	cResult := make(chan checks.ResultDTO)
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
