package fake

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strconv"

	"github.com/jterrazz/studio-station/src/agents/contract"
	"github.com/jterrazz/studio-station/src/world"
)

const (
	prOpen   = "open"
	prMerged = "merged"
)

type Forge struct {
	// invariant: the PRs of one VM's repositories in the world.
	inVM
}

var _ contract.Forge = (*Forge)(nil)

func NewForge(open int) *Forge {
	f := &Forge{alone()}
	f.agents().PRs = map[int]string{open: prOpen}
	return f
}

func ForgeOf(w *world.World, vm string) *Forge { return &Forge{inVM{w: w, vm: vm}} }

func (f *Forge) Merge(_ context.Context, _ string, number int) error {
	f.w.Lock()
	defer f.w.Unlock()
	held := f.agents()
	if held.PRs[number] != prOpen {
		return errors.New("PR #" + strconv.Itoa(number) + " is not open")
	}
	held.PRs[number] = prMerged
	return nil
}

func (f *Forge) Merged() []int {
	f.w.Lock()
	defer f.w.Unlock()
	held := f.agents().PRs
	return slices.Sorted(func(yield func(int) bool) {
		for n := range maps.Keys(held) {
			if held[n] == prMerged && !yield(n) {
				return
			}
		}
	})
}
