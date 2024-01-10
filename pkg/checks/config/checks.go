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

import (
	"net/http"
	"sync"
	"time"

	"github.com/caas-team/sparrow/internal/helper"
)

var (
	// BasicRetryConfig provides a default configuration for the retry mechanism
	DefaultRetry = helper.RetryConfig{
		Count: 3,
		Delay: time.Second,
	}
)

// CheckBase is a struct providing common fields used by implementations of the Check interface.
// It serves as a foundational structure that should be embedded in specific check implementations.
type CheckBase struct {
	Mu      sync.Mutex
	CResult chan<- Result
	Done    chan bool
	Client  *http.Client
}

// GlobalTarget includes the basic information regarding
// other Sparrow instances, which this Sparrow can communicate with.
type GlobalTarget struct {
	Url      string    `json:"url"`
	LastSeen time.Time `json:"lastSeen"`
}
