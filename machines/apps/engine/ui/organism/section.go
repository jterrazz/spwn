package organism

import (
	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/ui/atom"
	"github.com/jterrazz/studio-station/apps/engine/ui/token"
)

func Caption(st token.Styles, s page.Section) string {
	if len(s.Caption) == 0 {
		return ""
	}
	parts := make([]page.Value, 0, len(s.Caption))
	parts = append(parts, s.Caption...)
	return st.Muted.Render(atom.Show(page.List(parts...)))
}

func Title(st token.Styles, s page.Section) string {
	if s.Joined {
		return atom.Spaces(token.Margin) + st.Strong.Render(atom.Show(page.List(append([]page.Value{page.Text(s.Title)}, s.Caption...)...)))
	}
	// invariant: an open section's title counts nothing: its rows are there
	// to read (vision 07 "Updates"); a folded one counts them on its line.
	line := atom.Spaces(token.Margin) + st.Strong.Render(s.Title)
	if c := Caption(st, s); c != "" {
		line += atom.Spaces(token.Gap) + c
	}
	return line
}

func Rows(st token.Styles, s page.Section, width int) []string {
	body := (Grid{Styles: st, Section: s, Width: width}).Lines()
	lines := make([]string, 0, len(body))
	for _, l := range body {
		lines = append(lines, l.Text)
	}
	return lines
}

func Section(st token.Styles, s page.Section, width int) []string {
	body := Rows(st, s, width)
	lines := make([]string, 0, len(body)+1)
	// invariant: a section with no title and no caption reads on as its
	// rows (an Overview's lines under its Needs you, Awake's Macs), with no
	// blank head, whatever it counts.
	if s.Title != "" || len(s.Caption) > 0 {
		lines = append(lines, Title(st, s))
	}
	return append(lines, body...)
}
