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

package targets

import (
	"context"
	"time"

	"github.com/caas-team/sparrow/pkg/checks"
)

// TargetManager handles the management of globalTargets for
// a Sparrow instance
type TargetManager interface {
	// Reconcile fetches the global targets from the configured
	// endpoint and updates the local state
	Reconcile(ctx context.Context) error
	// GetTargets returns the current global targets
	GetTargets() []checks.GlobalTarget
	// Shutdown shuts down the target manager
	// and unregisters the instance as a global target
	Shutdown(ctx context.Context) error
}

// Config is the general configuration of the target manager
type Config struct {
	// The interval for the target reconciliation process
	CheckInterval time.Duration `yaml:"checkInterval" mapstructure:"checkInterval"`
	// How often the instance should register itself as a global target.
	// A duration of 0 means no registration.
	RegistrationInterval time.Duration `yaml:"registrationInterval" mapstructure:"registrationInterval"`
	// The amount of time a target can be unhealthy
	// before it is removed from the global target list.
	// A duration of 0 means no removal.
	UnhealthyThreshold time.Duration `yaml:"unhealthyThreshold" mapstructure:"unhealthyThreshold"`
}

type TargetManagerConfig struct {
	Config
	Gitlab GitlabTargetManagerConfig `yaml:"gitlab" mapstructure:"gitlab"`
}

func (tmc *TargetManagerConfig) Validate() error {
	if tmc.CheckInterval <= 0 {
		return ErrInvalidCheckInterval
	}
	if tmc.RegistrationInterval < 0 {
		return ErrInvalidRegistrationInterval
	}
	if tmc.UnhealthyThreshold < 0 {
		return ErrInvalidUnhealthyThreshold
	}
	return nil
}
