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
	r := pages.Run{Command: "init", Columns: Columns(), Empty: "steps"}
	for _, step := range report.Steps {
		r.Steps = append(r.Steps, pages.RunStep{ID: step.Name, State: step.State, Cells: []page.Value{page.Text(step.Detail)}, Detail: step.Log})
	}
	if len(report.Steps) == 0 {
		return pages.RunPage(r, thisMac)
	}
	r.Head, r.Extra = head(report), []page.Section{machinefile()}
	r.Next = []page.Hint{
		{Command: "station apps install <app>", Text: "one app, and what it requires"},
		{Command: "station settings apply <setting>", Text: "one setting"},
	}
	// invariant: a shell opened before its block was written reads it only
	// once restarted, so that comes first, in place of the setting.
	for _, step := range report.Steps {
		if step.Name == "zshrc" && step.Changed {
			r.Next = []page.Hint{{Command: "exec zsh", Text: "so this shell reads the station block", External: true}, r.Next[0]}
		}
	}
	return pages.RunPage(r, thisMac)
}

// value: what this machine carries: none until a Machinefile applies.
func machinefile() page.Section {
	return page.Section{ID: "machinefile", Title: "Machinefile", Fold: page.Open, Uncounted: true, Items: []page.Item{{Note: &page.Note{Text: "none yet: apps and settings come one at a time"}}}}
}

func head(report setup.Report) *pages.RunHead {
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
	h := &pages.RunHead{Label: "Known to station", Mark: page.OK, Words: []string{"nothing to change"}}
	switch {
	case failed > 0:
		h.Label, h.Mark, h.Words = "Not known to station yet", page.Broken, []string{steps(failed) + " failed"}
	case open > 0:
		h.Label, h.Mark, h.Words = "Known to station, with a step left", page.Attention, []string{steps(open) + " left"}
	case changed > 0:
		h.Words = []string{steps(changed) + " changed"}
	}
	return h
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
