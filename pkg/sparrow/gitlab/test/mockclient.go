package gitlabmock

import (
	"context"
	"fmt"

	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/sparrow/gitlab"
)

type MockClient struct {
	targets []checks.GlobalTarget
	err     error
}

func (m MockClient) PutFile(ctx context.Context, file gitlab.File) error {
	panic("implement me")
}

func (m MockClient) PostFile(ctx context.Context, f gitlab.File) error {
	panic("implement me")
}

func (m MockClient) FetchFiles(ctx context.Context) ([]checks.GlobalTarget, error) {
	return m.targets, m.err
}

// New creates a new MockClient to mock Gitlab interaction
func New(targets []checks.GlobalTarget, err bool) gitlab.Gitlab {
	var e error
	if err {
		e = fmt.Errorf("error")
	}
	return &MockClient{
		targets: targets,
		err:     e,
	}
}
