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

package dns

import (
	"testing"

	"github.com/caas-team/sparrow/test"
	"github.com/prometheus/client_golang/prometheus"
)

func TestMetrics_GetCollectors(t *testing.T) {
	test.MarkAsShort(t)

	tests := []struct {
		name    string
		metrics metrics
	}{
		{
			name:    "success with metrics constructor",
			metrics: newMetrics(),
		},
		{
			name: "success with custom metrics",
			metrics: metrics{
				status: prometheus.NewGaugeVec(
					prometheus.GaugeOpts{
						Name: "sparrow_dns_status",
						Help: "Specifies if the target can be resolved.",
					},
					[]string{"target"},
				),
				duration: prometheus.NewGaugeVec(
					prometheus.GaugeOpts{
						Name: "sparrow_dns_duration",
						Help: "Duration of DNS resolution attempts in seconds.",
					},
					[]string{"target"},
				),
				count: prometheus.NewCounterVec(
					prometheus.CounterOpts{
						Name: "sparrow_dns_check_count",
						Help: "Total number of DNS checks performed on the target and if they were successful.",
					},
					[]string{"target"},
				),
				histogram: prometheus.NewHistogramVec(
					prometheus.HistogramOpts{
						Name: "sparrow_dns_duration",
						Help: "Histogram of response times for DNS checks in seconds.",
					},
					[]string{"target"},
				),
			},
		},
	}
	for _, tt := range tests {
		tt.metrics.Set("test", make(map[string]result, 1), float64(1))

		if tt.metrics.GetCollectors() == nil {
			t.Errorf("metrics.GetCollectors() = %v", tt.metrics.GetCollectors())
		}
	}
}
