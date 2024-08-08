package checks

import (
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

const (
	name = "test"
)

func TestBase_Shutdown(t *testing.T) {
	cDone := make(chan struct{}, 1)
	c := &Base[*RuntimeMock]{
		Mu:       sync.Mutex{},
		DoneChan: cDone,
	}
	c.Shutdown()

	_, ok := <-cDone
	if !ok {
		t.Error("Shutdown() should be ok")
	}

	assert.Panics(t, func() {
		cDone <- struct{}{}
	}, "Channel is closed, should panic")

	ch := NewBase(name, &RuntimeMock{})
	ch.Shutdown()

	_, ok = <-ch.DoneChan
	if !ok {
		t.Error("Channel should be done")
	}

	assert.Panics(t, func() {
		ch.DoneChan <- struct{}{}
	}, "Channel is closed, should panic")
}

func TestBase_SetConfig(t *testing.T) {
	tests := []struct {
		name    string
		input   Runtime
		want    *mockConfig
		wantErr bool
	}{
		{
			name: "simple config",
			input: &mockConfig{
				Targets: []string{
					"example.com",
					"sparrow.com",
				},
				Interval: 10 * time.Second,
				Timeout:  30 * time.Second,
			},
			want: &mockConfig{
				Targets:  []string{"example.com", "sparrow.com"},
				Interval: 10 * time.Second,
				Timeout:  30 * time.Second,
			},
			wantErr: false,
		},
		{
			name:    "empty config",
			input:   &mockConfig{},
			want:    &mockConfig{},
			wantErr: false,
		},
		{
			name: "wrong type",
			input: &RuntimeMock{
				ForFunc: func() string { return "mock" },
			},
			want:    &mockConfig{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewBase(name, &mockConfig{})

			if err := c.SetConfig(tt.input); (err != nil) != tt.wantErr {
				t.Errorf("DNS.SetConfig() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !cmp.Equal(c.Config, tt.want) {
				t.Error(cmp.Diff(c.Config, tt.want))
			}
		})
	}
}

func TestBase_Name(t *testing.T) {
	c := NewBase(name, &RuntimeMock{})
	if c.Name() != name {
		t.Errorf("Name() should return %q", name)
	}
}

func TestBase_GetConfig(t *testing.T) {
	c := NewBase(name, &mockConfig{
		Targets:  []string{"example.com"},
		Interval: 10 * time.Second,
		Timeout:  30 * time.Second,
	})

	cfg := c.GetConfig().(*mockConfig)
	if len(cfg.Targets) != 1 {
		t.Error("Targets should contain 1 element")
	}
	if cfg.Interval != 10*time.Second {
		t.Error("Interval should be 10 seconds")
	}
	if cfg.Timeout != 30*time.Second {
		t.Error("Timeout should be 30 seconds")
	}
}

func TestBase_SendResult(t *testing.T) {
	cResult := make(chan ResultDTO, 1)
	defer close(cResult)

	c := NewBase(name, &RuntimeMock{})
	c.SendResult(cResult, name)

	r := <-cResult
	if r.Name != name {
		t.Error("Name should be 'test'")
	}
	if r.Result == nil {
		t.Error("Result should not be nil")
	}

	if r.Result.Data != name {
		t.Error("Data should be 'test'")
	}
}

type mockConfig struct {
	Targets  []string
	Interval time.Duration
	Timeout  time.Duration
}

func (m *mockConfig) For() string {
	return "mock"
}

func (m *mockConfig) Validate() error {
	return nil
}
