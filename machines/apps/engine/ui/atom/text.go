package atom

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/jterrazz/studio-station/apps/engine/ui/token"
)

func Width(s string) int { return lipgloss.Width(s) }

func Pad(s string, width int) string {
	if w := lipgloss.Width(s); w < width {
		return s + strings.Repeat(" ", width-w)
	}
	return s
}

func PadLeft(s string, width int) string {
	if w := lipgloss.Width(s); w < width {
		return strings.Repeat(" ", width-w) + s
	}
	return s
}

func Spaces(n int) string { return strings.Repeat(" ", max(n, 0)) }

func End(s string, width int) string {
	if lipgloss.Width(s) <= width {
		return s
	}
	if width <= 1 {
		return token.Ellipsis
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes))+1 > width {
		runes = runes[:len(runes)-1]
	}
	return strings.TrimRight(string(runes), " ") + token.Ellipsis
}

func Middle(s string, width int) string {
	if lipgloss.Width(s) <= width {
		return s
	}
	if width <= 2 {
		return End(s, width)
	}
	keep := width - 1
	head := (keep + 1) / 2
	return joined(s, head, keep-head)
}

func joined(s string, head, tail int) string {
	runes := []rune(s)
	i := 0
	for i < len(runes) && lipgloss.Width(string(runes[:i+1])) <= head {
		i++
	}
	j := len(runes)
	for j > i && lipgloss.Width(string(runes[j-1:])) <= tail {
		j--
	}
	return string(runes[:i]) + token.Ellipsis + string(runes[j:])
}
