package atom

import (
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/ui/token"
)

func Glyph(m page.Mark) string {
	switch m {
	case page.OK:
		return token.MarkOK
	case page.Attention:
		return token.MarkAttention
	case page.Absent:
		return token.MarkAbsent
	case page.Broken:
		return token.MarkBroken
	case page.Awaiting:
		return token.MarkPending
	case page.OffHere:
		return token.MarkOffHere
	case page.NotForThisMachine:
		return token.MarkNotFor
	case page.NotKnown:
		return token.MarkUnknown
	case page.Unjudged:
	}
	return ""
}

func Tone(st token.Styles, m page.Mark) lipgloss.Style {
	switch m {
	case page.OK:
		return st.Good
	case page.Attention:
		return st.Warn
	case page.Broken:
		return st.Bad
	case page.NotForThisMachine:
		return st.Subtle
	case page.Absent, page.Awaiting, page.OffHere, page.NotKnown, page.Unjudged:
	}
	return st.Muted
}

func Mark(st token.Styles, m page.Mark) string {
	g := Glyph(m)
	if g == "" {
		return Pad("", token.MarkWidth)
	}
	return Tone(st, m).Render(g) + " "
}

func Count(t page.Tally) string { return strconv.Itoa(t.N) + " " + Glyph(t.Mark) }

func Paint(st token.Styles, base lipgloss.Style, v page.Value, text string) string {
	// invariant: OU-01: in a value's text only its glyphs take their mark's
	// tone; the words around them stay in base, so a one-line section reads
	// `● 5 ok · ◐ 3 updates` with its two glyphs toned and nothing else.
	marks := map[string]page.Mark{}
	marksOf(v, marks)
	if len(marks) == 0 || text == "" {
		return base.Render(text)
	}
	var b, run strings.Builder
	flush := func() {
		if run.Len() > 0 {
			b.WriteString(base.Render(run.String()))
			run.Reset()
		}
	}
	words := strings.Split(text, " ")
	for i, w := range words {
		if i > 0 {
			run.WriteString(" ")
		}
		if m, ok := marks[w]; ok {
			flush()
			b.WriteString(Tone(st, m).Render(w))
			continue
		}
		run.WriteString(w)
	}
	flush()
	return b.String()
}

func marksOf(v page.Value, out map[string]page.Mark) {
	if g := Glyph(page.Mark(v.Label)); g != "" && v.Kind != page.KindLevels {
		out[g] = page.Mark(v.Label)
	}
	for _, it := range v.Items {
		marksOf(it, out)
	}
}
