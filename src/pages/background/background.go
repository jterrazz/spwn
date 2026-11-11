package background

import (
	"strconv"

	"github.com/jterrazz/studio-station/src/background"
	"github.com/jterrazz/studio-station/src/core/finding"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/pages"
)

const (
	KeyRemove = "x"
	KeyCopy   = "c"
)

const Schema = "station.background.list/v1"

type group struct {
	id, title string
	folds     bool
}

func groups() []group {
	return []group{
		{"failing", "Failing", false},
		{"orphans", "Orphans", false},
		{"station", "Station", true},
		{"known", "Known apps", true},
		{"unknown", "Unknown", true},
	}
}

// value: the group a task is listed under: failing first, then an orphan,
// then the group the job manager's label gives it.
func groupOf(t background.Task) string {
	switch {
	case t.Run == background.Failing && t.Group != background.GroupStation:
		return "failing"
	case t.Group == background.GroupOrphan:
		return "orphans"
	}
	return string(t.Group)
}

func columns() []page.Column {
	return []page.Column{
		{Key: "label", Kind: page.KindText, Keep: 100},
		{Key: "scope", Kind: page.KindText, Keep: 60},
		{Key: "state", Kind: page.KindList, Keep: 90},
		{Key: "owner", Kind: page.KindText, Keep: 40},
	}
}

func List(tasks []background.Task, thisMac page.Context, all bool) page.Page {
	// invariant: SPEC §5.8: Failing and Orphans open, then Station, Known apps
	// and Unknown folded; --all unfolds them.
	p := page.Page{ID: "background", Command: "background", Context: thisMac, Actions: Actions()}
	if !all {
		p.Hints = append(p.Hints, page.Hint{Command: "station background --all", Text: "for every row"})
	}
	p.Sections = append(p.Sections, summary(tasks, thisMac))
	if len(tasks) == 0 {
		p.Sections = append(p.Sections, page.Section{ID: "tasks", Title: "Background", Columns: columns(), Items: []page.Item{{Empty: &page.Empty{Noun: "background tasks"}}}})
	}
	for _, g := range groups() {
		var items []page.Item
		for _, t := range tasks {
			if groupOf(t) == g.id {
				r := Row(t)
				items = append(items, page.Item{Row: &r})
			}
		}
		if len(items) == 0 {
			continue
		}
		fold := page.Open
		if g.folds && !all {
			fold = page.Folded
		}
		p.Sections = append(p.Sections, page.Section{ID: g.id, Title: g.title, Fold: fold, Columns: columns(), Items: items, PipeCounts: true})
	}
	return page.Seal(p)
}

func summary(tasks []background.Task, thisMac page.Context) page.Section {
	failing, orphans := counts(tasks)
	words := []string{plural(len(tasks), "background task")}
	if failing > 0 {
		words = append(words, strconv.Itoa(failing)+" failing")
	}
	if orphans > 0 {
		words = append(words, plural(orphans, "orphan"))
	}
	mark := page.OK
	switch {
	case failing > 0:
		mark = page.Broken
	case orphans > 0:
		mark = page.Attention
	}
	label := "This Mac"
	if thisMac.Kind != string(machine.Computer) && thisMac.Machine != "" {
		label = thisMac.Machine
	}
	row := page.Row{ID: "summary/background", Label: label, Mark: mark, Cells: []page.Value{page.Texts(words...)}}
	return page.Section{ID: "summary", Title: "Summary", Role: page.Summary, Fold: page.Open, Uncounted: true,
		Columns: []page.Column{{Key: "area", Kind: page.KindText, Keep: 100}, {Key: "value", Kind: page.KindList, Keep: 60}}, Items: []page.Item{{Row: &row}}}
}

// value: how many tasks fail, and how many are orphans.
func counts(tasks []background.Task) (failing, orphans int) {
	for _, t := range tasks {
		if t.Run == background.Failing {
			failing++
		}
		if t.Group == background.GroupOrphan {
			orphans++
		}
	}
	return failing, orphans
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return strconv.Itoa(n) + " " + word + "s"
}

func Actions() []page.Action {
	return []page.Action{
		{Key: KeyRemove, Label: "remove", On: page.OnRow, Priority: 1, Mutates: true, Confirm: &page.Confirm{Yes: "remove"}},
		{Key: KeyCopy, Label: "copy the command", On: page.OnRow, Priority: 2},
	}
}

func Confirm(label string) page.Confirm {
	return page.Confirm{Question: "Remove " + label + "?", Yes: "remove"}
}

func State(t background.Task) page.Value {
	// value: a task's state in the words of every surface: running with its
	// pid, failing with its last exit, idle, not loaded; station's own job
	// not loaded is stopped.
	switch t.Run {
	case background.Running:
		return page.Texts("running", "pid "+strconv.Itoa(t.PID))
	case background.Failing:
		return page.Texts("failing", "last exit "+strconv.Itoa(t.Exit))
	case background.Unread:
		return page.Text("?")
	case background.Idle, background.NotLoaded:
	}
	if t.Group == background.GroupStation && t.Run == background.NotLoaded {
		return page.Text("stopped")
	}
	return page.Text(string(t.Run))
}

// value: the command that removes a task: station's own for a task of the
// owner's scope, the one to type by hand for a root one; none for station's.
func fixOf(t background.Task) string {
	switch {
	case t.Group == background.GroupStation:
		return ""
	case t.Scope == background.ScopeRoot:
		return t.ByHand
	}
	return "station background remove " + t.Label
}

func Row(t background.Task) page.Row {
	r := page.Row{ID: t.Label, Label: t.Label, Mark: pages.MarkOf(t.State),
		Cells: []page.Value{page.Text(t.Scope), State(t), owner(t)}}
	if t.Plist != "" {
		r.Detail = page.Path(t.Plist)
	}
	switch {
	case t.Run == background.Failing:
		r.Reason = page.Failing
	case t.Group == background.GroupOrphan:
		r.Reason = page.Orphan
	case t.Run == background.Unread:
		r.Reason = page.Late
	case t.Group == background.GroupStation && t.Run == background.NotLoaded:
		r.Reason, r.Hint = page.Stopped, page.Cmd("station engine start")
	}
	if r.Reason == page.Failing || r.Reason == page.Orphan {
		switch fix := fixOf(t); {
		case t.Plist == "":
			r.Detail = page.Text("managed by " + t.Owner + ", remove it from the app")
		case t.Scope == background.ScopeRoot && fix != "":
			r.Hint = &page.Hint{Command: fix, External: true}
		case fix != "":
			r.Hint = page.KeyHint(fix, KeyRemove)
		}
	}
	return r
}

func owner(t background.Task) page.Value {
	switch {
	case t.Owner == "":
		return page.None()
	case t.OwnerGone:
		return page.Text(t.Owner + " (gone)")
	}
	return page.Text(t.Owner)
}

func Removed(t background.Task, thisMac page.Context) page.Page {
	r := page.Row{ID: t.Label, Label: t.Label, Mark: page.OK, Cells: []page.Value{page.Text("removed")}}
	if t.Plist != "" {
		r.Cells = append(r.Cells, page.Text(t.Plist+" deleted"))
	}
	columns := []page.Column{{Key: "label", Kind: page.KindText, Keep: 100}, {Key: "outcome", Kind: page.KindText, Keep: 90}, {Key: "file", Kind: page.KindText, Keep: 40}}
	return page.Seal(page.Page{ID: "background-remove", Command: "background remove", Context: thisMac,
		Sections: []page.Section{{ID: "removed", Title: "Removed", Role: page.Banner, Uncounted: true, Columns: columns, Items: []page.Item{{Row: &r}}}}})
}

type FixJSON struct {
	Command string `json:"command"`
}

type TaskJSON struct {
	Label    string   `json:"label"`
	Group    string   `json:"group"`
	Scope    string   `json:"scope"`
	Owner    string   `json:"owner"`
	State    string   `json:"state"`
	PID      *int     `json:"pid"`
	LastExit *int     `json:"last_exit"`
	Plist    string   `json:"plist"`
	Fix      *FixJSON `json:"fix"`
}

type ListJSON struct {
	Schema   string         `json:"schema"`
	Findings []finding.JSON `json:"findings"`
	Tasks    []TaskJSON     `json:"tasks"`
}

func stateWord(t background.Task) string {
	switch t.Run {
	case background.Running:
		return "running"
	case background.Idle:
		return "loaded"
	case background.Failing:
		return "failing"
	case background.NotLoaded:
		return "not-loaded"
	case background.Unread:
	}
	return "unknown"
}

func ListContract(tasks []background.Task) ListJSON {
	body := ListJSON{Schema: Schema, Findings: finding.Doc(background.Findings(tasks, machine.Ref{})), Tasks: make([]TaskJSON, 0, len(tasks))}
	for _, t := range tasks {
		o := TaskJSON{Label: t.Label, Group: string(t.Group), Scope: t.Scope, Owner: t.Owner, State: stateWord(t), Plist: t.Plist}
		switch t.Run {
		case background.Running:
			o.PID = &t.PID
		case background.Failing:
			o.LastExit = &t.Exit
		case background.Idle, background.NotLoaded, background.Unread:
		}
		if fix := fixOf(t); fix != "" && (t.Scope == background.ScopeRoot || t.Run == background.Failing || t.Group == background.GroupOrphan) {
			o.Fix = &FixJSON{Command: fix}
		}
		body.Tasks = append(body.Tasks, o)
	}
	return body
}
