package config

type Config struct {
	Checks map[string]any
	Loader LoaderConfig
}

// LoaderConfig is the configuration for loader
type LoaderConfig struct {
	Type     string
	Interval int
	http     HttpLoaderConfig
}

// HttpLoaderConfig is the configuration
// for the specific http loader
type HttpLoaderConfig struct {
	url   string
	token string
}

// NewConfig creates a new Config
func NewConfig() *Config {
	return &Config{
		Checks: map[string]any{},
	}
}

// Validates the config
func (c *Config) Validate() error {
	return nil
}

// SetLoaderType sets the loader type
func (c *Config) SetLoaderType(loaderType string) {
	c.Loader.Type = loaderType
}

// SetLoaderInterval sets the loader interval
func (c *Config) SetLoaderInterval(loaderInterval int) {
	c.Loader.Interval = loaderInterval
}

// SetLoaderHttpUrl sets the loader http url
func (c *Config) SetLoaderHttpUrl(url string) {
	c.Loader.http.url = url
}

// SetLoaderHttpToken sets the loader http token
func (c *Config) SetLoaderHttpToken(token string) {
	c.Loader.http.token = token
}
