package wiki_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/wiki"
)

// value: the pages the verify of 2026-11-26 found quadratic, each a
// megabyte, so the tests hold them to a budget and the benchmark times them.
func longTag() string {
	var b strings.Builder
	b.WriteString("<Term")
	for i := 0; b.Len() < 1<<20-16; i++ {
		fmt.Fprintf(&b, " a%d", i)
	}
	return b.String() + "\n"
}

func unclosedBraces() string { return strings.Repeat("<A b={\n\n", (1<<20)/8) }

// value: how often the retired term is written, so its page stays under
// the 1 MiB a page is compiled from.
const retiredTimes = (1<<20)/2 - 64

func retiredEverywhere() []wiki.Page {
	body := "# T\n\n<Term id=\"a\" means=\"m.\" retired=\"2026-01-01\" />\n\n" + strings.Repeat("a ", retiredTimes) + "\n"
	return []wiki.Page{wiki.Compile("blueprint/r", body, spwnKinds())}
}

func brackets() string { return strings.Repeat("[", 1<<19) + strings.Repeat("[](", (1<<19)/3) }

func atMostAHundredAndOne(t *testing.T, findings []wiki.Finding) {
	t.Helper()
	perRule := map[string]int{}
	for _, f := range findings {
		perRule[f.Page+" "+f.Rule]++
	}
	for rule, n := range perRule {
		if n > 101 {
			t.Errorf("%d findings of %s; want 100 and one that counts the rest", n, rule)
		}
	}
	if len(perRule) == 0 {
		t.Errorf("a hostile page gives no finding")
	}
}

func TestATagOfAMegabyteOfAttributesCompilesInLinearTime(t *testing.T) {
	src := longTag()
	var p wiki.Page
	within(t, 400*time.Millisecond, "a 1 MiB tag of attributes", func() { p = wiki.Compile("blueprint/t", src, spwnKinds()) })
	atMostAHundredAndOne(t, p.Findings)
}

func TestAPageOfBracesNeverClosedCompilesInLinearTime(t *testing.T) {
	src := unclosedBraces()
	var p wiki.Page
	within(t, 400*time.Millisecond, "a 1 MiB page of braces never closed", func() { p = wiki.Compile("blueprint/b", src, spwnKinds()) })
	atMostAHundredAndOne(t, p.Findings)
}

func TestARetiredTermWrittenHalfAMillionTimesIsSaidAHundredTimesAndCounted(t *testing.T) {
	pages := retiredEverywhere()
	var report wiki.Report
	within(t, 400*time.Millisecond, "a retired term written 500k times", func() { report = checkAll(pages) })
	got := ruled(report, "H8")
	if want := fmt.Sprintf("blueprint/r and %d more like it on this page", retiredTimes-100); len(got) != 101 || got[100] != want {
		t.Errorf("H8 findings %d, the last %q; want 100, then %q", len(got), got[len(got)-1], want)
	}
	atMostAHundredAndOne(t, report.Findings)
	corpus := wiki.Corpus{Schema: spwnKinds(), Pages: pages, Report: report}
	within(t, 400*time.Millisecond, "serving the page of 500k findings", func() { _, _ = corpus.Serve("blueprint/r", wiki.Budget10min) })
}

func TestAProseOfBracketsIsReadInLinearTime(t *testing.T) {
	src := brackets()
	var out []wiki.Paragraph
	within(t, 200*time.Millisecond, "1 MiB of brackets", func() { out = wiki.Paragraphs(src) })
	if len(out) != 1 {
		t.Errorf("one line of brackets gives %d paragraphs, want 1", len(out))
	}
}

func TestACorpusPastItsLimitsIsRefusedWhole(t *testing.T) {
	files := corpusFiles(t)
	for i := range wiki.MaxPages {
		files[fmt.Sprintf("blueprint/n%04d.md", i)] = []byte("# N\n")
	}
	_, err := readAll(files)
	var past *wiki.TooLarge
	if !errors.As(err, &past) || past.Pages <= wiki.MaxPages {
		t.Errorf("err = %v; want a corpus of more than %d pages refused", err, wiki.MaxPages)
	}
	files = corpusFiles(t)
	for i := range 65 {
		files[fmt.Sprintf("blueprint/m%02d.md", i)] = make([]byte, 1<<20)
	}
	if _, err := readAll(files); !errors.As(err, &past) || past.Bytes <= wiki.MaxCorpusBytes {
		t.Errorf("err = %v; want a corpus of more than 64 MiB refused", err)
	}
}

func TestAPagePastAMebibyteIsServedWithItsFindingAndNothingItHolds(t *testing.T) {
	files := map[string]wiki.File{}
	for p, body := range corpusFiles(t) {
		files[p] = wiki.File{Size: int64(len(body)), Body: body}
	}
	files["product/stack/huge.md"] = wiki.File{Size: 3 << 20}
	c, err := wiki.Read(context.Background(), files)
	if err != nil {
		t.Fatal(err)
	}
	s, err := c.Serve("product/stack/huge", wiki.Budget2min)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(s.Findings, func(f wiki.ServedFinding) bool { return f.Rule == "H44" }) || s.Verdict.Hard == 0 {
		t.Errorf("findings %+v, verdict %+v; want H44 and a hard verdict", s.Findings, s.Verdict)
	}
}

func TestAnAbandonedReadEndsWithItsContextsError(t *testing.T) {
	files := map[string]wiki.File{}
	for p, body := range corpusFiles(t) {
		files[p] = wiki.File{Size: int64(len(body)), Body: body}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := wiki.Read(ctx, files); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v; want context.Canceled", err)
	}
}

// value: what one fuzz input may cost at most, read, checked and served;
// an input is a few kilobytes, so a quadratic path overruns it.
const fuzzBudget = 500 * time.Millisecond

func FuzzRead(f *testing.F) {
	f.Add("---\nneeds: [blueprint/b]\n---\n# A\n", "---\nneeds: [blueprint/a]\n---\n# B\n\n[[[[[[[[](")
	f.Add(strings.Repeat("<A b={\n\n", 64), strings.Repeat("<Term", 32)+strings.Repeat(" a", 512))
	f.Add("# A\n\n<Term id=\"a\" means=\"m.\" retired=\"2026-01-01\" />\n\n"+strings.Repeat("a ", 512), "# B\n\n<Page id=\"blueprint/a\" />\n")
	f.Fuzz(func(t *testing.T, a, b string) {
		files := corpusFiles(t)
		files["blueprint/a.md"], files["blueprint/b.md"] = []byte(a), []byte(b)
		within(t, fuzzBudget, "a read of two fuzzed pages", func() {
			c, err := readAll(files)
			if err != nil {
				t.Errorf("a corpus of two fuzzed pages is not read: %v", err)
				return
			}
			for _, id := range []string{"blueprint/a", "blueprint/b", "product/stack"} {
				for _, budget := range wiki.Budgets() {
					if _, err := c.Serve(id, budget); err != nil {
						t.Errorf("%s at %s is not served: %v", id, budget, err)
					}
				}
			}
		})
	})
}

func FuzzReadSchema(f *testing.F) {
	f.Add([]byte("wiki: spwn.wiki/v1\n"))
	if yaml, err := os.ReadFile(filepath.Join(corpusDir, "wiki.yaml")); err == nil {
		f.Add(yaml)
	}
	f.Fuzz(func(t *testing.T, body []byte) {
		within(t, fuzzBudget, "a fuzzed wiki.yaml", func() { _, _ = wiki.ReadSchema("wiki.yaml", body) })
	})
}

func BenchmarkTheWorstPages(b *testing.B) {
	for name, src := range map[string]string{"long-tag": longTag(), "unclosed-braces": unclosedBraces()} {
		b.Run(name, func(b *testing.B) {
			for b.Loop() {
				wiki.Compile("blueprint/x", src, spwnKinds())
			}
		})
	}
	retired := retiredEverywhere()
	b.Run("retired-term", func(b *testing.B) {
		for b.Loop() {
			checkAll(retired)
		}
	})
	src := brackets()
	b.Run("brackets", func(b *testing.B) {
		for b.Loop() {
			wiki.Paragraphs(src)
		}
	})
}
