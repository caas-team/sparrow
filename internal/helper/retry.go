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

package helper

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/caas-team/sparrow/internal/logger"
)

type RetryConfig struct {
	Count int           `yaml:"count"`
	Delay time.Duration `yaml:"delay"`
}

// Effector will be the function called by the Retry function
type Effector func(context.Context) error

// Retry will retry the run the effector function in an exponential backoff
func Retry(effector Effector, rc RetryConfig) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		log := logger.FromContext(ctx)
		for r := 1; ; r++ {
			err := effector(ctx)
			if err == nil || r > rc.Count {
				return err
			}

			delay := getExpBackoff(rc.Delay, r)
			log.WarnContext(ctx, fmt.Sprintf("Effector call failed, retrying in %v", delay))

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
	}
}

// calculate the exponential delay for a given iteration
// first iteration is 1
func getExpBackoff(initialDelay time.Duration, iteration int) time.Duration {
	if iteration <= 1 {
		return initialDelay
	}
	return time.Duration(math.Pow(2, float64(iteration-1))) * initialDelay
}
