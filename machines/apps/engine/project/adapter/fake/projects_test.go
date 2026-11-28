package fake_test

import (
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/project"
	"github.com/jterrazz/studio-station/apps/engine/project/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/project/contract"
	"github.com/jterrazz/studio-station/apps/engine/project/contract/contracttest"
)

// value: the folders of a project whose spwn.yaml renames none, as the real
// adapter reads them: the contract's suite declares no folders:, and an
// adapter's test imports no other adapter.
func rootFolders([]byte, []string) (map[string]string, error) {
	return map[string]string{"agents": "agents", "skills": "skills", "tools": "tools", "hooks": "hooks", "commands": "commands", "profiles": "profiles"}, nil
}

func TestProjects(t *testing.T) {
	contracttest.Projects(t, func(t *testing.T) (contract.Projects, string) { return fake.New(rootFolders), t.TempDir() })
}

func TestLayouts(t *testing.T) {
	contracttest.Layouts(t, func(t *testing.T) (contract.Layouts, contract.Projects, string) {
		p := fake.New(rootFolders)
		return p, p, t.TempDir()
	})
}

func TestALayoutTheFoldersReadingRefusesIsRefusedAndActNeverRuns(t *testing.T) {
	refused := &project.LayoutRefused{Old: true, Said: "spwn/ is the old project layout"}
	p := fake.New(func([]byte, []string) (map[string]string, error) { return nil, refused })
	made, err := p.Create(t.Context(), t.TempDir(), "site", catalog.Harnesses()[0], "")
	if err != nil {
		t.Fatal(err)
	}
	ran := false
	if err := p.Within(made.Root, func() error { ran = true; return nil }); ran || !errors.Is(err, refused) {
		t.Fatalf("Within ran %v, = %v; want %v and act not run", ran, err, refused)
	}
}
