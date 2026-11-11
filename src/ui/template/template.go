package template

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/ui/atom"
	"github.com/jterrazz/studio-station/src/ui/molecule"
	"github.com/jterrazz/studio-station/src/ui/organism"
	"github.com/jterrazz/studio-station/src/ui/token"
)

func Document(st token.Styles, p page.Page, width int) []string {
	// invariant: the CLI page starts at its own first line, at the terminal's
	// first column (vision 07): no header, no rule, no margin; the organisms lay
	// out at the margin the kit keeps, and the document takes it off.
	lines := body(st, p, width+token.Margin)
	for i, l := range lines {
		lines[i] = unindent(l)
	}
	for len(lines) > 0 && lines[0] == "" {
		lines = lines[1:]
	}
	return lines
}

func body(st token.Styles, p page.Page, width int) []string {
	var lines []string
	if p.Error != nil {
		lines = append(lines, "")
		for _, l := range molecule.ErrorLines(st, *p.Error) {
			lines = append(lines, atom.Spaces(token.Margin)+l)
		}
		return lines
	}
	first := firstLine(p.Sections)
	fold := foldWidth(p.Sections)
	for i, s := range p.Sections {
		if i == first+1 {
			lines = append(lines, "")
		}
		switch {
		case i == first && len(s.Items) == 1 && s.Items[0].Row != nil && !s.Joined:
			lines = append(lines, summed(st, *s.Items[0].Row, width, p.ID == page.Outcome)...)
			if h := s.Items[0].Row.Hint; h != nil && !elsewhere(p, i, h.Command) {
				lines = append(lines, atom.Spaces(token.Margin+token.Indent)+molecule.PageHint(st, *h, 0))
			}
		case i == first && s.Joined:
			lines = append(lines, sums(st, append([]page.Value{page.Text(s.Title)}, s.Caption...), width, true)...)
			if len(s.Items) > 0 {
				lines = append(lines, "")
				lines = append(lines, organism.Rows(st, s, width)...)
			}
		case s.Rest != nil && s.Top > 0 && len(s.Items) > s.Top:
			cut := s
			cut.Items = append(slices.Clone(s.Items[:s.Top]), *s.Rest)
			lines = append(lines, organism.Section(st, cut, width)...)
		case (s.Fold == page.Folded || s.Fold == page.Brief || s.Fold == page.Calm) && sized(s):
			lines = append(lines, sizes(st, s, width))
		case s.Fold == page.Folded || s.Fold == page.Brief || s.Fold == page.Calm:
			lines = append(lines, folded(st, s, fold, width))
		case single(s):
			lines = append(lines, oneRow(st, s, fold, width)...)
		case noted(s) && atom.Width(atom.Pad(s.Title, fold))+token.Gap+atom.Width(s.Items[0].Note.Text) < width-token.Margin:
			lines = append(lines, atom.Spaces(token.Margin)+st.Strong.Render(atom.Pad(s.Title, fold))+atom.Spaces(token.Gap)+st.Muted.Render(s.Items[0].Note.Text))
		default:
			lines = append(lines, organism.Section(st, s, width)...)
		}
	}
	if len(p.Hints) > 0 {
		lines = append(lines, "")
		at := 0
		for _, h := range p.Hints {
			if h.Text != "" {
				at = max(at, atom.Width(h.Command))
			}
		}
		for _, h := range p.Hints {
			lines = append(lines, atom.Spaces(token.Margin)+molecule.PageHint(st, h, at))
		}
	}
	return lines
}

// value: the section that sums the page up (SPEC §5: the first line), after
// the On line; the one blank line of a page's body follows it, the others
// read on without one (vision 07). -1 when the page has none.
func firstLine(sections []page.Section) int {
	for i, s := range sections {
		if s.ID == page.OnSection {
			continue
		}
		if s.Role == page.Banner || s.Role == page.Summary {
			return i
		}
		return -1
	}
	return -1
}

// value: a page summed up by one machine's row reads as its first line,
// `This Mac · 108 apps for this Mac · 2 updates` (vision 07 "station apps").
//
// invariant: an outcome's line leads with its mark, a glyph with its word
// (SPEC §2.3, §12.1), a step that passed ticked:
// `✓ This Mac stays awake until 18:00 · lid closed`.
func summed(st token.Styles, r page.Row, width int, outcome bool) []string {
	label := r.Label
	if r.Short != "" {
		label = r.Short
	}
	parts := []page.Value{page.Text(label)}
	for _, c := range append(r.Cells, r.Detail) {
		if c.Kind == page.KindList {
			parts = append(parts, c.Items...)
		} else if !c.IsNone() {
			parts = append(parts, c)
		}
	}
	if r.Sentence && len(parts) > 1 {
		parts = append([]page.Value{page.Text(label + " " + atom.Show(parts[1]))}, parts[2:]...)
	}
	if !outcome || r.Mark == page.Unjudged {
		return sums(st, parts, width, false)
	}
	glyph := atom.Glyph(r.Mark)
	if r.Mark == page.OK {
		glyph = token.Tick
	}
	lines := sums(st, parts, width-atom.Width(glyph)-1, false)
	lines[0] = atom.Spaces(token.Margin) + atom.Tone(st, r.Mark).Render(glyph) + " " + strings.TrimPrefix(lines[0], atom.Spaces(token.Margin))
	return lines
}

// invariant: a first line is text, as vision 07 draws it, and never cut: a
// part the line cannot hold goes on the next one, two columns in
// (`This Mac · jterrazz-macbook-pro · 9 things need you` / `  Engine running`).
// A titled one leads with its machine in bold.
func sums(st token.Styles, parts []page.Value, width int, titled bool) []string {
	room := width - 1 - token.Margin
	var lines []string
	line, plain := "", ""
	for i, v := range parts {
		text := atom.Show(v)
		style := st.Text
		if i == 0 && titled {
			style = st.Strong
		}
		switch {
		case text == "":
		case plain == "":
			line, plain = style.Render(text), text
		case atom.Width(plain)+atom.Width(token.Sep)+atom.Width(text) <= room:
			line, plain = line+st.Text.Render(token.Sep)+style.Render(text), plain+token.Sep+text
		default:
			lines = append(lines, line)
			line, plain = atom.Spaces(token.Indent)+style.Render(text), atom.Spaces(token.Indent)+text
		}
	}
	lines = append(lines, line)
	for i, l := range lines {
		lines[i] = atom.Spaces(token.Margin) + l
	}
	return lines
}

// value: whether a hint the first line leaves out is on another row of the
// page, or one of its next steps.
func elsewhere(p page.Page, skip int, command string) bool {
	for _, h := range p.Hints {
		if h.Command == command {
			return true
		}
	}
	var walk func(items []page.Item) bool
	walk = func(items []page.Item) bool {
		for _, it := range items {
			switch {
			case it.Row != nil && it.Row.Hint != nil && it.Row.Hint.Command == command:
				return true
			case it.Group != nil && (it.Group.Head.Hint != nil && it.Group.Head.Hint.Command == command || walk(it.Group.Items)):
				return true
			}
		}
		return false
	}
	for i, s := range p.Sections {
		if i != skip && walk(s.Items) {
			return true
		}
	}
	return false
}

var styled = regexp.MustCompile(`^((?:\x1b\[[0-9;:]*m)*) `)

// value: a line with the margin the organisms keep taken off, past the
// styles a line may open with.
func unindent(l string) string {
	for range token.Margin {
		l = styled.ReplaceAllString(l, "$1")
	}
	return l
}

// invariant: a calm section that only says something is one line, its title
// then what it says (`Not for this machine  7 apps live on other machines`),
// when the line holds it.
func noted(s page.Section) bool {
	return s.Title != "" && s.Role == page.Plain && len(s.Items) == 1 && s.Items[0].Note != nil
}

func foldWidth(sections []page.Section) int {
	n := 0
	for _, s := range sections {
		if (s.Fold == page.Folded || s.Fold == page.Brief || s.Fold == page.Calm) && !sized(s) || single(s) && !said(s) {
			n = max(n, atom.Width(s.Title))
		}
	}
	return n
}

// invariant: a folded section is one line: its title, its count by mark
// (`○ 34`), then as many of its rows' names as the line holds (vision 07:
// "Not installed   ○ 34 · yazi, dust, eza, …"); --all unfolds it.
func folded(st token.Styles, s page.Section, at, width int) string {
	line := atom.Spaces(token.Margin) + st.Strong.Render(atom.Pad(s.Title, at))
	var counts []string
	for _, t := range s.Counts {
		if t.N > 0 {
			counts = append(counts, atom.Tone(st, t.Mark).Render(atom.Glyph(t.Mark))+" "+st.Text.Render(strconv.Itoa(t.N)))
		}
	}
	if len(counts) == 0 {
		return strings.TrimRight(line, " ")
	}
	line += atom.Spaces(token.Gap) + strings.Join(counts, atom.Spaces(token.Gap))
	var names []string
	for _, it := range s.Items {
		switch {
		case it.Row != nil && it.Row.Short != "":
			names = append(names, it.Row.Short)
		case it.Row != nil:
			names = append(names, it.Row.Label)
		}
	}
	list := fit(names, width-1-atom.Width(line)-atom.Width(token.Sep))
	if list == "" {
		return line
	}
	return line + st.Muted.Render(token.Sep+list)
}

// value: as many names as room holds, then `…` for the others; empty when
// not even the first fits.
func fit(names []string, room int) string {
	list := ""
	for i, n := range names {
		next := list
		if i > 0 {
			next += ", "
		}
		next += n
		more := ""
		if i < len(names)-1 {
			more = ", " + token.Ellipsis
		}
		if atom.Width(next+more) > room {
			if list == "" {
				return ""
			}
			return list + ", " + token.Ellipsis
		}
		list = next
	}
	return list
}

// value: a folded section of sizes, not verdicts: its caption is its total.
func sized(s page.Section) bool {
	return s.Uncounted && s.Joined && len(s.Caption) > 0
}

// invariant: a folded section of sizes is one line: its title, its total,
// then as many of its rows, each with its size, as the line holds, a size in
// the total's unit without it (vision 07 "station disk": `Your data ·
// 41.6 GB · xcode 19.1, orbstack 15.8, …`).
func sizes(st token.Styles, s page.Section, width int) string {
	total := atom.Show(page.List(s.Caption...))
	line := atom.Spaces(token.Margin) + st.Strong.Render(s.Title+token.Sep+total)
	_, unit, _ := strings.Cut(total, " ")
	var names []string
	for _, it := range s.Items {
		if it.Row == nil {
			continue
		}
		name := shown(*it.Row)
		if len(it.Row.Cells) > 0 {
			size := atom.Show(it.Row.Cells[0])
			if n, u, ok := strings.Cut(size, " "); ok && u == unit {
				size = n
			}
			if size != "" {
				name += " " + size
			}
		}
		names = append(names, name)
	}
	list := fit(names, width-1-atom.Width(line)-atom.Width(token.Sep))
	if list == "" {
		return line
	}
	return line + st.Muted.Render(token.Sep+list)
}

func shown(r page.Row) string {
	if r.Short != "" {
		return r.Short
	}
	return r.Label
}

// invariant: a calm section of one row with nothing to do is one line
// (vision 07 "Off  ⊘ 1 · Claude token"): its title, then the row's mark,
// its label unless the title says it, and its cells; a row whose label
// says the title is the line alone (`– 12 not for this machine · Mac only`);
// a table, whose columns are headed, keeps its heads.
func single(s page.Section) bool {
	if s.Title == "" || s.Role != page.Plain && s.Role != page.Banner || len(s.Items) != 1 || s.Items[0].Row == nil {
		return false
	}
	for _, c := range s.Columns {
		if c.Label != "" {
			return false
		}
	}
	r := s.Items[0].Row
	return r.Hint == nil && r.Mark != page.Attention && r.Mark != page.Broken && r.Mark != page.Awaiting
}

func said(s page.Section) bool {
	label := s.Items[0].Row.Label
	return label != s.Title && strings.Contains(strings.ToLower(label), strings.ToLower(s.Title))
}

func oneRow(st token.Styles, s page.Section, at, width int) []string {
	r := *s.Items[0].Row
	var parts []string
	if r.Label != s.Title {
		parts = append(parts, r.Label)
	}
	for _, c := range append(r.Cells, r.Detail) {
		values := []page.Value{c}
		if c.Kind == page.KindList {
			values = c.Items
		}
		for _, v := range values {
			if text := atom.Show(v); text != "" && !v.IsNone() {
				parts = append(parts, text)
			}
		}
	}
	lead := atom.Spaces(token.Margin)
	if !said(s) {
		lead += st.Strong.Render(atom.Pad(s.Title, at)) + atom.Spaces(token.Gap)
	}
	if r.Mark != page.Unjudged {
		lead += atom.Tone(st, r.Mark).Render(atom.Glyph(r.Mark)) + " "
	}
	// invariant: like a first line, the line is never cut: a part it cannot
	// hold goes on the next one, under the first.
	room := width - 1 - token.Margin
	var lines []string
	line, used := lead, atom.Width(lead)-token.Margin
	first := true
	for _, part := range parts {
		switch {
		case first:
			line, used = line+st.Text.Render(part), used+atom.Width(part)
			first = false
		case used+atom.Width(token.Sep)+atom.Width(part) <= room:
			line, used = line+st.Text.Render(token.Sep+part), used+atom.Width(token.Sep)+atom.Width(part)
		default:
			lines = append(lines, line)
			pad := atom.Width(lead)
			line, used = atom.Spaces(pad)+st.Text.Render(part), pad-token.Margin+atom.Width(part)
		}
	}
	return append(lines, line)
}
