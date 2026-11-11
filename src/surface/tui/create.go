package tui

import (
	"slices"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/surface/tui/screens/gesture"
	"github.com/jterrazz/studio-station/src/surface/tui/screens/screen"
)

const (
	workspaceKind = "Workspace"
	serverKind    = "Server"
	throwaway     = "Throwaway"
	permanent     = "Permanent"
)

func (s shell) create() (tea.Model, tea.Cmd) { return s.createForm(machine.Workspace) }

// value: the Mac a VM is made on by default: the one shown, or the Mac of
// the VM shown; This Mac from All machines.
func (s shell) createOn() node {
	n := s.current()
	switch {
	case n.kind == machine.Computer:
		return n
	case n.all():
	default:
		if mac, ok := find(s.nodes, n.mac); ok {
			return mac
		}
	}
	for _, x := range s.nodes {
		if x.thisMac {
			return x
		}
	}
	return s.nodes[min(1, len(s.nodes)-1)]
}

// value: the first name of that stem no machine has (linux-1, linux-2).
func (s shell) freeName(stem string) string {
	for i := 1; ; i++ {
		name := stem + "-" + strconv.Itoa(i)
		if _, taken := find(s.nodes, name); !taken {
			return name
		}
	}
}

// invariant: SPEC §9.7: the create form asks the kind, the name, the Mac
// (the one shown first), a workspace's lifetime or a server's health check,
// its size and its network; enter runs machines create in the Progress
// strip, and the sidebar gains the VM under its Mac once it ends.
func (s shell) createForm(kind machine.Kind) (tea.Model, tea.Cmd) {
	on := s.createOn()
	macs := []string{on.label}
	for _, x := range s.nodes {
		if x.kind == machine.Computer && x.name != on.name {
			macs = append(macs, x.label)
		}
	}
	shape := func(k string) *screen.Form {
		f := &screen.Form{Title: "Create"}
		name := s.freeName("linux")
		if k == serverKind {
			name = s.freeName("server")
		}
		f.Fields = []screen.Field{{Label: "Kind", Value: k, Options: []string{workspaceKind, serverKind}}, {Label: "Name", Value: name}, {Label: "On", Value: macs[0], Options: macs}}
		if k == workspaceKind {
			f.Fields = append(f.Fields, screen.Field{Label: "Lifetime", Value: throwaway, Options: []string{throwaway, permanent}})
		} else {
			f.Fields = append(f.Fields, screen.Field{Label: "Health check", Note: "a URL or host:port; none: it only has to run"})
		}
		cpus, memory, disk := "4", "4G", "20G"
		if k == serverKind {
			memory, disk = "8G", "50G"
		}
		f.Fields = append(f.Fields, screen.Field{Label: "CPUs", Value: cpus}, screen.Field{Label: "Memory", Value: memory, Note: "8G, 512M"},
			screen.Field{Label: "Disk", Value: disk, Note: "a disk only grows"}, screen.Field{Label: "Network", Value: "nat", Options: []string{"nat", "bridged"}})
		return f
	}
	start := workspaceKind
	if kind == machine.Server {
		start = serverKind
	}
	f := shape(start)
	f.Reshape = func(a []string) *screen.Form {
		next := shape(a[0])
		next.Reshape = f.Reshape
		return next
	}
	d := &dialog{call: screen.Call{Form: f}, then: func(a []string) gesture.Gesture { return s.created(f.Reshape(a), a) }}
	for _, x := range f.Fields {
		d.answers = append(d.answers, x.Value)
	}
	s.dialog = d
	return s, nil
}

func (s shell) created(f *screen.Form, a []string) gesture.Gesture {
	answer := func(label string) string {
		i := slices.IndexFunc(f.Fields, func(x screen.Field) bool { return x.Label == label })
		if i < 0 || i >= len(a) {
			return ""
		}
		return strings.TrimSpace(a[i])
	}
	name := answer("Name")
	words := []string{"machines", "create", name}
	what := "workspace"
	if answer("Kind") == serverKind {
		words, what = append(words, "--server"), "server"
		if c := answer("Health check"); c != "" {
			words = append(words, "--check", c)
		}
	} else if answer("Lifetime") == permanent {
		words = append(words, "--permanent")
	}
	for _, x := range [][2]string{{"CPUs", "--cpus"}, {"Memory", "--memory"}, {"Disk", "--disk"}} {
		if v := answer(x[0]); v != "" {
			words = append(words, x[1], v)
		}
	}
	if answer("Network") == "bridged" {
		words = append(words, "--network", "bridged")
	}
	on := answer("On")
	for _, x := range s.nodes {
		if x.kind == machine.Computer && x.label == on && !x.thisMac {
			words = append(words, "--on", x.name)
		}
	}
	what = name + ", a " + what + " on " + on
	return gesture.Gesture{Steps: []gesture.Step{{Words: words, Doing: "Creating " + what, Done: "Created " + what}}}
}
