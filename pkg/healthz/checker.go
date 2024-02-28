package healthz

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/api"
	"github.com/caas-team/sparrow/pkg/checks"
)

//go:generate moq -out checker_moq.go . Checker
type Checker interface {
	CheckOverallHealth(ctx context.Context, cks []checks.Check) bool
}

// checker is used to check the health of the sparrow's endpoints
type checker struct {
	addr string
}

// New creates a new healthz checker
func New(cfg api.Config) Checker {
	return &checker{
		addr: cfg.ListeningAddress,
	}
}

func (c *checker) CheckOverallHealth(ctx context.Context, cks []checks.Check) bool {
	return c.isMetricsHealthy(ctx) && c.areChecksHealthy(ctx, cks)
}

// isMetricsHealthy checks if the metrics endpoint is healthy
func (c *checker) isMetricsHealthy(ctx context.Context) bool {
	log := logger.FromContext(ctx)
	client := &http.Client{}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://%s/metrics", c.addr), http.NoBody)
	if err != nil {
		log.Error("Failed to create request", "error", err)
		return false
	}

	resp, err := client.Do(req) //nolint:bodyclose // closed in defer
	if err != nil {
		return false
	}
	defer func(b io.ReadCloser) {
		err = b.Close()
		if err != nil {
			logger.FromContext(ctx).Error("Failed to close response body", "error", err)
		}
	}(resp.Body)

	return resp.StatusCode == http.StatusOK
}

// areChecksHealthy checks if the checks are healthy
func (c *checker) areChecksHealthy(ctx context.Context, cks []checks.Check) bool {
	client := &http.Client{}
	for _, ck := range cks {
		ok := c.isCheckHealthy(ctx, ck, client)
		if !ok {
			logger.FromContext(ctx).Warn("Check is unhealthy", "check", ck.Name())
			return false
		}
	}

	return true
}

// isCheckHealthy checks if a single check is healthy
func (c *checker) isCheckHealthy(ctx context.Context, ck checks.Check, client *http.Client) bool {
	log := logger.FromContext(ctx).With("check", ck.Name())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://%s/v1/metrics/%s", c.addr, ck.Name()), http.NoBody)
	if err != nil {
		log.Error("Failed to create request", "error", err)
		return false
	}

	resp, err := client.Do(req) //nolint:bodyclose // closed in defer
	if err != nil {
		log.Error("Failed to send request", "error", err)
		return false
	}
	defer func(b io.ReadCloser) {
		err = b.Close()
		if err != nil {
			log.Error("Failed to close response body", "error", err)
		}
	}(resp.Body)

	return resp.StatusCode == http.StatusOK
}
