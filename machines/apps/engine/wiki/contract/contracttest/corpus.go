package contracttest

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/wiki"
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
		s.Put("blueprint/.draft.md", "# Hidden\n")
		got, err := s.Corpus.Files(ctx)
		if err != nil {
			t.Fatalf("Files: %v", err)
		}
		keys := slices.Sorted(maps.Keys(got))
		page := got["blueprint/philosophy.md"]
		if !slices.Equal(keys, []string{"blueprint/philosophy.md", "wiki.yaml"}) || string(page.Body) != "# Philosophy\n" || page.Size != int64(len(page.Body)) {
			t.Fatalf("Files = %v; want wiki.yaml and the page, sized", keys)
		}
	})

	t.Run("a page past its limit is sized and never read", func(t *testing.T) {
		s := declared(t)
		s.Put("wiki.yaml", "wiki: spwn.wiki/v1\n")
		s.Put("blueprint/huge.md", strings.Repeat("a", wiki.MaxPageBytes+1))
		got, err := s.Corpus.Files(ctx)
		if err != nil {
			t.Fatalf("Files: %v", err)
		}
		if huge := got["blueprint/huge.md"]; huge.Body != nil || huge.Size != wiki.MaxPageBytes+1 {
			t.Fatalf("the huge page is %d bytes, %d of them read; want its size and nothing read", huge.Size, len(huge.Body))
		}
	})

	t.Run("a corpus past its limit stops its read", func(t *testing.T) {
		s := declared(t)
		s.Put("wiki.yaml", "wiki: spwn.wiki/v1\n")
		for i := range wiki.MaxPages + 1 {
			s.Put(fmt.Sprintf("blueprint/p%04d.md", i), "# P\n")
		}
		var past *wiki.TooLarge
		if _, err := s.Corpus.Files(ctx); !errors.As(err, &past) {
			t.Fatalf("Files: %v; want a corpus past %d pages refused", err, wiki.MaxPages)
		}
	})
}
