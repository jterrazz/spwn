package menubar

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageawake "github.com/jterrazz/studio-station/apps/engine/pages/awake"
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

// value: how many whole hours Until offers from the next one.
const untilHours = 8

func isThisMac(f pageawake.AllMachines, r awake.Report) bool { return f.This != "" && r.Mac == f.This }

func Awake(f pageawake.AllMachines, ctx page.Context) page.Page {
	// invariant: SPEC A4 Awake · All Macs: This Mac first, its switch, the
	// picker, Lid closed and On battery only where its capabilities have
	// them (E1), then the running work that pauses while it sleeps; then
	// each other Mac on its own card by its policy and what depends on it,
	// with Change… on its switch. No footer (judges r4 #11).
	p := page.Page{ID: AwakeID, Command: "awake", Context: ctx, Header: &page.Header{Back: Back, Title: "Keep awake", Glyph: symbol("sun.max", page.SignalQuiet)}}
	p.Sections = []page.Section{{ID: SectionSummary, Title: "Keep awake", Role: page.Summary, Caption: []page.Value{page.Text("Which Mac may sleep, and until when")}}}
	if f.Label == nil {
		self := ""
		for _, r := range f.Reports {
			if isThisMac(f, r) {
				self = r.Mac
			}
		}
		f.Label = func(mac string) string { return page.MacName(mac, self) }
	}
	for _, r := range f.Reports {
		if isThisMac(f, r) {
			p.Sections = append(p.Sections, thisMac(r, f.Zone))
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

// value: This Mac's card: Keep This Mac awake and its switch, then the
// switches its capabilities allow.
func thisMac(r awake.Report, zone *time.Location) page.Section {
	r.Zone = zone
	s := page.Section{ID: SectionThisMac, Title: page.ThisMac, Card: page.CardFlush, Uncounted: true, Columns: awakeColumns()}
	held := r.Keeping() != nil
	keep := page.Row{ID: r.Mac, Label: "Keep awake", Glyph: sun(held), Cells: switched(held), Mark: page.OK}
	if held {
		keep.Label = capital(r.Clauses()[0])
	}
	switch {
	case r.Engine == awake.EngineReading:
		keep.Mark, keep.Detail = page.Awaiting, page.Pending()
	case r.EngineDown():
		keep.Mark, keep.Detail = page.Absent, page.Text("the engine is stopped")
		keep.Hint = page.Cmd("station engine start")
	case held:
		keep.Verb = ActionLetGo
	default:
		keep.Verb = ActionKeepOn
	}
	s.Items = append(s.Items, page.Item{Row: &keep})
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

// value: the words of a Mac that sleeps as it always does.
const sleepsAsUsual = "Sleeps as usual"

func capital(s string) string { return strings.ToUpper(s[:1]) + s[1:] }

// value: the running work that pauses while This Mac sleeps, under its
// card; none, no line (SPEC A9 1-11).
func pausing(r awake.Report) (page.Section, bool) {
	if len(r.Pauses) == 0 {
		return page.Section{}, false
	}
	line := strings.Join(r.Pauses, " and ") + " " + pluralVerb(len(r.Pauses), "pauses", "pause") + " while This Mac sleeps."
	return page.Section{ID: SectionPauses, Title: "While This Mac sleeps", Uncounted: true, Items: []page.Item{{Note: &page.Note{Text: line}}}}, true
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

// value: another Mac's card row: its label the Awake verdict's sentence,
// what depends on it as its detail, its switch on while it stays awake; its
// mark the tone of the verdict.
func otherMac(r awake.Report, zone *time.Location) page.Row {
	r.Zone = zone
	row := page.Row{ID: r.Mac, Glyph: sun(false), Cells: switched(false), Detail: page.Text(dependants(r.Servers, r.Pauses))}
	row.Mark, row.Reason = markOf(r)
	v := awake.VerdictOf(r)
	if v == nil {
		row.Label, row.Detail, row.Cells = "Checking", page.Pending(), nil
		return row
	}
	row.Label = strings.TrimSuffix(v.Sentence, ".")
	if r.EngineDown() || r.Mode() == awake.SleepsNormally {
		row.Label = sleepsAsUsual
	}
	switch {
	case r.EngineDown():
		row.Detail = page.Texts(v.Line, dependants(r.Servers, r.Pauses))
		row.Hint = page.Cmd("station engine start")
	case r.Mode() == awake.NeverSleeps, r.Mode() == awake.AwakeUntil:
		row.Glyph, row.Cells = sun(true), switched(true)
	}
	return row
}

// value: the mark a Mac's card row takes from the tone of its Awake
// verdict: awaiting while there is none, OK, Attention for a guard, else
// Absent.
func markOf(r awake.Report) (page.Mark, page.Reason) {
	v := awake.VerdictOf(r)
	switch {
	case v == nil:
		return page.Awaiting, ""
	case v.Tone == "ok":
		return page.OK, ""
	case v.Tone == "warn":
		return page.Attention, page.Guard
	}
	return page.Absent, ""
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
	return page.Row{ID: s.Mac, Label: sleepsAsUsual, Glyph: symbol("moon", page.SignalAsleep), Mark: page.Absent, Signal: page.SignalAsleep, Detail: page.Texts(words...)}
}

func nanos(d time.Duration) string { return strconv.FormatInt(int64(d), 10) }

// value: an end as the call's input takes it, a JSON time.
func moment(t time.Time) string { return `"` + t.UTC().Format(time.RFC3339) + `"` }

func flags(lid, battery bool) map[string]string {
	return map[string]string{"Lid": strconv.FormatBool(lid), "Battery": strconv.FormatBool(battery), "By": "the menu bar"}
}

// invariant: a keep-awake keeps the switches as they are; a switch keeps
// the end the keep-awake holds until, or starts the one Until offers. The
// terms and the one picked are awake.Choices': Until offers the hour it
// offers, and says the hour a keep-awake holds until when it is picked.
func keeps(r awake.Report, now time.Time, zone *time.Location) []page.Action {
	if zone == nil {
		zone = time.UTC
	}
	var lid, battery bool
	kept := r.Keeping()
	held := kept != nil
	if held {
		lid, battery = kept.Lid, kept.Battery
	}
	rows := []string{r.Mac}
	var out []page.Action
	choices := awake.Choices(now, zone, kept)
	for _, c := range choices {
		if c.For == 0 {
			continue
		}
		in := flags(lid, battery)
		in["For"] = nanos(c.For)
		out = append(out, page.Action{ID: ActionKeep + strings.TrimSuffix(c.For.String(), "0m0s"), Label: c.Label, On: page.OnRow, Rows: rows, Priority: 1, Mutates: true, Picked: c.Picked, Run: &page.Run{Call: CallKeepAwake, In: in}})
	}
	field := untilField(now, zone)
	end := field.Default
	until := choices[len(choices)-1]
	out = append(out, page.Action{ID: ActionUntil, Label: until.Label, On: page.OnRow, Rows: rows, Priority: 1, Mutates: true, Picked: until.Picked,
		Form: &page.Form{Title: "Keep awake until", Fields: []page.Field{field}}, Run: &page.Run{Call: CallKeepAwake, In: flags(lid, battery)}})
	term := func(in map[string]string) map[string]string {
		if held {
			in["Until"] = moment(kept.Until)
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

// value: Until's hours from the next one, and the end awake.UntilOffered
// gives, the one it opens on; that end closes the list when it comes
// later.
func untilField(now time.Time, zone *time.Location) page.Field {
	next := now.In(zone).Truncate(time.Hour).Add(time.Hour)
	opts := make([]page.Option, 0, untilHours+1)
	for i := range untilHours {
		at := next.Add(time.Duration(i) * time.Hour)
		opts = append(opts, page.Option{Value: moment(at), Label: at.Format("15:04")})
	}
	at := awake.UntilOffered(now, zone)
	def := moment(at)
	if !slices.ContainsFunc(opts, func(o page.Option) bool { return o.Value == def }) {
		opts = append(opts, page.Option{Value: def, Label: at.In(zone).Format("15:04")})
	}
	return page.Field{Key: "Until", Label: "Until", Kind: page.FieldSelect, Options: opts, Default: def}
}

// invariant: AW-21: Change… on a Mac that never sleeps lets it sleep again,
// a y/n that names each server it hosts, which pauses while it sleeps; on
// one that sleeps it keeps it awake for good. It runs on that Mac.
func change(r awake.Report, name string) (page.Action, bool) {
	if r.Engine == awake.EngineReading || r.EngineDown() {
		return page.Action{}, false
	}
	run := &page.Run{Target: &page.Target{Name: r.Mac}, In: map[string]string{"Names": `["` + neverSleeps + `"]`, "Confirm": "yes"}}
	a := page.Action{ID: ActionChange + r.Mac, Label: "Change", On: page.OnRow, Rows: []string{r.Mac}, Priority: 2, Mutates: true, Run: run}
	if r.Mode() == awake.NeverSleeps {
		run.Call = CallSettingsOff
		return letSleep(name, r.Servers).action(yesOrNo, a), true
	}
	run.Call = CallSettingsApply
	a.Confirm = &page.Confirm{Question: "Keep " + name + " awake for good?", Lines: []page.Value{page.Text("Never sleeps: the always-on Mac policy, until you change it again.")}, Yes: "Never sleep", No: "Cancel"}
	return a, true
}
