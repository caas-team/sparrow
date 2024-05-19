package traceroute

import (
	"context"
	"sync"

	"github.com/caas-team/sparrow/internal/helper"
	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/internal/traceroute"
	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	_ checks.Check   = (*Traceroute)(nil)
	_ checks.Runtime = (*Config)(nil)
)

const CheckName = "traceroute"

// Traceroute is a check that performs a TCP traceroute to a list of targets
type Traceroute struct {
	checks.Base[*Config]
	tracer  traceroute.Tracer
	metrics metrics
}

// NewCheck creates a new traceroute check
func NewCheck() checks.Check {
	return &Traceroute{
		Base:    checks.NewBase(CheckName, &Config{}),
		metrics: newMetrics(),
	}
}

// result represents the result of a single hop in the traceroute
type result struct {
	// Target represents the target address
	Target string
	// Hops represents the hops to the target
	Hops []traceroute.Hop
}

// Run runs the check in a loop sending results to the provided channel
func (tr *Traceroute) Run(ctx context.Context, cResult chan checks.ResultDTO) error {
	return tr.StartCheck(ctx, cResult, tr.Config.Interval, func(ctx context.Context) any {
		return tr.check(ctx)
	})
}

// SetConfig sets the configuration of the traceroute check and initializes the tracer
func (tr *Traceroute) SetConfig(config checks.Runtime) error {
	err := tr.Base.SetConfig(config)
	if err != nil {
		return err
	}
	tr.tracer = traceroute.New(tr.Config.MaxHops, tr.Config.Timeout, traceroute.ICMP)
	return nil
}

// Schema returns an openapi3.SchemaRef of the result type returned by the check
func (tr *Traceroute) Schema() (*openapi3.SchemaRef, error) {
	return checks.OpenapiFromPerfData(map[string]result{})
}

// GetMetricCollectors allows the check to provide prometheus metric collectors
func (tr *Traceroute) GetMetricCollectors() []prometheus.Collector {
	return []prometheus.Collector{tr.metrics}
}

// check performs a TCP traceroute for all configured targets
func (tr *Traceroute) check(ctx context.Context) map[string]result {
	log := logger.FromContext(ctx)
	log.DebugContext(ctx, "Starting traceroute check")
	if len(tr.Config.Targets) == 0 {
		log.DebugContext(ctx, "No targets defined")
		return map[string]result{}
	}

	var wg sync.WaitGroup
	results := map[string]result{}
	var mu sync.Mutex

	log.DebugContext(ctx, "Tracerouting to each target in separate routine", "amount", len(tr.Config.Targets))
	for _, t := range tr.Config.Targets {
		target := t
		wg.Add(1)
		lo := log.With("target", target)

		retryExecutor := helper.Retry(func(ctx context.Context) error {
			hops, err := tr.tracer.Run(ctx, target)
			mu.Lock()
			defer mu.Unlock()
			results[target] = result{
				Target: target,
				Hops:   hops,
			}
			if err != nil {
				return err
			}
			return nil
		}, tr.Config.Retry)

		go func() {
			defer wg.Done()

			lo.Debug("Starting retry routine to traceroute to target")
			if err := retryExecutor(ctx); err != nil {
				lo.Error("Error while tracerouting", "error", err)
			}

			lo.DebugContext(ctx, "Finished traceroute to target")
			mu.Lock()
			defer mu.Unlock()
			tr.metrics.Set(target, results[target].Hops)
		}()
	}

	log.DebugContext(ctx, "Waiting for all traceroutes to finish")
	wg.Wait()

	log.DebugContext(ctx, "Successfully finished traceroute check")
	return results
}
