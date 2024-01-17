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

package config

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/caas-team/sparrow/pkg/checks/health"
	"github.com/caas-team/sparrow/pkg/checks/types"

	"github.com/stretchr/testify/require"

	"github.com/caas-team/sparrow/internal/logger"
	"github.com/jarcoal/httpmock"
)

func TestHttpLoader_GetRuntimeConfig(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	type httpResponder struct {
		statusCode int
		response   string
	}
	tests := []struct {
		name          string
		cfg           *Config
		httpResponder httpResponder
		want          *types.RuntimeConfig
		wantErr       bool
	}{
		{
			name: "Get runtime configuration",
			cfg: &Config{
				Loader: LoaderConfig{
					Type:     "http",
					Interval: 1 * time.Second,
				},
			},
			httpResponder: httpResponder{
				statusCode: 200,
				response:   httpmock.File("testdata/config.yaml").String(),
			},
			want: &types.RuntimeConfig{
				Checks: types.Checks{
					Health: &health.Config{
						Targets:  []string{"http://localhost:8080/health"},
						Interval: 1 * time.Second,
						Timeout:  1 * time.Second,
					},
				},
			},
		},
		{
			name: "Get runtime configuration with auth",
			cfg: &Config{
				Loader: LoaderConfig{
					Type:     "http",
					Interval: time.Second,
					Http: HttpLoaderConfig{
						Token: "SECRET",
					},
				},
			},
			httpResponder: httpResponder{
				statusCode: 200,
				response:   httpmock.File("testdata/config.yaml").String(),
			},
			want: &types.RuntimeConfig{
				Checks: types.Checks{
					Health: &health.Config{
						Targets:  []string{"http://localhost:8080/health"},
						Interval: 1 * time.Second,
						Timeout:  1 * time.Second,
					},
				},
			},
		},
		{
			name: "Get runtime configuration with statuscode 400",
			cfg: &Config{
				Loader: LoaderConfig{
					Type:     "http",
					Interval: time.Second,
				},
			},
			httpResponder: httpResponder{
				statusCode: 400,
				response:   httpmock.File("testdata/config.yaml").String(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := "https://api.test.com/test"
			httpmock.RegisterResponder("GET", endpoint,
				func(req *http.Request) (*http.Response, error) {
					if tt.cfg.Loader.Http.Token != "" {
						require.Equal(t, req.Header.Get("Authorization"), fmt.Sprintf("Bearer %s", tt.cfg.Loader.Http.Token))
						fmt.Println("TOKEN tested")
					}
					resp, _ := httpmock.NewStringResponder(tt.httpResponder.statusCode, tt.httpResponder.response)(req)
					return resp, nil
				},
			)

			handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
			ctx := logger.IntoContext(context.Background(), logger.NewLogger(handler).WithGroup("httpLoader-test"))
			ctx, cancel := context.WithTimeout(ctx, time.Second*10)
			defer cancel()

			gl := &HttpLoader{
				cfg:         tt.cfg,
				cConfigCfgs: make(chan<- types.RuntimeConfig, 1),
				client: &http.Client{
					Timeout: tt.cfg.Loader.Http.Timeout,
				},
			}
			gl.cfg.Loader.Http.Url = endpoint

			got, err := gl.GetRuntimeConfig(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("HttpLoader.GetRuntimeConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HttpLoader.GetRuntimeConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
