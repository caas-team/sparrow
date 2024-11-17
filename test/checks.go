package test

import (
	"testing"
	"time"

	"github.com/caas-team/sparrow/internal/helper"
	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/checks/dns"
	"github.com/caas-team/sparrow/pkg/checks/health"
	"github.com/caas-team/sparrow/pkg/checks/latency"
	"github.com/caas-team/sparrow/pkg/checks/traceroute"
	"github.com/goccy/go-yaml"
)

type CheckBuilder interface {
	// For returns the name of the check.
	For() string
	// Check returns the check.
	Check(t *testing.T) checks.Check
	// YAML returns the yaml representation of the check.
	YAML(t *testing.T) []byte
	// ExpectedWaitTime returns the expected wait time for the check.
	ExpectedWaitTime() time.Duration
}

// newCheck creates a new check with the given config.
func newCheck(t *testing.T, c checks.Check, config checks.Runtime) checks.Check {
	t.Helper()
	if err := config.Validate(); err != nil {
		t.Fatalf("[%T] is not a valid config: %v", config, err)
	}

	if err := c.UpdateConfig(config); err != nil {
		t.Fatalf("[%T] failed to update config: %v", c, err)
	}
	return c
}

// checkConfig is a map of check names to their configuration.
type checkConfig map[string]checks.Runtime

func newCheckAsYAML(t *testing.T, cfg checkConfig) []byte {
	t.Helper()
	out, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("[%T] failed to marshal config: %v", cfg, err)
		return []byte{}
	}
	return out
}

var _ CheckBuilder = (*healthCheckBuilder)(nil)

type healthCheckBuilder struct{ cfg health.Config }

// NewHealthCheck returns a new health check builder.
func NewHealthCheck() *healthCheckBuilder {
	return &healthCheckBuilder{cfg: health.Config{Retry: checks.DefaultRetry}}
}

// WithTargets sets the targets for the health check.
func (b *healthCheckBuilder) WithTargets(targets ...string) *healthCheckBuilder {
	b.cfg.Targets = targets
	return b
}

// WithInterval sets the interval for the health check.
func (b *healthCheckBuilder) WithInterval(interval time.Duration) *healthCheckBuilder {
	b.cfg.Interval = interval
	return b
}

// WithTimeout sets the timeout for the health check.
func (b *healthCheckBuilder) WithTimeout(timeout time.Duration) *healthCheckBuilder {
	b.cfg.Timeout = timeout
	return b
}

// WithRetry sets the retry count and delay for the health check.
func (b *healthCheckBuilder) WithRetry(count int, delay time.Duration) *healthCheckBuilder {
	b.cfg.Retry = helper.RetryConfig{Count: count, Delay: delay}
	return b
}

// Check returns the health check.
func (b *healthCheckBuilder) Check(t *testing.T) checks.Check {
	t.Helper()
	return newCheck(t, health.NewCheck(), &b.cfg)
}

// YAML returns the yaml representation of the health check.
func (b *healthCheckBuilder) YAML(t *testing.T) []byte {
	t.Helper()
	return newCheckAsYAML(t, checkConfig{b.cfg.For(): &b.cfg})
}

// ExpectedWaitTime returns the expected wait time for the health check.
func (b *healthCheckBuilder) ExpectedWaitTime() time.Duration {
	return b.cfg.Interval + b.cfg.Timeout + time.Duration(b.cfg.Retry.Count)*b.cfg.Retry.Delay
}

// For returns the name of the check.
func (b *healthCheckBuilder) For() string {
	return b.cfg.For()
}

var _ CheckBuilder = (*latencyConfigBuilder)(nil)

type latencyConfigBuilder struct{ cfg latency.Config }

// NewLatencyCheck returns a new latency check builder.
func NewLatencyCheck() *latencyConfigBuilder {
	return &latencyConfigBuilder{cfg: latency.Config{Retry: checks.DefaultRetry}}
}

// WithTargets sets the targets for the latency check.
func (b *latencyConfigBuilder) WithTargets(targets ...string) *latencyConfigBuilder {
	b.cfg.Targets = targets
	return b
}

// WithInterval sets the interval for the latency check.
func (b *latencyConfigBuilder) WithInterval(interval time.Duration) *latencyConfigBuilder {
	b.cfg.Interval = interval
	return b
}

// WithTimeout sets the timeout for the latency check.
func (b *latencyConfigBuilder) WithTimeout(timeout time.Duration) *latencyConfigBuilder {
	b.cfg.Timeout = timeout
	return b
}

// WithRetry sets the retry count and delay for the latency check.
func (b *latencyConfigBuilder) WithRetry(count int, delay time.Duration) *latencyConfigBuilder {
	b.cfg.Retry = helper.RetryConfig{Count: count, Delay: delay}
	return b
}

// Check returns the latency check.
func (b *latencyConfigBuilder) Check(t *testing.T) checks.Check {
	t.Helper()
	return newCheck(t, latency.NewCheck(), &b.cfg)
}

// YAML returns the yaml representation of the latency check.
func (b *latencyConfigBuilder) YAML(t *testing.T) []byte {
	t.Helper()
	return newCheckAsYAML(t, checkConfig{b.cfg.For(): &b.cfg})
}

// For returns the name of the check.
func (b *latencyConfigBuilder) For() string {
	return b.cfg.For()
}

// ExpectedWaitTime returns the expected wait time for the health check.
func (b *latencyConfigBuilder) ExpectedWaitTime() time.Duration {
	return b.cfg.Interval + b.cfg.Timeout + time.Duration(b.cfg.Retry.Count)*b.cfg.Retry.Delay
}

var _ CheckBuilder = (*dnsConfigBuilder)(nil)

type dnsConfigBuilder struct{ cfg dns.Config }

// NewDNSCheck returns a new dns check builder.
func NewDNSCheck() *dnsConfigBuilder {
	return &dnsConfigBuilder{cfg: dns.Config{Retry: checks.DefaultRetry}}
}

// WithTargets sets the targets for the dns check.
func (b *dnsConfigBuilder) WithTargets(targets ...string) *dnsConfigBuilder {
	b.cfg.Targets = targets
	return b
}

// WithInterval sets the interval for the dns check.
func (b *dnsConfigBuilder) WithInterval(interval time.Duration) *dnsConfigBuilder {
	b.cfg.Interval = interval
	return b
}

// WithTimeout sets the timeout for the dns check.
func (b *dnsConfigBuilder) WithTimeout(timeout time.Duration) *dnsConfigBuilder {
	b.cfg.Timeout = timeout
	return b
}

// WithRetry sets the retry count and delay for the dns check.
func (b *dnsConfigBuilder) WithRetry(count int, delay time.Duration) *dnsConfigBuilder {
	b.cfg.Retry = helper.RetryConfig{Count: count, Delay: delay}
	return b
}

// Check returns the dns check.
func (b *dnsConfigBuilder) Check(t *testing.T) checks.Check {
	t.Helper()
	return newCheck(t, dns.NewCheck(), &b.cfg)
}

// YAML returns the yaml representation of the dns check.
func (b *dnsConfigBuilder) YAML(t *testing.T) []byte {
	t.Helper()
	return newCheckAsYAML(t, checkConfig{b.cfg.For(): &b.cfg})
}

// ExpectedWaitTime returns the expected wait time for the health check.
func (b *dnsConfigBuilder) ExpectedWaitTime() time.Duration {
	return b.cfg.Interval + b.cfg.Timeout + time.Duration(b.cfg.Retry.Count)*b.cfg.Retry.Delay
}

// For returns the name of the check.
func (b *dnsConfigBuilder) For() string {
	return b.cfg.For()
}

var _ CheckBuilder = (*tracerouteConfigBuilder)(nil)

type tracerouteConfigBuilder struct{ cfg traceroute.Config }

// NewTracerouteCheck returns a new traceroute check builder.
func NewTracerouteCheck() *tracerouteConfigBuilder {
	return &tracerouteConfigBuilder{cfg: traceroute.Config{Retry: checks.DefaultRetry}}
}

// WithTargets sets the targets for the traceroute check.
func (b *tracerouteConfigBuilder) WithTargets(targets ...traceroute.Target) *tracerouteConfigBuilder {
	b.cfg.Targets = targets
	return b
}

// WithMaxHops sets the maximum number of hops for the traceroute check.
func (b *tracerouteConfigBuilder) WithMaxHops(maxHops int) *tracerouteConfigBuilder {
	b.cfg.MaxHops = maxHops
	return b
}

// WithInterval sets the interval for the traceroute check.
func (b *tracerouteConfigBuilder) WithInterval(interval time.Duration) *tracerouteConfigBuilder {
	b.cfg.Interval = interval
	return b
}

// WithTimeout sets the timeout for the traceroute check.
func (b *tracerouteConfigBuilder) WithTimeout(timeout time.Duration) *tracerouteConfigBuilder {
	b.cfg.Timeout = timeout
	return b
}

// WithRetry sets the retry count and delay for the traceroute check.
func (b *tracerouteConfigBuilder) WithRetry(count int, delay time.Duration) *tracerouteConfigBuilder {
	b.cfg.Retry = helper.RetryConfig{Count: count, Delay: delay}
	return b
}

// Check returns the traceroute check.
func (b *tracerouteConfigBuilder) Check(t *testing.T) checks.Check {
	t.Helper()
	return newCheck(t, traceroute.NewCheck(), &b.cfg)
}

// YAML returns the yaml representation of the traceroute check.
func (b *tracerouteConfigBuilder) YAML(t *testing.T) []byte {
	t.Helper()
	return newCheckAsYAML(t, checkConfig{b.cfg.For(): &b.cfg})
}

// ExpectedWaitTime returns the expected wait time for the health check.
func (b *tracerouteConfigBuilder) ExpectedWaitTime() time.Duration {
	return b.cfg.Interval + b.cfg.Timeout + time.Duration(b.cfg.Retry.Count)*b.cfg.Retry.Delay
}

// For returns the name of the check.
func (b *tracerouteConfigBuilder) For() string {
	return b.cfg.For()
}
