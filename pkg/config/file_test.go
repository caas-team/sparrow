package config

import (
	"context"
	"reflect"
	"testing"
)

func TestNewFileLoader(t *testing.T) {
	l := NewFileLoader(&Config{Loader: LoaderConfig{file: FileLoaderConfig{path: "config.yaml"}}}, make(chan<- map[string]any))

	if l.path != "config.yaml" {
		t.Errorf("Expected path to be config.yaml, got %s", l.path)
	}
	if l.c == nil {
		t.Errorf("Expected channel to be not nil")
	}
}

func TestFileLoader_Run(t *testing.T) {
	type fields struct {
		path string
		c    chan map[string]any
	}
	type args struct {
		ctx    *context.Context
		cancel *context.CancelFunc
	}
	type want struct {
		cfg map[string]any
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{name: "Loads config from file", fields: fields{path: "testdata/config.yaml", c: make(chan map[string]any)}, args: func() args {
			ctx, cancel := context.WithCancel(context.Background())
			return args{ctx: &ctx, cancel: &cancel}
		}(), want: want{cfg: map[string]any{"testCheck1": map[string]any{"enabled": true}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FileLoader{
				path: tt.fields.path,
				c:    tt.fields.c,
			}
			go f.Run(*tt.args.ctx)
			(*tt.args.cancel)()

			config := <-tt.fields.c

			if !reflect.DeepEqual(config, tt.want.cfg) {
				t.Errorf("Expected config to be %v, got %v", tt.want.cfg, config)
			}
		})
	}
}
