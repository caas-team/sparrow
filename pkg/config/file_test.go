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
	"io/fs"
	"reflect"
	"testing"
	"time"

	"github.com/caas-team/sparrow/pkg/checks/health"
	"github.com/caas-team/sparrow/pkg/checks/runtime"
	"github.com/caas-team/sparrow/pkg/config/test"
	"gopkg.in/yaml.v3"
)

func TestNewFileLoader(t *testing.T) {
	l := NewFileLoader(&Config{Loader: LoaderConfig{File: FileLoaderConfig{Path: "config.yaml"}}}, make(chan runtime.Config, 1))

	if l.config.File.Path != "config.yaml" {
		t.Errorf("Expected path to be config.yaml, got %s", l.config.File.Path)
	}
	if l.cRuntime == nil {
		t.Errorf("Expected channel to be not nil")
	}
	if l.fsys == nil {
		t.Errorf("Expected filesystem to be not nil")
	}
}

func TestFileLoader_Run(t *testing.T) {
	tests := []struct {
		name    string
		config  LoaderConfig
		want    runtime.Config
		wantErr bool
	}{
		{
			name: "Loads config from file",
			config: LoaderConfig{
				Type:     "file",
				Interval: 1 * time.Second,
				File: FileLoaderConfig{
					Path: "test/data/config.yaml",
				},
			},
			want: runtime.Config{
				Health: &health.Config{
					Targets:  []string{"http://localhost:8080/health"},
					Interval: 1 * time.Second,
					Timeout:  1 * time.Second,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result := make(chan runtime.Config, 1)
			defer close(result)
			f := NewFileLoader(&Config{
				Loader: tt.config,
			}, result)

			go func() {
				err := f.Run(ctx)
				if (err != nil) != tt.wantErr {
					t.Errorf("Run() error %v, want %v", err, tt.wantErr)
				}
			}()

			if !tt.wantErr {
				config := <-result
				if !reflect.DeepEqual(config, tt.want) {
					t.Errorf("Expected config to be %v, got %v", tt.want, config)
				}
			}
			f.Shutdown(ctx)
		})
	}
}

func TestFileLoader_getRuntimeConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  LoaderConfig
		mockFS  func(t *testing.T) fs.FS
		want    runtime.Config
		wantErr bool
	}{
		{
			name: "Invalid File Path",
			config: LoaderConfig{
				Type:     "file",
				Interval: 1 * time.Second,
				File: FileLoaderConfig{
					Path: "test/data/nonexistent.yaml",
				},
			},
			wantErr: true,
		},
		{
			name: "Malformed Config File",
			config: LoaderConfig{
				Type:     "file",
				Interval: 1 * time.Second,
				File: FileLoaderConfig{
					Path: "test/data/malformed.yaml",
				},
			},
			mockFS: func(_ *testing.T) fs.FS {
				return &test.MockFS{
					OpenFunc: func(name string) (fs.File, error) {
						content := []byte("this is not a valid yaml content")
						return &test.MockFile{Content: content}, nil
					},
				}
			},
			wantErr: true,
		},
		{
			name: "Failed to close file",
			config: LoaderConfig{
				Type:     "file",
				Interval: 1 * time.Second,
				File: FileLoaderConfig{
					Path: "test/data/valid.yaml",
				},
			},
			mockFS: func(t *testing.T) fs.FS {
				b, err := yaml.Marshal(LoaderConfig{
					Type:     "file",
					Interval: 1 * time.Second,
					File: FileLoaderConfig{
						Path: "test/data/valid.yaml",
					},
				})
				if err != nil {
					t.Fatalf("Failed marshaling response to bytes: %v", err)
				}

				return &test.MockFS{
					OpenFunc: func(name string) (fs.File, error) {
						return &test.MockFile{
							Content: b,
							CloseFunc: func() error {
								return fmt.Errorf("failed to close file")
							},
						}, nil
					},
				}
			},
			wantErr: true,
		},
		{
			name: "Malformed config file and failed to close file",
			config: LoaderConfig{
				Type:     "file",
				Interval: 1 * time.Second,
				File: FileLoaderConfig{
					Path: "test/data/malformed.yaml",
				},
			},
			mockFS: func(t *testing.T) fs.FS {
				return &test.MockFS{
					OpenFunc: func(name string) (fs.File, error) {
						return &test.MockFile{
							Content: []byte("this is not a valid yaml content"),
							CloseFunc: func() error {
								return fmt.Errorf("failed to close file")
							},
						}, nil
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := make(chan runtime.Config, 1)
			defer close(res)
			f := NewFileLoader(&Config{
				Loader: tt.config,
			}, res)
			if tt.mockFS != nil {
				f.fsys = tt.mockFS(t)
			}

			cfg, err := f.getRuntimeConfig(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("getRuntimeConfig() error %v, want %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if !reflect.DeepEqual(cfg, tt.want) {
					t.Errorf("Expected config to be %v, got %v", tt.want, cfg)
				}
			}
		})
	}
}
