package sparrow

import (
	"reflect"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestPrometheusMetrics_GetRegistry(t *testing.T) {
	type fields struct {
		registry *prometheus.Registry
	}
	tests := []struct {
		name   string
		fields fields
		want   *prometheus.Registry
	}{
		{
			name: "simple registry",
			fields: fields{
				registry: prometheus.NewRegistry(),
			},
			want: prometheus.NewRegistry(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &PrometheusMetrics{
				registry: tt.fields.registry,
			}
			if got := m.GetRegistry(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PrometheusMetrics.GetRegistry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewMetrics(t *testing.T) {
	testMetrics := NewMetrics()
	testGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "TEST_GAUGE",
		},
	)

	t.Run("Register a collector", func(t *testing.T) {
		testMetrics.(*PrometheusMetrics).registry.MustRegister(
			testGauge,
		)
	})
}
