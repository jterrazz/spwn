package palette

import (
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/confirm"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/core/section"
	"github.com/jterrazz/studio-station/src/surface/tui/screens/gesture"
	ui "github.com/jterrazz/studio-station/src/ui/render/tui"
)

type Target struct {
	// value: the machine the palette offers its calls on, as the sidebar
	// names it; a Kind of "" is All machines.
	Name, Label string
	Kind        machine.Kind
	ThisMachine bool
	// value: a VM's Mac, by name and as the sidebar says it, and whether
	// it is this one.
	Mac, MacLabel string
	MacHere       bool
	// value: the sections the machine shows (its capabilities decide them).
	Sections []section.ID
	// value: a Mac's VMs, each a target of its own.
	VMs []Target
}

type Field struct {
	Label, Value, Note string
	Secret             bool
}

type Item struct {
	Call  string
	Label string
	// value: the key the call has in its section's screen, when it has one.
	Key string
	// value: the confirmation the call asks, announced: `type infra`, `y/n`.
	Confirm string
	Gesture gesture.Gesture
	// value: a call that needs input first: the form's fields, and the
	// gesture its answers make.
	Fields []Field
	Then   func(answers []string) gesture.Gesture
	// value: the create form of that kind opens.
	Create machine.Kind
	// value: the call's words print what is copied (an ssh command).
	Copy []string
}

type subject int

const (
	// value: the call names no machine but its target.
	none subject = iota
	// value: the call names a VM: the target itself, or each VM of a Mac.
	vm
	// value: the call names a Mac other than this one.
	otherMac
)

type entry struct {
	// value: a section's call is offered where the machine shows that
	// section; another is offered on the kinds it lists ("" is All machines).
	kinds   []machine.Kind
	subject subject
	key     func(k machine.Kind) string
	// value: the item for that target and subject.
	item func(t, s Target) Item
}

func keyed(k string) func(machine.Kind) string { return func(machine.Kind) string { return k } }

func mac() []machine.Kind { return []machine.Kind{machine.Mac} }

func allMachines() []machine.Kind { return []machine.Kind{"", machine.Mac} }

func words(t Target, w ...string) []string {
	if t.Kind == machine.Mac && t.ThisMachine || t.Kind == "" {
		return w
	}
	return append(w, "--on", t.Name)
}

// value: a VM's call is made from its Mac, so it names that Mac when it is
// not this one.
func vmWords(v Target, w ...string) []string {
	if v.MacHere || v.Mac == "" {
		return w
	}
	return append(w, "--on", v.Mac)
}

func run(label string, w []string, verb gesture.Verb, what string) Item {
	return Item{Label: label, Gesture: gesture.Gesture{Steps: []gesture.Step{verb.Step(what, w)}}}
}

func open(t Target, id section.ID, label string) Item {
	return Item{Label: label, Gesture: gesture.Gesture{Visit: &gesture.Visit{Machine: t.Name, Section: string(id)}}}
}

func asked(label string, fields []Field, then func(a []string) gesture.Gesture) Item {
	return Item{Label: label, Fields: fields, Then: then}
}

func field(label string) []Field { return []Field{{Label: label}} }

func one(verb gesture.Verb, w func(a string) []string) func([]string) gesture.Gesture {
	return func(a []string) gesture.Gesture {
		v := strings.TrimSpace(a[0])
		return gesture.Gesture{Steps: []gesture.Step{verb.Step(v, w(v))}}
	}
}

func shown(w func(a string) []string) func([]string) gesture.Gesture {
	return func(a []string) gesture.Gesture { return gesture.Gesture{Read: w(strings.TrimSpace(a[0]))} }
}

func terminal(label string, w ...string) Item {
	return Item{Label: label, Gesture: gesture.Gesture{Terminal: "station " + strings.Join(w, " ")}}
}

// value: how the palette names a VM: its kind when a server, and its Mac
// when not this one.
func named(v Target) string {
	s := v.Label
	if v.Kind == machine.Server {
		s += " (server)"
	}
	return s
}

// invariant: TU-11: one entry per call of calls.json, by its name; the
// architecture suite fails when a call has none or is offered where its
// Applies refuses it.
func entries() map[string]entry {
	return map[string]entry{
		"accounts.list": {item: func(t, _ Target) Item { return open(t, section.Accounts, "Open Accounts") }},
		"accounts.share": {kinds: mac(), key: keyed("s"), item: func(t, _ Target) Item {
			return asked("Share an account with a VM"+ellipsis, []Field{{Label: "Account"}, {Label: "VM"}, {Label: "For", Value: "12h"}}, func(a []string) gesture.Gesture {
				acc, to := strings.TrimSpace(a[0]), strings.TrimSpace(a[1])
				return gesture.Gesture{Steps: []gesture.Step{gesture.V("Share", "Sharing", "Shared").Step(acc+" with "+to, words(t, "accounts", "share", acc, "--with", to, "--for", strings.TrimSpace(a[2])))}}
			})
		}},
		"accounts.take-back": {kinds: mac(), key: keyed("t"), item: func(t, _ Target) Item {
			return asked("Take an account back from a VM"+ellipsis, []Field{{Label: "Account"}, {Label: "VM"}}, func(a []string) gesture.Gesture {
				acc, from := strings.TrimSpace(a[0]), strings.TrimSpace(a[1])
				return gesture.Gesture{Steps: []gesture.Step{gesture.V("Take back", "Taking back", "Took back").Step(acc+" from "+from, words(t, "accounts", "take-back", acc, "--from", from))}}
			})
		}},
		"activity.read":   {item: func(t, _ Target) Item { return open(t, section.Activity, "Open Activity") }},
		"activity.sample": {key: keyed("w"), item: func(t, _ Target) Item { return open(t, section.Activity, "Sample Activity, every 2 s while shown") }},
		"agents.record": {kinds: []machine.Kind{machine.Workspace}, item: func(t, _ Target) Item {
			return run("Hook the agents of "+t.Label+", so station records what they do", words(t, "settings", "apply", catalog.AgentHooksSetting), gesture.V("Apply", "Applying", "Applied"), catalog.AgentHooksSetting)
		}},
		"agents.list": {kinds: []machine.Kind{machine.Workspace}, item: func(t, _ Target) Item {
			return Item{Label: "Read the agents of " + t.Label, Gesture: gesture.Gesture{Read: words(t, "agents")}}
		}},
		"agents.merge-pr": {kinds: []machine.Kind{machine.Workspace}, item: func(t, _ Target) Item {
			return asked("Merge the PR an agent opened"+ellipsis, field("Session"), one(gesture.V("Merge", "Merging", "Merged"), func(a string) []string { return words(t, "agents", "merge-pr", a) }))
		}},
		"agents.reply": {kinds: []machine.Kind{machine.Workspace}, item: func(t, _ Target) Item {
			return asked("Reply to an agent"+ellipsis, []Field{{Label: "Session"}, {Label: "Reply"}}, func(a []string) gesture.Gesture {
				session := strings.TrimSpace(a[0])
				return gesture.Gesture{Steps: []gesture.Step{gesture.V("Send", "Sending", "Sent").Step(session, words(t, "agents", "reply", session, strings.TrimSpace(a[1])))}}
			})
		}},
		"agents.start": {kinds: []machine.Kind{machine.Workspace}, item: func(t, _ Target) Item {
			return asked("Start an agent on a task"+ellipsis, field("Task"), one(gesture.V("Start", "Starting", "Started"), func(a string) []string { return words(t, "agents", "start", "--task", a) }))
		}},
		"apps.install": {key: keyed("i"), item: func(t, _ Target) Item {
			return asked("Install an app"+ellipsis, field("App"), one(gesture.V("Install", "Installing", "Installed"), func(a string) []string { return words(t, "apps", "install", a) }))
		}},
		"apps.list": {item: func(t, _ Target) Item { return open(t, section.Apps, "Open Apps") }},
		"apps.remove": {key: keyed("x"), item: func(t, _ Target) Item {
			return asked("Remove an app"+ellipsis, field("App"), one(gesture.V("Remove", "Removing", "Removed"), func(a string) []string { return words(t, "apps", "remove", a) }))
		}},
		"apps.show": {key: keyed("enter"), item: func(t, _ Target) Item {
			return asked("Show an app"+ellipsis, field("App"), shown(func(a string) []string { return words(t, "apps", a) }))
		}},
		"apps.update": {key: keyed("u"), item: func(t, _ Target) Item {
			return run("Update every app", words(t, "apps", "update"), gesture.V("Update", "Updating", "Updated"), "every app")
		}},
		"awake.keep": {kinds: mac(), key: keyed("4"), item: func(t, _ Target) Item {
			return run("Keep "+t.Label+" awake for 4 h", words(t, "awake", "4h"), gesture.V("Keep awake", "Keeping awake", "Kept awake"), t.Label)
		}},
		"awake.off": {kinds: mac(), key: keyed("o"), item: func(t, _ Target) Item {
			return run("Let "+t.Label+" sleep", words(t, "awake", "off"), gesture.V("Let sleep", "Letting sleep", "Sleeps normally:"), t.Label)
		}},
		"awake.read":      {kinds: mac(), item: func(t, _ Target) Item { return open(t, section.Awake, "Open Awake") }},
		"background.list": {item: func(t, _ Target) Item { return open(t, section.Background, "Open Background") }},
		"background.remove": {key: keyed("x"), item: func(t, _ Target) Item {
			return asked("Remove a background task"+ellipsis, field("Label"), one(gesture.V("Remove", "Removing", "Removed"), func(a string) []string { return words(t, "background", "remove", a) }))
		}},
		"disk.erase": {key: keyed("e"), item: func(t, _ Target) Item {
			return asked("Erase an app's data"+ellipsis, field("App"), one(gesture.V("Erase", "Erasing", "Erased"), func(a string) []string { return words(t, "disk", "erase", a) }))
		}},
		"disk.free": {key: keyed("f"), item: func(t, _ Target) Item {
			return run("Free what apps can rebuild", words(t, "disk", "free"), gesture.V("Free", "Freeing", "Freed"), "what apps can rebuild")
		}},
		"disk.move": {key: keyed("v"), item: func(t, _ Target) Item {
			return asked("Move a misplaced folder"+ellipsis, field("Folder"), one(gesture.V("Move", "Moving", "Moved"), func(a string) []string { return words(t, "disk", "move", a) }))
		}},
		"disk.read": {item: func(t, _ Target) Item { return open(t, section.Disk, "Open Disk") }},
		"engine.read": {kinds: mac(), item: func(t, _ Target) Item {
			return open(t, section.Overview, "Open "+t.Label+"'s Overview, its engine first")
		}},
		"engine.start": {kinds: mac(), key: keyed("s"), item: func(t, _ Target) Item {
			return run("Start the engine on "+t.Label, words(t, "engine", "start"), gesture.V("Start", "Starting", "Started"), "the engine")
		}},
		"engine.stop": {kinds: mac(), item: func(t, _ Target) Item {
			return run("Stop everything on "+t.Label, words(t, "engine", "stop"), gesture.V("Stop", "Stopping", "Stopped"), "everything on "+t.Label)
		}},
		"images.build": {kinds: allMachines(), key: keyed("b"), item: func(t, _ Target) Item {
			return run("Build the images", words(t, "images", "build"), gesture.V("Build", "Building", "Built"), "the images")
		}},
		"images.delete": {kinds: allMachines(), key: keyed("x"), item: func(t, _ Target) Item {
			return asked("Delete an image"+ellipsis, []Field{{Label: "Image", Note: "linux@<version>"}}, one(gesture.V("Delete", "Deleting", "Deleted"), func(a string) []string { return words(t, "images", "delete", a) }))
		}},
		"images.list": {kinds: allMachines(), item: func(_, _ Target) Item { return open(Target{}, section.Images, "Open Images") }},
		"images.update": {kinds: allMachines(), key: keyed("u"), item: func(t, _ Target) Item {
			return run("Update the images", words(t, "images", "update"), gesture.V("Update", "Updating", "Updated"), "the images")
		}},
		"machines.add": {kinds: allMachines(), item: func(_, _ Target) Item {
			return asked("Add a Mac"+ellipsis, []Field{{Label: "Mac", Note: "its name on " + catalog.TailnetProduct}}, one(gesture.V("Add", "Adding", "Added"), func(a string) []string { return []string{"machines", "add", a} }))
		}},
		"machines.create-server": {kinds: allMachines(), item: func(_, _ Target) Item {
			return Item{Label: "New server" + ellipsis, Create: machine.Server}
		}},
		"machines.create-workspace": {kinds: allMachines(), key: keyed("n"), item: func(_, _ Target) Item {
			return Item{Label: "New workspace" + ellipsis, Create: machine.Workspace}
		}},
		"machines.delete": {subject: vm, key: keyed("D"), item: func(_, v Target) Item {
			return run("Delete "+named(v)+onMac(v), vmWords(v, "machines", "delete", v.Name), gesture.V("Delete", "Deleting", "Deleted"), v.Name)
		}},
		"machines.edit": {subject: vm, key: keyed("e"), item: func(_, v Target) Item {
			return asked("Edit "+named(v)+onMac(v)+ellipsis, []Field{{Label: "CPUs"}, {Label: "Memory", Note: "8G, 512M"}, {Label: "Disk", Note: "a disk only grows"}}, func(a []string) gesture.Gesture {
				w := []string{"machines", "edit", v.Name}
				for i, flag := range []string{"--cpus", "--memory", "--disk"} {
					if x := strings.TrimSpace(a[i]); x != "" {
						w = append(w, flag, x)
					}
				}
				return gesture.Gesture{Steps: []gesture.Step{gesture.V("Edit", "Editing", "Edited").Step(v.Name, vmWords(v, w...))}}
			})
		}},
		"machines.forget": {kinds: mac(), subject: otherMac, item: func(_, m Target) Item {
			return run("Forget "+m.Label, []string{"machines", "forget", m.Name}, gesture.V("Forget", "Forgetting", "Forgot"), m.Label)
		}},
		"machines.list": {kinds: []machine.Kind{"", machine.Mac, machine.Workspace, machine.Server}, item: func(_, _ Target) Item { return open(Target{}, section.Overview, "Open All machines") }},
		"machines.log": {subject: vm, item: func(_, v Target) Item {
			return Item{Label: "Show the log of " + named(v), Gesture: gesture.Gesture{Read: vmWords(v, "machines", "log", v.Name)}}
		}},
		"machines.restart": {subject: vm, key: keyed("r"), item: func(_, v Target) Item {
			return run("Restart "+named(v)+onMac(v), vmWords(v, "machines", "restart", v.Name), gesture.V("Restart", "Restarting", "Restarted"), v.Name)
		}},
		"machines.restart-mac": {kinds: mac(), item: func(t, _ Target) Item {
			return run("Restart "+t.Label, []string{"machines", "restart", t.Name}, gesture.V("Restart", "Restarting", "Restarted"), t.Label)
		}},
		"machines.ssh": {subject: vm, key: keyed("c"), item: func(_, v Target) Item {
			return Item{Label: "Copy the command that reaches " + named(v), Copy: vmWords(v, "machines", string(section.SSH), v.Name)}
		}},
		"machines.start": {subject: vm, key: keyed("s"), item: func(_, v Target) Item {
			return run("Start "+named(v)+onMac(v), vmWords(v, "machines", "start", v.Name), gesture.V("Start", "Starting", "Started"), v.Name)
		}},
		"machines.stop": {subject: vm, key: func(k machine.Kind) string {
			if k == machine.Server {
				return "S"
			}
			return "s"
		}, item: func(_, v Target) Item {
			return run("Stop "+named(v)+onMac(v), vmWords(v, "machines", "stop", v.Name), gesture.V("Stop", "Stopping", "Stopped"), v.Name)
		}},
		"machines.unlock": {kinds: mac(), subject: otherMac, key: keyed("u"), item: func(_, m Target) Item {
			return terminal("Unlock "+m.Label+ellipsis+" (asks its password)", "machines", "unlock", m.Label)
		}},
		"overview.read": {item: func(t, _ Target) Item { return open(t, section.Overview, "Open Overview") }},
		"ports.declare": {kinds: mac(), key: keyed("d"), item: func(t, _ Target) Item {
			return asked("Declare a port public"+ellipsis, []Field{{Label: "Port"}, {Label: "Route", Value: catalog.RouteLAN, Note: routes()}}, func(a []string) gesture.Gesture {
				port := strings.TrimSpace(a[0])
				return gesture.Gesture{Steps: []gesture.Step{gesture.V("Declare", "Declaring", "Declared").Step(port+" public", []string{"ports", "declare", t.Name, port, "--via", strings.TrimSpace(a[1])})}}
			})
		}},
		"ports.list": {item: func(t, _ Target) Item { return open(t, section.Ports, "Open Ports") }},
		"ports.set-https": {kinds: mac(), key: keyed("h"), item: func(t, _ Target) Item {
			return asked("Turn HTTPS links on or off"+ellipsis, []Field{{Label: "HTTPS links", Value: "off", Note: "on or off"}}, one(gesture.V("Turn", "Turning", "Turned"), func(a string) []string { return words(t, "ports", "https", a) }))
		}},
		"ports.share": {subject: vm, key: keyed("s"), item: func(_, v Target) Item {
			return asked("Share a port of "+named(v)+ellipsis, field("Port"), one(gesture.V("Share", "Sharing", "Shared"), func(a string) []string { return vmWords(v, "ports", "share", v.Name, a) }))
		}},
		"ports.stop-sharing": {subject: vm, key: keyed("x"), item: func(_, v Target) Item {
			return asked("Stop sharing a port of "+named(v)+ellipsis, field("Port"), one(gesture.V("Stop sharing", "Stopping sharing", "Stopped sharing"), func(a string) []string {
				return vmWords(v, "ports", "stop-sharing", v.Name, a)
			}))
		}},
		"ports.undeclare": {kinds: mac(), key: keyed("u"), item: func(t, _ Target) Item {
			return asked("Undeclare a public port"+ellipsis, field("Port"), one(gesture.V("Undeclare", "Undeclaring", "Undeclared"), func(a string) []string { return []string{"ports", "undeclare", t.Name, a} }))
		}},
		"settings.run-containers": {item: func(t, _ Target) Item {
			return run("Remove the stopped containers", words(t, "run", catalog.ContainerShortcutsWas, "rm"), gesture.V("Remove", "Removing", "Removed"), "the stopped containers")
		}},
		"settings.run-git": {item: func(_, _ Target) Item {
			return asked("Run a Git shortcut in a new terminal"+ellipsis, []Field{{Label: "Shortcut", Note: "log, push, status"}}, func(a []string) gesture.Gesture {
				return gesture.Gesture{Terminal: "station run git " + strings.TrimSpace(a[0])}
			})
		}},
		"settings.apply": {key: keyed("a"), item: func(t, _ Target) Item {
			return run("Apply every setting not applied", words(t, "settings", "apply"), gesture.V("Apply", "Applying", "Applied"), "every setting not applied")
		}},
		"settings.diff": {key: keyed("d"), item: func(t, _ Target) Item {
			return asked("Diff a setting"+ellipsis, field("Setting"), shown(func(a string) []string { return words(t, "settings", "diff", a) }))
		}},
		"settings.list":   {item: func(t, _ Target) Item { return open(t, section.Settings, "Open Settings") }},
		"settings.matrix": {kinds: allMachines(), item: func(_, _ Target) Item { return open(Target{}, section.Settings, "Open the settings of every machine") }},
		"settings.off": {key: keyed("o"), item: func(t, _ Target) Item {
			return asked("Turn a setting off here"+ellipsis, field("Setting"), one(gesture.V("Turn off", "Turning off", "Turned off"), func(a string) []string { return words(t, "settings", "off", a) }))
		}},
		"settings.show": {key: keyed("enter"), item: func(t, _ Target) Item {
			return asked("Show a setting"+ellipsis, field("Setting"), shown(func(a string) []string { return words(t, "settings", a) }))
		}},
		"setup.run": {kinds: []machine.Kind{machine.Mac, machine.Workspace, machine.Server}, item: func(t, _ Target) Item {
			return terminal("Run setup in a new terminal", words(t, "setup")...)
		}},
		"settings.zsh-block": {kinds: []machine.Kind{machine.Mac, machine.Workspace, machine.Server}, item: func(_, _ Target) Item {
			return terminal("Print the shell block in a new terminal", "shell", "zsh")
		}},
	}
}

const ellipsis = ui.Ellipsis

// value: the routes a public port may take, as --via names them.
func routes() string {
	names := make([]string, 0, len(catalog.Routes))
	for _, r := range catalog.Routes {
		names = append(names, r.Name)
	}
	return strings.Join(names, ", ")
}

func onMac(v Target) string {
	if v.MacHere || v.Mac == "" {
		return ""
	}
	return " on " + v.MacLabel
}

func Offer(metas []call.Meta, t Target, macs []Target) []Item {
	// invariant: SPEC §9.5: every call of the catalogue that applies to the
	// target, in the catalogue's order; a section's call where the target shows
	// that section, a VM's call on that VM, or on each VM of a Mac, and the
	// confirmation the call asks announced (confirm.For, never the surface).
	var out []Item
	table := entries()
	for _, m := range metas {
		e, ok := table[m.Name]
		if !ok || !offered(e, m, t) {
			continue
		}
		var subjects []Target
		switch e.subject {
		case none:
			subjects = []Target{t}
		case vm:
			if t.Kind == machine.Mac {
				subjects = t.VMs
			} else if t.Kind != "" {
				subjects = []Target{t}
			}
		case otherMac:
			if t.Kind == machine.Mac && !t.ThisMachine {
				subjects = []Target{t}
			}
			if t.Kind == "" {
				subjects = slices.DeleteFunc(slices.Clone(macs), func(x Target) bool { return x.ThisMachine })
			}
		}
		for _, s := range subjects {
			it := e.item(t, s)
			it.Call = m.Name
			if e.key != nil {
				it.Key = e.key(s.Kind)
			}
			if m.Confirm != "" && (m.Confirm != confirm.RestartMac || s.Kind == machine.Mac) {
				switch confirm.For(m.Confirm, s.Kind) {
				case confirm.TypeName:
					it.Confirm = "type " + s.Name
				case confirm.YesNo:
					it.Confirm = "y/n"
				case confirm.None:
				}
			}
			out = append(out, it)
		}
	}
	return out
}

func offered(e entry, m call.Meta, t Target) bool {
	if e.subject == vm {
		return t.Kind != ""
	}
	if e.kinds != nil {
		return slices.Contains(e.kinds, t.Kind) || e.subject == otherMac && t.Kind == ""
	}
	return m.Section != "" && slices.Contains(t.Sections, m.Section)
}

func Search(items []Item, query string) []Item {
	// value: the items whose words hold every word of the query, by their
	// label or their call's name.
	words := strings.Fields(strings.ToLower(query))
	return slices.DeleteFunc(slices.Clone(items), func(it Item) bool {
		hay := strings.ToLower(it.Label + " " + it.Call)
		for _, w := range words {
			if !strings.Contains(hay, w) {
				return true
			}
		}
		return false
	})
}
