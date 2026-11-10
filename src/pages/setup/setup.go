package setup

import (
	"strconv"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/pages"
	"github.com/jterrazz/studio-station/src/setup"
)

func Outcome(report setup.Report, thisMac page.Context) page.Page {
	// invariant: init ends on what it made known, then on the empty state of
	// what the machine carries: no Machinefile applies yet, so apps and
	// settings come one at a time, and the next steps say how.
	rows := make([]page.Row, 0, len(report.Steps))
	for _, step := range report.Steps {
		r := pages.Done(step.Name, step.State, step.Detail)
		r.Cells = []page.Value{page.Text(step.Detail)}
		r.Detail = page.Text(step.Log)
		rows = append(rows, r)
	}
	if len(report.Steps) == 0 {
		return pages.Outcome("init", thisMac, Columns(), rows, "steps")
	}
	p := pages.Outcome("init", thisMac, Columns(), rows, "steps", machinefile())
	p.Sections = append([]page.Section{summary(report)}, p.Sections...)
	p.Hints = []page.Hint{
		{Command: "station apps install <app>", Text: "one app, and what it requires"},
		{Command: "station settings apply <setting>", Text: "one setting"},
	}
	return page.Seal(p)
}

// value: what this machine carries: none until a Machinefile applies.
func machinefile() page.Section {
	return page.Section{ID: "machinefile", Title: "Machinefile", Fold: page.Open, Uncounted: true, Items: []page.Item{{Note: &page.Note{Text: "none yet: apps and settings come one at a time"}}}}
}

func summary(report setup.Report) page.Section {
	changed, failed, open := 0, 0, 0
	for _, step := range report.Steps {
		switch {
		case step.State == catalog.StateBroken:
			failed++
		case step.State != catalog.StateOK:
			open++
		case step.Changed:
			changed++
		}
	}
	label, mark, words := "Known to station", page.OK, "nothing to change"
	switch {
	case failed > 0:
		label, mark, words = "Not known to station yet", page.Broken, steps(failed)+" failed"
	case open > 0:
		label, mark, words = "Known to station, with a step left", page.Attention, steps(open)+" left"
	case changed > 0:
		words = steps(changed) + " changed"
	}
	row := page.Row{ID: "summary/init", Label: label, Mark: mark, Cells: []page.Value{page.Text(words)}}
	return page.Section{ID: "summary", Title: "Summary", Role: page.Summary, Fold: page.Open, Uncounted: true, Columns: []page.Column{{Key: "area", Kind: page.KindText, Keep: 100}, {Key: "value", Kind: page.KindList, Keep: 60}}, Items: []page.Item{{Row: &row}}}
}

func steps(n int) string {
	if n == 1 {
		return "1 step"
	}
	return strconv.Itoa(n) + " steps"
}

func Columns() []page.Column {
	return []page.Column{{Key: "step", Kind: page.KindText, Keep: 100}, {Key: "detail", Kind: page.KindText, Keep: 50}}
}

func NameForm(suggestion string) page.Form {
	return page.Form{Fields: []page.Field{{
		Key: "name", Label: "Name this machine", Kind: page.FieldText, Default: suggestion,
		Note: "lowercase letters, digits and -, as in macbook or mac-mini",
	}}}
}
