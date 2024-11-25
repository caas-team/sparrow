// sparrow
// (C) 2024, Deutsche Telekom IT GmbH
//
// Deutsche Telekom IT GmbH and all other contributors /
// copyright owners license this file to you under the Apache
// License, Version 2.0 (the "License"); you may not use this
// file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package metrics

import (
	"context"
	"reflect"
	"testing"

	"github.com/caas-team/sparrow/test"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestPrometheusMetrics_GetRegistry(t *testing.T) {
	test.MarkAsShort(t)

	tests := []struct {
		name     string
		registry *prometheus.Registry
		want     *prometheus.Registry
	}{
		{
			name:     "simple registry",
			registry: prometheus.NewRegistry(),
			want:     prometheus.NewRegistry(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &manager{
				registry: tt.registry,
			}
			if got := m.GetRegistry(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PrometheusMetrics.GetRegistry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewMetrics(t *testing.T) {
	test.MarkAsShort(t)

	testMetrics := New(Config{})
	testGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "TEST_GAUGE",
		},
	)

	t.Run("Register a collector", func(t *testing.T) {
		testMetrics.(*manager).registry.MustRegister(
			testGauge,
		)
	})
}

func TestMetrics_InitTracing(t *testing.T) {
	test.MarkAsShort(t)

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "success - stdout exporter",
			config: Config{
				Exporter: STDOUT,
				Url:      "",
				Token:    "",
			},
			wantErr: false,
		},
		{
			name: "success - otlp exporter",
			config: Config{
				Exporter: HTTP,
				Url:      "http://localhost:4317",
				Token:    "",
			},
			wantErr: false,
		},
		{
			name: "success - otlp exporter with token",
			config: Config{
				Exporter: GRPC,
				Url:      "http://localhost:4317",
				Token:    "my-super-secret-token",
			},
			wantErr: false,
		},
		{
			name: "success - no exporter",
			config: Config{
				Exporter: NOOP,
				Url:      "",
				Token:    "",
			},
			wantErr: false,
		},
		{
			name: "failure - unsupported exporter",
			config: Config{
				Exporter: "unsupported",
				Url:      "",
				Token:    "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(tt.config)
			if err := m.InitTracing(context.Background()); (err != nil) != tt.wantErr {
				t.Errorf("Metrics.InitTracing() error = %v", err)
			}

			if tp, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider); !ok {
				t.Errorf("Metrics.InitTracing() type = %T, want = %T", tp, &sdktrace.TracerProvider{})
			}

			if err := m.Shutdown(context.Background()); err != nil {
				t.Fatalf("Metrics.Shutdown() error = %v", err)
			}
		})
	}
}
