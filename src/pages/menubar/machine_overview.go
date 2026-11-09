package menubar

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/activity"
	"github.com/jterrazz/studio-station/src/agents"
	"github.com/jterrazz/studio-station/src/awake"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/finding"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/core/section"
	"github.com/jterrazz/studio-station/src/engine"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/overview"
	"github.com/jterrazz/studio-station/src/page"
	pagemachines "github.com/jterrazz/studio-station/src/pages/machines"
)

const (
	// value: the needs a Mac's card shows before its fold (SPEC A4).
	needsShown = 3

	RowMoreNeeds = "more-needs"
	RowAwake     = "awake"
	RowUpkeep    = "upkeep"
)

type macOverview struct {
	m      Machine
	h      model.MacView
	label  string
	stale  bool
	self   string
	needs  []Need
	upkeep []Need
}

func overviewOf(m Machine, h model.MacView, label string, stale bool) macOverview {
	o := macOverview{m: m, h: h, label: label, stale: stale}
	for _, mac := range m.All.Macs {
		if mac.ThisMac {
			o.self = mac.Name
		}
	}
	r := NeedsReading{Now: m.Now, Zone: m.Zone, All: m.All, Ports: m.Ports, Accounts: m.Shares}
	if m.Summary != nil {
		r.Overviews = []overview.Summary{*m.Summary}
	}
	needs, upkeep := needsOf(r, o.self)
	o.needs, o.upkeep = scoped(needs, h.Name), scoped(upkeep, h.Name)
	for i, n := range o.upkeep {
		if n.Own != nil {
			o.upkeep[i].Verb = *n.Own
		}
	}
	return o
}

func macOverviewSections(m Machine, h model.MacView, label string, stale bool) []page.Section {
	// invariant: SPEC A4 A Mac's Overview: what needs you there (its own
	// and its VMs', red first, three before the fold), its gauges, its VMs
	// (servers first), then its care card: Awake and the upkeep folded.
	o := overviewOf(m, h, label, stale)
	var out []page.Section
	if len(o.needs) > 0 {
		out = append(out, o.needsCard())
	}
	return append(out, o.gauges(), o.vms(), o.care())
}

func overviewActions(m Machine, h model.MacView, label string) []page.Action {
	o := overviewOf(m, h, label, false)
	var out []page.Action
	for _, n := range append(o.needs, o.upkeep...) {
		out = append(out, n.Verb)
	}
	out = append(out, newVMSheets(m.All, m.Shares, o.self, h.Name, false)...)
	for _, vm := range h.VMs {
		if vm.State == model.Running || vm.State == model.Creating {
			out = append(out, attach(h, vm))
		}
	}
	if slices.ContainsFunc(h.VMs, model.VM.Finished) {
		c := ask(m.Asks, "cleanup", machine.Mac, label, "Delete the finished throwaways on "+label+"?", "delete", "Each goes with its disk, its ports and the accounts shared with it.")
		out = append(out, page.Action{ID: ActionCleanup, Label: "Clean up", On: page.OnRow, Priority: 3, Mutates: true, Confirm: c,
			Run: &page.Run{Call: CallCleanup, In: answered(c, map[string]string{}), Target: target(h)}})
	}
	if a, ok := o.awakeAction(); ok {
		out = append(out, a)
	}
	return out
}

// value: Open in Terminal on a VM, its row's quick slot.
func attach(h model.MacView, vm model.VM) page.Action {
	argv := []string{"--on", vm.Name}
	return page.Action{ID: ActionAttach + vm.Name, Label: "Open " + vm.Name + " in Terminal", Icon: "terminal", Key: "cmd+enter", On: page.OnRow, Rows: []string{vm.Name},
		Priority: 2, Run: &page.Run{Terminal: true, Argv: argv, Target: target(h)}, Hud: opened(vm.Name)}
}

func (o macOverview) needsCard() page.Section {
	rows := needRows(o.needs, o.h.Name, o.self)
	for i, n := range o.needs {
		if ownWords(n, o.h.Name, o.self) && len(n.Detail) == 0 && len(n.Facts) > 0 {
			rows[i].Row.Detail = page.List(droppable(n.Facts, page.MacName(o.h.Name, o.self))...)
		}
	}
	count := page.Text(strconv.Itoa(len(o.needs)))
	count.Signal = page.SignalWaits
	if slices.ContainsFunc(o.needs, func(n Need) bool { return n.Finding.Urgency == finding.Red }) {
		count.Signal = page.SignalFailing
	}
	s := page.Section{ID: SectionNeeds, Title: "Needs you", Segment: SegmentOverview, Card: page.CardFlush, Columns: columns(), Uncounted: true,
		Caption: []page.Value{count}, Items: rows}
	if len(rows) > needsShown {
		s.Top = needsShown
		s.Rest = &page.Item{Row: &page.Row{ID: RowMoreNeeds, Label: strconv.Itoa(len(rows)-needsShown) + " more", Detail: page.Text("Show less")}}
	}
	return s
}

// value: a need's facts on its own Mac's card yield to its words: the
// Mac's own name goes, the rest drop first when the row is narrow.
func droppable(facts []page.Value, mac string) []page.Value {
	out := make([]page.Value, 0, len(facts))
	for _, f := range facts {
		if f.Text == mac {
			continue
		}
		f.Pri = len(facts) - len(out)
		out = append(out, f)
	}
	return out
}

// value: the Mac's three gauges: CPU and the network with their sparks,
// its free memory over the bar of what is used, each read live.
func (o macOverview) gauges() page.Section {
	name := o.h.Name
	m := measured(o.m.Samples)
	cpu := page.Gauge{Title: "CPU", Value: live(pendingOr(m.cpu), engine.LiveKey(engine.LiveCPU, name)), Spark: m.cpus, Stale: o.stale}
	memory := page.Gauge{Title: "Memory", Value: live(pendingOr(freeWords(o.m.Samples)), engine.LiveKey(engine.LiveFree, name)), Stale: o.stale}
	if n := len(o.m.Samples); n > 0 {
		if s := o.m.Samples[n-1].Sample; s.MemTotal > 0 {
			bar := page.Ratio(min(s.MemUsed, s.MemTotal), s.MemTotal)
			memory.Bar = &bar
		}
	}
	net := page.Gauge{Title: "Network", Value: live(pendingOr(m.net), engine.LiveKey(engine.LiveNet, name)), Spark: m.nets, Stale: o.stale}
	return page.Section{ID: SectionGauges, Title: "Gauges", Segment: SegmentOverview, Card: page.CardGauges,
		Items: []page.Item{{Gauge: &cpu}, {Gauge: &memory}, {Gauge: &net}}}
}

func pendingOr(v page.Value) page.Value {
	if v.Kind == "" {
		return page.Pending()
	}
	return v
}

// value: a Mac's free memory in the gauge's words ("23 GB free"), in the
// GB macOS counts.
func freeWords(readings []activity.Reading) page.Value {
	if len(readings) == 0 {
		return page.Value{}
	}
	s := readings[len(readings)-1].Sample
	if s.MemTotal <= 0 {
		return page.Value{}
	}
	free := s.MemTotal - min(s.MemUsed, s.MemTotal)
	return page.Text(strconv.FormatInt((free+1<<29)>>30, 10) + " GB free")
}

func vmColumns() []page.Column {
	return []page.Column{{Key: "name", Kind: page.KindText, Keep: 100}, {Key: "cpu", Kind: page.KindPercent, Keep: 20}}
}

func (o macOverview) vms() page.Section {
	s := page.Section{ID: SectionVMs, Title: "VMs", Segment: SegmentOverview, Card: page.CardFlush, Columns: vmColumns(), Uncounted: true}
	if c, ok := model.CapacityOf(o.h); ok && len(o.h.VMs) > 0 {
		used := int64(0)
		for _, v := range c.VMs {
			used += v.Bytes
		}
		s.Caption = []page.Value{page.Text(strconv.FormatInt(used>>30, 10) + " GB in use")}
	}
	if len(o.h.VMs) == 0 {
		s.Card = page.CardEmpty
		empty := page.Empty{Text: "No VMs on " + o.label + " yet", Actions: []string{ActionNewVM}}
		if c, ok := model.CapacityOf(o.h); ok {
			empty.Sub = strconv.FormatInt(c.Free>>30, 10) + " GB free for the first one"
		}
		s.Items = []page.Item{{Empty: &empty}}
		return s
	}
	vms := slices.Clone(o.h.VMs)
	slices.SortStableFunc(vms, func(a, b model.VM) int { return rank(a) - rank(b) })
	for _, vm := range vms {
		row := o.vmRow(vm)
		s.Items = append(s.Items, page.Item{Row: &row})
	}
	return s
}

// value: a VM's place on its Mac's card: servers first, then the VMs an
// agent works or waits in, the other running ones, the finished, then the
// stopped (SPEC A4's linux-1, agent-7f, api).
func rank(vm model.VM) int {
	switch {
	case vm.Lifetime == model.AlwaysOn:
		return 0
	case vm.State != model.Running && vm.State != model.Creating:
		return 4
	case vm.Finished():
		return 3
	case vm.Agent != nil:
		return 1
	}
	return 2
}

// value: a VM on its Mac's page: the agent mark or the status dot, what
// it does now, its CPU as a share of the Mac, Open in Terminal in its
// quick slot; a creating VM its ring and birth bar, a ready one its check.
func (o macOverview) vmRow(vm model.VM) page.Row {
	row := page.Row{ID: vm.Name, Label: vm.Name, Mark: page.OK, Nav: machinePage(vm.Name), Trail: "cpu",
		Cells: []page.Value{o.cpu(vm)}}
	if vm.State == model.Running || vm.State == model.Creating {
		row.Quick = ActionAttach + vm.Name
	}
	life := lifeWord(vm)
	switch {
	case vm.State == model.Creating && vm.Birth != nil:
		row.Glyph = page.Glyph{Dot: page.DotRing, Progress: vm.Birth.Progress, Signal: page.SignalActive}
		row.Detail = page.Texts("creating", vm.Birth.Step)
		birth := page.Value{Kind: page.KindRatio, N: int64(vm.Birth.Progress), Of: 1000}
		row.Birth, row.Cells = &birth, nil
		row.Trail = ""
	case model.Fresh(vm, o.m.Now):
		row.Glyph = page.Glyph{Kind: "symbol:checkmark.circle.fill", Signal: page.SignalWell}
		row.Detail, row.Fresh = page.Texts("ready", "up 1 min"), true
		row.Cells, row.Trail = nil, ""
	case model.Down(vm):
		row.Mark, row.Reason = page.Broken, page.Down
		row.Glyph = page.Glyph{Dot: page.DotFill, Signal: page.SignalFailing}
		down := page.Text("down")
		if since := vm.CheckStatus.Since; !since.IsZero() {
			down = page.Text("down since " + (needSaid{r: NeedsReading{Zone: o.m.Zone}}).at(since))
		}
		down.Signal = page.SignalFailing
		row.Detail = page.List(down)
		row.Verb, row.Cells, row.Trail = "down:"+vm.Name, nil, ""
	case vm.Lifetime == model.AlwaysOn && vm.State == model.Running:
		row.Glyph = page.Glyph{Dot: page.DotFill, Signal: page.SignalWell}
		parts := []page.Value{page.Text(health(vm))}
		if !vm.BootAt.IsZero() && o.m.Now.After(vm.BootAt) {
			parts = append(parts, page.Text("up "+days(o.m.Now.Sub(vm.BootAt))))
		}
		row.Detail = page.List(parts...)
	case vm.State != model.Running:
		row.Mark, row.Reason = page.Absent, page.Stopped
		row.Glyph = page.Glyph{Dot: page.DotSquare, Signal: page.SignalQuiet}
		row.Detail = page.Texts("stopped", life)
		row.Cells, row.Trail = nil, ""
	case vm.Finished():
		row.Glyph = page.Glyph{Kind: "symbol:checkmark", Signal: page.SignalDone}
		row.Signal = page.SignalDone
		row.Detail = page.Texts("finished", cmpWord(vm.Agent.Reason, "done"))
		row.Verb, row.Cells, row.Trail = ActionCleanup, nil, ""
	case vm.Agent != nil && vm.Agent.State == string(agents.Waiting):
		row.Glyph = page.Glyph{Kind: "agent", Signal: page.SignalWaits}
		row.Detail = page.Text(cmpWord(vm.Agent.Task, "waits for you"))
		row.Cells, row.Trail = nil, ""
	case vm.Agent != nil:
		row.Glyph = page.Glyph{Kind: "agent", Signal: page.SignalActive}
		row.Detail = page.Text(cmpWord(vm.Agent.Task, "working"))
		if len(vm.Agent.Step) > 0 {
			row.Detail = page.Texts(vm.Agent.Step...)
		}
	default:
		row.Glyph = page.Glyph{Dot: page.DotFill, Signal: page.SignalWell}
		row.Detail = page.Texts("running", life)
	}
	if o.stale {
		row.Signal = page.SignalStale
	}
	return row
}

func lifeWord(vm model.VM) string {
	return strings.ToLower(pagemachines.Lifetime(vm.Lifetime).Text)
}

// value: a VM's CPU as a share of its Mac, from the Mac's last reading,
// read live under its key.
func (o macOverview) cpu(vm model.VM) page.Value {
	v := page.None()
	if n := len(o.m.Samples); n > 0 {
		for _, u := range o.m.Samples[n-1].VMs {
			if u.VM == vm.Name {
				v = page.Percent(int64(u.CPU * 10))
			}
		}
	}
	return live(v, engine.LiveKey(engine.LiveCPUMac, vm.Name))
}

// value: SPEC A4's care card: the Mac's Awake with its switch and what
// pauses if it sleeps, then its upkeep folded behind its worst item.
func (o macOverview) care() page.Section {
	s := page.Section{ID: SectionCare, Title: "Care", Segment: SegmentOverview, Card: page.CardCare, Columns: columns(), Uncounted: true}
	if row, ok := o.awakeRow(); ok {
		s.Items = append(s.Items, page.Item{Row: &row})
	}
	head := page.Row{ID: RowUpkeep, Label: "Upkeep", Glyph: symbol("wrench.adjustable", page.SignalQuiet), Mark: page.OK, Trail: "value"}
	children := o.upkeepRows()
	ready := o.ready()
	open := len(o.upkeep)
	if ready.gap() {
		open++
	}
	switch {
	case o.m.Summary == nil:
		head.Cells = []page.Value{page.Text("checking")}
	case open > 0:
		worst := page.Text(ready.words())
		if len(o.upkeep) > 0 {
			worst = page.Text(strings.ToLower(o.upkeep[0].What))
		}
		words := []page.Value{worst}
		if n := open - 1; n > 0 {
			words = append(words, page.Text(strconv.Itoa(n)+" more"))
		}
		v := page.List(words...)
		v.Signal = page.SignalWaits
		head.Cells = []page.Value{v}
	default:
		head.Cells = []page.Value{page.Text("all fine")}
	}
	s.Items = append(s.Items, page.Item{Group: &page.Group{Head: head, Fold: page.Folded, Items: children}})
	return s
}

func (o macOverview) awakeRow() (page.Row, bool) {
	r := o.m.Awake
	if r == nil {
		return page.Row{}, false
	}
	row := page.Row{ID: RowAwake, Label: "Awake", Mark: page.OK, Glyph: symbol("sun.max", page.SignalWaits)}
	on := false
	switch {
	case r.Mode() == awake.NeverSleeps:
		row.Label, on = "Never sleeps", true
		if len(r.Servers) > 0 {
			row.Detail = page.Text("hosts " + strings.Join(r.Servers, ", ") + ", which " + pluralVerb(len(r.Servers), "pauses", "pause") + " if it sleeps")
		}
	case r.Session != nil:
		on = true
		if !r.Session.Until.IsZero() {
			zone := o.m.Zone
			if zone == nil {
				zone = time.UTC
			}
			row.Label = "Awake until " + r.Session.Until.In(zone).Format("15:04")
		}
	default:
		row.Label = "Sleeps as usual"
		row.Glyph = symbol("moon", page.SignalQuiet)
	}
	if row.Detail.IsNone() {
		row.Detail = page.Text(o.pauses())
	}
	state := page.Value{Kind: page.KindOff}
	if on {
		state = page.Value{Kind: page.KindOn}
	}
	row.Cells = []page.Value{state}
	if a, ok := o.awakeAction(); ok {
		row.Verb = a.ID
	}
	return row, true
}

// value: what pauses if the Mac sleeps: its working agents, by name.
func (o macOverview) pauses() string {
	var working []string
	for _, vm := range o.h.VMs {
		if vm.State == model.Running && vm.Agent != nil && vm.Agent.State == string(agents.Working) {
			working = append(working, vm.Name)
		}
	}
	if len(working) == 0 {
		return "nothing pauses if it sleeps"
	}
	return strings.Join(working, ", ") + " " + pluralVerb(len(working), "pauses", "pause") + " if it sleeps"
}

// value: the care card's switch: This Mac's keep-awake, another Mac's
// Change, which asks first when it lets a Mac that never sleeps sleep.
func (o macOverview) awakeAction() (page.Action, bool) {
	r := o.m.Awake
	if r == nil {
		return page.Action{}, false
	}
	if o.h.ThisMac {
		a := awakeAction(r)
		a.Rows = []string{RowAwake}
		a.Label = "Keep " + o.label + " awake"
		return a, true
	}
	a, ok := change(*r, o.label)
	a.Rows = []string{RowAwake}
	a.Label = "Keep " + o.label + " awake"
	return a, ok
}

// value: the upkeep unfolded: each item under its section's name, its
// words and its detail, its verb; with none, each section's line.
func (o macOverview) upkeepRows() []page.Item {
	var out []page.Item
	for _, n := range o.upkeep {
		what := page.Text(strings.ToLower(n.What))
		what.Signal = page.SignalWaits
		parts := []page.Value{what}
		details := n.Detail
		if n.Verb.Label == VerbUpdate && len(details) == 1 {
			details = nil
		}
		for _, d := range details {
			if !strings.EqualFold(d, n.What) && !strings.EqualFold(d, n.Finding.Text) {
				detail := page.Text(d)
				detail.Pri = 1
				parts = append(parts, detail)
			}
		}
		row := page.Row{ID: n.Key, Label: sectionWord(n.Finding.Section), Glyph: symbol(sectionSymbol(n.Finding.Section), page.SignalQuiet),
			Mark: page.Attention, Reason: reasonOf(n.Finding), Detail: page.List(parts...), Verb: n.Verb.ID}
		out = append(out, page.Item{Row: &row})
	}
	if o.m.Summary == nil {
		return out
	}
	for _, id := range []section.ID{section.Settings, section.Background, section.Apps, section.Disk} {
		if slices.ContainsFunc(o.upkeep, func(n Need) bool { return n.Finding.Section == id }) {
			continue
		}
		v := lineOf(*o.m.Summary, string(id))
		if v.IsNone() {
			continue
		}
		row := page.Row{ID: "line:" + string(id), Label: sectionWord(id), Glyph: symbol(sectionSymbol(id), page.SignalQuiet), Mark: page.OK, Detail: v}
		out = append(out, page.Item{Row: &row})
	}
	if ready := o.ready(); ready.of > 0 {
		out = append(out, page.Item{Row: ready.row()})
	}
	return out
}

const RowReady = "ready"

// value: SPEC A4, §7.6: a Mac that hosts a server or never sleeps reads
// its readiness, so its upkeep counts the items ready and names the first
// that is not.
type readiness struct {
	ok, of int
	first  string
}

func (o macOverview) ready() readiness {
	var r readiness
	if o.m.Summary == nil {
		return r
	}
	for _, c := range o.m.Summary.Readiness {
		r.of++
		switch {
		case c.State == catalog.StateOK:
			r.ok++
		case r.first == "":
			r.first = strings.ReplaceAll(c.Name, "-", " ")
		}
	}
	return r
}

func (r readiness) gap() bool { return r.ok < r.of }

func (r readiness) words() string {
	return strconv.Itoa(r.ok) + " of " + strconv.Itoa(r.of) + " ready"
}

func (r readiness) row() *page.Row {
	row := &page.Row{ID: RowReady, Label: "Ready to host servers", Glyph: symbol("lock", page.SignalQuiet), Mark: page.OK,
		Detail: page.Text(strconv.Itoa(r.ok) + " of " + strconv.Itoa(r.of))}
	if r.gap() {
		count := page.Text(r.words())
		count.Signal = page.SignalWaits
		first := page.Text(r.first)
		first.Pri = 1
		row.Mark, row.Reason, row.Detail = page.Attention, page.Degraded, page.List(count, first)
	}
	return row
}

func sectionWord(id section.ID) string {
	words := map[section.ID]string{
		section.Settings: "Settings", section.Background: "Background", section.Accounts: "Accounts", section.Apps: "Apps", section.Disk: "Disk",
	}
	return cmpWord(words[id], titled(string(id)))
}

func sectionSymbol(id section.ID) string {
	symbols := map[section.ID]string{
		section.Settings: "slider.horizontal.3", section.Background: "list.bullet.rectangle", section.Accounts: "key",
		section.Apps: "square.grid.2x2", section.Disk: "internaldrive",
	}
	return cmpWord(symbols[id], "circle")
}

// value: SPEC A4's asleep Mac: the moon, what sleeping means, Wake and
// Tell me when it wakes, and what Wake does.
func asleep(label string) []page.Section {
	empty := page.Empty{Glyph: symbol("moon", page.SignalQuiet), Text: label + " is asleep",
		Sub: "A laptop that sleeps is not down. Its VMs are not known while it sleeps.", Actions: []string{ActionWake, ActionRemind}}
	note := page.Note{Text: "Wake sends a Wake-on-LAN packet from This Mac. It works when " + label + " is on power and on the same network."}
	return []page.Section{
		{ID: SectionAsleep, Title: label + " is asleep", Card: page.CardEmpty, Items: []page.Item{{Empty: &empty}}},
		{ID: "wake-note", Title: "Wake", Role: page.Banner, Uncounted: true, Items: []page.Item{{Note: &note}}},
	}
}

func wake(h model.MacView, label string) []page.Action {
	return []page.Action{
		{ID: ActionWake, Label: "Wake", Icon: "alarm", On: page.OnPage, Priority: 1, Mutates: true, Run: &page.Run{Call: CallWake, In: map[string]string{"Mac": h.Name}},
			Hud: hud("Waking " + label)},
		{ID: ActionRemind, Label: "Tell me when it wakes", Icon: "bell", On: page.OnPage, Priority: 2, Run: &page.Run{Call: CallRemind, In: map[string]string{"Mac": h.Name}},
			Hud: hud("You will be told when " + label + " wakes")},
	}
}
