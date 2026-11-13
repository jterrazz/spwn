package apps

import (
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/apps"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/pages"
)

type Run struct {
	// value: the machine the verb ran on ("" for this one) and how long it took.
	On   string
	Took time.Duration
}

func Outcomes(verb string, outcomes []apps.Outcome, run Run, thisMac page.Context) page.Page {
	// invariant: an outcome opens with what changed, counted, on which
	// machine and in how long (`Installed 2 apps on This Mac · 9s`), then
	// each app with the version and the method that now hold it and why it
	// came (`required by gh`), then what to do next.
	steps := make([]pages.RunStep, 0, len(outcomes))
	changed, failed := 0, 0
	var landed []string
	for _, o := range outcomes {
		steps = append(steps, pages.RunStep{ID: o.Name, State: o.Result, Cells: []page.Value{page.Version(o.Version), page.Texts(string(o.Method))}, Why: o.Reasons, Detail: o.Detail, Log: o.Log})
		switch {
		case o.Result == catalog.StateBroken:
			failed++
		case o.Result == catalog.StateOK && o.Detail != apps.AlreadyInstalled:
			changed++
		}
		if len(o.Path) > 0 {
			landed = append(landed, o.Name)
		}
		for _, s := range o.Steps {
			steps = append(steps, pages.RunStep{ID: o.Name + "/" + s.Package, Label: s.Package, State: s.Result, Detail: s.Detail})
		}
	}
	r := pages.Run{Command: "apps " + verb, Columns: Columns(), Steps: steps, Empty: "apps to " + verb}
	if len(landed) > 0 && run.On == "" {
		r.Next = []page.Hint{{Command: "exec zsh", Text: "so this shell finds what " + strings.Join(landed, ", ") + " installed", External: true}}
	}
	var words []string
	switch {
	case changed > 0:
		r.Head = &pages.RunHead{Label: done(verb) + " " + plural(changed, "app") + where(verb, run.On, thisMac), Mark: page.OK}
		if failed > 0 {
			r.Head.Mark = page.Broken
			words = append(words, strconv.Itoa(failed)+" failed")
		}
	case failed > 0:
		r.Head = &pages.RunHead{Label: plural(failed, "app") + " failed to " + verb + where(verb, run.On, thisMac), Mark: page.Broken}
	default:
		return pages.RunPage(r, thisMac)
	}
	if took := run.Took.Round(time.Second); took > 0 {
		words = append(words, pages.Term(took))
	}
	r.Head.Words = words
	return pages.RunPage(r, thisMac)
}

func done(verb string) string {
	switch verb {
	case "install":
		return "Installed"
	case "update":
		return "Updated"
	case "remove":
		return "Removed"
	case "move":
		return "Moved"
	}
	return verb
}

func where(verb, on string, thisMac page.Context) string {
	label := on
	if label == "" {
		label = machineLabel(thisMac)
	}
	if verb == "remove" {
		return " from " + label
	}
	return " on " + label
}

func machineLabel(thisMac page.Context) string {
	if thisMac.Kind == string(machine.Computer) || thisMac.Machine == "" {
		return "This Mac"
	}
	return thisMac.Machine
}

func Updated(outcomes []apps.Outcome, run Run, thisMac page.Context) page.Page {
	// invariant: an update with nothing to update says every app is up to
	// date, a calm line, never an empty list.
	if len(outcomes) > 0 {
		return Outcomes("update", outcomes, run, thisMac)
	}
	label := machineLabel(thisMac)
	if run.On != "" {
		label = run.On
	}
	row := page.Row{ID: "up-to-date", Label: "every app is up to date on " + label, Mark: page.OK}
	return pages.Outcome("apps update", thisMac, nil, []page.Row{row}, "apps to update")
}

type MachineUpdate struct {
	Machine string
	// value: what the machine answered; Skipped and Err say why it did not.
	Outcomes []apps.Outcome
	Skipped  string
	Err      string
}

func EveryMachine(updates []MachineUpdate, thisMac page.Context) page.Page {
	// invariant: --all-machines tells one line per machine: what it updated,
	// why it was skipped (a server, a stopped VM), or that it did not answer.
	rows := make([]page.Row, 0, len(updates))
	for _, u := range updates {
		row := page.Row{ID: u.Machine, Label: u.Machine, Mark: page.OK}
		switch {
		case u.Err != "":
			row.Mark, row.Reason, row.Detail = page.Broken, page.Failing, page.Text(u.Err)
		case u.Skipped != "":
			row.Mark, row.Detail = page.NotForThisMachine, page.Text("skipped ("+u.Skipped+")")
		default:
			failed, _ := apps.FailedOf(u.Outcomes)
			words := []string{"every app is up to date"}
			if n := len(u.Outcomes) - failed; n > 0 {
				words[0] = "updated " + plural(n, "app")
			}
			if failed > 0 {
				row.Mark, row.Reason = page.Broken, page.Failing
				words = append(words, strconv.Itoa(failed)+" failed")
			}
			row.Detail = page.Texts(words...)
		}
		rows = append(rows, row)
	}
	return pages.Outcome("apps update", thisMac, nil, rows, "machines to update")
}
