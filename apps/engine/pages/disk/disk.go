package disk

import (
	"strconv"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/disk"
	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/pages"
)

const (
	KeyMove  = "v"
	KeyFree  = "f"
	KeyErase = "e"
	KeySort  = "s"
	KeyOpen  = "enter"
)

func title(g disk.Group) string {
	switch g {
	case disk.Misplaced:
		return "Misplaced"
	case disk.CanRebuild:
		return "Can rebuild"
	case disk.YourData:
		return "Your data"
	case disk.NeverTouched:
		return "Never touched"
	}
	return string(g)
}

type RowJSON struct {
	App    string   `json:"app"`
	Group  string   `json:"group"`
	Bytes  int64    `json:"bytes"`
	Paths  []string `json:"paths"`
	MoveTo string   `json:"move_to,omitempty"`
}

type ReadJSON struct {
	Schema   string         `json:"schema"`
	Findings []finding.JSON `json:"findings"`
	Free     int64          `json:"free_bytes"`
	Total    int64          `json:"total_bytes"`
	Rows     []RowJSON      `json:"rows"`
}

func ReadContract(r disk.Reading) ReadJSON {
	out := ReadJSON{Schema: "station.disk.read/v1", Findings: finding.Doc(disk.Findings(r, machine.Ref{})), Free: r.Free, Total: r.Total, Rows: make([]RowJSON, 0, len(r.Rows))}
	for _, row := range r.Rows {
		out.Rows = append(out.Rows, RowJSON{App: row.App, Group: string(row.Group), Bytes: row.Bytes, Paths: row.Paths, MoveTo: row.MoveTo})
	}
	return out
}

func machineLabel(thisMac page.Context) string {
	if thisMac.Kind == string(machine.Computer) || thisMac.Machine == "" {
		return "This Mac"
	}
	return thisMac.Machine
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return strconv.Itoa(n) + " " + word + "s"
}

// value: the Can rebuild rows a terminal shows before the one that sums
// the rest (vision 07 "station disk").
const top = 3

func Read(r disk.Reading, thisMac page.Context, all bool) page.Page {
	p := page.Page{ID: "disk", Command: "disk", Context: thisMac, Actions: Actions(r)}
	misplaced := r.Of(disk.Misplaced)
	head := page.Row{ID: "volume", Label: machineLabel(thisMac), Mark: page.OK, Cells: []page.Value{
		page.List(page.SizeOf(r.Free, r.Total, "free"), page.Text(plural(len(misplaced), "misplaced folder"))),
	}}
	if len(misplaced) > 0 {
		head.Mark, head.Reason = page.Attention, page.Misplaced
		head.Hint = page.Cmd("station disk move " + misplaced[0].Paths[0])
	}
	p.Sections = append(p.Sections, page.Section{ID: "volume", Title: "Disk", Role: page.Banner, Uncounted: true, Columns: columns(), Items: []page.Item{{Row: &head}}})
	for _, g := range disk.Groups() {
		rows := r.Of(g)
		if len(rows) == 0 {
			continue
		}
		items := make([]page.Item, 0, len(rows))
		for _, row := range rows {
			pr := Row(row)
			items = append(items, page.Item{Row: &pr})
		}
		p.Sections = append(p.Sections, group(g, rows, items, all))
	}
	if disk.Sum(r.Of(disk.CanRebuild)) > 0 {
		p.Hints = append(p.Hints, page.Hint{Command: "station disk free", Text: "frees every Can rebuild row, nothing is lost"})
	}
	return page.Seal(p)
}

// invariant: a group with nothing to do is one line on a terminal (SPEC
// §5.5, vision 07 "station disk": `Your data · 41.6 GB · xcode 19.1, …`);
// Can rebuild, which frees, shows its largest rows then one for the rest;
// Misplaced is open; --all unfolds every group.
func group(g disk.Group, rows []disk.Row, items []page.Item, all bool) page.Section {
	s := page.Section{ID: string(g), Title: title(g), Fold: page.Open, Columns: columns(), Items: items, Uncounted: true}
	if g == disk.Misplaced || all {
		return s
	}
	s.Caption, s.Joined = []page.Value{page.Size(disk.Sum(rows))}, true
	if g != disk.CanRebuild {
		s.Fold = page.Brief
		return s
	}
	if len(rows) > top+1 {
		rest := rows[top:]
		s.Top = top
		s.Rest = &page.Item{Row: &page.Row{ID: string(g) + "/more", Label: "+ " + strconv.Itoa(len(rest)) + " more", Cells: []page.Value{page.Size(disk.Sum(rest))}}}
	}
	return s
}

func sized(n int64, unit string) page.Value {
	v := page.Size(n)
	v.Unit = unit
	return v
}

func columns() []page.Column {
	return []page.Column{
		{Key: "name", Kind: page.KindText, Keep: 100},
		{Key: "size", Label: "size", Kind: page.KindSize, Keep: 90},
		{Key: "folders", Label: "folders", Kind: page.KindList, Show: page.DetailOnly},
	}
}

func Row(r disk.Row) page.Row {
	cell := page.Size(r.Bytes)
	if r.Bytes < 0 {
		cell = page.Unknown()
	}
	row := page.Row{ID: string(r.Group) + "/" + r.App, Label: r.App, Mark: page.OK, Cells: []page.Value{cell, page.Texts(r.Paths...)}}
	if r.Group == disk.Misplaced {
		row.ID, row.Label = string(disk.Misplaced)+"/"+r.Paths[0], r.Paths[0]
		row.Mark, row.Reason = page.Attention, page.Misplaced
		row.Detail = page.Texts("misplaced", r.App)
		row.Hint = page.Cmd("station disk move " + r.Paths[0])
	}
	if r.Pending {
		// invariant: a row the walk has not answered shows its last-known
		// size, Muted, never a blank that a fill would move.
		row.Mark = page.Awaiting
		row.Cells = []page.Value{page.Stale(page.Size(r.Bytes)), page.Texts(r.Paths...)}
	}
	return row
}

func Actions(r disk.Reading) []page.Action {
	var move, free, erase, open []string
	for _, row := range r.Rows {
		if len(row.Paths) > 0 {
			open = append(open, Row(row).ID)
		}
		switch row.Group {
		case disk.Misplaced:
			move = append(move, string(disk.Misplaced)+"/"+row.Paths[0])
		case disk.CanRebuild:
			free = append(free, string(row.Group)+"/"+row.App)
		case disk.YourData:
			erase = append(erase, string(row.Group)+"/"+row.App)
		case disk.NeverTouched:
		}
	}
	var out []page.Action
	if len(move) > 0 {
		out = append(out, page.Action{Key: KeyMove, Label: "move", On: page.OnRow, Rows: move, Priority: 1, Mutates: true})
	}
	if len(free) > 0 {
		out = append(out, page.Action{Key: KeyFree, Label: "free", On: page.OnRow, Rows: free, Priority: 2, Mutates: true})
	}
	if len(erase) > 0 {
		out = append(out, page.Action{Key: KeyErase, Label: "erase", On: page.OnRow, Rows: erase, Priority: 3, Mutates: true, Confirm: &page.Confirm{Yes: "erase", No: "Keep", Danger: true}})
	}
	if len(open) > 0 {
		out = append(out, page.Action{Key: KeyOpen, Label: "open", On: page.OnRow, Rows: open, Priority: 4})
	}
	if len(r.Rows) > 1 {
		out = append(out, page.Action{Key: KeySort, Label: "sort", On: page.OnPage, Priority: 5})
	}
	return out
}

func Freed(outcomes []disk.Outcome, thisMac page.Context) page.Page {
	rows := make([]page.Row, 0, len(outcomes))
	var total int64
	for _, o := range outcomes {
		r := pages.Done(o.Name, o.Result, o.Reason)
		r.Cells = []page.Value{detail(o, "freed")}
		rows = append(rows, r)
		total += o.Freed
	}
	var extra []page.Section
	if len(outcomes) > 0 {
		sum := page.Row{ID: "freed", Label: "Freed", Mark: page.OK, Cells: []page.Value{page.Size(total)}, Detail: page.Text("nothing is lost")}
		extra = append(extra, page.Section{ID: "freed", Title: "Freed", Role: page.Banner, Uncounted: true, Columns: outcomeColumns(), Items: []page.Item{{Row: &sum}}})
	}
	return pages.Outcome("disk free", thisMac, outcomeColumns(), rows, "files to free", extra...)
}

func Erased(o disk.Outcome, thisMac page.Context) page.Page {
	r := pages.Done(o.Name, o.Result, o.Reason)
	r.Cells = []page.Value{detail(o, "erased")}
	return pages.Outcome("disk erase", thisMac, outcomeColumns(), []page.Row{r}, "data to erase")
}

func Moved(o disk.Outcome, thisMac page.Context) page.Page {
	r := pages.Done(o.Name, o.Result, "")
	r.Cells = []page.Value{page.Text(o.Detail)}
	return pages.Outcome("disk move", thisMac, outcomeColumns(), []page.Row{r}, "folders to move")
}

func outcomeColumns() []page.Column {
	return []page.Column{{Key: "name", Kind: page.KindText, Keep: 100}, {Key: "outcome", Kind: page.KindText, Keep: 50}}
}

func detail(o disk.Outcome, verb string) page.Value {
	if o.Result != catalog.StateOK {
		return page.Text(o.Detail)
	}
	return sized(o.Freed, verb)
}

type MisplacedJSON struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Bytes  int64  `json:"bytes"`
	Repair string `json:"repair"`
}

func MisplacedContract(misplaced []disk.MisplacedReport) []MisplacedJSON {
	out := make([]MisplacedJSON, 0, len(misplaced))
	for _, s := range misplaced {
		out = append(out, MisplacedJSON{Name: s.Name, Path: s.Path, Bytes: s.Bytes, Repair: s.Repair})
	}
	return out
}
