// sparrow
// (C) 2024, Deutsche Telekom IT GmbH
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

package git

import (
	"context"
	"fmt"

	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/sparrow/targets/remote"

	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

var _ remote.Interactor = (*client)(nil)

// client is the implementation of the remote.Interactor for git repositories
type client struct {
	// repoURL is the URL of the git repository
	repoURL string
	// auth is the authentication method used to interact with the repository
	auth *http.BasicAuth
	// repo is the git repository
	repo *repository
}

// Config contains the configuration for the git client
type Config struct {
	// RepoURL is the URL of the git repository
	RepoURL string `yaml:"repoUrl" mapstructure:"repoUrl"`
	// Token is the personal access token used to authenticate with the repository
	Token string `yaml:"token" mapstructure:"token"`
}

// New creates a new Git client
func New(cfg Config) remote.Interactor {
	return &client{
		repoURL: cfg.RepoURL,
		auth: &http.BasicAuth{
			Username: "sparrow",
			Password: cfg.Token,
		},
		repo: &repository{
			Repository: nil,
			remote:     newRemoteOperator(),
		},
	}
}

// FetchFiles fetches the files from the global targets repository
func (c *client) FetchFiles(ctx context.Context) ([]checks.GlobalTarget, error) {
	log := logger.FromContext(ctx)

	if err := c.syncWithRemote(ctx); err != nil {
		log.Error("Failed to sync local repository with remote", "error", err)
		return nil, err
	}

	tars, err := c.fetchAndProcessFiles(ctx)
	if err != nil {
		log.Error("Failed to fetch files and process them", "error", err)
		return nil, err
	}

	log.Info("Successfully fetched all target files", "files", len(tars))
	return tars, nil
}

// PutFile updates the file in the repository
func (c *client) PutFile(ctx context.Context, file remote.File) error { //nolint:gocritic // no performance concerns yet
	log := logger.FromContext(ctx)
	log.Debug("Updating file in repository", "file", file)

	return c.CommitFile(ctx, &file)
}

// PostFile creates the file in the repository
func (c *client) PostFile(ctx context.Context, file remote.File) error { //nolint:gocritic // no performance concerns yet
	log := logger.FromContext(ctx)
	log.Debug("Creating new file in repository", "file", file.Name)

	return c.CommitFile(ctx, &file)
}

// CommitFile commits the file to the repository
func (c *client) CommitFile(ctx context.Context, file *remote.File) error {
	log := logger.FromContext(ctx)

	if err := c.syncWithRemote(ctx); err != nil {
		log.Error("Failed to sync local repository with remote", "error", err)
		return err
	}

	if err := c.commitFile(ctx, file, modeAdd); err != nil {
		log.Error("Failed to process file for commit", "error", err)
		return err
	}

	if err := c.pushChanges(ctx); err != nil {
		log.Error("Failed to push changes", "error", err)
		return err
	}

	log.Info("File committed and pushed", "file", file.Name)
	return nil
}

// DeleteFile deletes the file from the repository
func (c *client) DeleteFile(ctx context.Context, file remote.File) error { //nolint:gocritic // no performance concerns yet
	log := logger.FromContext(ctx).With("file", file)

	if file.Name == "" {
		return fmt.Errorf("filename is empty")
	}

	// Ensure repository is updated with remote
	if err := c.syncWithRemote(ctx); err != nil {
		log.Error("Failed to sync local repository with remote", "error", err)
		return err
	}

	if err := c.commitFile(ctx, &file, modeDelete); err != nil {
		log.Error("Failed to process file for commit", "error", err)
		return err
	}

	if err := c.pushChanges(ctx); err != nil {
		log.Error("Failed to push changes", "error", err)
		return err
	}

	log.Info("File deleted, committed and pushed successfully", "file", file.Name)
	return nil
}
