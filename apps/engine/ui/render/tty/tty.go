package tty

import (
	"io"
	"strconv"
	"strings"

	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"

	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/ui/atom"
	"github.com/jterrazz/studio-station/apps/engine/ui/template"
	"github.com/jterrazz/studio-station/apps/engine/ui/token"
)

type Options struct {
	Width int
	Dark  bool
}

func Render(p page.Page, o Options) []byte {
	return []byte(strings.Join(template.Document(token.New(o.Dark), p, o.Width), "\n") + "\n")
}

func Colored(w io.Writer, environ []string) io.Writer { return colorprofile.NewWriter(w, environ) }

func Frame(p page.Page, o Options, height int) []string {
	st := token.New(o.Dark)
	lines := template.Document(st, p, o.Width)
	for i, l := range lines {
		lines[i] = ansi.Truncate(l, max(o.Width-1, 1), "")
	}
	if height <= 1 || len(lines) < height {
		return lines
	}
	room := max(height-2, 1)
	more := token.Ellipsis + " " + strconv.Itoa(len(lines)-room) + " more lines"
	if n := pendingRows(p); n > 0 {
		more = token.Ellipsis + " " + strconv.Itoa(n) + " rows pending"
	}
	return append(lines[:room], atom.Spaces(token.Margin)+st.Muted.Render(more))
}

func pendingRows(p page.Page) int {
	n := 0
	for _, s := range p.Sections {
		for _, t := range s.Counts {
			if t.Mark == page.Awaiting {
				n += t.N
			}
		}
	}
	return n
}

func Draw(lines []string, erase int) string {
	return Erase(erase) + ansi.HideCursor + strings.Join(lines, "\n") + "\n" + ansi.ShowCursor
}

func Erase(lines int) string {
	// invariant: the region is erased line by line, never by erasing the screen
	// below: once a clipped frame's top is the screen's first cell, that erase
	// clears the whole screen, and tmux's scroll-on-clear keeps every such screen
	// in its history.
	if lines == 0 {
		return ""
	}
	return "\r" + strings.Repeat(ansi.CUU1+ansi.EraseEntireLine, lines)
}
