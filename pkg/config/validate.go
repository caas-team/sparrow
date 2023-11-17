package config

import (
	"fmt"
	"log/slog"
	"net/url"
)

// Validates the config
func (c *Config) Validate(fm *RunFlagsNameMapping) error {
	// TODO: get logger from context
	log := slog.Default().WithGroup("validation")
	ok := true

	if _, err := url.ParseRequestURI(c.Loader.http.url); err != nil {
		ok = false
		log.Error("The loader http url is not a valid url",
			fm.LoaderHttpUrl, c.Loader.http.url)
	}

	if c.Loader.http.retryCfg.Count < 0 || c.Loader.http.retryCfg.Count >= 5 {
		ok = false
		log.Error("The amount of loader http retries should be above 0 and below 6",
			fm.LoaderHttpRetryCount, c.Loader.http.retryCfg.Count)
	}

	if !ok {
		return fmt.Errorf("validation of configuration failed")
	}
	return nil
}
