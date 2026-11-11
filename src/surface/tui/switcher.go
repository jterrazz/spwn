package tui

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/jterrazz/studio-station/src/core/machine"
	ui "github.com/jterrazz/studio-station/src/ui/render/tui"
)

// value: Switch machine (SPEC §9.6), the modal m opens: its filter, typed,
// and the cursor on the machines it keeps.
type switcher struct {
	filter string
	at     int
}

// value: the machines the TUI showed this session, the newest first, the
// one shown left out; "" is All machines.
type visited struct {
	recent []string
}

func (v visited) left(from, to string) visited {
	if from == to {
		return v
	}
	recent := slices.DeleteFunc(slices.Clone(v.recent), func(n string) bool { return n == from || n == to })
	return visited{recent: append([]string{from}, recent...)}
}

func (v visited) previous() (string, bool) {
	if len(v.recent) == 0 {
		return "", false
	}
	return v.recent[0], true
}

type choice struct {
	name   string
	line   string
	recent bool
}

// invariant: the recent machines first, then every machine as the tree
// draws it, each VM with its kind and state; the filter keeps a machine
// whose label or name holds it.
func (s shell) choices(filter string) []choice {
	var out []choice
	keep := func(n node) bool {
		f := strings.ToLower(filter)
		return f == "" || strings.Contains(strings.ToLower(n.label), f) || strings.Contains(strings.ToLower(n.name), f)
	}
	for _, name := range s.seen.recent {
		if n, ok := s.nodeNamed(name); ok && keep(n) {
			out = append(out, choice{name: name, line: s.choiceLine(n, false), recent: true})
		}
	}
	for _, n := range s.nodes {
		if keep(n) {
			out = append(out, choice{name: n.name, line: s.choiceLine(n, true)})
		}
	}
	return out
}

func (s shell) nodeNamed(name string) (node, bool) {
	if name == "" {
		return s.nodes[0], true
	}
	return find(s.nodes, name)
}

func (s shell) choiceLine(n node, tree bool) string {
	if n.all() {
		return ui.AllMachinesMark + " " + allMachines
	}
	lead := ""
	if tree && n.kind != machine.Computer {
		lead = "  "
	}
	line := lead + ui.Glyph(n.mark) + " " + n.label
	if n.kind == machine.Computer {
		return line + "  Mac"
	}
	return line + "  " + strings.Join(n.facts, ui.Sep)
}

func (s shell) switching(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	sw := *s.sw
	key := k.String()
	list := s.choices(sw.filter)
	switch key {
	case "esc":
		s.sw = nil
		return s, nil
	case "up":
		sw.at = max(sw.at-1, 0)
	case "down":
		sw.at = min(sw.at+1, max(len(list)-1, 0))
	case "enter":
		s.sw = nil
		if sw.at < len(list) {
			return s.open(list[sw.at].name)
		}
		return s, nil
	case "backspace":
		if r := []rune(sw.filter); len(r) > 0 {
			sw.filter, sw.at = string(r[:len(r)-1]), 0
		}
	case "-":
		if sw.filter == "" {
			s.sw = nil
			return s.previous()
		}
		sw.filter, sw.at = sw.filter+"-", 0
	default:
		if t := k.Key().Text; t != "" {
			sw.filter, sw.at = sw.filter+t, 0
		}
	}
	s.sw = &sw
	return s, nil
}

func (s shell) previous() (tea.Model, tea.Cmd) {
	name, ok := s.seen.previous()
	if !ok {
		return s, nil
	}
	return s.open(name)
}

func (s shell) switcherPanel() *ui.Panel {
	if s.sw == nil {
		return nil
	}
	p := ui.Panel{Title: "Switch machine"}
	p.Lines = append(p.Lines, "", " "+ui.Typed+" "+s.sw.filter+"_", "")
	list := s.choices(s.sw.filter)
	heading := ""
	for i, c := range list {
		h := "ALL MACHINES"
		if c.recent {
			h = "RECENT"
		}
		if h != heading {
			heading = h
			p.Lines = append(p.Lines, " "+h)
		}
		lead := "   "
		if i == s.sw.at {
			lead = " " + ui.Pointer + " "
		}
		p.Lines = append(p.Lines, lead+c.line)
	}
	if len(list) == 0 {
		p.Lines = append(p.Lines, " No machine holds "+s.sw.filter+".")
	}
	p.Lines = append(p.Lines, "", " enter open   - previous   esc cancel")
	return &p
}

func (s shell) switcherKeys() []ui.Binding {
	return []ui.Binding{bind("type", "filter"), bind("up/down", "move"), bind("enter", "open"), bind("-", "previous"), bind("esc", "cancel")}
}
