package gitlabmock

import (
	"context"

	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/sparrow/gitlab"
)

type MockClient struct {
	targets       []checks.GlobalTarget
	fetchFilesErr error
	putFileErr    error
	postFileErr   error
}

func (m *MockClient) PutFile(ctx context.Context, _ gitlab.File) error { //nolint: gocritic // irrelevant
	log := logger.FromContext(ctx).With("name", "MockPutFile")
	log.Debug("MockPutFile called", "err", m.putFileErr)
	return m.putFileErr
}

func (m *MockClient) PostFile(ctx context.Context, _ gitlab.File) error { //nolint: gocritic // irrelevant
	log := logger.FromContext(ctx).With("name", "MockPostFile")
	log.Debug("MockPostFile called", "err", m.postFileErr)
	return m.postFileErr
}

func (m *MockClient) FetchFiles(ctx context.Context) ([]checks.GlobalTarget, error) {
	log := logger.FromContext(ctx).With("name", "MockFetchFiles")
	log.Debug("MockFetchFiles called", "targets", len(m.targets), "err", m.fetchFilesErr)
	return m.targets, m.fetchFilesErr
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

// New creates a new MockClient to mock Gitlab interaction
func New(targets []checks.GlobalTarget) *MockClient {
	return &MockClient{
		targets: targets,
	}
}
