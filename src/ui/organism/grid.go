package organism

import (
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/ui/atom"
	"github.com/jterrazz/studio-station/src/ui/molecule"
	"github.com/jterrazz/studio-station/src/ui/token"
)

// value: an External command chained by `&&` breaks after each joint when it
// is wider than the terminal, so it stays whole and copyable (H3, G5).
const joint = " && "

type LineKind int

const (
	LineTitle LineKind = iota
	LineHeader
	LineRow
	LineHint
	LineOther
)

type Line struct {
	Key  string
	Kind LineKind
	Head bool
	Text string
}

type Grid struct {
	Styles  token.Styles
	Section page.Section
	Width   int
	Folded  func(key string) bool
	Match   func(r page.Row) bool
}

type column struct {
	kind  page.Kind
	label string
	keep  int
	whole bool
	width int
	firm  int
	cell  func(r page.Row) page.Value
}

type entry struct {
	empty  *page.Empty
	link   *page.Link
	row    page.Row
	prefix string
	head   bool
	key    string
	// value: this row's last cell is wider than the section can give beside
	// the others' details: it runs to the line's end on its own (§11.2).
	overflow bool
	// value: on a terminal, this row's name does not fit its column, or its
	// command does not fit beside the names: the name is written whole and
	// what follows it goes on the lines under it (OU-05).
	stack bool
}

func RowKey(section, row string) string { return section + "/" + row }

func shown(r page.Row) string {
	if r.Short != "" {
		return r.Short
	}
	return r.Label
}

func (g Grid) Lines() []Line {
	// invariant: a Summary wider than the narrow width reads as two columns, its
	// Break or its first half on the left (§13.1); each half keeps its own grid.
	if g.Section.Role == page.Summary && g.Width > token.NarrowWidth && len(g.Section.Items) > 1 {
		return g.split()
	}
	return g.lines()
}

func (g Grid) split() []Line {
	half := (len(g.Section.Items) + 1) / 2
	if b := g.Section.Break; b > 0 && b < len(g.Section.Items) {
		half = b
	}
	left, right := g, g
	left.Section.Items, right.Section.Items = g.Section.Items[:half], g.Section.Items[half:]
	natural := func(lines []Line) int {
		n := 0
		for _, l := range lines {
			n = max(n, atom.Width(l.Text))
		}
		return n
	}
	lw, rw := natural(left.lines()), natural(right.lines())
	leftWidth := max(lw+1, token.MinWidth)
	if lw+rw >= g.Width {
		leftWidth = max(g.Width/2, g.Width-rw)
	}
	if g.Width+1-leftWidth < token.MinWidth {
		return g.lines()
	}
	left.Width, right.Width = leftWidth, g.Width+1-leftWidth
	l, r := left.lines(), right.lines()
	out := make([]Line, 0, len(l))
	for i := range l {
		line := l[i]
		if i < len(r) {
			line.Text += atom.Spaces(leftWidth-1-atom.Width(line.Text)) + r[i].Text
		}
		out = append(out, line)
	}
	return append(out, r[min(len(l), len(r)):]...)
}

func (g Grid) lines() []Line {
	entries := g.entries()
	cols, labelWidth := g.layout(entries)
	var out []Line
	if header := g.header(cols, labelWidth); header != "" && len(entries) > 0 {
		out = append(out, Line{Kind: LineHeader, Text: header})
	}
	factWidth := 0
	for _, it := range g.Section.Items {
		if it.Fact != nil {
			factWidth = max(factWidth, atom.Width(it.Fact.Key))
		}
	}
	at := 0
	for _, it := range g.Section.Items {
		at = g.item(&out, it, entries, at, cols, labelWidth, factWidth)
	}
	return out
}

func (g Grid) item(out *[]Line, it page.Item, entries []entry, at int, cols []column, labelWidth, factWidth int) int {
	st, indent := g.Styles, atom.Spaces(token.Margin+g.indent())
	switch {
	case it.Row != nil, it.Group != nil:
		n := countEntries(it)
		for _, e := range entries[at : at+n] {
			switch {
			case e.empty != nil && e.key != "" && e.prefix != "":
				*out = append(*out, Line{Kind: LineOther, Text: atom.Spaces(token.Margin+g.indent()) + e.prefix + molecule.Empty(g.Styles, *e.empty)})
			case e.empty != nil && e.key != "":
				*out = append(*out, Line{Kind: LineOther, Text: atom.Spaces(token.Margin+g.indent()) + g.Styles.Subtle.Render(e.prefix) + atom.Spaces(token.MarkWidth) + molecule.Empty(g.Styles, *e.empty)})
			case e.link != nil && e.key != "":
				*out = append(*out, Line{Kind: LineOther, Text: g.link(e.prefix, *e.link, labelWidth)})
			case e.key != "":
				*out = append(*out, g.row(e, cols, labelWidth)...)
			}
		}
		return at + n
	case it.Fact != nil:
		*out = append(*out, Line{Kind: LineOther, Text: indent + molecule.Fact(st, it.Fact.Key, factWidth, atom.Show(it.Fact.Value))})
	case it.Empty != nil:
		*out = append(*out, Line{Kind: LineOther, Text: indent + molecule.Empty(st, *it.Empty)})
	case it.Error != nil:
		for _, l := range molecule.ErrorLines(st, *it.Error) {
			*out = append(*out, Line{Kind: LineOther, Text: indent + l})
		}
	case it.Note != nil:
		for _, l := range molecule.Note(st, *it.Note, max(g.Width-1-len(indent), 10)) {
			*out = append(*out, Line{Kind: LineOther, Text: indent + l})
		}
	case it.Link != nil:
		*out = append(*out, Line{Kind: LineOther, Text: g.link("", *it.Link, labelWidth)})
	}
	return at
}

// invariant: a link leads with its Lead (never cut) or its label, its value
// starts at the first column, cut by its kind (a URL keeps its scheme and
// port), and its detail is Whole: dropped when the line is short (§13.5).
func (g Grid) link(prefix string, l page.Link, labelWidth int) string {
	st := g.Styles
	end := g.Width - 1
	slot := labelWidth - atom.Width(prefix) + g.markWidth()
	head := atom.Show(l.Lead)
	if head == "" {
		head = atom.End(l.Label, max(slot, 1))
	}
	line := atom.Spaces(token.Margin+g.indent()) + st.Subtle.Render(prefix) + st.Strong.Render(head)
	at := token.Margin + g.indent() + atom.Width(prefix) + atom.Width(head)
	value, detail := atom.Show(l.Value), atom.Show(l.Detail)
	if value != "" {
		pad := max(slot-atom.Width(head), 0) + token.Gap
		if detail != "" && at+pad+atom.Width(value)+token.Gap+atom.Width(detail) > end {
			detail = ""
		}
		text := atom.Fit(l.Value, max(end-at-pad, 1))
		line += atom.Spaces(pad) + st.Text.Render(text)
		at += pad + atom.Width(text)
	}
	if detail != "" && at+token.Gap+atom.Width(detail) <= end {
		line += atom.Spaces(token.Gap) + st.Muted.Render(detail)
	}
	return line
}

func countEntries(it page.Item) int {
	switch {
	case it.Row != nil:
		return 1
	case it.Group != nil:
		n := 1
		for _, c := range it.Group.Items {
			n += countEntries(c)
		}
		return n
	case it.Link != nil, it.Empty != nil:
		return 1
	}
	return 0
}

func (g Grid) entries() []entry {
	var out []entry
	var walk func(items []page.Item, bars string, depth int, hidden bool)
	walk = func(items []page.Item, bars string, depth int, hidden bool) {
		last := -1
		for i, it := range items {
			if it.Row != nil || it.Group != nil || (depth > 0 && it.Link != nil) {
				last = i
			}
		}
		for i, it := range items {
			if it.Row == nil && it.Group == nil && (depth == 0 || (it.Link == nil && it.Empty == nil)) {
				continue
			}
			prefix := ""
			next := bars
			switch {
			case depth > 0 && it.Empty != nil:
				// invariant: a group's empty child is no branch of the tree: it
				// sits one step in, without a connector (§13.5 `no VMs`).
				prefix = bars + atom.Spaces(token.TreeStep)
			case depth > 0:
				if i == last {
					prefix, next = bars+token.TreeEnd, bars+atom.Spaces(token.TreeStep)
				} else {
					prefix, next = bars+token.TreeMid, bars+token.TreeBar
				}
			}
			if it.Empty != nil {
				e := entry{empty: it.Empty, prefix: prefix, key: "empty"}
				if hidden {
					e.key = ""
				}
				out = append(out, e)
				continue
			}
			if it.Link != nil {
				e := entry{link: it.Link, prefix: prefix, key: "link"}
				if hidden {
					e.key = ""
				}
				out = append(out, e)
				continue
			}
			if it.Row != nil {
				out = append(out, g.entry(*it.Row, prefix, false, hidden))
				continue
			}
			e := g.entry(it.Group.Head, prefix, true, hidden)
			out = append(out, e)
			shut := hidden || (g.Folded != nil && g.Folded(e.keyOr(g.Section.ID)))
			walk(it.Group.Items, next, depth+1, shut)
		}
	}
	walk(g.Section.Items, "", 0, false)
	return out
}

func (e entry) keyOr(section string) string {
	if e.key != "" {
		return e.key
	}
	return RowKey(section, e.row.ID)
}

func (g Grid) entry(r page.Row, prefix string, head, hidden bool) entry {
	key := RowKey(g.Section.ID, r.ID)
	r.Detail = atom.Detail(r)
	if hidden || (g.Match != nil && !g.Match(r)) {
		key = ""
	}
	return entry{row: r, prefix: prefix, head: head, key: key}
}

func (g Grid) columns() []column {
	var cols []column
	for i, c := range g.Section.Columns {
		if i == 0 || c.Show == page.PipeOnly || c.Show == page.DetailOnly {
			continue
		}
		at := i - 1
		if c.Kind == page.KindLevels {
			for li, level := range c.Levels {
				cols = append(cols, column{kind: page.KindSize, label: level, keep: c.Keep, cell: levelCell(at, level, li == 0)})
			}
			continue
		}
		cols = append(cols, column{kind: c.Kind, label: c.Label, keep: c.Keep, whole: c.Whole, cell: positional(at)})
	}
	return cols
}

func (g Grid) layout(entries []entry) (cols []column, labelWidth int) {
	cols = g.columns()
	if len(g.Section.Columns) > 0 {
		labelWidth = atom.Width(g.Section.Columns[0].Label)
	}
	g.stacks(entries, cols)
	for _, e := range entries {
		if e.key == "" || e.link != nil || e.empty != nil || e.stack {
			continue
		}
		labelWidth = max(labelWidth, atom.Width(e.prefix)+atom.Width(shown(e.row)))
		for i := range cols {
			v := cols[i].cell(e.row)
			cols[i].width = max(cols[i].width, atom.Width(atom.Show(v)))
			if i == len(cols)-1 && atom.Uncut(v.Kind) {
				cols[i].firm = max(cols[i].firm, atom.Width(atom.Show(v)))
			}
		}
	}
	labelWidth = min(labelWidth, token.LabelMax)
	for i := range cols {
		if cols[i].width > 0 && cols[i].label != "" {
			cols[i].width = max(cols[i].width, atom.Width(cols[i].label))
		}
	}
	cols = slices.DeleteFunc(cols, func(c column) bool { return c.width == 0 })
	avail := g.avail()
	g.overflow(entries, cols, labelWidth, avail)
	fixed := func() int {
		n := labelWidth
		for _, c := range cols {
			if c.width >= 0 {
				n += token.Gap + c.width
			}
		}
		return n
	}
	order := make([]int, len(cols))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int { return cols[a].keep - cols[b].keep })
	natural := make([]int, len(cols))
	for i := range cols {
		natural[i] = cols[i].width
	}
	for _, i := range order {
		switch over := fixed() - avail; {
		case over <= 0:
		case cols[i].whole:
			cols[i].width = -1
		default:
			cols[i].width = max(cols[i].width-over, min(atom.Floor(cols[i].kind), cols[i].width), cols[i].firm)
		}
	}
	if over := fixed() - avail; over > 0 {
		labelWidth = max(labelWidth-over, min(token.LabelMin, labelWidth))
	}
	for _, i := range order {
		if fixed() <= avail {
			break
		}
		if g.whole(cols[i]) {
			continue
		}
		cols[i].width = -1
	}
	// invariant: what a dropped column frees goes back to the columns cut
	// before it, the most kept first, so a drop never leaves a cut cell beside
	// empty room.
	for k := len(order) - 1; k >= 0; k-- {
		i := order[k]
		if slack := avail - fixed(); slack > 0 && cols[i].width >= 0 && cols[i].width < natural[i] {
			cols[i].width = min(natural[i], cols[i].width+slack)
		}
	}
	cols = slices.DeleteFunc(cols, func(c column) bool { return c.width < 0 })
	g.room(entries, cols, labelWidth, avail)
	for i, e := range entries {
		if e.row.Label != "" && atom.Width(e.prefix)+atom.Width(shown(e.row)) > labelWidth {
			entries[i].stack = true
		}
	}
	return cols, labelWidth
}

// value: a command column on a terminal: a command cut no longer runs, so
// it is never cut nor dropped (OU-05).
func (g Grid) whole(c column) bool {
	return c.kind == page.KindCommand
}

// invariant: on a terminal a name is never cut nor a command: a row whose
// name is wider than a name column may be, or whose command does not fit
// beside the names, stacks, the widest command first, until the others fit
// side by side, each column at its floor (OU-05).
func (g Grid) stacks(entries []entry, cols []column) {
	live := func(e entry) bool { return e.key != "" && e.link == nil && e.empty == nil && !e.stack }
	for i, e := range entries {
		if live(e) && atom.Width(e.prefix)+atom.Width(shown(e.row)) > token.LabelMax {
			entries[i].stack = true
		}
	}
	commands := func(e entry) int {
		n := 0
		for _, c := range cols {
			if g.whole(c) {
				n += atom.Width(atom.Show(c.cell(e.row)))
			}
		}
		return n
	}
	for {
		labelWidth, widths := 0, make([]int, len(cols))
		widest, most := -1, 0
		for i, e := range entries {
			if !live(e) {
				continue
			}
			labelWidth = max(labelWidth, atom.Width(e.prefix)+atom.Width(shown(e.row)))
			for ci, c := range cols {
				w := atom.Width(atom.Show(c.cell(e.row)))
				if !g.whole(c) {
					w = min(w, atom.Floor(c.kind))
				}
				widths[ci] = max(widths[ci], w)
			}
			if n := commands(e); n > most {
				widest, most = i, n
			}
		}
		used := labelWidth
		for _, w := range widths {
			if w > 0 {
				used += token.Gap + w
			}
		}
		if widest < 0 || used <= g.avail() {
			return
		}
		entries[widest].stack = true
	}
}

// invariant: a long path never crushes a row's detail: while the details of
// the rows that fill the last column lack room (up to DetailMin each; the
// other rows' details sit beside their own cells), a column holding a path is
// cut in the middle, the least kept first, down to the path's first and last
// segments and to its other cells' width; it then keeps only the width its
// cut cells take.
func (g Grid) room(entries []entry, cols []column, labelWidth, avail int) {
	rows := make([]page.Row, 0, len(entries))
	need := 0
	for _, e := range entries {
		if e.key == "" || e.link != nil || e.empty != nil || e.stack {
			continue
		}
		rows = append(rows, e.row)
		if len(cols) == 0 || atom.Show(cols[len(cols)-1].cell(e.row)) == "" {
			continue
		}
		if d := atom.Width(atom.Show(e.row.Detail)); d > 0 && !e.overflow {
			need = max(need, token.Gap+min(d, token.DetailMin))
		}
	}
	short := func() int {
		n := labelWidth + need
		for _, c := range cols {
			n += token.Gap + c.width
		}
		return n - avail
	}
	if need == 0 || short() <= 0 {
		return
	}
	order := make([]int, len(cols))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int { return cols[a].keep - cols[b].keep })
	for _, i := range order {
		c := &cols[i]
		floor, middle := 0, false
		for _, r := range rows {
			v := c.cell(r)
			w := atom.Width(atom.Show(v))
			if v.Kind == page.KindPath {
				middle = middle || atom.Least(v) < w
				w = atom.Least(v)
			}
			floor = max(floor, w)
		}
		if !middle || c.width <= floor {
			continue
		}
		target := max(floor, c.width-short())
		c.width = 0
		for _, r := range rows {
			c.width = max(c.width, atom.Width(atom.Fit(c.cell(r), target)))
		}
		if short() <= 0 {
			return
		}
	}
}

// invariant: per-row overflow: one row's long value (the daemon's two
// versions, a host's mismatch) never takes the room the other rows need:
// while the line or a detail does not fit, a last-column cell more than twice
// as wide as any other leaves the column's width and runs to the end of its
// own line, or under the row when even that is short (§11.2).
func (g Grid) overflow(entries []entry, cols []column, labelWidth, avail int) {
	if len(cols) == 0 {
		return
	}
	last := &cols[len(cols)-1]
	if last.whole || atom.Numeric(last.kind) {
		return
	}
	for {
		used := labelWidth
		for _, c := range cols {
			used += token.Gap + c.width
		}
		widest, first, second, need := -1, 0, 0, 0
		for i, e := range entries {
			if e.key == "" || e.link != nil || e.empty != nil || e.overflow || e.stack {
				continue
			}
			if d := atom.Width(atom.Show(e.row.Detail)); d > 0 {
				need = max(need, token.Gap+d)
			}
			switch w := atom.Width(atom.Show(last.cell(e.row))); {
			case w > first:
				widest, first, second = i, w, first
			case w > second:
				second = w
			}
		}
		short := used > avail || (need > 0 && avail-used < need)
		if widest < 0 || !short || first <= 2*second {
			return
		}
		entries[widest].overflow = true
		last.width = max(second, atom.Width(last.label))
	}
}

// invariant: on a terminal an Overview's section line is never cut: each
// part of it carries a mark and a state word (`◐ 1 misplaced`), so a list
// its line cannot hold goes on under its first part (OU-05).
func (g Grid) wraps(ci int, cols []column, v page.Value) bool {
	return g.flush() && ci == len(cols)-1 && v.Kind == page.KindList && atom.Width(atom.Show(v)) > cols[ci].width
}

// value: a list in lines of room columns, as many of its items on each as
// it holds; an item wider than room has a line of its own.
func wrapList(v page.Value, room int) []page.Value {
	var parts []page.Value
	var line []page.Value
	for _, it := range v.Items {
		if atom.Show(it) == "" {
			continue
		}
		if next := append(slices.Clone(line), it); len(line) == 0 || atom.Width(atom.Show(page.List(next...))) <= room {
			line = next
			continue
		}
		parts = append(parts, page.List(line...))
		line = []page.Value{it}
	}
	if len(line) > 0 {
		parts = append(parts, page.List(line...))
	}
	return parts
}

func cut(v page.Value, width int) string {
	if s := atom.Fit(v, width); atom.Width(s) <= width {
		return s
	}
	return atom.End(atom.Show(v), width)
}

// invariant: on a terminal, a banner with no title (an Overview's section
// lines, the On line) is drawn flush at the margin, as the vision's CLI pages
// are, and keeps no column for a mark none of its rows carries.
func (g Grid) flush() bool {
	return g.Section.Role == page.Banner && g.Section.Title == "" && len(g.Section.Caption) == 0
}

func (g Grid) indent() int {
	if g.flush() {
		return 0
	}
	return token.Indent
}

func (g Grid) markWidth() int {
	if !g.flush() {
		return token.MarkWidth
	}
	for _, it := range g.Section.Items {
		if it.Row != nil && it.Row.Mark != page.Unjudged || it.Group != nil {
			return token.MarkWidth
		}
	}
	return 0
}

func (g Grid) avail() int {
	return g.Width - 1 - token.Margin - g.indent() - g.markWidth()
}

func positional(at int) func(page.Row) page.Value {
	return func(r page.Row) page.Value {
		if at < len(r.Cells) {
			return r.Cells[at]
		}
		return page.None()
	}
}

func levelCell(at int, level string, first bool) func(page.Row) page.Value {
	return func(r page.Row) page.Value {
		if at >= len(r.Cells) {
			return page.None()
		}
		if c := r.Cells[at]; c.Kind != page.KindLevels {
			if first {
				return c
			}
			return page.None()
		}
		for _, l := range r.Cells[at].Items {
			if l.Label == level && l.N != 0 {
				return l
			}
		}
		return page.None()
	}
}

func (g Grid) header(cols []column, labelWidth int) string {
	labeled := false
	for _, c := range cols {
		labeled = labeled || c.label != ""
	}
	if !labeled {
		return ""
	}
	label := ""
	if len(g.Section.Columns) > 0 {
		label = g.Section.Columns[0].Label
	}
	var line strings.Builder
	line.WriteString(atom.Spaces(token.Margin+g.indent()+g.markWidth()) + atom.Pad(label, labelWidth))
	for _, c := range cols {
		cell := atom.Pad(c.label, c.width)
		if atom.Numeric(c.kind) {
			cell = atom.PadLeft(c.label, c.width)
		}
		line.WriteString(atom.Spaces(token.Gap) + cell)
	}
	return g.Styles.Muted.Render(strings.TrimRight(line.String(), " "))
}

func (g Grid) row(e entry, cols []column, labelWidth int) []Line {
	st, r := g.Styles, e.row
	indent := atom.Spaces(token.Margin + g.indent())
	lead := indent + st.Subtle.Render(e.prefix) + atom.Mark(st, r.Mark)
	if g.markWidth() == 0 {
		lead = indent + st.Subtle.Render(e.prefix)
	}
	label := atom.End(shown(r), max(labelWidth-atom.Width(e.prefix), 1))
	// invariant: a wrapped hint starts at the label's column (G5) in every
	// section, medium and width: never under a detail whose place depends on
	// the widest cell of the section.
	hintLines := func() []Line {
		pad := indent + atom.Spaces(atom.Width(e.prefix)+g.markWidth())
		h := molecule.Hint(st, *r.Hint)
		width := atom.Width(molecule.HintText(*r.Hint))
		if atom.Width(pad)+width < g.Width || !strings.Contains(r.Hint.Command, joint) {
			return []Line{{Kind: LineHint, Key: e.key, Text: pad + h}}
		}
		parts := strings.Split(r.Hint.Command, joint)
		out := make([]Line, 0, len(parts))
		for i, part := range parts {
			if i < len(parts)-1 {
				part += strings.TrimRight(joint, " ")
			}
			text := st.Text.Render(part)
			if i == 0 {
				text = st.Muted.Render(token.Arrow+" ") + text
			} else {
				text = atom.Spaces(token.Gap) + text
			}
			out = append(out, Line{Kind: LineHint, Key: e.key, Text: pad + text})
		}
		return out
	}
	if e.stack {
		return g.stacked(e, lead, hintLines)
	}
	if g.Width < token.MinWidth {
		lines := []Line{{Kind: LineRow, Key: e.key, Head: e.head, Text: lead + st.Strong.Render(label)}}
		var values []string
		for _, c := range cols {
			if s := atom.Show(c.cell(r)); s != "" {
				values = append(values, s)
			}
		}
		if s := atom.Show(r.Detail); s != "" {
			values = append(values, s)
		}
		if len(values) > 0 {
			lines = append(lines, Line{Kind: LineOther, Text: indent + atom.Spaces(token.MarkWidth*2) + st.Muted.Render(strings.Join(values, atom.Spaces(token.Gap)))})
		}
		if r.Hint != nil {
			lines = append(lines, hintLines()...)
		}
		return lines
	}
	var b strings.Builder
	b.WriteString(lead + st.Strong.Render(label) + atom.Spaces(labelWidth-atom.Width(e.prefix)-atom.Width(label)))
	used := labelWidth
	ownLine, ownUsed := lead+st.Strong.Render(label), labelWidth
	openLine, openUsed := b.String(), labelWidth
	var below []Line
	for ci, c := range cols {
		v := c.cell(r)
		if e.overflow && ci == len(cols)-1 {
			if w := atom.Width(atom.Show(v)); w > g.avail()-used-token.Gap {
				pad := indent + atom.Spaces(atom.Width(e.prefix)+g.markWidth())
				below = append(below, Line{Kind: LineOther, Key: e.key, Text: pad + st.Text.Render(atom.Fit(v, g.Width-1-atom.Width(pad)))})
				v = page.None()
			}
			c.width = max(min(atom.Width(atom.Show(v)), g.avail()-used-token.Gap), 0)
		}
		if g.wraps(ci, cols, v) {
			if parts := wrapList(v, g.avail()-used-token.Gap); len(parts) > 1 {
				pad := atom.Spaces(token.Margin + g.indent() + g.markWidth() + used + token.Gap)
				for _, part := range parts[1:] {
					below = append(below, Line{Kind: LineOther, Key: e.key, Text: pad + atom.Paint(st, st.Text, part, atom.Show(part))})
				}
				v = parts[0]
				c.width = max(c.width, atom.Width(atom.Show(v)))
			}
		}
		text := atom.Fit(v, c.width)
		// invariant: a row whose cells end before the last column keeps that
		// column's cell beside its own cells when the aligned place would cut
		// it and that one holds it whole.
		if whole := atom.Show(v); ci == len(cols)-1 && text != whole && openUsed < used && token.Gap+atom.Width(whole) <= g.avail()-openUsed {
			b.Reset()
			b.WriteString(openLine)
			text, used, c.width = whole, openUsed, atom.Width(whole)
		}
		styled := atom.Paint(st, st.Text, v, text)
		if v.Stale {
			styled = st.Muted.Render(text)
		}
		if text == "" {
			styled = ""
		}
		if atom.Numeric(c.kind) {
			b.WriteString(atom.Spaces(token.Gap+c.width-atom.Width(text)) + styled)
		} else {
			b.WriteString(atom.Spaces(token.Gap) + styled)
			if text != "" {
				ownLine, ownUsed = b.String(), used+token.Gap+atom.Width(text)
			}
			b.WriteString(atom.Spaces(c.width - atom.Width(text)))
		}
		used += token.Gap + c.width
		if text != "" && atom.Numeric(c.kind) {
			ownLine, ownUsed = b.String(), used
		}
		if text != "" {
			openLine, openUsed = b.String(), used
		}
	}
	rest := g.avail() - used
	detail := atom.Show(r.Detail)
	hint, hintWidth := "", 0
	if r.Hint != nil {
		hint = molecule.Hint(st, *r.Hint)
		hintWidth = atom.Width(molecule.HintText(*r.Hint))
	}
	// invariant: a row whose last columns are blank in it keeps its detail
	// beside its own cells when the aligned place would cut it and that one
	// holds it whole.
	if w := token.Gap + atom.Width(detail); detail != "" && w > rest && ownUsed < used && w <= g.avail()-ownUsed {
		b.Reset()
		b.WriteString(ownLine)
		rest = g.avail() - ownUsed
	}
	// invariant: a detail is never cut below its kind's floor (`~…`): a path
	// or a URL, which says where something is, goes on its own line under the
	// row at the label column, cut by its kind; any other detail is lost on
	// that width, as a dropped column is (§7.1, §11.2).
	if w := atom.Width(detail); detail != "" && rest-token.Gap < min(w, atom.Floor(r.Detail.Kind)) {
		if k := r.Detail.Kind; k == page.KindPath || k == page.KindURL {
			pad := indent + atom.Spaces(atom.Width(e.prefix)+g.markWidth())
			below = append(below, Line{Kind: LineOther, Key: e.key, Text: pad + st.Muted.Render(cut(r.Detail, g.Width-1-atom.Width(pad)))})
		}
		detail = ""
	}
	own := false
	switch {
	case detail != "" && hint != "":
		if room := rest - token.Gap*2 - hintWidth; room >= atom.Width(detail) {
			b.WriteString(atom.Spaces(token.Gap) + st.Muted.Render(detail) + atom.Spaces(token.Gap) + hint)
		} else {
			if room := rest - token.Gap; room >= 1 {
				b.WriteString(atom.Spaces(token.Gap) + st.Muted.Render(cut(r.Detail, room)))
			}
			own = true
		}
	case detail != "":
		if room := rest - token.Gap; room >= 1 {
			b.WriteString(atom.Spaces(token.Gap) + st.Muted.Render(cut(r.Detail, room)))
		}
	case hint != "":
		if token.Gap+hintWidth <= rest {
			b.WriteString(atom.Spaces(token.Gap) + hint)
		} else {
			own = true
		}
	}
	lines := append([]Line{{Kind: LineRow, Key: e.key, Head: e.head, Text: strings.TrimRight(b.String(), " ")}}, below...)
	if own {
		lines = append(lines, hintLines()...)
	}
	return lines
}

// invariant: a stacked row writes its name whole, then its cells, its detail
// and its hint under it at the name's column, each whole when its line holds
// it, else on a line of its own; only a detail is cut there, by its kind.
func (g Grid) stacked(e entry, lead string, hintLines func() []Line) []Line {
	st, r := g.Styles, e.row
	lines := []Line{{Kind: LineRow, Key: e.key, Head: e.head, Text: lead + st.Strong.Render(shown(r))}}
	pad := atom.Spaces(token.Margin + g.indent() + atom.Width(e.prefix) + g.markWidth())
	room := g.Width - 1 - atom.Width(pad)
	line, used := "", 0
	flush := func() {
		if line != "" {
			lines = append(lines, Line{Kind: LineOther, Key: e.key, Text: pad + line})
		}
		line, used = "", 0
	}
	put := func(v page.Value, muted, cuts bool) {
		text := atom.Show(v)
		if text == "" {
			return
		}
		gap := 0
		if used > 0 {
			gap = token.Gap
		}
		if used+gap+atom.Width(text) > room && used > 0 {
			flush()
			gap = 0
		}
		if cuts {
			text = cut(v, room)
		}
		style := st.Text
		if muted {
			style = st.Muted
		}
		line += atom.Spaces(gap) + atom.Paint(st, style, v, text)
		used += gap + atom.Width(text)
	}
	for _, c := range g.columns() {
		put(c.cell(r), false, !g.whole(c) && !atom.Uncut(c.kind))
	}
	put(r.Detail, true, !atom.Uncut(r.Detail.Kind))
	if r.Hint != nil {
		if w := atom.Width(molecule.HintText(*r.Hint)); used > 0 && used+token.Gap+w <= room {
			line += atom.Spaces(token.Gap) + molecule.Hint(st, *r.Hint)
			flush()
			return lines
		}
		flush()
		return append(lines, hintLines()...)
	}
	flush()
	return lines
}
