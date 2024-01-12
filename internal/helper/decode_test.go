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

package helper

import (
	"reflect"
	"testing"
	"time"
)

type MyConfig struct {
	Count     int
	Name      string
	Urls      []string
	Timeout   time.Duration
	Activated bool `mapstructure:"enabled"`
}

// Test case structure
type test[T any] struct {
	name      string
	input     any
	want      T
	expectErr bool
}

func TestDecode(t *testing.T) {
	tests := []test[MyConfig]{
		{
			name: "Valid input",
			input: map[string]any{
				"Count":   "123",
				"Name":    "example",
				"Urls":    "one,two,three",
				"Timeout": "30m",
				"Enabled": "true",
			},
			want: MyConfig{
				Count:     123,
				Name:      "example",
				Urls:      []string{"one", "two", "three"},
				Timeout:   30 * time.Minute,
				Activated: true,
			},
			expectErr: false,
		},
		{
			name:      "Invalid input type",
			input:     "invalid input",
			want:      MyConfig{},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Decode[MyConfig](tt.input)

			if (err != nil) != tt.expectErr {
				t.Errorf("Test %s failed: expected error: %v, got: %v", tt.name, tt.expectErr, err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Test %s failed: expected result: %+v, got: %+v", tt.name, tt.want, got)
			}
		})
	}
}
