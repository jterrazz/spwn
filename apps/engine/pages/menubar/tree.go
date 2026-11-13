package menubar

import (
	"time"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagehome "github.com/jterrazz/studio-station/apps/engine/pages/home"
	pagemachines "github.com/jterrazz/studio-station/apps/engine/pages/machines"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

const (
	KindMac         = "Mac"
	GroupServers    = "Servers"
	GroupSpaces     = "Workspaces"
	servers, spaces = "/servers", "/workspaces"
)

func treeColumns() []page.Column {
	return []page.Column{{Key: "name", Kind: page.KindText, Keep: 100}, {Key: "kind", Kind: page.KindText, Keep: 40}, {Key: "detail", Kind: page.KindList, Keep: 80}}
}

func Tree(f *model.AllMachines, svc *ports.AllMachines, line func(model.MacView) page.Value) page.Section {
	// invariant: SPEC §10.2's Machines: each Mac, then its Servers, then its
	// Workspaces (the running first), each group named; a row's ID is its machine's name, the page
	// it opens. line says what a Mac's row adds after its kind.
	s := page.Section{ID: SectionMachines, Title: "Machines", Columns: treeColumns()}
	if f == nil {
		s.Items = []page.Item{{Row: &page.Row{ID: "reading", Label: "reading", Mark: page.Awaiting}}}
		return s
	}
	for _, h := range f.Macs {
		g := page.Group{Head: macRow(*f, h, line(h))}
		var srv, ws, idle []page.Item
		for _, vm := range h.VMs {
			row := vmRow(vm, shared(svc, vm.Name), f.Read)
			if !h.Reachable && !h.ThisMac {
				row.Mark, row.Reason = page.NotKnown, ""
				row.Cells[1] = page.List(row.Cells[1].Items[0], page.Text("not known while "+model.Label(*f, h)+" does not answer"))
			}
			switch {
			case vm.Lifetime == model.AlwaysOn:
				srv = append(srv, page.Item{Row: &row})
			case vm.State == model.Running:
				ws = append(ws, page.Item{Row: &row})
			default:
				idle = append(idle, page.Item{Row: &row})
			}
		}
		ws = append(ws, idle...)
		if len(srv) > 0 {
			g.Items = append(g.Items, page.Item{Group: &page.Group{Head: page.Row{ID: h.Name + servers, Label: GroupServers}, Items: srv}})
		}
		if len(ws) > 0 {
			g.Items = append(g.Items, page.Item{Group: &page.Group{Head: page.Row{ID: h.Name + spaces, Label: GroupSpaces}, Items: ws}})
		}
		s.Items = append(s.Items, page.Item{Group: &g})
	}
	return s
}

func macRow(f model.AllMachines, h model.MacView, line page.Value) page.Row {
	row := page.Row{ID: h.Name, Label: model.Label(f, h), Mark: page.OK, Cells: []page.Value{page.Text(KindMac), page.List(line)}}
	if line.IsNone() {
		row.Cells[1] = page.None()
	}
	if !h.Reachable && !h.ThisMac {
		row.Mark, row.Reason, row.Cells[1] = page.Broken, page.Unreachable, page.Text("Unreachable")
	}
	return row
}

// value: a VM's row: a server's lifetime, health, time up (when its boot
// is known) and restarts, a workspace's lifetime and either its shared
// ports or its state.
func vmRow(vm model.VM, shares int, now time.Time) page.Row {
	row := page.Row{ID: vm.Name, Label: vm.Name, Mark: page.OK}
	kind := model.KindWord(vm.Lifetime.Kind())
	life := pagemachines.Lifetime(vm.Lifetime)
	switch {
	case model.Down(vm):
		row.Mark, row.Reason = page.Broken, page.Down
		row.Cells = []page.Value{page.Text(kind), page.List(life, pagemachines.State(vm.State))}
	case vm.State != model.Running:
		row.Mark, row.Reason = page.Absent, page.Stopped
		row.Cells = []page.Value{page.Text(kind), page.List(life, pagemachines.State(vm.State))}
	case vm.Lifetime == model.AlwaysOn:
		words := []page.Value{life, page.Text(health(vm))}
		if !vm.BootAt.IsZero() && !now.IsZero() && now.After(vm.BootAt) {
			words = append(words, pagehome.Up(now.Sub(vm.BootAt)))
		}
		row.Cells = []page.Value{page.Text(kind), page.List(append(words, page.Count(int64(vm.Restarts.Count), plural(vm.Restarts.Count, "restart")))...)}
		if health(vm) != "healthy" {
			row.Mark, row.Reason = page.Attention, page.Waiting
		}
	case shares > 0:
		row.Cells = []page.Value{page.Text(kind), page.List(life, page.Count(int64(shares), plural(shares, "port")+" shared"))}
	default:
		row.Cells = []page.Value{page.Text(kind), page.List(life, pagemachines.State(vm.State))}
	}
	return row
}

func plural(n int, noun string) string {
	if n == 1 {
		return noun
	}
	return noun + "s"
}

// value: a server is healthy when every step of its health passes.
func health(vm model.VM) string {
	for _, s := range pagemachines.Steps(vm) {
		if s.Mark != page.OK {
			return s.Label
		}
	}
	return "healthy"
}

func shared(svc *ports.AllMachines, vm string) int {
	if svc == nil {
		return 0
	}
	n := 0
	for _, h := range svc.Macs {
		for _, s := range h.Shared {
			if s.VM == vm {
				n++
			}
		}
	}
	return n
}
