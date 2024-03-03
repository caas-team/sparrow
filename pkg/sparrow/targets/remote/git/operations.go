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

	"github.com/go-git/go-billy/v5"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage"
)

// remoteOperator is the interface for the remote operations on the git repository
// All the remote operations should be performed using this interface
//
//go:generate moq -out operations_moq.go . remoteOperator
type remoteOperator interface {
	// CloneContext clones the git repository
	CloneContext(ctx context.Context, store storage.Storer, fs billy.Filesystem, o *git.CloneOptions) (*repository, error)
	// PullContext pulls the changes from the remote repository
	PullContext(ctx context.Context, w *git.Worktree, o *git.PullOptions) error
	// PushContext pushes the changes to the remote repository
	PushContext(ctx context.Context, r *repository, o *git.PushOptions) error
}

// repository is the wrapper around the go-git repository
type repository struct {
	// Repository is the go-git repository
	*git.Repository
	// remote is the operator for the remote operations
	// always use this to perform remote operations
	remote remoteOperator
}

// operator is the implementation of the RemoteOperator
type operator struct{}

// newRemoteOperator creates a new instance of the operator
func newRemoteOperator() remoteOperator {
	return &operator{}
}

// CloneContext clones the git repository
func (op *operator) CloneContext(ctx context.Context, store storage.Storer, fs billy.Filesystem, o *git.CloneOptions) (*repository, error) {
	r, err := git.CloneContext(ctx, store, fs, o)
	if err != nil {
		return nil, err
	}
	return &repository{Repository: r, remote: op}, nil
}

// PullContext pulls the changes from the remote repository
func (op *operator) PullContext(ctx context.Context, w *git.Worktree, o *git.PullOptions) error {
	return w.PullContext(ctx, o)
}

// PushContext pushes the changes to the remote repository
func (op *operator) PushContext(ctx context.Context, r *repository, o *git.PushOptions) error {
	return r.PushContext(ctx, o)
}
