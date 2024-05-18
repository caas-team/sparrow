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
//go:generate moq -out base_check_moq.go . Check
type Check interface {
	// Run is called once, to start running the check. The check should
	// run until the context is canceled and handle problems itself.
	// Returning a non-nil error will cause the shutdown of the check.
	Run(ctx context.Context, cResult chan ResultDTO) error
	// Shutdown is called once when the check is unregistered or sparrow shuts down
	Shutdown()
	// SetConfig is called once when the check is registered
	// This is also called while the check is running, if the remote config is updated
	// This should return an error if the config is invalid
	SetConfig(config Runtime) error
	// GetConfig returns the current configuration of the check
	GetConfig() Runtime
	// Name returns the name of the check
	Name() string
	// Schema returns an openapi3.SchemaRef of the result type returned by the check
	Schema() (*openapi3.SchemaRef, error)
	// GetMetricCollectors allows the check to provide prometheus metric collectors
	GetMetricCollectors() []prometheus.Collector
}

// Base is a struct providing common fields used by implementations of the Check interface.
// It serves as a foundational structure that should be embedded in specific check implementations.
type Base[T Runtime] struct {
	// name is the name of the check
	name string
	// Config is the current configuration of the check
	Config T
	// Mutex for thread-safe access to shared resources within the check implementation
	Mu sync.Mutex
	// Signal channel used to notify about shutdown of a check
	DoneChan chan struct{}
}

func NewBase[T Runtime](name string, config T) Base[T] {
	return Base[T]{
		name:     name,
		Config:   config,
		Mu:       sync.Mutex{},
		DoneChan: make(chan struct{}, 1),
	}
}

// Name returns the name of the check
func (b *Base[T]) Name() string {
	return b.name
}

// SetConfig sets the configuration of the check
func (b *Base[T]) SetConfig(config Runtime) error {
	if cfg, ok := config.(T); ok {
		b.Mu.Lock()
		defer b.Mu.Unlock()
		b.Config = cfg
		return nil
	}

	return ErrConfigMismatch{
		Expected: b.Name(),
		Current:  config.For(),
	}
}

// GetConfig returns the current configuration of the check
func (b *Base[T]) GetConfig() Runtime {
	b.Mu.Lock()
	defer b.Mu.Unlock()
	return b.Config
}

// SendResult sends the result of a check run to the provided channel
func (b *Base[T]) SendResult(channel chan ResultDTO, data any) {
	channel <- ResultDTO{
		Name:   b.Name(),
		Result: &Result{Data: data, Timestamp: time.Now()},
	}
}

// Shutdown shuts down the check
func (b *Base[T]) Shutdown() {
	b.DoneChan <- struct{}{}
	close(b.DoneChan)
}

// Runtime is the interface that all check configurations must implement
//
//go:generate moq -out base_runtime_moq.go . Runtime
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
