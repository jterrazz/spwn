package wiki_test

import (
	"slices"
	"strings"
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
	if links := pageLinks(s); !slices.Equal(links, []string{"product/stack#wiki", "product/stack"}) {
		t.Errorf("links served %q; want product/stack twice, the bell removed before the link was read", links)
	}
	if got := ruledOn(c.Report, "S30", "blueprint/l"); len(got) != 2 {
		t.Errorf("S30 findings %q; want the link with a space and the one with a tab", got)
	}
}

// value: the findings of one rule on one page, as ruled words them.
func ruledOn(r wiki.Report, rule, page string) []string {
	return slices.DeleteFunc(ruled(r, rule), func(f string) bool { return !strings.HasPrefix(f, page+" ") })
}

// value: the page links a served page's opening paragraph holds, each its
// page id and #anchor when it names one.
func pageLinks(s wiki.Served) []string {
	var links []string
	for _, p := range s.Paragraph {
		for _, r := range p.Runs {
			switch {
			case r.Kind == wiki.LinkPage && r.Anchor != "":
				links = append(links, r.Link+"#"+r.Anchor)
			case r.Kind == wiki.LinkPage:
				links = append(links, r.Link)
			}
		}
	}
	return links
}

func TestAPageLinkIsServedAsTheIdOfThePageItLandsOn(t *testing.T) {
	files := corpusFiles(t)
	files["blueprint/l.md"] = []byte("# L\n\nSee [the stack](../product/stack.md#wiki), [itself](l.md), [the philosophy](./philosophy), [none](nowhere.md), [out](../../stack.md), [far out](../../../etc/passwd.md) and [the root](/product/stack.md).\n")
	c, err := readAll(files)
	if err != nil {
		t.Fatal(err)
	}
	s, err := c.Serve("blueprint/l")
	if err != nil {
		t.Fatal(err)
	}
	if links := pageLinks(s); !slices.Equal(links, []string{"product/stack#wiki", "blueprint/l", "blueprint/philosophy"}) {
		t.Errorf("links served %q; want the three that land on a page of the corpus, by its id", links)
	}
	want := []string{
		`blueprint/l link to "nowhere.md" is no page, anchor or http or https address: served as its words`,
		`blueprint/l link to "../../stack.md" is no page, anchor or http or https address: served as its words`,
		`blueprint/l link to "../../../etc/passwd.md" is no page, anchor or http or https address: served as its words`,
		`blueprint/l link to "/product/stack.md" is no page, anchor or http or https address: served as its words`,
	}
	if got := ruledOn(c.Report, "S30", "blueprint/l"); !slices.Equal(got, want) {
		t.Errorf("S30 findings %q; want %q", got, want)
	}
}
