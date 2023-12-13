package targets

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
	FetchFiles(ctx context.Context) ([]globalTarget, error)
	PutFile(ctx context.Context, file GitlabFile) error
	PostFile(ctx context.Context, file GitlabFile) error
}

// gitlab implements Gitlab
type gitlab struct {
	// the base URL of the gitlab instance
	baseUrl string
	// the ID of the project containing the global targets
	projectID int
	// the token used to authenticate with the gitlab instance
	token  string
	client *http.Client
}

// gitlabTargetManager implements TargetManager
type gitlabTargetManager struct {
	targets []globalTarget
	gitlab  Gitlab
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
func NewGitlabManager(g Gitlab, checkInterval, unhealthyThreshold time.Duration) TargetManager {
	return &gitlabTargetManager{
		targets:            []globalTarget{},
		gitlab:             g,
		checkInterval:      checkInterval,
		unhealthyThreshold: unhealthyThreshold,
	}
}

// file represents a file in a gitlab repository
type file struct {
	Name string `json:"name"`
}

func NewGitlab(url, token string) Gitlab {
	return &gitlab{
		baseUrl: url,
		token:   token,
		client:  &http.Client{},
	}
}

// updateRegistration registers the current instance as a global target
func (t *gitlabTargetManager) updateRegistration(ctx context.Context) error {
	log := logger.FromContext(ctx).With("name", t.name, "registered", t.registered)
	log.Debug("Updating registration")

	f := GitlabFile{
		Branch:      "main",
		AuthorEmail: fmt.Sprintf("%s@sparrow", t.name),
		AuthorName:  t.name,
		Content:     globalTarget{Url: fmt.Sprintf("https://%s", t.name), LastSeen: time.Now().UTC()},
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
	}

	log.Debug("Successfully registered")
	t.registered = true
	return nil
}

// Reconcile reconciles the targets of the gitlabTargetManager.
// The global targets are parsed from a remote endpoint.
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
func (t *gitlabTargetManager) GetTargets() []globalTarget {
	return t.targets
}

// refreshTargets updates the targets of the gitlabTargetManager
// with the latest available healthy targets
func (t *gitlabTargetManager) refreshTargets(ctx context.Context) error {
	log := logger.FromContext(ctx).With("name", "updateGlobalTargets")
	var healthyTargets []globalTarget

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

// FetchFiles fetches the files from the global targets repository from the configured gitlab repository
func (g *gitlab) FetchFiles(ctx context.Context) ([]globalTarget, error) {
	log := logger.FromContext(ctx).With("name", "FetchFiles")
	fl, err := g.fetchFileList(ctx)
	if err != nil {
		log.Error("Failed to fetch files", "error", err)
		return nil, err
	}

	result, err := g.fetchFiles(ctx, fl)
	if err != nil {
		log.Error("Failed to fetch files", "error", err)
		return nil, err
	}
	log.Info("Successfully fetched all target files", "files", len(result))
	return result, nil
}

// fetchFiles fetches the files from the global targets repository from the configured gitlab repository
func (g *gitlab) fetchFiles(ctx context.Context, fl []string) ([]globalTarget, error) {
	var result []globalTarget
	log := logger.FromContext(ctx).With("name", "fetchFiles")
	log.Debug("Fetching global files")
	for _, f := range fl {
		// URL encode the name
		n := url.PathEscape(f)
		req, err := http.NewRequestWithContext(ctx,
			http.MethodGet,
			fmt.Sprintf("%s/api/v4/projects/%d/repository/files/%s/raw?ref=main", g.baseUrl, g.projectID, n),
			http.NoBody,
		)
		if err != nil {
			log.Error("Failed to create request", "error", err)
			return nil, err
		}
		req.Header.Add("PRIVATE-TOKEN", g.token)
		req.Header.Add("Content-Type", "application/json")

		res, err := g.client.Do(req)
		if err != nil {
			log.Error("Failed to fetch file", "file", f, "error", err)
			return nil, err
		}
		if res.StatusCode != http.StatusOK {
			log.Error("Failed to fetch file", "status", res.Status)
			return nil, fmt.Errorf("request failed, status is %s", res.Status)
		}

		defer res.Body.Close()
		var gt globalTarget
		err = json.NewDecoder(res.Body).Decode(&gt)
		if err != nil {
			log.Error("Failed to decode file after fetching", "file", f, "error", err)
			return nil, err
		}

		log.Debug("Successfully fetched file", "file", f)
		result = append(result, gt)
	}
	return result, nil
}

// fetchFileList fetches the files from the global targets repository from the configured gitlab repository
func (g *gitlab) fetchFileList(ctx context.Context) ([]string, error) {
	log := logger.FromContext(ctx).With("name", "fetchFileList")
	log.Debug("Fetching global files")
	type file struct {
		Name string `json:"name"`
	}

	req, err := http.NewRequestWithContext(ctx,
		http.MethodGet,
		fmt.Sprintf("%s/api/v4/projects/%d/repository/tree?ref=main", g.baseUrl, g.projectID),
		http.NoBody,
	)
	if err != nil {
		log.Error("Failed to create request", "error", err)
		return nil, err
	}

	req.Header.Add("PRIVATE-TOKEN", g.token)
	req.Header.Add("Content-Type", "application/json")

	res, err := g.client.Do(req)
	if err != nil {
		log.Error("Failed to fetch file list", "error", err)
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		log.Error("Failed to fetch file list", "status", res.Status)
		return nil, fmt.Errorf("request failed, status is %s", res.Status)
	}

	defer res.Body.Close()
	var fl []file
	err = json.NewDecoder(res.Body).Decode(&fl)
	if err != nil {
		log.Error("Failed to decode file list", "error", err)
		return nil, err
	}

	var result []string
	for _, f := range fl {
		result = append(result, f.Name)
	}

	log.Debug("Successfully fetched file list", "files", len(result))
	return result, nil
}

type GitlabFile struct {
	Branch        string       `json:"branch"`
	AuthorEmail   string       `json:"author_email"`
	AuthorName    string       `json:"author_name"`
	Content       globalTarget `json:"content"`
	CommitMessage string       `json:"commit_message"`
	fileName      string
}

// Bytes returns the bytes of the GitlabFile
func (g GitlabFile) Bytes() ([]byte, error) {
	b, err := json.Marshal(g)
	return b, err
}

// PutFile commits the current instance to the configured gitlab repository
// as a global target for other sparrow instances to discover
func (g *gitlab) PutFile(ctx context.Context, body GitlabFile) error {
	log := logger.FromContext(ctx).With("name", "AddRegistration")
	log.Debug("Registering sparrow instance to gitlab")

	// chose method based on whether the registration has already happened
	n := url.PathEscape(body.Content.Url)
	b, err := body.Bytes()
	if err != nil {
		log.Error("Failed to create request", "error", err)
		return err
	}
	req, err := http.NewRequestWithContext(ctx,
		http.MethodGet,
		fmt.Sprintf("%s/api/v4/projects/%d/repository/files/%s", g.baseUrl, g.projectID, n),
		bytes.NewBuffer(b),
	)
	if err != nil {
		log.Error("Failed to create request", "error", err)
		return err
	}

	req.Header.Add("PRIVATE-TOKEN", g.token)
	req.Header.Add("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		log.Error("Failed to push registration file", "error", err)
		return err
	}

	if resp.StatusCode != http.StatusAccepted {
		log.Error("Failed to push registration file", "status", resp.Status)
		return fmt.Errorf("request failed, status is %s", resp.Status)
	}

	return nil
}

// PostFile commits the current instance to the configured gitlab repository
// as a global target for other sparrow instances to discover
func (g *gitlab) PostFile(ctx context.Context, body GitlabFile) error {
	log := logger.FromContext(ctx).With("name", "AddRegistration")
	log.Debug("Registering sparrow instance to gitlab")

	// chose method based on whether the registration has already happened
	n := url.PathEscape(body.Content.Url)
	b, err := body.Bytes()
	if err != nil {
		log.Error("Failed to create request", "error", err)
		return err
	}
	req, err := http.NewRequestWithContext(ctx,
		http.MethodPost,
		fmt.Sprintf("%s/api/v4/projects/%d/repository/files/%s", g.baseUrl, g.projectID, n),
		bytes.NewBuffer(b),
	)
	if err != nil {
		log.Error("Failed to create request", "error", err)
		return err
	}

	req.Header.Add("PRIVATE-TOKEN", g.token)
	req.Header.Add("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		log.Error("Failed to push registration file", "error", err)
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		log.Error("Failed to push registration file", "status", resp.Status)
		return fmt.Errorf("request failed, status is %s", resp.Status)
	}

	return nil
}
