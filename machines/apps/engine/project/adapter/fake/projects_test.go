package fake_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/project/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/project/adapter/manifest"
	"github.com/jterrazz/studio-station/apps/engine/project/contract"
	"github.com/jterrazz/studio-station/apps/engine/project/contract/contracttest"
)

func TestProjects(t *testing.T) {
	contracttest.Projects(t, func(t *testing.T) (contract.Projects, string) { return fake.New(manifest.Folders), t.TempDir() })
}
