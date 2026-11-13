package fake

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/agents"
	"github.com/jterrazz/studio-station/apps/engine/agents/contract"
)

type Repo struct {
	// invariant: each folder's change in one VM of the world; the base is
	// the world's, whatever it is named.
	inVM
}

var _ contract.Repo = (*Repo)(nil)

func NewRepo(folder string, change agents.Change) *Repo {
	r := &Repo{alone()}
	r.agents().Changes = map[string]agents.Change{folder: change}
	return r
}

func (r *Repo) Change(_ context.Context, folder, _ string) (agents.Change, error) {
	r.w.Lock()
	defer r.w.Unlock()
	return r.agents().Changes[folder], nil
}
