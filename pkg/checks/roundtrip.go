package checks

import (
	"context"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
)

// ensure that RoundTrip implements the Check interface
var _ Check = &RoundTrip{}

type roundTripConfig struct{}
type PerfData struct {
	Ms int64 `json:"ms"`
}

// RoundTrip is a check that measures the round trip time of a request
type RoundTrip struct {
	name   string
	c      chan<- Result
	config roundTripConfig
}

// Constructor for the RoundtripCheck
func GetRoundtripCheck(name string) Check {
	return &RoundTrip{
		name: name,
	}
}

func (rt *RoundTrip) Run(ctx context.Context) (Result, error) {
	return Result{}, nil
}

func (rt *RoundTrip) Startup(ctx context.Context, cResult chan<- Result) error {
	// TODO register http handler for this check
	http.HandleFunc("/rtt", func(w http.ResponseWriter, r *http.Request) {
		// TODO handle
	})

	rt.c = cResult
	return nil
}

// Shutdown is called once when the check is unregistered or sparrow shuts down

func (rt *RoundTrip) Shutdown(ctx context.Context) error {
	http.Handle("/rtt", http.NotFoundHandler())

	return nil
}

// Name returns the name of the check
func (rt *RoundTrip) Name() string {
	return rt.name
}

func (rt *RoundTrip) SetConfig(ctx context.Context, config any) error {
	checkConfig, ok := config.(roundTripConfig)
	if !ok {
		return ErrInvalidConfig
	}
	rt.config = checkConfig
	return nil
}

func (rt *RoundTrip) Schema() (*openapi3.SchemaRef, error) {
	ref, err := openapi3gen.NewSchemaRefForValue(&Result{
		Data: PerfData{},
	}, openapi3.Schemas{})
	if err != nil {
		panic("Failed to generate schema" + err.Error())
	}

	return ref, nil
}

func NewRoundtrip() *RoundTrip {
	return &RoundTrip{
		name: "rtt",
	}
}
