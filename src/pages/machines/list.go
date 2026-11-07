package machines

import (
	"strconv"
	"strings"
	"time"

	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/page"
)

func listColumns() []page.Column {
	return []page.Column{
		{Key: "name", Label: "MACHINE", Kind: page.KindText, Keep: 100},
		{Key: "kind", Label: "KIND", Kind: page.KindText, Keep: 80, Whole: true},
		{Key: "mac", Kind: page.KindText, Show: page.PipeOnly},
		{Key: "state", Label: "STATE", Kind: page.KindList, Keep: 90},
		{Key: "cpu", Label: "CPU", Kind: page.KindText, Keep: 40},
		{Key: "memory", Label: "MEMORY", Kind: page.KindText, Keep: 30},
		{Key: "detail", Kind: page.KindList, Keep: 5},
	}
}

func List(f model.AllMachines, warnings []string, thisMac page.Context) page.Page {
	thisVersion := ""
	for _, h := range f.Macs {
		if h.ThisMac {
			thisVersion = h.StationVersion
		}
	}
	findings := model.Needs(f)
	items := make([]page.Item, 0, len(f.Macs))
	var hints []page.Hint
	for _, h := range f.Macs {
		g := page.Group{Head: macRow(model.Label(f, h), h, thisVersion, f.Read), Fold: page.Open}
		for _, vm := range model.Ordered(h.VMs) {
			row := listVMRow(h.Name, vm)
			g.Items = append(g.Items, page.Item{Row: &row})
		}
		if len(h.VMs) == 0 && h.Reachable {
			empty := &page.Empty{Noun: "VMs", Text: "No VMs yet"}
			if h.ThisMac {
				empty.Hint = &page.Hint{Key: KeyCreate, Text: "create one"}
			}
			g.Items = append(g.Items, page.Item{Empty: empty})
			create := "station machines create"
			if !h.ThisMac {
				create += " --on " + model.Label(f, h)
			}
			hints = append(hints, page.Hint{Command: create})
		}
		items = append(items, page.Item{Group: &g})
	}
	if len(items) == 0 {
		items = append(items, page.Item{Empty: &page.Empty{Noun: "Macs", Hint: page.Cmd("station machines add <mac>")}})
	}
	title, caption := firstLine(model.Tally(f), len(findings))
	sections := []page.Section{{ID: "machines", Title: title, Caption: caption, Joined: true, Role: page.Banner, Uncounted: true, Columns: listColumns(), Items: items}}
	if len(findings) > 0 {
		sections = append(sections, needsYou(findings))
	}
	if len(warnings) > 0 {
		notes := make([]page.Item, 0, len(warnings))
		for _, w := range warnings {
			notes = append(notes, page.Item{Note: &page.Note{Text: w}})
		}
		sections = append(sections, page.Section{ID: "warnings", Title: "Warnings", Role: page.Banner, Items: notes})
	}
	return page.Seal(page.Page{ID: "machines", Command: "machines", Context: thisMac, Sections: sections, Hints: hints, Actions: Actions()})
}

func firstLine(c model.Counts, needs int) (string, []page.Value) {
	parts := []page.Value{page.Text(strconv.Itoa(c.Running) + " running"), page.Text(strconv.Itoa(c.Stopped) + " stopped")}
	switch needs {
	case 0:
		parts = append(parts, page.Text("nothing needs you"))
	case 1:
		parts = append(parts, page.Text("1 needs you"))
	default:
		parts = append(parts, page.Text(strconv.Itoa(needs)+" need you"))
	}
	return many(c.Machines, "machine"), parts
}

func many(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return strconv.Itoa(n) + " " + noun + "s"
}

func needsYou(fs []model.Need) page.Section {
	items := make([]page.Item, 0, len(fs))
	for i, f := range fs {
		mark, reason := page.Attention, page.Degraded
		if f.Broken {
			mark, reason = page.Broken, page.Failing
		}
		says := strings.TrimPrefix(f.Text, f.Subject+" ")
		row := page.Row{ID: strconv.Itoa(i) + "-" + f.Subject, Label: f.Subject, Mark: mark, Reason: reason, Cells: []page.Value{page.Text(says), fixOf(f)}}
		items = append(items, page.Item{Row: &row})
	}
	columns := []page.Column{{Key: "subject", Kind: page.KindText, Keep: 100}, {Key: "finding", Kind: page.KindList, Keep: 90}, {Key: "fix", Kind: page.KindList, Keep: 50}}
	return page.Section{ID: "needs-you", Title: "Needs you", Uncounted: true, Columns: columns, Items: items}
}

func fixOf(f model.Need) page.Value {
	switch {
	case f.Fix == "":
		return page.None()
	case strings.HasPrefix(f.Fix, "station "):
		return page.Command(f.Fix)
	}
	return page.Text(f.Fix)
}

func macRow(label string, h model.MacView, thisVersion string, now time.Time) page.Row {
	row := page.Row{ID: h.Name, Label: h.Name, Short: label, Mark: page.OK}
	state, detail := page.Text("Online"), page.None()
	switch {
	case !h.Reachable && !h.Answered.IsZero():
		row.Mark, row.Reason, state = page.Broken, page.Unreachable, page.List(page.Text("Unreachable"), page.Text("last answer "+page.LastAnswer(h.Answered, now)))
	case !h.Reachable:
		row.Mark, row.Reason, state = page.Broken, page.Unreachable, page.Text("Unreachable")
	case h.Engine.Known && !h.Engine.Running && !h.Engine.Installed && h.ThisMac:
		row.Mark, row.Reason, state = page.Broken, page.Missing, page.Text("Engine not set up")
	case h.Engine.Known && !h.Engine.Running:
		row.Mark, row.Reason, state = page.Broken, page.Stopped, page.Text("Engine stopped")
	case h.Engine.Known:
		state = page.Text("Engine running")
	}
	if _, ok := model.Behind(h, thisVersion); ok && h.Reachable {
		row.Mark, row.Reason = page.Attention, page.Mismatch
		state = page.List(state, page.Text("runs "+h.StationVersion))
	}
	row.Cells = []page.Value{page.Text("Mac"), page.Text(h.Name), state, page.None(), page.None(), detail}
	return row
}

func listVMRow(mac string, vm model.VM) page.Row {
	row := page.Row{ID: vm.Name, Label: vm.Name, Mark: page.OK}
	lifetime := page.Text(vm.Lifetime.Word())
	state, detail := page.List(page.Text("Running"), lifetime), page.None()
	switch vm.State {
	case model.Running:
	case model.Stopped:
		row.Mark, state = page.Absent, page.List(page.Text("Stopped"), lifetime)
		if vm.Detail != "" {
			detail = page.Text(vm.Detail)
		}
	case model.Suspended:
		row.Mark, row.Reason, state = page.Attention, page.Stopped, page.List(page.Text("Suspended"), lifetime)
	case model.Stale:
		row.Mark, row.Reason, state = page.Attention, page.Outlived, page.List(page.Text("Ended unexpectedly"), lifetime)
	case model.Broken:
		row.Mark, row.Reason, state = page.Broken, page.Failing, page.List(page.Text("In a loop"), lifetime)
		detail = page.List(page.Text(vm.Detail), page.Text("stopped"))
	case model.UnknownState:
		row.Mark, row.Reason, state = page.NotKnown, page.Unreachable, page.List(page.Text("Unknown"), lifetime)
	case model.Creating:
		row.Mark, state = page.Unjudged, page.List(page.Text("Creating"), lifetime)
		if vm.Birth != nil {
			detail = page.Text(vm.Birth.Step)
		}
	}
	switch {
	case vm.Lifetime != model.AlwaysOn:
	case vm.Restarts.Looping:
		row.Mark, row.Reason, state = page.Broken, page.Failing, page.List(page.Text("In a loop"), lifetime)
		detail = page.Texts(vm.Restarts.Words(time.Time{})...)
	case vm.State == model.Stopped && vm.Detail == model.PoweredOffInside:
		row.Mark, row.Reason = page.Broken, page.Stopped
	case vm.State == model.Running && vm.CheckStatus.State == model.CheckFailing:
		row.Mark, row.Reason = page.Attention, page.Failing
		detail = page.Text("check failing since " + vm.CheckStatus.Since.Local().Format("15:04"))
	}
	r := vm.Resources
	row.Cells = []page.Value{page.Text(model.KindWord(vm.Lifetime.Kind())), page.Text(mac), state, page.Text(strconv.Itoa(r.CPUs) + " CPU"), Memory(r.MemoryBytes), detail}
	return row
}
