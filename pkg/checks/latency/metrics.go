package latency

import (
	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/prometheus/client_golang/prometheus"
)

// newMetrics initializes metric collectors of the latency check
func newMetrics() metrics {
	return metrics{
		totalDuration: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "sparrow_latency_seconds",
				Help: "Latency for each target",
			},
			[]string{
				"target",
			},
		),
		count: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "sparrow_latency_count",
				Help: "Count of latency checks done",
			},
			[]string{
				"target",
			},
		),
		histogram: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "sparrow_latency_duration",
				Help: "Latency of targets in seconds",
			},
			[]string{
				"target",
			},
		),
	}
}

// GetMetricCollectors returns all metric collectors of check
func (l *Latency) GetMetricCollectors() []prometheus.Collector {
	return []prometheus.Collector{
		l.metrics.totalDuration,
		l.metrics.count,
		l.metrics.histogram,
	}
}

func (m metrics) Remove(label string) error {
	if !m.totalDuration.Delete(map[string]string{"target": label}) {
		return checks.ErrMetricNotFound{Label: label}
	}

	if !m.count.Delete(map[string]string{"target": label}) {
		return checks.ErrMetricNotFound{Label: label}
	}

	if !m.histogram.Delete(map[string]string{"target": label}) {
		return checks.ErrMetricNotFound{Label: label}
	}

	return nil
}
