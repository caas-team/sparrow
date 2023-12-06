package checks

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/caas-team/sparrow/pkg/api"
	"github.com/jarcoal/httpmock"
)

func stringPointer(s string) *string {
	return &s
}

func TestLatency_check(t *testing.T) {
	httpmock.Activate()
	defer httpmock.Deactivate()

	httpmock.RegisterResponder(http.MethodGet, "http://success.com", httpmock.NewStringResponder(200, "ok"))
	httpmock.RegisterResponder(http.MethodGet, "http://fail.com", httpmock.NewStringResponder(500, "fail"))
	httpmock.RegisterResponder(http.MethodGet, "http://timeout.com", httpmock.NewErrorResponder(context.DeadlineExceeded))

	cResult := make(chan Result, 1)
	c := Latency{
		cfg:  LatencyConfig{},
		mu:   sync.Mutex{},
		c:    cResult,
		done: make(chan bool, 1),
	}
	results := make(chan Result, 1)
	_ = c.Startup(context.Background(), results)

	_ = c.SetConfig(context.Background(), LatencyConfig{
		Targets:  []string{"http://success.com", "http://fail.com", "http://timeout.com"},
		Interval: time.Second * 120,
		Timeout:  time.Second * 1,
	})
	defer func(c *Latency, _ context.Context) {
		_ = c.Shutdown(context.Background())
	}(&c, context.Background())

	data, err := c.check(context.Background())
	if err != nil {
		t.Errorf("Latency.check() error = %v", err)
	}

	wantData := map[string]LatencyResult{
		"http://success.com": {
			Code:  200,
			Error: nil,
			Total: 0,
		},
		"http://fail.com": {
			Code:  500,
			Error: nil,
			Total: 0,
		},
		"http://timeout.com": {
			Code:  0,
			Error: stringPointer("Get \"http://timeout.com\": context deadline exceeded"),
			Total: 0,
		},
	}

	for k, v := range wantData {
		if v.Code != data[k].Code {
			t.Errorf("Latency.Run() = %v, want %v", data[k].Code, v.Code)
		}
		if v.Total != data[k].Total {
			t.Errorf("Latency.Run() = %v, want %v", data[k].Total, v.Total)
		}
		if v.Error != nil && data[k].Error != nil {
			if *v.Error != *data[k].Error {
				t.Errorf("Latency.Run() = %v, want %v", *data[k].Error, *v.Error)
			}
		}
	}
}

func TestLatency_Run(t *testing.T) {
	httpmock.Activate()
	defer httpmock.Deactivate()

	httpmock.RegisterResponder(http.MethodGet, "http://success.com", httpmock.NewStringResponder(200, "ok"))
	httpmock.RegisterResponder(http.MethodGet, "http://fail.com", httpmock.NewStringResponder(500, "fail"))
	httpmock.RegisterResponder(http.MethodGet, "http://timeout.com", httpmock.NewErrorResponder(context.DeadlineExceeded))

	c := NewLatencyCheck()
	results := make(chan Result, 1)
	_ = c.Startup(context.Background(), results)

	_ = c.SetConfig(context.Background(), LatencyConfig{
		Targets:  []string{"http://success.com", "http://fail.com", "http://timeout.com"},
		Interval: time.Second * 120,
		Timeout:  time.Second * 1,
	})
	go func() {
		_ = c.Run(context.Background())
	}()
	defer func(c Check, ctx context.Context) {
		_ = c.Shutdown(ctx)
	}(c, context.Background())

	result := <-results
	wantResult := Result{
		Timestamp: result.Timestamp,
		Err:       "",
		Data: map[string]LatencyResult{
			"http://success.com": {
				Code:  200,
				Error: nil,
				Total: 0,
			},
			"http://fail.com": {
				Code:  500,
				Error: nil,
				Total: 0,
			},
			"http://timeout.com": {
				Code:  0,
				Error: stringPointer("Get \"http://timeout.com\": context deadline exceeded"),
				Total: 0,
			},
		},
	}

	if wantResult.Timestamp != result.Timestamp {
		t.Errorf("Latency.Run() = %v, want %v", result.Timestamp, wantResult.Timestamp)
	}
	if wantResult.Err != result.Err {
		t.Errorf("Latency.Run() = %v, want %v", result.Err, wantResult.Err)
	}
	wantData := wantResult.Data.(map[string]LatencyResult)
	data := result.Data.(map[string]LatencyResult)

	for k, v := range wantData {
		if v.Code != data[k].Code {
			t.Errorf("Latency.Run() = %v, want %v", data[k].Code, v.Code)
		}
		if v.Total != data[k].Total {
			t.Errorf("Latency.Run() = %v, want %v", data[k].Total, v.Total)
		}
		if v.Error != nil && data[k].Error != nil {
			if *v.Error != *data[k].Error {
				t.Errorf("Latency.Run() = %v, want %v", *data[k].Error, *v.Error)
			}
		}
	}
}

func TestLatency_Startup(t *testing.T) {
	c := Latency{}

	if err := c.Startup(context.Background(), make(chan<- Result, 1)); err != nil {
		t.Errorf("Startup() error = %v", err)
	}
}

func TestLatency_Shutdown(t *testing.T) {
	cDone := make(chan bool, 1)
	c := Latency{
		done: cDone,
	}
	err := c.Shutdown(context.Background())
	if err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}

	if !<-cDone {
		t.Error("Shutdown() should be ok")
	}
}

func TestLatency_SetConfig(t *testing.T) {
	c := Latency{}
	wantCfg := LatencyConfig{
		Targets: []string{"http://localhost:9090"},
	}

	err := c.SetConfig(context.Background(), wantCfg)
	if err != nil {
		t.Errorf("SetConfig() error = %v", err)
	}
	if !reflect.DeepEqual(c.cfg, wantCfg) {
		t.Errorf("SetConfig() = %v, want %v", c.cfg, wantCfg)
	}
}

func TestLatency_RegisterHandler(t *testing.T) {
	c := Latency{}

	rt := api.NewRoutingTree()
	c.RegisterHandler(context.Background(), rt)

	h, ok := rt.Get("GET", "v1alpha1/latency")

	if !ok {
		t.Error("RegisterHandler() should be ok")
	}
	if h == nil {
		t.Error("RegisterHandler() should not be nil")
	}
	c.DeregisterHandler(context.Background(), rt)
	h, ok = rt.Get("GET", "v1alpha1/latency")

	if ok {
		t.Error("DeregisterHandler() should not be ok")
	}

	if h != nil {
		t.Error("DeregisterHandler() should be nil")
	}
}

func TestLatency_Handler(t *testing.T) {
	c := Latency{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1alpha1/latency", http.NoBody)

	c.Handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Handler() should be ok, got %d", rec.Code)
	}
}

func TestNewLatencyCheck(t *testing.T) {
	c := NewLatencyCheck()
	if c == nil {
		t.Error("NewLatencyCheck() should not be nil")
	}
}
