package wiki_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/wiki"
)

// value: a run held to a budget, three tries, so a loaded host does not
// fail a linear path, and a try past ten budgets is a hang. The clock is a
// context's deadline: the wiki reads none of its own.
func within(t testing.TB, budget time.Duration, what string, run func()) {
	t.Helper()
	budget *= slowdown
	for range 3 {
		done := make(chan struct{})
		go func() {
			defer close(done)
			run()
		}()
		if finished(done, budget) {
			return
		}
		if !finished(done, 10*budget) {
			t.Fatalf("%s runs past %v: a path is not linear", what, 10*budget)
		}
	}
	t.Errorf("%s ran past %v three times: a path is not linear", what, budget)
}

func finished(done <-chan struct{}, budget time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), budget)
	defer cancel()
	select {
	case <-done:
		return true
	case <-ctx.Done():
		return false
	}
}

func needing(ids ...string) string {
	return "---\nneeds: [" + strings.Join(ids, ", ") + "]\n---\n# T\n"
}

func compiledAll(sources map[string]string) []wiki.Page {
	pages := make([]wiki.Page, 0, len(sources))
	for id, body := range sources {
		pages = append(pages, wiki.Compile(id, body, spwnKinds()))
	}
	return pages
}

func ruled(r wiki.Report, rule string) []string {
	var out []string
	for _, f := range r.Findings {
		if f.Rule == rule {
			out = append(out, f.Page+" "+f.Says)
		}
	}
	return out
}

// value: the needs graphs the verify of 2026-11-26 found exponential, so the
// tests hold them to a budget and the benchmark times them.
func clique() []wiki.Page {
	sources := map[string]string{"blueprint/p": needing("blueprint/a01")}
	var all []string
	for i := 1; i <= 15; i++ {
		all = append(all, fmt.Sprintf("blueprint/a%02d", i))
	}
	for _, id := range all {
		var others []string
		for _, o := range all {
			if o != id {
				others = append(others, o)
			}
		}
		sources[id] = needing(others...)
	}
	return compiledAll(sources)
}

func layered() []wiki.Page {
	const width, layers = 20, 100
	id := func(l, i int) string { return fmt.Sprintf("blueprint/l%03d-%02d", l, i) }
	sources := map[string]string{}
	for l := range layers {
		var next []string
		if l+1 < layers {
			for i := range width {
				next = append(next, id(l+1, i))
			}
		}
		for i := range width {
			sources[id(l, i)] = needing(next...)
		}
	}
	return compiledAll(sources)
}

func TestFifteenPagesNeedingEachOtherAreCheckedInMilliseconds(t *testing.T) {
	pages, schema := clique(), spwnKinds()
	var report wiki.Report
	within(t, budgetClique, "a clique of fifteen needs", func() { report = checkWith(schema, pages) })
	want := []string{"blueprint/a01 needs: blueprint/a01 -> blueprint/a02 -> blueprint/a01"}
	if got := ruled(report, "H10"); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("cycles %q, want %q", got, want)
	}
}

func TestTwoThousandPagesInLayersAreCheckedInLinearTime(t *testing.T) {
	pages, schema := layered(), spwnKinds()
	var report wiki.Report
	within(t, budgetLayered, "a corpus of 100 layers of 20 pages", func() { report = checkWith(schema, pages) })
	if got := ruled(report, "H10"); len(got) != 0 {
		t.Errorf("a layered corpus has no cycle; got %q", got)
	}
}

func TestEachSetOfPagesNeedingEachOtherIsSaidOnceFromItsFirstPage(t *testing.T) {
	sources := map[string]string{
		"blueprint/c": needing("blueprint/a"),
		"blueprint/a": needing("blueprint/b", "blueprint/c"),
		"blueprint/b": needing("blueprint/a"),
		"blueprint/s": needing("blueprint/s"),
		"blueprint/t": needing("blueprint/s"),
	}
	want := []string{
		"blueprint/a needs: blueprint/a -> blueprint/b -> blueprint/a",
		"blueprint/s needs: blueprint/s -> blueprint/s",
	}
	if got := ruled(checkAll(compiledAll(sources)), "H10"); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("cycles %q, want %q", got, want)
	}
}

func BenchmarkTheWorstNeeds(b *testing.B) {
	pages, schema := clique(), spwnKinds()
	b.Run("clique", func(b *testing.B) {
		for b.Loop() {
			checkWith(schema, pages)
		}
	})
	layers := layered()
	b.Run("layered", func(b *testing.B) {
		for b.Loop() {
			checkWith(schema, layers)
		}
	})
}
