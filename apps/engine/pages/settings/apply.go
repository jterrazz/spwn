package settings

import (
	"strconv"

	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/pages"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

type MachineApply struct {
	Machine string
	// value: what the machine answered; Skipped, Admin and Err say why it
	// did not: a server, a stopped workspace, a setting it needs admin
	// rights for (with the command that applies it), a Mac that does not
	// answer; Elsewhere, each setting named that does not live there.
	Changes   []settings.Change
	Skipped   string
	Admin     []AdminSetting
	Elsewhere []Elsewhere
	Err       string
}

type AdminSetting struct{ Name, Command string }

type Elsewhere struct{ Name, Why string }

func EveryMachine(applies []MachineApply, thisMac page.Context) page.Page {
	// invariant: --all-machines tells each machine in turn: each setting
	// applied there, each admin setting with the command that applies it, each
	// setting that does not live there,
	// why the machine was skipped, or that it did not answer.
	var rows []page.Row
	for _, a := range applies {
		switch {
		case a.Err != "":
			rows = append(rows, page.Row{ID: a.Machine, Label: a.Machine, Mark: page.Broken, Reason: page.Failing, Detail: page.Text(a.Err)})
			continue
		case a.Skipped != "":
			rows = append(rows, page.Row{ID: a.Machine, Label: a.Machine, Mark: page.NotForThisMachine, Detail: page.Text("skipped (" + a.Skipped + ")")})
			continue
		}
		for _, c := range a.Changes {
			row := pages.Done(a.Machine+"/"+c.Item, c.Result, "")
			row.Label, row.Detail = a.Machine, page.Texts(c.Item, summary(c))
			rows = append(rows, row)
		}
		for _, e := range a.Elsewhere {
			rows = append(rows, page.Row{ID: a.Machine + "/" + e.Name, Label: a.Machine, Mark: page.NotForThisMachine, Detail: page.Text("skipped (" + e.Why + ")")})
		}
		for _, s := range a.Admin {
			rows = append(rows, page.Row{ID: a.Machine + "/" + s.Name, Label: a.Machine, Mark: page.Absent, Detail: page.Texts(s.Name, "needs admin rights"), Hint: page.Cmd(s.Command)})
		}
	}
	return pages.Outcome("settings apply", thisMac, nil, rows, "settings to apply")
}

func summary(c settings.Change) string {
	if len(c.Steps) == 1 {
		return step(c.Steps[0])
	}
	return strconv.Itoa(len(c.Steps)) + " steps"
}
