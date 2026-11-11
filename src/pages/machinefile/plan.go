package machinefile

import (
	"slices"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/machinefile"
	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/pages"
)

// value: what a need asks of the owner, said once at the head of a plan.
// The password is asked by sudo itself; Touch ID is no word of station's
// until a setting turns it on.
func asks(n catalog.Need) string {
	switch n {
	case catalog.NeedSudo:
		return "asks your password once"
	case catalog.NeedTerminal:
		return "asks you on this terminal"
	case catalog.NeedLicense:
		return "asks you to accept a license"
	case catalog.NeedAppStoreLogin:
		return "needs the App Store signed in"
	}
	return string(n)
}

// value: what has to happen once a plan is applied, in the owner's words.
func restart(a catalog.AfterApply) string {
	switch a {
	case catalog.RestartDock:
		return "the Dock restarts"
	case catalog.RestartFinder:
		return "the Finder restarts"
	case catalog.RestartMenuBar:
		return "the menu bar restarts"
	case catalog.AfterLogout:
		return "log out and in again"
	case catalog.AfterReboot:
		return "restart the Mac"
	}
	return string(a)
}

func columns() []page.Column {
	return []page.Column{{Key: "name", Kind: page.KindText, Keep: 100}, {Key: "method", Kind: page.KindList, Keep: 40}}
}

// value: the values a step is given, as the plan shows them: `key value`,
// a `vault:` reference as itself, never what it names.
func values(s machinefile.Step) []string {
	keys := make([]string, 0, len(s.Values))
	for k := range s.Values {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k+" "+string(s.Values[k]))
	}
	return out
}

func lineRow(l machinefile.Line, words ...string) page.Row {
	row := page.Row{ID: l.Step.Item, Label: l.Step.Item, Mark: pages.MarkOf(l.Seen.State), Cells: []page.Value{page.Texts(l.Step.Method)}}
	if l.Seen.Condition == machinefile.NotManaged {
		row.Mark = page.NotKnown
	}
	if l.Seen.State == "" {
		row.Mark = page.Absent
	}
	detail := l.Seen.Detail
	if slices.Contains(words, detail) || detail == string(l.Seen.Condition) {
		detail = ""
	}
	parts := slices.DeleteFunc(append(append(slices.Clone(words), values(l.Step)...), detail), func(s string) bool { return s == "" })
	if len(parts) > 0 {
		row.Detail = page.Texts(parts...)
	}
	return row
}

func section(id, title string, rows []page.Row) page.Section {
	items := make([]page.Item, 0, len(rows))
	for i := range rows {
		items = append(items, page.Item{Row: &rows[i]})
	}
	return page.Section{ID: id, Title: title, Columns: columns(), Items: items}
}

func why(l machinefile.Line) string {
	if l.Step.Why == "listed" {
		return ""
	}
	return l.Step.Why
}

func needsSection(p machinefile.Plan) []page.Section {
	if len(p.Needs) == 0 {
		return nil
	}
	rows := make([]page.Row, 0, len(p.Needs))
	for _, a := range p.Needs {
		rows = append(rows, page.Row{ID: string(a.Need), Label: strings.Join(a.Items, ", "), Mark: page.Attention, Detail: page.Text(asks(a.Need))})
	}
	return []page.Section{section("needs", "Asks you", rows)}
}

func restartsSection(r []catalog.AfterApply) []page.Section {
	if len(r) == 0 {
		return nil
	}
	rows := make([]page.Row, 0, len(r))
	for _, a := range r {
		rows = append(rows, page.Row{ID: string(a), Label: restart(a)})
	}
	return []page.Section{section("after-apply", "Once applied", rows)}
}

// value: what `apply --remove` does with an extra, in the owner's words.
func fate(e machinefile.ExtraLine) string {
	switch {
	case e.Takes():
		return "listed no longer"
	case e.Action == machinefile.Remove:
		return "already gone"
	case e.Seen.Condition == machinefile.NotManaged || e.Seen.Condition == machinefile.Present || e.Seen.Condition == machinefile.Outdated:
		return "present another way since: kept"
	}
	return "reads " + condition(e.Seen.Condition) + ": kept"
}

func extraRows(p machinefile.Plan, lead ...string) []page.Row {
	rows := make([]page.Row, 0, len(p.Extra))
	for _, e := range p.Extra {
		words := append(slices.Clone(lead), "installed by station for "+e.Installed.Machinefile+", "+fate(e))
		if e.Takes() {
			for _, n := range e.Step.Needs {
				words = append(words, asks(n))
			}
		}
		rows = append(rows, page.Row{ID: e.Step.Item, Label: e.Step.Item, Mark: page.NotKnown, Cells: []page.Value{page.Texts(e.Step.Method)}, Detail: page.Texts(words...)})
	}
	return rows
}

func counted(n int, what string) string { return strconv.Itoa(n) + " " + what }

func head(label string, mark page.Mark, words []string) page.Section {
	row := page.Row{ID: "summary/head", Label: label, Mark: mark}
	if len(words) > 0 {
		row.Cells = []page.Value{page.Texts(words...)}
	}
	return page.Section{ID: "summary", Title: "Summary", Role: page.Summary, Fold: page.Open, Uncounted: true, Columns: []page.Column{{Key: "area", Kind: page.KindText, Keep: 100}, {Key: "value", Kind: page.KindList, Keep: 60}}, Items: []page.Item{{Row: &row}}}
}

func Plan(p machinefile.Plan, thisMac page.Context) page.Page {
	// invariant: a plan opens with what it asks of the owner, then what it
	// installs, sets and adopts, what it leaves as it is, what is extra,
	// and what has to happen once applied, each once; it ends on apply.
	byAction := func(a machinefile.Action) []page.Row {
		rows := make([]page.Row, 0, len(p.Lines))
		for _, l := range p.Of(a) {
			rows = append(rows, lineRow(l, why(l)))
		}
		return rows
	}
	var words []string
	sections := needsSection(p)
	for _, part := range []struct {
		action      machinefile.Action
		id, title   string
		count, noun string
	}{
		{machinefile.Install, "install", "To install", "to install", ""},
		{machinefile.Set, "set", "To set", "to set", ""},
		{machinefile.Adopt, "adopt", "To adopt", "to adopt", ""},
		{machinefile.Leave, "leave", "Left as it is", "left as it is", ""},
	} {
		if rows := byAction(part.action); len(rows) > 0 {
			sections = append(sections, section(part.id, part.title, rows))
			words = append(words, counted(len(rows), part.count))
		}
	}
	if kept := len(p.Of(machinefile.Keep)); kept > 0 {
		words = append(words, counted(kept, "present"))
	}
	if rows := extraRows(p); len(rows) > 0 {
		sections = append(sections, section("extra", "Extra", rows))
		words = append(words, counted(len(rows), "extra"))
	}
	sections = append(sections, restartsSection(p.Restarts)...)
	mark := page.OK
	var hints []page.Hint
	switch {
	case !p.Calm():
		mark = page.Absent
		hints = append(hints, page.Hint{Command: "station apply", Text: "carries it out"})
		if slices.ContainsFunc(p.Extra, func(e machinefile.ExtraLine) bool { return e.Action == machinefile.Remove }) {
			hints = append(hints, page.Hint{Command: "station apply --remove", Text: "also removes what is extra"})
		}
	case len(words) == 0:
		words = []string{"nothing to do"}
	}
	sections = append([]page.Section{head(p.Name, mark, words)}, sections...)
	return page.Seal(page.Page{ID: ID, Command: "plan", Context: thisMac, Sections: sections, Hints: hints})
}

// value: a condition in the owner's words.
func condition(c machinefile.Condition) string { return strings.ReplaceAll(string(c), "-", " ") }

func Drift(p machinefile.Plan, thisMac page.Context) page.Page {
	// invariant: drift is the plan as conditions, one row per item in the
	// plan's order, then what is extra; what is as the Machinefile asks
	// counts as present, and anything else is drift.
	rows := make([]page.Row, 0, len(p.Lines)+len(p.Extra))
	drifted := 0
	for _, l := range p.Lines {
		rows = append(rows, lineRow(l, condition(l.Seen.Condition)))
		if l.Seen.Condition != machinefile.Present {
			drifted++
		}
	}
	for _, r := range extraRows(p, condition(machinefile.Extra)) {
		rows = append(rows, r)
		drifted++
	}
	mark, words := page.OK, []string{"no drift"}
	var hints []page.Hint
	if drifted > 0 {
		mark, words = page.Attention, []string{counted(drifted, "drifted"), counted(len(rows)-drifted, "present")}
		hints = []page.Hint{{Command: "station plan", Text: "what apply would do"}}
	}
	sections := []page.Section{head(p.Name, mark, words), section("items", "Items", rows)}
	return page.Seal(page.Page{ID: ID, Command: "drift", Context: thisMac, Sections: sections, Hints: hints})
}

// value: a line's words in a run once it ended, filled by the surface for
// the reasons whose words are its own.

type Worded struct {
	machinefile.Done
	Words []string
}

func Applied(a machinefile.Applied, done, removed []Worded, thisMac page.Context) page.Page {
	// invariant: apply reads as every run does (pages.RunPage): the line
	// that sums it up, each item with its mark, its method and why it ran,
	// then each extra removed, what has to happen once applied, each once,
	// and what to do next.
	steps := make([]pages.RunStep, 0, len(done)+len(removed))
	changed, failed, interrupted := 0, 0, a.Interrupted
	for _, d := range append(slices.Clone(done), removed...) {
		detail := d.Detail
		var said []string
		if w := why(d.Line); w != "" {
			said = append(said, w)
		}
		if d.Line.Action == machinefile.Adopt {
			said = append(said, "adopted")
		}
		if d.Line.Action == machinefile.Remove && d.State == catalog.StateOK {
			said = append(said, "removed")
		}
		said = append(said, values(d.Line.Step)...)
		if len(d.Words) > 0 {
			said, detail = append(said, d.Words[:len(d.Words)-1]...), d.Words[len(d.Words)-1]
		}
		steps = append(steps, pages.RunStep{ID: d.Line.Step.Item, State: d.State, Cells: []page.Value{page.Version(d.Version), page.Texts(d.Line.Step.Method)}, Why: said, Detail: detail, Log: d.Log})
		switch {
		case d.Reason == machinefile.Interrupted:
			interrupted = true
		case d.State == catalog.StateBroken:
			failed++
		case d.Installed || (d.Line.Acts() && d.State == catalog.StateOK) || (d.Line.Action == machinefile.Remove && d.State == catalog.StateOK):
			changed++
		}
	}
	r := pages.Run{Command: "apply", Columns: []page.Column{{Key: "name", Kind: page.KindText, Keep: 100}, {Key: "version", Kind: page.KindVersion, Keep: 50}, {Key: "method", Kind: page.KindList, Keep: 30}}, Steps: steps, Empty: "items in " + a.Plan.Name, Extra: restartsSection(a.Plan.Restarts)}
	var words []string
	switch {
	case interrupted:
		r.Head = &pages.RunHead{Label: a.Plan.Name + " interrupted", Mark: page.Broken}
		r.Next = []page.Hint{{Command: "station apply", Text: "resumes"}}
	case failed > 0:
		r.Head = &pages.RunHead{Label: counted(failed, "of "+strconv.Itoa(len(steps))+" failed applying "+a.Plan.Name), Mark: page.Broken}
		r.Next = []page.Hint{{Command: "station apply", Text: "tries what failed again"}}
	default:
		r.Head = &pages.RunHead{Label: "Applied " + a.Plan.Name, Mark: page.OK}
	}
	if changed > 0 {
		words = append(words, counted(changed, "changed"))
	}
	if a.Bound != "" {
		words = append(words, "this machine carries "+a.Bound)
	}
	r.Head.Words = words
	return pages.RunPage(r, thisMac)
}
