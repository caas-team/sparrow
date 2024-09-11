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

var (
	_ checks.Check   = (*Traceroute)(nil)
	_ checks.Runtime = (*Config)(nil)
)

const CheckName = "traceroute"

type Target struct {
	// The address of the target to traceroute to. Can be a DNS name or an IP address
	Addr string `json:"addr" yaml:"addr" mapstructure:"addr"`
	// The port to traceroute to
	Port int `json:"port" yaml:"port" mapstructure:"port"`
}

func NewCheck() checks.Check {
	return &Traceroute{
		Base:       checks.NewBase(CheckName, &Config{}),
		traceroute: TraceRoute,
		metrics:    newMetrics(),
	}
}

type Traceroute struct {
	checks.Base[*Config]
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
	return tr.StartCheck(ctx, cResult, tr.Config.Interval, func(ctx context.Context) any {
		res := tr.check(ctx)
		tr.metrics.MinHops(res)
		return res
	})
}

func (tr *Traceroute) check(ctx context.Context) map[string]result {
	res := make(map[string]result)
	log := logger.FromContext(ctx)

	type internalResult struct {
		addr string
		res  result
	}

	var wg sync.WaitGroup
	cResult := make(chan internalResult, len(tr.Config.Targets))

	start := time.Now()
	wg.Add(len(tr.Config.Targets))
	for _, t := range tr.Config.Targets {
		go func(t Target) {
			defer wg.Done()
			l := log.With("target", t.Addr)
			l.Debug("Running traceroute")

			targetstart := time.Now()
			trace, err := tr.traceroute(ctx, tracerouteConfig{
				Dest:    t.Addr,
				Port:    t.Port,
				Timeout: tr.Config.Timeout,
				MaxHops: tr.Config.MaxHops,
				Rc:      tr.Config.Retry,
			})
			elapsed := time.Since(targetstart)
			if err != nil {
				l.Error("Error running traceroute", "error", err)
			}

			tr.metrics.CheckDuration(t.Addr, elapsed)

			l.Debug("Ran traceroute", "result", trace, "duration", elapsed)
			res := result{
				Hops:    trace,
				MinHops: tr.Config.MaxHops,
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

// Schema returns an openapi3.SchemaRef of the result type returned by the check
func (tr *Traceroute) Schema() (*openapi3.SchemaRef, error) {
	return checks.OpenapiFromPerfData(map[string]result{})
}

// GetMetricCollectors allows the check to provide prometheus metric collectors
func (tr *Traceroute) GetMetricCollectors() []prometheus.Collector {
	return tr.metrics.List()
}
