// sparrow
// (C) 2023, Deutsche Telekom IT GmbH
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

package sparrow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/caas-team/sparrow/internal/logger"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"
)

type encoder interface {
	Encode(v any) error
}

const (
	urlParamCheckName = "checkName"
	readHeaderTimeout = time.Second * 5
	shutdownTimeout   = time.Second * 5
)

var (
	ErrServeApi            = errors.New("failed to serve api")
	ErrApiContext          = errors.New("api context canceled")
	ErrCreateOpenapiSchema = errors.New("failed to get schema for check")
)

func (s *Sparrow) register(ctx context.Context) {
	s.router.Use(logger.Middleware(ctx))

	// Handles OpenApi spec
	s.router.Get("/openapi", s.getOpenapi)
	// Handles public user facing json api
	s.router.Get(fmt.Sprintf("/v1/metrics/{%s}", urlParamCheckName), s.getCheckMetrics)
	// Handles internal api
	// handlers are (de)registered by the checks themselves
	s.router.HandleFunc("/checks/*", s.handleChecks)
}

// Serves the data api.
//
// Blocks until context is done
func (s *Sparrow) api(ctx context.Context) error {
	log := logger.FromContext(ctx).WithGroup("api")
	cErr := make(chan error, 1)
	s.register(ctx)

	server := http.Server{Addr: s.cfg.Api.ListeningAddress, Handler: s.router, ReadHeaderTimeout: readHeaderTimeout}

	// run http server in goroutine
	go func(cErr chan error) {
		defer close(cErr)
		log.Info("Serving Api", "addr", s.cfg.Api.ListeningAddress)
		if err := server.ListenAndServe(); err != nil {
			log.Error("Failed to serve api", "error", err)
			cErr <- err
		}
	}(cErr)

	select {
	case <-ctx.Done():
		if ctx.Err() != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
			defer cancel()
			err := server.Shutdown(shutdownCtx)
			if err != nil {
				log.Error("Failed to shutdown api server", "error", err)
			}
			log.Error("Api context canceled", "error", ctx.Err())
			return fmt.Errorf("%w: %w", ErrApiContext, ctx.Err())
		}
	case err := <-cErr:
		if errors.Is(err, http.ErrServerClosed) || err == nil {
			log.Info("Api server closed")
			return nil
		}
		log.Error("failed to serve api", "error", err)
		return fmt.Errorf("%w: %w", ErrServeApi, err)
	}

	return nil
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

func (s *Sparrow) Openapi(ctx context.Context) (openapi3.T, error) {
	log := logger.FromContext(ctx)
	doc := oapiBoilerplate
	for name, c := range s.checks {
		ref, err := c.Schema()
		if err != nil {
			log.Error("failed to get schema for check", "error", err)
			return openapi3.T{}, fmt.Errorf("%w %s: %w", ErrCreateOpenapiSchema, name, err)
		}

		routeDesc := fmt.Sprintf("Returns the performance data for check %s", name)
		bodyDesc := fmt.Sprintf("Metrics for check %s", name)
		doc.Paths["/v1/metrics/"+name] = &openapi3.PathItem{
			Description: name,
			Get: &openapi3.Operation{
				Description: routeDesc,
				Tags:        []string{"Metrics", name},
				Responses: openapi3.Responses{
					"200": &openapi3.ResponseRef{
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

func (s *Sparrow) getCheckMetrics(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	name := chi.URLParam(r, urlParamCheckName)
	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte(http.StatusText(http.StatusBadRequest)))
		if err != nil {
			log.Error("Failed to write response", "error", err)
		}
		return
	}
	res, ok := s.db.Get(name)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		_, err := w.Write([]byte(http.StatusText(http.StatusNotFound)))
		if err != nil {
			log.Error("Failed to write response", "error", err)
		}
		return
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	if err := enc.Encode(res); err != nil {
		log.Error("failed to encode response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		_, err = w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
		if err != nil {
			log.Error("Failed to write response", "error", err)
		}
		return
	}
	w.Header().Add("Content-Type", "application/json")
}

func (s *Sparrow) getOpenapi(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	oapi, err := s.Openapi(r.Context())
	if err != nil {
		log.Error("failed to create openapi", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		_, err = w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
		if err != nil {
			log.Error("Failed to write response", "error", err)
		}
		return
	}

	mime := r.Header.Get("Accept")

	var marshaler encoder
	switch mime {
	case "application/json":
		marshaler = json.NewEncoder(w)
		w.Header().Add("Content-Type", "application/json")
	default:
		marshaler = yaml.NewEncoder(w)
		w.Header().Add("Content-Type", "text/yaml")
	}

	err = marshaler.Encode(oapi)
	if err != nil {
		log.Error("failed to marshal openapi", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		_, err = w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
		if err != nil {
			log.Error("Failed to write response", "error", err)
		}
		return
	}
}

// handleChecks handles all requests to /checks/*
// It delegates the request to the corresponding check handler
// Returns a 404 if no handler is registered for the request
func (s *Sparrow) handleChecks(w http.ResponseWriter, r *http.Request) {
	method := r.Method
	path := chi.URLParam(r, "*")
	log := logger.FromContext(r.Context())

	handler, ok := s.routingTree.Get(method, path)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		_, err := w.Write([]byte(http.StatusText(http.StatusNotFound)))
		if err != nil {
			log.Error("Failed to write response", "error", err)
		}
		return
	}

	handler(w, r)
}
