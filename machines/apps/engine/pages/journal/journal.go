package journal

import (
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/journal"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

func columns() []page.Column {
	return []page.Column{
		{Key: "name", Kind: page.KindText, Keep: 100},
		{Key: "time", Label: "time", Kind: page.KindText, Keep: 30},
		{Key: "machines", Label: "on", Kind: page.KindText, Keep: 60},
		{Key: "by", Label: "by", Kind: page.KindText, Keep: 40},
	}
}

func markOf(o journal.Outcome) (page.Mark, page.Reason) {
	switch o {
	case journal.Failed:
		return page.Broken, page.Failing
	case journal.Held:
		return page.Attention, page.Waiting
	case journal.Done:
	}
	return page.OK, ""
}

func Read(r journal.Reading, thisMac page.Context) page.Page {
	// invariant: one section per group of the read, newest first, its sum
	// under its title; the Macs that did not answer after them; the engine's
	// sentence when there is nothing to show.
	sections := make([]page.Section, 0, len(r.Groups)+1)
	for _, g := range r.Groups {
		s := page.Section{ID: g.Day, Title: g.Title, Columns: columns()}
		if g.Night {
			s.ID = g.Day + "-night"
		}
		if g.Sum != "" {
			s.Caption = []page.Value{page.Text(g.Sum)}
		}
		seen := map[string]int{}
		for _, l := range g.Lines {
			detail := page.Text(l.Detail)
			if l.Upgrade != nil && l.Detail == "" {
				// invariant: a moved line says both versions, the pipe too.
				detail = page.Text(l.Upgrade.From + " to " + l.Upgrade.To)
			}
			id := l.At.Format(time.RFC3339) + "/" + l.Text + "/" + strings.Join(l.On, ",")
			seen[id]++
			if seen[id] > 1 {
				id += "/" + strconv.Itoa(seen[id])
			}
			mark, reason := markOf(l.Outcome)
			if reason != "" && l.Detail == "" && l.Upgrade == nil {
				detail = page.Text(string(l.Outcome))
			}
			row := page.Row{ID: id, Label: l.Text, Mark: mark, Reason: reason, Detail: detail,
				Cells: []page.Value{page.Text(l.At.Format("15:04")), page.Text(strings.Join(l.On, ", ")), page.Text(l.By)}}
			s.Items = append(s.Items, page.Item{Row: &row})
		}
		sections = append(sections, s)
	}
	if len(r.Unheard) > 0 {
		s := page.Section{ID: "unheard", Title: "Not heard from", Columns: columns()[:1]}
		for _, u := range r.Unheard {
			row := page.Row{ID: u.Mac, Label: u.Mac, Mark: page.NotKnown, Detail: page.Text(u.Why)}
			s.Items = append(s.Items, page.Item{Row: &row})
		}
		sections = append(sections, s)
	}
	if r.Empty != "" {
		sections = append(sections, page.Section{ID: "journal", Title: "Journal", Items: []page.Item{{Empty: &page.Empty{Text: r.Empty}}}})
	}
	return page.Seal(page.Page{ID: "journal", Command: "journal", Context: thisMac, Sections: sections})
}
