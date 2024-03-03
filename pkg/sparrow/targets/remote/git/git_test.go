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
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/sparrow/targets/remote"
	repomock "github.com/caas-team/sparrow/pkg/sparrow/targets/remote/git/test"
	"github.com/go-git/go-billy/v5"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage"
)

func Test_FetchFiles(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		operator *remoteOperatorMock
		want     []checks.GlobalTarget
		wantErr  bool
	}{
		{
			name: "success with 0 targets",
			operator: &remoteOperatorMock{
				CloneContextFunc: func(ctx context.Context, store storage.Storer, fs billy.Filesystem, o *git.CloneOptions) (*repository, error) {
					rm := repomock.NewInMemory(t)
					return &repository{
						Repository: rm.Repository,
						remote:     &remoteOperatorMock{},
					}, nil
				},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "success with 1 target",
			operator: &remoteOperatorMock{
				CloneContextFunc: func(ctx context.Context, store storage.Storer, fs billy.Filesystem, o *git.CloneOptions) (*repository, error) {
					rm := repomock.NewInMemory(t)
					files := []repomock.File{
						{
							Name:    "example.com.json",
							Content: fmt.Sprintf(`{"url": "example.com", "lastSeen": "%v"}`, time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)),
						},
					}
					if err := rm.AddFiles(files...); err != nil {
						t.Fatalf("Failed to add file to local repo: %s", err)
					}
					return &repository{
						Repository: rm.Repository,
						remote:     &remoteOperatorMock{},
					}, nil
				},
			},
			want: []checks.GlobalTarget{
				{
					Url:      "example.com",
					LastSeen: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "success with 2 targets",
			operator: &remoteOperatorMock{
				CloneContextFunc: func(ctx context.Context, store storage.Storer, fs billy.Filesystem, o *git.CloneOptions) (*repository, error) {
					rm := repomock.NewInMemory(t)
					files := []repomock.File{
						{
							Name:    "first.example.com.json",
							Content: fmt.Sprintf(`{"url": "first.example.com", "lastSeen": "%v"}`, time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)),
						},
						{
							Name:    "second.example.com.json",
							Content: fmt.Sprintf(`{"url": "second.example.com", "lastSeen": "%v"}`, time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)),
						},
					}
					if err := rm.AddFiles(files...); err != nil {
						t.Fatalf("Failed to add file to local repo: %s", err)
					}
					return &repository{
						Repository: rm.Repository,
						remote:     &remoteOperatorMock{},
					}, nil
				},
			},
			want: []checks.GlobalTarget{
				{
					Url:      "first.example.com",
					LastSeen: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					Url:      "second.example.com",
					LastSeen: time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			wantErr: false,
		},
		{
			name: "clone fails",
			operator: &remoteOperatorMock{
				CloneContextFunc: func(ctx context.Context, store storage.Storer, fs billy.Filesystem, o *git.CloneOptions) (*repository, error) {
					return nil, fmt.Errorf("clone error")
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid JSON format in file",
			operator: &remoteOperatorMock{
				CloneContextFunc: func(ctx context.Context, store storage.Storer, fs billy.Filesystem, o *git.CloneOptions) (*repository, error) {
					rm := repomock.NewInMemory(t)
					files := []repomock.File{
						{
							Name:    "invalid.json",
							Content: "invalid JSON",
						},
					}
					if err := rm.AddFiles(files...); err != nil {
						t.Fatalf("Failed to add file with invalid JSON to local repo: %s", err)
					}
					return &repository{
						Repository: rm.Repository,
						remote:     &remoteOperatorMock{},
					}, nil
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newCommonClient(tt.operator)

			got, err := client.FetchFiles(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.FetchFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Client.FetchFiles() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_CommitFile(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		operator func(r *repomock.Repository) *remoteOperatorMock
		file     *remote.File
		want     struct {
			FileName string
			Content  checks.GlobalTarget
		}
		wantErr bool
	}{
		{
			name: "create new file",
			operator: func(r *repomock.Repository) *remoteOperatorMock {
				op := &remoteOperatorMock{
					PushContextFunc: func(ctx context.Context, r *repository, o *git.PushOptions) error {
						return nil
					},
				}
				op.CloneContextFunc = func(ctx context.Context, store storage.Storer, fs billy.Filesystem, o *git.CloneOptions) (*repository, error) {
					return &repository{
						Repository: r.Repository,
						remote:     op,
					}, nil
				}
				return op
			},
			file: &remote.File{
				Branch:        "master",
				AuthorEmail:   "example@sparrow",
				AuthorName:    "sparrow.com",
				CommitMessage: "Initial registration",
				Content:       checks.GlobalTarget{Url: "https://sparrow.com", LastSeen: time.Now().UTC()},
				Name:          "sparrow.com.json",
			},
			want: struct {
				FileName string
				Content  checks.GlobalTarget
			}{
				FileName: "sparrow.com.json",
				Content:  checks.GlobalTarget{Url: "https://sparrow.com", LastSeen: time.Now().UTC()},
			},
			wantErr: false,
		},
		{
			name: "update existing file",
			operator: func(r *repomock.Repository) *remoteOperatorMock { //nolint:dupl // keeping it for readability
				err := r.AddFiles(repomock.File{
					Name:    "existing_file.json",
					Content: fmt.Sprintf(`{"url": "initial.com", "lastSeen": %q}`, time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)),
				})
				if err != nil {
					t.Fatalf("Failed to add existing file: %s", err)
				}

				op := &remoteOperatorMock{
					PushContextFunc: func(ctx context.Context, r *repository, o *git.PushOptions) error {
						return nil
					},
				}
				op.CloneContextFunc = func(ctx context.Context, store storage.Storer, fs billy.Filesystem, o *git.CloneOptions) (*repository, error) {
					return &repository{
						Repository: r.Repository,
						remote:     op,
					}, nil
				}
				return op
			},
			file: &remote.File{
				Name:          "existing_file.json",
				Content:       checks.GlobalTarget{Url: "https://updated.com", LastSeen: time.Now().UTC()},
				CommitMessage: "Update existing file",
			},
			want: struct {
				FileName string
				Content  checks.GlobalTarget
			}{
				FileName: "existing_file.json",
				Content:  checks.GlobalTarget{Url: "https://updated.com", LastSeen: time.Now().UTC()},
			},
			wantErr: false,
		},
		{
			name: "push fails",
			operator: func(r *repomock.Repository) *remoteOperatorMock {
				op := &remoteOperatorMock{
					PushContextFunc: func(ctx context.Context, r *repository, o *git.PushOptions) error {
						return fmt.Errorf("push error")
					},
				}
				op.CloneContextFunc = func(ctx context.Context, store storage.Storer, fs billy.Filesystem, o *git.CloneOptions) (*repository, error) {
					return &repository{
						Repository: r.Repository,
						remote:     op,
					}, nil
				}
				return op
			},
			file: &remote.File{
				Branch:        "master",
				AuthorEmail:   "example@sparrow",
				AuthorName:    "sparrow.com",
				CommitMessage: "Initial registration",
				Content:       checks.GlobalTarget{Url: "https://sparrow.com", LastSeen: time.Now().UTC()},
				Name:          "sparrow.com.json",
			},
			wantErr: true,
		},
		{
			name: "clone fails",
			operator: func(r *repomock.Repository) *remoteOperatorMock {
				return &remoteOperatorMock{
					CloneContextFunc: func(ctx context.Context, store storage.Storer, fs billy.Filesystem, o *git.CloneOptions) (*repository, error) {
						return nil, fmt.Errorf("clone error")
					},
				}
			},
			file: &remote.File{
				Branch:        "master",
				AuthorEmail:   "example@sparrow",
				AuthorName:    "sparrow.com",
				CommitMessage: "Initial registration",
				Content:       checks.GlobalTarget{Url: "https://sparrow.com", LastSeen: time.Now().UTC()},
				Name:          "sparrow.com.json",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm := repomock.NewInMemory(t)
			client := newCommonClient(tt.operator(rm))

			if err := client.CommitFile(ctx, tt.file); (err != nil) != tt.wantErr {
				t.Errorf("Client.CommitFile() error = %v, wantErr %v", err, tt.wantErr)
			}

			files, err := rm.GetFiles(t)
			if err != nil {
				t.Fatalf("Failed to get files from mock repo: %s", err)
			}
			for _, f := range files {
				if f.Name == tt.want.FileName {
					var got checks.GlobalTarget
					err := json.Unmarshal([]byte(f.Content), &got)
					if err != nil {
						t.Fatalf("Failed to unmarshal file content: %s", err)
					}
					if got.Url != tt.want.Content.Url {
						t.Errorf("Client.CommitFile() = %v, want %v", got.Url, tt.want.Content.Url)
					}
				}
			}
		})
	}
}

func Test_DeleteFile(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		operator func(r *repomock.Repository) *remoteOperatorMock
		file     remote.File
		want     string
		wantErr  bool
	}{
		{
			name: "delete existing file",
			operator: func(r *repomock.Repository) *remoteOperatorMock { //nolint:dupl // keeping it for readability
				err := r.AddFiles(repomock.File{
					Name:    "existing_file.json",
					Content: fmt.Sprintf(`{"url": "initial.com", "lastSeen": %q}`, time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)),
				})
				if err != nil {
					t.Fatalf("Failed to add existing file: %s", err)
				}

				op := &remoteOperatorMock{
					PushContextFunc: func(ctx context.Context, r *repository, o *git.PushOptions) error {
						return nil
					},
				}
				op.CloneContextFunc = func(ctx context.Context, store storage.Storer, fs billy.Filesystem, o *git.CloneOptions) (*repository, error) {
					return &repository{
						Repository: r.Repository,
						remote:     op,
					}, nil
				}
				return op
			},
			file: remote.File{
				Branch:        "master",
				AuthorEmail:   "example@sparrow",
				AuthorName:    "sparrow.com",
				CommitMessage: "Delete existing file",
				Content:       checks.GlobalTarget{},
				Name:          "existing_file.json",
			},
			want:    "existing_file.json",
			wantErr: false,
		},
		{
			name: "delete non-existing file",
			operator: func(r *repomock.Repository) *remoteOperatorMock {
				op := &remoteOperatorMock{
					PushContextFunc: func(ctx context.Context, r *repository, o *git.PushOptions) error {
						return nil
					},
				}
				op.CloneContextFunc = func(ctx context.Context, store storage.Storer, fs billy.Filesystem, o *git.CloneOptions) (*repository, error) {
					return &repository{
						Repository: r.Repository,
						remote:     op,
					}, nil
				}
				return op
			},
			file: remote.File{
				Branch:        "master",
				AuthorEmail:   "example@sparrow",
				AuthorName:    "sparrow.com",
				Name:          "non_existing_file.json",
				CommitMessage: "Delete non-existing file",
			},
			want:    "non_existing_file.json",
			wantErr: true,
		},
		{
			name: "push fails",
			operator: func(r *repomock.Repository) *remoteOperatorMock {
				err := r.AddFiles(repomock.File{
					Name:    "existing_file.json",
					Content: fmt.Sprintf(`{"url": "initial.com", "lastSeen": %q}`, time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)),
				})
				if err != nil {
					t.Fatalf("Failed to add existing file: %s", err)
				}

				op := &remoteOperatorMock{
					PushContextFunc: func(ctx context.Context, r *repository, o *git.PushOptions) error {
						return fmt.Errorf("push error")
					},
				}
				op.CloneContextFunc = func(ctx context.Context, store storage.Storer, fs billy.Filesystem, o *git.CloneOptions) (*repository, error) {
					return &repository{
						Repository: r.Repository,
						remote:     op,
					}, nil
				}
				return op
			},
			file: remote.File{
				Branch:        "master",
				AuthorEmail:   "example@sparrow",
				AuthorName:    "sparrow.com",
				CommitMessage: "Delete existing file",
				Content:       checks.GlobalTarget{},
				Name:          "existing_file.json",
			},
			wantErr: true,
		},
		{
			name: "clone fails",
			operator: func(r *repomock.Repository) *remoteOperatorMock {
				return &remoteOperatorMock{
					CloneContextFunc: func(ctx context.Context, store storage.Storer, fs billy.Filesystem, o *git.CloneOptions) (*repository, error) {
						return nil, fmt.Errorf("clone error")
					},
				}
			},
			file: remote.File{
				Name: "existing_file.json",
			},
			wantErr: true,
		},
		{
			name: "empty filename",
			operator: func(r *repomock.Repository) *remoteOperatorMock {
				return &remoteOperatorMock{}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm := repomock.NewInMemory(t)
			client := newCommonClient(tt.operator(rm))

			if err := client.DeleteFile(ctx, tt.file); (err != nil) != tt.wantErr {
				t.Errorf("Client.DeleteFile() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				files, err := rm.GetFiles(t)
				if err != nil {
					t.Fatalf("Failed to get files from mock repo: %s", err)
				}
				for _, f := range files {
					if f.Name == tt.want {
						t.Errorf("Client.DeleteFile() = %v, want %v", f.Name, tt.want)
					}
				}
			}
		})
	}
}

func newCommonClient(op remoteOperator) *client {
	return &client{
		repoURL: "https://git.example.com/repo.git",
		auth:    &http.BasicAuth{Password: ""},
		repo:    &repository{remote: op},
	}
}
