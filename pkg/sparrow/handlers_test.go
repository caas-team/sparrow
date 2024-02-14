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
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/checks/runtime"
	"github.com/caas-team/sparrow/pkg/db"
	"github.com/caas-team/sparrow/pkg/healthz"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"
)

func TestSparrow_handleOpenAPI(t *testing.T) {
	s := Sparrow{
		controller: &ChecksController{
			checks: runtime.Checks{},
		},
	}

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

			s.handleOpenAPI(tt.args.response, tt.args.request)

			if err := tt.decoder(tt.args.response); err != nil {
				t.Errorf("failed to decode response Sparrow.getOpenapi() = %v", err)
			}

			if tt.args.response.Code != http.StatusOK {
				t.Errorf("Sparrow.getOpenapi() = %v, want %v", tt.args.response.Code, http.StatusOK)
			}
		})
	}
}

func TestSparrow_handleCheckMetrics(t *testing.T) {
	tests := []struct {
		name     string
		want     []byte
		wantCode int
	}{
		{
			name:     "no data",
			wantCode: http.StatusNotFound,
			want:     []byte(http.StatusText(http.StatusNotFound)),
		},
		{
			name:     "bad request data",
			wantCode: http.StatusBadRequest,
			want:     []byte(http.StatusText(http.StatusBadRequest)),
		},
		{
			name:     "has data",
			wantCode: http.StatusOK,
			want:     []byte(`{"name":"alpha","result":{"timestamp":"0001-01-01T00:00:00Z","err":"","data":1}}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Sparrow{
				db: db.NewInMemory(),
			}
			if tt.wantCode == http.StatusOK {
				s.db = testDb()
			}

			w := httptest.NewRecorder()
			r := chiRequest(httptest.NewRequest(http.MethodGet, "/v1/metrics/alpha", bytes.NewBuffer([]byte{})), "alpha")
			if tt.wantCode == http.StatusBadRequest {
				r = chiRequest(httptest.NewRequest(http.MethodGet, "/v1/metrics/", bytes.NewBuffer([]byte{})), "")
			}

			s.handleCheckMetrics(w, r)
			resp := w.Result() //nolint:bodyclose
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

func TestSparrow_handleHealthz(t *testing.T) {
	tests := []struct {
		name     string
		wantCode int
	}{
		{
			name:     "healthy",
			wantCode: http.StatusOK,
		},
		{
			name:     "unhealthy",
			wantCode: http.StatusServiceUnavailable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Sparrow{
				controller: NewChecksController(db.NewInMemory(), NewMetrics()),
				checker: &healthz.CheckerMock{
					CheckOverallHealthFunc: func(ctx context.Context, cks []checks.Check) bool {
						return tt.wantCode == http.StatusOK
					},
				},
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/healthz", bytes.NewBuffer([]byte{}))

			s.handleHealthz(w, r)
			resp := w.Result() //nolint:bodyclose // close is defer below
			defer func(b io.ReadCloser) {
				err := b.Close()
				if err != nil {
					t.Fatalf("Failed to close response body: %v", err)
				}
			}(resp.Body)
			if tt.wantCode != resp.StatusCode {
				t.Errorf("Sparrow.handleHealthz() Status = %v, want %v", resp.StatusCode, tt.wantCode)
			}
		})
	}
}

func chiRequest(r *http.Request, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("checkName", value)

	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	return r
}

func testDb() *db.InMemory {
	d := db.NewInMemory()
	d.Save(checks.ResultDTO{Name: "alpha", Result: &checks.Result{Timestamp: time.Now(), Data: 1}})
	d.Save(checks.ResultDTO{Name: "beta", Result: &checks.Result{Timestamp: time.Now(), Data: 1}})

	return d
}
