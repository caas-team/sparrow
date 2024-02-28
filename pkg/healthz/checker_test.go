package healthz

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/jarcoal/httpmock"
)

func TestChecker_isMetricsHealthy(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	ctx := context.Background()
	tests := []struct {
		name      string
		responder httpmock.Responder
		want      bool
	}{
		{
			name:      "healthy",
			responder: httpmock.NewStringResponder(http.StatusOK, http.StatusText(http.StatusOK)),
			want:      true,
		},
		{
			name:      "unhealthy",
			responder: httpmock.NewStringResponder(http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable)),
			want:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpmock.RegisterResponder(http.MethodGet, "http://:8080/metrics", tt.responder)
			c := checker{
				addr:   ":8080",
				client: &http.Client{},
			}

			if got := c.isMetricsHealthy(ctx); got != tt.want {
				t.Errorf("Checker.AreMetricsHealthy() = %v, want %v", got, tt.want)
			}
			httpmock.Reset()
		})
	}
}

func TestChecker_areChecksHealthy(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	ctx := context.Background()
	tests := []struct {
		name      string
		checks    []checks.Check
		responder []httpmock.Responder
		want      bool
	}{
		{
			name: "healthy",
			checks: []checks.Check{
				&checks.CheckMock{
					NameFunc: func() string {
						return "check1"
					},
				},
			},
			responder: []httpmock.Responder{
				httpmock.NewStringResponder(http.StatusOK, http.StatusText(http.StatusOK)),
			},
			want: true,
		},
		{
			name: "unhealthy",
			checks: []checks.Check{
				&checks.CheckMock{
					NameFunc: func() string {
						return "check1"
					},
				},
				&checks.CheckMock{
					NameFunc: func() string {
						return "check2"
					},
				},
			},
			responder: []httpmock.Responder{
				httpmock.NewStringResponder(http.StatusNotFound, http.StatusText(http.StatusNotFound)),
				httpmock.NewStringResponder(http.StatusNotFound, http.StatusText(http.StatusNotFound)),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i, c := range tt.checks {
				httpmock.RegisterResponder(http.MethodGet, fmt.Sprintf("http://:8080/v1/metrics/%s", c.Name()), tt.responder[i])
			}

			c := checker{
				addr:   ":8080",
				client: &http.Client{},
			}

			if got := c.areChecksHealthy(ctx, tt.checks); got != tt.want {
				t.Errorf("Checker.areChecksHealthy() = %v, want %v", got, tt.want)
			}
			httpmock.Reset()
		})
	}
}
