package apps

import (
	"strconv"

	"github.com/jterrazz/studio-station/apps/engine/apps"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

const (
	KeyInstall = "i"
	KeyRemove  = "x"
	KeyUpdate  = "u"
	// value: the stronger update: every update of one method, picked.
	KeyUpdateByMethod = "U"
	KeyOpen           = "enter"
)

type group struct {
	id, title string
	mark      page.Mark
}

func groupOf(s apps.State) group {
	switch s {
	case apps.UpdateAvailable:
		return group{"updates", "Updates", page.Attention}
	case apps.Failed:
		return group{"failed", "Failed", page.Broken}
	case apps.NotInstalled:
		return group{"not-installed", "Not installed", page.Absent}
	case apps.NotManaged:
		return group{"not-managed", "Not managed", page.NotKnown}
	case apps.OK:
		return group{"ok", "OK", page.OK}
	case apps.Pending:
		return group{"reading", "Reading", page.Awaiting}
	case apps.NotKnown:
	}
	return group{"not-known", "Not known", page.NotKnown}
}

func List(reports []apps.Report, known []catalog.App, thisMac page.Context, all bool) page.Page {
	// invariant: the Apps page groups by attention (SPEC §5.3): Updates and
	// Failed open, then Not installed, Not managed and OK folded; what lives
	// elsewhere is a count, unfolded with --all.
	p := page.Page{
		ID:      "apps",
		Command: "apps",
		Context: thisMac,
		Hints:   next(reports, all),
		Actions: Actions(),
	}
	if len(reports) == 0 {
		p.Sections = []page.Section{{ID: "apps", Title: "Apps", Role: page.Banner, Columns: Columns(), Items: []page.Item{{Empty: &page.Empty{Noun: "apps"}}}}}
		return page.Seal(p)
	}
	byState := map[apps.State][]page.Item{}
	for _, r := range reports {
		row := Row(r)
		byState[apps.StateOf(r)] = append(byState[apps.StateOf(r)], page.Item{Row: &row})
	}
	p.Sections = append(p.Sections, summary(reports, thisMac))
	if items := byState[apps.Pending]; len(items) > 0 {
		p.Sections = append(p.Sections, page.Section{ID: "reading", Title: "Reading", Fold: page.Open, Columns: Columns(), Items: items})
	}
	for _, state := range apps.Groups() {
		items := byState[state]
		if len(items) == 0 {
			continue
		}
		g := groupOf(state)
		fold := page.Open
		switch {
		case all:
		case state == apps.OK:
			fold = page.Folded
		case state == apps.NotInstalled, state == apps.NotManaged:
			fold = page.Brief
		}
		p.Sections = append(p.Sections, page.Section{ID: g.id, Title: g.title, Fold: fold, Columns: Columns(), Items: items, PipeCounts: true})
	}
	if elsewhere := apps.Elsewhere(known, reports); len(elsewhere) > 0 {
		s := page.Section{ID: "not-for-this-machine", Title: "Not for this machine", Fold: page.Open, Columns: Columns(), PipeCounts: true}
		if all {
			for _, t := range elsewhere {
				row := page.Row{ID: t.Name, Label: t.Name, Mark: page.NotForThisMachine, Detail: page.Text(apps.LivesOn(t.LivesOn()))}
				s.Items = append(s.Items, page.Item{Row: &row})
			}
		} else {
			s.Uncounted = true
			s.Items = []page.Item{{Note: &page.Note{Text: plural(len(elsewhere), "app") + " live on other machines"}}}
		}
		p.Sections = append(p.Sections, s)
	}
	return page.Seal(p)
}

// invariant: one or two next steps (vision 07 "station apps"): the updates
// that wait, then installing what is not installed, else every row.
func next(reports []apps.Report, all bool) []page.Hint {
	counts := apps.Counts(reports)
	var hints []page.Hint
	if n := counts[apps.UpdateAvailable]; n > 0 {
		text := "the update"
		if n > 1 {
			text = "the " + strconv.Itoa(n) + " updates"
		}
		hints = append(hints, page.Hint{Command: "station apps update", Text: text})
	}
	if counts[apps.NotInstalled] > 0 || len(reports) == 0 {
		hints = append(hints, page.Hint{Command: "station apps install <app...>", Text: "install some"})
	}
	if len(hints) < 2 && !all && len(reports) > 0 {
		hints = append(hints, page.Hint{Command: "station apps --all", Text: "for every row"})
	}
	return hints
}

func summary(reports []apps.Report, thisMac page.Context) page.Section {
	counts := apps.Counts(reports)
	label := machineLabel(thisMac)
	whose := label
	if label == "This Mac" {
		whose = "this Mac"
	}
	words := []string{plural(len(reports), "app") + " for " + whose}
	if n := counts[apps.UpdateAvailable]; n > 0 {
		words = append(words, plural(n, "update"))
	}
	if n := counts[apps.Failed]; n > 0 {
		words = append(words, strconv.Itoa(n)+" failed")
	}
	if n := counts[apps.NotInstalled]; n > 0 {
		words = append(words, strconv.Itoa(n)+" not installed")
	}
	mark := page.OK
	switch {
	case counts[apps.Failed] > 0:
		mark = page.Broken
	case counts[apps.UpdateAvailable] > 0:
		mark = page.Attention
	}
	row := page.Row{ID: "summary/apps", Label: label, Mark: mark, Cells: []page.Value{page.Texts(words...)}}
	return page.Section{ID: "summary", Title: "Summary", Role: page.Summary, Fold: page.Open, Uncounted: true, Columns: []page.Column{{Key: "area", Kind: page.KindText, Keep: 100}, {Key: "value", Kind: page.KindList, Keep: 60}}, Items: []page.Item{{Row: &row}}}
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return strconv.Itoa(n) + " " + word + "s"
}

func Columns() []page.Column {
	return []page.Column{
		{Key: "name", Kind: page.KindText, Keep: 100},
		{Key: "version", Kind: page.KindVersion, Keep: 50},
		{Key: "method", Kind: page.KindList, Keep: 30},
	}
}

func Actions() []page.Action {
	return []page.Action{
		{Key: KeyInstall, Label: "install", On: page.OnRow, Priority: 1, Mutates: true, Confirm: &page.Confirm{Yes: "install"}},
		{Key: KeyRemove, Label: "remove", On: page.OnRow, Priority: 2, Mutates: true, Confirm: &page.Confirm{Yes: "remove"}},
		{Key: KeyUpdate, Label: "update", On: page.OnRow, Priority: 3, Mutates: true},
		{Key: KeyUpdateByMethod, Label: "update by method", On: page.OnPage, Priority: 4, Mutates: true},
		{Key: KeyOpen, Label: "full page", On: page.OnRow, Priority: 5},
	}
}

func Row(r apps.Report) page.Row {
	name := r.App.Name
	row := page.Row{ID: name, Label: name, Cells: []page.Value{version(r), method(r)}}
	state := apps.StateOf(r)
	if state == apps.Pending {
		row.Mark, row.Cells[0] = page.Awaiting, page.Pending()
		return row
	}
	detail := r.Detail
	row.Mark = groupOf(state).mark
	switch state {
	case apps.OK:
		if detail == "" {
			detail = r.Live
		}
	case apps.UpdateAvailable:
		row.Reason, row.Hint, detail = page.Outdated, page.Cmd("station apps update "+name), "update available"
	case apps.NotInstalled:
		if r.Why == apps.WhyMissing {
			row.Reason, row.Hint = page.Missing, page.Cmd("station apps install "+r.Needs)
		} else {
			row.Hint = page.Cmd("station apps install " + name)
		}
	case apps.NotManaged:
		row.Reason = reason(r.Why)
		if r.Why == apps.WhyMissing {
			row.Hint = page.Cmd("station apps install " + r.Needs)
		}
		if detail == "" {
			detail = "not installed by " + string(r.Method)
		}
	case apps.NotKnown:
		row.Reason, row.Cells[0] = page.Late, page.Unknown()
	case apps.Failed:
		row.Reason = page.Failing
	case apps.Pending:
	}
	row.Detail = page.Text(detail)
	return row
}

func reason(w apps.Why) page.Reason {
	switch w {
	case apps.WhyMissing:
		return page.Missing
	case apps.WhyPartial:
		return page.Partial
	case apps.WhyFailing, apps.WhyInterrupted:
		return page.Failing
	case apps.WhyOutdated, apps.WhyUnmanaged, apps.WhyLate:
	}
	return page.Unmanaged
}

func version(r apps.Report) page.Value {
	if apps.StateOf(r) == apps.UpdateAvailable {
		return page.Upgrade(r.Version, r.Newest)
	}
	return page.Version(r.Version)
}

func method(r apps.Report) page.Value { return page.Texts(string(r.Method)) }
