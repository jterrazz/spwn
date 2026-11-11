package keepawake

import (
	"strconv"
	"time"

	"github.com/jterrazz/studio-station/src/awake"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/pages"
)

const (
	KeyAwake = "a"
	KeySleep = "s"
	RowAwake = "awake"
)

func columns() []page.Column {
	return []page.Column{
		{Key: "name", Kind: page.KindText, Keep: 100},
		{Key: "value", Kind: page.KindText, Keep: 100},
		{Key: "detail", Kind: page.KindText, Keep: 60},
		{Key: "more", Kind: page.KindText, Keep: 20, Whole: true},
	}
}

func clock(t time.Time, zone *time.Location) string {
	if zone == nil {
		zone = time.UTC
	}
	return t.In(zone).Format("15:04")
}

func left(r awake.Report) string { return pages.Term(r.Session.Until.Sub(r.Now)) + " left" }

func flags(s awake.Session) []string {
	var out []string
	if s.Lid {
		out = append(out, "lid")
	}
	if s.Battery {
		out = append(out, "battery")
	}
	return out
}

func until(s awake.Session, zone *time.Location) string {
	return "until " + clock(s.Until, zone) + " (" + pages.Term(s.Until.Sub(s.StartedAt)) + ")"
}

func ended(e *awake.End, zone *time.Location) page.Value {
	if e == nil {
		return page.Text("the Mac sleeps as macOS decides")
	}
	return page.Texts("ended "+clock(e.At, zone), e.Detail)
}

func stopped(r awake.Report) bool {
	return r.Engine == awake.EngineStopped || r.Engine == awake.EngineAbsent
}

func engineHint() *page.Hint { return page.Cmd("station engine start") }

func sessionRows(r awake.Report) []page.Row {
	switch {
	case r.Engine == awake.EnginePending:
		return []page.Row{{ID: RowAwake, Label: RowAwake, Mark: page.Awaiting, Cells: []page.Value{page.Pending()}}}
	case stopped(r):
		return []page.Row{{ID: RowAwake, Label: RowAwake, Mark: page.Attention, Reason: page.Stopped, Cells: []page.Value{page.Unknown(), page.Text("engine " + r.Engine)}, Hint: engineHint()}}
	case r.Session == nil:
		return []page.Row{{ID: RowAwake, Label: RowAwake, Mark: page.OK, Cells: []page.Value{page.Off(), ended(r.LastEnd, r.Zone)}, Hint: page.Cmd("station awake 4h")}}
	}
	s := *r.Session
	rows := []page.Row{{ID: RowAwake, Label: RowAwake, Mark: page.OK, Cells: []page.Value{
		page.On(), page.Texts(append(flags(s), until(s, r.Zone))...), page.Text("started " + clock(s.StartedAt, r.Zone) + " by " + s.By),
	}}}
	if s.Lid {
		rows = append(rows,
			page.Row{ID: "display", Label: "display", Mark: page.OK, Cells: []page.Value{page.Text("sleeps"), page.Text(catalog.LidDisplay)}},
			page.Row{ID: "clamshell", Label: "clamshell", Mark: page.OK, Cells: []page.Value{page.Text("held"), page.Text("back to normal at the end, at every engine start and quit")}},
		)
	} else {
		rows = append(rows, page.Row{ID: "display", Label: "display", Mark: page.OK, Cells: []page.Value{page.Text("as set"), page.Text("the display sleeps on its own timer (no display assertion)")}})
	}
	return rows
}

func near(r awake.Report, name string) bool {
	for _, g := range r.Guards {
		if g.Name == name {
			return g.State == awake.GuardNear
		}
	}
	return false
}

func source(rd awake.Reading) string {
	if rd.Source == "battery" {
		return "on battery"
	}
	return "AC attached"
}

func label(name string) string {
	if name == awake.ReasonLowPower {
		return "low power mode"
	}
	return name
}

type guard struct {
	name         string
	value        page.Value
	far, closeTo string
}

func guards(r awake.Report) []guard {
	rd := r.Reading
	battery := page.Text("no battery")
	if rd.HasBattery {
		battery = page.Texts(strconv.Itoa(rd.Percent)+"%", source(rd))
	}
	lid := page.Text(rd.Lid)
	switch {
	case rd.Lid == "":
		lid = page.Text("no lid")
	case rd.Lid == "closed" && rd.Source == "battery" && rd.HasBattery:
		lid = page.Text("closed on battery")
	}
	low := page.Off()
	if rd.LowPower {
		low = page.On()
	}
	return []guard{
		{awake.ReasonBattery, battery, "ends a session at 20% on battery", "the session ends at 20%"},
		{awake.ReasonLowPower, low, "ends a session when it turns on", ""},
		{awake.ReasonThermal, page.Text(awake.Level(rd.Thermal)), "ends a session at serious, or heavy for 5m lid closed on battery", "the session ends at serious"},
		{awake.ReasonLid, lid, "closing it on battery ends a --lid session without --battery", "allowed for this session"},
	}
}

func guardRow(r awake.Report, g guard, records bool) page.Row {
	row := page.Row{ID: g.name, Label: label(g.name), Mark: page.OK, Cells: []page.Value{g.value}}
	if near(r, g.name) {
		row.Mark, row.Reason = page.Attention, page.Guard
		row.Cells = append(row.Cells, page.Text(g.closeTo))
		return row
	}
	if !records {
		row.Cells = append(row.Cells, page.Text(g.far))
	}
	return row
}

func reading(rd awake.Reading) []page.Item {
	lid := rd.Lid
	if lid == "" {
		lid = "none"
	}
	src := page.Text("AC")
	if rd.Source == "battery" {
		src = page.Text("battery")
	}
	if rd.HasBattery {
		charging := "not charging"
		if rd.Charging {
			charging = "charging"
		}
		src = page.Texts(src.Text, strconv.Itoa(rd.Percent)+"%", charging)
	}
	low := page.Off()
	if rd.LowPower {
		low = page.On()
	}
	facts := []page.Fact{
		{Key: "lid", Value: page.Text(lid)},
		{Key: "source", Value: src},
		{Key: "low power mode", Value: low},
		{Key: "thermal", Value: page.Text(awake.Level(rd.Thermal) + " (" + strconv.Itoa(rd.Thermal) + ")")},
	}
	items := make([]page.Item, 0, len(facts))
	for i := range facts {
		items = append(items, page.Item{Fact: &facts[i]})
	}
	return items
}

func items(rows []page.Row) []page.Item {
	out := make([]page.Item, 0, len(rows))
	for i := range rows {
		out = append(out, page.Item{Row: &rows[i]})
	}
	return out
}

func Session(r awake.Report) page.Section {
	s := page.Section{ID: "keep-awake", Title: "Awake", Columns: columns(), Items: items(sessionRows(r))}
	if r.Session != nil && r.Engine == awake.EngineRunning {
		s.Role = page.Banner
		s.Caption = []page.Value{page.Texts(append(append([]string{"awake"}, flags(*r.Session)...), left(r))...)}
	}
	return s
}

func Page(r awake.Report, thisMac page.Context) page.Page {
	gs := guards(r)
	rows := make([]page.Row, 0, len(gs))
	for _, g := range gs {
		rows = append(rows, guardRow(r, g, false))
	}
	p := page.Page{ID: "keep-awake", Command: "awake", Context: thisMac, Sections: []page.Section{
		Session(r),
		{ID: "guards", Title: "Guards", Columns: columns(), Items: items(rows)},
		{ID: "reading", Title: "Reading", Uncounted: true, Columns: columns(), Items: reading(r.Reading)},
	}}
	if r.Engine == awake.EnginePending {
		p.State = page.Loading
	}
	p.Actions = Actions()
	return page.Seal(p)
}

func Actions() []page.Action {
	return []page.Action{
		{Key: KeyAwake, Label: "awake", On: page.OnPage, Priority: 1, Mutates: true},
		{Key: KeySleep, Label: "sleep", On: page.OnPage, Priority: 2, Mutates: true},
	}
}

func Awake(a awake.Awoken, zone *time.Location, thisMac page.Context) page.Page {
	r := pages.Done(RowAwake, catalog.StateOK, "")
	words := []string{"on"}
	if a.Refreshed {
		words = []string{"refreshed"}
	} else {
		words = append(words, flags(a.Session)...)
	}
	r.Detail = page.Texts(append(words, until(a.Session, zone))...)
	return pages.Outcome("awake", thisMac, nil, []page.Row{r}, "Awake session")
}

func Slept(s awake.Slept, thisMac page.Context) page.Page {
	if !s.Ended {
		return pages.Outcome("awake off", thisMac, nil, nil, "Awake session to end")
	}
	r := pages.Done(RowAwake, catalog.StateOK, "")
	r.Detail = page.Texts("off", "ended after "+pages.Term(s.After))
	return pages.Outcome("awake off", thisMac, nil, []page.Row{r}, "Awake session to end")
}
