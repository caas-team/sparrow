package checks

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/mitchellh/mapstructure"

	"github.com/caas-team/sparrow/internal/helper"
	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/api"
)

var _ Check = (*Latency)(nil)

func NewLatencyCheck() Check {
	return &Latency{
		mu:     sync.Mutex{},
		cfg:    LatencyConfig{},
		c:      nil,
		done:   make(chan bool, 1),
		client: &http.Client{},
	}
}

type Latency struct {
	cfg    LatencyConfig
	mu     sync.Mutex
	c      chan<- Result
	done   chan bool
	client *http.Client
}

type LatencyConfig struct {
	Targets  []string
	Interval time.Duration
	Timeout  time.Duration
	Retry    helper.RetryConfig
}

type LatencyResult struct {
	Code  int     `json:"code"`
	Error *string `json:"error"`
	Total int64   `json:"total"`
}

func (l *Latency) Run(ctx context.Context) error {
	ctx, cancel := logger.NewContextWithLogger(ctx, "latency")
	defer cancel()
	log := logger.FromContext(ctx)
	log.Info(fmt.Sprintf("Using latency check interval of %s", l.cfg.Interval.String()))

	for {
		select {
		case <-ctx.Done():
			log.Error("Context canceled", "err", ctx.Err())
			return ctx.Err()
		case <-l.done:
			return nil
		case <-time.After(l.cfg.Interval):
			results := l.check(ctx)
			errval := ""
			checkResult := Result{
				Data:      results,
				Err:       errval,
				Timestamp: time.Now(),
			}

			l.c <- checkResult
		}
	}
}

func (l *Latency) Startup(ctx context.Context, cResult chan<- Result) error {
	log := logger.FromContext(ctx).WithGroup("latency")
	log.Debug("Starting latency check")

	l.c = cResult
	return nil
}

func (l *Latency) Shutdown(_ context.Context) error {
	l.done <- true
	close(l.done)

	return nil
}

func (l *Latency) SetConfig(_ context.Context, config any) error {
	var c LatencyConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		return ErrInvalidConfig
	}
	c.Interval = time.Second * c.Interval
	c.Retry.Delay = time.Second * c.Retry.Delay
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cfg = c

	return nil
}

// SetClient sets the http client for the latency check
func (l *Latency) SetClient(c *http.Client) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.client = c
}

func (l *Latency) Schema() (*openapi3.SchemaRef, error) {
	return OpenapiFromPerfData(make(map[string]LatencyResult))
}

func (l *Latency) RegisterHandler(_ context.Context, router *api.RoutingTree) {
	router.Add(http.MethodGet, "v1alpha1/latency", l.Handler)
}

func (l *Latency) DeregisterHandler(_ context.Context, router *api.RoutingTree) {
	router.Remove(http.MethodGet, "v1alpha1/latency")
}

func (l *Latency) Handler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (l *Latency) check(ctx context.Context) map[string]LatencyResult {
	log := logger.FromContext(ctx).WithGroup("check")
	log.Debug("Checking latency")
	if len(l.cfg.Targets) == 0 {
		log.Debug("No targets defined")
		return map[string]LatencyResult{}
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	results := map[string]LatencyResult{}

	l.mu.Lock()
	l.client.Timeout = l.cfg.Timeout * time.Second
	l.mu.Unlock()
	for _, tar := range l.cfg.Targets {
		target := tar
		wg.Add(1)
		go func(target string) {
			defer wg.Done()
			lo := log.With("target", target)
			lo.Debug("Starting retry routine to get latency", "target", target)

			err := helper.Retry(func(ctx context.Context) error {
				lo.Debug("Getting latency", "timing out in", l.client.Timeout.String())
				res := getLatency(ctx, l.client, target)
				mu.Lock()
				defer mu.Unlock()
				results[target] = res
				return nil
			}, l.cfg.Retry)(ctx)
			if err != nil {
				lo.Error("Error while checking latency", "error", err)
			}
		}(target)
	}
	wg.Wait()

	return results
}

// getLatency performs an HTTP get request and returns ok if request succeeds
func getLatency(ctx context.Context, client *http.Client, url string) LatencyResult {
	log := logger.FromContext(ctx).With("url", url)
	var res LatencyResult

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		log.Error("Error while creating request", "error", err)
		errval := err.Error()
		res.Error = &errval
		return res
	}

	start := time.Now()
	resp, err := client.Do(req) //nolint:bodyclose // Closed in defer below
	if err != nil {
		log.Error("Error while checking latency", "error", err)
		errval := err.Error()
		res.Error = &errval
		return res
	}

	res.Code = resp.StatusCode
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	end := time.Now()

	res.Total = end.Sub(start).Milliseconds()
	return res
}
