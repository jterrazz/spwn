package menubar

import (
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/accounts"
	"github.com/jterrazz/studio-station/src/agents"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/finding"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/core/section"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/overview"
	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/ports"
)

const (
	NeedsID page.ID = "menubar.needs"

	SectionUpkeep = "upkeep"

	VerbReview  = "Review"
	VerbRestart = "Restart"
	VerbExtend  = "Extend"
	VerbHowTo   = "How to"
	VerbSignIn  = "Sign in"
	VerbShow    = "Show"
	VerbUpdate  = "Update"
	VerbCopyFix = "Copy fix"

	CallShareAccount = "accounts.share"
	extendBy         = 4 * time.Hour
)

type NeedsReading struct {
	Now  time.Time
	Zone *time.Location
	All  *model.AllMachines
	// value: each part nil is still being read, or did not answer.
	Ports    *ports.AllMachines
	Accounts *accounts.AllMachines
	// value: each answering Mac's Overview, This Mac's first.
	Overviews []overview.Summary
}

type Need struct {
	// value: SPEC B5.7: one thing that needs the owner, or one upkeep item:
	// its finding, which ranks it, and its words in three lengths: the hero's
	// title, the row (its Subject, then What) and Find's, the verb first,
	// each in parts the surface joins.
	Finding finding.Finding
	Key     string
	Glyph   page.Glyph
	Hero    string
	Subject string
	What    string
	Find    []string
	// value: the hero's facts line, and an upkeep item's detail.
	Facts  []page.Value
	Detail []string
	Verb   page.Action
	// value: the verb on its Mac's own page, when it differs there.
	Own *page.Action
	// value: the page the row opens, and the one a verb with no call
	// opens: a verb that runs nothing is the row's own Nav.
	Nav page.ID
	// value: the machines it is of: its Mac, then its VM.
	Scope []string
}

func (n Need) of(name string) bool {
	for _, s := range n.Scope {
		if s == name {
			return true
		}
	}
	return false
}

func needsOf(r NeedsReading, self string) (needs, upkeep []Need) {
	// invariant: SPEC B5.7: what needs the owner, ranked by finding.Rank (a
	// server down, a port exposed, an agent waiting on the owner, a share
	// ending under 30 min, soonest first), and upkeep apart, ranked by
	// finding.RankUpkeep, never in the badge. Two reds are two needs.
	w := needSaid{r: r, self: self}
	if r.All != nil {
		for _, h := range r.All.Macs {
			for _, vm := range h.VMs {
				switch {
				case model.Down(vm):
					needs = append(needs, w.down(h, vm))
				case vm.Agent != nil && vm.Agent.State == string(agents.Waiting):
					needs = append(needs, w.waiting(h, vm))
				}
			}
		}
	}
	if r.Ports != nil {
		for _, h := range r.Ports.Macs {
			for _, p := range h.Ports {
				if p.Red() {
					needs = append(needs, w.exposed(h.Name, p))
				}
			}
		}
	}
	if r.Accounts != nil {
		for _, c := range r.Accounts.Read().Shared {
			if c.Short {
				needs = append(needs, w.ending(c))
			}
		}
	}
	for _, s := range r.Overviews {
		upkeep = append(upkeep, w.upkeep(s)...)
	}
	of := func(n Need) finding.Finding { return n.Finding }
	return finding.RankBy(needs, of), finding.RankUpkeep(upkeep, of)
}

type needSaid struct {
	r    NeedsReading
	self string
}

func (w needSaid) mac(name string) string { return page.MacName(name, w.self) }

func (w needSaid) at(t time.Time) string {
	zone := w.r.Zone
	if zone == nil {
		zone = time.UTC
	}
	return t.In(zone).Format("15:04")
}

func (w needSaid) down(h model.MacView, vm model.VM) Need {
	what := "down"
	facts := []page.Value{}
	if vm.Cause != "" {
		facts = append(facts, page.Text(vm.Cause))
	}
	if since := vm.CheckStatus.Since; !since.IsZero() {
		what += " since " + w.at(since)
		facts = append(facts, page.Text("since "+w.at(since)))
	}
	facts = append(facts, page.Text("on "+w.mac(h.Name)))
	key := "down:" + vm.Name
	return Need{
		Finding: finding.Finding{Machine: machine.Ref{Name: vm.Name, On: h.Name}, Section: section.Overview, Severity: finding.Broken, Urgency: finding.Red, Subject: vm.Name},
		Key:     key, Glyph: symbol("xmark.octagon.fill", page.SignalFailing), Hero: vm.Name + " is down", Subject: vm.Name, What: what,
		Find: []string{VerbRestart + " " + vm.Name}, Facts: facts, Nav: machinePage(vm.Name), Scope: []string{h.Name, vm.Name},
		Verb: page.Action{ID: key, Label: VerbRestart, On: page.OnRow, Rows: []string{key}, Mutates: true, Default: true,
			Run: &page.Run{Call: CallRestartVM, In: map[string]string{"Name": vm.Name}, Target: target(h)}},
	}
}

func (w needSaid) waiting(h model.MacView, vm model.VM) Need {
	a := vm.Agent
	what := a.Ask
	if what == "" {
		what = "waits for your review"
	}
	facts := []page.Value{page.Text(what)}
	if !a.Since.IsZero() {
		facts = append(facts, page.Text(minutes(w.r.Now.Sub(a.Since))))
	}
	key := "waiting:" + vm.Name
	return Need{
		Finding: finding.Finding{Machine: machine.Ref{Name: vm.Name, On: h.Name}, Section: section.Overview, Severity: finding.NeedsALook, Urgency: finding.Blocking, Subject: vm.Name},
		Key:     key, Glyph: symbol("text.bubble", page.SignalWaits), Hero: vm.Name + " is waiting", Subject: vm.Name, What: what,
		Find: []string{VerbReview + " " + vm.Name, what}, Facts: facts, Nav: machinePage(vm.Name), Scope: []string{h.Name, vm.Name},
		Verb: page.Action{ID: key, Label: VerbReview, On: page.OnRow, Rows: []string{key}, Default: true},
	}
}

func (w needSaid) exposed(mac string, p ports.Port) Need {
	where := w.mac(p.Machine.Mac())
	what := "port " + strconv.Itoa(p.Port) + " exposed"
	facts := []page.Value{page.Text(where)}
	if app := catalog.AppOf(p.Process); app != "" {
		facts = append(facts, page.Text(app))
	}
	key := "exposed:" + p.Key()
	scope := []string{mac}
	if p.Machine.On != "" {
		scope = append(scope, p.Machine.Name)
	}
	return Need{
		Finding: finding.Finding{Machine: p.Machine, Section: section.Ports, Severity: finding.Broken, Urgency: finding.Red, Subject: strconv.Itoa(p.Port)},
		Key:     key, Glyph: symbol("exclamationmark.shield", page.SignalFailing), Hero: "Port " + strconv.Itoa(p.Port) + " is exposed", Subject: where, What: what,
		Find: []string{"Port " + strconv.Itoa(p.Port) + " is exposed"}, Facts: facts, Nav: page.ID(string(PortsID) + ":" + mac), Scope: scope,
		Verb: page.Action{ID: key, Label: VerbReview, On: page.OnRow, Rows: []string{key}, Default: true},
	}
}

func (w needSaid) ending(c accounts.SharedCopy) Need {
	account := accounts.Label(c.Provider)
	in := "ends in " + minutes(c.Left)
	key := "ending:" + c.Mac + "/" + c.VM + "/" + c.Provider
	run := &page.Run{Call: CallShareAccount, In: map[string]string{"Account": c.Provider, "With": c.VM, "For": strconv.FormatInt(int64(extendBy), 10)}}
	if c.Mac != w.self {
		run.Target = &page.Target{Name: c.Mac}
	}
	return Need{
		Finding: finding.Finding{Machine: machine.Ref{Name: c.VM, On: c.Mac}, Section: section.Accounts, Severity: finding.NeedsALook, Urgency: finding.Expiring, Ends: w.r.Now.Add(c.Left), Subject: c.Provider},
		Key:     key, Glyph: symbol("clock", page.SignalWaits), Hero: account + " on " + c.VM + " " + in, Subject: c.VM, What: account + " " + in,
		Find:  []string{VerbExtend + " " + account + " on " + c.VM},
		Facts: []page.Value{page.Text("shared from " + w.mac(c.Mac)), page.Text("ends " + w.at(w.r.Now.Add(c.Left)))},
		Nav:   machinePage(c.VM), Scope: []string{c.Mac, c.VM},
		Verb: page.Action{ID: key, Label: VerbExtend, On: page.OnRow, Rows: []string{key}, Mutates: true, Default: true, Run: run,
			Hud: hud(account+" on "+c.VM, minutes(c.Left+extendBy)+" left")},
	}
}

func (w needSaid) upkeep(s overview.Summary) []Need {
	// invariant: SPEC B5.7, B5.18: a Mac's upkeep is its Overview's findings
	// that block nothing: a setting, a background task, a sign-in, its
	// updates (one item for them all), a folder; each with its detail and
	// its verb.
	mac := s.Head.Name
	label := w.mac(mac)
	var out []Need
	updates := 0
	var update *Need
	for i, n := range s.Needs {
		ref := machine.Ref{Name: mac}
		if n.Machine != "" && n.Machine != mac {
			ref = machine.Ref{Name: n.Machine, On: mac}
		}
		base := finding.Finding{Machine: ref, Section: n.Section, Severity: n.Severity, Subject: n.Subject, Text: n.Text}
		key := "upkeep:" + mac + "/" + string(n.Section) + "/" + strconv.Itoa(i)
		item := Need{Finding: base, Key: key, Subject: label, Nav: machinePage(mac), Scope: []string{mac}, Detail: []string{n.Text}}
		if ref.On != "" {
			item.Scope = append(item.Scope, ref.Name)
		}
		row := func(title, what, verb string) {
			item.Hero, item.What, item.Find = title, what, []string{title}
			item.Verb = page.Action{ID: key, Label: verb, On: page.OnRow, Rows: []string{key}}
		}
		switch n.Section {
		case section.Settings:
			row(n.Text+" on "+label, strings.ToLower(n.Text), VerbHowTo)
			item.Glyph = symbol("shield", page.SignalWaits)
			item.Verb.Sheet = page.SheetHint + n.Subject
			if n.Path != "" {
				item.Detail = []string{"station never changes it for you"}
			}
		case section.Accounts:
			account := accounts.Label(n.Subject)
			row(account+" needs sign-in on "+label, account+" needs sign-in", VerbSignIn)
			item.Find = []string{VerbSignIn + " to " + account + " on " + label}
			item.Glyph = symbol("key", page.SignalWaits)
			item.Detail = []string{"it cannot be shared until you sign in"}
			item.Verb.Run = &page.Run{Terminal: true, Argv: onMac([]string{"accounts"}, mac, w.self)}
		case section.Background:
			title, what := "A login item failed on "+label, "1 login item failed"
			if !n.Broken {
				title, what = "A background task is left over on "+label, "1 task left over"
			}
			row(title, what, VerbShow)
			item.Glyph = symbol("exclamationmark.triangle", page.SignalWaits)
			if n.Command != "" {
				own := item.Verb
				own.Label, own.Run, own.Hud = VerbCopyFix, &page.Run{Copy: n.Command}, hud("Copied "+n.Command)
				item.Own = &own
			}
		case section.Apps:
			if !n.Broken && n.Call == CallUpdate {
				updates += leading(n.Text)
				if update == nil {
					item.Glyph = symbol("arrow.down.circle", page.SignalWaits)
					item.Verb = page.Action{ID: key, Label: VerbUpdate, On: page.OnRow, Rows: []string{key}, Mutates: true,
						Run: &page.Run{Call: CallUpdate, Target: macTarget(mac, w.self)}, Hud: hud("Everything is up to date")}
					update = &item
				} else {
					update.Detail = append(update.Detail, n.Text)
				}
				continue
			}
			row(n.Text+" on "+label, n.Text, VerbShow)
			item.Glyph = symbol("exclamationmark.triangle", page.SignalWaits)
		case section.Disk:
			row(n.Text+" on "+label, n.Text, VerbShow)
			item.Glyph = symbol("internaldrive", page.SignalWaits)
		default:
			continue
		}
		out = append(out, item)
	}
	if update != nil {
		said := counted(updates, "update", "updates")
		update.Hero, update.What, update.Find = said+" to install", said, []string{"Install " + said}
		update.Finding.Text = said
		out = append(out, *update)
	}
	return out
}

func hud(parts ...string) *page.Value {
	v := page.Texts(parts...)
	return &v
}

func symbol(name string, signal page.Signal) page.Glyph {
	return page.Glyph{Kind: "symbol:" + name, Signal: signal}
}

func machinePage(name string) page.ID { return page.ID(string(MachineID) + ":" + name) }

func macTarget(mac, self string) *page.Target {
	if mac == self {
		return nil
	}
	return &page.Target{Name: mac}
}

func onMac(argv []string, mac, self string) []string {
	if mac == self {
		return argv
	}
	return append(argv, "--on", mac)
}

func minutes(d time.Duration) string {
	// value: a wait or a time left as the panel says it: "6 min", "2 hr",
	// "4 hr 24 min"; never under 1 min.
	m := max(1, int(d.Round(time.Minute).Minutes()))
	if m < 60 {
		return strconv.Itoa(m) + " min"
	}
	if r := m % 60; r != 0 {
		return strconv.Itoa(m/60) + " hr " + strconv.Itoa(r) + " min"
	}
	return strconv.Itoa(m/60) + " hr"
}

func NeedsPage(r NeedsReading, of string, thisMac page.Context) page.Page {
	// invariant: SPEC A4, B5.7: menubar.needs lists every need with its verb,
	// most urgent first, then the upkeep apart; menubar.needs:<machine>
	// lists those of that machine (a Mac's with its VMs'). Nothing read yet
	// is the page's frame.
	if r.All == nil {
		return Opening(NeedsID, thisMac)
	}
	needs, upkeep := needsOf(r, thisMac.Machine)
	where := "All machines"
	if of != "" {
		needs, upkeep = scoped(needs, of), scoped(upkeep, of)
		where = page.MacName(of, thisMac.Machine)
		for i, n := range upkeep {
			if n.Own != nil && ownWords(n, of, thisMac.Machine) {
				upkeep[i].Verb = *n.Own
			}
		}
	}
	p := page.Page{ID: NeedsID, Command: "menubar", Context: thisMac, Header: &page.Header{Back: "All machines"}}
	if of != "" {
		p.Header.Back = where
	}
	facts := []page.Value{page.Text(where)}
	if len(needs) > 0 {
		facts = append(facts, page.Text(thingsFirst(len(needs))))
	}
	p.Sections = append(p.Sections, page.Section{ID: "summary", Title: "Needs you", Role: page.Summary, Caption: facts})
	list := page.Section{ID: SectionNeeds, Title: "Needs you", Card: page.CardFlush, Columns: columns(), Items: needRows(needs, of, thisMac.Machine)}
	if len(needs) == 0 {
		list.Items = []page.Item{{Empty: &page.Empty{Text: "Nothing needs you", Sub: "nothing blocks an agent or a server"}}}
	}
	p.Sections = append(p.Sections, list)
	if len(upkeep) > 0 {
		p.Sections = append(p.Sections, page.Section{ID: SectionUpkeep, Title: "Upkeep", Card: page.CardFlush, Columns: columns(), Uncounted: true,
			Caption: []page.Value{page.Text(strconv.Itoa(len(upkeep))), page.Text("when you have a minute")}, Items: needRows(upkeep, of, thisMac.Machine)})
	}
	for _, n := range append(needs, upkeep...) {
		p.Actions = append(p.Actions, n.Verb)
	}
	return page.Seal(p)
}

func scoped(ns []Need, of string) []Need {
	var out []Need
	for _, n := range ns {
		if n.of(of) {
			out = append(out, n)
		}
	}
	return out
}

func needRows(ns []Need, of, self string) []page.Item {
	out := make([]page.Item, 0, len(ns))
	for _, n := range ns {
		row := page.Row{ID: n.Key, Label: n.Subject, Detail: page.Text(n.What), Glyph: n.Glyph, Verb: n.Verb.ID, Nav: n.Nav, Mark: page.Attention, Reason: reasonOf(n.Finding)}
		if n.Finding.Urgency == finding.Red {
			row.Mark = page.Broken
		}
		if ownWords(n, of, self) {
			row.Label, row.Detail = n.What, page.None()
			if n.Finding.Section != section.Accounts {
				row.Label = titled(n.What)
			}
			switch {
			case len(n.Detail) > 0:
				row.Detail = page.Texts(n.Detail...)
			case len(n.Facts) > 0:
				row.Detail = page.List(n.Facts...)
			}
		}
		out = append(out, page.Item{Row: &row})
	}
	return out
}

func ownWords(n Need, of, self string) bool {
	return of != "" && (n.Subject == of || n.Subject == page.MacName(of, self))
}

func reasonOf(f finding.Finding) page.Reason {
	switch {
	case f.Urgency == finding.Red && f.Section == section.Ports:
		return page.Exposed
	case f.Urgency == finding.Red:
		return page.Down
	case f.Urgency == finding.Blocking:
		return page.Waiting
	case f.Urgency == finding.Expiring:
		return page.Expiring
	case f.Severity == finding.Broken:
		return page.Failing
	}
	return page.Degraded
}

func thingsFirst(n int) string {
	if n == 1 {
		return "1 thing"
	}
	return strconv.Itoa(n) + " things, most urgent first"
}
