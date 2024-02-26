package traceroute

import (
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/caas-team/sparrow/pkg/checks"
)

type Config struct {
	Targets  []Target      `json:"targets" yaml:"targets" mapstructure:"targets"`
	Retries  int           `json:"retries" yaml:"retries" mapstructure:"retries"`
	MaxHops  int           `json:"maxHops" yaml:"maxHops" mapstructure:"maxHops"`
	Interval time.Duration `json:"interval" yaml:"interval" mapstructure:"interval"`
	Timeout  time.Duration `json:"timeout" yaml:"timeout" mapstructure:"timeout"`
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
		net.ParseIP(t.Addr)
		if _, err := url.Parse(t.Addr); err != nil {
			return checks.ErrInvalidConfig{CheckName: CheckName, Field: fmt.Sprintf("traceroute.targets[%d].addr", i), Reason: "invalid url"}
		}
	}
	return nil
}
