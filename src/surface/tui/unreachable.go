package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/core/section"
	"github.com/jterrazz/studio-station/src/page"
	ui "github.com/jterrazz/studio-station/src/ui/render/tui"
)

// value: the Mac shown does not answer: the machines read says so, and its
// own Overview could not be read (SPEC §9.10).
func (s shell) unanswered() (node, bool) {
	n := s.current()
	if n.kind != machine.Computer || n.thisMac || n.reason != page.Unreachable || s.section() != section.Overview {
		return node{}, false
	}
	r := s.reads[s.key(n, section.Overview)]
	return n, r.done && r.err != nil
}

// value: the fix the machines read gives the Mac, when it is station's
// (`machines unlock mac-mini`).
func (s shell) unansweredFix(n node) []string {
	if s.machines == nil {
		return nil
	}
	for _, sec := range s.machines.Sections {
		if sec.ID != "needs-you" {
			continue
		}
		for _, it := range sec.Items {
			if it.Row == nil || it.Row.Label != n.label {
				continue
			}
			if w, ours := strings.CutPrefix(ui.Plain(column(sec, *it.Row, "fix")), "station "); ours {
				return strings.Fields(w)
			}
		}
	}
	return nil
}

func (s shell) unansweredLines(n node) []string {
	since := strings.TrimPrefix(n.engine, "Unreachable"+ui.Sep)
	head := " " + ui.Glyph(page.Broken) + " " + n.label + " does not answer"
	if since != n.engine {
		head += ui.Sep + since
	}
	lines := []string{head, "", " What station knows", "   " + ui.Glyph(page.Broken) + " " + n.label + " answers neither station nor ssh"}
	fix := s.unansweredFix(n)
	if len(fix) > 1 && fix[1] == "unlock" {
		lines = append(lines, "   "+ui.Glyph(page.Attention)+" It runs a server: after a power cut, its disk unlock needs you", "",
			" Likely: it restarted after a power cut and waits at the FileVault unlock.",
			" "+catalog.TailnetProduct+" starts only after the disk is unlocked, so unlock it from the",
			" same network as "+n.label+":", "",
			"   "+ui.Arrow+" u Unlock"+ui.Ellipsis+"   station "+strings.Join(fix, " ")+"   (asks its password)")
	}
	for _, x := range s.nodes {
		if x.mac == n.name && x.kind == machine.Server {
			lines = append(lines, "", " "+x.label+" is a server on "+n.label+": its state is unknown until "+n.label+" answers.",
				" It starts by itself once "+n.label+" is unlocked and logged in.")
		}
	}
	return lines
}

func (s shell) unansweredKeys(n node) []ui.Binding {
	keys := []ui.Binding{bind("r", "retry now")}
	if fix := s.unansweredFix(n); len(fix) > 1 && fix[1] == "unlock" {
		keys = append([]ui.Binding{bind("u", "unlock"+ui.Ellipsis)}, keys...)
	}
	return keys
}

// invariant: u opens a new terminal on the unlock, which asks the Mac's
// password there; r reads the Mac and the machines again at once.
func (s shell) unansweredPress(n node, key string) (shell, tea.Cmd, bool) {
	switch key {
	case "u":
		if fix := s.unansweredFix(n); len(fix) > 1 && fix[1] == "unlock" {
			line := "station " + strings.Join(fix, " ")
			opened, cmd := s.outside("A new terminal runs "+line, func() error { return s.deps.Terminal(s.ctx, line) })
			return opened, cmd, true
		}
	case "r":
		s.asked = with(s.asked, machinesKey, false)
		read, cmd := s.again()
		read, more := read.ask(machinesKey, []string{"machines"})
		return read, tea.Batch(cmd, more), true
	}
	return s, nil, false
}
