package framework

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/config"
	"github.com/caas-team/sparrow/pkg/sparrow"
	"github.com/caas-team/sparrow/test/framework/builder"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

var _ Runner = (*E2E)(nil)

// E2E is an end-to-end test.
type E2E struct {
	config  config.Config
	buf     bytes.Buffer
	sparrow *sparrow.Sparrow
	t       *testing.T
	checks  map[string]builder.Check
	server  *http.Server
	mu      sync.Mutex
	running bool
}

// WithChecks sets the checks in the test.
func (t *E2E) WithChecks(builders ...builder.Check) *E2E {
	for _, b := range builders {
		t.checks[b.For()] = b
		t.buf.Write(b.YAML(t.t))
	}
	return t
}

// WithRemote sets up a remote server to serve the check config.
func (t *E2E) WithRemote() *E2E {
	t.server = &http.Server{
		Addr:              "localhost:50505",
		Handler:           http.HandlerFunc(t.serveConfig),
		ReadHeaderTimeout: 3 * time.Second,
	}
	return t
}

// UpdateChecks updates the checks of the test.
func (t *E2E) UpdateChecks(builders ...builder.Check) *E2E {
	t.checks = map[string]builder.Check{}
	t.buf.Reset()
	for _, b := range builders {
		t.checks[b.For()] = b
		t.buf.Write(b.YAML(t.t))
	}

	// If the test is running with a remote server, we don't need to write the check config into a file.
	if t.server == nil {
		err := t.writeCheckConfig()
		if err != nil {
			t.t.Fatalf("Failed to write check config: %v", err)
		}
	}

	return t
}

// Run runs the test.
// Runs indefinitely until the context is canceled.
func (t *E2E) Run(ctx context.Context) error {
	if t.isRunning() {
		t.t.Fatal("E2E.Run must be called once")
	}

	if t.server != nil {
		go func() {
			err := t.server.ListenAndServe()
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				t.t.Errorf("Failed to start server: %v", err)
			}
		}()
		defer func() {
			err := t.server.Shutdown(ctx)
			if err != nil {
				t.t.Fatalf("Failed to shutdown server: %v", err)
			}
		}()
	} else {
		err := t.writeCheckConfig()
		if err != nil {
			t.t.Fatalf("Failed to write check config: %v", err)
		}
	}

	t.mu.Lock()
	t.running = true
	t.mu.Unlock()
	return t.sparrow.Run(ctx)
}

// AwaitStartup waits for the provided URL to be ready.
//
// Must be called after the e2e test started with [E2E.Run].
func (t *E2E) AwaitStartup(u string, failureTimeout time.Duration) *E2E {
	t.t.Helper()
	// To ensure the goroutine is started before we are checking if the test is running.
	const initialDelay = 100 * time.Millisecond
	<-time.After(initialDelay)
	if !t.isRunning() {
		t.t.Fatal("E2E.AwaitStartup must be called after E2E.Run")
	}

	const retryInterval = 100 * time.Millisecond
	start := time.Now()
	deadline := start.Add(failureTimeout)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, u, http.NoBody)
	if err != nil {
		t.t.Fatalf("Failed to create request: %v", err)
		return t
	}

	for {
		resp, err := http.DefaultClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			t.t.Logf("%s is ready after %v", u, time.Since(start))
			resp.Body.Close()
			return t
		}
		if time.Now().After(deadline) {
			t.t.Errorf("%s is not ready [%s (%d)] after %v: %v", u, http.StatusText(resp.StatusCode), resp.StatusCode, failureTimeout, err)
			return t
		}
		<-time.After(retryInterval)
	}
}

// AwaitLoader waits for the loader to reload the configuration.
//
// Must be called after the e2e test started with [E2E.Run].
func (t *E2E) AwaitLoader() *E2E {
	t.t.Helper()
	if !t.isRunning() {
		t.t.Fatal("E2E.AwaitLoader must be called after E2E.Run")
	}

	t.t.Logf("Waiting %s for loader to reload configuration", t.config.Loader.Interval.String())
	<-time.After(t.config.Loader.Interval)
	return t
}

// AwaitChecks waits for all checks to be executed before proceeding.
//
// Must be called after the e2e test started with [E2E.Run].
func (t *E2E) AwaitChecks() *E2E {
	t.t.Helper()
	if !t.isRunning() {
		t.t.Fatal("E2E.AwaitChecks must be called after E2E.Run")
	}

	wait := 5 * time.Second
	for _, check := range t.checks {
		wait = max(wait, check.ExpectedWaitTime())
	}
	t.t.Logf("Waiting %s for checks to be executed", wait.String())
	<-time.After(wait)
	return t
}

// writeCheckConfig writes the check config to a file at the provided path.
func (t *E2E) writeCheckConfig() error {
	const fileMode = 0o755
	path := "testdata/checks.yaml"
	err := os.MkdirAll(filepath.Dir(path), fileMode)
	if err != nil {
		return fmt.Errorf("failed to create %q: %w", filepath.Dir(path), err)
	}

	err = os.WriteFile(path, t.buf.Bytes(), fileMode)
	if err != nil {
		return fmt.Errorf("failed to write %q: %w", path, err)
	}
	return nil
}

// isRunning returns true if the test is running.
func (t *E2E) isRunning() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.running
}

// serveConfig serves the check config over HTTP as text/yaml.
func (t *E2E) serveConfig(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/yaml")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write(t.buf.Bytes())
	if err != nil {
		t.t.Fatalf("Failed to write response: %v", err)
	}
}

// e2eHttpAsserter is an HTTP asserter for end-to-end tests.
type e2eHttpAsserter struct {
	e2e      *E2E
	url      string
	response *e2eResponseAsserter
	schema   *openapi3.T
	router   routers.Router
}

// e2eResponseAsserter is a response asserter for end-to-end tests.
type e2eResponseAsserter struct {
	want     any
	asserter func(r *http.Response) error
}

// HttpAssertion creates a new HTTP assertion for the given URL.
func (t *E2E) HttpAssertion(u string) *e2eHttpAsserter {
	return &e2eHttpAsserter{e2e: t, url: u}
}

// Assert asserts the status code and optional validations against the response.
// Optional validations must be set before calling this method.
//
// Must be called after the e2e test started with [E2E.Run].
func (a *e2eHttpAsserter) Assert(status int) {
	a.e2e.t.Helper()
	if !a.e2e.isRunning() {
		a.e2e.t.Fatal("e2eHttpAsserter.Assert must be called after E2E.Run")
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, a.url, http.NoBody)
	if err != nil {
		a.e2e.t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		a.e2e.t.Errorf("Failed to get %s: %v", a.url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != status {
		a.e2e.t.Errorf("Want status code %d for %s, got %d", status, a.url, resp.StatusCode)
	}
	a.e2e.t.Logf("Got status code %d for %s", resp.StatusCode, a.url)

	if status == http.StatusOK {
		if a.schema != nil && a.router != nil {
			if err = a.assertSchema(req, resp); err != nil {
				a.e2e.t.Errorf("Response from %q does not match schema: %v", a.url, err)
			}
		}

		if a.response != nil {
			err = a.response.asserter(resp)
			if err != nil {
				a.e2e.t.Errorf("Failed to assert response: %v", err)
			}
		}
	}
}

// WithSchema fetches the OpenAPI schema and validates the response against it.
func (a *e2eHttpAsserter) WithSchema() *e2eHttpAsserter {
	a.e2e.t.Helper()
	schema, err := a.fetchSchema()
	if err != nil {
		a.e2e.t.Fatalf("Failed to fetch OpenAPI schema: %v", err)
	}

	router, err := gorillamux.NewRouter(schema)
	if err != nil {
		a.e2e.t.Fatalf("Failed to create router from OpenAPI schema: %v", err)
	}

	a.schema = schema
	a.router = router
	return a
}

// WithResult sets the expected result for the response.
// The result is validated against the response body.
func (a *e2eHttpAsserter) WithCheckResult(r checks.Result) *e2eHttpAsserter {
	a.e2e.t.Helper()
	a.response = &e2eResponseAsserter{
		want:     r,
		asserter: a.assertCheckResponse,
	}
	return a
}

// fetchSchema fetches the OpenAPI schema from the server.
func (a *e2eHttpAsserter) fetchSchema() (*openapi3.T, error) {
	ctx := context.Background()
	u, err := url.Parse(a.url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}
	u.Path = "/openapi"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), http.NoBody)
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

// assertSchema asserts the response body against the OpenAPI schema.
func (a *e2eHttpAsserter) assertSchema(req *http.Request, resp *http.Response) error {
	route, _, err := a.router.FindRoute(req)
	if err != nil {
		return fmt.Errorf("failed to find route: %w", err)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(data))

	responseRef := route.Operation.Responses.Status(resp.StatusCode)
	if responseRef == nil || responseRef.Value == nil {
		return fmt.Errorf("no response defined in OpenAPI schema for status code %d", resp.StatusCode)
	}

	mediaType := responseRef.Value.Content.Get("application/json")
	if mediaType == nil {
		return errors.New("no media type defined in OpenAPI schema for Content-Type 'application/json'")
	}

	var body map[string]any
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

// assertCheckResponse asserts the response body against the expected check result.
func (a *e2eHttpAsserter) assertCheckResponse(resp *http.Response) error {
	want, ok := a.response.want.(checks.Result)
	if !ok {
		a.e2e.t.Fatalf("Invalid response type: %T", a.response.want)
	}

	var got checks.Result
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		a.e2e.t.Errorf("Failed to decode response body: %v", err)
	}

	wantData := want.Data.(map[string]any)
	gotData, ok := got.Data.(map[string]any)
	if !ok {
		a.e2e.t.Errorf("Result.Data = %T (%v), want %T (%v)", got.Data, got.Data, want.Data, want.Data)
	}
	assertMapEqual(a.e2e.t, wantData, gotData)

	const deltaTimeThreshold = 5 * time.Minute
	if time.Since(got.Timestamp) > deltaTimeThreshold {
		a.e2e.t.Errorf("Response timestamp is not recent: %v", got.Timestamp)
	}

	return nil
}

// assertMapEqual asserts the equality of the want and got maps.
// Fails the test if the maps are not equal.
func assertMapEqual(t *testing.T, want, got map[string]any) {
	t.Helper()
	if len(want) != len(got) {
		t.Errorf("Want %d keys (%v), got %d keys (%v)", len(want), want, len(got), got)
	}

	for k, w := range want {
		g, ok := got[k]
		if !ok {
			t.Errorf("got[%q] not found (%v), want %v", k, got, w)
		}

		if err := assertValueEqual(t, w, g); err != nil {
			t.Errorf("got[%q]: %v", k, err)
		}
	}
}

// assertValueEqual asserts the equality of the want and got values.
// For values that cannot be compared directly, it uses a type-specific comparison.
// e.g. IP addresses, timestamps, etc.
func assertValueEqual(t *testing.T, want, got any) error {
	switch w := want.(type) {
	case map[string]any:
		gotMap, ok := got.(map[string]any)
		if !ok {
			return fmt.Errorf("%v (%T), want %v (%T)", got, got, w, w)
		}
		assertMapEqual(t, w, gotMap)
		return nil
	case time.Time, float32, float64:
		// Timestamps and floating-point numbers are time-sensitive and are never equal.
		return nil
	case int:
		// Unmarshaling JSON numbers as int will convert them to float64.
		// We need to compare them as float64 to avoid type mismatch errors.
		want = float64(w)
	case []string:
		// Unmarshaling JSON arrays as []string will convert them to []interface{}.
		// We need to compare them as []interface{} and cast the elements to string
		// to avoid type mismatch errors.
		gs, ok := got.([]any)
		if !ok {
			return fmt.Errorf("%v (%T), want %v (%T)", got, got, w, w)
		}
		gotSlice := make([]string, len(gs))
		for i, g := range gs {
			gotSlice[i] = g.(string)
		}
		for _, wantIPStr := range w {
			wantIP := net.ParseIP(wantIPStr)
			if wantIP == nil {
				// This is a special case for string slices that might contain IP addresses.
				// If the `want` value is not a valid IP address, we skip the IP validation
				// and proceed to the default case for a generic equality check.
				//
				// Using `goto` here avoids introducing an additional boolean flag or
				// nesting the logic further, which would make the code harder to read.
				// In this case it simplifies the control flow by explicitly directing the
				// execution to the default case.
				goto defaultCase
			}

			for _, gotIPStr := range gotSlice {
				gotIP := net.ParseIP(gotIPStr)
				if gotIP == nil {
					return fmt.Errorf("%q, want an IP address (%s)", gotIPStr, wantIP)
				}
			}
		}
		return nil
	}

defaultCase:
	if !reflect.DeepEqual(want, got) {
		return fmt.Errorf("%v (%T), want %v (%T)", got, got, want, want)
	}
	return nil
}
