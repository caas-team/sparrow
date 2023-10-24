package checks

import (
	"context"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

// Available Checks will be registered in this map
// The key is the name of the Check
// The name needs to map the configuration item key
var RegisteredChecks = map[string]func(string) Check{
	"rtt": GetRoundtripCheck,
}

type Check interface {
	// Run is called once per check interval
	// this should error if there is a problem running the check
	// Returns an error and a result. Returning a non nil error will cause a shutdown of the system
	Run(ctx context.Context) (Result, error)
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
	// Name returns the name of the check
	Name() string
	// Should return an openapi3.SchemaRef of the result type returned by the check
	Schema() (*openapi3.SchemaRef, error)
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
