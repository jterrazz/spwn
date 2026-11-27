package wiki

import (
	"fmt"
	"strings"
)

// value: whether a character draws nothing or reorders the text around it:
// a C0 control but the line feed and the tab, DEL, a C1 control, a bidi
// embedding, override or isolate, and the line and paragraph separators.
// The joiners and variation selectors an emoji is drawn with are kept.
func silent(r rune) bool {
	switch {
	case r == '\n' || r == '\t':
		return false
	case r < 0x20, r == 0x7f, r >= 0x80 && r <= 0x9f:
		return true
	case r >= 0x202a && r <= 0x202e, r >= 0x2066 && r <= 0x2069:
		return true
	}
	return r == 0x2028 || r == 0x2029
}

// invariant: text enters the compiled model only through audible: a page's
// id and source and wiki.yaml's bytes leave it with no silent character,
// and say how many it removed. A text that holds none, as every honest one
// does, is scanned once and returned as it is.
func audible(text string) (kept string, removed int) {
	if !strings.ContainsFunc(text, silent) {
		return text, 0
	}
	n := 0
	return strings.Map(func(r rune) rune {
		if silent(r) {
			n++
			return -1
		}
		return r
	}, text), n
}

// value: the finding that says a file held characters the engine removed.
func silenced(page string, n int) Finding {
	return finding("S32", page, "", At{Line: 1, Col: 1}, fmt.Sprintf("%d characters that draw nothing or reorder text were removed", n))
}
