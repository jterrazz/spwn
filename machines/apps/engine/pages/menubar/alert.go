package menubar

import (
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

const (
	// value: the fact a Mac's alert is graded by, by the words of the
	// engine's confirmation rule (confirm.Fact); a VM's facts are its own,
	// model.FactsOf's, the ones its call reads.
	FactHostsServer = "hosts-server"
)

func factsOf(vm model.VM) []string {
	facts := model.FactsOf(vm)
	out := make([]string, 0, len(facts))
	for _, f := range facts {
		out = append(out, string(f))
	}
	return out
}

// value: the level of a gesture on a page that is handed no rule: yes or no.
func yesOrNo(string, machine.Kind, ...string) string { return LevelYesNo }

const IconApp = "app"

// value: SPEC A5's alert of one gesture: its question, the consequence the
// engine composes from facts, and the button that runs it, red when the
// gesture loses something.
type alert struct {
	Gesture  string
	Kind     machine.Kind
	Name     string
	Facts    []string
	Question string
	Says     []string
	Losses   []string
	Verb     string
	Danger   bool
}

func (a alert) confirm(levels Levels) *page.Confirm {
	// invariant: the level is the engine's, graded by the gesture's facts:
	// none asks nothing, a typed one asks the name and is always red.
	level := levels(a.Gesture, a.Kind, a.Facts...)
	if level == LevelNone {
		return nil
	}
	lines := []page.Value{page.Text(strings.Join(a.Says, " "))}
	for _, l := range a.Losses {
		lines = append(lines, page.Text(l))
	}
	c := &page.Confirm{Question: a.Question, Lines: lines, Yes: a.Verb, No: "Cancel", Danger: a.Danger, Icon: IconApp}
	if level == LevelTypeName {
		c.Type, c.Danger = a.Name, true
	}
	return c
}

// value: the call's input once the alert is answered: yes for a yes or
// no; nothing for a typed one, whose name the owner types.
// invariant: the alert asks at the level its call asks, graded by the same
// facts, so a page never answers a typed question for the owner.
func (a alert) answered(c *page.Confirm, in map[string]string) map[string]string {
	if c != nil && c.Type == "" {
		in["Confirm"] = yes
	}
	return in
}

func (a alert) action(levels Levels, act page.Action) page.Action {
	act.Confirm = a.confirm(levels)
	if act.Run != nil {
		act.Run.In = a.answered(act.Confirm, act.Run.In)
	}
	return act
}

// value: a list said as a sentence: "a", "a and b", "a, b and c".
func and(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	}
	return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
}

func disk(vm model.VM) string {
	if vm.SizeGB <= 0 {
		return "its disk"
	}
	return "its disk (" + strconv.Itoa(vm.SizeGB) + " GB)"
}

// value: the site a server answers the world on: the host its health
// check asks, when that is no name of its own; "" when none.
func site(vm model.VM) string {
	c, ok := model.ParseCheck(vm.Check)
	if !ok || c.URL == nil {
		return ""
	}
	name := c.URL.Hostname()
	if name == vm.Name || name == vm.Device {
		return ""
	}
	return name
}

func restartServer(vm model.VM, on string) alert {
	return alert{Gesture: "restart-server", Kind: machine.Server, Name: vm.Name, Facts: factsOf(vm), Question: "Restart " + vm.Name + " on " + on + "?",
		Says: []string{vm.Name + " is unreachable for about a minute while it boots.", "Its health checks run again after the boot."}, Verb: "Restart " + vm.Name}
}

func stopServer(vm model.VM, on string) alert {
	says := []string{vm.Name + " stays stopped until you start it."}
	if s := site(vm); s != "" {
		says = append(says, s+" goes offline.")
	}
	return alert{Gesture: "stop", Kind: machine.Server, Name: vm.Name, Facts: factsOf(vm), Question: "Stop " + vm.Name + " on " + on + "?", Says: says, Verb: "Stop " + vm.Name, Danger: true}
}

func deleteServer(vm model.VM, ports int) alert {
	goes := []string{disk(vm), "its address"}
	if ports > 0 {
		goes[1] += " and " + counted(ports, "port", "ports")
	}
	says := "This can't be undone. It goes with " + strings.Join(goes, ", ")
	if s := site(vm); s != "" {
		says += ", and " + s + " stops answering."
	} else {
		says += "."
	}
	return alert{Gesture: "delete", Kind: machine.Server, Name: vm.Name, Facts: factsOf(vm), Question: "Delete " + vm.Name + "?", Says: []string{says}, Verb: "Delete " + vm.Name, Danger: true}
}

func deleteWorkspace(vm model.VM, shared, grants int) alert {
	goes := []string{disk(vm)}
	if shared > 0 {
		goes = append(goes, counted(shared, "shared port", "shared ports"))
	}
	if grants > 0 {
		goes = append(goes, counted(grants, "grant", "grants")+" made to it")
	}
	says := []string{"This can't be undone. It goes with " + and(goes) + "."}
	a := alert{Gesture: "delete", Kind: vm.Lifetime.Kind(), Name: vm.Name, Question: "Delete " + vm.Name + "?", Says: says, Verb: "Delete " + vm.Name, Danger: true}
	a.Facts = factsOf(vm)
	return a
}

// value: Stop everything on one Mac: its running workspaces stop and keep
// their disk; This Mac's Awake ends; a Mac hosting a server says it keeps
// running.
func stopEveryWorkspace(label string, thisMac bool, running []string, servers int) alert {
	says := []string{strings.Join(running, ", ") + " " + pluralVerb(len(running), "stops and keeps its disk.", "stop and keep their disk.")}
	if thisMac {
		says = append(says, "Awake ends.")
	}
	if servers > 0 {
		says = append(says, "Servers keep running.")
	}
	return alert{Gesture: "stop-engine", Kind: machine.Computer, Name: label, Question: "Stop every workspace on " + label + "?", Says: says,
		Verb: "Stop " + counted(len(running), "workspace", "workspaces"), Danger: true}
}

func restartMac(label string, servers []string) alert {
	a := alert{Gesture: "restart-mac", Kind: machine.Computer, Name: label, Question: "Restart " + label + "?",
		Says: []string{"Its VMs pause while it restarts, about 2 min."}, Verb: "Restart " + label, Danger: true}
	if len(servers) > 0 {
		a.Says = []string{pluralVerb(len(servers), "It hosts a server.", "It hosts servers."), and(servers) + " " + pluralVerb(len(servers), "is", "are") + " down while " + label + " restarts, about 2 min."}
	}
	return a
}

func forgetMac(label string) alert {
	return alert{Gesture: "forget", Kind: machine.Computer, Name: label, Question: "Forget " + label + "?",
		Says: []string{"Its VMs keep running; Machines stops showing it. You can add it again."}, Verb: "Forget " + label, Danger: true}
}

func stopShare(vm, port string) alert {
	return alert{Gesture: "stop-sharing", Kind: machine.Workspace, Name: vm + ":" + port, Question: "Stop sharing " + vm + ":" + port + "?",
		Says: []string{"The link stops answering on your tailnet. What answers on it keeps running inside the VM."}, Verb: "Stop sharing", Danger: true}
}

func letSleep(label string, servers []string) alert {
	a := alert{Gesture: "off-here", Kind: machine.Computer, Name: label, Question: "Let " + label + " sleep?",
		Says: []string{"Its VMs pause while it sleeps."}, Verb: "Let it sleep", Danger: len(servers) > 0}
	if len(servers) > 0 {
		a.Facts = []string{FactHostsServer}
		a.Says = []string{and(servers) + " " + pluralVerb(len(servers), "pauses", "pause") + " while " + label + " sleeps."}
	}
	return a
}

const (
	ActionRestartMac = "restart-mac"
	ActionForgetMac  = "forget-mac"

	CallRestartMac = "machines.restart-mac"
	CallForgetMac  = "machines.forget"
)

// value: SPEC A5's gestures of a Mac's page that ask first: Restart… and
// Forget… of another Mac, Stop everything… while a workspace runs on it.
func macAlerts(m Machine, h model.MacView, label string) []page.Action {
	var running, servers []string
	for _, vm := range h.VMs {
		switch {
		case vm.Lifetime.Kind() == machine.Server:
			servers = append(servers, vm.Name)
		case vm.State == model.Running:
			running = append(running, vm.Name)
		}
	}
	var out []page.Action
	asleep := model.Reach(h) == model.ReachAsleep
	if !h.ThisMac && !asleep {
		out = append(out, restartMac(label, servers).action(m.Asks, page.Action{ID: ActionRestartMac, Label: "Restart " + label, On: page.OnPage, Priority: 3, Mutates: true,
			Run: &page.Run{Call: CallRestartMac, In: map[string]string{"Name": strconv.Quote(h.Name)}}, Hud: hud(label + " is restarting")}))
	}
	if len(running) > 0 && !asleep {
		out = append(out, stopEveryWorkspace(label, h.ThisMac, running, len(servers)).action(m.Asks, page.Action{ID: ActionStopAll, Label: "Stop everything on " + label, On: page.OnPage, Priority: 3, Mutates: true,
			Run: &page.Run{Call: CallStop, In: map[string]string{"By": "the menu bar"}, Target: target(h)}}))
	}
	if !h.ThisMac {
		out = append(out, forgetMac(label).action(m.Asks, page.Action{ID: ActionForgetMac, Label: "Forget " + label, On: page.OnPage, Priority: 4, Mutates: true,
			Run: &page.Run{Call: CallForgetMac, In: map[string]string{"Name": strconv.Quote(h.Name)}}}))
	}
	return out
}
