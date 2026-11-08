package menubar

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/machine"
	coresection "github.com/jterrazz/studio-station/src/core/section"
	"github.com/jterrazz/studio-station/src/engine"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/ports"
)

const (
	SectionLatency  = "latency"
	SectionUptime   = "uptime"
	SectionLog      = "log"
	SectionTooling  = "tooling"
	ActionLog       = "log"
	ActionCopySSH   = "copy-ssh"
	ActionRestartUp = "restart-now"

	RowHealth = "health"
)

// value: what a server is doing, as its page draws it: one of these.
type serverState int

const (
	serverUp serverState = iota
	serverDown
	serverRestarting
	serverStopped
)

func stateOf(vm model.VM) serverState {
	switch {
	case vm.Restart != nil && vm.Restart.Step > 0:
		return serverRestarting
	case model.Down(vm):
		return serverDown
	case vm.State != model.Running:
		return serverStopped
	}
	return serverUp
}

type serverView struct {
	m     Machine
	h     model.MacView
	vm    model.VM
	on    string
	state serverState
}

func serverPage(m Machine, h model.MacView, vm model.VM) page.Page {
	// invariant: SPEC A4 A server: its header (the way back, the rack, its
	// menu), its facts, Overview and Activity; Overview says its health
	// first (folded while healthy, unfolded by itself while it restarts,
	// the red banner with Restart while down, Start once its owner stopped
	// it), then its check's latency, its last 30 days, why it went down, its
	// gauges, its ports and the tooling that manages its inside; the
	// footer is its ssh and Open in Terminal, which waits while it cannot
	// answer.
	s := serverView{m: m, h: h, vm: vm, on: model.Label(*m.All, h), state: stateOf(vm)}
	p := page.Page{ID: MachineID, Command: "menubar", Context: page.Context{Machine: vm.Name, Kind: string(machine.Server), On: s.on}}
	p.Header = &page.Header{Back: "All machines", Glyph: page.Glyph{Kind: string(machine.Server)}, Buttons: []string{ActionMenu}}
	p.Segments = []page.Segment{{ID: SegmentOverview, Label: "Overview", Picked: true}, {ID: SegmentActivity, Label: "Activity", Key: "cmd+i"}}
	p.Sections = []page.Section{{ID: SectionSummary, Title: vm.Name, Role: page.Summary, Caption: s.facts()}, s.health()}
	if card, ok := s.latency(); ok {
		p.Sections = append(p.Sections, card)
	}
	if log, ok := s.log(); ok {
		p.Sections = append(p.Sections, log)
	}
	if up, ok := s.uptime(); ok {
		p.Sections = append(p.Sections, up)
	}
	p.Sections = append(p.Sections, s.gauges(), s.ports(), s.tooling())
	act := activitySection(running(m, vm), vm.State == model.Running, "Activity", vm.Name)
	act.Segment = SegmentActivity
	p.Sections = append(p.Sections, act)
	p.Actions = s.actions()
	p.Footer = s.footer()
	p.Live = pageLive(p)
	return page.Seal(p)
}

// value: the server's facts: its kind, the Mac it is on, what it is
// doing (up 12 days and its restarts, down since 14:02, restarting,
// stopped), and that it restarts after a power cut, which drops first
// when the line is short.
func (s serverView) facts() []page.Value {
	part := func(text string, pri int) page.Value {
		v := page.Text(text)
		v.Pri = pri
		return v
	}
	on := part("on "+s.on, 1)
	on.Nav = machinePage(s.h.Name)
	out := []page.Value{page.Text(string(machine.Server)), on}
	switch s.state {
	case serverDown:
		down := page.Text("down")
		if since := s.vm.CheckStatus.Since; !since.IsZero() {
			down = page.Text("down since " + s.clock(since))
		}
		down.Signal = page.SignalFailing
		out = append(out, down)
	case serverRestarting:
		out = append(out, page.Text("restarting"))
	case serverStopped:
		out = append(out, page.Text("stopped"))
	default:
		if up := s.up(); up > 0 {
			out = append(out, page.Text("up "+days(up)))
		}
		if n := s.vm.Restarts.Count; n > 0 {
			out = append(out, part("restarted "+strconv.Itoa(n)+" "+plural(n, "time"), 2))
		}
	}
	if s.vm.Lifetime == model.AlwaysOn {
		out = append(out, part("restarts after a power cut", 3))
	}
	return out
}

func (s serverView) up() time.Duration {
	if !s.vm.BootAt.IsZero() && s.m.Now.After(s.vm.BootAt) {
		return s.m.Now.Sub(s.vm.BootAt)
	}
	if s.m.Summary != nil {
		return s.m.Summary.Head.Up
	}
	return 0
}

func (s serverView) clock(t time.Time) string {
	return (needSaid{r: NeedsReading{Zone: s.m.Zone}}).at(t)
}

// value: SPEC A4's Health card: one head that unfolds to the four checks
// of every boot, each in its 16 pt slot with its result (SPEC A9 3-9);
// while a restart runs, the step running is the one counted.
func (s serverView) health() page.Section {
	sec := page.Section{ID: SectionHealth, Title: "Health", Segment: SegmentOverview, Card: page.CardHealth, Columns: columns(), Uncounted: true}
	head := page.Row{ID: RowHealth, Mark: page.OK}
	group := page.Group{Fold: page.Folded}
	switch s.state {
	case serverDown:
		head.ID, head.Mark, head.Reason = "down", page.Broken, page.Down
		head.Label = "Down"
		if since := s.vm.CheckStatus.Since; !since.IsZero() {
			head.Label = "Down since " + s.clock(since)
		}
		head.Glyph = symbol("xmark.octagon.fill", page.SignalFailing)
		head.Detail = page.Text(titled(cmpWord(s.vm.Cause, "it stopped")))
		head.Verb = ActionRestartUp
		group.Fold = ""
	case serverStopped:
		head.ID, head.Mark, head.Reason = "stopped", page.Absent, page.Stopped
		head.Label, head.Detail = "Stopped", page.Text("by you")
		head.Glyph = page.Glyph{Dot: page.DotSquare, Signal: page.SignalQuiet}
		head.Verb = ActionStartVM
		group.Fold = ""
	case serverRestarting:
		w := *s.vm.Restart
		head.Label, head.Cells = "Restarting", []page.Value{page.Text("step " + w.Counted())}
		head.Glyph = page.Glyph{Spin: true, Signal: page.SignalActive}
		head.Detail = page.Text("started " + ago(s.m.Now.Sub(w.At)) + " ago")
		group.Fold = page.Open
		group.Items = s.walk(w.Step)
	default:
		checks := s.checks()
		passed := 0
		for _, it := range checks {
			if it.Row.Signal == page.SignalWell {
				passed++
			}
		}
		of := strconv.Itoa(passed) + " of " + strconv.Itoa(len(checks))
		head.Label, head.Cells, head.Glyph = "Healthy", []page.Value{page.Text(of)}, symbol("checkmark.circle.fill", page.SignalWell)
		if passed < len(checks) {
			head.Label, head.Mark, head.Reason = "Check failing", page.Attention, page.Failing
			head.Glyph = symbol("exclamationmark.triangle.fill", page.SignalWaits)
		}
		words := []page.Value{page.Text("checked after every boot")}
		if at := s.vm.CheckStatus.At; !at.IsZero() && s.m.Now.After(at) {
			words = append(words, page.Text(ago(s.m.Now.Sub(at))+" ago"))
		}
		head.Detail = page.List(words...)
		head.Menu = []string{ActionRestart, ActionStop}
		group.Items = checks
	}
	group.Head = head
	sec.Items = []page.Item{{Group: &group}}
	return sec
}

// value: the four checks of a healthy server, each passed with what it
// answered: its state, its address, and the check's code and time.
func (s serverView) checks() []page.Item {
	health := model.HealthOf(s.vm)
	check := s.vm.Check != ""
	answer := []string{}
	if a := s.vm.CheckStatus.Answer; a != "" {
		answer = append(answer, a)
	}
	if l, ok := model.LatencyOf(s.vm.Latency); ok {
		answer = append(answer, strconv.FormatInt(l.Now.Milliseconds(), 10)+" ms")
	}
	passes := health.SSH && (!check || s.vm.CheckStatus.State == model.CheckPassing)
	rows := []page.Row{
		checkRow("running", model.StepRunning, health.Running, "running"),
		checkRow("address", model.StepAddress, health.Address, s.vm.IP),
		checkRow(string(coresection.SSH), model.StepSSH, health.SSH),
		checkRow("check", model.StepCheck, passes, answer...),
	}
	return items(rows)
}

func checkRow(id, label string, passed bool, result ...string) page.Row {
	row := page.Row{ID: id, Label: label, Mark: page.OK, Signal: page.SignalWell}
	if said := slices.DeleteFunc(result, func(r string) bool { return r == "" }); len(said) > 0 {
		row.Detail = page.Texts(said...)
	}
	if !passed {
		row.Mark, row.Reason, row.Signal, row.Detail = page.Broken, page.Failing, page.SignalFailing, page.Text("failing")
	}
	return row
}

// value: a restart's four steps: those passed, the one running
// (checking), those next.
func (s serverView) walk(step int) []page.Item {
	labels := []string{model.StepRunning, model.StepAddress, model.StepSSH, model.StepCheck}
	ids := []string{"running", "address", string(coresection.SSH), "check"}
	rows := make([]page.Row, 0, len(labels))
	for i, label := range labels {
		row := page.Row{ID: ids[i], Label: label, Mark: page.OK}
		switch {
		case i+1 < step:
			row.Signal = page.SignalWell
			if i == 0 {
				row.Detail = page.Text("running")
			}
			if i == 1 && s.vm.IP != "" {
				row.Detail = page.Text(s.vm.IP)
			}
		case i+1 == step:
			row.Signal, row.Detail = page.SignalActive, page.Text("checking")
		default:
			row.Mark, row.Reason, row.Signal, row.Detail = page.Absent, page.Waiting, page.SignalQuiet, page.Text("next")
		}
		rows = append(rows, row)
	}
	return items(rows)
}

// value: SPEC B5.4's latency card: the site its check asks, its latency now,
// its p50 and its peak with when, over the last hour's runs; latency
// only, as decision D1 leaves requests per minute.
func (s serverView) latency() (page.Section, bool) {
	l, ok := model.LatencyOf(s.vm.Latency)
	if !ok || s.state != serverUp {
		return page.Section{}, false
	}
	site := s.vm.Check
	if c, ok := model.ParseCheck(s.vm.Check); ok && c.URL != nil {
		site = c.URL.Hostname()
	} else if name, _, found := strings.Cut(site, ":"); found {
		site = name
	}
	ms := func(d time.Duration) string { return strconv.FormatInt(d.Milliseconds(), 10) + " ms" }
	points := make([]float64, 0, len(s.vm.Latency))
	peak := 0
	for i, p := range s.vm.Latency {
		points = append(points, float64(p.Took.Milliseconds()))
		if p.Took > s.vm.Latency[peak].Took {
			peak = i
		}
	}
	peakAt := ago(s.m.Now.Sub(l.PeakAt)) + " ago"
	chart := page.Chart{Kind: page.ChartLine, Series: []page.Series{{Key: "latency", Points: points, Unit: "ms"}},
		Peak: &page.Peak{Index: peak, Value: page.Text(ms(l.Peak)), At: peakAt}, Ends: []string{"1 hr ago", "now"}}
	runs := page.Fact{Key: "health check", Value: page.Texts("p50 "+ms(l.P50), "peak "+ms(l.Peak)+", "+peakAt)}
	return page.Section{ID: SectionLatency, Title: site, Segment: SegmentOverview, Card: page.CardChart, Uncounted: true,
		Caption: []page.Value{page.Text(ms(l.Now))}, Items: []page.Item{{Fact: &runs}, {Chart: &chart}}}, true
}

// value: SPEC B5.2's last 30 days: a cell a day, oldest first, today
// last, each with its tooltip, and the share of the time it was up.
func (s serverView) uptime() (page.Section, bool) {
	v, ok := s.vm.Uptime.View(s.m.Now)
	if !ok {
		return page.Section{}, false
	}
	days := make([]page.Day, 0, len(v.Cells))
	for _, c := range v.Cells {
		days = append(days, page.Day{Date: c.Date, Mark: page.DayMark(c.Mark), Tip: s.tip(c)})
	}
	available := page.Text(strconv.FormatFloat(v.Availability, 'f', 2, 64) + "% available")
	return page.Section{ID: SectionUptime, Title: "Last 30 days", Segment: SegmentOverview, Card: page.CardUptime, Uncounted: true,
		Caption: []page.Value{available},
		Items:   []page.Item{{Uptime: &page.Uptime{Days: days, Availability: available}}, {Note: &page.Note{Text: "30 days ago"}}, {Note: &page.Note{Text: "today"}}}}, true
}

func (s serverView) tip(c model.UptimeCell) string {
	var parts []string
	for _, in := range c.Down {
		words := "Down from " + s.clock(in.From) + " to "
		if in.To.IsZero() {
			words += "now"
		} else {
			words += s.clock(in.To)
		}
		if in.Cause != "" {
			words += ", " + in.Cause
		}
		parts = append(parts, words)
	}
	if c.Restarts > 0 {
		parts = append(parts, "restarted "+strconv.Itoa(c.Restarts)+" "+plural(c.Restarts, "time"))
	}
	if len(parts) == 0 && c.Mark == model.DayOK {
		return "Up all day"
	}
	return strings.Join(parts, "; ")
}

// value: SPEC B5.5's Last lines card: the lines its console kept before
// it went down, each said as its time, its source and what it said, the
// cause line counted, the full log behind Show full log (SPEC A9 1-7).
func (s serverView) log() (page.Section, bool) {
	if s.state != serverDown || len(s.vm.LastLines) == 0 {
		return page.Section{}, false
	}
	lines := logLines(s.vm.LastLines, s.vm.CheckStatus.Since, s.m.Zone)
	l := page.Log{Lines: lines, More: ActionLog}
	for i, line := range s.vm.LastLines {
		if _, ok := catalog.CauseIn(line); ok {
			l.Cause = i + 1
			break
		}
	}
	return page.Section{ID: SectionLog, Title: "Last lines before it went down", Segment: SegmentOverview, Card: page.CardLog, Uncounted: true,
		Items: []page.Item{{Log: &l}}}, true
}

// value: a console's lines as the card says them: its time (a kernel
// stamp, seconds since boot, read back from the last line, which is when
// it went down), its source (`k3s`, `kernel` for a stamped line with
// none) and what it said.
func logLines(raw []string, end time.Time, zone *time.Location) []string {
	if zone == nil {
		zone = time.UTC
	}
	type parsed struct {
		stamp       float64
		stamped     bool
		source, say string
	}
	lines := make([]parsed, 0, len(raw))
	last := -1.0
	for _, line := range raw {
		p := parsed{say: strings.TrimSpace(line)}
		if rest, ok := strings.CutPrefix(p.say, "["); ok {
			if stamp, after, found := strings.Cut(rest, "]"); found {
				if v, err := strconv.ParseFloat(strings.TrimSpace(stamp), 64); err == nil {
					p.stamp, p.stamped, p.say = v, true, strings.TrimSpace(after)
					last = max(last, v)
				}
			}
		}
		if word, said, found := strings.Cut(p.say, " "); found && strings.HasSuffix(word, ":") {
			name := strings.TrimSuffix(word, ":")
			if i := strings.IndexByte(name, '['); i > 0 {
				name = name[:i]
			}
			p.source, p.say = name, strings.TrimSpace(said)
		} else if p.stamped {
			p.source = "kernel"
		}
		lines = append(lines, p)
	}
	out := make([]string, 0, len(lines))
	for _, p := range lines {
		at := end
		if p.stamped && !end.IsZero() {
			at = end.Add(-time.Duration((last - p.stamp) * float64(time.Second)))
		}
		clock := "--:--:--"
		if !at.IsZero() {
			clock = at.In(zone).Format("15:04:05")
		}
		out = append(out, clock+" "+cmpWord(p.source, "console")+" "+p.say)
	}
	return out
}

// value: the server's three gauges: CPU with its cores and spark, its
// free memory over its bar, its network; while it is down or stopped
// their last values, said last, and when it was last seen; while it
// restarts each waits as a skeleton.
func (s serverView) gauges() page.Section {
	name := s.vm.Name
	m := measured(s.m.Samples)
	cpu := page.Gauge{Title: "CPU", Value: live(pendingOr(m.cpu), engine.LiveKey(engine.LiveCPU, name)), Spark: m.cpus}
	if n := s.vm.Resources.CPUs; n > 0 {
		cpu.Note = strconv.Itoa(n) + " " + plural(n, "core")
	}
	memory := page.Gauge{Title: "Memory", Value: live(pendingOr(freeWords(s.m.Samples)), engine.LiveKey(engine.LiveFree, name))}
	if n := len(s.m.Samples); n > 0 {
		if last := s.m.Samples[n-1].Sample; last.MemTotal > 0 {
			bar := page.Ratio(min(last.MemUsed, last.MemTotal), last.MemTotal)
			memory.Bar = &bar
		}
	}
	net := page.Gauge{Title: "Network", Value: live(pendingOr(m.net), engine.LiveKey(engine.LiveNet, name)), Spark: m.nets}
	switch s.state {
	case serverRestarting:
		cpu = page.Gauge{Title: "CPU", Value: page.Pending()}
		memory = page.Gauge{Title: "Memory", Value: page.Pending()}
		net = page.Gauge{Title: "Network", Value: page.Pending()}
	case serverDown, serverStopped:
		cpu = page.Gauge{Title: "CPU", Note: "last", Value: pendingOr(m.cpu), Stale: true}
		memory = page.Gauge{Title: "Memory", Note: "last", Value: pendingOr(freeWords(s.m.Samples)), Stale: true}
		seen := page.None()
		if n := len(s.m.Samples); n > 0 {
			seen = page.Text(s.clock(s.m.Samples[n-1].Sample.At))
		}
		net = page.Gauge{Title: "Last seen", Value: seen, Stale: true}
	case serverUp:
	}
	return page.Section{ID: SectionGauges, Title: "Gauges", Segment: SegmentOverview, Card: page.CardGauges,
		Items: []page.Item{{Gauge: &cpu}, {Gauge: &memory}, {Gauge: &net}}}
}

// value: a port the server serves: its software, its number, what its
// row says of it, and the address that reaches it, which its row copies.
type served struct {
	named  catalog.PortName
	port   int
	detail page.Value
	link   string
}

// value: the server's ports: those declared public with their route,
// those its own tailnet device serves, those its Mac shares, each by its
// software.
func (s serverView) served() []served {
	if s.m.Ports == nil {
		return nil
	}
	var out []served
	seen := map[int]bool{}
	for _, mp := range s.m.Ports.Macs {
		for _, p := range mp.Ports {
			if p.Machine.Name != s.vm.Name || (p.Declared == nil && !p.Device) {
				continue
			}
			seen[p.Port] = true
			out = append(out, served{named: p.Named(), port: p.Port, detail: s.address(p), link: s.link(p)})
		}
		for _, sv := range mp.Shared {
			if sv.VM != s.vm.Name || seen[sv.GuestPort] {
				continue
			}
			addr := mp.PublicName + ":" + strconv.Itoa(sv.Reserved)
			if sv.URLs.Public != "" {
				addr = strings.TrimPrefix(sv.URLs.Public, "https://")
			}
			out = append(out, served{named: sv.Named(), port: sv.GuestPort, detail: page.Text(addr), link: cmpWord(sv.URLs.Public, addr)})
		}
	}
	return out
}

// value: the server's Ports card: a row per port it serves, greyed, and
// said so, while it cannot answer.
func (s serverView) ports() page.Section {
	sec := page.Section{ID: SectionPorts, Title: "Ports", Segment: SegmentOverview, Card: page.CardFlush, Uncounted: true}
	if s.m.Ports == nil {
		sec.Caption = []page.Value{page.Text("checking")}
		return sec
	}
	for _, p := range s.served() {
		sec.Items = append(sec.Items, s.portRow(p))
	}
	switch {
	case len(sec.Items) == 0:
		sec.Items = []page.Item{{Empty: &page.Empty{Text: "No port shared yet"}}}
	case s.state == serverDown:
		sec.Caption = []page.Value{page.Text("unreachable while down")}
	case s.state == serverRestarting:
		sec.Caption = []page.Value{page.Text("unreachable until ready")}
	case s.state == serverStopped:
		sec.Caption = []page.Value{page.Text("unreachable while stopped")}
	default:
		sec.Caption = []page.Value{page.Text(strconv.Itoa(len(sec.Items)) + " shared")}
	}
	return sec
}

func (s serverView) address(p ports.Port) page.Value {
	if p.Declared != nil {
		via := p.Declared.Via
		if r, ok := catalog.RouteNamed(via); ok {
			via = r.Label
		}
		return page.Texts(strconv.Itoa(p.Port)+" public", via)
	}
	return page.Text(s.vm.Name + ":" + strconv.Itoa(p.Port))
}

// value: the address that reaches a port: for one declared public, the
// site its check asks when that site is served on this port (its route
// reaches it there), else its own name and port on the tailnet.
func (s serverView) link(p ports.Port) string {
	if c, ok := model.ParseCheck(s.vm.Check); ok && c.URL != nil && p.Declared != nil {
		site, port := c.URL.Hostname(), c.URL.Port()
		if port == "" && c.URL.Scheme == "https" {
			port = "443"
		}
		if site != s.vm.Name && site != s.vm.Device && port == strconv.Itoa(p.Port) {
			return c.URL.Scheme + "://" + site
		}
	}
	return s.vm.Name + ":" + strconv.Itoa(p.Port)
}

func (s serverView) portRow(p served) page.Item {
	row := page.Row{ID: s.vm.Name + ":" + strconv.Itoa(p.port), Label: p.named.Name, Glyph: portGlyph(p.named), Mark: page.OK, Detail: p.detail}
	if s.state != serverUp {
		row.Signal = page.SignalFaint
	} else {
		row.Quick = "copy:" + row.ID
	}
	return page.Item{Row: &row}
}

// value: what station leaves to the server's own tooling: everything
// inside it.
func (s serverView) tooling() page.Section {
	return page.Section{ID: SectionTooling, Title: "Tooling", Segment: SegmentOverview, Uncounted: true,
		Items: []page.Item{{Note: &page.Note{Text: "Apps, settings and accounts: your own tooling (Ansible)."}}}}
}

// value: the ssh that opens a shell on it, as its own line says it, else
// as station names it.
func (s serverView) ssh() string {
	if s.m.Summary != nil {
		for _, l := range s.m.Summary.Lines {
			if l.Section == coresection.SSH && len(l.Parts) > 0 && l.Parts[0].Word != "" {
				if user, at, found := strings.Cut(l.Parts[0].Word, "@"); found {
					return catalog.SSHCommand(user, at)
				}
				return catalog.SSHCommand("", l.Parts[0].Word)
			}
		}
	}
	return catalog.SSHCommand("", s.vm.Name)
}

// value: the footer: its ssh to copy and Open in Terminal; while it
// cannot answer, why in place of the ssh, and the button waits.
func (s serverView) footer() *page.Footer {
	f := &page.Footer{Kind: page.FooterSSH, Buttons: []string{ActionOpenStation}}
	switch s.state {
	case serverUp:
		ssh := page.Command(s.ssh())
		f.Copy = &ssh
		f.Buttons = []string{ActionCopySSH, ActionOpenStation}
	case serverRestarting:
		f.Note = "Opens once SSH answers"
	default:
		f.Note = "Needs " + s.vm.Name + " running"
	}
	return f
}

func (s serverView) actions() []page.Action {
	m, h, vm := s.m, s.h, s.vm
	kind := machine.Server
	restart := page.Action{ID: ActionRestart, Label: "Restart", On: page.OnPage, Priority: 1, Mutates: true, Confirm: ask(m.Asks, "restart", kind, vm.Name, "Restart "+vm.Name+"?", "restart"),
		Run: &page.Run{Call: CallRestartVM, In: map[string]string{"Name": vm.Name}, Target: target(h)}}
	restart.Run.In = answered(restart.Confirm, restart.Run.In)
	stop := page.Action{ID: ActionStop, Label: "Stop", On: page.OnPage, Priority: 2, Mutates: true,
		Confirm: ask(m.Asks, "stop", kind, vm.Name, "Stop "+vm.Name+"?", "stop", "It stays stopped until you start it: the engine no longer restarts it."),
		Run:     &page.Run{Call: CallStopVM, In: map[string]string{"VM": vm.Name}, Target: target(h)}}
	stop.Run.In = answered(stop.Confirm, stop.Run.In)
	del := page.Action{ID: ActionDelete, Label: "Delete", On: page.OnPage, Priority: 3, Mutates: true,
		Confirm: ask(m.Asks, "delete", kind, vm.Name, "Delete "+vm.Name+"?", "delete", "Its disk is erased: nothing of it is kept."),
		Run:     &page.Run{Call: CallDeleteVM, In: map[string]string{"VM": vm.Name}, Target: target(h)}}
	del.Run.In, del.Destructive = answered(del.Confirm, del.Run.In), del.Confirm != nil
	out := []page.Action{restart, stop, del}
	menu := page.Action{ID: ActionMenu, Label: "More options for " + vm.Name, Icon: "ellipsis", On: page.OnPage, Submenu: []string{ActionRestart, ActionStop, ActionDelete}}
	switch s.state {
	case serverDown:
		out = append(out,
			page.Action{ID: ActionRestartUp, Label: "Restart", On: page.OnRow, Rows: []string{"down"}, Priority: 1, Mutates: true, Default: true,
				Run: &page.Run{Call: CallRestartVM, In: map[string]string{"Name": vm.Name}, Target: target(h)}, Hud: hud("Restarting " + vm.Name)},
			page.Action{ID: ActionLog, Label: "Show full log", On: page.OnPage, Priority: 4, Run: &page.Run{Terminal: true, Argv: []string{"machines", "log", vm.Name}, Target: target(h)}})
	case serverStopped:
		out = append(out, page.Action{ID: ActionStartVM, Label: "Start", On: page.OnRow, Rows: []string{"stopped"}, Priority: 1, Mutates: true, Default: true,
			Run: &page.Run{Call: CallStartVM, In: map[string]string{"VM": vm.Name}, Target: target(h)}, Hud: hud("Starting " + vm.Name)})
		menu.Submenu = []string{ActionStartVM, ActionDelete}
	case serverUp, serverRestarting:
	}
	if s.state == serverUp {
		ssh := s.ssh()
		out = append(out, page.Action{ID: ActionCopySSH, Label: "Copy " + ssh, Icon: "doc.on.doc", On: page.OnPage, Priority: 4, Run: &page.Run{Copy: ssh}, Hud: hud("Copied " + ssh)})
		for _, p := range s.served() {
			id := s.vm.Name + ":" + strconv.Itoa(p.port)
			out = append(out, page.Action{ID: "copy:" + id, Label: "Copy link to " + p.named.Name, Icon: "doc.on.doc", On: page.OnRow, Rows: []string{id}, Priority: 4,
				Run: &page.Run{Copy: p.link}, Hud: hud("Copied " + p.link)})
		}
	}
	open := openIn("--on", vm.Name)
	open.Icon = "terminal"
	if s.state != serverUp {
		open.Disabled = s.footer().Note
	}
	out = append(out, open, menu)
	return slices.Clip(out)
}
