package udptraceroute

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/aeden/traceroute"
	"github.com/caas-team/sparrow/internal/helper"
	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/checks/types"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/prometheus/client_golang/prometheus"
)

var _ checks.Check = (*UDPTraceroute)(nil)

type config struct {
	Targets  []Target
	Interval time.Duration      `json:"interval" yaml:"interval" mapstructure:"interval"`
	Timeout  time.Duration      `json:"timeout" yaml:"timeout" mapstructure:"timeout"`
	Retry    helper.RetryConfig `json:"retry" yaml:"retry" mapstructure:"retry"`
}

type Target struct {
	Addr string
}

type UDPTraceroute struct {
	types.CheckBase
	config  config
	cResult chan<- types.Result
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
			res := c.check(ctx)
			errval := ""
			r := types.Result{
				Data:      res,
				Err:       errval,
				Timestamp: time.Now(),
			}

			c.CResult <- r
			log.Debug("Successfully finished latency check run")
		}
	}

	return nil
}

func (c *UDPTraceroute) check(ctx context.Context) map[string]Result {
	res := make(map[string]Result)
	var resMu sync.Mutex
	for _, t := range c.config.Targets {
		tr, err := traceroute.Traceroute(t.Addr, nil)
		if err != nil {
			panic(err)
		}

		// ip := net.IPv4(res.DestinationAddress[0], res.DestinationAddress[1], res.DestinationAddress[2], res.DestinationAddress[3])

		r := Result{
			NumHops: len(tr.Hops),
			Hops:    []Hop{},
		}

		for _, h := range tr.Hops {
			r.Hops = append(r.Hops, Hop{
				Addr:    h.Address[:],
				Latency: h.ElapsedTime,
			})
		}

		resMu.Lock()
		defer resMu.Unlock()

		res[t.Addr] = r
	}
	return
}

// Startup is called once when the check is registered
// In the Run() method, the check should send results to the cResult channel
// this will cause sparrow to update its data store with the results
func (c *UDPTraceroute) Startup(ctx context.Context, cResult chan<- types.Result) error {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	c.cResult = cResult
	return nil
}

// Shutdown is called once when the check is unregistered or sparrow shuts down
func (c *UDPTraceroute) Shutdown(ctx context.Context) error {
	return nil
}

// SetConfig is called once when the check is registered
// This is also called while the check is running, if the remote config is updated
// This should return an error if the config is invalid
func (c *UDPTraceroute) SetConfig(ctx context.Context, cfg any) error {
	decoded, err := helper.Decode[config](cfg)
	if err != nil {
		return err
	}
	err = validateConfig(decoded)
	if err != nil {
		return err
	}
	c.Mu.Lock()
	defer c.Mu.Unlock()
	c.config = decoded
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

func validateConfig(c config) error {
	return nil
}
