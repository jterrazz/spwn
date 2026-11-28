package fake_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/project/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/project/contract"
	"github.com/jterrazz/studio-station/apps/engine/project/contract/contracttest"
)

// value: the folders of a project whose spwn.yaml renames none, as the real
// adapter reads them: the contract's suite declares no folders:, and an
// adapter's test imports no other adapter.
func rootFolders([]byte, []string) ([]string, error) {
	return []string{"agents", "skills", "tools", "hooks", "commands", "profiles"}, nil
}

func TestProjects(t *testing.T) {
	contracttest.Projects(t, func(t *testing.T) (contract.Projects, string) { return fake.New(rootFolders), t.TempDir() })
}
