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
	BaseURL   string `yaml:"baseUrl"`
	Token     string `yaml:"token"`
	ProjectID int    `yaml:"projectId"`
}

type TargetManagerConfig struct {
	CheckInterval        time.Duration             `yaml:"checkInterval"`
	RegistrationInterval time.Duration             `yaml:"registrationInterval"`
	UnhealthyThreshold   time.Duration             `yaml:"unhealthyThreshold"`
	Gitlab               GitlabTargetManagerConfig `yaml:"gitlab"`
}

type Config struct {
	// Name is the DNS name of the sparrow
	Name string
	// Checks is a map of configurations for the checks
	Checks        map[string]any
	Loader        LoaderConfig
	Api           ApiConfig
	TargetManager TargetManagerConfig
}

// ApiConfig is the configuration for the data API
type ApiConfig struct {
	ListeningAddress string
}

// LoaderConfig is the configuration for loader
type LoaderConfig struct {
	Type     string
	Interval time.Duration
	http     HttpLoaderConfig
	file     FileLoaderConfig
}

// HttpLoaderConfig is the configuration
// for the specific http loader
type HttpLoaderConfig struct {
	url      string
	token    string
	timeout  time.Duration
	retryCfg helper.RetryConfig
}

type FileLoaderConfig struct {
	path string
}

// NewTargetManagerConfig creates a new TargetManagerConfig
// from the passed file
func NewTargetManagerConfig(path string) TargetManagerConfig {
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

// SetName sets the DNS name of the sparrow
func (c *Config) SetName(name string) {
	c.Name = name
}

// SetLoaderType sets the loader type
func (c *Config) SetLoaderType(loaderType string) {
	c.Loader.Type = loaderType
}

func (c *Config) SetLoaderFilePath(loaderFilePath string) {
	c.Loader.file.path = loaderFilePath
}

// SetLoaderInterval sets the loader interval
// loaderInterval in seconds
func (c *Config) SetLoaderInterval(loaderInterval int) {
	c.Loader.Interval = time.Duration(loaderInterval) * time.Second
}

// SetLoaderHttpUrl sets the loader http url
func (c *Config) SetLoaderHttpUrl(url string) {
	c.Loader.http.url = url
}

// SetLoaderHttpToken sets the loader http token
func (c *Config) SetLoaderHttpToken(token string) {
	c.Loader.http.token = token
}

// SetLoaderHttpTimeout sets the loader http timeout
// timeout in seconds
func (c *Config) SetLoaderHttpTimeout(timeout int) {
	c.Loader.http.timeout = time.Duration(timeout) * time.Second
}

// SetLoaderHttpRetryCount sets the loader http retry count
func (c *Config) SetLoaderHttpRetryCount(retryCount int) {
	c.Loader.http.retryCfg.Count = retryCount
}

// SetLoaderHttpRetryDelay sets the loader http retry delay
// retryDelay in seconds
func (c *Config) SetLoaderHttpRetryDelay(retryDelay int) {
	c.Loader.http.retryCfg.Delay = time.Duration(retryDelay) * time.Second
}

// SetTargetManagerConfig sets the target manager config
func (c *Config) SetTargetManagerConfig(config TargetManagerConfig) {
	c.TargetManager = config
}
