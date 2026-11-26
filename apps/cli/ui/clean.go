package ui

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Clean is s with every rune strconv.IsPrint refuses written as Go escapes
// it (ESC as \x1b, LF as \n, tab as \t, U+202E and U+200B by their code
// point) and every byte that is not UTF-8 as
// \xNN: a file name, a link's text or any value a project wrote can then
// neither move the cursor, recolour, erase or reorder what spwn prints,
// nor hide a character. Every writer of this package passes the text it is
// given through Clean.
func Clean(s string) string {
	var b strings.Builder
	for len(s) > 0 {
		r, n := utf8.DecodeRuneInString(s)
		switch {
		case r == utf8.RuneError && n == 1:
			fmt.Fprintf(&b, `\x%02x`, s[0])
		case strconv.IsPrint(r):
			b.WriteString(s[:n])
		default:
			q := strconv.QuoteRune(r)
			b.WriteString(q[1 : len(q)-1])
		}
		s = s[n:]
	}
	return b.String()
}

// CleanLines is Clean of each line of s, the line breaks kept: the text of
// an error spwn wrote over several lines.
func CleanLines(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = Clean(l)
	}
	return strings.Join(lines, "\n")
}

// CleanError is CleanLines of err's text.
func CleanError(err error) string { return CleanLines(err.Error()) }
