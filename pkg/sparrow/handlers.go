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

package sparrow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/api"
	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"
)

type encoder interface {
	Encode(v any) error
}

const urlParamCheckName = "checkName"

func (s *Sparrow) startupAPI(ctx context.Context) error {
	routes := []api.Route{
		{
			Path: "/openapi", Method: http.MethodGet,
			Handler: s.handleOpenAPI,
		},
		{
			Path: fmt.Sprintf("/v1/metrics/{%s}", urlParamCheckName), Method: http.MethodGet,
			Handler: s.handleCheckMetrics,
		},
		{
			Path: "/metrics", Method: "*",
			Handler: promhttp.HandlerFor(
				s.metrics.GetRegistry(),
				promhttp.HandlerOpts{Registry: s.metrics.GetRegistry()},
			).ServeHTTP,
		},
	}

	err := s.api.RegisterRoutes(ctx, routes...)
	if err != nil {
		logger.FromContext(ctx).Error("Error while registering routes", "error", err)
		return err
	}
	return s.api.Run(ctx)
}

func (s *Sparrow) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	oapi, err := s.controller.GenerateCheckSpecs(r.Context())
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
