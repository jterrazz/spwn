package menubar

import (
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/overview"
	"github.com/jterrazz/studio-station/src/page"
	pagehome "github.com/jterrazz/studio-station/src/pages/home"
	pagemachines "github.com/jterrazz/studio-station/src/pages/machines"
	"slices"
	"strconv"
	"strings"
)

func workspacePage(m Machine, h model.MacView, vm model.VM) page.Page {
	p, head := vmHead(m, h, vm)
	kind := vm.Lifetime.Kind()
	p.Sections = []page.Section{head, activitySection(running(m, vm), vm.State == model.Running, "Activity", vm.Name), insideSection(m.Summary), sharedSection(m, h, vm)}
	var shares portsBody
	if vm.State == model.Running {
		for i := range p.Sections {
			p.Sections[i].Segment = SegmentOverview
		}
		shares = workspacePorts(m, vm)
		p.Segments = segments()
		p.Sections = append(p.Sections, workspacePortsCard(shares))
		p.Sections = append(p.Sections, shares.Sections...)
	}
	p.Sections = append([]page.Section{workspaceSummary(m, vm)}, p.Sections...)
	p.Header = &page.Header{Back: model.Label(*m.All, h), Glyph: page.Glyph{Kind: workspaceGlyph(vm)}, Buttons: []string{ActionMenu}}
	stop := page.Action{ID: ActionStop, Label: "Stop", On: page.OnPage, Priority: 1, Mutates: true,
		Confirm: ask(m.Asks, "stop", kind, vm.Name, "Stop "+vm.Name+"?", "stop", "Its disk is kept."),
		Run:     &page.Run{Call: CallStopVM, In: map[string]string{"VM": vm.Name}, Target: target(h)}}
	stop.Run.In = answered(stop.Confirm, stop.Run.In)
	if vm.State != model.Running && !model.Down(vm) {
		stop = page.Action{ID: ActionStartVM, Label: "Start", On: page.OnPage, Priority: 1, Mutates: true,
			Run: &page.Run{Call: CallStartVM, In: map[string]string{"VM": vm.Name}, Target: target(h)}}
	}
	del := page.Action{ID: ActionDelete, Label: "Delete", On: page.OnPage, Priority: 2, Mutates: true,
		Confirm: ask(m.Asks, "delete", kind, vm.Name, "Delete "+vm.Name+"?", "delete", "Its disk is erased: nothing of it is kept."),
		Run:     &page.Run{Call: CallDeleteVM, In: map[string]string{"VM": vm.Name}, Target: target(h)}}
	del.Run.In = answered(del.Confirm, del.Run.In)
	p.Actions = []page.Action{stop, del}
	if vm.State == model.Running {
		p.Actions = append(p.Actions, page.Action{ID: ActionShare, Label: "Share a port", On: page.OnPage, Priority: 3, Mutates: true,
			Run:  &page.Run{Call: CallSharePort, In: map[string]string{"VM": vm.Name}, Target: target(h)},
			Form: &page.Form{Title: "Share a port of " + vm.Name, Fields: []page.Field{{Key: "Port", Label: "Port", Kind: page.FieldText, Required: true, Note: "a port number, like 5173"}}}})
	}
	p.Actions = append(p.Actions, takeBacks(m, vm)...)
	if vm.State == model.Running {
		p.Actions = append(p.Actions, insideActions(vm.Name)...)
	}
	p.Actions = append(p.Actions, shares.Actions...)
	p.Actions = append(p.Actions, openIn("--on", vm.Name), page.Action{ID: ActionMenu, Label: "More options for " + vm.Name, Icon: "ellipsis", On: page.OnPage, Submenu: workspaceMenu(vm, stop.ID)})
	p.Live = liveKeys(p)
	return page.Seal(p)
}

// value: a workspace's Apps, Settings and Disk, as its Overview counts them
// inside it: station manages them there.
func insideSection(s *overview.Summary) page.Section {
	rows := []page.Row{{ID: "apps", Label: "Apps"}, {ID: "settings", Label: "Settings"}, {ID: "disk", Label: "Disk"}}
	for i := range rows {
		rows[i].Cells = []page.Value{page.Pending()}
		if s == nil {
			continue
		}
		rows[i].Cells = []page.Value{page.None()}
		for _, l := range s.Lines {
			if string(l.Section) == rows[i].ID {
				rows[i].Cells = []page.Value{inside(rows[i].ID, lineOf(*s, rows[i].ID))}
			}
		}
	}
	return page.Section{ID: SectionInside, Title: "Inside", Role: page.Summary, Columns: columns(), Items: items(rows)}
}

// value: an Inside line in the vision's few words (`38 installed`, `2
// need a look`, `12.4 GB free`): what is there and what needs you, never
// what station does not manage.
func inside(id string, v page.Value) page.Value {
	parts := v.Items
	if v.Kind != page.KindList {
		parts = []page.Value{v}
	}
	var out []page.Value
	look := 0
	for _, p := range parts {
		switch {
		case p.Label == string(page.Attention) || p.Label == string(page.Broken):
			switch {
			case p.Kind == page.KindCount:
				look += int(p.N)
			case id == "disk" && p.Kind == page.KindSize:
				out = append(out, p)
			default:
				look++
			}
		case id == "apps" && p.Label == string(page.OK):
			out = append(out, page.Count(p.N, "installed"))
		case id == "settings" && p.Label == string(page.Absent) && p.Kind == page.KindText:
			out = append(out, page.Text(p.Text+" set"))
		case id == "disk" && p.Kind == page.KindSize && p.Unit == "free":
			out = append(out, p)
		}
	}
	if look > 0 {
		out = append(out, page.Text(strconv.Itoa(look)+" "+pluralVerb(look, "needs", "need")+" a look"))
	}
	if len(out) == 0 {
		return v
	}
	return page.List(out...)
}

func pluralVerb(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// value: what opening Apps, Settings or Disk on a workspace's page does: a
// terminal on that family, on the workspace.
func insideActions(vm string) []page.Action {
	ids := []string{"apps", "settings", "disk"}
	out := make([]page.Action, 0, len(ids))
	for _, id := range ids {
		out = append(out, page.Action{ID: ActionOpenFamily + id, Label: "Open " + id, On: page.OnRow, Rows: []string{id}, Priority: 5, Run: &page.Run{Terminal: true, Argv: []string{id, "--on", vm}}})
	}
	return out
}

func lineOf(s overview.Summary, id string) page.Value {
	for _, body := range pagehome.Body(s, true) {
		for _, it := range body.Items {
			if body.ID == "sections" && it.Row != nil && it.Row.ID == id && len(it.Row.Cells) > 0 {
				return it.Row.Cells[0]
			}
		}
	}
	return page.None()
}

// value: a workspace's header facts: its lifetime, how long it has run,
// how many agents it hosts.
func workspaceSummary(m Machine, vm model.VM) page.Section {
	facts := []page.Value{page.Text(strings.ToLower(pagemachines.Lifetime(vm.Lifetime).Text))}
	if vm.State == model.Running && !vm.BootAt.IsZero() && m.Now.After(vm.BootAt) {
		facts = append(facts, page.Text("up "+days(m.Now.Sub(vm.BootAt))))
	} else if vm.State != model.Running {
		facts = append(facts, page.Text("stopped"))
	}
	n := len(vm.Agents)
	if n == 0 && vm.Agent != nil {
		n = 1
	}
	switch n {
	case 0:
		facts = append(facts, page.Text("no agent"))
	case 1:
		facts = append(facts, page.Text("1 agent"))
	default:
		facts = append(facts, page.Text(strconv.Itoa(n)+" agents"))
	}
	return page.Section{ID: SectionSummary, Title: vm.Name, Role: page.Summary, Caption: facts}
}

// value: a workspace's glyph: the agent mark while it hosts a live agent.
func workspaceGlyph(vm model.VM) string {
	if vm.Agent != nil && !vm.Finished() {
		return "agent"
	}
	return "workspace"
}

// value: a workspace's menu: Share a port… while it runs, then Stop (or
// Start), Delete and Open station.
func workspaceMenu(vm model.VM, stop string) []string {
	if vm.State != model.Running {
		return []string{stop, ActionDelete, ActionOpenStation}
	}
	return []string{ActionShare, stop, ActionDelete, ActionOpenStation}
}

// value: a workspace's Overview Ports card: its shares, each opening its
// link with Copy and QR beside, then its first listener offered with
// Share; nothing shared nor listening, the one line Nothing shared yet.
func workspacePortsCard(b portsBody) page.Section {
	items := slices.Clone(b.Shares)
	if len(b.Listeners) > 0 {
		items = append(items, b.Listeners[0])
	}
	if len(items) == 0 {
		items = []page.Item{{Empty: &page.Empty{Text: "Nothing shared yet"}}}
	}
	return page.Section{ID: SectionPorts, Title: TitlePorts, Segment: SegmentOverview, Caption: []page.Value{count(len(b.Shares))}, Columns: columns(), Items: items}
}
