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

	"github.com/caas-team/sparrow/pkg/api"
	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/checks/types"
	"github.com/caas-team/sparrow/pkg/config"
	"github.com/caas-team/sparrow/pkg/db"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"
)

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
	type fields struct {
		checks      map[string]checks.Check
		resultFanIn map[string]chan types.Result
		cResult     chan types.ResultDTO
		loader      config.Loader
		cfg         *config.Config
		cCfgChecks  chan map[string]any
		api         *api.API
		db          db.DB
	}

	type args struct {
		w *httptest.ResponseRecorder
		r *http.Request
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		want     []byte
		wantCode int
	}{
		{name: "no data", fields: fields{db: db.NewInMemory()}, args: args{w: httptest.NewRecorder(), r: chiRequest(httptest.NewRequest(http.MethodGet, "/v1/metrics/alpha", bytes.NewBuffer([]byte{})), "alpha")}, wantCode: http.StatusNotFound, want: []byte(http.StatusText(http.StatusNotFound))},
		{name: "bad request data", fields: fields{db: db.NewInMemory()}, args: args{w: httptest.NewRecorder(), r: chiRequest(httptest.NewRequest(http.MethodGet, "/v1/metrics/", bytes.NewBuffer([]byte{})), "")}, wantCode: http.StatusBadRequest, want: []byte(http.StatusText(http.StatusBadRequest))},
		{name: "has data", fields: fields{db: testDb()}, args: args{w: httptest.NewRecorder(), r: chiRequest(httptest.NewRequest(http.MethodGet, "/v1/metrics/alpha", bytes.NewBuffer([]byte{})), "alpha")}, wantCode: http.StatusOK, want: []byte(`{"name":"alpha","result":{"timestamp":"0001-01-01T00:00:00Z","err":"","data":1}}`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Sparrow{
				config: tt.fields.cfg,
				db:     tt.fields.db,
				api:    tt.fields.api,
				loader: tt.fields.loader,
				checksImpl: checksImpl{
					checks:      tt.fields.checks,
					resultFanIn: tt.fields.resultFanIn,
					cCfgChecks:  tt.fields.cCfgChecks,
					cResult:     tt.fields.cResult,
				},
			}

			s.handleCheckMetrics(tt.args.w, tt.args.r)
			resp := tt.args.w.Result() //nolint:bodyclose
			body, _ := io.ReadAll(resp.Body)

			if tt.wantCode == http.StatusOK {
				if tt.wantCode != resp.StatusCode {
					t.Errorf("Sparrow.getCheckMetrics() = %v, want %v", resp.StatusCode, tt.wantCode)
				}
				var got types.ResultDTO
				var want types.ResultDTO
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

func chiRequest(r *http.Request, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("checkName", value)

	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	return r
}

func testDb() *db.InMemory {
	d := db.NewInMemory()
	d.Save(types.ResultDTO{Name: "alpha", Result: &types.Result{Timestamp: time.Now(), Err: "", Data: 1}})
	d.Save(types.ResultDTO{Name: "beta", Result: &types.Result{Timestamp: time.Now(), Err: "", Data: 1}})

	return d
}
