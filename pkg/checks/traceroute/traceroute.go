package traceroute

import (
	"context"
	"fmt"
	"net"
	"net/url"
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

type Config struct {
	Targets  []Target      `json:"targets" yaml:"targets" mapstructure:"targets"`
	Retries  int           `json:"retries" yaml:"retries" mapstructure:"retries"`
	MaxHops  int           `json:"maxHops" yaml:"maxHops" mapstructure:"maxHops"`
	Interval time.Duration `json:"interval" yaml:"interval" mapstructure:"interval"`
	Timeout  time.Duration `json:"timeout" yaml:"timeout" mapstructure:"timeout"`
}

func (c Config) For() string {
	return CheckName
}

type Target struct {
	Addr string `json:"addr" yaml:"addr" mapstructure:"addr"`
	Port uint16 `json:"port" yaml:"port" mapstructure:"port"`
}

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
	// The minimum amount of hops required to reach the target
	NumHops int
	// The path taken to the destination
	Hops []Hop
}

type Hop struct {
	Addr    string
	Latency time.Duration
	Success bool
}

func (d *Traceroute) Run(ctx context.Context, cResult chan checks.ResultDTO) error {
	ctx, cancel := logger.NewContextWithLogger(ctx)
	defer cancel()
	log := logger.FromContext(ctx)
	log.Info("Starting traceroute check", "interval", d.config.Interval.String())

	for {
		select {
		case <-ctx.Done():
			log.Error("Context canceled", "err", ctx.Err())
			return ctx.Err()
		case <-d.DoneChan:
			return nil
		case <-time.After(d.config.Interval):
			res := d.check(ctx)

			cResult <- checks.ResultDTO{
				Name: d.Name(),
				Result: &checks.Result{
					Data:      res,
					Timestamp: time.Now(),
				},
			}
			log.Debug("Successfully finished traceroute check run")
		}
	}
}

func (c *Traceroute) GetConfig() checks.Runtime {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	return &c.config
}

func (c *Traceroute) check(ctx context.Context) map[string]result {
	res := make(map[string]result)
	log := logger.FromContext(ctx)

	type internalResult struct {
		addr string
		res  result
	}

	var wg sync.WaitGroup
	cResult := make(chan internalResult, len(c.config.Targets))

	for _, t := range c.config.Targets {
		wg.Add(1)
		go func(t Target) {
			defer wg.Done()
			log.Debug("Running traceroute", "target", t.Addr)
			start := time.Now()
			tr, trerr := c.traceroute(t.Addr, int(t.Port), int(c.config.Timeout/time.Millisecond), c.config.Retries, c.config.MaxHops)
			duration := time.Since(start)
			if trerr != nil {
				log.Error("Error running traceroute", "err", trerr, "target", t.Addr)
			}

			log.Debug("Ran traceroute", "result", tr, "duration", duration)

			r := result{
				NumHops: len(tr.Hops),
				Hops:    []Hop{},
			}

			for _, h := range tr.Hops {
				r.Hops = append(r.Hops, Hop{
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
func (c *Traceroute) Shutdown(_ context.Context) error {
	c.DoneChan <- struct{}{}
	close(c.DoneChan)
	return nil
}

// SetConfig is called once when the check is registered
// This is also called while the check is running, if the remote config is updated
// This should return an error if the config is invalid
func (c *Traceroute) SetConfig(cfg checks.Runtime) error {
	if cfg, ok := cfg.(*Config); ok {
		c.Mu.Lock()
		defer c.Mu.Unlock()
		c.config = *cfg
		return nil
	}

	return checks.ErrConfigMismatch{
		Expected: CheckName,
		Current:  cfg.For(),
	}
}

// Schema returns an openapi3.SchemaRef of the result type returned by the check
func (c *Traceroute) Schema() (*openapi3.SchemaRef, error) {
	return checks.OpenapiFromPerfData[map[string]result](map[string]result{})
}

// GetMetricCollectors allows the check to provide prometheus metric collectors
func (c *Traceroute) GetMetricCollectors() []prometheus.Collector {
	return []prometheus.Collector{}
}

func (c *Traceroute) Name() string {
	return CheckName
}

func (c *Config) Validate() error {
	if c.Timeout <= 0 {
		return checks.ErrInvalidConfig{CheckName: CheckName, Field: "traceroute.timeout", Reason: "must be greater than 0"}
	}
	if c.Interval <= 0 {
		return checks.ErrInvalidConfig{CheckName: CheckName, Field: "traceroute.interval", Reason: "must be greater than 0"}
	}

	for i, t := range c.Targets {
		net.ParseIP(t.Addr)
		if _, err := url.Parse(t.Addr); err != nil {
			return checks.ErrInvalidConfig{CheckName: CheckName, Field: fmt.Sprintf("traceroute.targets[%d].addr", i), Reason: "invalid url"}
		}
	}
	return nil
}
