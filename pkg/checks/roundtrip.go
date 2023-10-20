package checks

import (
	"context"
	"errors"
	"net/http"
)

// ensure that RoundTrip implements the Check interface
var _ Check = &RoundTrip{}

type roundTripConfig struct{}

// RoundTrip is a check that measures the round trip time of a request
type RoundTrip struct {
	name   string
	c      chan<- Result
	config roundTripConfig
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

var ErrInvalidConfig = errors.New("invalid config")
