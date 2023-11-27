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

	"github.com/caas-team/sparrow/internal/logger"
)

// Validates the config
func (c *Config) Validate(ctx context.Context, fm *RunFlagsNameMapping) error {
	ctx, cancel := logger.NewContextWithLogger(ctx, "configValidation")
	defer cancel()
	log := logger.FromContext(ctx)

	ok := true
	switch c.Loader.Type {
	case "http":
		if _, err := url.ParseRequestURI(c.Loader.http.url); err != nil {
			ok = false
			log.ErrorContext(ctx, "The loader http url is not a valid url",
				fm.LoaderHttpUrl, c.Loader.http.url)
		}
		if c.Loader.http.retryCfg.Count < 0 || c.Loader.http.retryCfg.Count >= 5 {
			ok = false
			log.Error("The amount of loader http retries should be above 0 and below 6",
				fm.LoaderHttpRetryCount, c.Loader.http.retryCfg.Count)
		}
	}

	if !ok {
		return fmt.Errorf("validation of configuration failed")
	}
	return nil
}
