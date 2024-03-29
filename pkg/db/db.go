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

package db

import (
	"sync"

	"github.com/caas-team/sparrow/pkg/checks"
)

type DB interface {
	Save(result checks.ResultDTO)
	Get(check string) (result checks.Result, ok bool)
	List() map[string]checks.Result
}

var _ DB = (*InMemory)(nil)

type InMemory struct {
	// if we want to save a timeseries
	// we can use a map of ringbuffers instead of a single value
	// this ensures that we can save the last N results, where N is the size of the ringbuffer
	// without having to worry about the size of the map
	data sync.Map
}

// NewInMemory creates a new in-memory database
func NewInMemory() *InMemory {
	return &InMemory{
		data: sync.Map{},
	}
}

func (i *InMemory) Save(result checks.ResultDTO) {
	i.data.Store(result.Name, result.Result)
}

func (i *InMemory) Get(check string) (checks.Result, bool) {
	tmp, ok := i.data.Load(check)
	if !ok {
		return checks.Result{}, false
	}
	// this should not fail, otherwise this will panic
	result := tmp.(*checks.Result)

	return *result, true
}

// List returns a copy of the map
func (i *InMemory) List() map[string]checks.Result {
	results := make(map[string]checks.Result)
	i.data.Range(func(key, value any) bool {
		// this assertion should not fail, unless we have a bug somewhere
		check := key.(string)
		result := value.(*checks.Result)

		results[check] = *result
		return true
	})

	return results
}
