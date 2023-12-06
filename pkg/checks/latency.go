package checks

import (
	"context"
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
	log := logger.FromContext(ctx).WithGroup("Latency")
	log.Info(l.cfg.Interval.String())
	for {
		select {
		case <-ctx.Done():
			log.Error("context canceled", "err", ctx.Err())
			return ctx.Err()
		case <-l.done:
			return nil
		case <-time.After(l.cfg.Interval):
			results, err := l.check(ctx)
			errval := ""
			if err != nil {
				errval = err.Error()
			}
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

func (l *Latency) Shutdown(ctx context.Context) error {
	l.done <- true
	close(l.done)

	return nil
}

func (l *Latency) SetConfig(ctx context.Context, config any) error {
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

func (l *Latency) RegisterHandler(ctx context.Context, router *api.RoutingTree) {
	router.Add(http.MethodGet, "v1alpha1/latency", l.Handler)
}

func (l *Latency) DeregisterHandler(ctx context.Context, router *api.RoutingTree) {
	router.Remove(http.MethodGet, "v1alpha1/latency")
}

func (l *Latency) Handler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (l *Latency) check(ctx context.Context) (map[string]LatencyResult, error) {
	log := logger.FromContext(ctx).WithGroup("check")
	log.Debug("Checking latency")
	if len(l.cfg.Targets) == 0 {
		log.Debug("No targets defined")
		return map[string]LatencyResult{}, nil
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	results := map[string]LatencyResult{}

	for _, tar := range l.cfg.Targets {
		target := tar
		wg.Add(1)
		go func(target string) {
			defer wg.Done()
			log := log.With("target", target)
			log.Debug("Starting retry routine to get latency", "target", target)

			err := helper.Retry(func(ctx context.Context) error {
				mu.Lock()
				defer mu.Unlock()
				res := getLatency(ctx, l.client, target)
				results[target] = res
				return nil
			}, l.cfg.Retry)(ctx)
			if err != nil {
				log.Error("Error while checking latency", "error", err)
			}
		}(target)
	}
	wg.Wait()

	log.Info("Successfully got latency to all targets")
	return results, nil
}

// getLatency performs an HTTP get request and returns ok if request succeeds
func getLatency(ctx context.Context, client *http.Client, url string) LatencyResult {
	log := logger.FromContext(ctx).With("url", url)
	var res LatencyResult

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		log.Error("Error while creating request", "error", err)
		errval := err.Error()
		res.Error = &errval
		return res
	}

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		log.Error("Error while checking latency", "error", err)
		errval := err.Error()
		res.Error = &errval
	} else {
		res.Code = resp.StatusCode
	}
	end := time.Now()

	res.Total = end.Sub(start).Milliseconds()

	return res
}
