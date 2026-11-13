package contracttest

import (
	"context"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/agents"
	"github.com/jterrazz/studio-station/apps/engine/agents/contract"
)

type RepoSubject struct {
	Repo contract.Repo
	// value: a folder whose work since Base is Want, and one with none.
	Folder, Unchanged, Base string
	Want                    agents.Change
}

func Repo(t *testing.T, open func(t *testing.T) RepoSubject) {
	ctx := context.Background()

	t.Run("a folder's change since its base is counted", func(t *testing.T) {
		s := open(t)
		if got, err := s.Repo.Change(ctx, s.Folder, s.Base); err != nil || got != s.Want {
			t.Errorf("Change = %+v, %v; want %+v", got, err, s.Want)
		}
	})

	t.Run("a folder with no change counts none", func(t *testing.T) {
		s := open(t)
		if got, err := s.Repo.Change(ctx, s.Unchanged, s.Base); err != nil || got != (agents.Change{}) {
			t.Errorf("Change = %+v, %v; want none", got, err)
		}
	})
}
