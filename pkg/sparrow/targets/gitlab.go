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
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/caas-team/sparrow/pkg/checks"

	"github.com/caas-team/sparrow/pkg/sparrow/gitlab"

	"github.com/caas-team/sparrow/internal/logger"
)

var _ TargetManager = &gitlabTargetManager{}

const shutdownTimeout = 30 * time.Second

// gitlabTargetManager implements TargetManager
type gitlabTargetManager struct {
	targets []checks.GlobalTarget
	mu      sync.RWMutex
	// channel to signal the "reconcile" routine to stop
	done chan struct{}
	// the DNS name used for self-registration
	name string
	// whether the instance has already registered itself as a global target
	registered bool
	cfg        Config
	gitlab     gitlab.Gitlab
}

type GitlabTargetManagerConfig struct {
	BaseURL   string `yaml:"baseUrl" mapstructure:"baseUrl"`
	Token     string `yaml:"token" mapstructure:"token"`
	ProjectID int    `yaml:"projectId" mapstructure:"projectId"`
}

// NewGitlabManager creates a new gitlabTargetManager
func NewGitlabManager(name string, gtmConfig TargetManagerConfig) *gitlabTargetManager {
	return &gitlabTargetManager{
		gitlab: gitlab.New(gtmConfig.Gitlab.BaseURL, gtmConfig.Gitlab.Token, gtmConfig.Gitlab.ProjectID),
		name:   name,
		cfg:    gtmConfig.Config,
		mu:     sync.RWMutex{},
		done:   make(chan struct{}, 1),
	}
}

// Reconcile reconciles the targets of the gitlabTargetManager.
// The global targets are parsed from a gitlab repository.
//
// The global targets are evaluated for healthiness and
// unhealthy gitlabTargetManager are removed.
func (t *gitlabTargetManager) Reconcile(ctx context.Context) error {
	log := logger.FromContext(ctx)
	log.Info("Starting global gitlabTargetManager reconciler")

	checkTimer := startTimer(t.cfg.CheckInterval)
	registrationTimer := startTimer(t.cfg.RegistrationInterval)
	updateTimer := startTimer(t.cfg.UpdateInterval)

	defer checkTimer.Stop()
	defer registrationTimer.Stop()
	defer updateTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Error("Error while reconciling gitlab targets", "err", ctx.Err())
			return ctx.Err()
		case <-t.done:
			log.Info("Gitlab target reconciliation ended")
			return nil
		case <-checkTimer.C:
			err := t.refreshTargets(ctx)
			if err != nil {
				log.Warn("Failed to get global targets", "error", err)
			}
			checkTimer.Reset(t.cfg.CheckInterval)
		case <-registrationTimer.C:
			err := t.register(ctx)
			if err != nil {
				log.Warn("Failed to register self as global target", "error", err)
			}
			registrationTimer.Reset(t.cfg.RegistrationInterval)
		case <-updateTimer.C:
			err := t.update(ctx)
			if err != nil {
				log.Warn("Failed to update registration", "error", err)
			}
			updateTimer.Reset(t.cfg.UpdateInterval)
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

	errC := ctx.Err()
	log := logger.FromContext(ctx)
	log.Debug("Shut down signal received")
	ctxS, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if t.Registered() {
		f := gitlab.File{
			Branch:        "main",
			AuthorEmail:   fmt.Sprintf("%s@sparrow", t.name),
			AuthorName:    t.name,
			CommitMessage: "Unregistering global target",
		}
		f.SetFileName(fmt.Sprintf("%s.json", t.name))
		err := t.gitlab.DeleteFile(ctxS, f)
		if err != nil {
			log.Error("Failed to shutdown gracefully", "error", err)
			return fmt.Errorf("failed to shutdown gracefully: %w", errors.Join(errC, err))
		}
		t.registered = false
	}

	select {
	case t.done <- struct{}{}:
		log.Debug("Stopping gitlab reconciliation routine")
	default:
	}

	return nil
}

// register registers the current instance as a global target
// in the gitlab repository
func (t *gitlabTargetManager) register(ctx context.Context) error {
	log := logger.FromContext(ctx)
	log.Debug("Registering as global target")

	t.mu.Lock()
	defer t.mu.Unlock()
	f := gitlab.File{
		Branch:        "main",
		AuthorEmail:   fmt.Sprintf("%s@sparrow", t.name),
		AuthorName:    t.name,
		CommitMessage: "Initial registration",
		Content:       checks.GlobalTarget{Url: fmt.Sprintf("https://%s", t.name), LastSeen: time.Now().UTC()},
	}
	f.SetFileName(fmt.Sprintf("%s.json", t.name))

	err := t.gitlab.PostFile(ctx, f)
	if err != nil {
		log.Error("Failed to register global gitlabTargetManager", "error", err)
		return err
	}

	log.Debug("Successfully registered")
	t.registered = true
	return nil
}

// update updates the registration file of the current sparrow instance
// in the gitlab repository
func (t *gitlabTargetManager) update(ctx context.Context) error {
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
		if time.Now().Add(-t.cfg.UnhealthyThreshold).After(target.LastSeen) {
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

// startTimer creates a new timer with the given duration.
// If the duration is 0, the timer is stopped.
func startTimer(d time.Duration) *time.Timer {
	res := time.NewTimer(d)
	if d == 0 {
		res.Stop()
	}
	return res
}
