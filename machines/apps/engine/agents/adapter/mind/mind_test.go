package mind_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/agents/adapter/mind"
	"github.com/jterrazz/studio-station/apps/engine/agents/contract"
	"github.com/jterrazz/studio-station/apps/engine/agents/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

func TestMinds(t *testing.T) {
	contracttest.Minds(t, func(t *testing.T) (contract.Minds, string) {
		t.Setenv("SPWN_HOME", t.TempDir())
		return mind.Minds{RemoveAll: local.New(nil, nil, nil, nil).FS.RemoveAll}, t.TempDir()
	})
}

func TestTheUsersOwnAgentsAreThoseOfNoProject(t *testing.T) {
	contracttest.Minds(t, func(t *testing.T) (contract.Minds, string) {
		t.Setenv("SPWN_HOME", t.TempDir())
		return mind.Minds{RemoveAll: local.New(nil, nil, nil, nil).FS.RemoveAll}, ""
	})
}
