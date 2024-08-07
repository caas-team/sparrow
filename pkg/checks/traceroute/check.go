package traceroute

import (
	"context"
	"sync"
	"time"

	"github.com/caas-team/sparrow/internal/helper"
	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/prometheus/client_golang/prometheus"
)

var _ checks.Check = (*Traceroute)(nil)

const CheckName = "traceroute"

type Target struct {
	// The address of the target to traceroute to. Can be a DNS name or an IP address
	Addr string `json:"addr" yaml:"addr" mapstructure:"addr"`
	// The port to traceroute to
	Port int `json:"port" yaml:"port" mapstructure:"port"`
}

func NewCheck() checks.Check {
	return &Traceroute{
		CheckBase: checks.CheckBase{
			Mu:       sync.Mutex{},
			DoneChan: make(chan struct{}, 1),
		},
		config:     Config{},
		traceroute: TraceRoute,
		metrics:    newMetrics(),
	}
}

type Traceroute struct {
	checks.CheckBase
	config     Config
	traceroute tracerouteFactory
	metrics    metrics
}

type tracerouteConfig struct {
	Dest    string
	Port    int
	Timeout time.Duration
	MaxHops int
	Rc      helper.RetryConfig
}
type tracerouteFactory func(ctx context.Context, cfg tracerouteConfig) (map[int][]Hop, error)

type result struct {
	// The minimum number of hops required to reach the target
	MinHops int `json:"min_hops" yaml:"min_hops" mapstructure:"min_hops"`
	// The path taken to the destination
	Hops map[int][]Hop `json:"hops" yaml:"hops" mapstructure:"hops"`
}

// Run runs the check in a loop sending results to the provided channel
func (tr *Traceroute) Run(ctx context.Context, cResult chan checks.ResultDTO) error {
	ctx, cancel := logger.NewContextWithLogger(ctx)
	defer cancel()
	log := logger.FromContext(ctx)

	log.Info("Starting traceroute check", "interval", tr.config.Interval.String())
	for {
		select {
		case <-ctx.Done():
			log.Error("Context canceled", "error", ctx.Err())
			return ctx.Err()
		case <-tr.DoneChan:
			return nil
		case <-time.After(tr.config.Interval):
			res := tr.check(ctx)
			tr.metrics.MinHops(res)
			cResult <- checks.ResultDTO{
				Name: tr.Name(),
				Result: &checks.Result{
					Data:      res,
					Timestamp: time.Now(),
				},
			}
			log.Debug("Successfully finished traceroute check run")
		}
	}
}

// GetConfig returns the current configuration of the check
func (tr *Traceroute) GetConfig() checks.Runtime {
	tr.Mu.Lock()
	defer tr.Mu.Unlock()
	return &tr.config
}

func (tr *Traceroute) check(ctx context.Context) map[string]result {
	res := make(map[string]result)
	log := logger.FromContext(ctx)

	type internalResult struct {
		addr string
		res  result
	}

	cResult := make(chan internalResult, len(tr.config.Targets))
	var wg sync.WaitGroup

	start := time.Now()

	wg.Add(len(tr.config.Targets))
	for _, t := range tr.config.Targets {
		go func(t Target) {
			defer wg.Done()
			l := log.With("target", t.Addr)
			l.Debug("Running traceroute")

			targetstart := time.Now()
			trace, err := tr.traceroute(ctx, tracerouteConfig{
				Dest:    t.Addr,
				Port:    t.Port,
				Timeout: tr.config.Timeout,
				MaxHops: tr.config.MaxHops,
				Rc:      tr.config.Retry,
			})
			elapsed := time.Since(targetstart)
			if err != nil {
				l.Error("Error running traceroute", "error", err)
			}

			tr.metrics.CheckDuration(t.Addr, elapsed)

			l.Debug("Ran traceroute", "result", trace, "duration", elapsed)
			res := result{
				Hops:    trace,
				MinHops: tr.config.MaxHops,
			}

			for ttl, hop := range trace {
				for _, attempt := range hop {
					if attempt.Reached && attempt.Ttl < res.MinHops {
						res.MinHops = ttl
					}
				}
			}

			cResult <- internalResult{addr: t.Addr, res: res}
		}(t)
	}

	wg.Wait()
	close(cResult)

	for r := range cResult {
		res[r.addr] = r.res
	}

	elapsed := time.Since(start)

	log.Info("Finished traceroute check", "duration", elapsed)

	return res
}

// Shutdown is called once when the check is unregistered or sparrow shuts down
func (tr *Traceroute) Shutdown() {
	tr.DoneChan <- struct{}{}
	close(tr.DoneChan)
}

// SetConfig is called once when the check is registered
// This is also called while the check is running, if the remote config is updated
// This should return an error if the config is invalid
func (tr *Traceroute) SetConfig(cfg checks.Runtime) error {
	if cfg, ok := cfg.(*Config); ok {
		tr.Mu.Lock()
		defer tr.Mu.Unlock()
		tr.config = *cfg
		return nil
	}

	return checks.ErrConfigMismatch{
		Expected: CheckName,
		Current:  cfg.For(),
	}
}

// Schema returns an openapi3.SchemaRef of the result type returned by the check
func (tr *Traceroute) Schema() (*openapi3.SchemaRef, error) {
	return checks.OpenapiFromPerfData(map[string]result{})
}

// GetMetricCollectors allows the check to provide prometheus metric collectors
func (tr *Traceroute) GetMetricCollectors() []prometheus.Collector {
	return tr.metrics.List()
}

// Name returns the name of the check
func (tr *Traceroute) Name() string {
	return CheckName
}
