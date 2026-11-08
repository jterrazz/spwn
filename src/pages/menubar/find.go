package menubar

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/accounts"
	"github.com/jterrazz/studio-station/src/awake"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/finding"
	"github.com/jterrazz/studio-station/src/core/machine"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/overview"
	"github.com/jterrazz/studio-station/src/page"
	pagemachines "github.com/jterrazz/studio-station/src/pages/machines"
	"github.com/jterrazz/studio-station/src/ports"
)

const (
	FindID page.ID = "menubar.find"

	FindNeeds    = "needs-you"
	FindMachines = "machines"
	FindPorts    = "ports"
	FindActions  = "actions"
	FindSections = "sections"
	// value: the verbs of each VM, a group headed by its name: the app adds
	// them once the query is a prefix of that name.
	FindVerbs = "verbs"

	ColumnTags = "tags"

	VerbOpen    = "Open"
	VerbCopy    = "Copy"
	VerbShare   = "Share"
	VerbRun     = "Run"
	VerbCleanUp = "Clean up"

	CallCleanup = "machines.cleanup"
	awakeFor    = 4 * time.Hour
)

type FindReading struct {
	// value: home's own reading, the one Find is built from: it costs no
	// read of its own.
	Reading
	Now  time.Time
	Zone *time.Location
	Asks Levels
}

type entry struct {
	row  page.Row
	tags []string
	verb page.Action
	alt  *page.Action
}

func Find(r FindReading, thisMac page.Context) page.Page {
	// invariant: SPEC B5.6: one entry per Mac, VM, shared port, listener,
	// section and action available now, each a row of its group: its glyph,
	// its title, where it is, its tags, its verb (whose Key is the entry's),
	// its alternate verb as the quick slot and the page it opens; the needs
	// first, ranked, in Find's words; each VM's verbs headed by its name.
	// The app scores and groups them; nothing read yet is the page's frame.
	if r.All == nil {
		return Opening(FindID, thisMac)
	}
	f := finder{r: r, self: thisMac.Machine}
	p := page.Page{ID: FindID, Command: "menubar", Context: thisMac}
	needs := f.needs()
	sections := []struct {
		id, title string
		entries   []entry
	}{
		{FindNeeds, "Needs you", needs},
		{FindMachines, "Machines", f.machines()},
		{FindPorts, "Ports", f.ports()},
		{FindActions, "Actions", f.actions()},
		{FindSections, "Sections", f.sections()},
	}
	for _, s := range sections {
		sec := page.Section{ID: s.id, Title: s.title, Columns: findColumns(), Uncounted: true}
		for _, e := range s.entries {
			row := f.add(&p, e)
			sec.Items = append(sec.Items, page.Item{Row: &row})
		}
		if len(sec.Items) > 0 {
			p.Sections = append(p.Sections, sec)
		}
	}
	if verbs := f.verbs(&p); len(verbs.Items) > 0 {
		p.Sections = append(p.Sections, verbs)
	}
	return page.Seal(p)
}

func findColumns() []page.Column {
	return []page.Column{{Key: "name", Kind: page.KindListing, Keep: 100}, {Key: ColumnTags, Label: "tags", Kind: page.KindList, Show: page.PipeOnly}}
}

type finder struct {
	r    FindReading
	self string
}

func (f finder) mac(name string) string { return page.MacName(name, f.self) }

func (f finder) at(t time.Time) string {
	zone := f.r.Zone
	if zone == nil {
		zone = time.UTC
	}
	return t.In(zone).Format("15:04")
}

func (f finder) add(p *page.Page, e entry) page.Row {
	row := e.row
	e.verb.On, e.verb.Rows = page.OnRow, []string{row.ID}
	row.Verb = e.verb.ID
	p.Actions = append(p.Actions, e.verb)
	if e.alt != nil {
		e.alt.On, e.alt.Rows = page.OnRow, []string{row.ID}
		row.Quick = e.alt.ID
		p.Actions = append(p.Actions, *e.alt)
	}
	if len(e.tags) > 0 {
		row.Cells = []page.Value{page.Texts(e.tags...)}
	}
	return row
}

func opens(id, key string) page.Action {
	// value: a verb that runs nothing opens its row's page.
	return page.Action{ID: id, Label: VerbOpen, Key: key}
}

func (f finder) needs() []entry {
	// value: SPEC A6: what needs you, ranked as Needs you ranks it, each
	// titled by its verb first ("Restart infra", "Review review-bot", "PR #42
	// is ready"); upkeep is not Find's.
	r := NeedsReading{Now: f.r.Now, Zone: f.r.Zone, All: f.r.All, Ports: f.r.Ports, Accounts: f.r.Accounts}
	if f.r.Home != nil {
		r.Overviews = append([]overview.Summary{*f.r.Home}, f.r.Away...)
	}
	needs, _ := needsOf(r, f.self)
	out := make([]entry, 0, len(needs))
	for _, n := range needs {
		row := page.Row{ID: n.Key, Label: n.Find[0], Detail: page.Texts(n.Find[1:]...), Glyph: n.Glyph, Nav: n.Nav, Mark: page.Attention, Reason: reasonOf(n.Finding)}
		if len(n.Find) == 1 {
			row.Detail = page.Text(n.What)
		}
		if n.Finding.Urgency == finding.Red {
			row.Mark = page.Broken
		}
		verb := n.Verb
		verb.ID = "need:" + n.Key
		out = append(out, entry{row: row, verb: verb})
	}
	return out
}

func awakeMac(h model.MacView) bool { return h.ThisMac || h.Reachable }

func macGlyph(h model.MacView, signal page.Signal, dot page.Dot) page.Glyph {
	kind := "desktop"
	if h.Laptop {
		kind = "laptop"
	}
	return page.Glyph{Kind: kind, Signal: signal, Dot: dot}
}

func free(h model.MacView) (said string, bytes int64, known bool) {
	c, ok := model.CapacityOf(h)
	if !ok {
		return "", 0, false
	}
	return strconv.FormatInt(c.Free>>30, 10) + " GB free", c.Free, true
}

func (f finder) machines() []entry {
	// value: each Mac in home's order, ⌘1 to ⌘9, then each VM where its Mac
	// lists it, with its state and its Mac.
	var macs, vms []entry
	for i, h := range f.r.All.Macs {
		row := page.Row{ID: h.Name, Label: f.mac(h.Name), Nav: machinePage(h.Name)}
		switch {
		case !awakeMac(h):
			row.Glyph = macGlyph(h, page.SignalAsleep, page.DotDash)
			row.Detail = page.Text("asleep")
			if !h.Answered.IsZero() && !f.r.Now.IsZero() {
				row.Detail = page.Texts("asleep", "seen "+minutes(f.r.Now.Sub(h.Answered))+" ago")
			}
		default:
			row.Glyph = macGlyph(h, page.SignalWell, page.DotFill)
			if slices.ContainsFunc(h.VMs, model.Down) {
				row.Glyph.Signal = page.SignalFailing
			}
			words := []string{counted(len(h.VMs), "VM", "VMs")}
			if said, _, ok := free(h); ok {
				words = append(words, said)
			}
			row.Detail = page.Texts(words...)
		}
		key := ""
		if i < 9 {
			key = "cmd+" + strconv.Itoa(i+1)
		}
		macs = append(macs, entry{row: row, tags: []string{"mac", "machine"}, verb: opens("open:"+h.Name, key)})
		for _, vm := range model.Ordered(h.VMs) {
			vms = append(vms, f.vm(h, vm))
		}
	}
	return append(macs, vms...)
}

func (f finder) vm(h model.MacView, vm model.VM) entry {
	word, signal, dot := f.state(vm)
	kind := vm.Lifetime.Kind()
	glyph := "workspace"
	switch {
	case kind == machine.Server:
		glyph = "server"
	case vm.Agent != nil:
		glyph = "agent"
	}
	row := page.Row{ID: vm.Name, Label: vm.Name, Detail: page.Texts(word, f.mac(h.Name)), Glyph: page.Glyph{Kind: glyph, Signal: signal, Dot: dot}, Nav: machinePage(vm.Name)}
	tags := []string{}
	if vm.Agent != nil {
		tags = append(tags, "agent")
	}
	tags = append(tags, strings.Fields(word)[0], "vm", strings.ToLower(model.KindWord(kind)))
	e := entry{row: row, tags: tags, verb: opens("open:"+vm.Name, "")}
	if kind != machine.Server && vm.State == model.Running {
		attach := attachOf(vm)
		e.alt = &attach
	}
	return e
}

// value: a VM's state in a word or two, as its row says it: a down server
// since when, a server's health, an agent's state, else the VM's own.
func (f finder) state(vm model.VM) (string, page.Signal, page.Dot) {
	switch {
	case model.Down(vm):
		if since := vm.CheckStatus.Since; !since.IsZero() {
			return "down since " + f.at(since), page.SignalFailing, page.DotFill
		}
		return "down", page.SignalFailing, page.DotFill
	case vm.State != model.Running:
		return string(vm.State), page.SignalQuiet, page.DotRing
	case vm.Lifetime == model.AlwaysOn:
		if h := health(vm); h != "healthy" {
			return h, page.SignalWaits, page.DotFill
		}
		return "healthy", page.SignalWell, page.DotFill
	case vm.Agent != nil && vm.Agent.State == model.AgentFinished:
		return vm.Agent.State, page.SignalDone, page.DotFill
	case vm.Agent != nil && vm.Agent.State == "waiting":
		return vm.Agent.State, page.SignalWaits, page.DotFill
	case vm.Agent != nil && vm.Agent.State != "":
		return vm.Agent.State, page.SignalActive, page.DotFill
	}
	return string(vm.State), page.SignalWell, page.DotFill
}

func attachOf(vm model.VM) page.Action {
	return page.Action{ID: "attach:" + vm.Name, Label: "Open in Terminal", Key: "cmd+enter", Run: &page.Run{Terminal: true, Argv: []string{"--on", vm.Name}},
		Hud: hud("Opened " + vm.Name + " in Terminal")}
}

func (f finder) ports() []entry {
	// value: every shared port by its software and its address, opened when
	// it is a web page, else copied; then every port a VM listens on that
	// is not shared, shared by its verb.
	if f.r.Ports == nil {
		return nil
	}
	var shared, listening []entry
	taken := map[string]bool{}
	for _, h := range f.r.Ports.Macs {
		for _, s := range h.Shared {
			named := s.Named()
			addr := s.VM + ":" + strconv.Itoa(s.GuestPort)
			taken[addr] = true
			url := s.URLs.Public
			row := page.Row{ID: "shared:" + addr, Label: named.Name, Detail: page.Texts(addr), Glyph: portGlyph(named)}
			e := entry{row: row, tags: portTags(named, s.GuestPort, s.VM, "shared")}
			copyLink := page.Action{ID: "copy:" + addr, Label: VerbCopy, Run: &page.Run{Copy: url}, Hud: hud("Copied " + url)}
			if s.HTTP && url != "" {
				e.verb = page.Action{ID: "open-url:" + addr, Label: VerbOpen, Run: &page.Run{Open: url}}
				copyLink.Label, copyLink.Key = "Copy link", "cmd+enter"
				e.alt = &copyLink
			} else {
				e.verb = copyLink
			}
			shared = append(shared, e)
		}
	}
	for _, h := range f.r.Ports.Macs {
		for _, p := range h.Ports {
			if p.Machine.On == "" {
				continue
			}
			addr := p.Key()
			if taken[addr] {
				continue
			}
			named := p.Named()
			row := page.Row{ID: "listening:" + addr, Label: named.Name, Detail: page.Texts(addr), Glyph: portGlyph(named)}
			if p.Reach == ports.Outside {
				row.Detail = page.Text("public")
			}
			e := entry{row: row, tags: portTags(named, p.Port, p.Machine.Name, "listening")}
			if f.workspace(p.Machine.Name) {
				e.verb = page.Action{ID: "share:" + addr, Label: VerbShare, Mutates: true,
					Run: &page.Run{Call: CallSharePort, In: map[string]string{"VM": p.Machine.Name, "Port": strconv.Itoa(p.Port)}, Target: macTarget(h.Name, f.self)},
					Hud: hud("Shared " + named.Name + " of " + p.Machine.Name)}
			} else {
				row.Nav = page.ID(string(PortsID) + ":" + p.Machine.Name)
				e.row, e.verb = row, opens("open-ports:"+addr, "")
			}
			listening = append(listening, e)
		}
	}
	return append(shared, listening...)
}

// value: a running workspace, the one kind whose listener the menu bar
// shares.
func (f finder) workspace(name string) bool {
	_, vm, ok := model.FindVM(*f.r.All, name)
	return ok && vm.Lifetime.Kind() != machine.Server && vm.State == model.Running
}

func portGlyph(n catalog.PortName) page.Glyph {
	if n.Logo != "" {
		return page.Glyph{Kind: "port:" + n.Logo}
	}
	return symbol("network", "")
}

func portTags(n catalog.PortName, port int, vm, state string) []string {
	number := strconv.Itoa(port)
	return []string{"port", number, vm + ":" + number, strings.ToLower(n.Name), vm, state}
}

func (f finder) sections() []entry {
	// value: Ports of and Activity of each awake Mac, then the three pages
	// of every machine, each with its key.
	var out []entry
	for _, h := range f.r.All.Macs {
		if !awakeMac(h) {
			continue
		}
		name := f.mac(h.Name)
		portsOf, activityOf := page.ID(string(PortsID)+":"+h.Name), page.ID(string(ActivityID)+":"+h.Name)
		out = append(out,
			entry{row: page.Row{ID: "ports:" + h.Name, Label: "Ports of " + name, Glyph: symbol("network", ""), Nav: portsOf}, tags: []string{"ports", "share"}, verb: opens("open-ports:"+h.Name, "")},
			entry{row: page.Row{ID: "activity:" + h.Name, Label: "Activity of " + name, Glyph: symbol("cpu", ""), Nav: activityOf}, tags: []string{"activity", "cpu", "chart"}, verb: opens("open-activity:"+h.Name, "")},
		)
	}
	every := []struct {
		id, label, where, symbol, key string
		nav                           page.ID
		tags                          []string
	}{
		{"ports", "Ports", "All machines", "network", "cmd+p", PortsID, []string{"ports", "every", "machines"}},
		{"accounts", "Accounts", "All machines", "key", "cmd+u", AccountsID, []string{"accounts", "sign-in", "share"}},
		{"awake", "Awake", "All Macs", "sun.max", "cmd+e", AwakeID, []string{"awake", "sleep"}},
	}
	for _, s := range every {
		out = append(out, entry{row: page.Row{ID: s.id, Label: s.label, Detail: page.Texts(s.where), Glyph: symbol(s.symbol, ""), Nav: s.nav}, tags: s.tags, verb: opens("open-"+s.id, s.key)})
	}
	return out
}

func (f finder) actions() []entry {
	// value: SPEC B5.6's actions available now: This Mac's keep-awake, never
	// shortening the term it holds; New VM on each awake Mac, the roomiest
	// first and on ⌘N; each server's restart, asking first while it is
	// healthy; each Mac's clean-up of its finished throwaways.
	out := f.awake()
	roomiest := model.Roomiest(*f.r.All)
	if roomiest == "" {
		roomiest = f.self
	}
	type room struct {
		h    model.MacView
		free int64
	}
	var rooms []room
	for _, h := range f.r.All.Macs {
		if !awakeMac(h) {
			continue
		}
		_, n, _ := free(h)
		rooms = append(rooms, room{h, n})
	}
	slices.SortStableFunc(rooms, func(a, b room) int { return cmp.Compare(b.free, a.free) })
	for _, m := range rooms {
		a := newVM()
		a.ID, a.Key, a.Priority = "new-vm:"+m.h.Name, "", 0
		a.Run.Target = macTarget(m.h.Name, f.self)
		a.Form.Subtitle = "On " + f.mac(m.h.Name)
		if m.h.Name == roomiest {
			a.Key = "cmd+n"
		}
		row := page.Row{ID: "new-vm:" + m.h.Name, Label: "New VM on " + f.mac(m.h.Name), Glyph: symbol("plus", "")}
		if said, _, ok := free(m.h); ok {
			row.Detail = page.Text(said)
		}
		out = append(out, entry{row: row, tags: []string{"create", "vm", "workspace", "server", "new"}, verb: a})
	}
	for _, h := range f.r.All.Macs {
		for _, vm := range model.Ordered(h.VMs) {
			if vm.Lifetime != model.AlwaysOn {
				continue
			}
			out = append(out, f.restart(h, vm))
		}
	}
	for _, h := range f.r.All.Macs {
		if e, ok := f.cleanup(h); ok {
			out = append(out, e)
		}
	}
	return out
}

func (f finder) awake() []entry {
	r := f.r.Awake
	if r == nil || r.Engine != awake.EngineRunning || r.NeverSleeps {
		return nil
	}
	in := map[string]string{"For": nanos(awakeFor)}
	label, until := "Keep This Mac awake 4 hr", f.r.Now.Add(awakeFor)
	if s := r.Session; s != nil {
		in = flags(s.Lid, s.Battery)
		in["For"] = nanos(awakeFor)
		if s.Until.Sub(f.r.Now) > awakeFor {
			until = s.Until.Add(awakeFor)
			label = "Extend awake to " + f.at(until)
			delete(in, "For")
			in["Until"] = moment(until)
		}
	}
	verb := page.Action{ID: "awake-for", Label: "Keep awake", Mutates: true, Run: &page.Run{Call: CallKeepAwake, In: in},
		Hud: hud("This Mac stays awake until " + f.at(until))}
	return []entry{{row: page.Row{ID: "awake", Label: label, Glyph: symbol("sun.max", "")}, tags: []string{"awake", "caffeinate", "keep"}, verb: verb}}
}

func (f finder) restart(h model.MacView, vm model.VM) entry {
	row := page.Row{ID: "restart:" + vm.Name, Label: "Restart " + vm.Name, Glyph: symbol("arrow.clockwise", "")}
	verb := page.Action{ID: "restart:" + vm.Name, Label: VerbRestart, Mutates: true,
		Run: &page.Run{Call: CallRestartVM, In: map[string]string{"Name": vm.Name}, Target: target(h)}}
	switch {
	case model.Down(vm):
		row.Detail = page.Text("down")
	case f.r.Asks != nil:
		verb.Confirm = ask(f.r.Asks, "restart", machine.Server, vm.Name, "Restart "+vm.Name+"?", "restart")
		verb.Run.In = answered(verb.Confirm, verb.Run.In)
		if verb.Confirm != nil {
			row.Detail = page.Text("asks first")
		}
	}
	return entry{row: row, tags: []string{"restart", "server", "reboot"}, verb: verb}
}

func (f finder) cleanup(h model.MacView) (entry, bool) {
	gone, _ := model.Cleanup(h.VMs)
	if len(gone) == 0 || !awakeMac(h) {
		return entry{}, false
	}
	cands := make([]model.Candidate, 0, len(gone))
	for _, vm := range gone {
		cands = append(cands, model.CandidateOf(vm, f.r.Now))
	}
	c := pagemachines.CleanupConfirm(cands)
	verb := page.Action{ID: "cleanup:" + h.Name, Label: VerbCleanUp, Mutates: true, Confirm: &c,
		Run: &page.Run{Call: CallCleanup, In: map[string]string{"Confirm": yes}, Target: macTarget(h.Name, f.self)}}
	row := page.Row{ID: "cleanup:" + h.Name, Label: "Clean up throwaways on " + f.mac(h.Name), Glyph: symbol("trash", "")}
	return entry{row: row, tags: []string{"delete", "throwaway", "finished"}, verb: verb}, true
}

func (f finder) verbs(p *page.Page) page.Section {
	// value: SPEC A6: a VM's verbs, which the app offers once two letters
	// begin its name: Open in Terminal for a running workspace, Share a port
	// of it, Lend an account to a workspace, Restart for a server.
	s := page.Section{ID: FindVerbs, Title: "Verbs", Columns: findColumns(), Uncounted: true}
	for _, h := range f.r.All.Macs {
		for _, vm := range model.Ordered(h.VMs) {
			var es []entry
			server := vm.Lifetime.Kind() == machine.Server
			running := vm.State == model.Running
			if !server && running {
				attach := attachOf(vm)
				attach.ID = "verb-attach:" + vm.Name
				es = append(es, entry{row: page.Row{ID: "attach:" + vm.Name, Label: "Open " + vm.Name + " in Terminal", Glyph: symbol("terminal", "")}, verb: attach})
			}
			if running {
				share := page.Action{ID: "verb-share:" + vm.Name, Label: VerbShare, Mutates: true,
					Run:  &page.Run{Call: CallSharePort, In: map[string]string{"VM": vm.Name}, Target: target(h)},
					Form: &page.Form{Title: "Share a port of " + vm.Name, Fields: []page.Field{{Key: "Port", Label: "Port", Kind: page.FieldText, Required: true, Note: "a port number, like 5173"}}}}
				es = append(es, entry{row: page.Row{ID: "share:" + vm.Name, Label: "Share a port of " + vm.Name, Glyph: symbol("network", "")}, verb: share})
			}
			if shareAccount, ok := f.shareAccount(h, vm); !server && running && ok {
				es = append(es, entry{row: page.Row{ID: "share-account:" + vm.Name, Label: "Lend an account to " + vm.Name, Glyph: symbol("key", "")}, verb: shareAccount})
			}
			if server {
				restart := f.restart(h, vm).verb
				restart.ID = "verb-restart:" + vm.Name
				es = append(es, entry{row: page.Row{ID: "restart:" + vm.Name, Label: "Restart " + vm.Name, Glyph: symbol("arrow.clockwise", "")}, verb: restart})
			}
			if len(es) == 0 {
				continue
			}
			g := page.Group{Head: page.Row{ID: vm.Name, Label: vm.Name}}
			for _, e := range es {
				row := f.add(p, e)
				g.Items = append(g.Items, page.Item{Row: &row})
			}
			s.Items = append(s.Items, page.Item{Group: &g})
		}
	}
	return s
}

func (f finder) shareAccount(h model.MacView, vm model.VM) (page.Action, bool) {
	// value: Lend an account: one of the accounts the VM's Mac is signed in
	// to, for 4 hr; none to share is no verb.
	if f.r.Accounts == nil {
		return page.Action{}, false
	}
	var opts []page.Option
	for _, m := range f.r.Accounts.Macs {
		if m.Name != h.Name {
			continue
		}
		for _, a := range m.Accounts {
			if a.Origin == accounts.OriginOwn && a.State == catalog.StateOK {
				opts = append(opts, page.Option{Value: a.Provider, Label: accounts.Label(a.Provider)})
			}
		}
	}
	if len(opts) == 0 {
		return page.Action{}, false
	}
	return page.Action{ID: "verb-share-account:" + vm.Name, Label: VerbRun, Mutates: true,
		Run: &page.Run{Call: CallShareAccount, In: map[string]string{"With": vm.Name, "For": nanos(extendBy)}, Target: macTarget(h.Name, f.self)},
		Form: &page.Form{Title: "Lend an account to " + vm.Name, Subtitle: "From " + f.mac(h.Name), Fields: []page.Field{
			{Key: "Account", Label: "Account", Kind: page.FieldSelect, Required: true, Default: opts[0].Value, Options: opts},
		}}}, true
}
