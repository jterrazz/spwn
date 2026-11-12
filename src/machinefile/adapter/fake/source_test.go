package fake_test

import (
	"maps"
	"testing"

	"github.com/jterrazz/studio-station/src/machinefile/adapter/fake"
	"github.com/jterrazz/studio-station/src/machinefile/contract"
	"github.com/jterrazz/studio-station/src/machinefile/contract/contracttest"
	"github.com/jterrazz/studio-station/src/world"
)

func TestTheOwnersFilesOfAMacKeepTheContract(t *testing.T) {
	contracttest.Source(t, func(_ *testing.T, held map[string]string) contract.Source {
		w := &world.World{}
		w.Mac("laptop").Files = maps.Clone(held)
		return fake.UserOf(w, "laptop")
	})
}

const (
	head      = "1a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d"
	moved     = "ffffffffffffffffffffffffffffffffffffffff"
	rewritten = "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
)

func TestARepositoryOfTheWorldKeepsTheContract(t *testing.T) {
	contracttest.Source(t, func(_ *testing.T, held map[string]string) contract.Source {
		src := contract.SharedSource{Name: "studio", URL: "https://example.com/studio.git"}
		if held == nil {
			held = map[string]string{}
		}
		w := &world.World{Sources: map[string]*world.Repository{src.URL: {Head: head, Commits: map[string]map[string]string{head: maps.Clone(held)}}}}
		return fake.SharedOf(w, "laptop", src)
	})
}

func TestAMacsCloneFollowsItsRepositorysHead(t *testing.T) {
	contracttest.Follows(t, func(*testing.T) (contract.Source, func() string, func() string) {
		src := contract.SharedSource{Name: "studio", URL: "https://example.com/studio.git"}
		repo := &world.Repository{Head: head, Commits: map[string]map[string]string{head: {}, moved: {}, rewritten: {}}, Parents: map[string]string{moved: head}}
		w := &world.World{Sources: map[string]*world.Repository{src.URL: repo}}
		to := func(commit string) func() string {
			return func() string {
				w.Lock()
				defer w.Unlock()
				repo.Head = commit
				return commit
			}
		}
		return fake.SharedOf(w, "laptop", src), to(moved), to(rewritten)
	})
}
