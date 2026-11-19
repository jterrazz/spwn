package machinefile

import (
	"slices"
	"strconv"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/pages"
)

// value: one app's line of the lock as the owner reads it: its version
// and the method that holds it.
func lockedWords(a *catalog.LockedApp) string {
	if a.Version == "" {
		return "by " + a.Method
	}
	return a.Version + " by " + a.Method
}

func lockSection(d machinefile.LockDiff) []page.Section {
	if d.Empty() {
		return nil
	}
	rows := make([]page.Row, 0, len(d.Apps)+1)
	for _, c := range d.Apps {
		row := page.Row{ID: "lock/" + c.App, Label: c.App, Mark: page.OK}
		switch {
		case c.Was == nil:
			row.Detail = page.Text("added, " + lockedWords(c.Now))
		case c.Now == nil:
			row.Detail, row.Mark = page.Text("listed no longer"), page.Absent
		default:
			row.Detail = page.Text(lockedWords(c.Now) + ", was " + lockedWords(c.Was))
		}
		rows = append(rows, row)
	}
	if d.Source != nil {
		rows = append(rows, page.Row{ID: "lock/source", Label: "shared source", Mark: page.OK, Detail: page.Text(short(d.Source[1]) + ", was " + short(d.Source[0]))})
	}
	return []page.Section{{ID: "lock", Title: "Lock", Columns: columns(), Items: itemsOf(rows)}}
}

// value: why the shared source was not pulled, when it was not: the run
// went on from its clone as it stood.
func unpulledSection(why string) []page.Section {
	if why == "" {
		return nil
	}
	row := page.Row{ID: "source/unpulled", Label: "shared source", Mark: page.Attention, Detail: page.Text("not pulled, read where its clone stands: " + why)}
	return []page.Section{{ID: "source", Title: "Source", Columns: columns(), Items: itemsOf([]page.Row{row})}}
}

// value: each newer version past its range, and the line that takes it:
// `node  25.1.0 available: change '24' to '25' in <file>`.
func waitingSection(w []machinefile.Waiting) []page.Section {
	if len(w) == 0 {
		return nil
	}
	rows := make([]page.Row, 0, len(w))
	for _, v := range w {
		rows = append(rows, page.Row{ID: "waiting/" + v.App, Label: v.App, Mark: page.Attention, Detail: page.Text(v.Words())})
	}
	return []page.Section{{ID: "waiting", Title: "Waiting", Columns: columns(), Items: itemsOf(rows)}}
}

func itemsOf(rows []page.Row) []page.Item {
	items := make([]page.Item, 0, len(rows))
	for i := range rows {
		items = append(items, page.Item{Row: &rows[i]})
	}
	return items
}

// value: a source as `<name>@<commit>`, its commit cut to the first seven
// characters; none, when the lock read no shared source.
func short(source string) string {
	if source == "" {
		return "none"
	}
	if at := len(source) - 40; at > 0 && source[at-1] == '@' {
		return source[:at+7]
	}
	return source
}

func Upgraded(u machinefile.Upgraded, done []Worded, thisMac page.Context) page.Page {
	// invariant: upgrade reads as every run does (pages.RunPage): the line
	// that sums it up, each app with its mark, its version and its method
	// in the order it ran (the managers first), a version the Machinefile
	// pins said kept, the owner's run.upgrade lines, then what the lock
	// moved, and what to do next.
	steps := make([]pages.RunStep, 0, len(done))
	updated, failed, interrupted := 0, 0, u.Interrupted
	for _, d := range done {
		var said []string
		detail := d.Detail
		if d.Reason == machinefile.SelfUpdating {
			said = append(said, "updates itself: station checks it is there")
		} else if d.Reason == machinefile.Pinned {
			said = append(said, "pinned "+d.Line.Step.Version.String()+" by "+u.Plan.Name+": kept")
		} else if v := d.Line.Step.Version; v.Exact() {
			said = append(said, "held at "+v.String())
		} else if !v.Any() {
			said = append(said, "within "+v.String())
		}
		if len(d.Words) > 0 {
			said, detail = append(said, d.Words[:len(d.Words)-1]...), d.Words[len(d.Words)-1]
		}
		method := d.Method
		if method == "" {
			method = d.Line.Method()
		}
		steps = append(steps, pages.RunStep{ID: d.Line.Step.Package, State: d.State, Cells: []page.Value{page.Version(d.Version), page.Texts(method)}, Why: said, Detail: detail, Log: d.Log})
		switch {
		case d.Reason == machinefile.Interrupted:
			interrupted = true
		case d.State == catalog.StateBroken:
			failed++
		case d.Updated && d.State == catalog.StateOK && (d.Version != d.Line.Seen.Version || d.Line.Step.Within() == ""):
			updated++
		}
	}
	for _, s := range ranStep(u.Ran) {
		steps = append(steps, s)
		if s.State == catalog.StateBroken {
			failed++
		}
	}
	r := pages.Run{Command: "upgrade", Columns: []page.Column{{Key: "name", Kind: page.KindText, Keep: 100}, {Key: "version", Kind: page.KindVersion, Keep: 50}, {Key: "method", Kind: page.KindList, Keep: 30}}, Steps: steps, Empty: "apps there in " + u.Plan.Name, Extra: slices.Concat(unpulledSection(u.Unpulled), lockSection(u.Lock), waitingSection(u.Waiting))}
	switch {
	case interrupted:
		r.Head = &pages.RunHead{Label: u.Plan.Name + " interrupted", Mark: page.Broken}
		r.Next = []page.Hint{{Command: "station upgrade", Text: "resumes"}}
	case failed > 0:
		r.Head = &pages.RunHead{Label: counted(failed, "of "+strconv.Itoa(len(steps))+" failed upgrading "+u.Plan.Name), Mark: page.Broken}
		r.Next = []page.Hint{{Command: "station upgrade", Text: "tries what failed again"}}
	default:
		r.Head = &pages.RunHead{Label: "Upgraded " + u.Plan.Name, Mark: page.OK}
	}
	if u.Unbound {
		r.Next = append(r.Next, page.Hint{Command: "station apply --machinefile " + u.Plan.Name, Text: "binds this machine: no night upgrades it until then"})
	}
	words := []string{counted(updated, "updated")}
	if u.Lock.Empty() {
		words = append(words, "lock unchanged")
	} else {
		words = append(words, counted(len(u.Lock.Apps), "moved in "+u.Plan.Lock.File))
	}
	r.Head.Words = words
	return pages.RunPage(r, thisMac)
}

type LockChangeJSON struct {
	App string `json:"app"`
	// value: the app's line before and after, `<version> by <method>`;
	// empty where the lock held none.
	Was string `json:"was"`
	Now string `json:"now"`
}

type UpgradedJSON struct {
	Schema      string           `json:"schema"`
	Machinefile string           `json:"machinefile"`
	Lines       []LineJSON       `json:"lines"`
	Lock        []LockChangeJSON `json:"lock"`
	// value: the shared source before and after, `<name>@<commit>`, when
	// the upgrade moved it.
	Source []string `json:"source,omitempty"`
	// value: each newer version past its range, waiting for the owner.
	Waiting     []WaitingJSON `json:"waiting"`
	Run         *RunJSON      `json:"run,omitempty"`
	Interrupted bool          `json:"interrupted"`
	// value: why the shared source was not pulled, when it was not.
	Unpulled string `json:"unpulled,omitempty"`
}

type WaitingJSON struct {
	App    string `json:"app"`
	Newest string `json:"newest"`
	Range  string `json:"range"`
	Moved  string `json:"moved"`
	File   string `json:"file,omitempty"`
}

func UpgradedContract(schema string, u machinefile.Upgraded, done []Worded) UpgradedJSON {
	// value: station.machinefile.upgrade/v2: each app as the upgrade left
	// it, and what moved in the lock.
	out := UpgradedJSON{Schema: schema, Machinefile: u.Plan.Name, Lines: []LineJSON{}, Lock: []LockChangeJSON{}, Waiting: []WaitingJSON{}, Run: ranJSON(u.Ran), Interrupted: u.Interrupted, Unpulled: u.Unpulled}
	for _, w := range u.Waiting {
		out.Waiting = append(out.Waiting, WaitingJSON(w))
	}
	for _, d := range done {
		l := lineJSON(d.Line, string(d.State), d.Detail)
		if d.Method != "" {
			l.Method = d.Method
		}
		if d.Reason == machinefile.Pinned {
			l.Action = string(machinefile.Keep)
		}
		out.Lines = append(out.Lines, l)
	}
	said := func(a *catalog.LockedApp) string {
		if a == nil {
			return ""
		}
		return lockedWords(a)
	}
	for _, c := range u.Lock.Apps {
		out.Lock = append(out.Lock, LockChangeJSON{App: c.App, Was: said(c.Was), Now: said(c.Now)})
	}
	if u.Lock.Source != nil {
		out.Source = []string{u.Lock.Source[0], u.Lock.Source[1]}
	}
	return out
}
