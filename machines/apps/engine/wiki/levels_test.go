package wiki_test

import (
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/wiki"
)

func rankOf(d wiki.Detail) int { return slices.Index(wiki.Details(), d) }

// value: what a served page's sections hold, in reading order, a group's own
// blocks and prose after it.
func everyItem(s wiki.Served) []wiki.ServedItem {
	var out []wiki.ServedItem
	var walk func(items []wiki.ServedItem)
	walk = func(items []wiki.ServedItem) {
		for _, it := range items {
			out = append(out, it)
			if it.Group != nil {
				walk(it.Group.Items)
			}
		}
	}
	for _, sec := range s.Sections {
		walk(sec.Items)
	}
	return out
}

func itemName(it wiki.ServedItem) string {
	switch {
	case it.Block != nil:
		return it.Block.Address
	case it.Group != nil:
		return it.Group.Address
	case it.Prose != nil:
		return it.Prose.ID
	}
	return "?"
}

// value: whether the faces of a block, a group or prose rise with the levels and stop at its most:
// a slider stop that shows less than the one before it is a stop that
// takes back what the reader opened.
func risesToItsMost(ranks []wiki.Detail, most wiki.Detail) bool {
	if len(ranks) != len(wiki.Details()) {
		return false
	}
	for i, r := range ranks {
		if rankOf(r) < 0 || rankOf(r) > rankOf(most) || (i > 0 && rankOf(r) < rankOf(ranks[i-1])) {
			return false
		}
	}
	return true
}

func TestEveryItemsFacesRiseWithTheLevelsAndStopAtTheMostItOpens(t *testing.T) {
	n := 0
	for id, s := range servedCorpus(t) {
		if !risesToItsMost(s.Lede.Ranks, wiki.DetailFull) {
			t.Errorf("%s: lede ranks %v", id, s.Lede.Ranks)
		}
		for _, it := range everyItem(s) {
			n++
			if !risesToItsMost(it.Ranks, it.Max) {
				t.Errorf("%s %s: ranks %v over max %q; want four that never fall nor pass it", id, itemName(it), it.Ranks, it.Max)
			}
			want := wiki.DetailFull
			if it.Prose != nil || it.Group != nil || (it.Block != nil && it.Block.Figure != nil) {
				want = wiki.DetailCard
			}
			if it.Max != want {
				t.Errorf("%s %s: max %q; want %q", id, itemName(it), it.Max, want)
			}
		}
	}
	if n < 60 {
		t.Fatalf("%d items over the corpus; want the whole corpus walked", n)
	}
}

// value: the seconds a reading label says, "≈ 1 min 15 s" being 75.
func labelSeconds(t testing.TB, label string) int {
	t.Helper()
	fields := strings.Fields(strings.TrimPrefix(label, "≈ "))
	total := 0
	for i := 0; i+1 < len(fields); i += 2 {
		n, err := strconv.Atoi(fields[i])
		if err != nil {
			t.Fatalf("label %q: %v", label, err)
		}
		switch fields[i+1] {
		case "min":
			total += 60 * n
		case "s":
			total += n
		default:
			t.Fatalf("label %q: unit %q", label, fields[i+1])
		}
	}
	return total
}

// value: whether a label is one of the round forms: a multiple of 5 s from 5 s
// to 55 s, of 15 s from a minute up to five, of a minute past them.
func roundLabel(t testing.TB, label string) bool {
	t.Helper()
	s := labelSeconds(t, label)
	switch {
	case s < 60:
		return s >= 5 && s%5 == 0
	case s < 300:
		return s%15 == 0
	}
	return s%60 == 0
}

func TestEveryLevelReadsAsManyWordsAndARoundLabelNeverShorterOrSaysItShowsTheSame(t *testing.T) {
	for id, s := range servedCorpus(t) {
		if len(s.Reading) != len(wiki.Details()) || s.Reading[0].Same {
			t.Fatalf("%s: readings %+v; want four, the first never the same", id, s.Reading)
		}
		if !roundLabel(t, s.Reading[0].Label) {
			t.Errorf("%s: label %q is not round", id, s.Reading[0].Label)
		}
		for i := 1; i < len(s.Reading); i++ {
			was, is := s.Reading[i-1], s.Reading[i]
			if !roundLabel(t, is.Label) {
				t.Errorf("%s @%s: label %q is not round", id, wiki.Details()[i], is.Label)
			}
			if is.Same && (is.Words != was.Words || is.Label != was.Label) {
				t.Errorf("%s @%s: the same as %s but %+v against %+v", id, wiki.Details()[i], wiki.Details()[i-1], is, was)
			}
			if is.Words < was.Words || labelSeconds(t, is.Label) < labelSeconds(t, was.Label) {
				t.Errorf("%s @%s: %d words %q after %d words %q; want as many words at least and a label no shorter", id, wiki.Details()[i], is.Words, is.Label, was.Words, was.Label)
			}
		}
	}
}

func TestBoundariesShowsSomethingNewAtEveryLevel(t *testing.T) {
	for _, r := range servedCorpus(t)["product/boundaries"].Reading {
		if r.Same {
			t.Errorf("reading %+v is the same; every level of Boundaries opens something", r)
		}
	}
}

func TestAGroupIsMirroredByTheTableWhoseRowsAreItsBlocks(t *testing.T) {
	served := servedCorpus(t)
	for _, c := range []struct{ page, group, table string }{
		{"product/boundaries", "machines-refuses", "machines"},
		{"product/stack", "parts", "all"},
		{"growth/positioning", "alternatives", "all"},
	} {
		var group *wiki.ServedGroup
		var table *wiki.ServedTable
		for _, it := range everyItem(served[c.page]) {
			if it.Group != nil && it.Group.ID == c.group {
				group = it.Group
			}
			if it.Block != nil && it.Block.ID == c.table {
				table = it.Block.Table
			}
		}
		if group == nil || table == nil {
			t.Fatalf("%s: group %q or table %q not served", c.page, c.group, c.table)
		}
		if group.MirroredBy != c.table || group.MirroredFrom != wiki.DetailLine || table.Mirrors != c.group || table.Count != len(table.Rows) {
			t.Errorf("%s: group mirrored by %q from %q, table mirrors %q counts %d of %d rows", c.page, group.MirroredBy, group.MirroredFrom, table.Mirrors, table.Count, len(table.Rows))
		}
	}
	for id, s := range served {
		for _, it := range everyItem(s) {
			if g := it.Group; g != nil && g.MirroredBy != "" && !slices.Contains([]string{"machines-refuses", "parts", "alternatives"}, g.ID) {
				t.Errorf("%s: %s mirrored by %s; no other table's rows are a group's blocks", id, g.ID, g.MirroredBy)
			}
		}
	}
}

func TestARecordsPageHeadsWithTheDecisionItsRecordAndTheBlockNamingItSay(t *testing.T) {
	served := servedCorpus(t)
	for _, c := range []struct{ page, number, title, decides string }{
		{"product/decisions/002-spwn-is-one-stack-of-six-apps-on-one-ladder", "ADR-002", "Spwn is one stack of six apps on one ladder", "Spwn is one stack of six apps on one ladder."},
		{"machines/decisions/035-a-port-is-a-right-with-an-audience", "ADR-035", "A port is a right with an audience", ""},
	} {
		s := served[c.page]
		d := s.Page.Decision
		if s.Page.Number != c.number || s.Page.Title != c.title || d == nil || !d.Resolves || d.Number != c.number || d.Status == "" || d.Date == "" {
			t.Fatalf("%s: head %q %q, decision %+v", c.page, s.Page.Number, s.Page.Title, d)
		}
		if c.decides != "" && d.Decides != c.decides {
			t.Errorf("%s: decides %q; want %q", c.page, d.Decides, c.decides)
		}
		if d.Decides == "" {
			t.Errorf("%s: decides nothing; a block of the corpus names it", c.page)
		}
		if got := details(s.Lede.Ranks); got != "line,line,full,full" {
			t.Errorf("%s: lede ranks %s; want the decision's line, then its full face", c.page, got)
		}
	}
	for id, s := range served {
		if !strings.Contains(id, "/decisions/") && (s.Page.Decision != nil || s.Page.Number != "") {
			t.Errorf("%s: no record, yet heads with %q %+v", id, s.Page.Number, s.Page.Decision)
		}
	}
}

const tinySchema = "wiki: spwn.wiki/v1\nfloors:\n    - id: intention\n      name: Intention\n      question: Why\n    - id: product\n      name: Product\n      question: What\n    - id: domains\n      name: Domains\n      question: Which words\n    - id: architecture\n      name: Architecture\n      question: Where\n    - id: code\n      name: Code\n      question: Which lines\nseats:\n    - id: vision\n      name: Vision\n      path: [blueprint/]\n      reads: [tell, define]\n      home: blueprint/home\nbudget: 2min\n"

func TestAProsesFirstSentenceEndsWhereItsTextEndsOne(t *testing.T) {
	for _, c := range []struct {
		name, prose, first string
		more               bool
	}{
		{"a full stop", "One thing. Then another.", "One thing.", true},
		{"a colon", "The rule: never twice. Then more.", "The rule:", true},
		{"a question", "Why so? Because.", "Why so?", true},
		{"no end", "Words with no stop", "Words with no stop", false},
		{"one sentence", "Only this one.", "Only this one.", false},
		{"a stop at the very end", "Ends here. A", "Ends here.", false},
		{"an abbreviation", "A tool, e.g. a hammer, works. Then.", "A tool, e.g. a hammer, works.", true},
		{"another abbreviation", "Read it, i.e. twice. Then.", "Read it, i.e. twice.", true},
		{"an initial", "Ask J. Doe first. Then.", "Ask J. Doe first.", true},
		{"a stop in code", "Run `make check. now` first. Then.", "Run `make check. now` first.", true},
		{"a stop in a link's words", "See [the vision. here](./vision.md) twice. Then.", "See [the vision.](blueprint/vision)", true},
		{"a link's target", "See [the vision](./a.b. c.md) twice. Then.", "See [the vision](blueprint/a.b. c) twice.", true},
		{"a dot in a word", "Version 0.21 ships. Then.", "Version 0.21 ships.", true},
		{"across paragraphs", "- a point with no stop\n- a second one. Then.", "a point with no stop a second one.", true},
		{"strong", "A **bold claim. Here** then.", "A **bold claim.**", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			files := map[string][]byte{"wiki.yaml": []byte(tinySchema), "blueprint/home.md": []byte("# Home\n\nThe page.\n\n## Part\n\n" + c.prose + "\n"), "blueprint/vision.md": []byte("# Vision\n"), "blueprint/a.b. c.md": []byte("# A\n")}
			corpus, err := readAll(files)
			if err != nil {
				t.Fatal(err)
			}
			s, err := corpus.Serve("blueprint/home")
			if err != nil {
				t.Fatal(err)
			}
			prose := s.Sections[len(s.Sections)-1].Items[0].Prose
			if got := strings.Trim(runs(wiki.Paragraph{Runs: prose.First}), `"`); got != c.first {
				t.Fatalf("first = %q; want %q", got, c.first)
			}
			if prose.More != c.more {
				t.Errorf("more = %v; want %v", prose.More, c.more)
			}
		})
	}
}

// invariant: a block whose line Face renders no words has no line, and its
// line face shows its word, so the level that opens it to a line reads no
// fewer words than the level before.
func TestABlockWhoseLineSaysNothingShowsItsWordAndNoLevelReadsLess(t *testing.T) {
	for _, c := range []struct{ name, source string }{
		{"blank, the input FuzzServe found", "000000000\n\n<Term id=\"00000\" means=\" \"/>"},
		{"blank under a heading", "# Home\n\nThe page.\n\n## A\n\n<Term id=\"seat\" means=\"   \" />\n"},
		{"code of a space", "# Home\n\nThe page.\n\n## A\n\n<Term id=\"seat\" means=\"` `\" />\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			corpus, err := readAll(map[string][]byte{"wiki.yaml": []byte(tinySchema), "blueprint/home.md": []byte(c.source)})
			if err != nil {
				t.Fatal(err)
			}
			s, err := corpus.Serve("blueprint/home")
			if err != nil {
				t.Fatal(err)
			}
			for _, it := range everyItem(s) {
				if it.Block != nil && it.Block.Line != "" {
					t.Errorf("%s: line %q", it.Block.ID, it.Block.Line)
				}
			}
			for i := 1; i < len(s.Reading); i++ {
				if s.Reading[i].Words < s.Reading[i-1].Words {
					t.Errorf("@%d: %+v after %+v", i, s.Reading[i], s.Reading[i-1])
				}
			}
		})
	}
}

// value: tinySchema with one kind of the corpus, its word and line faces the
// templates given.
func noteSchema(word, line string) string {
	return tinySchema + "kinds:\n    note:\n        title:\n            en: Note\n            fr: Note\n        weighs: 1\n        fields:\n            says:\n                type: sentence\n                required: true\n            then:\n                type: sentence\n        faces:\n            word: " + strconv.Quote(word) + "\n            line: " + strconv.Quote(line) + "\n"
}

// value: a text of three characters the engine removes, an escape, a
// right-to-left override and a line separator, around an emoji drawn with
// a zero-width joiner, which it keeps; built at run time so no line of the
// source holds one.
var silentText = "an " + string(rune(0x1b)) + "x escape, a " + string(rune(0x202e)) + " bidi override and a " + string(rune(0x2028)) + " separator, its \U0001F468" + string(rune(0x200d)) + "\U0001F469 kept."

// value: whether a served answer holds a character that draws nothing or
// reorders text, walked by reflection through every field at any depth,
// unexported and untagged ones too, so a field added later is seen.
func holdsSilent(t testing.TB, answer any) string {
	t.Helper()
	silent := func(s string) bool {
		return strings.ContainsFunc(s, func(r rune) bool {
			return (r < 0x20 && r != '\n' && r != '\t') || (r >= 0x7f && r <= 0x9f) || (r >= 0x202a && r <= 0x202e) || (r >= 0x2066 && r <= 0x2069) || r == 0x2028 || r == 0x2029
		})
	}
	var walk func(v reflect.Value) string
	walk = func(v reflect.Value) string {
		switch v.Kind() {
		case reflect.String:
			if silent(v.String()) {
				return v.String()
			}
		case reflect.Pointer, reflect.Interface:
			if !v.IsNil() {
				return walk(v.Elem())
			}
		case reflect.Struct:
			for i := range v.NumField() {
				if s := walk(v.Field(i)); s != "" {
					return s
				}
			}
		case reflect.Slice, reflect.Array:
			for i := range v.Len() {
				if s := walk(v.Index(i)); s != "" {
					return s
				}
			}
		case reflect.Map:
			for _, k := range v.MapKeys() {
				if s := walk(k); s != "" {
					return s
				}
				if s := walk(v.MapIndex(k)); s != "" {
					return s
				}
			}
		default:
		}
		return ""
	}
	return walk(reflect.ValueOf(answer))
}

func TestAnEscapeWikiYamlDecodesIsRemovedLikeARawCharacter(t *testing.T) {
	for _, tc := range []struct {
		name, word, line string
		removed          int
	}{
		{name: "a C0 control spelled \\x10 (FuzzServe's input)", word: `"0"`, line: `"\x10"`, removed: 1},
		{name: "a bidi override spelled \\u202e", word: `"\` + `u202e{says}"`, line: `"{says}"`, removed: 1},
		{name: "an escape spelled \\e", word: `"\e{says}"`, line: `"{says}\e"`, removed: 2},
		{name: "a line separator spelled \\L", word: `"{says}"`, line: `"\L{says}"`, removed: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			yaml := strings.Replace(strings.Replace(noteSchema("WORD", "LINE"), `"WORD"`, tc.word, 1), `"LINE"`, tc.line, 1)
			corpus, err := readAll(map[string][]byte{"wiki.yaml": []byte(yaml), "blueprint/home.md": []byte("# Home\n\n<Note/>\n\n<Note id=\"a\" says=\"one.\" />\n")})
			if err != nil {
				t.Fatal(err)
			}
			s, err := corpus.Serve("blueprint/home")
			if err != nil {
				t.Fatal(err)
			}
			if bad := holdsSilent(t, s); bad != "" {
				t.Errorf("wiki.read serves %q", bad)
			}
			m, err := corpus.Map("blueprint/home")
			if err != nil {
				t.Fatal(err)
			}
			if bad := holdsSilent(t, m); bad != "" {
				t.Errorf("wiki.map serves %q", bad)
			}
			if bad := holdsSilent(t, corpus); bad != "" {
				t.Errorf("the compiled corpus holds %q", bad)
			}
			want := wiki.Finding{Rule: "S32", Page: "wiki.yaml", Line: 1, Col: 1, Says: strconv.Itoa(tc.removed) + " characters that draw nothing or reorder text were removed"}
			if !slices.ContainsFunc(corpus.Report.Findings, func(f wiki.Finding) bool {
				return f.Rule == want.Rule && f.Page == want.Page && f.Says == want.Says
			}) {
				t.Errorf("no %+v in %+v", want, corpus.Report.Findings)
			}
		})
	}
}

func FuzzServe(f *testing.F) {
	for _, source := range corpusSources(f) {
		f.Add(source, "{id}", "{says}")
	}
	f.Add("# Home\n\nOne. Two: three e.g. four.\n\n## A\n\n<Term id=\"a\" means=\"b\" />\n", "{id}", "{says}")
	f.Add("000000000\n\n<Term id=\"00000\" means=\" \"/>", "", "")
	f.Add("# Home\n\n<Note id=\"a\" says=\"one.\" then=\"two.\" />\n", "{says} {then} {says} {then} {says}", "{id}")
	f.Add("# H"+silentText+"\n\nA [x](a"+silentText+"b).\n\n<Note id=\"a\" says=\""+silentText+"\" />\n", "{says}", "{says}")
	f.Add("<Note/>", "0", "\x10")
	f.Fuzz(func(t *testing.T, source, word, line string) {
		corpus, err := readAll(map[string][]byte{"wiki.yaml": []byte(noteSchema(word, line)), "blueprint/home.md": []byte(source)})
		if err != nil {
			return
		}
		s, err := corpus.Serve("blueprint/home")
		if err != nil {
			return
		}
		if bad := holdsSilent(t, s); bad != "" {
			t.Fatalf("wiki.read serves %q", bad)
		}
		m, err := corpus.Map("blueprint/home")
		if err == nil {
			if bad := holdsSilent(t, m); bad != "" {
				t.Fatalf("wiki.map serves %q", bad)
			}
		}
		if !risesToItsMost(s.Lede.Ranks, wiki.DetailFull) || len(s.Reading) != len(wiki.Details()) || s.Reading[0].Same {
			t.Fatalf("lede %v, readings %+v", s.Lede.Ranks, s.Reading)
		}
		for _, it := range everyItem(s) {
			if !risesToItsMost(it.Ranks, it.Max) {
				t.Fatalf("%s: ranks %v over max %q", itemName(it), it.Ranks, it.Max)
			}
		}
		for i := 1; i < len(s.Reading); i++ {
			was, is := s.Reading[i-1], s.Reading[i]
			if (is.Same && is.Label != was.Label) || is.Words < was.Words || labelSeconds(t, is.Label) < labelSeconds(t, was.Label) || !roundLabel(t, is.Label) {
				t.Fatalf("@%d: %+v after %+v", i, is, was)
			}
		}
	})
}
