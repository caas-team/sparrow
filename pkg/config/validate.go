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
	"fmt"
	"net/url"
	"regexp"

	"github.com/caas-team/sparrow/internal/logger"
)

// Validate validates the startup config
func (c *Config) Validate(ctx context.Context) error {
	ctx, cancel := logger.NewContextWithLogger(ctx)
	defer cancel()
	log := logger.FromContext(ctx)

	ok := true

	if !isDNSName(c.SparrowName) {
		ok = false
		log.Error("The name of the sparrow must be DNS compliant")
	}

	switch c.Loader.Type {
	case "http":
		if _, err := url.ParseRequestURI(c.Loader.Http.Url); err != nil {
			ok = false
			log.Error("The loader http url is not a valid url")
		}
		if c.Loader.Http.RetryCfg.Count < 0 || c.Loader.Http.RetryCfg.Count >= 5 {
			ok = false
			log.Error("The amount of loader http retries should be above 0 and below 6", "retryCount", c.Loader.Http.RetryCfg.Count)
		}
	case "file":
		if c.Loader.File.Path == "" {
			ok = false
			log.Error("The loader file path cannot be empty")
		}
	}

	if !ok {
		return fmt.Errorf("validation of configuration failed")
	}
	return nil
}

// isDNSName checks if the given string is a valid DNS name
func isDNSName(s string) bool {
	re := regexp.MustCompile(`^([a-z0-9]([a-z0-9\-]{0,61}[a-z0-9])?\.)+[a-z]{2,}$`)
	return re.MatchString(s)
}
