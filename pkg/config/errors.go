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

package config

import "errors"

var (
	// ErrInvalidSparrowName is returned when the sparrow name is invalid
	ErrInvalidSparrowName = errors.New("invalid sparrow name")
	// ErrInvalidLoaderInterval is returned when the loader interval is invalid
	ErrInvalidLoaderInterval = errors.New("invalid loader interval")
	// ErrInvalidLoaderHttpURL is returned when the loader http url is invalid
	ErrInvalidLoaderHttpURL = errors.New("invalid loader http url")
	// ErrInvalidLoaderHttpRetryCount is returned when the loader http retry count is invalid
	ErrInvalidLoaderHttpRetryCount = errors.New("invalid loader http retry count")
	// ErrInvalidLoaderFilePath is returned when the loader file path is invalid
	ErrInvalidLoaderFilePath = errors.New("invalid loader file path")
)
