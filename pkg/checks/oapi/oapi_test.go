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

package oapi

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
