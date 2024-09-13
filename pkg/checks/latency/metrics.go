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

package latency

import (
	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/prometheus/client_golang/prometheus"
)

// metrics defines the metric collectors of the latency check
type metrics struct {
	totalDuration *prometheus.GaugeVec
	count         *prometheus.CounterVec
	histogram     *prometheus.HistogramVec
}

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

// Remove removes the metrics which have the passed target as a label
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
