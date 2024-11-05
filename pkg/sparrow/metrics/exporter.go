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
	case HTTP, GRPC, STDOUT, NOOP, "":
		return nil
	default:
		return fmt.Errorf("unsupported exporter type: %s", e.String())
	}
}

// IsExporting returns true if the protocol is exporting the traces
func (e Exporter) IsExporting() bool {
	return e == HTTP || e == GRPC
}

// exporterFactory is a function that creates a new exporter
type exporterFactory func(ctx context.Context, config *Config) (sdktrace.SpanExporter, error)

// registry contains the mapping of the exporter to the factory function
var registry = map[Exporter]exporterFactory{
	HTTP:   newHTTPExporter,
	GRPC:   newGRPCExporter,
	STDOUT: newStdoutExporter,
	NOOP:   newNoopExporter,
	"":     newNoopExporter,
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
	cfg, err := newExporterConfig(config)
	if err != nil {
		return nil, err
	}

	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(config.Url),
		otlptracehttp.WithHeaders(cfg.headers),
	}
	if !config.TLS.Enabled {
		opts = append(opts, otlptracehttp.WithInsecure())
		return otlptracehttp.New(ctx, opts...)
	}
	if cfg.tls != nil {
		opts = append(opts, otlptracehttp.WithTLSClientConfig(cfg.tls))
	}

	return otlptracehttp.New(ctx, opts...)
}

// newGRPCExporter creates a new gRPC exporter
func newGRPCExporter(ctx context.Context, config *Config) (sdktrace.SpanExporter, error) {
	cfg, err := newExporterConfig(config)
	if err != nil {
		return nil, err
	}

	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(config.Url),
		otlptracegrpc.WithHeaders(cfg.headers),
	}

	if !config.TLS.Enabled {
		opts = append(opts, otlptracegrpc.WithInsecure())
		return otlptracegrpc.New(ctx, opts...)
	}
	if cfg.tls != nil {
		opts = append(opts, otlptracegrpc.WithTLSCredentials(credentials.NewTLS(cfg.tls)))
	}

	return otlptracegrpc.New(ctx, opts...)
}

// newStdoutExporter creates a new stdout exporter
func newStdoutExporter(_ context.Context, _ *Config) (sdktrace.SpanExporter, error) {
	return stdouttrace.New(stdouttrace.WithPrettyPrint())
}

// newNoopExporter creates a new noop exporter
func newNoopExporter(_ context.Context, _ *Config) (sdktrace.SpanExporter, error) {
	return nil, nil
}

// exporterConfig contains the common configuration for the exporters
type exporterConfig struct {
	// headers contains the headers to send with spans
	headers map[string]string
	// tls contains the TLS configuration for the exporter
	tls *tls.Config
}

// newExporterConfig returns the common configuration for the exporters
func newExporterConfig(config *Config) (exporterConfig, error) {
	headers := map[string]string{}
	if config.Token != "" {
		headers["Authorization"] = fmt.Sprintf("Bearer %s", config.Token)
	}

	if config.TLS.Enabled {
		tlsCfg, err := getTLSConfig(config.TLS.CertPath)
		if err != nil {
			return exporterConfig{}, fmt.Errorf("failed to create TLS configuration: %w", err)
		}
		return exporterConfig{
			headers: headers,
			tls:     tlsCfg,
		}, nil
	}

	return exporterConfig{
		headers: headers,
		tls:     nil,
	}, nil
}

// FileOpener is the function used to open a file
type FileOpener func(string) (fs.File, error)

// openFile is the function used to open a file
var openFile FileOpener = func() FileOpener {
	return func(name string) (fs.File, error) {
		return os.Open(name) // #nosec G304 // How else to open the file?
	}
}()

func getTLSConfig(certFile string) (*tls.Config, error) {
	if certFile == "" {
		return nil, nil
	}

	file, err := openFile(certFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open certificate file: %w", err)
	}
	defer func() {
		err = errors.Join(err, file.Close())
	}()

	b, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file: %w", err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(b) {
		return nil, fmt.Errorf("failed to append certificate(s) from file: %s", certFile)
	}

	return &tls.Config{
		RootCAs:            pool,
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
	}, nil
}
