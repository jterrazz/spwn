package wiki_test

import (
	"cmp"
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/jterrazz/studio-station/apps/engine/wiki"
)

// value: the graph the approved prototype built on the client from the seven
// pages it embeds, dumped by running its own functions (buildGraph, labelOf,
// kidMarks, needN, secIcon, the faces' words and the home's counts) in a
// browser.
type prototypeGraph struct {
	Roots []string        `json:"roots"`
	Nodes []prototypeNode `json:"nodes"`
	Home  struct {
		Word       string `json:"word"`
		Line       string `json:"line"`
		Pages      int    `json:"pages"`
		Seats      int    `json:"seats"`
		Needs      int    `json:"needs"`
		NeedsPages int    `json:"needsPages"`
		Hard       int    `json:"hard"`
		Soft       int    `json:"soft"`
	} `json:"home"`
	Floors []struct {
		Index int    `json:"index"`
		Name  string `json:"name"`
		Label string `json:"label"`
	} `json:"floors"`
}

type prototypeNode struct {
	ID     string   `json:"id"`
	Depth  int      `json:"depth"`
	Kind   string   `json:"kind"`
	Parent *string  `json:"parent"`
	Kids   []string `json:"kids"`
	Label  string   `json:"label"`
	Marks  []bool   `json:"marks"`
	Needs  int      `json:"needs"`
	Icon   string   `json:"icon"`
	Band   int      `json:"band"`
	Home   string   `json:"home"`
	Count  int      `json:"count"`
	Kicker string   `json:"kicker"`
	Title  string   `json:"title"`
	Line   string   `json:"line"`
	Ref    string   `json:"ref"`
	Anchor string   `json:"anchor"`
}

// value: what the engine serves where the prototype drew otherwise, each
// with its reason; every other field of every node is the prototype's. The
// prototype's kind sec is the engine's section, a name no face draws.
func mapDiffers() map[string]string {
	return map[string]string{
		"B:growth/positioning:lead-with-the-thread.icon": "a block's icon is the one wiki.read serves it with, its Face's, then its kind's, then its kind, so the canvas and the sheet draw the same icon for the same block; the prototype drew one icon per kind, and this rule's Face is its force, soft",
	}
}

// value: how the prototype wrote a line of parts and a cut, read from the
// fixture itself, since the glyphs are the kit's alone: a floor's label joins
// its letter and its name with the separator, and a paragraph whose title
// runs past its label had its label cut, the ellipsis after its words.
type protoGlyphs struct{ sep, cut string }

func glyphsOf(t *testing.T, g prototypeGraph) protoGlyphs {
	t.Helper()
	var out protoGlyphs
	for _, f := range g.Floors {
		letter := "E" + strconv.Itoa(f.Index)
		if !strings.HasPrefix(f.Label, letter) || !strings.HasSuffix(f.Label, f.Name) || len(f.Label) <= len(letter)+len(f.Name) {
			t.Fatalf("floor %+v: no separator between %q and %q", f, letter, f.Name)
		}
		sep := f.Label[len(letter) : len(f.Label)-len(f.Name)]
		if out.sep != "" && sep != out.sep {
			t.Fatalf("floor %d joins with %q; floor 0 with %q", f.Index, sep, out.sep)
		}
		out.sep = sep
	}
	for _, n := range g.Nodes {
		if n.Kind != "prose" || n.Title == n.Label {
			continue
		}
		_, size := utf8.DecodeLastRuneInString(n.Label)
		cut, kept := n.Label[len(n.Label)-size:], n.Label[:len(n.Label)-size]
		if !strings.HasPrefix(n.Title, kept+" ") || (out.cut != "" && cut != out.cut) {
			t.Fatalf("%s: label %q is not its title %q cut", n.ID, n.Label, n.Title)
		}
		out.cut = cut
	}
	if out.sep == "" || out.cut == "" {
		t.Fatalf("the fixture shows no separator (%q) or no cut (%q)", out.sep, out.cut)
	}
	return out
}

func (g protoGlyphs) split(s string) string {
	if s == "" {
		return partsField(nil, false)
	}
	kept, cut := strings.CutSuffix(s, g.cut)
	return partsField(strings.Split(kept, g.sep), cut)
}

func (g protoGlyphs) view(n prototypeNode) prototypeNode {
	n.Label, n.Title, n.Kicker, n.Line = g.split(n.Label), g.split(n.Title), g.split(n.Kicker), g.split(n.Line)
	return n
}

func partsField(parts []string, cut bool) string {
	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf("%q", parts) + map[bool]string{true: "+cut"}[cut]
}

func engineView(n wiki.MapNode) prototypeNode {
	kind := string(n.Kind)
	if n.Kind == wiki.NodeSection {
		kind = "sec"
	}
	out := prototypeNode{ID: n.ID, Depth: n.Depth, Kind: kind, Parent: n.Parent, Kids: n.Kids, Label: partsField(partsOf(n.Label), n.LabelCut), Needs: n.Needs, Icon: n.Icon, Count: n.Count, Kicker: partsField(n.KickerParts, false), Line: partsField(n.LineParts, false)}
	out.Marks = slices.Clone(n.Marks)
	for range n.More {
		out.Marks = append(out.Marks, false)
	}
	title := cmp.Or(partsField(n.TitleParts, n.TitleCut), out.Label)
	switch n.Kind {
	case wiki.NodeSeat:
		out.Band, out.Home = n.Band, n.Home
	case wiki.NodePage:
		out.Title = title
	case wiki.NodeSection:
		out.Anchor, out.Title = n.Ref, title
	case wiki.NodeBlock, wiki.NodeGroup, wiki.NodeProse:
		out.Ref, out.Title = n.Ref, title
	}
	return out
}

func protoView(n prototypeNode) prototypeNode {
	if n.Kids == nil {
		n.Kids = []string{}
	}
	if n.Marks == nil {
		n.Marks = []bool{}
	}
	if n.Kind == "page" {
		n.Kids = n.Kids[:0:0]
	}
	if len(n.Marks) > wiki.MaxMarks {
		n.Marks = slices.Concat(n.Marks[:wiki.MaxMarks], make([]bool, len(n.Marks)-wiki.MaxMarks))
	}
	return n
}

func fieldsOf(n prototypeNode) map[string]string {
	parent := "null"
	if n.Parent != nil {
		parent = *n.Parent
	}
	return map[string]string{
		"depth": strconv.Itoa(n.Depth), "kind": n.Kind, "parent": parent, "kids": strings.Join(n.Kids, ","),
		"label": n.Label, "marks": fmt.Sprint(n.Marks), "needs": strconv.Itoa(n.Needs), "icon": n.Icon,
		"band": strconv.Itoa(n.Band), "home": n.Home, "count": strconv.Itoa(n.Count), "kicker": n.Kicker,
		"title": n.Title, "line": n.Line, "ref": n.Ref, "anchor": n.Anchor,
	}
}

// invariant: the map is the graph the owner approved on the canvas, node for
// node and field for field, but for mapDiffers. A page's kids are compared
// where it is open, every other node in the map that holds it.
func TestTheMapIsTheGraphThePrototypeBuilt(t *testing.T) {
	var proto prototypeGraph
	if err := json.Unmarshal([]byte(read(t, filepath.Join("testdata", "prototype", "graph.json"))), &proto); err != nil {
		t.Fatal(err)
	}
	glyphs := glyphsOf(t, proto)
	files := corpusFiles(t)
	delete(files, unfinished+".md")
	c := corpusOf(t, files)
	root := mapOf(t, c, "")
	engine := map[string]prototypeNode{}
	for _, n := range root.Nodes {
		engine[n.ID] = engineView(n)
	}
	for _, id := range slices.DeleteFunc(corpusPages(), func(id string) bool { return id == unfinished }) {
		for _, n := range mapOf(t, c, id).Nodes {
			if n.Depth > 1 && n.ID != "P:"+id {
				v := engineView(n)
				if n.Depth == 2 {
					v.Kids = []string{}
				}
				engine[n.ID] = v
			}
		}
	}
	if !slices.Equal(root.Roots, proto.Roots) {
		t.Errorf("roots %v; the prototype's %v", root.Roots, proto.Roots)
	}
	seen, differs := map[string]bool{}, mapDiffers()
	for _, p := range proto.Nodes {
		seen[p.ID] = true
		e, ok := engine[p.ID]
		if !ok {
			t.Errorf("%s: the prototype drew it; the map serves none", p.ID)
			continue
		}
		want, got := fieldsOf(glyphs.view(protoView(p))), fieldsOf(e)
		for field, w := range want {
			if got[field] == w {
				continue
			}
			if _, stated := differs[p.ID+"."+field]; stated {
				continue
			}
			t.Errorf("%s.%s: %q; the prototype's %q", p.ID, field, got[field], w)
		}
	}
	for id := range engine {
		if !seen[id] {
			t.Errorf("%s: served; the prototype drew none", id)
		}
	}
	h := root.Home
	if got := fmt.Sprint(h.Word, h.Line, h.Pages, h.Seats, h.Needs, h.NeedsPages, h.Hard, h.Soft); got != fmt.Sprint(proto.Home.Word, proto.Home.Line, proto.Home.Pages, proto.Home.Seats, proto.Home.Needs, proto.Home.NeedsPages, proto.Home.Hard, proto.Home.Soft) {
		t.Errorf("home %s; the prototype's %+v", got, proto.Home)
	}
	for i, f := range proto.Floors {
		if i >= len(root.Floors) || partsField(root.Floors[i].LabelParts, false) != glyphs.split(f.Label) || root.Floors[i].Name != f.Name || root.Floors[i].Index != f.Index {
			t.Errorf("floor %d: %+v; the prototype's %+v", i, root.Floors, f)
		}
	}
}
