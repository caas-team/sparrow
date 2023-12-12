package targets

import (
	"context"
	"net/http"
	"time"

	"github.com/caas-team/sparrow/internal/logger"
)

var (
	_ Gitlab        = &gitlab{}
	_ TargetManager = &gitlabTargetManager{}
)

// Gitlab handles interaction with a gitlab repository containing
// the global targets for the Sparrow instance
type Gitlab interface {
	ReadGlobalTargets(ctx context.Context) ([]globalTarget, error)
	RegisterSelf(ctx context.Context) error
}

// gitlabTargetManager implements TargetManager
type gitlabTargetManager struct {
	targets []globalTarget
	gitlab  Gitlab
	// the interval for the target reconciliation process
	checkInterval time.Duration
	// the amount of time a target can be
	// unhealthy before it is removed from the global target list
	unhealthyThreshold time.Duration
}

// NewGitlabManager creates a new gitlabTargetManager
func NewGitlabManager(g Gitlab, checkInterval, unhealthyThreshold time.Duration) TargetManager {
	return &gitlabTargetManager{
		targets:            []globalTarget{},
		gitlab:             g,
		checkInterval:      checkInterval,
		unhealthyThreshold: unhealthyThreshold,
	}
}

// gitlab implements Gitlab
type gitlab struct {
	url    string
	token  string
	client *http.Client
}

func NewGitlabClient(url, token string) Gitlab {
	return &gitlab{
		url:    url,
		token:  token,
		client: &http.Client{},
	}
}

func (t *gitlabTargetManager) Register(ctx context.Context) {
	log := logger.FromContext(ctx).With("name", "RegisterGlobalTargets")
	log.Debug("Registering global gitlabTargetManager")

	err := t.gitlab.RegisterSelf(ctx)
	if err != nil {
		log.Error("Failed to register global gitlabTargetManager", "error", err)
	}
}

// Reconcile reconciles the targets of the gitlabTargetManager.
// The global gitlabTargetManager are parsed from a remote endpoint.
//
// The global gitlabTargetManager are evaluated for healthiness and
// unhealthy gitlabTargetManager are removed.
func (t *gitlabTargetManager) Reconcile(ctx context.Context) {
	log := logger.FromContext(ctx).With("name", "ReconcileGlobalTargets")
	log.Debug("Starting global gitlabTargetManager reconciler")

	for {
		// start a timer
		timer := time.NewTimer(t.checkInterval)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				log.Error("Context canceled", "error", err)
				return
			}
		case <-timer.C:
			log.Debug("Getting global gitlabTargetManager")
			err := t.updateTargets(ctx)
			if err != nil {
				log.Error("Failed to get global gitlabTargetManager", "error", err)
				continue
			}
		}
	}
}

// GetTargets returns the current targets of the gitlabTargetManager
func (t *gitlabTargetManager) GetTargets() []globalTarget {
	return t.targets
}

// updateTargets sets the global gitlabTargetManager
func (t *gitlabTargetManager) updateTargets(ctx context.Context) error {
	log := logger.FromContext(ctx).With("name", "updateGlobalTargets")
	var healthyTargets []globalTarget

	targets, err := t.gitlab.ReadGlobalTargets(ctx)
	if err != nil {
		log.Error("Failed to update global targets", "error", err)
		return err
	}

	for _, target := range targets {
		if time.Now().Add(-t.unhealthyThreshold).After(target.lastSeen) {
			continue
		}
		healthyTargets = append(healthyTargets, target)
	}

	t.targets = healthyTargets
	log.Debug("Updated global targets", "targets", len(t.targets))
	return nil
}

// ReadGlobalTargets fetches the global gitlabTargetManager from the configured gitlab repository
func (g *gitlab) ReadGlobalTargets(ctx context.Context) ([]globalTarget, error) {
	log := logger.FromContext(ctx).With("name", "ReadGlobalTargets")
	log.Debug("Fetching global gitlabTargetManager")

	// TODO: pull file list from repo and marshal into []globalTarget

	return nil, nil
}

// RegisterSelf commits the current instance to the configured gitlab repository
// as a global target for other sparrow instances to discover
func (g *gitlab) RegisterSelf(ctx context.Context) error {
	log := logger.FromContext(ctx).With("name", "RegisterSelf")
	log.Debug("Registering sparrow instance to gitlab")

	// TODO: update & commit self as target to gitlab

	return nil
}
