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
	"fmt"

	"github.com/caas-team/sparrow/internal/logger"
)

// Config holds the configuration for OpenTelemetry
type Config struct {
	// Enabled is a flag to enable or disable the OpenTelemetry
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`
	// Exporter is the otlp exporter used to export the traces
	Exporter Exporter `yaml:"exporter" mapstructure:"exporter"`
	// Url is the Url of the collector to which the traces are exported
	Url string `yaml:"url" mapstructure:"url"`
	// Token is the token used to authenticate with the collector
	Token string `yaml:"token" mapstructure:"token"`
	// CertPath is the path to the tls certificate file
	CertPath string `yaml:"certPath" mapstructure:"certPath"`
}

func (c *Config) Validate(ctx context.Context) error {
	log := logger.FromContext(ctx)
	if err := c.Exporter.Validate(); err != nil {
		log.ErrorContext(ctx, "Invalid exporter", "error", err)
		return err
	}

	if c.Exporter.IsExporting() {
		if c.Url == "" {
			log.ErrorContext(ctx, "Url is required for otlp exporter", "exporter", c.Exporter)
			return fmt.Errorf("url is required for otlp exporter %q", c.Exporter)
		}
	}
	return nil
}
