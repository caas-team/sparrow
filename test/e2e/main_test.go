package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/caas-team/sparrow/test/framework"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

const (
	checkInterval = 20 * time.Second
	checkTimeout  = 15 * time.Second
)

func TestSparrow_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests")
	}

	tests := []struct {
		name          string
		startup       framework.ConfigBuilder
		checks        []framework.CheckBuilder
		wantEndpoints map[string]int
	}{
		{
			name:    "no checks",
			startup: *framework.NewConfig(),
			checks:  nil,
			wantEndpoints: map[string]int{
				"http://localhost:8080/v1/metrics/health":     http.StatusNotFound,
				"http://localhost:8080/v1/metrics/latency":    http.StatusNotFound,
				"http://localhost:8080/v1/metrics/dns":        http.StatusNotFound,
				"http://localhost:8080/v1/metrics/traceroute": http.StatusNotFound,
			},
		},
		{
			name:    "with health check",
			startup: *framework.NewConfig(),
			checks: []framework.CheckBuilder{
				framework.NewHealthCheck().
					WithInterval(checkInterval).
					WithTimeout(checkTimeout).
					WithTargets("https://www.example.com/", "https://www.google.com/"),
			},
			wantEndpoints: map[string]int{
				"http://localhost:8080/v1/metrics/health":     http.StatusOK,
				"http://localhost:8080/v1/metrics/latency":    http.StatusNotFound,
				"http://localhost:8080/v1/metrics/dns":        http.StatusNotFound,
				"http://localhost:8080/v1/metrics/traceroute": http.StatusNotFound,
			},
		},
		{
			name:    "with health, latency and dns checks",
			startup: *framework.NewConfig(),
			checks: []framework.CheckBuilder{
				framework.NewHealthCheck().
					WithInterval(checkInterval).
					WithTimeout(checkTimeout).
					WithTargets("https://www.example.com/"),
				framework.NewLatencyCheck().
					WithInterval(checkInterval).
					WithTimeout(checkTimeout).
					WithTargets("https://www.example.com/"),
				framework.NewDNSCheck().
					WithInterval(checkInterval).
					WithTimeout(checkTimeout).
					WithTargets("www.example.com"),
			},
			wantEndpoints: map[string]int{
				"http://localhost:8080/v1/metrics/health":     http.StatusOK,
				"http://localhost:8080/v1/metrics/latency":    http.StatusOK,
				"http://localhost:8080/v1/metrics/dns":        http.StatusOK,
				"http://localhost:8080/v1/metrics/traceroute": http.StatusNotFound,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := framework.New(t)

			e2e := f.E2E(tt.startup.Config(t))
			for _, check := range tt.checks {
				e2e = e2e.WithCheck(check)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
			defer cancel()

			finish := make(chan error, 1)
			go func() {
				finish <- e2e.Run(ctx)
			}()

			// Wait for sparrow to be ready with a readiness probe.
			readinessProbe(t, "http://localhost:8080", checkTimeout)

			// Wait for the checks to be executed.
			wait := 5 * time.Second
			if len(tt.checks) > 0 {
				wait = checkInterval + checkTimeout + 5*time.Second
			}
			t.Logf("Waiting %s for checks to be executed", wait.String())
			<-time.After(wait)

			// Fetch, parse and create a new router from the OpenAPI schema, to be able to validate the responses.
			schema, err := fetchOpenAPISchema("http://localhost:8080/openapi")
			if err != nil {
				t.Fatalf("Failed to fetch OpenAPI schema: %v", err)
			}
			router, err := gorillamux.NewRouter(schema)
			if err != nil {
				t.Fatalf("Failed to create router from OpenAPI schema: %v", err)
			}

			for url, status := range tt.wantEndpoints {
				validateResponse(t, router, url, status)
			}

			cancel()
			<-finish
		})
	}
}

func fetchOpenAPISchema(url string) (*openapi3.T, error) {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to GET OpenAPI schema: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read OpenAPI schema: %w", err)
	}

	loader := openapi3.NewLoader()
	schema, err := loader.LoadFromData(data)
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI schema: %w", err)
	}

	if err = schema.Validate(ctx); err != nil {
		return nil, fmt.Errorf("OpenAPI schema validation error: %w", err)
	}

	return schema, nil
}

func validateResponse(t *testing.T, router routers.Router, url string, wantStatus int) {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, http.NoBody)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Errorf("Failed to get %s: %v", url, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != wantStatus {
		t.Errorf("Want status code %d for %s, got %d", wantStatus, url, resp.StatusCode)
		return
	}

	if wantStatus == http.StatusOK {
		if err = validateResponseSchema(router, req, resp); err != nil {
			t.Errorf("Response from %q does not match schema: %v", url, err)
			return
		}
	}

	t.Logf("Got status code %d for %s", resp.StatusCode, url)
}

func validateResponseSchema(router routers.Router, req *http.Request, resp *http.Response) error {
	route, _, err := router.FindRoute(req)
	if err != nil {
		return fmt.Errorf("failed to find route: %w", err)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	// Reset the response body for potential further use
	resp.Body = io.NopCloser(bytes.NewBuffer(data))

	responseRef := route.Operation.Responses.Status(resp.StatusCode)
	if responseRef == nil || responseRef.Value == nil {
		return fmt.Errorf("no response defined in OpenAPI schema for status code %d", resp.StatusCode)
	}

	mediaType := responseRef.Value.Content.Get("application/json")
	if mediaType == nil {
		return errors.New("no media type defined in OpenAPI schema for Content-Type 'application/json'")
	}

	var body any
	if err = json.Unmarshal(data, &body); err != nil {
		return fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	// Validate the response body against the schema
	err = mediaType.Schema.Value.VisitJSON(body)
	if err != nil {
		return fmt.Errorf("response body does not match schema: %w", err)
	}

	return nil
}

func readinessProbe(t *testing.T, url string, timeout time.Duration) {
	t.Helper()
	const retryInterval = 100 * time.Millisecond
	deadline := time.Now().Add(timeout)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, http.NoBody)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
		return
	}

	for {
		resp, err := http.DefaultClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			t.Log("Sparrow is ready")
			resp.Body.Close()
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("Sparrow not ready [%s (%d)] after %v: %v", http.StatusText(resp.StatusCode), resp.StatusCode, timeout, err)
			return
		}
		<-time.After(retryInterval)
	}
}
