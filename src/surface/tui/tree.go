package tui

import (
	"cmp"
	"strings"

	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/core/section"
	"github.com/jterrazz/studio-station/src/page"
	ui "github.com/jterrazz/studio-station/src/ui/render/tui"
)

type node struct {
	name    string
	label   string
	kind    machine.Kind
	mark    page.Mark
	mac     string
	thisMac bool
	engine  string
	facts   []string
	// value: why the machines read marks it (a Mac that does not answer).
	reason page.Reason
}

const allMachines = "All machines"

// value: the state word of a Mac whose engine is stopped: it is ◐, not
// broken, as the owner stopped it (SPEC §9.2).
const engineStopped = "Engine stopped"

func (n node) all() bool { return n.kind == "" }

func (n node) heading() string {
	switch n.kind {
	case "":
		return "ALL MACHINES"
	case machine.Computer:
		return strings.ToUpper(n.label)
	default:
		return strings.ToUpper(n.label) + ui.Sep + strings.ToUpper(string(n.kind))
	}
}

func tree(thisMac page.Context, p *page.Page) []node {
	nodes := []node{{label: allMachines}}
	this := node{name: thisMac.Machine, label: "This Mac", kind: machine.Computer, mark: page.Awaiting, thisMac: true}
	if p == nil {
		return append(nodes, this)
	}
	for _, s := range p.Sections {
		for _, it := range s.Items {
			if it.Group == nil {
				continue
			}
			h := it.Group.Head
			mac := node{name: h.ID, label: cmp.Or(h.Short, h.Label), kind: machine.Computer, mark: h.Mark, thisMac: h.ID == thisMac.Machine, reason: h.Reason}
			mac.engine = ui.Plain(column(s, h, "state"))
			if mac.engine == engineStopped {
				mac.mark = page.Attention
			}
			mac.facts = []string{h.ID}
			nodes = append(nodes, mac)
			var servers, workspaces []node
			for _, v := range it.Group.Items {
				if v.Row == nil {
					continue
				}
				r := *v.Row
				kind := ui.Plain(column(s, r, "kind"))
				vm := node{name: r.ID, label: r.Label, mark: r.Mark, mac: mac.name, kind: machine.Kind(strings.ToLower(kind))}
				vm.facts = append(vm.facts, kind)
				for _, w := range column(s, r, "state").Items {
					word := ui.Plain(w)
					vm.facts = append(vm.facts, strings.ToUpper(word[:1])+word[1:])
				}
				vm.facts = append(vm.facts, "on "+mac.label)
				if vm.kind == machine.Server {
					servers = append(servers, vm)
				} else {
					workspaces = append(workspaces, vm)
				}
			}
			nodes = append(append(nodes, servers...), workspaces...)
		}
	}
	return nodes
}

// value: a row's cell by its column's key, never by its place.
func column(sec page.Section, r page.Row, key string) page.Value {
	for i, c := range sec.Columns {
		if c.Key == key && i > 0 && i-1 < len(r.Cells) {
			return r.Cells[i-1]
		}
	}
	return page.None()
}

func find(nodes []node, name string) (node, bool) {
	for _, n := range nodes {
		if n.name == name {
			return n, true
		}
	}
	return node{}, false
}

func words(n node, s section.ID) []string {
	if n.all() {
		switch s {
		case section.Overview:
			return []string{"machines"}
		case section.Images:
			return []string{"images"}
		case section.Settings:
			return []string{"settings", "--all-machines", "--all"}
		default:
			return []string{string(s), "--all-machines"}
		}
	}
	var w []string
	switch s {
	case section.Overview:
		w = []string{"overview", "--all"}
	case section.SSH:
		return []string{"machines", string(s), n.name}
	default:
		w = []string{string(s)}
	}
	if !n.thisMac {
		w = append(w, "--on", n.name)
	}
	return w
}
