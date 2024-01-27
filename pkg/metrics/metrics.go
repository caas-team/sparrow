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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

//go:generate moq -out metrics_moq.go . Metrics
type Metrics interface {
	// GetRegistry returns the prometheus registry instance
	// containing the registered prometheus collectors
	GetRegistry() *prometheus.Registry
}

type PrometheusMetrics struct {
	registry *prometheus.Registry
}

// NewMetrics initializes the metrics and returns the PrometheusMetrics
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
