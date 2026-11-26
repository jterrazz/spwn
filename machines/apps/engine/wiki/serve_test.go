package wiki_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/wiki"
)

const serveGolden = "serve-corpus.txt"

func corpusFiles(t *testing.T) map[string][]byte {
	t.Helper()
	out := map[string][]byte{"wiki.yaml": []byte(read(t, filepath.Join(corpusDir, "wiki.yaml")))}
	for id, source := range corpusSources(t) {
		out[id+".md"] = []byte(source)
	}
	return out
}

func outline(s wiki.Served) string {
	var b strings.Builder
	line := func(depth int, parts ...string) {
		b.WriteString(strings.Repeat("  ", depth) + strings.Join(parts, " ") + "\n")
	}
	seats := make([]string, 0, len(s.Seats))
	for _, seat := range s.Seats {
		mark := seat.ID
		if seat.Active {
			mark = "[" + mark + "]"
		}
		seats = append(seats, mark)
	}
	line(0, "page", s.Page.ID, "@"+string(s.Budget), "floor="+strconv.Itoa(s.Floor.Index), s.Floor.Name, "seats="+strings.Join(seats, ","), "needs-you="+strconv.Itoa(s.NeedsYou), "unshown="+strconv.Itoa(s.Unshown))
	ladder := make([]string, 0, len(s.Floors))
	for _, f := range s.Floors {
		mark := strconv.Itoa(f.Index) + ":" + strconv.Itoa(f.Blocks)
		if f.Current {
			mark = "[" + mark + "]"
		}
		ladder = append(ladder, mark)
	}
	line(1, "floors", strings.Join(ladder, " "))
	for _, p := range s.Paragraph {
		line(1, "paragraph", runs(p))
	}
	for _, f := range s.Findings {
		line(1, "finding", string(f.Level), f.Rule, f.Says)
	}
	for _, f := range s.Attention {
		line(1, "attention", string(f.Level), f.Rule, "at="+f.At, strconv.Quote(f.Word))
	}
	var items func(depth int, its []wiki.ServedItem)
	items = func(depth int, its []wiki.ServedItem) {
		for _, it := range its {
			switch {
			case it.Prose != nil:
				for _, p := range it.Prose {
					line(depth, string(p.Form), runs(p))
				}
			case it.Group != nil:
				g := it.Group
				head := []string{"group", g.ID, string(g.Intent), "blocks=" + strconv.Itoa(g.Blocks), "folded=" + string(g.Folded)}
				if g.OwnFloor != nil {
					head = append(head, "own-floor="+strconv.Itoa(g.OwnFloor.Index))
				}
				if g.NeedsYou {
					head = append(head, "needs-you")
				}
				line(depth, append(head, "lead="+strconv.Quote(g.Lead))...)
				items(depth+1, g.Items)
			case it.Block != nil:
				k := it.Block
				head := []string{"block", k.Address, k.Kind, "icon=" + k.Icon, "tier=" + string(k.Tier), "shown=" + string(k.Shown)}
				if k.OwnFloor != nil {
					head = append(head, "own-floor="+strconv.Itoa(k.OwnFloor.Index))
				}
				if k.NeedsYou {
					head = append(head, "needs-you")
				}
				line(depth, append(head, strconv.Quote(k.Line))...)
				for _, f := range k.Fields {
					line(depth+1, "field", f.Name, strconv.Quote(f.Text), refs(f.Refs))
				}
				if k.Table != nil {
					line(depth+1, "table", k.Table.Of, "in="+k.Table.In.Address, "where="+strconv.Quote(k.Table.Where), "columns="+strings.Join(k.Table.Columns, ","))
					for _, r := range k.Table.Rows {
						cells := make([]string, 0, len(r.Cells))
						for _, c := range r.Cells {
							cells = append(cells, c.Name+"="+strconv.Quote(c.Text)+refs(c.Refs))
						}
						line(depth+2, "row", r.Address, strings.Join(cells, " "))
					}
				}
				if k.Sub != nil {
					line(depth+1, "sub", k.Sub.ID, strconv.Quote(k.Sub.Line))
				}
				if k.Figure != nil {
					line(depth+1, "figure", k.Figure.Form, "shows="+k.Figure.Shows.Address, strconv.Quote(k.Figure.Caption))
					for _, p := range k.Figure.Parts {
						line(depth+2, "part", strconv.Quote(p.Name), "at="+p.At, "n="+strconv.Itoa(p.N))
					}
				}
				if d := k.Decision; d != nil {
					line(depth+1, "decision", "page="+d.Page, d.Number, strconv.Quote(d.Title), "status="+strconv.Quote(d.Status), "accepted="+strconv.FormatBool(d.Accepted), "date="+d.Date, "amended-by="+strings.Join(d.AmendedBy, ","), "amends="+strings.Join(d.Amends, ","), "resolves="+strconv.FormatBool(d.Resolves), strconv.Quote(d.Decides))
				}
				for _, f := range k.Finding {
					line(depth+1, "finding", string(f.Level), f.Rule, f.Says)
				}
			}
		}
	}
	for _, sec := range s.Sections {
		if sec.Heading != "" {
			line(1, "section", sec.Anchor, strconv.Quote(sec.Heading), "rests="+strconv.FormatBool(sec.Rests))
		}
		items(2, sec.Items)
	}
	return b.String()
}

func runs(p wiki.Paragraph) string {
	if p.Cells != nil {
		cells := make([]string, 0, len(p.Cells))
		for _, c := range p.Cells {
			cells = append(cells, runs(wiki.Paragraph{Runs: c}))
		}
		return strings.Join(cells, " | ")
	}
	var out []string
	for _, r := range p.Runs {
		switch {
		case r.Strong:
			out = append(out, "**"+r.Text+"**")
		case r.Code:
			out = append(out, "`"+r.Text+"`")
		case r.Link != "":
			out = append(out, "["+r.Text+"]("+r.Link+")")
		default:
			out = append(out, r.Text)
		}
	}
	return strconv.Quote(strings.Join(out, ""))
}

func refs(rs []wiki.ServedRef) string {
	out := make([]string, 0, len(rs))
	for _, r := range rs {
		mark := r.Address
		if !r.Resolves {
			mark += "?"
		}
		out = append(out, mark)
	}
	if len(out) == 0 {
		return ""
	}
	return " refs=" + strings.Join(out, ",")
}

func TestEveryPageIsServedToItsGoldenAtEachBudget(t *testing.T) {
	c, err := wiki.Read(corpusFiles(t))
	if err != nil {
		t.Fatal(err)
	}
	var got strings.Builder
	for _, id := range corpusPages() {
		for _, budget := range wiki.Budgets() {
			s, served := c.Serve(id, budget)
			if served != nil {
				t.Fatalf("%s: %v", id, served)
			}
			got.WriteString(outline(s) + "\n")
		}
	}
	path := filepath.Join(goldenDir, serveGolden)
	if *update {
		if wrote := os.WriteFile(path, []byte(got.String()), 0o600); wrote != nil {
			t.Fatal(wrote)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s is missing (go test ./apps/engine/wiki/ -update writes it)", path)
	}
	if got.String() != string(want) {
		t.Errorf("%s differs (go test ./apps/engine/wiki/ -update rewrites it):\n%s", path, got.String())
	}
}

func TestACorpusIsOpenedAtItsFirstSeatsHomeAndItsOwnBudget(t *testing.T) {
	c, err := wiki.Read(corpusFiles(t))
	if err != nil {
		t.Fatal(err)
	}
	s, err := c.Serve(c.Home(), "")
	if err != nil || s.Page.ID != "blueprint/philosophy" || s.Budget != wiki.Budget2min || len(s.Pages) != len(corpusPages()) {
		t.Fatalf("Serve(home) = %s @%s, %d pages, %v", s.Page.ID, s.Budget, len(s.Pages), err)
	}
}

func TestAPageNoFileHoldsIsUnknown(t *testing.T) {
	c, err := wiki.Read(corpusFiles(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Serve("product/nowhere", ""); !errors.Is(err, wiki.ErrUnknownPage) {
		t.Fatalf("Serve = %v; want ErrUnknownPage", err)
	}
}

func TestACorpusWithNoSchemaIsNotRead(t *testing.T) {
	if _, err := wiki.Read(map[string][]byte{"a.md": []byte("# A\n")}); !errors.Is(err, wiki.ErrNoSchema) {
		t.Fatalf("Read = %v; want ErrNoSchema", err)
	}
}

func TestProseIsReadAsParagraphsItemsAndRuns(t *testing.T) {
	got := wiki.Paragraphs("One **strong** line\nthat wraps, with `code`.\n\n- an [item](./a.md)\n  that wraps\n1. a numbered one\n\n> a quote\n\n| a | b |\n|---|---|\n| 1 | 2 |\n")
	lines := make([]string, 0, len(got))
	for _, p := range got {
		lines = append(lines, string(p.Form)+" "+runs(p))
	}
	want := []string{
		`plain "One **strong** line that wraps, with ` + "`code`" + `."`,
		`item "an [item](./a.md) that wraps"`,
		`item "a numbered one"`,
		`quote "a quote"`,
		`head "a" | "b"`,
		`row "1" | "2"`,
	}
	if !slices.Equal(lines, want) {
		t.Fatalf("Paragraphs =\n%s\nwant\n%s", strings.Join(lines, "\n"), strings.Join(want, "\n"))
	}
}

func TestAFigureWithNoPartsWrittenIsDrawnFromTheGroupItShowsByItsTitle(t *testing.T) {
	files := corpusFiles(t)
	files["growth/positioning.md"] = []byte(strings.Replace(string(files["growth/positioning.md"]),
		`parts={[{ name: "by hand", at: "y" }, { name: "one big agent", at: "x" }, { name: "spwn", at: "xy" }]}`, "", 1))
	c, err := wiki.Read(files)
	if err != nil {
		t.Fatal(err)
	}
	s, err := c.Serve("growth/positioning", wiki.Budget2min)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, sec := range s.Sections {
		for _, it := range sec.Items {
			if it.Block != nil && it.Block.Figure != nil {
				if shows := it.Block.Figure.Shows; !shows.Resolves || shows.Word != "What a person uses today" {
					t.Errorf("shows = %+v; want the group, by its title", shows)
				}
				for _, p := range it.Block.Figure.Parts {
					names = append(names, p.Name+"@"+p.At)
				}
			}
		}
	}
	want := []string{"by-hand@", "one-big-agent@", "managed-platform@", "unrelated-tools@"}
	if !slices.Equal(names, want) {
		t.Fatalf("parts = %v; want %v", names, want)
	}
}
