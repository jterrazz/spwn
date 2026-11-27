package wiki_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/wiki"
)

// value: the ceilings a corpus of a schema at every cap and one page of
// minimal blocks within MaxPageBytes is held to, in each of its read, its
// serve and its map: the bytes allocated, and a time five times or more the
// slowest measured by BenchmarkThePagesUnderASchemaAtEveryCap on
// development-linux-mini (2026-11-27).
const (
	ceilingAtCapsBytes = 256 << 20
	budgetAtCapsRead   = 2 * time.Second
	budgetAtCapsServe  = time.Second
)

func padded(head string, n int) string { return head + strings.Repeat("a", n-len(head)) }

// value: the field a face template names, the shortest a field may be, so
// a template of MaxTemplateBytes names it the most times.
const shortField = "a"

func templateAtCap() string {
	return strings.Repeat("{"+shortField+"}", wiki.MaxTemplateBytes/len("{"+shortField+"}"))
}

// value: wiki.yaml at every cap the schema reads, within MaxPageBytes: five
// floors of MaxFloorTextBytes, MaxPaths floor_of folders, MaxSeats seats of
// MaxNameBytes holding MaxPaths folders and opening on a home of
// MaxPageIDBytes no page is, MaxRepos repositories and mounts, one kind of
// MaxFields fields, each but the short one an enum of MaxValues values, its
// three faces MaxTemplateBytes naming its short field, and MaxIcons icons;
// kinds more kinds of one field each.
func schemaAtCaps(kinds int) string {
	var lines []string
	w := func(format string, args ...any) { lines = append(lines, fmt.Sprintf(format, args...)) }
	w("wiki: spwn.wiki/v1\nfloors:\n")
	for _, id := range []string{"intention", "product", "domains", "architecture", "code"} {
		w("    - id: %s\n      name: %s\n      question: %s\n", id, padded("N", wiki.MaxFloorTextBytes), padded("Q", wiki.MaxFloorTextBytes))
	}
	w("floor_of:\n")
	for i := range wiki.MaxPaths {
		w("    fo%03d/: %d\n", i, i%5)
	}
	w("seats:\n")
	per := wiki.MaxPaths / wiki.MaxSeats
	for i := range wiki.MaxSeats {
		paths := make([]string, 0, per)
		for j := range per {
			p := fmt.Sprintf("s%02d-%d/", i, j)
			if i == 0 && j == 0 {
				p = "product/"
			}
			paths = append(paths, p)
		}
		w("    - id: %s\n      name: %s\n      path: [%s]\n      reads: [tell, define, decide, locate, guard, prove]\n      home: %s\n",
			padded(fmt.Sprintf("s%02d", i), wiki.MaxNameBytes), padded("Seat", wiki.MaxNameBytes), strings.Join(paths, ", "), padded("product/w#h", wiki.MaxPageIDBytes))
	}
	w("repos:\n")
	for i := range wiki.MaxRepos {
		w("    %s: %s\n", padded(fmt.Sprintf("r%02d", i), wiki.MaxNameBytes), padded("../r", wiki.MaxPageIDBytes))
	}
	w("mounts:\n")
	for i := range wiki.MaxRepos {
		w("    - at: %s\n      repo: %s\n      path: %s\n", padded(fmt.Sprintf("m%02d", i), wiki.MaxNameBytes), padded("r00", wiki.MaxNameBytes), padded("docs", wiki.MaxPageIDBytes))
	}
	w("budget: 2min\nkinds:\n    wide:\n        fields:\n            %s:\n                type: sentence\n", shortField)
	for i := 1; i < wiki.MaxFields; i++ {
		values := make([]string, 0, wiki.MaxValues)
		for v := range wiki.MaxValues {
			values = append(values, padded(fmt.Sprintf("v%02d", v), wiki.MaxNameBytes))
		}
		w("            %s:\n                type: enum\n                values: [%s]\n", padded(fmt.Sprintf("f%02d", i), wiki.MaxNameBytes), strings.Join(values, ", "))
	}
	w("        faces:\n            word: %q\n            line: %q\n            icon: %q\n", templateAtCap(), templateAtCap(), templateAtCap())
	for i := 1; i < kinds; i++ {
		w("    k%d:\n        fields:\n            x:\n                type: sentence\n", i)
	}
	w("icons:\n")
	grid := strings.NewReplacer("0", ".", "1", "#")
	for i := range wiki.MaxIcons {
		w("    %s: [%q]\n", padded(fmt.Sprintf("i%03d", i), wiki.MaxNameBytes), grid.Replace(fmt.Sprintf("%012b", i)))
	}
	return strings.Join(lines, "")
}

// value: the schema at the caps with the most corpus kinds the engine reads
// beside its own, found by the schema it reads.
func kindsAtCaps(tb testing.TB) string {
	tb.Helper()
	for n := 16; n > 0; n-- {
		if _, err := wiki.ReadSchema("wiki.yaml", []byte(schemaAtCaps(n))); err == nil {
			return schemaAtCaps(n)
		}
	}
	_, err := wiki.ReadSchema("wiki.yaml", []byte(schemaAtCaps(1)))
	tb.Fatalf("the schema at the caps is refused: %v", err)
	return ""
}

func blocksFilling(unit func(i int) string) func() string {
	return func() string {
		var b strings.Builder
		b.WriteString("# W\n\nThe page.\n\n")
		for i := 0; ; i++ {
			u := unit(i)
			if b.Len()+len(u) > wiki.MaxPageBytes {
				return b.String()
			}
			b.WriteString(u)
		}
	}
}

type atCapsPage struct {
	name string
	page func() string
}

func atCapsPages() []atCapsPage {
	field := padded("f01", wiki.MaxNameBytes)
	value := padded("v00", wiki.MaxNameBytes)
	return []atCapsPage{
		{"B4 groups each read by every seat", blocksFilling(func(i int) string {
			return fmt.Sprintf("<Group id=\"g%d\" title=\"g\" intent=\"tell\">\n\n<Wide %s=\"x\" />\n\n</Group>\n\n", i, shortField)
		})},
		{"B5 blocks on a floor of their own", blocksFilling(func(int) string { return "<Wide " + shortField + "=\"x\" floor={4} />\n\n" })},
		{"B6 blocks writing the kind's fields", blocksFilling(func(i int) string {
			return fmt.Sprintf("<Wide id=\"b%d\" %s=\"x\" %s=%q />\n\n", i, shortField, field, value)
		})},
		{"B6 blocks naming an icon of none", blocksFilling(func(int) string { return "<Wide " + shortField + "=\"x\" icon=\"none\" />\n\n" })},
		{"B6 tables whose where indexes every field", func() string {
			parts := []string{"# W\n\nThe page.\n\n"}
			n := len(parts[0])
			for i := 0; n < wiki.MaxPageBytes/2; i++ {
				parts = append(parts, fmt.Sprintf("<Wide id=\"b%d\" %s=\"x\" %s=%q />\n\n", i, shortField, field, value))
				n += len(parts[len(parts)-1])
			}
			for i := 1; n < wiki.MaxPageBytes-400; i++ {
				f := padded(fmt.Sprintf("f%02d", 1+i%(wiki.MaxFields-1)), wiki.MaxNameBytes)
				parts = append(parts, fmt.Sprintf("\n\n<Table id=\"t%d\" of=\"wide\" in={ref(%q)} where={query({ %s: { is: %q } })} columns={[%q]} />\n", i, worstID, f, value, shortField))
				n += len(parts[len(parts)-1])
			}
			return strings.Join(parts, "")
		}},
		{"B6 blocks whose addresses are near every seat's home", blocksFilling(func(i int) string {
			return fmt.Sprintf("<Wide id=%q %s=\"x\" />\n\n", padded(fmt.Sprintf("h%d-", i), wiki.MaxPageIDBytes-len(worstID)-1), shortField)
		})},
	}
}

func filesAtCaps(tb testing.TB, page string) map[string][]byte {
	tb.Helper()
	schema := kindsAtCaps(tb)
	if len(schema) > wiki.MaxPageBytes || len(page) > wiki.MaxPageBytes {
		tb.Fatalf("the schema at the caps is %d bytes and the page %d, past MaxPageBytes", len(schema), len(page))
	}
	return map[string][]byte{"wiki.yaml": []byte(schema), worstID + ".md": []byte(page)}
}

func TestAPageUnderASchemaAtEveryCapIsReadServedAndMappedWithinItsCeilings(t *testing.T) {
	for _, w := range atCapsPages() {
		t.Run(w.name, func(t *testing.T) {
			files := filesAtCaps(t, w.page())
			var c wiki.Corpus
			var err error
			read := allocated(func() { c, err = readAll(files) })
			if err != nil {
				t.Fatal(err)
			}
			var s wiki.Served
			var m wiki.Map
			var serr, merr error
			served := allocated(func() { s, serr = c.Serve(worstID) })
			mapped := allocated(func() { m, merr = c.Map(worstID) })
			if serr != nil || merr != nil {
				t.Fatal(serr, merr)
			}
			t.Logf("WORST %s: read %d MiB, serve %d MiB, map %d MiB allocated, %d findings, cut %d and %d", w.name, read>>20, served>>20, mapped>>20, len(c.Report.Findings), s.Cut, m.Cut)
			for what, n := range map[string]uint64{"reading": read, "serving": served, "mapping": mapped} {
				if n > ceilingAtCapsBytes {
					t.Errorf("%s allocates %d MiB; want %d MiB at most", what, n>>20, ceilingAtCapsBytes>>20)
				}
			}
			within(t, budgetAtCapsRead, "reading "+w.name, func() { _, _ = readAll(files) })
			within(t, budgetAtCapsServe, "serving and mapping "+w.name, func() {
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
		})
	}
}

func invalidFaces() string {
	return "# W\n\nThe page.\n\n" + strings.Repeat("<Wide "+shortField+"=\""+strings.Repeat("\xff", 48)+"\" />\n\n", 2000)
}

func TestFacesOfInvalidUTF8AreMeteredAtTheirJSONCost(t *testing.T) {
	c, err := readAll(filesAtCaps(t, invalidFaces()))
	if err != nil {
		t.Fatal(err)
	}
	s, err := c.Serve(worstID)
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) > wiki.MaxAnswerBytes+answerSlack || s.Cut == 0 {
		t.Errorf("2,000 blocks of faces of invalid UTF-8 answer %d bytes, %d cut; want %d at most, some cut", len(body), s.Cut, wiki.MaxAnswerBytes+answerSlack)
	}
}

func BenchmarkThePagesUnderASchemaAtEveryCap(b *testing.B) {
	for _, w := range atCapsPages() {
		files := filesAtCaps(b, w.page())
		b.Run(w.name+"/read", func(b *testing.B) {
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
		b.Run(w.name+"/serve", func(b *testing.B) {
			for b.Loop() {
				_, _ = c.Serve(worstID)
			}
		})
		b.Run(w.name+"/map", func(b *testing.B) {
			for b.Loop() {
				_, _ = c.Map(worstID)
			}
		})
	}
}

func schemaFindings(c wiki.Corpus, rule string) []string {
	var out []string
	for _, f := range c.Report.Findings {
		if f.Page == "wiki.yaml" && f.Rule == rule {
			out = append(out, f.Says)
		}
	}
	return out
}

func TestASchemaAtEveryCapIsReadWithNoFindingOfACap(t *testing.T) {
	c, err := readAll(filesAtCaps(t, "# W\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got := schemaFindings(c, "H44"); len(got) > 0 {
		t.Errorf("a schema at its caps says %q; want no H44", got)
	}
}

func TestTheFixtureHitsNoCap(t *testing.T) {
	c, err := readAll(corpusFiles(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range c.Report.Findings {
		if f.Rule == "H44" || f.Rule == "S31" {
			t.Errorf("the fixture corpus says %s on %s: %q; want no cap reached", f.Rule, f.Page, f.Says)
		}
	}
}

func TestEachCapOfTheSchemaPassedIsOneHardFindingAndKeepsTheCap(t *testing.T) {
	wide := padded("f01", wiki.MaxNameBytes)
	for _, c := range []struct {
		cap      string
		from, to string
		kept     func(wiki.Schema) int
		want     int
	}{
		{"MaxSeats", "repos:\n", "    - id: zz\n      name: Z\n      path: [zz/]\n      home: zz/home\nrepos:\n", func(s wiki.Schema) int { return len(s.Seats) }, wiki.MaxSeats},
		{"MaxPaths", "path: [product/,", "path: [zz/, product/,", func(s wiki.Schema) int {
			n := 0
			for _, s := range s.Seats {
				n += len(s.Paths)
			}
			return n
		}, wiki.MaxPaths},
		{"MaxFields", "        fields:\n            " + shortField + ":\n", "        fields:\n            zz:\n                type: sentence\n            " + shortField + ":\n", func(s wiki.Schema) int { return len(s.Kinds["wide"].Fields) }, wiki.MaxFields},
		{"MaxIcons", "icons:\n", "icons:\n    zz: [\"#.#\"]\n", func(s wiki.Schema) int { return len(s.Icons) }, wiki.MaxIcons},
		{"MaxValues", "values: [", "values: [zz, ", func(s wiki.Schema) int {
			for _, f := range s.Kinds["wide"].Fields {
				if f.Name == wide {
					return len(f.Values)
				}
			}
			return -1
		}, wiki.MaxValues},
		{"MaxTemplateBytes", "word: \"{", "word: \"zz{", func(s wiki.Schema) int { return len(s.Kinds["wide"].Faces.Word) }, wiki.MaxTemplateBytes},
		{"MaxFloorTextBytes", "question: Q", "question: QQ", func(s wiki.Schema) int { return len(s.Floors[0].Question) }, wiki.MaxFloorTextBytes},
		{"MaxNameBytes", "name: Seat", "name: SSeat", func(s wiki.Schema) int { return len(s.Seats[0].Name) }, wiki.MaxNameBytes},
		{"MaxPageIDBytes", "home: product/w#h", "home: pproduct/w#h", func(s wiki.Schema) int { return len(s.Seats[0].Home) }, wiki.MaxPageIDBytes},
		{"MaxRepos", "repos:\n", "repos:\n    zz: ../zz\n", func(s wiki.Schema) int { return len(s.Repos) }, wiki.MaxRepos},
	} {
		t.Run(c.cap, func(t *testing.T) {
			yaml := kindsAtCaps(t)
			if !strings.Contains(yaml, c.from) {
				t.Fatalf("the schema holds no %q", c.from)
			}
			corpus, err := readAll(map[string][]byte{"wiki.yaml": []byte(strings.Replace(yaml, c.from, c.to, 1)), worstID + ".md": []byte("# W\n")})
			if err != nil {
				t.Fatal(err)
			}
			got := schemaFindings(corpus, "H44")
			if len(got) != 1 || !strings.Contains(got[0], c.cap) {
				t.Errorf("one cap passed says %q; want one H44 naming %s", got, c.cap)
			}
			if n := c.kept(corpus.Schema); n != c.want {
				t.Errorf("%d kept; want the cap, %d", n, c.want)
			}
		})
	}
}
