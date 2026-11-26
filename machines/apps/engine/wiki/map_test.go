package wiki_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/wiki"
)

const mapGolden = "map-corpus.txt"

func corpusOf(t testing.TB, files map[string][]byte) wiki.Corpus {
	t.Helper()
	c, err := readAll(files)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func mapOf(t testing.TB, c wiki.Corpus, open string) wiki.Map {
	t.Helper()
	m, err := c.Map(open)
	if err != nil {
		t.Fatalf("open %q: %v", open, err)
	}
	return m
}

func nodesOf(m wiki.Map) map[string]wiki.MapNode {
	out := map[string]wiki.MapNode{}
	for _, n := range m.Nodes {
		out[n.ID] = n
	}
	return out
}

func marksOf(n wiki.MapNode) string {
	var b strings.Builder
	for _, y := range n.Marks {
		b.WriteString(map[bool]string{true: "y", false: "."}[y])
	}
	if n.More > 0 {
		b.WriteString("+" + strconv.Itoa(n.More))
	}
	return b.String()
}

func nodeLine(n wiki.MapNode) string {
	parts := []string{string(n.Kind), n.ID, strconv.Quote(n.Label) + cutOf(n.LabelCut), "needs=" + strconv.Itoa(n.Needs), "marks=" + marksOf(n), "icon=" + n.Icon, "ref=" + n.Ref}
	if n.Kind == wiki.NodeSeat {
		parts = append(parts, "band="+strconv.Itoa(n.Band), "home="+n.Home)
	}
	if n.Count > 0 {
		parts = append(parts, "count="+strconv.Itoa(n.Count))
	}
	for _, f := range []struct {
		name  string
		parts []string
		cut   bool
	}{{"title", n.TitleParts, n.TitleCut}, {"kicker", n.KickerParts, false}, {"line", n.LineParts, false}} {
		if f.parts != nil {
			parts = append(parts, f.name+"="+fmt.Sprintf("%q", f.parts)+cutOf(f.cut))
		}
	}
	return strings.Repeat("  ", n.Depth-1) + strings.Join(parts, " ")
}

func cutOf(cut bool) string {
	if cut {
		return "+cut"
	}
	return ""
}

func mapOutline(m wiki.Map, from string) string {
	nodes := nodesOf(m)
	var b strings.Builder
	var walk func(id string)
	walk = func(id string) {
		n := nodes[id]
		b.WriteString(nodeLine(n) + "\n")
		for _, k := range n.Kids {
			walk(k)
		}
	}
	if from != "" {
		walk(from)
		return b.String()
	}
	h := m.Home
	head := fmt.Sprintf("home %q %q pages=%d seats=%d needs=%d on=%d hard=%d soft=%d\n", h.Word, h.Line, h.Pages, h.Seats, h.Needs, h.NeedsPages, h.Hard, h.Soft)
	b.WriteString(head)
	for _, f := range m.Floors {
		floor := fmt.Sprintf("floor %d %q %q %q\n", f.Index, f.Name, f.LabelParts, f.Question)
		b.WriteString(floor)
	}
	names := make([]string, 0, len(m.Icons))
	for name := range m.Icons {
		names = append(names, name)
	}
	slices.Sort(names)
	b.WriteString("icons " + strings.Join(names, ",") + "\n")
	for _, r := range m.Roots {
		walk(r)
	}
	return b.String()
}

func TestTheMapIsServedToItsGoldenAtTheRootAndWithEachPageOpen(t *testing.T) {
	c := corpusOf(t, corpusFiles(t))
	var got strings.Builder
	got.WriteString(mapOutline(mapOf(t, c, ""), "") + "\n")
	for _, id := range corpusPages() {
		m := mapOf(t, c, id)
		got.WriteString("open " + m.Open + "\n" + mapOutline(m, m.Open) + "\n")
	}
	path := filepath.Join(goldenDir, mapGolden)
	if *update {
		if wrote := os.WriteFile(path, []byte(got.String()), 0o600); wrote != nil {
			t.Fatal(wrote)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s is missing (go test ./apps/engine/wiki/ -update writes it)", path)
	}
	if got.String() != string(want) {
		t.Errorf("%s differs (go test ./apps/engine/wiki/ -update rewrites it):\n%s", path, got.String())
	}
}

// invariant: every node counts what needs the owner under it, its kids' and
// its own: a seat and a section own none, a page owns the findings on its
// head, a block its findings, a group its blocks'.
func needsHold(t *testing.T, c wiki.Corpus, m wiki.Map) {
	t.Helper()
	nodes := nodesOf(m)
	total := 0
	for _, r := range m.Roots {
		total += nodes[r].Needs
	}
	if total != m.Home.Needs {
		t.Errorf("open %q: the seats count %d; the home %d", m.Open, total, m.Home.Needs)
	}
	for _, n := range m.Nodes {
		sum := 0
		for _, k := range n.Kids {
			kid, ok := nodes[k]
			if !ok {
				t.Fatalf("%s: kid %s is not served", n.ID, k)
			}
			if kid.Parent == nil || *kid.Parent != n.ID {
				t.Errorf("%s: kid %s names another parent", n.ID, k)
			}
			sum += kid.Needs
		}
		own := n.Needs - sum
		switch {
		case own < 0:
			t.Errorf("%s: needs %d under its kids' %d", n.ID, n.Needs, sum)
		case (n.Kind == wiki.NodeSeat || n.Kind == wiki.NodeSection) && own != 0:
			t.Errorf("%s: owns %d needs", n.ID, own)
		case n.Kind == wiki.NodePage && len(n.Kids) > 0 && own != headNeeds(c, n.Ref):
			t.Errorf("%s: owns %d needs; its head holds %d", n.ID, own, headNeeds(c, n.Ref))
		}
	}
}

func headNeeds(c wiki.Corpus, page string) int {
	n := 0
	for _, f := range c.Report.Findings {
		if f.Page == page && f.Block == "" && f.Level == wiki.LevelNeedsYou {
			n++
		}
	}
	return n
}

func TestEveryNodeCountsTheNeedsOfItsKidsAndItsOwn(t *testing.T) {
	c := corpusOf(t, corpusFiles(t))
	needsHold(t, c, mapOf(t, c, ""))
	for _, id := range corpusPages() {
		m := mapOf(t, c, id)
		needsHold(t, c, m)
		page := nodesOf(m)["P:"+id]
		if s, err := c.Serve(id); err != nil || page.Needs != s.NeedsYou {
			t.Errorf("%s: the map counts %d needs; wiki.read %d (%v)", id, page.Needs, s.NeedsYou, err)
		}
	}
}

func TestAPageIsTheSameNodeOpenOrNot(t *testing.T) {
	c := corpusOf(t, corpusFiles(t))
	closed := nodesOf(mapOf(t, c, ""))
	for _, id := range corpusPages() {
		m := mapOf(t, c, id)
		open := nodesOf(m)["P:"+id]
		was := closed["P:"+id]
		if len(was.Kids) != 0 || len(open.Kids) != open.Count {
			t.Errorf("%s: kids %v closed, %v open of %d", id, was.Kids, open.Kids, open.Count)
		}
		if open.Parent == nil || was.Parent == nil || *open.Parent != *was.Parent {
			t.Errorf("%s: parent %v open, %v closed", id, open.Parent, was.Parent)
		}
		open.Kids, was.Kids, open.Parent, was.Parent = nil, nil, nil, nil
		if fmt.Sprint(open) != fmt.Sprint(was) {
			t.Errorf("%s: open %+v; closed %+v", id, open, was)
		}
		for _, n := range m.Nodes {
			if n.Depth > 2 && !strings.HasPrefix(n.ID, "C:"+id+":") && !strings.HasPrefix(n.ID, "B:"+id+":") {
				t.Errorf("open %s serves %s", id, n.ID)
			}
		}
	}
}

func servedID(it wiki.ServedItem) string {
	switch {
	case it.Block != nil:
		return it.Block.ID
	case it.Group != nil:
		return it.Group.ID
	}
	return it.Prose.ID
}

func TestTheMapNamesEverySectionAndItemAsWikiReadServesIt(t *testing.T) {
	c := corpusOf(t, corpusFiles(t))
	for _, id := range corpusPages() {
		s, err := c.Serve(id)
		if err != nil {
			t.Fatal(err)
		}
		nodes := nodesOf(mapOf(t, c, id))
		var sections []string
		for si, sec := range s.Sections {
			if len(sec.Items) == 0 {
				continue
			}
			cid := "C:" + id + ":" + strconv.Itoa(si)
			sections = append(sections, cid)
			n := nodes[cid]
			if n.Ref != sec.Anchor || n.Label != sec.Label || n.Icon != sec.Icon {
				t.Errorf("%s: ref %q label %q icon %q; wiki.read %q %q %q", cid, n.Ref, n.Label, n.Icon, sec.Anchor, sec.Label, sec.Icon)
			}
			var items []string
			for _, it := range sec.Items {
				bid := "B:" + id + ":" + servedID(it)
				items = append(items, bid)
				item := nodes[bid]
				if item.Ref != servedID(it) {
					t.Errorf("%s: ref %q", bid, item.Ref)
				}
				if b := it.Block; b != nil && (item.Label != b.Word || item.Icon != b.Icon || !slices.Equal(item.KickerParts, []string{b.Title}) || !slices.Equal(item.LineParts, partsOf(b.Line)) || item.Needs != needsOfFindings(b.Finding)) {
					t.Errorf("%s: %+v; wiki.read %+v", bid, item, *b)
				}
				if g := it.Group; g != nil && item.Label != g.Title {
					t.Errorf("%s: label %q; wiki.read %q", bid, item.Label, g.Title)
				}
			}
			if !slices.Equal(n.Kids, items) {
				t.Errorf("%s: kids %v; wiki.read %v", cid, n.Kids, items)
			}
		}
		if page := nodes["P:"+id]; !slices.Equal(page.Kids, sections) {
			t.Errorf("%s: kids %v; wiki.read %v", id, page.Kids, sections)
		}
	}
}

func partsOf(s string) []string {
	if s == "" {
		return nil
	}
	return []string{s}
}

// invariant: a part says something: the window joins what is served, so an
// empty part would draw a separator between nothing.
func partsSaySomething(n wiki.MapNode) bool {
	for _, parts := range [][]string{n.TitleParts, n.KickerParts, n.LineParts} {
		if parts != nil && (len(parts) == 0 || slices.Contains(parts, "")) {
			return false
		}
	}
	return (!n.LabelCut || n.Label != "") && (!n.TitleCut || n.TitleParts != nil)
}

func TestAParagraphIsLabelledByItsFirstWordsAndSaysWhenTheyAreCut(t *testing.T) {
	for _, c := range []struct {
		prose, label, title string
		labelCut, titleCut  bool
	}{
		{prose: "Short one.", label: "Short one."},
		{prose: "Exactly three words.", label: "Exactly three words."},
		{prose: "Four words in all.", label: "Four words in", labelCut: true, title: "Four words in all."},
		{prose: "Five words are here now.", label: "Five words are", labelCut: true, title: "Five words are here", titleCut: true},
	} {
		files := map[string][]byte{"wiki.yaml": []byte(tinySchema), "blueprint/home.md": []byte("# Home\n\nThe home.\n\n## A\n\n" + c.prose + "\n")}
		m := mapOf(t, corpusOf(t, files), "blueprint/home")
		n := nodesOf(m)["B:blueprint/home:p1.0"]
		if n.Label != c.label || n.LabelCut != c.labelCut || !slices.Equal(n.TitleParts, partsOf(c.title)) || n.TitleCut != c.titleCut {
			t.Errorf("%q: %q cut %v, title %q cut %v", c.prose, n.Label, n.LabelCut, n.TitleParts, n.TitleCut)
		}
	}
}

func needsOfFindings(fs []wiki.ServedFinding) int {
	n := 0
	for _, f := range fs {
		if f.Level == wiki.LevelNeedsYou {
			n++
		}
	}
	return n
}

func TestAMapOpenAtNoPageOfItsOwnIsRefused(t *testing.T) {
	c := corpusOf(t, corpusFiles(t))
	for _, open := range []string{"blueprint/notes", "product", "P:product/stack"} {
		if _, err := c.Map(open); !errors.Is(err, wiki.ErrUnknownPage) {
			t.Errorf("open %q: %v; want ErrUnknownPage", open, err)
		}
	}
	if m := mapOf(t, c, ""); m.Open != "" {
		t.Errorf("open %q with none asked", m.Open)
	}
}

func TestANodeServesElevenMarksAtMostAndCountsTheRest(t *testing.T) {
	files := map[string][]byte{"wiki.yaml": []byte(tinySchema), "blueprint/home.md": []byte("# Home\n\nThe home.\n")}
	for i := range 13 {
		files["blueprint/p"+strconv.Itoa(10+i)+".md"] = []byte("# Page " + strconv.Itoa(10+i) + "\n\nA page.\n")
	}
	m := mapOf(t, corpusOf(t, files), "")
	seat := nodesOf(m)["S:vision"]
	if len(seat.Kids) != 14 || len(seat.Marks) != wiki.MaxMarks || seat.More != 3 || seat.Count != 14 || seat.Home != "P:blueprint/home" {
		t.Errorf("seat %+v", seat)
	}
}

func FuzzMap(f *testing.F) {
	for _, source := range corpusSources(f) {
		f.Add(source)
	}
	f.Add("# Home\n\nOne. Two.\n\n## A\n\n<Term id=\"a\" means=\"b\" />\n\n## A\n\nprose\n")
	f.Add("<Term/>")
	f.Fuzz(func(t *testing.T, source string) {
		corpus, err := readAll(map[string][]byte{"wiki.yaml": []byte(tinySchema), "blueprint/home.md": []byte(source), "blueprint/other.md": []byte("# Other\n\nIt rests.\n")})
		if err != nil {
			return
		}
		for _, open := range []string{"", "blueprint/home", "blueprint/other"} {
			m, err := corpus.Map(open)
			if err != nil {
				t.Fatalf("open %q: %v", open, err)
			}
			needsHold(t, corpus, m)
			ids := map[string]bool{}
			for _, n := range m.Nodes {
				if ids[n.ID] {
					t.Fatalf("open %q: %s twice", open, n.ID)
				}
				ids[n.ID] = true
				if n.Kids == nil || n.Marks == nil || len(n.Marks) > wiki.MaxMarks || !partsSaySomething(n) {
					t.Fatalf("open %q: %+v", open, n)
				}
			}
		}
	})
}
