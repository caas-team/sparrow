package e2e

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/test"
)

const (
	checkInterval = 10 * time.Second
	checkTimeout  = 10 * time.Second
)

func TestE2E_Sparrow_WithChecks_ConfigureOnce(t *testing.T) {
	framework := test.NewFramework(t)
	tests := []struct {
		name          string
		startup       test.ConfigBuilder
		checks        []test.CheckBuilder
		wantEndpoints map[string]int
	}{
		{
			name:    "no checks",
			startup: *test.NewSparrowConfig(),
			checks:  nil,
			wantEndpoints: map[string]int{
				"http://localhost:8080/v1/metrics/health":     http.StatusNotFound,
				"http://localhost:8080/v1/metrics/latency":    http.StatusNotFound,
				"http://localhost:8080/v1/metrics/dns":        http.StatusNotFound,
				"http://localhost:8080/v1/metrics/traceroute": http.StatusNotFound,
			},
		},
		{
			name:    "with health check",
			startup: *test.NewSparrowConfig(),
			checks: []test.CheckBuilder{
				test.NewHealthCheck().
					WithInterval(checkInterval).
					WithTimeout(checkTimeout).
					WithTargets("https://www.example.com/", "https://www.google.com/"),
			},
			wantEndpoints: map[string]int{
				"http://localhost:8080/v1/metrics/health":     http.StatusOK,
				"http://localhost:8080/v1/metrics/latency":    http.StatusNotFound,
				"http://localhost:8080/v1/metrics/dns":        http.StatusNotFound,
				"http://localhost:8080/v1/metrics/traceroute": http.StatusNotFound,
			},
		},
		{
			name:    "with health, latency and dns checks",
			startup: *test.NewSparrowConfig(),
			checks: []test.CheckBuilder{
				test.NewHealthCheck().
					WithInterval(checkInterval).
					WithTimeout(checkTimeout).
					WithTargets("https://www.example.com/"),
				test.NewLatencyCheck().
					WithInterval(checkInterval).
					WithTimeout(checkTimeout).
					WithTargets("https://www.example.com/"),
				test.NewDNSCheck().
					WithInterval(checkInterval).
					WithTimeout(checkTimeout).
					WithTargets("www.example.com"),
			},
			wantEndpoints: map[string]int{
				"http://localhost:8080/v1/metrics/health":     http.StatusOK,
				"http://localhost:8080/v1/metrics/latency":    http.StatusOK,
				"http://localhost:8080/v1/metrics/dns":        http.StatusOK,
				"http://localhost:8080/v1/metrics/traceroute": http.StatusNotFound,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e2e := framework.E2E(t, tt.startup.Config(t)).WithChecks(tt.checks...)
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
			defer cancel()

			finish := make(chan error, 1)
			go func() {
				finish <- e2e.Run(ctx)
			}()
			e2e.AwaitStartup("http://localhost:8080", checkTimeout).AwaitChecks()

			for url, status := range tt.wantEndpoints {
				e2e.HttpAssertion(url).WithSchema().Assert(status)
			}

			cancel()
			<-finish
		})
	}
}

const loaderInterval = 5 * time.Second

func TestE2E_Sparrow_WithChecks_Reconfigure(t *testing.T) {
	framework := test.NewFramework(t)

	type result struct {
		status   int
		response checks.Result
	}
	tests := []struct {
		name          string
		startup       test.ConfigBuilder
		initialChecks []test.CheckBuilder
		wantInitial   map[string]result
		secondChecks  []test.CheckBuilder
		wantSecond    map[string]result
	}{
		{
			name: "with health check then latency check",
			startup: *test.NewSparrowConfig().WithLoader(
				test.NewLoaderConfig().
					WithInterval(loaderInterval).
					Build(),
			),
			initialChecks: []test.CheckBuilder{
				test.NewHealthCheck().
					WithInterval(checkInterval).
					WithTimeout(checkTimeout).
					WithTargets("https://www.example.com/", "https://www.google.com/"),
			},
			wantInitial: map[string]result{
				"http://localhost:8080/v1/metrics/health": {
					status: http.StatusOK,
					response: checks.Result{
						Data: map[string]any{
							"https://www.example.com/": "healthy",
							"https://www.google.com/":  "healthy",
						},
						Timestamp: time.Now(),
					},
				},
				"http://localhost:8080/v1/metrics/latency":    {status: http.StatusNotFound},
				"http://localhost:8080/v1/metrics/dns":        {status: http.StatusNotFound},
				"http://localhost:8080/v1/metrics/traceroute": {status: http.StatusNotFound},
			},
			secondChecks: []test.CheckBuilder{
				test.NewLatencyCheck().
					WithInterval(checkInterval).
					WithTimeout(checkTimeout).
					WithTargets("https://www.example.com/"),
			},
			wantSecond: map[string]result{
				"http://localhost:8080/v1/metrics/health": {
					status: http.StatusOK,
					response: checks.Result{
						Data: map[string]any{
							"https://www.example.com/": "healthy",
							"https://www.google.com/":  "healthy",
						},
						Timestamp: time.Now(),
					},
				},
				"http://localhost:8080/v1/metrics/latency": {
					status: http.StatusOK,
					response: checks.Result{
						Data: map[string]any{
							"https://www.example.com/": map[string]any{
								"code":  http.StatusOK,
								"error": nil,
								"total": time.Since(time.Now().Add(-100 * time.Millisecond)).Seconds(),
							},
						},
						Timestamp: time.Now(),
					},
				},
				"http://localhost:8080/v1/metrics/dns":        {status: http.StatusNotFound},
				"http://localhost:8080/v1/metrics/traceroute": {status: http.StatusNotFound},
			},
		},
		{
			name: "with health check then dns check",
			startup: *test.NewSparrowConfig().WithLoader(
				test.NewLoaderConfig().
					WithInterval(loaderInterval).
					Build(),
			),
			initialChecks: []test.CheckBuilder{
				test.NewHealthCheck().
					WithInterval(checkInterval).
					WithTimeout(checkTimeout).
					WithTargets("https://www.example.com/"),
			},
			wantInitial: map[string]result{
				"http://localhost:8080/v1/metrics/health": {
					status: http.StatusOK,
					response: checks.Result{
						Data: map[string]any{
							"https://www.example.com/": "healthy",
						},
						Timestamp: time.Now(),
					},
				},
				"http://localhost:8080/v1/metrics/latency":    {status: http.StatusNotFound},
				"http://localhost:8080/v1/metrics/dns":        {status: http.StatusNotFound},
				"http://localhost:8080/v1/metrics/traceroute": {status: http.StatusNotFound},
			},
			secondChecks: []test.CheckBuilder{
				test.NewDNSCheck().
					WithInterval(checkInterval).
					WithTimeout(checkTimeout).
					WithTargets("www.example.com"),
			},
			wantSecond: map[string]result{
				"http://localhost:8080/v1/metrics/health": {
					status: http.StatusOK,
					response: checks.Result{
						Data: map[string]any{
							"https://www.example.com/": "healthy",
						},
						Timestamp: time.Now(),
					},
				},
				"http://localhost:8080/v1/metrics/latency": {status: http.StatusNotFound},
				"http://localhost:8080/v1/metrics/dns": {
					status: http.StatusOK,
					response: checks.Result{
						Data: map[string]any{
							"www.example.com": map[string]any{
								"resolved": []string{"1.2.3.4"},
								"error":    nil,
								"total":    time.Since(time.Now().Add(-100 * time.Millisecond)).Seconds(),
							},
						},
						Timestamp: time.Now(),
					},
				},
				"http://localhost:8080/v1/metrics/traceroute": {status: http.StatusNotFound},
			},
		},
		{
			name: "with health check then updated health check",
			startup: *test.NewSparrowConfig().WithLoader(
				test.NewLoaderConfig().
					WithInterval(loaderInterval).
					Build(),
			),
			initialChecks: []test.CheckBuilder{
				test.NewHealthCheck().
					WithInterval(checkInterval).
					WithTimeout(checkTimeout).
					WithTargets("https://www.example.com/"),
			},
			wantInitial: map[string]result{
				"http://localhost:8080/v1/metrics/health": {
					status: http.StatusOK,
					response: checks.Result{
						Data: map[string]any{
							"https://www.example.com/": "healthy",
						},
						Timestamp: time.Now(),
					},
				},
				"http://localhost:8080/v1/metrics/latency":    {status: http.StatusNotFound},
				"http://localhost:8080/v1/metrics/dns":        {status: http.StatusNotFound},
				"http://localhost:8080/v1/metrics/traceroute": {status: http.StatusNotFound},
			},
			secondChecks: []test.CheckBuilder{
				test.NewHealthCheck().
					WithInterval(checkInterval).
					WithTimeout(checkTimeout).
					WithTargets("https://www.google.com/"),
			},
			wantSecond: map[string]result{
				"http://localhost:8080/v1/metrics/health": {
					status: http.StatusOK,
					response: checks.Result{
						Data: map[string]any{
							"https://www.google.com/": "healthy",
						},
						Timestamp: time.Now(),
					},
				},
				"http://localhost:8080/v1/metrics/latency":    {status: http.StatusNotFound},
				"http://localhost:8080/v1/metrics/dns":        {status: http.StatusNotFound},
				"http://localhost:8080/v1/metrics/traceroute": {status: http.StatusNotFound},
			},
		},
		{
			name: "with health check then no checks",
			startup: *test.NewSparrowConfig().WithLoader(
				test.NewLoaderConfig().
					WithInterval(loaderInterval).
					Build(),
			),
			initialChecks: []test.CheckBuilder{
				test.NewHealthCheck().
					WithInterval(checkInterval).
					WithTimeout(checkTimeout).
					WithTargets("https://www.example.com/"),
			},
			wantInitial: map[string]result{
				"http://localhost:8080/v1/metrics/health": {
					status: http.StatusOK,
					response: checks.Result{
						Data: map[string]any{
							"https://www.example.com/": "healthy",
						},
						Timestamp: time.Now(),
					},
				},
				"http://localhost:8080/v1/metrics/latency":    {status: http.StatusNotFound},
				"http://localhost:8080/v1/metrics/dns":        {status: http.StatusNotFound},
				"http://localhost:8080/v1/metrics/traceroute": {status: http.StatusNotFound},
			},
			secondChecks: nil,
			wantSecond: map[string]result{
				"http://localhost:8080/v1/metrics/health": {
					status: http.StatusOK,
					response: checks.Result{
						Data: map[string]any{
							"https://www.example.com/": "healthy",
						},
						Timestamp: time.Now(),
					},
				},
				"http://localhost:8080/v1/metrics/latency":    {status: http.StatusNotFound},
				"http://localhost:8080/v1/metrics/dns":        {status: http.StatusNotFound},
				"http://localhost:8080/v1/metrics/traceroute": {status: http.StatusNotFound},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e2e := framework.E2E(t, tt.startup.Config(t)).WithChecks(tt.initialChecks...)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			finish := make(chan error, 1)
			go func() {
				finish <- e2e.Run(ctx)
			}()
			e2e.AwaitStartup("http://localhost:8080", checkTimeout).AwaitChecks()

			for url, result := range tt.wantInitial {
				e2e.HttpAssertion(url).
					WithSchema().
					WithCheckResult(result.response).
					Assert(result.status)
			}

			e2e.UpdateChecks(tt.secondChecks...).AwaitLoader().AwaitChecks()
			for url, result := range tt.wantSecond {
				e2e.HttpAssertion(url).
					WithSchema().
					WithCheckResult(result.response).
					Assert(result.status)
			}

			cancel()
			<-finish
		})
	}
}

func TestE2E_Sparrow_WithRemoteConfig(t *testing.T) {}

func TestE2E_Sparrow_WithTargetManager(t *testing.T) {}
