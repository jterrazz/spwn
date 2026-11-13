package git_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/agents"
	"github.com/jterrazz/studio-station/apps/engine/agents/adapter/git"
	"github.com/jterrazz/studio-station/apps/engine/agents/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

type repository struct{}

func (repository) Output(_ context.Context, c local.Cmd) (local.Output, error) {
	switch {
	case slices.Equal(c.Args, []string{"-C", "/Users/admin/app", "diff", "--shortstat", "main...HEAD"}):
		return local.Output{Stdout: " 3 files changed, 86 insertions(+), 12 deletions(-)\n"}, nil
	case slices.Equal(c.Args, []string{"-C", "/Users/admin/api", "diff", "--shortstat", "main...HEAD"}):
		return local.Output{}, nil
	}
	return local.Output{ExitCode: 128, Stderr: "fatal: not a git repository"}, nil
}

func TestGitKeepsTheRepoContract(t *testing.T) {
	contracttest.Repo(t, func(*testing.T) contracttest.RepoSubject {
		return contracttest.RepoSubject{Repo: git.New(repository{}), Folder: "/Users/admin/app", Unchanged: "/Users/admin/api", Base: "main", Want: agents.Change{Files: 3, Plus: 86, Minus: 12}}
	})
}

func TestAChangeOfOneFileWithOnlyDeletionsIsCounted(t *testing.T) {
	r := git.New(oneDeletion{})
	if got, err := r.Change(context.Background(), "/x", "main"); err != nil || got != (agents.Change{Files: 1, Minus: 4}) {
		t.Errorf("Change = %+v, %v", got, err)
	}
}

type oneDeletion struct{}

func (oneDeletion) Output(context.Context, local.Cmd) (local.Output, error) {
	return local.Output{Stdout: " 1 file changed, 4 deletions(-)\n"}, nil
}
