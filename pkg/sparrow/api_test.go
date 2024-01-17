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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/caas-team/sparrow/pkg/api"
	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/config"
	"github.com/caas-team/sparrow/pkg/db"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"
)

func TestSparrow_register(t *testing.T) {
	r := chi.NewRouter()
	s := Sparrow{
		router:  r,
		metrics: NewMetrics(),
	}

	s.register(context.Background())

	expectedRoutes := []string{"/openapi.yaml", "/v1/metrics/{checkName}", "/checks/*", "/metrics", "/"}
	routes := r.Routes()
	for _, route := range expectedRoutes {
		found := 0

		for _, foundRoute := range routes {
			if foundRoute.Pattern == route {
				found += 1
				break
			}
		}
	}

	if len(expectedRoutes) != len(routes) {
		t.Errorf("Sparrow.register() = %v, want %v", len(routes), len(expectedRoutes))
	}
}

func TestSparrow_api_shutdownWhenContextCanceled(t *testing.T) {
	s := Sparrow{
		cfg:     &config.Config{Api: config.ApiConfig{ListeningAddress: ":8080"}},
		router:  chi.NewRouter(),
		metrics: NewMetrics(),
		server:  &http.Server{}, //nolint:gosec
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := s.api(ctx); !errors.Is(err, context.Canceled) {
		t.Error("Expected ErrApiContext")
	}
}

func testDb() *db.InMemory {
	d := db.NewInMemory()
	d.Save(checks.ResultDTO{Name: "alpha", Result: &checks.Result{Timestamp: time.Now(), Err: "", Data: 1}})
	d.Save(checks.ResultDTO{Name: "beta", Result: &checks.Result{Timestamp: time.Now(), Err: "", Data: 1}})

	return d
}

func chiRequest(r *http.Request, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("checkName", value)

	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	return r
}

func TestSparrow_getCheckMetrics(t *testing.T) {
	type args struct {
		w *httptest.ResponseRecorder
		r *http.Request
	}
	tests := []struct {
		name     string
		db       db.DB
		args     args
		want     []byte
		wantCode int
	}{
		{
			name: "no data", db: db.NewInMemory(),

			args: args{w: httptest.NewRecorder(), r: chiRequest(httptest.NewRequest(http.MethodGet, "/v1/metrics/alpha", bytes.NewBuffer([]byte{})), "alpha")}, wantCode: http.StatusNotFound, want: []byte(http.StatusText(http.StatusNotFound)),
		},
		{name: "bad request data", db: db.NewInMemory(), args: args{w: httptest.NewRecorder(), r: chiRequest(httptest.NewRequest(http.MethodGet, "/v1/metrics/", bytes.NewBuffer([]byte{})), "")}, wantCode: http.StatusBadRequest, want: []byte(http.StatusText(http.StatusBadRequest))},
		{name: "has data", db: testDb(), args: args{w: httptest.NewRecorder(), r: chiRequest(httptest.NewRequest(http.MethodGet, "/v1/metrics/alpha", bytes.NewBuffer([]byte{})), "alpha")}, wantCode: http.StatusOK, want: []byte(`{"name":"alpha","result":{"timestamp":"0001-01-01T00:00:00Z","err":"","data":1}}`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Sparrow{
				db: tt.db,
			}
			s.getCheckMetrics(tt.args.w, tt.args.r)
			resp := tt.args.w.Result() //nolint:bodyclose
			body, _ := io.ReadAll(resp.Body)

			if tt.wantCode == http.StatusOK {
				if tt.wantCode != resp.StatusCode {
					t.Errorf("Sparrow.getCheckMetrics() = %v, want %v", resp.StatusCode, tt.wantCode)
				}
				var got checks.ResultDTO
				var want checks.ResultDTO
				err := json.Unmarshal(body, &got)
				if err != nil {
					t.Error("Expected valid json")
				}
				err = json.Unmarshal(tt.want, &want)
				if err != nil {
					t.Error("Expected valid json")
				}

				if reflect.DeepEqual(got, want) {
					t.Errorf("Sparrow.getCheckMetrics() = %v, want %v", got, want)
				}
			} else {
				if tt.wantCode != resp.StatusCode {
					t.Errorf("Sparrow.getCheckMetrics() = %v, want %v", resp.StatusCode, tt.wantCode)
				}
				if !reflect.DeepEqual(body, tt.want) {
					t.Errorf("Sparrow.getCheckMetrics() = %v, want %v", body, tt.want)
				}
			}
		})
	}
}

func addRouteParams(r *http.Request, values map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range values {
		rctx.URLParams.Add(k, v)
	}

	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestSparrow_handleChecks(t *testing.T) {
	type route struct {
		Method  string
		Path    string
		Handler http.HandlerFunc
	}
	type args struct {
		w *httptest.ResponseRecorder
		r *http.Request
	}
	tests := []struct {
		name     string
		rTree    *api.RoutingTree
		routes   []route
		args     args
		want     []byte
		wantCode int
	}{
		{
			name:     "no check handlers",
			rTree:    api.NewRoutingTree(),
			args:     args{w: httptest.NewRecorder(), r: httptest.NewRequest(http.MethodGet, "/v1/notfound", bytes.NewBuffer([]byte{}))},
			wantCode: http.StatusNotFound,
			want:     []byte(http.StatusText(http.StatusNotFound)),
		},
		{
			name:     "has check handlers",
			rTree:    api.NewRoutingTree(),
			routes:   []route{{Method: http.MethodGet, Path: "/v1/test", Handler: func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("test")) }}}, //nolint:errcheck
			args:     args{w: httptest.NewRecorder(), r: addRouteParams(httptest.NewRequest(http.MethodGet, "/v1/test", bytes.NewBuffer([]byte{})), map[string]string{"*": "/v1/test"})},
			wantCode: http.StatusOK,
			want:     []byte("test"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Sparrow{
				routingTree: tt.rTree,
			}

			for _, route := range tt.routes {
				s.routingTree.Add(route.Method, route.Path, route.Handler)
			}
			s.handleChecks(tt.args.w, tt.args.r)
			resp := tt.args.w.Result() //nolint:bodyclose
			body, _ := io.ReadAll(resp.Body)

			if tt.wantCode != resp.StatusCode {
				t.Errorf("Sparrow.handleChecks() = %v, want %v", resp.StatusCode, tt.wantCode)
			}
			if !reflect.DeepEqual(body, tt.want) {
				t.Errorf("Sparrow.handleChecks() = %v, want %v", body, tt.want)
			}
		})
	}
}

func TestSparrow_getOpenapi(t *testing.T) {
	s := Sparrow{}

	type args struct {
		request  *http.Request
		response *httptest.ResponseRecorder
		headers  map[string]string
	}

	type test struct {
		name    string
		args    args
		decoder func(*httptest.ResponseRecorder) error
	}

	tests := []test{
		{name: "yaml is default", args: args{request: httptest.NewRequest(http.MethodGet, "/openapi.yaml", bytes.NewBuffer([]byte{})), response: httptest.NewRecorder(), headers: map[string]string{}}, decoder: func(rr *httptest.ResponseRecorder) error {
			b := rr.Body.Bytes()
			return yaml.Unmarshal(b, &openapi3.T{})
		}},
		{name: "set json via accept header", args: args{request: httptest.NewRequest(http.MethodGet, "/openapi.yaml", bytes.NewBuffer([]byte{})), response: httptest.NewRecorder(), headers: map[string]string{"Accept": "application/json"}}, decoder: func(rr *httptest.ResponseRecorder) error {
			b := rr.Body.Bytes()
			return json.Unmarshal(b, &openapi3.T{})
		}},
		{name: "set yaml via accept header", args: args{request: httptest.NewRequest(http.MethodGet, "/openapi.yaml", bytes.NewBuffer([]byte{})), response: httptest.NewRecorder(), headers: map[string]string{"Accept": "text/yaml"}}, decoder: func(rr *httptest.ResponseRecorder) error {
			b := rr.Body.Bytes()
			return yaml.Unmarshal(b, &openapi3.T{})
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for h, v := range tt.args.headers {
				tt.args.request.Header.Add(h, v)
			}

			s.getOpenapi(tt.args.response, tt.args.request)

			if err := tt.decoder(tt.args.response); err != nil {
				t.Errorf("failed to decode response Sparrow.getOpenapi() = %v", err)
			}

			if tt.args.response.Code != http.StatusOK {
				t.Errorf("Sparrow.getOpenapi() = %v, want %v", tt.args.response.Code, http.StatusOK)
			}
		})
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
