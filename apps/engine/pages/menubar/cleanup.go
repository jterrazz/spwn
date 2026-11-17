package menubar

import (
	"encoding/json"
	"strings"
	"time"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagemachines "github.com/jterrazz/studio-station/apps/engine/pages/machines"
)

// value: what one Mac gives Clean up: its stopped ephemeral workspaces,
// and the ephemeral ones that stay because they run.
type cleanupOf struct {
	mac     model.MacView
	gone    []model.VM
	running []string
}

// value: each Mac's Clean up.
func cleanupsOf(macs []model.MacView) []cleanupOf {
	out := make([]cleanupOf, 0, len(macs))
	for _, m := range macs {
		c := cleanupOf{mac: m}
		for _, vm := range m.VMs {
			switch {
			case vm.Lifetime != model.Ephemeral:
			case vm.State == model.Stopped:
				c.gone = append(c.gone, vm)
			case vm.State == model.Running:
				c.running = append(c.running, vm.Name)
			}
		}
		out = append(out, c)
	}
	return out
}

func (c cleanupOf) any() bool { return len(c.gone) > 0 }

// invariant: SPEC A5 Clean up (A2): the stopped ephemeral workspaces, each
// checked, each going with its disk, its ports and the accounts lent to
// it; the note names what stays. The button is worded from what is
// checked, red. One Mac's, or every Mac's when scoped to none.
func cleanupSheet(id string, all model.AllMachines, of []cleanupOf, now time.Time, scoped bool) page.Action {
	vmsField := page.Field{Key: "VMs", Label: "Stopped ephemeral workspaces", Kind: page.FieldMulti, Note: "each goes with its disk, its ports and the accounts lent to it"}
	var runs []page.Run
	var running []string
	for _, c := range of {
		label := model.Label(all, c.mac)
		var names []string
		for _, vm := range c.gone {
			parts := []string{}
			if !scoped {
				parts = append(parts, label)
			}
			parts = append(parts, "stopped", pagemachines.DiskSize(model.CandidateOf(vm, now).DiskBytes).Text+" disk")
			vmsField.Options = append(vmsField.Options, page.Option{Value: vm.Name, Label: vm.Name, Note: strings.Join(parts, "{sep}")})
			names = append(names, vm.Name)
		}
		if len(names) > 0 {
			list, _ := json.Marshal(names)
			runs = append(runs, page.Run{Call: CallCleanup, In: map[string]string{"VMs": string(list), "Confirm": yes}, Target: target(c.mac)})
		}
		running = append(running, c.running...)
	}
	vmsField.Default = values(vmsField.Options)
	fields := []page.Field{}
	if len(vmsField.Options) > 0 {
		fields = append(fields, vmsField)
	}
	if note := stays(running); note != "" {
		fields = append(fields, page.Field{Key: "stays", Kind: page.FieldNote, Note: note})
	}
	var primary []page.Primary
	for k := len(vmsField.Options); k >= 1; k-- {
		primary = append(primary, page.Primary{When: &page.When{Key: "VMs", Count: k}, Label: "Delete " + counted(k, "ephemeral workspace", "ephemeral workspaces"), Danger: true})
	}
	title, sub := "Clean up", "Stopped ephemeral workspaces."
	if scoped && len(of) == 1 {
		title = "Clean up on " + model.Label(all, of[0].mac)
	}
	a := page.Action{ID: id, Label: "Clean up", On: page.OnPage, Priority: 3, Mutates: true, Sheet: page.SheetCleanup,
		Form: &page.Form{Title: title, Subtitle: sub, Fields: fields, Primary: primary}}
	if len(runs) > 0 {
		first := runs[0]
		first.And = runs[1:]
		a.Run = &first
	}
	return a
}

func values(opts []page.Option) string {
	out := make([]string, 0, len(opts))
	for _, o := range opts {
		out = append(out, o.Value)
	}
	return strings.Join(out, ",")
}

// value: the note naming what stays: the ephemeral workspaces still
// running.
func stays(running []string) string {
	switch n := len(running); {
	case n == 1:
		return running[0] + " is still running, so it stays."
	case n > 1:
		return joinAnd(running) + " are still running, so they stay."
	}
	return ""
}

func joinAnd(names []string) string {
	if len(names) == 1 {
		return names[0]
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}
