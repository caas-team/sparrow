package targets

import (
	"context"
	"fmt"
	"time"

	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/sparrow/gitlab"

	"github.com/caas-team/sparrow/internal/logger"
)

var _ TargetManager = &gitlabTargetManager{}

// gitlabTargetManager implements TargetManager
type gitlabTargetManager struct {
	targets []checks.GlobalTarget
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
func NewGitlabManager(g gitlab.Gitlab, checkInterval, unhealthyThreshold time.Duration) TargetManager {
	return &gitlabTargetManager{
		gitlab:             g,
		checkInterval:      checkInterval,
		unhealthyThreshold: unhealthyThreshold,
	}
}

// updateRegistration registers the current instance as a global target
func (t *gitlabTargetManager) updateRegistration(ctx context.Context) error {
	log := logger.FromContext(ctx).With("name", t.name, "registered", t.registered)
	log.Debug("Updating registration")

	f := gitlab.File{
		Branch:      "main",
		AuthorEmail: fmt.Sprintf("%s@sparrow", t.name),
		AuthorName:  t.name,
		Content:     checks.GlobalTarget{Url: fmt.Sprintf("https://%s", t.name), LastSeen: time.Now().UTC()},
	}

	if t.registered {
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

// Reconcile reconciles the targets of the gitlabTargetManager.
// The global targets are parsed from a gitlab repository.
//
// The global targets are evaluated for healthiness and
// unhealthy gitlabTargetManager are removed.
func (t *gitlabTargetManager) Reconcile(ctx context.Context) {
	log := logger.FromContext(ctx).With("name", "ReconcileGlobalTargets")
	log.Debug("Starting global gitlabTargetManager reconciler")

	for {
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				log.Error("Context canceled", "error", err)
				return
			}
			// check if this blocks when context is canceled
		case <-time.After(t.checkInterval):
			log.Debug("Getting global gitlabTargetManager")
			err := t.refreshTargets(ctx)
			if err != nil {
				log.Error("Failed to get global gitlabTargetManager", "error", err)
				continue
			}
		case <-time.After(t.registrationInterval):
			log.Debug("Registering global gitlabTargetManager")
			err := t.updateRegistration(ctx)
			if err != nil {
				log.Error("Failed to register global gitlabTargetManager", "error", err)
				continue
			}
		}
	}
}

// GetTargets returns the current targets of the gitlabTargetManager
func (t *gitlabTargetManager) GetTargets() []checks.GlobalTarget {
	return t.targets
}

// refreshTargets updates the targets of the gitlabTargetManager
// with the latest available healthy targets
func (t *gitlabTargetManager) refreshTargets(ctx context.Context) error {
	log := logger.FromContext(ctx).With("name", "updateGlobalTargets")
	var healthyTargets []checks.GlobalTarget

	targets, err := t.gitlab.FetchFiles(ctx)
	if err != nil {
		log.Error("Failed to update global targets", "error", err)
		return err
	}

	// filter unhealthy targets - this may be removed in the future
	for _, target := range targets {
		if time.Now().Add(-t.unhealthyThreshold).After(target.LastSeen) {
			continue
		}
		healthyTargets = append(healthyTargets, target)
	}

	t.targets = healthyTargets
	log.Debug("Updated global targets", "targets", len(t.targets))
	return nil
}
