package traceroute

import (
	"context"
	"sync"
	"time"

	"github.com/aeden/traceroute"
	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/prometheus/client_golang/prometheus"
)

var _ checks.Check = (*Traceroute)(nil)

const CheckName = "traceroute"

type Target struct {
	Addr string `json:"addr" yaml:"addr" mapstructure:"addr"`
	Port uint16 `json:"port" yaml:"port" mapstructure:"port"`
}

// Config is the configuration for the traceroute check
func NewCheck() checks.Check {
	return &Traceroute{
		config:     Config{},
		traceroute: newTraceroute,
		CheckBase: checks.CheckBase{
			Mu:       sync.Mutex{},
			DoneChan: make(chan struct{}),
		},
	}
}

type Traceroute struct {
	checks.CheckBase
	config     Config
	traceroute tracerouteFactory
}

type tracerouteFactory func(dest string, port, timeout, retries, maxHops int) (traceroute.TracerouteResult, error)

func newTraceroute(dest string, port, timeout, retries, maxHops int) (traceroute.TracerouteResult, error) {
	opts := &traceroute.TracerouteOptions{}
	opts.SetTimeoutMs(timeout)
	opts.SetRetries(retries)
	opts.SetMaxHops(maxHops)
	opts.SetPort(port)
	return traceroute.Traceroute(dest, opts)
}

type result struct {
	// The minimum number of hops required to reach the target
	NumHops int
	// The path taken to the destination
	Hops []hop
}

type hop struct {
	Addr    string
	Latency time.Duration
	Success bool
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
			log.Error("Context canceled", "err", ctx.Err())
			return ctx.Err()
		case <-tr.DoneChan:
			return nil
		case <-time.After(tr.config.Interval):
			res := tr.check(ctx)

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

	var wg sync.WaitGroup
	cResult := make(chan internalResult, len(tr.config.Targets))

	for _, t := range tr.config.Targets {
		wg.Add(1)
		go func(t Target) {
			defer wg.Done()
			log.Debug("Running traceroute", "target", t.Addr)
			start := time.Now()
			tr, trerr := tr.traceroute(t.Addr, int(t.Port), int(tr.config.Timeout/time.Millisecond), tr.config.Retries, tr.config.MaxHops)
			duration := time.Since(start)
			if trerr != nil {
				log.Error("Error running traceroute", "err", trerr, "target", t.Addr)
			}

			log.Debug("Ran traceroute", "result", tr, "duration", duration)

			r := result{
				NumHops: len(tr.Hops),
				Hops:    []hop{},
			}

			for _, h := range tr.Hops {
				r.Hops = append(r.Hops, hop{
					Addr:    h.Host,
					Latency: h.ElapsedTime,
					Success: h.Success,
				})
			}
			cResult <- internalResult{addr: t.Addr, res: r}
		}(t)
	}

	log.Debug("Waiting for traceroute checks to finish")

	go func() {
		wg.Wait()
		close(cResult)
	}()

	log.Debug("All traceroute checks finished")

	for r := range cResult {
		res[r.addr] = r.res
	}

	log.Debug("Finished traceroute checks", "result", res)

	return res
}

// Shutdown is called once when the check is unregistered or sparrow shuts down
func (tr *Traceroute) Shutdown(_ context.Context) error {
	tr.DoneChan <- struct{}{}
	close(tr.DoneChan)
	return nil
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
	return checks.OpenapiFromPerfData[map[string]result](map[string]result{})
}

// GetMetricCollectors allows the check to provide prometheus metric collectors
func (tr *Traceroute) GetMetricCollectors() []prometheus.Collector {
	return []prometheus.Collector{}
}

// Name returns the name of the check
func (tr *Traceroute) Name() string {
	return CheckName
}
