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

package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/caas-team/sparrow/internal/logger"
	"github.com/go-chi/chi/v5"
)

//go:generate moq -out api_moq.go . API
type API interface {
	Run(ctx context.Context) error
	Shutdown(ctx context.Context) error
	RegisterRoutes(ctx context.Context, routes ...Route) error
}

type api struct {
	server    *http.Server
	router    chi.Router
	tlsConfig TLSConfig
}

// Config is the configuration for the data API
type Config struct {
	ListeningAddress string    `yaml:"address" mapstructure:"address"`
	Tls              TLSConfig `yaml:"tls" mapstructure:"tls"`
}

type TLSConfig struct {
	Enabled  bool   `yaml:"enabled" mapstructure:"enabled"`
	CertPath string `yaml:"certPath" mapstructure:"certPath"`
	KeyPath  string `yaml:"keyPath" mapstructure:"keyPath"`
}

const (
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 30 * time.Second
)

func (a *Config) Validate() error {
	if a.ListeningAddress == "" {
		return fmt.Errorf("listening address cannot be empty")
	}
	if a.Tls.Enabled {
		if a.Tls.CertPath == "" {
			return fmt.Errorf("tls cert path cannot be empty")
		}
		if a.Tls.KeyPath == "" {
			return fmt.Errorf("tls key path cannot be empty")
		}
	}
	return nil
}

// New creates a new api
func New(cfg Config) API {
	r := chi.NewRouter()

	return &api{
		server:    &http.Server{Addr: cfg.ListeningAddress, Handler: r, ReadHeaderTimeout: readHeaderTimeout},
		router:    r,
		tlsConfig: cfg.Tls,
	}
}

// Run serves the data api
// Blocks until context is done
func (a *api) Run(ctx context.Context) error {
	log := logger.FromContext(ctx)
	cErr := make(chan error, 1)

	if len(a.router.Routes()) == 0 {
		return fmt.Errorf("failed serving API: no routes initialized")
	}

	// run http server in goroutine
	go func(cErr chan error) {
		defer close(cErr)
		log.Info("Serving Api", "addr", a.server.Addr)
		if a.tlsConfig.Enabled {
			if err := a.server.ListenAndServeTLS(a.tlsConfig.CertPath, a.tlsConfig.KeyPath); err != nil {
				log.Error("Failed to serve api", "error", err, "scheme", "https")
				cErr <- err
			}
			return
		}
		if err := a.server.ListenAndServe(); err != nil {
			log.Error("Failed to serve api", "error", err, "scheme", "http")
			cErr <- err
		}
	}(cErr)

	select {
	case <-ctx.Done():
		return fmt.Errorf("failed serving API: %w", ctx.Err())
	case err := <-cErr:
		if errors.Is(err, http.ErrServerClosed) || err == nil {
			log.Info("Api server closed")
			return nil
		}
		log.Error("Failed serving API", "error", err)
		return fmt.Errorf("failed serving API: %w", err)
	}
}

// Shutdown gracefully shuts down the api server
// Returns an error if an error is present in the context
// or if the server cannot be shut down
func (a *api) Shutdown(ctx context.Context) error {
	errC := ctx.Err()
	log := logger.FromContext(ctx)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	err := a.server.Shutdown(shutdownCtx)
	if err != nil {
		log.Error("Failed to shutdown api server", "error", err)
		return fmt.Errorf("failed shutting down API: %w", errors.Join(errC, err))
	}
	return errC
}

type Route struct {
	Path    string
	Method  string
	Handler http.HandlerFunc
}

// RegisterRoutes sets up all endpoint handlers for the given routes
func (a *api) RegisterRoutes(ctx context.Context, routes ...Route) error {
	a.router.Use(logger.Middleware(ctx))
	for _, route := range routes {
		if route.Method == "*" {
			a.router.HandleFunc(route.Path, route.Handler)
		} else {
			err := a.registerDefaultRoute(route)
			if err != nil {
				return err
			}
		}
	}

	// Handles requests with simple http ok
	// Required for global tarMan in checks
	a.router.Handle("/", OkHandler(ctx))

	return nil
}

// registerDefaultRoute registers a route using default HTTP methods such as GET, POST, etc.
// Returns an error if the method is unsupported.
func (a *api) registerDefaultRoute(route Route) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("unsupported method for %s: %s", route.Path, route.Method)
		}
	}()
	a.router.Method(route.Method, route.Path, route.Handler)
	return nil
}

// OkHandler returns a handler that will serve status ok
func OkHandler(ctx context.Context) http.Handler {
	log := logger.FromContext(ctx)

	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("ok"))
		if err != nil {
			log.Error("Could not write response", "error", err.Error())
		}
	})
}
