package sparrow

import (
	"reflect"
	"testing"

	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/config"
	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v3"
)

func TestSparrow_getOpenapi(t *testing.T) {
	type fields struct {
		checks []checks.Check
		config *config.Config
		c      chan checks.Result
	}
	type test struct {
		name    string
		fields  fields
		want    openapi3.T
		wantErr bool
	}
	tests := []test{
		{name: "no checks registered", fields: fields{checks: []checks.Check{}, config: config.NewConfig()}, want: oapiBoilerplate, wantErr: false},
		{name: "check registered", fields: fields{checks: []checks.Check{checks.GetRoundtripCheck("rtt")}, config: config.NewConfig()}, want: oapiBoilerplate, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Sparrow{
				checks: tt.fields.checks,
				cfg:    tt.fields.config,
				c:      tt.fields.c,
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
