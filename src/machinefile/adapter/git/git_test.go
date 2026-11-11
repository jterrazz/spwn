package git_test

import (
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/jterrazz/studio-station/src/machinefile/adapter/git"
	"github.com/jterrazz/studio-station/src/machinefile/contract"
	"github.com/jterrazz/studio-station/src/machinefile/contract/contracttest"
	"github.com/jterrazz/studio-station/src/platform/local"
	"github.com/jterrazz/studio-station/src/platform/local/localtest"
)

const pinned = "1a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d"

// value: a clone as git leaves it: the files at the commit, HEAD detached
// at it.
func clone(t *testing.T, dir string, held map[string]string) {
	t.Helper()
	held = maps.Clone(held)
	if held == nil {
		held = map[string]string{}
	}
	held[".git/HEAD"] = pinned + "\n"
	for p, body := range held {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, p)), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, p), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTheSharedSourceKeepsTheContract(t *testing.T) {
	contracttest.Source(t, func(t *testing.T, held map[string]string) contract.Source {
		cache := t.TempDir()
		dir := filepath.Join(cache, "sources", "studio")
		run := localtest.NewRunner(t)
		run.On("git", "clone", "--quiet", "--no-checkout", "https://example.com/studio.git", dir).Answer(func() local.Output {
			clone(t, dir, held)
			return local.Output{}
		})
		run.On("git", "-C", dir, "-c", "advice.detachedHead=false", "checkout", "--quiet", "--detach", pinned).Reply(local.Output{})
		return git.New(run, local.New(nil, nil, nil, nil).FS, cache, contract.SharedSource{Name: "studio", URL: "https://example.com/studio.git", Commit: pinned})
	})
}

func TestAClonePinnedAlreadyRunsNothing(t *testing.T) {
	cache := t.TempDir()
	clone(t, filepath.Join(cache, "sources", "studio"), nil)
	src := git.New(localtest.NewRunner(t), local.New(nil, nil, nil, nil).FS, cache, contract.SharedSource{Name: "studio", URL: "https://example.com/studio.git", Commit: pinned})
	if _, err := src.Read(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func TestABranchIsNoPin(t *testing.T) {
	src := git.New(localtest.NewRunner(t), local.New(nil, nil, nil, nil).FS, t.TempDir(), contract.SharedSource{Name: "studio", URL: "https://example.com/studio.git", Commit: "main"})
	if _, err := src.Read(t.Context()); err == nil {
		t.Fatal("a source read at a branch")
	}
}
