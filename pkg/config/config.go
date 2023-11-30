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

type Config struct {
	Checks map[string]any
	Loader LoaderConfig
	Api    ApiConfig
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

// NewConfig creates a new Config
func NewConfig() *Config {
	return &Config{
		Checks: map[string]any{},
	}
}

func (c *Config) SetApiAddress(address string) {
	c.Api.ListeningAddress = address
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
