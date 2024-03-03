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
	"testing"

	repomock "github.com/caas-team/sparrow/pkg/sparrow/targets/remote/git/test"
	"github.com/go-git/go-billy/v5"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage"
)

func TestClient_syncWithRemote(t *testing.T) {
	tests := []struct {
		name    string
		g       *client
		wantErr bool
	}{
		{
			name: "successfully clone repository",
			g: &client{
				repo: &repository{
					remote: &remoteOperatorMock{
						CloneContextFunc: func(ctx context.Context, store storage.Storer, fs billy.Filesystem, o *git.CloneOptions) (*repository, error) {
							return &repository{}, nil
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "failed to clone repository",
			g: &client{
				repo: &repository{
					remote: &remoteOperatorMock{
						CloneContextFunc: func(ctx context.Context, store storage.Storer, fs billy.Filesystem, o *git.CloneOptions) (*repository, error) {
							return nil, fmt.Errorf("failed to clone repository")
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "successfully pull from repository",
			g: &client{
				repo: &repository{
					Repository: repomock.NewInMemory(t).Repository,
					remote: &remoteOperatorMock{
						PullContextFunc: func(ctx context.Context, w *git.Worktree, o *git.PullOptions) error {
							return nil
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "failed to pull from repository",
			g: &client{
				repo: &repository{
					Repository: repomock.NewInMemory(t).Repository,
					remote: &remoteOperatorMock{
						PullContextFunc: func(ctx context.Context, w *git.Worktree, o *git.PullOptions) error {
							return fmt.Errorf("failed to pull from repository")
						},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.g.syncWithRemote(context.Background()); (err != nil) != tt.wantErr {
				t.Errorf("Client.syncWithRemote() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
