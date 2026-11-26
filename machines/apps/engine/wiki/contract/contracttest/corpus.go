package contracttest

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/wiki/contract"
)

type Subject struct {
	Corpus contract.Corpus
	// value: writes a file of the corpus's tree, by slash path under its root.
	Put func(path, body string)
}

func Corpus(t *testing.T, declared func(t *testing.T) Subject, none func(t *testing.T) contract.Corpus) {
	ctx := context.Background()

	t.Run("a machine that declares no wiki reads none", func(t *testing.T) {
		if _, err := none(t).Files(ctx); !errors.Is(err, contract.ErrNone) {
			t.Fatalf("Files: %v; want ErrNone", err)
		}
	})

	t.Run("a declared corpus reads its wiki.yaml and its pages, nothing else", func(t *testing.T) {
		s := declared(t)
		s.Put("wiki.yaml", "wiki: spwn.wiki/v1\n")
		s.Put("blueprint/philosophy.md", "# Philosophy\n")
		s.Put("blueprint/board.png", "png")
		s.Put("product/notes.txt", "notes")
		s.Put(".git/HEAD.md", "ref")
		s.Put("product/wiki.yaml", "not the root's")
		got, err := s.Corpus.Files(ctx)
		if err != nil {
			t.Fatalf("Files: %v", err)
		}
		keys := slices.Sorted(maps.Keys(got))
		if !slices.Equal(keys, []string{"blueprint/philosophy.md", "wiki.yaml"}) || string(got["blueprint/philosophy.md"]) != "# Philosophy\n" {
			t.Fatalf("Files = %v; want wiki.yaml and the page", keys)
		}
	})
}
