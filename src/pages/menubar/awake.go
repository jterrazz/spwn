package menubar

import (
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/awake"
	"github.com/jterrazz/studio-station/src/page"
	pageawake "github.com/jterrazz/studio-station/src/pages/awake"
)

const (
	AwakeID page.ID = "menubar.awake"

	ActionKeep    = "keep:"
	ActionUntil   = "keep-until"
	ActionKeepOn  = "keep-on"
	ActionLid     = "lid"
	ActionBattery = "battery"
	ActionLetGo   = "let-sleep"
	ActionChange  = "change:"

	CallSettingsOff   = "settings.off"
	CallSettingsApply = "settings.apply"

	SectionThisMac = "this-mac"
	SectionPauses  = "pauses"

	RowLid     = "lid"
	RowBattery = "battery"

	neverSleeps = "never-sleeps"
)

// value: the terms the picker offers, then how many whole hours Until
// offers from the next one, and the end it offers first: the end of a
// working day (SPEC A9 2-11: `1 hr · 4 hr · Until 18:00`).
func terms() []time.Duration { return []time.Duration{time.Hour, 4 * time.Hour} }

const (
	untilHours = 8
	untilHour  = 18
)

func Awake(f pageawake.AllMachines, ctx page.Context) page.Page {
	// invariant: SPEC A4 Awake · All Macs: This Mac first, its switch, the
	// picker, Lid closed and On battery only where its capabilities have
	// them (E1), then the running work that pauses while it sleeps; then
	// each other Mac on its own card by its policy and what depends on it,
	// with Change… on its switch. No footer (judges r4 #11).
	p := page.Page{ID: AwakeID, Command: "awake", Context: ctx, Header: &page.Header{Back: "All machines", Glyph: symbol("sun.max", page.SignalQuiet)}}
	p.Sections = []page.Section{{ID: SectionSummary, Title: "Awake", Role: page.Summary, Caption: []page.Value{page.Text("All Macs"), page.Text("which may sleep, and what pauses")}}}
	if f.Label == nil {
		self := ""
		for _, r := range f.Reports {
			if r.ThisMac {
				self = r.Mac
			}
		}
		f.Label = func(mac string) string { return page.MacName(mac, self) }
	}
	for _, r := range f.Reports {
		if r.ThisMac {
			p.Sections = append(p.Sections, thisMac(r))
			if s, ok := pausing(r); ok {
				p.Sections = append(p.Sections, s)
			}
			p.Actions = append(p.Actions, keeps(r, f.Now, f.Zone)...)
			continue
		}
		name := label(f, r.Mac)
		a, ok := change(r, name)
		row := otherMac(r, f.Zone)
		if ok && r.Mode() != awake.AwakeUntil {
			row.Verb = a.ID
			p.Actions = append(p.Actions, a)
		}
		p.Sections = append(p.Sections, macCard(r.Mac, name, row))
	}
	for _, s := range f.Silent {
		p.Sections = append(p.Sections, macCard(s.Mac, label(f, s.Mac), silentMac(s, f.Now)))
	}
	for _, r := range f.Reports {
		if r.Engine == awake.EngineReading {
			p.State = page.Loading
		}
	}
	return page.Seal(p)
}

func label(f pageawake.AllMachines, mac string) string {
	if f.Label != nil {
		return f.Label(mac)
	}
	return mac
}

func awakeColumns() []page.Column {
	return []page.Column{{Key: "name", Kind: page.KindText, Keep: 100}, {Key: "state", Kind: page.KindText, Keep: 100}}
}

func switched(on bool) []page.Value {
	if on {
		return []page.Value{page.On()}
	}
	return []page.Value{page.Off()}
}

// value: a Mac's sun while it stays awake, its moon while it may sleep
// (SPEC M14).
func sun(on bool) page.Glyph {
	if on {
		return symbol("sun.max", page.SignalWaits)
	}
	return symbol("moon", page.SignalQuiet)
}

func macCard(mac, name string, row page.Row) page.Section {
	return page.Section{ID: mac, Title: name, Card: page.CardFlush, Uncounted: true, Columns: awakeColumns(), Items: []page.Item{{Row: &row}}}
}

// value: This Mac's card: Keep This Mac awake and its switch, the line
// that says the run it follows, then the switches its capabilities allow.
func thisMac(r awake.Report) page.Section {
	s := page.Section{ID: SectionThisMac, Title: page.ThisMac, Card: page.CardFlush, Uncounted: true, Columns: awakeColumns()}
	held := r.Session != nil && r.Mode() == awake.AwakeUntil
	keep := page.Row{ID: r.Mac, Label: "Keep This Mac awake", Glyph: sun(held), Cells: switched(held), Detail: page.Text("Stays on after agents finish"), Mark: page.OK}
	switch {
	case r.Engine == awake.EngineReading:
		keep.Mark, keep.Detail = page.Awaiting, page.Pending()
	case stoppedEngine(r):
		keep.Mark, keep.Detail = page.Absent, page.Text("the engine is stopped")
		keep.Hint = page.Cmd("station engine start")
	case held:
		keep.Verb = ActionLetGo
	default:
		keep.Verb = ActionKeepOn
	}
	s.Items = append(s.Items, page.Item{Row: &keep})
	if held && r.Session.Run != "" {
		s.Items = append(s.Items, page.Item{Note: &page.Note{Text: "Until " + r.Session.Run + " is done"}})
	}
	lid, battery := held && r.Session.Lid, held && r.Session.Battery
	if r.Caps.Lid {
		row := page.Row{ID: RowLid, Label: "Stay awake with the lid closed", Glyph: symbol("laptopcomputer", page.SignalQuiet), Cells: switched(lid), Verb: ActionLid, Mark: page.OK}
		s.Items = append(s.Items, page.Item{Row: &row})
	}
	if r.Caps.Battery {
		row := page.Row{ID: RowBattery, Label: "Stay awake on battery", Glyph: symbol("battery.75percent", page.SignalQuiet), Cells: switched(battery), Verb: ActionBattery, Mark: page.OK}
		s.Items = append(s.Items, page.Item{Row: &row})
	}
	return s
}

// value: the running work that pauses while This Mac sleeps, under its
// card; none, no line (SPEC A9 1-11).
func pausing(r awake.Report) (page.Section, bool) {
	if len(r.Pauses) == 0 {
		return page.Section{}, false
	}
	line := strings.Join(r.Pauses, " and ") + " " + pluralVerb(len(r.Pauses), "pauses", "pause") + " while This Mac sleeps."
	return page.Section{ID: SectionPauses, Title: "While This Mac sleeps", Uncounted: true, Items: []page.Item{{Note: &page.Note{Text: line}}}}, true
}

func stoppedEngine(r awake.Report) bool {
	return r.Engine != "" && r.Engine != awake.EngineRunning && r.Engine != awake.EngineReading
}

// value: what depends on a Mac: the servers it hosts, else the running
// work that pauses, else nothing.
func dependants(servers, pauses []string) string {
	switch {
	case len(servers) > 0:
		return "hosts " + strings.Join(servers, ", ") + ", which " + pluralVerb(len(servers), "pauses", "pause") + " if it sleeps"
	case len(pauses) > 0:
		return strings.Join(pauses, ", ") + " " + pluralVerb(len(pauses), "pauses", "pause") + " if it sleeps"
	}
	return "nothing depends on it"
}

// value: another Mac's card row: its policy, what depends on it, its
// switch on while it stays awake.
func otherMac(r awake.Report, zone *time.Location) page.Row {
	row := page.Row{ID: r.Mac, Label: "May sleep", Glyph: sun(false), Cells: switched(false), Mark: page.Absent, Detail: page.Text(dependants(r.Servers, r.Pauses))}
	switch {
	case r.Engine == awake.EngineReading:
		row.Label, row.Mark, row.Detail = "Checking", page.Awaiting, page.Pending()
		row.Cells = nil
	case stoppedEngine(r):
		row.Detail = page.Texts("the engine is stopped", dependants(r.Servers, r.Pauses))
		row.Hint = page.Cmd("station engine start")
	case r.Mode() == awake.NeverSleeps:
		row.Label, row.Glyph, row.Cells, row.Mark = "Never sleeps", sun(true), switched(true), page.OK
		if !r.NeverHeld {
			row.Mark, row.Reason = page.Attention, page.Guard
			row.Detail = page.Texts("let go while the battery is low", dependants(r.Servers, r.Pauses))
		}
	case r.Mode() == awake.AwakeUntil:
		if zone == nil {
			zone = time.UTC
		}
		row.Label, row.Glyph, row.Cells, row.Mark = "Awake until "+r.Session.Until.In(zone).Format("15:04"), sun(true), switched(true), page.OK
		if r.Session.Run != "" {
			row.Label = "Awake until " + r.Session.Run + " is done"
		}
	case r.Mode() == awake.StoppedByGuard:
		row.Mark, row.Reason = page.Attention, page.Guard
		row.Detail = page.Texts("Awake stopped", r.LastEnd.Detail)
	}
	return row
}

// value: a Mac that did not answer: asleep, when it was last seen and
// what depended on it then, dimmed; else not answering, and why.
func silentMac(s pageawake.Silent, now time.Time) page.Row {
	if !s.Asleep {
		return page.Row{ID: s.Mac, Label: "Not answering", Glyph: symbol("questionmark.circle", page.SignalQuiet), Mark: page.NotKnown, Reason: page.Unreachable, Detail: page.Text(s.Why)}
	}
	words := []string{"asleep"}
	if !s.Answered.IsZero() && now.After(s.Answered) {
		words = append(words, "seen "+ago(now.Sub(s.Answered))+" ago")
	}
	words = append(words, dependants(s.Servers, nil))
	return page.Row{ID: s.Mac, Label: "May sleep", Glyph: symbol("moon", page.SignalAsleep), Mark: page.Absent, Signal: page.SignalAsleep, Detail: page.Texts(words...)}
}

func nanos(d time.Duration) string { return strconv.FormatInt(int64(d), 10) }

// value: an end as the call's input takes it, a JSON time.
func moment(t time.Time) string { return `"` + t.UTC().Format(time.RFC3339) + `"` }

func flags(lid, battery bool) map[string]string {
	return map[string]string{"Lid": strconv.FormatBool(lid), "Battery": strconv.FormatBool(battery), "By": "the menu bar"}
}

// invariant: a keep-awake keeps the switches as they are; a switch keeps
// the end the keep-awake holds until, or starts the one Until offers. The
// term in force is the one picked; one that follows a run picks none.
// Until says the time it holds until when it is the term in force, else
// the one its form offers first.
func keeps(r awake.Report, now time.Time, zone *time.Location) []page.Action {
	if zone == nil {
		zone = time.UTC
	}
	var lid, battery bool
	held := r.Session != nil && r.Mode() == awake.AwakeUntil
	if held {
		lid, battery = r.Session.Lid, r.Session.Battery
	}
	follows := held && r.Session.Run != ""
	rows := []string{r.Mac}
	var out []page.Action
	picked := false
	for _, t := range terms() {
		in := flags(lid, battery)
		in["For"] = nanos(t)
		on := held && !follows && asked(*r.Session) == t
		picked = picked || on
		out = append(out, page.Action{ID: ActionKeep + strings.TrimSuffix(t.String(), "0m0s"), Label: pageTerm(t), On: page.OnRow, Rows: rows, Priority: 1, Mutates: true, Picked: on, Run: &page.Run{Call: CallKeepAwake, In: in}})
	}
	field := untilField(now, zone)
	end := field.Default
	until := "Until " + optionLabel(field, end)
	untilPicked := held && !follows && !picked
	if untilPicked {
		until = "Until " + r.Session.Until.In(zone).Format("15:04")
	}
	out = append(out, page.Action{ID: ActionUntil, Label: until, On: page.OnRow, Rows: rows, Priority: 1, Mutates: true, Picked: untilPicked,
		Form: &page.Form{Title: "Keep awake until", Fields: []page.Field{field}}, Run: &page.Run{Call: CallKeepAwake, In: flags(lid, battery)}})
	term := func(in map[string]string) map[string]string {
		if held {
			in["Until"] = moment(r.Session.Until)
		} else {
			in["Until"] = end
		}
		return in
	}
	if !held {
		out = append(out, page.Action{ID: ActionKeepOn, Label: "Keep This Mac awake", On: page.OnRow, Rows: rows, Priority: 2, Mutates: true, Run: &page.Run{Call: CallKeepAwake, In: term(flags(lid, battery))}})
	}
	if r.Caps.Lid {
		out = append(out, page.Action{ID: ActionLid, Label: "Lid closed", On: page.OnRow, Rows: rows, Priority: 2, Mutates: true, Run: &page.Run{Call: CallKeepAwake, In: term(flags(!lid, battery))}})
	}
	if r.Caps.Battery {
		out = append(out, page.Action{ID: ActionBattery, Label: "On battery", On: page.OnRow, Rows: rows, Priority: 2, Mutates: true, Run: &page.Run{Call: CallKeepAwake, In: term(flags(lid, !battery))}})
	}
	if held {
		out = append(out, page.Action{ID: ActionLetGo, Label: "Let it sleep", On: page.OnRow, Rows: rows, Priority: 3, Mutates: true, Run: &page.Run{Call: CallAwakeOff}})
	}
	return out
}

// value: the term a session was asked for: its own word, else its span
// from its start.
func asked(s awake.Session) time.Duration {
	if s.For > 0 {
		return s.For.Round(time.Minute)
	}
	return s.Until.Sub(s.StartedAt).Round(time.Minute)
}

func pageTerm(d time.Duration) string { return strconv.Itoa(int(d.Hours())) + " hr" }

// value: Until's hours from the next one; it offers 18:00 first while
// 18:00 is among them, else the middle one.
func untilField(now time.Time, zone *time.Location) page.Field {
	next := now.In(zone).Truncate(time.Hour).Add(time.Hour)
	opts := make([]page.Option, 0, untilHours)
	def := ""
	for i := range untilHours {
		at := next.Add(time.Duration(i) * time.Hour)
		opts = append(opts, page.Option{Value: moment(at), Label: at.Format("15:04")})
		if at.Hour() == untilHour && def == "" {
			def = moment(at)
		}
	}
	if def == "" {
		def = opts[len(opts)/2].Value
	}
	return page.Field{Key: "Until", Label: "Until", Kind: page.FieldSelect, Options: opts, Default: def}
}

func optionLabel(f page.Field, value string) string {
	for _, o := range f.Options {
		if o.Value == value {
			return o.Label
		}
	}
	return value
}

// invariant: AW-21: Change… on a Mac that never sleeps lets it sleep again,
// a y/n that names each server it hosts, which pauses while it sleeps; on
// one that sleeps it keeps it awake for good. It runs on that Mac.
func change(r awake.Report, name string) (page.Action, bool) {
	if r.Engine == awake.EngineReading || (r.Engine != "" && r.Engine != awake.EngineRunning) {
		return page.Action{}, false
	}
	run := &page.Run{Target: &page.Target{Name: r.Mac}, In: map[string]string{"Names": `["` + neverSleeps + `"]`, "Confirm": "yes"}}
	a := page.Action{ID: ActionChange + r.Mac, Label: "Change", On: page.OnRow, Rows: []string{r.Mac}, Priority: 2, Mutates: true, Run: run}
	if r.Mode() == awake.NeverSleeps {
		run.Call = CallSettingsOff
		lines := make([]page.Value, 0, len(r.Servers))
		for _, s := range r.Servers {
			lines = append(lines, page.Text("It hosts "+s+" (server): "+s+" pauses while "+name+" sleeps."))
		}
		a.Confirm = &page.Confirm{Question: "Let " + name + " sleep again?", Lines: lines, Yes: "Allow sleep", No: "Keep it awake", Danger: len(r.Servers) > 0}
		return a, true
	}
	run.Call = CallSettingsApply
	a.Confirm = &page.Confirm{Question: "Keep " + name + " awake for good?", Lines: []page.Value{page.Text("Never sleeps: the always-on Mac policy, until you change it again.")}, Yes: "Never sleep", No: "Cancel"}
	return a, true
}
