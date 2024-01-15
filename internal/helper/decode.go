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

import "github.com/mitchellh/mapstructure"

// Decode is a generic function that takes an input of any type and attempts to decode it into a specified type T.
// Returns the decoded value of type 'T' and any error encountered during decoding.
//
// Example Usage:
// Suppose we have a struct named MyConfig with fields A (int), B (string), and C ([]string).
// We can use Decode to convert a map[string]any to MyConfig type as shown below:
//
//	configMap := map[string]any{
//	    "A": "123",
//	    "B": "example",
//	    "C": "one,two,three",
//	}
//	var myConfig MyConfig
//	myConfig, err := Decode[MyConfig](configMap)
//	if err != nil {
//	    // handle error
//	}
func Decode[T any](input any) (T, error) {
	var result T
	config := &mapstructure.DecoderConfig{
		Metadata:         nil,
		WeaklyTypedInput: true,
		Result:           &result,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		),
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return result, err
	}

	if err := decoder.Decode(input); err != nil {
		return result, err
	}

	return result, nil
}
