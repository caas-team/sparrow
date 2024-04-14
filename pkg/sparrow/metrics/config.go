package metrics

import (
	"context"
	"fmt"

	"github.com/caas-team/sparrow/internal/logger"
)

// Config holds the configuration for OpenTelemetry
type Config struct {
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
