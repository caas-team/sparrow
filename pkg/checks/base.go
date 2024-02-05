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

// CheckBase is a struct providing common fields used by implementations of the Check interface.
// It serves as a foundational structure that should be embedded in specific check implementations.
type CheckBase struct {
	// Mutex for thread-safe access to shared resources within the check implementation
	Mu sync.Mutex
	// Essential for passing check results back to the Sparrow; must be utilized by Check implementations
	CResult chan<- Result
	// Signal channel used to notify about shutdown of a check
	Done chan bool
}

// Runtime is the interface that all check configurations must implement
type Runtime interface {
	// For returns the name of the check being configured
	For() string
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
