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
	"time"

	"github.com/caas-team/sparrow/internal/helper"
)

type GitlabTargetManagerConfig struct {
	BaseURL   string `yaml:"baseUrl" mapstructure:"baseUrl"`
	Token     string `yaml:"token" mapstructure:"token"`
	ProjectID int    `yaml:"projectId" mapstructure:"projectId"`
}

type TargetManagerConfig struct {
	CheckInterval        time.Duration             `yaml:"checkInterval" mapstructure:"checkInterval"`
	RegistrationInterval time.Duration             `yaml:"registrationInterval" mapstructure:"registrationInterval"`
	UnhealthyThreshold   time.Duration             `yaml:"unhealthyThreshold" mapstructure:"unhealthyThreshold"`
	Gitlab               GitlabTargetManagerConfig `yaml:"gitlab" mapstructure:"gitlab"`
}

type Config struct {
	// SparrowName is the DNS name of the sparrow
	SparrowName string `yaml:"name" mapstructure:"name"`
	// Verbosity toggles verbose logging
	Verbosity bool `yaml:"verbosity" mapstructure:"verbosity"`
	// Checks is a map of configurations for the checks
	Checks        map[string]any      `yaml:"checks" mapstructure:"checks"`
	Loader        LoaderConfig        `yaml:"loader" mapstructure:"loader"`
	Api           ApiConfig           `yaml:"api" mapstructure:"api"`
	TargetManager TargetManagerConfig `yaml:"targetManager" mapstructure:"targetManager"`
}

// ApiConfig is the configuration for the data API
type ApiConfig struct {
	ListeningAddress string `yaml:"address" mapstructure:"address"`
}

// LoaderConfig is the configuration for loader
type LoaderConfig struct {
	Type     string           `yaml:"type" mapstructure:"type"`
	Interval time.Duration    `yaml:"interval" mapstructure:"interval"`
	Http     HttpLoaderConfig `yaml:"http" mapstructure:"http"`
	File     FileLoaderConfig `yaml:"file" mapstructure:"file"`
}

// HttpLoaderConfig is the configuration
// for the specific http loader
type HttpLoaderConfig struct {
	Url      string             `yaml:"url" mapstructure:"url"`
	Token    string             `yaml:"token" mapstructure:"token"`
	Timeout  time.Duration      `yaml:"timeout" mapstructure:"timeout"`
	RetryCfg helper.RetryConfig `yaml:"retry" mapstructure:"retry"`
}

type FileLoaderConfig struct {
	Path string `yaml:"path" mapstructure:"path"`
}

// NewConfig creates a new Config
func NewConfig() *Config {
	return &Config{
		Checks: map[string]any{},
	}
}

// HasTargetManager returns true if the config has a target manager
func (c *Config) HasTargetManager() bool {
	return c.TargetManager != TargetManagerConfig{}
}
