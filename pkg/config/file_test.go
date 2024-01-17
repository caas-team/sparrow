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
	"reflect"
	"testing"
	"time"

	"github.com/caas-team/sparrow/pkg/checks/runtime"

	"github.com/caas-team/sparrow/pkg/checks/health"
)

func TestNewFileLoader(t *testing.T) {
	l := NewFileLoader(&Config{Loader: LoaderConfig{File: FileLoaderConfig{Path: "config.yaml"}}}, make(chan runtime.Config, 1))

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
		c    chan runtime.Config
	}
	type args struct {
		ctx    *context.Context
		cancel *context.CancelFunc
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   runtime.Config
	}{
		{
			name:   "Loads config from file",
			fields: fields{path: "testdata/config.yaml", c: make(chan runtime.Config, 1)},
			args: func() args {
				ctx, cancel := context.WithCancel(context.Background())
				return args{ctx: &ctx, cancel: &cancel}
			}(),
			want: runtime.Config{
				Checks: runtime.Checks{
					Health: &health.Config{
						Targets:  []string{"http://localhost:8080/health"},
						Interval: 1 * time.Second,
						Timeout:  1 * time.Second,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FileLoader{
				path: tt.fields.path,
				c:    tt.fields.c,
			}
			go func() {
				err := f.Run(*tt.args.ctx)
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
					return
				}
			}()
			(*tt.args.cancel)()

			config := <-tt.fields.c

			if !reflect.DeepEqual(config, tt.want) {
				t.Errorf("Expected config to be %v, got %v", tt.want, config)
			}
		})
	}
}
