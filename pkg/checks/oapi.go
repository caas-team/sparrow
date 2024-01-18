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

package checks

import (
	"github.com/caas-team/sparrow/pkg/checks/types"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
)

// OpenapiFromPerfData takes in check perfdata and returns an openapi3.SchemaRef of a result wrapping the perfData
// this is a workaround, since the openapi3gen.NewSchemaRefForValue function does not work with any types
func OpenapiFromPerfData[T any](data T) (*openapi3.SchemaRef, error) {
	checkSchema, err := openapi3gen.NewSchemaRefForValue(types.Result{}, openapi3.Schemas{})
	if err != nil {
		return nil, err
	}
	perfdataSchema, err := openapi3gen.NewSchemaRefForValue(data, openapi3.Schemas{}, openapi3gen.UseAllExportedFields())
	if err != nil {
		return nil, err
	}

	checkSchema.Value.Properties["data"] = perfdataSchema
	return checkSchema, nil
}
