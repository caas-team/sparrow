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

package gitlabmock

import (
	"context"
	"sync"

	"github.com/caas-team/sparrow/pkg/checks"

	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/sparrow/gitlab"
)

type MockClient struct {
	targets        []checks.GlobalTarget
	mu             sync.Mutex
	fetchFilesErr  error
	putFileErr     error
	postFileErr    error
	deleteFileErr  error
	putFileCalled  bool
	postFileCalled bool
}

func (m *MockClient) PutFile(ctx context.Context, _ gitlab.File) error { //nolint: gocritic // irrelevant
	log := logger.FromContext(ctx)
	log.Info("MockPutFile called", "err", m.putFileErr)
	m.mu.Lock()
	m.putFileCalled = true
	m.mu.Unlock()
	return m.putFileErr
}

func (m *MockClient) PostFile(ctx context.Context, _ gitlab.File) error { //nolint: gocritic // irrelevant
	log := logger.FromContext(ctx)
	log.Info("MockPostFile called", "err", m.postFileErr)
	m.mu.Lock()
	m.postFileCalled = true
	m.mu.Unlock()
	return m.postFileErr
}

func (m *MockClient) FetchFiles(ctx context.Context) ([]checks.GlobalTarget, error) {
	log := logger.FromContext(ctx)
	log.Info("MockFetchFiles called", "targets", len(m.targets), "err", m.fetchFilesErr)
	return m.targets, m.fetchFilesErr
}

func (m *MockClient) DeleteFile(ctx context.Context, file gitlab.File) error { //nolint: gocritic // irrelevant
	log := logger.FromContext(ctx)
	log.Info("MockDeleteFile called", "filename", file, "err", m.deleteFileErr)
	return m.deleteFileErr
}

// SetFetchFilesErr sets the error returned by FetchFiles
func (m *MockClient) SetFetchFilesErr(err error) {
	m.fetchFilesErr = err
}

// SetPutFileErr sets the error returned by PutFile
func (m *MockClient) SetPutFileErr(err error) {
	m.putFileErr = err
}

// SetPostFileErr sets the error returned by PostFile
func (m *MockClient) SetPostFileErr(err error) {
	m.postFileErr = err
}

// SetDeleteFileErr sets the error returned by DeleteFile
func (m *MockClient) SetDeleteFileErr(err error) {
	m.deleteFileErr = err
}

// PutFileCalled returns true if PutFile was called
func (m *MockClient) PutFileCalled() bool {
	return m.putFileCalled
}

func (m *MockClient) PostFileCalled() bool {
	return m.postFileCalled
}

// New creates a new MockClient to mock Gitlab interaction
func New(targets []checks.GlobalTarget) *MockClient {
	return &MockClient{
		targets: targets,
	}
}
