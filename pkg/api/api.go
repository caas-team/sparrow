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
	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/config"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
)

type API interface {
	Run(ctx context.Context) error
	Shutdown(ctx context.Context) error
	RegisterRoutes(ctx context.Context, routes ...Route) error
}

type api struct {
	server *http.Server
	router chi.Router
}

const (
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 30 * time.Second
)

// New creates a new api
func New(cfg config.ApiConfig) API {
	r := chi.NewRouter()
	return &api{
		server: &http.Server{Addr: cfg.ListeningAddress, Handler: r, ReadHeaderTimeout: readHeaderTimeout},
		router: r,
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
		if err := a.server.ListenAndServe(); err != nil {
			log.Error("Failed to serve api", "error", err)
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
		switch route.Method {
		case http.MethodGet:
			a.router.Get(route.Path, route.Handler)
		case http.MethodPost:
			a.router.Post(route.Path, route.Handler)
		case http.MethodPut:
			a.router.Put(route.Path, route.Handler)
		case http.MethodDelete:
			a.router.Delete(route.Path, route.Handler)
		case http.MethodPatch:
			a.router.Patch(route.Path, route.Handler)
		case "Handle":
			a.router.Handle(route.Path, route.Handler)
		case "HandleFunc":
			a.router.HandleFunc(route.Path, route.Handler)
		default:
			return fmt.Errorf("unsupported method for %s: %s", route.Path, route.Method)
		}
	}

	// Handles requests with simple http ok
	// Required for global tarMan in checks
	a.router.Handle("/", okHandler(ctx))

	return nil
}

// okHandler returns a handler that will serve status ok
func okHandler(ctx context.Context) http.Handler {
	log := logger.FromContext(ctx)

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("ok"))
		if err != nil {
			log.Error("Could not write response", "error", err.Error())
		}
	})
}

var oapiBoilerplate = openapi3.T{
	// this object should probably be user defined
	OpenAPI: "3.0.0",
	Info: &openapi3.Info{
		Title:       "Sparrow Metrics API",
		Description: "Serves metrics collected by sparrows checks",
		Contact: &openapi3.Contact{
			URL:   "https://caas.telekom.de",
			Email: "caas-request@telekom.de",
			Name:  "CaaS Team",
		},
	},
	Paths:      make(openapi3.Paths),
	Extensions: make(map[string]any),
	Components: &openapi3.Components{
		Schemas: make(openapi3.Schemas),
	},
	Servers: openapi3.Servers{},
}

// GenerateCheckSpecs generates the OpenAPI specifications for the given checks
// Returns the complete OpenAPI specification for all checks
func GenerateCheckSpecs(ctx context.Context, cks map[string]checks.Check) (openapi3.T, error) {
	log := logger.FromContext(ctx)
	doc := oapiBoilerplate
	for name, c := range cks {
		ref, err := c.Schema()
		if err != nil {
			log.Error("Failed to get schema for check", "name", name, "error", err)
			return openapi3.T{}, &ErrCreateOpenapiSchema{name: name, err: err}
		}

		routeDesc := fmt.Sprintf("Returns the performance data for check %s", name)
		bodyDesc := fmt.Sprintf("Metrics for check %s", name)
		doc.Paths["/v1/metrics/"+name] = &openapi3.PathItem{
			Description: name,
			Get: &openapi3.Operation{
				Description: routeDesc,
				Tags:        []string{"Metrics", name},
				Responses: openapi3.Responses{
					fmt.Sprint(http.StatusOK): &openapi3.ResponseRef{
						Value: &openapi3.Response{
							Description: &bodyDesc,
							Content:     openapi3.NewContentWithSchemaRef(ref, []string{"application/json"}),
						},
					},
				},
			},
		}
	}

	return doc, nil
}
