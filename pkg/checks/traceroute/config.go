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

// ProtocolType is the type for the protocol to use for the traceroute
type ProtocolType string

// String returns the string representation of the protocol type
func (p ProtocolType) String() string {
	return string(p)
}

// ToProtocol converts the protocol type to a traceroute protocol
func (p ProtocolType) ToProtocol() traceroute.Protocol {
	switch p {
	case "icmp":
		return traceroute.ICMP
	case "udp":
		return traceroute.UDP
	case "tcp":
		return traceroute.TCP
	default:
		return -1
	}
}

// Config is the configuration for the traceroute check
type Config struct {
	// Targets is a list of targets to traceroute to
	Targets []Target `json:"targets" yaml:"targets" mapstructure:"targets"`
	// Protocol is the protocol to use for the traceroute
	Protocol ProtocolType `json:"protocol" yaml:"protocol" mapstructure:"protocol"`
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
	if !c.hasElevatedCapabilities() {
		return checks.ErrInvalidConfig{CheckName: CheckName, Field: "traceroute", Reason: "requires either elevated capabilities (CAP_NET_RAW) or running as root"}
	}

	switch c.Protocol {
	case "icmp", "udp", "tcp":
	default:
		return checks.ErrInvalidConfig{CheckName: CheckName, Field: "traceroute.protocol", Reason: "must be one of 'icmp', 'udp', 'tcp'"}
	}

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

func (c *Config) hasElevatedCapabilities() bool {
	return helper.HasCapabilities(helper.CAP_NET_RAW)
}
