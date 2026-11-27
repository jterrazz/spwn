package wiki_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/wiki"
)

// value: the page of the verify of 2026-11-27 (B2): half of it tables of
// every term of the page, then half terms, so each table would find every
// term and the page would hold terms times tables rows.
func tablesOverTerms() (page string, terms, tables int) {
	var b strings.Builder
	b.WriteString("# H\n\n")
	for b.Len() < 1<<19 {
		b.WriteString("<Table id=\"x" + strconv.Itoa(tables) + "\" of=\"term\" in={ref(\"blueprint/h\")} columns={[\"means\"]} />\n")
		tables++
	}
	for b.Len() < 1<<20-128 {
		b.WriteString("<Term id=\"t" + strconv.Itoa(terms) + "\" means=\"m.\" />\n")
		terms++
	}
	return b.String(), terms, tables
}

// value: the bytes a run allocates, all goroutines counted.
func allocated(run func()) uint64 {
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	run()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}

// value: the page of tables' budgets: its time at least five times its time
// measured on the development VM at a load of 18 or less (2026-11-27), and
// the bytes a read and a serve of it allocate, which the quadratic rows
// took past 4 GiB on the same page.
const (
	budgetTablesRead  = 2 * time.Second
	budgetTablesServe = 500 * time.Millisecond
	ceilingTablesRead = 768 << 20
	ceilingTablesServ = 256 << 20
)

func TestAPageOfTablesOverAPageOfTermsKeepsAndServesTheirFirstRowsOnly(t *testing.T) {
	page, terms, tables := tablesOverTerms()
	files := corpusFiles(t)
	files["blueprint/h.md"] = []byte(page)
	var c wiki.Corpus
	var err error
	read := allocated(func() { c, err = readAll(files) })
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range c.Report.Readings {
		for _, b := range r.Blocks {
			if len(b.Rows) > wiki.MaxTableRows {
				t.Fatalf("%s keeps %d rows; want %d at most", b.Address, len(b.Rows), wiki.MaxTableRows)
			}
		}
	}
	t.Logf("read: %d tables over %d terms, %d MiB allocated", tables, terms, read>>20)
	if read > ceilingTablesRead {
		t.Errorf("reading %d tables over %d terms allocates %d MiB; want at most %d", tables, terms, read>>20, ceilingTablesRead>>20)
	}
	within(t, budgetTablesRead, "reading a page of tables over a page of terms", func() { _, _ = readAll(files) })
	if got := ruled(c.Report, "S31"); len(got) != 101 || got[100] != fmt.Sprintf("blueprint/h and %d more like it on this page", tables-100) {
		t.Errorf("S31 findings %d; want 100 and one that counts the other %d tables", len(got), tables-100)
	}
	var s wiki.Served
	served := allocated(func() { s, err = c.Serve("blueprint/h") })
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("serve: %d MiB allocated, %d items cut", served>>20, s.Cut)
	if served > ceilingTablesServ {
		t.Errorf("serving the page allocates %d MiB; want at most %d", served>>20, ceilingTablesServ>>20)
	}
	within(t, budgetTablesServe, "serving a page of tables over a page of terms", func() { _, _ = c.Serve("blueprint/h") })
	n := 0
	for _, sec := range s.Sections {
		for _, it := range sec.Items {
			if it.Block == nil || it.Block.Table == nil {
				continue
			}
			n++
			if tb := it.Block.Table; len(tb.Rows) != wiki.MaxTableRows || tb.Count != terms || tb.More != terms-wiki.MaxTableRows {
				t.Fatalf("table %s: %d rows, count %d, %d more; want %d, %d, %d", it.Block.ID, len(tb.Rows), tb.Count, tb.More, wiki.MaxTableRows, terms, terms-wiki.MaxTableRows)
			}
		}
	}
	if n == 0 || s.Cut == 0 {
		t.Errorf("%d tables served, %d items cut; want the first tables, and the rest cut past %d bytes", n, s.Cut, wiki.MaxAnswerBytes)
	}
	answer, _ := json.Marshal(s)
	t.Logf("wiki.read answers %d bytes, %d tables served", len(answer), n)
	if len(answer) > wiki.MaxAnswerBytes+1<<20 {
		t.Errorf("wiki.read answers %d bytes; want at most %d and the page's findings", len(answer), wiki.MaxAnswerBytes)
	}
	m, err := c.Map("blueprint/h")
	if err != nil {
		t.Fatal(err)
	}
	mapped, _ := json.Marshal(m)
	t.Logf("wiki.map answers %d bytes, %d nodes cut", len(mapped), m.Cut)
	if len(mapped) > wiki.MaxAnswerBytes+1<<20 || m.Cut == 0 {
		t.Errorf("wiki.map answers %d bytes, %d nodes cut; want at most %d, the rest cut", len(mapped), m.Cut, wiki.MaxAnswerBytes)
	}
}

func TestNoAnswerOfTheFixtureCorpusIsCut(t *testing.T) {
	c, err := readAll(corpusFiles(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range c.Report.Findings {
		if f.Rule == "S31" || f.Cut {
			t.Errorf("%s: %s %q; a page of the fixture hits no limit", f.Page, f.Rule, f.Says)
		}
	}
	for _, p := range c.Pages {
		s, _ := c.Serve(p.ID)
		for _, it := range everyItem(s) {
			if b := it.Block; b != nil {
				fields := b.Fields
				if b.Table != nil {
					for _, r := range b.Table.Rows {
						fields = append(fields, r.Cells...)
					}
				}
				for _, f := range fields {
					if f.More != 0 {
						t.Errorf("%s: field %s serves %d entries fewer than it holds; want all", b.Address, f.Name, f.More)
					}
				}
			}
		}
	}
	for _, p := range c.Pages {
		s, err := c.Serve(p.ID)
		if err != nil {
			t.Fatal(err)
		}
		m, err := c.Map(p.ID)
		if errors.Is(err, wiki.ErrUnknownPage) {
			m, err = c.Map("")
		}
		if err != nil {
			t.Fatal(err)
		}
		if s.Cut != 0 || m.Cut != 0 {
			t.Errorf("%s: %d items and %d nodes cut; want none", p.ID, s.Cut, m.Cut)
		}
	}
}
