package healthz

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/checks"
)

//go:generate moq -out checker_moq.go . Checker
type Checker interface {
	CheckOverallHealth(ctx context.Context, cks []checks.Check) bool
}

// checker is used to check the health of the sparrow's endpoints
type checker struct {
	addr   string
	client *http.Client
}

// New creates a new healthz checker
// address is the address of the API
func New(address string) Checker {
	return &checker{
		addr:   formatAddress(address),
		client: &http.Client{},
	}
}

func (c *checker) CheckOverallHealth(ctx context.Context, cks []checks.Check) bool {
	return c.isMetricsHealthy(ctx) && c.areChecksHealthy(ctx, cks)
}

// isMetricsHealthy checks if the metrics endpoint is healthy
func (c *checker) isMetricsHealthy(ctx context.Context) bool {
	log := logger.FromContext(ctx)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://%s/metrics", c.addr), http.NoBody)
	if err != nil {
		log.Error("Failed to create request", "error", err)
		return false
	}

	resp, err := c.client.Do(req) //nolint:bodyclose // closed in defer
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
	for _, ck := range cks {
		ok := c.isCheckHealthy(ctx, ck)
		if !ok {
			logger.FromContext(ctx).Warn("Check is unhealthy", "check", ck.Name())
			return false
		}
	}

	return true
}

// isCheckHealthy checks if a single check is healthy
func (c *checker) isCheckHealthy(ctx context.Context, ck checks.Check) bool {
	log := logger.FromContext(ctx).With("check", ck.Name())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://%s/v1/metrics/%s", c.addr, ck.Name()), http.NoBody)
	if err != nil {
		log.Error("Failed to create request", "error", err)
		return false
	}

	resp, err := c.client.Do(req) //nolint:bodyclose // closed in defer
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

// formatAddress formats the address to be used in the healthz checker
func formatAddress(addr string) string {
	// Localhost is a special case, since it's the only address that doesn't need to be formatted
	if addr == "localhost" || addr == "127.0.0.1" || addr == net.IPv6loopback.String() {
		return addr
	}

	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return net.JoinHostPort("localhost", "8080")
	}

	return net.JoinHostPort("localhost", port)
}
