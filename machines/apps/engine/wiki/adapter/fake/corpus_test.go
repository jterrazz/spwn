package fake_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/wiki/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/wiki/contract"
	"github.com/jterrazz/studio-station/apps/engine/wiki/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

func TestCorpus(t *testing.T) {
	contracttest.Corpus(t, func(*testing.T) contracttest.Subject {
		w := &world.World{}
		repo := &world.Repository{Head: "c1", Commits: map[string]map[string]string{"c1": {}}}
		w.Mac("mac-mini").Wiki = repo
		return contracttest.Subject{Corpus: fake.Of(w, "mac-mini"), Put: func(path, body string) { repo.Commits["c1"][path] = body }}
	}, func(*testing.T) contract.Corpus { return fake.Of(&world.World{}, "mac-mini") })
}
