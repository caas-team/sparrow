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
	"fmt"
	"sync"
	"time"

	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/sparrow/gitlab"

	"github.com/caas-team/sparrow/internal/logger"
)

var _ TargetManager = &gitlabTargetManager{}

// gitlabTargetManager implements TargetManager
type gitlabTargetManager struct {
	targets []checks.GlobalTarget
	mu      sync.RWMutex
	done    chan struct{}
	gitlab  gitlab.Gitlab
	// the DNS name used for self-registration
	name string
	// the interval for the target reconciliation process
	checkInterval time.Duration
	// the amount of time a target can be
	// unhealthy before it is removed from the global target list
	unhealthyThreshold time.Duration
	// how often the instance should register itself as a global target
	registrationInterval time.Duration
	// whether the instance has already registered itself as a global target
	registered bool
}

// NewGitlabManager creates a new gitlabTargetManager
func NewGitlabManager(g gitlab.Gitlab, name string, checkInterval, unhealthyThreshold, regInterval time.Duration) *gitlabTargetManager {
	return &gitlabTargetManager{
		gitlab:               g,
		name:                 name,
		checkInterval:        checkInterval,
		registrationInterval: regInterval,
		unhealthyThreshold:   unhealthyThreshold,
		mu:                   sync.RWMutex{},
		done:                 make(chan struct{}, 1),
	}
}

// Reconcile reconciles the targets of the gitlabTargetManager.
// The global targets are parsed from a gitlab repository.
//
// The global targets are evaluated for healthiness and
// unhealthy gitlabTargetManager are removed.
func (t *gitlabTargetManager) Reconcile(ctx context.Context) {
	log := logger.FromContext(ctx)
	log.Info("Starting global gitlabTargetManager reconciler")

	checkTimer := time.NewTimer(t.checkInterval)
	registrationTimer := time.NewTimer(t.registrationInterval)

	defer checkTimer.Stop()
	defer registrationTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				log.Error("Context canceled", "error", err)
				err = t.Shutdown(ctx)
				if err != nil {
					log.Error("Failed to shutdown gracefully", "error", err)
					return
				}
			}
		case <-t.done:
			log.Info("Ending Reconcile routine.")
			return
		case <-checkTimer.C:
			err := t.refreshTargets(ctx)
			if err != nil {
				log.Error("Failed to get global targets", "error", err)
			}
			checkTimer.Reset(t.checkInterval)
		case <-registrationTimer.C:
			err := t.updateRegistration(ctx)
			if err != nil {
				log.Error("Failed to register self as global target", "error", err)
			}
			registrationTimer.Reset(t.registrationInterval)
		}
	}
}

// GetTargets returns the current targets of the gitlabTargetManager
func (t *gitlabTargetManager) GetTargets() []checks.GlobalTarget {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.targets
}

// Shutdown shuts down the gitlabTargetManager and deletes the file containing
// the sparrow's registration from Gitlab
func (t *gitlabTargetManager) Shutdown(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	log := logger.FromContext(ctx)
	log.Info("Shutting down global gitlabTargetManager")
	t.registered = false

	select {
	case t.done <- struct{}{}:
		log.Debug("Stopping reconcile routine")
	default:
	}

	return nil
}

// updateRegistration registers the current instance as a global target
func (t *gitlabTargetManager) updateRegistration(ctx context.Context) error {
	log := logger.FromContext(ctx)
	log.Debug("Updating registration")

	t.mu.Lock()
	defer t.mu.Unlock()
	f := gitlab.File{
		Branch:      "main",
		AuthorEmail: fmt.Sprintf("%s@sparrow", t.name),
		AuthorName:  t.name,
		Content:     checks.GlobalTarget{Url: fmt.Sprintf("https://%s", t.name), LastSeen: time.Now().UTC()},
	}
	f.SetFileName(fmt.Sprintf("%s.json", t.name))

	if t.Registered() {
		f.CommitMessage = "Updated registration"
		err := t.gitlab.PutFile(ctx, f)
		if err != nil {
			log.Error("Failed to update registration", "error", err)
			return err
		}
		log.Debug("Successfully updated registration")
		return nil
	}

	f.CommitMessage = "Initial registration"
	err := t.gitlab.PostFile(ctx, f)
	if err != nil {
		log.Error("Failed to register global gitlabTargetManager", "error", err)
		return err
	}

	log.Debug("Successfully registered")
	t.registered = true
	return nil
}

// refreshTargets updates the targets of the gitlabTargetManager
// with the latest available healthy targets
func (t *gitlabTargetManager) refreshTargets(ctx context.Context) error {
	log := logger.FromContext(ctx)
	t.mu.Lock()
	defer t.mu.Unlock()
	var healthyTargets []checks.GlobalTarget
	targets, err := t.gitlab.FetchFiles(ctx)
	if err != nil {
		log.Error("Failed to update global targets", "error", err)
		return err
	}

	// filter unhealthy targets - this may be removed in the future
	for _, target := range targets {
		if !t.Registered() && target.Url == fmt.Sprintf("https://%s", t.name) {
			log.Debug("Found self as global target", "lastSeenMin", time.Since(target.LastSeen).Minutes())
			t.registered = true
		}
		if time.Now().Add(-t.unhealthyThreshold).After(target.LastSeen) {
			log.Debug("Skipping unhealthy target", "target", target)
			continue
		}
		healthyTargets = append(healthyTargets, target)
	}

	t.targets = healthyTargets
	log.Debug("Updated global targets", "targets", len(t.targets))
	return nil
}

// Registered returns whether the instance is registered as a global target
func (t *gitlabTargetManager) Registered() bool {
	return t.registered
}
