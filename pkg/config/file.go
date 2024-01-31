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
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/caas-team/sparrow/pkg/checks/runtime"
	"gopkg.in/yaml.v3"

	"github.com/caas-team/sparrow/internal/logger"
)

var _ Loader = (*FileLoader)(nil)

type FileLoader struct {
	config   LoaderConfig
	cRuntime chan<- runtime.Config
	done     chan struct{}
	fsys     fs.FS
}

func NewFileLoader(cfg *Config, cRuntime chan<- runtime.Config) *FileLoader {
	return &FileLoader{
		config:   cfg.Loader,
		cRuntime: cRuntime,
		done:     make(chan struct{}, 1),
		fsys:     os.DirFS(filepath.Dir(cfg.Loader.File.Path)),
	}
}

// Run gets the runtime configuration from the local file.
// The config will be loaded periodically defined by the loader interval configuration.
// Returns an error if the loader is shutdown or the context is done.
func (f *FileLoader) Run(ctx context.Context) error {
	ctx, cancel := logger.NewContextWithLogger(ctx)
	defer cancel()
	log := logger.FromContext(ctx)
	tick := time.NewTicker(f.config.Interval)
	defer tick.Stop()

	for {
		select {
		case <-f.done:
			log.Info("File Loader terminated")
			return nil
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			runtimeCfg, err := f.getRuntimeConfig(ctx)
			if err != nil {
				log.Warn("Could not get local runtime configuration", "error", err)
				tick.Reset(f.config.Interval)
				continue
			}

			log.Info("Successfully got local runtime configuration")
			f.cRuntime <- runtimeCfg
			tick.Reset(f.config.Interval)
		}
	}
}

// getRuntimeConfig gets the local runtime configuration from the specified file.
func (f *FileLoader) getRuntimeConfig(ctx context.Context) (runtime.Config, error) {
	log := logger.FromContext(ctx)
	var cfg runtime.Config

	file, err := f.fsys.Open(filepath.Base(f.config.File.Path))
	if err != nil {
		log.Error("Failed to open config file", "path", f.config.File.Path, "error", err)
		return cfg, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	b, err := io.ReadAll(file)
	if err != nil {
		log.Error("Failed to read config file", "path", f.config.File.Path, "error", err)
		return cfg, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(b, &cfg); err != nil {
		log.Error("Failed to parse config file", "error", err)
		return cfg, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

func (f *FileLoader) Shutdown(ctx context.Context) {
	log := logger.FromContext(ctx)
	select {
	case f.done <- struct{}{}:
		log.Debug("Sending signal to shut down file loader")
	default:
	}
}
