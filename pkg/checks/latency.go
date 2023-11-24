package checks

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptrace"
	"sync"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"golang.org/x/sync/errgroup"

	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/api"
)

var _ Check = (*Latency)(nil)

func NewLatencyCheck() Check {
	return &Latency{
		mu:   sync.Mutex{},
		cfg:  LatencyConfig{},
		c:    nil,
		done: make(chan bool),
	}

}

type Latency struct {
	cfg  LatencyConfig
	mu   sync.Mutex
	c    chan<- Result
	done chan bool
}

type LatencyConfig struct {
	Endpoints []string
	Interval  time.Duration
}

type LatencyResult struct {
	Code int
	DNS  Metric
	TLS  Metric
	Dial Metric
}

type Metric struct {
	Start time.Time
	End   time.Time
}

func (m Metric) Duration() time.Duration {
	return m.End.Sub(m.Start)
}

func WithLatency(ctx context.Context, l *LatencyResult) context.Context {
	return httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		DNSStart:          func(di httptrace.DNSStartInfo) { l.DNS.Start = time.Now() },
		DNSDone:           func(di httptrace.DNSDoneInfo) { l.DNS.End = time.Now() },
		TLSHandshakeStart: func() { l.TLS.Start = time.Now() },
		TLSHandshakeDone:  func(cs tls.ConnectionState, err error) { l.TLS.End = time.Now() },
		ConnectStart:      func(network, addr string) { l.Dial.Start = time.Now() },
		ConnectDone:       func(network, addr string, err error) { l.Dial.End = time.Now() },
	})
}

func (l *Latency) Run(ctx context.Context) error {
	log := logger.FromContext(ctx).WithGroup("Latency")
	for {
		select {
		case <-ctx.Done():
			log.Error("context canceled", "err", ctx.Err())
			return ctx.Err()
		case <-l.done:
			return nil
		case <-time.After(l.cfg.Interval):
			results, err := l.check(ctx)
			checkResult := Result{
				Data:      results,
				Err:       err.Error(),
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
	c, ok := config.(LatencyConfig)
	if !ok {
		return ErrInvalidConfig
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cfg = c

	return nil
}

func (l *Latency) Schema() (*openapi3.SchemaRef, error) {
	return OpenapiFromPerfData(make(map[string]LatencyResult))
}

func (l *Latency) RegisterHandler(ctx context.Context, router *api.RoutingTree) {
	router.Add(http.MethodGet, "/v1alpha1/latency", l.Handler)
}

func (l *Latency) DeregisterHandler(ctx context.Context, router *api.RoutingTree) {
	router.Remove(http.MethodGet, "/v1alpha1/latency")
}

func (l *Latency) Handler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (l *Latency) check(ctx context.Context) (map[string]LatencyResult, error) {
	cl := http.Client{}
	results := map[string]LatencyResult{}
	wg, ctx := errgroup.WithContext(ctx)
	// TODO mutex
	for _, e := range l.cfg.Endpoints {
		wg.Go(func(ctx context.Context, e string) func() error {
			return func() error {
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, e, nil)
				if err != nil {
					return err
				}
				var result LatencyResult
				ctx = WithLatency(req.Context(), &result)

				req = req.WithContext(ctx)

				response, err := cl.Do(req)
				if err != nil {
					return err
				}

				result.Code = response.StatusCode
				// This does not need a mutex since the map key we're writing to is not dynamic
				results[e] = result
				return nil
			}
		}(ctx, e))
	}

	if err := wg.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}
