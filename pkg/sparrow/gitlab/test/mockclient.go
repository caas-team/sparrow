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

	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/checks/config"
	"github.com/caas-team/sparrow/pkg/sparrow/gitlab"
)

type MockClient struct {
	targets       []config.GlobalTarget
	fetchFilesErr error
	putFileErr    error
	postFileErr   error
	deleteFileErr error
}

func (m *MockClient) PutFile(ctx context.Context, _ gitlab.File) error { //nolint: gocritic // irrelevant
	log := logger.FromContext(ctx)
	log.Info("MockPutFile called", "err", m.putFileErr)
	return m.putFileErr
}

func (m *MockClient) PostFile(ctx context.Context, _ gitlab.File) error { //nolint: gocritic // irrelevant
	log := logger.FromContext(ctx)
	log.Info("MockPostFile called", "err", m.postFileErr)
	return m.postFileErr
}

func (m *MockClient) FetchFiles(ctx context.Context) ([]config.GlobalTarget, error) {
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

// New creates a new MockClient to mock Gitlab interaction
func New(targets []config.GlobalTarget) *MockClient {
	return &MockClient{
		targets: targets,
	}
}
