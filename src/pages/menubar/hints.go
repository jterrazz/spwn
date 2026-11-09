package menubar

import (
	"strconv"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/ports"
	"github.com/jterrazz/studio-station/src/settings"
)

const (
	HintAddMac        = "add-mac"
	HintDeclarePublic = "declare-public"

	ActionDeclareHint = "declare-public:"

	VerbDone = "Done"
)

// value: one command of a hint sheet and the line that leads it.
type hintLine struct{ Lead, Command string }

// invariant: SPEC A5 Hints: a hint is a sheet that asks nothing: its
// title, two lines of why, each command in its box to copy, and Done;
// every command is the page's own, a leaf station has today or a line
// the owner types.
func hintSheet(a page.Action, kind, title, why string, lines ...hintLine) page.Action {
	form := &page.Form{Title: title, Subtitle: why, Primary: []page.Primary{{Label: VerbDone}}}
	for i, l := range lines {
		form.Fields = append(form.Fields, page.Field{Key: "command-" + strconv.Itoa(i+1), Kind: page.FieldCommand, Note: l.Lead, Hint: page.Cmd(l.Command)})
	}
	a.Sheet, a.Form = page.SheetHint+kind, form
	return a
}

// value: Add a Mac: station installed on the new Mac, then added from
// this one by its name.
func addMacHint(a page.Action) page.Action {
	return hintSheet(a, HintAddMac, "Add a Mac", "Install station on the new Mac, then add it from This Mac by its Tailscale name. It appears here once its ssh and its station answer.",
		hintLine{"On the new Mac", AddMacCommands()[0]}, hintLine{"Then on This Mac", AddMacCommands()[1]})
}

func AddMacCommands() []string {
	return []string{catalog.InstallLine, "station machines add <mac>"}
}

// value: a checked setting's How to: the line that turns it on when the
// catalogue has one, else where System Settings does it.
func settingHint(a page.Action, subject string) page.Action {
	it, ok := settings.CheckedNamed(subject)
	if !ok {
		a.Sheet = page.SheetHint + subject
		return a
	}
	title := it.Turn
	if title == "" {
		title = "Turn " + it.Label + " on"
	}
	why := "station never changes a security setting for you. Open " + it.Fix + "."
	if it.Command == "" {
		return hintSheet(a, subject, title, why)
	}
	why = "station never changes a security setting for you. Run this in Terminal, or open " + it.Fix + "."
	return hintSheet(a, subject, title, why, hintLine{Command: it.Command})
}

// value: an exposed port's Declare public…: what declaring means, and
// the leaf that declares it, the route the owner's to name.
func declareHint(p ports.Port, mac string, self string) page.Action {
	a := page.Action{ID: ActionDeclareHint + p.Key(), Label: "Declare public", On: page.OnRow, Rows: []string{p.Key()}, Priority: 2}
	why := "A public port is one you meant to open. Declaring it moves it from Exposed to Public; station then watches that it stays as declared. The route is " + catalog.RouteNames() + "."
	return hintSheet(a, HintDeclarePublic, "Declare a port public", why, hintLine{Command: DeclareCommand(p, mac, self)})
}

func DeclareCommand(p ports.Port, mac string, self string) string {
	line := "station ports declare "
	switch {
	case p.Machine.On != "":
		line += p.Machine.Name + " "
	case mac != "" && mac != self:
		line += mac + " "
	}
	return line + strconv.Itoa(p.Port) + " --via <route>"
}
