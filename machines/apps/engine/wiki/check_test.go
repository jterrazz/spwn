package wiki_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/wiki"
)

const checkGolden = "check-corpus.txt"

func corpusSources(t testing.TB) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, id := range corpusPages() {
		out[id] = read(t, filepath.Join(corpusDir, id+".md"))
	}
	return out
}

func checked(schema wiki.Schema, sources map[string]string) wiki.Report {
	ids := make([]string, 0, len(sources))
	for id := range sources {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	pages := make([]wiki.Page, 0, len(ids))
	for _, id := range ids {
		pages = append(pages, wiki.Compile(id, sources[id], schema))
	}
	return checkWith(schema, pages)
}

func checkWith(schema wiki.Schema, pages []wiki.Page) wiki.Report {
	r, err := wiki.Check(context.Background(), schema, pages)
	if err != nil {
		panic(err)
	}
	return r
}

func checkAll(pages []wiki.Page) wiki.Report { return checkWith(spwnKinds(), pages) }

func TestAnAbandonedCheckEndsWithItsContextsError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := wiki.Check(ctx, spwnKinds(), []wiki.Page{wiki.Compile("blueprint/a", "# A\n", spwnKinds())}); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func line(f wiki.Finding) string {
	level := string(f.Level)
	if f.Hard {
		level = "hard"
	}
	return fmt.Sprintf("%s:%d:%d %s %s %s", f.Page, f.Line, f.Col, f.Rule, level, f.Says)
}

// value: a report as a reviewer reads it: its findings, then each page's
// seat and floor and each block's floor, weight with its causes, rows and lead.
func render(r wiki.Report) string {
	var b strings.Builder
	b.WriteString("findings\n")
	for _, f := range r.Findings {
		b.WriteString("  " + line(f) + "\n")
	}
	b.WriteString("readings\n")
	for _, p := range r.Readings {
		b.WriteString("  " + p.Page + " seat=" + p.Seat + " floor=" + strconv.Itoa(p.Floor) + "\n")
		for _, blk := range p.Blocks {
			b.WriteString("    " + blk.Address + " " + blk.Kind + " floor=" + strconv.Itoa(blk.Floor) + weighed(blk.Weight))
			if blk.Lead != "" {
				b.WriteString(" lead=" + blk.Lead)
			}
			if len(blk.ReadBy) > 0 {
				b.WriteString(" readBy=" + strings.Join(blk.ReadBy, ","))
			}
			if blk.Kind == "table" {
				b.WriteString(" rows=" + strings.Join(blk.Rows, ","))
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}

func weighed(w *wiki.Weight) string {
	if w == nil {
		return ""
	}
	why := make([]string, 0, len(w.Why))
	for _, c := range w.Why {
		why = append(why, c.Name+":"+strconv.Itoa(c.Points))
	}
	out := " score=" + strconv.Itoa(w.Score) + " tier=" + string(w.Tier)
	if w.By != "" {
		out += " by=" + string(w.By)
	}
	return out + " why=" + strings.Join(why, ",")
}

func TestTheCorpusChecksToItsGolden(t *testing.T) {
	got := render(checked(spwnKinds(), corpusSources(t)))
	path := filepath.Join(goldenDir, checkGolden)
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s is missing (go test ./apps/engine/wiki/ -update writes it)", path)
	}
	if got != string(want) {
		t.Errorf("%s differs (go test ./apps/engine/wiki/ -update rewrites it):\n%s", path, got)
	}
}

// value: the one page of the fixture corpus written with a hard finding,
// so the served golden proves such a page is still served.
const unfinished = "blueprint/unfinished"

func TestTheCorpusHasNoHardFindingButOnItsUnfinishedPage(t *testing.T) {
	held := 0
	for _, f := range checked(spwnKinds(), corpusSources(t)).Findings {
		switch {
		case f.Hard && f.Page == unfinished:
			held++
		case f.Hard:
			t.Errorf("%s", line(f))
		}
	}
	if held != 1 {
		t.Errorf("%s holds %d hard findings; want its one H4", unfinished, held)
	}
}

type checkCase struct {
	name string
	// value: the pages the case adds to the corpus, by id; a body is put
	// under `# Case` and a blank line, so its first line is line 3.
	pages map[string]string
	// value: a line of the corpus's wiki.yaml and what replaces it.
	from, to string
	want     []string
}

// value: a case's page: its frontmatter first when it opens on one, then
// `# Case` and a blank line.
func casePage(body string) string {
	if rest, ok := strings.CutPrefix(body, "---\n"); ok {
		front, after, _ := strings.Cut(rest, "---\n")
		return "---\n" + front + "---\n# Case\n\n" + after
	}
	return "# Case\n\n" + body
}

// value: a finding less its place, so a case that moves the lines of
// wiki.yaml still finds the corpus's own findings in the baseline.
func unplaced(f wiki.Finding) string { return f.Page + " " + f.Rule + " " + f.Says }

// value: the case's report, less every finding the corpus alone has.
func (c checkCase) findings(t *testing.T, base []string) []string {
	t.Helper()
	schema := spwnKinds()
	if c.from != "" {
		yaml := read(t, filepath.Join(corpusDir, "wiki.yaml"))
		if !strings.Contains(yaml, c.from) {
			t.Fatalf("%s: wiki.yaml has no %q", c.name, c.from)
		}
		var err error
		if schema, err = wiki.ReadSchema("wiki.yaml", []byte(strings.Replace(yaml, c.from, c.to, 1))); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
	}
	sources := corpusSources(t)
	for id, body := range c.pages {
		sources[id] = casePage(body)
	}
	var got []string
	for _, f := range checked(schema, sources).Findings {
		if !slices.Contains(base, unplaced(f)) {
			got = append(got, line(f))
		}
	}
	return got
}

const (
	caseID   = "blueprint/case"
	termX    = `<Term id="x" means="m." />`
	hardRule = `<Rule id="r" says="s." force="hard" guards={ref("blueprint/case")} proof="spwn:a.spec.yaml" />`
)

func caseOf(body string) map[string]string { return map[string]string{caseID: body} }

// value: a tag eight times, its `dN` id numbered 1 to 8, each after a blank line.
func eight(tag string) string {
	var b strings.Builder
	for i := 1; i <= 8; i++ {
		b.WriteString(strings.Replace(tag, "dN", fmt.Sprintf("d%d", i), 1) + "\n\n")
	}
	return b.String()
}

func checkCases() []checkCase {
	c := "blueprint/case:"
	long := strings.Repeat("a", 121)
	return []checkCase{
		{name: "a kind written wrong", pages: caseOf(`<Rul id="x" />`), want: []string{c + `1:1 S8 ambient page "blueprint/case" is prose only`, c + `3:1 H1 hard unknown kind "Rul" (did you mean Rule?)`}},
		{name: "a required field missing", pages: caseOf(`<Term id="x" />`), want: []string{c + `3:1 H2 hard term "x": required field "means" is missing`}},
		{name: "an enum value that does not exist", pages: caseOf(`<Rule id="x" says="s." force="strict" guards={ref("blueprint/case")} />`), want: []string{c + `3:24 H2 hard rule "x": force: strict does not exist, it is hard, soft or hardened`}},
		{name: "a value of another type", pages: caseOf(`<Term id="x" means={["m"]} />`), want: []string{c + `3:14 H2 hard term "x": means is a list, it takes sentence`}},
		{name: "a sentence past its cap", pages: caseOf(`<Rule id="x" says="` + strings.Repeat("s", 161) + `" force="soft" guards={ref("blueprint/case")} read_at="spwn:a.md" />`), want: []string{c + `3:14 H2 hard rule "x": says is 161 characters, 160 at most`}},
		{name: "a ref to a block of another kind", pages: caseOf(`<Conviction id="x" says="s." opposite="o." carried_in={[ref("product/boundaries#no-model")]} broken_by="b." />`), want: []string{c + `3:57 H2 hard conviction "x": carried_in points to product/boundaries#no-model, a refusal; it takes a part`}},
		{name: "a scenario with no sentences and no spec proof", pages: caseOf(`<Scenario id="x" of={ref("blueprint/case")} proof="spwn:a_test.go" />`), want: []string{c + `3:1 H2 hard scenario "x": required field "given" is missing`, c + `3:1 H2 hard scenario "x": required field "when" is missing`, c + `3:1 H2 hard scenario "x": required field "then" is missing`}},
		{name: "an id twice", pages: caseOf(termX + "\n\n" + termX), want: []string{c + `5:1 H3 hard id "x" is defined twice in blueprint/case`}},
		{name: "an id of no form", pages: caseOf(`<Term id="X_1" means="m." />`), want: []string{c + `3:1 H3 hard id "X_1" is no id: lower-case letters and digits, joined by dashes`}},
		{name: "a ref to no page", pages: caseOf(`<Refusal id="x" refuses="r." because={ref("blueprint/philosofy")} />`), want: []string{c + `3:39 H4 hard "blueprint/philosofy" is no page (did you mean blueprint/philosophy?)`}},
		{name: "a ref to no block", pages: caseOf(`<Refusal id="x" refuses="r." because={ref("product/stack#wikk")} />`), want: []string{c + `3:39 H4 hard "product/stack#wikk" is no block of product/stack (did you mean product/stack#wiki?)`}},
		{name: "a seat opening on no page", from: "home: growth/positioning", to: "home: growth/positionin", want: []string{`wiki.yaml:41:1 H4 hard seat "growth" opens on "growth/positionin", which is no page (did you mean growth/positioning?)`}},
		{name: "an anchor of no repository", pages: caseOf(`<Decision id="x" adr="web:docs/a.md" decides="d." />`), want: []string{c + `3:22 H5 hard anchor web:docs/a.md does not resolve: "web" is no repository of wiki.yaml's repos`}},
		{name: "a hard rule with no proof", pages: caseOf(`<Rule id="x" says="s." force="hard" guards={ref("blueprint/case")} />`), want: []string{c + `3:1 H6 hard rule "x" is hard and names no proof: write proof, or make it soft`}},
		{name: "a term defined on two pages", pages: caseOf(`<Term id="right" means="m." />`), want: []string{`machines/reference/ports:9:1 H8 hard term "right" is defined in blueprint/case and in machines/reference/ports: one owner`}},
		{name: "a retired term written", pages: caseOf(`<Term id="old-word" means="m." retired="2026-01-01" replaced_by={ref("machines/reference/ports#right")} />` + "\n\nWe still say Old word here."), want: []string{c + `5:14 H8 hard "old word" is retired, write "right"`}},
		{name: "a block above its page's floor", pages: map[string]string{"product/case": termX[:len(termX)-2] + `floor={0} />`}, want: []string{`product/case:3:25 H9 hard "x" is on floor 0, above its page's floor 1: a block is on its page's floor or deeper`}},
		{name: "a needs cycle", pages: map[string]string{caseID: "---\nneeds: [blueprint/other]\n---\n" + termX, "blueprint/other": "---\nneeds: blueprint/case\n---\n" + `<Term id="y" means="m." />`}, want: []string{c + `2:1 H10 hard needs: blueprint/case -> blueprint/other -> blueprint/case`}},
		{name: "a line face past its cap", pages: map[string]string{caseID: "---\nline: " + long + "\n---\n" + termX}, want: []string{c + `2:1 H12 hard line face of blueprint/case is 121 characters, 120 at most`}},
		{name: "a component made of a higher level", pages: caseOf(`<Component id="a" level="atom" aims="a." source="spwn:a.go" made_of={[ref("blueprint/case#b")]} />` + "\n\n" + `<Component id="b" level="organism" aims="b." source="spwn:b.go" />`), want: []string{c + `3:71 H14 hard component "a" is a atom made of "b", a organism: a component is made of lower levels only`}},
		{name: "a weight that does not exist", pages: caseOf(`<Term id="x" means="m." weight="high" />`), want: []string{c + `3:25 H15 hard weight: high does not exist: lead or quiet, or nothing and the engine weighs`}},
		{name: "a weight on a figure", pages: caseOf(`<Figure id="f" form="flow" shows={ref("blueprint/case")} caption="c." weight="lead" />`), want: []string{c + `3:71 H15 hard a figure is not weighed: weight is written on a block the engine weighs`}},
		{name: "two written leads in a group", pages: caseOf("<Group id=\"g\" title=\"G\" intent=\"tell\">\n\n" + `<Term id="a" means="m." weight="lead" />` + "\n\n" + `<Term id="b" means="m." weight="lead" />` + "\n\n</Group>"), want: []string{c + `7:25 H16 hard group "g" already has a written lead, "a": one per group; write quiet, or nothing`}},
		{name: "an intent that does not exist", pages: caseOf("<Group id=\"g\" title=\"G\" intent=\"explain\">\n\n" + termX + "\n\n</Group>"), want: []string{c + `3:25 H17 hard intent: explain does not exist: tell, define, decide, locate, guard or prove`}},
		{name: "a table of no kind", pages: caseOf(`<Table id="t" of="alternatives" in={ref("blueprint/case")} columns={["says"]} />`), want: []string{c + `3:15 H19 hard "alternatives" is no kind: a table shows the blocks of one kind`}},
		{name: "a column of no field", pages: caseOf(`<Table id="t" of="alternative" in={ref("growth/positioning")} columns={["price"]} />`), want: []string{c + `3:73 H20 hard column "price" is neither a field of alternative (gives, costs, loses) nor derived (@state, @vital, @proof, @opens, @weight)`}},
		{name: "a table written row by row", pages: caseOf(`<Table id="t" of="part" in={ref("product/stack")} columns={["keeps"]} rows={[]} />`), want: []string{c + `3:71 H22 hard a table is not written row by row: a row is a block of kind part; write the block, the table finds it`}},
		{name: "a sub-page that is no page", pages: caseOf(`<Page id="blueprint/case/nowhere" />`), want: []string{c + `3:1 H23 hard sub-page "blueprint/case/nowhere" is no page`}},
		{name: "a sub-page that is not a child", pages: caseOf(`<Page id="product/stack" />`), want: []string{c + `3:1 H24 hard "product/stack" is not a child of this page: a page that is not yours is linked, not placed`}},
		{name: "a sub-page set twice", pages: map[string]string{caseID: `<Page id="blueprint/case/child" />` + "\n" + `<Page id="blueprint/case/child" />`, "blueprint/case/child": termX}, want: []string{c + `4:1 H25 hard "blueprint/case/child" is placed twice in this page`}},
		{name: "a form that does not exist", pages: caseOf(`<Figure id="f" form="quadrant" shows={ref("blueprint/case")} caption="c." />`), want: []string{c + `3:16 H26 hard form: quadrant does not exist: orbit, gate, flow, stack or quad`}},
		{name: "a caption past its cap", pages: caseOf(`<Figure id="f" form="flow" shows={ref("blueprint/case")} caption="` + long + `" />`), want: []string{c + `3:58 H28 hard the caption is 121 characters, 120 at most`}},
		{name: "an icon that does not exist", pages: caseOf(`<Term id="x" means="m." icon="zebra" />`), want: []string{c + `3:25 H29 hard icon "zebra" is in no icons: of wiki.yaml and is not built in`}},
		{name: "a kind's icon that does not exist", from: "        icon: conviction\n", to: "        icon: convictions\n", want: []string{`wiki.yaml:55:1 H29 hard icon "convictions" is in no icons: of wiki.yaml and is not built in`}},
		{name: "a grid of another mark", from: `        - "..#.."`, to: `        - "..o.."`, want: []string{`wiki.yaml:129:1 H30 hard icon "conviction": row 1 holds "o"; only # and .`}},
		{name: "a grid past twelve", from: `        - "....#"`, to: `        - ".............#"`, want: []string{`wiki.yaml:149:1 H30 hard icon "growth" is 14 by 4, 12 by 12 at most`}},
		{name: "two icons of one grid", from: "        - \"#.#.#\"\n        - \".#.#.\"\n        - \"#.#.#\"\n", to: "        - \"##.##\"\n        - \"#...#\"\n        - \"##.##\"\n", want: []string{`wiki.yaml:145:1 H31 hard icons "product" and "engineering" are the same grid`}},
		{name: "a page under no seat", pages: map[string]string{"design/case": termX}, want: []string{`design/case:1:1 H32 hard design/case.md is under no seat's path`}},
		{name: "a seat with no page", from: "      home: growth/positioning\n", to: "      home: growth/positioning\n    - id: design\n      name: Design\n      path: [design/]\n      reads: [tell]\n      home: growth/positioning\n", want: []string{`wiki.yaml:46:1 H33 hard seat "design" owns no page: a seat opens the day it has content`, `wiki.yaml:46:1 S21 ambient seat "design" has no icon: the dot stands in`}},
		{name: "a fold that is no bool", pages: caseOf("<Group id=\"g\" title=\"G\" intent=\"tell\" fold=\"maybe\">\n\n" + termX + "\n\n</Group>"), want: []string{c + `3:39 H35 hard fold: maybe is true or false`}},
		{name: "a state that does not exist", pages: caseOf(`<Table id="t" of="conviction" in={ref("blueprint/philosophy")} where={query({ carried_in: { has: state("sent") } })} columns={["says"]} />`), want: []string{c + `3:64 H36 hard "sent" is no state of part: shipped or making`}},
		{name: "a condition on no field", pages: caseOf(`<Table id="t" of="part" in={ref("product/stack")} where={query({ shade: { is: "red" } })} columns={["keeps"]} />`), want: []string{c + `3:51 H36 hard where: "shade" is no field of part (space, lets_go, keeps, state)`}},
		{name: "a tag never closed", pages: caseOf(`<Term id="a" means="b">`), want: []string{c + `1:1 S8 ambient page "blueprint/case" is prose only`, c + `3:1 H38 hard <Term at line 3 ends with >: a block ends with />, only a Group holds what follows it`}},
		{name: "an expression in braces", pages: caseOf(`<Term id="a" means={1 + 2} />`), want: []string{c + `1:1 S8 ambient page "blueprint/case" is prose only`, c + `3:20 H39 hard {1 + 2} is an expression: in braces, a value, ref(...) or query(...), nothing else`}},
		{name: "a group of two floors", pages: map[string]string{"product/case": "<Group id=\"g\" title=\"G\" intent=\"tell\">\n\n" + termX + "\n\n" + `<Term id="y" means="m." floor={3} />` + "\n\n</Group>"}, want: []string{`product/case:3:1 H40 hard <Group "g"> holds blocks of floors 1 and 3: a group lives on one floor`}},
		{name: "an attribute its kind lacks", pages: caseOf(`<Term id="a" means="m." price="1" />`), want: []string{c + `3:25 H41 hard "price" is not a field of term`}},
		{name: "a frontmatter key that does not exist", pages: map[string]string{caseID: "---\nshade: red\n---\n" + termX}, want: []string{c + `2:1 H41 hard "shade" is no key of a page's frontmatter: kind, floor, needs, line, word, icon, rests_on or retired`}},
		{name: "a soft rule broken twice", pages: caseOf(`<Rule id="x" says="s." force="soft" guards={ref("blueprint/case")} read_at="spwn:a.md" precedents={[{ at: "2026-01-01" }, { at: "2026-02-01" }]} />`), want: []string{c + `3:88 S2 remark rule "x" is soft and broke 2 times: it asks for a proof`}},
		{name: "a refused link with no proof", pages: caseOf(`<Link id="x" from={ref("blueprint/case")} to="the cloud" right="refused" because="r." />`), want: []string{c + `3:1 S3 remark link "x" is refused and names no proof: nothing would catch it`}},
		{name: "a scenario with no proof", pages: caseOf(`<Scenario id="x" of={ref("blueprint/case")} given="g." when="w." then="t." />`), want: []string{c + `3:1 S4 ambient scenario "x" has no proof: shown as pending`}},
		{name: "a block resting on a reading", pages: caseOf(`<Term id="x" means="m." rests_on="reading" />`), want: []string{c + `3:25 S6 needs-you "x" rests on this page's reading: you have not ruled`}},
		{name: "a page past 1 MiB", pages: caseOf(strings.Repeat("a", 1<<20)), want: []string{c + `1:1 H44 hard blueprint/case.md is 1048584 bytes, past the 1048576 a page is compiled from: split it`, c + `1:1 S8 ambient page "blueprint/case" is prose only`}},
		{name: "a page of prose only", pages: caseOf("Only prose."), want: []string{c + `1:1 S8 ambient page "blueprint/case" is prose only`}},
		{name: "a page past its budget, none leading", pages: caseOf(eight(`<Decision id="dN" adr="spwn:a.md" decides="d." />`)), want: []string{c + `1:1 S11 ambient page "blueprint/case" does not fit 2min: 8 lines`, c + `1:1 S13 ambient page "blueprint/case": 8 blocks, none leads`}},
		{name: "a group of eight blocks", pages: caseOf("<Group id=\"g\" title=\"G\" intent=\"tell\">\n\n" + eight(`<Term id="dN" means="m." />`) + "</Group>"), want: []string{c + `1:1 S13 ambient page "blueprint/case": 8 blocks, none leads`, c + `3:1 S15 remark group "g" holds 8 blocks: split it`}},
		{name: "a page of eight blocks that shows no child", pages: map[string]string{caseID: eight(`<Term id="dN" means="m." />`), "blueprint/case/child": termX}, want: []string{c + `1:1 S13 ambient page "blueprint/case": 8 blocks, none leads`, c + `1:1 S18 ambient page "blueprint/case" shows none of its 1 children`}},
		{name: "a written lead far below its group", pages: caseOf(hardRule + "\n\n" + `<Term id="x" means="m." weight="lead" />`), want: []string{c + `5:25 S12 remark "x" is written lead, its score is 0 and its section's top is 3`}},
		{name: "an empty group", pages: caseOf(`<Group id="g" title="G" intent="tell" />` + "\n\n" + termX), want: []string{c + `3:1 S14 ambient group "g" is empty`}},
		{name: "a table of an empty scope", pages: caseOf(termX + "\n\n" + `<Table id="t" of="alternative" in={ref("blueprint/case")} columns={["gives"]} />`), want: []string{c + `5:1 S16 ambient no alternative under blueprint/case: the table is empty, not wrong`}},
		{name: "a table of seven columns", pages: caseOf(`<Table id="t" of="part" in={ref("product/stack")} columns={["space", "lets_go", "keeps", "state", "@state", "@weight", "@proof"]} />`), want: []string{c + `3:1 S17 remark table "t" has 7 columns`}},
		{name: "an intent no seat reads", from: "reads: [define, decide, locate, guard, prove]", to: "reads: [define, decide, locate, guard]", want: []string{`wiki.yaml:25:1 S22 remark intent "prove" is read by no seat`}},
		{name: "a tag past twelve lines", pages: caseOf("<Conviction\nid=\"x\"\nsays=\"s.\"\nopposite=\"o.\"\ncarried_in={[\nref(\"product/stack#machines\"),\nref(\"product/stack#agents\"),\nref(\"product/stack#wiki\"),\nref(\"product/stack#specs\"),\nref(\"product/stack#design\"),\nref(\"product/stack#os\")\n]}\nbroken_by=\"b.\"\n/>"), want: []string{c + `3:1 S26 remark <Conviction "x"> runs 14 lines: put attributes two to a line`}},
		{name: "a markdown table of a kind's fields", pages: caseOf(termX + "\n\n| Gives | Costs |\n| --- | --- |\n| a | b |"), want: []string{c + `5:1 S28 remark this table's columns are the fields of alternative: write blocks and a <Table>`}},
		{name: "a rule asking a waiver", pages: caseOf(`<Rule id="x" says="s." force="soft" guards={ref("blueprint/case")} read_at="spwn:a.md" waiver="the owner may waive it." />`), want: []string{c + `3:88 S29 needs-you rule "x" asks to be waived: the owner may waive it.`}},
	}
}

func TestEachRuleOfTheCheckSaysWhere(t *testing.T) {
	report := checked(spwnKinds(), corpusSources(t))
	base := make([]string, 0, len(report.Findings))
	for _, f := range report.Findings {
		base = append(base, unplaced(f))
	}
	for _, c := range checkCases() {
		if got := c.findings(t, base); !slices.Equal(got, c.want) {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
}

func TestEveryRuleOfTheRegistryHasACase(t *testing.T) {
	named := map[string]bool{}
	for _, c := range checkCases() {
		for _, w := range c.want {
			named[strings.Fields(w)[1]] = true
		}
	}
	for _, code := range wiki.Rules() {
		if !named[code] {
			t.Errorf("rule %s has no case in checkCases", code)
		}
	}
}

func TestEveryRuleTheEngineNamesIsInTheRegistry(t *testing.T) {
	registered := map[string]bool{}
	for _, code := range wiki.Rules() {
		registered[code] = true
	}
	sources, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	named := regexp.MustCompile(`"([HS]\d+)"`)
	for _, entry := range sources {
		file := entry.Name()
		if !strings.HasSuffix(file, ".go") || strings.HasSuffix(file, "_test.go") {
			continue
		}
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range named.FindAllStringSubmatch(string(body), -1) {
			if !registered[m[1]] {
				t.Errorf("%s names rule %s, which the registry does not hold: its finding would panic", file, m[1])
			}
		}
	}
}

func FuzzCheck(f *testing.F) {
	for _, c := range checkCases() {
		for _, body := range c.pages {
			f.Add(casePage(body))
		}
	}
	f.Fuzz(func(t *testing.T, source string) {
		sources := map[string]string{"blueprint/fuzz": source, "product/stack": read(t, filepath.Join(corpusDir, "product", "stack.md"))}
		first := render(checked(spwnKinds(), sources))
		if again := render(checked(spwnKinds(), sources)); first != again {
			t.Fatalf("two checks of one corpus differ")
		}
	})
}
