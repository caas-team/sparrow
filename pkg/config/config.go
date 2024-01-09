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
	"os"
	"time"

	"gopkg.in/yaml.v3"

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
	// Checks is a map of configurations for the checks
	Checks        map[string]any      `yaml:"checks" mapstructure:"checks"`
	Loader        LoaderConfig        `yaml:"loader" mapstructure:"loader"`
	Api           ApiConfig           `yaml:"api" mapstructure:"api"`
	TargetManager TargetManagerConfig `yaml:"targetmanager" mapstructure:"targetmanager"`
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

// NewTargetManagerConfig creates a new TargetManagerConfig
// from the passed file
func NewTargetManagerConfig(path string) TargetManagerConfig {
	if path == "" {
		return TargetManagerConfig{}
	}

	var res TargetManagerConfig
	f, err := os.ReadFile(path) //#nosec G304
	if err != nil {
		panic("failed to read config file " + err.Error())
	}

	err = yaml.Unmarshal(f, &res)
	if err != nil {
		panic("failed to parse config file: " + err.Error())
	}

	return res
}

// NewConfig creates a new Config
func NewConfig() *Config {
	return &Config{
		Checks: map[string]any{},
	}
}

func (c *Config) SetApiAddress(address string) {
	c.Api.ListeningAddress = address
}

// SetSparrowName sets the DNS name of the sparrow
func (c *Config) SetSparrowName(name string) {
	c.SparrowName = name
}

// SetLoaderType sets the loader type
func (c *Config) SetLoaderType(loaderType string) {
	c.Loader.Type = loaderType
}

func (c *Config) SetLoaderFilePath(loaderFilePath string) {
	c.Loader.File.Path = loaderFilePath
}

// SetLoaderInterval sets the loader interval
// loaderInterval in seconds
func (c *Config) SetLoaderInterval(loaderInterval int) {
	c.Loader.Interval = time.Duration(loaderInterval) * time.Second
}

// SetLoaderHttpUrl sets the loader http url
func (c *Config) SetLoaderHttpUrl(url string) {
	c.Loader.Http.Url = url
}

// SetLoaderHttpToken sets the loader http token
func (c *Config) SetLoaderHttpToken(token string) {
	c.Loader.Http.Token = token
}

// SetLoaderHttpTimeout sets the loader http timeout
// timeout in seconds
func (c *Config) SetLoaderHttpTimeout(timeout int) {
	c.Loader.Http.Timeout = time.Duration(timeout) * time.Second
}

// SetLoaderHttpRetryCount sets the loader http retry count
func (c *Config) SetLoaderHttpRetryCount(retryCount int) {
	c.Loader.Http.RetryCfg.Count = retryCount
}

// SetLoaderHttpRetryDelay sets the loader http retry delay
// retryDelay in seconds
func (c *Config) SetLoaderHttpRetryDelay(retryDelay int) {
	c.Loader.Http.RetryCfg.Delay = time.Duration(retryDelay) * time.Second
}

// SetTargetManagerConfig sets the target manager config
func (c *Config) SetTargetManagerConfig(config TargetManagerConfig) {
	c.TargetManager = config
}

// HasTargetManager returns true if the config has a target manager
func (c *Config) HasTargetManager() bool {
	return c.TargetManager != TargetManagerConfig{}
}
