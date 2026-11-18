package menubar

import (
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/overview"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagehome "github.com/jterrazz/studio-station/apps/engine/pages/home"
	pagemachines "github.com/jterrazz/studio-station/apps/engine/pages/machines"
)

const (
	SectionFootnote = "footnote"
	SectionStopped  = "stopped"
)

func workspacePage(m Machine, h model.MacView, vm model.VM) page.Page {
	// invariant: SPEC A4 A workspace: its Overview lists what needs you
	// there, its gauges, its ports and what its life means; its Ports segment lists
	// its ports by reach, its Accounts its loans; a stopped one says so with
	// Start and has no segment and no footer.
	p, _ := vmHead(m, h, vm)
	label := model.Label(*m.All, h)
	p.Header = &page.Header{Back: Back, Title: vm.Name, Glyph: page.Glyph{Kind: "workspace", Signal: page.SignalQuiet}, Buttons: []string{ActionMenu}}
	p.Sections = []page.Section{{ID: SectionSummary, Title: vm.Name, Role: page.Summary, Caption: workspaceFacts(m, h, vm, label)}}
	menu := page.Action{ID: ActionMenu, Label: "More options for " + vm.Name, Icon: "ellipsis", On: page.OnPage}
	shared, grants := len(workspacePorts(m, vm).Shares), grantedOn(m, vm.Name)
	stop := stopWorkspace(vm, label, shared).action(m.Asks, page.Action{ID: ActionStop, Label: "Stop", On: page.OnPage, Priority: 1, Mutates: true,
		Run: &page.Run{Call: CallStopVM, In: map[string]string{"VM": vm.Name}, Target: target(h)}})
	running := vm.State == model.Running
	if !running && !model.Down(vm) {
		stop = page.Action{ID: ActionStartVM, Label: "Start " + vm.Name, On: page.OnPage, Priority: 1, Mutates: true,
			Run: &page.Run{Call: CallStartVM, In: map[string]string{"VM": vm.Name}, Target: target(h)}, Hud: hud(vm.Name + " is starting")}
	}
	del := deleteWorkspace(vm, shared, grants).action(m.Asks, page.Action{ID: ActionDelete, Label: "Delete", On: page.OnPage, Priority: 2, Mutates: true,
		Run: &page.Run{Call: CallDeleteVM, In: map[string]string{"VM": vm.Name}, Target: target(h)}})
	p.Actions = []page.Action{stop, del}
	if running {
		menu.Submenu = append(menu.Submenu, ActionShare)
	}
	menu.Submenu = append(menu.Submenu, stop.ID, ActionDelete)
	if !running {
		p.Sections = append(p.Sections, stoppedCard(vm))
		p.Actions = append(p.Actions, menu)
		p.Header.Buttons = []string{ActionMenu}
		return page.Seal(p)
	}
	p.Segments = segments()
	if needs := vmNeeds(m, h, vm, label); needs != nil {
		p.Sections = append(p.Sections, *needs)
	}
	shares := workspacePorts(m, vm)
	p.Sections = append(p.Sections, vmGauges(m, vm), workspacePortsCard(shares), footnote(m, vm, label))
	p.Sections = append(p.Sections, shares.Sections...)
	signIns := vmAccountsSection(m, h, vm)
	signIns.Segment = SegmentAccounts
	p.Sections = append(p.Sections, signIns)
	p.Actions = append(p.Actions, sharePortSheets(shares.Listeners, []workspace{{VM: vm.Name, Mac: h.Name}}, m.self())...)
	if n, ok := needsOfVM(m, h, vm, label); ok {
		for _, need := range n {
			p.Actions = append(p.Actions, need.Verb)
		}
	}
	p.Actions = append(p.Actions, vmAccountActions(m, h, vm)...)
	p.Actions = append(p.Actions, shares.Actions...)
	shell := openIn("--on", vm.Name)
	shell.Key, shell.Icon, shell.Hud = "cmd+enter", "terminal", opened(vm.Name)
	ssh := catalog.SSHCommand("", vm.Name)
	p.Actions = append(p.Actions, shell, copied(ActionCopySSH, "Copy "+ssh, ssh), menu)
	p.Footer = &page.Footer{Kind: page.FooterSSH, Copy: &page.Value{Kind: page.KindCommand, Text: ssh}, Buttons: []string{ActionOpenStation}}
	p.Live = pageLive(p)
	return page.Seal(p)
}

// value: SPEC A4's facts of a workspace: its life, its Mac and how long it
// has run; a stopped one says it stopped. On
// a narrow line its uptime goes first, then its Mac, then its life: what
// it runs stays.
func workspaceFacts(m Machine, h model.MacView, vm model.VM, label string) []page.Value {
	out := append(vmStanding(vm), dropFirst(page.Text(lifeWord(vm)), 1), dropFirst(page.Linked(page.Text("on "+label), machinePage(h.Name)), 2))
	if vm.State != model.Running {
		return out
	}
	if !vm.BootAt.IsZero() && m.Now.After(vm.BootAt) {
		out = append(out, dropFirst(page.Text("up "+days(m.Now.Sub(vm.BootAt))), 3))
	}
	return out
}

// value: what needs you on the workspace.
func needsOfVM(m Machine, h model.MacView, vm model.VM, label string) ([]Need, bool) {
	out := scoped(overviewOf(m, h, label, false).needs, vm.Name)
	return out, len(out) > 0
}

func vmNeeds(m Machine, h model.MacView, vm model.VM, label string) *page.Section {
	needs, ok := needsOfVM(m, h, vm, label)
	if !ok {
		return nil
	}
	o := overviewOf(m, model.MacView{Name: vm.Name}, label, false)
	o.needs = needs
	s := o.needsCard()
	return &s
}

// value: the workspace's gauges, read inside it.
func vmGauges(m Machine, vm model.VM) page.Section {
	s := macOverview{m: m, h: model.MacView{Name: vm.Name}}.gauges()
	return s
}

// value: what a workspace's life means, and whether it pauses when its
// Mac sleeps.
func footnote(m Machine, vm model.VM, label string) page.Section {
	life := "Persistent: kept until you delete it"
	if vm.Lifetime == model.Ephemeral {
		life = "Ephemeral: deleted when it powers off from inside"
	}
	sleeps := "pauses when " + label + " sleeps"
	if m.Awake != nil && m.Awake.Mode() == awake.NeverSleeps {
		sleeps = "never paused"
	}
	note := page.Note{Text: life + "; " + sleeps + "."}
	return page.Section{ID: SectionFootnote, Title: "Life", Role: page.Banner, Segment: SegmentOverview, Uncounted: true, Items: []page.Item{{Note: &note}}}
}

// value: SPEC A4's stopped workspace: what it keeps and the memory it
// takes again, with Start.
func stoppedCard(vm model.VM) page.Section {
	sub := []string{"Disk kept"}
	if b := vm.Resources.MemoryBytes; b > 0 {
		sub = append(sub, "starts with "+pagemachines.Memory(b).Text)
	}
	empty := page.Empty{Glyph: page.Glyph{Kind: "workspace", Signal: page.SignalQuiet}, Text: vm.Name + " is stopped", Sub: strings.Join(sub, ", "), Actions: []string{ActionStartVM}}
	return page.Section{ID: SectionStopped, Title: vm.Name + " is stopped", Card: page.CardEmpty, Items: []page.Item{{Empty: &empty}}}
}

func pluralVerb(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func lineOf(s overview.Summary, id string) page.Value {
	for _, body := range pagehome.Body(s, nil, true) {
		for _, it := range body.Items {
			if body.ID == "sections" && it.Row != nil && it.Row.ID == id && len(it.Row.Cells) > 0 {
				return it.Row.Cells[0]
			}
		}
	}
	return page.None()
}

// value: a workspace's Overview Ports card: its shares, each opening its
// link with Copy and QR beside and its audience after its name, then its first listener offered with
// Share; nothing shared nor listening, the one line Nothing shared yet.
func workspacePortsCard(b portsBody) page.Section {
	items := make([]page.Item, 0, len(b.Shares)+1)
	for _, it := range b.Shares {
		row := *it.Row
		if word := b.Audience[row.ID]; !word.IsNone() && len(row.Cells) == 0 {
			row.Cells = []page.Value{signaled(word, page.SignalQuiet)}
		}
		items = append(items, page.Item{Row: &row})
	}
	if len(b.Listeners) > 0 {
		items = append(items, b.Listeners[0])
	}
	if len(items) == 0 {
		items = []page.Item{{Empty: &page.Empty{Text: "Nothing shared yet"}}}
	}
	return page.Section{ID: SectionPorts, Title: TitlePorts, Segment: SegmentOverview, Caption: []page.Value{count(len(b.Shares))}, Columns: columns(), Items: items}
}
