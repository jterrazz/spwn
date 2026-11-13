package pages

import (
	"strconv"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

func MarkOf(s catalog.State) page.Mark {
	switch s {
	case catalog.StateOK:
		return page.OK
	case catalog.StateDegraded, catalog.StateUnknown:
		return page.Attention
	case catalog.StateAbsent:
		return page.Absent
	case catalog.StateBroken:
		return page.Broken
	case catalog.StatePending:
		return page.Awaiting
	}
	return page.Unjudged
}

func Failure(id page.ID, command string, thisMac page.Context, message, hint string) page.Page {
	return page.Seal(page.Page{ID: id, Command: command, Context: thisMac, Error: &page.Error{Message: message, Hint: hint}})
}

func Outcome(command string, thisMac page.Context, columns []page.Column, rows []page.Row, empty string, extra ...page.Section) page.Page {
	items := make([]page.Item, 0, len(rows))
	for i := range rows {
		items = append(items, page.Item{Row: &rows[i]})
	}
	if len(items) == 0 {
		items = append(items, page.Item{Empty: &page.Empty{Noun: empty}})
	}
	if len(columns) == 0 {
		columns = OutcomeColumns()
	}
	sections := append([]page.Section{{ID: "changes", Title: "Changes", Role: page.Banner, Columns: columns, Items: items}}, extra...)
	return page.Seal(page.Page{ID: page.Outcome, Command: command, Context: thisMac, Sections: sections})
}

func OutcomeColumns() []page.Column {
	return []page.Column{{Key: "name", Kind: page.KindText, Keep: 100}}
}

func Done(id string, state catalog.State, detail string) page.Row {
	r := page.Row{ID: id, Label: id, Mark: MarkOf(state), Detail: page.Text(detail)}
	switch r.Mark {
	case page.Attention:
		r.Reason = page.Partial
	case page.Broken:
		r.Reason = page.Failing
	case page.OK, page.Absent, page.Awaiting, page.OffHere, page.NotForThisMachine, page.NotKnown, page.Unjudged:
	}
	return r
}

func Nest(s page.Section, parts []page.Section) page.Section {
	// invariant: a page's sub-lists (Secrets' Lend list and Kept, A19;
	// Tailnet's Peers and Serve on the Fleet tab) are heading groups of ONE
	// section, so folding the section folds them.
	s.Fold = page.Auto
	if len(parts) > 0 && len(s.Columns) < len(parts[0].Columns) {
		s.Columns = parts[0].Columns
	}
	for _, sub := range parts {
		items := make([]page.Item, 0, len(sub.Items))
		for _, it := range sub.Items {
			if it.Row != nil {
				row := *it.Row
				row.ID = sub.ID + "/" + row.ID
				it.Row = &row
			}
			items = append(items, it)
		}
		s.Items = append(s.Items, page.Item{Group: &page.Group{Head: page.Row{ID: sub.ID, Label: sub.Title}, Items: items, Fold: page.Open}})
	}
	return s
}

func Term(d time.Duration) string {
	// value: a term as the owner says it: whole days, hours and minutes, two
	// units at most (`12h`, `1h 30m`, `7d`); under a minute, its seconds, as
	// the duration value says them (`6s`), never `0m`.
	s := int64(d.Seconds())
	if s < 60 {
		return strconv.FormatInt(max(s, 0), 10) + "s"
	}
	parts := []string{}
	for _, u := range []struct {
		n    int64
		name string
	}{{86400, "d"}, {3600, "h"}, {60, "m"}} {
		if s >= u.n && len(parts) < 2 {
			parts = append(parts, strconv.FormatInt(s/u.n, 10)+u.name)
			s %= u.n
		}
	}
	if len(parts) == 2 {
		return parts[0] + " " + parts[1]
	}
	return parts[0]
}
