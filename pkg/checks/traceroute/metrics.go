package traceroute

import (
	"fmt"

	"github.com/caas-team/sparrow/internal/traceroute"
	"github.com/prometheus/client_golang/prometheus"
)

// metrics contains the metric collectors for the traceroute check
type metrics struct {
	*prometheus.GaugeVec
}

// newMetrics initializes metric collectors of the traceroute check
func newMetrics() metrics {
	return metrics{
		GaugeVec: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "traceroute_hop_duration_seconds",
				Help: "Duration of each hop in the traceroute",
			},
			[]string{"target", "hop"},
		),
	}
}

// Set sets the metrics for the traceroute results
func (m *metrics) Set(target string, hops []traceroute.Hop) {
	for _, hop := range hops {
		if hop.Error == "" {
			m.WithLabelValues(target, fmt.Sprintf("%d", hop.Tracepoint)).Set(hop.Duration)
		}
	}
}
