package framework

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/caas-team/sparrow/pkg/config"
	"github.com/caas-team/sparrow/pkg/sparrow"
)

type Framework struct {
	t *testing.T
}

func New(t *testing.T) *Framework {
	return &Framework{t: t}
}

type E2ETest struct {
	t       *testing.T
	sparrow *sparrow.Sparrow
	buf     bytes.Buffer
	path    string
}

func (f *Framework) E2E(cfg *config.Config) *E2ETest {
	if cfg == nil {
		cfg = NewConfig().Config(f.t)
	}

	return &E2ETest{
		t:       f.t,
		sparrow: sparrow.New(cfg),
	}
}

func (t *E2ETest) WithConfigFile(path string) *E2ETest {
	t.path = path
	return t
}

func (t *E2ETest) WithCheck(builder CheckBuilder) *E2ETest {
	t.buf.Write(builder.YAML(t.t))
	return t
}

// Run runs the test.
// Runs indefinitely until the context is canceled.
func (t *E2ETest) Run(ctx context.Context) error {
	if t.path == "" {
		t.path = "testdata/checks.yaml"
	}

	const fileMode = 0o755
	err := os.MkdirAll("testdata", fileMode)
	if err != nil {
		t.t.Fatalf("failed to create testdata directory: %v", err)
	}

	err = os.WriteFile(t.path, t.buf.Bytes(), fileMode)
	if err != nil {
		t.t.Fatalf("failed to write testdata/checks.yaml: %v", err)
		return err
	}

	return t.sparrow.Run(ctx)
}
