package token

import (
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/term"
)

func IsTerminal(f term.File) bool { return term.IsTerminal(f.Fd()) }

func Size(f term.File) (width, height int) {
	w, h, err := term.GetSize(f.Fd())
	if err != nil {
		return 0, 0
	}
	return w, h
}

func DetectDark(in, out term.File) bool { return lipgloss.HasDarkBackground(in, out) }
