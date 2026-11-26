package contracttest

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strconv"
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

	t.Run("a signature holds while nothing changes, moves when a page is added or resized, and a hidden file never moves it", func(t *testing.T) {
		s := declared(t)
		s.Put("wiki.yaml", "wiki: spwn.wiki/v1\n")
		s.Put("blueprint/philosophy.md", "# Philosophy\n")
		signature := func() string {
			t.Helper()
			sig, err := s.Corpus.Signature(ctx)
			if err != nil {
				t.Fatalf("Signature: %v", err)
			}
			return sig
		}
		first := signature()
		if again := signature(); again != first {
			t.Fatalf("two signatures of one corpus differ: %q, %q", first, again)
		}
		s.Put(".notes.md", "# Hidden, never a page\n")
		if hidden := signature(); hidden != first {
			t.Fatalf("a hidden file moved the signature")
		}
		s.Put("blueprint/philosophy.md", "# Philosophy, longer\n")
		resized := signature()
		s.Put("blueprint/vision.md", "# Vision\n")
		if added := signature(); resized == first || added == resized {
			t.Fatalf("signatures %q, %q, %q; want each change seen", first, resized, added)
		}
	})

	t.Run("a page past its limit is sized and never read", func(t *testing.T) {
		s := declared(t)
		s.Put("wiki.yaml", "wiki: spwn.wiki/v1\n")
		s.Put("blueprint/huge.md", string(make([]byte, wiki.MaxPageBytes+1)))
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
			s.Put("blueprint/p"+strconv.Itoa(i)+".md", "# P\n")
		}
		var past *wiki.TooLarge
		if _, err := s.Corpus.Files(ctx); !errors.As(err, &past) {
			t.Fatalf("Files: %v; want a corpus past %d pages refused", err, wiki.MaxPages)
		}
	})
}
