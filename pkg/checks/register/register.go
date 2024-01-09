package register

import (
	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/checks/health"
	"github.com/caas-team/sparrow/pkg/checks/latency"
)

var (
	// RegisteredChecks will be registered in this map
	// The key is the name of the Check
	// The name needs to map the configuration item key
	RegisteredChecks = map[string]func() checks.Check{
		"health":  health.NewHealthCheck,
		"latency": latency.NewLatencyCheck,
	}
)
