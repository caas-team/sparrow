package checks

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/api"
	"github.com/getkin/kin-openapi/openapi3"
)

// ensure that RoundTrip implements the Check interface
var _ Check = (*RoundTrip)(nil)

type RoundTripConfig struct{}
type roundTripData struct {
	Ms int64 `json:"ms"`
}

// RoundTrip is a check that measures the round trip time of a request
type RoundTrip struct {
	c      chan<- Result
	config RoundTripConfig
}

// Constructor for the RoundtripCheck
func GetRoundtripCheck() Check {
	return &RoundTrip{}
}

func (rt *RoundTrip) Run(ctx context.Context) error {
	ctx, cancel := logger.NewContextWithLogger(ctx, "roundTrip")
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
			fmt.Println("Sending data to db")
			rt.c <- Result{Timestamp: time.Now(), Err: "", Data: roundTripData{Ms: 1000}}
		}

	}
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

func (rt *RoundTrip) SetConfig(ctx context.Context, config any) error {
	checkConfig, ok := config.(RoundTripConfig)
	if !ok {
		return ErrInvalidConfig
	}
	rt.config = checkConfig
	return nil
}

func (rt *RoundTrip) Schema() (*openapi3.SchemaRef, error) {
	return OpenapiFromPerfData[roundTripData](roundTripData{})

}

func (rt *RoundTrip) RegisterHandler(ctx context.Context, router *api.RoutingTree) {
	router.Add(http.MethodGet, "/rtt", rt.handleRoundtrip)
}

func (rt *RoundTrip) DeregisterHandler(ctx context.Context, router *api.RoutingTree) {
	router.Remove(http.MethodGet, "/rtt")
}

func (rt *RoundTrip) handleRoundtrip(w http.ResponseWriter, r *http.Request) {
	// TODO handle
}
