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
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/sparrow/targets/remote"
	"github.com/go-git/go-billy/v5/memfs"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
)

// syncWithRemote ensures the local repository is up to date with the remote repository
func (g *client) syncWithRemote(ctx context.Context) error {
	log := logger.FromContext(ctx)

	if g.repo.Repository == nil {
		if err := g.cloneRepository(ctx); err != nil {
			log.Error("Failed to clone repository", "error", err)
			return err
		}
		return nil
	}

	w, err := g.repo.Worktree()
	if err != nil {
		log.Error("Failed to get worktree", "error", err)
		return err
	}

	err = w.Reset(&git.ResetOptions{Mode: git.HardReset})
	if err != nil {
		log.Error("Failed to reset the worktree", "error", err)
		return err
	}

	err = g.repo.remote.PullContext(ctx, w, &git.PullOptions{
		RemoteName: "origin",
		Auth:       g.auth,
		Depth:      1,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		log.Error("Failed to pull from repository", "error", err)
		return err
	}

	return nil
}

// cloneRepository clones the Git repository
func (g *client) cloneRepository(ctx context.Context) (err error) {
	repo, err := g.repo.remote.CloneContext(ctx, memory.NewStorage(), memfs.New(), &git.CloneOptions{
		URL:   g.repoURL,
		Auth:  g.auth,
		Depth: 1,
	})
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}
	g.repo = repo
	return nil
}

// pushChanges pushes the new commits to the remote repository
func (g *client) pushChanges(ctx context.Context) error {
	return g.repo.remote.PushContext(ctx, g.repo, &git.PushOptions{
		Auth:  g.auth,
		Force: true,
	})
}

// fetchAndProcessFiles fetches the files from the repository and processes the relevant ones into GlobalTargets
func (g *client) fetchAndProcessFiles(ctx context.Context) ([]checks.GlobalTarget, error) {
	log := logger.FromContext(ctx)

	tree, err := g.getLatestCommitTree()
	if err != nil {
		log.Error("Failed to get latest commit tree", "error", err)
		return nil, err
	}

	return g.processTreeFiles(ctx, tree)
}

// getLatestCommitTree gets the latest commit tree from the repository
func (g *client) getLatestCommitTree() (*object.Tree, error) {
	ref, err := g.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	commit, err := g.repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	t, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree: %w", err)
	}

	return t, nil
}

// processTreeFiles processes the files in the tree and returns the GlobalTargets
func (g *client) processTreeFiles(ctx context.Context, tree *object.Tree) ([]checks.GlobalTarget, error) {
	log := logger.FromContext(ctx)

	var result []checks.GlobalTarget
	err := tree.Files().ForEach(func(f *object.File) error {
		if !strings.HasSuffix(f.Name, ".json") {
			return nil
		}

		content, fErr := f.Contents()
		if fErr != nil {
			log.Error("Failed to read file contents", "file", f.Name, "error", fErr)
			return fErr
		}

		var gt checks.GlobalTarget
		if fErr = json.Unmarshal([]byte(content), &gt); fErr != nil {
			log.Error("Failed to unmarshal file", "file", f.Name, "error", fErr)
			return fErr
		}

		result = append(result, gt)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

type mode string

const (
	// modeAdd is the mode for adding a file
	modeAdd mode = "add"
	// modeDelete is the mode for deleting a file
	modeDelete mode = "delete"
)

// commitFile commits the file to the repository with the given mode
// The mode is used to determine if the file should be added or deleted
func (g *client) commitFile(ctx context.Context, file *remote.File, mode mode) error {
	log := logger.FromContext(ctx)

	w, err := g.repo.Worktree()
	if err != nil {
		log.Error("Failed to get worktree", "error", err)
		return err
	}

	if mode == modeDelete {
		err = g.deleteFile(w, file)
		if err != nil {
			log.Error("Failed to delete file", "error", err)
			return err
		}
		return nil
	}

	if err = g.writeFile(w, file); err != nil {
		log.Error("Failed to write file to worktree", "error", err)
		return err
	}

	return g.addFile(w, file)
}

// writeFile writes the file to the worktree of the repository
func (g *client) writeFile(w *git.Worktree, file *remote.File) (err error) {
	content, err := json.Marshal(file.Content)
	if err != nil {
		return fmt.Errorf("failed to marshal file content as json: %w", err)
	}

	path := filepath.Join(w.Filesystem.Root(), file.Name)
	f, err := w.Filesystem.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file in worktree: %w", err)
	}
	defer func() {
		if cErr := f.Close(); cErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to close file: %w", cErr))
		}
	}()

	if _, err := f.Write(content); err != nil {
		return fmt.Errorf("failed to write file content: %w", err)
	}

	return nil
}

// addFile stages and commits the file to the repository
func (g *client) addFile(w *git.Worktree, file *remote.File) error {
	if _, err := w.Add(file.Name); err != nil {
		return fmt.Errorf("failed to stage file: %w", err)
	}

	_, err := w.Commit(file.CommitMessage, &git.CommitOptions{
		Author: &object.Signature{
			Name:  file.AuthorName,
			Email: file.AuthorEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit file: %w", err)
	}

	return nil
}

// deleteFile removes the file from the worktree and commits the change
func (g *client) deleteFile(w *git.Worktree, file *remote.File) error {
	if _, err := w.Remove(file.Name); err != nil {
		return fmt.Errorf("failed to remove file: %w", err)
	}

	_, err := w.Commit(file.CommitMessage, &git.CommitOptions{
		Author: &object.Signature{
			Name:  file.AuthorName,
			Email: file.AuthorEmail,
			When:  time.Now(),
		},
		AllowEmptyCommits: true,
	})
	if err != nil {
		return fmt.Errorf("failed to commit file: %w", err)
	}
	return nil
}
