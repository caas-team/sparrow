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
	"errors"
	"fmt"
	"net/url"
	"regexp"

	"github.com/caas-team/sparrow/internal/logger"
)

// Validate validates the startup config
func (c *Config) Validate(ctx context.Context) (err error) {
	log := logger.FromContext(ctx)
	if !isDNSName(c.SparrowName) {
		log.Error("The name of the sparrow must be DNS compliant")
		err = errors.Join(err, ErrInvalidSparrowName)
	}

	if vErr := c.Loader.Validate(ctx); vErr != nil {
		log.Error("The loader configuration is invalid")
		err = errors.Join(err, vErr)
	}

	if c.HasTargetManager() {
		if vErr := c.TargetManager.Validate(ctx); vErr != nil {
			log.Error("The target manager configuration is invalid")
			err = errors.Join(err, vErr)
		}
	}

	if c.HasTelemetry() {
		if vErr := c.Telemetry.Validate(ctx); vErr != nil {
			log.Error("The telemetry configuration is invalid")
			err = errors.Join(err, vErr)
		}
	}

	if err != nil {
		return fmt.Errorf("validation of configuration failed: %w", err)
	}
	return nil
}

// Validate validates the loader configuration
func (c *LoaderConfig) Validate(ctx context.Context) error {
	log := logger.FromContext(ctx)

	if c.Interval < 0 {
		log.Error("The loader interval should be equal or above 0", "interval", c.Interval)
		return ErrInvalidLoaderInterval
	}

	switch c.Type {
	case "http":
		if _, err := url.ParseRequestURI(c.Http.Url); err != nil {
			log.Error("The loader http url is not a valid url")
			return ErrInvalidLoaderHttpURL
		}
		if c.Http.RetryCfg.Count < 0 || c.Http.RetryCfg.Count >= 5 {
			log.Error("The amount of loader http retries should be above 0 and below 6", "retryCount", c.Http.RetryCfg.Count)
			return ErrInvalidLoaderHttpRetryCount
		}
	case "file":
		if c.File.Path == "" {
			log.Error("The loader file path cannot be empty")
			return ErrInvalidLoaderFilePath
		}
	}

	return nil
}

// isDNSName checks if the given string is a valid DNS name
func isDNSName(s string) bool {
	re := regexp.MustCompile(`^([a-z0-9]([a-z0-9\-]{0,61}[a-z0-9])?\.)+[a-z]{2,}$`)
	return re.MatchString(s)
}
