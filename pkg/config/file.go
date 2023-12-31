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
	"os"

	"gopkg.in/yaml.v3"

	"github.com/caas-team/sparrow/internal/logger"
)

var _ Loader = (*FileLoader)(nil)

type FileLoader struct {
	path string
	c    chan<- map[string]any
}

func NewFileLoader(cfg *Config, cCfgChecks chan<- map[string]any) *FileLoader {
	return &FileLoader{
		path: cfg.Loader.file.path,
		c:    cCfgChecks,
	}
}

func (f *FileLoader) Run(ctx context.Context) {
	log := logger.FromContext(ctx).WithGroup("FileLoader")
	log.Info("Reading config from file", "file", f.path)
	// TODO refactor this to use fs.FS
	b, err := os.ReadFile(f.path)
	if err != nil {
		log.Error("Failed to read config file", "path", f.path, "error", err)
		panic("failed to read config file " + err.Error())
	}

	var cfg RuntimeConfig

	if err := yaml.Unmarshal(b, &cfg); err != nil {
		log.Error("Failed to parse config file", "error", err)
		panic("failed to parse config file: " + err.Error())
	}

	f.c <- cfg.Checks
}
