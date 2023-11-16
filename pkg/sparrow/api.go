package sparrow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v3"
)

func (s *Sparrow) register() {
	// TODO register handlers
	// GET /openapi
	s.router.Get("/openapi.yaml", s.getOpenapi)

	// GET /v1/metrics/*checks
}

// Serves the data api.
//
// Blocks until context is done
func (s *Sparrow) api(ctx context.Context) error {
	// TODO
	// implement a way for checks to dynamically register and deregister at runtime
	// easiest way is probably to use a map and write a handler that just does a plain
	// old map.Get(path) and returns 404 if not found or call the checks handlers

	cErr := make(chan error)
	s.register()
	server := http.Server{Addr: ":8081", Handler: s.router}

	// run http server in goroutine
	go func(cErr chan error) {
		if err := server.ListenAndServe(); err != nil {
			cErr <- err
		}
		close(cErr)
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

	var marshaler Encoder
	switch mime {
	case "application/json":
		marshaler = json.NewEncoder(w)
	case "application/yaml":
		marshaler = yaml.NewEncoder(w)
	default:
		marshaler = yaml.NewEncoder(w)
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

var ErrServeApi = errors.New("failed to serve api")
var ErrApiContext = errors.New("api context cancelled")
var ErrCreateOpenapiSchema = errors.New("failed to get schema for check")

type Encoder interface {
	Encode(v any) error
}
