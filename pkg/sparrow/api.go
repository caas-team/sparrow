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

func (s *Sparrow) register(ctx context.Context) {
	// TODO register handlers
	// GET /openapi
	s.router.Get("/openapi", s.getOpenapi)
	// GET /v1/metrics/*checks
	s.router.Get(fmt.Sprintf("/v1/metrics/{%s}", urlParamCheckName), s.getCheckMetrics)
	// * /checks/*
	s.router.HandleFunc("/checks/*", s.handleChecks)
	s.router.Use(logger.Middleware(ctx))
}

// Serves the data api.
//
// Blocks until context is done
func (s *Sparrow) api(ctx context.Context) error {
	cErr := make(chan error)
	s.register(ctx)
	server := http.Server{Addr: s.cfg.Api.Port, Handler: s.router}

	// run http server in goroutine
	go func(cErr chan error) {
		defer close(cErr)
		if err := server.ListenAndServe(); err != nil {
			cErr <- err
		}
	}(cErr)

	select {
	case <-ctx.Done():
		if ctx.Err() != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()
			server.Shutdown(shutdownCtx)
			return fmt.Errorf("%w: %w", ErrApiContext, ctx.Err())
		}
	case err := <-cErr:
		if err == http.ErrServerClosed || err == nil {
			return nil
		}
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
	Extensions: make(map[string]interface{}),
	Components: &openapi3.Components{
		Schemas: make(openapi3.Schemas),
	},
	Servers: openapi3.Servers{},
}

func (s *Sparrow) Openapi() (openapi3.T, error) {
	doc := oapiBoilerplate
	for name, c := range s.checks {
		ref, err := c.Schema()
		if err != nil {
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
	name := chi.URLParam(r, urlParamCheckName)
	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(http.StatusText(http.StatusBadRequest)))
		return
	}
	res, ok := s.db.Get(name)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(http.StatusText(http.StatusNotFound)))
		return
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(res); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
		return
	}
	w.Header().Add("Content-Type", "application/json")
}

func (s *Sparrow) getOpenapi(w http.ResponseWriter, r *http.Request) {
	oapi, err := s.Openapi()
	if err != nil {
		// TODO use logger
		fmt.Println("failed to create openapi")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
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
		// TODO use logger
		fmt.Println("failed to marshal openapi")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
		return
	}
}

func (s *Sparrow) handleChecks(w http.ResponseWriter, r *http.Request) {
	method := r.Method
	path := r.URL.Path

	handler, ok := s.routingTree.Get(method, path)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(http.StatusText(http.StatusNotFound)))
		return
	}

	handler(w, r)
}

var ErrServeApi = errors.New("failed to serve api")
var ErrApiContext = errors.New("api context cancelled")
var ErrCreateOpenapiSchema = errors.New("failed to get schema for check")

var urlParamCheckName = "checkName"
