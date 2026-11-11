package tui

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/surface/tui/screens/palette"
	"github.com/jterrazz/studio-station/src/surface/tui/screens/screen"
	ui "github.com/jterrazz/studio-station/src/ui/render/tui"
)

// value: the palette ctrl+p opens (SPEC §9.5): its query, its cursor and
// the machine it offers the calls of, by its place in the tree.
type actionsState struct {
	query  string
	at     int
	target int
}

// value: the calls the palette ran this session, the newest first.
const recentActions = 5

func (s shell) target(n node) palette.Target {
	t := palette.Target{Name: n.name, Label: n.label, Kind: n.kind, ThisMachine: n.thisMac, Mac: n.mac, Sections: s.sections(n)}
	if mac, ok := find(s.nodes, n.mac); ok && n.mac != "" {
		t.MacHere, t.MacLabel = mac.thisMac, mac.label
	}
	if n.kind == machine.Computer {
		for _, v := range s.nodes {
			if v.mac == n.name && v.kind != machine.Computer {
				t.VMs = append(t.VMs, s.target(v))
			}
		}
	}
	return t
}

func (s shell) targets() []palette.Target {
	out := make([]palette.Target, 0, len(s.nodes))
	for _, n := range s.nodes {
		out = append(out, s.target(n))
	}
	return out
}

func (s shell) offered() []palette.Item {
	if s.pal == nil {
		return nil
	}
	ts := s.targets()
	t := ts[min(s.pal.target, len(ts)-1)]
	var macs []palette.Target
	for _, x := range ts {
		if x.Kind == machine.Computer {
			macs = append(macs, x)
		}
	}
	return palette.Search(palette.Offer(s.deps.Calls, t, macs), s.pal.query)
}

func (s shell) actions() (tea.Model, tea.Cmd) {
	at := max(slices.IndexFunc(s.nodes, func(n node) bool { return n.name == s.current().name }), 0)
	s.pal = &actionsState{target: at}
	return s, nil
}

func (s shell) choosing(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	p := *s.pal
	items := s.offered()
	switch k.String() {
	case "esc":
		s.pal = nil
		return s, nil
	case "up":
		p.at = max(p.at-1, 0)
	case "down":
		p.at = min(p.at+1, max(len(items)-1, 0))
	case "tab":
		p.target, p.at = (p.target+1)%len(s.nodes), 0
	case "shift+tab":
		p.target, p.at = (p.target+len(s.nodes)-1)%len(s.nodes), 0
	case "backspace":
		if r := []rune(p.query); len(r) > 0 {
			p.query, p.at = string(r[:len(r)-1]), 0
		}
	case "enter":
		s.pal = nil
		if p.at < len(items) {
			return s.runAction(items[p.at])
		}
		return s, nil
	default:
		if t := k.Key().Text; t != "" {
			p.query, p.at = p.query+t, 0
		}
	}
	s.pal = &p
	return s, nil
}

// invariant: an entry runs as its section's key does: its command in the
// Progress strip, asking what the call asks; one that needs input opens
// its form first.
func (s shell) runAction(it palette.Item) (tea.Model, tea.Cmd) {
	s.recent = append([]palette.Item{it}, slices.DeleteFunc(slices.Clone(s.recent), func(x palette.Item) bool { return x.Label == it.Label })...)
	s.recent = s.recent[:min(len(s.recent), recentActions)]
	switch {
	case it.Create != "":
		return s.createForm(it.Create)
	case it.Copy != nil:
		return s.call(screen.Call{Words: it.Copy, Copy: true, Doing: "Reading the ssh command"})
	case it.Then != nil:
		f := &screen.Form{Title: strings.TrimSuffix(it.Label, ui.Ellipsis)}
		for _, x := range it.Fields {
			f.Fields = append(f.Fields, screen.Field{Label: x.Label, Value: x.Value, Note: x.Note, Secret: x.Secret})
		}
		d := &dialog{call: screen.Call{Form: f}, then: it.Then}
		for _, x := range f.Fields {
			d.answers = append(d.answers, x.Value)
		}
		s.dialog = d
		return s, nil
	}
	return s.perform(it.Gesture)
}

func (s shell) actionsPanel() *ui.Panel {
	if s.pal == nil {
		return nil
	}
	ts := s.targets()
	t := ts[min(s.pal.target, len(ts)-1)]
	on := "on " + t.Label
	p := ui.Panel{Title: "Actions" + ui.Sep + on}
	p.Lines = append(p.Lines, "", " "+ui.Typed+" "+s.pal.query+"_", "")
	items := s.offered()
	width := s.width - 6
	room := max(s.height-14, 3)
	first := max(0, s.pal.at-room+1)
	for i := first; i < len(items) && i < first+room; i++ {
		it := items[i]
		lead := "   "
		if i == s.pal.at {
			lead = " " + ui.Pointer + " "
		}
		text := it.Label
		if it.Confirm != "" {
			text += ui.Sep + it.Confirm
		}
		p.Lines = append(p.Lines, lead+ui.Keyed(text, it.Key, width))
	}
	if len(items) == 0 {
		p.Lines = append(p.Lines, " Nothing "+on+" matches "+s.pal.query+".")
	}
	if s.pal.query == "" && len(s.recent) > 0 {
		p.Lines = append(p.Lines, "", " RECENT")
		for _, it := range s.recent {
			p.Lines = append(p.Lines, "   "+it.Label)
		}
	}
	next := ts[(s.pal.target+1)%len(ts)]
	p.Lines = append(p.Lines, "", " tab "+on+" "+ui.Arrow+" on "+next.Label+"   enter run   esc close")
	return &p
}

func (s shell) actionsKeys() []ui.Binding {
	return []ui.Binding{bind("type", "search"), bind("up/down", "move"), bind("tab", "next machine"), bind("enter", "run"), bind("esc", "close")}
}
