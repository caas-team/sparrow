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
// It leverages the mapstructure package for decoding the input data.
//
// Parameters:
//   - input: This is the source data that needs to be decoded.
//     The input can be of any type, such as a map or a struct.
//
// Returns:
//   - T: The function returns a value of type T, which is the target type that the input is decoded into.
//     The generic type T allows this function to be flexible and used with various types.
//   - error: If the decoding process encounters any issues, an error is returned.
//     This could be due to a mismatch
//     between the input and the target type T or other decoding issues.
//
// The function utilizes a DecoderConfig from the mapstructure package to set up the decoding process.
// This configuration includes:
//   - Setting WeaklyTypedInput to true, allowing for more flexible and forgiving decoding.
//   - Custom DecodeHooks to convert string values to specific types like time.Duration or slice of strings,
//     enhancing the function's ability to handle different types of input data.
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
//
// This function is particularly useful in settings where the type of the input data might vary,
// such as when dealing with configuration files in different formats (JSON, YAML, etc.).
// It provides a convenient and type-safe way to handle such dynamic data.
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
