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

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"
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
		want          *RuntimeConfig
		wantErr       bool
	}{
		{
			name: "Get runtime configuration",
			cfg: &Config{
				Loader: LoaderConfig{
					Type:     "http",
					Interval: time.Second,
				},
			},
			httpResponder: httpResponder{
				statusCode: 200,
				response:   httpmock.File("testdata/config.yaml").String(),
			},
			want: &RuntimeConfig{
				Checks: map[string]any{
					"testCheck1": map[string]any{
						"enabled": true,
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
					http: HttpLoaderConfig{
						token: "SECRET",
					},
				},
			},
			httpResponder: httpResponder{
				statusCode: 200,
				response:   httpmock.File("testdata/config.yaml").String(),
			},
			want: &RuntimeConfig{
				Checks: map[string]any{
					"testCheck1": map[string]any{
						"enabled": true,
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
			want:    nil,
			wantErr: true,
		},
		{
			name: "Get runtime configuration payload not yaml",
			cfg: &Config{
				Loader: LoaderConfig{
					Type:     "http",
					Interval: time.Second,
				},
			},
			httpResponder: httpResponder{
				statusCode: 200,
				response:   `this is not yaml`,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := "https://api.test.com/test"
			httpmock.RegisterResponder("GET", endpoint,
				func(req *http.Request) (*http.Response, error) {
					if tt.cfg.Loader.http.token != "" {
						require.Equal(t, req.Header.Get("Authorization"), fmt.Sprintf("Bearer %s", tt.cfg.Loader.http.token))
						fmt.Println("TOKEN tested")
					}
					resp, _ := httpmock.NewStringResponder(tt.httpResponder.statusCode, tt.httpResponder.response)(req)
					return resp, nil
				},
			)

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()

			handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
			gl := &HttpLoader{
				log:        slog.New(handler),
				cfg:        tt.cfg,
				cCfgChecks: make(chan<- map[string]any),
			}
			gl.cfg.Loader.http.url = endpoint

			got, err := gl.GetRuntimeConfig(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("HttpLoader.GetRuntimeConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HttpLoader.GetRuntimeConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
