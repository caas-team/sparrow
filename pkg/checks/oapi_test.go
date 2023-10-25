package checks

import (
	"reflect"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestOpenapiFromPerfData(t *testing.T) {
	type args[T any] struct {
		perfData T
	}
	type cases[T any] struct {
		name    string
		args    args[T]
		want    *openapi3.SchemaRef
		wantErr bool
	}
	tests := []cases[string]{
		{name: "int", args: args[string]{perfData: "hello world"}, want: &openapi3.SchemaRef{Value: openapi3.NewObjectSchema().WithProperties(map[string]*openapi3.Schema{"error": {Type: "string"}, "data": {Type: "string"}, "timestamp": {Type: "string", Format: "date-time"}})}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := OpenapiFromPerfData(tt.args.perfData)
			if (err != nil) != tt.wantErr {
				t.Errorf("OpenapiFromPerfData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OpenapiFromPerfData() = %v, want %v", got, tt.want)
			}

		})
	}
}
