package traceroute

import (
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/caas-team/sparrow/internal/helper"
	"github.com/caas-team/sparrow/internal/traceroute"
	"github.com/caas-team/sparrow/pkg/checks"
)

// Config is the configuration for the traceroute check
type Config struct {
	// Targets is a list of targets to traceroute to
	Targets []Target `json:"targets" yaml:"targets" mapstructure:"targets"`
	// Protocol is the protocol to use for the traceroute
	Protocol traceroute.Protocol `json:"protocol" yaml:"protocol" mapstructure:"protocol"`
	// Interval is the time to wait between check iterations
	Interval time.Duration `json:"interval" yaml:"interval" mapstructure:"interval"`
	// Timeout is the maximum time to wait for a response from a hop
	Timeout time.Duration `json:"timeout" yaml:"timeout" mapstructure:"timeout"`
	// MaxHops is the maximum number of hops to try before giving up
	MaxHops int `json:"maxHops" yaml:"maxHops" mapstructure:"maxHops"`
	// Retry is the configuration for the retry mechanism for each target
	Retry helper.RetryConfig `json:"retry" yaml:"retry" mapstructure:"retry"`
}

// Target represents a target to traceroute to
type Target struct {
	// Addr is the address of the target
	Addr string `json:"addr" yaml:"addr" mapstructure:"addr"`
	// Port is the port of the target
	Port uint16 `json:"port" yaml:"port" mapstructure:"port"`
}

func (c *Config) For() string {
	return CheckName
}

func (c *Config) Validate() error {
	if err := c.Protocol.Validate(); err != nil {
		return checks.ErrInvalidConfig{CheckName: CheckName, Field: "traceroute.protocol", Reason: err.Error()}
	}

	if c.Timeout <= 0 {
		return checks.ErrInvalidConfig{CheckName: CheckName, Field: "traceroute.timeout", Reason: "must be greater than 0"}
	}
	if c.Interval <= 0 {
		return checks.ErrInvalidConfig{CheckName: CheckName, Field: "traceroute.interval", Reason: "must be greater than 0"}
	}

	for i, t := range c.Targets {
		if ip := net.ParseIP(t.Addr); ip != nil {
			continue
		}

		if _, err := url.Parse(t.Addr); err != nil {
			return checks.ErrInvalidConfig{CheckName: CheckName, Field: fmt.Sprintf("traceroute.targets[%d].addr", i), Reason: "invalid url or ip"}
		}

		if t.Port == 0 {
			c.Targets[i].Port = 80
		}
	}

	return nil
}
