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
	Port string
}

// LoaderConfig is the configuration for loader
type LoaderConfig struct {
	Type     string
	Interval time.Duration
	http     HttpLoaderConfig
}

// HttpLoaderConfig is the configuration
// for the specific http loader
type HttpLoaderConfig struct {
	url      string
	token    string
	timeout  time.Duration
	retryCfg helper.RetryConfig
}

// NewConfig creates a new Config
func NewConfig() *Config {
	return &Config{
		Checks: map[string]any{},
	}
}

// SetLoaderType sets the loader type
func (c *Config) SetLoaderType(loaderType string) {
	c.Loader.Type = loaderType
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
