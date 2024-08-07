package traceroute

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	labelTarget = "target"
)

type metrics struct {
	minHops       *prometheus.GaugeVec
	checkDuration *prometheus.GaugeVec
}

func (m metrics) List() []prometheus.Collector {
	return []prometheus.Collector{
		m.minHops,
		m.checkDuration,
	}
}

func (m metrics) MinHops(data map[string]result) {
	for target, hops := range data {
		m.minHops.With(prometheus.Labels{labelTarget: target}).Set(float64(hops.MinHops))
	}
}

func (m metrics) CheckDuration(target string, n time.Duration) {
	m.checkDuration.With(prometheus.Labels{labelTarget: target}).Set(float64(n.Milliseconds()))
}

func newMetrics() metrics {
	return metrics{
		minHops: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "sparrow_traceroute",
			Name:      "minimum_hops",
		}, []string{labelTarget}),
		checkDuration: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "sparrow_traceroute",
			Name:      "check_duration_ms",
		}, []string{labelTarget}),
	}
}
