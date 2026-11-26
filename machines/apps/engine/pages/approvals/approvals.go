package approvals

import (
	"github.com/jterrazz/studio-station/apps/engine/approvals"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

const command = "approvals"

func columns() []page.Column {
	return []page.Column{
		{Key: "approval", Kind: page.KindText, Keep: 100},
		{Key: "state", Label: "state", Kind: page.KindText, Keep: 90},
		{Key: "lets", Label: "lets", Kind: page.KindText, Keep: 40, Whole: true},
	}
}

func markOf(s approvals.State) (page.Mark, page.Reason) {
	switch s {
	case approvals.Approved:
		return page.OK, ""
	case approvals.Refused:
		return page.Attention, page.Refused
	case approvals.Checking:
		return page.Awaiting, ""
	case approvals.Unknown:
	}
	return page.NotKnown, ""
}

func Page(l approvals.Listing, thisMac page.Context) page.Page {
	// invariant: one row per approval in the table's order, its state and
	// what it lets Spwn do, then the gesture of a row that owes a fix.
	section := page.Section{ID: "approvals", Title: l.Mac, Columns: columns()}
	for _, r := range l.Rows {
		mark, reason := markOf(r.State)
		row := page.Row{ID: string(r.ID), Label: r.ID.Word(), Mark: mark, Reason: reason,
			Cells: []page.Value{page.Text(r.State.Word()), page.Text(r.Sentence)}}
		section.Items = append(section.Items, page.Item{Row: &row})
		if r.Fix.Path != nil {
			section.Items = append(section.Items, page.Item{Note: &page.Note{Text: *r.Fix.Path}})
		}
	}
	return page.Seal(page.Page{ID: command, Command: command, Context: thisMac, Sections: []page.Section{section}})
}
