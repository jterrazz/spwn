package mind_test

import (
	"testing"

	spwnproject "spwn.sh/packages/project"

	"github.com/jterrazz/studio-station/apps/engine/agents/adapter/mind"
	"github.com/jterrazz/studio-station/apps/engine/agents/contract"
	"github.com/jterrazz/studio-station/apps/engine/agents/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

// value: the layout of a project as the root module holds it, which the
// project family's adapter words: an adapter's test imports no other
// adapter.
type rootModule struct{}

func (rootModule) Within(root string, act func() error) error {
	_, err := spwnproject.Within(root, func() (struct{}, error) { return struct{}{}, act() })
	return err
}

func TestMinds(t *testing.T) {
	contracttest.Minds(t, func(t *testing.T) (contract.Minds, string) {
		t.Setenv("SPWN_HOME", t.TempDir())
		return mind.Minds{RemoveAll: local.New(nil, nil, nil, nil).FS.RemoveAll, Layouts: rootModule{}}, t.TempDir()
	})
}

func TestTheUsersOwnAgentsAreThoseOfNoProject(t *testing.T) {
	contracttest.Minds(t, func(t *testing.T) (contract.Minds, string) {
		t.Setenv("SPWN_HOME", t.TempDir())
		return mind.Minds{RemoveAll: local.New(nil, nil, nil, nil).FS.RemoveAll, Layouts: rootModule{}}, ""
	})
}
