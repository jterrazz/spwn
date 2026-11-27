package wiki_test

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/jterrazz/studio-station/apps/engine/wiki"
)

// value: the ceilings one wiki.read or wiki.map of one page within the
// limits is held to, whatever the page: the bytes it allocates, a sixteenth
// of what the widest table took before its columns were capped (4,875 MiB,
// measured 2026-11-27), its time, five times or more the slowest worst page
// measured, and its JSON, MaxAnswerBytes and the answer's few fields the
// meter does not count.
const (
	ceilingServeBytes = 64 << 20
	budgetWorstServe  = time.Second
	answerSlack       = 64 << 10
)

// value: a page of the worst inputs the verify and the measures of
// 2026-11-27 found, each within MaxPageBytes, and what serving it must say.
type worstPage struct {
	name string
	page func() string
	says func(t *testing.T, c wiki.Corpus, s wiki.Served)
}

const worstID = "product/w"

func rowsOf(kind string, n int, fields string) string {
	var b strings.Builder
	b.WriteString("# W\n\nThe page.\n\n")
	for i := range n {
		b.WriteString("<" + kind + " id=\"r" + strconv.Itoa(i) + "\" " + fields + " />\n\n")
	}
	return b.String()
}

func columns(name string, n int) string {
	return "[" + strings.Repeat(`"`+name+`",`, n) + "]"
}

func wideTable() string {
	return rowsOf("Refusal", 200, `refuses="x" because="y"`) + fmt.Sprintf("<Table id=\"t\" of=\"refusal\" in={ref(%q)} columns={%s} />\n", worstID, columns("refuses", 90000))
}

func wideLists() string {
	refs := "[" + strings.Repeat(`ref("product/boundaries"),`, 100) + "]"
	return rowsOf("Conviction", 200, `says="x" opposite="y" carried_in={`+refs+`} broken_by="z"`) + fmt.Sprintf("<Table id=\"t\" of=\"conviction\" in={ref(%q)} columns={%s} />\n", worstID, columns("carried_in", 2000))
}

func tableOf(s wiki.Served) *wiki.ServedTable {
	for _, sec := range s.Sections {
		for _, it := range sec.Items {
			if it.Block != nil && it.Block.Table != nil {
				return it.Block.Table
			}
		}
	}
	return nil
}

func saidOnce(t *testing.T, c wiki.Corpus, rule, says string) {
	t.Helper()
	n := 0
	for _, f := range c.Report.Findings {
		if f.Page == worstID && f.Rule == rule && strings.Contains(f.Says, says) {
			n++
		}
	}
	if n != 1 {
		t.Errorf("%d findings %s saying %q; want one", n, rule, says)
	}
}

func headings() string {
	var b strings.Builder
	b.WriteString("# S\n\nThe page.\n\n")
	for b.Len() < wiki.MaxPageBytes-8 {
		b.WriteString("## a\n\n")
	}
	return b.String()
}

func longKeys(key string) func() string {
	return func() string {
		var b strings.Builder
		b.WriteString("---\n")
		for i := range 100 {
			name := fmt.Sprintf("k%03d", i)
			b.WriteString(name + key + ": 1\n")
		}
		b.WriteString("---\n# H\n\nThe page.\n")
		return b.String()
	}
}

func saysCut(t *testing.T, c wiki.Corpus, s wiki.Served) {
	cut := 0
	for _, f := range c.Report.Findings {
		if len(f.Says) > wiki.MaxSaysBytes || len(f.Says) < wiki.MaxSaysBytes-utf8.UTFMax && f.Cut || !utf8.ValidString(f.Says) {
			t.Fatalf("%s says %d bytes; want %d at most, cut on a character", f.Rule, len(f.Says), wiki.MaxSaysBytes)
		}
		if f.Page == worstID && f.Cut {
			cut++
		}
	}
	served := 0
	for _, f := range s.Findings {
		if f.Cut {
			served++
		}
	}
	if cut != 100 || served != 100 {
		t.Errorf("%d findings cut, %d served cut; want the 100 that quote a key", cut, served)
	}
}

func filled(head, unit, tail string) func() string {
	return func() string {
		return head + strings.Repeat(unit, (wiki.MaxPageBytes-len(head)-len(tail)-1)/len(unit)) + tail
	}
}

func termsAndTables() string {
	var b strings.Builder
	b.WriteString("# H\n\n")
	for i := 0; b.Len() < 1<<19; i++ {
		b.WriteString("<Table id=\"x" + strconv.Itoa(i) + "\" of=\"term\" in={ref(\"" + worstID + "\")} columns={[\"means\"]} />\n")
	}
	for i := 0; b.Len() < wiki.MaxPageBytes-128; i++ {
		b.WriteString("<Term id=\"t" + strconv.Itoa(i) + "\" means=\"m.\" />\n")
	}
	return b.String()
}

func cutSomewhere(t *testing.T, _ wiki.Corpus, s wiki.Served) {
	if s.Cut == 0 {
		t.Errorf("nothing cut; want the page past the meter")
	}
}

func keptFirst(what string, n int) func(t *testing.T, c wiki.Corpus, s wiki.Served) {
	return func(t *testing.T, c wiki.Corpus, _ wiki.Served) {
		saidOnce(t, c, "S31", fmt.Sprintf("keeps its first %d %s", n, what))
	}
}

func fitsAnyway(*testing.T, wiki.Corpus, wiki.Served) {}

func worstPages() []worstPage {
	return []worstPage{
		{"tables of every term of the page over a page of terms (B2)", termsAndTables, cutSomewhere},
		{"a megabyte of tiny blocks", filled("# B\n\nThe page.\n\n", "<Term means=\"m\" />\n\n", ""), cutSomewhere},
		{"a paragraph of a quarter million code runs", filled("# P\n\nThe page.\n\n", "`a` ", "\n"), cutSomewhere},
		{"a table line of half a million cells", filled("# P\n\nThe page.\n\n|", "a|", "\n"), cutSomewhere},
		{"a lede of a megabyte of escaped characters", filled("# P\n\n", "<\"", "\n"), cutSomewhere},
		{"a heading of a megabyte", filled("# P\n\nThe page.\n\n## ", "a", "\n"), fitsAnyway},
		{"a block whose id is a megabyte", filled("# P\n\nThe page.\n\n<Term id=\"", "a", "\" means=\"m\" />\n"), fitsAnyway},
		{"a figure of seventy thousand parts written", filled("# P\n\nThe page.\n\n<Figure id=\"f\" form=\"stack\" parts={[", `{ name: "a" }, `, "]} caption=\"c.\" />\n"), keptFirst("parts", wiki.MaxTableRows)},
		{"100 frontmatter keys of 10,000 ASCII bytes", longKeys(strings.Repeat("a", 10000)), saysCut},
		{"100 frontmatter keys of 10,000 bytes of invalid UTF-8", longKeys(strings.Repeat("\x80", 10000)), saysCut},
		{"100 frontmatter keys of 5,000 two-byte characters", longKeys(strings.Repeat("é", 5000)), saysCut},
		{"a page of 174,760 headings", headings, func(t *testing.T, c wiki.Corpus, s wiki.Served) {
			saidOnce(t, c, "S31", fmt.Sprintf("keeps its first %d sections: %d more headings", wiki.MaxSections, (wiki.MaxPageBytes-8-len("# S\n\nThe page.\n\n"))/6+1-wiki.MaxSections+1))
			if len(s.Sections) > wiki.MaxSections || s.Cut == 0 {
				t.Errorf("%d sections served, %d cut; want %d at most, the headings past them read as prose and cut past the meter", len(s.Sections), s.Cut, wiki.MaxSections)
			}
		}},
		{"a table of 90,000 columns over 200 rows", wideTable, func(t *testing.T, c wiki.Corpus, s wiki.Served) {
			saidOnce(t, c, "S31", fmt.Sprintf("keeps its first %d columns, and drops %d more", wiki.MaxTableColumns, 90000-wiki.MaxTableColumns))
			if tb := tableOf(s); tb == nil || len(tb.Columns) != wiki.MaxTableColumns || len(tb.Rows) != 200 {
				t.Errorf("table %+v; want %d columns and 200 rows served", tb != nil, wiki.MaxTableColumns)
			}
		}},
		{"a table of 2,000 columns of a list of 100 refs over 200 rows", wideLists, func(t *testing.T, c wiki.Corpus, s wiki.Served) {
			saidOnce(t, c, "S31", fmt.Sprintf("keeps its first %d columns, and drops %d more", wiki.MaxTableColumns, 2000-wiki.MaxTableColumns))
			if tableOf(s) != nil || s.Cut == 0 {
				t.Errorf("a table of %d by 200 cells of %d refs is served, %d cut; want it past the meter, cut", wiki.MaxTableColumns, wiki.MaxCellRefs, s.Cut)
			}
			var first *wiki.ServedBlock
			for _, it := range everyItem(s) {
				if it.Block != nil && first == nil {
					first = it.Block
				}
			}
			if first == nil {
				t.Fatalf("no block served")
			}
			for _, f := range first.Fields {
				if f.Name == "carried_in" && (len(f.Refs) != wiki.MaxCellRefs || f.More != 100-wiki.MaxCellRefs) {
					t.Errorf("a field of %d refs, %d more; want %d and %d", len(f.Refs), f.More, wiki.MaxCellRefs, 100-wiki.MaxCellRefs)
				}
			}
		}},
	}
}

func TestEveryWorstPageIsReadAndMappedWithinItsCeilings(t *testing.T) {
	for _, w := range worstPages() {
		t.Run(w.name, func(t *testing.T) {
			files := corpusFiles(t)
			files[worstID+".md"] = []byte(w.page())
			if n := len(files[worstID+".md"]); n > wiki.MaxPageBytes {
				t.Fatalf("the page is %d bytes, past MaxPageBytes", n)
			}
			c, err := readAll(files)
			if err != nil {
				t.Fatal(err)
			}
			var s wiki.Served
			var m wiki.Map
			var serr, merr error
			got := allocated(func() { s, serr = c.Serve(worstID) })
			mapped := allocated(func() { m, merr = c.Map(worstID) })
			if serr != nil || merr != nil {
				t.Fatal(serr, merr)
			}
			t.Logf("WORST %s: serve %d KiB, map %d KiB allocated, cut %d and %d", w.name, got>>10, mapped>>10, s.Cut, m.Cut)
			if got > ceilingServeBytes || mapped > ceilingServeBytes {
				t.Errorf("serving allocates %d MiB, mapping %d MiB; want %d MiB at most", got>>20, mapped>>20, ceilingServeBytes>>20)
			}
			within(t, budgetWorstServe, "serving and mapping "+w.name, func() {
				_, _ = c.Serve(worstID)
				_, _ = c.Map(worstID)
			})
			for what, answer := range map[string]any{"wiki.read": s, "wiki.map": m} {
				body, err := json.Marshal(answer)
				if err != nil {
					t.Fatal(err)
				}
				if len(body) > wiki.MaxAnswerBytes+answerSlack {
					t.Errorf("%s answers %d bytes; want %d at most", what, len(body), wiki.MaxAnswerBytes+answerSlack)
				}
			}
			w.says(t, c, s)
		})
	}
}
