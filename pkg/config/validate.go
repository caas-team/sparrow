package config

import (
	"fmt"
	"log/slog"
)

// Validates the config
func (c *Config) Validate() error {
	// TODO: get logger from context
	log := slog.Default().WithGroup("validation")
	ok := true

	if c.Loader.http.url == "" {
		ok = false
		log.Error("The loaderHttpUrl is not set")
	}

	if c.Loader.http.timeout > c.Loader.Interval {
		ok = false
		log.Error("The loader http timeout must be smaller than the loader interval",
			"loaderHttpTimeout", c.Loader.http.timeout,
			"loaderInterval", c.Loader.Interval)
	}

	if c.Loader.http.retryCfg.Count < 0 || c.Loader.http.retryCfg.Count >= 5 {
		ok = false
		log.Error("The amount of loader http retries should be above 0 and below 6",
			"loaderHttpRetryCount", c.Loader.http.retryCfg.Count)
	}

	if !ok {
		return fmt.Errorf("validation of configuration failed")
	}
	return nil
}
