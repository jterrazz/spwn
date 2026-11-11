package settings

import (
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/finding"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/pages"
	"github.com/jterrazz/studio-station/src/settings"
)

const (
	ID       = "settings"
	KeyApply = "a"
	KeyOff   = "o"
	KeyDiff  = "d"
	KeyFix   = "f"
)

func columns() []page.Column {
	return []page.Column{{Key: "name", Kind: page.KindText, Keep: 100}, {Key: "state", Kind: page.KindText, Keep: 60}}
}

// value: how long ago, in its largest unit alone (`2 d ago`).
func ago(since, now time.Time) string {
	d := now.Sub(since)
	for _, u := range []struct {
		n    time.Duration
		name string
	}{{24 * time.Hour, "d"}, {time.Hour, "h"}, {time.Minute, "min"}} {
		if d >= u.n {
			return strconv.Itoa(int(d/u.n)) + " " + u.name + " ago"
		}
	}
	return "just now"
}

func Row(r settings.Row, now time.Time) page.Row {
	// value: one setting as a row: its mark, its state in words, where it
	// lives, and the next step.
	name := r.Name
	row := page.Row{ID: name, Label: name}
	if r.Group == settings.Checked && r.State != settings.StateReading {
		return checkedRow(r, row)
	}
	switch r.State {
	case settings.StateApplied:
		row.Mark, row.Cells = page.OK, []page.Value{page.Texts("applied", r.Where)}
	case settings.StateChangedSince:
		row.Mark, row.Reason = page.Attention, page.Differs
		row.Cells = []page.Value{page.Text("changed since " + ago(r.Since, now))}
		row.Hint = page.KeyHint("station settings diff "+name, KeyDiff)
	case settings.StateNotApplied:
		row.Mark, row.Cells = page.Absent, []page.Value{page.Text("not applied")}
		row.Hint = page.KeyHint("station settings apply "+name, KeyApply)
		if r.Needs != "" {
			row.Reason = page.Missing
			row.Cells = []page.Value{page.Texts("not applied", "needs "+r.Item.NeedsSaid())}
			row.Hint = page.Cmd("station apps install " + r.Needs)
		}
	case settings.StateOffThisMac:
		row.Mark, row.Cells = page.OffHere, []page.Value{page.Text("off here")}
	case settings.StateNotForThisMachine:
		row.Mark, row.Cells = page.NotForThisMachine, []page.Value{page.Texts("not for this machine", r.Why)}
	case settings.StateReading:
		row.Mark, row.Detail = page.Awaiting, page.Pending()
	}
	return row
}

func checkedRow(r settings.Row, row page.Row) page.Row {
	switch r.Checked {
	case settings.AsExpected:
		row.Mark, row.Cells = page.OK, []page.Value{page.Texts("as expected", r.Value)}
	case settings.NeedsFix:
		row.Mark, row.Reason = page.Attention, page.Mismatch
		row.Cells = []page.Value{page.Texts(r.Value, r.Expected)}
		row.Hint = page.Cmd(r.Fix.Path)
	case settings.CheckedUnknown:
		row.Mark, row.Reason, row.Cells = page.NotKnown, page.Late, []page.Value{page.Texts("unknown", r.Value)}
	case settings.CheckedOffThisMac:
		row.Mark, row.Cells = page.OffHere, []page.Value{page.Texts("off here", "not checked on this machine")}
	case settings.CheckedNotForThisMachine:
		row.Mark, row.Cells = page.NotForThisMachine, []page.Value{page.Texts("not for this machine", r.Why)}
	}
	return row
}

func checkedOrder(s settings.CheckedState) int {
	switch s {
	case settings.NeedsFix:
		return 0
	case settings.CheckedUnknown:
		return 1
	case settings.AsExpected:
		return 2
	case settings.CheckedOffThisMac:
		return 3
	case settings.CheckedNotForThisMachine:
		return 5
	}
	return 4
}

func order(s settings.State) int {
	switch s {
	case settings.StateChangedSince:
		return 0
	case settings.StateNotApplied:
		return 1
	case settings.StateApplied:
		return 2
	case settings.StateOffThisMac:
		return 3
	case settings.StateReading:
		return 4
	case settings.StateNotForThisMachine:
		return 5
	}
	return 6
}

func rank(r settings.Row) int {
	if r.Group == settings.Checked && r.State != settings.StateReading {
		return checkedOrder(r.Checked)
	}
	return order(r.State)
}

func sorted(rows []settings.Row) []settings.Row {
	out := append([]settings.Row(nil), rows...)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && rank(out[j]) < rank(out[j-1]); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func items(rows []settings.Row, now time.Time) []page.Item {
	out := make([]page.Item, 0, len(rows))
	for _, r := range sorted(rows) {
		row := Row(r, now)
		out = append(out, page.Item{Row: &row})
	}
	return out
}

// value: the rows that are not for this machine, folded on one line with
// their reasons, unless every group is unfolded.
func notHere(rows []settings.Row) page.Row {
	var why []string
	for _, r := range rows {
		if !contains(why, r.Why) {
			why = append(why, r.Why)
		}
	}
	return page.Row{ID: "not-for-this-machine", Label: strconv.Itoa(len(rows)) + " not for this machine", Mark: page.NotForThisMachine, Cells: []page.Value{page.Texts(why...)}}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// value: the Always-on Mac group folded: none of it applied on this machine.
func alwaysOnFolded(rows []settings.Row) page.Row {
	return page.Row{ID: "always-on-mac", Label: "Always-on Mac", Mark: page.Absent, Cells: []page.Value{page.Texts(strconv.Itoa(len(rows)), "for a Mac that hosts servers")}}
}

func touched(rows []settings.Row) bool {
	for _, r := range rows {
		if r.State == settings.StateApplied || r.State == settings.StateChangedSince || r.State == settings.StateOffThisMac {
			return true
		}
	}
	return false
}

func List(rows []settings.Row, now time.Time, all bool, thisMac page.Context) page.Page {
	// invariant: the first line sums the machine up, then Needs a look lifts
	// every row that needs one first, whatever its group; then the Set by station
	// group, the Always-on Mac group folded on one line until one of it is
	// applied here, the Checked group and what is not for this machine, each
	// one line on a terminal (vision 07); --all unfolds them.
	var set, always, away, checked, look []settings.Row
	for _, r := range rows {
		if lookAt(r) {
			look = append(look, r)
		}
		switch {
		case r.Group == settings.Checked:
			checked = append(checked, r)
		case r.State == settings.StateNotForThisMachine:
			away = append(away, r)
		case r.Group == settings.AlwaysOnMac:
			always = append(always, r)
		default:
			set = append(set, r)
		}
	}
	p := page.Page{ID: ID, Command: "settings", Context: thisMac, Actions: actions()}
	p.Sections = append(p.Sections, firstLine(rows, len(look), thisMac))
	if len(look) > 0 {
		p.Sections = append(p.Sections, page.Section{ID: "needs-a-look", Title: "Needs a look", Role: page.Lifted, Fold: page.Open, Columns: columns(), Items: items(look, now)})
	}
	if len(set) > 0 {
		fold := page.Calm
		if all {
			fold = page.Open
		}
		p.Sections = append(p.Sections, page.Section{ID: "set", Title: "Set by station", Fold: fold, Columns: columns(), Items: items(set, now), PipeCounts: true})
	}
	if len(always) > 0 {
		s := page.Section{ID: "always-on-mac", Title: "Always-on Mac", Columns: columns(), PipeCounts: true}
		if all || touched(always) {
			s.Fold, s.Items = brief(all), items(always, now)
		} else {
			s.Uncounted, s.PipeCounts = true, false
			folded := alwaysOnFolded(always)
			s.Items = []page.Item{{Row: &folded}}
		}
		p.Sections = append(p.Sections, s)
	}
	if len(checked) > 0 {
		p.Sections = append(p.Sections, checkedSection(checked, all, now))
	}
	if len(away) > 0 {
		s := page.Section{ID: "not-for-this-machine", Title: "Not for this machine", Columns: columns(), Uncounted: true}
		if all {
			s.Items = items(away, now)
		} else {
			folded := notHere(away)
			s.Items = []page.Item{{Row: &folded}}
		}
		p.Sections = append(p.Sections, s)
	}
	p.Hints = next(set, all)
	return page.Seal(p)
}

// invariant: one or two next steps (SPEC §5): apply what station sets,
// counted when some of it waits, then every row, or one setting's detail
// once every row shows.
func next(set []settings.Row, all bool) []page.Hint {
	apply := page.Hint{Command: "station settings apply"}
	n := 0
	for _, r := range set {
		if r.State == settings.StateNotApplied || r.State == settings.StateChangedSince {
			n++
		}
	}
	if n > 0 {
		apply.Text = "apply the " + strconv.Itoa(n)
	}
	if all {
		return []page.Hint{apply, *page.Cmd("station settings <setting>")}
	}
	return []page.Hint{apply, {Command: "station settings --all", Text: "for every row"}}
}

// value: a row that needs a look: changed since it was applied, a checked
// setting that is not as expected, or one that failed.
func lookAt(r settings.Row) bool {
	if r.Group == settings.Checked && r.State != settings.StateReading {
		return r.Checked == settings.NeedsFix
	}
	return r.State == settings.StateChangedSince
}

// value: a calm group is one line on a terminal's page, open in the TUI
// and the menu bar; --all unfolds it everywhere.
func brief(all bool) page.Fold {
	if all {
		return page.Open
	}
	return page.Brief
}

// value: the first line: the machine, what station sets on it, what it
// checks, and what needs a look (vision 07 "station settings").
func firstLine(rows []settings.Row, look int, thisMac page.Context) page.Section {
	set, checked := 0, 0
	for _, r := range rows {
		switch {
		case r.State == settings.StateNotForThisMachine, r.Group == settings.Checked && r.Checked == settings.CheckedNotForThisMachine:
		case r.Group == settings.Checked:
			checked++
		case r.Group == settings.SetByStation:
			set++
		}
	}
	label := "This Mac"
	if thisMac.Machine != "" && thisMac.Kind != string(machine.Computer) {
		label = thisMac.Machine
	}
	words := []string{strconv.Itoa(set) + " set by station here"}
	if checked > 0 {
		words = append(words, strconv.Itoa(checked)+" checked")
	}
	switch look {
	case 0:
		words = append(words, "nothing needs a look")
	case 1:
		words = append(words, "1 needs a look")
	default:
		words = append(words, strconv.Itoa(look)+" need a look")
	}
	mark := page.OK
	if look > 0 {
		mark = page.Attention
	}
	row := page.Row{ID: "summary/settings", Label: label, Mark: mark, Cells: []page.Value{page.Texts(words...)}}
	return page.Section{ID: "summary", Title: "Summary", Role: page.Summary, Fold: page.Open, Uncounted: true, Columns: []page.Column{{Key: "area", Kind: page.KindText, Keep: 100}, {Key: "value", Kind: page.KindList, Keep: 60}}, Items: []page.Item{{Row: &row}}}
}

// value: the Checked group; on a machine none of it lives on, one folded
// line with the reason, unless every group is unfolded.
func checkedSection(rows []settings.Row, all bool, now time.Time) page.Section {
	s := page.Section{ID: "checked", Title: "Checked", Fold: brief(all), Columns: columns(), Items: items(rows, now), PipeCounts: true}
	for _, r := range rows {
		if r.Checked != settings.CheckedNotForThisMachine {
			return s
		}
	}
	s.Fold, s.Uncounted, s.PipeCounts = "", true, false
	if !all {
		folded := page.Row{ID: "checked-not-for-this-machine", Label: strconv.Itoa(len(rows)) + " checked", Mark: page.NotForThisMachine, Cells: []page.Value{page.Texts("not for this machine", rows[0].Why)}}
		s.Items = []page.Item{{Row: &folded}}
	}
	return s
}

func actions() []page.Action {
	return []page.Action{
		{Key: KeyApply, Label: "apply", On: page.OnRow, Priority: 1, Mutates: true},
		{Key: KeyDiff, Label: "diff", On: page.OnRow, Priority: 2},
		{Key: KeyOff, Label: "off here", On: page.OnRow, Priority: 3, Mutates: true, Confirm: &page.Confirm{Yes: "off"}},
		{Key: KeyFix, Label: "open the fix", On: page.OnRow, Priority: 4},
	}
}

func OffConfirm(names []string, also ...string) page.Confirm {
	// value: also is what else the off does, said in the question: the
	// servers that pause with a Mac that sleeps again.
	q := "Turn " + strings.Join(names, ", ") + " off here? station stops applying and watching it on this machine"
	if len(also) > 0 {
		q += "; " + strings.Join(also, "; ")
	}
	return page.Confirm{Question: q, Yes: "off"}
}

func ApplyConfirm(names []string) page.Confirm {
	what := strings.Join(names, ", ")
	if what == "" {
		what = "every setting not applied or changed since"
	}
	return page.Confirm{Question: "Apply " + what + "? A setting changed since is overwritten: yours is lost", Yes: "apply"}
}

func Show(r settings.Row, now time.Time, thisMac page.Context) page.Page {
	// value: one setting's detail: its group, state, where it lives and what
	// it does, and the machines it lives on.
	head := Row(r, now)
	if r.Group == settings.Checked {
		return showChecked(r, head, thisMac)
	}
	group := page.Text("Set by station")
	if r.Group == settings.AlwaysOnMac {
		group = page.Texts("Set by station", "Always-on Mac")
	}
	facts := []page.Item{
		{Fact: &page.Fact{Key: "group", Value: group}},
		{Fact: &page.Fact{Key: "does", Value: page.Text(r.Item.Description)}},
		{Fact: &page.Fact{Key: "lives on", Value: page.Text(livesOn(r.Item.Kinds))}},
	}
	if r.Where != "" {
		facts = append(facts, page.Item{Fact: &page.Fact{Key: "where", Value: page.Text(r.Where)}})
	}
	if !r.Since.IsZero() {
		facts = append(facts, page.Item{Fact: &page.Fact{Key: "applied", Value: page.Text(ago(r.Since, now))}})
	}
	p := page.Page{ID: ID, Command: "settings " + r.Name, Context: thisMac, Sections: []page.Section{
		{ID: "setting", Title: r.Name, Role: page.Banner, Columns: columns(), Items: []page.Item{{Row: &head}}},
		{ID: "detail", Title: "Detail", Columns: pages.OutcomeColumns(), Items: facts},
	}}
	return page.Seal(p)
}

func showChecked(r settings.Row, head page.Row, thisMac page.Context) page.Page {
	it := r.CheckedItem()
	facts := []page.Item{
		{Fact: &page.Fact{Key: "group", Value: page.Text("Checked")}},
		{Fact: &page.Fact{Key: "does", Value: page.Text(it.Description)}},
		{Fact: &page.Fact{Key: "lives on", Value: page.Text(livesOn(it.Kinds))}},
		{Fact: &page.Fact{Key: "fix", Value: page.Text(it.Fix)}},
	}
	if r.Expected != "" {
		facts = append(facts, page.Item{Fact: &page.Fact{Key: "expected", Value: page.Text(r.Expected)}})
	}
	return page.Seal(page.Page{ID: ID, Command: "settings " + it.Name, Context: thisMac, Sections: []page.Section{
		{ID: "setting", Title: it.Name, Role: page.Banner, Columns: columns(), Items: []page.Item{{Row: &head}}},
		{ID: "detail", Title: "Detail", Columns: pages.OutcomeColumns(), Items: facts},
	}})
}

func livesOn(livesOn []catalog.MachineKind) string {
	words := make([]string, 0, len(livesOn))
	for _, p := range livesOn {
		switch p {
		case catalog.Mac:
			words = append(words, "Macs")
		case catalog.Linux:
			words = append(words, "Linux VMs")
		case catalog.MacOS:
			words = append(words, "macOS VMs")
		}
	}
	return strings.Join(words, ", ")
}

func Values(r settings.Row, before, after []settings.KeyValue, thisMac page.Context) page.Page {
	// value: a setting that writes no file, what station would set against
	// what is there: one row per value, before then after; when every value
	// is already station's, the setting's own row.
	was := map[string]string{}
	for _, kv := range before {
		was[kv.Key] = kv.Value
	}
	rows := make([]page.Item, 0, len(after))
	differs := false
	for _, kv := range after {
		now := was[kv.Key]
		if now == "" {
			now = "unset"
		}
		row := page.Row{ID: kv.Key, Label: kv.Key, Mark: page.Absent, Cells: []page.Value{page.Text(now), page.Text(kv.Value)}}
		if now == kv.Value {
			row.Mark = page.OK
		} else {
			differs = true
		}
		rows = append(rows, page.Item{Row: &row})
	}
	cols := []page.Column{{Key: "name", Kind: page.KindText, Keep: 100}, {Key: "before", Kind: page.KindText, Keep: 40}, {Key: "after", Kind: page.KindText, Keep: 40}}
	s := page.Section{ID: "values", Title: r.Name, Columns: cols, Items: rows}
	if !differs {
		head := Row(r, time.Time{})
		s.Items = []page.Item{{Row: &head}}
	}
	return page.Seal(page.Page{ID: ID, Command: "settings diff " + r.Name, Context: thisMac, Sections: []page.Section{s}})
}

func Changes(verb string, changes []settings.Change, thisMac page.Context) page.Page {
	// value: each setting a verb ran on, its outcome and its steps.
	var rows []page.Row
	var extra []page.Section
	for _, c := range changes {
		extra = append(extra, values(c.Item, "before", c.Before)...)
		extra = append(extra, values(c.Item, "after", c.After)...)
		head := pages.Done(c.Item, c.Result, "")
		if len(c.Steps) == 1 {
			s := c.Steps[0]
			head.Detail = page.Text(step(s))
			rows = append(rows, head)
			continue
		}
		rows = append(rows, head)
		for i, s := range c.Steps {
			r := pages.Done(c.Item+"/"+strconv.Itoa(i), s.State, s.Detail)
			r.Label = s.Label
			rows = append(rows, r)
		}
	}
	return pages.Outcome("settings "+verb, thisMac, nil, rows, "settings to "+verb, extra...)
}

func values(item, when string, list []settings.KeyValue) []page.Section {
	if len(list) == 0 {
		return nil
	}
	items := make([]page.Item, 0, len(list))
	for _, kv := range list {
		items = append(items, page.Item{Fact: &page.Fact{Key: kv.Key, Value: page.Text(kv.Value)}})
	}
	return []page.Section{{ID: item + "-" + when, Title: item + " " + when, Columns: pages.OutcomeColumns(), Items: items}}
}

func step(s settings.ChangeStep) string {
	if s.Detail == "" {
		return s.Label
	}
	return s.Label + ": " + s.Detail
}

type SettingJSON struct {
	Name      string     `json:"name"`
	Label     string     `json:"label"`
	Group     string     `json:"group"`
	State     string     `json:"state"`
	AppliedAt *time.Time `json:"applied_at"`
	Where     string     `json:"where,omitempty"`
	Why       string     `json:"why,omitempty"`
	Admin     bool       `json:"admin"`
	Value     string     `json:"value,omitempty"`
	Fix       string     `json:"fix,omitempty"`
}

type ListJSON struct {
	Schema   string         `json:"schema"`
	Findings []finding.JSON `json:"findings"`
	Settings []SettingJSON  `json:"settings"`
}

func Object(r settings.Row) SettingJSON {
	if r.Group == settings.Checked {
		state := string(r.Checked)
		if state == "" {
			state = string(r.State)
		}
		return SettingJSON{Name: r.Name, Label: r.CheckedItem().Label, Group: string(r.Group), State: state, Why: r.Why, Value: r.Value, Fix: r.Fix.Path}
	}
	o := SettingJSON{Name: r.Name, Label: r.Item.Description, Group: string(r.Group), State: string(r.State), Where: r.Where, Why: r.Why, Admin: r.Item.NeedsRoot}
	if !r.Since.IsZero() {
		at := r.Since.UTC()
		o.AppliedAt = &at
	}
	return o
}

func ListContract(rows []settings.Row) ListJSON {
	body := ListJSON{Schema: "station.settings.list/v1", Findings: finding.Doc(settings.Findings(rows, machine.Ref{})), Settings: make([]SettingJSON, 0, len(rows))}
	for _, r := range rows {
		body.Settings = append(body.Settings, Object(r))
	}
	return body
}

type ShowJSON struct {
	Schema  string      `json:"schema"`
	Setting SettingJSON `json:"setting"`
}

func ShowContract(r settings.Row) ShowJSON {
	return ShowJSON{Schema: "station.settings.show/v1", Setting: Object(r)}
}

func Removed(o settings.ContainersOutcome, command string, thisMac page.Context) page.Page {
	// value: what one of the Docker shortcuts removed, by what it removes,
	// under the command that ran it.
	rows := make([]page.Row, 0, len(o.Removals))
	for _, r := range o.Removals {
		detail := page.Text("none to remove")
		if r.Count > 0 {
			detail = page.Count(int64(r.Count), "removed")
		}
		row := pages.Done(r.What, catalog.StateOK, "")
		row.Detail = detail
		rows = append(rows, row)
	}
	return pages.Outcome(command, thisMac, nil, rows, "steps")
}

func Diff(name, diff string, thisMac page.Context) page.Page {
	// value: a file's unified diff as a page, one line per line, for the
	// renderers that cannot print it raw (the TUI's detail pane).
	lines := strings.Split(strings.TrimRight(diff, "\n"), "\n")
	items := make([]page.Item, 0, len(lines))
	for _, l := range lines {
		items = append(items, page.Item{Note: &page.Note{Text: l}})
	}
	return page.Seal(page.Page{ID: ID, Command: "settings diff " + name, Context: thisMac, Sections: []page.Section{{ID: "diff", Title: name, Uncounted: true, Items: items}}})
}
