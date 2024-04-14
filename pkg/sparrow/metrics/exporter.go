package metrics

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc/credentials"
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

// FileOpener is the function used to open a file
type FileOpener func(string) (fs.File, error)

// OpenFile is the function used to open a file
var OpenFile FileOpener = func() FileOpener {
	return func(name string) (fs.File, error) {
		return os.Open(name)
	}
}()

// TLSConfig returns the TLS configuration based on the certificate file
func (e Exporter) TLSConfig(certFile string) (conf *tls.Config, err error) {
	if certFile == "" || certFile == "insecure" {
		return nil, nil
	}

	file, err := OpenFile(certFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open certificate file: %w", err)
	}
	defer func() {
		if cErr := file.Close(); cErr != nil {
			err = errors.Join(err, cErr)
		}
	}()

	b, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file: %w", err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(b) {
		return nil, fmt.Errorf("failed to append certificate from file: %s", certFile)
	}

	return &tls.Config{
		RootCAs:            certPool,
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
	}, nil
}

// exporterFactory is a function that creates a new exporter
type exporterFactory func(ctx context.Context, config *Config) (sdktrace.SpanExporter, error)

// registry contains the mapping of the exporter to the factory function
var registry = map[Exporter]exporterFactory{
	HTTP:   newHTTPExporter,
	GRPC:   newGRPCExporter,
	STDOUT: newStdoutExporter,
	NOOP:   newNoopExporter,
}

// Create creates a new exporter based on the configuration
func (e Exporter) Create(ctx context.Context, config *Config) (sdktrace.SpanExporter, error) {
	if factory, ok := registry[e]; ok {
		return factory(ctx, config)
	}
	return nil, fmt.Errorf("unsupported exporter type: %s", config.Exporter.String())
}

// newHTTPExporter creates a new HTTP exporter
func newHTTPExporter(ctx context.Context, config *Config) (sdktrace.SpanExporter, error) {
	headers, tlsCfg, err := getCommonConfig(config)
	if err != nil {
		return nil, err
	}

	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(config.Url),
		otlptracehttp.WithHeaders(headers),
	}
	if tlsCfg != nil {
		opts = append(opts, otlptracehttp.WithTLSClientConfig(tlsCfg))
	} else {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	return otlptracehttp.New(ctx, opts...)
}

// newGRPCExporter creates a new gRPC exporter
func newGRPCExporter(ctx context.Context, config *Config) (sdktrace.SpanExporter, error) {
	headers, tlsCfg, err := getCommonConfig(config)
	if err != nil {
		return nil, err
	}

	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(config.Url),
		otlptracegrpc.WithHeaders(headers),
	}
	if tlsCfg != nil {
		opts = append(opts, otlptracegrpc.WithTLSCredentials(credentials.NewTLS(tlsCfg)))
	} else {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	return otlptracegrpc.New(ctx, opts...)
}

// getCommonConfig returns the common configuration for the exporters
func getCommonConfig(config *Config) (map[string]string, *tls.Config, error) {
	headers := make(map[string]string)
	if config.Token != "" {
		headers["Authorization"] = fmt.Sprintf("Bearer %s", config.Token)
	}

	tlsCfg, err := config.Exporter.TLSConfig(config.CertPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create TLS configuration: %w", err)
	}

	return headers, tlsCfg, nil
}

// newStdoutExporter creates a new stdout exporter
func newStdoutExporter(_ context.Context, _ *Config) (sdktrace.SpanExporter, error) {
	return stdouttrace.New(stdouttrace.WithPrettyPrint())
}

// newNoopExporter creates a new noop exporter
func newNoopExporter(_ context.Context, _ *Config) (sdktrace.SpanExporter, error) {
	return nil, nil
}
