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

package checks

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/caas-team/sparrow/internal/helper"
	"github.com/caas-team/sparrow/pkg/api"
)

var (
	// RegisteredChecks will be registered in this map
	// The key is the name of the Check
	// The name needs to map the configuration item key
	RegisteredChecks = map[string]func() Check{
		"health":  NewHealthCheck,
		"latency": NewLatencyCheck,
	}
	// DefaultRetry provides a default configuration for the retry mechanism
	DefaultRetry = helper.RetryConfig{
		Count: 3,
		Delay: time.Second,
	}
)

// Check implementations are expected to perform specific monitoring tasks and report results.
//
//go:generate moq -out checks_moq.go . Check
type Check interface {
	// Run is called once, to start running the check. The check should
	// run until the context is canceled and handle problems itself.
	// Returning a non-nil error will cause the shutdown of the check.
	Run(ctx context.Context) error
	// Startup is called once when the check is registered
	// In the Run() method, the check should send results to the cResult channel
	// this will cause sparrow to update its data store with the results
	Startup(ctx context.Context, cResult chan<- Result) error
	// Shutdown is called once when the check is unregistered or sparrow shuts down
	Shutdown(ctx context.Context) error
	// SetConfig is called once when the check is registered
	// This is also called while the check is running, if the remote config is updated
	// This should return an error if the config is invalid
	SetConfig(config Config) error
	// GetConfig returns the current configuration of the check
	GetConfig() Config
	// Name returns the name of the check
	Name() string
	// Schema returns an openapi3.SchemaRef of the result type returned by the check
	Schema() (*openapi3.SchemaRef, error)
	// RegisterHandler Allows the check to register a handler on sparrows http server at runtime
	RegisterHandler(ctx context.Context, router *api.RoutingTree)
	// DeregisterHandler allows the check to deregister a handler on sparrows http server at runtime
	DeregisterHandler(ctx context.Context, router *api.RoutingTree)
	// GetMetricCollectors allows the check to provide prometheus metric collectors
	GetMetricCollectors() []prometheus.Collector
}

// New creates a new check instance from the given name
func New(cfg Config) (Check, error) {
	if f, ok := RegisteredChecks[cfg.For()]; ok {
		c := f()
		err := c.SetConfig(cfg)
		return f(), err
	}
	return nil, errors.New("unknown check type")
}

// NewChecksFromConfig creates all checks defined provided config
func NewChecksFromConfig(cfg RuntimeConfig) (map[string]Check, error) {
	checks := make(map[string]Check)
	for _, c := range cfg.Checks.Iter() {
		check, err := New(c)
		if err != nil {
			return nil, err
		}
		checks[check.Name()] = check
	}
	return checks, nil
}

// CheckBase is a struct providing common fields used by implementations of the Check interface.
// It serves as a foundational structure that should be embedded in specific check implementations.
type CheckBase struct {
	// Mutex for thread-safe access to shared resources within the check implementation
	mu sync.Mutex
	// Essential for passing check results back to the Sparrow; must be utilized by Check implementations
	cResult chan<- Result
	// Signal channel used to notify about shutdown of a check
	done chan bool
}

// Result encapsulates the outcome of a check run.
type Result struct {
	// data contains performance metrics about the check run
	Data any `json:"data"`
	// Timestamp is the UTC time the check was run
	Timestamp time.Time `json:"timestamp"`
	// Err should be nil if the check ran successfully indicating the check is "healthy"
	// if the check failed, this should be an error message that will be logged and returned to an API user
	Err string `json:"error"`
}

// GlobalTarget includes the basic information regarding
// other Sparrow instances, which this Sparrow can communicate with.
type GlobalTarget struct {
	Url      string    `json:"url"`
	LastSeen time.Time `json:"lastSeen"`
}

// ResultDTO is a data transfer object used to associate a check's name with its result.
type ResultDTO struct {
	Name   string
	Result *Result
}

// Config is the interface that all check configurations must implement
type Config interface {
	// Validate validates the check's configuration
	Validate() error
	// For returns the name of the check being configured
	For() string
}
