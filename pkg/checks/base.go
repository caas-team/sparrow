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
	"context"
	"sync"
	"time"

	"github.com/caas-team/sparrow/internal/helper"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/prometheus/client_golang/prometheus"
)

// DefaultRetry provides a default configuration for the retry mechanism
var DefaultRetry = helper.RetryConfig{
	Count: 3,
	Delay: time.Second,
}

// Check implementations are expected to perform specific monitoring tasks and report results.
//
//go:generate moq -out base_moq.go . Check
type Check interface {
	// Run is called once, to start running the check. The check should
	// run until the context is canceled and handle problems itself.
	// Returning a non-nil error will cause the shutdown of the check.
	Run(ctx context.Context, cResult chan ResultDTO) error
	// Shutdown is called once when the check is unregistered or sparrow shuts down.
	Shutdown()
	// UpdateConfig updates the configuration of the check.
	// It is called when the runtime configuration is updated.
	// The check should handle the update itself.
	// Returns an error if the configuration is invalid.
	UpdateConfig(config Runtime) error
	// GetConfig returns the current configuration of the check.
	GetConfig() Runtime
	// Name returns the name of the check.
	Name() string
	// Schema returns an openapi3.SchemaRef of the result type returned by the check.
	Schema() (*openapi3.SchemaRef, error)
	// GetMetricCollectors allows the check to provide prometheus metric collectors.
	GetMetricCollectors() []prometheus.Collector
	// RemoveLabelledMetrics allows the check to remove the prometheus metrics
	// of the check whose `target` label matches the passed value.
	RemoveLabelledMetrics(target string) error
}

// Base is a struct providing common fields and methods used by implementations of the [Check] interface.
// It serves as a foundational structure that should be embedded in specific check implementations.
type Base struct {
	// Mutex for thread-safe access to shared resources within the check implementation.
	Mutex sync.Mutex
	// Done channel is used to notify about shutdown of a check.
	Done chan struct{}
	// closed is a flag indicating if the check has been shut down.
	closed bool
}

// NewBase creates a new instance of the [Base] struct.
func NewBase() Base {
	return Base{
		Mutex:  sync.Mutex{},
		Done:   make(chan struct{}, 1),
		closed: false,
	}
}

// Shutdown closes the DoneChan to signal the check to stop running.
func (b *Base) Shutdown() {
	b.Mutex.Lock()
	defer b.Mutex.Unlock()
	if !b.closed {
		close(b.Done)
		b.closed = true
	}
}

// Runtime is the interface that all check configurations must implement
type Runtime interface {
	// For returns the name of the check being configured
	For() string
	// Validate checks if the configuration is valid
	Validate() error
}

// Result encapsulates the outcome of a check run.
type Result struct {
	// Data contains performance metrics about the check run
	Data any `json:"data"`
	// Timestamp is the UTC time the check was run
	Timestamp time.Time `json:"timestamp"`
}

// ResultDTO is a data transfer object used to associate a check's name with its result.
type ResultDTO struct {
	Name   string
	Result *Result
}

// GlobalTarget includes the basic information regarding
// other Sparrow instances, which this Sparrow can communicate with.
type GlobalTarget struct {
	Url      string    `json:"url"`
	LastSeen time.Time `json:"lastSeen"`
}
