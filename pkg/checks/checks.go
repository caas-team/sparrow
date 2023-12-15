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
	"net/http"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/caas-team/sparrow/pkg/api"
)

// RegisteredChecks will be registered in this map
// The key is the name of the Check
// The name needs to map the configuration item key
var RegisteredChecks = map[string]func() Check{
	"health":  NewHealthCheck,
	"latency": NewLatencyCheck,
}

//go:generate moq -out checks_moq.go . Check
type Check interface {
	// Run is called once per check interval
	// this should error if there is a problem running the check
	// Returns an error and a result. Returning a non nil error will cause a shutdown of the system
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
	SetConfig(ctx context.Context, config any) error
	// SetClient sets an HTTP client for the check. This method is used to configure
	// the check with a specific HTTP client, which can be used for network requests
	// during the check's execution
	SetClient(c *http.Client)
	// Should return an openapi3.SchemaRef of the result type returned by the check
	Schema() (*openapi3.SchemaRef, error)
	// Allows the check to register a handler on sparrows http server at runtime
	RegisterHandler(ctx context.Context, router *api.RoutingTree)
	// Allows the check to deregister a handler on sparrows http server at runtime
	DeregisterHandler(ctx context.Context, router *api.RoutingTree)
	// Allows the check to provide prometheus metric collectors
	GetMetricCollectors() []prometheus.Collector
}

type Result struct {
	// data contains performance metrics about the check run
	Data any `json:"data"`
	// Timestamp is the UTC time the check was run
	Timestamp time.Time `json:"timestamp"`
	// Err should be nil if the check ran successfully indicating the check is "healthy"
	// if the check failed, this should be an error message that will be logged and returned to an API user
	Err string `json:"error"`
}

type ResultDTO struct {
	Name   string
	Result *Result
}
