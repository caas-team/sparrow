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
	config HealthConfig
	c      chan<- Result
	done   chan bool
}

// Constructor for the HealthCheck
func GetHealthCheck() Check {
	return &Health{}
}

func (h *Health) Run(ctx context.Context) error {
	ctx, cancel := logger.NewContextWithLogger(ctx, "health")
	defer cancel()
	log := logger.FromContext(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-h.done:
			return nil
		case <-time.After(time.Second * 10):
			log.Info("Run health check")
			healthData := h.Check(ctx)
			fmt.Println("Sending data to db")
			h.c <- Result{Timestamp: time.Now(), Data: healthData}
		}
	}
}

func (h *Health) Startup(ctx context.Context, cResult chan<- Result) error {
	h.c = cResult
	return nil
}

// Shutdown is called once when the check is unregistered or sparrow shuts down
func (h *Health) Shutdown(ctx context.Context) error {
	http.Handle("/health", http.NotFoundHandler())
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
	router.Add(http.MethodGet, "/health", h.handleHealth)
}

func (h *Health) DeregisterHandler(ctx context.Context, router *api.RoutingTree) {
	router.Remove(http.MethodGet, "/health")
}

func (h *Health) handleHealth(w http.ResponseWriter, r *http.Request) {
	return
}

func (h *Health) Check(ctx context.Context) healthData {
	log := logger.FromContext(ctx)

	var healthData healthData
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, target := range h.config.Targets {
		target := target
		wg.Add(1)
		l := log.With("target", target)
		l.Info("Getting health status")

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

			if err := getHealthRetry(ctx); err != nil {
				targetData.Status = "unhealthy"
			}
			l.Info("Health status", "status", targetData.Status)
			mu.Lock()
			healthData.Targets = append(healthData.Targets, targetData)
			mu.Unlock()
		}()
	}
	if len(h.config.Targets) != 0 {
		log.Info("Wait for health status on all targets")
		wg.Wait()
		log.Info("Successfully got status from all targets")
	} else {
		log.Info("No targets defined")
	}

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
