package udptraceroute

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/aeden/traceroute"
	"github.com/caas-team/sparrow/internal/helper"
	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	_         checks.Check = (*UDPTraceroute)(nil)
	CheckName              = "udptraceroute"
)

type config struct {
	Targets  []Target
	Interval time.Duration      `json:"interval" yaml:"interval" mapstructure:"interval"`
	Timeout  time.Duration      `json:"timeout" yaml:"timeout" mapstructure:"timeout"`
	Retry    helper.RetryConfig `json:"retry" yaml:"retry" mapstructure:"retry"`
}

func (c config) For() string {
	return CheckName
}

type Target struct {
	Addr string
}

func NewCheck() checks.Check {
	return &UDPTraceroute{
		config:     config{},
		traceroute: newTraceroute,
		CheckBase: checks.CheckBase{
			Mu:      sync.Mutex{},
			CResult: make(chan checks.Result),
			Done:    make(chan bool),
		},
	}
}

type UDPTraceroute struct {
	checks.CheckBase
	config     config
	traceroute tracerouteFactory
}

type tracerouteFactory func(dest string) (traceroute.TracerouteResult, error)

func newTraceroute(dest string) (traceroute.TracerouteResult, error) {
	return traceroute.Traceroute(dest, nil)
}

type Result struct {
	// The minimum amount of hops required to reach the target
	NumHops int
	// The path taken to the destination
	Hops []Hop
}

type Hop struct {
	Addr    net.IP
	Latency time.Duration
	Success bool
}

// Run is called once per check interval
// this should error if there is a problem running the check
// Returns an error and a result. Returning a non nil error will cause a shutdown of the system
func (c *UDPTraceroute) Run(ctx context.Context) error {
	ctx, cancel := logger.NewContextWithLogger(ctx)
	defer cancel()
	log := logger.FromContext(ctx)
	log.Info("Starting latency check", "interval", c.config.Interval.String())

	for {
		select {
		case <-ctx.Done():
			log.Error("Context canceled", "err", ctx.Err())
			return ctx.Err()
		case <-c.Done:
			return nil
		case <-time.After(c.config.Interval):
			res, _ := c.check(ctx)
			errval := ""
			r := checks.Result{
				Data:      res,
				Err:       errval,
				Timestamp: time.Now(),
			}

			c.CResult <- r
			log.Debug("Successfully finished latency check run")
		}
	}
}

func (c *UDPTraceroute) GetConfig() checks.Runtime {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	return c.config
}

func (c *UDPTraceroute) check(_ context.Context) (map[string]Result, error) {
	res := make(map[string]Result)
	var err error
	for _, t := range c.config.Targets {
		tr, trerr := c.traceroute(t.Addr)
		if trerr != nil {
			err = trerr
			continue
		}

		result := Result{
			NumHops: len(tr.Hops),
			Hops:    []Hop{},
		}

		for _, h := range tr.Hops {
			result.Hops = append(result.Hops, Hop{
				Addr:    net.IPv4(h.Address[0], h.Address[1], h.Address[2], h.Address[3]),
				Latency: h.ElapsedTime,
				Success: h.Success,
			})
		}
		res[t.Addr] = result
	}
	return res, err
}

type MultiError struct {
	errors []error
}

func (m MultiError) Error() string {
	result := "["
	if len(m.errors) > 0 {
		result = m.errors[0].Error()
	}
	for _, err := range m.errors {
		result = fmt.Sprintf("%s, %s", result, err.Error())
	}
	result += "]"
	return result
}

// Startup is called once when the check is registered
// In the Run() method, the check should send results to the cResult channel
// this will cause sparrow to update its data store with the results
func (c *UDPTraceroute) Startup(_ context.Context, cResult chan<- checks.Result) error {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	c.CheckBase.CResult = cResult
	return nil
}

// Shutdown is called once when the check is unregistered or sparrow shuts down
func (c *UDPTraceroute) Shutdown(_ context.Context) error {
	return nil
}

// SetConfig is called once when the check is registered
// This is also called while the check is running, if the remote config is updated
// This should return an error if the config is invalid
func (c *UDPTraceroute) SetConfig(cfg checks.Runtime) error {
	newConfg, ok := cfg.(*config)
	if ok {
		c.Mu.Lock()
		defer c.Mu.Unlock()
		c.config = *newConfg
		return checks.ErrConfigMismatch{
			Expected: CheckName,
			Current:  cfg.For(),
		}
	}
	if err := validateConfig(*newConfg); err != nil {
		return checks.ErrConfigMismatch{
			Expected: CheckName,
			Current:  cfg.For(),
		}
	}

	return nil
}

// Schema returns an openapi3.SchemaRef of the result type returned by the check
func (c *UDPTraceroute) Schema() (*openapi3.SchemaRef, error) {
	return checks.OpenapiFromPerfData[Result](Result{})
}

// GetMetricCollectors allows the check to provide prometheus metric collectors
func (c *UDPTraceroute) GetMetricCollectors() []prometheus.Collector {
	return []prometheus.Collector{}
}

func (c *UDPTraceroute) Name() string {
	return CheckName
}

func validateConfig(_ config) error {
	return nil
}
