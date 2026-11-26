package wiki_test

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/wiki"
)

// value: the fixture corpus's pages copied until it holds n pages, each copy
// under its own page's folder, as an honest corpus grows.
func copiedCorpus(tb testing.TB, n int) map[string][]byte {
	tb.Helper()
	files := map[string][]byte{"wiki.yaml": []byte(read(tb, filepath.Join(corpusDir, "wiki.yaml")))}
	sources := corpusSources(tb)
	ids := slices.Sorted(maps.Keys(sources))
	for _, id := range ids {
		files[id+".md"] = []byte(sources[id])
	}
	for i := range n - len(ids) {
		id := ids[i%len(ids)]
		files[fmt.Sprintf("%s-%04d.md", id, i)] = []byte(sources[id])
	}
	return files
}

// value: the most pages a corpus is read with, in 100 layers of 20, each
// page needing every page of the next layer, under one home that needs the
// first: the needs graph with the most paths a corpus may hold.
func layeredCorpus() map[string][]byte {
	const width, layers = 20, 100
	id := func(l, i int) string { return fmt.Sprintf("blueprint/l%03d-%02d", l, i) }
	layer := func(l int) []string {
		var out []string
		for i := range width {
			if l < layers && len(out)+l*width < wiki.MaxPages-1 {
				out = append(out, id(l, i))
			}
		}
		return out
	}
	files := map[string][]byte{"wiki.yaml": []byte(tinySchema), "blueprint/home.md": []byte(needing(layer(0)...) + "\nThe home of the layers.\n")}
	for l := range layers {
		next := layer(l + 1)
		for _, page := range layer(l) {
			files[page+".md"] = []byte(needing(next...) + "\nA page of a layer, needing the next.\n")
		}
	}
	return files
}

// value: each large corpus's budget, at least five times its time measured
// on the development VM at a load of 18 or less (2026-11-27): the read is
// the compile and the check of every page; the map and the serve are of
// the corpus once read, as the cache keeps it.
const (
	budgetReadCopied    = 2 * time.Second
	budgetReadLayered   = 2 * time.Second
	budgetMapLarge      = 500 * time.Millisecond
	budgetServeLarge    = 50 * time.Millisecond
	budgetMapOpenLarge  = 500 * time.Millisecond
	largeCorpusPages    = 500
	layeredCorpusPages  = 2000
	layeredCorpusOpened = "blueprint/l050-10"
)

func TestTwoThousandPagesInLayersAreReadMappedAndServedInLinearTime(t *testing.T) {
	files := layeredCorpus()
	if pages := len(files) - 1; pages != layeredCorpusPages {
		t.Fatalf("the layered corpus holds %d pages; want %d", pages, layeredCorpusPages)
	}
	var c wiki.Corpus
	var err error
	within(t, budgetReadLayered, "reading 2,000 pages in layers", func() { c, err = readAll(files) })
	if err != nil {
		t.Fatal(err)
	}
	if got := ruled(c.Report, "H10"); len(got) != 0 {
		t.Errorf("a layered corpus has no cycle; got %q", got)
	}
	var m wiki.Map
	within(t, budgetMapLarge, "the map of 2,000 pages, none open", func() { m, err = c.Map("") })
	if err != nil || len(m.Nodes) != layeredCorpusPages+1 {
		t.Fatalf("the map of 2,000 pages: %d nodes, %v; want a seat and its pages", len(m.Nodes), err)
	}
	within(t, budgetMapOpenLarge, "the map of 2,000 pages, one open", func() { m, err = c.Map(layeredCorpusOpened) })
	if err != nil || m.Open != "P:"+layeredCorpusOpened {
		t.Fatalf("the map open at %s: %q, %v", layeredCorpusOpened, m.Open, err)
	}
	var s wiki.Served
	within(t, budgetServeLarge, "one page of 2,000 served at its four levels", func() { s, err = c.Serve(layeredCorpusOpened) })
	if err != nil || len(s.Reading) != len(wiki.Details()) {
		t.Fatalf("serving %s: %d readings, %v", layeredCorpusOpened, len(s.Reading), err)
	}
}

func TestFiveHundredPagesAreReadMappedAndServedInLinearTime(t *testing.T) {
	files := copiedCorpus(t, largeCorpusPages)
	var c wiki.Corpus
	var err error
	within(t, budgetReadCopied, "reading 500 pages", func() { c, err = readAll(files) })
	if err != nil {
		t.Fatal(err)
	}
	within(t, budgetMapLarge, "the map of 500 pages, none open", func() { _, err = c.Map("") })
	if err != nil {
		t.Fatal(err)
	}
	within(t, budgetMapOpenLarge, "the map of 500 pages, one open", func() { _, err = c.Map("product/boundaries") })
	if err != nil {
		t.Fatal(err)
	}
	within(t, budgetServeLarge, "one page of 500 served at its four levels", func() { _, err = c.Serve("product/boundaries") })
	if err != nil {
		t.Fatal(err)
	}
}

func BenchmarkTheLargeCorpora(b *testing.B) {
	for name, files := range map[string]map[string][]byte{"copied-500": copiedCorpus(b, largeCorpusPages), "layered-2000": layeredCorpus()} {
		open := "product/boundaries"
		if name == "layered-2000" {
			open = layeredCorpusOpened
		}
		b.Run(name+"/read", func(b *testing.B) {
			for b.Loop() {
				if _, err := readAll(files); err != nil {
					b.Fatal(err)
				}
			}
		})
		c, err := readAll(files)
		if err != nil {
			b.Fatal(err)
		}
		b.Run(name+"/map", func(b *testing.B) {
			for b.Loop() {
				_, _ = c.Map("")
			}
		})
		b.Run(name+"/map-open", func(b *testing.B) {
			for b.Loop() {
				_, _ = c.Map(open)
			}
		})
		b.Run(name+"/serve", func(b *testing.B) {
			for b.Loop() {
				_, _ = c.Serve(open)
			}
		})
	}
}
