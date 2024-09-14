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

package traceroute

import (
	"time"

	"github.com/caas-team/sparrow/pkg/checks"

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

// Remove removes a metric with a specific label
func (m metrics) Remove(label string) error {
	if !m.minHops.DeleteLabelValues(label) {
		return checks.ErrMetricNotFound{Label: label}
	}
	if !m.checkDuration.DeleteLabelValues(label) {
		return checks.ErrMetricNotFound{Label: label}
	}
	return nil
}
