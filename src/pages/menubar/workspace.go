package menubar

import (
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/overview"
	"github.com/jterrazz/studio-station/src/page"
	pagehome "github.com/jterrazz/studio-station/src/pages/home"
	"strconv"
)

func workspacePage(m Machine, h model.MacView, vm model.VM) page.Page {
	p, head := vmHead(m, h, vm)
	kind := vm.Lifetime.Kind()
	p.Sections = []page.Section{head, activitySection(running(m, vm), vm.State == model.Running, "Activity", vm.Name), insideSection(m.Summary), portsSection(m, h, vm), sharedSection(m, h, vm)}
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
	p.Actions = append(p.Actions, stopSharings(m, vm)...)
	p.Actions = append(p.Actions, openIn("--on", vm.Name))
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
