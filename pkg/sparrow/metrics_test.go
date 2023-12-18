// sparrow
// (C) 2023, Deutsche Telekom IT GmbH
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
