package metrics

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Exporter is the protocol used to export the traces
type Exporter string

const (
	// HTTP is the protocol used to export the traces via HTTP/1.1
	HTTP Exporter = "http"
	// GRPC is the protocol used to export the traces via HTTP/2 (gRPC)
	GRPC Exporter = "grpc"
	// STDOUT is the protocol used to export the traces to the standard output
	STDOUT Exporter = "stdout"
	// NOOP is the protocol used to not export the traces
	NOOP Exporter = "noop"
)

// String returns the string representation of the protocol
func (e Exporter) String() string {
	return string(e)
}

// Validate validates the protocol
func (e Exporter) Validate() error {
	switch e {
	case HTTP, GRPC, STDOUT, NOOP:
		return nil
	default:
		return fmt.Errorf("unsupported exporter type: %s", e.String())
	}
}

// IsExporting returns true if the protocol is exporting the traces
func (e Exporter) IsExporting() bool {
	return e == HTTP || e == GRPC
}

// createExporter creates a new span exporter based on the protocol
func (m *metrics) createExporter(ctx context.Context) (sdktrace.SpanExporter, error) {
	headers := make(map[string]string)
	if m.config.Token != "" {
		headers["Authorization"] = fmt.Sprintf("Bearer %s", m.config.Token)
	}

	switch m.config.Exporter {
	case HTTP:
		return otlptracehttp.New(ctx,
			otlptracehttp.WithEndpoint(m.config.Url),
			otlptracehttp.WithHeaders(headers),
		)
	case GRPC:
		return otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(m.config.Url),
			otlptracegrpc.WithHeaders(headers),
		)
	case STDOUT:
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	case NOOP, "":
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported exporter type: %s", m.config.Exporter.String())
	}
}
