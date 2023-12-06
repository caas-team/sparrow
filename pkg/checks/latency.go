package checks

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/sync/errgroup"

	"github.com/caas-team/sparrow/internal/helper"
	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/api"
)

var _ Check = (*Latency)(nil)

func NewLatencyCheck() Check {
	return &Latency{
		mu:   sync.Mutex{},
		cfg:  LatencyConfig{},
		c:    nil,
		done: make(chan bool, 1),
	}
}

type Latency struct {
	cfg  LatencyConfig
	mu   sync.Mutex
	c    chan<- Result
	done chan bool
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

func (l *Latency) check(ctx context.Context) (map[string]LatencyResult, error) {
	log := logger.FromContext(ctx).WithGroup("check")
	log.Debug("Checking latency")

	var resultMutex sync.Mutex
	results := map[string]LatencyResult{}

	wg, ctx := errgroup.WithContext(ctx)
	for _, e := range l.cfg.Targets {
		wg.Go(func(ctx context.Context, e string) func() error {
			return func() error {
				cl := http.Client{
					Timeout: l.cfg.Timeout * time.Second,
				}
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, e, http.NoBody)
				if err != nil {
					log.Error("Error while creating request", "error", err)
					return err
				}

				var result LatencyResult

				req = req.WithContext(ctx)

				helper.Retry(func(ctx context.Context) error {
					start := time.Now()
					response, err := cl.Do(req) //nolint:bodyclose // Tom has refactored this function
					if err != nil {
						errval := err.Error()
						result.Error = &errval
						log.Error("Error while checking latency", "error", err)
					} else {
						result.Code = response.StatusCode
					}
					end := time.Now()

					result.Total = end.Sub(start).Milliseconds()

					resultMutex.Lock()
					defer resultMutex.Unlock()
					results[e] = result

					return err
				}, l.cfg.Retry)(ctx) //nolint: errcheck //ignore return value, since we set it in the closure
				return nil
			}
		}(ctx, e))
	}

	if err := wg.Wait(); err != nil {
		log.Error("Error while checking latency", "error", err)
		return nil, err
	}

	return results, nil
}
