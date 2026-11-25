package wiki_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/wiki"
)

var update = flag.Bool("update", false, "rewrite testdata/__expected__ from the compiler")

const (
	pagesDir  = "testdata/pages"
	goldenDir = "testdata/__expected__"
)

// value: the kinds of the spwn corpus's wiki.yaml, SPEC-v1 §5.
func spwnKinds() wiki.Schema {
	return wiki.Schema{Kinds: map[string]wiki.Kind{
		"conviction": {Icon: "conviction", Weighs: 1, Fields: []wiki.Field{
			{Name: "says", Type: "sentence", Max: 160, Required: true},
			{Name: "opposite", Type: "sentence", Max: 200, Required: true},
			{Name: "carried_in", Type: "list<ref>", To: "part", Required: true},
			{Name: "broken_by", Type: "sentence", Max: 200, Required: true},
			{Name: "adr", Type: "anchor"},
		}, Faces: wiki.FaceTemplates{Word: "{id}", Line: "{says}"}},
		"alternative": {Fields: []wiki.Field{
			{Name: "gives", Type: "sentence", Required: true},
			{Name: "costs", Type: "sentence", Required: true},
			{Name: "loses", Type: "enum", Values: []string{"scale", "authorship"}},
		}, Faces: wiki.FaceTemplates{Line: "{costs}"}},
		"part": {State: "state", Fields: []wiki.Field{
			{Name: "space", Type: "enum", Values: []string{"machine", "agent", "os"}, Required: true},
			{Name: "lets_go", Type: "sentence", Required: true},
			{Name: "keeps", Type: "sentence", Required: true},
			{Name: "state", Type: "enum", Values: []string{"shipped", "making"}, Required: true},
		}, Faces: wiki.FaceTemplates{Line: "{keeps}"}},
		"refusal": {Fields: []wiki.Field{
			{Name: "by", Type: "ref", To: "part"},
			{Name: "refuses", Type: "sentence", Required: true},
			{Name: "because", Type: "ref|anchor|sentence", Required: true},
		}, Faces: wiki.FaceTemplates{Line: "{refuses}"}},
	}}
}

type fixture struct {
	name, id, source string
}

func fixtures(t *testing.T) []fixture {
	t.Helper()
	entries, err := os.ReadDir(pagesDir)
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]string{"philosophy": "blueprint/philosophy", "boundaries": "product/boundaries", "positioning": "growth/positioning", "forms": "forms"}
	out := make([]fixture, 0, len(entries))
	for _, e := range entries {
		body, err := os.ReadFile(filepath.Join(pagesDir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		out = append(out, fixture{name: name, id: ids[name], source: string(body)})
	}
	return out
}

func TestEachPageCompilesToItsGolden(t *testing.T) {
	for _, f := range fixtures(t) {
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		if err := enc.Encode(wiki.Compile(f.id, f.source, spwnKinds())); err != nil {
			t.Fatal(err)
		}
		got := buf.String()
		path := filepath.Join(goldenDir, f.name+".json")
		if *update {
			if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s is missing (go test ./apps/engine/wiki/ -update writes it)", path)
		}
		if got != string(want) {
			t.Errorf("%s differs (go test ./apps/engine/wiki/ -update rewrites it):\n%s", path, got)
		}
	}
}

func TestTheFixturePagesCompileWithNoFinding(t *testing.T) {
	for _, f := range fixtures(t) {
		if p := wiki.Compile(f.id, f.source, spwnKinds()); len(p.Findings) > 0 {
			t.Errorf("%s: %+v", f.name, p.Findings)
		}
	}
}

// invariant: the round trip of a page: every line is read once, as the
// frontmatter, the title, a heading, prose, a tag or a group's close, and a
// line no record holds is blank; so the records lose no line and copy none.
func TestEveryLineOfAPageIsReadOnce(t *testing.T) {
	for _, f := range fixtures(t) {
		p := wiki.Compile(f.id, f.source, spwnKinds())
		held := map[int]int{}
		hold := func(s wiki.Span) {
			for n := s.From; n <= s.To; n++ {
				held[n]++
			}
		}
		if p.Front != nil {
			hold(p.Front.Span)
		}
		if p.Title != nil {
			hold(*p.Title)
		}
		var items func([]wiki.Item)
		items = func(list []wiki.Item) {
			for _, it := range list {
				switch it := it.(type) {
				case *wiki.Prose:
					hold(it.Span)
				case *wiki.Block:
					hold(it.Span)
				case *wiki.Group:
					hold(it.Tag)
					if it.Span.To != it.Tag.To {
						hold(wiki.Span{From: it.Span.To, To: it.Span.To})
					}
					items(it.Items)
				}
			}
		}
		for _, s := range p.Sections {
			if s.Heading != nil {
				hold(wiki.Span{From: s.Heading.At.Line, To: s.Heading.At.Line})
			}
			items(s.Items)
		}
		for i, line := range strings.Split(f.source, "\n") {
			n := i + 1
			switch {
			case held[n] > 1:
				t.Errorf("%s:%d is read %d times: %q", f.name, n, held[n], line)
			case held[n] == 0 && strings.TrimSpace(line) != "":
				t.Errorf("%s:%d is read by no record: %q", f.name, n, line)
			}
		}
	}
}
