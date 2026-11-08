package menubar

import (
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/core/finding"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/overview"
	"github.com/jterrazz/studio-station/src/page"
)

const (
	SectionHero = "hero"
	RowHero     = "hero"
	LinkNeeds   = "more-needs"

	heroRows = 3
)

// value: what the hero is reading, by Reading: the needs ranked, the Mac
// still answering, and the facts of a fleet that is fine.
type heroReading struct {
	r     Reading
	self  string
	needs []Need
	stale string
}

func (r Reading) needsReading() NeedsReading {
	overviews := make([]overview.Summary, 0, len(r.Away)+1)
	if r.Home != nil {
		overviews = append(overviews, *r.Home)
	}
	overviews = append(overviews, r.Away...)
	return NeedsReading{Now: r.now(), Zone: r.Zone, All: r.All, Ports: r.Ports, Accounts: r.Accounts, Overviews: overviews}
}

// value: SPEC B5.15: the first Mac not heard from for 30 s that is not
// asleep, as the interface names it; empty when every Mac answers.
func staleMac(r Reading, self string) string {
	if r.All == nil {
		return ""
	}
	for _, h := range r.All.Macs {
		if !h.Asleep && model.Unheard(h, r.now()) {
			return page.MacName(h.Name, self)
		}
	}
	return ""
}

func (r Reading) now() time.Time {
	if !r.Now.IsZero() {
		return r.Now
	}
	if r.All != nil {
		return r.All.Read
	}
	return time.Time{}
}

// invariant: SPEC A4 Home 1: the hero is decided from the ranked needs, in
// one order: a server restarting, a red need (the first ranked; two reds
// are never merged), exactly one need, several (the count, the badge's
// number, then at most three rows and a link to the rest), nothing (the
// fleet's facts). An empty fleet has no hero but the empty state. A Mac
// still answering dims the needs that are of it and says it checks it.
func hero(r Reading, self string, ranked []Need) (page.Section, *page.Value, bool) {
	s := page.Section{ID: SectionHero, Title: "Needs you", Role: page.Summary, Card: page.CardHero, Columns: columns()}
	if r.All == nil {
		return s, nil, false
	}
	h := heroReading{r: r, self: self, needs: ranked, stale: staleMac(r, self)}
	var stale *page.Value
	if h.stale != "" {
		v := page.Text("checking " + h.stale)
		v.Signal = page.SignalStale
		stale = &v
	}
	if empty(*r.All) {
		s.Card = page.CardEmpty
		s.Items = []page.Item{{Empty: h.empty()}}
		return s, stale, true
	}
	head, rest := h.top()
	s.Items = append([]page.Item{{Row: &head}}, rest...)
	return s, stale, true
}

func empty(all model.AllMachines) bool {
	for _, h := range all.Macs {
		if len(h.VMs) > 0 {
			return false
		}
	}
	return true
}

func (h heroReading) empty() *page.Empty {
	e := &page.Empty{Text: "No VMs yet", Glyph: symbol("shippingbox", page.SignalQuiet), Actions: []string{ActionNewVM}}
	for _, m := range h.r.All.Macs {
		if c, ok := model.CapacityOf(m); ok && m.ThisMac {
			e.Sub = strconv.FormatInt(c.Free>>30, 10) + " GB free on " + page.ThisMac
		}
	}
	return e
}

func (h heroReading) top() (page.Row, []page.Item) {
	if vm, mac, ok := restarting(*h.r.All); ok {
		head := page.Row{ID: RowHero, Label: "Restarting " + vm.Name, Mark: page.OK, Glyph: page.Glyph{Spin: true, Signal: page.SignalActive},
			Detail: page.List(dropFirst(page.Text("on "+page.MacName(mac, h.self)), 2), page.Text(strings.ToLower(vm.Restart.Name())), page.Text(vm.Restart.Counted()))}
		return head, h.also(h.needs)
	}
	for i, n := range h.needs {
		if n.Finding.Urgency != finding.Red {
			continue
		}
		others := append(append([]Need{}, h.needs[:i]...), h.needs[i+1:]...)
		head := page.Row{ID: RowHero, Label: n.Hero, Mark: page.Broken, Reason: reasonOf(n.Finding), Glyph: n.Glyph, Detail: heroFacts(n.Facts), Verb: n.Verb.ID, Nav: n.Nav}
		return head, h.also(others)
	}
	switch len(h.needs) {
	case 0:
		return page.Row{ID: RowHero, Label: "All machines are fine", Mark: page.OK, Glyph: symbol("checkmark.circle.fill", page.SignalWell), Detail: h.fine()}, nil
	case 1:
		n := h.needs[0]
		head := page.Row{ID: RowHero, Label: n.Hero, Mark: page.Attention, Reason: reasonOf(n.Finding), Glyph: symbol("exclamationmark.circle.fill", page.SignalWaits),
			Detail: heroFacts(n.Facts), Verb: n.Verb.ID, Nav: n.Nav, Signal: h.signal(n)}
		return head, nil
	}
	head := page.Row{ID: RowHero, Label: things(len(h.needs)), Mark: page.Attention, Reason: page.Waiting, Glyph: symbol("exclamationmark.circle.fill", page.SignalWaits)}
	rows := h.rows(h.needs[:min(heroRows, len(h.needs))])
	if rest := len(h.needs) - heroRows; rest > 0 {
		rows = append(rows, page.Item{Link: &page.Link{Label: moreNeedYou(rest), Page: NeedsID, Row: LinkNeeds}})
	}
	return head, rows
}

// value: the needs a busy or red hero leaves: the one row itself, or
// several folded into one line naming each subject once.
func (h heroReading) also(others []Need) []page.Item {
	switch len(others) {
	case 0:
		return nil
	case 1:
		return h.rows(others)
	}
	var names []string
	for _, n := range others {
		if !contains(names, n.Subject) {
			names = append(names, n.Subject)
		}
	}
	return []page.Item{{Link: &page.Link{Label: moreNeedYou(len(others)), Detail: page.Text(strings.Join(names, ", ")), Page: NeedsID, Row: LinkNeeds}}}
}

func (h heroReading) rows(ns []Need) []page.Item {
	out := make([]page.Item, 0, len(ns))
	for _, n := range ns {
		row := page.Row{ID: n.Key, Label: n.Subject, Detail: page.Text(n.What), Glyph: n.Glyph, Verb: n.Verb.ID, Nav: n.Nav, Mark: page.Attention, Reason: reasonOf(n.Finding), Signal: h.signal(n)}
		if n.Finding.Urgency == finding.Red {
			row.Mark = page.Broken
		}
		out = append(out, page.Item{Row: &row})
	}
	return out
}

// value: a need of the Mac still answering is drawn stale.
func (h heroReading) signal(n Need) page.Signal {
	if h.stale == "" || h.r.All == nil {
		return ""
	}
	for _, m := range h.r.All.Macs {
		if page.MacName(m.Name, h.self) == h.stale && n.of(m.Name) {
			return page.SignalStale
		}
	}
	return ""
}

// value: the fine hero's one facts line: each server's health and how long
// it is up (dropped first), whether every Mac is awake, the agents at work;
// a Mac still answering is the whole line.
func (h heroReading) fine() page.Value {
	if h.stale != "" {
		v := page.Text("checking " + h.stale)
		v.Signal = page.SignalStale
		return v
	}
	var parts []page.Value
	awake, working, servers := 0, 0, 0
	for _, m := range h.r.All.Macs {
		if !m.Asleep {
			awake++
		}
		for _, vm := range m.VMs {
			if vm.Agent != nil && vm.Agent.State == "working" {
				working++
			}
			if vm.Lifetime != model.AlwaysOn {
				continue
			}
			servers++
			switch vm.State {
			case model.Running:
				parts = append(parts, page.Text(vm.Name+" healthy"))
				if !vm.BootAt.IsZero() {
					parts = append(parts, dropFirst(page.Text("up "+upFor(h.r.now().Sub(vm.BootAt))), 3))
				}
			case model.Stopped:
				parts = append(parts, page.Text(vm.Name+" stopped"))
			case model.Suspended, model.Stale, model.Broken, model.UnknownState, model.Creating:
			}
		}
	}
	if n := len(h.r.All.Macs); n > 1 {
		said := "all " + strconv.Itoa(n) + " Macs answer"
		if awake < n {
			said = strconv.Itoa(awake) + " of " + strconv.Itoa(n) + " Macs awake"
		}
		parts = append(parts, dropFirst(page.Text(said), 1))
	}
	if working > 0 {
		parts = append(parts, dropFirst(page.Text(counted(working, "agent working", "agents working")), 2))
	}
	if servers == 0 {
		vms := 0
		for _, m := range h.r.All.Macs {
			vms += len(m.VMs)
		}
		parts = append(parts, page.Text(counted(len(h.r.All.Macs), "Mac", "Macs")), page.Text(counted(vms, "VM", "VMs")))
	}
	return page.List(parts...)
}

// value: a need's facts as the hero says them: when there are several, the
// last one goes first when the line is too long.
func heroFacts(vs []page.Value) page.Value {
	out := append([]page.Value{}, vs...)
	if len(out) > 1 {
		out[len(out)-1] = dropFirst(out[len(out)-1], 1)
	}
	return page.List(out...)
}

// value: a part the line drops by its priority, the highest first.
func dropFirst(v page.Value, pri int) page.Value {
	v.Pri = pri
	return v
}

func restarting(all model.AllMachines) (model.VM, string, bool) {
	for _, h := range all.Macs {
		for _, vm := range h.VMs {
			if vm.Lifetime == model.AlwaysOn && vm.Restart != nil {
				return vm, h.Name, true
			}
		}
	}
	return model.VM{}, "", false
}

func moreNeedYou(n int) string {
	if n == 1 {
		return "1 more needs you"
	}
	return strconv.Itoa(n) + " more need you"
}

// value: how long a server is up, as the panel says it: "12 days", "3 hr",
// "1 min".
func upFor(d time.Duration) string {
	switch {
	case d >= 24*time.Hour:
		return counted(int(d/(24*time.Hour)), "day", "days")
	case d >= time.Hour:
		return strconv.Itoa(int(d/time.Hour)) + " hr"
	}
	return strconv.Itoa(max(int(d/time.Minute), 1)) + " min"
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// value: the verbs of what needs you, one per need, for the hero's rows.
func verbs(ns []Need) []page.Action {
	out := make([]page.Action, 0, len(ns))
	for _, n := range ns {
		out = append(out, n.Verb)
	}
	return out
}
