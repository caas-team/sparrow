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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestAPI_RegisterRoutes(t *testing.T) {
	tests := []struct {
		name   string
		routes []Route
		want   []struct {
			method string
			path   string
			status int
		}
		wantErr bool
	}{
		{
			name: "Register one route",
			routes: []Route{
				{Path: "/get", Method: http.MethodGet, Handler: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}},
			},
			want: []struct {
				method string
				path   string
				status int
			}{
				{method: http.MethodGet, path: "/get", status: http.StatusOK},
			},
			wantErr: false,
		},
		{
			name: "Register multiple routes",
			routes: []Route{
				{Path: "/get", Method: http.MethodGet, Handler: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}},
				{Path: "/post", Method: http.MethodPost, Handler: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusCreated)
				}},
				{Path: "/put", Method: http.MethodPut, Handler: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}},
				{Path: "/delete", Method: http.MethodDelete, Handler: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNoContent)
				}},
				{Path: "/patch", Method: http.MethodPatch, Handler: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}},
				{Path: "/handle", Method: "Handle", Handler: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}},
				{Path: "/handlefunc", Method: "HandleFunc", Handler: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}},
			},
			want: []struct {
				method string
				path   string
				status int
			}{
				{method: http.MethodGet, path: "/get", status: http.StatusOK},
				{method: http.MethodPost, path: "/post", status: http.StatusCreated},
				{method: http.MethodPut, path: "/put", status: http.StatusOK},
				{method: http.MethodDelete, path: "/delete", status: http.StatusNoContent},
				{method: http.MethodPatch, path: "/patch", status: http.StatusOK},
				{method: http.MethodGet, path: "/handle", status: http.StatusOK},
				{method: http.MethodGet, path: "/handlefunc", status: http.StatusOK},
			},
			wantErr: false,
		},
		{
			name: "Unsupported Method",
			routes: []Route{
				{Path: "/unknown", Method: "unknown", Handler: func(w http.ResponseWriter, r *http.Request) {}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := api{
				server: &http.Server{}, //nolint:gosec
				router: chi.NewRouter(),
			}

			err := a.RegisterRoutes(context.Background(), tt.routes...)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterRoutes() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				for _, req := range tt.want {
					request := httptest.NewRequest(req.method, req.path, http.NoBody)
					recorder := httptest.NewRecorder()

					a.router.ServeHTTP(recorder, request)

					if recorder.Code != req.status {
						t.Errorf("Unexpected status code for %s %s. Got %d, wanted %d", req.method, req.path, recorder.Code, req.status)
					}
				}
			}
		})
	}
}

func TestAPI_ShutdownWhenContextCanceled(t *testing.T) {
	a := api{
		router: chi.NewRouter(),
		server: &http.Server{}, //nolint:gosec
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := a.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Error("Expected ErrApiContext")
	}
}

func Test_okHandler(t *testing.T) {
	ctx := context.Background()

	req, err := http.NewRequestWithContext(ctx, "GET", "/okHandler", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := okHandler(ctx)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := "ok"
	if rr.Body.String() != expected {
		t.Errorf("Handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}
