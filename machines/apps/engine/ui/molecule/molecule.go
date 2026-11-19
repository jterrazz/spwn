package molecule

import (
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/ui/atom"
	"github.com/jterrazz/studio-station/apps/engine/ui/token"
)

func HintText(h page.Hint) string { return token.Arrow + " " + h.Command }

func Hint(st token.Styles, h page.Hint) string {
	return st.Muted.Render(token.Arrow+" ") + st.Text.Render(h.Command)
}

func PageHint(st token.Styles, h page.Hint, at int) string {
	// value: a next step, its words after the command at a column `at` wide
	// (the widest command of the page's steps), so the steps read as one list.
	line := st.Muted.Render(token.Arrow+" ") + st.Text.Render(h.Command)
	if h.Text != "" {
		line += atom.Spaces(max(at-atom.Width(h.Command), 0)+token.Gap) + st.Muted.Render(h.Text)
	}
	return line
}

func Empty(st token.Styles, e page.Empty) string {
	text := "no " + e.Noun
	if e.Text != "" {
		text = e.Text
	}
	line := st.Muted.Render(text)
	if e.Hint == nil || e.Hint.Command == "" {
		return line
	}
	return line + atom.Spaces(token.Gap) + Hint(st, *e.Hint)
}

func ErrorLines(st token.Styles, e page.Error) []string {
	lines := []string{st.Error.Render("Error:") + " " + st.Text.Render(e.Message)}
	if e.Hint != "" {
		lines = append(lines, st.Muted.Render("Hint:")+" "+st.Text.Render(e.Hint))
	}
	return lines
}

func Note(st token.Styles, n page.Note, width int) []string {
	var lines []string
	line := ""
	for _, word := range strings.Fields(n.Text) {
		switch {
		case line == "":
			line = word
		case atom.Width(line)+1+atom.Width(word) <= width:
			line += " " + word
		default:
			lines = append(lines, st.Text.Render(line))
			line = word
		}
	}
	if line != "" {
		lines = append(lines, st.Text.Render(line))
	}
	return lines
}

func Fact(st token.Styles, key string, keyWidth int, value string) string {
	return st.Strong.Render(atom.Pad(key, keyWidth)) + atom.Spaces(token.Gap) + st.Text.Render(value)
}
