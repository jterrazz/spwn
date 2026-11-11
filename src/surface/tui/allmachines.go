package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/core/section"
	"github.com/jterrazz/studio-station/src/page"
	machinespage "github.com/jterrazz/studio-station/src/pages/machines"
	ui "github.com/jterrazz/studio-station/src/ui/render/tui"
)

const sparkPoints = 7

func (s shell) table() bool { return s.current().all() && s.section() == section.Overview }

type joinedRead struct {
	n  node
	id section.ID
}

// invariant: the All machines table reads, besides the machines page, the
// pages of Activity and Ports across every machine, and each Mac's
// Overview, which the sidebar reads already for its Needs you counts.
func (s shell) joined() []joinedRead {
	all := s.nodes[0]
	return []joinedRead{{all, section.Activity}, {all, section.Ports}}
}

func (s shell) load() (shell, tea.Cmd) {
	n, id := s.current(), s.section()
	if sc := s.screen(); sc != nil && sc.Live() > 0 {
		s.asked = with(s.asked, s.key(n, id), false)
	}
	s, cmd := s.ask(s.key(n, id), s.words(n, id))
	cmds := []tea.Cmd{cmd}
	for _, r := range s.besides(n, id) {
		var c tea.Cmd
		s, c = s.ask(besideKey(n, r), r.Words)
		cmds = append(cmds, c)
	}
	if !s.table() {
		return s, tea.Batch(cmds...)
	}
	for _, j := range s.joined() {
		var c tea.Cmd
		s, c = s.ask(s.key(j.n, j.id), s.words(j.n, j.id))
		cmds = append(cmds, c)
	}
	return s, tea.Batch(cmds...)
}

func (s shell) again() (shell, tea.Cmd) {
	s.asked = with(s.asked, s.key(s.current(), s.section()), false)
	for _, r := range s.besides(s.current(), s.section()) {
		s.asked = with(s.asked, besideKey(s.current(), r), false)
	}
	if s.table() {
		for _, j := range s.joined() {
			s.asked = with(s.asked, s.key(j.n, j.id), false)
		}
	}
	return s.load()
}

func (s shell) list(key string) ui.List {
	l := s.lists[key]
	if key != machinesKey || s.machines == nil {
		return s.viewed(key, l)
	}
	in := machinespage.TableIn{Machines: *s.machines, Overviews: map[string]page.Page{}, Samples: s.samples}
	for _, j := range s.joined() {
		if p, ok := s.page(j.n, j.id); ok {
			switch j.id {
			case section.Activity:
				in.Activity = &p
			default:
				in.Ports = &p
			}
		}
	}
	for _, n := range s.nodes {
		if p, ok := s.page(n, section.Overview); ok && n.kind == machine.Computer {
			in.Overviews[n.name] = p
		}
	}
	return l.Show(machinespage.Table(in))
}

func (s shell) opened() (string, bool) {
	_, row, ok := s.list(machinesKey).Current()
	if !ok {
		return "", false
	}
	name, _, _ := strings.Cut(row.ID, ":")
	_, known := find(s.nodes, name)
	return name, known
}
