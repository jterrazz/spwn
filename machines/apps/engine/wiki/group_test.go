package wiki_test

import (
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/wiki"
)

const groupThatNeedsYou = "# G\n\nA page whose group rests on its reading.\n\n<Group id=\"words\" title=\"The words\" intent=\"define\" rests_on=\"reading\">\n\n<Term id=\"seat\" means=\"who reads a page.\" />\n\n</Group>\n"

func TestAFindingOnAGroupIsServedOnItAndCountedWhereverItsPageIs(t *testing.T) {
	files := corpusFiles(t)
	files["blueprint/g.md"] = []byte(groupThatNeedsYou)
	c, err := readAll(files)
	if err != nil {
		t.Fatal(err)
	}
	s, err := c.Serve("blueprint/g")
	if err != nil {
		t.Fatal(err)
	}
	var g *wiki.ServedGroup
	for _, sec := range s.Sections {
		for _, it := range sec.Items {
			if it.Group != nil {
				g = it.Group
			}
		}
	}
	if g == nil || !g.NeedsYou || !slices.ContainsFunc(g.Finding, func(f wiki.ServedFinding) bool { return f.Rule == "S6" && f.At == "blueprint/g#words" }) {
		t.Fatalf("group %+v; want its S6 served on it, needing the owner", g)
	}
	if s.NeedsYou != 1 || !slices.ContainsFunc(s.Attention, func(f wiki.ServedFinding) bool { return f.Rule == "S6" && f.At == "blueprint/g#words" }) {
		t.Errorf("page needs %d, attention %+v; want the group's S6 counted and listed", s.NeedsYou, s.Attention)
	}
	m, err := c.Map("blueprint/g")
	if err != nil {
		t.Fatal(err)
	}
	needs := map[string]int{}
	for _, n := range m.Nodes {
		needs[n.ID] = n.Needs
	}
	if needs["B:blueprint/g:words"] != 1 || needs["P:blueprint/g"] != 1 {
		t.Errorf("map needs %v; want the group and its page at 1", needs)
	}
	closed, err := c.Map("")
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range closed.Nodes {
		if n.ID == "P:blueprint/g" && n.Needs != 1 {
			t.Errorf("the closed page needs %d; want 1", n.Needs)
		}
	}
}
