package sparrow

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

//go:generate moq -out metrics_moq.go . Metrics
type Metrics interface {
	GetRegistry() *prometheus.Registry
}

type PrometheusMetrics struct {
	registry *prometheus.Registry
}

// InitMetrics initializes the metrics and returns the PrometheusMetrics
func NewMetrics() Metrics {
	registry := prometheus.NewRegistry()

	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	return &PrometheusMetrics{registry: registry}
}

// GetRegistry returns the registry to register prometheus metrics
func (m *PrometheusMetrics) GetRegistry() *prometheus.Registry {
	return m.registry
}
