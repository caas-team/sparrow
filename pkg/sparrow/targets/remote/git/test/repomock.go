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

package repomock

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"testing"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
)

type Repository struct {
	*git.Repository
}

type File struct {
	Name    string
	Content string
}

func NewInMemory(t *testing.T) *Repository {
	fsys := memfs.New()
	storer := memory.NewStorage()

	r, err := git.Init(storer, fsys)
	if err != nil {
		t.Fatalf("Failed to initialize in-memory git repository: %v", err)
	}

	w, err := r.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree while initializing in-memory git repository: %v", err)
	}

	_, err = w.Commit("Initial commit", &git.CommitOptions{
		AllowEmptyCommits: true,
		Author: &object.Signature{
			Name:  "Test Author",
			Email: "author@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit to in-memory git repository: %v", err)
	}

	return &Repository{Repository: r}
}

// AddFiles adds files to the repository
func (r *Repository) AddFiles(files ...File) error {
	for _, f := range files {
		w, err := r.Worktree()
		if err != nil {
			return err
		}

		fl, err := w.Filesystem.Create(f.Name)
		if err != nil {
			return err
		}

		_, err = fl.Write([]byte(f.Content))
		if err != nil {
			return err
		}

		_, err = w.Add(f.Name)
		if err != nil {
			return err
		}

		_, err = w.Commit(fmt.Sprintf("Registered %s sparrow instance", f.Name), &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Test Author",
				Email: "author@example.com",
				When:  time.Now(),
			},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// GetFiles returns the files in the repository
func (r *Repository) GetFiles(t *testing.T) ([]File, error) {
	w, err := r.Worktree()
	if err != nil {
		return nil, err
	}

	files, err := w.Filesystem.ReadDir(".")
	if err != nil {
		return nil, err
	}

	var result []File
	for _, file := range files {
		t.Logf("Found file: %s", file.Name())
		b, err := readFile(w, file)
		if err != nil {
			return nil, err
		}

		result = append(result, File{
			Name:    file.Name(),
			Content: string(b),
		})
	}

	return result, nil
}

func readFile(w *git.Worktree, file fs.FileInfo) (content []byte, err error) {
	f, err := w.Filesystem.OpenFile(file.Name(), 0, 0)
	if err != nil {
		return nil, err
	}
	defer func() {
		cErr := f.Close()
		if cErr != nil {
			err = errors.Join(err, cErr)
		}
	}()

	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return b, nil
}
