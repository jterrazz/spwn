package wiki_test

import (
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/wiki"
)

func TestALinkWithASpaceAroundItOrATabIsServedAsItsWords(t *testing.T) {
	files := corpusFiles(t)
	files["blueprint/l.md"] = []byte("# L\n\nSee [the wiki](../product/stack.md#wiki), [a space]( ../product/stack.md), [a tab](../product/stack.md\t) and [a bell](../product/stack.md\a).\n")
	c, err := readAll(files)
	if err != nil {
		t.Fatal(err)
	}
	s, err := c.Serve("blueprint/l")
	if err != nil {
		t.Fatal(err)
	}
	var links []string
	for _, p := range s.Paragraph {
		for _, r := range p.Runs {
			if r.Kind == wiki.LinkPage {
				links = append(links, r.Link)
			}
		}
	}
	if want := []string{"../product/stack.md#wiki", "../product/stack.md"}; !slices.Equal(links, want) {
		t.Errorf("links served %q; want %q, the bell removed before the link was read", links, want)
	}
	if got := ruled(c.Report, "S30"); len(got) != 2 {
		t.Errorf("S30 findings %q; want the link with a space and the one with a tab", got)
	}
}
