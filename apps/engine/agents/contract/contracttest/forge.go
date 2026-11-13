package contracttest

import (
	"context"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/agents/contract"
)

type ForgeSubject struct {
	Forge contract.Forge
	// value: a PR open in Folder's repository, and the PRs merged so far.
	Folder string
	Open   int
	Merged func() []int
}

func Forge(t *testing.T, open func(t *testing.T) ForgeSubject) {
	ctx := context.Background()

	t.Run("an open PR merged is merged", func(t *testing.T) {
		s := open(t)
		if err := s.Forge.Merge(ctx, s.Folder, s.Open); err != nil {
			t.Fatalf("Merge: %v", err)
		}
		if got := s.Merged(); !slices.Equal(got, []int{s.Open}) {
			t.Errorf("merged %v, want %d", got, s.Open)
		}
	})

	t.Run("a PR that is not open is refused", func(t *testing.T) {
		s := open(t)
		if err := s.Forge.Merge(ctx, s.Folder, s.Open+1000); err == nil {
			t.Error("a PR that is not open was merged")
		}
		if got := s.Merged(); len(got) != 0 {
			t.Errorf("merged %v, want none", got)
		}
	})
}
