package pages

import (
	"strings"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/page"
)

type Run struct {
	// value: the page's command, `apps install`, `init`, `apply`.
	Command string
	// value: the line that sums the run up, first on a terminal; nil for a
	// run that changed nothing and failed nothing.
	Head    *RunHead
	Columns []page.Column
	Steps   []RunStep
	// value: the noun of the empty state, a run with no step.
	Empty string
	Extra []page.Section
	Next  []page.Hint
}

type RunHead struct {
	Label string
	Mark  page.Mark
	Words []string
}

type RunStep struct {
	ID, Label string
	State     catalog.State
	Cells     []page.Value
	// value: why the step ran (`required by gh`), said before what befell it.
	Why    []string
	Detail string
	// value: the log that keeps a failed step's program words: the row then
	// says only that path, whole, on a line of its own where it cannot fit.
	Log string
}

func RunPage(r Run, thisMac page.Context) page.Page {
	// invariant: every run reads alike, an install, init and apply: the line
	// that sums it up, each step with its mark and why it ran, then what to
	// do next; a step's own words stay muted, after its cells.
	rows := make([]page.Row, 0, len(r.Steps))
	for _, s := range r.Steps {
		row := Done(s.ID, s.State, s.Detail)
		if parts := append(append([]string{}, s.Why...), nonEmpty(s.Detail)...); len(parts) > 1 {
			row.Detail = page.Texts(parts...)
		} else if len(parts) == 1 {
			row.Detail = page.Text(parts[0])
		}
		if s.Log != "" {
			row.Detail = page.Path(s.Log)
			if len(s.Why) > 0 {
				row.Detail = page.List(append(page.Texts(s.Why...).Items, page.Path(s.Log))...)
			}
		}
		if s.Label != "" {
			row.Label = s.Label
		}
		row.Cells = s.Cells
		rows = append(rows, row)
	}
	p := Outcome(r.Command, thisMac, r.Columns, rows, r.Empty, r.Extra...)
	p.Hints = r.Next
	if r.Head == nil {
		return page.Seal(p)
	}
	words := strings.Fields(r.Command)
	head := page.Row{ID: "summary/" + words[len(words)-1], Label: r.Head.Label, Mark: r.Head.Mark}
	if len(r.Head.Words) > 0 {
		head.Cells = []page.Value{page.Texts(r.Head.Words...)}
	}
	summary := page.Section{ID: "summary", Title: "Summary", Role: page.Summary, Fold: page.Open, Uncounted: true, Columns: []page.Column{{Key: "area", Kind: page.KindText, Keep: 100}, {Key: "value", Kind: page.KindList, Keep: 60}}, Items: []page.Item{{Row: &head}}}
	p.Sections = append([]page.Section{summary}, p.Sections...)
	return page.Seal(p)
}

func nonEmpty(s string) []string {
	if s == "" {
		return nil
	}
	return []string{s}
}
