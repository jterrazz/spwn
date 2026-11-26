package fake_test

import (
	"context"
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

func TestASignatureOfTheWorldSeesEveryByteAsTheWorldKeepsNoTime(t *testing.T) {
	w := &world.World{}
	repo := &world.Repository{Head: "c1", Commits: map[string]map[string]string{"c1": {"wiki.yaml": "wiki: spwn.wiki/v1\n", "blueprint/a.md": "# A\n"}}}
	w.Mac("mac-mini").Wiki = repo
	c := fake.Of(w, "mac-mini")
	first, err := c.Signature(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	repo.Commits["c1"]["blueprint/a.md"] = "# B\n"
	if moved, err := c.Signature(context.Background()); err != nil || moved == first {
		t.Fatalf("a page rewritten at its size kept the world's signature: %v", err)
	}
}
