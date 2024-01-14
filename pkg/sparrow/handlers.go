package sparrow

import (
	"encoding/json"
	"net/http"

	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/api"
	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"
)

type encoder interface {
	Encode(v any) error
}

const urlParamCheckName = "checkName"

func (s *Sparrow) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	oapi, err := api.OpenAPI(r.Context(), s.checks)
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

	handler, ok := s.api.RoutingTree.Get(method, path)
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

func (s *Sparrow) handleCheckMetrics(w http.ResponseWriter, r *http.Request) {
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
