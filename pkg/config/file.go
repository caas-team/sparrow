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
	"os"

	"github.com/caas-team/sparrow/pkg/checks/runtime"

	"gopkg.in/yaml.v3"

	"github.com/caas-team/sparrow/internal/logger"
)

var _ Loader = (*FileLoader)(nil)

type FileLoader struct {
	path     string
	cRuntime chan<- runtime.Config
	done     chan struct{}
}

func NewFileLoader(cfg *Config, cRuntime chan<- runtime.Config) *FileLoader {
	return &FileLoader{
		path:     cfg.Loader.File.Path,
		cRuntime: cRuntime,
	}
}

func (f *FileLoader) Run(ctx context.Context) error {
	log := logger.FromContext(ctx)
	log.Info("Reading config from file", "file", f.path)
	// TODO refactor this to use fs.FS
	b, err := os.ReadFile(f.path)
	if err != nil {
		log.Error("Failed to read config file", "path", f.path, "error", err)
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg runtime.Config

	if err := yaml.Unmarshal(b, &cfg); err != nil {
		log.Error("Failed to parse config file", "error", err)
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	f.cRuntime <- cfg
	return nil
}

func (f *FileLoader) Shutdown(ctx context.Context) {
	// proper implementation must still be done
	// https://github.com/caas-team/sparrow/issues/85
	log := logger.FromContext(ctx)
	select {
	case f.done <- struct{}{}:
		log.Debug("Sending signal to shut down file loader")
	default:
	}
}
