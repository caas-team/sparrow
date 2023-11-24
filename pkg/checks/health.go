package checks

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/caas-team/sparrow/internal/helper"
	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/api"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/mitchellh/mapstructure"
)

type HealthConfig struct {
	Enabled        bool     `json:"enabled,omitempty"`
	Targets        []string `json:"targets,omitempty"`
	HealthEndpoint bool     `json:"healthEndpoint,omitempty"`
}
type healthData struct {
	Targets []Target `json:"targets"`
}
type Target struct {
	Target string `json:"target"`
	Status string `json:"status"`
}

// Health is a check that measures the availability of an endpoint
type Health struct {
	route  string
	config HealthConfig
	c      chan<- Result
	done   chan bool
}

// Constructor for the HealthCheck
func GetHealthCheck() Check {
	return &Health{
		route: "/health",
	}
}

func (h *Health) Run(ctx context.Context) error {
	ctx, cancel := logger.NewContextWithLogger(ctx, "health")
	defer cancel()
	log := logger.FromContext(ctx)

	for {
		delay := time.Minute
		log.Info("Next health check will run after delay", "delay", delay.String())
		select {
		case <-ctx.Done():
			log.Debug("Context closed. Stopping health check")
			return ctx.Err()
		case <-h.done:
			log.Debug("Soft shut down")
			return nil
		case <-time.After(delay):
			log.Info("Start health check run")
			healthData := h.Check(ctx)

			log.Debug("Saving health check data to database")
			h.c <- Result{Timestamp: time.Now(), Data: healthData}

			log.Info("Successfully finished health check run")
		}
	}
}

func (h *Health) Startup(ctx context.Context, cResult chan<- Result) error {
	h.c = cResult
	return nil
}

// Shutdown is called once when the check is unregistered or sparrow shuts down
func (h *Health) Shutdown(ctx context.Context) error {
	http.Handle(h.route, http.NotFoundHandler())
	h.done <- true

	return nil
}

func (h *Health) SetConfig(ctx context.Context, config any) error {
	var checkCfg HealthConfig
	if err := mapstructure.Decode(config, &checkCfg); err != nil {
		return ErrInvalidConfig
	}
	h.config = checkCfg
	return nil
}

func (h *Health) Schema() (*openapi3.SchemaRef, error) {
	return OpenapiFromPerfData[healthData](healthData{})

}

func (h *Health) RegisterHandler(ctx context.Context, router *api.RoutingTree) {
	if h.config.Enabled {
		router.Add(http.MethodGet, h.route, func(_ http.ResponseWriter, _ *http.Request) { return })
	}
}

func (h *Health) DeregisterHandler(ctx context.Context, router *api.RoutingTree) {
	router.Remove(http.MethodGet, h.route)
}

func (h *Health) Check(ctx context.Context) healthData {
	log := logger.FromContext(ctx)
	if len(h.config.Targets) != 0 {
		log.Debug("No targets defined")
		return healthData{}
	}
	log.Debug("Getting health status for each target in separate routine", "amount", len(h.config.Targets))

	var healthData healthData
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, target := range h.config.Targets {
		target := target
		wg.Add(1)
		l := log.With("target", target)

		getHealthRetry := helper.Retry(func(ctx context.Context) error {
			return getHealth(ctx, target)
		}, helper.RetryConfig{
			Count: 3,
			Delay: time.Microsecond,
		})

		go func() {
			defer wg.Done()
			targetData := Target{
				Target: target,
				Status: "healthy",
			}

			l.Debug("Starting retry routine to get health of target")
			if err := getHealthRetry(ctx); err != nil {
				targetData.Status = "unhealthy"
			}

			l.Debug("Successfully got health status of target", "status", targetData.Status)
			mu.Lock()
			healthData.Targets = append(healthData.Targets, targetData)
			mu.Unlock()
		}()
	}

	log.Debug("Waiting for all routines to finish")
	wg.Wait()

	log.Info("Successfully got health status from all targets")
	return healthData
}

func getHealth(ctx context.Context, url string) error {
	log := logger.FromContext(ctx).With("url", url)

	client := http.DefaultClient
	client.Timeout = time.Second * 5

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Error("Could not create http GET request", "error", err.Error())
		return err
	}

	res, err := client.Do(req)
	if err != nil {
		log.Error("Http get request failed", "error", err.Error())
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Error("Http get request failed", "status", res.Status)
		return fmt.Errorf("request failed, status is %s", res.Status)
	}

	return nil
}
