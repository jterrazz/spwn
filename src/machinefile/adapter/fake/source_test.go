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

func TestARepositoryOfTheWorldKeepsTheContract(t *testing.T) {
	contracttest.Source(t, func(_ *testing.T, held map[string]string) contract.Source {
		src := contract.SharedSource{Name: "studio", URL: "https://example.com/studio.git", Commit: "1a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d"}
		w := &world.World{Sources: map[string]*world.Repository{src.URL: {Commits: map[string]map[string]string{src.Commit: maps.Clone(held)}}}}
		if held == nil {
			w.Sources[src.URL].Commits[src.Commit] = map[string]string{}
		}
		return fake.SharedOf(w, src)
	})
}
