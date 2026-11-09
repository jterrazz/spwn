package menubar

import (
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/agents"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/page"
	pagemachines "github.com/jterrazz/studio-station/src/pages/machines"
)

// value: what one Mac gives Clean up: its empty throwaways, its finished
// agents by their own names, and the VMs that stay because an agent lives
// there or they run.
type cleanupOf struct {
	mac      model.MacView
	gone     []model.VM
	finished []agents.Agent
	live     []string
	running  []string
}

// value: each Mac's Clean up, its finished agents read from the fleet's
// agents, every one of them, whatever the VM's speaker is.
func cleanupsOf(macs []model.MacView, as []agents.Agent) []cleanupOf {
	out := make([]cleanupOf, 0, len(macs))
	for _, m := range macs {
		gone, kept := model.Cleanup(m.VMs)
		c := cleanupOf{mac: m, gone: gone}
		for _, a := range as {
			if a.State == agents.Finished && (a.Mac == "" || a.Mac == m.Name) && slices.ContainsFunc(m.VMs, func(vm model.VM) bool { return vm.Name == a.VM }) {
				c.finished = append(c.finished, a)
			}
		}
		for _, vm := range kept {
			if vm.Agent != nil && !vm.Finished() {
				c.live = append(c.live, vm.Name)
			} else {
				c.running = append(c.running, vm.Name)
			}
		}
		out = append(out, c)
	}
	return out
}

func (c cleanupOf) any() bool { return len(c.gone) > 0 || len(c.finished) > 0 }

// invariant: SPEC A5 Clean up (A2): finished agents and empty throwaways
// apart, each checked: an agent is forgotten and its workspace stays; a
// throwaway goes with its disk, its ports and the accounts lent to it, and
// checking it checks the finished agents that ran in it; every finished
// agent has its own forget, which the client makes only while the agent is
// checked and its throwaway is not; the note names
// what stays. The button is worded from what is checked, red once a
// throwaway is. One Mac's, or every Mac's when scoped to none.
func cleanupSheet(id string, all model.AllMachines, of []cleanupOf, now time.Time, scoped bool) page.Action {
	agentsField := page.Field{Key: "Agent", Label: "Finished agents", Kind: page.FieldMulti, Note: "each is forgotten; its workspace stays", Follows: "VMs"}
	vmsField := page.Field{Key: "VMs", Label: "Empty throwaways", Kind: page.FieldMulti, Note: "no live agent; each goes with its disk, its ports and the accounts lent to it"}
	var runs []page.Run
	var live, running []string
	for _, c := range of {
		label := model.Label(all, c.mac)
		var names []string
		for _, vm := range c.gone {
			cand := model.CandidateOf(vm, now)
			parts := []string{}
			if !scoped {
				parts = append(parts, label)
			}
			if cand.Stopped && cand.Ended.IsZero() {
				parts = append(parts, "stopped")
			} else {
				parts = append(parts, "idle "+minutes(cand.Idle))
			}
			parts = append(parts, pagemachines.DiskSize(cand.DiskBytes).Text+" disk")
			vmsField.Options = append(vmsField.Options, page.Option{Value: vm.Name, Label: vm.Name, Note: strings.Join(parts, "{sep}")})
			names = append(names, vm.Name)
		}
		if len(names) > 0 {
			list, _ := json.Marshal(names)
			runs = append(runs, page.Run{Call: CallCleanup, In: map[string]string{"VMs": string(list), "Confirm": yes}, Target: target(c.mac)})
		}
		for _, a := range c.finished {
			parts := []string{"on " + a.VM}
			if !a.Ended.IsZero() && now.After(a.Ended) {
				parts = append(parts, "finished "+minutes(now.Sub(a.Ended))+" ago")
			}
			if a.Reason != "" {
				parts = append(parts, a.Reason)
			}
			o := page.Option{Value: a.Name, Label: a.Name, Note: strings.Join(parts, "{sep}")}
			if contains(names, a.VM) {
				o.Of = a.VM
			}
			runs = append(runs, *agentRun(c.mac, a)(CallAgentsForget, nil))
			agentsField.Options = append(agentsField.Options, o)
		}
		live, running = append(live, c.live...), append(running, c.running...)
	}
	vmsField.Default = values(vmsField.Options, false)
	agentsField.Default = values(agentsField.Options, true)
	fields := []page.Field{}
	if len(agentsField.Options) > 0 {
		fields = append(fields, agentsField)
	}
	if len(vmsField.Options) > 0 {
		fields = append(fields, vmsField)
	}
	if note := stays(live, running); note != "" {
		fields = append(fields, page.Field{Key: "stays", Kind: page.FieldNote, Note: note})
	}
	var primary []page.Primary
	for k := len(vmsField.Options); k >= 1; k-- {
		primary = append(primary, page.Primary{When: &page.When{Key: "VMs", Count: k}, Label: "Delete " + counted(k, "throwaway", "throwaways"), Danger: true})
	}
	for k := len(agentsField.Options); k >= 1; k-- {
		primary = append(primary, page.Primary{When: &page.When{Key: "Agent", Count: k}, Label: "Forget " + counted(k, "agent", "agents")})
	}
	title, sub := "Clean up", "Finished agents and empty throwaways."
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

func values(opts []page.Option, own bool) string {
	var out []string
	for _, o := range opts {
		if own && o.Of != "" {
			continue
		}
		out = append(out, o.Value)
	}
	return strings.Join(out, ",")
}

// value: the note naming what stays: the VMs a live agent is in, then the
// throwaways still running.
func stays(live []string, running []string) string {
	agent := "a working agent"
	var lines []string
	if n := len(live); n == 1 {
		lines = append(lines, live[0]+" has "+agent+", so it stays.")
	} else if n > 1 {
		lines = append(lines, joinAnd(live)+" have "+agent+", so they stay.")
	}
	if n := len(running); n == 1 {
		lines = append(lines, running[0]+" is still running, so it stays.")
	} else if n > 1 {
		lines = append(lines, joinAnd(running)+" are still running, so they stay.")
	}
	return strings.Join(lines, " ")
}

func joinAnd(names []string) string {
	if len(names) == 1 {
		return names[0]
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}
