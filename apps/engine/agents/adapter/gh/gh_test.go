package gh_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/agents/adapter/gh"
	"github.com/jterrazz/studio-station/apps/engine/agents/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

type forge struct {
	// value: a recorded forge holding PR #42 open in /Users/admin/app.
	merged []int
}

func (f *forge) Output(_ context.Context, c local.Cmd) (local.Output, error) {
	want := []string{"pr", "merge", "42", "--merge"}
	if c.Name == "gh" && c.Dir == "/Users/admin/app" && slices.Equal(c.Args, want) {
		f.merged = append(f.merged, 42)
		return local.Output{}, nil
	}
	return local.Output{ExitCode: 1, Stderr: "no pull requests found"}, nil
}

func TestGhKeepsTheForgeContract(t *testing.T) {
	contracttest.Forge(t, func(*testing.T) contracttest.ForgeSubject {
		f := &forge{}
		return contracttest.ForgeSubject{Forge: gh.New(f), Folder: "/Users/admin/app", Open: 42, Merged: func() []int { return f.merged }}
	})
}
