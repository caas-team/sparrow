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

	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/sparrow/targets/interactor"
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

// General is the general configuration of the target manager
type General struct {
	// The interval for the target reconciliation process
	CheckInterval time.Duration `yaml:"checkInterval" mapstructure:"checkInterval"`
	// How often the instance should register itself as a global target.
	// A duration of 0 means no registration.
	RegistrationInterval time.Duration `yaml:"registrationInterval" mapstructure:"registrationInterval"`
	// How often the instance should update its registration as a global target.
	// A duration of 0 means no update.
	UpdateInterval time.Duration `yaml:"updateInterval" mapstructure:"updateInterval"`
	// The amount of time a target can be unhealthy
	// before it is removed from the global target list.
	// A duration of 0 means no removal.
	UnhealthyThreshold time.Duration `yaml:"unhealthyThreshold" mapstructure:"unhealthyThreshold"`
}

// TargetManagerConfig is the configuration for the target manager
type TargetManagerConfig struct {
	// Type defines which target manager to use
	Type interactor.Type `yaml:"type" mapstructure:"type"`
	// General is the general configuration of the target manager
	General `yaml:",inline" mapstructure:",squash"`
	// Config is the configuration for the Config target manager
	interactor.Config `yaml:",inline" mapstructure:",squash"`
}

func (c *TargetManagerConfig) Validate(ctx context.Context) error {
	log := logger.FromContext(ctx)
	if c.CheckInterval <= 0 {
		log.Error("The check interval should be above 0", "interval", c.CheckInterval)
		return ErrInvalidCheckInterval
	}
	if c.RegistrationInterval < 0 {
		log.Error("The registration interval should be equal or above 0", "interval", c.RegistrationInterval)
		return ErrInvalidRegistrationInterval
	}
	if c.UnhealthyThreshold < 0 {
		log.Error("The unhealthy threshold should be equal or above 0", "threshold", c.UnhealthyThreshold)
		return ErrInvalidUnhealthyThreshold
	}
	if c.UpdateInterval < 0 {
		log.Error("The update interval should be equal or above 0", "interval", c.UpdateInterval)
		return ErrInvalidUpdateInterval
	}

	switch c.Type {
	case interactor.Gitlab:
		return nil
	default:
		log.Error("Invalid interactor type", "type", c.Type)
		return ErrInvalidInteractorType
	}
}
