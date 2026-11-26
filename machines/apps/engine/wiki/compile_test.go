package wiki_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/wiki"
)

var update = flag.Bool("update", false, "rewrite testdata/__expected__ from the compiler")

const (
	corpusDir = "testdata/corpus"
	formsPage = "testdata/pages/forms.md"
	goldenDir = "testdata/__expected__"
)

// value: the fixture corpus's pages, by id: the five pages of the first
// slice in tag form, the path of each under the corpus its id.
func corpusPages() []string {
	return []string{"blueprint/philosophy", "product/stack", "product/boundaries", "growth/positioning", "machines/reference/ports", "product/decisions/002-spwn-is-one-stack-of-six-apps-on-one-ladder", "machines/decisions/035-a-port-is-a-right-with-an-audience"}
}

// value: the schema of the fixture corpus, read from its wiki.yaml.
var spwnSchema = sync.OnceValues(func() (wiki.Schema, error) {
	body, err := os.ReadFile(filepath.Join(corpusDir, "wiki.yaml"))
	if err != nil {
		return wiki.Schema{}, err
	}
	return wiki.ReadSchema("wiki.yaml", body)
})

func spwnKinds() wiki.Schema {
	s, err := spwnSchema()
	if err != nil {
		panic(err)
	}
	return s
}

type fixture struct {
	name, id, source string
}

func read(t testing.TB, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

// value: every page the compile is proved on: the corpus's, then one
// synthetic page of every form.
func fixtures(t *testing.T) []fixture {
	t.Helper()
	out := make([]fixture, 0, len(corpusPages())+1)
	for _, id := range corpusPages() {
		out = append(out, fixture{name: strings.ReplaceAll(id, "/", "-"), id: id, source: read(t, filepath.Join(corpusDir, id+".md"))})
	}
	return append(out, fixture{name: "forms", id: "forms", source: read(t, formsPage)})
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
		path := filepath.Join(goldenDir, "compile-"+f.name+".json")
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
