package token

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

type Palette struct {
	Text    color.Color
	Muted   color.Color
	Subtle  color.Color
	Surface color.Color
	Accent  color.Color
	Good    color.Color
	Warn    color.Color
	Bad     color.Color
}

func Colors(isDark bool) Palette {
	// value: Catppuccin Latte (light) and Macchiato (dark), the flavor the
	// embedded dotfiles already paint the terminal with.
	pick := lipgloss.LightDark(isDark)
	pair := func(light, dark string) color.Color {
		return pick(lipgloss.Color(light), lipgloss.Color(dark))
	}
	return Palette{
		Text:    pair("#4c4f69", "#cad3f5"),
		Muted:   pair("#6c6f85", "#a5adcb"),
		Subtle:  pair("#9ca0b0", "#6e738d"),
		Surface: pair("#ccd0da", "#363a4f"),
		Accent:  pair("#1e66f5", "#8aadf4"),
		Good:    pair("#40a02b", "#a6da95"),
		Warn:    pair("#df8e1d", "#eed49f"),
		Bad:     pair("#d20f39", "#ed8796"),
	}
}

type Styles struct {
	Colors  Palette
	Text    lipgloss.Style
	Strong  lipgloss.Style
	Muted   lipgloss.Style
	Subtle  lipgloss.Style
	Accent  lipgloss.Style
	Title   lipgloss.Style
	Good    lipgloss.Style
	Warn    lipgloss.Style
	Bad     lipgloss.Style
	Error   lipgloss.Style
	Surface lipgloss.Style
	Button  lipgloss.Style
	Idle    lipgloss.Style
}

func New(isDark bool) Styles {
	c := Colors(isDark)
	text := lipgloss.NewStyle().Foreground(c.Text)
	accent := lipgloss.NewStyle().Foreground(c.Accent)
	bad := lipgloss.NewStyle().Foreground(c.Bad)
	return Styles{
		Colors:  c,
		Text:    text,
		Strong:  text.Bold(true),
		Muted:   lipgloss.NewStyle().Foreground(c.Muted),
		Subtle:  lipgloss.NewStyle().Foreground(c.Subtle),
		Accent:  accent,
		Title:   accent.Bold(true),
		Good:    lipgloss.NewStyle().Foreground(c.Good),
		Warn:    lipgloss.NewStyle().Foreground(c.Warn),
		Bad:     bad,
		Error:   bad.Bold(true),
		Surface: lipgloss.NewStyle().Background(c.Surface),
		Button:  lipgloss.NewStyle().Foreground(c.Surface).Background(c.Accent).Bold(true).Padding(0, 1),
		Idle:    lipgloss.NewStyle().Foreground(c.Muted).Background(c.Surface).Padding(0, 1),
	}
}
