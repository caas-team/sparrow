package traceroute

import (
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/caas-team/sparrow/pkg/checks"
)

// Config is the configuration for the traceroute check
type Config struct {
	// Targets is a list of targets to traceroute to
	Targets []Target `json:"targets" yaml:"targets" mapstructure:"targets"`
	// Retries is the number of times to retry the traceroute for a target, if it fails
	Retries int `json:"retries" yaml:"retries" mapstructure:"retries"`
	// MaxHops is the maximum number of hops to try before giving up
	MaxHops int `json:"maxHops" yaml:"maxHops" mapstructure:"maxHops"`
	// Interval is the time to wait between check iterations
	Interval time.Duration `json:"interval" yaml:"interval" mapstructure:"interval"`
	// Timeout is the maximum time to wait for a response from a hop
	Timeout time.Duration `json:"timeout" yaml:"timeout" mapstructure:"timeout"`
}

func (c *Config) For() string {
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
		ip := net.ParseIP(t.Addr)
		if ip != nil {
			continue
		}

		_, err := url.Parse(t.Addr)
		if err != nil {
			return checks.ErrInvalidConfig{CheckName: CheckName, Field: fmt.Sprintf("traceroute.targets[%d].addr", i), Reason: "invalid url or ip"}
		}
	}
	return nil
}
